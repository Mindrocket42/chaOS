const ACTION_BEGIN = 'CHAOS_ACTION_BEGIN';
const ACTION_END = 'CHAOS_ACTION_END';
const RESULT_BEGIN = 'CHAOS_RESULT_BEGIN';
const RESULT_END = 'CHAOS_RESULT_END';
const ALLOWED_ACTIONS = new Set([
  'health', 'fs.list', 'fs.read', 'fs.search', 'fs.write', 'fs.replace',
  'proc.run', 'proc.read', 'proc.write', 'proc.stop'
]);

const rendered = new WeakSet();
const processed = new Set();
let autoEnabled = false;
let autoNonce = null;
let autoBudget = 0;
let scanTimer = null;
let sending = false;

function isGenerating() {
  return Boolean(
    document.querySelector('[data-testid="stop-button"]') ||
    [...document.querySelectorAll('button')].some((button) => /stop (generating|response)/i.test(button.getAttribute('aria-label') || ''))
  );
}

function composer() {
  return document.querySelector('#prompt-textarea') ||
    document.querySelector('textarea[placeholder]') ||
    document.querySelector('[contenteditable="true"][role="textbox"]');
}

function setComposerText(node, text) {
  node.focus();
  if (node instanceof HTMLTextAreaElement || node instanceof HTMLInputElement) {
    const prototype = Object.getPrototypeOf(node);
    const setter = Object.getOwnPropertyDescriptor(prototype, 'value')?.set;
    if (setter) setter.call(node, text);
    else node.value = text;
    node.dispatchEvent(new Event('input', { bubbles: true }));
    return;
  }

  const selection = window.getSelection();
  const range = document.createRange();
  range.selectNodeContents(node);
  selection?.removeAllRanges();
  selection?.addRange(range);
  if (!document.execCommand('insertText', false, text)) {
    node.textContent = text;
    node.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: text }));
  }
}

async function waitForSendButton(timeoutMs = 10000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const button = document.querySelector('button[data-testid="send-button"]') ||
      [...document.querySelectorAll('button')].find((candidate) => /^send/i.test(candidate.getAttribute('aria-label') || ''));
    if (button && !button.disabled) return button;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  return null;
}

async function sendChatMessage(text) {
  if (sending) throw new Error('chaOS is already sending a bridge message');
  if (isGenerating()) throw new Error('ChatGPT is still generating');
  sending = true;
  try {
    const input = composer();
    if (!input) throw new Error('Could not find the ChatGPT composer');
    setComposerText(input, text);
    const button = await waitForSendButton();
    if (!button) throw new Error('Could not find an enabled ChatGPT Send button');
    button.click();
  } finally {
    sending = false;
  }
}

function bridgeInstructions(nonce) {
  return `chaOS local bridge is enabled for this chat. This is a transport instruction, not a new task.\n\n` +
    `When the current task needs local-machine I/O, output exactly one action and no surrounding prose:\n\n` +
    `${ACTION_BEGIN}\n` +
    `{"nonce":"${nonce}","id":"<unique-id>","action":"<action>","args":{}}\n` +
    `${ACTION_END}\n\n` +
    `Then stop. The extension will execute that action and send back a ${RESULT_BEGIN}/${RESULT_END} receipt as the next user message. ` +
    `Continue from the receipt. Repeat only when another local action is genuinely required.\n\n` +
    `Available actions: health, fs.list, fs.read, fs.search, fs.write, fs.replace, proc.run, proc.read, proc.write, proc.stop. ` +
    `Call health before the first local operation to learn the approved virtual roots and live permissions. ` +
    `Prefer direct file/process actions over delegating to another harness. Never invent a receipt. Never claim an action succeeded before its receipt. ` +
    `Never reuse a successful action id. When the task is complete, answer normally and emit no chaOS action.`;
}

function parseAction(text) {
  const start = text.indexOf(ACTION_BEGIN);
  if (start === -1) return null;
  const end = text.indexOf(ACTION_END, start + ACTION_BEGIN.length);
  if (end === -1) return null;
  try {
    const value = JSON.parse(text.slice(start + ACTION_BEGIN.length, end).trim());
    if (!value || typeof value !== 'object') return null;
    if (typeof value.id !== 'string' || typeof value.action !== 'string') return null;
    if (!ALLOWED_ACTIONS.has(value.action)) return null;
    return value;
  } catch {
    return null;
  }
}

function resultText(action, response) {
  const envelope = { nonce: action.nonce || null, id: action.id, response };
  return `${RESULT_BEGIN}\n${JSON.stringify(envelope)}\n${RESULT_END}`;
}

function addStatus(message, text) {
  const status = document.createElement('div');
  status.dataset.chaosBridgeStatus = 'true';
  status.style.cssText = 'margin:8px 0;padding:6px 8px;border:1px solid currentColor;border-radius:8px;font:12px/1.4 system-ui;opacity:.85;';
  status.textContent = text;
  message.append(status);
  return status;
}

async function executeAction(message, action, automatic) {
  if (processed.has(action.id)) return;
  processed.add(action.id);
  const status = addStatus(message, automatic ? `chaOS: running ${action.action}…` : `chaOS action: ${action.action}`);
  try {
    const response = await chrome.runtime.sendMessage({
      type: 'chaos-call',
      call: { id: action.id, action: action.action, args: action.args || {} }
    });
    const text = resultText(action, response);
    if (automatic) {
      status.textContent = response?.ok === true ? `chaOS: ${action.action} complete; returning receipt…` : `chaOS: ${action.action} failed; returning receipt…`;
      await sendChatMessage(text);
      status.textContent = response?.ok === true ? `chaOS: ${action.action} receipt returned` : `chaOS: ${action.action} failure returned`;
    } else {
      status.textContent = text;
      status.style.whiteSpace = 'pre-wrap';
      const copy = document.createElement('button');
      copy.type = 'button';
      copy.textContent = 'Copy result';
      copy.style.cssText = 'display:block;margin-top:6px;padding:4px 8px;';
      copy.addEventListener('click', () => void navigator.clipboard.writeText(text));
      status.append(copy);
    }
  } catch (error) {
    status.textContent = `chaOS bridge error: ${String(error?.message || error)}`;
    processed.delete(action.id);
  }
}

function addManualControl(message, action) {
  if (rendered.has(message)) return;
  rendered.add(message);
  const box = addStatus(message, `chaOS action: ${action.action}`);
  const run = document.createElement('button');
  run.type = 'button';
  run.textContent = 'Run with chaOS';
  run.style.cssText = 'display:block;margin-top:6px;padding:4px 8px;';
  box.append(run);
  run.addEventListener('click', async () => {
    run.disabled = true;
    box.remove();
    rendered.delete?.(message);
    try { await executeAction(message, action, false); }
    catch { run.disabled = false; }
  });
}

async function scan() {
  scanTimer = null;
  if (sending || isGenerating()) return;
  const messages = [...document.querySelectorAll('[data-message-author-role="assistant"]')].slice(-16).reverse();
  for (const message of messages) {
    const action = parseAction(message.innerText || message.textContent || '');
    if (!action || processed.has(action.id)) continue;
    if (autoEnabled) {
      if (action.nonce !== autoNonce) {
        if (!rendered.has(message)) {
          rendered.add(message);
          addStatus(message, 'chaOS: action ignored — this chat does not hold the active bridge nonce');
        }
        continue;
      }
      if (autoBudget <= 0) {
        autoEnabled = false;
        if (!rendered.has(message)) {
          rendered.add(message);
          addStatus(message, 'chaOS: automatic bridge stopped after 50 actions; re-enable it explicitly to continue');
        }
        continue;
      }
      autoBudget -= 1;
      await executeAction(message, action, true);
      return;
    }
    addManualControl(message, action);
  }
}

function scheduleScan() {
  if (scanTimer) return;
  scanTimer = setTimeout(() => void scan(), 500);
}

new MutationObserver(scheduleScan).observe(document.documentElement, { subtree: true, childList: true, characterData: true });
scan();

chrome.runtime.onMessage.addListener((message, _sender, respond) => {
  if (message?.type === 'chaos-auto-state') {
    respond({ ok: true, enabled: autoEnabled, remaining: autoBudget });
    return false;
  }
  if (message?.type === 'chaos-auto-disable') {
    autoEnabled = false;
    autoNonce = null;
    autoBudget = 0;
    respond({ ok: true, enabled: false });
    return false;
  }
  if (message?.type === 'chaos-auto-enable') {
    void (async () => {
      if (isGenerating()) throw new Error('Wait for the current ChatGPT answer to finish before enabling chaOS');
      autoNonce = crypto.randomUUID();
      autoBudget = 50;
      autoEnabled = true;
      await sendChatMessage(bridgeInstructions(autoNonce));
      respond({ ok: true, enabled: true, remaining: autoBudget });
    })().catch((error) => {
      autoEnabled = false;
      autoNonce = null;
      autoBudget = 0;
      respond({ ok: false, error: String(error?.message || error) });
    });
    return true;
  }
  return false;
});
