# chaOS refresh — 2026-10-16

## Objective

Rebase chaOS on the current Chat On Steroids core, keep the useful local capability plumbing, and remove the product machinery that turns a local bridge into a second ChatGPT runtime.

The primary target is deliberately small:

```text
ChatGPT in Chrome
       |
       | explicit chaOS action/result envelope
       v
chaOS extension
       |
       | Chrome Native Messaging
       v
chaOS capability kernel
       |
       +-- files / search / patch
       +-- shell / retained processes
       +-- git
       +-- optional desktop control
       +-- optional local harness launch (Codex / OpenCode / Claude Code / other CLI)
```

ChatGPT is the orchestrator. chaOS is the executor and evidence boundary.

MCP remains an optional compatibility transport. chaOS does not depend on it.

## Non-negotiable architecture rules

1. **MCP is optional transport, not architecture.** No orchestration policy, worker lifecycle, conversation lifecycle, capability implementation or durable reasoning state may depend on MCP.
2. **Browser-native is the primary local path.** Chrome Native Messaging carries extension-to-host traffic directly. It needs no localhost HTTP server, tunnel id, tunnel-client, public endpoint or ChatGPT connector registration.
3. **One capability kernel.** Browser-native and any retained MCP compatibility adapter call the same local capability layer. Transports do not implement filesystem/process policy independently.
4. **Transport ownership must be obvious.** In browser-native mode Chrome owns the native messaging pipe and launches the chaOS host. Disabling/removing the extension removes that browser path. In MCP compatibility mode chaOS owns any tunnel child process and exposes its PID/path/state.
5. **Browser automation is not orchestration.** The extension may carry explicit action/result envelopes between the current ChatGPT conversation and the native host. It must not grow workers, tab schedulers, model-selection automation, Goal/Loop, conversation recovery machinery or a second agent runtime.
6. **No local agent framework.** No browser-backed workers, Goal, Loop, swarm, worker retirement, worker chat archival, automatic Continue, or chat-tab lifecycle as a dependency of local work.
7. **No hidden persistence requirement.** Session recording is opt-in. Local execution does not depend on recording ChatGPT conversations.
8. **Durable work state belongs in the workspace.** For long work, ChatGPT writes objective, plan, findings, decisions and receipts to project files. Conversation history is not the source of truth.
9. **Direct capability beats delegation.** Read/patch/run directly when sufficient. Launch a coding harness only when the harness adds material capability.
10. **Every extra layer must justify itself.** If removing a component leaves the required user outcome unchanged, remove it.

## What to keep from current CoS

- Approved-root filesystem boundary.
- Search / patch / file inspection implementations.
- Shell and retained-process support.
- Git support.
- Child-process environment hardening and process-tree cleanup.
- Diagnostics that report real state rather than inferred state.
- Installer/packaging/test hardening.
- Optional native desktop control.
- OpenAI Secure MCP Tunnel and generic HTTPS/manual adapters only as compatibility transports while useful.

## What to sideline or remove from the normal product

- Browser companion as a conversation recorder/control plane.
- Built-in ChatGPT browser as an execution dependency.
- Browser-backed workers / multi-agent ChatGPT tabs.
- Goal and Loop.
- Automatic Continue / fresh-turn rotation machinery.
- Compact & Resume automation as a product dependency.
- Automatic plugin-refresh clicks in the ChatGPT UI.
- ChatGPT DOM/error-string recovery loops.
- Session recording as an invariant.
- Local scraping of Codex/Claude usage logs.
- Desktop pets and unrelated UI/product extras.
- Multiple ChatGPT-facing tunnels for Core/Desktop/Plugins.
- Any orchestration abstraction around a direct file/process capability.

## Minimal orchestration model

ChatGPT already provides the reasoning/orchestration loop. chaOS only needs deterministic capabilities and durable process handles.

Minimum useful action families:

```text
health
fs.list / fs.read / fs.search / fs.write / fs.replace
proc.run / proc.read / proc.write / proc.stop
git.status / git.diff / git.commit                 # may initially be shell-backed
computer.screenshot / computer.input               # optional
harness.run / harness.status / harness.stop         # optional thin wrappers
```

A long-running task is therefore:

```text
ChatGPT decides next action
  -> emits one chaOS action
  -> extension hands it to native host
  -> chaOS executes and returns receipt
  -> receipt returns to ChatGPT
  -> ChatGPT decides next action
```

Parallel work is just multiple retained process or harness handles. It does not require worker chats, browser tabs, a swarm scheduler or a second model pretending to be the user.

## Browser-native transport contract

Browser mode has no MCP server and no Secure MCP Tunnel:

```text
ChatGPT tab
   -> chaOS content bridge
   -> extension service worker
   -> chrome.runtime.connectNative("pro.ironhead.chaos")
   -> chaOS packaged executable in native-host mode
   -> capability kernel
```

The extension popup must answer the operational question directly:

```text
chaOS Local Bridge
Connected

Transport:   Chrome Native Messaging
MCP tunnel:  Not used
Native host: Running
```

No hidden local port exists in this mode.

The first browser proof uses explicit user-approved action controls in the ChatGPT page. Full automatic result hand-back is a later transport refinement; it must remain a tiny shim, not become a DOM-driven agent framework.

## MCP compatibility contract

If MCP is retained, it is just another adapter over the same capability kernel.

```text
[STOPPED]
Start MCP compatibility transport

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

There must be no question of "where is MCP?" If compatibility mode is on, chaOS identifies every owned component. If it is off, no chaOS-owned MCP/tunnel process exists.

## Initial default posture

Fresh install target:

- Browser-native transport preferred.
- Approved roots empty until the user selects them.
- Core filesystem/process capabilities unavailable until explicitly granted.
- Desktop off.
- MCP compatibility transport off.
- Browser worker machinery off.
- Multi-agent workers off.
- Goal / Loop off.
- Automatic Continue / turn rotation off.
- Session recording off.
- Usage-log harvesting off.
- Local control API off.

## Acceptance tests

1. With no ChatGPT MCP connector and no tunnel-client running, the Chrome extension can ping the chaOS native host.
2. Through browser-native mode, an approved action can list/read/search an approved root and run a retained process.
3. The extension popup explicitly reports `Chrome Native Messaging` and `MCP tunnel: Not used`.
4. Native-host mode does not acquire the normal desktop app single-instance lock, create windows, start tunnels or start browser-worker machinery.
5. Closing the native messaging connection terminates the native-host process and any child processes it owns.
6. No second Desktop or Plugins tunnel is required for browser-native use.
7. A fresh install does not open or manipulate ChatGPT browser tabs by itself.
8. A fresh install does not record conversations or start workers, Goal, Loop, background chats or usage harvesting.
9. ChatGPT can start two independent local process/harness jobs and poll both through handles without browser workers.
10. MCP compatibility mode can be disabled without disabling the local capability kernel.
11. All core tests and packaged Windows smoke tests pass.

## Repository state

- Previous chaOS state is preserved on `archive/chaos-core-first-2026-09-29`.
- `main` was reset to the current upstream CoS head (`8c3b9611d8c0f1a9fe54fbff883e003269160b50`) on 2026-10-16.
- `chaos-core-first-refresh` contains the browser-native prototype and remains a draft until the vertical slice is built and exercised.
