// Global Application State
export const state = {
  currentUser: null,
  authMode: 'login',
  activeNodes: [],
  gatewayURL: 'http://localhost:8080'
};

export function loadUserSession() {
  const saved = localStorage.getItem('walras_user');
  if (saved) {
    try {
      state.currentUser = JSON.parse(saved);
      return state.currentUser;
    } catch (e) {
      console.error('Failed to parse saved user:', e);
    }
  }
  return null;
}

export function saveUserSession(user) {
  state.currentUser = user;
  localStorage.setItem('walras_user', JSON.stringify(user));
}

export function clearUserSession() {
  state.currentUser = null;
  localStorage.removeItem('walras_user');
}
