# Historical: PR 2 first-refresh failure report

> Historical record only. This report applied to `chaos-core-first-refresh` in October 2026. The second-refresh line replaced the inherited CoS CI surface with focused chaOS CI, so this document is not a current merge gate. The missing live browser proof remains relevant as historical context, but current acceptance gates are defined in `docs/CURRENT-ARCHITECTURE.md`.

Original status: **DO NOT MERGE**

Reviewed head: `dbcf57373c152ea90ff468d4898b78aee25f73d2`  
Reviewed: 2026-10-16

## Blocking defects at the time

1. **Ubuntu verification failed in the browser dashboard/native transport check.**
   - Actions run: `38922754919`
   - Job: `verify (ubuntu-latest)` (`117046665714`)
   - Failing step: `Run verification script`
   - Failure: the embedded verification exited with `AssertionError` immediately after `[verify] browser dashboard/native transport wiring`.
   - Windows and macOS verification passed; the CI failure was Linux-specific.

2. **Required end-to-end browser smoke proof was outstanding.**
   - The PR description explicitly stated that live/browser proof was incomplete.
   - It also stated that the branch remained draft until the packaged Windows native-host path was built, registered, and exercised end-to-end.
   - For the browser-native redesign, that proof was part of the acceptance surface.

## What passed at the time

- Unit tests: **167 passed, 7 skipped**.
- Ruff: clean.
- CLI/help and integration wiring checks: pass.
- Matrix REST authentication/persistence checks: pass.
- Windows verification matrix job: pass.
- macOS verification matrix job: pass.

## Required remediation at the time

- Fix the Ubuntu browser dashboard/native transport verification assertion.
- Re-run the full verification matrix to green.
- Produce the missing end-to-end browser smoke evidence against the real dashboard/transport path.
- Only then re-evaluate the PR for merge.

## Supersession note

The October 30 second-refresh housekeeping pass retired the inherited CoS CI/release/triage workflows from the current chaOS line. Current source and runtime verification is now defined by:

- `.github/workflows/ci.yml` — focused Go/extension checks;
- `.github/workflows/browser-native-proof.yml` — Windows framing/build proof;
- `.github/workflows/codeql.yml` — extension static analysis;
- `docs/CURRENT-ARCHITECTURE.md` — current acceptance gates.
