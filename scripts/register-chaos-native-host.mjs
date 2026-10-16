#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const HOST = 'pro.ironhead.chaos';
const EXTENSION_ID = 'oneodmlppiiihbbligdiloibjagnfach';
const ORIGIN = `chrome-extension://${EXTENSION_ID}/`;
const REGISTRY_KEY = `HKCU\\Software\\Google\\Chrome\\NativeMessagingHosts\\${HOST}`;

function fail(message) {
  console.error(message);
  process.exit(1);
}

function reg(args) {
  const result = spawnSync('reg.exe', args, { encoding: 'utf8', windowsHide: true });
  if (result.status !== 0) {
    fail((result.stderr || result.stdout || `reg.exe failed with ${result.status}`).trim());
  }
}

if (process.platform !== 'win32') fail('This registration helper currently supports Windows only.');

const unregister = process.argv.includes('--unregister');
const appData = process.env.LOCALAPPDATA || path.join(os.homedir(), 'AppData', 'Local');
const manifestDir = path.join(appData, 'chaOS', 'native-messaging');
const manifestPath = path.join(manifestDir, `${HOST}.json`);

if (unregister) {
  spawnSync('reg.exe', ['delete', REGISTRY_KEY, '/f'], { encoding: 'utf8', windowsHide: true });
  rmSync(manifestPath, { force: true });
  console.log(`Removed ${HOST} from Chrome Native Messaging.`);
  process.exit(0);
}

const exeFlag = process.argv.indexOf('--exe');
const candidate = exeFlag >= 0 ? process.argv[exeFlag + 1] : process.execPath;
if (!candidate) fail('Usage: node scripts/register-chaos-native-host.mjs --exe "C:\\path\\to\\chaOS.exe"');
const executable = path.resolve(candidate);

mkdirSync(manifestDir, { recursive: true });
writeFileSync(
  manifestPath,
  `${JSON.stringify({
    name: HOST,
    description: 'chaOS local capability host for ChatGPT in Chrome',
    path: executable,
    type: 'stdio',
    allowed_origins: [ORIGIN]
  }, null, 2)}\n`,
  'utf8'
);

reg(['add', REGISTRY_KEY, '/ve', '/t', 'REG_SZ', '/d', manifestPath, '/f']);

console.log(`Registered ${HOST}`);
console.log(`Host manifest: ${manifestPath}`);
console.log(`Executable: ${executable}`);
console.log(`Allowed extension: ${EXTENSION_ID}`);
