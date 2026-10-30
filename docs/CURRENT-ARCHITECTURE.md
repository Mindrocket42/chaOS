# chaOS current architecture

Updated: 2026-10-30
Current development branch: `chaos-core-second-refresh`
Baseline: `chaos-core-first-refresh` at `1fff35f829dde23e20d33f1028908a153510c88d`

## Product invariant

> ChatGPT decides. chaOS executes. Receipts prove what happened.

chaOS is a deliberately small browser-to-local execution bridge for ChatGPT. It is not a local agent runtime, worker scheduler, conversation recorder, model router or desktop shell.

The primary runtime is:

```text
ChatGPT in Chrome
       |
       | explicit action envelope
       v
extension-chaos
       |
       | Chrome Native Messaging
       v
chaos-host (Go)
       |
       +-- approved virtual filesystem roots
       +-- retained local processes
       +-- ordinary local programs / coding harnesses
```

The browser extension is transport glue. The Go host owns local capability enforcement and evidence receipts. ChatGPT remains the orchestrator.

## Current source of truth

For current chaOS behavior, use these files in this order:

1. `docs/CURRENT-ARCHITECTURE.md` — product/runtime contract.
2. `AGENTS.md` — repository working rules for agents and contributors.
3. `README.md` — setup, use and current verification status.
4. `native-host/` — local capability kernel.
5. `extension-chaos/` — Chrome transport and per-browser-document opt-in.
6. `docs/CHAOS-SECOND-REFRESH-WORKING-STATE.md` — implementation/audit record for the second refresh.

Older CoS documents, Electron sources and root npm metadata remain provenance/upstream reference only unless explicitly adopted here. They do not override the files above.

The synchronized CoS source is preserved on `upstream-sync-2026-10-16`; the pre-refresh chaOS line is preserved on `archive/chaos-core-first-2026-09-29`.

## Supported actions

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

- Filesystem actions use approved virtual roots such as `/project`.
- Fresh configuration has no approved roots and all capabilities disabled.
- `health` exposes virtual root names, not host paths.
- `read`, `search`, `write` and `command` are separate explicit local grants.
- `command` is intentionally not represented as root-confined: when enabled, commands run with the logged-in OS user's normal authority.

### Caller provenance

The extension background service derives caller context from Chrome transport metadata. Model output cannot choose or overwrite it.

For privileged calls the host requires:

- a ChatGPT browser surface;
- tab id;
- frame id;
- Chrome document id;
- an allowed ChatGPT host classification.

Document id matters because a reload creates a new browser document even if the tab id remains the same.

### Receipts

Completed native calls carry a receipt containing, where applicable:

- receipt schema version;
- native-host instance id;
- request id and action;
- caller context;
- start/finish timestamps and duration;
- capability fingerprint;
- action-specific evidence.

Successful `fs.write` and `fs.replace` receipts include the resulting virtual path, byte count and SHA-256 digest.

The capability fingerprint is derived from the host/config version, approved roots and current grants. It lets the caller detect that the local authority surface changed without exposing host paths in `health`.

### Idempotency

Successful request ids are cached for one native-host lifetime and bound to action + args + caller context.

- Exact replay returns the original response/receipt.
- Reusing a successful id with different args or caller context is rejected.
- This is not durable cross-restart deduplication.

### Retained process handles

A numeric `process_id` is not sufficient authority by itself.

- `proc.run` records the starting browser document and returns a random `process_token` when the OS RNG is available.
- The starting document can operate its retained process directly.
- A later/different browser document must supply the returned `process_token` to recover that handle after a reload/navigation boundary.
- Missing, incorrect or unknown process ownership fails closed.

This makes long-lived process recovery possible without turning a guessable integer into a bearer credential.

## Browser modes

### Manual

ChatGPT emits one `CHAOS_ACTION_BEGIN` / `CHAOS_ACTION_END` envelope. The extension shows **Run with chaOS**. The user runs it and copies the resulting receipt back to the conversation.

### Automatic — explicit browser-document opt-in

The popup can enable the bridge for the current ChatGPT browser document. The extension generates a random nonce, instructs ChatGPT to emit one matching action at a time, executes it, and returns the receipt as the next user message.

Automatic mode:

- is opt-in per page/document;
- disappears on reload;
- has a 50-action budget;
- stops when ChatGPT answers without another action;
- does not create workers, manage tabs, choose models, recover conversations or run a second orchestration loop.

A retained OS process may outlive that browser document; its `process_token` is the explicit recovery capability for a later document.

## Explicit non-goals

Do not reintroduce these into chaOS core without a new architectural decision:

- Goal / Loop;
- worker chats or worker lifecycle;
- automatic Continue/turn rotation;
- durable conversation recording as execution state;
- browser tab scheduling/recovery machinery;
- local model routing;
- a local agent framework;
- MCP or an OpenAI tunnel as a required transport;
- Electron as a required runtime.

MCP may exist in future as an optional adapter. It must not become the authority owner for local execution.

## Development and verification

Browser-native source work requires Go 1.23+ and Node only for JavaScript syntax/proof scripts. It does not require `npm ci`, Electron or the inherited root npm dependency tree.

Primary checks:

```text
cd native-host && go test ./... && go vet ./...
node --check extension-chaos/background.js
node --check extension-chaos/content.js
node --check extension-chaos/popup.js
```

On Windows, `TEST-BROWSER-NATIVE.cmd` performs extension syntax checks when Node is present, runs Go tests/vet, builds `chaos-host.exe`, registers the per-user Chrome Native Messaging host, shows the closed-by-default configuration and prints the remaining live proof steps.

Active GitHub workflows:

```text
.github/workflows/ci.yml
.github/workflows/browser-native-proof.yml
.github/workflows/codeql.yml
```

## Current verification state

Verified during the 2026-10-30 housekeeping run:

- current `extension-chaos/background.js`, `content.js` and `popup.js` contents fetched from `chaos-core-second-refresh` all passed `node --check` under Node 22.16.0;
- the active workflow directory contains only the three chaOS workflows listed above.

Previously verified:

- the first-refresh standalone native-host/framing path passed its focused proof.

Still required before the second refresh is promoted from draft:

1. `go test ./...` and `go vet ./...` against the current second-refresh native host;
2. Windows Chrome Native Messaging framing proof against the current head;
3. live Chrome mutation receipt proving document identity, host instance, capability fingerprint and file SHA-256;
4. retained-process recovery across a page reload using `process_token`.

The connected execution container has Go 1.23.2 and Node 22.16.0 but cannot resolve github.com, so it cannot clone the branch for an exact Go run. GitHub Actions also exposed no current PR-head run through the connected API. Those receipts remain pending rather than inferred.

Do not convert tests-written, static review, or historical first-refresh proof into a claim that the second-refresh runtime has passed these gates.
