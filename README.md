# chaOS

**ChatGPT decides. chaOS executes. Receipts prove what happened.**

chaOS is a small browser-to-local execution bridge for ChatGPT. Its current path is a Chrome extension plus a standalone Go native host. It does **not** require Electron, MCP, an OpenAI Secure MCP Tunnel, worker chats, Goal/Loop, or a second orchestration runtime.

> Current development line: `chaos-core-second-refresh`
>
> Canonical architecture: [`docs/CURRENT-ARCHITECTURE.md`](docs/CURRENT-ARCHITECTURE.md)

## Quick start — Windows browser-native build

From an existing checkout:

```text
git fetch origin && git switch chaos-core-second-refresh && git pull --ff-only origin chaos-core-second-refresh
```

Then run:

```text
TEST-BROWSER-NATIVE.cmd
```

The script:

1. checks that Go is available;
2. runs `go test ./...` in `native-host`;
3. builds `chaos-host.exe`;
4. registers it as the current Windows user's Chrome Native Messaging host;
5. prints the local chaOS configuration.

The source build requires **Go 1.23 or newer**. Browser-native chaOS does **not** require `npm ci`, `npm run dev`, Electron, or the inherited root npm dependency tree.

After the script passes:

1. open `chrome://extensions`;
2. enable **Developer mode**;
3. choose **Load unpacked**;
4. select this repository's `extension-chaos` folder;
5. open ChatGPT in Chrome;
6. open **chaOS Local Bridge**.

Fresh configuration starts closed:

```text
approved roots: none
read:            off
search:          off
write:           off
command:         off
```

A successful connection proves transport availability. It does **not** grant filesystem or command access.

To remove the native-host registration:

```text
chaos-host.exe --unregister
```

If `npm run dev` starts Vite/Electron, stop. That is inherited Chat On Steroids code, not the current chaOS runtime.

## Architecture

```text
ChatGPT in Chrome
       |
       | explicit action envelope
       v
extension-chaos
       |
       | Chrome Native Messaging
       v
chaos-host.exe
       |
       +-- approved virtual filesystem roots
       +-- retained local processes
       +-- ordinary local programs / coding harnesses
```

ChatGPT remains the orchestrator. The extension is transport glue. The Go host owns local capability enforcement and receipts.

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

### Filesystem boundary

File actions operate through approved virtual roots such as `/project`. `health` exposes virtual root names, not host paths.

Local grants are explicit:

```text
chaos-host.exe --status
chaos-host.exe --add-root project "C:\path\to\project"
chaos-host.exe --remove-root project
chaos-host.exe --set-capability read on
chaos-host.exe --set-capability search on
chaos-host.exe --set-capability write on
chaos-host.exe --set-capability command on
```

`command` is intentionally separate from the virtual-root boundary. When enabled, a shell command runs with the logged-in OS user's normal authority and is **not** represented as confined to approved roots.

## Provenance and receipts

The second refresh strengthens the execution boundary without adding an agent framework.

The extension derives caller provenance from Chrome transport metadata. Privileged calls are bound to a specific ChatGPT browser document rather than trusting model-supplied identity or a tab number alone.

Native receipts can include:

- receipt schema version;
- native-host instance id;
- request id and action;
- browser caller context, including Chrome document id;
- start/finish timestamps and duration;
- capability fingerprint;
- action-specific evidence.

Successful `fs.write` and `fs.replace` receipts include the resulting virtual path, byte count and SHA-256 digest.

Successful request ids are idempotent within one native-host lifetime: exact replay returns the original result; reuse with different args or caller context is rejected.

## Retained processes

A numeric `process_id` is not sufficient authority by itself.

`proc.run` records the browser document that started the retained process and, when the OS RNG is available, returns a random `process_token`.

- The starting document can read/write/stop its process directly.
- After a reload or other browser-document boundary, a later ChatGPT document must supply the returned `process_token` to recover that process handle.
- Missing, incorrect, or unknown ownership fails closed.

The token therefore preserves useful long-running process recovery without making a guessable integer a bearer credential.

## Browser modes

### Manual

ChatGPT emits one `CHAOS_ACTION_BEGIN` / `CHAOS_ACTION_END` action. The extension shows **Run with chaOS** and exposes the real result for copying back to the conversation.

### Automatic — explicit tab opt-in

Choose **Enable automatic bridge in this chat** from the extension popup.

The extension then:

1. creates a random nonce for that browser document;
2. sends ChatGPT the action/result protocol;
3. accepts only matching nonce-bound actions;
4. executes one action at a time;
5. returns the real receipt as the next user message;
6. stops naturally when ChatGPT answers without another action.

Automatic mode disappears on reload and is capped at 50 actions. It does not become a worker scheduler, tab manager, recovery engine, model selector, or local agent framework.

A retained OS process may survive a reload; use its `process_token` for explicit recovery from the new document.

## Why this exists

chaOS began as a fork of Chat On Steroids (CoS). CoS contains substantial engineering, but its product/runtime grew around Electron, MCP connectors/tunnels, durable conversation recording, browser automation, workers, Goal/Loop and other orchestration machinery.

chaOS takes a narrower position:

1. **Direct capability beats delegation.** Read/write/run directly instead of spawning a local worker to do it.
2. **Transport is replaceable.** Local authority must not depend on MCP or a tunnel.
3. **Execution must be inspectable.** One browser extension plus one native host should explain what is running.
4. **Access starts closed.** No approved roots or capabilities are silently granted.
5. **ChatGPT orchestrates.** chaOS does not duplicate the model's planning/runtime layer locally.
6. **Durable work state belongs in the workspace.** Conversation recording is not the execution source of truth.
7. **Receipts beat assertions.** Local mutation claims should carry evidence that can be checked.

## Current vs inherited repository content

The repository still contains upstream CoS history and source for provenance and cribbing. That material is **not** the current chaOS operating contract unless explicitly referenced by the current architecture.

Current authority:

```text
docs/CURRENT-ARCHITECTURE.md
AGENTS.md
README.md
native-host/
extension-chaos/
```

Historical/reference lines:

```text
archive/chaos-core-first-2026-09-29
upstream-sync-2026-10-16
chaos-core-first-refresh
```

The root `package.json`, Electron sources, old CoS docs and historical release machinery are upstream/reference material unless and until deliberately adopted into chaOS. Do not infer the current runtime from them.

## Development checks

Primary current checks:

```text
cd native-host && go test ./...
node --check extension-chaos/background.js
node --check extension-chaos/content.js
node --check extension-chaos/popup.js
```

Windows proof:

```text
TEST-BROWSER-NATIVE.cmd
```

The focused GitHub workflow is `.github/workflows/browser-native-proof.yml`.

## Verification status

This is still a proof line, not a release.

Verified:

- the first-refresh standalone native-host/framing path previously passed its focused proof;
- the final second-refresh `extension-chaos/background.js` parsed under Node 22.16.0 during the refresh work.

Still required before the second refresh should be promoted from draft:

1. `go test ./...` against the current second-refresh head;
2. Windows Chrome Native Messaging framing proof against the current head;
3. a live Chrome mutation receipt proving browser document identity, host instance, capability fingerprint and file SHA-256;
4. retained-process recovery across a page reload using the returned `process_token`.

Do not treat tests-written, static review, or first-refresh proof as evidence that these second-refresh runtime gates have passed.

See [`docs/CHAOS-SECOND-REFRESH-WORKING-STATE.md`](docs/CHAOS-SECOND-REFRESH-WORKING-STATE.md) for the implementation/audit record.

MIT licensed.
