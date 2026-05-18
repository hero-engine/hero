package now

// nowStyles is the Now-home-specific stylesheet, inlined as HeadExtra
// so we don't fan out static-asset routes for a single page. The
// chrome / shared-fragment styles live in the shell's shell.css. This
// covers only the Now sections that no other home reuses: inbox list,
// plate grid, agents grid, transcript preview, feed/timeline rows, and
// the launch input.
//
// Tokens (`--hero-blue-*`, `--ink-*`, `--bg-*`, `--border*`, `--warn`,
// `--success`, `--danger`) are owned by shell.css; this stylesheet only
// references them.
const nowStyles = `<style>
/* --- Inbox --- */
.now-inbox-list { display: flex; flex-direction: column; }
.now-inbox-row {
  display: grid;
  grid-template-columns: 24px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 8px;
  border-bottom: 1px solid var(--border);
  border-radius: 6px;
}
.now-inbox-row:last-child { border-bottom: none; }
.now-inbox-row:hover { background: var(--bg-soft); }
.now-inbox-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin: 0 auto;
}
.now-dot-proposal { background: var(--hero-blue-500); }
.now-dot-handoff  { background: #a855f7; }
.now-dot-import   { background: var(--warn); }
.now-dot-review   { background: var(--success); }
.now-inbox-main { min-width: 0; }
.now-inbox-summary {
  font-size: 14px;
  color: var(--hero-ink);
  font-weight: 500;
  margin-bottom: 2px;
}
.now-inbox-summary a { color: var(--hero-ink); }
.now-inbox-summary a:hover { color: var(--hero-blue-700); text-decoration: none; }
.now-inbox-meta {
  font-size: 12px;
  color: var(--ink-4);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.now-inbox-meta .sep { color: var(--ink-5); }
.now-inbox-actions {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 13px;
  font-weight: 500;
}
.now-inbox-actions .danger { color: var(--danger); }
.now-inbox-actions .muted  { color: var(--ink-4); }

/* --- Section --- */
.now-section { padding: 28px 0; scroll-margin-top: 72px; }
.now-section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 18px;
  gap: 16px;
}
.now-section-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--hero-ink);
  margin: 0;
  letter-spacing: -0.005em;
}
.now-section-meta { color: var(--ink-4); font-size: 13px; font-weight: 400; margin-left: 8px; }
.now-section-link {
  font-size: 13px;
  color: var(--hero-blue-700);
  font-weight: 500;
}

/* --- Empty state (inline per section) --- */
.now-empty {
  font-size: 13px;
  color: var(--ink-4);
  padding: 14px 8px;
  font-style: italic;
}

/* --- Plate --- */
.now-plate-grid {
  display: grid;
  grid-template-columns: 1.5fr 1fr;
  gap: 24px;
}
.now-plate {
  background: var(--bg-soft);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 22px 24px;
}
.now-plate-secondary {
  background: transparent;
  border-color: var(--border);
}
.now-plate-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}
.now-plate-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--hero-ink);
  margin: 0;
  letter-spacing: -0.005em;
}
.now-plate-title a { color: var(--hero-ink); }
.now-plate-title a:hover { color: var(--hero-blue-700); text-decoration: none; }
.now-plate-desc {
  color: var(--ink-3);
  font-size: 13px;
  margin: 4px 0 16px 0;
}
.now-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 9px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
}
.now-status-delivering { background: var(--hero-blue-50); color: var(--hero-blue-700); }
.now-status-review     { background: #fff7ed; color: var(--warn); }
.now-status-planning   { background: var(--bg-softer); color: var(--ink-3); }
.now-status-completed  { background: #ecfdf5; color: var(--success); }
.now-progress-row {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}
.now-progress-line {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--ink-3);
}
.now-progress-line .label { min-width: 130px; color: var(--ink-3); }
.now-progress-bar {
  flex: 1;
  height: 6px;
  background: var(--border);
  border-radius: 3px;
  overflow: hidden;
}
.now-progress-fill { height: 100%; background: var(--hero-blue-500); border-radius: 3px; }
.now-progress-fill.success { background: var(--success); }
.now-progress-line .value { color: var(--hero-ink); font-weight: 600; font-variant-numeric: tabular-nums; min-width: 56px; text-align: right; }
.now-plate-meta {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--ink-4);
  padding-top: 14px;
  border-top: 1px solid var(--border);
  margin-top: 4px;
}
.now-meta-item { display: flex; align-items: center; gap: 6px; }
.now-meta-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--ink-5);
}
.now-meta-dot.live {
  background: var(--hero-blue-500);
  box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.7);
  animation: now-pulse 1.8s ease-out infinite;
}
@keyframes now-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.45); }
  70%  { box-shadow: 0 0 0 6px rgba(108, 182, 255, 0); }
  100% { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0); }
}
.now-plate-actions {
  margin-top: 14px;
  display: flex;
  gap: 18px;
  flex-wrap: wrap;
  font-size: 13px;
}
.now-plate-actions a { font-weight: 500; }

/* --- Agents --- */
.now-agents-grid {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 28px;
}
.now-agent-block-title {
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-weight: 600;
  color: var(--ink-4);
  margin: 0 0 12px 0;
}
.now-agent-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 20px;
  background: var(--bg);
}
.now-agent-card-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.now-agent-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, #a78bfa, #6366f1);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
}
.now-agent-name { font-size: 14px; font-weight: 600; color: var(--hero-ink); }
.now-agent-on   { font-size: 12px; color: var(--ink-3); }
.now-agent-on a { font-weight: 500; }
.now-live-tag {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--hero-blue-700);
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.now-transcript {
  background: var(--bg-panel);
  border-radius: 8px;
  padding: 14px 16px;
  font-family: 'SF Mono', ui-monospace, Menlo, monospace;
  font-size: 12px;
  line-height: 1.7;
  color: var(--ink-2);
  margin: 12px 0 14px 0;
}
.now-transcript .role { color: var(--ink-4); font-weight: 600; margin-right: 6px; }
.now-transcript .role.assistant { color: var(--hero-blue-700); }
.now-transcript .tool { color: var(--ink-4); }
.now-transcript .ok { color: var(--success); }
.now-transcript .pending { color: var(--ink-4); font-style: italic; }
.now-agent-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--border);
  font-size: 12px;
  color: var(--ink-4);
}
.now-agent-foot .cost { color: var(--hero-ink); font-weight: 600; }
.now-agent-foot a { font-weight: 500; }
.now-today-stats {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 18px;
}
.now-today-stat {
  background: var(--bg-soft);
  border-radius: 8px;
  padding: 12px 14px;
}
.now-today-stat .v {
  font-size: 18px;
  font-weight: 600;
  color: var(--hero-ink);
  letter-spacing: -0.01em;
  font-variant-numeric: tabular-nums;
}
.now-today-stat .v .qual { font-size: 13px; color: var(--ink-4); font-weight: 500; margin-left: 2px; }
.now-today-stat .v .qual.warn { color: var(--warn); }
.now-today-stat .l {
  font-size: 11px;
  color: var(--ink-4);
  margin-top: 2px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  font-weight: 500;
}
.now-today-list { display: flex; flex-direction: column; }
.now-today-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  font-size: 13px;
  color: var(--ink-2);
  border-top: 1px solid var(--border);
}
.now-today-row:first-child { border-top: none; }
.now-today-row .status {
  width: 14px;
  color: var(--success);
  display: flex;
  align-items: center;
  justify-content: center;
}
.now-today-row .spec { color: var(--hero-ink); font-weight: 500; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.now-today-row .dur  { font-size: 12px; color: var(--ink-4); font-variant-numeric: tabular-nums; }
.now-sparkline-mini {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 16px;
  margin-top: 8px;
}
.now-sparkline-mini span {
  width: 4px;
  background: var(--hero-blue-300);
  border-radius: 1px;
}

/* --- Feed (Since you were here) --- */
.now-feed-list { display: flex; flex-direction: column; }
.now-feed-row {
  display: grid;
  grid-template-columns: 56px 22px 1fr;
  gap: 14px;
  align-items: center;
  padding: 10px 8px;
  font-size: 13px;
  border-radius: 6px;
}
.now-feed-row:hover { background: var(--bg-soft); }
.now-feed-time {
  font-size: 12px;
  color: var(--ink-4);
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.now-feed-icon {
  width: 20px;
  height: 20px;
  color: var(--ink-4);
  display: flex;
  align-items: center;
  justify-content: center;
}
.now-feed-text { color: var(--ink-2); }
.now-feed-text strong, .now-feed-text a { color: var(--hero-ink); font-weight: 500; }
.now-feed-text a:hover { color: var(--hero-blue-700); text-decoration: none; }
.now-feed-actor { color: var(--ink-4); }
.now-feed-mono  { font-family: 'SF Mono', ui-monospace, monospace; font-size: 12px; }
.now-feed-limited-hint {
  font-size: 12px;
  color: var(--ink-4);
  font-style: italic;
  padding: 0 8px 8px 8px;
}

/* --- Quick launch --- */
.now-launch { text-align: center; padding: 16px 0 8px 0; }
.now-launch-title-wrap { text-align: center; margin-bottom: 22px; }
.now-launch-input-wrap {
  max-width: 720px;
  margin: 0 auto;
  position: relative;
}
.now-launch-input {
  width: 100%;
  height: 64px;
  border-radius: 14px;
  border: 1px solid var(--border-strong);
  background: var(--bg);
  padding: 0 64px 0 56px;
  font-size: 15px;
  color: var(--hero-ink);
  font-family: inherit;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.now-launch-input::placeholder { color: var(--ink-4); }
.now-launch-input:focus {
  border-color: var(--hero-blue-500);
  box-shadow: 0 0 0 4px rgba(108, 182, 255, 0.18);
}
.now-launch-icon {
  position: absolute;
  left: 22px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--hero-blue-700);
}
.now-launch-submit {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  height: 44px;
  width: 44px;
  border-radius: 10px;
  background: var(--hero-blue-700);
  color: white;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
}
.now-launch-submit:hover { background: var(--hero-blue-900); }
.now-intent-chips {
  display: flex;
  justify-content: center;
  gap: 10px;
  margin-top: 18px;
  flex-wrap: wrap;
}
.now-intent-chip {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 12px;
  color: var(--ink-3);
  background: var(--bg-softer);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 4px 12px;
}
.now-intent-chip:hover {
  color: var(--hero-blue-700);
  border-color: var(--hero-blue-300);
  background: var(--hero-blue-50);
  text-decoration: none;
}
.now-launch-try {
  margin-top: 18px;
  font-size: 13px;
  color: var(--ink-4);
}
.now-launch-try a { margin: 0 4px; }

/* --- Tile extras (segmented bar, sparkline) --- */
.now-seg-bar {
  margin-top: 10px;
  display: flex;
  height: 6px;
  border-radius: 3px;
  overflow: hidden;
  background: rgba(0,0,0,0.06);
}
.now-seg-bar span { display: block; height: 100%; }
.now-seg-done   { background: var(--success); }
.now-seg-review { background: var(--hero-blue-500); }
.now-seg-risk   { background: var(--warn); }
.now-seg-todo   { background: var(--ink-5); }
.now-seg-legend {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 8px;
  font-size: 10px;
  color: var(--ink-4);
  flex-wrap: wrap;
}
.now-seg-legend .lg { display: inline-flex; align-items: center; gap: 4px; }
.now-seg-legend .sw { width: 7px; height: 7px; border-radius: 2px; display: inline-block; }
.now-metric-progress {
  margin-top: 8px;
  height: 5px;
  background: rgba(0,0,0,0.06);
  border-radius: 3px;
  overflow: hidden;
}
.now-metric-progress-fill { height: 100%; background: var(--hero-blue-500); border-radius: 3px; }

@media (max-width: 900px) {
  .now-plate-grid, .now-agents-grid { grid-template-columns: 1fr; }
}
</style>`

// nowScript is the per-page inline JS for the Now home: intent-chip
// prefill behavior, `/` keyboard focus on the launch input, and SSE
// subscriber for live section refresh. The metric-tab toggler is
// shipped by the shell's tabbed-metric-strip fragment (we do not
// duplicate it).
const nowScript = `<script>
(function () {
  // Intent chips populate the launch input.
  document.addEventListener('click', function (e) {
    var chip = e.target.closest('.now-intent-chip');
    if (!chip) return;
    e.preventDefault();
    var input = document.querySelector('.now-launch-input');
    if (!input) return;
    input.value = chip.textContent.trim() + ' ';
    input.focus();
  });

  // Focus the launch input on '/' when no other input is active.
  document.addEventListener('keydown', function (e) {
    if (e.key !== '/') return;
    var t = document.activeElement;
    if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)) return;
    var input = document.querySelector('.now-launch-input');
    if (!input) return;
    e.preventDefault();
    input.focus();
  });

  // SSE subscriber — re-fetches the named section fragment on each
  // event. The server publishes 'inbox', 'plate', 'agents', 'changes',
  // 'hero', and 'capability' event names. Section names map to a fetch
  // + outerHTML swap; 'hero' swaps the page-hero subhead text in place
  // (no fetch); 'capability' refetches the Quick launch section so the
  // empty-state notice flips on adapter connect/disconnect.
  if (typeof EventSource === 'undefined') return;
  var es;
  try {
    es = new EventSource('/api/now/events');
  } catch (err) {
    return; // SSE not available
  }
  function refresh(path, targetId) {
    fetch(path, { headers: { 'Accept': 'text/html' } })
      .then(function (r) { return r.ok ? r.text() : null; })
      .then(function (html) {
        if (html == null) return;
        var target = document.getElementById(targetId);
        if (!target) return;
        target.outerHTML = html;
      })
      .catch(function () { /* swallow */ });
  }
  ['inbox', 'plate', 'agents', 'changes'].forEach(function (s) {
    es.addEventListener(s, function () {
      refresh('/api/now/' + s, 'now-' + s);
    });
  });
  es.addEventListener('hero', function (e) {
    var el = document.querySelector('[data-page-hero-subhead]');
    if (el) el.textContent = e.data;
  });
  es.addEventListener('capability', function () {
    refresh('/api/now/quicklaunch', 'now-quicklaunch');
  });
  window.addEventListener('beforeunload', function () { try { es.close(); } catch (e) {} });
})();
</script>`
