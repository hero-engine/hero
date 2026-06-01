---
title: Hero Surface Polish v3 — Disabled Chat States, Detail Sub-nav, Feed Dedup, mdrender Tables, Sub-route Titles
slug: hero-surface-polish-v3
type: feature
status: completed
tags: [serve, surface, polish, ui, mdrender, web-app]
created: 2026-05-18
relations:
  - target: hero-surface-polish
    kind: parent
  - target: hero-surface-polish-v2
    kind: relates-to
  - target: hero-now-home
    kind: relates-to
  - target: hero-work-home
    kind: relates-to
  - target: hero-knowledge-home
    kind: relates-to
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Context

[hero-surface-polish-v2](../../../specs/hero-surface-polish-v2/spec.md)
landed per-item detail routes, filter reconcile, CSS extraction,
verb cleanup, and the inline chat-input on every home. A fresh
triage on 2026-05-18 confirms two v2 carry-overs and surfaces three
new issues.

### 1. Chat-input on the 4 non-Now homes has no disabled state

Confirmed via curl on a workspace with no adapter connected:

```
_now         disabled=0 empty-state=2  (✓ shows the full empty-state notice)
_work        disabled=0 empty-state=0  (looks identical with/without adapter)
_knowledge   disabled=0 empty-state=0
_agents      disabled=0 empty-state=0
_people      disabled=0 empty-state=0
```

V2 spec called for: "inline chat-input … renders disabled with a
short placeholder when no adapter is connected." That part didn't
land. The four homes show a fully-enabled-looking input even when
submission would do nothing.

### 2. Detail-view sub-nav is inconsistent

- `/knowledge/<slug>` renders the Knowledge sub-nav row but with
  NO tab marked active (`class="subnav-tab"` with zero active
  variants).
- `/work/spec/<slug>` renders NO sub-nav at all.

Two different patterns for two detail views in the same surface.
Neither tells the user where they are in the hierarchy. The V2
spec's kickoff flagged this for v3.

### 3. Changes feed shows repetitive event rows

`/now`'s "Since you were here" feed shows 6 of 6 rows as `Peer call
invoked on …` / `Peer call completed on …` — the same handful of
peer events appear over and over with slightly different timestamps.
The feed should collapse repeat events of the same type within a
window into a single row ("6 peer calls to hero-code · within the
last 13h"), with the original entries one click away.

### 4. mdrender drops tables / blockquotes / nested lists

`internal/serve/mdrender` was ship'd in v2 as a tiny in-house
renderer to avoid a new dependency. It handles headings, paragraphs,
inline code, links, and flat lists. It does NOT handle:

- Pipe-style tables (common in spec body — e.g., feature matrix,
  status tables)
- Blockquotes (used for "**Status: delivered**" callouts in
  recently-completed specs' kickoffs)
- Nested lists (specs use them constantly under Approach)

Right now those constructs render as raw markdown source in the
detail view. Quality risk grows as we render more substantive specs
in the detail views.

### 5. Sub-route page-hero titles don't change

`/work/blocked` and `/work/kanban` both render `<h1
class="page-title">Work</h1>` in the page hero. The sub-nav row
shows the active tab, but the page hero is identical across all
sub-routes. Same on Knowledge / Agents / People sub-routes.

Should read "Work · Blocked", "Work · Kanban", "Knowledge ·
Staleness", etc. — the active sub-view name appended after a thin
separator. Honest about where the user is.

## Goal

After this polish pass:

1. **Disabled chat-input on the four non-Now homes** when no
   adapter is connected. Muted styling, "Connect a chat adapter to
   enable" placeholder, input non-interactive.
2. **Detail views render consistent sub-nav.** Both `/knowledge/<slug>`
   and `/work/spec/<slug>` show the parent home's sub-nav row with
   the matching tab active (Browse for Knowledge entries, no active
   tab + "Detail" pseudo-crumb for Work specs since spec detail
   doesn't correspond to a sub-view). Plus a small breadcrumb above
   the page hero: `Knowledge › Entry name` or `Work › Spec › Slug`.
3. **Changes feed deduplicates repeat events.** Same-type events
   within a 1-hour window collapse into one row with a count
   (`6 peer calls to hero-code · within the last 13h`). The
   collapsed row is expandable to show the original entries inline.
4. **mdrender handles tables, blockquotes, and nested lists** with
   sane CSS so spec detail and entry detail views render cleanly.
5. **Sub-route page-hero titles include the sub-view name**:
   `Work · Blocked`, `Knowledge · Staleness`, `Agents · Proposals`,
   `People · Activity`. Detail views show the artifact name:
   `Knowledge · Hero serve grammar pivot`,
   `Work · Spec · hero-surface-polish-v2`.

## Approach

### Fix 1: Disabled chat-input on non-Now homes

The shell's `chat-input` fragment renders an `<input>` with full
styling. To support a disabled state, add a `Disabled bool` field
to the `shell.ChatInput` struct. The template:

```html
<div class="chat-input {{ .Variant }}{{ if .Disabled }} chat-input-disabled{{ end }}"
     data-chat-input-variant="{{ .Variant }}"{{ if .Disabled }} data-chat-disabled="1"{{ end }}>
  <input type="text"
         placeholder="{{ .Placeholder }}"
         {{ if .Disabled }}disabled aria-disabled="true"{{ end }} />
  <!-- ... -->
</div>
```

Add `.chat-input-disabled` rules to `shell.css`:
- Muted background (`var(--bg-soft)`)
- Muted text color on the input (`var(--ink-4)`)
- Cursor: not-allowed
- Submit button (if any in inline variant) hidden or muted

Each non-Now home's `chatInputFor` helper (already exists in
`pages/{work,knowledge,agentspage,people}/page.go`) needs to:

1. Check chat capability via `chat.Resolve(deps.ChatRegistry, "")`
2. Set `Disabled: capability.Interactive == ""`
3. Set `Placeholder: "Connect a chat adapter to enable"` when disabled,
   else the existing per-home placeholder.

Now home's hero-variant input stays the same — it pairs with the
full empty-state notice instead of the disabled-input pattern.

### Fix 2: Detail-view sub-nav consistency

Both detail views render via the parent home's sub-nav builder, but:

- **`renderEntryDetail`** (Knowledge): pass `activeSlug: "browse"` to
  the sub-nav builder so the Browse tab is highlighted (the entry
  came from Browse).
- **`renderSpecDetail`** (Work): Work has NO sub-nav row in v2 (Work
  uses an in-page view-toggle row, not a shell sub-nav). Add a
  small "Detail" pseudo-tab? No — that conflicts with the toggle
  pattern. Instead, just render the Work page chrome WITHOUT the
  view toggle row on detail views (the user is no longer choosing a
  view) and show a breadcrumb (Fix 2b) for orientation.

Fix 2b: **Breadcrumb above page hero on detail views**

A small breadcrumb element:
- `Knowledge › Hero serve grammar pivot` (entry detail)
- `Work › Spec › hero-surface-polish-v2` (spec detail)
- `Agents › Session › abc123` (session detail stub)
- `People › Profile › bwheeler` (profile detail stub)

Markup:
```html
<nav aria-label="Breadcrumb" class="page-breadcrumb">
  <ol>
    <li><a href="/work">Work</a></li>
    <li class="sep">›</li>
    <li>Spec</li>
    <li class="sep">›</li>
    <li aria-current="page">hero-surface-polish-v2</li>
  </ol>
</nav>
```

Render at the top of the detail page, immediately below the top nav
and above the page hero. CSS: 12px font, muted color, hero-blue on
hover for the links.

The breadcrumb is a shared shell fragment so any future detail view
gets it for free. Add `shell/templates/page-breadcrumb.html` and a
`shell.PageBreadcrumb` struct.

### Fix 3: Changes feed dedup

In `pages/now/data/changes.go`, after fetching the raw event rows
(currently the top N most-recent), apply a dedup pass:

```go
func dedupeWithinWindow(rows []ChangeRow, window time.Duration) []ChangeRow {
    out := make([]ChangeRow, 0, len(rows))
    var lastByType map[string]int // type → index in out
    for _, r := range rows {
        if idx, ok := lastByType[r.Kind]; ok {
            // Within window of the previous same-type row?
            if out[idx].Time.Sub(r.Time) < window {
                out[idx].Count++
                out[idx].CollapsedRows = append(out[idx].CollapsedRows, r)
                continue
            }
        }
        out = append(out, r)
        lastByType[r.Kind] = len(out) - 1
    }
    return out
}
```

Render in `templates/changes.html`:
- If `.Count > 1`: render the row with a count badge ("6 peer calls
  to hero-code · within the last 13h") and a small "expand ▼"
  affordance that toggles the collapsed rows visible inline (CSS
  details/summary or a tiny vanilla JS handler).
- If `.Count == 1`: render as today (single row, no badge).

Window default: 1 hour. Configurable via constant in changes.go (no
need to expose in `hero.json` for v3).

The dedup also applies kind-aware: peer.call.invoked and
peer.call.completed are different types — but for display, collapse
them together as "peer.call.*". Add a helper that maps event types
to display groups.

### Fix 4: mdrender tables, blockquotes, nested lists

Extend `internal/serve/mdrender/mdrender.go` to handle:

**Pipe-style tables**:
```
| Col A | Col B |
| ----- | ----- |
| val   | val   |
```
Parse: lines starting with `|`, separator row of `---` style, body
rows. Emit `<table><thead><tr><th>` etc. Don't worry about
alignment markers (`:---:`) for v3.

**Blockquotes**:
```
> Status: delivered
> Some more text.
```
Lines starting with `> `. Group consecutive blockquote lines into
one `<blockquote>` containing rendered paragraphs.

**Nested lists**:
```
- top
  - nested
  - nested
- top again
```
Detect indentation (2-space or 4-space) on `- ` items. Emit nested
`<ul>`. Same for ordered (`1. `).

Add CSS in `shell.css`:
```css
.spec-body table, .kn-detail-body table {
  border-collapse: collapse; margin: 16px 0; font-size: 13px;
}
.spec-body th, .spec-body td, .kn-detail-body th, .kn-detail-body td {
  border: 1px solid var(--border); padding: 6px 10px; text-align: left;
}
.spec-body th, .kn-detail-body th {
  background: var(--bg-soft); font-weight: 600;
}
.spec-body blockquote, .kn-detail-body blockquote {
  border-left: 3px solid var(--hero-blue-500);
  padding: 4px 12px; margin: 12px 0;
  background: var(--bg-soft); color: var(--ink-3);
}
.spec-body ul ul, .spec-body ol ol, .spec-body ul ol, .spec-body ol ul {
  margin-top: 4px; margin-bottom: 4px;
}
```

Tests in `mdrender_test.go`:
- Table with 2 cols, 2 body rows → expected HTML
- Blockquote spanning 3 lines → expected HTML
- Nested unordered list 2 levels deep → expected HTML
- Mixed (table inside blockquote etc.) — only worry about the
  combinations spec body actually uses; don't over-test.

### Fix 5: Sub-route page-hero titles

Each home's per-view `render*` function passes a `PageHero` struct
with a `Title` string. Today the title is the home name only
("Work", "Knowledge", etc.). Change so the title includes the
sub-view name on non-default routes.

Format: `<Home> · <Sub-view>` — e.g., `Work · Blocked`,
`Knowledge · Staleness`, `Agents · Proposals`, `People · Activity`.

Detail views: `<Home> · <Kind> · <Slug>` (the breadcrumb already
shows this; the title can be shorter):
- `Knowledge · Hero serve grammar pivot` (entry title)
- `Work · Spec · hero-surface-polish-v2` (slug)
- `Agents · Session · abc123`
- `People · Profile · bwheeler`

The `<title>` element (browser tab) follows the same pattern.

Implementation: each `render*` function builds its own PageHero
with the right title. Or, add a `subView string` param to a shared
helper.

### What is OUT of scope for v3

- Loading states / skeletons (perceptual perf).
- Error states for failed data fetchers (right now they probably
  render empty section partials; OK for v3, file v4).
- Mobile responsiveness (none of the mocks address; defer).
- Sparkline / chart rendering quality audit (the metric tiles
  render but I haven't verified visual quality).
- Real `cost_usd` payload on delivery events (defer to runner work).
- Cross-pack search (defer).

## Changes

### Disabled chat-input (Fix 1)

- `internal/serve/shell/types.go` — added `Disabled bool` and
  `ConnectHref string` fields to `ChatInput`.
- `internal/serve/shell/templates/chat-input.html` — renders
  `disabled` / `aria-disabled` / `chat-input-disabled` class and an
  optional `Connect adapter →` link when disabled; submit button
  hidden in the disabled state.
- `internal/serve/shell/static/shell.css` — added
  `.chat-input-disabled` and `.chat-input-connect` rules.
- `internal/serve/pages/work/page.go` — added
  `Deps.ChatInteractiveConnected func() bool` probe; `chatInputFor`
  flips to disabled when nil/false. (Probe rather than registry
  import keeps the package free of `chat` dependency per
  `import_test`.)
- `internal/serve/pages/knowledge/page.go` — same probe wiring.
- `internal/serve/pages/agentspage/page.go` — same probe wiring.
- `internal/serve/pages/people/page.go` — same probe wiring.
- `internal/serve/server.go` — added `Server.chatInteractiveConnected`
  method; wires it into all four home Deps bundles.

### Detail-view sub-nav + breadcrumb (Fix 2)

- `internal/serve/shell/templates/page-breadcrumb.html` (new) —
  shared breadcrumb fragment.
- `internal/serve/shell/types.go` — `PageBreadcrumb` + `BreadcrumbCrumb`
  types; `Page.Breadcrumb` field.
- `internal/serve/shell/shell.go` — `RenderPage` passes `Breadcrumb`
  into the layout template.
- `internal/serve/shell/templates/page-layout.html` — slot the
  breadcrumb between the sub-nav and the page-container.
- `internal/serve/shell/static/shell.css` — `.page-breadcrumb`
  styling (12px font, ~28px row).
- `internal/serve/pages/knowledge/page.go` — `renderEntryDetail`
  highlights the Browse tab and emits breadcrumb
  `Knowledge › <entry title>`; added `serveDetail` helper.
- `internal/serve/pages/work/page.go` — `renderSpecDetail` emits
  breadcrumb `Work › Spec › <slug>`; no view-toggle row on detail
  (Work has no shell sub-nav).
- `internal/serve/pages/agentspage/page.go` — session detail emits
  breadcrumb `Agents › Session › <id>`.
- `internal/serve/pages/people/page.go` — profile detail emits
  breadcrumb `People › Profile › <user>`; added `serveDetail`
  helper.

### Changes feed dedup (Fix 3)

- `internal/serve/pages/now/data/types.go` — added `TimeAt`,
  `DisplayGroup`, `GroupLabel`, `Count`, `WindowText`, `CollapsedRows`
  fields to `ChangeRow`.
- `internal/serve/pages/now/data/changes.go` — added
  `dedupeWithinWindow`, `displayGroupFor`, `groupLabelFor`,
  `humanWindowText`; `LoadChanges` over-fetches then dedupes within
  1h and truncates to 6 display rows.
- `internal/serve/pages/now/data/changes_test.go` — added 4 tests
  covering peer-call collapse, outside-window non-collapse,
  different-group non-collapse, and `displayGroupFor` mapping.
- `internal/serve/pages/now/templates/changes.html` — renders the
  count badge + expand chevron for `Count > 1` rows; inline
  collapsed-rows div.
- `internal/serve/pages/now/styles.go` — added
  `.now-feed-count`, `.now-feed-expand`, `.now-feed-collapsed-rows`
  CSS; added a click-toggle handler to `nowScript`.

### mdrender extension (Fix 4)

- `internal/serve/mdrender/mdrender.go` — added pipe-table parser
  (with malformed-shape fallback), blockquote handling (groups
  consecutive `>` lines, renders inner recursively), and nested-
  list handling (2- or 4-space indent on `- ` / `1. ` items).
- `internal/serve/mdrender/mdrender_test.go` — added 5 tests
  (table happy-path, malformed table without separator, table
  with col-count mismatch, blockquote, nested bullet list).
- `internal/serve/shell/static/shell.css` — added table / th-td /
  blockquote / nested-list rules under `.wsd-body` (Work spec
  detail) and `.kdetail-body` (Knowledge entry detail).

### Sub-route page-hero titles (Fix 5)

- `internal/serve/pages/work/page.go::buildPageHero` — added
  `subView string` param; title becomes `Work · <Sub>` when set.
  Call sites updated for `/work` (`""`), `/work/blocked`
  (`"Blocked"`), and stub views (`view`).
- `internal/serve/pages/knowledge/page.go::buildPageHero` — added
  `subView string` param; title becomes `Knowledge · <Sub>` when
  set. Call sites updated for browse, why, staleness, stubs, and
  the entry detail (uses entry title via `serveDetail`).
- `internal/serve/pages/agentspage/page.go::buildPageHero` — added
  `subView string` param; `buildPageWith` accepts a `subViewLabel`
  and propagates it to the page title + browser `<title>`.
- `internal/serve/pages/people/page.go::buildPulsePageHero` —
  added `subView string` param; `buildROIPageHero` title becomes
  `People · ROI Overview`; profile detail title is `People · Profile
  · <user>`.

### Tests

- `internal/serve/mdrender/mdrender_test.go` — 5 new tests (above).
- `internal/serve/pages/now/data/changes_test.go` — 4 new tests
  (above).
- Existing import-guard tests in `pages/{work,knowledge,agentspage,
  people}/import_test.go` continue to pass — the probe-function
  approach keeps these packages chat/runner-free.

## Boundaries

- **No new home content** beyond breadcrumbs and titles.
- **No new event types or pipelines.**
- **No new third-party dependencies** — mdrender stays in-house.
- **No design pivot** on the disabled chat-input look — pick the
  greyed-out pattern and ship it.

## Acceptance Criteria

- WHEN the user opens any of `/work`, `/knowledge`, `/agents`, or
  `/people` AND no chat adapter is connected THE SYSTEM SHALL
  render the inline chat-input with the `disabled` attribute,
  `chat-input-disabled` class, muted placeholder, and non-
  interactive cursor.
- WHEN the user opens the same routes AND an adapter IS connected
  THE SYSTEM SHALL render the inline chat-input fully enabled.
- WHEN the user opens `/knowledge/<slug>` THE SYSTEM SHALL render
  the Knowledge sub-nav with `Browse` tab active AND a breadcrumb
  reading `Knowledge › <entry title>`.
- WHEN the user opens `/work/spec/<slug>` THE SYSTEM SHALL render
  the Work page chrome WITHOUT the in-page view toggle AND a
  breadcrumb reading `Work › Spec › <slug>`.
- WHEN the user opens `/agents/session/<id>` THE SYSTEM SHALL
  render a breadcrumb `Agents › Session › <id>`.
- WHEN the user opens `/people/profiles/<user>` THE SYSTEM SHALL
  render a breadcrumb `People › Profile › <user>`.
- WHILE the changes feed contains more than one event of the same
  display group within a 1-hour window THE SYSTEM SHALL render the
  group as a single row with a count badge AND an expand
  affordance.
- WHEN the user clicks the expand affordance THE SYSTEM SHALL
  reveal the original event rows inline below the collapsed row.
- WHEN a spec or knowledge entry body contains a pipe-style table
  THE SYSTEM SHALL render it as an HTML `<table>` with styled
  borders.
- WHEN a body contains `> ` blockquote lines THE SYSTEM SHALL
  render them as an HTML `<blockquote>` with hero-blue left
  accent.
- WHEN a body contains nested list items (2-space or 4-space
  indented under a `- `) THE SYSTEM SHALL render them as nested
  `<ul>` or `<ol>`.
- WHEN the user opens any non-default sub-route THE SYSTEM SHALL
  render the page-hero title as `<Home> · <Sub-view>`.
- WHEN the user opens any detail route THE SYSTEM SHALL render the
  page-hero title and the browser `<title>` to include the
  artifact's name.

## Risks

- **Disabled-input keyboard-focus accessibility.** A `disabled`
  input is unfocusable; users using keyboard navigation might
  wonder where the input is. Mitigation: keep `aria-disabled="true"`
  + visible CTA pointing at the empty-state path (a small "Connect
  adapter →" link beside the input, even on non-Now homes; lighter
  than Now's full notice).
- **Breadcrumb on every detail page** adds vertical real estate.
  Mitigation: 12px font, ~28px tall row, ~12px below top nav. Total
  cost ~40px; acceptable.
- **Changes feed dedup might hide events** the user wanted to see
  individually. Mitigation: count + expand affordance always
  visible; expand is one click.
- **mdrender's pipe-table parser is finicky.** Malformed tables
  (missing separator row, mismatched col count) could break the
  render. Mitigation: parser falls back to "render as plain
  paragraphs" when shape doesn't match; test the failure path.
- **Page-hero title change might surprise** users who bookmarked
  sub-routes expecting the same title. Acceptable — minor.

## Validation

- Manual: stop hero-code (or never start it); open each non-Now
  home; verify chat-input looks disabled (muted, cursor:not-allowed,
  greyed placeholder).
- Manual: visit `/knowledge/<real-slug>`; verify breadcrumb +
  active Browse tab.
- Manual: visit `/work/spec/<real-slug>`; verify breadcrumb + no
  view toggle row.
- Manual: visit `/now`; confirm the changes feed shows count-badge
  rows for repeated peer.call events; click expand to verify rows
  appear.
- Manual: render a spec that has a markdown table (e.g.,
  hero-surface-polish-v1 spec's feature matrix) at `/work/spec/...`;
  verify the table renders with borders.
- Manual: render a blockquote (hero-surface-shell spec's status
  callout) and verify hero-blue left accent.
- Manual: render a nested list (hero-now-home spec's Approach
  section) and verify nesting depth shows.
- Manual: visit `/work/blocked`; verify page-hero says
  `Work · Blocked`. Same for the other sub-routes.
- Tests per the Changes list (25-29).
- `go build ./... && go test ./...` must pass.

## Kickoff

**Status: delivered 2026-05-18.** All five fixes shipped. Live
verification on port 7463:

- Chat-input disabled markers: 2 on each non-Now home (`/work`,
  `/knowledge`, `/agents`, `/people`) — `disabled` + `aria-disabled`
  attrs + `chat-input-disabled` class + inline "Connect adapter →"
  link.
- Breadcrumb on all 4 detail routes (knowledge entry, work spec,
  agents session stub, people profile stub).
- Page-hero titles include sub-view name: `Work · Blocked`,
  `Knowledge · Staleness`, `Agents · Proposals`, `People · Activity`.
- mdrender on the `active-context-management` spec renders 2 tables
  and 14 nested `<ul>` opens. Blockquote support landed but the
  test spec doesn't use them — `agent-cold-start` verification in
  the engineer's run confirmed 1 blockquote rendered.
- Changes feed dedup: 13 dedup-related markers (count badges +
  collapsed-rows wrappers + data-feed-collapsed hooks) on `/now`.
- All 5 home roots + sub-routes still 200; no regressions.

**Pick up at: file v4 follow-ups** in
[hero-surface-polish](../../../planning/initiatives/hero-surface-polish/spec.md)
as they surface. Known carry-overs from this delivery:

- **Browser `<title>` for spec/profile detail uses the slug**; page
  hero shows display title. Align (probably keep slug in `<title>`
  for bookmarkability, display title in hero).
- **Dedup group mapping incomplete** for several event kinds
  (`spec.complete`, `claim_acquired`, etc.) — they fall through
  ungrouped. Add as those events appear in real feeds.
- **Table CSS uses `width: 100%`** which could overflow narrow
  rendered-body containers. Revisit if a spec lands with very wide
  tables.
- **`splitTableRow` doesn't escape pipes inside cells** —
  markdown's `\|` is unimplemented; specs don't currently use it.
- **`.now-feed-collapsed-rows` style** uses an internal grid
  without an icon column; could tighten visually.
