<p align="center"><img src="docs/images/readme-hero.svg?v=2" width="960" alt="Turn ChatGPT into Codex-style local coding. Chat On Steroids: Your files. Your terminal. Your ChatGPT plan." /></p>

<p align="center">
  <a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest/download/Chat-On-Steroids-Setup-x64.exe"><img src="docs/images/download-windows.svg" width="208" height="56" alt="Download for Windows x64" /></a>&nbsp;
  <a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest/download/Chat-On-Steroids-macOS-arm64.dmg"><img src="docs/images/download-macos.svg" width="208" height="56" alt="Download for macOS Apple silicon" /></a>&nbsp;
  <a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest/download/Chat-On-Steroids-Linux-x64.deb"><img src="docs/images/download-linux.svg" width="208" height="56" alt="Download for Linux x64" /></a>
</p>

<p align="center"><a href="https://github.com/totec448-spec/chat-on-steroids/releases/latest">All downloads</a></p>

<p align="center"><sub>Independent beta. Use at your own risk and within your provider's rules. <a href="#responsible-use-and-provider-rules">Read the usage notice</a> before connecting.</sub></p>

<br />

<p align="center"><a href="docs/images/demo.mp4"><img src="docs/images/demo.gif" width="960" alt="Chat On Steroids in action: model selection, task plans, live tool results and reusable workers" /></a></p>

<p align="center"><a href="#get-started">Get started</a> &nbsp;·&nbsp; <a href="docs/images/demo.mp4">Watch the demo</a> &nbsp;·&nbsp; <a href="CHANGELOG.md">What’s new</a></p>

<br />

<h2 align="center">Local tools first. Extra machinery only when it earns its keep.</h2>

**Work on the real project.** Core MCP gives ChatGPT approved local files, patches and terminals. That path does not require a browser extension.

**Keep browser automation optional.** The companion is only for features that actually depend on the ChatGPT web UI: CoS-managed tabs, browser-backed workers, model discovery and browser conversation recording.

**Use the shortest local path.** chaOS is a bridge from ChatGPT to your machine, not an agent framework. Use its file, process, desktop and integration capabilities directly; invoke Codex, OpenCode or another harness CLI only when that program is useful for a particular task. See [Local bridge boundary](docs/local-bridge-boundary.md).

<p align="center"><strong>Core MCP works without Chrome.</strong><br /><sub>The optional browser companion still automates ChatGPT.com for the legacy managed-chat workflow.</sub></p>

## Responsible use and provider rules

Chat On Steroids is an independent, open-source workspace for coding and other authorized tasks with your own files and tools. It is intended to support productive work within the rules of the services you use. **It is not intended to bypass usage limits, account restrictions or safety controls.**

Use CoS in accordance with OpenAI's applicable [Terms of Use](https://openai.com/policies/terms-of-use/) ([Europe Terms](https://openai.com/policies/eu-terms-of-use/) for the EEA, Switzerland and UK), [Usage Policies](https://openai.com/policies/usage-policies/) and [Service Terms](https://openai.com/policies/service-terms/), plus your workspace's rules and any connected service's terms.

- **Respect limits and access decisions.** Workers, Goal/Loop, Compact & Resume and finish checkpoints organize work; they do not grant extra quota or model access and must not be used to evade rate limits, usage caps or account restrictions. Do not switch accounts, chats, connectors or tunnels to evade a restriction.
- **Respect safety decisions.** Do not use local tools, browser control, plugins or another worker to carry out an action that the provider blocked for safety. A local permission or an enabled MCP connector is not permission to override a provider refusal.
- **Understand the integration.** CoS connects local tools through MCP. Its companion also observes and automates the ChatGPT browser UI and records conversation content locally. This browser integration is not a public ChatGPT automation API. MCP availability does not establish permission for every form of browser automation or recording; OpenAI's terms also restrict automated or programmatic extraction of data or output.
- **Use at your own risk.** Review the rules for your account and intended workflow before connecting, supervise automation and review tool actions and outputs. CoS cannot guarantee policy compliance, continued service access or protection from account warnings, restrictions or suspension. If a workflow is restricted or receives a policy warning, stop that workflow and seek clarification through the provider's support or appeal process.

This notice states the project's intended use; it does not certify compliance or change provider rules. CoS is not affiliated with, endorsed by or approved by OpenAI. The software is provided as-is under the [MIT license](LICENSE); applicable statutory rights remain unaffected. See [Security](SECURITY.md) for local permissions and risks.

<br />

## Get started

1. **Install CoS** and approve your project folder in **Settings → Workspace**.
2. **Connect Core** through **Settings → Setup** and add it to ChatGPT. [Tunnel setup →](docs/setup.md#tunnel-setup)
3. **Start working in ChatGPT.** Core can read/edit approved files and run permitted commands now. No browser extension is required.
4. **Optional:** load the browser companion only if you want CoS to drive ChatGPT.com itself, manage browser-backed worker chats, discover web UI model choices, or record browser conversations.

<details>
<summary>Requirements &amp; installation notes</summary>

Windows 10/11, **macOS 13 Ventura or newer**, or a current desktop Linux, plus a ChatGPT account/workspace that can connect the Core MCP app. Chrome 125+, current Edge or Brave is needed only for the optional browser companion. [Check account availability](https://help.openai.com/en/articles/12584461-developer-mode-and-mcp-apps-in-chatgpt).

- **Unsigned beta:** Windows is not publisher-signed; macOS is unsigned and unnotarized. Verify the package against the release checksums.
- **Linux:** a Secret Service keyring is required. Prefer the DEB; when unprivileged user namespaces are disabled, the AppImage launcher can fall back to <code>--no-sandbox</code>.
- **Permissions:** choose your approved folders and review capabilities before connecting. Fresh installs enable Core capabilities; browser-backed workers start disabled. Windows also enables Desktop permissions. Shell commands run with your normal user privileges.
- **Languages:** English, German, Spanish, French, Portuguese (Portugal), Turkish, Japanese, and Simplified and Traditional Chinese. Choose one in **Appearance → Language**.
- **After updating:** reload the companion extension and refresh the CoS apps in ChatGPT when prompted.

</details>

<details>
<summary>More screenshots</summary>

![Conversation, workers and task plan](docs/images/workspace.png)

![Model and reasoning selection](docs/images/model-picker.png)

![Folder and capability settings](docs/images/settings.png)

</details>

<br />

---

<p align="center"><a href="docs/setup.md">Setup &amp; help</a> &nbsp;·&nbsp; <a href="docs/plugins.md">Plugins</a> &nbsp;·&nbsp; <a href="CONTRIBUTING.md">Contribute</a> &nbsp;·&nbsp; <a href="SECURITY.md">Security</a> &nbsp;·&nbsp; <a href="LICENSE">MIT license</a></p>

<p align="center">Built with our <a href="CONTRIBUTORS.md">community contributors</a>. Thank you to the people behind the code, designs, bug reports and testing.</p>

<p align="center"><sub>Not affiliated with or endorsed by OpenAI. ChatGPT and Codex are OpenAI trademarks.</sub></p>
