# chaOS

**ChatGPT decides. chaOS executes.**

chaOS is a deliberately small local I/O bridge for ChatGPT. Its primary browser path does not use MCP, an OpenAI Secure MCP Tunnel, `tunnel-client`, a localhost HTTP server, worker chats, Goal/Loop or a second orchestration runtime.

```text
ChatGPT in Chrome
       |
       | nonce-bound action / result envelopes
       v
chaOS extension
       |
       | Chrome Native Messaging
       v
chaos-host.exe
       |
       +-- approved files / search / writes
       +-- retained local processes
       +-- ordinary local programs and coding harnesses
```

ChatGPT remains the orchestrator. The native host exposes deterministic capabilities and returns receipts.

## Why this exists

The project began as a fork of Chat On Steroids. Current CoS contains substantial useful engineering, but its normal operating model also grew around multiple MCP connectors/tunnels, ChatGPT browser automation, durable conversation recording, worker chats, Goal/Loop and other machinery.

chaOS takes the opposite position:

1. **Direct capability beats delegation.** Read a file directly; do not create a worker to read it.
2. **Transport is replaceable.** MCP may exist as a compatibility adapter, but local capability does not depend on it.
3. **The user must be able to answer "what is running?"** Browser-native mode is one extension plus one native host. No hidden tunnel process exists.
4. **Access starts closed.** No roots or local capabilities are enabled by default.
5. **ChatGPT orchestrates.** Parallel work is multiple process handles, not a local swarm implementation.
6. **Durable work state belongs in the workspace.** Conversation recording is not the source of truth.

## Current browser-native proof

The Windows proof builds a standalone `chaos-host.exe` in Go. The host has no Electron or Node runtime dependency and has been tested using the same length-prefixed stdin/stdout framing used by Chrome Native Messaging.

Supported actions:

```text
health
fs.list
fs.read
fs.search
fs.write
fs.replace
proc.run
proc.read
proc.write
proc.stop
```

The file capabilities operate only through approved virtual roots such as `/project`. `health` exposes virtual root names, not host paths.

`command` is intentionally separate from the file boundary. When enabled, a shell command runs with the normal privileges of the logged-in OS account and is **not** represented as confined to approved roots.

## Browser orchestration

The extension supports two modes.

### Manual

ChatGPT emits one `CHAOS_ACTION_BEGIN` action. The extension displays **Run with chaOS** beside it and exposes the resulting receipt for copying back to the conversation.

### Automatic — explicit per-chat opt-in

Open the extension popup in the ChatGPT tab and choose **Enable automatic bridge in this chat**.

The extension then:

1. generates a random nonce for that tab;
2. sends ChatGPT the small action/result protocol;
3. accepts only actions carrying that nonce;
4. passes one action at a time to the native host;
5. sends the real receipt back to the same conversation;
6. stops naturally when ChatGPT answers without another action.

The opt-in disappears on page reload. An automatic session is capped at 50 actions and must then be explicitly re-enabled.

This browser code is transport glue. It is not allowed to become a worker scheduler, tab manager, model selector, recovery engine or agent framework.

## Local permissions

Fresh configuration:

```text
approved roots: none
read:            off
search:          off
write:           off
command:         off
```

The standalone host supports explicit local configuration:

```text
chaos-host.exe --status
chaos-host.exe --add-root project "C:\path\to\project"
chaos-host.exe --remove-root project
chaos-host.exe --set-capability read on
chaos-host.exe --set-capability search on
chaos-host.exe --set-capability write on
chaos-host.exe --set-capability command on
```

Register/unregister the Chrome native host:

```text
chaos-host.exe --register
chaos-host.exe --unregister
```

Registration is per-user on Windows and does not require an OpenAI connector or tunnel ID.

## Repository history

The previous chaOS fork state is preserved at:

```text
archive/chaos-core-first-2026-09-29
```

The synchronized current Chat On Steroids source is preserved at:

```text
upstream-sync-2026-10-16
```

The browser-native rewrite is being developed on:

```text
chaos-core-first-refresh
```

See `docs/CHAOS-REFRESH-2026-10-16.md` for the architecture and acceptance tests.

## Status

This is still a proof branch, not a release. The standalone Windows host and native-message framing are verified in CI. The next acceptance gate is a live Chrome test on a real Windows machine: register the proof host, load the unpacked extension, enable a ChatGPT tab, and complete a `health -> local action -> receipt -> next action` loop.

MIT licensed. The upstream CoS history remains available in the repository branches noted above.
