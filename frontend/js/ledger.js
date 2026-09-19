import { state } from './state.js';

export function addLedgerEntry(reqId, model, usage) {
  const tbody = document.getElementById('ledgerBody');
  if (!tbody) return;

  const tr = document.createElement('tr');
  const now = new Date().toTimeString().split(' ')[0] + '.' + Math.floor(Math.random() * 900 + 100);

  const inPrice = 2; // microUSD
  const outPrice = 2;
  const gross = ((usage.prompt_tokens * inPrice + usage.completion_tokens * outPrice) / 1000000).toFixed(6);
  const fee = (gross * 0.01).toFixed(6);
  const surcharge = (gross * 0.03).toFixed(6);

  tr.innerHTML = `
    <td>${now}</td>
    <td style="color: var(--accent-cyan);">${reqId}</td>
    <td>${state.currentUser ? state.currentUser.name : 'Enterprise'}</td>
    <td>83cd96bc.. (RTX 4090)</td>
    <td>${model}</td>
    <td>${usage.prompt_tokens} / ${usage.completion_tokens}</td>
    <td>$${gross}</td>
    <td><span class="fee-pill">-$${fee}</span></td>
    <td><span class="fee-pill" style="color: var(--accent-emerald);">+$${surcharge}</span></td>
    <td style="color: var(--accent-emerald);">✅ SETTLED</td>
  `;
  tbody.insertBefore(tr, tbody.firstChild);
}
