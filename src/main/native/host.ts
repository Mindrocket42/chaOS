/** Chrome Native Messaging host carried by the packaged chaOS executable. */

import os from 'node:os';
import { app } from 'electron';
import { initConfigPath, loadConfig } from '../config.js';
import { NativeKernelError, runNativeAction, shutdownNativeKernel } from './kernel.js';
import {
  CHAOS_EXTENSION_ORIGIN,
  NATIVE_PROTOCOL,
  type NativeRequest,
  type NativeResponse,
  validRequestId
} from './protocol.js';

const MAX_REQUEST_BYTES = 8 * 1024 * 1024;
const MAX_RESPONSE_BYTES = 900 * 1024;
const MAX_SUCCESSFUL_CALLS = 512;
const LITTLE_ENDIAN = os.endianness() === 'LE';

interface SuccessfulCall {
  signature: string;
  response: NativeResponse;
}

const successfulCalls = new Map<string, SuccessfulCall>();

function writeUint32(buffer: Buffer, value: number): void {
  if (LITTLE_ENDIAN) buffer.writeUInt32LE(value, 0);
  else buffer.writeUInt32BE(value, 0);
}

function readUint32(buffer: Buffer): number {
  return LITTLE_ENDIAN ? buffer.readUInt32LE(0) : buffer.readUInt32BE(0);
}

function responseBytes(response: NativeResponse): Buffer {
  const body = Buffer.from(JSON.stringify(response), 'utf8');
  if (body.length > MAX_RESPONSE_BYTES) {
    const fallback: NativeResponse = {
      type: 'result',
      id: response.id,
      ok: false,
      protocol: NATIVE_PROTOCOL,
      error: { code: 'RESULT_TOO_LARGE', message: 'Native result exceeded the browser messaging limit' }
    };
    return Buffer.from(JSON.stringify(fallback), 'utf8');
  }
  return body;
}

async function writeResponse(response: NativeResponse): Promise<void> {
  const body = responseBytes(response);
  const prefix = Buffer.allocUnsafe(4);
  writeUint32(prefix, body.length);
  const frame = Buffer.concat([prefix, body]);
  await new Promise<void>((resolve, reject) => {
    process.stdout.write(frame, (error) => error ? reject(error) : resolve());
  });
}

function failure(id: string, code: string, message: string): NativeResponse {
  return { type: 'result', id, ok: false, protocol: NATIVE_PROTOCOL, error: { code, message } };
}

function callSignature(action: string, args: unknown): string {
  try {
    return JSON.stringify([action, args ?? null]);
  } catch {
    throw new NativeKernelError('BAD_ARGS', 'args must be JSON-serializable');
  }
}

function rememberSuccessfulCall(id: string, signature: string, response: NativeResponse): void {
  successfulCalls.set(id, { signature, response });
  while (successfulCalls.size > MAX_SUCCESSFUL_CALLS) {
    const oldest = successfulCalls.keys().next().value as string | undefined;
    if (!oldest) break;
    successfulCalls.delete(oldest);
  }
}

async function handleRequest(value: unknown): Promise<NativeResponse> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return failure('invalid', 'BAD_REQUEST', 'Native request must be an object');
  }
  const request = value as Partial<NativeRequest> & Record<string, unknown>;
  const id = validRequestId(request.id) ? request.id : 'invalid';
  if (!validRequestId(request.id)) return failure(id, 'BAD_REQUEST', 'Request id is missing or invalid');
  if (request.type === 'ping') {
    return {
      type: 'result',
      id,
      ok: true,
      protocol: NATIVE_PROTOCOL,
      result: { app: 'chaOS', transport: 'chrome-native-messaging', protocol: NATIVE_PROTOCOL }
    };
  }
  if (request.type !== 'call' || typeof request.action !== 'string' || request.action.length === 0 || request.action.length > 128) {
    return failure(id, 'BAD_REQUEST', 'Expected a call request with an action');
  }
  try {
    const signature = callSignature(request.action, request.args);
    const previous = successfulCalls.get(id);
    if (previous) {
      if (previous.signature !== signature) {
        return failure(id, 'REQUEST_ID_REUSED', 'This successful action id was already used for different arguments');
      }
      return previous.response;
    }
    const result = await runNativeAction(request.action, request.args);
    const response: NativeResponse = { type: 'result', id, ok: true, protocol: NATIVE_PROTOCOL, result };
    rememberSuccessfulCall(id, signature, response);
    return response;
  } catch (error) {
    if (error instanceof NativeKernelError) return failure(id, error.code, error.message);
    const message = error instanceof Error ? error.message : String(error);
    return failure(id, 'INTERNAL_ERROR', message.slice(0, 2000));
  }
}

export async function runNativeMessagingHost(origin: string): Promise<void> {
  // Chrome also checks allowed_origins in the host manifest. Keep the process-side check because
  // the executable is a security boundary too, not merely an implementation detail of Chrome.
  if (origin !== CHAOS_EXTENSION_ORIGIN) {
    process.stderr.write(`chaOS native host refused origin ${origin}\n`);
    app.exit(2);
    return;
  }

  // stdout is the protocol. Human diagnostics must never corrupt it.
  console.log = (...args: unknown[]) => process.stderr.write(`${args.map(String).join(' ')}\n`);
  console.info = console.log;
  console.warn = console.log;
  console.error = console.log;

  await app.whenReady();
  initConfigPath(app.getPath('userData'));
  await loadConfig();

  let buffered = Buffer.alloc(0);
  let chain: Promise<void> = Promise.resolve();
  let ending = false;

  const stop = async (code: number): Promise<void> => {
    if (ending) return;
    ending = true;
    await shutdownNativeKernel().catch(() => undefined);
    app.exit(code);
  };

  process.stdin.on('data', (chunk: Buffer) => {
    if (ending) return;
    buffered = buffered.length ? Buffer.concat([buffered, chunk]) : Buffer.from(chunk);
    for (;;) {
      if (buffered.length < 4) return;
      const size = readUint32(buffered.subarray(0, 4));
      if (size < 2 || size > MAX_REQUEST_BYTES) {
        process.stderr.write(`chaOS native host received invalid frame length ${size}\n`);
        void stop(3);
        return;
      }
      if (buffered.length < 4 + size) return;
      const body = buffered.subarray(4, 4 + size);
      buffered = buffered.subarray(4 + size);
      chain = chain.then(async () => {
        let value: unknown;
        try {
          value = JSON.parse(body.toString('utf8'));
        } catch {
          await writeResponse(failure('invalid', 'BAD_JSON', 'Request was not valid JSON'));
          return;
        }
        await writeResponse(await handleRequest(value));
      }).catch((error: unknown) => {
        const message = error instanceof Error ? error.message : String(error);
        process.stderr.write(`chaOS native host request failure: ${message}\n`);
      });
    }
  });

  process.stdin.once('end', () => { void chain.finally(() => stop(0)); });
  process.stdin.once('error', (error) => {
    process.stderr.write(`chaOS native host stdin error: ${error.message}\n`);
    void chain.finally(() => stop(4));
  });
  process.stdout.once('error', () => { void stop(5); });
  process.stdin.resume();
}
