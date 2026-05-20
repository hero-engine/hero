---
type: feature
status: planning
tags: [serve, dashboard, ui, now-page, work-page, activity-feed, themes]
relates-to: [hero-serve-multi-project, knowledge-flywheel]
created: 2026-05-19
---
# hero serve dashboard redesign — Now and Work pages

## Context

The `hero serve` dashboard is the operator's daily landing surface. After
a productive day — 20+ specs touched, decisions captured, peer activity,
commits, agent sessions — the user reopens `/now` and `/work` expecting
to *see what happened* and *what to look at next*. Today they get the
opposite: a wall of empty-state tiles, configuration prompts, and a
chat input that occupies most of the viewport. The real activity is
buried below the fold or absent entirely.

This is a mission alignment failure. Hero's job is to *make the next
session start smarter than the last one ended*. The dashboard headline
is the most expensive surface Hero owns; it currently spends that
budget on prompts to configure things instead of compounding what just
happened.

Verified failure modes (against the running daemon and the code):

**Now page (`/now`)** — handler in `internal/serve/server.go:317`, page
assembled from fragments in `internal/serve/pages/now/templates/page.html`:

- The lead headline reads "no agent running · since 19h ago" — accurate
  but useless as a lead.
- Four-tile metric strip ("2 specs shipped this week," "0 commits last
  7 days," "— longest open spec," "— your committed specs") reads as
  empty/stale even on heavy-activity days. Sources:
  `internal/serve/pages/now/data/metrics.go`.
- "Needs your input" inbox is empty even when real signals exist
  (peer handoffs, agent proposals). Source:
  `internal/serve/pages/now/data/inbox.go`.
- The "Tell Hero what to do next" chat input, "Install hero-code"
  panel, and quick-command chips consume ~70% of the viewport.
  Template: `internal/serve/pages/now/templates/quicklaunch.html`.
  Actual work substance gets ~10%.

**Work page (`/work`)** — handler in `internal/serve/server.go:333`,
page in `internal/serve/pages/work/templates/page.html`:

- Default tab is "This sprint" — renders "no active sprint configured ·
  configure sprint in hero.json" as the *headline* for a user who has
  never run sprints. Source: `internal/serve/pages/work/data/metrics.go`.
- The strip ("22 specs delivering · 0 days remaining · 0 blocked · No
  active sprint · 225 specs total") is four-fifths sprint-shaped numbers
  that are meaningless without sprint config.
- Horizons / Kanban / Graph / Blocked tabs (the *good* structural
  views) are buried under sprint noise.

### Why now

This spec runs alongside `hero-serve-multi-project` (in flight,
`.hero/planning/features/hero-serve-multi-project/spec.md`) which is
introducing `/p/<slug>/...` URL routing and an aggregate `/p/all/...`
mode. The two redesigns share the same page handlers and templates;
deciding the Now/Work layout now means multi-project lands clean
instead of dragging old empty-state UX through the routing migration.

## Kickoff

Replaces the empty-tile / sprint-headline Now and Work pages with an
activity-feed-led layout and rolling windows so a heavy-activity day
looks heavy, not empty.

**Status:** planning — spec just landed, no code yet.

**Pick up at:** start with the Now page activity feed since it carries
the biggest visible win. Add `internal/serve/pages/now/data/activity.go`
that reads recent graph events (spec status transitions, decisions,
notes, conventions, peer calls, commits, agent sessions) and a matching
`activity.html` fragment. Wire it into `page.html` as the new first
section, above the in-flight strip.

→ `.hero/planning/features/hero-serve-dashboard-redesign/spec.md`

**Files:** `internal/serve/pages/now/templates/page.html`, `internal/serve/pages/now/data/metrics.go`, `internal/serve/pages/work/templates/page.html`, `internal/serve/pages/work/data/metrics.go`, `internal/serve/shell/templates/tabbed-metric-strip.html`
**Skip:** redesigning Knowledge / People / Agents pages, fixing the data bugs (0 commits / 2 shipped / empty inbox / install panel state) — those land separately.

## Goal

After this ships, an operator who landed on `/now` or `/work` after a
productive work session sees the *substance of that session* above the
fold, with zero configuration required. Specifically:

- The Now headline is an **activity feed** of recent graph events,
  populated by default with a rolling 7-day window. Empty only when
  the project is genuinely empty.
- The Now page surfaces an **in-flight strip** of currently-active
  specs, a **"Hero noticed"** themes section (auto-detected work
  clusters), and a real **Needs your input** inbox — in that order,
  before the chat input.
- The chat input is a single-line widget at the bottom of the page.
  It shrinks to roughly 10% of vertical real estate.
- The Work page's default tab is **"This week"** (Mon–Sun rolling
  window). Sprint-specific UI renders only when `sprint:` is
  configured in `hero.json`.
- Work-page metric tiles are rolling-window counts ("Touched this
  week," "Shipped this week," "Started this week," "Stale >14d") that
  always have meaningful values without configuration.
- Both pages render cleanly under the in-flight `/p/<slug>/...` and
  `/p/all/...` URL scheme. Aggregate mode merges activity across
  projects.

## Approach

### Design thesis

**Surface activity, not configuration.** Empty states are acceptable
for genuinely-empty projects. For active projects, configuration
prompts must yield to real signal. The 70/30 viewport split flips.

**Drop the sprint metaphor as the default.** Hero's primary user is a
solo continuous-flow operator without a tracker. For them:

- Rolling activity windows (today, this week Mon–Sun, this month)
  replace planned-capacity sprints. They populate automatically and
  require zero config.
- Hero-detected themes extend the existing `knowledge-flywheel`
  pattern detection to work clusters ("you touched 5 specs in
  `internal/serve/` this week," "3 recent decisions about CLI
  lifecycle").
- `sprint:` config stays in `hero.json` as an explicit opt-in for
  teams that run planned cadences. Operators without sprint config
  never see sprint UI.

### Architectural choices

- **Activity feed is a new data source**, not a recomposition of
  existing sections. Read graph events via the existing graph DB
  query layer used by `hero list` / `hero recap`. Materialize into a
  page-local struct in `internal/serve/pages/now/data/activity.go`.
  Cache per-request (no global cache yet); revisit if perf becomes a
  concern.
- **Themes use `knowledge-flywheel` detection**, extended to work
  clusters. Add a `work-cluster` cluster type. Surface only with
  high confidence (initial threshold: at least 3 related items in
  the window). Conservative on purpose — bad clusters are worse
  than no clusters.
- **Sprint UI is opt-in.** A single `hasSprintConfig(cfg)` check
  gates whether the Sprint tab and sprint-shaped metrics render.
  When unset, "This week" is the only top-tab and metrics are
  rolling-window.
- **Component contracts must be state-aware.** The chat input
  fragment (`internal/serve/shell/templates/chat-input.html` and the
  shell `empty-state-notice.html` it composes with) must render
  cleanly in both adapter-connected and adapter-missing states, with
  no overlap or contradictory chips. The redesign defines the
  contract; the *bug* of misreading adapter state is fixed
  separately under a `.hero/planning/bugs/` spec.
- **Forward-compat with multi-project routing.** All new sections
  read project context from the handler-provided `Deps` (project
  root, hero dir) rather than from `s.projectRoot` directly, so
  they slot into `/p/<slug>/...` without rework. Aggregate `/p/all`
  mode merges feed entries across projects and detects cross-project
  themes.
- **Forward-compat with the Project section.** The redesign keeps
  Now focused on *activity* and Work focused on the *spec catalog*.
  Per-project utility info (config, paths, integrations, health)
  belongs in the dedicated Project page and is explicitly out of
  scope here.

### Patterns to follow

- Follow the existing page composition pattern: one fragment per
  section, mounted from `page.html`, each wrapped in a `<section
  id="now-<slug>">` so SSE swaps replace it whole. Existing fragments
  (`inbox.html`, `plate.html`, `agents.html`, `changes.html`,
  `quicklaunch.html`) are the canonical model.
- Follow the Work page convention from
  `internal/serve/pages/work/templates/page.html` — sections are
  composed by `page.html` and share the shell-owned `view-toolbar`.
- Match the data-package layout (`pages/<page>/data/<section>.go`)
  with sibling `_test.go` files. See
  `internal/serve/pages/now/data/changes.go` and `changes_test.go`
  as the pattern.

### Patterns to avoid

- Do not bake activity-feed rendering into the existing metrics
  package — the data shapes are unrelated.
- Do not introduce a global activity cache yet. Materialize per
  request; defer caching/pre-computation until the feed exists and
  we can measure.
- Do not delete sprint code paths. Make them conditional. Existing
  users with `sprint:` configured must continue to work unchanged.

## Acceptance Criteria

- THE SYSTEM SHALL render an activity feed as the first content section
  on the Now page, populated with graph events from the last 7 days by
  default.
- THE SYSTEM SHALL include in the activity feed at minimum these event
  kinds: spec status transitions, decisions captured, notes captured,
  conventions detected, peer calls, agent sessions, commits.
- WHEN a user clicks an activity-feed entry THE SYSTEM SHALL navigate
  to the underlying spec, decision, note, or commit detail page.
- WHEN multiple commits land on the same spec within a single feed
  window THE SYSTEM SHALL collapse them into a single grouped entry
  showing the count.
- WHERE the activity-feed time-window filter is set to `today`, `week`,
  `month`, or `all` THE SYSTEM SHALL filter entries to that window.
- THE SYSTEM SHALL render an in-flight strip on the Now page listing
  specs currently in `planning`, `delivering`, or `investigating` with
  a last-touched timestamp.
- THE SYSTEM SHALL render a "Hero noticed" themes section on the Now
  page showing auto-detected work clusters from the rolling window.
- IF no theme reaches the confidence threshold THEN THE SYSTEM SHALL
  omit the "Hero noticed" section entirely rather than rendering an
  empty header.
- THE SYSTEM SHALL render the chat-input widget as a single horizontal
  row at the bottom of the Now page, with quick-command chips inline.
- WHERE a chat adapter is connected THE SYSTEM SHALL render the input
  field; WHERE no adapter is connected THE SYSTEM SHALL render the
  install-prompt fragment in the same slot with no layout overlap.
- THE SYSTEM SHALL render "This week" as the default top tab on the
  Work page, replacing "This sprint" as the default landing.
- WHERE `sprint:` IS configured in `hero.json` THE SYSTEM SHALL render
  a "Sprint" tab alongside "This week."
- IF `sprint:` is not configured THEN THE SYSTEM SHALL NOT render any
  sprint-shaped tab, metric, or empty-state prompt on the Work page.
- THE SYSTEM SHALL render four rolling-window metric tiles on the Work
  page: "Touched this week," "Shipped this week," "Started this week,"
  "Stale (no touch >14d)."
- WHEN a user clicks a Work-page metric tile THE SYSTEM SHALL filter
  the spec list below the tiles to that window/category.
- THE SYSTEM SHALL render a Themes row on the Work page showing
  detected work clusters from the rolling window.
- THE SYSTEM SHALL preserve the existing Horizons, Kanban, Graph, and
  Blocked views; they remain reachable via the view-toolbar.
- THE SYSTEM SHALL render the Now and Work pages cleanly under the
  `/p/<slug>/now`, `/p/<slug>/work`, `/p/all/now`, and `/p/all/work`
  routes introduced by `hero-serve-multi-project`.
- WHILE the active route is `/p/all/...` THE SYSTEM SHALL aggregate
  activity feed entries across all registered projects.

## Changes

1. **Add the activity-feed data layer for Now.**
   - New file `internal/serve/pages/now/data/activity.go` with an
     `Activity` struct (entries, window, total counts) and a
     `LoadActivity(deps Deps, window Window) (Activity, error)`.
   - Query the graph DB (use the same query layer as `hero recap` /
     `hero list`; see callers of `graph.Query` in `internal/cli/`
     for the canonical pattern).
   - Define an `Entry` type with fields: `Kind` (spec-transition,
     decision, note, convention, peer-call, agent-session, commit),
     `Title`, `Timestamp`, `Link`, `Subtitle`, `GroupCount`.
   - Implement grouping: when ≥2 entries share `(Kind, ParentSlug)`
     within the window, collapse to one entry with `GroupCount > 1`.
   - Sibling `activity_test.go` covering empty window, single-kind
     window, grouping, and aggregate-mode merge across projects.

2. **Add the activity-feed template and wire it in.**
   - New file `internal/serve/pages/now/templates/activity.html`
     rendering the feed with a window-filter pill row (today / week
     / month / all) and a chronological entry list.
   - Update `internal/serve/pages/now/templates/page.html` to render
     `activity.html` as the first section, above the existing
     `inbox`/`plate`/`agents`/`changes` blocks.
   - Wrap in `<section id="now-activity">` for SSE swap.

3. **Add the in-flight strip.**
   - New file `internal/serve/pages/now/data/inflight.go` plus
     `inflight_test.go` returning specs whose status is in
     `{planning, delivering, investigating}` ordered by
     last-touched timestamp.
   - New template `internal/serve/pages/now/templates/inflight.html`.
   - Mount in `page.html` between activity and inbox.

4. **Add the Hero-noticed themes section to Now.**
   - New file `internal/serve/pages/now/data/themes.go` plus
     `themes_test.go`. Call into the existing knowledge-flywheel
     cluster detection in `internal/knowledge/flywheel/` (verify
     exact package path during implementation). Add a
     `work-cluster` cluster kind that groups by file-path prefix
     and decision tag.
   - Require minimum 3 related items in the window before surfacing
     a theme.
   - New template `internal/serve/pages/now/templates/themes.html`.
   - Mount in `page.html` after inflight, before inbox.

5. **Reframe and tighten "Needs your input."**
   - Update `internal/serve/pages/now/data/inbox.go` to source from
     real signals: agent proposals (existing
     `s.snapshotProposals`), peer handoffs awaiting accept (read
     from `hero handoff status` data layer), decisions awaiting
     confirmation. The *data wiring* bug fix is out of scope; the
     redesign defines the contract.
   - Inbox stays in `page.html` but moves below themes.

6. **Shrink the chat input.**
   - Update `internal/serve/pages/now/templates/quicklaunch.html`
     to render as a single-row, bottom-anchored widget.
   - Move quick-command chips inline with the input.
   - Coordinate with `internal/serve/shell/templates/chat-input.html`
     and `empty-state-notice.html` so the adapter-connected and
     adapter-missing states render in the same slot without
     overlap or contradictory chips.

7. **Replace Now metric strip with windowed activity counts.**
   - Update `internal/serve/pages/now/data/metrics.go` to compute
     rolling-window counts (specs touched, specs shipped, decisions
     captured, notes captured) — same shape as today's four-tile
     strip but populated from real graph events.
   - Update `internal/serve/shell/templates/tabbed-metric-strip.html`
     (or the page-specific call site) so tiles are clickable and
     filter the activity feed.

8. **Drop "This sprint" as the default Work tab.**
   - Update `internal/serve/pages/work/data/types.go` and the tab
     selector logic in the page handler to default to "This week."
   - Add a `hasSprintConfig(cfg)` helper (likely in
     `internal/serve/pages/work/data/metrics.go` or a sibling
     `sprint.go`) reading `sprint:` from `hero.json`.
   - Gate the Sprint tab rendering on `hasSprintConfig` in
     `internal/serve/pages/work/templates/page.html` and the
     shell's tab selector.

9. **Replace Work metric tiles with rolling-window metrics.**
   - Update `internal/serve/pages/work/data/metrics.go` to compute:
     touched this week, shipped this week, started this week, stale
     (no touch >14d).
   - Each tile carries a filter key wired to the spec list below.
   - Keep the sprint-shaped tile shapes available behind
     `hasSprintConfig` for users who opt in.

10. **Add Themes row to Work.**
    - New file `internal/serve/pages/work/data/themes.go` plus
      `themes_test.go` reusing the work-cluster detection from
      step 4 but scoped to the rolling window of the active tab.
    - New template `internal/serve/pages/work/templates/themes.html`
      mounted between the metric strip and the Roadmap block.

11. **Move the "Plan sprint" button.**
    - Currently a top-right primary action. Move into a secondary
      slot (overflow menu or footer of the Sprint tab) so users
      without sprint config never see it as a primary CTA.
    - Verify exact current location in
      `internal/serve/pages/work/templates/page.html` and the
      shell-owned `page-hero.html`.

12. **Forward-compat with multi-project routing.**
    - All new section data loaders must accept project context via
      the existing `Deps` struct (project root, hero dir). Do not
      reference `s.projectRoot` directly inside new code.
    - Add aggregate-mode support to `activity.go` and `themes.go`:
      when `Deps` indicates `all`, fan out across registered
      projects (use the existing registry from
      `internal/serve/registry.go`).
    - Coordinate with `hero-serve-multi-project` on the exact
      `Deps` shape — that spec is in flight and may evolve the
      contract.

13. **Tests.**
    - Unit tests for every new `data/*.go` file, covering empty,
      populated, grouped, and aggregate cases.
    - Update `internal/serve/pages/now/page_test.go` and
      `internal/serve/pages/work/page_test.go` to assert section
      ordering and conditional rendering of sprint UI.
    - Add a shell-render test covering chat-input state-aware
      rendering (no overlap when adapter missing).

## Boundaries

Explicitly out of scope:

- **Fixing the data bugs that produce empty/stale numbers today.**
  Specifically: 0 commits last 7 days, 2 shipped this week looking
  wrong, empty inbox, install panel rendering while connected, chat
  input rendering bugs. These are handled by separate bug specs in
  `.hero/planning/bugs/` from the parallel diagnose pass. This
  redesign assumes those land.
- **The dedicated Project section page.** Per-project info, health,
  switching, integrations belong there. Separate spec, comes next.
- **Knowledge, People, and Agents page redesigns.** The same design
  thesis applies, but those are separate specs.
- **Sprint planning UI for teams that opt in.** Keep the existing
  sprint UI working when `sprint:` is configured; do not redesign
  the sprint flow itself.
- **Team server / multi-human work.** Tracked in the in-flight
  `hero-team-server` spec.
- **User identity / "you" attribution.** "Your commits" / "your
  committed specs" / "needs your input" all depend on a coherent
  definition of user identity. That bug is tracked separately and
  this spec assumes it lands.
- **The graph-query layer itself.** Use whatever the existing
  `hero list` / `hero recap` callers use. Don't re-architect.

## Risks

- **Activity-feed density on heavy days.** A 7-day window on a busy
  project could produce 100+ entries. Mitigation: grouping
  (step 1), conservative defaults, density controls (compact /
  expanded toggle, optional). Watch the heaviest known project
  (this repo) during rollout.
- **Bad themes worse than no themes.** Conservative threshold (≥3
  items) and silent omission when no theme reaches confidence.
  Bias toward false negatives. Open question: do we want a
  "diagnostic" mode that surfaces low-confidence clusters during
  development? Defer until we see the feature live.
- **Migration UX for sprint users.** Existing users with `sprint:`
  in `hero.json` must not regress. The `hasSprintConfig` gate
  ensures the Sprint tab still appears for them. Test explicitly.
- **Coordination with `hero-serve-multi-project`.** That spec is
  in flight and is evolving the `Deps` contract and URL scheme.
  Coordinate handoff timing — ideally multi-project lands first,
  this spec rebases on the final shape. Worst case, both land in
  parallel and we reconcile at merge.
- **Performance of activity-feed graph queries.** Per-request
  materialization is fine for solo operators, less fine for big
  team projects. Mention but don't over-engineer; revisit if SSE
  tick rate degrades or page render exceeds 500ms.
- **Chat-input state-aware contract relies on a separate bug fix.**
  If the adapter-state bug doesn't land before this redesign, the
  chat-input slot may still render incorrectly. Mitigation: define
  the contract in templates so the bug fix is a *state read* not
  a *template change*.

## Validation

- All new and updated data loaders covered by unit tests with the
  patterns in step 13.
- Page tests assert section ordering and conditional sprint
  rendering on Now and Work.
- Manual verification on this repo (a heavy-activity project):
  - Open `/now` after a 20-spec day. The activity feed shows
    representative events. The fold contains feed + in-flight +
    themes — not chat input.
  - Open `/work`. Default tab is "This week." Four tiles all show
    nonzero values. No sprint UI visible (no `sprint:` configured).
  - Add `sprint:` to `hero.json`. Reload `/work`. The Sprint tab
    reappears. The existing sprint UI works unchanged.
- Manual verification on a fresh project:
  - Open `/now` immediately after `hero scan`. Activity feed
    shows the scan event(s). Themes section omitted (no
    threshold met). Inbox shows real signals if any.
- Cross-check the multi-project routes:
  - `/p/<slug>/now`, `/p/<slug>/work` render identically to the
    direct routes.
  - `/p/all/now` shows merged activity across projects.
- Lighthouse / dev-tools spot check: first-paint of `/now` should
  not regress from the current baseline. Activity-feed query
  should complete within the existing page-render budget.

## Relations

- `relates-to`: `hero-serve-multi-project` — shares page handlers
  and templates; coordinate URL routing and `Deps` shape.
- `relates-to`: `knowledge-flywheel` (skill) — extend pattern
  detection to work clusters.
- Depends conceptually (not blocking) on the parallel
  `.hero/planning/bugs/` specs for: install panel adapter-state,
  empty inbox wiring, 0-commits metric source, "shipped" metric
  accuracy, user-identity definition. Redesign treats these as
  assumed fixes; structure is independent.
