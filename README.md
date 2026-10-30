# chaOS — archived

> **Project discontinued: 30 October 2026.**
>
> chaOS is retained as a historical engineering record. It is no longer under active development and should not be treated as a supported local-execution path for ChatGPT.

## Closure reason

The project failed its practical end-to-end acceptance test.

The Windows host could be built, tested and registered successfully, and the local CLI confirmed an explicit read-only grant for `B:\CORPORA`. However, a fresh Chrome-originated `health` call continued to report chaOS's closed default configuration: no roots and all capabilities disabled. Reconnects and fresh request IDs did not change that result.

The exact Windows host/config identity mismatch was not resolved before closure. Automatic bridge execution had also shown independent reliability problems. Passing unit tests, a connected popup and a registered Native Messaging manifest were therefore insufficient evidence that the complete browser-to-host capability path worked reliably.

That violates the core operating requirement:

> **ChatGPT decides. chaOS executes. Receipts prove what happened.**

If the receipt does not reliably correspond to the authority configured on the real host, the system is not dependable enough to keep.

## Status

- **Development:** stopped
- **Support:** none
- **Recommended use:** historical reference only
- **Last active architecture:** Chrome extension + standalone Go Native Messaging host
- **Last development branch:** `chaos-core-second-refresh`
- **Canonical design record:** [`docs/CURRENT-ARCHITECTURE.md`](docs/CURRENT-ARCHITECTURE.md)
- **Implementation/audit record:** [`docs/CHAOS-SECOND-REFRESH-WORKING-STATE.md`](docs/CHAOS-SECOND-REFRESH-WORKING-STATE.md)

The repository history preserves the implementation, experiments, tests and earlier README documentation.

MIT licensed.
