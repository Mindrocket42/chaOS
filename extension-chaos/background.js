const HOST = 'pro.ironhead.chaos';
let port = null;
const pending = new Map();

function rejectPending(error) {
  for (const entry of pending.values()) {
    clearTimeout(entry.timer);
    entry.reject(error);
  }
  pending.clear();
}

function disconnect() {
  const held = port;
  port = null;
  rejectPending(new Error('chaOS native host disconnected'));
  if (held) {
    try { held.disconnect(); } catch { /* already gone */ }
  }
}

function getPort() {
  if (port) return port;
  port = chrome.runtime.connectNative(HOST);
  const connectedPort = port;
  connectedPort.onMessage.addListener((message) => {
    const entry = message && typeof message.id === 'string' ? pending.get(message.id) : null;
    if (!entry) return;
    pending.delete(message.id);
    clearTimeout(entry.timer);
    entry.resolve(message);
  });
  connectedPort.onDisconnect.addListener(() => {
    if (port === connectedPort) port = null;
    const reason = chrome.runtime.lastError?.message || 'chaOS native host disconnected';
    rejectPending(new Error(reason));
  });
  return connectedPort;
}

function send(request) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      pending.delete(request.id);
      reject(new Error('chaOS native host timeout'));
    }, 60000);
    pending.set(request.id, { resolve, reject, timer });
    try {
      getPort().postMessage(request);
    } catch (error) {
      clearTimeout(timer);
      pending.delete(request.id);
      reject(error);
    }
  });
}

function callerFromSender(sender) {
  const frameId = Number.isInteger(sender?.frameId) ? sender.frameId : 0;
  const tabId = Number.isInteger(sender?.tab?.id) ? sender.tab.id : -1;
  const documentId = typeof sender?.documentId === 'string' ? sender.documentId : '';
  const sourceUrl = sender?.tab?.url || sender?.url || '';
  let host = '';
  try { host = new URL(sourceUrl).host.toLowerCase(); } catch { /* leave blank */ }

  if (tabId >= 0 && documentId && (host === 'chatgpt.com' || host === 'chat.openai.com')) {
    return { surface: 'chatgpt-tab', tab_id: tabId, frame_id: frameId, document_id: documentId, host };
  }
  return { surface: 'extension-page', frame_id: frameId, document_id: documentId, host: chrome.runtime.id };
}

chrome.runtime.onMessage.addListener((message, sender, respond) => {
  if (!message || typeof message !== 'object') return false;
  if (message.type === 'chaos-status') {
    const id = `status-${Date.now()}`;
    send({ type: 'ping', id }).then(
      (response) => respond({ connected: response?.ok === true, response }),
      (error) => respond({ connected: false, error: String(error?.message || error) })
    );
    return true;
  }
  if (message.type === 'chaos-disconnect') {
    disconnect();
    respond({ ok: true, connected: false });
    return false;
  }
  if (message.type === 'chaos-call' && message.call) {
    const call = message.call;
    const caller = callerFromSender(sender);
    send({ type: 'call', id: call.id, action: call.action, args: call.args || {}, caller }).then(
      respond,
      (error) => respond({ type: 'result', id: call.id, ok: false, protocol: 1, error: { code: 'NATIVE_HOST_UNAVAILABLE', message: String(error?.message || error) } })
    );
    return true;
  }
  return false;
});
