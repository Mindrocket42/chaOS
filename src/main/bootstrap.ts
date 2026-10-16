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
  const { runNativeMessagingHost } = await import('./native/host.js');
  await runNativeMessagingHost(origin);
} else {
  await import('./index.js');
}
