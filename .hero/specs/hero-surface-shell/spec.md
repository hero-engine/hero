---
title: Hero Surface Shell — Slim Top Nav, Page Routing, Shared Fragments
slug: hero-surface-shell
type: feature
status: completed
tags: [serve, surface, shell, chrome, routing, ui, web-app]
created: 2026-05-17
relations:
  - target: hero-surface-architecture
    kind: parent
  - target: hero-surface-deployment-and-rendering
    kind: depends-on
  - target: hero-chat-and-model
    kind: relates-to
horizon: now
---

## Context

The shipped dashboard
([internal/serve/ui/index.html](../../../../internal/serve/ui/index.html)
+ `app.js` + `style.css`) is a static three-tab workspace browser. It
opens to project stats. It is not an operating surface, and it does
not have any room to grow into one as the
[hero-surface-architecture](../../initiatives/hero-surface-architecture/spec.md)
initiative lands its five top-level homes (Now, Work, Knowledge,
Agents, People).

This spec replaces that v1 dashboard with the **thin runtime** that
all five homes ride on. The shell is small on purpose: a slim top
nav, a router, a handful of shared page-fragment templates, the
mount point for the ⌘K overlay island, edition gating at the route
layer, and a per-user session store that remembers where you were.

**Hero serve is a web app companion to the CLI** — opened in a
browser tab to deep-dive into a project beyond what one CLI command
can show. It is not a desktop workspace. Earlier shell drafts in
this folder modeled the chrome on the PM/QA desktop packs (fixed
left rail + VS Code tab strip + bottom verb strip + permanent right
ambient rail + role-switcher pill row + a view registry / pack
abstraction). That grammar was rejected. PM/QA is a different
product (end-user desktop apps) and out of scope for hero serve.
Engineering is the only "pack" — which means it isn't a pack at all,
it's just the hero serve UI. **No view registry. No pack
extensibility. No fixed rails. No tab strip.**

The locked visual source of truth is
[hero-now-home/mockups/01-now-default.html](../hero-now-home/mockups/01-now-default.html).
The other homes
([Work](../hero-work-home/mockups/01-work-roadmap.html),
[Agents](../hero-agents-home/mockups/01-agents-sessions.html))
demonstrate the optional sub-nav row pattern this shell supports.

## Goal

Hero serve, on startup, brings up a thin web shell that:

1. **Renders the slim top nav** (~56px) on every page route — Hero
   bolt mark + workspace name, the five top-nav text-link tabs with
   hero-blue underline on active, the ⌘K search pill, workspace
   state, avatar.
2. **Routes the five home pages and their per-item children** under
   one `net/http` server. Each home owns its handlers and templates;
   the shell wires routes and provides the outer layout.
3. **Provides shared page-fragment templates** that any home can
   compose: page-hero, tabbed-metric-strip, sub-nav row, footer,
   chat-input, empty-state-notice.
4. **Mounts the ⌘K command-bar island** on every page so the hotkey
   works anywhere. The island itself is owned by
   [hero-chat-and-model](../hero-chat-and-model/spec.md); the shell
   guarantees it is loaded and bound.
5. **Filters routes by edition.** `HERO_EDITION` (per
   [hero-surface-deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md))
   determines which routes register and which top-nav tabs render.
6. **Persists per-user session state** in SQLite so that reopening
   hero serve returns the user to the home they were last on (and to
   the same in-home tab where the home supports it).

When this lands, the five home specs can each focus on their own
content — page-hero copy, sections, tile rows, lists — and inherit
chrome, routing, and the shared fragments from this shell.

## Approach

### Top nav

A single Go template, served by the layout wrapper on every route.
Sticky at the top, max-width ~1200px centered with 32px horizontal
padding, height 56px, 1px bottom border in `--border`.

```html
<nav class="topnav">
  <div class="topnav-inner">
    <a href="/" class="brand">
      <svg class="bolt" viewBox="0 0 90 90" fill="currentColor"
           aria-hidden="true">
        <path d="M52 8 L22 46 L40 46 L34 82 L68 42 L50 42 Z"/>
      </svg>
      Hero
      <span class="workspace">{{ .Workspace }}</span>
    </a>

    <div class="nav-tabs">
      {{ range .Tabs }}
        <a href="{{ .Href }}"
           class="nav-tab{{ if .Active }} active{{ end }}">
          {{ .Label }}
          {{ if .Count }}<span class="count">{{ .Count }}</span>{{ end }}
        </a>
      {{ end }}
    </div>

    <div class="topnav-right">
      <a href="#" class="search-pill"
         data-command-bar-trigger
         aria-label="Open command bar">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none"
             stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/>
        </svg>
        Search specs, code, knowledge…
        <span class="kbd">⌘K</span>
      </a>
      <span class="workspace-state">
        <span class="dot"></span>{{ .Branch }}
      </span>
      <div class="avatar" title="{{ .UserName }}">{{ .UserInitials }}</div>
    </div>
  </div>
</nav>
```

Tabs are produced by the router (see below): each registered home
contributes one tab record `{Slug, Label, Href, Count, Active,
Editions}`. The router filters by `HERO_EDITION` before passing the
slice to the template. The active tab is matched by URL prefix
(`/work` highlights `Work`, `/work/spec/<slug>` likewise). Counts
(`Work 18`, `Agents 1`) are supplied per-home as small data
fetchers registered alongside the route — null counts are simply
omitted.

The visual contract (colors, spacing, underline geometry, brand
mark, font stack, focus ring) is copied verbatim from the Now mock
so designers can keep iterating in the HTML mock and engineering
keeps parity.

### Routing

Standard library `net/http` with `http.ServeMux` (Go 1.22+ method
syntax). The shell exposes one `Router` type:

```go
type Router struct {
    mux     *http.ServeMux
    homes   []Home   // ordered; defines the top-nav tab order
    edition edition.Edition
}

type Home struct {
    Slug     string                          // "now", "work", "knowledge", "agents", "people"
    Label    string                          // "Now", "Work", ...
    Href     string                          // "/now", "/work", ...
    Count    func(ctx Context) (int, bool)   // optional badge fetcher
    Render   http.HandlerFunc                // handler for the home root
    Items    []ItemRoute                     // per-item child routes
    Editions []string                        // edition filter; empty = all
}

type ItemRoute struct {
    Pattern string                  // "GET /work/spec/{slug}"
    Render  http.HandlerFunc
}
```

Each home spec calls `shell.RegisterHome(Home{...})` at server
startup. The shell calls the home's `Render` for the home root,
mounts each `ItemRoute.Pattern`, and slots the home into the
top-nav tab list. Per-item routes (`/work/spec/<slug>`,
`/knowledge/why?target=...`, `/agents/session/<id>`,
`/people/roi`, etc.) are conventional `net/http` patterns. There
is no SPA history magic — back button is browser-native.

The router resolves `/` to a redirect to the user's last-visited
home (see [Session state](#session-state)), defaulting to `/now`.

All home handlers compose their page through `shell.RenderPage`:

```go
shell.RenderPage(w, r, shell.Page{
    ActiveHome: "work",
    PageTitle:  "Work · Hero",
    Content:    func(w io.Writer) { workTemplate.Execute(w, data) },
})
```

`RenderPage` wraps the content with the top nav, optional sub-nav
slot, page container, and footer. Nothing more.

### Shared page fragments

Each fragment is a parameterized Go template under
`internal/serve/shell/templates/`. Home pages compose them inside
their own templates via `{{ template "..." . }}` calls so the
fragments stay server-side and SSE-swappable.

**`page-hero.html`** — the eyebrow + title + subhead + inline
action row pattern at the top of every home. Parameters:

```go
type PageHero struct {
    Eyebrow template.HTML  // "hero · main · solo edition" with optional bolt-sm
    Title   string         // 32px, weight 600
    Subhead template.HTML  // strong/dot-sep markup permitted
    Actions []PageHeroAction
}

type PageHeroAction struct {
    Kind  string         // "primary" | "ghost" | "chip"
    Label string
    Href  string         // optional
    Icon  template.HTML
    Chip  string         // "Solo" etc. for kind="chip"
}
```

**`tabbed-metric-strip.html`** — the text-link tabs above a
swappable tile row. Each home declares its tabs and the tiles each
tab contains. The strip wires the tab-click → pane-swap behavior
client-side using the same vanilla JS pattern locked in the Now
mock (no framework).

```go
type MetricStrip struct {
    Tabs    []MetricTab
    AllLink string  // "View all metrics →"
}

type MetricTab struct {
    Slug   string         // "sprint", "week", "roi"
    Label  string
    Active bool
    Tiles  []MetricTile
}

type MetricTile struct {
    Value  template.HTML  // includes <span class="unit"> etc.
    Label  string
    Footer template.HTML  // sparkline SVG, seg-bar, progress, sub-text
    Accent string         // optional: "warn" colors the value
}
```

**`sub-nav.html`** — the second row of text-link tabs (`Sessions`,
`Proposals`, `Scheduled`, …) used by Knowledge / Agents / People for
in-home navigation. Same underline idiom as the top nav, smaller
(44px tall, 13px font). The shell does **not** render this on its
own; the home decides whether to include it. The shell ships the
template so each home is consistent.

```go
type SubNav struct {
    Tabs []SubNavTab
}

type SubNavTab struct {
    Label    string
    Href     string
    Active   bool
    Badge    string  // optional count
    Variant  string  // "" | "amber" | "locked"
    LockMeta string  // edition gating hint when Variant == "locked"
}
```

**`footer.html`** — the quiet text-link row at the end of scroll.
Workspace label + version + edition on the left; `Docs`, `GitHub`,
`Status`, `⌘K` on the right. Static; no parameters beyond version
and edition strings.

**`chat-input.html`** — the big rounded input used by Now's quick
launch (full-width 64px-tall) and by the ⌘K overlay (the overlay
calls it in a narrower variant). Owned conceptually by
[hero-chat-and-model](../hero-chat-and-model/spec.md) — the shell
just provides the template file at this stable path so that spec's
JS can hydrate it consistently. Parameters: `Variant` (`hero` |
`overlay` | `inline`), `Placeholder`, `Context` (page + artifact
chips to render below).

**`empty-state-notice.html`** — the no-adapter CTA from
chat-and-model ("Hero needs hero-code …"). Lives in the shell
template directory so it can render above the chat-input fragment
without circular template references.

**`page-layout.html`** — the outer wrapper any home page uses.
Slots: top-nav (always rendered), sub-nav (optional, page supplies
the `SubNav` struct), page container (max-width 1200px, 32px
horizontal padding), footer (always rendered).

### Edition gating

The shell's `Router` accepts an `edition.Edition` value (resolved
from `HERO_EDITION` per the
[deployment-and-rendering](../hero-surface-deployment-and-rendering/spec.md)
decision: `local | team | cloud | enterprise | ce`). When
`RegisterHome` is called:

- If `Home.Editions` is non-empty and does not include the active
  edition, the home is skipped entirely — no top-nav tab, no route
  registered. The user gets a 404 if they try the URL directly.
- If `Home.Editions` is empty, the home is available in every
  edition.

Per-item routes inherit their parent home's edition gate. A future
need for per-route gating below the home level can be added by
giving `ItemRoute` its own `Editions` field; for now,
home-granularity matches the matrix in the deployment-and-rendering
decision.

The `edition` package exposes one resolver and a tiny middleware
for downstream code that wants to assert a minimum edition:

```go
edition.Resolve()                              // reads HERO_EDITION; default "local"
edition.Require("team", "cloud", "enterprise") // 404 if not in set
```

### Session state

Per-user shell state lives in SQLite, in the existing hero serve
database. One table:

```sql
CREATE TABLE shell_sessions (
    user_id        TEXT PRIMARY KEY,
    last_home      TEXT NOT NULL,              -- "now" | "work" | ...
    home_tab_state TEXT NOT NULL DEFAULT '{}', -- JSON: { "agents": "proposals" }
    updated_at     INTEGER NOT NULL            -- unix ms
);
```

A small `session` package wraps this:

```go
session.LastHome(userID) (slug string, ok bool)
session.SetLastHome(userID, slug string) error
session.TabState(userID, home string) (tab string, ok bool)
session.SetTabState(userID, home, tab string) error
```

The router calls `SetLastHome` on every successful home render
(home-root only, not per-item pages — we don't want to deep-link
back into a specific spec). Sub-nav home pages call `SetTabState`
when the user picks a sub-nav tab; on next mount of the home, the
handler reads `TabState` to pick the initial pane.

`/` redirects to `LastHome` (falling back to `/now`).

Per-user identity is whatever the auth layer already supplies. In
`local` edition this is the local OS user; in `team`/`cloud` it is
the authenticated user id. The shell does not implement identity —
it consumes it via a `User(r *http.Request) string` helper that
already exists in
[internal/serve/users.go](../../../../internal/serve/users.go).

The store is intentionally tiny. If session-state needs grow
(multi-device sync, per-device state, replay), this layer is small
enough to swap.

### Theme + style tokens

Lifted verbatim from the Now mock so engineering and design stay
in lock-step. The full token block lives in
`internal/serve/shell/static/shell.css`:

```css
:root {
  --hero-blue-300: #9bc1e6;
  --hero-blue-500: #6cb6ff;
  --hero-blue-700: #2a6cb5;
  --hero-blue-900: #1a4a7d;
  --hero-blue-50:  #eff6ff;

  --hero-ink: #14181e;
  --ink-2: #2a2f37;
  --ink-3: #5a626d;
  --ink-4: #8a929c;
  --ink-5: #b4bac3;

  --bg: #ffffff;
  --bg-soft: #fafbfc;
  --bg-softer: #f5f7fa;
  --bg-panel: #f7f8fa;
  --border: #eef1f5;
  --border-strong: #e3e7ee;

  --success: #16a249;
  --warn: #d97706;
  --danger: #dc2626;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 14px;
  color: var(--ink-2);
  background: var(--bg);
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
}
```

`shell.css` carries: tokens, body baseline, anchor + button
baselines, top-nav, page container, page-hero, tabbed-metric-strip,
sub-nav, footer, status-chip family, button family. Anything
home-specific lives in the home's own CSS. There is no top-level
build step; the file is served directly.

`favicon.svg` is the bolt mark with `fill="currentColor"` and the
hero-blue-700 set as a wrapper attribute.

## Changes

1. `internal/serve/shell/shell.go` — `Router` type, `Home`,
   `ItemRoute`, `Page`, `RegisterHome`, `RenderPage`. Wires
   `http.ServeMux`, applies edition filter, builds the top-nav
   tab slice each request, redirects `/` to last-visited home.
2. `internal/serve/shell/templates/top-nav.html` — the slim top
   nav exactly as sketched above, served on every route via the
   page layout.
3. `internal/serve/shell/templates/page-hero.html` —
   parameterized fragment; takes `PageHero` struct.
4. `internal/serve/shell/templates/tabbed-metric-strip.html` —
   parameterized fragment; takes `MetricStrip` struct; ships the
   small vanilla JS that swaps panes on tab click (same script
   block as the Now mock).
5. `internal/serve/shell/templates/sub-nav.html` — parameterized
   fragment; takes `SubNav` struct.
6. `internal/serve/shell/templates/footer.html` — static fragment
   with version + edition strings injected.
7. `internal/serve/shell/templates/chat-input.html` — shared
   template, three variants (`hero`, `overlay`, `inline`).
   Hydration belongs to
   [hero-chat-and-model](../hero-chat-and-model/spec.md); shell
   owns the stable template path only.
8. `internal/serve/shell/templates/empty-state-notice.html` — the
   no-adapter CTA from chat-and-model.
9. `internal/serve/shell/templates/page-layout.html` — outer
   wrapper: `<!doctype html>` → head (favicon, shell.css, page-
   specific `<link>`s via `Page.HeadExtra`) → body → top-nav →
   optional sub-nav → page container with `Content` slot →
   footer → command-bar island
   `<script type="module" src="/static/islands/command-bar.js">`.
10. `internal/serve/shell/static/shell.css` — token block plus all
    chrome / shared-fragment styles. Imported by
    `page-layout.html`.
11. `internal/serve/shell/static/favicon.svg` — bolt mark.
12. `internal/serve/session/session.go` — `LastHome`,
    `SetLastHome`, `TabState`, `SetTabState`. SQLite-backed via
    the existing serve db handle.
13. `internal/serve/session/schema.sql` — `shell_sessions` table
    creation; wired into the existing migration runner.
14. `internal/serve/edition/edition.go` — `Edition` type,
    `Resolve()` (reads `HERO_EDITION`), `Require(...)`
    middleware, `Allowed(home Home) bool` helper used by the
    router filter.
15. `internal/serve/server.go` — replace the existing v1 dashboard
    mount with `shell.Router`. The five home specs register their
    own homes by calling `shell.RegisterHome` from their own
    package init.
16. **Delete** `internal/serve/ui/index.html`,
    `internal/serve/ui/app.js`, `internal/serve/ui/style.css`. The
    embedded filesystem in
    [internal/serve/embed.go](../../../../internal/serve/embed.go)
    drops the `ui/` directory and gains
    `internal/serve/shell/templates` +
    `internal/serve/shell/static` via `//go:embed`. The home
    specs absorb whatever content the old dashboard tabs carried
    (Roadmap → hero-work-home, etc.) via their own specs.
17. `internal/serve/shell/shell_test.go` — table-driven tests for
    edition filtering, top-nav tab activation, `/`-redirect.
18. `internal/serve/session/session_test.go` — round-trip tests
    for `LastHome` / `TabState`.
19. `internal/serve/edition/edition_test.go` — `Resolve` reads
    env var; `Allowed` matches the documented matrix.

### Delivered files (Phases 3–6)

Phases 1–2 (`edition`, `session` packages + tests) landed
earlier. Phases 3–6, this delivery, touched:

- `internal/serve/shell/shell.go` — `Router`, `New`,
  `RegisterHome`, `Handler`, `RenderPage`; 50ms count-fetcher
  timeout; `/` → last-home redirect; duplicate-pattern startup
  panic; `SetLastHome` only on home-root render.
- `internal/serve/shell/types.go` — all shell DTOs (`Home`,
  `ItemRoute`, `Page`, `Chrome`, `ChromeTab`, `Footer`,
  `PageHero`, `PageHeroAction`, `MetricStrip`, `MetricTab`,
  `MetricTile`, `SubNav`, `SubNavTab`, `ChatInput`,
  `ChatContextChip`, `EmptyState`, `EmptyStateAction`).
- `internal/serve/shell/embed.go` — `//go:embed templates/*.html
  static/* static/islands/*`; `StaticFS()` exporter.
- `internal/serve/shell/log.go` — stderr destination.
- `internal/serve/shell/kitchen_sink.go` — `/_kitchen-sink` dev
  showcase.
- `internal/serve/shell/stubs.go` — `RegisterStubHomes` for the
  five top-nav homes; each replaced by its home spec later.
- `internal/serve/shell/templates/{page-layout,top-nav,sub-nav,footer,page-hero,tabbed-metric-strip,chat-input,empty-state-notice}.html` — every template the spec calls for.
- `internal/serve/shell/static/shell.css` — tokens + chrome +
  shared-fragment styles lifted from the Now mock.
- `internal/serve/shell/static/favicon.svg` — hero-blue bolt.
- `internal/serve/shell/static/islands/command-bar.js` —
  placeholder ⌘K binding (real island ships with
  hero-chat-and-model).
- `internal/serve/shell/{shell_test,shell_render_test,import_test}.go` — edition filter / tab activation / `/`-redirect / home-root vs item `SetLastHome` / duplicate-pattern panic / stub render / kitchen-sink render / per-fragment golden checks / forbidden-import AST scan.
- `internal/serve/server.go` — compose `shell.Router` on top of
  the API handler in `Run`; new `buildShellRouter` wires
  edition, session store, workspace name, git branch, version;
  registers the stub homes.
- `internal/serve/api.go` — drop v1 `/` and `/ui/` mounts; drop
  `uiEnabled` from `API`; `NewAPI` no longer takes the flag.
- `internal/serve/api_test.go` — replace the two dashboard tests
  with `TestAPI_RootNotHandledByAPI` (the shell composition owns
  `/`).
- **Deleted** `internal/serve/embed.go` (moved into the shell
  package).
- **Deleted** `internal/serve/ui/{index.html,app.js,style.css}` —
  v1 dashboard.

## Boundaries

- **No view registry. No pack abstraction. No openables manifest.**
  Engineering is the only "pack" so there is nothing to register
  against. If a future Hero web companion (Sales, Ops) ever wants
  to ride on hero serve, we add extensibility then with the
  benefit of a real second consumer's requirements.
- **No VS Code-style tab strip.** Per-item things (specs,
  sessions, notes) are routes, not tabs. Open in a new browser
  tab if you want two at once.
- **No fixed left rail. No fixed right rail. No fixed bottom
  strip.** Actions live inline with the content they affect.
  Context lives in page sections or in slide-overs invoked on
  demand by the home's own JS island (rare).
- **No role-switcher pill row in the top nav.** Top-nav tabs ARE
  the navigation.
- **No chat dispatch logic.** That belongs to
  [hero-chat-and-model](../hero-chat-and-model/spec.md). The shell
  hosts the chat-input template and the ⌘K command-bar island; it
  does not implement dispatch, streaming, adapters, slashes, or
  cost.
- **No home content.** Each home's sections, tile values, lists,
  copy, and JS islands belong to that home's own spec. The shell
  provides chrome, routing, and the fragment library — nothing
  more.
- **No bundler, no React/Vue/Svelte, no top-level build step.**
  Templates + plain CSS + the small vanilla JS the metric-strip
  needs for pane swapping. Islands authored by other specs arrive
  as standalone `.js` files served from
  `internal/serve/shell/static/islands/`.
- **No SPA history hacks.** Anchor links navigate, the browser
  handles back/forward. SSE swaps fragments where data changes
  (lists, counts), not whole pages.

## Acceptance Criteria

- WHEN hero serve starts THE SYSTEM SHALL serve the slim top nav
  on every page route registered through the shell router.
- WHEN the user clicks a top-nav tab THE SYSTEM SHALL load the
  corresponding home page AND highlight that tab with the
  hero-blue underline matching the locked Now-mock visual.
- WHEN the user presses ⌘K (or Ctrl+K) on any page THE SYSTEM
  SHALL mount the command-bar island and open the overlay above
  the current page.
- WHERE `HERO_EDITION` omits a route's allowed editions THE SYSTEM
  SHALL not register that route AND SHALL not render the
  corresponding top-nav tab.
- WHEN the user reopens hero serve at `/` THE SYSTEM SHALL
  redirect to the home recorded as the user's last-visited home,
  defaulting to `/now` if none is recorded.
- WHEN a home with sub-nav state is rendered THE SYSTEM SHALL
  restore the user's last-selected sub-nav tab for that home from
  the session store.
- WHERE the page declares a sub-nav row THE SYSTEM SHALL render
  it directly below the top nav using the shared `sub-nav.html`
  template and the same text-link-with-underline idiom as the top
  nav.
- WHEN a home renders THE SYSTEM SHALL wrap its content with
  `page-layout.html` so the top nav, optional sub-nav, container,
  footer, and command-bar island are present without per-home
  duplication.
- THE SYSTEM SHALL render the page container at max-width 1200px
  centered with 32px horizontal padding.
- THE SYSTEM SHALL provide the `page-hero`,
  `tabbed-metric-strip`, `sub-nav`, `footer`, `chat-input`, and
  `empty-state-notice` templates as reusable fragments callable
  from any home's template via `{{ template "..." . }}`.
- THE SYSTEM SHALL serve `shell.css` and `favicon.svg` from
  `internal/serve/shell/static/` via the embedded filesystem
  without a top-level build step.
- IF a home is requested whose edition gate excludes the active
  `HERO_EDITION` THEN THE SYSTEM SHALL return HTTP 404.
- IF the session store fails to load THEN THE SYSTEM SHALL fall
  back to `/now` and log the error without blocking the request.
- THE SYSTEM SHALL NOT import any chat dispatch, adapter,
  streaming, or inference code from the shell package.

## Risks

- **Temptation to pre-build a pack/registry abstraction.** Two
  drafts of this spec have now over-built for a future
  multi-pack world that does not yet exist. Engineering is the
  only consumer; treat the shell as a private contract between
  the shell package and the five home packages, refactored when
  a real second consumer appears.
- **Cross-pack URL collisions.** There are no cross-pack URLs —
  routes are owned by homes and we control all five home specs.
  If a home's per-item route collides with the top-nav tab list
  (e.g., a spec slugged `now`), the router must validate
  `RegisterHome` patterns at startup and panic. Cheap and
  catches the only realistic foot-gun.
- **Deleting the v1 dashboard.** No external Hero consumer
  embeds `internal/serve/ui/*` (it is served only via hero
  serve's own HTTP handler), but a release-notes call-out is
  required since users may have bookmarks pointing at the old
  `/#sprint` style fragments. Add a redirect from `/` → last
  home and let stale fragments harmlessly land on the new Now.
- **Session state at scale.** Per-user state in a single SQLite
  table is fine for solo and team. Cloud edition will need
  multi-tenant scoping; the `session` package's small surface
  must allow swap-out for a different store. Boundary: do not
  build that store now.
- **Top-nav tab counts as fetchers.** A `Home.Count` callback
  runs on every page render. If a count is expensive, it will
  slow every page. Mitigation: counts are advisory; the router
  invokes them with a 50ms deadline and renders an unbadged tab
  on timeout, recorded as a warning.
- **Edition gate drift from the matrix.** The shell encodes the
  edition→home mapping. If the deployment-and-rendering matrix
  changes, this code must change with it. Mitigation: keep the
  list in one place (`shell.RegisterHome` calls in each home's
  `init`) and add a smoke test that asserts which homes are
  registered under each edition.

## Validation

**Manual:**

- Open `/` on a fresh workspace. Top nav renders with `Now`,
  `Work`, `Knowledge`, `Agents`, `People` tabs and the ⌘K pill.
- Click each tab in turn; the corresponding home loads with the
  active tab underlined.
- Press ⌘K from each of the five homes; the overlay opens
  centered.
- Set `HERO_EDITION=ce` and restart hero serve; verify that any
  home marked enterprise-only does not appear in the top nav or
  resolve at its URL (returns 404).
- Navigate to a sub-nav-bearing home (e.g., `/agents`), pick a
  non-default sub-nav tab, close the tab, reopen `/agents` — the
  same sub-nav tab is selected.
- Close the browser; reopen `/`. Hero serve redirects to the home
  last visited.
- Resize the viewport to 800px wide; the top-nav tabs collapse
  per the responsive rule in the Now mock (handled by
  `shell.css`).

**Automated:**

- `internal/serve/shell/shell_test.go` — table-driven cases:
  - `RegisterHome` filters by edition.
  - Top-nav tab activation matches URL prefix.
  - `/` redirects to recorded last home.
  - Duplicate route patterns panic at startup.
- `internal/serve/session/session_test.go` — round-trip
  `LastHome` / `TabState`; concurrent writes serialize cleanly;
  missing-user reads return `ok=false` not error.
- `internal/serve/edition/edition_test.go` — `Resolve` reads env
  var with documented default; `Allowed` matches the matrix in
  the deployment-and-rendering decision.
- `internal/serve/shell/shell_render_test.go` — golden tests for
  each shared-fragment template, run with a representative
  struct so a designer changing copy in the mock can update
  goldens with intent.

**Build-time:**

- `go vet ./internal/serve/shell/...` clean.
- Static analysis: the `shell` package must not import
  `internal/serve/chat/*` or `internal/runner/*`. Enforce with a
  small import-allowlist test.

## Kickoff

**Status: shell delivered 2026-05-17.** The thin runtime is live —
`hero serve` boots on the new chrome at `/`, redirects to the
user's last-visited home (defaulting to `/now`), and routes five
engineering-home stubs through `shell.Router`. Each home spec
replaces its stub when it delivers.

**Pick up at: invoke `/deliver hero-chat-and-model`** next per the
[initiative build order](../../initiatives/hero-surface-architecture/spec.md).
The shell hosts the chat-input fragment + the command-bar island
mount point, but the dispatcher / adapters / ⌘K overlay logic are
owned by hero-chat-and-model. After chat lands, the five home
specs can each replace their stub by adding their
`shell.RegisterHome` call in their own package.

Foundation packages this delivery introduced:

- [internal/serve/edition](../../../../internal/serve/edition) —
  `HERO_EDITION` resolver + `Allowed()` helper + `Require()`
  middleware.
- [internal/serve/session](../../../../internal/serve/session) —
  per-user `shell_sessions` SQLite store at
  `~/.hero/shell-sessions.db`; `LastHome`/`TabState` round-trip
  helpers; `UserID(*http.Request)` resolves OS user with cookie /
  header overlay for team mode.
- [internal/serve/shell](../../../../internal/serve/shell) —
  `Router`, `Home`, `ItemRoute`, `Page`, `RegisterHome`,
  `RenderPage`; six shared fragment templates (page-layout,
  top-nav, sub-nav, page-hero, tabbed-metric-strip, footer,
  chat-input, empty-state-notice); kitchen-sink dev route at
  `/_kitchen-sink`; five stub home handlers in
  [stubs.go](../../../../internal/serve/shell/stubs.go).

The shell is small on purpose. If a change request looks like it
needs a new abstraction (a registry, a manifest, a pluggable
extension point), push back and ask which engineering home needs
it. If none do, it doesn't belong here yet.
