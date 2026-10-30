# chaOS changelog

chaOS is currently a proof-stage project rather than a versioned release line.

This file records the chaOS divergence and current architectural changes. The full inherited Chat On Steroids release changelog remains available on the synchronized upstream branch:

```text
upstream-sync-2026-10-16
```

and in repository history.

## 2026-10-30 — Second core refresh and repository housekeeping

Development branch: `chaos-core-second-refresh`

### Execution/evidence boundary

- Added transport-generated caller provenance from Chrome sender metadata.
- Bound privileged caller identity to the Chrome browser document, not only a tab id.
- Added native receipt envelopes with request/action identity, browser caller context, native-host instance, timing and capability fingerprint.
- Added post-action SHA-256 evidence for `fs.write` and `fs.replace`.
- Strengthened successful request-id replay so it is bound to action + args + caller context within one native-host lifetime.
- Added fail-closed caller gates for privileged mutations.
- Made pending extension calls reject immediately when the native-messaging connection disconnects.

### Retained-process authority

- Stopped treating numeric `process_id` as sufficient authority across browser documents.
- `proc.run` records the starting browser document and returns a random `process_token` when OS randomness is available.
- The starting browser document can operate its retained process directly.
- A later browser document can recover the retained handle by presenting the returned `process_token`.
- Missing, wrong and unknown process ownership fail closed.

### Tests

Added tests covering:

- capability fingerprint stability/change;
- missing caller/document identity;
- mutation SHA-256 receipts;
- exact replay without re-execution;
- caller-bound request ids;
- retained-process ownership;
- cross-document process capability challenge and recovery.

### Housekeeping

Added `docs/CURRENT-ARCHITECTURE.md` as the canonical current contract.

Replaced stale inherited CoS guidance in:

- `README.md`;
- `AGENTS.md`;
- `CLAUDE.md`;
- `CONTRIBUTING.md`;
- `SECURITY.md`;
- pull request and issue templates.

Updated browser instructions to document `process_token` recovery.

Replaced inherited CoS CI with focused chaOS Go/extension checks and retargeted CodeQL to the current extension.

Removed active inherited CoS release, canary, Firefox signing, issue-version, PR triage/checklist and maintenance workflows. The active workflow directory now contains only:

```text
.github/workflows/ci.yml
.github/workflows/browser-native-proof.yml
.github/workflows/codeql.yml
```

Dependabot now maintains GitHub Actions only rather than the inherited CoS npm dependency tree.

Moved the first-refresh PR 2 failure report into `docs/history/` and marked it historical.

### Verification state

The branch remains proof-stage. Current acceptance gates are defined in `docs/CURRENT-ARCHITECTURE.md` and `README.md`.

Final executable checks must be rerun after the housekeeping edits before promotion from draft.

## 2026-10-16 — First core refresh

Development branch: `chaos-core-first-refresh`

The first refresh established the browser-native direction:

- standalone Go native host;
- Chrome Native Messaging;
- unpacked `extension-chaos` browser bridge;
- approved virtual filesystem roots;
- closed-by-default local capabilities;
- direct file and retained-process actions;
- no required Electron runtime;
- no required MCP/tunnel path;
- no local worker/Goal/Loop orchestration runtime.

The original architecture record remains at `docs/CHAOS-REFRESH-2026-10-16.md`.

The original PR 2 verification failure record is preserved as `docs/history/PR-2-FIRST-REFRESH-FAILURE-REPORT.md`.

## 2026-09-29 — Pre-refresh chaOS archive

The earlier fork state is preserved on:

```text
archive/chaos-core-first-2026-09-29
```

## Upstream Chat On Steroids history

chaOS began as a fork of Chat On Steroids. Upstream release notes, version history, Electron/MCP features and contributor history remain available on:

```text
upstream-sync-2026-10-16
```

They are provenance/reference material, not current chaOS release notes.
