const state = document.querySelector('#state');
const host = document.querySelector('#host');
const chatState = document.querySelector('#chat-state');
const detail = document.querySelector('#detail');
const refresh = document.querySelector('#refresh');
const disconnect = document.querySelector('#disconnect');
const enableChat = document.querySelector('#enable-chat');
const disableChat = document.querySelector('#disable-chat');
const copy = document.querySelector('#copy');
const copyStatus = document.querySelector('#copy-status');

const BRIDGE_INSTRUCTIONS = `Use the chaOS browser bridge for local-machine I/O in this chat.

When local I/O is required, emit exactly one action with no surrounding prose:

CHAOS_ACTION_BEGIN
{"id":"<unique-id>","action":"<action>","args":{}}
CHAOS_ACTION_END

Then stop and wait for a CHAOS_RESULT_BEGIN / CHAOS_RESULT_END receipt before continuing or claiming the action succeeded.

Available actions:
- health
- fs.list
- fs.read
- fs.search
- fs.write
- fs.replace
- proc.run
- proc.read
- proc.write
- proc.stop

Call health first to learn the approved virtual roots, capability fingerprint and live permissions. Prefer direct file/process actions over delegating to another coding harness. Never invent a result and never repeat a successful mutation merely because a previous receipt was not understood.

For retained processes: keep the process_id from proc.run. proc.run may also return a process_token. The starting browser document can use its process directly; after a reload/new browser document, include that process_token in proc.read/proc.write/proc.stop args to recover the retained handle. Never invent or guess a process token.`;

function paintDisconnected(message = 'Native host is stopped for this extension session.') {
  state.textContent = 'Not connected';
  state.className = 'state bad';
  host.textContent = 'Stopped';
  detail.textContent = message;
}

async function activeChatTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id) throw new Error('No active browser tab');
  return tab.id;
}

async function chatMessage(message) {
  return await chrome.tabs.sendMessage(await activeChatTab(), message);
}

async function updateChatState() {
  try {
    const result = await chatMessage({ type: 'chaos-auto-state' });
    if (result?.enabled) {
      chatState.textContent = `Automatic · ${result.remaining} actions left`;
      chatState.className = 'ok';
    } else {
      chatState.textContent = 'Manual';
      chatState.className = '';
    }
  } catch {
    chatState.textContent = 'Not a ChatGPT tab';
    chatState.className = '';
  }
}

async function update() {
  refresh.disabled = true;
  state.textContent = 'Checking…';
  state.className = 'state';
  host.textContent = 'Checking…';
  detail.textContent = '';
  try {
    const result = await chrome.runtime.sendMessage({ type: 'chaos-status' });
    if (result?.connected) {
      state.textContent = 'Connected';
      state.className = 'state ok';
      host.textContent = 'Running';
      detail.textContent = `Protocol ${result.response?.protocol ?? '?'}. Local I/O is available without an MCP tunnel.`;
    } else {
      paintDisconnected(result?.error || 'Chrome could not reach the chaOS native host.');
    }
  } catch (error) {
    paintDisconnected(String(error?.message || error));
  } finally {
    refresh.disabled = false;
  }
  await updateChatState();
}

refresh.addEventListener('click', () => void update());
disconnect.addEventListener('click', async () => {
  disconnect.disabled = true;
  try {
    await chrome.runtime.sendMessage({ type: 'chaos-disconnect' });
    paintDisconnected();
  } finally {
    disconnect.disabled = false;
  }
});
enableChat.addEventListener('click', async () => {
  enableChat.disabled = true;
  copyStatus.textContent = '';
  try {
    const result = await chatMessage({ type: 'chaos-auto-enable' });
    if (!result?.ok) throw new Error(result?.error || 'Could not enable this chat');
    chatState.textContent = `Automatic · ${result.remaining} actions left`;
    chatState.className = 'ok';
    copyStatus.textContent = 'Automatic bridge enabled only for this browser document. Reloading the page disables it; retained processes can be recovered with their process_token.';
  } catch (error) {
    copyStatus.textContent = `Could not enable this chat: ${String(error?.message || error)}`;
  } finally {
    enableChat.disabled = false;
  }
});
disableChat.addEventListener('click', async () => {
  disableChat.disabled = true;
  try {
    await chatMessage({ type: 'chaos-auto-disable' });
    chatState.textContent = 'Manual';
    chatState.className = '';
    copyStatus.textContent = 'Automatic bridge disabled for this browser document.';
  } catch (error) {
    copyStatus.textContent = `Could not disable this chat: ${String(error?.message || error)}`;
  } finally {
    disableChat.disabled = false;
  }
});
copy.addEventListener('click', async () => {
  try {
    await navigator.clipboard.writeText(BRIDGE_INSTRUCTIONS);
    copyStatus.textContent = 'Manual bridge instructions copied.';
  } catch (error) {
    copyStatus.textContent = `Could not copy: ${String(error?.message || error)}`;
  }
});

void update();
