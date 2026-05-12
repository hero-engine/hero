// Hero Dashboard — Vanilla JS SPA
// No build step, no dependencies. Consumes the Hero REST API + SSE events.

'use strict';

// ─── State ──────────────────────────────────────────────────────────────────
const state = {
  project: null,       // current project slug
  projects: [],        // list of {slug, path}
  sse: null,           // EventSource instance
  activityLog: [],     // last N SSE events for the activity feed
  sortCol: null,       // bugs table sort column
  sortDir: 'asc',      // bugs table sort direction
};

const MAX_ACTIVITY = 50;

// ─── API Client ─────────────────────────────────────────────────────────────
const api = {
  async get(path) {
    const res = await fetch(path);
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    return res.json();
  },

  projects()          { return this.get('/api/projects'); },
  status(p)           { return this.get(`/api/${p}/status`); },
  specs(p, params='') { return this.get(`/api/${p}/specs${params ? '?' + params : ''}`); },
  spec(p, slug)       { return this.get(`/api/${p}/specs/${slug}`); },
  search(p, q, params='') {
    const qs = new URLSearchParams(params);
    qs.set('q', q);
    return this.get(`/api/${p}/search?${qs}`);
  },
  check(p)            { return this.get(`/api/${p}/check`); },
  knowledge(p, type='') {
    return this.get(`/api/${p}/knowledge${type ? '?type=' + type : ''}`);
  },
  inventory(p)        { return this.get(`/api/${p}/inventory`); },

  // Team API
  jobs(status='')     { return this.get(`/api/jobs${status ? '?status=' + status : ''}`); },
  teamStatus()        { return this.get('/api/team/status'); },
  teamUsage(since='') { return this.get(`/api/team/usage${since ? '?since=' + since : ''}`); },
  async postJSON(path, body) {
    const res = await fetch(path, { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body) });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    return res.json();
  },
  approveJob(id)      { return this.postJSON(`/api/jobs/${id}/approve`, {}); },
  rejectJob(id, reason='') { return this.postJSON(`/api/jobs/${id}/reject`, {reason}); },
  cancelJob(id)       { return this.postJSON(`/api/jobs/${id}/cancel`, {}); },
};

// ─── Router ─────────────────────────────────────────────────────────────────
function getRoute() {
  const hash = location.hash.replace(/^#/, '') || '/';
  const parts = hash.split('/').filter(Boolean);
  if (parts.length === 0) return { page: 'overview', params: {} };
  const page = parts[0];
  if (page === 'spec' && parts[1]) return { page: 'spec', params: { slug: parts.slice(1).join('/') } };
  return { page, params: {} };
}

function navigate(hash) {
  location.hash = hash;
}

window.addEventListener('hashchange', () => render());

// ─── Rendering ──────────────────────────────────────────────────────────────
const $content = () => document.getElementById('content');

function setActiveNav(page) {
  document.querySelectorAll('.nav-links a').forEach(a => {
    a.classList.toggle('active', a.dataset.page === page);
  });
}

async function render() {
  const { page, params } = getRoute();
  setActiveNav(page);

  if (!state.project) {
    $content().innerHTML = '<div class="loading">No project selected.</div>';
    return;
  }

  $content().innerHTML = '<div class="loading">Loading\u2026</div>';

  try {
    switch (page) {
      case 'overview': await renderOverview(); break;
      case 'board':    await renderBoard(); break;
      case 'bugs':     await renderBugs(); break;
      case 'knowledge':await renderKnowledge(); break;
      case 'search':   await renderSearch(); break;
      case 'spec':     await renderSpec(params.slug); break;
      case 'jobs':     await renderJobs(); break;
      case 'team':     await renderTeam(); break;
      case 'health':   await renderHealth(); break;
      default:         $content().innerHTML = '<div class="loading">Page not found.</div>';
    }
  } catch (err) {
    $content().innerHTML = `<div class="loading">Error: ${esc(err.message)}</div>`;
    console.error(err);
  }
}

// ─── Overview Page ──────────────────────────────────────────────────────────
async function renderOverview() {
  const [statusData, checkData] = await Promise.all([
    api.status(state.project),
    api.check(state.project),
  ]);

  const cards = [
    { label: 'Planning', value: statusData.planning, cls: 'planning' },
    { label: 'In Review', value: statusData.in_review, cls: 'in-review' },
    { label: 'Delivering', value: statusData.delivering, cls: 'delivering' },
    { label: 'Completed', value: statusData.completed, cls: 'completed' },
    { label: 'Knowledge', value: statusData.knowledge, cls: 'knowledge' },
  ];

  // Count bugs from specs list
  const bugCount = (statusData.specs || []).filter(s => s.type === 'bug').length;
  if (bugCount > 0) {
    cards.splice(4, 0, { label: 'Bugs', value: bugCount, cls: 'bugs' });
  }

  let warningsHTML = '';
  const staleCount = (checkData.stale || []).length;
  const unclaimedCount = (checkData.unclaimed || []).length;
  if (staleCount > 0 || unclaimedCount > 0) {
    warningsHTML = '<div class="warnings">';
    if (staleCount > 0) {
      warningsHTML += `<div class="warning-item"><span class="count">${staleCount}</span> stale spec${staleCount > 1 ? 's' : ''} (no updates in ${checkData.stale_days}+ days)</div>`;
    }
    if (unclaimedCount > 0) {
      warningsHTML += `<div class="warning-item"><span class="count">${unclaimedCount}</span> unclaimed spec${unclaimedCount > 1 ? 's' : ''} in delivering status</div>`;
    }
    warningsHTML += '</div>';
  }

  $content().innerHTML = `
    <div class="page-header">
      <h2>Overview</h2>
      <p>${esc(state.project)} \u2014 ${statusData.total} total specs</p>
    </div>
    <div class="stat-cards">
      ${cards.map(c => `
        <div class="stat-card ${c.cls}">
          <div class="label">${c.label}</div>
          <div class="value">${c.value}</div>
        </div>
      `).join('')}
    </div>
    ${warningsHTML}
    <div class="activity-feed">
      <h3>Recent Activity</h3>
      <div id="activity-list">${renderActivityList()}</div>
    </div>
  `;
}

function renderActivityList() {
  if (state.activityLog.length === 0) {
    return '<div class="activity-empty">No activity yet. Events will appear as specs change.</div>';
  }
  return '<ul class="activity-list">' + state.activityLog.map(ev => `
    <li class="activity-item">
      <span class="activity-type">${esc(ev.type)}</span>
      <span>${esc(ev.slug || ev.message || '')}</span>
      <span class="activity-time">${formatTime(ev.timestamp)}</span>
    </li>
  `).join('') + '</ul>';
}

// ─── Board Page ─────────────────────────────────────────────────────────────
async function renderBoard() {
  const data = await api.specs(state.project);
  const specs = data.specs || [];

  // Separate work specs from knowledge
  const work = specs.filter(s => !isKnowledgeType(s.type));

  const columns = {
    planning:   work.filter(s => s.status === 'planning'),
    'in-review': work.filter(s => s.status === 'in-review'),
    delivering: work.filter(s => s.status === 'delivering'),
    completed:  work.filter(s => s.status === 'completed'),
  };

  const colLabels = { planning: 'Planning', 'in-review': 'In Review', delivering: 'Delivering', completed: 'Completed' };

  $content().innerHTML = `
    <div class="page-header">
      <h2>Board</h2>
      <p>Work specs by status</p>
    </div>
    <div class="board">
      ${Object.entries(columns).map(([key, items]) => `
        <div class="board-column">
          <div class="board-column-header">
            <span>${colLabels[key]}</span>
            <span class="col-count">${items.length}</span>
          </div>
          <div class="board-cards">
            ${items.length === 0 ? '<div class="table-empty">None</div>' :
              items.map(s => `
                <div class="board-card" onclick="location.hash='#/spec/${esc(s.slug)}'">
                  <div class="card-title">${esc(s.title)}</div>
                  <div class="card-meta">
                    <span class="badge ${s.type}">${s.type}</span>
                    ${s.claimed_by ? `<span>${esc(s.claimed_by)}</span>` : ''}
                  </div>
                </div>
              `).join('')
            }
          </div>
        </div>
      `).join('')}
    </div>
  `;
}

// ─── Bugs Page ──────────────────────────────────────────────────────────────
async function renderBugs() {
  // Try inventory endpoint first, fall back to filtered specs
  let bugs = [];
  try {
    const inv = await api.inventory(state.project);
    bugs = inv.bugs || inv.specs || [];
  } catch {
    const data = await api.specs(state.project, 'type=bug');
    bugs = (data.specs || []).map(s => ({
      ...s,
      tracker_id: s.tracker_id || '',
      priority: s.priority || '',
      created: s.created || '',
    }));
  }

  $content().innerHTML = `
    <div class="page-header">
      <h2>Bug Inventory</h2>
      <p>${bugs.length} bug${bugs.length !== 1 ? 's' : ''} tracked</p>
    </div>
    <div class="filter-bar">
      <select id="bug-severity-filter" onchange="applyBugFilters()">
        <option value="">All severities</option>
        <option value="critical">Critical</option>
        <option value="high">High</option>
        <option value="medium">Medium</option>
        <option value="low">Low</option>
      </select>
      <select id="bug-status-filter" onchange="applyBugFilters()">
        <option value="">All statuses</option>
        <option value="planning">Planning</option>
        <option value="delivering">Delivering</option>
        <option value="completed">Completed</option>
      </select>
    </div>
    <div class="data-table-wrap">
      <table class="data-table" id="bugs-table">
        <thead>
          <tr>
            <th data-col="tracker_id" onclick="sortBugs('tracker_id')">ID</th>
            <th data-col="title" onclick="sortBugs('title')">Title</th>
            <th data-col="priority" onclick="sortBugs('priority')">Severity</th>
            <th data-col="status" onclick="sortBugs('status')">Status</th>
            <th data-col="claimed_by" onclick="sortBugs('claimed_by')">Assignee</th>
            <th data-col="created" onclick="sortBugs('created')">Created</th>
          </tr>
        </thead>
        <tbody id="bugs-tbody"></tbody>
      </table>
    </div>
  `;

  // Store bugs for filtering/sorting
  window._bugs = bugs;
  renderBugRows(bugs);
}

function renderBugRows(bugs) {
  const tbody = document.getElementById('bugs-tbody');
  if (!tbody) return;

  if (bugs.length === 0) {
    tbody.innerHTML = '<tr><td colspan="6" class="table-empty">No bugs found</td></tr>';
    return;
  }

  tbody.innerHTML = bugs.map(b => `
    <tr onclick="location.hash='#/spec/${esc(b.slug)}'">
      <td>${esc(b.tracker_id || b.slug)}</td>
      <td>${esc(b.title)}</td>
      <td><span class="severity-badge ${(b.priority || '').toLowerCase()}">${esc(b.priority || '\u2014')}</span></td>
      <td><span class="status-badge ${b.status}">${esc(b.status)}</span></td>
      <td>${esc(b.claimed_by || '\u2014')}</td>
      <td>${b.created ? formatDate(b.created) : '\u2014'}</td>
    </tr>
  `).join('');

  // Update sort indicators
  document.querySelectorAll('#bugs-table th').forEach(th => {
    th.classList.remove('sorted-asc', 'sorted-desc');
    if (th.dataset.col === state.sortCol) {
      th.classList.add(state.sortDir === 'asc' ? 'sorted-asc' : 'sorted-desc');
    }
  });
}

window.sortBugs = function(col) {
  if (state.sortCol === col) {
    state.sortDir = state.sortDir === 'asc' ? 'desc' : 'asc';
  } else {
    state.sortCol = col;
    state.sortDir = 'asc';
  }
  applyBugFilters();
};

window.applyBugFilters = function() {
  let bugs = window._bugs || [];
  const sevFilter = (document.getElementById('bug-severity-filter') || {}).value || '';
  const statusFilter = (document.getElementById('bug-status-filter') || {}).value || '';

  if (sevFilter) bugs = bugs.filter(b => (b.priority || '').toLowerCase() === sevFilter);
  if (statusFilter) bugs = bugs.filter(b => b.status === statusFilter);

  if (state.sortCol) {
    bugs = [...bugs].sort((a, b) => {
      const av = (a[state.sortCol] || '').toString().toLowerCase();
      const bv = (b[state.sortCol] || '').toString().toLowerCase();
      const cmp = av < bv ? -1 : av > bv ? 1 : 0;
      return state.sortDir === 'asc' ? cmp : -cmp;
    });
  }

  renderBugRows(bugs);
};

// ─── Spec Detail Page ───────────────────────────────────────────────────────
async function renderSpec(slug) {
  const detail = await api.spec(state.project, slug);

  // Parse frontmatter from content
  const fm = parseFrontmatter(detail.content || '');
  const body = stripFrontmatter(detail.content || '');

  const tags = detail.tags || fm.tags || [];
  const trackerId = fm.tracker_id || '';
  const priority = fm.priority || '';

  $content().innerHTML = `
    <div class="spec-detail">
      <div class="spec-body">
        <a href="#/board" class="back-link">\u2190 Back</a>
        <div class="md-content">${renderMarkdown(body)}</div>
      </div>
      <div class="spec-meta">
        <h3>Metadata</h3>
        <div class="meta-row"><div class="meta-label">Type</div><div class="meta-value"><span class="badge ${detail.type}">${detail.type}</span></div></div>
        <div class="meta-row"><div class="meta-label">Status</div><div class="meta-value"><span class="status-badge ${detail.status}">${detail.status}</span></div></div>
        ${detail.claimed_by ? `<div class="meta-row"><div class="meta-label">Claimed by</div><div class="meta-value">${esc(detail.claimed_by)}</div></div>` : ''}
        ${trackerId ? `<div class="meta-row"><div class="meta-label">Tracker ID</div><div class="meta-value">${esc(trackerId)}</div></div>` : ''}
        ${priority ? `<div class="meta-row"><div class="meta-label">Priority</div><div class="meta-value"><span class="severity-badge ${priority.toLowerCase()}">${esc(priority)}</span></div></div>` : ''}
        ${tags.length > 0 ? `<div class="meta-row"><div class="meta-label">Tags</div><div class="meta-value">${tags.map(t => `<span class="tag">${esc(t)}</span>`).join('')}</div></div>` : ''}
        <div class="meta-row"><div class="meta-label">Slug</div><div class="meta-value" style="font-family: var(--mono); font-size: 12px;">${esc(detail.slug)}</div></div>
      </div>
    </div>
  `;
}

// ─── Knowledge Page ─────────────────────────────────────────────────────────
async function renderKnowledge() {
  const data = await api.knowledge(state.project);
  const entries = data.entries || [];

  const groups = {};
  for (const e of entries) {
    const t = e.type || 'other';
    if (!groups[t]) groups[t] = [];
    groups[t].push(e);
  }

  const typeOrder = ['convention', 'decision', 'rule', 'context', 'note', 'external'];
  const sortedTypes = Object.keys(groups).sort((a, b) => {
    const ai = typeOrder.indexOf(a);
    const bi = typeOrder.indexOf(b);
    return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi);
  });

  $content().innerHTML = `
    <div class="page-header">
      <h2>Knowledge Base</h2>
      <p>${entries.length} entries across ${sortedTypes.length} categories</p>
    </div>
    <div class="knowledge-groups">
      ${sortedTypes.map(type => `
        <div class="knowledge-group">
          <h3>${esc(type)}s (${groups[type].length})</h3>
          <ul class="knowledge-list">
            ${groups[type].map(e => `
              <li class="knowledge-item" onclick="location.hash='#/spec/${esc(e.slug)}'">
                <span class="ki-title">${esc(e.title)}</span>
                <span class="ki-status"><span class="status-badge ${e.status}">${e.status}</span></span>
              </li>
            `).join('')}
          </ul>
        </div>
      `).join('')}
      ${sortedTypes.length === 0 ? '<div class="table-empty">No knowledge entries found.</div>' : ''}
    </div>
  `;
}

// ─── Search Page ────────────────────────────────────────────────────────────
async function renderSearch() {
  const existingQ = (window._lastSearchQuery || '');

  $content().innerHTML = `
    <div class="page-header">
      <h2>Search</h2>
      <p>Full-text search across specs and knowledge</p>
    </div>
    <div class="search-input-wrap">
      <input type="text" id="search-input" placeholder="Search specs, knowledge, decisions\u2026" value="${esc(existingQ)}" autofocus />
    </div>
    <div id="search-results"></div>
  `;

  const input = document.getElementById('search-input');
  let debounce = null;
  input.addEventListener('input', () => {
    clearTimeout(debounce);
    debounce = setTimeout(() => doSearch(input.value.trim()), 250);
  });

  // Re-run search if there was a previous query
  if (existingQ) doSearch(existingQ);
}

async function doSearch(q) {
  window._lastSearchQuery = q;
  const container = document.getElementById('search-results');
  if (!container) return;

  if (!q) {
    container.innerHTML = '';
    return;
  }

  try {
    const data = await api.search(state.project, q);
    const results = data.results || [];

    if (results.length === 0) {
      container.innerHTML = '<div class="table-empty">No results found.</div>';
      return;
    }

    container.innerHTML = `
      <p style="color: var(--text-muted); font-size: 13px; margin-bottom: 12px;">${results.length} result${results.length !== 1 ? 's' : ''}</p>
      <ul class="search-results">
        ${results.map(r => `
          <li class="search-result-item" onclick="location.hash='#/spec/${esc(r.slug)}'">
            <div class="sr-title">${esc(r.title)}</div>
            <div class="sr-meta">
              <span class="badge ${r.type}">${r.type}</span>
              <span class="status-badge ${r.status}">${r.status}</span>
              ${r.claimed_by ? `<span>\u2014 ${esc(r.claimed_by)}</span>` : ''}
            </div>
          </li>
        `).join('')}
      </ul>
    `;
  } catch (err) {
    container.innerHTML = `<div class="table-empty">Search error: ${esc(err.message)}</div>`;
  }
}

// ─── SSE ────────────────────────────────────────────────────────────────────
function connectSSE() {
  if (state.sse) {
    state.sse.close();
    state.sse = null;
  }

  const url = state.project
    ? `/api/events?project=${encodeURIComponent(state.project)}`
    : '/api/events';

  const es = new EventSource(url);
  state.sse = es;

  es.addEventListener('connected', () => {
    setSSEStatus(true);
  });

  const eventTypes = ['spec.created', 'spec.modified', 'spec.deleted', 'index.rebuilt', 'health.check'];
  for (const type of eventTypes) {
    es.addEventListener(type, (e) => {
      let data;
      try { data = JSON.parse(e.data); } catch { data = {}; }
      data.type = type;
      onSSEEvent(data);
    });
  }

  // Also handle generic message events
  es.onmessage = (e) => {
    let data;
    try { data = JSON.parse(e.data); } catch { data = {}; }
    if (data.type) onSSEEvent(data);
  };

  es.onerror = () => {
    setSSEStatus(false);
    // EventSource auto-reconnects
  };

  es.onopen = () => {
    setSSEStatus(true);
  };
}

function onSSEEvent(data) {
  // Add to activity log
  state.activityLog.unshift(data);
  if (state.activityLog.length > MAX_ACTIVITY) state.activityLog.pop();

  // Update last-update timestamp
  const el = document.getElementById('last-update');
  if (el) el.textContent = formatTime(data.timestamp || new Date().toISOString());

  // Live-update the current page if applicable
  const { page } = getRoute();
  if (page === 'overview') {
    const actList = document.getElementById('activity-list');
    if (actList) actList.innerHTML = renderActivityList();
    // Refresh overview stats on index.rebuilt
    if (data.type === 'index.rebuilt') renderOverview();
  } else if (page === 'board' && (data.type === 'spec.created' || data.type === 'spec.modified' || data.type === 'spec.deleted')) {
    renderBoard();
  } else if (page === 'bugs' && data.type === 'index.rebuilt') {
    renderBugs();
  }
}

function setSSEStatus(connected) {
  const dot = document.getElementById('sse-status');
  if (!dot) return;
  dot.classList.toggle('connected', connected);
  dot.classList.toggle('disconnected', !connected);
  dot.title = connected ? 'SSE connected' : 'SSE disconnected';
}

// ─── Markdown Renderer (lightweight) ────────────────────────────────────────
// Minimal markdown-to-HTML. Handles headers, bold, italic, code, links, lists,
// blockquotes, tables, horizontal rules. No external dependency.
function renderMarkdown(md) {
  if (!md) return '';

  // Normalize line endings
  md = md.replace(/\r\n/g, '\n');

  const lines = md.split('\n');
  const out = [];
  let inCode = false;
  let codeLang = '';
  let codeLines = [];
  let inList = false;
  let listType = 'ul';
  let inTable = false;
  let tableRows = [];

  function flushList() {
    if (inList) { out.push(`</${listType}>`); inList = false; }
  }
  function flushTable() {
    if (!inTable) return;
    if (tableRows.length > 0) {
      let html = '<table>';
      tableRows.forEach((row, i) => {
        const tag = i === 0 ? 'th' : 'td';
        const wrap = i === 0 ? 'thead' : (i === 1 ? 'tbody' : '');
        if (wrap === 'thead') html += '<thead>';
        if (wrap === 'tbody') html += '</thead><tbody>';
        html += '<tr>' + row.map(c => `<${tag}>${inlineMarkdown(c.trim())}</${tag}>`).join('') + '</tr>';
      });
      html += '</tbody></table>';
      out.push(html);
    }
    inTable = false;
    tableRows = [];
  }

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Fenced code blocks
    if (line.startsWith('```')) {
      if (inCode) {
        out.push(`<pre><code>${escHtml(codeLines.join('\n'))}</code></pre>`);
        inCode = false;
        codeLines = [];
        continue;
      }
      flushList();
      flushTable();
      inCode = true;
      codeLang = line.slice(3).trim();
      codeLines = [];
      continue;
    }
    if (inCode) {
      codeLines.push(line);
      continue;
    }

    // Table detection
    if (line.includes('|') && line.trim().startsWith('|')) {
      const cells = line.split('|').slice(1, -1);
      // Skip separator rows (|---|---|)
      if (cells.every(c => /^[\s:-]+$/.test(c))) continue;
      if (!inTable) {
        flushList();
        inTable = true;
        tableRows = [];
      }
      tableRows.push(cells);
      continue;
    }
    flushTable();

    // Blank line
    if (line.trim() === '') {
      flushList();
      continue;
    }

    // Headings
    const hm = line.match(/^(#{1,6})\s+(.+)/);
    if (hm) {
      flushList();
      const level = hm[1].length;
      out.push(`<h${level}>${inlineMarkdown(hm[2])}</h${level}>`);
      continue;
    }

    // Horizontal rule
    if (/^(-{3,}|_{3,}|\*{3,})$/.test(line.trim())) {
      flushList();
      out.push('<hr>');
      continue;
    }

    // Blockquote
    if (line.startsWith('>')) {
      flushList();
      out.push(`<blockquote><p>${inlineMarkdown(line.replace(/^>\s?/, ''))}</p></blockquote>`);
      continue;
    }

    // Unordered list
    const ulm = line.match(/^(\s*)[-*+]\s+(.+)/);
    if (ulm) {
      if (!inList || listType !== 'ul') {
        flushList();
        out.push('<ul>');
        inList = true;
        listType = 'ul';
      }
      out.push(`<li>${inlineMarkdown(ulm[2])}</li>`);
      continue;
    }

    // Ordered list
    const olm = line.match(/^(\s*)\d+\.\s+(.+)/);
    if (olm) {
      if (!inList || listType !== 'ol') {
        flushList();
        out.push('<ol>');
        inList = true;
        listType = 'ol';
      }
      out.push(`<li>${inlineMarkdown(olm[2])}</li>`);
      continue;
    }

    // Paragraph
    flushList();
    out.push(`<p>${inlineMarkdown(line)}</p>`);
  }

  // Flush remaining
  if (inCode) out.push(`<pre><code>${escHtml(codeLines.join('\n'))}</code></pre>`);
  flushList();
  flushTable();

  return out.join('\n');
}

function inlineMarkdown(text) {
  return text
    // Images
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" style="max-width:100%">')
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
    // Bold+italic
    .replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>')
    // Bold
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/__(.+?)__/g, '<strong>$1</strong>')
    // Italic
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    .replace(/_(.+?)_/g, '<em>$1</em>')
    // Strikethrough
    .replace(/~~(.+?)~~/g, '<del>$1</del>')
    // Inline code
    .replace(/`([^`]+)`/g, '<code>$1</code>');
}

// ─── Frontmatter Parsing ────────────────────────────────────────────────────
function parseFrontmatter(content) {
  const m = content.match(/^---\n([\s\S]*?)\n---/);
  if (!m) return {};
  const fm = {};
  for (const line of m[1].split('\n')) {
    const kv = line.match(/^(\w[\w_-]*):\s*(.+)/);
    if (kv) {
      let val = kv[2].trim();
      // Parse array: [item1, item2]
      if (val.startsWith('[') && val.endsWith(']')) {
        val = val.slice(1, -1).split(',').map(s => s.trim().replace(/^["']|["']$/g, ''));
      }
      fm[kv[1]] = val;
    }
  }
  return fm;
}

function stripFrontmatter(content) {
  return content.replace(/^---\n[\s\S]*?\n---\n*/, '');
}

// ─── Helpers ────────────────────────────────────────────────────────────────
function esc(str) { return escHtml(String(str || '')); }
function escHtml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function isKnowledgeType(type) {
  return ['convention', 'decision', 'rule', 'context', 'note', 'external'].includes(type);
}

function formatTime(ts) {
  if (!ts) return '';
  const d = new Date(ts);
  if (isNaN(d.getTime())) return ts;
  const now = new Date();
  const diff = (now - d) / 1000;
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return d.toLocaleDateString();
}

function formatDate(ts) {
  if (!ts) return '';
  const d = new Date(ts);
  if (isNaN(d.getTime())) return ts;
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

// ─── Init ───────────────────────────────────────────────────────────────────
async function init() {
  try {
    const data = await api.projects();
    state.projects = data.projects || [];
  } catch (err) {
    $content().innerHTML = `<div class="loading">Failed to connect to Hero daemon: ${esc(err.message)}</div>`;
    return;
  }

  // Populate project selector
  const sel = document.getElementById('project-select');
  sel.innerHTML = state.projects.map(p =>
    `<option value="${esc(p.slug)}">${esc(p.slug)}</option>`
  ).join('');

  if (state.projects.length > 0) {
    state.project = state.projects[0].slug;
    sel.value = state.project;
  }

  sel.addEventListener('change', () => {
    state.project = sel.value;
    state.activityLog = [];
    connectSSE();
    render();
  });

  connectSSE();
  render();
}

// ─── Jobs Page ─────────────────────────────────────────────────────────────
async function renderJobs() {
  let jobs = [];
  try { jobs = await api.jobs(); } catch(e) { /* team server may not be running */ }

  const statusOrder = { running: 0, queued: 1, awaiting_approval: 2, completed: 3, failed: 4, cancelled: 5 };
  jobs.sort((a, b) => (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9));

  let html = '<h2>Jobs</h2>';

  if (jobs.length === 0) {
    html += '<p class="text-muted">No jobs yet. Submit one with <code>hero jobs submit deliver &lt;slug&gt;</code></p>';
    $content().innerHTML = html;
    return;
  }

  html += '<div class="jobs-grid">';
  for (const j of jobs) {
    const dur = j.completed_at && j.submitted_at
      ? formatDuration(new Date(j.completed_at) - new Date(j.submitted_at))
      : '';
    const cost = j.est_cost > 0 ? `$${j.est_cost.toFixed(2)}` : '';

    let actions = '';
    if (j.status === 'awaiting_approval') {
      actions = `<button class="btn btn-primary" onclick="approveJob('${esc(j.id)}')">Approve</button>
                 <button class="btn btn-danger" onclick="rejectJob('${esc(j.id)}')">Reject</button>`;
    } else if (j.status === 'queued' || j.status === 'running') {
      actions = `<button class="btn btn-danger" onclick="cancelJob('${esc(j.id)}')">Cancel</button>`;
    }

    html += `<div class="job-card">
      <div class="job-info">
        <div class="job-id">${esc(j.id)}</div>
        <div class="job-command">${esc(j.command)} ${esc(j.args || '')}</div>
        <div class="job-meta">
          <span class="job-status ${esc(j.status)}">${esc(j.status)}</span>
          ${j.submitted_by ? ` · ${esc(j.submitted_by)}` : ''}
          ${j.turns ? ` · ${j.turns} turns` : ''}
          ${cost ? ` · ${cost}` : ''}
          ${dur ? ` · ${dur}` : ''}
        </div>
        ${j.error ? `<div class="job-meta" style="color:var(--red);margin-top:4px">${esc(j.error)}</div>` : ''}
      </div>
      <div class="job-actions">${actions}</div>
    </div>`;
  }
  html += '</div>';
  $content().innerHTML = html;
}

async function approveJob(id) {
  try { await api.approveJob(id); renderJobs(); } catch(e) { alert('Approve failed: ' + e.message); }
}
async function rejectJob(id) {
  const reason = prompt('Rejection reason (optional):');
  try { await api.rejectJob(id, reason || ''); renderJobs(); } catch(e) { alert('Reject failed: ' + e.message); }
}
async function cancelJob(id) {
  try { await api.cancelJob(id); renderJobs(); } catch(e) { alert('Cancel failed: ' + e.message); }
}

// ─── Team Page ─────────────────────────────────────────────────────────────
async function renderTeam() {
  let teamData = { sessions: [], running_jobs: [], queued_jobs: [], awaiting_approval: [] };
  let usage = [];
  try {
    [teamData, usage] = await Promise.all([api.teamStatus(), api.teamUsage()]);
  } catch(e) { /* team server may not be running */ }

  let html = '<h2>Team</h2>';

  // Summary cards
  html += '<div class="team-grid">';

  // Active sessions
  html += '<div class="team-section"><h3>Active Sessions</h3>';
  if (!teamData.sessions || teamData.sessions.length === 0) {
    html += '<p class="text-muted">No active sessions</p>';
  } else {
    for (const s of teamData.sessions) {
      html += `<div class="session-card">
        <div class="session-user">${esc(s.user_id || 'unknown')}</div>
        <div class="session-detail">${esc(s.agent || '')} · ${esc(s.command || '')} ${esc(s.spec_slug || '')}</div>
      </div>`;
    }
  }
  html += '</div>';

  // Queue status
  html += '<div class="team-section"><h3>Queue</h3>';
  const running = teamData.running_jobs?.length || 0;
  const queued = teamData.queued_jobs?.length || 0;
  const awaiting = teamData.awaiting_approval?.length || 0;
  html += `<div class="session-card">
    <div class="session-user">${running} running · ${queued} queued · ${awaiting} awaiting approval</div>
  </div>`;
  html += '</div>';
  html += '</div>';

  // Usage table
  if (usage && usage.length > 0) {
    html += '<h3 style="margin-bottom:12px">Usage (7 days)</h3>';
    html += '<table class="usage-table"><thead><tr>';
    html += '<th>User</th><th>Jobs</th><th>Input tokens</th><th>Output tokens</th><th>Cost</th>';
    html += '</tr></thead><tbody>';
    for (const u of usage) {
      html += `<tr>
        <td>${esc(String(u.user_id || ''))}</td>
        <td>${u.jobs || 0}</td>
        <td>${(u.input_tokens || 0).toLocaleString()}</td>
        <td>${(u.output_tokens || 0).toLocaleString()}</td>
        <td>$${(u.total_cost || 0).toFixed(2)}</td>
      </tr>`;
    }
    html += '</tbody></table>';
  }

  $content().innerHTML = html;
}

// ─── Health Page ───────────────────────────────────────────────────────────
async function renderHealth() {
  const p = state.project;
  let statusData = {}, checkData = {};
  try {
    [statusData, checkData] = await Promise.all([api.status(p), api.check(p)]);
  } catch(e) {}

  const counts = statusData.by_status || {};
  const total = Object.values(counts).reduce((a, b) => a + b, 0);
  const delivering = counts.delivering || 0;
  const planning = counts.planning || 0;
  const completed = counts.completed || 0;

  let html = '<h2>Project Health</h2>';

  // Health cards
  html += '<div class="health-grid">';
  html += healthCard('Completed', completed, 'good');
  html += healthCard('In Flight', delivering + planning, delivering + planning > 10 ? 'warn' : 'good');
  html += healthCard('Stale', checkData.stale?.length || 0, (checkData.stale?.length || 0) > 0 ? 'warn' : 'good');
  html += healthCard('Issues', checkData.issue_count || 0, (checkData.issue_count || 0) > 0 ? 'bad' : 'good');
  html += '</div>';

  // Stale specs
  if (checkData.stale && checkData.stale.length > 0) {
    html += '<h3>Stale Specs</h3><div class="drift-list">';
    for (const s of checkData.stale) {
      html += `<div class="drift-item">${esc(s.slug)} — ${esc(s.title)} (${esc(s.status)})</div>`;
    }
    html += '</div>';
  }

  // Unclaimed
  if (checkData.unclaimed && checkData.unclaimed.length > 0) {
    html += '<h3 style="margin-top:16px">Unclaimed</h3><div class="suggestion-list">';
    for (const s of checkData.unclaimed) {
      html += `<div class="suggestion-item">${esc(s.slug)} — ${esc(s.title)}</div>`;
    }
    html += '</div>';
  }

  // Status drift
  if (checkData.drift && checkData.drift.length > 0) {
    html += '<h3 style="margin-top:16px">Status Drift</h3><div class="drift-list">';
    for (const d of checkData.drift) {
      html += `<div class="drift-item">${esc(d.slug)} — ${esc(d.current)} → ${esc(d.suggested)} (${esc(d.evidence || '')})</div>`;
    }
    html += '</div>';
  }

  $content().innerHTML = html;
}

function healthCard(label, value, status) {
  return `<div class="health-card ${status}">
    <div class="health-label">${esc(label)}</div>
    <div class="health-value">${value}</div>
  </div>`;
}

function formatDuration(ms) {
  if (ms < 1000) return '<1s';
  const s = Math.floor(ms / 1000);
  if (s < 60) return s + 's';
  const m = Math.floor(s / 60);
  if (m < 60) return m + 'm ' + (s % 60) + 's';
  const h = Math.floor(m / 60);
  return h + 'h ' + (m % 60) + 'm';
}

init();
