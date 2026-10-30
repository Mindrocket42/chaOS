# Contributing to chaOS

chaOS is currently a proof-stage browser-to-local execution bridge for ChatGPT.

Before changing it, read:

- [`docs/CURRENT-ARCHITECTURE.md`](docs/CURRENT-ARCHITECTURE.md)
- [`AGENTS.md`](AGENTS.md)
- [`README.md`](README.md)

Security problems should be reported privately through GitHub's **Security → Report a vulnerability** flow, not as a public issue or PR.

## Current development surface

The active browser-native implementation is:

```text
native-host/
extension-chaos/
TEST-BROWSER-NATIVE.cmd
scripts/verify-chaos-native-host.mjs
.github/workflows/browser-native-proof.yml
```

The repository still contains substantial inherited Chat On Steroids Electron/MCP code and historical documentation. That material is reference/provenance unless the current architecture explicitly adopts it.

Do not use `npm ci`, `npm run dev`, Electron, MCP or tunnel setup as the default chaOS development path.

## Change rules

Keep changes narrow and authority-preserving.

A useful contribution should normally do one of four things:

1. strengthen the browser/native transport;
2. strengthen the local capability boundary;
3. strengthen evidence/receipts and deterministic verification;
4. simplify or remove inherited machinery that is no longer part of chaOS.

Preserve these invariants unless the PR explicitly changes the architecture:

- fresh access starts closed;
- approved virtual roots govern filesystem actions;
- command execution is a separate, powerful grant and is not root-confined;
- privileged caller identity comes from Chrome transport metadata, not model text;
- mutation claims carry receipts/evidence rather than assertions;
- successful request ids are caller-bound and idempotent within one host lifetime;
- `process_id` alone is not authority for retained processes;
- long-lived process recovery uses the explicit `process_token` capability;
- ChatGPT remains the orchestrator;
- chaOS core does not become a local worker/agent framework.

Prefer deleting stale machinery over adding another fallback, scheduler, watcher, timer or duplicated state owner.

## Verification

Focused source checks:

```text
cd native-host && go test ./...
node --check extension-chaos/background.js
node --check extension-chaos/content.js
node --check extension-chaos/popup.js
```

Windows browser-native bootstrap/proof:

```text
TEST-BROWSER-NATIVE.cmd
```

For behavior changes, include the narrowest deterministic regression test practical. If a claim depends on live Chrome or Windows Native Messaging, provide that live proof rather than treating unit tests as equivalent evidence.

Do not say a check passed unless it actually ran against the submitted head.

## Pull requests

A PR should state:

### Why

What authority boundary, failure mode or user problem is being corrected?

### What changed

What behavior changed, and what was deliberately not changed?

### Verification

List exact checks run and their results. Separate:

- **verified** — actually executed against the PR head;
- **inferred/static** — code inspection or syntax-level reasoning;
- **pending** — live/runtime evidence still required.

### Architecture impact

State one of:

- `No architecture change`; or
- identify the changed contract and update `docs/CURRENT-ARCHITECTURE.md` in the same PR.

For browser/runtime changes, include live evidence when practical. A screenshot, exact receipt, or focused log excerpt is preferable to prose saying it worked.

Keep real usernames, private paths, chat contents, IDs and credentials out of examples and logs.

## Historical/upstream work

If a task explicitly targets inherited CoS code, say so in the PR. Do not mix unrelated CoS maintenance into a chaOS core change.

The synchronized upstream snapshot is preserved on:

```text
upstream-sync-2026-10-16
```

The previous chaOS lines are preserved on:

```text
archive/chaos-core-first-2026-09-29
chaos-core-first-refresh
```

Use those branches for archaeology instead of keeping stale current-branch instructions alive merely for history.

Contributions are accepted under the MIT licence in [`LICENSE`](LICENSE).
