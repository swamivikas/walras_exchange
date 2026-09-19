import { getPricingAPI } from './api.js';

export function initPricing() {
  const tokenSlider = document.getElementById('tokenSlider');
  const tokenCountDisplay = document.getElementById('tokenCountDisplay');
  const walrasCostDisplay = document.getElementById('walrasCostDisplay');
  const openaiCostDisplay = document.getElementById('openaiCostDisplay');
  const savingsDisplay = document.getElementById('savingsDisplay');

  if (tokenSlider) {
    tokenSlider.addEventListener('input', (e) => {
      const millionTokens = parseFloat(e.target.value);
      if (tokenCountDisplay) tokenCountDisplay.innerText = `${millionTokens}M Tokens / Mo`;

      // Walras Blended Avg: ~$0.55/1M vs OpenAI Avg: ~$5.00/1M vs AWS: ~$6.50/1M
      const walrasCost = (millionTokens * 0.55).toFixed(2);
      const openaiCost = (millionTokens * 5.00).toFixed(2);
      const savings = (openaiCost - walrasCost).toFixed(2);

      if (walrasCostDisplay) walrasCostDisplay.innerText = `$${walrasCost}`;
      if (openaiCostDisplay) openaiCostDisplay.innerText = `$${openaiCost}`;
      if (savingsDisplay) savingsDisplay.innerText = `Save $${savings} / mo (${Math.round((savings / openaiCost) * 100)}%)`;
    });
  }

  // Fetch dynamic model rates from /v1/pricing
  loadLivePricing();
}

async function loadLivePricing() {
  try {
    const data = await getPricingAPI();
    const tableBody = document.getElementById('pricingTableBody');
    if (!tableBody || !data.models) return;

    tableBody.innerHTML = data.models.map(m => {
      const inPrice = (m.input_price_per_1m_usd !== undefined ? m.input_price_per_1m_usd : (m.inputPricePer1M || 0)).toFixed(2);
      const outPrice = (m.output_price_per_1m_usd !== undefined ? m.output_price_per_1m_usd : (m.outputPricePer1M || 0)).toFixed(2);
      const legacyPrice = (m.proprietary_price_per_1m_usd !== undefined ? m.proprietary_price_per_1m_usd : (m.proprietaryPrice || 0)).toFixed(2);
      const savingsPct = m.savings_percentage !== undefined ? m.savings_percentage : (m.savingsPercentage || 85);

      return `
        <tr>
          <td>
            <div style="font-weight: 600; color: #fff;">${m.name}</div>
            <div style="font-size: 12px; color: var(--text-muted);">${m.description}</div>
          </td>
          <td style="color: var(--accent-emerald); font-weight: 600; font-family: var(--font-mono);">$${inPrice} / 1M</td>
          <td style="color: var(--accent-cyan); font-weight: 600; font-family: var(--font-mono);">$${outPrice} / 1M</td>
          <td style="color: var(--text-muted); text-decoration: line-through; font-family: var(--font-mono);">$${legacyPrice} / 1M</td>
          <td>
            <span style="background: rgba(16, 185, 129, 0.12); color: var(--accent-emerald); border: 1px solid rgba(16, 185, 129, 0.3); padding: 3px 8px; border-radius: 4px; font-weight: 600; font-size: 12px;">
              ${savingsPct}% OFF
            </span>
          </td>
        </tr>
      `;
    }).join('');
  } catch (err) {
    console.log('Pricing live sync note:', err.message);
  }
}
