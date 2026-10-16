const state = document.querySelector('#state');
const host = document.querySelector('#host');
const detail = document.querySelector('#detail');
const refresh = document.querySelector('#refresh');
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

Call health first to learn the approved virtual roots and live permissions. Prefer direct file/process actions over delegating to another coding harness. Never invent a result and never repeat a successful mutation merely because a previous receipt was not understood.`;

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
      state.textContent = 'Not connected';
      state.className = 'state bad';
      host.textContent = 'Unavailable';
      detail.textContent = result?.error || 'Chrome could not reach the chaOS native host.';
    }
  } catch (error) {
    state.textContent = 'Not connected';
    state.className = 'state bad';
    host.textContent = 'Unavailable';
    detail.textContent = String(error?.message || error);
  } finally {
    refresh.disabled = false;
  }
}

refresh.addEventListener('click', () => void update());
copy.addEventListener('click', async () => {
  try {
    await navigator.clipboard.writeText(BRIDGE_INSTRUCTIONS);
    copyStatus.textContent = 'Bridge instructions copied. Paste them once into the ChatGPT conversation you want chaOS to serve.';
  } catch (error) {
    copyStatus.textContent = `Could not copy: ${String(error?.message || error)}`;
  }
});

void update();
