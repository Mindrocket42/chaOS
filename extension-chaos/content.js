const ACTION_BEGIN = 'CHAOS_ACTION_BEGIN';
const ACTION_END = 'CHAOS_ACTION_END';
const RESULT_BEGIN = 'CHAOS_RESULT_BEGIN';
const RESULT_END = 'CHAOS_RESULT_END';

const rendered = new WeakSet();

function parseAction(text) {
  const start = text.indexOf(ACTION_BEGIN);
  if (start === -1) return null;
  const end = text.indexOf(ACTION_END, start + ACTION_BEGIN.length);
  if (end === -1) return null;
  try {
    const value = JSON.parse(text.slice(start + ACTION_BEGIN.length, end).trim());
    if (!value || typeof value !== 'object') return null;
    if (typeof value.id !== 'string' || typeof value.action !== 'string') return null;
    return value;
  } catch {
    return null;
  }
}

function resultText(action, response) {
  return `${RESULT_BEGIN}\n${JSON.stringify({ id: action.id, response })}\n${RESULT_END}`;
}

function addActionControl(message, action) {
  if (rendered.has(message)) return;
  rendered.add(message);

  const box = document.createElement('div');
  box.style.cssText = 'margin:8px 0;padding:8px;border:1px solid currentColor;border-radius:8px;font:12px/1.4 system-ui;';

  const label = document.createElement('div');
  label.textContent = `chaOS action: ${action.action}`;
  box.append(label);

  const run = document.createElement('button');
  run.type = 'button';
  run.textContent = 'Run with chaOS';
  run.style.cssText = 'margin-top:6px;padding:4px 8px;';
  box.append(run);

  const output = document.createElement('pre');
  output.style.cssText = 'white-space:pre-wrap;max-height:260px;overflow:auto;margin:8px 0 0;';
  box.append(output);

  run.addEventListener('click', async () => {
    run.disabled = true;
    output.textContent = 'Running…';
    try {
      const response = await chrome.runtime.sendMessage({
        type: 'chaos-call',
        call: { id: action.id, action: action.action, args: action.args || {} }
      });
      const text = resultText(action, response);
      output.textContent = text;
      const copy = document.createElement('button');
      copy.type = 'button';
      copy.textContent = 'Copy result';
      copy.style.cssText = 'margin-top:6px;padding:4px 8px;';
      copy.addEventListener('click', () => void navigator.clipboard.writeText(text));
      box.append(copy);
    } catch (error) {
      output.textContent = `chaOS error: ${String(error?.message || error)}`;
      run.disabled = false;
    }
  });

  message.append(box);
}

function scan() {
  for (const message of document.querySelectorAll('[data-message-author-role="assistant"]')) {
    if (rendered.has(message)) continue;
    const action = parseAction(message.innerText || message.textContent || '');
    if (action) addActionControl(message, action);
  }
}

new MutationObserver(scan).observe(document.documentElement, { subtree: true, childList: true, characterData: true });
scan();
