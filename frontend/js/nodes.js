import { state } from './state.js';
import { getUserAgentsAPI, deploySellerNodeAPI } from './api.js';

export function initNodes() {
  const btnOpen = document.getElementById('btnOpenDeploy');
  const btnClose = document.getElementById('btnCloseDeploy');
  const btnSubmit = document.getElementById('btnSubmitDeploy');
  const btnCopyAndClose = document.getElementById('btnCopyAndCloseDeploy');
  const btnDone = document.getElementById('btnCloseDeployDone');

  if (btnOpen) btnOpen.addEventListener('click', openDeployModal);
  if (btnClose) btnClose.addEventListener('click', closeDeployModal);
  if (btnSubmit) btnSubmit.addEventListener('click', handleSubmitNewSellerAgent);

  if (btnCopyAndClose) {
    btnCopyAndClose.addEventListener('click', () => {
      const rawKey = document.getElementById('rawKeyOutput').innerText;
      if (rawKey && rawKey !== 'walras_live_org_...') {
        navigator.clipboard.writeText(rawKey).then(() => {
          btnCopyAndClose.innerText = '✓ Key Copied!';
          setTimeout(() => {
            closeDeployModal();
            btnCopyAndClose.innerText = '📋 Copy Key & Return to Dashboard';
          }, 600);
        });
      } else {
        closeDeployModal();
      }
    });
  }

  if (btnDone) {
    btnDone.addEventListener('click', closeDeployModal);
  }
}

export function openDeployModal() {
  if (state.currentUser && state.currentUser.user_type === 'ENTERPRISE_BUYER') {
    return; // Buyer account disabled
  }
  document.getElementById('deployNodeModal').classList.remove('hidden');
  document.getElementById('newKeyReveal').classList.add('hidden');
}

export function closeDeployModal() {
  document.getElementById('deployNodeModal').classList.add('hidden');
  document.getElementById('newKeyReveal').classList.add('hidden');
  fetchAndRenderAgents();
}

export async function fetchAndRenderAgents() {
  if (!state.currentUser || !state.currentUser.owner_id) return;
  const container = document.getElementById('userAgentList');
  const btnOpen = document.getElementById('btnOpenDeploy');
  const isBuyer = state.currentUser.user_type === 'ENTERPRISE_BUYER';

  // Apply visual blurring and tooltip if Buyer account
  if (btnOpen) {
    if (isBuyer) {
      btnOpen.style.filter = 'blur(1px) opacity(0.45)';
      btnOpen.style.cursor = 'not-allowed';
      btnOpen.style.background = 'rgba(255, 255, 255, 0.08)';
      btnOpen.style.borderColor = 'rgba(255, 255, 255, 0.15)';
      btnOpen.style.boxShadow = 'none';
      btnOpen.title = '🔒 Buyer Account: Node hosting disabled. Switch or register as an Agent Seller to monetize agents.';
    } else {
      btnOpen.style.filter = 'none';
      btnOpen.style.cursor = 'pointer';
      btnOpen.style.background = 'linear-gradient(135deg, #0284c7 0%, #2563eb 50%, #4f46e5 100%)';
      btnOpen.style.borderColor = 'rgba(255, 255, 255, 0.15)';
      btnOpen.style.boxShadow = '0 4px 14px rgba(37, 99, 235, 0.35)';
      btnOpen.title = 'Deploy a new autonomous agent or MCP tool server';
    }
  }

  try {
    const agents = await getUserAgentsAPI(state.currentUser.owner_id);
    state.activeNodes = agents;

    if (!agents || agents.length === 0) {
      if (isBuyer) {
        container.innerHTML = `
          <tr>
            <td colspan="7" style="padding: 24px; text-align: center; color: var(--text-secondary); font-size: 13px;">
              <span style="color: var(--accent-cyan); font-weight: 700;">🏢 Enterprise Buyer Mode Active:</span> Your account is configured to consume AI agents &amp; MCP tools via API. Node deployment and monetized hosting is active for <b>Agent Seller</b> and <b>Hybrid</b> accounts.
            </td>
          </tr>
        `;
      } else {
        container.innerHTML = `
          <tr>
            <td colspan="7" style="padding: 24px; text-align: center; color: var(--text-secondary); font-size: 13px;">
              <span style="color: var(--accent-emerald); font-weight: 700;">⚡ Agent Seller Fleet Ready:</span> No custom agents connected yet. Click <b>"+ Add Agent / Node"</b> above to register your first model or MCP tool server.
            </td>
          </tr>
        `;
      }
      return;
    }

    container.innerHTML = agents.map(a => {
      const isSeller = a.role === 'SELLER';
      const statusClass = a.approval_status === 'APPROVED' ? 'status-tag approved' : 'status-tag pending';
      const modelDisplayName = a.model || (isSeller ? 'deepseek-r1' : 'General Buyer Ingress');
      const vramInfo = a.vram_gb ? `${a.vram_gb}GB VRAM` : '24GB VRAM';
      const roleLabel = isSeller ? '⚡ SELLER' : '🏢 BUYER';
      const keyPrefix = a.api_key_prefix || 'walras_live_org_...';
      const rawId = a.agent_id || a.id || '';
      const shortId = rawId ? (rawId.length > 8 ? rawId.substring(0, 8) + '...' : rawId) : '--';
      const descriptionText = a.description || (isSeller ? 'Autonomous reasoning & compute worker' : 'Ingress client endpoint');
      
      // Calculate token and fixed pricing display
      const inPriceUSD = a.input_token_price_micro_usd ? (a.input_token_price_micro_usd / 1000).toFixed(3) : '0.002';
      const outPriceUSD = a.output_token_price_micro_usd ? (a.output_token_price_micro_usd / 1000).toFixed(3) : '0.002';
      const fixedFeeDisplay = a.fixed_cost_per_call_micro_usd > 0 ? ` + ${a.fixed_cost_per_call_micro_usd}µ/call` : '';
      const repScore = a.reputation_score ? a.reputation_score.toFixed(2) : '5.00';

      return `
        <tr style="border-bottom: 1px solid rgba(255, 255, 255, 0.05); transition: background 0.15s ease;">
          
          <!-- Column 1: Agent ID -->
          <td style="padding: 14px 14px; white-space: nowrap;">
            <span style="font-family: var(--font-mono); font-size: 12px; color: var(--accent-cyan); cursor: pointer;" title="${rawId}">
              ${shortId}
            </span>
          </td>

          <!-- Column 2: Agent / Node Name & Type -->
          <td style="padding: 14px 14px; white-space: nowrap;">
            <div style="font-weight: 700; color: #fff; font-size: 14px;">
              ${a.name}
            </div>
            <div style="font-size: 11px; color: var(--accent-cyan); margin-top: 2px; font-family: var(--font-mono);">
              ${a.agent_type || 'GENERAL_LLM_INFERENCE'}
            </div>
          </td>

          <!-- Column 3: Role -->
          <td style="padding: 14px 14px; white-space: nowrap;">
            <span style="background: ${isSeller ? 'rgba(245, 158, 11, 0.15)' : 'rgba(56, 189, 248, 0.15)'}; color: ${isSeller ? 'var(--accent-amber)' : 'var(--accent-cyan)'}; font-size: 11px; font-weight: 700; padding: 3px 8px; border-radius: 4px; border: 1px solid ${isSeller ? 'rgba(245, 158, 11, 0.3)' : 'rgba(56, 189, 248, 0.3)'};">
              ${roleLabel}
            </span>
          </td>

          <!-- Column 4: Capability & Description -->
          <td style="padding: 14px 14px;">
            <div style="font-weight: 600; color: #fff; font-size: 13px; display: flex; align-items: center; gap: 6px;">
              <span>${isSeller ? '⚡' : '🧠'} ${modelDisplayName}</span>
              <span style="color: var(--text-muted); font-size: 11px;">(${vramInfo} • ${a.max_concurrency || 1}x)</span>
            </div>
            <div style="font-size: 11px; color: var(--text-secondary); margin-top: 3px; max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${descriptionText}">
              ${descriptionText}
            </div>
          </td>

          <!-- Column 5: Pricing & Reputation -->
          <td style="padding: 14px 14px; white-space: nowrap;">
            <div style="font-family: var(--font-mono); font-size: 12px; color: var(--accent-emerald);">
              $${inPriceUSD} / $${outPriceUSD}${fixedFeeDisplay}
            </div>
            <div style="font-size: 11px; color: var(--accent-amber); margin-top: 2px;">
              ⭐ ${repScore} / 5.00
            </div>
          </td>

          <!-- Column 6: Status & Approval -->
          <td style="padding: 14px 14px; white-space: nowrap;">
            <div style="display: flex; align-items: center; gap: 6px;">
              ${a.status === 'ACTIVE' 
                ? '<span style="color: var(--accent-emerald); font-weight: 600; font-size: 12px;">● ACTIVE</span>'
                : (a.status === 'OFFLINE'
                  ? '<span style="color: var(--text-muted); font-weight: 600; font-size: 12px;">○ OFFLINE</span>'
                  : '<span style="color: #f87171; font-weight: 600; font-size: 12px;">✕ SUSPENDED</span>')
              }
              <span class="${a.approval_status === 'APPROVED' ? 'status-tag approved' : 'status-tag pending'}" style="font-size: 10px; padding: 2px 6px;">
                ${a.approval_status || 'PENDING_REVIEW'}
              </span>
            </div>
          </td>

          <!-- Column 7: Scoped API Key with Copy in front -->
          <td style="padding: 14px 14px; white-space: nowrap;">
            <div style="display: flex; align-items: center; gap: 8px;">
              <span style="font-family: var(--font-mono); font-size: 12px; color: var(--accent-emerald); background: rgba(0, 0, 0, 0.4); padding: 5px 10px; border-radius: 4px; border: 1px dashed rgba(16, 185, 129, 0.35); max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${keyPrefix}">
                ${keyPrefix}
              </span>
              <button type="button" class="btn-copy-node-key" data-key="${keyPrefix}" style="background: rgba(56, 189, 248, 0.12); border: 1px solid rgba(56, 189, 248, 0.3); color: var(--accent-cyan); padding: 5px 10px; border-radius: 4px; font-size: 11px; font-weight: 700; cursor: pointer; white-space: nowrap; transition: all 0.2s;">
                📋 Copy
              </button>
            </div>
          </td>

          <!-- Column 8: Action -->
          <td style="padding: 14px 14px; text-align: right; white-space: nowrap;">
            ${isSeller ? `
              <button type="button" class="btn-copy-daemon-cmd" data-cmd="python daemon.py --key=${keyPrefix} --endpoint=ws://localhost:8080/v1/provider/tunnel" style="background: rgba(245, 158, 11, 0.12); border: 1px solid rgba(245, 158, 11, 0.35); color: var(--accent-amber); padding: 5px 12px; border-radius: 4px; font-size: 11px; font-weight: 700; cursor: pointer; white-space: nowrap; transition: all 0.2s;">
                ⚡ Tunnel Cmd
              </button>
            ` : `
              <span style="font-size: 11px; color: var(--text-muted);">Ingress Active</span>
            `}
          </td>

        </tr>
      `;
    }).join('');

    // Attach copy listeners for Node API keys
    container.querySelectorAll('.btn-copy-node-key').forEach(btn => {
      btn.addEventListener('click', () => {
        const key = btn.getAttribute('data-key');
        if (key) {
          navigator.clipboard.writeText(key).then(() => {
            const originalText = btn.innerText;
            btn.innerText = '✓ Copied!';
            btn.style.color = 'var(--accent-emerald)';
            btn.style.borderColor = 'rgba(16, 185, 129, 0.6)';
            setTimeout(() => {
              btn.innerText = originalText;
              btn.style.color = 'var(--accent-cyan)';
              btn.style.borderColor = 'rgba(56, 189, 248, 0.3)';
            }, 2000);
          });
        }
      });
    });

    // Attach copy listeners for Daemon Commands
    container.querySelectorAll('.btn-copy-daemon-cmd').forEach(btn => {
      btn.addEventListener('click', () => {
        const cmd = btn.getAttribute('data-cmd');
        if (cmd) {
          navigator.clipboard.writeText(cmd).then(() => {
            const originalText = btn.innerText;
            btn.innerText = '✓ Cmd Copied!';
            btn.style.color = 'var(--accent-emerald)';
            setTimeout(() => {
              btn.innerText = originalText;
              btn.style.color = 'var(--accent-amber)';
            }, 2000);
          });
        }
      });
    });
  } catch (err) {
    console.error('Failed to fetch agents:', err);
  }
}

export async function handleSubmitNewSellerAgent() {
  if (!state.currentUser) return;
  const name = document.getElementById('newNodeName').value.trim();
  const agentType = document.getElementById('newNodeAgentType') ? document.getElementById('newNodeAgentType').value : 'GENERAL_LLM_INFERENCE';
  const model = document.getElementById('newNodeModel').value.trim();
  const description = document.getElementById('newNodeDescription') ? document.getElementById('newNodeDescription').value.trim() : '';
  const vram = parseInt(document.getElementById('newNodeVram').value) || 0;
  const concurrency = parseInt(document.getElementById('newNodeConcurrency').value) || 1;
  const inPrice = parseInt(document.getElementById('newNodeInPrice').value) || 0;
  const outPrice = parseInt(document.getElementById('newNodeOutPrice').value) || 0;
  const fixedCost = document.getElementById('newNodeFixedCost') ? parseInt(document.getElementById('newNodeFixedCost').value) || 0 : 0;
  const jsonSchema = document.getElementById('newNodeJsonSchema') ? document.getElementById('newNodeJsonSchema').value.trim() : '{"type": "object", "properties": {}}';

  if (!name || !model) {
    alert('Please specify both Node Name and Model/Tool Identifier.');
    return;
  }

  try {
    const data = await deploySellerNodeAPI(state.currentUser.owner_id, {
      node_name: name,
      agent_type: agentType,
      tool_name: model,
      description: description,
      vram_gb: vram,
      max_concurrency: concurrency,
      input_token_price_micro_usd: inPrice,
      output_token_price_micro_usd: outPrice,
      fixed_cost_per_call_micro_usd: fixedCost,
      json_schema: jsonSchema
    });

    const key = data.node_api_key || data.api_key || 'walras_live_org_...';
    document.getElementById('rawKeyOutput').innerText = key;
    document.getElementById('newKeyReveal').classList.remove('hidden');
    fetchAndRenderAgents();
  } catch (err) {
    alert('Error provisioning node: ' + err.message);
  }
}
