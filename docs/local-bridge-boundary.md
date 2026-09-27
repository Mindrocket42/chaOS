# Local bridge boundary

chaOS exists to give ChatGPT a controlled path into the local machine for work that the native
ChatGPT/Codex surfaces cannot do, cannot reach, or do badly.

It is not an agent framework and it is not a Codex supervisor.

## Architecture

Use the shortest local capability path.

```text
ChatGPT
   |
   |  MCP over the configured tunnel
   v
chaOS Core
   |
   +--> approved filesystem
   +--> shell and retained local processes
   +--> local search / patch / inspection
   +--> desktop and browser control when explicitly enabled
   +--> local MCP/plugin integrations
   +--> ordinary local programs and harness CLIs when useful
```

The tunnel is only transport from ChatGPT to chaOS.

Core is the product boundary: it exposes bounded local capabilities, enforces the user's approved
roots and permissions, executes work locally, and returns evidence of what actually happened.

## Selection rule

Prefer the lowest layer that can complete the job.

1. Use a direct chaOS primitive when one exists.
2. Use an installed local program when the operating system already has the capability.
3. Use a harness CLI such as Codex only when delegating to that harness is actually useful.
4. Use browser/UI automation only when the target capability genuinely lives in a browser/UI.

Do not build orchestration around a tool merely because it is available.

## Codex

Codex is one local executable/harness among others. chaOS may invoke it from the shell when it is
better at a particular coding task, but Codex does not define chaOS architecture and chaOS does not
need to manage Codex threads as its normal operating model.

Examples:

- ChatGPT needs to inspect or patch a file: use Core filesystem/patch tools directly.
- ChatGPT needs to run a test or script: use Core process tools directly.
- ChatGPT wants a bounded independent coding pass: it may invoke `codex exec`, OpenCode, or another
  installed harness as an ordinary local process.
- ChatGPT needs a capability Codex lacks, such as an existing local terminal/session, desktop UI,
  clipboard, application integration or another MCP service: chaOS supplies that capability itself.

The existing Codex Desktop bridge RFC remains an optional experiment for the narrow case where
controlling an existing Codex task is useful. It is not a migration target for chaOS.

## Browser companion

The ChatGPT browser companion is also optional.

Keep it only for capabilities that actually require the ChatGPT web UI or an attached browser tab,
for example browser conversation capture, ChatGPT tab management, model discovery, or the legacy
browser-backed worker implementation.

It must not be required for Core MCP, filesystem access, shell/process execution, or ordinary local
work.

## Defaults

Fresh installs should provide the local bridge with the least accidental machinery:

- Core local capabilities available;
- browser-backed workers disabled;
- unattributed calls disabled;
- no browser-extension setup requirement unless a selected feature needs it.

## Non-goals

- No Juggle implementation.
- No swarm architecture.
- No requirement to turn every task into an agent task.
- No Codex-first control plane.
- No browser UI as the default transport.
- No generic arbitrary RPC passthrough.
- No new abstraction where a direct file/process/application capability already solves the problem.

The test is simple: if ChatGPT can reach the required local capability directly through chaOS, that
is the path. Everything else has to justify its existence.
