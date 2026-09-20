/**
 * WALRAS EXCHANGE - NETWORK OPERATIONS CENTER (NOC) WAR ROOM CONTROLLER
 * Zero-Latency Event-Driven Streaming Engine (<1ms Server-Sent Events SSE Push)
 */

// Global State
const STATE = {
  isTapePaused: false,
  sseConnected: false,
  eventSource: null,
  clearedCount: 0,
  cumulativeProfit: 0,
  seenTxIds: new Set(),
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

  // 1. Initial hydration of historical settlement records from DB
  fetchSettlementTape();

  // 2. Open Persistent Zero-Latency Event Stream (<1ms Push)
  initSSEStream();

  // 3. Resilient Fallback Poller (Active ONLY if SSE drops)
  setInterval(() => {
    if (!STATE.sseConnected) {
      console.warn('[NOC] SSE stream disconnected, executing fallback poll...');
      fetchTelemetryFallback();
    }
  }, 4000);
});

// --------------------------------------------------------------------------
// CLOCK ENGINE
// --------------------------------------------------------------------------
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

// --------------------------------------------------------------------------
// EVENT LISTENERS
// --------------------------------------------------------------------------
function setupEventListeners() {
  // Manual sync button
  DOM.btnManualRefresh.addEventListener('click', () => {
    if (STATE.eventSource) {
      STATE.eventSource.close();
    }
    initSSEStream();
    fetchSettlementTape();
    showToast('Re-establishing Hot-RAM Live Event Stream...');
  });

  // Tape Pause / Resume toggle
  DOM.btnPauseTape.addEventListener('click', () => {
    STATE.isTapePaused = !STATE.isTapePaused;
    DOM.btnPauseTape.textContent = STATE.isTapePaused ? 'RESUME TAPE' : 'PAUSE TAPE';
    DOM.btnPauseTape.style.color = STATE.isTapePaused ? 'var(--color-amber)' : 'var(--text-secondary)';
    showToast(STATE.isTapePaused ? 'Settlement Tape Paused' : 'Settlement Tape Resumed');
  });
}

// --------------------------------------------------------------------------
// PERSISTENT ZERO-LATENCY EVENT STREAM (SSE)
// --------------------------------------------------------------------------
function initSSEStream() {
  try {
    if (STATE.eventSource) {
      STATE.eventSource.close();
    }

    const evtSource = new EventSource('/v1/ops/stream');
    STATE.eventSource = evtSource;

    evtSource.onopen = () => {
      STATE.sseConnected = true;
      DOM.statusLed.className = 'pulsing-led online streaming';
      DOM.statusLabel.textContent = 'STREAMING LIVE (SSE)';
      DOM.statusPing.textContent = '<0.1ms PUSH';
      console.log('[NOC] ⚡ Real-Time SSE Stream Connected. Zero polling active.');
    };

    // Instant Hot-RAM Pool State Changes (Node connect/disconnect, in-flight jobs)
    evtSource.addEventListener('telemetry', (e) => {
      try {
        const data = JSON.parse(e.data);
        applyTelemetry(data);
      } catch (err) {
        console.error('[NOC] Telemetry parse error:', err);
      }
    });

    // Instant Settlement Push (<0.1ms upon job completion)
    evtSource.addEventListener('transaction', (e) => {
      try {
        const tx = JSON.parse(e.data);
        prependSettlementItem(tx);
      } catch (err) {
        console.error('[NOC] Transaction push parse error:', err);
      }
    });

    evtSource.onerror = (err) => {
      STATE.sseConnected = false;
      DOM.statusLed.className = 'pulsing-led error';
      DOM.statusLabel.textContent = 'RECONNECTING STREAM...';
      DOM.statusPing.textContent = 'RETRIEVING';
    };

  } catch (err) {
    console.error('[NOC] Failed to initialize SSE stream:', err);
    STATE.sseConnected = false;
  }
}

// --------------------------------------------------------------------------
// TELEMETRY RENDER (HOT-RAM REFLECTION)
// --------------------------------------------------------------------------
function applyTelemetry(data) {
  STATE.telemetryData = data;

  const totalNodes = data.total_online_nodes ?? data.total_nodes_online ?? (data.nodes ? data.nodes.length : 0);
  const inFlight = data.total_in_flight_jobs ?? 0;
  const maxCapacity = data.total_network_capacity ?? data.total_max_capacity ?? 0;
  const availableSlots = Math.max(0, maxCapacity - inFlight);

  // Global Stat Cards
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

  // Render Domain Pool Cards
  renderDomainPools(data.domain_pools || {});
}

async function fetchTelemetryFallback() {
  try {
    const res = await fetch('/v1/ops/telemetry');
    if (res.ok) {
      const data = await res.json();
      applyTelemetry(data);
    }
  } catch (err) {
    // Suppress network fallback errors
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
// REAL-TIME SETTLEMENT TAPE
// --------------------------------------------------------------------------
async function fetchSettlementTape() {
  try {
    const res = await fetch('/v1/ops/settlement-tape');
    if (!res.ok) return;
    const data = await res.json();
    STATE.settlementData = data;

    const records = data.transactions || [];
    STATE.clearedCount = records.length;
    DOM.tapeCountBadge.textContent = `${STATE.clearedCount} Cleared`;

    let totalProfitDollars = 0;
    STATE.seenTxIds.clear();

    DOM.settlementTapeBody.innerHTML = '';

    if (records.length === 0) {
      DOM.settlementTapeBody.innerHTML = `
        <tr id="emptyTapeRow">
          <td colspan="9" style="text-align: center; color: var(--text-muted); padding: 24px;">
            No transactions cleared yet. Dispatch compute jobs to populate settlement tape.
          </td>
        </tr>
      `;
      DOM.statNetSpread.textContent = '$0.0000';
      return;
    }

    records.forEach(tx => {
      STATE.seenTxIds.add(tx.tx_id);
      const cleanProfit = parseFloat(String(tx.net_profit_usd || '0').replace('$', '')) || 0;
      totalProfitDollars += cleanProfit;

      const tr = createTapeRow(tx, false);
      DOM.settlementTapeBody.appendChild(tr);
    });

    STATE.cumulativeProfit = totalProfitDollars;
    DOM.statNetSpread.textContent = `$${totalProfitDollars.toFixed(4)}`;

  } catch (err) {
    console.warn('[NOC] Initial tape fetch error:', err);
  }
}

function prependSettlementItem(tx) {
  if (STATE.isTapePaused) return;

  // Deduplicate
  if (STATE.seenTxIds.has(tx.tx_id)) return;
  STATE.seenTxIds.add(tx.tx_id);

  // Remove empty state placeholder row if present
  const emptyRow = document.getElementById('emptyTapeRow');
  if (emptyRow) {
    emptyRow.remove();
  }

  // Create row with dynamic flash highlight
  const tr = createTapeRow(tx, true);
  DOM.settlementTapeBody.insertBefore(tr, DOM.settlementTapeBody.firstChild);

  // Prune table to prevent unbounded DOM memory
  while (DOM.settlementTapeBody.children.length > 40) {
    DOM.settlementTapeBody.lastElementChild.remove();
  }

  // Update transaction counter
  STATE.clearedCount++;
  DOM.tapeCountBadge.textContent = `${STATE.clearedCount} Cleared`;

  // Update net spread profit
  const cleanProfit = parseFloat(String(tx.net_profit_usd || '0').replace('$', '')) || 0;
  STATE.cumulativeProfit += cleanProfit;
  DOM.statNetSpread.textContent = `$${STATE.cumulativeProfit.toFixed(4)}`;

  // Flash subtle toast notification
  showToast(`TX Settled: ${tx.buyer_name} ➔ ${tx.seller_node_name} (${tx.latency_ms}ms, ${tx.net_profit_usd} spread)`);
}

function createTapeRow(tx, isNew = false) {
  const tr = document.createElement('tr');
  if (isNew) {
    tr.className = 'flash-new-row';
  }

  const shortTx = (tx.tx_id || '').substring(0, 14);
  tr.innerHTML = `
    <td style="color: var(--text-muted);">${tx.created_at || '--:--:--'}</td>
    <td style="color: var(--color-cyan); font-weight: 600;">${shortTx}...</td>
    <td style="color: #fff; font-weight: 600;">${tx.buyer_name || 'Enterprise Buyer'}</td>
    <td style="color: var(--text-secondary);">${tx.seller_node_name || 'Worker Node'}</td>
    <td style="color: var(--color-emerald); font-weight: 700;">${tx.latency_ms}ms</td>
    <td>${tx.total_tokens || 0} <span style="font-size: 9px; color: var(--text-muted);">(${tx.prompt_tokens || 0}/${tx.completion_tokens || 0})</span></td>
    <td style="color: #fff; font-weight: 700;">${tx.total_charged_usd || '$0.0000'}</td>
    <td style="color: var(--color-amber); font-weight: 700;">${tx.net_profit_usd || '$0.0000'}</td>
    <td><span class="badge-status-cleared">● ${tx.status || 'SETTLED'}</span></td>
  `;
  return tr;
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
  }, 3000);
}
