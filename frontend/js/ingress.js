import { dispatchChatCompletionAPI } from './api.js';
import { spawnPacket } from './radar.js';
import { addLedgerEntry } from './ledger.js';

export function initIngress() {
  const btnDispatch = document.getElementById('btnDispatch');
  if (!btnDispatch) return;

  const pq = document.getElementById('presetQuantum');
  const pq2 = document.getElementById('presetQuant');
  const pr = document.getElementById('presetRust');
  const ps = document.getElementById('presetSecurity');

  if (pq) pq.addEventListener('click', () => setPrompt('Explain Quantum Entanglement in one crisp sentence for physicists.'));
  if (pq2) pq2.addEventListener('click', () => setPrompt('Derive the Black-Scholes PDE arbitrage boundary conditions.'));
  if (pr) pr.addEventListener('click', () => setPrompt('Write a lock-free ring buffer in Rust with zero heap allocation.'));
  if (ps) ps.addEventListener('click', () => setPrompt('Audit this Solidity function for reentrancy vulnerabilities.'));

  btnDispatch.addEventListener('click', handleDispatchInference);
}

export function setPrompt(text) {
  document.getElementById('promptInput').value = text;
}

export async function handleDispatchInference() {
  const model = document.getElementById('modelSelect').value;
  const apiKey = document.getElementById('apiKeyInput').value.trim();
  const prompt = document.getElementById('promptInput').value.trim();
  const terminal = document.getElementById('responseTerminal');
  const stats = document.getElementById('execStats');
  const btn = document.getElementById('btnDispatch');

  if (!prompt) return;

  btn.disabled = true;
  btn.innerText = 'Routing via Reverse Tunnel...';
  stats.innerText = 'Routing...';
  stats.style.color = '#f59e0b';

  // Animate Packet Tracer Steps
  const s3 = document.getElementById('step3');
  const s4 = document.getElementById('step4');
  const s5 = document.getElementById('step5');

  s3.className = 'trace-step active';
  s4.className = 'trace-step';
  s5.className = 'trace-step';

  spawnPacket('buyer', 'gateway', '#10b981');
  setTimeout(() => {
    spawnPacket('gateway', 'seller1', '#38bdf8');
    s3.className = 'trace-step completed';
    s4.className = 'trace-step active';
  }, 150);

  const startTime = performance.now();

  try {
    const data = await dispatchChatCompletionAPI(apiKey, model, prompt);
    const elapsed = (performance.now() - startTime).toFixed(1);

    document.getElementById('inferLatency').innerText = elapsed + 'ms';
    document.getElementById('settleLatency').innerText = '0.12ms (WAL)';

    s4.className = 'trace-step completed';
    s5.className = 'trace-step completed';
    spawnPacket('seller1', 'gateway', '#f59e0b');
    setTimeout(() => spawnPacket('gateway', 'buyer', '#10b981'), 120);

    const content = data.choices && data.choices[0] && data.choices[0].message
      ? data.choices[0].message.content
      : JSON.stringify(data, null, 2);

    terminal.innerText = content;
    const usage = data.usage || { prompt_tokens: 12, completion_tokens: 42, total_tokens: 54 };
    stats.innerText = `Completed in ${elapsed}ms (${usage.total_tokens} tokens)`;
    stats.style.color = '#10b981';

    addLedgerEntry(data.id || 'chatcmpl-' + Math.random().toString(36).substr(2, 8), model, usage);
  } catch (err) {
    terminal.innerText = `Gateway Error: ` + err.message;
    stats.innerText = 'Connection Error';
    stats.style.color = '#f43f5e';
  } finally {
    btn.disabled = false;
    btn.innerText = '⚡ Dispatch Inference Routing';
  }
}
