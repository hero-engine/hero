---
title: "Local Dashboard UI for Hero Serve"
type: feature
status: completed
created: 2026-04-13
tags: [serve, ui, dashboard]
horizon: now
---

# Local Dashboard UI for Hero Serve

## Goal

Add a browser-based dashboard to `hero serve` that gives engineers and leads a visual overview of their Hero workspace — specs in flight, bug inventory, knowledge base, and live activity. The dashboard is served from the existing HTTP daemon at `localhost:7437` and requires no separate process or build step.

## Background

`hero serve` already exposes a full REST API and SSE event stream. The CLI provides `hero dashboard` and `hero status` for terminal use, but a visual interface adds value for:

- **Bug triage** — the inventory report as a filterable, sortable table instead of a markdown file
- **Sprint awareness** — specs in columns by status, with who's working on what
- **Knowledge discovery** — browsing conventions, decisions, and notes is easier visually
- **Activity monitoring** — live feed of spec changes, auto-refresh events, and agent activity

## Design

### Technology choice: embedded SPA with Go `embed`

The dashboard is a single-page application built with vanilla JS + a lightweight CSS framework (no React, no build step). The compiled HTML/CSS/JS is embedded into the Go binary via `//go:embed` so there's zero runtime dependency — `hero serve` just serves it.

**Why not htmx?** htmx would require server-side HTML rendering in Go templates, adding complexity to the API layer that's already JSON-based. A thin JS client consuming the existing JSON API is simpler.

**Why not React/Vue?** Adds a build step, node_modules, and bundler complexity for what is fundamentally a read-heavy dashboard. Vanilla JS with `fetch()` and template literals is sufficient.

### Pages

#### 1. Overview (`/`)

The landing page. Shows at a glance:

- Project name and health status (from `/api/{project}/check`)
- Spec counts by status (planning / in-review / delivering / completed) as cards or a bar
- Stale specs and unclaimed work warnings
- Recent activity feed (SSE events: spec created/modified/completed, auto-refresh results)

Data sources: `/api/{project}/status`, `/api/{project}/check`, SSE `/api/events`

#### 2. Board (`/board`)

Kanban-style columns for work specs:

| Planning | In Review | Delivering | Completed |
|----------|-----------|------------|-----------|

Each card shows: spec title, type badge (feature/bug), claimed_by, age. Click to expand inline or navigate to the spec detail view.

Filters: type (feature/bug/initiative), tag, claimed_by.

Data source: `/api/{project}/specs?type=feature` + `/api/{project}/specs?type=bug`

#### 3. Bug Inventory (`/bugs`)

The inventory report rendered as an interactive table:

| ID | Title | Severity | Reporter | Created | Age | Status |
|----|-------|----------|----------|---------|-----|--------|

Sortable by any column. Filterable by severity, age, status. Click an ID to see the full spec or link to the tracker issue.

This replaces the static `inventory.md` for triage purposes. The data comes from either:
- The local specs (imported bugs in `.hero/planning/bugs/`)
- A new API endpoint `/api/{project}/inventory` that reads the inventory report or queries specs directly

Data source: `/api/{project}/specs?type=bug` (new: include tracker_id, priority, created in spec response)

#### 4. Spec Detail (`/spec/{slug}`)

Rendered markdown view of a single spec:

- Frontmatter displayed as a metadata sidebar (type, status, claimed_by, tracker_id, tags, relations)
- Rendered markdown body (use a lightweight markdown renderer — marked.js or similar, embedded)
- Link to tracker issue if tracker_id exists
- Status timeline if we track status changes (future)

Data source: `/api/{project}/specs/{slug}`

#### 5. Knowledge (`/knowledge`)

Tree/list view of the knowledge base:

- Grouped by type: conventions, decisions, rules, context, notes
- Each entry shows title, status (active/draft/superseded), tags
- Click to view rendered content
- Search bar that queries the full-text index

Data source: `/api/{project}/knowledge`, `/api/{project}/search?q=...`

#### 6. Search (`/search`)

Full-text search across all specs and knowledge entries:

- Search bar with instant results
- Results grouped by type
- Highlight matching terms (if index supports snippets)

Data source: `/api/{project}/search?q=...`

### Layout

- Sidebar navigation (collapsible): Overview, Board, Bugs, Knowledge, Search
- Top bar: project selector (multi-project support), health indicator
- Content area: the active page
- Footer: connection status (SSE connected/disconnected), last refresh time

### Live updates via SSE

The dashboard connects to `/api/events` and updates in real-time:

- New spec created → appears on the board
- Spec status changed → card moves to new column
- Auto-refresh event → bug inventory updates
- Index rebuilt → search results refresh

### File structure

```
internal/serve/
  ui/                     # embedded SPA
    index.html            # single HTML file with inline CSS
    app.js                # application logic
    style.css             # styles (or inline in HTML)
  embed.go                # //go:embed directive
  server.go               # updated to serve UI at /
```

### API additions needed

1. **`GET /api/{project}/specs`** — extend spec response to include `tracker_id`, `priority`, `created`, `tags` (currently missing from `specSummary`)
2. **`GET /api/{project}/inventory`** — returns the inventory report data as JSON (not markdown), or reads from imported bug specs with enriched fields
3. **`GET /`** — serve the embedded SPA `index.html`
4. **`GET /app.js`**, **`GET /style.css`** — serve embedded static assets

### Configuration

```json
{
  "serve": {
    "ui": true,
    "port": 7437
  }
}
```

`serve.ui` defaults to `true`. Set to `false` to disable the dashboard (API-only mode).

## Changes

### Go files

- `internal/serve/ui/index.html` — new: dashboard HTML shell
- `internal/serve/ui/app.js` — new: SPA application logic
- `internal/serve/ui/style.css` — new: dashboard styles
- `internal/serve/embed.go` — new: `//go:embed ui/*` directive
- `internal/serve/server.go` — add static file serving at `/` for embedded UI
- `internal/serve/api.go` — extend `specSummary` with tracker_id, priority, created, tags; add inventory endpoint
- `internal/config/config.go` — add `UI bool` to `ServeConfig`

### No external dependencies

All JS and CSS is vanilla or embedded from a CDN-free source. No npm, no build step, no node_modules.

## Acceptance Criteria

1. `hero serve` opens a browser-usable dashboard at `http://localhost:7437/`
2. Overview page shows spec counts, health warnings, and live activity
3. Board page shows specs in kanban columns by status
4. Bug inventory page shows imported bugs as a sortable/filterable table
5. Spec detail page renders markdown with metadata sidebar
6. Knowledge page lists entries grouped by type with search
7. SSE events update the UI in real-time without page refresh
8. Dashboard works with no external network requests (all assets embedded)
9. `--ui=false` flag or `serve.ui: false` disables the dashboard
10. Multi-project support: project selector switches between registered projects

## Risks

- **Markdown rendering in vanilla JS** — need a lightweight embedded renderer. marked.js is 50KB minified, acceptable to embed.
- **CSS framework size** — need to keep it minimal. Consider Pico CSS (~10KB) or hand-rolled styles.
- **Go embed binary size** — HTML/CSS/JS adds a few hundred KB to the binary. Acceptable.
- **Browser compatibility** — target modern browsers only (Chrome, Firefox, Safari, Edge). No IE11.

## Notes

This spec is for the initial dashboard. Future iterations could add:
- Agent activity monitoring (if agents report through the API)
- Drag-and-drop spec prioritization on the board
- Inline spec editing
- Diff viewer (spec vs. git changes)
- Cost tracking visualization
- Sprint burndown from tracker data
