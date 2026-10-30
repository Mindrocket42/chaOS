# chaOS browser bridge

```text
ChatGPT in Chrome
  -> content bridge
  -> extension service worker
  -> Chrome Native Messaging
  -> standalone chaos-host.exe
  -> approved local capabilities
```

No OpenAI Secure MCP Tunnel, tunnel id, `tunnel-client`, localhost HTTP port, ChatGPT custom connector, Electron runtime or browser worker system is required.

The extension requests only:

- host access to `chatgpt.com` / `chat.openai.com`;
- `nativeMessaging`;
- `activeTab`, used when the user explicitly enables the automatic bridge for the current ChatGPT tab.

It does not request debugger, broad tab-management, cookies, localhost, or arbitrary web-host access.

## Modes

### Manual

An assistant message containing a `CHAOS_ACTION_BEGIN` envelope gets a visible **Run with chaOS** control. The real native receipt is shown beside the message and can be copied back to ChatGPT.

### Automatic — per-chat opt-in

Choose **Enable automatic bridge in this chat** from the extension popup.

That user gesture:

1. creates a fresh random nonce for the tab;
2. sends ChatGPT the tiny action/result protocol;
3. allows only nonce-matching action envelopes;
4. executes one local action at a time;
5. sends the real receipt back into the same conversation;
6. lets ChatGPT decide the next action.

The mode is deliberately ephemeral: page reload disables it. It also stops after 50 automatic actions and requires an explicit re-enable.

This is transport glue, not a local agent framework. Do not add worker-chat creation, model selection, tab scheduling, Goal/Loop, conversation recording, recovery orchestration, or browser-wide automation here.

## Native host

The extension connects only to:

```text
pro.ironhead.chaos
```

The unpacked extension has a fixed ID:

```text
oneodmlppiiihbbligdiloibjagnfach
```

The native host manifest accepts only that extension origin.

`chaos-host.exe` is a standalone Go console executable because Chrome Native Messaging requires an unpolluted binary stdin/stdout channel. The earlier attempt to use the packaged Electron executable was rejected by the wire test on Windows: Electron emitted CRLF (`0d0a`) onto stdout before a native frame. The standalone host passes the same framing test in CI.

## Local access

The host starts closed:

```text
roots:   none
read:    off
search:  off
write:   off
command: off
```

File operations use virtual approved roots. Command access is a separate explicit capability and is not represented as root-confined; enabled commands run with the normal Windows user account permissions.

## Proof bundle

The `Browser-native proof` workflow produces `chaos-browser-native-windows-x64` containing:

- `chaos-host.exe`
- this unpacked extension
- `register-host.cmd`
- `unregister-host.cmd`
- `status.cmd`
- explicit folder/capability helpers

The exact executable placed in the artifact is exercised with Chrome's length-prefixed native-messaging wire format before upload.
