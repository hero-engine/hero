package work

// workStyles is the Work-home-specific stylesheet, inlined as HeadExtra
// so we don't fan out static-asset routes for a single page. The
// chrome / shared-fragment styles live in shell.css; this covers only
// the Work classes that no other home reuses: view toolbar, roadmap
// columns, spec cards (standard + initiative + dual bars + signals),
// blocked list rows, and the recently-shipped feed rows.
//
// Tokens (`--hero-blue-*`, `--ink-*`, `--bg-*`, `--border*`, `--warn`,
// `--success`, `--danger`) are owned by shell.css; this stylesheet
// only references them.
const workStyles = `<style>
/* --- Section --- */
.work-section { padding: 28px 0; scroll-margin-top: 72px; }
.work-section.section-spacious { padding: 40px 0; }
.work-empty {
  font-size: 13px;
  color: var(--ink-4);
  padding: 14px 8px;
  font-style: italic;
}

/* --- View toolbar --- */
.view-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 24px;
  padding-bottom: 2px;
}
.view-tabs { display: flex; align-items: center; gap: 2px; }
.view-tab {
  padding: 8px 12px 10px 12px;
  color: var(--ink-4);
  font-size: 13px;
  font-weight: 500;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  background: none;
  border-top: none;
  border-left: none;
  border-right: none;
  cursor: pointer;
  text-decoration: none;
}
.view-tab:first-child { padding-left: 0; }
.view-tab:hover { color: var(--hero-ink); text-decoration: none; }
.view-tab.active {
  color: var(--hero-ink);
  border-bottom-color: var(--hero-blue-700);
  font-weight: 600;
}
.view-tab .badge {
  margin-left: 6px;
  font-size: 11px;
  color: var(--ink-4);
  background: var(--bg-softer);
  border-radius: 10px;
  padding: 1px 7px;
  font-weight: 500;
}
.view-tab.active .badge {
  background: var(--hero-blue-50);
  color: var(--hero-blue-700);
}
.filter-chips { display: flex; align-items: center; gap: 8px; }
.filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: var(--bg-softer);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 12px;
  color: var(--ink-3);
  font-weight: 500;
  cursor: pointer;
}
.filter-chip:hover {
  color: var(--hero-blue-700);
  border-color: var(--hero-blue-300);
  background: var(--hero-blue-50);
  text-decoration: none;
}
.filter-chip svg { color: var(--ink-4); }

/* --- Section header (shared with blocked/shipped) --- */
.section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 18px;
  gap: 16px;
}
.section-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--hero-ink);
  margin: 0;
  letter-spacing: -0.005em;
}
.section-meta { color: var(--ink-4); font-size: 13px; font-weight: 400; margin-left: 8px; }
.section-link {
  font-size: 13px;
  color: var(--hero-blue-700);
  font-weight: 500;
}

/* --- Roadmap columns --- */
.roadmap {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}
.roadmap-col { display: flex; flex-direction: column; gap: 16px; }
.roadmap-col-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 4px;
}
.roadmap-col-title {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--ink-3);
  display: flex;
  align-items: center;
  gap: 8px;
}
.roadmap-col-title .pulse {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--hero-blue-500);
  box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.5);
  animation: work-pulse 1.8s ease-out infinite;
}
@keyframes work-pulse {
  0%   { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.45); }
  70%  { box-shadow: 0 0 0 6px rgba(108, 182, 255, 0); }
  100% { box-shadow: 0 0 0 0 rgba(108, 182, 255, 0); }
}
.roadmap-col-count {
  font-size: 12px;
  color: var(--ink-4);
  font-weight: 500;
}

/* --- Roadmap cards --- */
.rm-card {
  background: var(--bg-soft);
  border-radius: 10px;
  padding: 16px 18px;
  transition: background 0.12s ease;
  position: relative;
}
.rm-card:hover { background: var(--bg-softer); }
.rm-card-initiative {
  background: var(--bg);
  border: 1px solid var(--border-strong);
  padding-top: 18px;
}
.rm-card-initiative::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--hero-blue-500), var(--hero-blue-700));
  border-radius: 10px 10px 0 0;
}
.rm-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  flex-wrap: wrap;
}
.rm-slug {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 12px;
  color: var(--hero-blue-700);
  font-weight: 500;
}
.rm-card:hover .rm-slug { text-decoration: underline; }
.rm-type-chip {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.rm-type-feature { background: var(--hero-blue-50); color: var(--hero-blue-700); }
.rm-type-bug { background: #fef2f2; color: var(--danger); }
.rm-type-initiative { background: #f3e8ff; color: #7c3aed; }
.rm-type-decision { background: #ecfeff; color: #0891b2; }
.rm-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--hero-ink);
  line-height: 1.4;
  margin: 0 0 10px 0;
  letter-spacing: -0.005em;
}
.rm-title-initiative {
  font-size: 15px;
  font-weight: 600;
}
.rm-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 11px;
  color: var(--ink-4);
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.rm-meta .meta-item { display: inline-flex; align-items: center; gap: 5px; }
.rm-status-dot { width: 6px; height: 6px; border-radius: 50%; }
.rm-status-delivering { background: var(--hero-blue-500); }
.rm-status-planning { background: var(--ink-5); }
.rm-status-review { background: var(--warn); }
.rm-status-blocked { background: var(--danger); }
.rm-status-done { background: var(--success); }
.rm-avatar-mini {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--hero-blue-500), var(--hero-blue-900));
  color: white;
  font-size: 8px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.rm-avatar-mini.unclaimed {
  background: var(--bg-softer);
  color: var(--ink-4);
  border: 1px dashed var(--border-strong);
}

/* --- Dual bars on delivering specs --- */
.rm-bars {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 12px;
}
.rm-bar-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 10px;
  color: var(--ink-4);
}
.rm-bar-row .lbl { min-width: 38px; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 500; }
.rm-bar {
  flex: 1;
  height: 4px;
  background: rgba(0,0,0,0.06);
  border-radius: 2px;
  overflow: hidden;
}
.rm-bar-fill { height: 100%; background: var(--hero-blue-500); border-radius: 2px; }
.rm-bar-fill.success { background: var(--success); }
.rm-bar-val {
  color: var(--ink-3);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  min-width: 38px;
  text-align: right;
}

/* --- Signal chips --- */
.rm-signals { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.rm-signal {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--ink-4);
  background: var(--bg-softer);
  padding: 2px 7px;
  border-radius: 4px;
  font-weight: 500;
}
.rm-signal.drift { color: var(--warn); background: #fff7ed; }
.rm-signal.drift-major { color: var(--danger); background: #fef2f2; }
.rm-signal.ci-pass { color: var(--success); background: #ecfdf5; }
.rm-signal.ci-fail { color: var(--danger); background: #fef2f2; }
.rm-signal.agent {
  color: var(--hero-blue-700);
  background: var(--hero-blue-50);
}
.rm-signal.agent .live-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--hero-blue-500);
  box-shadow: 0 0 0 0 rgba(108, 182, 255, 0.6);
  animation: work-pulse 1.8s ease-out infinite;
}

/* --- Initiative children --- */
.rm-children {
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.rm-child {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  padding: 3px 0;
}
.rm-child .rm-status-dot { flex-shrink: 0; }
.rm-child .child-slug {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 11px;
  color: var(--ink-2);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rm-child .child-progress {
  font-size: 10px;
  color: var(--ink-4);
  font-variant-numeric: tabular-nums;
}
.rm-expand {
  margin-top: 10px;
  font-size: 12px;
  color: var(--hero-blue-700);
  font-weight: 500;
  display: inline-block;
}
.rm-quiet { font-size: 11px; color: var(--ink-4); margin-top: 8px; }

/* --- Blocked list --- */
.blocked-list { display: flex; flex-direction: column; }
.blocked-row {
  display: grid;
  grid-template-columns: 12px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 8px;
  border-bottom: 1px solid var(--border);
  border-radius: 6px;
}
.blocked-row:last-child { border-bottom: none; }
.blocked-row:hover { background: var(--bg-soft); }
.blocked-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--danger);
  margin: 0 auto;
}
.blocked-dot.warn { background: var(--warn); }
.blocked-main { min-width: 0; }
.blocked-summary {
  font-size: 14px;
  color: var(--hero-ink);
  font-weight: 500;
  margin-bottom: 3px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.blocked-summary a { color: var(--hero-ink); }
.blocked-summary a:hover { color: var(--hero-blue-700); text-decoration: none; }
.blocked-summary .mono {
  font-family: 'SF Mono', ui-monospace, monospace;
  font-size: 13px;
  color: var(--hero-blue-700);
}
.blocked-reason {
  font-size: 12px;
  color: var(--ink-3);
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.blocked-reason .reason-chip {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: #fff7ed;
  color: var(--warn);
}
.blocked-reason .reason-chip.dep { background: var(--bg-softer); color: var(--ink-3); }
.blocked-reason .reason-chip.decision { background: #ecfeff; color: #0891b2; }
.blocked-actions {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 13px;
  font-weight: 500;
}
.blocked-actions .muted { color: var(--ink-4); }

/* --- Recently shipped feed --- */
.feed-list { display: flex; flex-direction: column; }
.feed-row {
  display: grid;
  grid-template-columns: 56px 22px 1fr;
  gap: 14px;
  align-items: center;
  padding: 10px 8px;
  font-size: 13px;
  border-radius: 6px;
}
.feed-row:hover { background: var(--bg-soft); }
.feed-time {
  font-size: 12px;
  color: var(--ink-4);
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.feed-icon {
  width: 20px;
  height: 20px;
  color: var(--ink-4);
  display: flex;
  align-items: center;
  justify-content: center;
}
.feed-text { color: var(--ink-2); }
.feed-text strong, .feed-text a { color: var(--hero-ink); font-weight: 500; }
.feed-text a:hover { color: var(--hero-blue-700); text-decoration: none; }
.feed-actor { color: var(--ink-4); }
.feed-mono { font-family: 'SF Mono', ui-monospace, monospace; font-size: 12px; color: var(--hero-blue-700); }

@media (max-width: 1000px) {
  .roadmap { grid-template-columns: 1fr; }
}
</style>`

// workScript is the per-page inline JS for the Work home: SSE
// subscriber for live section refresh. The metric-tab toggler is
// shipped by the shell's tabbed-metric-strip fragment (we do not
// duplicate it). View-toggle tabs navigate (no client-side swap) so
// no extra JS for them.
const workScript = `<script>
(function () {
  if (typeof EventSource === 'undefined') return;
  var es;
  try {
    es = new EventSource('/api/work/events');
  } catch (err) {
    return;
  }
  function refresh(section) {
    fetch('/api/work/' + section, { headers: { 'Accept': 'text/html' } })
      .then(function (r) { return r.ok ? r.text() : null; })
      .then(function (html) {
        if (html == null) return;
        var target = document.getElementById('work-' + section);
        if (!target) return;
        target.outerHTML = html;
      })
      .catch(function () { /* swallow */ });
  }
  ['roadmap', 'blocked', 'shipped'].forEach(function (s) {
    es.addEventListener(s, function () { refresh(s); });
  });
  window.addEventListener('beforeunload', function () { try { es.close(); } catch (e) {} });
})();
</script>`
