const state = document.querySelector('#state');
const host = document.querySelector('#host');
const detail = document.querySelector('#detail');
const refresh = document.querySelector('#refresh');

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
void update();
