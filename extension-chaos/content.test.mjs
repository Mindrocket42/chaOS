import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { runInNewContext } from 'node:vm';
import { setTimeout as delay } from 'node:timers/promises';
import { JSDOM } from 'jsdom';

const source = await readFile(new URL('./content.js', import.meta.url), 'utf8');

function runtime(markup) {
  const dom = new JSDOM(`<!doctype html><body>${markup}<textarea id="prompt-textarea"></textarea><button data-testid="send-button">Send</button></body>`, {
    url: 'https://chatgpt.com/c/test',
    runScripts: 'outside-only'
  });
  const calls = [];
  const listeners = [];
  dom.window.chrome = {
    runtime: {
      onMessage: { addListener: listener => listeners.push(listener) },
      sendMessage: async message => {
        calls.push(message);
        return { id: message.call?.id, ok: true, result: { connection: 'connected' } };
      }
    }
  };
  runInNewContext(source, {
    document: dom.window.document,
    window: dom.window,
    chrome: dom.window.chrome,
    MutationObserver: dom.window.MutationObserver,
    HTMLElement: dom.window.HTMLElement,
    HTMLTextAreaElement: dom.window.HTMLTextAreaElement,
    HTMLInputElement: dom.window.HTMLInputElement,
    Event: dom.window.Event,
    InputEvent: dom.window.InputEvent,
    setTimeout,
    clearTimeout,
    crypto: { randomUUID: () => 'test-nonce' }
  });
  return { dom, calls, listeners };
}

const envelope = nonce => `CHAOS_ACTION_BEGIN {"nonce":"${nonce}","id":"health-test-1","action":"health","args":{}} CHAOS_ACTION_END`;

{
  const { dom, calls, listeners } = runtime('');
  let enabled;
  const enabledResult = new Promise(resolve => { enabled = resolve; });
  listeners[0]({ type: 'chaos-auto-enable' }, {}, enabled);
  assert.equal((await enabledResult).enabled, true, 'automatic mode should enable for this test document');

  const assistant = dom.window.document.createElement('div');
  assistant.setAttribute('data-markdown-text-style', 'assistant-message');
  assistant.textContent = envelope('test-nonce');
  dom.window.document.body.prepend(assistant);
  await delay(550);
  assert.equal(calls.length, 1, 'current ChatGPT assistant markdown should dispatch one health call');
  assert.equal(calls[0].call.action, 'health');
  assert.match(dom.window.document.querySelector('[data-chaos-bridge-status="true"]')?.textContent ?? '', /receipt returned/,
    'automatic execution should return the native receipt');

  const user = dom.window.document.createElement('div');
  user.setAttribute('data-user-message-bubble', 'true');
  user.textContent = envelope('test-nonce');
  dom.window.document.body.prepend(user);
  await delay(550);
  assert.equal(calls.length, 1, 'an action-shaped user message must not dispatch local actions');
  assert.equal(user.querySelector('[data-chaos-bridge-status="true"]'), null,
    'an action-shaped user message must not be decorated as assistant output');
  dom.window.close();
}

{
  const { dom, calls, listeners } = runtime(`<div data-message-author-role="assistant">${envelope('test-nonce')}</div>`);
  await delay(550);
  const status = dom.window.document.querySelector('[data-chaos-bridge-status="true"]');
  assert.match(status?.textContent ?? '', /Run with chaOS/,
    'the legacy assistant role selector must remain supported');
  assert.equal(calls.length, 0, 'legacy manual mode must not dispatch without an explicit click');
  dom.window.close();
}
