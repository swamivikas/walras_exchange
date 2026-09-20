/**
 * WALRAS EXCHANGE - NETWORK OPERATIONS CENTER (NOC) WAR ROOM CONTROLLER
 * Real-time Go Hot-RAM Telemetry & Clearinghouse Operations Engine
 */

// Global state
const STATE = {
  isTapePaused: false,
  pollInterval: 1500,
  telemetryData: null,
  settlementData: null
};

// DOM References
const DOM = {
  clockDisplay: document.getElementById('clockDisplay'),
  statusLed: document.getElementById('statusLed'),
  statusLabel: document.getElementById('statusLabel'),
  statusPing: document.getElementById('statusPing'),
  btnManualRefresh: document.getElementById('btnManualRefresh'),
  statOnlineNodes: document.getElementById('statOnlineNodes'),
  statNodesSub: document.getElementById('statNodesSub'),
  statInFlight: document.getElementById('statInFlight'),
  flightBadge: document.getElementById('flightBadge'),
  statFlightSub: document.getElementById('statFlightSub'),
  statMaxCapacity: document.getElementById('statMaxCapacity'),
  statSlotsSub: document.getElementById('statSlotsSub'),
  statUtilization: document.getElementById('statUtilization'),
  statUtilBar: document.getElementById('statUtilBar'),
  statNetSpread: document.getElementById('statNetSpread'),
  poolsContainer: document.getElementById('poolsContainer'),
  settlementTapeBody: document.getElementById('settlementTapeBody'),
  tapeCountBadge: document.getElementById('tapeCountBadge'),
  btnPauseTape: document.getElementById('btnPauseTape'),
  toastContainer: document.getElementById('toastContainer')
};

// Initialize War Room Engine
document.addEventListener('DOMContentLoaded', () => {
  initClock();
  setupEventListeners();
  fetchTelemetry();
  fetchSettlementTape();

  // High-frequency polling loops
  setInterval(fetchTelemetry, STATE.pollInterval);
  setInterval(fetchSettlementTape, STATE.pollInterval);
});

function initClock() {
  function updateTime() {
    const now = new Date();
    const h = String(now.getHours()).padStart(2, '0');
    const m = String(now.getMinutes()).padStart(2, '0');
    const s = String(now.getSeconds()).padStart(2, '0');
    const ms = String(Math.floor(now.getMilliseconds() / 10)).padStart(2, '0');
    DOM.clockDisplay.textContent = `${h}:${m}:${s}.${ms}`;
  }
  updateTime();
  setInterval(updateTime, 50);
}

function setupEventListeners() {
  // Manual refresh
  DOM.btnManualRefresh.addEventListener('click', () => {
    fetchTelemetry();
    fetchSettlementTape();
    showToast('Hot-RAM Telemetry Synced');
  });

  // Tape Pause / Resume
  DOM.btnPauseTape.addEventListener('click', () => {
    STATE.isTapePaused = !STATE.isTapePaused;
    DOM.btnPauseTape.textContent = STATE.isTapePaused ? 'RESUME TAPE' : 'PAUSE TAPE';
    DOM.btnPauseTape.style.color = STATE.isTapePaused ? 'var(--color-amber)' : 'var(--text-secondary)';
  });
}

// --------------------------------------------------------------------------
// TELEMETRY FETCH & RENDER
// --------------------------------------------------------------------------
async function fetchTelemetry() {
  const startTime = performance.now();
  try {
    const res = await fetch('/v1/ops/telemetry');
    const latency = Math.round(performance.now() - startTime);

    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();
    STATE.telemetryData = data;

    // Update Connection State
    DOM.statusLed.classList.remove('error');
    DOM.statusLed.classList.add('online');
    DOM.statusLabel.textContent = 'GATEWAY CONNECTED';
    DOM.statusPing.textContent = `${latency}ms`;

    // Extract exact variable names from Go Hot-RAM
    const totalNodes = data.total_online_nodes ?? data.total_nodes_online ?? (data.nodes ? data.nodes.length : 0);
    const inFlight = data.total_in_flight_jobs ?? 0;
    const maxCapacity = data.total_network_capacity ?? data.total_max_capacity ?? 0;
    const availableSlots = Math.max(0, maxCapacity - inFlight);

    // Render Global Statistics
    DOM.statOnlineNodes.textContent = totalNodes;
    DOM.statNodesSub.textContent = `${totalNodes} Registered in Hot RAM`;

    DOM.statInFlight.textContent = inFlight;
    DOM.statMaxCapacity.textContent = maxCapacity;
    DOM.statSlotsSub.textContent = `${availableSlots} slots available`;
    
    const utilPct = (data.global_utilization_pct || 0).toFixed(1);
    DOM.statUtilization.textContent = `${utilPct}%`;
    DOM.statUtilBar.style.width = `${Math.min(utilPct, 100)}%`;

    if (inFlight > 0) {
      DOM.flightBadge.textContent = 'EXECUTING';
      DOM.flightBadge.className = 'stat-badge computing';
      DOM.statFlightSub.textContent = `${inFlight} active GPU job${inFlight > 1 ? 's' : ''} in flight`;
    } else {
      DOM.flightBadge.textContent = 'IDLE';
      DOM.flightBadge.className = 'stat-badge';
      DOM.statFlightSub.textContent = '0 active executions';
    }

    // Render Domain Pools
    renderDomainPools(data.domain_pools || {});

  } catch (err) {
    DOM.statusLed.classList.add('error');
    DOM.statusLed.classList.remove('online');
    DOM.statusLabel.textContent = 'RECONNECTING...';
    DOM.statusPing.textContent = 'TIMEOUT';
  }
}

function renderDomainPools(domainPools) {
  const defaultDomains = [
    { key: 'MATH_SOLVER', title: 'MATH SOLVER POOL', model: 'DeepSeek-R1 / Symbolic', price: '$0.0003 / call' },
    { key: 'CODE_GENERATOR', title: 'CODE GENERATOR POOL', model: 'Claude 3.5 Sonnet / Vibe', price: '$0.0005 / call' },
    { key: 'OCR_DOCUMENT_PARSER', title: 'OCR DOCUMENT PARSER', model: 'Vision-OCR / Multimodal', price: '$0.0004 / call' },
    { key: 'GENERAL_LLM_INFERENCE', title: 'GENERAL LLM POOL', model: 'Llama 3.3 70B / Fast Chat', price: '$0.0002 / call' },
  ];

  DOM.poolsContainer.innerHTML = '';

  defaultDomains.forEach(domainMeta => {
    const stats = domainPools[domainMeta.key] || {
      online_nodes: 0,
      in_flight: 0,
      active_in_flight: 0,
      max_capacity: 0,
      total_capacity: 0,
      available_slots: 0,
      utilization_pct: 0
    };

    const isOnline = stats.online_nodes > 0;
    const inFlightCount = stats.in_flight ?? stats.active_in_flight ?? 0;
    const capacityCount = stats.total_capacity ?? stats.max_capacity ?? 0;
    const availCount = stats.available_slots !== undefined ? stats.available_slots : Math.max(0, capacityCount - inFlightCount);
    const util = (stats.utilization_pct || 0).toFixed(1);

    // Color gradient for gauge
    let gaugeColor = 'var(--color-emerald)';
    if (stats.utilization_pct > 75) gaugeColor = 'var(--color-danger)';
    else if (stats.utilization_pct > 30) gaugeColor = 'var(--color-cyan)';

    const card = document.createElement('div');
    card.className = 'pool-card';
    card.innerHTML = `
      <div class="pool-card-header">
        <div class="pool-title-group">
          <span class="pool-domain">${domainMeta.title}</span>
          <span class="pool-model">${domainMeta.model}</span>
        </div>
        <span class="pool-health-badge ${isOnline ? 'operational' : 'idle'}">
          ● ${isOnline ? 'OPERATIONAL' : 'STANDBY'}
        </span>
      </div>

      <div class="pool-metrics-row">
        <div class="pool-metric">
          <span class="metric-label">NODES</span>
          <span class="metric-val ${isOnline ? 'glow-emerald' : ''}">${stats.online_nodes}</span>
        </div>
        <div class="pool-metric">
          <span class="metric-label">IN-FLIGHT</span>
          <span class="metric-val ${inFlightCount > 0 ? 'glow-cyan' : ''}">${inFlightCount}</span>
        </div>
        <div class="pool-metric">
          <span class="metric-label">AVAILABLE</span>
          <span class="metric-val ${availCount > 0 ? 'glow-emerald' : ''}">${availCount}</span>
        </div>
        <div class="pool-metric">
          <span class="metric-label">CAPACITY</span>
          <span class="metric-val">${capacityCount}</span>
        </div>
      </div>

      <div class="pool-gauge-container">
        <div class="gauge-header">
          <span>Pool Load</span>
          <span style="color: ${gaugeColor}; font-family: var(--font-mono); font-weight: 700;">${util}% (${inFlightCount} busy / ${availCount} free)</span>
        </div>
        <div class="gauge-track">
          <div class="gauge-fill" style="width: ${Math.min(util, 100)}%; background: ${gaugeColor};"></div>
        </div>
      </div>
    `;

    DOM.poolsContainer.appendChild(card);
  });
}

// --------------------------------------------------------------------------
// SETTLEMENT TAPE FETCH & RENDER
// --------------------------------------------------------------------------
async function fetchSettlementTape() {
  if (STATE.isTapePaused) return;

  try {
    const res = await fetch('/v1/ops/settlement-tape');
    if (!res.ok) return;
    const data = await res.json();
    STATE.settlementData = data;

    const records = data.transactions || [];
    DOM.tapeCountBadge.textContent = `${records.length} Cleared`;

    // Sum total platform net profit
    let totalProfitDollars = 0;
    records.forEach(r => {
      const cleanProfit = parseFloat(String(r.net_profit_usd || '0').replace('$', '')) || 0;
      totalProfitDollars += cleanProfit;
    });
    DOM.statNetSpread.textContent = `$${totalProfitDollars.toFixed(4)}`;

    DOM.settlementTapeBody.innerHTML = '';

    if (records.length === 0) {
      DOM.settlementTapeBody.innerHTML = `
        <tr>
          <td colspan="9" style="text-align: center; color: var(--text-muted); padding: 24px;">
            No transactions cleared yet. Dispatch compute jobs to populate settlement tape.
          </td>
        </tr>
      `;
      return;
    }

    records.forEach(tx => {
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <td style="color: var(--text-muted);">${tx.created_at}</td>
        <td style="color: var(--color-cyan);">${tx.tx_id.substring(0, 14)}...</td>
        <td style="color: #fff; font-weight: 600;">${tx.buyer_name}</td>
        <td style="color: var(--text-secondary);">${tx.seller_node_name}</td>
        <td style="color: var(--color-emerald);">${tx.latency_ms}ms</td>
        <td>${tx.total_tokens} <span style="font-size: 9px; color: var(--text-muted);">(${tx.prompt_tokens}/${tx.completion_tokens})</span></td>
        <td style="color: #fff; font-weight: 700;">${tx.total_charged_usd}</td>
        <td style="color: var(--color-amber); font-weight: 700;">${tx.net_profit_usd}</td>
        <td><span class="badge-status-cleared">● ${tx.status}</span></td>
      `;
      DOM.settlementTapeBody.appendChild(tr);
    });

  } catch (err) {
    console.warn('Tape poll error:', err);
  }
}

// --------------------------------------------------------------------------
// TOAST NOTIFICATIONS
// --------------------------------------------------------------------------
function showToast(message, isDanger = false) {
  const toast = document.createElement('div');
  toast.className = `toast-msg ${isDanger ? 'danger' : ''}`;
  toast.textContent = `[NOC] ${message}`;
  DOM.toastContainer.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}
