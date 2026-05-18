package people

// peopleStyles is the People & ROI home stylesheet, inlined via
// HeadExtra so we don't fan out static-asset routes for a single page.
// Tokens are owned by the shell's shell.css; this stylesheet only
// references them. Mirrors the Now-home approach.
const peopleStyles = `<style>
/* --- Sub-nav (people-specific tweaks; shell shell.css owns base) --- */
.subnav .subnav-divider {
  display: inline-block;
  width: 1px;
  height: 18px;
  background: var(--border-strong, #e3e7ee);
  margin: 0 8px;
  vertical-align: middle;
}

/* --- Section frame --- */
.people-section { padding: 28px 0; scroll-margin-top: 72px; }
.people-section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 18px;
  gap: 16px;
}
.people-section-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--hero-ink);
  margin: 0;
  letter-spacing: -0.005em;
}
.people-section-meta {
  color: var(--ink-4);
  font-size: 13px;
  font-weight: 400;
  margin-left: 8px;
}
.people-empty {
  font-size: 13px;
  color: var(--ink-4);
  padding: 14px 8px;
  font-style: italic;
}

/* --- Pulse: right-now pill, presence grid, feed --- */
.people-rightnow {
  display: inline-block;
  padding: 10px 16px;
  background: var(--bg-soft);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 14px;
  color: var(--ink-2);
  margin-bottom: 18px;
}
.people-rightnow strong { color: var(--hero-ink); font-weight: 600; }
.people-presence-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
}
.people-presence-card {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px 16px;
}
.people-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--hero-blue-500), var(--hero-blue-700));
  color: white;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
}
.people-presence-text { flex: 1; min-width: 0; }
.people-presence-name { font-size: 14px; font-weight: 600; color: var(--hero-ink); }
.people-presence-meta { font-size: 12px; color: var(--ink-4); margin-top: 2px; }
.people-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.people-badge.agent { background: var(--hero-blue-50); color: var(--hero-blue-700); }
.people-badge.awaits { background: #fff7ed; color: var(--warn); }

.people-feed-list { display: flex; flex-direction: column; }
.people-feed-row {
  display: grid;
  grid-template-columns: 48px 1fr;
  gap: 14px;
  align-items: center;
  padding: 8px 4px;
  border-radius: 4px;
}
.people-feed-row:hover { background: var(--bg-soft); }
.people-feed-time {
  font-size: 12px;
  color: var(--ink-4);
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.people-feed-text { color: var(--ink-2); font-size: 13px; }
.people-feed-text strong { color: var(--hero-ink); font-weight: 500; }
.people-feed-actor { color: var(--ink-4); }

/* --- Methodology footnote --- */
.people-methodology-note {
  max-width: 720px;
  margin: 0 auto;
  padding: 14px 18px;
  background: var(--bg-soft);
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 12.5px;
  color: var(--ink-3);
  line-height: 1.6;
  text-align: center;
}
.people-methodology-note strong { color: var(--ink-2); font-weight: 600; }
.people-methodology-note a { font-weight: 500; margin: 0 4px; }

/* --- How time was spent --- */
.people-timebar {
  width: 100%;
  height: 56px;
  display: flex;
  border-radius: 10px;
  overflow: hidden;
  background: var(--bg-softer);
}
.people-timebar span {
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 13px;
  font-weight: 600;
}
.people-timebar .time-auto { background: var(--hero-blue-700); }
.people-timebar .time-review { background: var(--hero-blue-500); }
.people-timebar .time-human { background: var(--ink-5); color: var(--hero-ink); }
.people-timebar-caption {
  max-width: 720px;
  margin: 18px 0 0 0;
  font-size: 14px;
  color: var(--ink-3);
  line-height: 1.55;
}
.people-timebar-caption strong { color: var(--hero-ink); font-weight: 600; }

/* --- Savings list --- */
.people-savings-list { display: flex; flex-direction: column; }
.people-savings-row {
  display: grid;
  grid-template-columns: 14px 1fr auto auto;
  gap: 14px;
  align-items: center;
  font-size: 14px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border);
}
.people-savings-row:last-child { border-bottom: none; }
.people-savings-dot { width: 10px; height: 10px; border-radius: 50%; }
.people-savings-label { color: var(--ink-2); font-weight: 500; }
.people-savings-pct { color: var(--ink-4); font-size: 12px; margin-top: 2px; }
.people-savings-hours { color: var(--ink-3); font-size: 13px; font-variant-numeric: tabular-nums; text-align: right; }
.people-savings-dollars { color: var(--hero-ink); font-weight: 600; font-variant-numeric: tabular-nums; text-align: right; min-width: 84px; }

/* --- Trend toggle --- */
.people-trend-toggle { display: flex; gap: 6px; flex-wrap: wrap; }
.people-toggle-chip {
  font-size: 12px;
  color: var(--ink-3);
  background: var(--bg-softer);
  border: 1px solid var(--border);
  border-radius: 999px;
  padding: 4px 12px;
  cursor: pointer;
  font-family: inherit;
}
.people-toggle-chip.active {
  color: var(--hero-blue-700);
  background: var(--hero-blue-50);
  border-color: var(--hero-blue-300);
  font-weight: 600;
}
.people-trend-placeholder {
  padding: 24px;
  font-size: 13px;
  color: var(--ink-4);
  background: var(--bg-soft);
  border: 1px dashed var(--border);
  border-radius: 8px;
  text-align: center;
}

/* --- Contributors table --- */
.people-contrib-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}
.people-contrib-table thead th {
  text-align: left;
  font-size: 11px;
  font-weight: 600;
  color: var(--ink-4);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0 12px 12px 12px;
  border-bottom: 1px solid var(--border);
}
.people-contrib-table thead th.num { text-align: right; }
.people-contrib-table tbody td {
  padding: 14px 12px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}
.people-contrib-table tbody tr:last-child td { border-bottom: none; }
.people-contrib-name { display: flex; align-items: center; gap: 12px; }
.people-contrib-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  color: white;
  flex-shrink: 0;
  background: linear-gradient(135deg, #94a3b8, #475569);
}
.people-contrib-name-text { display: flex; flex-direction: column; }
.people-contrib-name-text .nm { color: var(--hero-ink); font-weight: 500; }
.people-contrib-badge {
  display: inline-block;
  font-size: 9.5px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 3px;
  margin-top: 2px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  align-self: flex-start;
}
.people-contrib-badge.badge-human { background: var(--bg-softer); color: var(--ink-3); }
.people-contrib-badge.badge-agent { background: var(--hero-blue-50); color: var(--hero-blue-700); }
.num-cell { text-align: right; color: var(--hero-ink); font-variant-numeric: tabular-nums; font-weight: 500; }
.people-hours-cell { display: flex; align-items: center; justify-content: flex-end; gap: 10px; }
.people-hours-bar { width: 80px; height: 5px; background: var(--bg-softer); border-radius: 3px; overflow: hidden; }
.people-hours-bar-fill { height: 100%; background: var(--hero-blue-500); border-radius: 3px; }

/* --- What-changed list --- */
.people-changes-list { display: flex; flex-direction: column; }
.people-changes-row {
  display: grid;
  grid-template-columns: 18px 1fr auto;
  gap: 14px;
  align-items: center;
  padding: 14px 4px;
  border-bottom: 1px solid var(--border);
  font-size: 14px;
}
.people-changes-row:last-child { border-bottom: none; }
.people-changes-dot { width: 8px; height: 8px; border-radius: 50%; margin: 0 auto; }
.change-good { background: var(--success); }
.change-warn { background: var(--warn); }
.people-changes-text { color: var(--ink-2); }
.people-changes-when { color: var(--ink-4); font-size: 12px; }

/* --- Metric segmented bar (Net value tile add-on) --- */
.metric-segbar {
  margin-top: 8px;
  height: 8px;
  display: flex;
  border-radius: 4px;
  overflow: hidden;
  background: rgba(0,0,0,0.04);
}
.metric-segbar span { display: block; height: 100%; }
.metric-segbar .seg-saved { background: var(--hero-blue-500); }
.metric-segbar .seg-spend { background: var(--ink-5); }
</style>`

// peopleScript is the per-page inline JS for the People home — currently
// only the SSE subscriber that re-fetches the named section fragment.
// The metric-tab toggler ships in the shell's tabbed-metric-strip
// fragment; the trend-chip toggle is local visual state handled inline.
const peopleScript = `<script>
(function () {
  // Trend-chip toggle (visual only — chart data does not swap yet).
  document.querySelectorAll('.people-toggle-chip').forEach(function (chip) {
    chip.addEventListener('click', function () {
      document.querySelectorAll('.people-toggle-chip').forEach(function (c) {
        c.classList.toggle('active', c === chip);
      });
    });
  });

  // SSE subscriber — re-fetches the named section fragment on each
  // event. The server publishes events.log change notifications which
  // we route into 'pulse' refreshes.
  if (typeof EventSource === 'undefined') return;
  var es;
  try {
    es = new EventSource('/api/people/events');
  } catch (err) { return; }
  function refresh(section) {
    fetch('/api/people/' + section, { headers: { 'Accept': 'text/html' } })
      .then(function (r) { return r.ok ? r.text() : null; })
      .then(function (html) {
        if (html == null) return;
        var target = document.getElementById('people-' + section);
        if (!target) return;
        target.outerHTML = html;
      })
      .catch(function () { /* swallow */ });
  }
  ['pulse'].forEach(function (s) {
    es.addEventListener(s, function () { refresh(s); });
  });
  window.addEventListener('beforeunload', function () { try { es.close(); } catch (e) {} });
})();
</script>`
