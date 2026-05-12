---
title: "Satellite Scope Extras — Why Traversal, Spec Move, Cloud Filtering"
type: feature
status: planning
priority: medium
horizon: now
tags: [monorepo, satellites, scoping, traversal, cloud]
relations:
  - target: monorepo-satellite-installs
    kind: parent
  - target: satellite-corpus-integration
    kind: parent
---

# Satellite Scope Extras — Why Traversal, Spec Move, Cloud Filtering

## Problem

The previous two specs ([monorepo-satellite-installs](../monorepo-satellite-installs/spec.md), [satellite-corpus-integration](../satellite-corpus-integration/spec.md)) made subproject scope a first-class corpus facet on the local CLI. Three real follow-ups remain:

1. **`hero why` is scope-blind.** Origin-chain traversal renders hops with title and type but doesn't surface the subproject. When you trace why a spec exists from inside a satellite, you can't tell at a glance which upstream hops are in your scope vs. cross-scope. The data is there (specs carry `subproject:` now); the renderer ignores it.

2. **No re-scoping command.** `hero spec stamp-scope` exists, but it only auto-detects from cwd or the spec's path-under-a-declared-prefix. There is no way to say "move this spec from scope `engines/mlx` to scope `engines/cuda`" — which is what happens when work crosses subproject boundaries during refactoring. Today the only path is hand-edit frontmatter and re-index.

3. **Cloud is fully scope-blind.** The cloud sync stores spec metadata but has no `subproject` column, doesn't accept a scope filter on list endpoints, and the dashboard UI has no scope facet. A team that consolidates a monorepo gets a great local experience and a flat cloud view.

## Goal

Three small, independent additions that close the remaining scope-aware gaps:

- `hero why` renders each hop with its subproject and visually marks in-scope vs. out-of-scope hops when invoked from a scoped cwd.
- `hero spec move <slug> --to-scope <new-scope>` updates the spec's `subproject:` frontmatter, optionally relocates the file under a scope-prefixed path, and re-indexes — all in one command.
- Cloud database, API, and dashboard UI gain a `subproject` dimension: column on the specs table, query-param filter on the list endpoint, sync round-trips it from the local frontmatter, and the dashboard adds a subproject facet/filter on the specs page.

**Mission-fit.** Without these, a developer who works in a monorepo daily hits a friction wall at the boundaries: the moment they trace a why, re-scope a spec, or open the cloud dashboard, scope evaporates and they're back to grepping. Each item is small, but together they make scope a continuous experience instead of a leaky one.

The non-goal is per-scope traversal *ranking* (boosting in-scope nodes in the chain). That requires a model of "what does in-scope-ness mean for a non-spec node like a commit or a person?" — open enough that this spec leaves it for a follow-on once we see how scope-aware traversal actually gets used.

## Design

### 1. `hero why` scope annotation

Add a `Subproject string` field to `traversal.Hop`. Populate it from the matching spec's frontmatter at hop-resolve time (the resolver already loads the spec node — this is one extra column in the lookup query, or a post-resolution stamp from the index).

The text renderer adds a subproject indicator to each hop line, with visual distinction for in-scope hops:

```
Why does spec `cuda-fp16-fallback` exist?

  feature  cuda-fp16-fallback        [scope: engines/cuda]  ← in-scope
  feature  cuda-engine               [scope: engines/cuda]  ← in-scope
  feature  shared-numerics           [scope: engines/shared]
  feature  <vision-spec>  [scope: (root)]
```

When invoked from a scoped cwd, hops in the active scope are marked (e.g. with `← in-scope` or a leading `*`). When invoked at root or with `--subproject all`, all hops show their scope without highlighting (since there's no active scope to compare against).

A `--subproject <name|all>` flag on `hero why` overrides the cwd default, same as `hero list` and `hero search`. This lets the user trace from any scope's perspective without `cd`-ing.

The JSON renderer adds the `subproject` field to each hop unconditionally so downstream tools can consume it.

### 2. `hero spec move` command

A new subcommand under `hero spec`:

```
hero spec move <slug> --to-scope <new-scope> [--relocate] [--dry-run]
```

Behavior:

1. **Resolve the spec** by slug. Refuse cleanly if the slug doesn't exist.
2. **Validate the target scope** against `subprojects.json`. The new scope must be either `""` / `root` (re-rooting) or a declared subproject path. Refuse otherwise with a clear error and the list of available scopes.
3. **Update `subproject:` in frontmatter.** Same write helper as `stamp-scope` (replace-or-append in YAML frontmatter).
4. **Optionally relocate the file.** Without `--relocate`, the file stays at its current path — only the frontmatter changes. With `--relocate`, the file moves to a scope-prefixed path under `.hero/planning/<bucket>/<scope>/<slug>/spec.md` (the same path shape the migration command produces). `git mv` is used when the file is tracked.
5. **Re-index.** Trigger `hero index --if-stale` so the new scope reaches the search/list filter immediately.
6. **Emit an event.** Append a `subproject_changed` feed event with old + new scope so the activity feed records re-scoping.

`--dry-run` reports what would change without writing.

**Why a single command and not just `mv` + `stamp-scope`?** Because the three steps (frontmatter, optional file move, re-index, event) are always done together. Splitting them is the kind of API design that produces "I forgot to re-index" bugs.

**Why is `--relocate` opt-in instead of default?** Most re-scoping is metadata-only — the spec file lives at a slug-derived path that's already correct. Relocating is occasionally useful (e.g. when a bucket changes too: feature → bug). Defaulting to no-relocate keeps the operation reversible and the diff small.

### 3. Cloud scope plumbing

**Database.** Add a migration (version 8) that:

```sql
ALTER TABLE specs ADD COLUMN IF NOT EXISTS subproject TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_specs_subproject
  ON specs (org_id, subproject) WHERE subproject != '';
```

Index is org-scoped because cloud queries always filter by org via RLS — same pattern as `idx_specs_org`.

**Sync.** The CLI's `hero sync` payload already carries arbitrary spec frontmatter via the raw_content field; the cloud-side sync handler unpacks structured fields out of the parsed spec. Add `Subproject string` to `cloud/store.Spec` and to the upsert SQL so the cloud round-trips it. The CLI's local `Spec.Subproject` field is already populated, so the sync side just needs to include it in the request body.

**API.** `handleListSpecs` accepts a new `?subproject=<name>` query parameter, threaded through to `ListRepoSpecs(ctx, repoID, specType, status, subproject)`. The org-wide endpoint `ListOrgSpecs` gets the same filter. `?subproject=all` is treated as "no filter" for symmetry with the CLI.

**API response shape.** `Spec` JSON gains `subproject` field (omitempty). Existing clients that ignore unknown fields are unaffected.

**Dashboard UI.** The Preact SPA adds:
- A `subproject` field on the spec card (small text, only shown when non-empty).
- A subproject filter pill in the spec-list view's filter bar, listing the distinct subprojects for the current repo. "All scopes" is the default.
- Filter state persists in the URL hash (same pattern as `?type=` and `?status=`) so a teammate can share a filtered view.

The repo's distinct-subproject list comes from a small new endpoint `/repos/:repo_id/subprojects` that runs `SELECT DISTINCT subproject FROM specs WHERE repo_id = $1 AND subproject != '' ORDER BY 1`. Cheap, indexed, no need for a dedicated table — subprojects are derived from the corpus, not master-data.

### Design decisions

**Why is the cloud column a flat string instead of a foreign key to a `subprojects` table?** Because `subprojects.json` is the source of truth, lives in git per-repo, and is not master-data the cloud should own. A FK relationship would force the cloud to learn about `subprojects.json` and stay in sync with what the repo declares — extra work for no benefit. A denormalized string column is the same shape as `tags` and `tracker_id` and works the same way.

**Why a separate `/subprojects` endpoint instead of computing distinct values client-side?** A repo with thousands of specs shouldn't ship the full list to the client just so the dropdown can compute distinct values. The endpoint is a cheap indexed query, the response is small (typically <10 entries per repo), and clients can cache it.

**Why doesn't `hero spec move` accept multiple slugs?** Single-spec moves are typically thoughtful operations driven by a refactor. Bulk re-scoping is a different kind of operation (often "all specs that touched <files>") that deserves its own command if/when it's needed. Forcing every consumer of `move` to either iterate or pass a list muddles the common case.

**Why does `--to-scope ""` re-root a spec instead of being an error?** Because moving a spec out of all subprojects is a real operation — promoting a spec from scope-local to repo-wide. Treating empty as a sentinel for "root scope" makes this expressible without a separate `--unscope` flag.

**Why does `hero why` mark in-scope hops instead of filtering them out?** Because the chain *needs* the out-of-scope hops to be coherent — a why-chain that skips hops becomes a why-misdirection. Marking in-scope hops keeps the visual scan fast without removing context. Filtering is what `--subproject` already does on `hero list`; traversal is a different shape.

**Why does the cloud return `subproject` even when the client didn't filter?** Because the dashboard renders it on the card and the same response shape powers both filtered and unfiltered views. Conditional inclusion would force two render paths for no real win.

## Acceptance Criteria

- THE SYSTEM SHALL add a `Subproject string` field to `traversal.Hop`, populated from the underlying spec's `subproject` frontmatter at resolve time.
- WHEN `hero why` renders a trace in text format from a scoped cwd THE SYSTEM SHALL annotate each hop with its scope and visually distinguish in-scope hops from out-of-scope hops.
- WHEN `hero why` renders a trace in JSON format THE SYSTEM SHALL include the `subproject` field on each hop unconditionally.
- THE SYSTEM SHALL accept a `--subproject <name|all>` flag on `hero why` that overrides the cwd-derived active scope.
- THE SYSTEM SHALL provide a `hero spec move <slug> --to-scope <new-scope>` subcommand that updates the spec's `subproject:` frontmatter to the new value.
- IF `--to-scope` is non-empty AND the value is neither "root" nor a path declared in `.hero/subprojects.json` THEN THE SYSTEM SHALL refuse with an error listing the available scopes.
- WHEN `hero spec move` runs with `--relocate` THE SYSTEM SHALL move the spec file to a scope-prefixed path under `.hero/planning/<bucket>/<scope>/<slug>/spec.md`, using `git mv` when the file is tracked.
- WHEN `hero spec move` completes successfully (and not in `--dry-run`) THE SYSTEM SHALL trigger an `hero index --if-stale` rebuild and emit a `subproject_changed` feed event recording the old and new scope.
- WHEN `hero spec move` runs with `--dry-run` THE SYSTEM SHALL report all changes that would be made without writing or relocating anything.
- THE SYSTEM SHALL add a `subproject` column to the cloud `specs` table via a numbered migration, with a partial index `(org_id, subproject) WHERE subproject != ''`.
- WHEN the CLI syncs specs to the cloud THE SYSTEM SHALL include the `Subproject` value in the payload and the cloud upsert SHALL persist it.
- THE SYSTEM SHALL expose `subproject` on the cloud `Spec` JSON shape (omitempty).
- WHEN `GET /repos/:repo_id/specs?subproject=<name>` is called THE SYSTEM SHALL filter results to specs with that subproject; `subproject=all` SHALL disable subproject filtering.
- THE SYSTEM SHALL expose an endpoint `GET /repos/:repo_id/subprojects` returning the distinct non-empty subproject values for that repo, ordered alphabetically.
- THE SYSTEM SHALL render the spec's subproject (when non-empty) on the dashboard spec card and provide a subproject filter pill in the spec-list view, with "All scopes" as the default option.
- WHEN a subproject filter is applied in the dashboard THE SYSTEM SHALL persist the selection in the URL hash so the filtered view is shareable.

## Changes

### New files
- `internal/cli/spec_move.go` — `hero spec move` subcommand
- `internal/cli/spec_move_test.go` — unit tests for the move command

### Modified files
- `internal/traversal/why.go` — `Hop.Subproject` field; populate in `resolveTarget` and `walkOrigins`
- `internal/traversal/render.go` (or wherever the text/JSON renderers live) — render scope annotations
- `internal/cli/brief.go` — `--subproject` flag on `whyCmd`
- `cloud/store/migrations.go` — migration version 8 (subproject column + index)
- `cloud/store/specs.go` — `Subproject string` field, upsert SQL, list filter parameter
- `cloud/api/specs.go` — `?subproject=` query param threading; new `/subprojects` handler
- `cloud/api/router.go` — register the new subprojects route
- `cloud/web/app.js` — subproject filter UI, render on spec card, URL hash persistence
- `internal/cli/sync.go` (or equivalent) — include `Subproject` in sync payload

## Phasing

### Phase 1 — `hero why` scope annotation
Smallest piece. Adds `Subproject` to `Hop`, populates from spec lookup, annotates renderers, threads `--subproject` flag. Acceptance: `cd engines/mlx && hero why some-spec` shows in-scope vs out-of-scope hops at a glance.

### Phase 2 — `hero spec move` command
Self-contained command with frontmatter rewrite, optional file relocation, re-index, event emission. Acceptance: `hero spec move foo --to-scope engines/cuda --dry-run` previews; without `--dry-run` the spec carries the new scope and the activity feed records the change.

### Phase 3 — Cloud scope column + API + dashboard
Database migration, sync round-trip, list endpoint filter, distinct-subprojects endpoint, dashboard UI filter pill. Acceptance: a teammate looking at the cloud dashboard can filter specs to scope `engines/mlx` from a dropdown, the URL is shareable, and the spec card shows the scope.

## Kickoff

Resume by reading `.hero/planning/features/satellite-scope-extras/spec.md`. Phases 1 (`hero why` annotation) and 2 (`hero spec move`) are local-CLI work; Phase 3 (cloud) touches `cloud/store`, `cloud/api`, and `cloud/web/app.js`. Parent specs that already shipped: [monorepo-satellite-installs](../monorepo-satellite-installs/spec.md) (foundation) and [satellite-corpus-integration](../satellite-corpus-integration/spec.md) (scope flows through corpus).
