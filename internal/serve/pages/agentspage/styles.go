package agentspage

// agentsStyles is the Agents-home-specific stylesheet, inlined as
// HeadExtra so we don't fan out static asset routes for a single home.
// All shared chrome and fragment styles live in the shell's shell.css;
// this stylesheet covers only the Agents-specific shapes lifted from
// the mockup at .hero/planning/features/hero-agents-home/mockups/
// 01-agents-sessions.html — sub-nav active tab tint, sessions list
// separator-line layout, light-grey transcript panel, proposal-preview
// amber panel, approval-row + timeline-row + compact-row.
//
// Tokens (`--hero-blue-*`, `--ink-*`, `--bg-*`, `--border*`, `--warn`,
// `--success`, `--danger`, `--bg-panel`, `--warn-bg`, `--warn-border`)
// are owned by shell.css; this stylesheet only references them. If a
// token is missing on the shell side, locally-shadowed CSS variables
// at the top of the block keep colors faithful to the mockup.
const agentsStyles = `<style>
:root {
  --warn-bg: #fff7ed;
  --warn-border: #fde3c2;
  --bg-panel: #f7f8fa;
}

/* --- Sub-nav badges (shell renders the sub-nav frame; we only paint
       the badge variants here) --- */
.subnav-tab .badge {
  font-size: 11px;
  color: var(--ink-4);
  background: var(--bg-softer);
  border-radius: 10px;
  padding: 1px 7px;
  font-weight: 500;
}
.subnav-tab.active .badge {
  background: var(--hero-blue-50);
  color: var(--hero-blue-700);
}
.subnav-tab.amber .badge {
  background: var(--warn-bg);
  color: var(--warn);
}

/* --- Section primitives --- */
.agents-section { padding: 28px 0; scroll-margin-top: 72px; }
.agents-section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 18px;
  gap: 16px;
}
.agents-section-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--hero-ink);
  margin: 0;
  letter-spacing: -0.005em;
}
.agents-section-meta { color: var(--ink-4); font-size: 13px; font-weight: 400; margin-left: 8px; }
.agents-section-link {
  font-size: 13px;
  color: var(--hero-blue-700);
  font-weight: 500;
}
.section-actions {
  display: flex;
  align-items: center;
  gap: 18px;
}
.agents-empty {
  font-size: 13px;
  color: var(--ink-4);
  padding: 14px 8px;
  font-style: italic;
}

/* --- Filter chips --- */
.filter-chips { display: flex; align-items: center; gap: 6px; }
.filter-chip {
  font-size: 12px;
  color: var(--ink-3);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 999px;
  padding: 4px 11px;
  font-weight: 500;
  cursor: pointer;
}
.filter-chip:hover { background: var(--bg-softer); color: var(--hero-ink); }
.filter-chip.active {
  background: var(--hero-blue-50);
  color: var(--hero-blue-700);
  border-color: #d6e7fb;
}

/* --- Session blocks (separator lines — NOT cards) --- */
.sessions-list { display: flex; flex-direction: column; }
.session-block {
  padding: 22px 0;
  border-top: 1px solid var(--border);
}
.session-block:first-child { border-top: none; padding-top: 4px; }

.session-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}
.agent-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
  background: linear-gradient(135deg, var(--hero-blue-500), var(--hero-blue-900));
}
.agent-avatar.opus     { background: linear-gradient(135deg, #a78bfa, #6366f1); }
.agent-avatar.sonnet   { background: linear-gradient(135deg, #60a5fa, #2a6cb5); }
.agent-avatar.engineer { background: linear-gradient(135deg, #f59e0b, #b45309); }
.agent-avatar.debug    { background: linear-gradient(135deg, #ef4444, #b91c1c); }

.session-id-block { min-width: 0; flex: 1; }
.session-id-block .agent-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--hero-ink);
}
.session-id-block .on {
  font-size: 14px;
  color: var(--ink-3);
  margin-left: 2px;
}
.session-id-block .on a { color: var(--hero-ink); font-weight: 500; }
.session-id-block .on a:hover { color: var(--hero-blue-700); text-decoration: none; }

.session-status-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 3px 10px;
  border-radius: 4px;
}
.session-status-tag.live  { color: var(--hero-blue-700); background: var(--hero-blue-50); }
.session-status-tag.amber { color: var(--warn); background: var(--warn-bg); }

.live-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--hero-blue-500);
  box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.7);
  animation: agents-pulse 1.2s ease-out infinite;
}
.live-dot.amber {
  background: var(--warn);
  box-shadow: 0 0 0 0 rgba(217, 119, 6, 0.7);
  animation: agents-pulse-amber 1.2s ease-out infinite;
}
.live-dot.sm { width: 7px; height: 7px; }
@keyframes agents-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.45); }
  70%  { box-shadow: 0 0 0 6px rgba(108, 182, 255, 0); }
  100% { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0); }
}
@keyframes agents-pulse-amber {
  0%   { box-shadow: 0 0 0 0 rgba(217, 119, 6, 0.45); }
  70%  { box-shadow: 0 0 0 6px rgba(217, 119, 6, 0); }
  100% { box-shadow: 0 0 0 0 rgba(217, 119, 6, 0); }
}

.session-meta {
  font-size: 12px;
  color: var(--ink-4);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0;
  margin-bottom: 14px;
  padding-left: 44px;
}
.session-meta .meta-item { display: inline-flex; align-items: center; gap: 4px; }
.session-meta .sep { color: var(--ink-5); margin: 0 8px; }
.session-meta .cost { color: var(--hero-ink); font-weight: 600; }
.session-meta code {
  font-size: 11px;
  color: var(--ink-3);
  background: var(--bg-softer);
  border-radius: 4px;
  padding: 1px 6px;
}

/* --- Light-grey transcript preview panel (NOT a dark terminal) --- */
.transcript {
  margin-left: 44px;
  background: var(--bg-panel);
  border-radius: 8px;
  padding: 14px 18px;
  font-family: 'SF Mono', ui-monospace, Menlo, monospace;
  font-size: 12px;
  line-height: 1.75;
  color: var(--ink-2);
  margin-bottom: 14px;
  overflow: hidden;
}
.transcript .line { display: block; }
.transcript .role { color: var(--ink-4); font-weight: 600; margin-right: 6px; }
.transcript .role.assistant { color: var(--hero-blue-700); }
.transcript .tool { color: var(--ink-4); }
.transcript .ok   { color: var(--success); }
.transcript .pending { color: var(--warn); font-style: italic; }
.transcript .danger  { color: var(--danger); }
.transcript .dim     { color: var(--ink-4); }
.transcript .streaming::after {
  content: '';
  display: inline-block;
  width: 7px;
  height: 13px;
  background: var(--hero-blue-700);
  margin-left: 3px;
  vertical-align: text-bottom;
  animation: agents-blink 0.95s steps(2, start) infinite;
}
@keyframes agents-blink { to { visibility: hidden; } }
.transcript.compact { line-height: 1.6; padding: 12px 16px; }

/* --- Proposal-preview (amber variant) --- */
.proposal-preview {
  margin-left: 44px;
  margin-bottom: 14px;
  border: 1px solid var(--warn-border);
  background: #fffaf2;
  border-radius: 8px;
  padding: 14px 16px;
}
.proposal-preview .pp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--warn);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.proposal-preview .pp-files {
  color: var(--ink-3);
  font-weight: 500;
  text-transform: none;
  letter-spacing: 0;
  font-size: 12px;
}
.proposal-preview .pp-files code {
  color: var(--ink-3);
  background: var(--bg-softer);
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 11px;
}
.proposal-preview .pp-diff {
  font-family: 'SF Mono', ui-monospace, Menlo, monospace;
  font-size: 11.5px;
  line-height: 1.7;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 10px 12px;
  color: var(--ink-2);
  overflow: hidden;
}
.pp-diff .diff-add { background: #e7f7ec; color: #0a6b2c; display: block; padding-left: 6px; }
.pp-diff .diff-rem { background: #fdecec; color: #9b1c1c; display: block; padding-left: 6px; }
.pp-diff .diff-ctx { color: var(--ink-3); display: block; padding-left: 6px; }

/* --- Inline session actions + tool inventory strip --- */
.session-actions {
  padding-left: 44px;
  display: flex;
  align-items: center;
  gap: 18px;
  font-size: 13px;
  font-weight: 500;
  flex-wrap: wrap;
}
.session-actions .primary { color: var(--hero-blue-700); }
.session-actions .danger  { color: var(--danger); }
.session-actions .muted   { color: var(--ink-4); }
.tool-strip {
  padding-left: 44px;
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 11px;
  color: var(--ink-4);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 500;
}
.tool-strip .tool-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--bg-softer);
  border-radius: 4px;
  padding: 3px 8px;
  color: var(--ink-3);
  font-family: 'SF Mono', ui-monospace, Menlo, monospace;
  font-size: 11px;
  text-transform: none;
  letter-spacing: 0;
}
.tool-strip .tool-pill .n { color: var(--hero-ink); font-weight: 600; }

/* --- Approval row (flat list, not cards) --- */
.approval-list { display: flex; flex-direction: column; }
.approval-row {
  display: grid;
  grid-template-columns: 24px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 8px;
  border-bottom: 1px solid var(--border);
  border-radius: 6px;
}
.approval-row:last-child { border-bottom: none; }
.approval-row:hover { background: var(--bg-soft); }
.approval-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin: 0 auto;
  background: var(--warn);
}
.approval-main { min-width: 0; }
.approval-summary {
  font-size: 14px;
  color: var(--hero-ink);
  font-weight: 500;
  margin-bottom: 2px;
}
.approval-summary a { color: var(--hero-ink); }
.approval-summary a:hover { color: var(--hero-blue-700); text-decoration: none; }
.approval-meta {
  font-size: 12px;
  color: var(--ink-4);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.approval-meta .sep { color: var(--ink-5); }
.approval-meta .add { color: var(--success); font-weight: 600; }
.approval-meta .rem { color: var(--danger); font-weight: 600; }
.approval-actions {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 13px;
  font-weight: 500;
}
.approval-actions .danger { color: var(--danger); }

/* --- Completed timeline --- */
.timeline-list { display: flex; flex-direction: column; }
.timeline-row {
  display: grid;
  grid-template-columns: 56px 22px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 10px 8px;
  font-size: 13px;
  border-radius: 6px;
}
.timeline-row:hover { background: var(--bg-soft); }
.timeline-time {
  font-size: 12px;
  color: var(--ink-4);
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.timeline-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--ink-4);
}
.timeline-icon.ok     { color: var(--success); }
.timeline-icon.warn   { color: var(--warn); }
.timeline-icon.review { color: var(--hero-blue-700); }
.timeline-text { color: var(--ink-2); min-width: 0; }
.timeline-text .agent { color: var(--ink-3); }
.timeline-text a { color: var(--hero-ink); font-weight: 500; }
.timeline-text a:hover { color: var(--hero-blue-700); text-decoration: none; }
.timeline-meta {
  font-size: 12px;
  color: var(--ink-4);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* --- Scheduled/automations preview split --- */
.split-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 28px;
}
.split-block .split-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 12px;
}
.split-block .split-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--hero-ink);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.compact-list { display: flex; flex-direction: column; }
.compact-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 8px;
  border-top: 1px solid var(--border);
  border-radius: 6px;
}
.compact-row:first-child { border-top: none; }
.compact-row:hover { background: var(--bg-soft); }
.compact-row .main { flex: 1; min-width: 0; }
.compact-row .name { font-size: 14px; color: var(--hero-ink); font-weight: 500; }
.compact-row .sub  { font-size: 12px; color: var(--ink-4); }
.compact-row .sub code {
  font-size: 11px;
  background: var(--bg-softer);
  color: var(--ink-3);
  padding: 1px 6px;
  border-radius: 4px;
}
.compact-row .when {
  font-size: 12px;
  color: var(--ink-3);
  text-align: right;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.compact-row .when .when-sub { color: var(--ink-4); font-size: 11px; }

@media (max-width: 900px) {
  .split-grid { grid-template-columns: 1fr; }
  .session-meta, .session-actions, .tool-strip, .transcript, .proposal-preview {
    padding-left: 0;
    margin-left: 0;
  }
}
</style>`

// agentsScript is the per-page inline JS for the Agents home: filter
// chip visual toggle, SSE subscriber for live section refresh, and the
// metric-tab toggler is shipped by the shell's tabbed-metric-strip
// fragment so we do not duplicate it.
const agentsScript = `<script>
(function () {
  // Filter chips — visual toggle only (server-side filtering arrives
  // when the live ledger lands; v1 keeps the chips functional for layout
  // without an XHR round-trip).
  document.addEventListener('click', function (e) {
    var chip = e.target.closest('.filter-chip');
    if (!chip) return;
    var row = chip.parentElement;
    if (!row) return;
    row.querySelectorAll('.filter-chip').forEach(function (c) { c.classList.remove('active'); });
    chip.classList.add('active');
  });

  // SSE subscriber — re-fetches the named section fragment on each
  // event. The server publishes 'sessions', 'approvals', 'completed',
  // and 'scheduled-preview' event names; we re-render in place.
  if (typeof EventSource === 'undefined') return;
  var es;
  try {
    es = new EventSource('/api/agents/events');
  } catch (err) {
    return;
  }
  function refresh(section) {
    fetch('/api/agents/' + section, { headers: { 'Accept': 'text/html' } })
      .then(function (r) { return r.ok ? r.text() : null; })
      .then(function (html) {
        if (html == null) return;
        var target = document.getElementById('agents-' + section);
        if (!target) return;
        target.outerHTML = html;
      })
      .catch(function () { /* swallow */ });
  }
  ['sessions', 'approvals', 'completed', 'scheduled-preview'].forEach(function (s) {
    es.addEventListener(s, function () { refresh(s); });
  });
  window.addEventListener('beforeunload', function () { try { es.close(); } catch (e) {} });
})();
</script>`
