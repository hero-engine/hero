// ============================================================================
// Hero Cloud Dashboard — Preact + HTM Single-File SPA
// ============================================================================

import { h, render } from './lib/preact.module.js';
import { useState, useEffect, useCallback, useRef, useMemo } from './lib/hooks.module.js';
import { html } from './lib/htm-preact.module.js';
import { marked } from './lib/marked.min.js';

// ============================================================================
// API Client
// ============================================================================

const api = {
  getOrg() {
    return localStorage.getItem('hero-org');
  },

  setOrg(id) {
    localStorage.setItem('hero-org', id);
  },

  orgPath(path) {
    return `/api/v1/orgs/${this.getOrg()}${path}`;
  },

  async _request(method, path, body) {
    const token = localStorage.getItem('hero-token');
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const opts = { method, headers };
    if (body) opts.body = JSON.stringify(body);

    const res = await fetch(`/api/v1${path}`, opts);

    if (res.status === 401) {
      localStorage.removeItem('hero-token');
      window.location.hash = '#/login';
      throw new Error('Unauthorized');
    }

    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: res.statusText }));
      throw new Error(err.error || res.statusText);
    }

    return res.json();
  },

  get(path) { return this._request('GET', path); },
  post(path, body) { return this._request('POST', path, body); },
  del(path) { return this._request('DELETE', path); },
};

// ============================================================================
// Router
// ============================================================================

function parseHash(hash) {
  const raw = (hash || '').replace(/^#\/?/, '');
  if (!raw || raw === '/') return { page: 'overview', params: {} };

  const [first, ...rest] = raw.split('/');
  const params = {};

  if (first === 'auth' && rest[0] === 'callback') {
    const qs = raw.includes('?') ? raw.split('?')[1] : '';
    for (const pair of qs.split('&')) {
      const [k, v] = pair.split('=');
      if (k) params[decodeURIComponent(k)] = decodeURIComponent(v || '');
    }
    return { page: 'auth', params };
  }

  if (rest.length > 0) params.id = rest.join('/');
  return { page: first, params };
}

function useRouter() {
  const [route, setRoute] = useState(() => parseHash(location.hash));

  useEffect(() => {
    const handler = () => setRoute(parseHash(location.hash));
    window.addEventListener('hashchange', handler);
    return () => window.removeEventListener('hashchange', handler);
  }, []);

  return route;
}

// ============================================================================
// Auth
// ============================================================================

function decodeJwtPayload(token) {
  try {
    const base64 = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
    return JSON.parse(atob(base64));
  } catch {
    return null;
  }
}

function useAuth() {
  const [token, setToken] = useState(() => localStorage.getItem('hero-token'));

  const user = token ? decodeJwtPayload(token) : null;

  const logout = useCallback(() => {
    localStorage.removeItem('hero-token');
    localStorage.removeItem('hero-refresh');
    setToken(null);
    window.location.hash = '#/login';
  }, []);

  // Listen for storage changes (e.g. from auth callback)
  useEffect(() => {
    const handler = () => setToken(localStorage.getItem('hero-token'));
    window.addEventListener('storage', handler);
    return () => window.removeEventListener('storage', handler);
  }, []);

  return { token, user, logout, setToken };
}

function AuthCallback() {
  useEffect(() => {
    const hash = location.hash;
    const qs = hash.includes('?') ? hash.split('?')[1] : '';
    const params = new URLSearchParams(qs);
    const token = params.get('token');
    const refresh = params.get('refresh');

    if (token) localStorage.setItem('hero-token', token);
    if (refresh) localStorage.setItem('hero-refresh', refresh);

    window.location.hash = '#/';
    // Force a reload so App re-reads token
    window.location.reload();
  }, []);

  return html`<div class="page-loading">Signing in...</div>`;
}

function LoginPage() {
  return html`
    <div class="login-page">
      <div class="login-card">
        <svg class="login-logo" viewBox="0 0 90 90" width="64" height="64">
          <path d="M52 8 L22 46 L40 46 L34 82 L68 42 L50 42 Z" fill="var(--accent)" />
        </svg>
        <h1 class="login-title">Hero Cloud</h1>
        <p class="login-subtitle">Spec-driven engineering dashboard</p>
        <a class="btn btn-primary login-btn" href="/api/v1/auth/github">
          <svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor">
            <path d="M10 0C4.48 0 0 4.48 0 10c0 4.42 2.87 8.17 6.84 9.49.5.09.68-.22.68-.48v-1.7c-2.78.6-3.37-1.34-3.37-1.34-.46-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.61.07-.61 1 .07 1.53 1.03 1.53 1.03.89 1.53 2.34 1.09 2.91.83.09-.65.35-1.09.63-1.34-2.22-.25-4.56-1.11-4.56-4.94 0-1.09.39-1.98 1.03-2.68-.1-.25-.45-1.27.1-2.64 0 0 .84-.27 2.75 1.02A9.56 9.56 0 0110 4.84c.85.004 1.71.11 2.51.33 1.91-1.29 2.75-1.02 2.75-1.02.55 1.37.2 2.39.1 2.64.64.7 1.03 1.59 1.03 2.68 0 3.84-2.34 4.68-4.57 4.93.36.31.68.92.68 1.85v2.75c0 .27.18.58.69.48A10.01 10.01 0 0020 10c0-5.52-4.48-10-10-10z"/>
          </svg>
          Sign in with GitHub
        </a>
      </div>
    </div>
  `;
}

// ============================================================================
// SSE (Server-Sent Events) Hook
// ============================================================================

function useSSE(onEvent) {
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    const token = localStorage.getItem('hero-token');
    const orgID = api.getOrg();
    if (!token || !orgID) return;

    const url = `/api/v1/orgs/${orgID}/events`;
    let es;
    let reconnectTimer;

    function connect() {
      // EventSource doesn't support auth headers, so we use fetch-based SSE
      const ctrl = new AbortController();
      fetch(url, {
        headers: { 'Authorization': `Bearer ${token}` },
        signal: ctrl.signal,
      }).then(res => {
        if (!res.ok || !res.body) return;
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        function read() {
          reader.read().then(({ done, value }) => {
            if (done) {
              // Reconnect after disconnect
              reconnectTimer = setTimeout(connect, 3000);
              return;
            }
            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split('\n');
            buffer = lines.pop() || '';
            for (const line of lines) {
              if (line.startsWith('data: ')) {
                try {
                  const event = JSON.parse(line.slice(6));
                  onEventRef.current(event);
                } catch {}
              }
            }
            read();
          }).catch(() => {
            reconnectTimer = setTimeout(connect, 5000);
          });
        }
        read();
      }).catch(() => {
        reconnectTimer = setTimeout(connect, 5000);
      });

      return ctrl;
    }

    const ctrl = connect();
    return () => {
      if (ctrl && ctrl.abort) ctrl.abort();
      clearTimeout(reconnectTimer);
    };
  }, []);
}

// ============================================================================
// Theme Toggle
// ============================================================================

function ThemeToggle() {
  const [theme, setTheme] = useState(() => localStorage.getItem('hero-theme') || '');

  const toggle = useCallback(() => {
    const next = theme === 'light' ? '' : 'light';
    setTheme(next);
    document.documentElement.dataset.theme = next;
    localStorage.setItem('hero-theme', next);
  }, [theme]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, []);

  const isLight = theme === 'light';

  return html`
    <button class="icon-btn theme-toggle" onClick=${toggle} title="Toggle theme">
      ${isLight
        ? html`<svg viewBox="0 0 20 20" width="18" height="18" fill="currentColor"><circle cx="10" cy="10" r="4"/><path d="M10 1v2m0 14v2m-7-9h2m14 0h2m-4.95-6.36l-1.41 1.41m-7.07 7.07l-1.41 1.41m0-9.9l1.41 1.41m7.07 7.07l1.41 1.41" stroke="currentColor" stroke-width="1.5" fill="none"/></svg>`
        : html`<svg viewBox="0 0 20 20" width="18" height="18" fill="currentColor"><path d="M17.29 13.29A8 8 0 116.71 2.71a7 7 0 0010.58 10.58z"/></svg>`
      }
    </button>
  `;
}

// ============================================================================
// Shared Components
// ============================================================================

function StatCard({ title, value, delta, icon }) {
  const deltaClass = delta > 0 ? 'positive' : delta < 0 ? 'negative' : '';
  const deltaStr = delta > 0 ? `+${delta}%` : delta < 0 ? `${delta}%` : null;

  return html`
    <div class="stat-card">
      <div class="stat-card-header">
        <span class="stat-card-title">${title}</span>
        ${icon && html`<span class="stat-card-icon">${icon}</span>`}
      </div>
      <div class="stat-card-value">${value}</div>
      ${deltaStr && html`<div class="stat-card-delta ${deltaClass}">${deltaStr}</div>`}
    </div>
  `;
}

function Badge({ type, label }) {
  return html`<span class="badge badge-${type}">${label || type}</span>`;
}

function DataTable({ columns, data, onRowClick, sortable }) {
  const [sortKey, setSortKey] = useState(null);
  const [sortDir, setSortDir] = useState('asc');

  const handleSort = useCallback((key) => {
    if (!sortable) return;
    if (sortKey === key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
  }, [sortable, sortKey]);

  const sorted = sortKey
    ? [...data].sort((a, b) => {
        const av = a[sortKey] ?? '';
        const bv = b[sortKey] ?? '';
        const cmp = typeof av === 'string' ? av.localeCompare(bv) : av - bv;
        return sortDir === 'asc' ? cmp : -cmp;
      })
    : data;

  return html`
    <div class="data-table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            ${columns.map(col => html`
              <th
                class=${sortable && col.sortable !== false ? 'sortable' : ''}
                onClick=${() => col.sortable !== false && handleSort(col.key)}
              >
                ${col.label}
                ${sortKey === col.key ? (sortDir === 'asc' ? ' ↑' : ' ↓') : ''}
              </th>
            `)}
          </tr>
        </thead>
        <tbody>
          ${sorted.map(row => html`
            <tr
              class=${onRowClick ? 'clickable' : ''}
              onClick=${() => onRowClick && onRowClick(row)}
            >
              ${columns.map(col => html`
                <td>${col.render ? col.render(row[col.key], row) : row[col.key]}</td>
              `)}
            </tr>
          `)}
        </tbody>
      </table>
    </div>
  `;
}

function SlideOver({ open, onClose, title, children }) {
  useEffect(() => {
    if (!open) return;
    const handler = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, onClose]);

  if (!open) return null;

  return html`
    <div class="slideover-overlay" onClick=${onClose}>
      <div class="slideover-panel open" onClick=${(e) => e.stopPropagation()}>
        <div class="slideover-header">
          <h2 class="slideover-title">${title}</h2>
          <button class="icon-btn" onClick=${onClose}>
            <svg viewBox="0 0 20 20" width="18" height="18" fill="currentColor">
              <path d="M6 6l8 8m0-8l-8 8" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
        <div class="slideover-body">
          ${children}
        </div>
      </div>
    </div>
  `;
}

function FilterBar({ filters, activeFilters, onFilterChange, searchValue, onSearchChange }) {
  return html`
    <div class="filter-bar">
      ${onSearchChange && html`
        <input
          class="filter-search"
          type="text"
          placeholder="Search..."
          value=${searchValue || ''}
          onInput=${(e) => onSearchChange(e.target.value)}
        />
      `}
      <div class="filter-chips">
        ${(filters || []).map(f => html`
          <select
            class="filter-select"
            value=${(activeFilters || {})[f.key] || ''}
            onChange=${(e) => onFilterChange({ ...activeFilters, [f.key]: e.target.value })}
          >
            <option value="">${f.label}</option>
            ${f.options.map(o => html`<option value=${o.value}>${o.label}</option>`)}
          </select>
        `)}
      </div>
    </div>
  `;
}

function Toast({ message, type, onDismiss }) {
  useEffect(() => {
    const t = setTimeout(onDismiss, 5000);
    return () => clearTimeout(t);
  }, [onDismiss]);

  return html`
    <div class="toast toast-${type || 'info'}" onClick=${onDismiss}>
      ${message}
    </div>
  `;
}

function EmptyState({ icon, title, description, action }) {
  return html`
    <div class="empty-state">
      ${icon && html`<div class="empty-state-icon">${icon}</div>`}
      <h3 class="empty-state-title">${title}</h3>
      ${description && html`<p class="empty-state-desc">${description}</p>`}
      ${action && html`<div class="empty-state-action">${action}</div>`}
    </div>
  `;
}

function LoadingSpinner({ message }) {
  return html`
    <div class="page-loading">
      <div class="spinner"></div>
      <span>${message || 'Loading...'}</span>
    </div>
  `;
}

function ErrorBanner({ message, onRetry }) {
  return html`
    <div class="error-banner">
      <svg viewBox="0 0 20 20" width="18" height="18" fill="var(--danger)">
        <path d="M10 0C4.48 0 0 4.48 0 10s4.48 10 10 10 10-4.48 10-10S15.52 0 10 0zm1 15H9v-2h2v2zm0-4H9V5h2v6z"/>
      </svg>
      <span>${message || 'Something went wrong.'}</span>
      ${onRetry && html`<button class="btn btn-secondary btn-sm" onClick=${onRetry}>Retry</button>`}
    </div>
  `;
}

function CommandPalette({ open, onClose }) {
  const [query, setQuery] = useState('');
  const [selected, setSelected] = useState(0);
  const [searchResults, setSearchResults] = useState([]);
  const inputRef = useRef(null);

  const pages = [
    { label: 'Overview', action: () => { window.location.hash = '#/'; } },
    { label: 'Specs', action: () => { window.location.hash = '#/specs'; } },
    { label: 'Activity', action: () => { window.location.hash = '#/activity'; } },
    { label: 'Compliance', action: () => { window.location.hash = '#/compliance'; } },
    { label: 'Audit', action: () => { window.location.hash = '#/audit'; } },
    { label: 'Analytics', action: () => { window.location.hash = '#/analytics'; } },
    { label: 'Knowledge', action: () => { window.location.hash = '#/knowledge'; } },
  ];

  const actions = [
    { label: 'Toggle Theme', action: () => { document.querySelector('.theme-toggle')?.click(); } },
    { label: 'Logout', action: () => { localStorage.removeItem('hero-token'); window.location.hash = '#/login'; window.location.reload(); } },
  ];

  // Search specs/knowledge when query is 3+ chars
  useEffect(() => {
    if (!open || query.length < 3 || !api.getOrg()) { setSearchResults([]); return; }
    const t = setTimeout(() => {
      api.get(`/orgs/${api.getOrg()}/knowledge/search?q=${encodeURIComponent(query)}`)
        .then(data => {
          const results = [];
          (data.specs || []).forEach(s => results.push({
            label: `Spec: ${s.title || s.slug}`,
            action: () => { window.location.hash = `#/specs/${s.slug}`; },
          }));
          (data.conventions || []).forEach(c => results.push({
            label: `Convention: ${c.title || c.slug}`,
            action: () => { window.location.hash = `#/knowledge`; },
          }));
          setSearchResults(results);
        })
        .catch(() => setSearchResults([]));
    }, 200);
    return () => clearTimeout(t);
  }, [query, open]);

  const lq = query.toLowerCase();
  const filteredPages = pages.filter(p => p.label.toLowerCase().includes(lq));
  const filteredActions = actions.filter(a => a.label.toLowerCase().includes(lq));
  const allItems = [...filteredPages, ...searchResults, ...filteredActions];

  useEffect(() => {
    if (open && inputRef.current) {
      inputRef.current.focus();
      setQuery('');
      setSelected(0);
    }
  }, [open]);

  const handleKeyDown = useCallback((e) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelected(s => Math.min(s + 1, allItems.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelected(s => Math.max(s - 1, 0));
    } else if (e.key === 'Enter' && allItems[selected]) {
      allItems[selected].action();
      onClose();
    } else if (e.key === 'Escape') {
      onClose();
    }
  }, [allItems, selected, onClose]);

  if (!open) return null;

  return html`
    <div class="command-overlay" onClick=${onClose}>
      <div class="command-palette" onClick=${(e) => e.stopPropagation()}>
        <input
          ref=${inputRef}
          class="command-input"
          type="text"
          placeholder="Type a command..."
          value=${query}
          onInput=${(e) => { setQuery(e.target.value); setSelected(0); }}
          onKeyDown=${handleKeyDown}
        />
        <div class="command-results">
          ${filteredPages.length > 0 && html`
            <div class="command-group-label">Pages</div>
            ${filteredPages.map((item, i) => html`
              <div
                class="command-item ${selected === i ? 'selected' : ''}"
                onClick=${() => { item.action(); onClose(); }}
                onMouseEnter=${() => setSelected(i)}
              >${item.label}</div>
            `)}
          `}
          ${searchResults.length > 0 && html`
            <div class="command-group-label">Search Results</div>
            ${searchResults.map((item, i) => {
              const idx = filteredPages.length + i;
              return html`
                <div
                  class="command-item ${selected === idx ? 'selected' : ''}"
                  onClick=${() => { item.action(); onClose(); }}
                  onMouseEnter=${() => setSelected(idx)}
                >${item.label}</div>
              `;
            })}
          `}
          ${filteredActions.length > 0 && html`
            <div class="command-group-label">Actions</div>
            ${filteredActions.map((item, i) => {
              const idx = filteredPages.length + searchResults.length + i;
              return html`
                <div
                  class="command-item ${selected === idx ? 'selected' : ''}"
                  onClick=${() => { item.action(); onClose(); }}
                  onMouseEnter=${() => setSelected(idx)}
                >${item.label}</div>
              `;
            })}
          `}
          ${allItems.length === 0 && html`<div class="command-empty">No results</div>`}
        </div>
      </div>
    </div>
  `;
}

// ============================================================================
// Utilities
// ============================================================================

function relativeTime(iso) {
  if (!iso) return '';
  const diff = Date.now() - new Date(iso).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

const EVENT_VERBS = {
  'sync': 'synced specs from',
  'spec.created': 'created spec',
  'spec.synced': 'synced spec',
  'spec.delivered': 'delivered spec',
  'pr.checked': 'checked PR',
  'convention.matched': 'convention matched in',
  'scope.drift': 'scope drift in',
};

function eventDescription(event) {
  const name = event.actor || event.user || 'Someone';
  const verb = EVENT_VERBS[event.type] || event.type;
  const target = event.target || event.spec || event.repo || '';
  return `${name} ${verb} ${target}`;
}

function initialsAvatar(name) {
  if (!name) return '?';
  const parts = name.split(/[\s.@]+/);
  return parts.slice(0, 2).map(p => p[0] || '').join('').toUpperCase();
}

// ============================================================================
// Sidebar Icons (inline SVGs)
// ============================================================================

const NAV_ITEMS = [
  {
    key: 'overview', label: 'Overview', route: '',
    icon: html`<svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor"><rect x="2" y="2" width="7" height="7" rx="1"/><rect x="11" y="2" width="7" height="7" rx="1"/><rect x="2" y="11" width="7" height="7" rx="1"/><rect x="11" y="11" width="7" height="7" rx="1"/></svg>`,
  },
  {
    key: 'specs', label: 'Specs', route: 'specs',
    icon: html`<svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor"><rect x="3" y="1" width="14" height="18" rx="2"/><line x1="6" y1="6" x2="14" y2="6" stroke="var(--bg-primary)" stroke-width="1.5"/><line x1="6" y1="9" x2="14" y2="9" stroke="var(--bg-primary)" stroke-width="1.5"/><line x1="6" y1="12" x2="11" y2="12" stroke="var(--bg-primary)" stroke-width="1.5"/></svg>`,
  },
  {
    key: 'activity', label: 'Activity', route: 'activity',
    icon: html`<svg viewBox="0 0 20 20" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="1,10 5,10 8,4 12,16 15,10 19,10"/></svg>`,
  },
  {
    key: 'compliance', label: 'Compliance', route: 'compliance',
    icon: html`<svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor"><path d="M10 1L2 5v5c0 4.42 3.36 8.54 8 9.5 4.64-.96 8-5.08 8-9.5V5l-8-4z"/><polyline points="7,10 9,12 13,8" fill="none" stroke="var(--bg-primary)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
  },
  {
    key: 'audit', label: 'Audit', route: 'audit',
    icon: html`<svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor"><path d="M5 1h10a2 2 0 012 2v14a2 2 0 01-2 2H5a2 2 0 01-2-2V3a2 2 0 012-2z"/><rect x="7" y="0" width="6" height="3" rx="1" fill="var(--bg-secondary)"/><line x1="6" y1="7" x2="14" y2="7" stroke="var(--bg-primary)" stroke-width="1.5"/><line x1="6" y1="10" x2="14" y2="10" stroke="var(--bg-primary)" stroke-width="1.5"/><line x1="6" y1="13" x2="11" y2="13" stroke="var(--bg-primary)" stroke-width="1.5"/></svg>`,
  },
  {
    key: 'analytics', label: 'Analytics', route: 'analytics',
    icon: html`<svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor"><rect x="2" y="10" width="3" height="8" rx="0.5"/><rect x="7" y="6" width="3" height="12" rx="0.5"/><rect x="12" y="3" width="3" height="15" rx="0.5"/></svg>`,
  },
  {
    key: 'knowledge', label: 'Knowledge', route: 'knowledge',
    icon: html`<svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor"><path d="M2 3a2 2 0 012-2h4l2 2h4a2 2 0 012 2v2H2V3z"/><path d="M1 7h18v10a2 2 0 01-2 2H3a2 2 0 01-2-2V7z"/></svg>`,
  },
];

// ============================================================================
// Layout Shell
// ============================================================================

function OrgPicker() {
  const [orgs, setOrgs] = useState([]);
  const [open, setOpen] = useState(false);
  const [currentOrg, setCurrentOrg] = useState(() => api.getOrg());
  const [orgName, setOrgName] = useState('');

  useEffect(() => {
    api.get('/orgs').then(data => {
      const list = data.orgs || data || [];
      setOrgs(list);
      if (currentOrg) {
        const found = list.find(o => (o.id || o.slug) === currentOrg);
        if (found) setOrgName(found.name || found.slug || currentOrg);
        else setOrgName(currentOrg);
      } else if (list.length > 0) {
        const first = list[0];
        const id = first.id || first.slug;
        api.setOrg(id);
        setCurrentOrg(id);
        setOrgName(first.name || first.slug || id);
      }
    }).catch(() => {});
  }, []);

  const selectOrg = useCallback((org) => {
    const id = org.id || org.slug;
    api.setOrg(id);
    setCurrentOrg(id);
    setOrgName(org.name || org.slug || id);
    setOpen(false);
    window.location.reload();
  }, []);

  return html`
    <div class="org-picker">
      <button class="org-picker-btn" onClick=${() => setOpen(!open)}>
        ${orgName || 'Select Org'}
        <svg viewBox="0 0 20 20" width="12" height="12" fill="currentColor"><path d="M6 8l4 4 4-4"/></svg>
      </button>
      ${open && html`
        <div class="org-picker-dropdown">
          ${orgs.map(org => html`
            <button
              class="org-picker-item ${(org.id || org.slug) === currentOrg ? 'active' : ''}"
              onClick=${() => selectOrg(org)}
            >
              ${org.name || org.slug || org.id}
            </button>
          `)}
          ${orgs.length === 0 && html`<div class="org-picker-empty">No organizations</div>`}
        </div>
      `}
    </div>
  `;
}

function TopBar({ sidebarOpen, onToggleSidebar, auth, onOpenCmd }) {
  return html`
    <header class="topbar">
      <div class="topbar-left">
        <button class="icon-btn hamburger" onClick=${onToggleSidebar}>
          <svg viewBox="0 0 20 20" width="20" height="20" fill="currentColor">
            <rect x="2" y="4" width="16" height="2" rx="1"/>
            <rect x="2" y="9" width="16" height="2" rx="1"/>
            <rect x="2" y="14" width="16" height="2" rx="1"/>
          </svg>
        </button>
        <a class="topbar-brand" href="#/">
          <svg viewBox="0 0 90 90" width="24" height="24">
            <path d="M52 8 L22 46 L40 46 L34 82 L68 42 L50 42 Z" fill="var(--accent)" />
          </svg>
          <span class="topbar-brand-text">Hero</span>
        </a>
      </div>
      <div class="topbar-center">
        <${OrgPicker} />
      </div>
      <div class="topbar-right">
        <${ThemeToggle} />
        <button class="cmd-k-btn" onClick=${onOpenCmd} title="Command Palette (Cmd+K)">
          <kbd>⌘K</kbd>
        </button>
        <div class="avatar-circle" title=${auth.user?.name || auth.user?.login || ''}>
          ${initialsAvatar(auth.user?.name || auth.user?.login || auth.user?.email || '')}
        </div>
      </div>
    </header>
  `;
}

function Sidebar({ route, sidebarOpen, onToggleSidebar }) {
  return html`
    <nav class="sidebar ${sidebarOpen ? '' : 'collapsed'}">
      <div class="sidebar-nav">
        ${NAV_ITEMS.map(item => {
          const active = route.page === item.key || (item.key === 'overview' && route.page === 'overview');
          return html`
            <a
              class="nav-item ${active ? 'active' : ''}"
              href="#/${item.route}"
              title=${item.label}
            >
              <span class="nav-icon">${item.icon}</span>
              ${sidebarOpen && html`<span class="nav-label">${item.label}</span>`}
            </a>
          `;
        })}
      </div>
      <div class="sidebar-bottom">
        <button class="nav-item" onClick=${onToggleSidebar} title=${sidebarOpen ? 'Collapse' : 'Expand'}>
          <span class="nav-icon">
            <svg viewBox="0 0 20 20" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              ${sidebarOpen
                ? html`<polyline points="13,4 7,10 13,16"/>`
                : html`<polyline points="7,4 13,10 7,16"/>`
              }
            </svg>
          </span>
          ${sidebarOpen && html`<span class="nav-label">Collapse</span>`}
        </button>
      </div>
    </nav>
  `;
}

function Layout({ route, auth }) {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [cmdOpen, setCmdOpen] = useState(false);
  const pendingG = useRef(null);

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e) => {
      // Don't intercept if typing in an input
      const tag = e.target.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

      // Cmd+K / Ctrl+K
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setCmdOpen(o => !o);
        return;
      }

      // Escape
      if (e.key === 'Escape') {
        setCmdOpen(false);
        return;
      }

      // [ toggle sidebar
      if (e.key === '[') {
        setSidebarOpen(o => !o);
        return;
      }

      // g then letter navigation
      if (e.key === 'g') {
        pendingG.current = Date.now();
        setTimeout(() => { pendingG.current = null; }, 500);
        return;
      }

      if (pendingG.current && Date.now() - pendingG.current < 500) {
        pendingG.current = null;
        const map = { o: '', s: 'specs', a: 'activity', c: 'compliance', u: 'audit', n: 'analytics', k: 'knowledge' };
        if (map[e.key] !== undefined) {
          window.location.hash = `#/${map[e.key]}`;
        }
      }
    };

    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const pageMap = {
    overview: OverviewPage,
    specs: SpecsPage,
    activity: ActivityPage,
    compliance: CompliancePage,
    audit: AuditPage,
    analytics: AnalyticsPage,
    knowledge: KnowledgePage,
  };

  const PageComponent = pageMap[route.page] || OverviewPage;

  return html`
    <div class="layout">
      <${TopBar}
        sidebarOpen=${sidebarOpen}
        onToggleSidebar=${() => setSidebarOpen(o => !o)}
        auth=${auth}
        onOpenCmd=${() => setCmdOpen(true)}
      />
      <div class="layout-body">
        <${Sidebar} route=${route} sidebarOpen=${sidebarOpen} onToggleSidebar=${() => setSidebarOpen(o => !o)} />
        <main class="content">
          <${PageComponent} route=${route} auth=${auth} />
        </main>
      </div>
      <${CommandPalette} open=${cmdOpen} onClose=${() => setCmdOpen(false)} />
    </div>
  `;
}

// ============================================================================
// Overview Page
// ============================================================================

function OverviewPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [stats, setStats] = useState({ total: 0, active: 0, prRate: 0, compliance: 0 });
  const [activity, setActivity] = useState([]);
  const [pipeline, setPipeline] = useState({});
  const [insights, setInsights] = useState([]);
  const [intelligenceStatus, setIntelligenceStatus] = useState(false);

  // Real-time activity via SSE
  useSSE(useCallback((event) => {
    setActivity(prev => [{
      type: event.type,
      actor: event.payload?.user || '',
      target: event.payload?.repo || event.payload?.slug || '',
      timestamp: event.timestamp,
      ...event.payload,
    }, ...prev.slice(0, 19)]);
  }, []));

  useEffect(() => {
    if (!api.getOrg()) { setLoading(false); return; }

    const orgBase = `/orgs/${api.getOrg()}`;

    Promise.allSettled([
      api.get(`${orgBase}/activity?limit=10`),
      api.get(`${orgBase}/specs/pipeline`),
      api.get(`${orgBase}/governance/stats`),
      api.get(`${orgBase}/audit/summary`),
      api.get(`${orgBase}/insights`),
      api.get(`${orgBase}/intelligence/status`),
    ]).then(([actResult, pipResult, govResult, sumResult, insResult, intStatus]) => {
      if (actResult.status === 'fulfilled') {
        const data = actResult.value;
        setActivity(data.events || data.activity || data || []);
      }
      if (pipResult.status === 'fulfilled') {
        const p = pipResult.value.pipeline || {};
        setPipeline(p);
        const total = Object.values(p).reduce((s, v) => s + v, 0);
        const active = (p['draft'] || 0) + (p['approved'] || 0) + (p['in-progress'] || 0);
        setStats(prev => ({ ...prev, total, active }));
      }
      if (govResult.status === 'fulfilled') {
        setStats(prev => ({ ...prev, prRate: Math.round(govResult.value.link_rate_pct || 0) }));
      }
      if (sumResult.status === 'fulfilled') {
        const summary = sumResult.value.summary || {};
        const totalEvents = Object.values(summary).reduce((s, v) => s + v, 0);
        const matched = summary['convention.matched'] || 0;
        const score = totalEvents > 0 ? Math.round((matched / totalEvents) * 100) : 0;
        setStats(prev => ({ ...prev, compliance: score }));
      }
      if (insResult.status === 'fulfilled') {
        setInsights(insResult.value.insights || []);
      }
      if (intStatus.status === 'fulfilled') {
        setIntelligenceStatus(intStatus.value.opted_in || false);
      }
      setLoading(false);
    });
  }, []);

  if (loading) return html`<${LoadingSpinner} />`;

  const pipelineStatuses = ['draft', 'approved', 'in-progress', 'delivered', 'completed'];
  const pipelineTotal = pipelineStatuses.reduce((sum, k) => sum + (pipeline[k] || 0), 0) || 1;

  return html`
    <div class="page">
      <h1 class="page-title">Overview</h1>
      <div class="grid-4">
        <${StatCard} title="Total Specs" value=${stats.total} />
        <${StatCard} title="Active Specs" value=${stats.active} />
        <${StatCard} title="PR Link Rate" value=${stats.prRate + '%'} />
        <${StatCard} title="Compliance Score" value=${stats.compliance + '%'} />
      </div>
      <div class="grid-2">
        <div class="card">
          <h3 class="card-title">Recent Activity</h3>
          <div class="activity-list">
            ${activity.length === 0
              ? html`<div class="activity-empty">No recent activity</div>`
              : activity.map(ev => html`
                <div class="activity-item">
                  <div class="activity-avatar">${initialsAvatar(ev.actor || ev.user || '')}</div>
                  <div class="activity-body">
                    <span class="activity-desc">${eventDescription(ev)}</span>
                    <span class="activity-time">${relativeTime(ev.timestamp || ev.created_at)}</span>
                  </div>
                </div>
              `)
            }
          </div>
        </div>
        <div class="card">
          <h3 class="card-title">Spec Pipeline</h3>
          <div class="pipeline">
            ${pipelineStatuses.map(status => {
              const count = pipeline[status] || 0;
              const pct = Math.round((count / pipelineTotal) * 100);
              return html`
                <div class="pipeline-row">
                  <span class="pipeline-label">${status}</span>
                  <div class="pipeline-bar-bg">
                    <div class="pipeline-bar badge-${status}" style="width: ${pct}%"></div>
                  </div>
                  <span class="pipeline-count">${count}</span>
                </div>
              `;
            })}
          </div>
        </div>
      </div>
      ${(insights.length > 0 || intelligenceStatus) && html`
        <div class="card insights-card">
          <h3 class="card-title">Cross-Project Insights</h3>
          ${insights.length === 0
            ? html`<div class="insights-empty">No insights available yet. As the Hero network grows, recommendations for projects like yours will appear here.</div>`
            : html`
              <div class="insights-list">
                ${insights.map(ins => html`
                  <div class="insight-item insight-${ins.type}">
                    <div class="insight-header">
                      <span class="insight-type badge-${ins.type === 'convention' ? 'active' : ins.type === 'pattern' ? 'warning' : 'info'}">${ins.type}</span>
                      <span class="insight-confidence">${ins.confidence}% confidence</span>
                    </div>
                    <div class="insight-title">${ins.title}</div>
                    <div class="insight-desc">${ins.description}</div>
                    <div class="insight-source">${ins.source}</div>
                  </div>
                `)}
              </div>
            `
          }
        </div>
      `}
    </div>
  `;
}

// ============================================================================
// Specs Page
// ============================================================================

function SpecsPage({ route }) {
  const [view, setView] = useState('table');
  const [specs, setSpecs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({});
  const [search, setSearch] = useState('');
  const [selectedSpec, setSelectedSpec] = useState(null);
  const [specTab, setSpecTab] = useState('rendered');

  useEffect(() => {
    if (!api.getOrg()) { setLoading(false); return; }
    const params = new URLSearchParams({ limit: 200 });
    if (filters.type) params.set('type', filters.type);
    if (filters.status) params.set('status', filters.status);
    if (filters.subproject) params.set('subproject', filters.subproject);
    if (search) params.set('q', search);
    api.get(`/orgs/${api.getOrg()}/specs?${params}`).then(data => {
      setSpecs(data.specs || data.results || data || []);
    }).catch(() => {}).finally(() => setLoading(false));
  }, [filters, search]);

  // Distinct subprojects across the loaded set — populates the filter
  // dropdown. Subprojects are derived from corpus, not master-data.
  const subprojectOptions = useMemo(() => {
    const seen = new Set();
    for (const s of specs) {
      if (s.subproject) seen.add(s.subproject);
    }
    return [...seen].sort().map(v => ({ value: v, label: v }));
  }, [specs]);

  const filterDefs = [
    { key: 'type', label: 'Type', options: [
      { value: 'feature', label: 'Feature' },
      { value: 'bug', label: 'Bug' },
      { value: 'enhancement', label: 'Enhancement' },
    ]},
    { key: 'status', label: 'Status', options: [
      { value: 'draft', label: 'Draft' },
      { value: 'approved', label: 'Approved' },
      { value: 'in-progress', label: 'In Progress' },
      { value: 'delivered', label: 'Delivered' },
      { value: 'completed', label: 'Completed' },
    ]},
  ];
  // Only show the scope filter when at least one spec carries one —
  // repos without monorepo subprojects don't need the noise.
  if (subprojectOptions.length > 0) {
    filterDefs.push({ key: 'subproject', label: 'Scope', options: subprojectOptions });
  }

  const filtered = specs.filter(s => {
    if (filters.type && s.type !== filters.type) return false;
    if (filters.status && s.status !== filters.status) return false;
    if (filters.subproject && s.subproject !== filters.subproject) return false;
    if (search) {
      const lq = search.toLowerCase();
      if (!(s.title || '').toLowerCase().includes(lq) && !(s.slug || '').toLowerCase().includes(lq)) return false;
    }
    return true;
  });

  const columns = [
    { key: 'slug', label: 'Slug', sortable: true },
    { key: 'title', label: 'Title', sortable: true },
    { key: 'type', label: 'Type', render: (v) => v ? html`<${Badge} type=${v} />` : '' },
    { key: 'status', label: 'Status', render: (v) => v ? html`<${Badge} type=${v} />` : '' },
    { key: 'subproject', label: 'Scope', render: (v) => v ? html`<span class="scope-badge">${v}</span>` : '' },
    { key: 'repo', label: 'Repo', sortable: true },
    { key: 'updated_at', label: 'Updated', render: (v) => relativeTime(v), sortable: true },
  ];

  const statusColumns = ['draft', 'approved', 'in-progress', 'delivered', 'completed'];

  if (loading) return html`<${LoadingSpinner} />`;

  return html`
    <div class="page">
      <div class="page-header">
        <h1 class="page-title">Specs</h1>
        <div class="view-toggle">
          <button class="icon-btn ${view === 'table' ? 'active' : ''}" onClick=${() => setView('table')} title="Table view">
            <svg viewBox="0 0 20 20" width="18" height="18" fill="currentColor"><rect x="2" y="3" width="16" height="2"/><rect x="2" y="7" width="16" height="2"/><rect x="2" y="11" width="16" height="2"/><rect x="2" y="15" width="16" height="2"/></svg>
          </button>
          <button class="icon-btn ${view === 'board' ? 'active' : ''}" onClick=${() => setView('board')} title="Board view">
            <svg viewBox="0 0 20 20" width="18" height="18" fill="currentColor"><rect x="1" y="2" width="5" height="16" rx="1"/><rect x="7.5" y="2" width="5" height="12" rx="1"/><rect x="14" y="2" width="5" height="14" rx="1"/></svg>
          </button>
        </div>
      </div>

      <${FilterBar}
        filters=${filterDefs}
        activeFilters=${filters}
        onFilterChange=${setFilters}
        searchValue=${search}
        onSearchChange=${setSearch}
      />

      ${view === 'table'
        ? html`
          <${DataTable}
            columns=${columns}
            data=${filtered}
            sortable
            onRowClick=${(row) => setSelectedSpec(row)}
          />
          ${filtered.length === 0 && html`
            <${EmptyState}
              title="No specs found"
              description="Try adjusting your filters or import specs from your repositories."
            />
          `}
        `
        : html`
          <div class="board">
            ${statusColumns.map(status => {
              const cards = filtered.filter(s => s.status === status);
              return html`
                <div class="board-column">
                  <div class="board-column-header">
                    <${Badge} type=${status} />
                    <span class="board-column-count">${cards.length}</span>
                  </div>
                  <div class="board-column-body">
                    ${cards.map(s => html`
                      <div class="board-card" onClick=${() => setSelectedSpec(s)}>
                        <div class="board-card-title">${s.title || s.slug}</div>
                        <div class="board-card-meta">
                          ${s.type && html`<${Badge} type=${s.type} />`}
                          ${s.repo && html`<span class="board-card-repo">${s.repo}</span>`}
                        </div>
                      </div>
                    `)}
                  </div>
                </div>
              `;
            })}
          </div>
        `
      }

      <${SlideOver}
        open=${!!selectedSpec}
        onClose=${() => setSelectedSpec(null)}
        title=${selectedSpec?.title || selectedSpec?.slug || 'Spec'}
      >
        ${selectedSpec && html`<${SpecDetail} spec=${selectedSpec} tab=${specTab} onTabChange=${setSpecTab} />`}
      <//>
    </div>
  `;
}

function SpecDetail({ spec, tab, onTabChange }) {
  const [conflicts, setConflicts] = useState([]);

  useEffect(() => {
    if (!spec?.slug || !api.getOrg()) return;
    apiFetch(`/api/v1/orgs/${api.getOrg()}/specs/${spec.slug}/conflicts`)
      .then(r => r.json())
      .then(d => setConflicts(d.conflicts || []))
      .catch(() => setConflicts([]));
  }, [spec?.slug]);

  return html`
    <div class="spec-detail">
      <div class="spec-meta">
        <div class="spec-meta-row">
          <span class="spec-meta-label">Status</span>
          <${Badge} type=${spec.status} />
        </div>
        <div class="spec-meta-row">
          <span class="spec-meta-label">Type</span>
          <${Badge} type=${spec.type} />
        </div>
        ${spec.repo && html`
          <div class="spec-meta-row">
            <span class="spec-meta-label">Repo</span>
            <span>${spec.repo}</span>
          </div>
        `}
        ${spec.score != null && html`
          <div class="spec-meta-row">
            <span class="spec-meta-label">Score</span>
            <span>${spec.score}</span>
          </div>
        `}
        ${spec.priority && html`
          <div class="spec-meta-row">
            <span class="spec-meta-label">Priority</span>
            <span>${spec.priority}</span>
          </div>
        `}
        ${spec.tracker_id && html`
          <div class="spec-meta-row">
            <span class="spec-meta-label">Tracker</span>
            <span>${spec.tracker_id}</span>
          </div>
        `}
      </div>

      ${conflicts.length > 0 && html`
        <div class="conflicts-section">
          <h4 class="conflicts-heading">File Conflicts (${conflicts.length})</h4>
          <div class="conflicts-list">
            ${conflicts.map(c => html`
              <div class="conflict-item">
                <div class="conflict-header">
                  <${Badge} type=${c.status} />
                  <strong>${c.slug}</strong>
                </div>
                <div class="conflict-title">${c.title}</div>
                <div class="conflict-files">
                  ${c.overlapping_files.map(f => html`<code class="conflict-file">${f}</code>`)}
                </div>
              </div>
            `)}
          </div>
        </div>
      `}

      <div class="spec-tabs">
        <button class="tab-btn ${tab === 'rendered' ? 'active' : ''}" onClick=${() => onTabChange('rendered')}>Rendered</button>
        <button class="tab-btn ${tab === 'source' ? 'active' : ''}" onClick=${() => onTabChange('source')}>Source</button>
      </div>

      <div class="spec-content">
        ${tab === 'rendered'
          ? html`<div class="markdown-body" dangerouslySetInnerHTML=${{ __html: marked(spec.content || spec.body || '') }} />`
          : html`<pre class="spec-source">${spec.content || spec.body || ''}</pre>`
        }
      </div>
    </div>
  `;
}

// ============================================================================
// Activity Page
// ============================================================================

function ActivityPage() {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({});
  const [hasMore, setHasMore] = useState(true);
  const [page, setPage] = useState(1);

  // Real-time events via SSE
  useSSE(useCallback((event) => {
    // Only prepend if no type filter or matches filter
    if (filters.type && event.type !== filters.type) return;
    setEvents(prev => [{
      event_type: event.type,
      type: event.type,
      actor: event.payload?.user || '',
      target: event.payload?.repo || event.payload?.slug || '',
      timestamp: event.timestamp,
      created_at: event.timestamp,
      ...event.payload,
    }, ...prev]);
  }, [filters.type]));

  const fetchEvents = useCallback((pageNum, append) => {
    if (!api.getOrg()) { setLoading(false); return; }
    const params = new URLSearchParams({ page: pageNum, limit: 20 });
    if (filters.type) params.set('type', filters.type);

    api.get(`/orgs/${api.getOrg()}/activity?${params}`).then(data => {
      const list = data.events || data.activity || data || [];
      if (append) {
        setEvents(prev => [...prev, ...list]);
      } else {
        setEvents(list);
      }
      setHasMore(list.length >= 20);
    }).catch(() => {}).finally(() => setLoading(false));
  }, [filters]);

  useEffect(() => {
    setLoading(true);
    setPage(1);
    fetchEvents(1, false);
  }, [filters]);

  const loadMore = useCallback(() => {
    const next = page + 1;
    setPage(next);
    fetchEvents(next, true);
  }, [page, fetchEvents]);

  const filterDefs = [
    { key: 'type', label: 'Event Type', options: [
      { value: 'sync', label: 'Sync' },
      { value: 'spec.created', label: 'Spec Created' },
      { value: 'spec.synced', label: 'Spec Synced' },
      { value: 'spec.delivered', label: 'Spec Delivered' },
      { value: 'pr.checked', label: 'PR Checked' },
      { value: 'convention.matched', label: 'Convention Matched' },
      { value: 'scope.drift', label: 'Scope Drift' },
    ]},
  ];

  if (loading && events.length === 0) return html`<${LoadingSpinner} />`;

  return html`
    <div class="page">
      <h1 class="page-title">Activity</h1>
      <${FilterBar}
        filters=${filterDefs}
        activeFilters=${filters}
        onFilterChange=${setFilters}
      />
      <div class="activity-feed">
        ${events.length === 0
          ? html`<${EmptyState} title="No activity yet" description="Activity will appear here as you work with specs." />`
          : events.map(ev => html`
            <div class="activity-item">
              <div class="activity-avatar">${initialsAvatar(ev.actor || ev.user || '')}</div>
              <div class="activity-body">
                <span class="activity-desc">${eventDescription(ev)}</span>
                <span class="activity-time">${relativeTime(ev.timestamp || ev.created_at)}</span>
              </div>
            </div>
          `)
        }
        ${hasMore && events.length > 0 && html`
          <button class="btn btn-secondary load-more" onClick=${loadMore}>Load More</button>
        `}
      </div>
    </div>
  `;
}

// ============================================================================
// Compliance Page
// ============================================================================

function CompliancePage() {
  const [loading, setLoading] = useState(true);
  const [conventions, setConventions] = useState([]);
  const [summary, setSummary] = useState({});
  const [govStats, setGovStats] = useState(null);
  const [selectedConv, setSelectedConv] = useState(null);
  const [convTab, setConvTab] = useState('rendered');

  useEffect(() => {
    if (!api.getOrg()) { setLoading(false); return; }
    const orgBase = `/orgs/${api.getOrg()}`;

    Promise.allSettled([
      api.get(`${orgBase}/conventions`),
      api.get(`${orgBase}/audit/summary`),
      api.get(`${orgBase}/governance/stats`),
    ]).then(([convResult, sumResult, govResult]) => {
      if (convResult.status === 'fulfilled') {
        setConventions(convResult.value.conventions || []);
      }
      if (sumResult.status === 'fulfilled') {
        setSummary(sumResult.value.summary || {});
      }
      if (govResult.status === 'fulfilled') {
        setGovStats(govResult.value);
      }
      setLoading(false);
    });
  }, []);

  if (loading) return html`<${LoadingSpinner} />`;

  const activeConventions = conventions.filter(c => c.status === 'active').length;
  const totalChecks = govStats ? govStats.total_prs || 0 : 0;
  const compliancePct = govStats ? Math.round(govStats.link_rate_pct || 0) : 0;
  const conventionMatches = summary['convention.matched'] || 0;

  const columns = [
    { key: 'slug', label: 'Slug', sortable: true },
    { key: 'title', label: 'Title', sortable: true },
    { key: 'status', label: 'Status', render: (v) => html`<${Badge} type=${v} />` },
    { key: 'scope', label: 'Scope', render: (v) => (v || []).join(', ') || '--' },
    { key: 'synced_at', label: 'Last Synced', render: (v) => relativeTime(v), sortable: true },
  ];

  return html`
    <div class="page">
      <h1 class="page-title">Compliance</h1>
      <div class="grid-4">
        <${StatCard} title="Active Conventions" value=${activeConventions} />
        <${StatCard} title="PRs Checked" value=${totalChecks} />
        <${StatCard} title="Spec Link Rate" value=${compliancePct + '%'} />
        <${StatCard} title="Convention Matches" value=${conventionMatches} />
      </div>

      ${conventions.length === 0
        ? html`<${EmptyState}
            title="No conventions yet"
            description="Conventions will appear here once synced from your repositories."
            icon=${html`<svg viewBox="0 0 20 20" width="48" height="48" fill="var(--text-muted)"><path d="M10 1L2 5v5c0 4.42 3.36 8.54 8 9.5 4.64-.96 8-5.08 8-9.5V5l-8-4z"/></svg>`}
          />`
        : html`
          <div class="card" style="margin-top: 24px">
            <h3 class="card-title">Conventions</h3>
            <${DataTable}
              columns=${columns}
              data=${conventions}
              sortable
              onRowClick=${(row) => setSelectedConv(row)}
            />
          </div>
        `
      }

      ${govStats && html`
        <div class="card" style="margin-top: 24px">
          <h3 class="card-title">Governance Overview</h3>
          <div class="grid-3">
            <div class="stat-mini"><span class="stat-mini-value">${govStats.total_prs || 0}</span><span class="stat-mini-label">Total PRs</span></div>
            <div class="stat-mini"><span class="stat-mini-value">${govStats.linked_prs || 0}</span><span class="stat-mini-label">Linked to Specs</span></div>
            <div class="stat-mini"><span class="stat-mini-value">${govStats.unlinked_prs || 0}</span><span class="stat-mini-label">Unlinked</span></div>
          </div>
        </div>
      `}

      <${SlideOver}
        open=${!!selectedConv}
        onClose=${() => setSelectedConv(null)}
        title=${selectedConv?.title || selectedConv?.slug || 'Convention'}
      >
        ${selectedConv && html`
          <div class="spec-detail">
            <div class="spec-meta">
              <div class="spec-meta-row">
                <span class="spec-meta-label">Status</span>
                <${Badge} type=${selectedConv.status} />
              </div>
              <div class="spec-meta-row">
                <span class="spec-meta-label">Scope</span>
                <span>${(selectedConv.scope || []).join(', ') || 'None'}</span>
              </div>
              <div class="spec-meta-row">
                <span class="spec-meta-label">Last Synced</span>
                <span>${relativeTime(selectedConv.synced_at)}</span>
              </div>
            </div>
            <div class="spec-tabs">
              <button class="tab-btn ${convTab === 'rendered' ? 'active' : ''}" onClick=${() => setConvTab('rendered')}>Rendered</button>
              <button class="tab-btn ${convTab === 'source' ? 'active' : ''}" onClick=${() => setConvTab('source')}>Source</button>
            </div>
            <div class="spec-content">
              ${convTab === 'rendered'
                ? html`<div class="markdown-body" dangerouslySetInnerHTML=${{ __html: marked(selectedConv.content || '') }} />`
                : html`<pre class="spec-source">${selectedConv.content || ''}</pre>`
              }
            </div>
          </div>
        `}
      <//>
    </div>
  `;
}

// ============================================================================
// Audit Page
// ============================================================================

function AuditPage() {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({});
  const [hasMore, setHasMore] = useState(true);
  const [page, setPage] = useState(0);
  const [since, setSince] = useState('');
  const [until, setUntil] = useState('');
  const pageSize = 50;

  const buildParams = useCallback((offset) => {
    const params = new URLSearchParams({ limit: pageSize, offset });
    if (filters.type) params.set('types', filters.type);
    if (since) params.set('since', new Date(since).toISOString());
    if (until) params.set('until', new Date(until).toISOString());
    return params;
  }, [filters, since, until]);

  const fetchEvents = useCallback((offset, append) => {
    if (!api.getOrg()) { setLoading(false); return; }
    const params = buildParams(offset);

    api.get(`/orgs/${api.getOrg()}/audit?${params}`).then(data => {
      const list = data.events || [];
      if (append) {
        setEvents(prev => [...prev, ...list]);
      } else {
        setEvents(list);
      }
      setHasMore(list.length >= pageSize);
    }).catch(() => {}).finally(() => setLoading(false));
  }, [buildParams]);

  useEffect(() => {
    setLoading(true);
    setPage(0);
    fetchEvents(0, false);
  }, [filters, since, until]);

  const loadMore = useCallback(() => {
    const nextOffset = (page + 1) * pageSize;
    setPage(p => p + 1);
    fetchEvents(nextOffset, true);
  }, [page, fetchEvents]);

  const exportCsv = useCallback(() => {
    if (!api.getOrg()) return;
    const params = buildParams(0);
    params.set('format', 'csv');
    params.set('limit', '10000');
    const token = localStorage.getItem('hero-token');
    const url = `/api/v1/orgs/${api.getOrg()}/audit?${params}`;
    // Open in new tab with auth via fetch + blob
    fetch(url, { headers: { 'Authorization': `Bearer ${token}` } })
      .then(r => r.blob())
      .then(blob => {
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = 'audit-export.csv';
        a.click();
        URL.revokeObjectURL(a.href);
      });
  }, [buildParams]);

  const filterDefs = [
    { key: 'type', label: 'Event Type', options: [
      { value: 'spec.created', label: 'Spec Created' },
      { value: 'spec.delivered', label: 'Spec Delivered' },
      { value: 'spec.merged', label: 'Spec Merged' },
      { value: 'spec.approved', label: 'Spec Approved' },
      { value: 'spec.synced', label: 'Spec Synced' },
      { value: 'pr.checked', label: 'PR Checked' },
      { value: 'pr.linked', label: 'PR Linked' },
      { value: 'convention.matched', label: 'Convention Matched' },
      { value: 'scope.drift', label: 'Scope Drift' },
    ]},
  ];

  function payloadSummary(payload) {
    if (!payload) return '';
    let p = payload;
    if (typeof p === 'string') {
      try { p = JSON.parse(p); } catch { return p; }
    }
    const parts = [];
    if (p.repo) parts.push(p.repo);
    if (p.slug) parts.push(p.slug);
    if (p.pr_number) parts.push(`PR #${p.pr_number}`);
    if (p.conclusion) parts.push(p.conclusion);
    if (p.convention) parts.push(p.convention);
    return parts.join(' | ') || JSON.stringify(p).slice(0, 80);
  }

  const columns = [
    { key: 'created_at', label: 'Time', render: (v) => relativeTime(v), sortable: true },
    { key: 'event_type', label: 'Event', render: (v) => html`<${Badge} type=${v} label=${v} />`, sortable: true },
    { key: 'user_id', label: 'User', render: (v) => v || '--' },
    { key: 'repo_id', label: 'Repo', render: (v) => v || '--' },
    { key: 'payload', label: 'Details', render: (v) => payloadSummary(v), sortable: false },
  ];

  if (loading && events.length === 0) return html`<${LoadingSpinner} />`;

  return html`
    <div class="page">
      <div class="page-header">
        <h1 class="page-title">Audit Trail</h1>
        <button class="btn btn-secondary" onClick=${exportCsv} title="Export as CSV">
          <svg viewBox="0 0 20 20" width="16" height="16" fill="currentColor" style="margin-right:6px;vertical-align:middle">
            <path d="M10 1v12m0 0l-4-4m4 4l4-4M3 15v2a1 1 0 001 1h12a1 1 0 001-1v-2"/>
          </svg>
          Export CSV
        </button>
      </div>

      <div class="filter-bar">
        <div class="filter-chips">
          ${filterDefs.map(f => html`
            <select
              class="filter-select"
              value=${filters[f.key] || ''}
              onChange=${(e) => setFilters({ ...filters, [f.key]: e.target.value })}
            >
              <option value="">${f.label}</option>
              ${f.options.map(o => html`<option value=${o.value}>${o.label}</option>`)}
            </select>
          `)}
          <input
            class="filter-date"
            type="date"
            value=${since}
            onChange=${(e) => setSince(e.target.value)}
            title="Since"
            placeholder="Since"
          />
          <input
            class="filter-date"
            type="date"
            value=${until}
            onChange=${(e) => setUntil(e.target.value)}
            title="Until"
            placeholder="Until"
          />
        </div>
      </div>

      ${events.length === 0
        ? html`<${EmptyState} title="No audit events" description="Audit events will appear as governance checks run." />`
        : html`
          <${DataTable}
            columns=${columns}
            data=${events}
            sortable
          />
          ${hasMore && html`
            <button class="btn btn-secondary load-more" style="margin-top:16px" onClick=${loadMore}>Load More</button>
          `}
        `
      }
    </div>
  `;
}

// ============================================================================
// Analytics Page
// ============================================================================

function AnalyticsPage() {
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState('30');
  const [overview, setOverview] = useState(null);
  const [velocity, setVelocity] = useState([]);
  const [heatmap, setHeatmap] = useState([]);
  const [pipeline, setPipeline] = useState({});
  const chartRef = useRef(null);
  const chartInstance = useRef(null);

  const fetchData = useCallback((days) => {
    if (!api.getOrg()) { setLoading(false); return; }
    const orgBase = `/orgs/${api.getOrg()}`;
    const since = new Date(Date.now() - days * 86400000).toISOString();
    const until = new Date().toISOString();
    const rangeParams = `since=${since}&until=${until}`;

    Promise.allSettled([
      api.get(`${orgBase}/analytics/overview?${rangeParams}`),
      api.get(`${orgBase}/analytics/velocity?${rangeParams}&interval=${days <= 14 ? 'day' : 'week'}`),
      api.get(`${orgBase}/analytics/heatmap`),
      api.get(`${orgBase}/specs/pipeline`),
    ]).then(([ovResult, velResult, heatResult, pipResult]) => {
      if (ovResult.status === 'fulfilled') setOverview(ovResult.value);
      if (velResult.status === 'fulfilled') setVelocity(velResult.value.buckets || []);
      if (heatResult.status === 'fulfilled') setHeatmap(heatResult.value.days || []);
      if (pipResult.status === 'fulfilled') setPipeline(pipResult.value.pipeline || {});
      setLoading(false);
    });
  }, []);

  useEffect(() => {
    setLoading(true);
    fetchData(parseInt(range));
  }, [range]);

  // Render velocity chart with Chart.js
  useEffect(() => {
    if (!chartRef.current || velocity.length === 0) return;

    import('./lib/chart.min.js').then(({ Chart, registerables }) => {
      if (registerables) Chart.register(...registerables);

      if (chartInstance.current) chartInstance.current.destroy();

      const labels = velocity.map(b => {
        const d = new Date(b.period);
        return `${d.getMonth()+1}/${d.getDate()}`;
      });

      const style = getComputedStyle(document.documentElement);
      const accent = style.getPropertyValue('--accent').trim();
      const success = style.getPropertyValue('--success').trim();
      const textMuted = style.getPropertyValue('--text-muted').trim();
      const border = style.getPropertyValue('--border').trim();

      chartInstance.current = new Chart(chartRef.current, {
        type: 'line',
        data: {
          labels,
          datasets: [
            {
              label: 'Delivered',
              data: velocity.map(b => b.delivered),
              borderColor: success,
              backgroundColor: success + '22',
              fill: true,
              tension: 0.3,
            },
            {
              label: 'Created',
              data: velocity.map(b => b.created),
              borderColor: accent,
              backgroundColor: accent + '22',
              fill: true,
              tension: 0.3,
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: { legend: { labels: { color: textMuted } } },
          scales: {
            x: { ticks: { color: textMuted }, grid: { color: border } },
            y: { ticks: { color: textMuted }, grid: { color: border }, beginAtZero: true },
          },
        },
      });
    }).catch(() => {});

    return () => { if (chartInstance.current) chartInstance.current.destroy(); };
  }, [velocity]);

  if (loading) return html`<${LoadingSpinner} />`;

  const pipelineStatuses = ['draft', 'approved', 'in-progress', 'delivered', 'completed'];
  const pipelineTotal = pipelineStatuses.reduce((sum, k) => sum + (pipeline[k] || 0), 0) || 1;

  // Build heatmap grid (last 52 weeks)
  const heatmapMap = {};
  heatmap.forEach(d => {
    const key = new Date(d.date).toISOString().slice(0, 10);
    heatmapMap[key] = d.count;
  });
  const maxCount = Math.max(1, ...heatmap.map(d => d.count));

  function heatmapColor(count) {
    if (!count) return 'var(--bg-elevated)';
    const intensity = Math.min(count / maxCount, 1);
    if (intensity < 0.25) return 'var(--accent-subtle)';
    if (intensity < 0.5) return 'color-mix(in srgb, var(--accent) 40%, var(--accent-subtle))';
    if (intensity < 0.75) return 'color-mix(in srgb, var(--accent) 70%, var(--accent-subtle))';
    return 'var(--accent)';
  }

  // Generate last 365 days for heatmap
  const heatmapDays = [];
  for (let i = 364; i >= 0; i--) {
    const d = new Date(Date.now() - i * 86400000);
    const key = d.toISOString().slice(0, 10);
    heatmapDays.push({ key, count: heatmapMap[key] || 0, dow: d.getDay() });
  }

  return html`
    <div class="page">
      <div class="page-header">
        <h1 class="page-title">Analytics</h1>
        <div class="range-selector">
          ${['7', '30', '90'].map(d => html`
            <button
              class="btn ${range === d ? 'btn-primary' : 'btn-secondary'} btn-sm"
              onClick=${() => setRange(d)}
            >${d}d</button>
          `)}
        </div>
      </div>

      <div class="grid-4">
        <${StatCard} title="Specs Delivered" value=${overview?.specs_delivered ?? 0} />
        <${StatCard} title="Avg Time to Merge" value=${(overview?.avg_time_to_merge_hours ?? 0).toFixed(1) + 'h'} />
        <${StatCard} title="Rework Rate" value=${(overview?.rework_rate_pct ?? 0).toFixed(1) + '%'} />
        <${StatCard} title="AI Leverage" value=${(overview?.ai_leverage_pct ?? 0).toFixed(1) + '%'} />
      </div>

      <div class="grid-2" style="margin-top:24px">
        <div class="card">
          <h3 class="card-title">Delivery Velocity</h3>
          <div style="height:280px;position:relative">
            ${velocity.length === 0
              ? html`<div class="activity-empty" style="padding-top:80px">No velocity data yet</div>`
              : html`<canvas ref=${chartRef}></canvas>`
            }
          </div>
        </div>
        <div class="card">
          <h3 class="card-title">Spec Pipeline</h3>
          <div class="pipeline">
            ${pipelineStatuses.map(status => {
              const count = pipeline[status] || 0;
              const pct = Math.round((count / pipelineTotal) * 100);
              return html`
                <div class="pipeline-row">
                  <span class="pipeline-label">${status}</span>
                  <div class="pipeline-bar-bg">
                    <div class="pipeline-bar badge-${status}" style="width: ${pct}%"></div>
                  </div>
                  <span class="pipeline-count">${count}</span>
                </div>
              `;
            })}
          </div>
        </div>
      </div>

      <div class="card" style="margin-top:24px">
        <h3 class="card-title">Activity Heatmap</h3>
        <div class="heatmap-grid">
          ${heatmapDays.map(d => html`
            <div
              class="heatmap-cell"
              style="background:${heatmapColor(d.count)}"
              title="${d.key}: ${d.count} events"
            />
          `)}
        </div>
        <div class="heatmap-legend">
          <span class="text-muted" style="font-size:12px">Less</span>
          <div class="heatmap-cell" style="background:var(--bg-elevated)"></div>
          <div class="heatmap-cell" style="background:var(--accent-subtle)"></div>
          <div class="heatmap-cell" style="background:color-mix(in srgb, var(--accent) 40%, var(--accent-subtle))"></div>
          <div class="heatmap-cell" style="background:color-mix(in srgb, var(--accent) 70%, var(--accent-subtle))"></div>
          <div class="heatmap-cell" style="background:var(--accent)"></div>
          <span class="text-muted" style="font-size:12px">More</span>
        </div>
      </div>
    </div>
  `;
}

// ============================================================================
// Knowledge Page
// ============================================================================

function KnowledgePage() {
  const [loading, setLoading] = useState(true);
  const [conventions, setConventions] = useState([]);
  const [search, setSearch] = useState('');
  const [searchResults, setSearchResults] = useState(null);
  const [tab, setTab] = useState('conventions');
  const [selectedItem, setSelectedItem] = useState(null);
  const [itemTab, setItemTab] = useState('rendered');

  useEffect(() => {
    if (!api.getOrg()) { setLoading(false); return; }
    api.get(`/orgs/${api.getOrg()}/conventions`).then(data => {
      setConventions(data.conventions || []);
    }).catch(() => {}).finally(() => setLoading(false));
  }, []);

  const doSearch = useCallback(() => {
    if (!search.trim() || !api.getOrg()) return;
    api.get(`/orgs/${api.getOrg()}/knowledge/search?q=${encodeURIComponent(search)}`).then(data => {
      setSearchResults(data);
    }).catch(() => {});
  }, [search]);

  // Debounced search
  useEffect(() => {
    if (!search.trim()) { setSearchResults(null); return; }
    const t = setTimeout(doSearch, 300);
    return () => clearTimeout(t);
  }, [search, doSearch]);

  if (loading) return html`<${LoadingSpinner} />`;

  const displayItems = searchResults
    ? (tab === 'conventions'
        ? searchResults.conventions || []
        : tab === 'specs'
          ? searchResults.specs || []
          : [...(searchResults.conventions || []), ...(searchResults.specs || [])])
    : (tab === 'conventions' ? conventions : []);

  return html`
    <div class="page">
      <h1 class="page-title">Knowledge Base</h1>

      <div class="filter-bar">
        <input
          class="filter-search"
          type="text"
          placeholder="Search knowledge..."
          value=${search}
          onInput=${(e) => setSearch(e.target.value)}
        />
      </div>

      <div class="spec-tabs" style="margin-bottom:20px">
        <button class="tab-btn ${tab === 'conventions' ? 'active' : ''}" onClick=${() => setTab('conventions')}>Conventions</button>
        <button class="tab-btn ${tab === 'specs' ? 'active' : ''}" onClick=${() => setTab('specs')}>Specs</button>
        <button class="tab-btn ${tab === 'all' ? 'active' : ''}" onClick=${() => setTab('all')}>All</button>
      </div>

      ${displayItems.length === 0
        ? html`<${EmptyState}
            title=${search ? 'No results found' : (tab === 'specs' ? 'Specs search' : 'No conventions yet')}
            description=${search ? 'Try a different search term.' : (tab === 'specs' ? 'Type a search term to find specs.' : 'Conventions will appear once synced from your repositories.')}
            icon=${html`<svg viewBox="0 0 20 20" width="48" height="48" fill="var(--text-muted)"><path d="M2 3a2 2 0 012-2h4l2 2h4a2 2 0 012 2v2H2V3z"/><path d="M1 7h18v10a2 2 0 01-2 2H3a2 2 0 01-2-2V7z"/></svg>`}
          />`
        : html`
          <div class="knowledge-grid">
            ${displayItems.map(item => html`
              <div class="knowledge-card" onClick=${() => { setSelectedItem(item); setItemTab('rendered'); }}>
                <div class="knowledge-card-header">
                  <${Badge} type=${item.status || item.type || 'unknown'} />
                  ${item.slug && html`<span class="text-muted" style="font-size:12px">${item.slug}</span>`}
                </div>
                <h4 class="knowledge-card-title">${item.title || item.slug || 'Untitled'}</h4>
                ${item.scope && item.scope.length > 0 && html`
                  <div class="knowledge-card-scope">${item.scope.join(', ')}</div>
                `}
              </div>
            `)}
          </div>
        `
      }

      <${SlideOver}
        open=${!!selectedItem}
        onClose=${() => setSelectedItem(null)}
        title=${selectedItem?.title || selectedItem?.slug || 'Item'}
      >
        ${selectedItem && html`
          <div class="spec-detail">
            <div class="spec-meta">
              <div class="spec-meta-row">
                <span class="spec-meta-label">Type</span>
                <${Badge} type=${selectedItem.type || 'convention'} />
              </div>
              ${selectedItem.status && html`
                <div class="spec-meta-row">
                  <span class="spec-meta-label">Status</span>
                  <${Badge} type=${selectedItem.status} />
                </div>
              `}
              ${selectedItem.scope && html`
                <div class="spec-meta-row">
                  <span class="spec-meta-label">Scope</span>
                  <span>${(selectedItem.scope || []).join(', ')}</span>
                </div>
              `}
            </div>
            <div class="spec-tabs">
              <button class="tab-btn ${itemTab === 'rendered' ? 'active' : ''}" onClick=${() => setItemTab('rendered')}>Rendered</button>
              <button class="tab-btn ${itemTab === 'source' ? 'active' : ''}" onClick=${() => setItemTab('source')}>Source</button>
            </div>
            <div class="spec-content">
              ${itemTab === 'rendered'
                ? html`<div class="markdown-body" dangerouslySetInnerHTML=${{ __html: marked(selectedItem.content || selectedItem.body || '') }} />`
                : html`<pre class="spec-source">${selectedItem.content || selectedItem.body || ''}</pre>`
              }
            </div>
          </div>
        `}
      <//>
    </div>
  `;
}

// ============================================================================
// App Bootstrap
// ============================================================================

function App() {
  const route = useRouter();
  const auth = useAuth();

  if (route.page === 'auth') return html`<${AuthCallback} />`;
  if (!auth.token) return html`<${LoginPage} />`;
  return html`<${Layout} route=${route} auth=${auth} />`;
}

render(html`<${App} />`, document.getElementById('app'));
