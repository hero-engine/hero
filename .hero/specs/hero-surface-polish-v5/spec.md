---
title: Hero Surface Polish v5 — Knowledge Dir-Style Entries, Corpus Undercount, Agents Root Title, SSE/Table/Toolbar Cleanups
slug: hero-surface-polish-v5
type: feature
status: completed
tags: [serve, surface, polish, ui, knowledge, web-app]
created: 2026-05-18
relations:
  - target: hero-surface-polish
    kind: parent
  - target: hero-surface-polish-v4
    kind: relates-to
horizon: now
completed_at: 2026-05-18T17:57:21Z
---

## Context

[hero-surface-polish-v4](../../../specs/hero-surface-polish-v4/spec.md)
landed the Work view-tab active state, mdrender wrapped list items,
extended `/now` dedup groups, detail browser titles, table CSS, and
collapsed-rows nesting.

A fresh triage on 2026-05-18 (port 7459, post-v4 binary) surfaces
three new high-impact bugs and confirms three v4 carry-overs that
are cheap to close out in the same pass.

### 1. Knowledge entry loader misses dir-style + nested entries (NEW)

`internal/serve/pages/knowledge/data/entry.go::LoadEntry` only looks
for `<heroDir>/knowledge/<subdir>/<slug>.md`. But the knowledge
tree mixes shapes:

```
.hero/knowledge/
├── notes/                          # flat .md files (one entry per file)
│   ├── hero-serve-grammar-pivot.md
│   └── active-context-management-design/   # ← dir-style entry, invisible
│       └── spec.md
├── context/                        # all dir-style entries
│   ├── dev-workflow/spec.md
│   ├── project-overview/spec.md
│   ├── architecture-overview/spec.md
│   └── temporal-supersession-pattern/spec.md
├── conventions/                    # mostly flat .md
├── decisions/                      # mostly flat .md
└── rules/                          # nested dirs
    └── project-rules/
        └── auto-generated-spec-sections.md
```

Verified live: `/knowledge/hero-serve-grammar-pivot` → 200,
`/knowledge/dev-workflow` → 404, same for `project-overview`,
`architecture-overview`, `temporal-supersession-pattern`. Every
`context/<slug>/spec.md` is unreachable from the web surface today.

### 2. Knowledge corpus listing undercounts (NEW)

`internal/serve/pages/knowledge/data/corpus.go::LoadCorpus` walks
the same shape and skips `f.IsDir()` explicitly:

```go
for _, f := range files {
    if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
        continue
    }
    …
}
```

Result: `/knowledge` shows 15 kn-cards, `.hero/knowledge/` actually
contains 35 markdown files (17 flat + 17+ dir-style + nested). The
metric strip says `15 entries in the corpus`, which is the same
undercount surfaced as a metric.

Same blind spot as fix 1, separate code path.

### 3. `/agents` root page-title is "Sessions" (NEW)

`internal/serve/pages/agentspage/page.go::buildPageHero` line 487
hard-codes the home root title:

```go
title := "Sessions"
if subView != "" {
    title = "Agents · " + subView
}
```

Live verification:

```
/agents                  <h1>Sessions</h1>
/agents/proposals        <h1>Agents · Proposals</h1>
/agents/scheduled        <h1>Agents · Scheduled</h1>
…
```

Sibling homes all render the home name on root: `/now` → "Now",
`/work` → "Work", `/knowledge` → "Knowledge", `/people` → "People".
Agents is the only home that swaps in a sub-view name on root,
which looks like a typo / leftover from an earlier design.

### 4. SSE toolbar fragment defaults to horizons (v4 carry-over)

`internal/serve/pages/work/page.go::renderSection("toolbar")` builds
the toolbar fragment with `Active: "horizons"`. The blocked-badge
SSE channel is the only consumer today, but the moment any
`/work/*` sub-route uses the toolbar SSE channel to refresh, the
active tab will snap back to Horizons.

### 5. mdrender `splitTableRow` over-protects `\\|` (v4 carry-over)

The escape-aware walk treats any `\|` as a literal pipe — including
the case where the `\` itself was escaped (`\\|` = literal backslash
followed by column separator). Two-byte look-back doesn't
distinguish `\|` from `\\|`. Real specs don't write `\\|` so this
is a corner case, but cheap to fix and write the test.

### 6. `toolbarData.Active` zero-value audit (v4 carry-over)

`Active string` defaults to `""`, which the template renders as "no
tab active" (no class match). No current consumer hits this — all
four `render*` handlers + the SSE fragment set it explicitly — but
any new caller that forgets renders a row of inert tabs. Worth
either a typed constant set or a documented "you must set Active"
contract enforced by the test added in v4
(`TestRegister_ViewToolbarActiveStateMatchesRoute`).

## Goal

After this polish pass:

1. **Every `.hero/knowledge/` entry is reachable.** Both
   `<subdir>/<slug>.md` and `<subdir>/<slug>/spec.md` resolve via
   `/knowledge/<slug>`. Nested sub-dirs (e.g.
   `rules/project-rules/<slug>.md`) also resolve.
2. **`/knowledge` shows every entry.** The corpus index counts and
   lists every file the entry loader can resolve. Corpus count =
   entry-loader-reachable count.
3. **`/agents` root page-title is "Agents".** Consistent with
   sibling homes. Sub-route titles ("Agents · Proposals", etc.)
   unchanged.
4. **SSE toolbar fragment carries the active tab.** Either the
   fragment endpoint accepts a route hint or the toolbar partial
   stops owning active state (lifted to a separate partial). v5
   picks the simpler: accept `view` query-param on the fragment
   route and pass it through to `toolbarData.Active`.
5. **`splitTableRow` distinguishes `\|` from `\\|`.** Escape parser
   tracks a real escape state instead of a 2-byte look-back.
6. **`toolbarData.Active` requires an explicit value.** Either
   panic on empty in `view-toolbar.html` (loud failure mode for new
   callers) or change the field to a typed enum that has no zero
   meaning. v5 picks: keep `Active string` but assert non-empty in
   the work-page `renderToolbar` helper.

## Approach

### Fix 1: Knowledge entry loader — dir-style + nested

`LoadEntry` becomes a two-pass walk:

1. First pass (existing): check each direct subdir for
   `<slug>.md` (flat shape).
2. New: for each subdir entry that IS a directory, check for
   `<subdir>/<slug>/spec.md` (dir-style shape).
3. New: walk one level deeper into nested directories (e.g.
   `rules/project-rules/<slug>.md`) — only one level, since
   spec authors don't nest deeper.
4. Existing loose-top-level pass stays.

Picking precedence: if `<slug>.md` and `<slug>/spec.md` both exist
in the same subdir (unlikely but possible), the flat file wins
since that's the legacy shape.

The `readEntry` helper currently takes a path and the `dir`
(subdirectory name) for kind/domain. For dir-style entries the
"dir" is still the top-level subdir name (e.g. "context"), not the
slug-named intermediate dir. Update call sites accordingly.

Risk: nested-dir walk could explode if the knowledge tree grows
deeply nested. Mitigation: depth-limit to 2 levels below the
top-level kind subdir.

### Fix 2: Corpus listing — same shape change

`LoadCorpus` gets the same two-pass walk:

1. Existing flat `<slug>.md` collection.
2. New: subdirectory-shape `<slug>/spec.md` collection — `slug` =
   the intermediate dir name.
3. New: one-level-deeper nested-dir collection.

Same depth limit. Sort + dedup at the end (a `seen[slug]` map keyed
by `kind + slug` so a flat and dir-style collision doesn't
double-list).

The metric strip's `TotalEntries` count is just `len(entries)`
after dedup; the count will jump from 15 → ~30+ once the fix lands.
That's the intended correction.

### Fix 3: `/agents` root page-title

In `internal/serve/pages/agentspage/page.go::buildPageHero`,
change:

```go
title := "Sessions"
if subView != "" {
    title = "Agents · " + subView
}
```

to:

```go
title := "Agents"
if subView != "" {
    title = "Agents · " + subView
}
```

The sub-nav tab labeled "Sessions" still points at `/agents` root
and renders the sessions content — only the page-hero `<h1>` and
the browser `<title>` change. The eyebrow already says
`hero · main · agents`, which is unaffected.

The active-tab logic for the sub-nav still treats `/agents` as
`activeSlug == "sessions"` (see `buildSubNav` at line 448), which
is fine — that's about which tab is highlighted, not about the
page title.

### Fix 4: SSE toolbar fragment route hint

`renderSection("toolbar")` is reached via a `GET /work/_fragments/toolbar`
or similar (verify exact path during delivery). Currently:

```go
case "toolbar":
    data := toolbarData{Active: "horizons", BlockedCount: ...}
```

Change to:

```go
case "toolbar":
    view := req.URL.Query().Get("view")
    if view == "" { view = "horizons" }
    data := toolbarData{Active: view, BlockedCount: ...}
```

And client-side: where the SSE channel triggers the toolbar
refresh, include the current `view` in the URL. Verify whether any
client code constructs the fragment URL today; if no client does,
the change is forward-only.

### Fix 5: splitTableRow real escape state

Rewrite the cell-walking loop to track a `escaped bool`:

```go
for _, r := range row {
    if escaped {
        cell.WriteRune(r)
        escaped = false
        continue
    }
    if r == '\\' {
        escaped = true
        continue
    }
    if r == '|' {
        cells = append(cells, cell.String())
        cell.Reset()
        continue
    }
    cell.WriteRune(r)
}
```

Now `\|` → literal pipe in cell, `\\|` → literal `\` then column
separator, `\\` → literal `\` (terminal escape state at end of row
just drops). Add tests:
- `| a \| b | c |` → `["a | b", "c"]` (existing)
- `| a \\| b | c |` → `["a \\", "b", "c"]`
- `| a \\\| b | c |` → `["a \\| b", "c"]`

### Fix 6: `toolbarData.Active` non-empty assertion

The cleanest enforcement is at the call-site: the `renderToolbar`
helper that all four `render*` handlers + the SSE fragment use
should panic / log-and-default if `Active == ""`. Since panic in a
web handler is bad, do log + default + a unit test that catches
the issue.

Alternative: keep `Active string` but add a test that constructs
`toolbarData{Active: ""}` and verifies the rendered HTML contains
exactly zero `view-tab active` matches — making the "no tab active"
behavior documented and intentional.

v5 picks the test approach: it documents the contract without
forcing new callers to thread a value they may not have yet.

### What is OUT of scope for v5

- Knowledge corpus index dedup quality (cross-kind dedup, fuzzy
  match) — only naive slug+kind dedup in v5.
- Knowledge tree-depth-3+ entries — depth-2 walk only.
- Spec corpus equivalent of these bugs — `/work/spec/<slug>`
  already uses a different loader that's known to walk specs/
  + planning/ recursively. (Verify quickly during delivery; if
  the same shape bug exists, add to v5 — but I don't have evidence
  it does.)
- Filter-chip functionality on `/work/*` — still placeholders.
- Settings page `/settings/chat` — still out of scope.
- Other home page-hero metric accuracy audit.

## Changes

Files modified / created in this delivery:

- `internal/serve/pages/knowledge/data/walk.go` — new shared helper
  `collectKnowledgeFiles(root)` covering the three entry shapes
  (flat, dir-style `<kind>/<slug>/spec.md`, nested
  `<kind>/<nested>/<slug>.md`) with depth-2 limit and skip of
  hidden / underscore dirs. Flat wins on (kind, slug) collisions.
- `internal/serve/pages/knowledge/data/entry.go` — `LoadEntry`
  rewritten on top of `collectKnowledgeFiles`; loose top-level
  `<slug>.md` fallback preserved. `readEntry` now takes an explicit
  `slug` so dir-style entries don't resolve to `"spec"`.
- `internal/serve/pages/knowledge/data/corpus.go` — `LoadCorpus`
  rewritten on top of `collectKnowledgeFiles`; count now matches
  loader-resolvable entries.
- `internal/serve/pages/knowledge/data/walk_test.go` — new tests:
  three-shape coverage, hidden/underscore-dir skip, flat-vs-dir
  collision, LoadEntry flat/dir-style/nested/missing, LoadCorpus
  count + cross-check against LoadEntry.
- `internal/serve/pages/agentspage/page.go` — empty-subView title
  now "Agents" (was "Sessions"); buildPageHero doc comment updated.
- `internal/serve/pages/agentspage/page_test.go` — assert
  `<h1 class="page-title">Agents</h1>` on `/agents` root and
  `Agents · Proposals` still on `/agents/proposals`.
- `internal/serve/pages/work/page.go` — `SectionFragment` and
  `renderSection` now take an explicit `view string`; toolbar case
  uses it as `Active` (default "horizons"). `toolbarData.Active`
  doc comment lists valid values + empty-fallback semantics.
- `internal/serve/api/work.go` — `handleSection` threads
  `?view=<slug>` through to `SectionFragment` so the toolbar SSE
  fragment honors the current view.
- `internal/serve/pages/work/page_test.go` — empty-Active no-active-
  tab assertion + table-driven view-param coverage.
- `internal/serve/mdrender/mdrender.go` — `splitTableRow` rewritten
  with proper `escaped bool` state machine; trailing-pipe trim now
  counts backslashes to distinguish `\|` from `\\|`.
- `internal/serve/mdrender/mdrender_test.go` — escape-state machine
  cases for `\|`, `\\|`, `\\\|`, and bare `\\`.

## Boundaries

- **No new dependencies.**
- **No knowledge schema changes** — entries continue to be
  identified by frontmatter `slug` (or filename) and live where
  they live.
- **No new homes or sub-routes.**
- **No layout shift** beyond the page-title text change on
  `/agents` root.
- **Knowledge entry kind / domain stay the same.** A dir-style
  entry under `context/<slug>/spec.md` has `Kind: "context"`,
  `Domain: "context"` — matching the flat-style convention.
- **Don't recursively walk arbitrarily deep.** Depth-2 limit below
  each top-level kind dir.

## Acceptance Criteria

- WHEN the user opens `/knowledge/<slug>` for an entry stored as
  `.hero/knowledge/<kind>/<slug>/spec.md` THE SYSTEM SHALL render
  the entry detail (200) with title, body, and breadcrumb.
- WHEN the user opens `/knowledge/<slug>` for an entry stored as
  `.hero/knowledge/<kind>/<nested>/<slug>.md` (one level deeper) THE
  SYSTEM SHALL also resolve and render the entry.
- WHEN the user opens `/knowledge/<missing-slug>` THE SYSTEM SHALL
  return 404.
- WHEN the user opens `/knowledge` THE SYSTEM SHALL render a
  kn-card for every entry the entry loader can resolve, AND the
  `<N> entries in the corpus` metric SHALL match that count.
- WHEN the user opens `/agents` THE SYSTEM SHALL render the
  page-hero `<h1>` as `Agents` (not `Sessions`).
- WHEN the user opens `/agents/proposals` (and other sub-routes)
  THE SYSTEM SHALL continue to render `Agents · <Sub>` as today.
- WHEN the SSE toolbar fragment is requested with `?view=blocked`
  THE SYSTEM SHALL render the toolbar with the Blocked tab active.
- WHEN a markdown table cell contains `\\|` THE SYSTEM SHALL
  interpret the leading `\\` as a literal backslash and the
  trailing `|` as a column separator (i.e. the cell ends at `\`).
- WHEN a markdown table cell contains `\|` THE SYSTEM SHALL render
  the cell text with a literal `|` (existing behavior preserved).

## Risks

- **Knowledge corpus count jump.** Going from 15 → ~30+ entries
  surfaces entries that have never been web-visible. Some may have
  malformed frontmatter or no title. Mitigation: `readEntry`'s
  existing `humanize(slug)` fallback handles missing titles; the
  card renderer tolerates missing description.
- **Dir-style entry kind ambiguity.** A `context/<slug>/spec.md`
  entry has frontmatter that may declare a different `type` than
  "context". The current loader uses the directory name as kind
  regardless. Mitigation: keep the directory-as-kind convention
  for consistency with the listing.
- **Two-level nested walk** could find entries that aren't meant
  to be web-visible (e.g. cached snapshots, working drafts).
  Mitigation: only descend into directories that themselves
  contain at least one `.md` file directly. Skip directories whose
  names start with `.` or `_`.
- **Adding `view` query-param to fragment URL** could break any
  CDN/cache key that wasn't expecting it. Mitigation: low risk —
  no CDN in front of the fragment endpoint, and the URL is opaque
  to clients today.
- **The "Agents" rename** might surprise users who associated the
  page hero with the sessions content. Mitigation: minor — the
  sessions content (metric strip, sessions list) stays exactly the
  same. Only the `<h1>` and `<title>` text change.

## Validation

- Manual: `go install ./cmd/hero` then
  `hero serve --port 7459 --no-watch`.
- `curl /knowledge` and count `class="kn-card"` matches: should be
  ~30+ (matches `find .hero/knowledge -name '*.md' | wc -l` minus
  any non-entry files like calibration.json).
- `curl /knowledge/dev-workflow` → 200 with rendered detail.
- `curl /knowledge/project-overview` → 200 ditto.
- `curl /knowledge/architecture-overview` → 200 ditto.
- `curl /knowledge/temporal-supersession-pattern` → 200 ditto.
- `curl /knowledge/never-existed-slug` → 404.
- `curl /agents` and grep `<h1 class="page-title">Agents</h1>` →
  one match.
- `curl /agents/proposals` and grep `Agents · Proposals` → still
  one match.
- `curl '/work/_fragments/toolbar?view=blocked'` (or whatever the
  fragment path is) and grep `view-tab active">Blocked` → one
  match.
- `go build ./...` and `go test ./...` both pass.

## Kickoff

**Status: delivered 2026-05-18.** All six fixes shipped.
`go build ./...` and `go test ./...` both green. Live verification
on port 7459:

- **Knowledge dir-style + nested resolution:** `/knowledge/dev-workflow`,
  `/knowledge/project-overview`, `/knowledge/architecture-overview`,
  `/knowledge/temporal-supersession-pattern` all return 200 (were
  404). `/knowledge/active-context-management-design` also resolves
  (notes/ dir-style). `/knowledge/nonexistent-slug-xyz` → 404.
- **Corpus count corrected:** `/knowledge` shows **36 kn-cards**
  with metric-strip reading `36 entries` (was 15).
- **`/agents` root title:** `<h1 class="page-title">Agents</h1>`
  (was "Sessions"). `/agents/proposals` still
  `Agents · Proposals`.
- **SSE toolbar `?view=` honored:** `/api/work/toolbar?view=blocked`
  → `view-tab active">Blocked`, `?view=kanban` → Kanban,
  `?view=graph` → Graph, no param → Horizons (default preserved).
- **Escape state machine:** `TestSplitTableRow_EscapeStateMachine`
  covers `\|`, `\\|`, `\\\|`, `\\\\`. Existing
  `TestRender_TableEscapedPipe` and `TestSplitTableRow_EscapedPipe`
  still pass.
- **toolbar zero-value contract documented:**
  `TestRegister_ViewToolbarEmptyActiveRendersNoActiveTab` pins the
  no-active-tab fallback. `toolbarData.Active` doc comment lists
  valid values.

**Pick up at: file v6 follow-ups** in
[hero-surface-polish](../../initiatives/hero-surface-polish/spec.md)
as they surface. Known carry-overs from this delivery:

- **Loose top-level knowledge entries excluded from `/knowledge`
  index.** `knowledge/<slug>.md` (depth-1, e.g.
  `hero-native-harness-contract.md`,
  `harness-instruction-file-survey.md`) still resolve via
  `/knowledge/<slug>` but don't appear in the browse listing.
  Decide whether to hoist them into a "(root)" pseudo-kind or
  normalize the layout.
- **No SSE toolbar subscriber today.** The `?view=` contract is
  correct but `workScript` only listens for `roadmap | blocked |
  shipped` events, not `toolbar`. When a `toolbar` subscriber is
  added, the JS needs to pass `?view=` from the page's active tab.
- **Knowledge corpus dedup is per-kind only.** Same slug under two
  kinds yields two entries; `LoadEntry` returns whichever the walk
  emits first. v6 could add an explicit cross-kind precedence rule.
- **Depth-2 limit is hardcoded.** A `rules/<area>/<topic>/<rule>.md`
  shape would silently 404. Decide whether to bump or alert.
- **`/agents` sub-nav "Sessions" tab still active on root.** The
  `<h1>` now says "Agents" but the highlighted sub-tab is still
  "Sessions". Consider relabeling the tab or restructuring nav.
- **`renderToolbar` helper.** Four call sites still build
  `toolbarData{...}` inline. Could extract a one-liner helper that
  asserts `Active != ""` once usage stabilizes (the spec's
  "panic on empty" option, deferred from v5).

Established patterns (don't re-litigate):
- shell.RegisterHome with Items: []shell.ItemRoute for sub-routes
- Each home in internal/serve/pages/<home>/ owns its page.go +
  templates/ + data/
- Shell-owned shared fragments
- internal/serve/mdrender for markdown rendering
- internal/serve/chat is forbidden from page.go imports — homes
  consume it via a Deps.* function injected by server.go
- coming-soon stub is the shared template for sub-routes without
  real content
