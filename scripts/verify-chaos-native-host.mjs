#!/usr/bin/env node

import { spawn } from 'node:child_process';
import path from 'node:path';

const executable = process.argv[2];
if (!executable) {
  console.error('Usage: node scripts/verify-chaos-native-host.mjs <packaged-executable>');
  process.exit(2);
}

const extensionId = 'oneodmlppiiihbbligdiloibjagnfach';
const origin = `chrome-extension://${extensionId}/`;
const request = Buffer.from(JSON.stringify({ type: 'ping', id: 'wire-proof' }), 'utf8');
const frame = Buffer.allocUnsafe(4 + request.length);
frame.writeUInt32LE(request.length, 0);
request.copy(frame, 4);

const child = spawn(path.resolve(executable), [origin, '--parent-window=0'], {
  windowsHide: true,
  stdio: ['pipe', 'pipe', 'pipe']
});

const stdout = [];
const stderr = [];
child.stdout.on('data', (chunk) => stdout.push(Buffer.from(chunk)));
child.stderr.on('data', (chunk) => stderr.push(Buffer.from(chunk)));

const timeout = setTimeout(() => {
  child.kill();
  console.error('Native host did not exit after stdin closed.');
  process.exitCode = 1;
}, 15000);

child.stdin.end(frame);

child.once('error', (error) => {
  clearTimeout(timeout);
  console.error(`Could not start native host: ${error.message}`);
  process.exitCode = 1;
});

child.once('close', (code, signal) => {
  clearTimeout(timeout);
  const output = Buffer.concat(stdout);
  const errors = Buffer.concat(stderr).toString('utf8').trim();
  const diagnostic = `exit=${code} signal=${signal ?? 'none'} stdout_hex=${output.toString('hex') || '<empty>'} stderr=${JSON.stringify(errors)}`;
  if (output.length < 4) {
    console.error(`Native host produced ${output.length} stdout bytes; expected a framed response. ${diagnostic}`);
    process.exitCode = 1;
    return;
  }
  const size = output.readUInt32LE(0);
  if (size < 2 || size > 1024 * 1024 || output.length !== size + 4) {
    console.error(`Native host returned an invalid frame: declared ${size} bytes, received ${output.length - 4}. ${diagnostic}`);
    process.exitCode = 1;
    return;
  }
  let response;
  try {
    response = JSON.parse(output.subarray(4).toString('utf8'));
  } catch (error) {
    console.error(`Native host response was not JSON: ${error.message}. ${diagnostic}`);
    process.exitCode = 1;
    return;
  }
  if (response?.type !== 'result' || response?.id !== 'wire-proof' || response?.ok !== true || response?.protocol !== 1) {
    console.error(`Unexpected native host response: ${JSON.stringify(response)}. ${diagnostic}`);
    process.exitCode = 1;
    return;
  }
  if (code !== 0 || signal) {
    console.error(`Native host answered correctly but exited abnormally. ${diagnostic}`);
    process.exitCode = 1;
    return;
  }
  console.log(`Native messaging wire proof passed: ${JSON.stringify(response)}`);
  if (errors) console.log(`Native host stderr: ${errors}`);
});
