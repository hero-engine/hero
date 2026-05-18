// command-bar.js — ⌘K overlay island.
//
// Single-file ES module island. No build step, no bundler, no external
// deps. Lazily constructs an overlay DOM the first time the user opens
// ⌘K, then keeps it alive for the lifetime of the page.
//
// Wire contract: see .hero/planning/features/hero-chat-and-model/api-contract.md
// Visual source of truth: see .hero/planning/features/hero-now-home/mockups/02-command-bar.html
//
// Endpoints consumed:
//   GET  /api/chat/capability       — adapter + cost snapshot
//   POST /api/chat/turn             — submit a chat turn; returns sse_topic
//   POST /api/chat/preference       — switch interactive adapter
//   GET  /api/chat/slashes          — slash palette catalog (optional; falls back)
//   GET  /api/search?q=<query>      — unified search (optional; degrades gracefully)
//   GET  /api/events?topic=<topic>  — SSE event stream the chat turn publishes on
//
// Mounted by templates/page-layout.html as:
//   <script type="module" src="/static/shell/islands/command-bar.js" defer>
(function () {
  'use strict';

  // -----------------------------------------------------------------
  // Helpers
  // -----------------------------------------------------------------

  function isTypingTarget(el) {
    if (!el) return false;
    const tag = el.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || el.isContentEditable;
  }

  function escapeHTML(s) {
    if (s == null) return '';
    return String(s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function highlight(text, needle) {
    const s = String(text == null ? '' : text);
    if (!needle) return escapeHTML(s);
    const n = String(needle).trim();
    if (!n) return escapeHTML(s);
    const lower = s.toLowerCase();
    const target = n.toLowerCase();
    let out = '';
    let i = 0;
    while (i < s.length) {
      const idx = lower.indexOf(target, i);
      if (idx < 0) { out += escapeHTML(s.slice(i)); break; }
      out += escapeHTML(s.slice(i, idx));
      out += '<em>' + escapeHTML(s.slice(idx, idx + target.length)) + '</em>';
      i = idx + target.length;
    }
    return out;
  }

  function debounce(fn, ms) {
    let t = null;
    return function () {
      const args = arguments;
      const self = this;
      if (t) clearTimeout(t);
      t = setTimeout(function () { fn.apply(self, args); }, ms);
    };
  }

  function formatUSD(n) {
    if (typeof n !== 'number' || !isFinite(n)) return '$0.00';
    if (n < 0) n = 0;
    return '$' + n.toFixed(2);
  }

  function el(tag, attrs, children) {
    const node = document.createElement(tag);
    if (attrs) {
      for (const k in attrs) {
        if (!Object.prototype.hasOwnProperty.call(attrs, k)) continue;
        const v = attrs[k];
        if (v == null || v === false) continue;
        if (k === 'class') node.className = v;
        else if (k === 'html') node.innerHTML = v;
        else if (k === 'text') node.textContent = v;
        else if (k.indexOf('on') === 0 && typeof v === 'function') {
          node.addEventListener(k.slice(2).toLowerCase(), v);
        } else if (v === true) {
          node.setAttribute(k, '');
        } else {
          node.setAttribute(k, v);
        }
      }
    }
    if (children) {
      if (!Array.isArray(children)) children = [children];
      for (let i = 0; i < children.length; i++) {
        const c = children[i];
        if (c == null) continue;
        if (typeof c === 'string') node.appendChild(document.createTextNode(c));
        else node.appendChild(c);
      }
    }
    return node;
  }

  async function jsonFetch(method, url, body) {
    const init = {
      method: method,
      headers: { 'Accept': 'application/json' },
      credentials: 'same-origin',
    };
    if (body != null) {
      init.headers['Content-Type'] = 'application/json';
      init.body = JSON.stringify(body);
    }
    const res = await fetch(url, init);
    const contentType = res.headers.get('content-type') || '';
    let payload = null;
    if (contentType.indexOf('application/json') >= 0) {
      try { payload = await res.json(); } catch (e) { payload = null; }
    } else {
      try { payload = await res.text(); } catch (e) { payload = null; }
    }
    return { ok: res.ok, status: res.status, body: payload };
  }

  // -----------------------------------------------------------------
  // Inline SVG icons
  // -----------------------------------------------------------------

  const SVG_SEARCH = '<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/></svg>';
  const SVG_BOLT   = '<svg width="16" height="16" viewBox="0 0 90 90" fill="currentColor" aria-hidden="true"><path d="M52 8 L22 46 L40 46 L34 82 L68 42 L50 42 Z"/></svg>';
  const SVG_PLUS   = '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14"/></svg>';
  const SVG_ARROW  = '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M13 6l6 6-6 6"/></svg>';

  // -----------------------------------------------------------------
  // CSS — injected once on first overlay open
  // -----------------------------------------------------------------

  const CSS = `
.cmd-overlay {
  position: fixed; inset: 0; z-index: 9999;
  display: flex; align-items: flex-start; justify-content: center;
  padding: 12vh 24px 24px 24px;
  background: rgba(20, 24, 30, 0.18);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
  opacity: 0; pointer-events: none;
  transition: opacity .12s ease;
}
.cmd-overlay[data-open="true"] { opacity: 1; pointer-events: auto; }

.cmd-card {
  width: 640px; max-width: 100%;
  max-height: 80vh;
  background: var(--bg, #ffffff);
  border: 1px solid var(--border, #eef1f5);
  border-radius: 12px;
  box-shadow: 0 16px 48px rgba(20, 24, 30, 0.16);
  overflow: hidden;
  display: flex; flex-direction: column;
  transform: translateY(-6px);
  transition: transform .14s ease;
}
.cmd-overlay[data-open="true"] .cmd-card { transform: translateY(0); }

.cmd-input-row {
  display: flex; align-items: center; gap: 14px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--border, #eef1f5);
  min-height: 64px;
  flex-shrink: 0;
}
.cmd-kind-icon {
  width: 18px; height: 18px;
  color: var(--ink-4, #8a929c);
  flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
}
.cmd-kind-icon.bolt  { color: var(--hero-blue-700, #2a6cb5); }
.cmd-kind-icon.slash {
  color: var(--hero-blue-700, #2a6cb5);
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 18px; font-weight: 600;
}
.cmd-input {
  flex: 1;
  border: none; outline: none; background: transparent;
  font-family: inherit;
  font-size: 18px;
  color: var(--hero-ink, #14181e);
  padding: 0; min-width: 0;
}
.cmd-input::placeholder { color: var(--ink-4, #8a929c); }
.cmd-esc {
  font-size: 11px;
  background: var(--bg-softer, #f5f7fa);
  border: 1px solid var(--border-strong, #e3e7ee);
  border-radius: 4px;
  padding: 2px 7px;
  color: var(--ink-3, #5a626d);
  font-family: 'SF Mono', ui-monospace, monospace;
  flex-shrink: 0;
}

.cmd-context-row {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 18px;
  border-bottom: 1px solid var(--border, #eef1f5);
  background: var(--bg-soft, #fafbfc);
}
.cmd-context-row[hidden] { display: none; }
.ctx-chip {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 3px 9px 3px 8px;
  background: var(--hero-blue-50, #eff6ff);
  border: 1px solid #d6e7fb;
  border-radius: 999px;
  font-size: 12px;
  color: var(--hero-blue-700, #2a6cb5);
  font-weight: 500;
}
.ctx-chip code {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 11px;
  color: var(--hero-blue-900, #1a4a7d);
}
.ctx-chip .ctx-x {
  margin-left: 4px;
  background: none; border: none;
  color: var(--ink-4, #8a929c);
  font-size: 11px;
  cursor: pointer;
  line-height: 1;
  padding: 0 2px;
}
.ctx-chip .ctx-x:hover { color: var(--ink-2, #2a2f37); }

.cmd-body {
  flex: 1 1 auto;
  overflow-y: auto;
  display: flex; flex-direction: column;
  min-height: 0;
}

.mode-divider {
  display: flex; align-items: center; gap: 8px;
  padding: 10px 18px 8px 18px;
  font-size: 10px; font-weight: 600;
  letter-spacing: 0.1em; text-transform: uppercase;
  color: var(--ink-4, #8a929c);
  background: var(--bg-softer, #f5f7fa);
  border-bottom: 1px solid var(--border, #eef1f5);
}
.mode-divider::after {
  content: '';
  flex: 1; height: 1px;
  background: var(--border, #eef1f5);
  margin-left: 6px;
}

.group-label {
  padding: 12px 18px 4px 18px;
  font-size: 10px; font-weight: 600;
  letter-spacing: 0.08em; text-transform: uppercase;
  color: var(--ink-4, #8a929c);
}
.result-row, .slash-row {
  padding: 8px 18px 8px 22px;
  position: relative;
  border-left: 3px solid transparent;
  cursor: pointer;
}
.result-row:hover, .slash-row:hover { background: var(--bg-soft, #fafbfc); }
.result-row.active, .slash-row.active {
  background: var(--hero-blue-50, #eff6ff);
  border-left-color: var(--hero-blue-700, #2a6cb5);
}
.result-title {
  font-size: 13px;
  color: var(--hero-ink, #14181e);
  font-weight: 500;
  display: flex; align-items: center; gap: 8px;
  margin-bottom: 2px;
}
.result-title em, .slash-name em {
  background: rgba(108, 182, 255, 0.25);
  color: var(--hero-blue-900, #1a4a7d);
  font-style: normal;
  border-radius: 2px;
  padding: 0 2px;
  font-weight: 600;
}
.result-sub { font-size: 12px; color: var(--ink-4, #8a929c); }
.result-sub .mono, .result-title .mono {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 11px;
}

.slash-row {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 14px; align-items: center;
  padding: 10px 18px 10px 22px;
}
.slash-name {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 13px;
  color: var(--hero-blue-700, #2a6cb5);
  font-weight: 600;
}
.slash-desc { font-size: 12px; color: var(--ink-3, #5a626d); }
.slash-arg {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 11px;
  color: var(--ink-4, #8a929c);
  background: var(--bg-softer, #f5f7fa);
  border: 1px solid var(--border, #eef1f5);
  border-radius: 4px;
  padding: 1px 6px;
}

.chat-area {
  padding: 14px 18px 16px 18px;
  display: flex; flex-direction: column; gap: 10px;
}
.chat-user {
  align-self: flex-end;
  max-width: 88%;
  font-size: 12px;
  color: var(--ink-3, #5a626d);
  background: var(--bg-softer, #f5f7fa);
  border: 1px solid var(--border, #eef1f5);
  border-radius: 10px;
  padding: 6px 10px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
}
.tool-line {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 11px;
  color: var(--ink-4, #8a929c);
  display: flex; align-items: center; gap: 6px;
  flex-wrap: wrap;
}
.tool-line .arrow { color: var(--hero-blue-700, #2a6cb5); font-weight: 600; }
.tool-line .ok    { color: var(--success, #16a249); }
.tool-line .name  { color: var(--ink-3, #5a626d); }
.assistant-stream {
  font-size: 13px;
  color: var(--ink-2, #2a2f37);
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
}
.assistant-stream .cursor {
  display: inline-block;
  width: 7px; height: 14px;
  margin-left: 2px;
  background: var(--hero-blue-700, #2a6cb5);
  vertical-align: -2px;
  animation: cmd-blink 1s steps(2) infinite;
}
.chat-error {
  font-size: 12px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: #991b1b;
  border-radius: 8px;
  padding: 8px 10px;
}
.chat-error a { color: #991b1b; text-decoration: underline; }
@keyframes cmd-blink {
  0%, 50% { opacity: 1; }
  50.01%, 100% { opacity: 0; }
}

.mode-hint {
  padding: 8px 18px 12px 18px;
  font-size: 11px;
  color: var(--ink-4, #8a929c);
  display: flex; align-items: center; gap: 6px;
  flex-wrap: wrap;
}
.mode-hint .kbd {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 10px;
  background: var(--bg, #ffffff);
  border: 1px solid var(--border-strong, #e3e7ee);
  border-radius: 3px;
  padding: 1px 5px;
  color: var(--ink-3, #5a626d);
}
.mode-hint .sep { color: var(--ink-5, #b4bac3); margin: 0 2px; }

.cmd-empty {
  padding: 28px 24px 22px 24px;
  display: flex; flex-direction: column;
  align-items: center; text-align: center; gap: 14px;
}
.cmd-empty .lead {
  font-size: 14px;
  color: var(--ink-2, #2a2f37);
  max-width: 380px;
  line-height: 1.55;
  margin: 0;
}
.cmd-empty .lead strong { color: var(--hero-ink, #14181e); font-weight: 600; }
.cmd-empty code {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 13px;
  color: var(--hero-blue-700, #2a6cb5);
}
.cmd-empty .buttons { display: flex; gap: 10px; margin-top: 4px; flex-wrap: wrap; justify-content: center; }
.cmd-empty .btn {
  display: inline-flex; align-items: center; gap: 6px;
  height: 34px; padding: 0 14px;
  border-radius: 8px;
  font-size: 13px; font-weight: 500;
  border: 1px solid transparent;
  background: transparent;
  color: var(--ink-2, #2a2f37);
  text-decoration: none;
  cursor: pointer;
}
.cmd-empty .btn.primary {
  background: var(--hero-blue-700, #2a6cb5);
  color: white;
  border-color: var(--hero-blue-700, #2a6cb5);
}
.cmd-empty .btn.primary:hover { background: var(--hero-blue-900, #1a4a7d); }
.cmd-empty .btn.ghost {
  background: var(--bg, #ffffff);
  border-color: var(--border-strong, #e3e7ee);
}
.cmd-empty .btn.ghost:hover { background: var(--bg-softer, #f5f7fa); }
.cmd-empty .foot {
  font-size: 12px;
  color: var(--ink-4, #8a929c);
  max-width: 420px;
  line-height: 1.5;
  margin: 2px 0 0 0;
}

.cmd-input-row.muted .cmd-kind-icon { color: var(--ink-5, #b4bac3); }
.cmd-input-row.muted .cmd-input::placeholder { color: var(--ink-5, #b4bac3); }

.kind-chip {
  display: inline-flex; align-items: center;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 10px; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.05em;
  font-family: 'Inter', sans-serif;
  flex-shrink: 0;
}
.kind-spec      { background: var(--hero-blue-50, #eff6ff); color: var(--hero-blue-700, #2a6cb5); }
.kind-knowledge { background: #f3e8ff; color: #7c3aed; }
.kind-note      { background: #f3e8ff; color: #7c3aed; }
.kind-code      { background: #ecfdf5; color: var(--success, #16a249); }
.kind-commit    { background: #fff7ed; color: var(--warn, #d97706); }
.kind-default   { background: var(--bg-softer, #f5f7fa); color: var(--ink-3, #5a626d); }

.cmd-blank {
  padding: 28px 24px;
  text-align: center;
  font-size: 13px;
  color: var(--ink-4, #8a929c);
}

.cmd-footer {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 18px;
  height: 44px;
  border-top: 1px solid var(--border, #eef1f5);
  background: var(--bg-soft, #fafbfc);
  flex-shrink: 0;
}
.cmd-adapter-chip {
  display: inline-flex; align-items: center; gap: 7px;
  font-size: 12px;
  color: var(--ink-2, #2a2f37);
  font-weight: 500;
  cursor: pointer;
  padding: 4px 8px;
  border: 0; background: transparent;
  border-radius: 6px;
  position: relative;
  font-family: inherit;
}
.cmd-adapter-chip:hover { background: var(--bg-softer, #f5f7fa); }
.cmd-adapter-chip .ad-dot {
  width: 7px; height: 7px; border-radius: 50%;
  background: var(--hero-blue-500, #6cb6ff);
  box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.7);
  animation: cmd-pulse 1.8s ease-out infinite;
}
.cmd-adapter-chip.muted { color: var(--ink-4, #8a929c); }
.cmd-adapter-chip.muted .ad-dot {
  background: var(--ink-5, #b4bac3);
  box-shadow: none;
  animation: none;
}
.cmd-adapter-chip .ad-ver {
  color: var(--ink-4, #8a929c);
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 11px;
}
@keyframes cmd-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.45); }
  70%  { box-shadow: 0 0 0 6px rgba(108, 182, 255, 0); }
  100% { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0); }
}

.cmd-adapter-popover {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  background: var(--bg, #ffffff);
  border: 1px solid var(--border-strong, #e3e7ee);
  box-shadow: 0 8px 24px rgba(20, 24, 30, 0.12);
  border-radius: 8px;
  padding: 6px;
  min-width: 220px;
  z-index: 10;
  display: none;
}
.cmd-adapter-popover[data-open="true"] { display: block; }
.cmd-adapter-popover .pop-item {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  font-size: 12px;
  color: var(--ink-2, #2a2f37);
  cursor: pointer;
  background: none; border: none;
  width: 100%; text-align: left;
  font-family: inherit;
}
.cmd-adapter-popover .pop-item:hover { background: var(--bg-softer, #f5f7fa); }
.cmd-adapter-popover .pop-item.active { color: var(--hero-blue-700, #2a6cb5); font-weight: 600; }
.cmd-adapter-popover .pop-item .pop-ver {
  color: var(--ink-4, #8a929c);
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 10px;
  margin-left: auto;
}
.cmd-adapter-popover .pop-empty {
  padding: 8px;
  font-size: 12px;
  color: var(--ink-4, #8a929c);
}

.cmd-cost-ticker {
  font-size: 12px;
  color: var(--ink-3, #5a626d);
  font-variant-numeric: tabular-nums;
}
.cmd-cost-ticker strong { color: var(--hero-ink, #14181e); font-weight: 600; }
.cmd-cost-ticker .sep   { color: var(--ink-5, #b4bac3); margin: 0 4px; }
`;

  function injectCSS() {
    if (document.getElementById('hero-cmd-style')) return;
    const style = document.createElement('style');
    style.id = 'hero-cmd-style';
    style.textContent = CSS;
    document.head.appendChild(style);
  }

  // -----------------------------------------------------------------
  // Slash catalog (fallback when /api/chat/slashes is unavailable)
  // -----------------------------------------------------------------

  const FALLBACK_SLASHES = [
    { name: 'design',    desc: 'Produce a spec for a feature, enhancement, or platform change', arg: '<thing>',  needs_adapter: true  },
    { name: 'deliver',   desc: 'Execute a spec — implement, validate, complete the work item',  arg: '<slug>',   needs_adapter: true  },
    { name: 'diagnose',  desc: 'Investigate a bug, classify root cause, produce a fix spec',     arg: '<bug>',    needs_adapter: true  },
    { name: 'ask',       desc: 'Read-only Q&A over indexed corpus',                              arg: '<query>',  needs_adapter: false },
    { name: 'note',      desc: 'Capture a note to .hero/knowledge/',                             arg: '<text>',   needs_adapter: false },
    { name: 'scheduled', desc: 'Convert input into a scheduled-agent definition',                arg: '<schedule>', needs_adapter: false },
  ];

  // -----------------------------------------------------------------
  // State
  // -----------------------------------------------------------------

  const state = {
    built: false,
    open: false,
    mode: 'search',                // 'search' | 'chat' | 'slash' | 'empty'
    capability: null,              // last GET /api/chat/capability response
    capabilityFetchedAt: 0,
    capabilityPromise: null,
    artifact: null,                // { kind, slug } or null
    pageContext: null,             // { page, artifact, workspace }
    slashes: null,                 // fetched once, cached for the page lifetime
    slashesPromise: null,
    activeIndex: 0,
    searchResults: [],             // [{ kind, title, sub, href }]
    searchAbort: null,
    conversationId: null,
    sseSource: null,
    chatTurnInFlight: false,
    chatMessages: [],              // [{ role: 'user'|'assistant'|'tool'|'error', ... }]
    convCost: 0,
    todayCost: 0,
    previouslyFocused: null,
    adapterPopoverOpen: false,
  };

  // Restore conversation id from sessionStorage so a refresh doesn't
  // strand the running adapter on an abandoned topic.
  try {
    const persisted = sessionStorage.getItem('hero-cmd-conv-global');
    if (persisted) state.conversationId = persisted;
  } catch (_) { /* private mode etc. — fine */ }

  // -----------------------------------------------------------------
  // DOM refs (filled in build())
  // -----------------------------------------------------------------

  const dom = {
    overlay: null,
    card: null,
    inputRow: null,
    input: null,
    kindIcon: null,
    contextRow: null,
    contextChip: null,
    body: null,
    footer: null,
    adapterChip: null,
    adapterPopover: null,
    costTicker: null,
  };

  // -----------------------------------------------------------------
  // Build the overlay (lazy — only on first ⌘K)
  // -----------------------------------------------------------------

  function build() {
    if (state.built) return;
    injectCSS();

    dom.overlay = el('div', {
      class: 'cmd-overlay',
      role: 'dialog',
      'aria-modal': 'true',
      'aria-label': 'Hero command bar',
    });

    dom.kindIcon = el('span', { class: 'cmd-kind-icon', 'aria-label': 'search', html: SVG_SEARCH });
    dom.input = el('input', {
      class: 'cmd-input',
      type: 'text',
      placeholder: 'Search, ask Hero, or type / for commands…',
      autocomplete: 'off',
      spellcheck: 'false',
    });
    dom.inputRow = el('div', { class: 'cmd-input-row' }, [
      dom.kindIcon,
      dom.input,
      el('span', { class: 'cmd-esc', text: 'ESC' }),
    ]);

    dom.contextChip = el('span', { class: 'ctx-chip' });
    dom.contextRow = el('div', { class: 'cmd-context-row', hidden: true }, [dom.contextChip]);

    dom.body = el('div', { class: 'cmd-body' });

    dom.adapterChip = el('button', {
      class: 'cmd-adapter-chip muted',
      type: 'button',
      'aria-label': 'No Hero adapter connected',
    }, [
      el('span', { class: 'ad-dot' }),
      el('span', { class: 'ad-label', text: 'no adapter' }),
    ]);
    dom.adapterPopover = el('div', { class: 'cmd-adapter-popover', role: 'menu' });
    dom.adapterChip.appendChild(dom.adapterPopover);

    dom.costTicker = el('div', { class: 'cmd-cost-ticker', text: '—' });
    dom.footer = el('div', { class: 'cmd-footer' }, [dom.adapterChip, dom.costTicker]);

    dom.card = el('div', { class: 'cmd-card' }, [
      dom.inputRow,
      dom.contextRow,
      dom.body,
      dom.footer,
    ]);
    dom.overlay.appendChild(dom.card);
    document.body.appendChild(dom.overlay);

    wireEvents();
    state.built = true;
  }

  function wireEvents() {
    dom.overlay.addEventListener('click', function (e) {
      if (e.target === dom.overlay) close();
    });

    dom.input.addEventListener('input', onInputChange);
    dom.input.addEventListener('keydown', onInputKeydown);

    dom.adapterChip.addEventListener('click', function (e) {
      // Don't capture clicks coming from inside the popover.
      if (dom.adapterPopover.contains(e.target)) return;
      e.preventDefault();
      e.stopPropagation();
      toggleAdapterPopover();
    });

    document.addEventListener('click', function (e) {
      if (!state.adapterPopoverOpen) return;
      if (dom.adapterChip.contains(e.target)) return;
      closeAdapterPopover();
    });
  }

  // -----------------------------------------------------------------
  // Page-context detection
  // -----------------------------------------------------------------

  function readPageContext() {
    const ctx = { page: null, artifact: null, workspace: null };

    const metaCtx = document.querySelector('meta[name="hero-page-context"]');
    if (metaCtx && metaCtx.content) {
      try {
        const parsed = JSON.parse(metaCtx.content);
        if (parsed && typeof parsed === 'object') {
          if (parsed.page)      ctx.page      = parsed.page;
          if (parsed.artifact)  ctx.artifact  = parsed.artifact;
          if (parsed.workspace) ctx.workspace = parsed.workspace;
        }
      } catch (_) { /* fall back to derivation below */ }
    }

    const metaArt = document.querySelector('meta[name="hero-artifact"]');
    if (metaArt && metaArt.content) {
      try {
        const parsed = JSON.parse(metaArt.content);
        if (parsed && parsed.kind && parsed.slug) ctx.artifact = parsed;
      } catch (_) { /* ignore */ }
    }

    if (!ctx.page) {
      // Derive a minimal page descriptor from the path.
      // e.g. /work/spec/per-feature-smoke-coverage → home=work, artifact=spec/per-feature-smoke-coverage
      const path = window.location.pathname || '/';
      const segments = path.split('/').filter(Boolean);
      const home = segments[0] || 'now';
      ctx.page = { home: home };
      if (!ctx.artifact && segments.length >= 3) {
        ctx.artifact = { kind: segments[1], slug: segments[2] };
      }
    }

    return ctx;
  }

  function renderContextChip() {
    if (!state.artifact) {
      dom.contextRow.hidden = true;
      dom.contextChip.innerHTML = '';
      return;
    }
    const kind = state.artifact.kind || 'item';
    const slug = state.artifact.slug || '';
    const kindLabel = kind.charAt(0).toUpperCase() + kind.slice(1);
    dom.contextChip.innerHTML = '';
    dom.contextChip.appendChild(document.createTextNode('📎 Context: ' + kindLabel + ' · '));
    dom.contextChip.appendChild(el('code', { text: slug }));
    const x = el('button', { class: 'ctx-x', type: 'button', 'aria-label': 'Remove context', text: '✕' });
    x.addEventListener('click', function (e) {
      e.preventDefault();
      e.stopPropagation();
      state.artifact = null;
      renderContextChip();
    });
    dom.contextChip.appendChild(x);
    dom.contextRow.hidden = false;
  }

  // -----------------------------------------------------------------
  // Open / close
  // -----------------------------------------------------------------

  function open() {
    build();
    if (state.open) return;
    state.previouslyFocused = document.activeElement;
    state.pageContext = readPageContext();
    state.artifact = state.pageContext.artifact || null;
    renderContextChip();

    state.open = true;
    dom.overlay.setAttribute('data-open', 'true');
    dom.input.value = '';
    dom.input.focus();

    ensureCapability().then(updateAdapterUI).catch(function () { updateAdapterUI(); });
    ensureSlashes();
    updateCostTickerUI();
    routeMode(); // initial render based on (empty) input
  }

  function close() {
    if (!state.open) return;
    state.open = false;
    dom.overlay.setAttribute('data-open', 'false');
    closeAdapterPopover();
    if (state.previouslyFocused && state.previouslyFocused.focus) {
      try { state.previouslyFocused.focus(); } catch (_) { /* noop */ }
    }
  }

  function toggle() { state.open ? close() : open(); }

  // -----------------------------------------------------------------
  // Mode routing — driven entirely off input value
  // -----------------------------------------------------------------

  function detectMode(value) {
    if (state.mode === 'chat' && state.chatMessages.length > 0) return 'chat';
    if (!value) {
      if (state.capability && state.capability.interactive === '' && state.capability.adapters.length === 0) {
        return 'empty';
      }
      return 'search';
    }
    if (value.charAt(0) === '/') return 'slash';
    if (value.charAt(0) === '?') return 'chat';
    return 'search';
  }

  function setKindIcon(mode) {
    dom.kindIcon.className = 'cmd-kind-icon' + (mode === 'chat' ? ' bolt' : mode === 'slash' ? ' slash' : '');
    if (mode === 'chat') {
      dom.kindIcon.innerHTML = SVG_BOLT;
      dom.kindIcon.setAttribute('aria-label', 'chat');
    } else if (mode === 'slash') {
      dom.kindIcon.innerHTML = '/';
      dom.kindIcon.setAttribute('aria-label', 'slash');
    } else {
      dom.kindIcon.innerHTML = SVG_SEARCH;
      dom.kindIcon.setAttribute('aria-label', 'search');
    }
  }

  function routeMode() {
    const value = dom.input.value;
    const next = detectMode(value);
    state.mode = next;
    setKindIcon(next);
    state.activeIndex = 0;

    if (next === 'empty') {
      renderEmpty();
    } else if (next === 'slash') {
      renderSlash(value);
    } else if (next === 'chat') {
      renderChat();
    } else {
      renderSearch(value);
    }
  }

  function onInputChange() {
    // Chat mode is sticky once you've sent a message — typing then is
    // composing the next turn, not switching modes.
    if (state.mode === 'chat' && state.chatMessages.length > 0) {
      // No mode rerender; we just update the height (textarea behavior
      // unnecessary here — single-line input is fine).
      return;
    }
    routeMode();
  }

  function onInputKeydown(e) {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
      return;
    }
    if (state.mode === 'search') {
      if (e.key === 'ArrowDown') { e.preventDefault(); moveActive(1); return; }
      if (e.key === 'ArrowUp')   { e.preventDefault(); moveActive(-1); return; }
      if (e.key === 'Enter') {
        e.preventDefault();
        const result = state.searchResults[state.activeIndex];
        if (result && result.href) {
          window.location.href = result.href;
        } else if (dom.input.value) {
          // No result match — promote to chat with current query.
          startChatTurn(dom.input.value);
        }
        return;
      }
      if (e.key === 'ArrowRight') {
        if (dom.input.selectionStart === dom.input.value.length) {
          e.preventDefault();
          if (dom.input.value) startChatTurn(dom.input.value);
        }
        return;
      }
    } else if (state.mode === 'slash') {
      if (e.key === 'ArrowDown') { e.preventDefault(); moveActive(1); return; }
      if (e.key === 'ArrowUp')   { e.preventDefault(); moveActive(-1); return; }
      if (e.key === 'Enter') {
        e.preventDefault();
        const slashList = filteredSlashes(dom.input.value);
        const chosen = slashList[state.activeIndex];
        if (!chosen) return;
        // If the user hasn't started typing args yet, complete the slash
        // and a trailing space so they can begin typing arguments. If
        // they already have args, send the turn.
        const rest = dom.input.value.indexOf(' ') >= 0
          ? dom.input.value.slice(dom.input.value.indexOf(' ') + 1)
          : '';
        if (!rest) {
          dom.input.value = '/' + chosen.name + ' ';
          // Keep mode in slash so the palette stays visible until they type.
          routeMode();
          return;
        }
        startChatTurn(dom.input.value, { name: chosen.name, args: rest });
        return;
      }
    } else if (state.mode === 'chat') {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        const value = dom.input.value.trim();
        if (!value || state.chatTurnInFlight) return;
        startChatTurn(value);
        return;
      }
      // ⇧Enter on a plain input just inserts a newline; the browser
      // doesn't allow newlines in <input> so the spec calls for it
      // to expand. We honor the intent by ignoring it cleanly.
    }
  }

  function moveActive(delta) {
    let max = 0;
    if (state.mode === 'search') max = state.searchResults.length;
    else if (state.mode === 'slash') max = filteredSlashes(dom.input.value).length;
    if (max <= 0) return;
    state.activeIndex = (state.activeIndex + delta + max) % max;
    if (state.mode === 'search') renderSearch(dom.input.value, /* skipFetch */ true);
    else if (state.mode === 'slash') renderSlash(dom.input.value);
  }

  // -----------------------------------------------------------------
  // Empty-state mode
  // -----------------------------------------------------------------

  function renderEmpty() {
    dom.inputRow.classList.add('muted');
    dom.body.innerHTML = '';
    const empty = el('div', { class: 'cmd-empty' });
    empty.appendChild(el('p', {
      class: 'lead',
      html: '<strong>No runner connected.</strong> <code>/ask</code> and <code>/note</code> still work here.',
    }));
    const buttons = el('div', { class: 'buttons' });
    const install = el('a', {
      class: 'btn primary',
      href: 'https://heroengine.ai/install/hero-code',
      target: '_blank',
      rel: 'noopener noreferrer',
      html: SVG_PLUS + ' Install hero-code',
    });
    const connectIDE = el('a', {
      class: 'btn ghost',
      href: 'https://heroengine.ai/install/ide',
      target: '_blank',
      rel: 'noopener noreferrer',
      html: 'Connect your IDE ' + SVG_ARROW,
    });
    buttons.appendChild(install);
    buttons.appendChild(connectIDE);
    empty.appendChild(buttons);
    empty.appendChild(el('p', {
      class: 'foot',
      text: "Using Claude Code, Cursor, or Codex with the Hero IDE adapter? Make sure it's running — it'll appear here automatically.",
    }));
    dom.body.appendChild(empty);
  }

  // -----------------------------------------------------------------
  // Search mode
  // -----------------------------------------------------------------

  const fetchSearchDebounced = debounce(fetchSearchImmediate, 150);

  async function fetchSearchImmediate(q) {
    if (state.searchAbort) {
      try { state.searchAbort.abort(); } catch (_) { /* noop */ }
    }
    const ctrl = (typeof AbortController !== 'undefined') ? new AbortController() : null;
    state.searchAbort = ctrl;
    try {
      const url = '/api/search?q=' + encodeURIComponent(q);
      const res = await fetch(url, {
        credentials: 'same-origin',
        headers: { 'Accept': 'application/json' },
        signal: ctrl ? ctrl.signal : undefined,
      });
      if (!res.ok) {
        if (res.status !== 404) {
          console.warn('hero command-bar: search returned', res.status);
        }
        state.searchResults = [];
        renderSearch(q, /* skipFetch */ true);
        return;
      }
      const data = await res.json().catch(function () { return null; });
      state.searchResults = normalizeSearchResults(data);
      renderSearch(q, /* skipFetch */ true);
    } catch (e) {
      if (e && e.name === 'AbortError') return;
      console.warn('hero command-bar: search failed', e);
      state.searchResults = [];
      renderSearch(q, /* skipFetch */ true);
    }
  }

  function normalizeSearchResults(data) {
    if (!data) return [];
    const rows = Array.isArray(data) ? data : (Array.isArray(data.results) ? data.results : []);
    const out = [];
    for (let i = 0; i < rows.length; i++) {
      const r = rows[i];
      if (!r) continue;
      out.push({
        kind: (r.kind || r.type || 'item') + '',
        title: r.title || r.name || r.slug || '(untitled)',
        sub: r.sub || r.summary || r.description || '',
        href: r.href || r.url || '',
      });
    }
    return out;
  }

  function groupByKind(rows) {
    const order = [];
    const groups = {};
    for (let i = 0; i < rows.length; i++) {
      const k = rows[i].kind;
      if (!groups[k]) { groups[k] = []; order.push(k); }
      groups[k].push(rows[i]);
    }
    return { order, groups };
  }

  function kindGroupLabel(kind) {
    const k = kind.toLowerCase();
    if (k === 'spec' || k === 'specs') return 'Specs';
    if (k === 'knowledge' || k === 'note' || k === 'notes') return 'Knowledge';
    if (k === 'code') return 'Code';
    if (k === 'commit' || k === 'commits') return 'Commits';
    return kind.charAt(0).toUpperCase() + kind.slice(1);
  }

  function kindChipClass(kind) {
    const k = kind.toLowerCase();
    if (k === 'spec')      return 'kind-chip kind-spec';
    if (k === 'knowledge' || k === 'note') return 'kind-chip kind-knowledge';
    if (k === 'code')      return 'kind-chip kind-code';
    if (k === 'commit')    return 'kind-chip kind-commit';
    return 'kind-chip kind-default';
  }

  function kindChipLabel(kind) {
    const k = kind.toLowerCase();
    if (k === 'spec')      return 'Spec';
    if (k === 'knowledge') return 'Note';
    if (k === 'note')      return 'Note';
    if (k === 'code')      return 'Code';
    if (k === 'commit')    return 'Commit';
    return kind;
  }

  function renderSearch(q, skipFetch) {
    dom.inputRow.classList.remove('muted');
    if (!skipFetch && q) fetchSearchDebounced(q);

    dom.body.innerHTML = '';
    const trimmed = (q || '').trim();
    if (!trimmed) {
      dom.body.appendChild(el('div', {
        class: 'cmd-blank',
        text: 'Type to search specs, knowledge, code, and commits. Or press / for commands.',
      }));
      appendSearchHint();
      return;
    }
    if (state.searchResults.length === 0) {
      dom.body.appendChild(el('div', {
        class: 'cmd-blank',
        text: 'No results. Press → to ask Hero about your query.',
      }));
      appendSearchHint();
      return;
    }
    const { order, groups } = groupByKind(state.searchResults);
    let flatIdx = 0;
    for (let i = 0; i < order.length; i++) {
      const k = order[i];
      dom.body.appendChild(el('div', { class: 'group-label', text: kindGroupLabel(k) }));
      const rows = groups[k];
      for (let j = 0; j < rows.length; j++) {
        const r = rows[j];
        const myIdx = flatIdx++;
        const row = el('div', {
          class: 'result-row' + (myIdx === state.activeIndex ? ' active' : ''),
        });
        const title = el('div', { class: 'result-title' });
        title.appendChild(el('span', { class: kindChipClass(r.kind), text: kindChipLabel(r.kind) }));
        title.appendChild(el('span', { html: highlight(r.title, trimmed) }));
        row.appendChild(title);
        if (r.sub) {
          row.appendChild(el('div', { class: 'result-sub', html: highlight(r.sub, trimmed) }));
        }
        row.addEventListener('mouseenter', function () {
          state.activeIndex = myIdx;
          // Cheap visual update: tweak classes in place.
          const all = dom.body.querySelectorAll('.result-row');
          for (let a = 0; a < all.length; a++) all[a].classList.toggle('active', a === myIdx);
        });
        row.addEventListener('click', function () {
          if (r.href) window.location.href = r.href;
        });
        dom.body.appendChild(row);
      }
    }
    appendSearchHint();
  }

  function appendSearchHint() {
    dom.body.appendChild(el('div', {
      class: 'mode-hint',
      html:
        '<span class="kbd">↑</span><span class="kbd">↓</span> navigate' +
        '<span class="sep">·</span>' +
        '<span class="kbd">↵</span> open' +
        '<span class="sep">·</span>' +
        '<span class="kbd">→</span> Ask Hero about your query…',
    }));
  }

  // -----------------------------------------------------------------
  // Slash mode
  // -----------------------------------------------------------------

  async function ensureSlashes() {
    if (state.slashes) return state.slashes;
    if (state.slashesPromise) return state.slashesPromise;
    state.slashesPromise = (async function () {
      try {
        const res = await jsonFetch('GET', '/api/chat/slashes');
        if (res.ok && res.body) {
          const list = Array.isArray(res.body) ? res.body : (Array.isArray(res.body.slashes) ? res.body.slashes : null);
          if (list && list.length) {
            state.slashes = list.map(function (s) {
              return {
                name: s.name || '',
                desc: s.desc || s.description || '',
                arg:  s.arg  || s.args        || '',
                needs_adapter: !!s.needs_adapter,
              };
            }).filter(function (s) { return s.name; });
            if (state.slashes.length) return state.slashes;
          }
        }
      } catch (_) { /* fall through */ }
      state.slashes = FALLBACK_SLASHES.slice();
      return state.slashes;
    })();
    return state.slashesPromise;
  }

  function filteredSlashes(value) {
    if (!state.slashes) return FALLBACK_SLASHES.slice();
    const v = (value || '').replace(/^\//, '');
    const space = v.indexOf(' ');
    const stem = (space >= 0 ? v.slice(0, space) : v).toLowerCase();
    if (!stem) return state.slashes.slice();
    return state.slashes.filter(function (s) { return s.name.toLowerCase().indexOf(stem) >= 0; });
  }

  function renderSlash(value) {
    dom.inputRow.classList.remove('muted');
    ensureSlashes().then(function () {
      // Rerender once slashes have loaded.
      if (state.mode === 'slash') paintSlash(value);
    });
    paintSlash(value);
  }

  function paintSlash(value) {
    dom.body.innerHTML = '';
    const v = (value || '');
    const stem = v.replace(/^\//, '').split(' ')[0];
    dom.body.appendChild(el('div', {
      class: 'mode-divider',
      text: 'Slash mode' + (stem ? ' · input: /' + stem : ''),
    }));
    const list = filteredSlashes(v);
    if (list.length === 0) {
      dom.body.appendChild(el('div', { class: 'cmd-blank', text: 'No matching slash command.' }));
    } else {
      if (state.activeIndex >= list.length) state.activeIndex = 0;
      for (let i = 0; i < list.length; i++) {
        const s = list[i];
        const row = el('div', { class: 'slash-row' + (i === state.activeIndex ? ' active' : '') });
        row.appendChild(el('span', { class: 'slash-name', html: '/' + highlight(s.name, stem) }));
        row.appendChild(el('span', { class: 'slash-desc', text: s.desc }));
        row.appendChild(el('span', { class: 'slash-arg', text: s.arg || '' }));
        const myIdx = i;
        row.addEventListener('mouseenter', function () {
          state.activeIndex = myIdx;
          const all = dom.body.querySelectorAll('.slash-row');
          for (let a = 0; a < all.length; a++) all[a].classList.toggle('active', a === myIdx);
        });
        row.addEventListener('click', function () {
          dom.input.value = '/' + s.name + ' ';
          dom.input.focus();
          routeMode();
        });
        dom.body.appendChild(row);
      }
    }
    dom.body.appendChild(el('div', {
      class: 'mode-hint',
      html:
        '<span class="kbd">↑</span><span class="kbd">↓</span> navigate' +
        '<span class="sep">·</span>' +
        '<span class="kbd">↵</span> select' +
        '<span class="sep">·</span>' +
        '<span class="kbd">ESC</span> close',
    }));
  }

  // -----------------------------------------------------------------
  // Chat mode
  // -----------------------------------------------------------------

  function renderChat() {
    dom.inputRow.classList.remove('muted');
    dom.body.innerHTML = '';
    dom.body.appendChild(el('div', { class: 'mode-divider', text: 'Chat mode' }));
    const area = el('div', { class: 'chat-area' });
    for (let i = 0; i < state.chatMessages.length; i++) {
      const m = state.chatMessages[i];
      area.appendChild(renderChatMessage(m));
    }
    dom.body.appendChild(area);
    dom.body.appendChild(el('div', {
      class: 'mode-hint',
      html:
        '<span class="kbd">↵</span> send' +
        '<span class="sep">·</span>' +
        '<span class="kbd">⇧↵</span> new line' +
        '<span class="sep">·</span>' +
        '<span class="kbd">ESC</span> close',
    }));
    // Always scroll to the bottom so the latest token is visible.
    dom.body.scrollTop = dom.body.scrollHeight;
  }

  function renderChatMessage(m) {
    if (m.role === 'user') {
      return el('div', { class: 'chat-user', text: m.content || '' });
    }
    if (m.role === 'tool_call') {
      const argsStr = (m.args && typeof m.args === 'object') ? safeStringify(m.args) : (m.args || '');
      const line = el('div', { class: 'tool-line' });
      line.appendChild(el('span', { class: 'arrow', text: '→' }));
      line.appendChild(document.createTextNode(' '));
      line.appendChild(el('span', { class: 'name', text: m.name || 'tool' }));
      if (argsStr) line.appendChild(el('span', { text: '(' + truncate(argsStr, 80) + ')' }));
      return line;
    }
    if (m.role === 'tool_result') {
      const line = el('div', { class: 'tool-line' });
      line.appendChild(el('span', { class: 'arrow', text: '←' }));
      line.appendChild(document.createTextNode(' '));
      line.appendChild(el('span', { class: 'name', text: m.name || 'tool' }));
      if (m.preview) line.appendChild(el('span', { text: ' ' + truncate(m.preview, 80) }));
      line.appendChild(el('span', { class: 'ok', text: '✓' }));
      return line;
    }
    if (m.role === 'error') {
      const box = el('div', { class: 'chat-error' });
      box.appendChild(document.createTextNode(m.message || 'Error'));
      if (m.link) {
        box.appendChild(document.createTextNode(' '));
        box.appendChild(el('a', { href: m.link, target: '_blank', rel: 'noopener noreferrer', text: 'Open →' }));
      }
      return box;
    }
    // assistant stream
    const div = el('div', { class: 'assistant-stream', text: m.content || '' });
    if (m.streaming) div.appendChild(el('span', { class: 'cursor' }));
    return div;
  }

  function safeStringify(o) {
    try { return JSON.stringify(o); } catch (_) { return ''; }
  }
  function truncate(s, n) {
    s = String(s == null ? '' : s);
    return s.length > n ? s.slice(0, n - 1) + '…' : s;
  }

  // -----------------------------------------------------------------
  // Chat turn lifecycle
  // -----------------------------------------------------------------

  function startChatTurn(prompt, slash) {
    if (state.chatTurnInFlight) return;
    const value = (prompt || '').trim();
    if (!value) return;

    // Flip into chat mode and clear input.
    state.mode = 'chat';
    state.chatMessages.push({ role: 'user', content: value });
    state.chatMessages.push({ role: 'assistant', content: '', streaming: true });
    dom.input.value = '';
    setKindIcon('chat');
    renderChat();

    submitTurn(value, slash || null);
  }

  async function submitTurn(prompt, slash) {
    state.chatTurnInFlight = true;
    const ctx = state.pageContext || readPageContext();
    const body = {
      prompt: prompt,
      conversation_id: state.conversationId || '',
      page_scope: 'global',
      context: {
        page: ctx.page || null,
        artifact: state.artifact || null,
        workspace: ctx.workspace || null,
      },
    };
    if (slash) body.slash = slash;

    let res;
    try {
      res = await jsonFetch('POST', '/api/chat/turn', body);
    } catch (e) {
      finalizeWithError({ message: 'Network error reaching /api/chat/turn.' });
      return;
    }
    if (!res.ok || !res.body || typeof res.body !== 'object') {
      finalizeWithError({
        message: 'Hero serve rejected the turn (' + (res ? res.status : 'no response') + ').',
      });
      return;
    }
    const convId = res.body.conversation_id || '';
    const topic  = res.body.sse_topic || '';
    if (convId) {
      state.conversationId = convId;
      try { sessionStorage.setItem('hero-cmd-conv-global', convId); } catch (_) {}
    }
    if (!topic) {
      finalizeWithError({ message: 'No SSE topic returned for this turn.' });
      return;
    }
    openSSE(topic);
  }

  function openSSE(topic) {
    closeSSE();
    let source;
    try {
      source = new EventSource('/api/events?topic=' + encodeURIComponent(topic));
    } catch (e) {
      finalizeWithError({ message: 'EventSource unsupported in this browser.' });
      return;
    }
    state.sseSource = source;

    function handlePayload(eventType, raw) {
      let payload = null;
      try { payload = JSON.parse(raw); } catch (_) { payload = null; }
      if (!payload) return;
      // Server-side topic filter is set, but defensive client-side check:
      if (payload.conversation_id && state.conversationId &&
          payload.conversation_id !== state.conversationId) {
        return;
      }
      switch (eventType) {
        case 'chat.token':       onTokenEvent(payload); break;
        case 'chat.tool_call':   onToolCallEvent(payload); break;
        case 'chat.tool_result': onToolResultEvent(payload); break;
        case 'chat.error':       onErrorEvent(payload); break;
        case 'chat.cost':        onCostEvent(payload); break;
        case 'chat.done':        onDoneEvent(payload); break;
      }
    }

    // The SSE bus may publish events with named types, or with a single
    // generic 'message' event whose data carries the type. Subscribe to
    // both shapes.
    const namedTypes = ['chat.token', 'chat.tool_call', 'chat.tool_result', 'chat.error', 'chat.cost', 'chat.done'];
    for (let i = 0; i < namedTypes.length; i++) {
      const t = namedTypes[i];
      source.addEventListener(t, function (e) { handlePayload(t, e.data); });
    }
    source.addEventListener('message', function (e) {
      // Generic frame: try to interpret data as { type, payload }.
      let frame = null;
      try { frame = JSON.parse(e.data); } catch (_) { frame = null; }
      if (!frame) return;
      const t = frame.type || frame.Type;
      const p = frame.payload || frame.Payload || frame;
      if (!t) return;
      handlePayload(t, typeof p === 'string' ? p : JSON.stringify(p));
    });

    source.addEventListener('error', function () {
      // SSE auto-reconnects; a transport error mid-turn shouldn't
      // permanently break the UI. If the turn hasn't completed within
      // ~30s of disconnect, we'll show a soft error.
      if (state.chatTurnInFlight) {
        console.warn('hero command-bar: SSE error during turn');
      }
    });
  }

  function closeSSE() {
    if (state.sseSource) {
      try { state.sseSource.close(); } catch (_) { /* noop */ }
      state.sseSource = null;
    }
  }

  function currentAssistantStream() {
    for (let i = state.chatMessages.length - 1; i >= 0; i--) {
      const m = state.chatMessages[i];
      if (m.role === 'assistant' && m.streaming) return m;
    }
    // No streaming assistant — start one.
    const fresh = { role: 'assistant', content: '', streaming: true };
    state.chatMessages.push(fresh);
    return fresh;
  }

  function onTokenEvent(p) {
    const m = currentAssistantStream();
    m.content += (p.text || '');
    renderChat();
  }

  function onToolCallEvent(p) {
    // Insert tool indicators *before* the streaming assistant message.
    insertBeforeStream({ role: 'tool_call', name: p.name || '', args: p.args || null });
    renderChat();
  }

  function onToolResultEvent(p) {
    insertBeforeStream({ role: 'tool_result', name: p.name || '', preview: p.preview || '' });
    renderChat();
  }

  function insertBeforeStream(message) {
    let insertAt = state.chatMessages.length;
    for (let i = state.chatMessages.length - 1; i >= 0; i--) {
      if (state.chatMessages[i].role === 'assistant' && state.chatMessages[i].streaming) {
        insertAt = i;
        break;
      }
    }
    state.chatMessages.splice(insertAt, 0, message);
  }

  function onErrorEvent(p) {
    insertBeforeStream({
      role: 'error',
      message: p.message || ('Error (' + (p.code || 'unknown') + ').'),
      link: p.link || '',
    });
    renderChat();
  }

  function onCostEvent(p) {
    if (typeof p.usd === 'number') {
      state.convCost += p.usd;
    }
    updateCostTickerUI();
  }

  function onDoneEvent(p) {
    if (typeof p.usd === 'number' && p.usd > 0 && state.convCost === 0) {
      // Some adapters skip incremental chat.cost and only report at done.
      state.convCost += p.usd;
    }
    for (let i = state.chatMessages.length - 1; i >= 0; i--) {
      const m = state.chatMessages[i];
      if (m.role === 'assistant' && m.streaming) {
        m.streaming = false;
        break;
      }
    }
    state.chatTurnInFlight = false;
    updateCostTickerUI();
    renderChat();
    closeSSE();
  }

  function finalizeWithError(opts) {
    insertBeforeStream({
      role: 'error',
      message: (opts && opts.message) || 'Something went wrong.',
      link: (opts && opts.link) || '',
    });
    for (let i = state.chatMessages.length - 1; i >= 0; i--) {
      const m = state.chatMessages[i];
      if (m.role === 'assistant' && m.streaming) {
        m.streaming = false;
        break;
      }
    }
    state.chatTurnInFlight = false;
    renderChat();
    closeSSE();
  }

  // -----------------------------------------------------------------
  // Adapter chip + capability + cost ticker
  // -----------------------------------------------------------------

  async function ensureCapability() {
    const now = Date.now();
    if (state.capability && (now - state.capabilityFetchedAt) < 30 * 1000) {
      return state.capability;
    }
    if (state.capabilityPromise) return state.capabilityPromise;
    state.capabilityPromise = (async function () {
      try {
        const res = await jsonFetch('GET', '/api/chat/capability');
        if (res.ok && res.body && typeof res.body === 'object') {
          state.capability = {
            adapters: Array.isArray(res.body.adapters) ? res.body.adapters : [],
            interactive: res.body.interactive || '',
            headless: res.body.headless || '',
            user_preferred: res.body.user_preferred || '',
            today_cost: typeof res.body.today_cost === 'number' ? res.body.today_cost : 0,
          };
        } else {
          state.capability = { adapters: [], interactive: '', headless: '', user_preferred: '', today_cost: 0 };
        }
      } catch (e) {
        console.warn('hero command-bar: /api/chat/capability failed', e);
        state.capability = { adapters: [], interactive: '', headless: '', user_preferred: '', today_cost: 0 };
      }
      state.capabilityFetchedAt = Date.now();
      state.capabilityPromise = null;
      if (typeof state.capability.today_cost === 'number') {
        state.todayCost = state.capability.today_cost;
      }
      return state.capability;
    })();
    return state.capabilityPromise;
  }

  function activeAdapter() {
    const cap = state.capability;
    if (!cap || !cap.interactive || !cap.adapters || !cap.adapters.length) return null;
    for (let i = 0; i < cap.adapters.length; i++) {
      if (cap.adapters[i].id === cap.interactive) return cap.adapters[i];
    }
    return null;
  }

  function updateAdapterUI() {
    const cap = state.capability;
    const active = activeAdapter();
    const label = dom.adapterChip.querySelector('.ad-label');
    const existingVer = dom.adapterChip.querySelector('.ad-ver');
    if (existingVer) existingVer.remove();

    if (!cap || !active) {
      dom.adapterChip.classList.add('muted');
      dom.adapterChip.setAttribute('aria-label', 'No Hero adapter connected');
      label.textContent = 'no adapter';
    } else {
      dom.adapterChip.classList.remove('muted');
      const adapterName = active.adapter || 'adapter';
      dom.adapterChip.setAttribute('aria-label', 'Connected adapter: ' + adapterName + (active.version ? ' ' + active.version : ''));
      label.textContent = 'via ' + adapterName;
      if (active.version) {
        const ver = el('span', { class: 'ad-ver', text: '· ' + active.version });
        // insert before the popover element so it sits inline with the label.
        dom.adapterChip.insertBefore(ver, dom.adapterPopover);
      }
    }
    // If the current mode is search and the cap is empty, switch to
    // empty so the CTA renders.
    if (state.open && state.mode === 'search' && (!cap || !cap.adapters || cap.adapters.length === 0) && !dom.input.value) {
      state.mode = 'empty';
      renderEmpty();
    }
  }

  function toggleAdapterPopover() {
    if (state.adapterPopoverOpen) { closeAdapterPopover(); return; }
    paintAdapterPopover();
    state.adapterPopoverOpen = true;
    dom.adapterPopover.setAttribute('data-open', 'true');
  }

  function closeAdapterPopover() {
    state.adapterPopoverOpen = false;
    dom.adapterPopover.setAttribute('data-open', 'false');
  }

  function paintAdapterPopover() {
    dom.adapterPopover.innerHTML = '';
    const cap = state.capability;
    const adapters = (cap && cap.adapters) ? cap.adapters : [];
    if (adapters.length === 0) {
      dom.adapterPopover.appendChild(el('div', {
        class: 'pop-empty',
        text: 'No adapters connected. Install hero-code to enable agent workflows.',
      }));
      return;
    }
    for (let i = 0; i < adapters.length; i++) {
      const a = adapters[i];
      const isActive = cap && cap.interactive === a.id;
      const btn = el('button', { class: 'pop-item' + (isActive ? ' active' : ''), type: 'button' });
      btn.appendChild(el('span', { text: a.adapter || a.id }));
      if (a.version) btn.appendChild(el('span', { class: 'pop-ver', text: a.version }));
      btn.addEventListener('click', function () {
        setPreferredAdapter(a.adapter || '');
      });
      dom.adapterPopover.appendChild(btn);
    }
  }

  async function setPreferredAdapter(adapter) {
    closeAdapterPopover();
    try {
      await jsonFetch('POST', '/api/chat/preference', { interactive_adapter: adapter });
    } catch (e) {
      console.warn('hero command-bar: /api/chat/preference failed', e);
      return;
    }
    state.capability = null;
    state.capabilityFetchedAt = 0;
    await ensureCapability();
    updateAdapterUI();
  }

  function updateCostTickerUI() {
    if (!state.convCost && !state.todayCost) {
      dom.costTicker.textContent = '—';
      return;
    }
    dom.costTicker.innerHTML = '';
    dom.costTicker.appendChild(el('strong', { text: formatUSD(state.convCost) }));
    dom.costTicker.appendChild(document.createTextNode(' this conversation'));
    if (state.todayCost) {
      dom.costTicker.appendChild(el('span', { class: 'sep', text: '·' }));
      dom.costTicker.appendChild(el('strong', { text: formatUSD(state.todayCost) }));
      dom.costTicker.appendChild(document.createTextNode(' today'));
    }
  }

  // -----------------------------------------------------------------
  // Global hotkey binding (preserves placeholder's typing-target guard)
  // -----------------------------------------------------------------

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && state.open) {
      e.preventDefault();
      close();
      return;
    }
    const mod = e.metaKey || e.ctrlKey;
    if (!mod || (e.key !== 'k' && e.key !== 'K')) return;
    // While the overlay is open, the input lives inside it — guard only
    // catches typing targets *outside* the overlay so people can ⌘K to
    // refocus from anywhere.
    const active = document.activeElement;
    if (!state.open && isTypingTarget(active)) return;
    e.preventDefault();
    if (state.open) {
      dom.input.focus();
      dom.input.select();
    } else {
      open();
      // Legacy hook: any data-command-bar-trigger element still works
      // for tests / e2e probes that simulated the placeholder's click.
      const trigger = document.querySelector('[data-command-bar-trigger]');
      if (trigger) { try { trigger.click(); } catch (_) {} }
    }
  });
})();
