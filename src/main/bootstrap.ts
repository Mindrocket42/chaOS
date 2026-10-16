/**
 * Process entrypoint.
 *
 * Normal launches load the Electron desktop app. Chrome Native Messaging launches the same
 * executable with the calling extension origin as argv[1]; those launches run only the native
 * bridge and never acquire the app's single-instance lock, create windows, start tunnels or
 * touch browser automation.
 */

const CHROME_EXTENSION_ORIGIN = /^chrome-extension:\/\/[a-p]{32}\/$/;

export function nativeMessagingOrigin(argv: readonly string[] = process.argv): string | null {
  return argv.find((arg) => CHROME_EXTENSION_ORIGIN.test(arg)) ?? null;
}

function fail(label: string, error: unknown): void {
  const message = error instanceof Error ? error.stack ?? error.message : String(error);
  process.stderr.write(`${label}: ${message}\n`);
  process.exitCode = 1;
}

const origin = nativeMessagingOrigin();
const registerHost = process.argv.includes('--register-chaos-native-host');
const unregisterHost = process.argv.includes('--unregister-chaos-native-host');

if (registerHost || unregisterHost) {
  void import('./native/register.js')
    .then(({ registerNativeMessagingHost, unregisterNativeMessagingHost }) => {
      if (registerHost) {
        const result = registerNativeMessagingHost();
        process.stdout.write(`chaOS Chrome native host registered.\nManifest: ${result.manifest}\nExecutable: ${result.executable}\n`);
      } else {
        unregisterNativeMessagingHost();
        process.stdout.write('chaOS Chrome native host unregistered.\n');
      }
    })
    .catch((error: unknown) => fail('chaOS native-host registration failed', error));
} else if (origin) {
  void import('./native/host.js')
    .then(({ runNativeMessagingHost }) => runNativeMessagingHost(origin))
    .catch((error: unknown) => fail('chaOS native host failed to start', error));
} else {
  void import('./index.js').catch((error: unknown) => fail('chaOS desktop app failed to start', error));
}
