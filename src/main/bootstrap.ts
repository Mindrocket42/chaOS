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

const origin = nativeMessagingOrigin();

if (origin) {
  void import('./native/host.js')
    .then(({ runNativeMessagingHost }) => runNativeMessagingHost(origin))
    .catch((error: unknown) => {
      const message = error instanceof Error ? error.stack ?? error.message : String(error);
      process.stderr.write(`chaOS native host failed to start: ${message}\n`);
      process.exitCode = 1;
    });
} else {
  void import('./index.js').catch((error: unknown) => {
    const message = error instanceof Error ? error.stack ?? error.message : String(error);
    process.stderr.write(`chaOS desktop app failed to start: ${message}\n`);
    process.exitCode = 1;
  });
}
