import { describe, expect, it } from 'vitest';
import {
  CHAOS_EXTENSION_ID,
  CHAOS_EXTENSION_ORIGIN,
  NATIVE_HOST_NAME,
  NATIVE_PROTOCOL,
  validRequestId
} from '../src/main/native/protocol.js';

describe('chaOS native messaging protocol', () => {
  it('keeps the host and extension identities stable', () => {
    expect(NATIVE_PROTOCOL).toBe(1);
    expect(NATIVE_HOST_NAME).toBe('pro.ironhead.chaos');
    expect(CHAOS_EXTENSION_ID).toBe('oneodmlppiiihbbligdiloibjagnfach');
    expect(CHAOS_EXTENSION_ORIGIN).toBe('chrome-extension://oneodmlppiiihbbligdiloibjagnfach/');
  });

  it('accepts bounded receipt ids and rejects values unsafe for the wire contract', () => {
    expect(validRequestId('health-1')).toBe(true);
    expect(validRequestId('job:17.read_2')).toBe(true);
    expect(validRequestId('')).toBe(false);
    expect(validRequestId('contains spaces')).toBe(false);
    expect(validRequestId('../escape')).toBe(false);
    expect(validRequestId('x'.repeat(129))).toBe(false);
    expect(validRequestId(17)).toBe(false);
  });
});
