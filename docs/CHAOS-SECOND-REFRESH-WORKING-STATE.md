# chaOS second refresh — working state

Date: 2026-10-30
Branch: `chaos-core-second-refresh`
Baseline: `chaos-core-first-refresh` at `1fff35f829dde23e20d33f1028908a153510c88d`

## Objective

Strengthen chaOS as an execution and evidence boundary without importing Chat On Steroids' product/runtime sprawl.

The second refresh keeps the first-refresh rule intact:

> ChatGPT decides. chaOS executes. Receipts prove what happened.

## Scope accepted from current Chat On Steroids

1. **Caller provenance** — local actions carry transport-generated attribution rather than model-supplied identity.
2. **Evidence-rich receipts** — successful mutations return machine-verifiable facts, not only `ok: true`.
3. **Fail-closed ambiguity** — when the bridge cannot establish local caller context, privileged actions do not silently proceed.
4. **Capability-scoped retained handles** — a retained process id is not authority by itself.
5. **Idempotency discipline** — duplicate action ids do not produce a second mutation inside one host lifetime.
6. **Capability fingerprinting** — health exposes a stable fingerprint of the currently granted local execution surface.
7. **Verification tests** — these invariants belong in the native-host suite.

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

Identity/replay violations also receive a receipt, so provenance does not disappear precisely when the caller boundary is rejected.

### C. Mutation evidence

Successful `fs.write` and `fs.replace` calls produce receipt evidence containing the canonical virtual path, resulting byte count and resulting SHA-256 digest. The existing action result remains unchanged, so this is additive evidence rather than a new mutation API.

### D. Privileged-call attribution gate

The native host refuses `fs.write`, `fs.replace`, `proc.run`, `proc.write` and `proc.stop` unless the call is attributed to a specific Chrome document in a ChatGPT tab on an allowed ChatGPT host. `proc.read` also requires attributed caller context because process output is part of the retained handle's authority surface. Health and ordinary filesystem reads/searches remain available according to their global local capability grants.

The gate is transport policy, not conversation identity. chaOS still does not infer or persist ChatGPT conversation ownership.

### E. Retained process ownership and recovery capability

A numeric `process_id` is no longer sufficient authority to operate a retained process.

- `proc.run` records the starting browser-document caller and mints a cryptographically random `process_token` when the OS RNG is available.
- The starting document can `proc.read`, `proc.write` or `proc.stop` without resupplying that token.
- A later/different ChatGPT browser document must provide the `process_token` returned by `proc.run`.
- A wrong token is rejected with `PROCESS_CAPABILITY_MISMATCH`; a missing cross-document token is rejected with `PROCESS_CAPABILITY_REQUIRED`; an unowned id is rejected with `PROCESS_OWNER_UNKNOWN`.
- If token generation fails, cross-document recovery stays closed rather than falling back to a weak token; same-document ownership still works.

This avoids both bad outcomes: a globally guessable process handle and a process permanently stranded merely because ChatGPT reloaded.

### F. Idempotency binding

The existing successful-call cache binds a request id to action + args + caller document context. Replaying the same successful call returns the original response/receipt; reusing the id from another tab/document is rejected rather than borrowing the prior result.

This guarantee is deliberately scoped to one native-host lifetime and is made explicit by the receipt's host-instance id. Durable cross-restart deduplication has not been added.

### G. Capability fingerprint

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
- a successful request id cannot be replayed from a different attributed browser caller;
- a retained process is directly usable by its starting browser document;
- another document is challenged for the process capability;
- a wrong process capability fails closed;
- the correct capability permits recovery from another document;
- an unknown process owner fails closed.

Existing tests still cover closed-by-default configuration, virtual-root path secrecy, path traversal rejection and Chrome Native Messaging framing.

## Working findings

- First refresh already starts closed: no roots and all capabilities disabled.
- Request ids already had an in-memory successful-call cache; second refresh strengthens its identity boundary rather than adding a scheduler or durable conversation store.
- Caller attribution can be generated entirely in the extension background layer from Chrome transport metadata; there is no reason to put conversation ownership into the capability kernel.
- Tab id is insufficient as provenance because tabs reload and are eventually reused; Chrome document identity is the useful immediate execution boundary.
- Long-lived retained handles need a separate bearer capability if they must survive a browser-document boundary.
- Evidence hashes belong in receipts, not in orchestration policy.
- The first-refresh branch still carries its earlier merge blockers: the Linux verification failure recorded in `PR_2_FAILURE_REPORT.MD` and the missing real Chrome end-to-end proof. This refresh does not redefine those as solved.

## Verification status

Implementation is complete for the bounded second-refresh slice, subject to executable proof.

Verified:

- `extension-chaos/background.js` parses cleanly with `node --check` under Node 22.16.0 using the final branch content.
- PR #4 is mergeable against `chaos-core-first-refresh` at the repository level; it remains draft because executable acceptance evidence is incomplete.

A temporary draft PR (#5) was opened against `main` solely to request the repository's default-branch pull-request workflows. GitHub exposed no workflow run for that PR/head during the verification attempt, so #5 was closed without claiming a CI pass. The actual review PR remains #4 against `chaos-core-first-refresh`.

Pending evidence before this branch can be treated as candidate truth:

1. `go test ./...` in `native-host` on CI or an environment holding the branch checkout;
2. the existing Windows Chrome-framing proof against this head;
3. real Chrome end-to-end proof that a ChatGPT-tab mutation receipt contains browser document identity, native-host instance, capability fingerprint and file SHA-256;
4. real retained-process proof across a page reload using the returned `process_token`;
5. inherited first-refresh Linux verification blocker resolved or explicitly retired by replacing that inherited verification path.
