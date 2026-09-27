# Runtime boundary

chaOS should not require a browser extension to do work that MCP, a local process or a supported
agent runtime can do directly.

## Architecture rule

Use the shortest supported control path.

```text
ChatGPT / another MCP client
        |
        v
     Core MCP
        |
        +--> approved files / patches / terminal / local evidence
        |
        +--> runtime adapters
                |
                +--> Codex Desktop / CLI      preferred delegation path
                +--> ChatGPT browser tabs     compatibility path only

Optional surfaces:
  Desktop MCP  -> browser/native UI control
  Plugins MCP  -> external MCP integrations
```

The tunnel is transport to Core MCP. It is not a worker runtime.

The browser companion is a UI adapter. It is not the Core transport and must not be a prerequisite
for local file, terminal or MCP operation.

## What belongs where

### Core MCP

Core owns the stable local capability boundary:

- approved-root filesystem access;
- patching and search;
- command execution and retained process sessions;
- capability enforcement and read-only mode;
- bounded session/tool evidence;
- runtime-adapter tools that are safe to expose to a model.

Core must start and remain useful with no browser extension installed.

### Runtime adapters

A runtime adapter supervises another agent/harness without pretending that its UI is the protocol.

Each adapter owns:

- exact runtime/thread identity;
- create or attach semantics;
- bounded input and output;
- status and completion evidence;
- cancellation/teardown where supported;
- provenance and authority boundaries;
- read-back after mutating operations.

The model gets a small stable contract. It does not get arbitrary runtime RPC.

### Codex Desktop / CLI

Codex is the first preferred adapter because the repository has already proved exact-thread
background delivery to an existing Desktop task through the supported CLI.

Use the supported surfaces in this order:

1. bounded list/read/status from the Codex app-server or first-party CLI;
2. exact-thread send/continue;
3. read-back confirming the intended thread and observed input;
4. bounded wait/status evidence;
5. non-interactive `codex exec` for disposable delegated jobs.

Do not treat `codex queue` exit zero as completion. Queue acceptance is not a consumption receipt,
unloaded threads may remain pending, and queued text currently arrives with user-turn authority.
The adapter must fail closed when identity or consumption cannot be established.

### Browser companion

The companion remains available for capabilities that genuinely require ChatGPT.com UI access:

- CoS-managed ChatGPT conversations;
- browser-backed worker tabs;
- model discovery from the web UI;
- browser conversation capture and attribution;
- Compact & Resume transitions that create or navigate provider chats.

It is optional compatibility machinery, not the default execution substrate.

## Defaults

Fresh installs should have:

- Core capabilities available;
- browser-backed workers disabled;
- unattributed calls disabled;
- no missing-extension warning unless an enabled feature actually requires the companion.

Installing Chrome must never be a prerequisite for Core MCP.

## Migration sequence

1. **Core-first default**  
   Remove the unconditional browser dependency and stop enabling browser workers on first launch.

2. **Codex runtime vertical slice**  
   Implement exact-thread `list/read/send/status` with separate read/control permissions and
   deterministic tests. Add `exec`-backed disposable delegation only after its output and sandbox
   contract is bounded.

3. **Worker abstraction**  
   Move orchestration state from “worker == ChatGPT tab” to “worker == runtime handle”. Keep the
   current browser worker implementation behind a compatibility adapter.

4. **Delete coupling, not capability**  
   Once runtime-backed delegation covers the required behavior, remove browser-specific assumptions
   from agent ownership, recovery and setup. Only then consider moving the companion under a
   clearly named legacy/adapter directory.

## Non-goals

- No direct writes to Codex databases.
- No private-pipe attachment when a supported surface exists.
- No generic app-server RPC passthrough to the model.
- No browser-cookie or profile proxy.
- No title-only mutation of a runtime thread.
- No agent-authored text promoted to human authority without explicit provenance handling.
- No “agent swarm” feature merely because multiple processes can be launched.

The test for every added layer is simple: if removing it leaves the same capability and evidence,
the layer does not belong in the architecture.
