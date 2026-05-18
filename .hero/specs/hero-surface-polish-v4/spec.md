---
title: Hero Surface Polish v4 — Work View-Tab Active, mdrender Wrapped List Items, Dedup Group Coverage, Detail Title Alignment, Table CSS Width
slug: hero-surface-polish-v4
type: feature
status: completed
tags: [serve, surface, polish, ui, mdrender, web-app]
created: 2026-05-18
relations:
  - target: hero-surface-polish
    kind: parent
  - target: hero-surface-polish-v3
    kind: relates-to
horizon: now
---

## Context

[hero-surface-polish-v3](../../../specs/hero-surface-polish-v3/spec.md)
landed disabled chat-input states on the four non-Now homes, detail-
view breadcrumbs + sub-nav alignment, changes-feed dedup, mdrender
tables / blockquotes / nested lists, and sub-route page-hero titles.

A fresh triage on 2026-05-18 (port 7459) confirms five v3 carry-
overs and surfaces two new high-impact issues. v4 bundles them all
into one pass.

### 1. Work view-toolbar hard-codes "Horizons" as active (NEW)

`internal/serve/pages/work/templates/view-toolbar.html` ships the
view-tab row with `<a href="/work" class="view-tab active">Horizons</a>`
literally hard-coded. Verified live:

```
/work          — no view-toolbar rendered at all (root /work skips the row)
/work/kanban   — view-toolbar present, "Horizons" marked active
/work/blocked  — view-toolbar present, "Horizons" marked active
/work/graph    — view-toolbar present, "Horizons" marked active
```

Two bugs stacked: (a) the active state never reflects the current
route, and (b) `/work` root doesn't render the toolbar even though
that's where "Horizons" lives.

### 2. mdrender splits soft-wrapped list items (NEW)

`internal/serve/mdrender` recognises a list item by its `- ` marker
on the leading line, but if the source line wraps to a second line
without a marker the continuation falls out of the list and renders
as a sibling `<p>`. Visible all over the v3 spec body rendered at
`/work/spec/hero-surface-polish-v3`:

```html
<ul>
  <li>Blockquotes (used for "<strong>Status: delivered</strong>" callouts in</li>
</ul>
<p>recently-completed specs' kickoffs)</p>
```

Spec authors hard-wrap bullet text constantly (the existing
convention is ~70-char lines). Every multi-line bullet currently
renders as a broken-out paragraph after a single-line `<li>`. High
visual impact across every rendered spec.

### 3. Dedup display-group mapping is incomplete (v3 carry-over)

`internal/serve/pages/now/data/changes.go::displayGroupFor` knows
about `peer.call.*` and a few others, but several event kinds in
real feeds fall through ungrouped:

- `spec.complete`
- `claim_acquired`
- `peer_id` minted (migration)
- commits / handoffs that come in clusters

The `/now` feed still works (these render as individual rows) but
runs of the same event kind don't collapse, so the feed
re-fragments on busy days.

### 4. Browser `<title>` on detail routes uses the slug (v3 carry-over)

Today:

```
/work/spec/hero-surface-polish-v3
  <title>Work · Spec · hero-surface-polish-v3 · Hero</title>
  <h1 class="page-title">Hero Surface Polish v3 — Disabled Chat …</h1>
```

The v3 spec deliberately left the slug in `<title>` for
bookmarkability. v4 picks the resolution: keep the slug, but append
the display title so the browser tab is searchable by either:
`Work · Spec · hero-surface-polish-v3 — Hero Surface Polish v3 — Disabled Chat … · Hero`.
Same for `/knowledge/<slug>` and `/people/profiles/<user>` (the
latter already renders the user as the display name, so no change
needed).

### 5. Rendered-body table CSS uses `width: 100%` (v3 carry-over)

`shell.css` table rules under `.wsd-body` / `.kdetail-body` set
`width: 100%`, which causes tables with few short columns to stretch
edge-to-edge in narrow detail containers. Change to `width: auto;
max-width: 100%;` so the table sizes to its content but never
overflows.

### 6. mdrender doesn't handle `\|` inside table cells (v3 carry-over)

`splitTableRow` splits on raw `|` so a markdown table cell
containing `\|` (escaped pipe) breaks the row shape and the table
either renders with wrong columns or falls back to paragraphs.

### 7. `.now-feed-collapsed-rows` visual tightening (v3 carry-over)

Expanded rows under a collapsed group use the same `.now-feed-row`
grid columns as the top-level feed but without the leading
time/icon, so the text column starts at the leftmost grid track
and looks misaligned with the parent row. Add a left-inset so the
expanded rows sit visually nested under the collapsed parent.

## Goal

After this polish pass:

1. **Work view-toolbar reflects the current route.** `/work` shows
   Horizons active. `/work/kanban`, `/work/blocked`, `/work/graph`
   each show the matching tab active. The toolbar renders on
   `/work` root too (so Horizons is reachable from elsewhere).
2. **mdrender keeps soft-wrapped list items inside their `<li>`.**
   Wrapped continuation lines (non-marker, deeper-or-equal indent
   relative to the bullet's text column) join the current item with
   a space until a blank line, a new list marker, or a different
   block kind.
3. **Changes feed dedup covers the common event kinds.** At
   minimum: `peer.call.*`, `spec.complete*`, `claim_acquired`,
   `peer_id.minted`, `commit`, `handoff.*`, `check.*`. Unknown
   kinds keep falling through as singletons (no regression).
4. **Browser `<title>` on `/work/spec/<slug>` and `/knowledge/<slug>`
   includes the display title** after the slug, separated by `—`.
   `/people/profiles/<user>` already shows the user; leave alone.
5. **Rendered-body tables size to content** with a `max-width: 100%`
   guard so very wide tables don't overflow but short tables don't
   stretch.
6. **mdrender splits table rows on unescaped `|` only.** A literal
   `\|` inside a cell renders as `|` text.
7. **`.now-feed-collapsed-rows` rows visually nest** under their
   parent — left-padded to align under the parent text column.

## Approach

### Fix 1: Work view-toolbar active state + root toolbar

`internal/serve/pages/work/templates/view-toolbar.html` becomes
parameterized:

```html
<div class="view-tabs">
  <a href="/work"
     class="view-tab{{ if eq .Active "horizons" }} active{{ end }}">Horizons</a>
  <a href="/work/kanban"
     class="view-tab{{ if eq .Active "kanban" }} active{{ end }}">Kanban</a>
  <a href="/work/graph"
     class="view-tab{{ if eq .Active "graph" }} active{{ end }}">Graph</a>
  <a href="/work/blocked"
     class="view-tab{{ if eq .Active "blocked" }} active{{ end }}">Blocked
    {{- if gt .BlockedCount 0 }} <span class="badge">{{ .BlockedCount }}</span>{{ end }}
  </a>
</div>
```

Each `render*` handler in `page.go` builds its `toolbarData` with
`Active: "horizons"|"kanban"|"graph"|"blocked"`. The root handler
(`handle` for `GET /work`) also renders the toolbar with
`Active: "horizons"` — currently it skips the toolbar entirely, so
this adds the partial back into the root response.

Add `Active string` to whatever struct is passed to the template
(`toolbarData` in `page.go`).

### Fix 2: mdrender wrapped list items

Extend `renderList` so after writing the marker body it peeks
forward for **continuation lines**: lines that are not blank, not a
new list marker at the current or shallower indent, not a fenced
code block, not a blockquote, and not a heading. Append them to the
current `<li>` body with a joining space before running `inline()`.
A nested-list child (deeper indent) still wins — continuation only
applies when the next line is plain text at the body's column or
deeper but not a list marker.

Concrete change in `internal/serve/mdrender/mdrender.go`:

```go
// After fmt.Fprintf(out, "  <li>%s", inline(body))
// Look for continuation lines.
bodyText := body
for i < len(lines) {
    nextRaw := lines[i]
    nextTrim := strings.TrimSpace(nextRaw)
    if nextTrim == "" { break }
    // Stop at any block marker we recognise.
    if _, _, _ := listMarker(nextRaw); /* is a list marker */ break
    if isBlockquote(nextTrim) { break }
    if strings.HasPrefix(nextTrim, "```") { break }
    if strings.HasPrefix(nextTrim, "#") { break }
    if isTableHeader(lines, i) { break }
    // Otherwise treat as continuation: join with a space.
    bodyText += " " + nextTrim
    i++
}
// Re-render the joined body inline. (Refactor so inline() runs after
// continuation collection rather than on the first line only.)
```

The cleanest shape: collect continuation lines first into a single
string, *then* call `inline()` once. The current code emits inline
immediately; refactor to defer the write until continuation
collection is done.

Add `mdrender_test.go` cases:

```
- top bullet that wraps
  to a second source line
- single line
```
expected `<li>top bullet that wraps to a second source line</li>`
followed by `<li>single line</li>`.

Also a wrapped-bullet-followed-by-nested-list case.

### Fix 3: dedup display-group expansion

In `internal/serve/pages/now/data/changes.go::displayGroupFor` and
`groupLabelFor`, extend the mapping:

| Event kind            | Display group   | Group label              |
| --------------------- | --------------- | ------------------------ |
| peer.call.*           | peer-call       | "peer calls"             |
| spec.complete         | spec-complete   | "specs completed"        |
| spec.complete.*       | spec-complete   | "specs completed"        |
| claim_acquired        | claim-acquired  | "claims acquired"        |
| peer_id.minted        | peer-id-minted  | "peer ids minted"        |
| handoff.*             | handoff         | "handoffs"               |
| commit                | commit          | "commits"                |
| check.*               | check           | "checks"                 |

Singular vs plural: when `Count == 1` the original message renders;
when `Count > 1` the group label renders (with the count). So the
labels are plural-only and "peer calls" remains correct.

Update the table-driven tests in `changes_test.go` to exercise:
- `spec.complete` x3 within window → collapses.
- `commit` x2 within window → collapses.
- `claim_acquired` + `peer_id.minted` (different groups) → does not
  collapse.

### Fix 4: detail-route browser title alignment

For `/work/spec/<slug>` and `/knowledge/<slug>`, the detail handler
already loads the spec / knowledge entry and has its display title
available. Build the page meta `Title` as:

- Spec detail: `Work · Spec · <slug> — <display title> · Hero`
- Knowledge detail: `Knowledge · <slug> — <display title> · Hero`

If the display title is empty (loader couldn't parse a title), fall
back to the slug-only form. Keep the page-hero `<h1>` exactly as
today.

Touch points:
- `internal/serve/pages/work/page.go::renderSpecDetail` — set
  `Page.Title` to include display title fallback to slug.
- `internal/serve/pages/knowledge/page.go::renderEntryDetail` —
  same.

People profile detail already renders the user as the title; no
change.

### Fix 5: rendered-body table CSS

In `internal/serve/shell/static/shell.css`, change the table rules
applied to `.wsd-body` / `.kdetail-body`:

```css
/* before */
.wsd-body table, .kdetail-body table {
  border-collapse: collapse; margin: 16px 0; font-size: 13px;
  width: 100%;
}
/* after */
.wsd-body table, .kdetail-body table {
  border-collapse: collapse; margin: 16px 0; font-size: 13px;
  width: auto; max-width: 100%;
}
```

(Adjust the selector path if v3 ship'd it under different scope
names; verify in the actual CSS file before editing.)

### Fix 6: mdrender escaped pipes in table cells

Update `splitTableRow` to walk the input character-by-character and
treat `\|` as a literal `|` inside the current cell, then unescape
before returning. Don't unescape outside table cells — the inline
renderer doesn't currently handle `\|` either, but we don't ship
that yet for v4.

Tests: a table row `| a \| b | c |` parses to two cells `["a | b", "c"]`.

### Fix 7: collapsed-rows visual nesting

In `internal/serve/pages/now/styles.go`, give
`.now-feed-collapsed-rows .now-feed-row` a left padding (e.g.,
`padding-left: 36px;` to clear the parent's time + icon columns)
and a thinner border-left or a subtle indent so it reads as nested.
Keep the same column structure so the time / icon / text columns
align with the parent if present.

### What is OUT of scope for v4

- Mobile responsiveness (defer).
- Loading skeletons / perceptual perf (defer).
- Real session ledger on `/agents/sessions/<id>` beyond the stub
  (substrate work, not polish).
- Search relevance / ranking changes on `/knowledge/search` (defer).
- Markdown autolink improvements, footnotes, raw HTML, etc. — v4
  keeps mdrender in its in-house shape.
- Settings page (`/settings/chat`) — chat-input-connect link still
  points there but the page is out of scope.

## Changes

### Work view-toolbar active state (Fix 1)

- `internal/serve/pages/work/templates/view-toolbar.html` — added
  `.Active` parameter; conditional `active` class per tab.
- `internal/serve/pages/work/templates/page.html` — comment refresh;
  no template-body changes (view-toolbar render is added in page.go
  outside this partial).
- `internal/serve/pages/work/page.go` — `toolbarData` gained
  `Active string`; root `buildPage` now executes `view-toolbar.html`
  with `Active: "horizons"`; `renderBlocked` passes
  `Active: "blocked"`; `renderStub` (kanban/graph) passes the slug as
  `Active`; `renderSection`'s SSE-fragment payload defaults to
  `Active: "horizons"`.
- `internal/serve/pages/work/detail_test.go` — repurposed
  `TestWorkFilterUI_RmFiltersOnlyNoViewToolbar` into
  `TestWorkFilterUI_RootRendersViewToolbarAndRmFilters` (v2 removal
  reverted by v4).
- `internal/serve/pages/work/page_test.go` — updated
  `TestRegister_RendersAllSections` (view-toolbar back on root); added
  `TestRegister_ViewToolbarActiveStateMatchesRoute` covering all four
  Work sub-routes with an exactly-one-active assertion.

### mdrender soft-wrapped list items + escaped pipes (Fix 2, Fix 6)

- `internal/serve/mdrender/mdrender.go::renderList` — collects
  continuation lines (non-blank, non-marker, non-block-construct) into
  the current `<li>` body, joined by spaces, before running `inline()`.
  Continuation halts at blank line, new list marker, blockquote, fence,
  heading, table header, or horizontal rule.
- `internal/serve/mdrender/mdrender.go::splitTableRow` — char-by-char
  walk that treats `\|` as a literal cell character and unescapes it
  on return; outer-pipe trim guards against eating an escaped trailing
  pipe.
- `internal/serve/mdrender/mdrender_test.go` — added
  `TestRender_WrappedBulletItem`, `TestRender_WrappedBulletThenNestedList`,
  `TestRender_WrappedBulletStopsAtBlank`, `TestRender_TableEscapedPipe`,
  and `TestSplitTableRow_EscapedPipe`.

### Dedup group expansion (Fix 3)

- `internal/serve/pages/now/data/changes.go::displayGroupFor` and
  `groupLabelFor` — added groups: `spec.complete*` →
  `"specs completed"`, `claim_acquired` → `"claims acquired"`,
  `peer_id.*` → `"peer ids minted"`, `handoff.*` → `"handoffs"`,
  `check.*` → `"checks"`. `commit` / `files_modified` continue to
  collapse as `"commits"` (kept from v3).
- `internal/serve/pages/now/data/changes_test.go` — extended
  `TestDisplayGroupFor` table; added `TestGroupLabelFor_NewGroups`,
  `TestDedupeWithinWindow_SpecCompleteCollapses`, and
  `TestDedupeWithinWindow_DistinctNewGroupsDoNotCollapse`.

### Detail-route browser title alignment (Fix 4)

- `internal/serve/pages/work/page.go::renderSpecDetail` — page
  title now `Work · Spec · <slug> — <display title> · Hero` with
  slug-only fallback when title is empty or matches the slug.
- `internal/serve/pages/knowledge/page.go::renderEntryDetail` — same
  shape: `Knowledge · <slug> — <display title> · Hero` with slug-only
  fallback. Note: the breadcrumb (page chrome) was already using the
  display title; only the browser `<title>` changes.

### Rendered-body table CSS (Fix 5)

- `internal/serve/shell/static/shell.css` — `.wsd-body table` /
  `.kdetail-body table` updated from `width: 100%` to `width: auto;
  max-width: 100%` so short tables size to content while the overflow
  guard stays for wide tables.

### Collapsed-rows nesting (Fix 7)

- `internal/serve/pages/now/styles.go` — `.now-feed-collapsed-rows`
  gained `margin-left: 92px` (parent's time + icon + gaps), a subtle
  `border-left: 1px solid var(--border)` for nesting cue, and a
  right-only `border-radius` so the left edge reads as a continuation.
  Grid columns inside collapsed rows are unchanged.

## Boundaries

- **No new dependencies.** mdrender stays in-house.
- **No new event types.** Dedup mapping only covers events that
  already flow through `changes.go`.
- **No new homes or sub-routes.**
- **No layout shift on /work root** beyond adding the view-toolbar
  partial (a single row already styled).
- **Detail-route `<h1>`** stays exactly as today — only `<title>`
  changes.

## Acceptance Criteria

- WHEN the user opens `/work` THE SYSTEM SHALL render the
  view-toolbar with the Horizons tab marked active.
- WHEN the user opens `/work/kanban` THE SYSTEM SHALL render the
  view-toolbar with the Kanban tab marked active (and no other).
- WHEN the user opens `/work/blocked` THE SYSTEM SHALL render the
  view-toolbar with the Blocked tab marked active.
- WHEN the user opens `/work/graph` THE SYSTEM SHALL render the
  view-toolbar with the Graph tab marked active.
- WHEN a markdown bullet line wraps to a second source line without
  a list marker THE SYSTEM SHALL render the wrapped text as part of
  the same `<li>`, not as a sibling `<p>` outside the `<ul>`.
- WHEN the user opens `/now` AND the recent feed contains three or
  more `spec.complete` events within the dedup window THE SYSTEM
  SHALL collapse them into a single count row reading
  `<N> specs completed · within the last <window>`.
- WHEN the user opens `/work/spec/<slug>` THE SYSTEM SHALL set the
  browser `<title>` to `Work · Spec · <slug> — <display title> · Hero`
  (or `Work · Spec · <slug> · Hero` when the display title is
  unavailable).
- WHEN the user opens `/knowledge/<slug>` THE SYSTEM SHALL set the
  browser `<title>` to `Knowledge · <slug> — <display title> · Hero`
  (or slug-only when the display title is unavailable).
- WHEN a rendered spec or knowledge entry contains a table with
  three or fewer narrow columns THE SYSTEM SHALL size the table to
  its content (not to the container width).
- WHEN a rendered table cell contains an escaped pipe (`\|`) THE
  SYSTEM SHALL render the cell text containing a literal `|` and
  preserve the surrounding column count.
- WHEN the user expands a collapsed group on the `/now` feed THE
  SYSTEM SHALL render the expanded rows visually inset to the right
  of the parent row's time/icon columns.

## Risks

- **List-item continuation could over-greedy.** If we accept too
  many lines as continuation we'll swallow legitimately separate
  paragraphs that authors typed without a blank line. Mitigation:
  stop on any block marker we recognise (blockquote, heading,
  fence, table, new list item, blank line). When in doubt, prefer
  the under-greedy interpretation.
- **Table CSS change might surprise wide tables.** A few specs may
  have intentionally wide tables relying on full-width stretch.
  Mitigation: `max-width: 100%` keeps the overflow guard; this is
  a soft regression at worst.
- **Browser title change** might surprise users who pinned tabs by
  the slug-only title. The new title still leads with the slug, so
  prefix-matching bookmarks survive. Acceptable.
- **View-toolbar on /work root** introduces a single new row at
  ~32px of vertical real estate. Mitigation: the row was always
  there on sub-routes; adding it to root means the page chrome is
  consistent.
- **Continuation refactor in renderList** is the riskiest single
  change — the function already has nesting logic that's been
  patched once. Mitigation: tests for happy path, wrapped item,
  wrapped item followed by nested list, wrapped item at start vs
  end of list.

## Validation

- Manual: `go install ./cmd/hero` then `hero serve --port 7459
  --no-watch`.
- Curl `/work`, `/work/kanban`, `/work/graph`, `/work/blocked`;
  grep for `view-tab active` and verify the active tab matches the
  route.
- Curl `/work/spec/hero-surface-polish-v3`; grep for raw `</li>`-
  before-`<p>` patterns where the `<p>` clearly continues the
  bullet (should be zero after the fix).
- Curl `/now`; grep for the new group labels (`specs completed`,
  `commits`, `handoffs`) if the feed contains those event clusters.
- Curl `/work/spec/<slug>` and `/knowledge/<slug>`; grep `<title>`
  contains the display title after the slug separator.
- Curl `/work/spec/hero-surface-polish-v2`; render through a
  browser tab; verify the feature-matrix table doesn't stretch to
  full container width.
- `go build ./...` and `go test ./...` both pass.

## Kickoff

**Status: delivered 2026-05-18.** All seven fixes shipped.
`go build ./...` and `go test ./...` both green. Live verification
on port 7459:

- **View-tab active state:** `/work` shows `view-tab active">Horizons`,
  `/work/kanban` → Kanban, `/work/blocked` → Blocked, `/work/graph`
  → Graph. Exactly one tab active per route. `/work` root now
  renders the view-toolbar partial (previously skipped).
- **mdrender wrapped list items:** the v3 spec body rendered at
  `/work/spec/hero-surface-polish-v3` has **zero** `</li></ul><p>`
  continuation patterns where the `<p>` was a wrapped bullet
  continuation. Original bug case (`Blockquotes (used for "Status:
  delivered" callouts in recently-completed specs' kickoffs`) now
  renders as a single `<li>`.
- **Dedup group expansion:** `/now` feed renders `2 commits` next
  to existing `peer calls` collapses. `displayGroupFor` now covers
  spec.complete*, claim_acquired, peer_id.*, handoff.*, check.*.
- **Detail-route browser title:** `<title>Work · Spec ·
  hero-surface-polish-v3 — Hero Surface Polish v3 — Disabled Chat
  States, Detail Sub-nav, Feed Dedup, mdrender Tables, Sub-route
  Titles · Hero</title>` (slug then `—` then display title).
  Knowledge detail same shape.
- **Rendered-body table CSS:** `.wsd-body table` / `.kdetail-body
  table` now `width: auto; max-width: 100%` — short tables no
  longer stretch.
- **Escaped pipes in tables:** `splitTableRow` is char-by-char
  escape-aware. `TestRender_TableEscapedPipe` and
  `TestSplitTableRow_EscapedPipe` cover it.
- **Collapsed-rows nesting:** `.now-feed-collapsed-rows` gets
  `margin-left: 92px` + subtle left border so expanded rows visually
  nest under their parent.

**Pick up at: file v5 follow-ups** in
[hero-surface-polish](../../initiatives/hero-surface-polish/spec.md)
as they're discovered. Known carry-overs from this delivery:

- **Knowledge slug routing inconsistency.** `/knowledge/<slug>`
  resolves `hero-serve-grammar-pivot` but not several other entries
  that exist in `.hero/knowledge/context/<slug>/spec.md` (e.g.,
  `dev-workflow`, `project-overview`, `architecture-overview`,
  `temporal-supersession-pattern`). Loader may be searching a
  narrower path subset than the corpus index. v5 should reconcile.
- **SSE toolbar fragment defaults `Active: "horizons"`.** The
  `renderSection("toolbar")` payload hard-defaults to horizons
  because the SSE channel today only refreshes the blocked badge.
  If a non-Horizons sub-route ever asks for a toolbar refresh via
  SSE, the active state will reset wrongly. v5 could parameterise
  the fragment endpoint with the route, or split the active state
  out of the toolbar partial.
- **`splitTableRow` over-protects `\\|`.** A double-backslash before
  a pipe (`\\|`) is a literal `\` followed by a column separator,
  but the current 2-byte look-back treats the pipe as escaped. Real
  specs don't write `\\|` so it's a corner case; document or fix in
  v5.
- **Author guidance for list continuation.** The renderer now joins
  wrapped bullet lines unless a blank line separates them. Authors
  writing a list followed by a description may expect a blank line
  to terminate the list but currently it's the only way to. Worth a
  knowledge-base note about the rendering rule.
- **`toolbarData.Active` zero-value renders no tab active.** Any
  out-of-package consumer of `toolbarData` (e.g., future SSE
  handlers) would emit a toolbar with zero active tabs unless they
  set `Active`. Not breaking today, but worth an audit before any
  new caller lands.

Established patterns (don't re-litigate):
- shell.RegisterHome with Items: []shell.ItemRoute for sub-routes
- Each home in internal/serve/pages/<home>/ owns its page.go +
  templates/ + data/
- Shell-owned shared fragments (page-hero, sub-nav, page-breadcrumb,
  tabbed-metric-strip, chat-input, empty-state-notice, footer,
  coming-soon)
- internal/serve/mdrender for markdown rendering
- internal/serve/chat is forbidden from page.go imports — homes
  consume it via a Deps.* function injected by server.go
- coming-soon stub is the shared template for sub-routes without
  real content
