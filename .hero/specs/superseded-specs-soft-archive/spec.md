---
title: Superseded Specs — Frontmatter-Based Soft Archive
slug: superseded-specs-soft-archive
type: feature
status: completed
priority: high
tags: [retrieval, search, context-injection, knowledge-graph, drift, v1-v2]
created: 2026-05-30
completed_at: 2026-05-31T01:05:26Z
---

# Superseded Specs — Frontmatter-Based Soft Archive

## Goal

A spec marked superseded by another spec is de-weighted in search, annotated (not dropped) in context-injection, and still reachable from `hero why` traversals — so the next agent session follows the current direction without losing the history of how we got here.

## Kickoff

We keep getting bitten by v1 specs in `.hero/specs/` winning retrieval over their v2 replacements; agents adhere to outdated direction because the cold-start window doesn't know which spec is current.

**Status:** planning — design just landed; nothing implemented.

**Pick up at:** Phase 1 — extend `internal/spec/spec.go` to parse a `superseded_by:` frontmatter field, then extend `internal/index/index.go`'s `specs` table + `BuildNudge` to filter on it. Read this spec, then `internal/spec/spec.go` (Status enum + parseFrontmatter) and `internal/index/index.go` (Search/SearchFiltered + BuildNudge) before writing code.

→ `.hero/planning/features/superseded-specs-soft-archive.md`

## Problem

Hero's mission is *"does this session start smarter than the last one ended."* Right now it's actively making the next session dumber in one specific way: superseded specs are indistinguishable from current specs to the retrieval layer.

Concrete examples from this very workspace:

- `.hero/specs/hero-surface-polish-v1/` and `.hero/specs/hero-surface-polish-v2/` both exist as `status: completed`. Cold-start FTS for "surface polish" returns both; the v1 spec wins by recency or rank-tie because nothing tells the indexer it has been superseded.
- `.hero/specs/hero-v2-system-design/` is the *initiative* that produced almost every "current" pattern in the repo, but it lives next to dozens of v1-era specs (`hero-dashboard-v2`, `v2-agents-and-skills`, `v2-commands`, etc.) that explicitly replaced earlier specs that are still in `.hero/specs/` with no replacement pointer.
- `contentfs-legacy-fallback-removal` documents removing a legacy mode that several still-on-disk v1 specs assume. Agents reading the older specs get instructions that no longer compile.

The model is stateless. The only signals it has about which direction is current are (a) the spec contents and (b) the rank/order in which the retrieval layer surfaces them. Today, both signals point equally at v1 and v2. That is exactly the failure mode Hero exists to prevent.

There is already partial groundwork:

- `spec.Status` defines `StatusSuperseded = "superseded"` (`internal/spec/spec.go:67`).
- `internal/spec/select.go`'s `isClosedStatus` treats `superseded` as closed (excluded from `hero queue` by default).
- `internal/spec/graph_ingest.go` maps a `supersedes:` relation kind to a `supersedes` graph edge.
- `internal/traversal/why.go` walks `supersedes` as an origin edge — so `hero why` already follows the chain when an edge exists.

What's missing — and what this spec adds — is the *frontmatter field that wires it up*. No legacy v1 spec actually carries `status: superseded` or a `supersedes:` relation, because nothing has ever asked the author to set one. The supersede plumbing exists; the on-ramp doesn't.

## Design

### Decision 1 — Frontmatter shape: separate `superseded_by:` field, NOT a `status:` value

The clean shape is a new top-level frontmatter field:

```yaml
superseded_by: hero-surface-polish-v2
```

Status stays whatever it was (`completed` in almost every real case). Reasoning:

- **Lifecycle is orthogonal to genealogy.** "This shipped" (status) and "this was later replaced" (genealogy) are independent facts. A spec can be `completed` and superseded; a spec can be `planning` and superseded by a different direction; a `convention` is normally `active` and gets superseded by a newer convention also marked `active`. Forcing them into one enum loses information.
- **The existing `StatusSuperseded` enum value stays, but as a non-required signal.** Callers that already treat `superseded` as closed (`select.go`, `check.go`, `sync_import.go`, `sitegen.go`) keep working. The new field is the authoritative signal; the old enum value remains valid for the rare case where the user wants both ("yes, also flip the status to keep it out of every closed-filter").
- **Single source of truth for the indexer.** One field, one column, one query predicate — much simpler than "status is superseded OR has a supersedes relation pointing somewhere."
- **Humans editing frontmatter only have to look at one field.** `superseded_by: <slug>` is self-documenting; `status: superseded` requires the reader to scroll down looking for *which* spec replaced it.

The existing `supersedes:` relation (parsed in `spec.go:449`, mapped to a graph edge in `graph_ingest.go:225`) becomes the **inverse view, computed on ingest from `superseded_by`** — so the new spec automatically gets a `supersedes` edge pointing back. Authors only edit one side.

### Decision 2 — Context-injection: ANNOTATE, don't drop

Superseded specs surfaced by retrieval are kept in the output but tagged with a redirect marker:

```
- **hero-surface-polish-v1** (completed) [SUPERSEDED by hero-surface-polish-v2 — follow that one]
```

Reasoning:

- If the user mentioned a v1-era concept by name, dropping the v1 spec leaves the agent confused. Annotating it teaches the agent two things at once: "this is the old answer" and "this is where the new answer lives."
- The annotation is *cheaper* than the model fetching both specs to compare. The marker tells it not to bother reading the old one unless explicitly asked for history.
- Annotation degrades gracefully: a marker the model ignores still gets it pointed at a real, current spec via the title.

Drop behavior (`--exclude-superseded`) is available as an opt-in flag for cases where the caller wants a clean current-direction-only view (e.g. machine-generated documentation indexes).

### Decision 3 — `hero search` UX: dimmed-and-suffixed by default, hidden behind a flag

Default: superseded results render with a `[SUPERSEDED → <slug>]` suffix and rank below non-superseded peers (de-weight factor 0.3× applied to the final score — same multiplier shape as `typeBoost` in `internal/retrieval/retrieval.go`, just an attenuator instead of an amplifier).

Flag: `hero search --include-superseded` returns superseded specs at full weight, with the marker still visible so the human knows what they're looking at.

Hidden-entirely was rejected because (a) `hero why` users sometimes find a current spec and ask "what was this before?" and a discoverable search hit is the most natural answer, and (b) silently hiding things violates the project rule "don't assume — surface tradeoffs."

### Decision 4 — Backfill: hybrid — scripted candidate detection + manual confirmation via `hero supersede`

Pure manual is too much labor for ~160 existing specs. Pure scripted is too risky — false positives mean wrongly hidden current direction.

Phase 0 (one-time, ships with this feature): a `hero supersede --scan` mode that walks `.hero/specs/` and `.hero/planning/`, looks for candidate supersede pairs using these heuristics:

1. **Slug suffix pairs** — `foo-v1` and `foo-v2`, `foo` and `foo-v2`, `foo-old` and `foo`.
2. **In-body "replaces"/"supersedes" mentions** — body text matching `(replaces|supersedes|deprecates) [`a-z0-9-]+` where the target slug exists.
3. **Same-area `completed` clusters** where a newer spec's Changes section overlaps with an older spec's by ≥50% of file paths AND the newer spec post-dates the older by some threshold.

Each candidate is written to `.hero/reports/supersede-candidates.md` for human review. The user runs `hero supersede <old> --by <new>` per pair to confirm.

`--scan --auto` is deliberately NOT offered. The risk of wrongly marking a still-relevant spec as superseded is higher than the cost of clicking through ~100 confirmations once.

### Frontmatter schema change

In `internal/spec/spec.go`:

1. Add `SupersededBy string` field to the `Spec` struct (after `Pinned`).
2. In `parseFrontmatter`, add a case `"superseded_by"` that sets `s.SupersededBy = val`. Validation: trim; warn (not error) if the target slug doesn't exist when `Discover` finishes a full pass — log to stderr in debug builds, surface in `hero check`.
3. Add helper `func (s *Spec) IsSuperseded() bool { return s.SupersededBy != "" || s.Status == StatusSuperseded }` so callers don't have to know about both signals.
4. `SetFrontmatterField` already supports adding arbitrary keys — no change needed for the writer side.

### Indexer change

In `internal/index/index.go`:

1. Add `superseded_by TEXT NOT NULL DEFAULT ''` column to the `specs` table (migration that ALTERs the table; the migrate loop already supports idempotent ALTERs).
2. Add `SupersededBy string` to the `SearchResult` struct.
3. Update the writer (`UpsertSpec` / wherever specs land — check `refresh.go`) to populate `superseded_by` from `spec.SupersededBy`.
4. Update every `SELECT` that emits search results (`Search`, `searchFilteredImpl`, `listFilteredImpl`, `SearchByFile`, `AllSpecs`, etc.) to include `s.superseded_by` in the projection. The column is appended to `SearchResult` and is empty for non-superseded specs.
5. Add a `superseded_by` column to `fts_nodes`/`node_index` write path so the unified node index sees the same signal.

### Search ranking change

In `internal/retrieval/retrieval.go`:

1. Add a constant `supersededDeweight = 0.3` next to `typeBoost`.
2. In `retrieveViaNodeIndex` and `retrieveViaFTS`, multiply `score` by `supersededDeweight` when the result row's `superseded_by` is non-empty. The annotation marker is added to `Snippet` (prefix: `[SUPERSEDED → <slug>] `) so any caller that just prints results gets the redirect for free.
3. Extend `Query` with `IncludeSuperseded bool`. When true, the de-weight is skipped (multiplier stays 1.0). The annotation marker is *still* added — the de-weight is the rank effect; the annotation is the visibility effect. They're independent.
4. `hero search` accepts `--include-superseded`; default false.

### Context-injection change

In `internal/index/index.go` `BuildNudge`:

1. `RelatedSpecs` currently filters to `r.Status == spec.StatusCompleted`. Extend the filter to also include superseded specs (so they surface to the agent) but tag them. Add a `SupersededBy string` field to `ContextEntry`.
2. In the rendering layer (`internal/cli/relevant.go` or wherever the nudge becomes markdown), when `SupersededBy != ""`, append `[SUPERSEDED by <slug> — follow <slug> instead]` to the bullet.
3. Add `--exclude-superseded` flag to `hero relevant` for the rare callers that want a clean current-only block.

### Spec body convention — auto-prepended banner at render time

The spec file itself does NOT get its body rewritten. Instead:

- When `hero read-spec` or any spec-rendering surface (MCP `hero_read_spec`, web view, `/resume` injection) emits a superseded spec, prepend a banner above the `# Title` line:

  ```
  > **SUPERSEDED by [hero-surface-polish-v2](../hero-surface-polish-v2/spec.md)**
  > This spec is kept for genealogy. Follow the replacement for current direction.
  ```

- The on-disk spec stays clean. The banner is a render-time concern. This keeps `git blame` stable and means re-marking a spec (or unsupersding it) doesn't churn the file body.
- Implementation: a `RenderSpecBody(s *Spec) string` helper in `internal/spec/spec.go` that returns the file body with the banner prepended when `s.SupersededBy != ""`. All read paths call it.

### CLAUDE.md / skill guidance update

Add one sentence to the top of `CLAUDE.md` under "Important Rules":

> When two specs cover the same topic and one carries `superseded_by:`, always follow the spec it points to. Treat the superseded one as historical context only.

Update `domains/engineering/skills/context-injection/SKILL.md`:

- New subsection "Handling superseded specs" right after "Interpreting the sections," explaining that entries marked `[SUPERSEDED → <slug>]` are intentionally surfaced for genealogy but the agent should read and follow the replacement, not the superseded entry.

Update `domains/engineering/skills/spec-format/SKILL.md`:

- Add `superseded_by:` to the frontmatter field list.
- Add a "Superseding a spec" subsection: "If a new spec replaces an older one, run `hero supersede <old> --by <new>` rather than hand-editing — it sets the field, records the inverse relation, reindexes, and updates the graph atomically."

### `hero supersede` command

New command at `internal/cli/supersede.go`.

```
hero supersede <old-slug> --by <new-slug> [--reason "..."]
hero supersede --scan                 # detect candidates, write report
hero supersede --list                 # list current supersede chains
hero supersede --unset <slug>         # clear superseded_by on a spec (rare)
```

The main flow (`hero supersede <old> --by <new>`):

1. Resolve both slugs via `spec.Discover` — error if either is missing.
2. Refuse if `new` is itself superseded (no chains-pointing-into-archives). Hint: "Did you mean `--by <slug-the-target-points-to>`?"
3. Refuse if the operation would create a cycle (walk the existing `superseded_by` chain from `new` and abort if `old` appears).
4. Use `spec.SetFrontmatterField(content, "superseded_by", newSlug)` to update the old spec's frontmatter; preserve everything else.
5. Append a `supersedes: <old-slug>` line to the new spec's frontmatter relations (or add to the YAML-list-of-objects relations block if that form is in use).
6. Run `hero index --if-stale -q` to reindex.
7. Re-run graph ingest for both specs so the `supersedes` edge appears (it already would on next index, but doing it inline keeps the operation atomic from the user's perspective).
8. Print a confirmation including how many search results were de-weighted and which context-injection callers will now annotate the old spec.

`--reason` is optional; when present, it's stored as a comment in frontmatter (`# superseded_reason: ...`) so future archaeology has the rationale.

### Backfill plan

Phase A — scan-only candidate report (`hero supersede --scan`, manual confirm per pair):

1. Walk `.hero/specs/` and `.hero/planning/` once.
2. For each pair the heuristics flag, write a row to `.hero/reports/supersede-candidates.md`:
   ```
   - old: hero-surface-polish-v1  |  new: hero-surface-polish-v2  |  heuristic: slug-suffix  |  confidence: high
   - old: graph-memory             |  new: graph-memory-federation  |  heuristic: body-mention  |  confidence: medium
   ```
3. Human reviews the report, runs `hero supersede` for each confirmed pair.

Phase B — for `hero-v2-system-design` and its children specifically, the user (or a one-off scripted pass after the feature ships) walks the v2 initiative's `child` relations and marks every still-on-disk predecessor manually. This is small and high-value enough not to need scripting.

No automatic supersede detection without human approval is offered. That's an explicit out-of-scope item.

### Graph / genealogy preservation — explicit confirmation

`internal/traversal/why.go:53` lists `supersedes` in `originEdgeTypes`. `hero why <slug>` already walks backward across supersede edges. After this feature:

- Marking a spec superseded *adds* a `supersedes` graph edge (via the existing `graph_ingest.go` machinery, fed by the auto-generated inverse relation).
- The node itself is **never** removed from the graph. It stays as a first-class node with the same `id`, the same `key`, and the same edges. Only the rendering and ranking change.
- `hero why hero-surface-polish-v2` walks the `supersedes` edge backward and surfaces `hero-surface-polish-v1` as an origin hop — the trail to "how we got here" stays intact.
- Verification: add a test case to `internal/traversal/why_test.go` that supersedes one spec by another and asserts that `Why` on the replacement still surfaces the original at depth 1.

### Implementation order (minimizes rollout risk)

1. Frontmatter parsing (`spec.go`) + tests — additive, no behavior change yet.
2. Indexer column + struct field + writer + reader — additive; existing queries keep working because the column defaults to empty.
3. `hero supersede` command (set/unset/list) — gives the user the on-ramp before any consumer behavior changes.
4. Context-injection annotation (`BuildNudge` + render layer) — observable behavior change #1. Roll out behind a feature flag (`HERO_SUPERSEDE_ANNOTATE=1`) for one release if a flag is wanted; otherwise ship directly since the marker is non-destructive.
5. Search ranking de-weight (`retrieval.go`) — observable behavior change #2. Same flag-or-ship choice.
6. `hero search --include-superseded` flag + spec body banner render — UX polish.
7. `hero supersede --scan` candidate detection + report — backfill enabler.
8. CLAUDE.md + skill updates — agent-facing documentation.
9. Human-driven backfill pass over the existing 163 specs.

## Changes

Files touched by delivery:

- `internal/spec/spec.go` — `Spec.SupersededBy` field, `parseFrontmatter` case for `superseded_by`, `IsSuperseded()` helper, `RenderSpecBody()` render-time banner helper.
- `internal/spec/spec_test.go` — `TestParseSupersededBy`, `TestIsSuperseded_LegacyStatus`, `TestRenderSpecBody_SupersededBanner`.
- `internal/index/index.go` — `specs.superseded_by` column + index, `SearchResult.SupersededBy`, `ContextEntry.SupersededBy`, `UpsertSpec` propagation, projection on every SELECT that feeds `scanSearchResults`, `BuildNudge` surfaces superseded specs instead of filtering them out.
- `internal/index/index_test.go` — `TestBuildNudge_SurfacesSupersededWithMarker`, `TestUpsertSpec_PersistsSupersededBy`.
- `internal/retrieval/retrieval.go` — `supersededDeweight = 0.3` constant, `Query.IncludeSuperseded` flag, de-weight + `[SUPERSEDED → <slug>]` snippet annotation in both `retrieveViaFTS` and `retrieveViaNodeIndex`, score-based re-sort in FTS path.
- `internal/retrieval/retrieval_test.go` — `addFTSSpecWithFields` helper, `TestSupersededDeweightRanksAfterPeers`, `TestIncludeSupersededSkipsDeweight`.
- `internal/cli/supersede.go` — new `hero supersede` command (set / unset / list / scan), candidate detector (`detectSupersedeCandidates`), cycle check, `appendSupersedesRelation` helper.
- `internal/cli/supersede_test.go` — scan heuristics, cycle detection, idempotent relation append.
- `internal/cli/search.go` — `--include-superseded` flag plumbed into `retrieval.Query`.
- `internal/cli/relevant.go` — `--exclude-superseded` flag, redirect annotation in `printAssertiveNudge`.
- `internal/cli/root.go` — register `supersedeCmd`.
- `internal/serve/mcp_tools.go` — `toolReadSpec` pipes content through `spec.RenderSpecBody` so the SUPERSEDED banner prepends without mutating the on-disk file.
- `internal/traversal/why_test.go` — `TestWhy_WalksSupersedesEdge`.
- `CLAUDE.md` — "Follow the replacement, not the archive" rule.
- `core/skills/context-injection/SKILL.md` — "Handling superseded specs" subsection.
- `core/skills/spec-format/SKILL.md` — `superseded_by` row in the frontmatter table, "Superseding a spec" subsection.

## Acceptance Criteria

- THE SYSTEM SHALL parse `superseded_by:` from spec frontmatter into `Spec.SupersededBy`.
- THE SYSTEM SHALL persist `superseded_by` in the spec index and include it in every `SearchResult` projection.
- WHEN a search query returns a spec whose `superseded_by` is non-empty THE SYSTEM SHALL multiply that result's score by the de-weight factor (default 0.3) so non-superseded peers rank ahead of it.
- WHEN a search query returns a superseded spec THE SYSTEM SHALL prefix the result's snippet with `[SUPERSEDED → <slug>]` so the redirect is visible to any caller that prints the snippet.
- WHEN `hero search --include-superseded` is passed THE SYSTEM SHALL skip the de-weight multiplier but still emit the `[SUPERSEDED → <slug>]` annotation.
- WHEN `BuildNudge` (context-injection) is called for a file touched by a superseded spec THE SYSTEM SHALL include that spec in `RelatedSpecs` with `SupersededBy` populated.
- WHEN the context-injection markdown renderer emits a superseded entry THE SYSTEM SHALL append `[SUPERSEDED by <slug> — follow <slug> instead]` to the bullet.
- WHEN `hero relevant --exclude-superseded` is passed THE SYSTEM SHALL omit superseded specs from the output entirely.
- WHEN any read path renders a superseded spec body THE SYSTEM SHALL prepend a "SUPERSEDED by <slug>" banner above the title without modifying the on-disk file.
- WHEN `hero supersede <old> --by <new>` is run THE SYSTEM SHALL set `superseded_by: <new>` on the old spec, add an inverse `supersedes: <old>` relation on the new spec, and reindex.
- IF `hero supersede <old> --by <new>` is run AND `new` itself carries a non-empty `superseded_by` THEN THE SYSTEM SHALL refuse the operation and suggest the chain target.
- IF `hero supersede <old> --by <new>` is run AND following the existing supersede chain from `new` reaches `old` THEN THE SYSTEM SHALL refuse the operation as a cycle.
- WHEN `hero supersede --scan` is run THE SYSTEM SHALL write a candidate report to `.hero/reports/supersede-candidates.md` with one row per detected pair and never mutate any spec.
- WHEN `hero why <replacement-slug>` is run on a spec that supersedes another THE SYSTEM SHALL surface the superseded spec as an upstream hop via the `supersedes` edge.
- WHILE a spec lacks the `superseded_by:` field THE SYSTEM SHALL treat it exactly as it did before this feature (no behavior change, full search weight, no annotation).

## Risks

- **Stale chains.** A supersedes B; later B itself gets superseded by C. Search now de-weights both A and B; that's correct. But `hero supersede` should refuse to set A.superseded_by = B if B is itself superseded — handled by Decision 2 in the command flow above. Verified by AC.
- **Cycles.** A.superseded_by = B and B.superseded_by = A would create infinite traversal in `hero why`. Cycle check at command time + defensive depth bound in the render-time banner ("if chain exceeds N hops, stop and warn").
- **Accidentally hiding still-relevant context.** The annotation-rather-than-drop default is the mitigation. The de-weight (0.3) is calibrated so a strong text match on a superseded spec still beats a weak match on a non-superseded peer — superseded specs aren't gone, they're moved down the list.
- **Index column drift.** Adding a column requires a migration. The migrate loop already supports idempotent ALTER TABLE; verified by the `migrations` slice pattern in `index.go`. New installs get the column from the CREATE statement; old installs get it from the ALTER.
- **Skill / CLAUDE.md sync drift.** The guidance lives in three places (CLAUDE.md, context-injection skill, spec-format skill). If we change the marker format later, all three need updating. Mitigation: pick the marker format once, in this spec, and not bikeshed it later.
- **False positives in `--scan` heuristics.** Mitigated by never auto-applying — the scan only writes a report, the human confirms each pair.
- **Existing `StatusSuperseded` enum value semantics.** A spec that already carries `status: superseded` without a `superseded_by:` field is ambiguous — superseded by what? The loader treats `StatusSuperseded` as closed (good) but the annotation marker has no slug to point to. Behavior: if `status: superseded` is set without `superseded_by:`, emit the annotation as `[SUPERSEDED — replacement unknown]` and surface in `hero check` so the author can fix it.
- **Cross-repo supersede.** Peer specs can carry slugs in another workspace's namespace. Out of scope here — `superseded_by:` is local-slug-only for v1. Cross-repo supersede follows the same `peer/slug` convention used elsewhere (deferred to a future spec).

## Out of Scope

- **Physical archive moves.** Specs stay where they are on disk. No new `.hero/archive/` directory.
- **Deletion.** Superseded specs are never deleted by this feature. Cleanup of truly obsolete specs is a separate manual decision.
- **Automatic supersede detection without human approval.** `--scan` produces a report only; no `--auto-apply` mode.
- **Cross-repo supersede.** Marking a peer-workspace spec as superseded from this side is out of scope; this feature handles local slugs only.
- **Changing the `StatusSuperseded` enum semantics.** The enum value stays as-is for backward compatibility; the new `superseded_by:` field is the authoritative signal going forward.
- **Embedding-layer de-weight.** Vector retrieval (`embeddings.QuerySimilar`) is not modified in this spec — superseded specs in the embeddings corpus still come back at full similarity. Tracked in follow-up spec [`embeddings-superseded-respect`](./embeddings-superseded-respect.md) (closes this leak via a query-time overlay in `retrieveHybrid` / `fuseRRF` — no re-embedding required).
