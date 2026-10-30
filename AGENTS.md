# chaOS repository instructions

Read [`docs/CURRENT-ARCHITECTURE.md`](docs/CURRENT-ARCHITECTURE.md) before changing this repository.

The current product is **chaOS**, not Chat On Steroids.

> ChatGPT decides. chaOS executes. Receipts prove what happened.

The synchronized CoS source remains available on `upstream-sync-2026-10-16` and in repository history. Do not treat inherited Electron/MCP/worker/Goal/Loop code or documents as the current chaOS contract unless a current chaOS document explicitly adopts them.

## 1. Current authority order

When sources disagree, use this order:

1. `docs/CURRENT-ARCHITECTURE.md`
2. this file
3. `README.md`
4. current code in `native-host/` and `extension-chaos/`
5. `docs/CHAOS-SECOND-REFRESH-WORKING-STATE.md` as an implementation/audit record
6. inherited CoS files only as historical/reference material

Do not let an old worklog, Electron source, root npm script, release workflow, or CoS document silently override the current browser-native architecture.

## 2. Product boundary

Current runtime:

```text
ChatGPT in Chrome
  -> extension-chaos
  -> Chrome Native Messaging
  -> native-host (Go)
  -> approved files / retained processes / local programs
```

ChatGPT is the orchestrator. chaOS supplies deterministic local capabilities and evidence.

Do not introduce a second orchestration runtime into chaOS core.

Explicit non-goals unless a new architectural decision says otherwise:

- Goal / Loop;
- worker chats or worker lifecycle;
- automatic Continue/turn rotation;
- conversation recording as execution state;
- browser tab scheduling/recovery machinery;
- local model routing;
- a local agent framework;
- MCP or an OpenAI tunnel as a required transport;
- Electron as a required runtime.

MCP may become an optional adapter. It must not become the authority owner for local execution.

## 3. Core invariants

### Closed by default

Fresh local configuration has:

```text
approved roots: none
read:            off
search:          off
write:           off
command:         off
```

Do not widen this implicitly.

### One authority owner

The Go native host owns local capability enforcement. The extension transports requests and derives browser caller provenance. ChatGPT decides what action to request.

Do not mirror permission or execution authority in another state machine.

### Caller provenance

Privileged local actions must use transport-generated caller context. Model text must never be able to choose or overwrite browser caller identity.

Chrome document identity matters. A tab id alone is not durable provenance across reload/navigation.

### Receipts are evidence

Do not equate `ok: true`, a button click, request dispatch, or expected side effect with proof.

Receipts should identify what executed and carry action-specific evidence where practical. Current file mutations include post-action SHA-256 evidence.

### Idempotency

A successful request id is bound to action + args + caller context for one native-host lifetime. Exact replay may return the original receipt. Reuse for a different request/caller must fail closed.

Do not claim durable cross-restart idempotency unless it is actually implemented and verified.

### Retained process authority

`process_id` is not a bearer credential.

`proc.run` associates a retained process with its starting browser document and returns a random `process_token` when available. A different/later browser document needs that token to recover the handle.

Do not regress this into globally guessable process ids. Do not bind processes so rigidly to one document that a reload strands them with no explicit recovery path.

### Command scope

`command` is intentionally powerful. When enabled, commands run with the logged-in OS user's normal authority. Approved virtual roots do not sandbox arbitrary command execution.

Never describe command authority as root-confined.

## 4. Change discipline

Prefer deletion and simplification over adding another scheduler, fallback, watcher, timer, retry loop, or mirrored state owner.

Repair the earliest wrong authority/identity boundary instead of compensating downstream.

For every meaningful change:

1. state the invariant being changed or preserved;
2. add/adjust deterministic tests where practical;
3. verify the narrowest relevant path;
4. update `docs/CURRENT-ARCHITECTURE.md` when the contract changes;
5. update README when setup, user operation, current branch, or verification status changes;
6. record verification honestly—written tests are not passed tests.

Do not manufacture success from unavailable CI or an unrun local toolchain.

## 5. Current development path

Current branch:

```text
chaos-core-second-refresh
```

Browser-native checks:

```text
cd native-host && go test ./...
node --check extension-chaos/background.js
node --check extension-chaos/content.js
node --check extension-chaos/popup.js
```

Windows bootstrap/proof:

```text
TEST-BROWSER-NATIVE.cmd
```

Do **not** run `npm ci` or `npm run dev` as the browser-native chaOS test path. Those commands belong to inherited CoS/Electron code unless a task explicitly targets that legacy surface.

## 6. Repository history and inherited material

Preserved history/reference branches:

```text
archive/chaos-core-first-2026-09-29
upstream-sync-2026-10-16
chaos-core-first-refresh
```

Large historical docs and upstream source may be useful for cribbing implementation ideas. Extract principles and mechanisms that fit chaOS; do not wholesale-import CoS product/runtime sprawl.

If an inherited file is actively misleading current development, either:

- replace it with current chaOS guidance;
- mark it explicitly historical/reference-only; or
- remove it from the current line when history/upstream branches already preserve it.

## 7. Security/privacy discipline

This is a public repository.

Never commit:

- private local paths containing real usernames;
- API keys, tokens, cookies, credentials, connector URLs, tunnel secrets or private session URLs;
- real private conversation contents unless deliberately supplied for a public fixture;
- hidden provenance/session trailers from coding tools.

Use placeholders in docs/tests.

Local capability changes deserve security review because chaOS deliberately sits between browser/model intent and the logged-in OS account.

## 8. Verification state

The second-refresh architecture is implemented but not fully runtime-verified.

Known outstanding acceptance evidence is listed in `docs/CURRENT-ARCHITECTURE.md` and `README.md`. Keep those documents current whenever a gate passes, fails, or is retired.

Do not promote the branch from draft based only on static review.
