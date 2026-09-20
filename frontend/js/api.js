import { state } from './state.js';

export async function loginAPI(email, password) {
  const res = await fetch(`${state.gatewayURL}/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.details || data.error || 'Authentication failed');
  return data;
}

export async function signupAPI(name, email, password, userType) {
  const res = await fetch(`${state.gatewayURL}/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, email, password, user_type: userType })
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.details || data.error || 'Registration failed');
  return data;
}

export async function getUserAgentsAPI(ownerID) {
  const res = await fetch(`${state.gatewayURL}/v1/user/agents?owner_id=${ownerID}`);
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Failed to fetch user agents');

  if (data.credit_limit_micro_usd !== undefined && state.currentUser) {
    state.currentUser.credit_limit_micro_usd = data.credit_limit_micro_usd;
    state.currentUser.balance_usd = data.balance_usd;
    sessionStorage.setItem('walras_user', JSON.stringify(state.currentUser));
    const verifyBalance = document.getElementById('verifyBalance');
    const verifyMicroBalance = document.getElementById('verifyMicroBalance');
    const userCreditDisplay = document.getElementById('userCreditDisplay');
    if (verifyBalance) verifyBalance.innerText = data.balance_usd;
    if (userCreditDisplay) userCreditDisplay.innerText = data.balance_usd;
    if (verifyMicroBalance) verifyMicroBalance.innerText = `${data.credit_limit_micro_usd.toLocaleString()} µUSD`;
  }

  return data.agents || [];
}

export async function deploySellerNodeAPI(ownerID, payload) {
  const res = await fetch(`${state.gatewayURL}/v1/seller/agents?owner_id=${ownerID}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Failed to deploy node');
  return data;
}

export async function dispatchChatCompletionAPI(apiKey, model, prompt) {
  const res = await fetch(`${state.gatewayURL}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + apiKey
    },
    body: JSON.stringify({
      model: model,
      messages: [{ role: 'user', content: prompt }]
    })
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `Gateway Error (${res.status})`);
  return data;
}

export async function getPricingAPI() {
  const res = await fetch(`${state.gatewayURL}/v1/pricing`);
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || 'Failed to fetch pricing data');
  return data;
}

