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

## Current proof slice

The current content bridge deliberately requires a user click for each detected `CHAOS_ACTION_BEGIN` action. The result is displayed beside the assistant message and can be copied back into the conversation. This proves the browser-to-native capability path without making ChatGPT DOM automation part of the local security boundary.

The next refinement may automate result hand-back, but it must remain a thin transport shim. It must not grow workers, tab schedulers, Goal/Loop, conversation persistence, model selection, or recovery orchestration.

## Windows development setup

1. Build/package chaOS so there is a real packaged executable.
2. Register that executable as the native host:

   `npm run native:register -- --exe "C:\\path\\to\\chaOS.exe"`

3. In Chrome, open `chrome://extensions`, enable Developer mode, choose **Load unpacked**, and select `extension-chaos`.
4. Open the extension popup. It must report:

```text
Connected
Transport: Chrome Native Messaging
MCP tunnel: Not used
Native host: Running
```

5. Use **Copy ChatGPT bridge instructions** and paste them once into the ChatGPT conversation being used for the proof.

Unregister the native host with:

`npm run native:unregister`

The extension has a fixed manifest key, therefore its unpacked extension id is stable: `oneodmlppiiihbbligdiloibjagnfach`. The native host accepts only that extension origin.
