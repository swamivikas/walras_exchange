export function initNavigation() {
  const navLinks = document.querySelectorAll('.nav-link');
  navLinks.forEach(link => {
    link.addEventListener('click', (e) => {
      e.preventDefault();
      const targetId = link.getAttribute('data-target') || link.getAttribute('href')?.replace('#', '');
      if (targetId) {
        const targetElem = document.getElementById(targetId);
        if (targetElem) {
          const navOffset = 80;
          const elementPosition = targetElem.getBoundingClientRect().top;
          const offsetPosition = elementPosition + window.pageYOffset - navOffset;

          window.scrollTo({
            top: offsetPosition,
            behavior: 'smooth'
          });
        }
      }
    });
  });

  // Docs language tab switching
  const docTabs = document.querySelectorAll('.doc-tab');
  docTabs.forEach(tab => {
    tab.addEventListener('click', () => {
      const lang = tab.getAttribute('data-lang');
      docTabs.forEach(t => t.classList.remove('active'));
      tab.classList.add('active');

      document.querySelectorAll('.doc-snippet').forEach(snippet => {
        snippet.style.display = snippet.getAttribute('data-lang') === lang ? 'block' : 'none';
      });
    });
  });
}
