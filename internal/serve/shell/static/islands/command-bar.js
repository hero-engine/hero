// command-bar.js — placeholder shell-owned stub.
//
// The real ⌘K command-bar island is delivered by hero-chat-and-model.
// The shell mounts a script tag at this stable path in page-layout.html
// so that, on every page, the keyboard shortcut binds even before the
// chat-and-model spec lands. When that spec ships it replaces this
// file in-place.
(function () {
  'use strict';

  function isTypingTarget(el) {
    if (!el) return false;
    const tag = el.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || el.isContentEditable;
  }

  document.addEventListener('keydown', function (e) {
    const mod = e.metaKey || e.ctrlKey;
    if (!mod || (e.key !== 'k' && e.key !== 'K')) return;
    if (isTypingTarget(document.activeElement)) return;
    e.preventDefault();
    const trigger = document.querySelector('[data-command-bar-trigger]');
    if (trigger) {
      trigger.click();
      trigger.focus();
    }
  });
})();
