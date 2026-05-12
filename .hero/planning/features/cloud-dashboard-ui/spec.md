---
title: "Hero Cloud Dashboard UI"
type: feature
status: approved
priority: high
tags: [cloud, dashboard, frontend, phase-5]
horizon: next
smoke: deferred
---

# Hero Cloud Dashboard UI

## Problem

Hero Cloud has a full REST API (specs, activity, audit, governance, compliance) but no web interface. CTOs and engineering leads need a visual dashboard to prove AI-driven engineering is working — spec fidelity, rework rates, compliance status, delivery velocity. Without a UI, the cloud backend is invisible.

## Proposed Solution

A lightweight SPA served by the Go backend at `cloud.herospec.dev`. Embedded in the Go binary via `embed.FS`. No heavy build toolchain — Preact + HTM (tagged template literals, no JSX transform needed) with client-side routing.

---

## Tech Stack

### Frontend
- **Preact 10** — 3KB gzipped, full component model, hooks
- **HTM** — tagged template literal JSX alternative, no build step needed
- **Preact Router** — client-side hash routing (`#/specs`, `#/activity`, etc.)
- **Chart.js** — lightweight charts (stat cards, time-series, sparklines)
- All loaded via ESM from CDN (unpkg/esm.sh), zero build toolchain

### Why not other options
- **Vanilla JS/HTML**: too much boilerplate for 7 pages with shared state (auth, org context, filters)
- **Alpine.js/htmx**: better for server-rendered pages; this is a data-dense SPA with client-side filtering, sorting, view toggles
- **React/Vue**: too heavy, requires build toolchain
- **Preact + HTM**: component model without a build step — best of both worlds

### Backend Integration
- Go serves the SPA via `embed.FS` from `cloud/web/` directory
- All routes serve `index.html` (SPA fallback), API routes take precedence
- Static files: `index.html`, `app.js`, `style.css`, plus any vendored libs

### File Embedding

```go
// cloud/web.go
package cloud

import "embed"

//go:embed web/*
var WebAssets embed.FS
```

```go
// In router.go — add SPA handler after API routes
func serveSPA(assets embed.FS) http.Handler {
    fs := http.FileServer(http.FS(assets))
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Try static file first, fall back to index.html
        path := "web" + r.URL.Path
        if _, err := assets.Open(path); err != nil {
            // SPA fallback
            r.URL.Path = "/"
        }
        fs.ServeHTTP(w, r)
    })
}
```

---

## Design System

### Color Palette

```
Background:     #14181e    --bg
Surface:        #1c2128    --surface
Elevated:       #252b33    --elevated
Border:         #2d333b    --border
Text primary:   #e0e6ed    --text
Text secondary: #8b949e    --text-muted
Accent:         #6cb6ff    --accent
Accent subtle:  #1a3a5c    --accent-subtle
Success:        #3fb950    --success
Warning:        #d29922    --warning
Danger:         #f85149    --danger
```

### Typography
- **Font stack**: `Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`
- **Monospace**: `"JetBrains Mono", "Fira Code", "SF Mono", monospace`
- **Scale**: 12px (small/labels), 14px (base), 16px (headings), 20px (page titles), 28px (stat values)
- **Line height**: 1.5 (body), 1.2 (headings), 1.0 (stat numbers)

### Spacing
- 4px grid. Spacing tokens: 4, 8, 12, 16, 20, 24, 32, 48px.

### Border Radius
- Small (inputs, badges): 4px
- Medium (cards, panels): 8px
- Large (modals): 12px

### Shadows
- None by default — rely on borders and surface color differentiation
- Elevated panels (slide-overs, modals, dropdowns): `0 8px 24px rgba(0,0,0,0.4)`

---

## Layout

```
+--------------------------------------------------+
| Top Bar (48px)                                    |
|  [=] Hero Cloud    [org picker]   [Cmd+K] [avatar]|
+--------+-----------------------------------------+
| Side   | Content Area                             |
| bar    |                                          |
| 240px  |  Page content with max-width 1400px      |
| (48px  |  centered, 24px padding                  |
|  when  |                                          |
|  coll- |                                          |
|  apsed)|                                          |
|        |                                          |
+--------+-----------------------------------------+
```

### Top Bar (48px height)
- Left: hamburger toggle (collapses sidebar), Hero logo wordmark
- Center: org picker dropdown (shows current org name, switch between orgs)
- Right: Cmd+K search trigger, user avatar + dropdown (settings, logout)

### Sidebar (240px expanded, 48px collapsed)
- Collapsible via hamburger or keyboard shortcut (`[`)
- Collapsed state shows icons only with tooltips
- Navigation sections with icons:

| Section    | Icon              | Route         |
|------------|-------------------|---------------|
| Overview   | grid/dashboard    | `#/`          |
| Specs      | document          | `#/specs`     |
| Activity   | pulse/timeline    | `#/activity`  |
| Compliance | shield-check      | `#/compliance`|
| Audit      | clipboard-list    | `#/audit`     |
| Analytics  | chart-bar         | `#/analytics` |
| Knowledge  | book-open         | `#/knowledge` |

- Bottom of sidebar: settings gear, collapse toggle
- Active state: accent background (`--accent-subtle`), accent left border (3px)

### Content Area
- Max-width: 1400px, centered
- Padding: 24px
- Page title at top (20px font, `--text`), optional subtitle/description below in `--text-muted`

---

## Shared Components

### StatCard
Rectangular card showing a KPI metric.
```
+---------------------------+
| Total Specs          [icon]|
| 247                        |
| +12 this week    ~~spark~~ |
+---------------------------+
```
- Surface: `--surface`, border: `--border`
- Title: 12px uppercase `--text-muted`
- Value: 28px bold `--text`
- Delta: 12px, colored (green for positive, red for negative)
- Optional sparkline (last 30 days) in bottom-right corner, 40px tall

### DataTable
Sortable, filterable table with fixed header.
- Header: `--elevated` background, 12px uppercase `--text-muted`, sortable columns show arrow
- Rows: `--surface` background, hover `--elevated`, 14px
- Row click: opens detail slide-over
- Pagination: bottom bar with page size selector and prev/next
- Empty state: centered illustration + message

### SlideOver
Detail panel that slides in from the right (480px wide).
- Overlay: `rgba(0,0,0,0.5)` backdrop
- Panel: `--surface` background, elevated shadow
- Header: title + close button (X or Escape key)
- Content: scrollable body
- Keyboard: Escape to close, Tab to navigate within

### FilterBar
Horizontal bar above tables/feeds with filter chips.
- Dropdown filters: type, status, repo, time range
- Active filters shown as pills with X to remove
- Search input on the right
- "Clear all" link when filters active

### Badge
Inline status/type indicator.
- Status badges: draft (gray), active/approved (blue), in-progress (yellow), delivered (green), completed (green), blocked (red)
- Type badges: feature (purple), bug (red), enhancement (blue), plan (gray)
- Pill shape, 12px font, muted background

### CommandPalette (Cmd+K)
Full-screen overlay search.
- Dark overlay with centered input (600px wide)
- Type-ahead results grouped by category (specs, repos, pages, actions)
- Keyboard navigation: up/down arrows, Enter to select, Escape to close
- Recent searches shown when empty

### Toast
Bottom-right notification.
- Auto-dismiss after 5s
- Types: success (green), error (red), info (blue)
- Stacks vertically for multiple

### EmptyState
Centered placeholder when no data.
- Icon, heading, description, optional CTA button

---

## Page Designs

### 1. Overview (`#/`)

The landing page. At-a-glance health of the engineering org.

**Layout:**
```
[Page Title: "Overview"]

[StatCard row - 4 cards]
  Total Specs | Active Specs | PR Link Rate | Compliance Score

[Two-column layout]
  Left (60%): Activity Feed (last 20 events)
  Right (40%): Spec Pipeline (status breakdown)
```

**Stat Cards:**
1. **Total Specs** — count from search endpoint (empty query, count result)
2. **Active Specs** — specs with status `in-progress` or `approved`
3. **PR Link Rate** — from `/governance/stats` → `link_rate_pct`
4. **Compliance Score** — derived from convention match rate in audit summary

**Activity Feed (mini):**
- Last 20 events from `/activity`
- Each row: avatar (from user_id), action verb, target, relative timestamp
- Event type → verb mapping: `sync` → "synced specs from", `spec.created` → "created spec", `pr.checked` → "checked PR", etc.
- "View all" link → Activity page

**Spec Pipeline:**
- Vertical bar chart or stacked horizontal bars showing spec counts by status
- Statuses: draft, approved, in-progress, delivered, completed
- Click a status bar → navigates to Specs page pre-filtered

**API endpoints used:**
- `GET /api/v1/orgs/{org}/search?q=` (total count)
- `GET /api/v1/orgs/{org}/activity` (recent events)
- `GET /api/v1/orgs/{org}/governance/stats` (PR link rate)
- `GET /api/v1/orgs/{org}/audit/summary` (event type counts)

---

### 2. Specs (`#/specs`)

Cross-repo spec browser with table and kanban views.

**Layout:**
```
[Page Title: "Specs"]  [View Toggle: Table | Board]

[FilterBar]
  Repo: [dropdown]  Type: [dropdown]  Status: [dropdown]  [Search...]

[Table View - default]
  | Slug | Title | Type | Status | Repo | Score | Updated |

[Board View - toggle]
  | Draft | Approved | In-Progress | Delivered | Completed |
  | cards | cards    | cards       | cards     | cards     |
```

**Table View:**
- Columns: slug (monospace, clickable), title, type badge, status badge, repo name, score (if present), updated_at (relative)
- Sort by any column (click header)
- Row click → SlideOver with full spec detail

**Board View (Kanban):**
- 5 columns: draft, approved, in-progress, delivered, completed
- Each card: title, slug (small mono), type badge, repo name
- Cards sorted by updated_at within columns
- No drag-and-drop (read-only dashboard)

**Spec SlideOver:**
```
+-- Spec Detail -------------------+
| cloud-dashboard-ui           [X] |
| Hero Cloud Dashboard UI          |
|                                  |
| Status: [draft]  Type: [feature] |
| Repo: hero       Score: 85       |
| Priority: high                   |
| Tracker: HERO-142                |
|                                  |
| Tags: cloud, dashboard, frontend |
|                                  |
| --- Raw Content (collapsible) ---|
| (markdown rendered)              |
+----------------------------------+
```

**API endpoints used:**
- `GET /api/v1/orgs/{org}/repos` (repo list for filter dropdown)
- `GET /api/v1/orgs/{org}/repos/{repo}/specs?type=&status=` (per-repo, filtered)
- `GET /api/v1/orgs/{org}/search?q=` (cross-repo search)
- `GET /api/v1/orgs/{org}/repos/{repo}/specs/{slug}` (detail, includes raw_content)

**Note:** The current API lists specs per-repo. The dashboard needs to aggregate across repos. Two options:
1. Client-side: fetch all repos, then fetch specs for each repo, merge client-side
2. New endpoint: `GET /api/v1/orgs/{org}/specs` that queries across repos (preferred, add to API)

**New endpoint needed:**
```
GET /api/v1/orgs/{org_id}/specs?type=&status=&repo=&q=&sort=&limit=&offset=
```
Returns specs across all repos in the org with filtering. This avoids N+1 repo queries from the frontend.

---

### 3. Activity (`#/activity`)

Chronological event feed.

**Layout:**
```
[Page Title: "Activity"]

[FilterBar]
  Event Type: [dropdown]  Repo: [dropdown]  Time: [range picker]

[Activity Feed - full width]
  [avatar] user.name verb target — 2 hours ago
  [avatar] user.name verb target — 3 hours ago
  ...

[Load More button]
```

**Event rendering:**
Each event type maps to a human-readable sentence:

| event_type          | Rendered as                                    |
|---------------------|------------------------------------------------|
| sync                | "synced 12 specs from **hero**"                |
| spec.created        | "created spec **cloud-dashboard-ui**"          |
| spec.synced         | "synced spec **cloud-auth** (feature, draft)"  |
| spec.delivered      | "delivered spec **batch-diagnosis**"            |
| pr.checked          | "checked PR #42 on **org/repo** — linked"      |
| convention.matched  | "convention **naming-conventions** matched in PR #42" |
| scope.drift         | "scope drift detected in PR #42 for **export-csv**" |

- Avatar: generated from user_id (initials in colored circle, or GitHub avatar if available)
- Timestamp: relative ("2 hours ago"), full timestamp on hover
- Payload details expandable on click

**API endpoints used:**
- `GET /api/v1/orgs/{org}/activity?limit=50&offset=0`
- Pagination via offset, "Load More" increments offset

**New endpoint needed:**
Activity endpoint should support filtering by event_type and time range (similar to audit endpoint). Currently it only supports limit/offset.

---

### 4. Compliance (`#/compliance`)

Convention compliance overview across repos.

**Layout:**
```
[Page Title: "Compliance"]

[StatCard row - 3 cards]
  Active Conventions | PRs Checked | Overall Compliance %

[Convention Table]
  | Convention | Scope | Repos | Matches | Last Matched |

[Per-Repo Compliance]
  | Repo | Conventions Active | PRs Checked | Link Rate | Compliance |
```

**Convention Table:**
- Lists all active conventions from the org
- Scope column: glob patterns as code chips
- Matches: count of convention.matched audit events for this convention
- Click → SlideOver with convention content (rendered markdown)

**Per-Repo Compliance:**
- Aggregated view: for each repo, how many conventions are active, how many PRs were checked, what % had spec links
- Compliance % = (PRs with spec + no violations) / total PRs checked

**Data derivation:**
- Convention list: not directly exposed via API yet. Need new endpoint.
- Match counts: derived from audit events with `event_type = convention.matched`
- PR stats: from `/governance/stats` (org-level only currently)

**New endpoints needed:**
```
GET /api/v1/orgs/{org_id}/conventions
GET /api/v1/orgs/{org_id}/repos/{repo_id}/governance/stats
```

---

### 5. Audit (`#/audit`)

Filterable audit trail for compliance/SOC2 export.

**Layout:**
```
[Page Title: "Audit Trail"]    [Export CSV]

[FilterBar]
  Event Type: [multi-select]  Since: [date]  Until: [date]

[Audit Table]
  | Timestamp | Event | User | Repo | Details |

[Pagination: < 1 2 3 ... 10 >]
```

**Table:**
- Timestamp: ISO 8601 in monospace, sorted descending
- Event: badge with event_type (e.g., "spec.synced", "pr.checked")
- User: user_id (display name if available)
- Repo: repo_id (name if available)
- Details: truncated JSON payload, expandable on row click

**Export CSV:**
- Button triggers download via `/audit?format=csv` with current filters applied
- Opens in new tab / triggers download

**API endpoints used:**
- `GET /api/v1/orgs/{org}/audit?types=&since=&until=&limit=&offset=&format=csv`
- `GET /api/v1/orgs/{org}/audit/summary` (for stat cards at top)

---

### 6. Analytics (`#/analytics`)

The CTO sale page. Metrics that prove AI-driven engineering works.

**Layout:**
```
[Page Title: "Analytics"]   [Time Range: 7d | 30d | 90d | Custom]

[StatCard row - 4 cards]
  Specs Delivered | Avg Time-to-Merge | Rework Rate | AI Leverage

[Two-column layout]
  Left: Delivery Velocity (time-series chart)
  Right: Spec Fidelity (bar chart)

[Full-width]
  Spec Pipeline Funnel (horizontal funnel chart)

[Two-column layout]
  Left: Quality Score Distribution (histogram)
  Right: Activity Heatmap (GitHub-style contribution grid)
```

**Metrics derivation (all from existing audit events):**

1. **Specs Delivered** — count of `spec.delivered` events in time range
2. **Avg Time-to-Merge** — time between `spec.approved` and `spec.merged` events for same slug
3. **Rework Rate** — specs with multiple `spec.delivered` events / total delivered
4. **AI Leverage** — ratio of agent-delivered vs manually-delivered specs (from `claimed_by` field or payload data)
5. **Delivery Velocity** — time-series of `spec.delivered` events per week
6. **Spec Fidelity** — specs that went from delivered → completed without rework
7. **Pipeline Funnel** — count at each status: draft → approved → in-progress → delivered → completed
8. **Quality Score Distribution** — histogram of spec `score` values
9. **Activity Heatmap** — daily event counts for last 52 weeks (GitHub contribution style)

**Charts (Chart.js):**
- Time-series: line chart with filled area, accent color
- Bar charts: horizontal bars, accent color with opacity variants
- Funnel: custom horizontal stacked bars narrowing
- Heatmap: CSS grid of colored squares (no Chart.js needed, pure CSS/HTML)

**New endpoints needed:**
The analytics page needs aggregated metrics. Rather than computing everything client-side from raw audit events, add:
```
GET /api/v1/orgs/{org_id}/analytics/overview?since=&until=
  → { specs_delivered, avg_time_to_merge_hours, rework_rate, ai_leverage_pct }

GET /api/v1/orgs/{org_id}/analytics/velocity?since=&until=&interval=week
  → { buckets: [{ period: "2026-W15", delivered: 4, created: 7 }] }

GET /api/v1/orgs/{org_id}/analytics/heatmap?since=&until=
  → { days: [{ date: "2026-04-19", count: 12 }] }
```

---

### 7. Knowledge (`#/knowledge`)

Searchable knowledge base — conventions, decisions, context.

**Layout:**
```
[Page Title: "Knowledge"]

[Search bar - full width, prominent]

[Tab bar: Conventions | Decisions | All]

[Knowledge Items Grid/List]
  +--card--+  +--card--+  +--card--+
  | Conv:  |  | Conv:  |  | Dec:   |
  | naming |  | error  |  | auth   |
  | patt.. |  | handl..|  | flow   |
  +--------+  +--------+  +--------+
```

**Conventions Tab:**
- Grid of cards, each showing: title, scope globs, repo name, status badge
- Click → SlideOver with full convention content (markdown rendered)
- Filter by repo

**Decisions Tab:**
- Placeholder for future ADR (Architectural Decision Record) sync
- Shows "Coming soon" empty state for now

**Search:**
- Full-text search across convention titles and content
- Uses existing `/search` endpoint (currently searches specs, could extend to knowledge)

**API endpoints needed:**
```
GET /api/v1/orgs/{org_id}/conventions?repo=&q=
GET /api/v1/orgs/{org_id}/knowledge/search?q=
```

---

## Keyboard Shortcuts

| Shortcut   | Action                        |
|------------|-------------------------------|
| `Cmd+K`    | Open command palette          |
| `[`        | Toggle sidebar                |
| `Escape`   | Close slide-over / palette    |
| `g o`      | Go to Overview                |
| `g s`      | Go to Specs                   |
| `g a`      | Go to Activity                |
| `g c`      | Go to Compliance              |
| `g u`      | Go to Audit                   |
| `g n`      | Go to Analytics               |
| `g k`      | Go to Knowledge               |
| `/`        | Focus search (on pages with search) |
| `j` / `k`  | Navigate table rows           |
| `Enter`    | Open selected item            |

---

## Authentication Flow

1. User hits `cloud.herospec.dev` → SPA loads
2. JS checks for JWT in `localStorage`
3. If no token → redirect to login page (`#/login`)
4. Login page has "Sign in with GitHub" button → `/api/v1/auth/github` OAuth flow
5. OAuth callback sets JWT in localStorage, redirects to `#/`
6. All API calls include `Authorization: Bearer <token>` header
7. On 401 response → clear token, redirect to login
8. Token refresh: before expiry, call `/api/v1/auth/refresh`

---

## File Structure

```
cloud/
  web/
    index.html          — shell HTML, loads app.js
    style.css           — all styles (CSS custom properties + utility classes)
    app.js              — main application (Preact + HTM, all components)
    lib/
      preact.module.js  — vendored Preact (avoid CDN dependency in prod)
      htm.module.js     — vendored HTM
      chart.min.js      — vendored Chart.js
  web.go                — embed.FS declaration
  api/
    router.go           — add SPA fallback handler
    dashboard.go        — new analytics/aggregation endpoints
```

**Single-file approach for v1:** Put all components in `app.js` (~2000-3000 lines). Split into modules later if needed. This avoids needing any module bundler while keeping the codebase navigable with clear section comments.

---

## New API Endpoints Required

Summary of endpoints the dashboard needs that don't exist yet:

| Endpoint | Purpose | Priority |
|----------|---------|----------|
| `GET /orgs/{id}/specs` | Cross-repo spec listing with filters | P0 |
| `GET /orgs/{id}/conventions` | List conventions for org | P0 |
| `GET /orgs/{id}/analytics/overview` | Aggregated KPI metrics | P1 |
| `GET /orgs/{id}/analytics/velocity` | Time-bucketed delivery counts | P1 |
| `GET /orgs/{id}/analytics/heatmap` | Daily activity counts | P2 |
| `GET /orgs/{id}/repos/{id}/governance/stats` | Per-repo PR check stats | P2 |
| Activity filtering by event_type, time range | Enhanced activity feed | P1 |
| `GET /orgs/{id}/knowledge/search` | Cross-knowledge-type search | P2 |

---

## Implementation Plan

### Phase 1: Shell + Overview (3-4 days)
1. Set up `cloud/web/` directory with `index.html`, `style.css`, `app.js`
2. Implement `web.go` embed and SPA fallback in `router.go`
3. Build layout shell: top bar, sidebar, routing
4. Implement design system: CSS custom properties, component primitives (StatCard, Badge, DataTable)
5. Build Overview page with stat cards and activity feed
6. Wire up auth flow (login page, token management, API client)

### Phase 2: Specs + Activity (2-3 days)
1. Add cross-repo spec endpoint to API (`GET /orgs/{id}/specs`)
2. Build Specs page: table view, filter bar, search
3. Build spec detail SlideOver
4. Build kanban board view toggle
5. Build Activity page with event feed
6. Enhance activity endpoint with filtering

### Phase 3: Compliance + Audit (2 days)
1. Add conventions endpoint to API
2. Build Compliance page: convention table, per-repo stats
3. Build Audit page: filterable table, CSV export button
4. Build time range picker component

### Phase 4: Analytics + Knowledge (2-3 days)
1. Add analytics aggregation endpoints to API
2. Build Analytics page: stat cards, Chart.js charts
3. Build activity heatmap (CSS grid)
4. Build Knowledge page: convention browser, search
5. Implement Cmd+K command palette

### Phase 5: Polish (1-2 days)
1. Keyboard shortcuts (vim-style navigation, go-to shortcuts)
2. Loading states, error states, empty states
3. Responsive tweaks for narrower viewports
4. Performance: debounced search, pagination, lazy chart loading

**Total estimate: 10-14 days**

---

## Acceptance Criteria

1. All 7 pages render with real data from the API
2. SPA is embedded in Go binary and served without external dependencies
3. Auth flow works end-to-end (GitHub OAuth → JWT → API calls)
4. Tables are sortable and filterable
5. Spec detail slide-over shows full spec information
6. Audit export produces valid CSV download
7. Analytics charts render with Chart.js
8. Cmd+K command palette navigates to pages and searches specs
9. Keyboard shortcuts work (sidebar toggle, page navigation, table navigation)
10. No build toolchain required — `go build` produces a working binary with the UI included
11. Page load under 500ms on fast connection (small JS payload, no framework bloat)

---

## Decisions

1. **User display names**: Embed `display_name` and `avatar_url` in event payload at write time. Simpler queries, faster reads.
2. **Real-time updates**: Server-Sent Events (SSE) for activity feed. Endpoint: `GET /api/v1/orgs/{id}/activity/stream`. Falls back gracefully if connection drops.
3. **Multi-org**: Yes, org picker needed — users can belong to multiple orgs via `org_members`.
4. **Spec content rendering**: Vendor `marked.js` (~7KB). Spec detail SlideOver has tab pills: **Rendered** | **Source** — toggle between rendered markdown and raw source with syntax highlighting.
5. **Theme**: Both dark and light mode from day one. CSS custom properties for all colors. Toggle in top bar. Persist preference in localStorage. Default: dark.

### Light Mode Palette
```
Background:     #f6f8fa    --bg
Surface:        #ffffff    --surface
Elevated:       #f0f2f5    --elevated
Border:         #d1d9e0    --border
Text primary:   #1f2328    --text
Text secondary: #636c76    --text-muted
Accent:         #0969da    --accent
Accent subtle:  #ddf4ff    --accent-subtle
Success:        #1a7f37    --success
Warning:        #9a6700    --warning
Danger:         #d1242f    --danger
```
