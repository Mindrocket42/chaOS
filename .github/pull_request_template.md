## Why

<!-- What authority boundary, failure mode or user problem does this solve? -->

## What changed

<!-- Describe the behavior change and what was deliberately left unchanged. -->

## Verification

<!-- Separate evidence explicitly. Delete examples that do not apply. -->

Verified:
- <!-- exact command/live proof and result -->

Static/inferred:
- <!-- code inspection, syntax checks, reasoning only -->

Pending:
- <!-- runtime/live evidence still required, or "none" -->

## Architecture impact

<!-- Write "No architecture change" or describe the contract change and update docs/CURRENT-ARCHITECTURE.md. -->

## Live proof

<!-- For browser/native runtime changes, include an exact receipt, focused log excerpt, screenshot or other direct evidence when practical. -->

## Checklist

- [ ] I read `AGENTS.md` and `docs/CURRENT-ARCHITECTURE.md`.
- [ ] `cd native-host && go test ./...` passes, or I explained why it was not applicable/available.
- [ ] Changed extension JavaScript passes `node --check`.
- [ ] I did not use `npm ci` / Electron / MCP as a substitute for the browser-native chaOS test path unless this PR explicitly targets inherited CoS code.
- [ ] Permissions still start closed unless this PR explicitly changes that architecture.
- [ ] Privileged caller identity still comes from transport metadata, not model text.
- [ ] Mutation/completion claims are backed by receipts or explicit verification evidence.
- [ ] `process_id` is not treated as a bearer credential; retained-process recovery preserves the `process_token` boundary.
- [ ] README/current architecture were updated if setup, behavior, branch status or verification state changed.
- [ ] Examples/logs contain no real credentials, private paths, private chat content or private identifiers.
