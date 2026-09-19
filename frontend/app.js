import { loadUserSession } from './js/state.js';
import { initAuth, transitionToDashboard, transitionToLanding } from './js/auth.js';
import { initRadar } from './js/radar.js';
import { initIngress } from './js/ingress.js';
import { initNodes } from './js/nodes.js';
import { initPricing } from './js/pricing.js';
import { initNavigation } from './js/navigation.js';

window.addEventListener('DOMContentLoaded', () => {
  // Initialize subsystems
  initAuth();
  initRadar();
  initIngress();
  initNodes();
  initPricing();
  initNavigation();

  // Check if session exists
  const user = loadUserSession();
  if (user) {
    transitionToDashboard();
  } else {
    transitionToLanding();
  }
});

