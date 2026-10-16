/** Browser <-> native-host contract. Deliberately tiny and transport-only. */

export const NATIVE_PROTOCOL = 1;
export const NATIVE_HOST_NAME = 'pro.ironhead.chaos';

/**
 * Stable ID for the unpacked chaOS extension. `extension-chaos/manifest.json` carries the
 * corresponding public key, so Chrome derives this same id on every machine.
 */
export const CHAOS_EXTENSION_ID = 'oneodmlppiiihbbligdiloibjagnfach';
export const CHAOS_EXTENSION_ORIGIN = `chrome-extension://${CHAOS_EXTENSION_ID}/`;

export interface NativeCall {
  type: 'call';
  id: string;
  action: string;
  args?: unknown;
}

export interface NativePing {
  type: 'ping';
  id: string;
}

export type NativeRequest = NativeCall | NativePing;

export interface NativeSuccess {
  type: 'result';
  id: string;
  ok: true;
  protocol: number;
  result: unknown;
}

export interface NativeFailure {
  type: 'result';
  id: string;
  ok: false;
  protocol: number;
  error: {
    code: string;
    message: string;
  };
}

export type NativeResponse = NativeSuccess | NativeFailure;

export function validRequestId(value: unknown): value is string {
  return typeof value === 'string' && /^[A-Za-z0-9._:-]{1,128}$/.test(value);
}
