/** Self-registration for the packaged Windows Chrome Native Messaging host. */

import { spawnSync } from 'node:child_process';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { CHAOS_EXTENSION_ORIGIN, NATIVE_HOST_NAME } from './protocol.js';

const REGISTRY_KEY = `HKCU\\Software\\Google\\Chrome\\NativeMessagingHosts\\${NATIVE_HOST_NAME}`;

function manifestPath(): string {
  const base = process.env.LOCALAPPDATA || path.join(os.homedir(), 'AppData', 'Local');
  return path.join(base, 'chaOS', 'native-messaging', `${NATIVE_HOST_NAME}.json`);
}

function runReg(args: string[], tolerateFailure = false): void {
  const result = spawnSync('reg.exe', args, { encoding: 'utf8', windowsHide: true });
  if (!tolerateFailure && result.status !== 0) {
    throw new Error((result.stderr || result.stdout || `reg.exe failed with ${result.status}`).trim());
  }
}

export function registerNativeMessagingHost(executable = process.execPath): { manifest: string; executable: string } {
  if (process.platform !== 'win32') throw new Error('Native-host self-registration currently supports Windows only.');
  const manifest = manifestPath();
  mkdirSync(path.dirname(manifest), { recursive: true });
  writeFileSync(
    manifest,
    `${JSON.stringify({
      name: NATIVE_HOST_NAME,
      description: 'chaOS local capability host for ChatGPT in Chrome',
      path: path.resolve(executable),
      type: 'stdio',
      allowed_origins: [CHAOS_EXTENSION_ORIGIN]
    }, null, 2)}\n`,
    'utf8'
  );
  runReg(['add', REGISTRY_KEY, '/ve', '/t', 'REG_SZ', '/d', manifest, '/f']);
  return { manifest, executable: path.resolve(executable) };
}

export function unregisterNativeMessagingHost(): void {
  if (process.platform !== 'win32') throw new Error('Native-host self-registration currently supports Windows only.');
  runReg(['delete', REGISTRY_KEY, '/f'], true);
  rmSync(manifestPath(), { force: true });
}
