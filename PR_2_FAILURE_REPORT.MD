# PR 2 FAILURE REPORT

Status: **DO NOT MERGE**

Reviewed head: `dbcf57373c152ea90ff468d4898b78aee25f73d2`  
Reviewed: 2026-10-16

## Blocking defects

1. **Ubuntu verification fails in the browser dashboard/native transport check.**
   - Actions run: `38922754919`
   - Job: `verify (ubuntu-latest)` (`117046665714`)
   - Failing step: `Run verification script`
   - Failure: the embedded verification exits with `AssertionError` immediately after `[verify] browser dashboard/native transport wiring`.
   - Windows and macOS verification pass; the current CI failure is Linux-specific.

2. **Required end-to-end browser smoke proof is still outstanding.**
   - The PR description explicitly states that live/browser proof is incomplete.
   - It also states that the branch remains draft until the packaged Windows native-host path is built, registered, and exercised end-to-end.
   - For a browser-native transport/dashboard redesign, that proof is part of the acceptance surface.

## What already passes

- Unit tests: **167 passed, 7 skipped**.
- Ruff: clean.
- CLI/help and integration wiring checks: pass.
- Matrix REST authentication/persistence checks: pass.
- Windows verification matrix job: pass.
- macOS verification matrix job: pass.

## Required remediation

- Fix the Ubuntu browser dashboard/native transport verification assertion.
- Re-run the full verification matrix to green.
- Produce the missing end-to-end browser smoke evidence against the real dashboard/transport path.
- Only then re-evaluate this PR for merge.
