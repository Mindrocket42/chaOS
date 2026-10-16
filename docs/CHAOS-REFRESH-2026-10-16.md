# chaOS refresh — 2026-10-16

## Objective

Rebase chaOS on the current Chat On Steroids core, keep the useful local capability plumbing, and remove the product machinery that turns a local bridge into a second ChatGPT runtime.

The target is deliberately small:

```text
ChatGPT
   |
   | one MCP connection boundary
   v
chaOS Core
   |
   +-- files / search / patch
   +-- shell / retained processes
   +-- git
   +-- optional desktop control
   +-- optional local harness launch (Codex / OpenCode / Claude Code / other CLI)
```

ChatGPT is the orchestrator. chaOS is the executor and evidence boundary.

## Non-negotiable architecture rules

1. **MCP is a wire protocol, not an operating system.** No orchestration policy, worker lifecycle, conversation lifecycle, or durable reasoning state belongs in MCP.
2. **One normal connection.** Core is the only required ChatGPT plugin/surface. Desktop tools, if enabled, are published through Core rather than requiring a second tunnel. External local tools are invoked downstream by Core rather than republished as additional ChatGPT-facing tunnels unless there is a demonstrated need.
3. **The tunnel is owned by chaOS.** It is a child process of chaOS, not ambient machine state. If chaOS stops, its tunnel stops.
4. **Transport state must be observable.** The UI and diagnostics must show: transport kind, executable path, PID, running/stopped, tunnel id, local MCP endpoint, last successful control-plane handshake, last request from ChatGPT, last tool call, and last exit/error. Start / Stop / Restart must be explicit controls.
5. **Browser automation is not part of the normal path.** No Chrome/Firefox extension is required for files, shell, Git, desktop control, or local harness execution.
6. **No local agent framework.** No browser-backed workers, Goal, Loop, swarm, worker retirement, worker chat archival, automatic Continue, or chat-tab lifecycle as a dependency of local work.
7. **No hidden persistence requirement.** Session recording is opt-in. Local execution does not depend on recording ChatGPT conversations.
8. **Durable work state belongs in the workspace.** For long work, ChatGPT writes objective, plan, findings, decisions and receipts to project files. Conversation history is not the source of truth.
9. **Direct capability beats delegation.** Read/patch/run directly when sufficient. Launch a coding harness only when the harness adds material capability.
10. **Every extra layer must justify itself.** If removing a component leaves the required user outcome unchanged, remove it.

## What to keep from current CoS

- Core MCP tool implementations and schemas that directly expose useful local capability.
- Approved-root filesystem boundary.
- Search / patch / file inspection.
- Shell and retained-process support.
- Git support.
- OpenAI Secure MCP Tunnel adapter and generic HTTPS/manual adapter as transport options.
- Tunnel health probing, proxy support and route-settle fixes.
- Diagnostics that report real state rather than inferred state.
- Installer/packaging/test hardening.
- Optional native desktop control, but folded into the Core surface.
- Optional local-control API only if it remains loopback-only and off by default.

## What to sideline or remove from the normal product

- Browser companion requirement.
- Built-in ChatGPT browser as an execution dependency.
- Browser-backed workers / multi-agent ChatGPT tabs.
- Goal and Loop.
- Automatic Continue / fresh-turn rotation machinery.
- Compact & Resume automation as a product dependency.
- Automatic plugin-refresh clicks in the ChatGPT UI.
- ChatGPT DOM/error-string automation and recovery loops.
- Session recording as an invariant.
- Local scraping of Codex/Claude usage logs.
- Desktop pets and unrelated UI/product extras.
- Multiple ChatGPT-facing tunnels for Core/Desktop/Plugins.
- Any orchestration abstraction around a direct file/process capability.

## Minimal orchestration model

ChatGPT already provides the reasoning/orchestration loop. chaOS only needs deterministic tools and durable process handles.

Minimum useful tool families:

```text
fs.read / fs.search / fs.patch / fs.list
proc.run / proc.status / proc.read / proc.stop
git.status / git.diff / git.commit
computer.screenshot / computer.input       # optional
harness.run / harness.status / harness.stop # optional thin wrappers
```

A long-running task is therefore:

```text
ChatGPT decides next action
  -> calls chaOS tool
  -> chaOS executes and returns receipt
  -> ChatGPT decides next action
```

Parallel work is just multiple retained process or harness handles. It does not require worker chats, browser tabs, a swarm scheduler or a second model pretending to be the user.

## Tunnel contract

The current OpenAI product cannot call a developer-machine localhost MCP server directly. A remote HTTPS endpoint or Secure MCP Tunnel is therefore required for ordinary ChatGPT-to-local-tool access. The tunnel is transport only.

chaOS must make that transport boring and explicit:

```text
[STOPPED]
Start connection

[RUNNING]
Transport: OpenAI Secure MCP Tunnel
PID: 12345
Binary: C:\...\resources\tunnel\tunnel-client.exe
Tunnel: tunnel_xxx
Local server: http://127.0.0.1:PORT/<secret>/mcp
Last OpenAI handshake: 12:41:07
Last ChatGPT request: 12:41:19
Last tool call: 12:41:20
[Stop] [Restart] [Open diagnostics]
```

There must be no question of "where is MCP?" The answer must always be visible: **the MCP server is inside chaOS; the tunnel-client is a child process owned by chaOS; this is its PID/path/state.**

## Initial default posture

Fresh install:

- Core file/search/patch/process/Git capabilities: available once a workspace root is approved.
- Desktop: off.
- Browser companion: off / not required.
- Multi-agent workers: off.
- Goal / Loop: off.
- Automatic Continue / turn rotation: off.
- Session recording: off.
- Usage-log harvesting: off.
- Local control API: off.
- Tunnel auto-connect: off until configured; once configured, one explicit Connect controls the owned child process.

## Acceptance tests

1. With no browser extension installed, ChatGPT can list/read/patch an approved project, run a test and inspect Git status.
2. Killing chaOS kills the tunnel child; no chaOS-owned tunnel-client remains.
3. The app shows the tunnel executable path and PID while running.
4. One button stops the local server/tunnel path; one button starts it.
5. No second Desktop or Plugins tunnel is required for ordinary use.
6. A fresh install does not open or manipulate ChatGPT browser tabs.
7. A fresh install does not record conversations.
8. A fresh install does not start workers, Goal, Loop, background chats or usage harvesting.
9. ChatGPT can start two independent local harness jobs and poll both through process/job handles without browser workers.
10. All core tests and packaged Windows smoke tests pass.

## Repository state

- Previous chaOS state is preserved on `archive/chaos-core-first-2026-09-29`.
- `main` was reset to the current upstream CoS head (`8c3b9611d8c0f1a9fe54fbff883e003269160b50`) on 2026-10-16.
- This work proceeds on `chaos-core-first-refresh`.
