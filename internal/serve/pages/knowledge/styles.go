package knowledge

// knowledgeStyles is the Knowledge-home-specific stylesheet, inlined as
// HeadExtra so we don't fan out static-asset routes for a single page.
// The chrome / shared-fragment styles live in the shell's shell.css.
// This covers only the Knowledge sections that no other home reuses:
// card grid, kind chips, traversal grid, detail pane, neighbor rows,
// staleness block, and the empty-state helpers.
//
// Tokens (`--hero-blue-*`, `--ink-*`, `--bg-*`, `--border*`, `--warn`,
// `--success`, `--danger`) are owned by shell.css; this stylesheet
// only references them. Additional color tokens used by the mockup
// (purple, green, amber) are declared here as fall-backs so the page
// renders consistently when the shell tokens aren't yet shipping them.
const knowledgeStyles = `<style>
:root {
  --kn-purple-50: #f5f3ff;
  --kn-purple-700: #6d28d9;
  --kn-green-50: #ecfdf5;
  --kn-green-700: #047857;
  --kn-amber-50: #fffbeb;
  --kn-amber-500: #f59e0b;
  --kn-amber-700: #b45309;
}

/* --- Section --- */
.kn-section { padding: 28px 0; scroll-margin-top: 72px; }
.kn-section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 18px;
  gap: 16px;
}
.kn-section-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--hero-ink);
  margin: 0;
  letter-spacing: -0.005em;
}
.kn-section-meta { color: var(--ink-4); font-size: 13px; font-weight: 400; margin-left: 8px; }
.kn-section-link {
  font-size: 13px;
  color: var(--hero-blue-700);
  font-weight: 500;
}

/* --- Empty state --- */
.kn-empty {
  font-size: 13px;
  color: var(--ink-4);
  padding: 14px 8px;
  font-style: italic;
}
.kn-empty-block {
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--bg-soft);
  padding: 16px 20px;
  font-size: 13px;
  color: var(--ink-3);
}
.kn-empty-block strong { color: var(--hero-ink); font-weight: 600; }
.kn-empty-block p { margin: 6px 0 0 0; line-height: 1.55; }

/* --- Browse card grid --- */
.kn-card-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.kn-card {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.kn-card:hover { border-color: var(--border-strong); }
.kn-card-top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.kn-card-slug {
  font-size: 11px;
  color: var(--ink-4);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.kn-card-title {
  font-size: 14px;
  font-weight: 600;
  margin: 4px 0 0 0;
  color: var(--hero-ink);
  letter-spacing: -0.005em;
  line-height: 1.3;
}
.kn-card-title a { color: var(--hero-ink); }
.kn-card-title a:hover { color: var(--hero-blue-700); text-decoration: none; }
.kn-card-desc {
  font-size: 12.5px;
  color: var(--ink-3);
  margin: 0;
  line-height: 1.5;
}
.kn-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  color: var(--ink-4);
  margin-top: auto;
  padding-top: 10px;
  border-top: 1px solid var(--border);
}

/* --- Kind chips --- */
.kn-kind-chip {
  display: inline-flex;
  align-items: center;
  padding: 2px 9px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.kn-kind-decision   { background: var(--kn-purple-50); color: var(--kn-purple-700); }
.kn-kind-convention { background: var(--kn-green-50);  color: var(--kn-green-700); }
.kn-kind-spec       { background: var(--hero-blue-50); color: var(--hero-blue-700); }
.kn-kind-learning,
.kn-kind-pattern    { background: var(--kn-amber-50); color: var(--kn-amber-700); }
.kn-kind-note,
.kn-kind-rule,
.kn-kind-context    { background: var(--bg-softer); color: var(--ink-3); }

/* --- Traversal grid (provenance section) --- */
.kn-traversal-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(0, 1fr);
  gap: 24px;
}
.kn-graph-pane,
.kn-detail-pane {
  background: var(--bg-soft);
  border-radius: 12px;
  padding: 20px;
  min-height: 320px;
}
.kn-detail-pane { padding: 22px 24px; display: flex; flex-direction: column; gap: 14px; }
.kn-graph-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--ink-4);
}
.kn-target-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 9px;
  background: var(--hero-blue-50);
  border-radius: 6px;
  color: var(--hero-blue-700);
  font-size: 11px;
  font-weight: 500;
}
.kn-graph-wrap {
  border-radius: 8px;
  background: var(--bg-panel);
  padding: 60px 20px;
  text-align: center;
}
.kn-chain-crumb {
  font-size: 12px;
  color: var(--ink-4);
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.kn-chain-crumb .step { color: var(--ink-3); }
.kn-chain-crumb .arrow { color: var(--ink-5); }
.kn-type-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.kn-type-chip {
  display: inline-flex;
  align-items: center;
  padding: 2px 9px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.kn-type-decision   { background: var(--kn-purple-50); color: var(--kn-purple-700); }
.kn-type-spec       { background: var(--hero-blue-50); color: var(--hero-blue-700); }
.kn-type-convention { background: var(--kn-green-50);  color: var(--kn-green-700); }
.kn-type-learning   { background: var(--kn-amber-50);  color: var(--kn-amber-700); }
.kn-status-chip {
  display: inline-flex;
  align-items: center;
  padding: 2px 9px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
  background: var(--kn-green-50);
  color: var(--kn-green-700);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.kn-slug-text { font-size: 12px; color: var(--ink-3); }
.kn-date-text { font-size: 12px; color: var(--ink-4); margin-left: auto; }
.kn-detail-title {
  font-size: 19px;
  font-weight: 600;
  color: var(--hero-ink);
  margin: 0;
  letter-spacing: -0.01em;
  line-height: 1.3;
}
.kn-body-h {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--ink-4);
  font-weight: 600;
  margin: 6px 0 4px 0;
}
.kn-body-p {
  font-size: 13px;
  color: var(--ink-2);
  line-height: 1.6;
}

/* --- Summary --- */
.kn-summary-block {
  max-width: 760px;
  font-size: 15px;
  color: var(--ink-2);
  line-height: 1.7;
}
.kn-summary-block p { margin: 0; }
.kn-authored-by {
  margin-top: 14px;
  font-size: 12px;
  color: var(--ink-4);
  display: flex;
  align-items: center;
  gap: 8px;
}
.kn-agent-dot {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: linear-gradient(135deg, #a78bfa, #6366f1);
  color: white;
  font-size: 8px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* --- Neighbor rows --- */
.kn-neighbor-list { display: flex; flex-direction: column; }
.kn-neighbor-row {
  display: grid;
  grid-template-columns: 90px 240px 1fr;
  gap: 16px;
  align-items: center;
  padding: 12px 8px;
  border-bottom: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
}
.kn-neighbor-row:last-child { border-bottom: none; }
.kn-neighbor-row:hover { background: var(--bg-soft); }
.kn-neighbor-slug { color: var(--hero-ink); font-weight: 500; font-size: 12.5px; }
.kn-neighbor-slug:hover { color: var(--hero-blue-700); text-decoration: none; }
.kn-neighbor-desc { color: var(--ink-3); }

/* --- Staleness --- */
.kn-stale-block {
  background: var(--kn-amber-50);
  border: 1px solid #fde68a;
  border-radius: 10px;
  padding: 16px 20px;
  display: flex;
  align-items: flex-start;
  gap: 14px;
  max-width: 920px;
}
.kn-stale-icon {
  width: 28px;
  height: 28px;
  color: var(--kn-amber-700);
  flex-shrink: 0;
}
.kn-stale-body {
  flex: 1;
  font-size: 13px;
  color: var(--ink-2);
  line-height: 1.55;
}
.kn-stale-body strong { color: var(--kn-amber-700); font-weight: 600; }
.kn-stale-actions {
  margin-top: 8px;
  display: flex;
  gap: 16px;
  font-size: 13px;
}
.kn-stale-actions a { color: var(--kn-amber-700); font-weight: 500; }
.kn-stale-actions a.muted { color: var(--ink-4); }

@media (max-width: 980px) {
  .kn-traversal-grid { grid-template-columns: 1fr; }
  .kn-card-grid      { grid-template-columns: 1fr; }
  .kn-neighbor-row   { grid-template-columns: 80px 1fr; }
  .kn-neighbor-row .kn-neighbor-desc { grid-column: 1 / -1; margin-top: 4px; }
}
</style>`

// knowledgeScript is the per-page inline JS for the Knowledge home: an
// SSE subscriber that re-fetches the named section fragment on each
// server-sent event. Mirrors the Now-home pattern verbatim — only the
// section names and channel path differ.
const knowledgeScript = `<script>
(function () {
  if (typeof EventSource === 'undefined') return;
  var es;
  try {
    es = new EventSource('/api/knowledge/events');
  } catch (err) {
    return;
  }
  function refresh(section) {
    fetch('/api/knowledge/' + section, { headers: { 'Accept': 'text/html' } })
      .then(function (r) { return r.ok ? r.text() : null; })
      .then(function (html) {
        if (html == null) return;
        var target = document.getElementById('knowledge-' + section);
        if (!target) return;
        target.outerHTML = html;
      })
      .catch(function () { /* swallow */ });
  }
  ['browse', 'provenance', 'summary', 'neighbors', 'staleness'].forEach(function (s) {
    es.addEventListener(s, function () { refresh(s); });
  });
  window.addEventListener('beforeunload', function () { try { es.close(); } catch (e) {} });
})();
</script>`
