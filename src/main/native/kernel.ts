/**
 * Small local capability kernel used by browser Native Messaging mode.
 *
 * This intentionally does not depend on MCP, browser conversation state, workers, Goal/Loop or
 * session recording. It reuses the existing approved-root sandbox and child-process environment
 * hardening, then exposes a deliberately small set of deterministic actions.
 */

import { spawn, type ChildProcessWithoutNullStreams } from 'node:child_process';
import path from 'node:path';
import { rawPromises as fs } from '../rawfs.js';
import { getConfig } from '../config.js';
import { resolvePath } from '../sandbox.js';
import { childEnv, findPowerShell, terminateProcessTree } from '../exec.js';
import { DEFAULT_EXCLUDES, search } from '../search.js';
import { APP_VERSION } from '../version.js';
import type { Capability } from '../../shared/types.js';
import { WRITE_CAPABILITIES } from '../../shared/types.js';

const MAX_TEXT_WRITE_BYTES = 4 * 1024 * 1024;
const MAX_READ_BYTES = 1024 * 1024;
const MAX_PROCESS_BUFFER_BYTES = 1024 * 1024;
const MAX_PROCESS_CHUNKS = 2000;
const MAX_COMMAND_CHARS = 32_000;

export class NativeKernelError extends Error {
  readonly code: string;

  constructor(code: string, message: string) {
    super(message);
    this.name = 'NativeKernelError';
    this.code = code;
  }
}

interface ProcessChunk {
  seq: number;
  stream: 'stdout' | 'stderr';
  text: string;
  bytes: number;
}

interface ProcessEntry {
  id: number;
  child: ChildProcessWithoutNullStreams;
  workdir: string;
  startedAt: number;
  exitedAt: number | null;
  exitCode: number | null;
  signal: NodeJS.Signals | null;
  nextSeq: number;
  droppedThrough: number;
  bytes: number;
  chunks: ProcessChunk[];
}

let nextProcessId = 1;
const processes = new Map<number, ProcessEntry>();

function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new NativeKernelError('BAD_ARGS', 'args must be an object');
  }
  return value as Record<string, unknown>;
}

function requiredString(args: Record<string, unknown>, key: string, max = 4096): string {
  const value = args[key];
  if (typeof value !== 'string' || value.length === 0 || value.length > max) {
    throw new NativeKernelError('BAD_ARGS', `${key} must be a non-empty string up to ${max} characters`);
  }
  return value;
}

function optionalString(args: Record<string, unknown>, key: string, max = 4096): string | undefined {
  const value = args[key];
  if (value === undefined) return undefined;
  if (typeof value !== 'string' || value.length > max) {
    throw new NativeKernelError('BAD_ARGS', `${key} must be a string up to ${max} characters`);
  }
  return value;
}

function boundedInteger(value: unknown, fallback: number, min: number, max: number, name: string): number {
  if (value === undefined) return fallback;
  if (!Number.isInteger(value) || (value as number) < min || (value as number) > max) {
    throw new NativeKernelError('BAD_ARGS', `${name} must be an integer between ${min} and ${max}`);
  }
  return value as number;
}

function requireCapability(capability: Capability): void {
  const config = getConfig();
  if (!config.capabilities[capability]) {
    throw new NativeKernelError('CAPABILITY_DISABLED', `${capability} capability is disabled in chaOS`);
  }
  if (config.readOnly && (WRITE_CAPABILITIES as readonly Capability[]).includes(capability)) {
    throw new NativeKernelError('READ_ONLY', 'chaOS is in read-only mode');
  }
}

function virtualChild(parent: string, name: string): string {
  const base = parent.endsWith('/') ? parent.slice(0, -1) : parent;
  return `${base}/${name}`;
}

async function listDirectory(args: Record<string, unknown>): Promise<unknown> {
  requireCapability('browse');
  const requested = requiredString(args, 'path');
  const limit = boundedInteger(args.limit, 200, 1, 500, 'limit');
  const resolved = await resolvePath(getConfig().roots, requested);
  const stat = await fs.stat(resolved.real);
  if (!stat.isDirectory()) throw new NativeKernelError('NOT_DIRECTORY', `${resolved.virtual} is not a directory`);
  const entries = await fs.readdir(resolved.real, { withFileTypes: true });
  entries.sort((left, right) => left.name.localeCompare(right.name));
  const sliced = entries.slice(0, limit).map((entry) => ({
    path: virtualChild(resolved.virtual, entry.name),
    type: entry.isDirectory() ? 'directory' : entry.isFile() ? 'file' : entry.isSymbolicLink() ? 'link' : 'other'
  }));
  return { path: resolved.virtual, entries: sliced, truncated: entries.length > sliced.length };
}

async function readFile(args: Record<string, unknown>): Promise<unknown> {
  requireCapability('read');
  const requested = requiredString(args, 'path');
  const offset = boundedInteger(args.offset, 0, 0, Number.MAX_SAFE_INTEGER, 'offset');
  const limit = boundedInteger(args.limit, 256 * 1024, 1, MAX_READ_BYTES, 'limit');
  const resolved = await resolvePath(getConfig().roots, requested);
  const stat = await fs.stat(resolved.real);
  if (!stat.isFile()) throw new NativeKernelError('NOT_FILE', `${resolved.virtual} is not a regular file`);
  if (offset >= stat.size) {
    return { path: resolved.virtual, size: stat.size, offset, data: '', encoding: 'utf8', eof: true };
  }
  const length = Math.min(limit, stat.size - offset);
  const handle = await fs.open(resolved.real, 'r');
  try {
    const buffer = Buffer.alloc(length);
    const { bytesRead } = await handle.read(buffer, 0, length, offset);
    const data = buffer.subarray(0, bytesRead);
    const binary = data.subarray(0, Math.min(data.length, 8192)).includes(0);
    return {
      path: resolved.virtual,
      size: stat.size,
      offset,
      bytes: bytesRead,
      encoding: binary ? 'base64' : 'utf8',
      data: binary ? data.toString('base64') : data.toString('utf8'),
      eof: offset + bytesRead >= stat.size
    };
  } finally {
    await handle.close();
  }
}

async function searchFiles(args: Record<string, unknown>): Promise<unknown> {
  requireCapability('search');
  const requested = requiredString(args, 'path');
  const query = requiredString(args, 'query', 2000);
  const modeValue = args.mode ?? 'content';
  if (modeValue !== 'content' && modeValue !== 'name') {
    throw new NativeKernelError('BAD_ARGS', 'mode must be "content" or "name"');
  }
  const include = optionalString(args, 'include', 500);
  const maxResults = boundedInteger(args.max_results, 50, 1, 200, 'max_results');
  const caseSensitive = args.case_sensitive === true;
  const regex = args.regex === true;
  const excludeValue = args.exclude;
  let exclude: readonly string[] = DEFAULT_EXCLUDES;
  if (excludeValue !== undefined) {
    if (!Array.isArray(excludeValue) || excludeValue.length > 64 || excludeValue.some((item) => typeof item !== 'string')) {
      throw new NativeKernelError('BAD_ARGS', 'exclude must be an array of at most 64 strings');
    }
    exclude = excludeValue as string[];
  }
  const resolved = await resolvePath(getConfig().roots, requested);
  const stat = await fs.stat(resolved.real);
  if (!stat.isDirectory()) throw new NativeKernelError('NOT_DIRECTORY', `${resolved.virtual} is not a directory`);
  return await search({
    realDir: resolved.real,
    virtualDir: resolved.virtual,
    query,
    mode: modeValue,
    include,
    exclude,
    caseSensitive,
    regex,
    maxResults
  });
}

async function writeFile(args: Record<string, unknown>): Promise<unknown> {
  const requested = requiredString(args, 'path');
  const content = requiredString(args, 'content', MAX_TEXT_WRITE_BYTES);
  if (Buffer.byteLength(content, 'utf8') > MAX_TEXT_WRITE_BYTES) {
    throw new NativeKernelError('TOO_LARGE', `content exceeds ${MAX_TEXT_WRITE_BYTES} bytes`);
  }
  const resolved = await resolvePath(getConfig().roots, requested, { allowMissing: true });
  const exists = await fs.stat(resolved.real).then(() => true, (error: NodeJS.ErrnoException) => {
    if (error.code === 'ENOENT') return false;
    throw error;
  });
  requireCapability(exists ? 'edit' : 'create');
  if (exists) {
    const stat = await fs.stat(resolved.real);
    if (!stat.isFile()) throw new NativeKernelError('NOT_FILE', `${resolved.virtual} is not a regular file`);
  } else {
    await fs.mkdir(path.dirname(resolved.real), { recursive: true });
  }
  await fs.writeFile(resolved.real, content, 'utf8');
  return { path: resolved.virtual, bytes: Buffer.byteLength(content, 'utf8'), created: !exists };
}

async function replaceText(args: Record<string, unknown>): Promise<unknown> {
  requireCapability('edit');
  const requested = requiredString(args, 'path');
  const before = requiredString(args, 'before', MAX_TEXT_WRITE_BYTES);
  const after = typeof args.after === 'string' ? args.after : (() => { throw new NativeKernelError('BAD_ARGS', 'after must be a string'); })();
  const expected = boundedInteger(args.expected, 1, 1, 1000, 'expected');
  const resolved = await resolvePath(getConfig().roots, requested);
  const original = await fs.readFile(resolved.real, 'utf8');
  let count = 0;
  let from = 0;
  while ((from = original.indexOf(before, from)) !== -1) {
    count += 1;
    from += before.length;
    if (count > expected) break;
  }
  if (count !== expected) {
    throw new NativeKernelError('REPLACE_COUNT_MISMATCH', `expected ${expected} exact match(es), found ${count}`);
  }
  const updated = original.split(before).join(after);
  if (Buffer.byteLength(updated, 'utf8') > MAX_TEXT_WRITE_BYTES) {
    throw new NativeKernelError('TOO_LARGE', `updated file exceeds ${MAX_TEXT_WRITE_BYTES} bytes`);
  }
  await fs.writeFile(resolved.real, updated, 'utf8');
  return { path: resolved.virtual, replacements: count, bytes: Buffer.byteLength(updated, 'utf8') };
}

function appendProcessChunk(entry: ProcessEntry, stream: 'stdout' | 'stderr', chunk: Buffer): void {
  const text = chunk.toString('utf8');
  if (!text) return;
  const item: ProcessChunk = {
    seq: entry.nextSeq++,
    stream,
    text,
    bytes: Buffer.byteLength(text, 'utf8')
  };
  entry.chunks.push(item);
  entry.bytes += item.bytes;
  while (entry.chunks.length > MAX_PROCESS_CHUNKS || entry.bytes > MAX_PROCESS_BUFFER_BYTES) {
    const removed = entry.chunks.shift();
    if (!removed) break;
    entry.bytes -= removed.bytes;
    entry.droppedThrough = removed.seq;
  }
}

function shellCommand(command: string): { file: string; args: string[] } {
  if (process.platform === 'win32') {
    const powershell = findPowerShell();
    if (powershell) {
      const prefix = '[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); ' +
        '$OutputEncoding=[System.Text.UTF8Encoding]::new($false); ';
      const encoded = Buffer.from(`${prefix}${command}`, 'utf16le').toString('base64');
      return { file: powershell, args: ['-NoLogo', '-NoProfile', '-EncodedCommand', encoded] };
    }
    return { file: process.env.ComSpec || 'cmd.exe', args: ['/d', '/s', '/c', command] };
  }
  const shell = process.env.SHELL || '/bin/sh';
  return { file: shell, args: ['-lc', command] };
}

async function runProcess(args: Record<string, unknown>): Promise<unknown> {
  requireCapability('command');
  const command = requiredString(args, 'command', MAX_COMMAND_CHARS);
  const workdir = requiredString(args, 'workdir');
  const resolved = await resolvePath(getConfig().roots, workdir);
  const stat = await fs.stat(resolved.real);
  if (!stat.isDirectory()) throw new NativeKernelError('NOT_DIRECTORY', `${resolved.virtual} is not a directory`);
  const prepared = shellCommand(command);
  const child = spawn(prepared.file, prepared.args, {
    cwd: resolved.real,
    env: childEnv(),
    windowsHide: true,
    shell: false,
    detached: process.platform !== 'win32',
    stdio: ['pipe', 'pipe', 'pipe']
  });
  const id = nextProcessId++;
  const entry: ProcessEntry = {
    id,
    child,
    workdir: resolved.virtual,
    startedAt: Date.now(),
    exitedAt: null,
    exitCode: null,
    signal: null,
    nextSeq: 1,
    droppedThrough: 0,
    bytes: 0,
    chunks: []
  };
  processes.set(id, entry);
  child.stdout.on('data', (chunk: Buffer) => appendProcessChunk(entry, 'stdout', chunk));
  child.stderr.on('data', (chunk: Buffer) => appendProcessChunk(entry, 'stderr', chunk));
  child.once('error', (error) => appendProcessChunk(entry, 'stderr', Buffer.from(`process error: ${error.message}\n`, 'utf8')));
  child.once('close', (code, signal) => {
    entry.exitCode = code;
    entry.signal = signal;
    entry.exitedAt = Date.now();
  });
  return { process_id: id, workdir: resolved.virtual, pid: child.pid ?? null, running: true, started_at: entry.startedAt };
}

function processEntry(args: Record<string, unknown>): ProcessEntry {
  const id = boundedInteger(args.process_id, 0, 1, Number.MAX_SAFE_INTEGER, 'process_id');
  const entry = processes.get(id);
  if (!entry) throw new NativeKernelError('NO_SUCH_PROCESS', `no chaOS process ${id}`);
  return entry;
}

async function readProcess(args: Record<string, unknown>): Promise<unknown> {
  const entry = processEntry(args);
  const after = boundedInteger(args.after, 0, 0, Number.MAX_SAFE_INTEGER, 'after');
  const chunks = entry.chunks
    .filter((chunk) => chunk.seq > after)
    .map(({ seq, stream, text }) => ({ seq, stream, text }));
  return {
    process_id: entry.id,
    running: entry.exitedAt === null,
    exit_code: entry.exitCode,
    signal: entry.signal,
    started_at: entry.startedAt,
    exited_at: entry.exitedAt,
    dropped_through: entry.droppedThrough,
    last_seq: entry.nextSeq - 1,
    chunks
  };
}

async function writeProcess(args: Record<string, unknown>): Promise<unknown> {
  requireCapability('command');
  const entry = processEntry(args);
  if (entry.exitedAt !== null || entry.child.stdin.destroyed) {
    throw new NativeKernelError('PROCESS_EXITED', `chaOS process ${entry.id} has exited`);
  }
  const text = requiredString(args, 'text', 256 * 1024);
  await new Promise<void>((resolve, reject) => {
    entry.child.stdin.write(text, (error) => error ? reject(error) : resolve());
  });
  return { process_id: entry.id, bytes: Buffer.byteLength(text, 'utf8') };
}

async function stopProcess(args: Record<string, unknown>): Promise<unknown> {
  const entry = processEntry(args);
  if (entry.exitedAt === null && entry.child.pid !== undefined) {
    await terminateProcessTree(entry.child.pid);
  }
  return { process_id: entry.id, stopped: true };
}

export async function shutdownNativeKernel(): Promise<void> {
  const active = [...processes.values()].filter((entry) => entry.exitedAt === null && entry.child.pid !== undefined);
  await Promise.all(active.map((entry) => terminateProcessTree(entry.child.pid!)));
}

export async function runNativeAction(action: string, rawArgs: unknown): Promise<unknown> {
  const args = rawArgs === undefined ? {} : record(rawArgs);
  switch (action) {
    case 'health': {
      const config = getConfig();
      return {
        app: 'chaOS',
        version: APP_VERSION,
        transport: 'chrome-native-messaging',
        read_only: config.readOnly,
        roots: config.roots.map((root) => `/${root.name}`),
        capabilities: { ...config.capabilities },
        actions: [
          'health', 'fs.list', 'fs.read', 'fs.search', 'fs.write', 'fs.replace',
          'proc.run', 'proc.read', 'proc.write', 'proc.stop'
        ]
      };
    }
    case 'fs.list': return await listDirectory(args);
    case 'fs.read': return await readFile(args);
    case 'fs.search': return await searchFiles(args);
    case 'fs.write': return await writeFile(args);
    case 'fs.replace': return await replaceText(args);
    case 'proc.run': return await runProcess(args);
    case 'proc.read': return await readProcess(args);
    case 'proc.write': return await writeProcess(args);
    case 'proc.stop': return await stopProcess(args);
    default: throw new NativeKernelError('UNKNOWN_ACTION', `unknown chaOS action: ${action}`);
  }
}
