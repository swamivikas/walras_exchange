import { state, saveUserSession, clearUserSession } from './state.js';
import { loginAPI, signupAPI } from './api.js';
import { fetchAndRenderAgents } from './nodes.js';

export function initAuth() {
  // Wire open modal triggers
  const btnHeroGetStarted = document.getElementById('btnHeroGetStarted');
  const btnNavGetStarted = document.getElementById('btnNavGetStarted');
  const btnNavSignIn = document.getElementById('btnNavSignIn');
  const btnCloseAuthModal = document.getElementById('btnCloseAuthModal');
  const authModal = document.getElementById('authModal');

  if (btnHeroGetStarted) btnHeroGetStarted.addEventListener('click', () => openAuthModal('signup'));
  if (btnNavGetStarted) btnNavGetStarted.addEventListener('click', () => openAuthModal('signup'));
  if (btnNavSignIn) btnNavSignIn.addEventListener('click', () => openAuthModal('login'));
  if (btnCloseAuthModal) btnCloseAuthModal.addEventListener('click', closeAuthModal);

  document.querySelectorAll('.open-signup-btn').forEach(btn => {
    btn.addEventListener('click', () => openAuthModal('signup'));
  });
  document.querySelectorAll('.open-login-btn').forEach(btn => {
    btn.addEventListener('click', () => openAuthModal('login'));
  });

  if (authModal) {
    authModal.addEventListener('click', (e) => {
      if (e.target === authModal) closeAuthModal();
    });
  }

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeAuthModal();
  });

  // Toggle between Signup and Login on the Landing Auth Card
  const toggleBtn = document.getElementById('authModeToggle');
  if (toggleBtn) {
    toggleBtn.addEventListener('click', toggleAuthMode);
  }

  // Wire role preset buttons
  const chipBuyer = document.getElementById('chipBuyer');
  const chipSeller = document.getElementById('chipSeller');
  const chipHybrid = document.getElementById('chipHybrid');

  if (chipBuyer) chipBuyer.addEventListener('click', () => selectRolePreset('ENTERPRISE_BUYER'));
  if (chipSeller) chipSeller.addEventListener('click', () => selectRolePreset('SELLER_OWNER'));
  if (chipHybrid) chipHybrid.addEventListener('click', () => selectRolePreset('HYBRID'));

  // Wire submit button & Enter key on inputs
  const submitBtn = document.getElementById('btnAuthSubmit');
  if (submitBtn) {
    submitBtn.addEventListener('click', handleAuthSubmit);
  }

  const authEmailInput = document.getElementById('authEmail');
  const authPasswordInput = document.getElementById('authPassword');
  const authNameInput = document.getElementById('authName');

  [authEmailInput, authPasswordInput, authNameInput].forEach(input => {
    if (input) {
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          handleAuthSubmit();
        }
      });
    }
  });

  // Wire logout button in the dashboard
  const logoutBtn = document.getElementById('btnLogout');
  if (logoutBtn) {
    logoutBtn.addEventListener('click', handleLogout);
  }
}

export function setAuthMode(mode) {
  state.authMode = mode;
  const isSignup = mode === 'signup';

  const groupName = document.getElementById('groupName');
  const groupRole = document.getElementById('groupRole');
  const authGrantBadge = document.getElementById('authGrantBadge');
  const authCardTitle = document.getElementById('authCardTitle');
  const authCardSubtitle = document.getElementById('authCardSubtitle');
  const submitBtn = document.getElementById('btnAuthSubmit');
  const toggleBtn = document.getElementById('authModeToggle');

  if (groupName) groupName.style.display = isSignup ? 'flex' : 'none';
  if (groupRole) groupRole.style.display = isSignup ? 'flex' : 'none';
  if (authGrantBadge) authGrantBadge.style.display = isSignup ? 'inline-block' : 'none';

  if (authCardTitle) {
    authCardTitle.innerText = isSignup 
      ? 'Claim $50 Credit & Get Started' 
      : 'Sign in to Walras Exchange';
  }

  if (authCardSubtitle) {
    authCardSubtitle.innerText = isSignup
      ? 'Instant access to decentralized AI Agents & MCP clearinghouse'
      : 'Access your compute, agents, and clearinghouse balance';
  }

  if (submitBtn) {
    submitBtn.innerHTML = isSignup
      ? '<span>Claim $50 Free Trial &amp; Enter &rarr;</span>'
      : '<span>Sign In to Account &rarr;</span>';
  }

  if (toggleBtn) {
    toggleBtn.innerText = isSignup
      ? 'Already registered? Sign In'
      : 'Don\'t have an account? Claim $50 & Sign Up';
  }

  clearAuthAlert();
}

export function openAuthModal(mode = 'signup') {
  const modal = document.getElementById('authModal');
  if (!modal) return;
  setAuthMode(mode);
  modal.classList.remove('hidden');
}

export function closeAuthModal() {
  const modal = document.getElementById('authModal');
  if (modal) modal.classList.add('hidden');
  clearAuthAlert();
}

export function toggleAuthMode() {
  const nextMode = state.authMode === 'signup' ? 'login' : 'signup';
  setAuthMode(nextMode);
}

export function selectRolePreset(role) {
  const hiddenInput = document.getElementById('authUserType');
  if (hiddenInput) hiddenInput.value = role;

  // Toggle active styling on buttons
  const chipBuyer = document.getElementById('chipBuyer');
  const chipSeller = document.getElementById('chipSeller');
  const chipHybrid = document.getElementById('chipHybrid');

  if (chipBuyer) chipBuyer.classList.toggle('active', role === 'ENTERPRISE_BUYER');
  if (chipSeller) chipSeller.classList.toggle('active', role === 'SELLER_OWNER');
  if (chipHybrid) chipHybrid.classList.toggle('active', role === 'HYBRID');
}

export async function handleAuthSubmit() {
  const btn = document.getElementById('btnAuthSubmit');
  const email = document.getElementById('authEmail').value.trim();
  const password = document.getElementById('authPassword').value;
  const userType = document.getElementById('authUserType').value;
  
  if (state.authMode === 'signup') {
    const name = document.getElementById('authName').value.trim();
    if (!name) {
      showAuthAlert('Please enter your Enterprise name or Username.', 'error');
      return;
    }
  }

  if (!email || !password) {
    showAuthAlert('Please enter both email and password.', 'error');
    return;
  }

  btn.disabled = true;

  try {
    if (state.authMode === 'login') {
      const data = await loginAPI(email, password);
      saveUserSession(data);
      transitionToDashboard();
    } else {
      const name = document.getElementById('authName').value.trim();
      const data = await signupAPI(name, email, password, userType);

      if (data.default_agent && data.default_agent.api_key && data.default_agent.agent_id) {
        localStorage.setItem('walras_agent_key_' + data.default_agent.agent_id, data.default_agent.api_key);
      }

      const userSession = {
        owner_id: data.owner_id,
        name: data.name,
        email: data.email,
        user_type: data.user_type,
        credit_limit_micro_usd: 50000000,
        balance_usd: '$50.00 USD',
        agents: [data.default_agent]
      };

      saveUserSession(userSession);
      transitionToDashboard();
    }
  } catch (err) {
    showAuthAlert(err.message, 'error');
  } finally {
    btn.disabled = false;
  }
}

export function transitionToDashboard() {
  if (!state.currentUser) return;

  // Close modal if open
  closeAuthModal();

  // Switch Views: Hide landing, show dashboard / verification view
  document.getElementById('landingView').classList.add('hidden');
  document.getElementById('dashboardView').classList.remove('hidden');

  const u = state.currentUser;

  // Populate Header Elements
  const welcomeUserName = document.getElementById('welcomeUserName');
  const userCreditDisplay = document.getElementById('userCreditDisplay');
  const userRoleDisplay = document.getElementById('userRoleDisplay');
  
  if (welcomeUserName) welcomeUserName.innerText = u.name || 'Developer';
  if (userCreditDisplay) userCreditDisplay.innerText = u.balance_usd || '$50.00 USD';
  if (userRoleDisplay) userRoleDisplay.innerText = u.user_type || 'ENTERPRISE_BUYER';

  // Populate Identity & Financial Fields
  const verifyName = document.getElementById('verifyName');
  const verifyEmail = document.getElementById('verifyEmail');
  const verifyOwnerId = document.getElementById('verifyOwnerId');
  const verifyBalance = document.getElementById('verifyBalance');
  const verifyMicroBalance = document.getElementById('verifyMicroBalance');
  const verifyApiKey = document.getElementById('verifyApiKey');
  const verifyAgentId = document.getElementById('verifyAgentId');

  if (verifyName) verifyName.innerText = u.name || '--';
  if (verifyEmail) verifyEmail.innerText = u.email || '--';
  if (verifyOwnerId) verifyOwnerId.innerText = u.owner_id || '--';
  if (verifyBalance) verifyBalance.innerText = u.balance_usd || '$50.00 USD';
  
  const microVal = u.credit_limit_micro_usd || 50000000;
  if (verifyMicroBalance) verifyMicroBalance.innerText = `${microVal.toLocaleString()} µUSD`;

  // Toggle Buyer Authorization Key Card (Hidden for Pure Sellers, visible for Buyers & Hybrids)
  const buyerApiKeyCard = document.getElementById('buyerApiKeyCard');
  if (buyerApiKeyCard) {
    buyerApiKeyCard.style.display = u.user_type === 'SELLER_OWNER' ? 'none' : 'flex';
  }

  // Tailor Agent Fleet section and disable Add Node for Buyer accounts
  const fleetTitle = document.getElementById('fleetSectionTitle');
  const fleetSubtitle = document.getElementById('fleetSectionSubtitle');
  const btnOpenDeploy = document.getElementById('btnOpenDeploy');
  const isBuyer = u.user_type === 'ENTERPRISE_BUYER' || u.user_type === 'BUYER';

  if (btnOpenDeploy) {
    btnOpenDeploy.disabled = isBuyer;
    if (isBuyer) {
      btnOpenDeploy.style.filter = 'opacity(0.45)';
      btnOpenDeploy.style.cursor = 'not-allowed';
      btnOpenDeploy.style.background = 'rgba(255, 255, 255, 0.08)';
      btnOpenDeploy.style.borderColor = 'rgba(255, 255, 255, 0.15)';
      btnOpenDeploy.style.boxShadow = 'none';
      btnOpenDeploy.title = '🔒 Buyer Account: Node hosting disabled. Switch or register as an Agent Seller to monetize agents.';
    } else {
      btnOpenDeploy.style.filter = 'none';
      btnOpenDeploy.style.cursor = 'pointer';
      btnOpenDeploy.style.background = 'linear-gradient(135deg, #0284c7 0%, #2563eb 50%, #4f46e5 100%)';
      btnOpenDeploy.style.borderColor = 'rgba(255, 255, 255, 0.15)';
      btnOpenDeploy.style.boxShadow = '0 4px 14px rgba(37, 99, 235, 0.35)';
      btnOpenDeploy.title = 'Deploy a new autonomous agent or MCP tool server';
    }
  }

  if (fleetTitle && fleetSubtitle) {
    if (u.user_type === 'SELLER_OWNER') {
      fleetTitle.innerHTML = '⚡ Monetized Agent Fleet &amp; Worker Nodes';
      fleetSubtitle.innerText = 'All registered agent workers, reasoning models, and their dedicated reverse tunnel API keys.';
    } else {
      fleetTitle.innerHTML = '🤖 Agent Fleet &amp; MCP Servers';
      fleetSubtitle.innerText = 'Deploy and connect your autonomous AI agents or MCP tool servers to the clearinghouse';
    }
  }

  // Populate API key from default agent if present
  let apiKey = 'walras_live_org_...';
  let agentId = '--';

  if (u.agents && u.agents.length > 0) {
    const first = u.agents[0];
    if (first.api_key) apiKey = first.api_key;
    if (first.agent_id) agentId = first.agent_id;
  }

  if (verifyApiKey) verifyApiKey.innerText = apiKey;
  if (verifyAgentId) verifyAgentId.innerText = agentId;

  // Wire up Copy API Key button
  const btnCopy = document.getElementById('btnCopyApiKey');
  const copyLabel = document.getElementById('copySuccessLabel');
  if (btnCopy) {
    btnCopy.onclick = () => {
      navigator.clipboard.writeText(apiKey).then(() => {
        if (copyLabel) {
          copyLabel.style.display = 'inline';
          setTimeout(() => { copyLabel.style.display = 'none'; }, 2500);
        }
      }).catch(err => {
        console.error('Failed to copy key:', err);
      });
    };
  }

  // Load and render user's registered agents & nodes
  fetchAndRenderAgents();
}

export function transitionToLanding() {
  document.getElementById('dashboardView').classList.add('hidden');
  document.getElementById('landingView').classList.remove('hidden');
}

export function handleLogout() {
  clearUserSession();
  location.reload();
}

function showAuthAlert(msg, type) {
  const b = document.getElementById('authAlert');
  if (!b) return;
  
  let userMsg = msg;
  if (msg === 'invalid_email_or_password') {
    userMsg = 'Invalid email or password. Please verify your credentials.';
  } else if (msg === 'email_already_registered') {
    userMsg = 'This email is already registered. Please sign in instead.';
  } else if (msg === 'email_and_password_required') {
    userMsg = 'Please enter both your email address and password.';
  }

  b.style.display = 'block';
  b.innerText = userMsg;
  b.style.background = type === 'error' ? 'rgba(239, 68, 68, 0.15)' : 'rgba(16, 185, 129, 0.15)';
  b.style.border = type === 'error' ? '1px solid rgba(239, 68, 68, 0.4)' : '1px solid rgba(16, 185, 129, 0.4)';
  b.style.color = type === 'error' ? '#f87171' : '#34d399';
  b.style.fontWeight = '500';
}

function clearAuthAlert() {
  const b = document.getElementById('authAlert');
  if (b) b.style.display = 'none';
}
