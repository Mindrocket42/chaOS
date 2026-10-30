# chaOS second refresh — working state

Date: 2026-10-30
Branch: `chaos-core-second-refresh`
Baseline: `chaos-core-first-refresh` at `1fff35f829dde23e20d33f1028908a153510c88d`

## Objective

Strengthen chaOS as an execution and evidence boundary without importing Chat On Steroids' product/runtime sprawl.

The second refresh keeps the first-refresh rule intact:

> ChatGPT decides. chaOS executes. Receipts prove what happened.

## Scope accepted from current Chat On Steroids

1. **Caller provenance** — local actions should carry transport-generated attribution rather than model-supplied identity.
2. **Evidence-rich receipts** — successful mutations should return machine-verifiable facts, not only `ok: true`.
3. **Fail-closed ambiguity** — when the bridge cannot establish the action's local caller context, mutating actions should not silently proceed.
4. **Idempotency discipline** — duplicate action ids must not produce a second mutation inside one host lifetime.
5. **Capability fingerprinting** — health should expose a stable fingerprint of the currently granted local execution surface so ChatGPT can detect permission/root changes.
6. **Verification tests** — the above invariants belong in the native-host test suite.

## Explicit exclusions

Do not import:

- Goal / Loop;
- worker chats or worker lifecycle;
- automatic Continue / turn rotation;
- ChatGPT tab scheduling or recovery;
- conversation recording as execution state;
- a local agent framework;
- UI/product features unrelated to the capability kernel;
- browser DOM state as the source of truth for local execution.

## Implemented

### A. Transport-generated caller context

`extension-chaos/background.js` derives caller context from Chrome's `sender` object and forwards only that generated context to the native host. The model-facing action envelope still contains only action id, action name and args; model output cannot choose or overwrite native caller attribution.

Current context:

- surface (`chatgpt-tab` or `extension-page`);
- tab id when Chrome supplies one;
- frame id;
- **Chrome document id**, so a reload/new document in the same tab is a different caller;
- host classification (`chatgpt.com` / `chat.openai.com`), never the raw page URL.

Mutating calls require the document id as well as the ChatGPT host classification. A tab id alone is not treated as durable caller identity.

The extension also rejects outstanding bridge calls immediately if the native messaging port disconnects instead of leaving them to expire on the normal timeout.

### B. Native receipt envelope

`native-host/evidence.go` and `native-host/main.go` attach receipt metadata to completed calls:

- receipt schema version;
- ephemeral **host-instance id**, identifying the native-host lifetime that executed the call;
- action;
- request id;
- transport-generated caller context, including browser document id;
- started/finished timestamps;
- duration;
- capability fingerprint;
- action-specific evidence where available.

`health` exposes the same host-instance id, receipt schema and capability fingerprint, allowing later receipts to be tied to the authority surface and host lifetime observed before execution.

### C. Mutation evidence

Successful `fs.write` and `fs.replace` calls produce receipt evidence containing the canonical virtual path, resulting byte count and resulting SHA-256 digest. The existing action result remains unchanged, so this is additive evidence rather than a new mutation API.

### D. Mutating-call attribution gate

The native host refuses `fs.write`, `fs.replace`, `proc.run`, `proc.write` and `proc.stop` unless the call is attributed to a specific Chrome document in a ChatGPT tab on an allowed ChatGPT host. Health/read/search remain usable for diagnostics without widening mutation authority.

The gate is transport policy, not conversation identity. chaOS still does not infer or persist ChatGPT conversation ownership.

### E. Idempotency binding

The existing successful-call cache binds a request id to action + args + caller document context. Replaying the same successful call returns the original response/receipt; reusing the id from another tab/document is rejected rather than borrowing the prior result.

This guarantee is deliberately scoped to one native-host lifetime and is made explicit by the receipt's host-instance id. Durable cross-restart deduplication has not been added.

### F. Capability fingerprint

The health response includes a SHA-256 capability fingerprint derived from host version, config version, approved roots and capability grants. The actual root paths are inputs to the digest but are not emitted by health.

Equivalent authority configurations produce the same fingerprint regardless of root ordering.

## Tests added

`native-host/main_test.go` and `native-host/evidence_test.go` cover:

- capability fingerprint changes when the authority surface changes;
- equivalent authority surfaces produce the same fingerprint;
- mutating calls without caller attribution fail closed and do not touch disk;
- a nominal ChatGPT tab caller without Chrome document identity also fails closed;
- write receipts include browser document identity, native-host lifetime, capability fingerprint and resulting SHA-256;
- duplicate successful mutation ids return the original receipt and do not execute the mutation again;
- a successful request id cannot be replayed from a different attributed browser caller.

Existing tests still cover closed-by-default configuration, virtual-root path secrecy, path traversal rejection and Chrome Native Messaging framing.

## Working findings

- First refresh already starts closed: no roots and all capabilities disabled.
- Request ids already had an in-memory successful-call cache; second refresh strengthens its identity boundary rather than adding a scheduler or durable conversation store.
- Caller attribution can be generated entirely in the extension background layer from Chrome transport metadata; there is no reason to put conversation ownership into the capability kernel.
- Tab id is insufficient as provenance because tabs reload and are eventually reused; Chrome document identity is the useful local execution boundary.
- Evidence hashes belong in receipts, not in orchestration policy.
- The first-refresh branch still carries its earlier merge blockers: the Linux verification failure recorded in `PR_2_FAILURE_REPORT.MD` and the missing real Chrome end-to-end proof. This refresh does not redefine those as solved.

## Verification status

Implementation is complete for the bounded second-refresh slice, subject to executable proof.

Pending evidence before this branch can be treated as candidate truth:

1. `go test ./...` in `native-host` on CI or a local Go toolchain;
2. `node --check extension-chaos/background.js`;
3. the existing Windows Chrome-framing proof;
4. real Chrome end-to-end proof that a ChatGPT-tab mutation receipt contains browser document identity, native-host instance, capability fingerprint and file SHA-256;
5. inherited first-refresh Linux verification blocker resolved or explicitly retired by replacing that inherited verification path.
