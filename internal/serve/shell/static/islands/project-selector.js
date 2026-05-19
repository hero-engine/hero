// Project selector — top-nav dropdown for switching the active project.
//
// Behavior:
//  - Click trigger to open the menu; click outside to close.
//  - Click an option to navigate to /p/<slug>/<current-page> and persist
//    the selection to localStorage and a cookie.
//  - Persistence keys:
//      localStorage: hero.activeProject
//      cookie:       hero_active_project
//  - The cookie is what the server reads to resolve the default project
//    on legacy URLs (/now → /p/<default>/now), so writing it on every
//    selection keeps subsequent loads stable.

(function () {
  'use strict';

  const STORAGE_KEY = 'hero.activeProject';
  const COOKIE_NAME = 'hero_active_project';

  function setCookie(name, value) {
    // 30-day expiry; Path=/ so every shell route reads the same value.
    const exp = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toUTCString();
    document.cookie = name + '=' + encodeURIComponent(value) + '; Path=/; Expires=' + exp + '; SameSite=Lax';
  }

  function persistSelection(slug) {
    try {
      window.localStorage.setItem(STORAGE_KEY, slug);
    } catch (e) {
      // ignore — private mode etc.
    }
    setCookie(COOKIE_NAME, slug);
  }

  function init(root) {
    const trigger = root.querySelector('[data-project-selector-trigger]');
    const menu = root.querySelector('[data-project-selector-menu]');
    if (!trigger || !menu) return;

    const currentPage = root.getAttribute('data-current-page') || 'now';

    function close() {
      menu.hidden = true;
      trigger.setAttribute('aria-expanded', 'false');
    }
    function open() {
      menu.hidden = false;
      trigger.setAttribute('aria-expanded', 'true');
    }

    trigger.addEventListener('click', (ev) => {
      ev.stopPropagation();
      if (menu.hidden) open(); else close();
    });

    document.addEventListener('click', (ev) => {
      if (!root.contains(ev.target)) close();
    });

    menu.querySelectorAll('[data-project-selector-option]').forEach((opt) => {
      opt.addEventListener('click', () => {
        const slug = opt.getAttribute('data-slug');
        if (!slug) return;
        persistSelection(slug);
        window.location.href = '/p/' + slug + '/' + currentPage;
      });
    });
  }

  function bootstrap() {
    document.querySelectorAll('[data-project-selector]').forEach(init);

    // On first paint, reconcile localStorage → cookie so a user who
    // cleared the cookie but kept localStorage still gets a stable
    // default-project resolution on the next legacy-URL load.
    try {
      const stored = window.localStorage.getItem(STORAGE_KEY);
      if (stored) setCookie(COOKIE_NAME, stored);
    } catch (e) { /* ignore */ }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bootstrap);
  } else {
    bootstrap();
  }
})();
