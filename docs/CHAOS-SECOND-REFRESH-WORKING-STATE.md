# chaOS second refresh — working state

Date: 2026-10-30
Branch: `chaos-core-second-refresh`
Baseline: `chaos-core-first-refresh` at `1fff35f829dde23e20d33f1028908a153510c88d`
Canonical current contract: `docs/CURRENT-ARCHITECTURE.md`

## Objective

Strengthen chaOS as an execution and evidence boundary without importing Chat On Steroids' product/runtime sprawl.

> ChatGPT decides. chaOS executes. Receipts prove what happened.

## Second-refresh mechanisms

1. **Transport-generated caller provenance** — local privileged actions carry Chrome-derived caller context rather than model-supplied identity.
2. **Browser-document identity** — tab id alone is not treated as durable provenance across reload/navigation.
3. **Evidence-rich receipts** — completed calls identify the native-host lifetime, caller, action and authority surface; file mutations include post-action SHA-256 evidence.
4. **Fail-closed privileged calls** — missing/ambiguous caller context does not silently mutate local state.
5. **Caller-bound idempotency** — successful request ids cannot be reused across different args or caller contexts within one host lifetime.
6. **Capability fingerprinting** — `health` exposes a stable digest of the local execution authority surface without exposing host root paths.
7. **Retained-process capability recovery** — `process_id` is not authority by itself; cross-document recovery uses a random `process_token`.
8. **Immediate disconnect failure** — outstanding bridge calls reject when the native-messaging port disconnects rather than waiting for ordinary timeout.

## Explicit exclusions

The second refresh does not import:

- Goal / Loop;
- worker chats or worker lifecycle;
- automatic Continue / turn rotation;
- durable conversation recording as execution state;
- ChatGPT tab scheduling/recovery machinery;
- a local model router;
- a local agent framework;
- MCP or an OpenAI tunnel as a required transport;
- Electron as a required runtime.

## Implementation detail

### Caller context

`extension-chaos/background.js` derives caller context from Chrome's `sender` metadata.

Current context contains:

- surface (`chatgpt-tab` or `extension-page`);
- tab id;
- frame id;
- Chrome document id;
- allowed host classification (`chatgpt.com` / `chat.openai.com`).

The model-facing action contains action id, action name and args; it cannot choose or overwrite native caller attribution.

### Receipts

`native-host/evidence.go` and `native-host/main.go` attach:

- receipt schema version;
- ephemeral native-host instance id;
- action and request id;
- caller context;
- start/finish timestamps and duration;
- capability fingerprint;
- action-specific evidence where applicable.

Identity/replay violations also receive receipts.

### File mutation evidence

Successful `fs.write` and `fs.replace` calls include:

- canonical virtual path;
- resulting byte count;
- resulting SHA-256 digest.

### Retained process ownership

`proc.run` associates a retained process with the starting browser document and returns a random `process_token` when OS randomness is available.

- Starting document: may operate the process directly.
- Different/later document: must present the token.
- Missing token: `PROCESS_CAPABILITY_REQUIRED`.
- Wrong token: `PROCESS_CAPABILITY_MISMATCH`.
- Unknown owner: `PROCESS_OWNER_UNKNOWN`.
- RNG failure: same-document ownership still works; cross-document recovery remains closed.

This prevents a guessable numeric process id from being a bearer credential while avoiding the opposite failure mode of stranding a live process after a page reload.

## Tests added

`native-host/main_test.go` and `native-host/evidence_test.go` cover:

- closed-by-default configuration;
- virtual-root path secrecy;
- path traversal rejection;
- Chrome Native Messaging framing;
- capability fingerprint changes when the authority surface changes;
- equivalent authority configurations producing the same fingerprint;
- unattributed mutation failure without disk side effects;
- rejection of nominal ChatGPT caller context without document identity;
- write receipt identity/fingerprint/SHA-256 evidence;
- exact successful mutation replay without re-execution;
- request-id rejection across caller boundaries;
- starting-document retained-process ownership;
- cross-document process capability challenge;
- wrong process capability rejection;
- valid process capability recovery;
- unknown process-owner failure.

## Housekeeping pass — 2026-10-30

The second-refresh line was reconciled so repository metadata and automation describe the product that actually exists.

### Canonical documentation

Added `docs/CURRENT-ARCHITECTURE.md` and made it the current architecture contract.

Updated:

- `README.md` — current branch, setup, runtime, receipts, process tokens and verification status;
- `AGENTS.md` — replaced the 452 KB inherited CoS agent/product map with concise current chaOS repository rules;
- `CLAUDE.md` — current browser-native development rules;
- `CONTRIBUTING.md` — chaOS contribution/test/evidence requirements;
- `SECURITY.md` — current chaOS capability/provenance/process security model.

The historical first-refresh `PR_2_FAILURE_REPORT.MD` was moved to `docs/history/PR-2-FIRST-REFRESH-FAILURE-REPORT.md` and explicitly marked historical.

### Browser instructions

Updated `extension-chaos/content.js` and `extension-chaos/popup.js` so manual/automatic instructions tell ChatGPT to preserve and use `process_token` after a browser-document boundary.

### GitHub intake

Updated:

- `.github/pull_request_template.md`;
- `.github/ISSUE_TEMPLATE/bug_report.yml`;
- `.github/ISSUE_TEMPLATE/feature_request.yml`;
- `.github/ISSUE_TEMPLATE/config.yml`.

The security contact now points at `Mindrocket42/chaOS`, not upstream CoS.

### CI and automation

Replaced inherited CoS `ci.yml` with focused chaOS checks:

- extension JS syntax checks;
- `go test ./...`;
- `go vet ./...`;
- Linux Go build;
- Windows Go tests/build;
- Windows Chrome Native Messaging framing proof.

Retargeted CodeQL to the current extension JavaScript.

Dependabot now updates GitHub Actions only; it no longer maintains the inherited CoS npm dependency tree.

Removed active inherited CoS automation for:

- canary publishing;
- release-note drafting;
- Firefox signing;
- app-version issue enforcement;
- needs-info automation;
- old PR checklist/fail-first logic;
- PR auto-drafting/triage;
- publish/release/release verification;
- release checklist/targets;
- similar-issue automation;
- waiting-PR cleanup;
- weekly digest;
- welcome automation.

The current workflow directory now contains only:

```text
.github/workflows/ci.yml
.github/workflows/browser-native-proof.yml
.github/workflows/codeql.yml
```

## Working findings

- The first refresh already established the correct minimal runtime direction: Chrome extension + Go native host, closed by default.
- The second refresh strengthens authority and evidence rather than adding orchestration.
- Browser document identity is the immediate caller boundary; long-lived process handles require a separate recovery capability.
- Evidence belongs in receipts; orchestration policy does not substitute for post-action evidence.
- Inherited CoS files may remain useful for archaeology/cribbing, but current root guidance and executable automation must not present them as the active product.

## Verification status

### Verified

- `extension-chaos/background.js` parsed cleanly under Node 22.16.0 during the second-refresh implementation work before the housekeeping pass.
- The repository-level branch/PR structure exists and PR #4 targets `chaos-core-first-refresh`.
- The active workflow directory has been reduced to the three chaOS workflows listed above.

### Requires re-verification after housekeeping

Because `content.js`, `popup.js`, CI configuration and documentation changed during housekeeping, the final second-refresh head still needs fresh executable evidence before promotion:

1. `node --check` against all three final extension JavaScript files;
2. `go test ./...` against the final `native-host` head;
3. `go vet ./...` and a current build;
4. Windows Chrome Native Messaging framing proof against the final head;
5. live Chrome mutation receipt showing document identity, native-host instance, capability fingerprint and file SHA-256;
6. live retained-process recovery across page reload using `process_token`.

The old first-refresh Linux CoS verification failure is no longer a current chaOS gate because the inherited CoS CI path was retired from this line. Its historical record remains under `docs/history/`.

Do not promote the branch based on static review alone.
