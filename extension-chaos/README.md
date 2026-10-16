# chaOS browser-native prototype

This extension is the primary transport experiment for chaOS.

```text
ChatGPT in Chrome
  -> extension content bridge
  -> extension service worker
  -> Chrome Native Messaging
  -> chaOS native host
  -> approved local capabilities
```

It does **not** use an OpenAI Secure MCP Tunnel, `tunnel-client`, a localhost HTTP port, or a ChatGPT custom connector.

The extension itself asks only for ChatGPT host access plus Chrome Native Messaging. It does not request debugger, tabs, tab-group, cookies, broad web-host or localhost permissions.

## Current proof slice

The current content bridge deliberately requires a user click for each detected `CHAOS_ACTION_BEGIN` action. The result is displayed beside the assistant message and can be copied back into the conversation. This proves the browser-to-native capability path without making ChatGPT DOM automation part of the local security boundary.

The next refinement may automate result hand-back, but it must remain a thin transport shim. It must not grow workers, tab schedulers, Goal/Loop, conversation persistence, model selection, or recovery orchestration.

This first slice is deliberately not a finished replacement for MCP yet. The acceptance boundary is narrower: prove that ChatGPT browser output can request a bounded action, Chrome can hand it directly to chaOS, and chaOS can return a real receipt without a tunnel.

Operationally, the bridge has one obvious stop control: **Disconnect local bridge** in the extension popup. There is no separate tunnel process to find or kill in browser-native mode.

## Windows proof bundle

The `Browser-native proof` GitHub workflow builds an unpacked Windows x64 application and packages it with this extension plus two one-line controls:

- `register-host.cmd` — registers the packaged executable as Chrome's `pro.ironhead.chaos` native host.
- `unregister-host.cmd` — removes that registration.

No Node installation is needed to use the proof bundle.

1. Double-click `register-host.cmd`.
2. Open `chrome://extensions`, enable Developer mode, choose **Load unpacked**, and select the bundle's `extension-chaos` folder.
3. Open the extension popup. It must report:

```text
Connected
Transport: Chrome Native Messaging
MCP tunnel: Not used
Native host: Running
```

4. Use **Copy ChatGPT bridge instructions** and paste them once into the ChatGPT conversation being used for the proof.
5. When ChatGPT emits a chaOS action, click **Run with chaOS**, then use **Copy result** to return the receipt to the conversation.

The extension has a fixed manifest key, therefore its unpacked extension id is stable: `oneodmlppiiihbbligdiloibjagnfach`. The native host accepts only that extension origin.
