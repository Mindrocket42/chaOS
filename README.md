<p align="center"><img src="docs/images/readme-hero.svg?v=2" width="960" alt="Turn ChatGPT into Codex-style local coding. Chat On Steroids: Your files. Your terminal. Your ChatGPT plan." /></p>

<p align="center">
  <a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest/download/Chat-On-Steroids-Setup-x64.exe"><img src="docs/images/download-windows.svg" width="208" height="56" alt="Download for Windows x64" /></a>&nbsp;
  <a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest/download/Chat-On-Steroids-macOS-arm64.dmg"><img src="docs/images/download-macos.svg" width="208" height="56" alt="Download for macOS Apple silicon" /></a>&nbsp;
  <a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest/download/Chat-On-Steroids-Linux-x64.deb"><img src="docs/images/download-linux.svg" width="208" height="56" alt="Download for Linux x64" /></a>
</p>

<p align="center"><a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest">All downloads</a></p>

# chaOS / Chat On Steroids

chaOS gives ChatGPT controlled access to your local machine through MCP.

The normal path is deliberately small:

```text
ChatGPT
   |
   | MCP through one HTTPS tunnel
   v
chaOS Core
   |
   +-- approved files
   +-- patches
   +-- terminal / local processes
   +-- Git
   +-- other local capabilities you explicitly enable
```

**You do not need the Chrome extension for ordinary local coding.**

The browser companion is optional. Use it only if you want chaOS to automate ChatGPT.com itself, manage browser-backed worker chats, discover model choices from the web UI, or record browser conversations.

---

## Quick start

### What you need

- Windows 10/11, macOS 13+, or a current desktop Linux.
- A ChatGPT account/workspace that supports MCP apps.
- chaOS installed.
- One HTTPS route from ChatGPT to the local chaOS MCP server.

For most users the easiest route is **OpenAI Secure MCP Tunnel**.

### 1. Install chaOS

Install the appropriate release and launch it.

Then open:

**Settings → Workspace**

Add the project folder(s) ChatGPT may access.

chaOS restricts file operations to the roots you approve.

### 2. Create the tunnel

Open:

**OpenAI Platform → Tunnels**

Create a tunnel in the same OpenAI workspace you use with ChatGPT.

You will get a **tunnel ID**.

### 3. Create the tunnel API key

Open:

**OpenAI Platform → API keys**

Create a **Restricted** API key with only:

- **Tunnels: Read**
- **Tunnels: Use**

This key authenticates the tunnel connection. It is not an additional model API requirement for normal ChatGPT use.

### 4. Connect chaOS to the tunnel

In chaOS open:

**Settings → Setup**

Enter:

- the tunnel ID;
- the restricted API key.

Press **Connect**.

chaOS should show the Core MCP endpoint as connected before you continue.

### 5. Add Core to ChatGPT

In ChatGPT open:

**Plugins → Add → Create MCP App**

Choose:

- **Connection:** Tunnel
- **Tunnel:** the tunnel you created
- **Authentication:** No authentication

Name the app:

```text
Chat On Steroids Core
```

Enable the app and approve its actions.

Older ChatGPT versions may require Developer mode first under **Settings → Security and login**.

### 6. Verify the connection

Start a normal ChatGPT conversation with the Core app enabled.

Ask ChatGPT to do something harmless inside an approved folder, for example:

```text
List the files in this project root.
```

Then test a local command:

```text
Run git status in this repository.
```

If both work, the basic setup is complete.

You now have:

```text
ChatGPT → tunnel → chaOS Core → local files / shell / Git
```

No Chrome extension is required for this path.

---

## What is mandatory and what is optional?

| Component | Required for normal local coding? | Purpose |
| --- | --- | --- |
| **chaOS desktop app** | Yes | Hosts the local capabilities and permission boundary. |
| **Core MCP app** | Yes | Gives ChatGPT access to files, patches, shell/process tools and local session capabilities. |
| **HTTPS tunnel** | Yes | ChatGPT runs remotely and cannot call your localhost directly. |
| **Tunnel API key** | Usually | Authenticates OpenAI Secure MCP Tunnel. Other tunnel methods have their own authentication. |
| **Chrome extension / browser companion** | **No** | Only for browser-specific ChatGPT automation and browser-backed workflows. |
| **Browser bridge** | **No** | Local communication channel used by the optional browser companion. |
| **Desktop connector** | No | Screen, mouse, keyboard and clipboard control. |
| **Plugins connector** | No | External MCP services such as Playwright, Blender or other custom servers. |
| **Codex / OpenCode / other coding harness** | No | Optional local programs chaOS may invoke when useful. |

The rule is simple: **use the lowest layer that can complete the task.**

If Core can read the file, edit it, run the test and commit the result, do not introduce browser automation or another agent harness.

See [Local bridge boundary](docs/local-bridge-boundary.md).

---

## Optional: browser companion

Install the Chrome companion only if you specifically need one of these:

- chaOS-managed ChatGPT tabs;
- browser-backed worker chats;
- model discovery from the ChatGPT web UI;
- Compact & Resume browser transitions;
- browser conversation recording.

To install it:

1. In chaOS press **Open extension folder**.
2. Open `chrome://extensions`.
3. Enable **Developer mode**.
4. Choose **Load unpacked**.
5. Select the folder opened by chaOS.

The companion pairs with the local browser bridge automatically.

If you do not need the features above, skip this entire section.

---

## Optional: Desktop control

The separate **Desktop** connector adds:

- screen inspection;
- mouse control;
- keyboard input;
- clipboard access.

On macOS, grant Screen Recording and Accessibility permission in System Settings.

Desktop control is not required for repository work.

---

## Optional: external MCP plugins

The **Plugins** connector is for other MCP services and integrations.

Examples include Playwright, Blender, memory services, or custom local/remote MCP servers.

It is not required for Core filesystem, shell or Git access.

See [Plugin guide](docs/plugins.md).

---

## Alternative tunnels

OpenAI Secure MCP Tunnel is the most direct supported setup, but chaOS can also use other HTTPS routes.

### Cloudflare quick tunnel

Connect the Cloudflare option in chaOS and use the public URL it displays as the MCP server URL in ChatGPT.

The generated path is secret and may change after restart.

### Your own HTTPS tunnel

Forward a public HTTPS endpoint to the loopback MCP URL shown by chaOS.

Preserve the secret path and treat the resulting URL as a credential.

---

## Troubleshooting

### ChatGPT cannot see Core tools

1. Confirm chaOS shows the tunnel as connected.
2. Confirm **Chat On Steroids Core** is enabled in the current ChatGPT conversation.
3. Refresh the MCP app in ChatGPT.
4. Confirm the requested file is inside an approved Workspace root.

### Tunnel rejects the API key

Confirm:

- the tunnel ID is correct;
- the API key belongs to the same OpenAI workspace;
- the key has **Tunnels: Read** and **Tunnels: Use**.

The Chrome extension does not authenticate the tunnel.

### File access works but commands fail

Check chaOS permissions and **Read-only mode**.

Shell commands run with your normal operating-system user privileges.

### The README told me to install Chrome before Core worked

That was the old architecture assumption.

**Core does not require the browser companion.**

The browser companion is now an optional compatibility/UI automation layer.

### Extension pairing fails

Only relevant if you chose to install the companion.

Reload the unpacked extension after updating chaOS and refresh ChatGPT.

The supported browser bridge ports are 8765-8769; **Auto** selects the first available port.

### ChatGPT reports a tool safety refusal

Do not assume the local action ran.

Check chaOS local tool history for an actual request/result before retrying a potentially destructive command.

---

## Permissions and security

You choose the approved folders and capabilities.

- File tools enforce approved roots.
- Shell commands execute as your normal OS user.
- Desktop access applies to the desktop when enabled.
- External plugins have their own permission models.
- **Read-only mode** disables writes, command execution and desktop control.
- Credentials are stored using operating-system secure storage.

Review [SECURITY.md](SECURITY.md) before exposing sensitive projects.

---

## Browser bridge port

The optional companion uses a local browser bridge.

In:

**Settings → Browser & history → Browser bridge port**

choose **Auto** or one of:

```text
8765 8766 8767 8768 8769
```

If you are not using the browser companion, this setting is irrelevant to the normal Core MCP path.

---

## Development

```sh
npm ci
npm run dev
npm run verify
```

Build targets:

```sh
npm run dist:x64
npm run dist:arm64
npm run dist:mac:x64
npm run dist:mac:arm64
npm run dist:linux:x64
npm run dist:linux:arm64
```

Build on the target OS.

Read [AGENTS.md](AGENTS.md) before changing the app and [CONTRIBUTING.md](CONTRIBUTING.md) before contributing upstream.

---

## Responsible use

chaOS is an independent open-source tool for authorized work with your own files and systems.

It does not grant extra model quota, bypass provider restrictions, or override safety decisions. Use it within the terms and limits of ChatGPT, OpenAI and any connected service.

The browser companion automates the ChatGPT web UI and can record conversation content locally. That is separate from the Core MCP path and should only be enabled when you actually need those browser-specific features.

See [Security](SECURITY.md) for local permissions and risks.

---

[Detailed setup reference](docs/setup.md) · [Tool reference](docs/tool-surface.md) · [Local bridge boundary](docs/local-bridge-boundary.md) · [Plugins](docs/plugins.md) · [MIT license](LICENSE)

Not affiliated with or endorsed by OpenAI. ChatGPT and Codex are OpenAI trademarks.
