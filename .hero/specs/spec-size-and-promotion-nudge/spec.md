---
title: "Spec Size Field and Promotion Nudge — A Living `size:` That Raises the Floor"
slug: spec-size-and-promotion-nudge
type: feature
domain: engineering
status: completed
size: large
created: 2026-05-31
horizon: now
priority: P1
kind: new
owner: engineering
tags: [spec-types, agent-guidance, sizing, promotion, drift, tracker-mapping]
relations:
  - { kind: relates-to, target: spec-status-integrity }
  - { kind: relates-to, target: master-ingest-restore }
completed_at: 2026-06-01T14:31:40Z
---

# Spec Size Field and Promotion Nudge

## Goal

Add a **living `size:` frontmatter field** to `feature`, `bug`,
`enhancement`, `epic`, and `initiative` specs, using a shared 6-tier
ladder (`trivial / small / medium / large / x-large / giant`). Plus the
agent guidance that **proactively asks "promote to initiative?"** when a
leaf spec hits `x-large` or `giant`, surfaces drift between declared
size and computed estimate, rolls child sizes up to container declared
size, and bidirectionally maps to tracker fields when a tracker is
configured.

"Done" looks like: any agent reading or authoring a spec can see the
declared size, gets a soft/strong/super-strong nudge at the right tier,
and the snapshot/check pipeline calls out specs where declared size and
reality have drifted apart.

## Kickoff

```
spec-size-and-promotion-nudge — DELIVERED across 5 slices + AC#16
follow-through (commits fc982e7, 985e95d, 705130f, 7d5dc1a, 3d0b942,
3753bdf). Living `size:` field + size_ack lands on feature/bug/epic/
initiative; `hero size` CLI + drift detection in `hero check` /
`hero_warnings`; `spec-sizing` skill drives delivery-lead nudges per
tier; tracker mapping wired bidirectionally with non-destructive
CreateIssue payload writes on Jira/Linear/GitHub. One residual:
in-place tracker UpdateSize (Jira PUT, Linear issueUpdate, GitHub
label rotation) needs a new interface method + per-adapter handlers
— spawned as a follow-up. To pick that up: extend Tracker interface
with UpdateSize(issueID, localTier), wire into runSync replacing the
warn-only PlanSizePush hint, add the conflict gate via the existing
PlanSizePush logic. Audit report at
.hero/planning/features/spec-size-and-promotion-nudge/delivery-audit.md.
```

## Problem

Hero today has the plumbing for big work — parent/child relations,
`/split`, `/compose`, `hero estimate`, snapshot rollup over
initiative→children — but **no proactive guidance** that asks the user
"this is getting big; promote it to an initiative with children?" The
senior engineer with strong taste knows to interrupt themselves and
split. Everyone else ships a sprawling spec or, worse, an agent
delivery loop chases a giant scope until the engineer gives up.

This is most acute when Hero is the **sole source of truth for
hierarchy** — no tracker, or a tracker without strong epic/sub-issue
support. There is no other system telling the team "this looks like
an epic now"; Hero has to.

Today `hero estimate` exists and outputs a computed bucket
(`trivial/small/medium/large/x-large`), but:

- It is **computed-only** — there's no declared intent to compare
  against, so drift between "what I planned to build" and "what the
  spec has become" is invisible.
- It does not influence agent behavior — no agent reads the bucket
  and changes how it answers.
- It does not roll up — an initiative's declared scope and its
  children's summed cost are not compared.
- It has no top tier above `x-large`; truly enormous specs are
  silently lumped in with merely-big ones, removing the signal that
  triggers a promotion conversation.

## Approach

### The ladder — Option C (shared vocabulary, absolute magnitudes, per-type bands)

One vocabulary across all sized spec types. **Each tier is roughly the
same absolute magnitude of effort** regardless of which type carries
it. The "comfortable range" differs per type — what is normal for an
initiative is alarming for a feature.

| Tier | Magnitude | Feature / Bug / Enhancement | Epic | Initiative |
|---|---|---|---|---|
| `trivial` | hours | normal | rare | — |
| `small` | ~1 day | normal | normal | small |
| `medium` | a few days | normal | normal | normal |
| `large` | week+ | **soft nudge** | normal | normal |
| `x-large` | weeks | **strong nudge** | nudge | normal |
| `giant` | month+ | **promote → small initiative** | **promote → initiative or `/compose`** | **`/compose` into phases** |

The elegance of Option C: **a `giant feature` ≈ a `small initiative`
in absolute effort.** Promotion is literally moving up one tier on the
shared ladder. One promotion rule everywhere: "you're at `giant` →
promote to the next type up."

We considered (A) same words / different per-type meanings — rejected
because it's ambiguous in reports — and (B) distinct taxonomy per type
— rejected because it doubles the vocabulary and breaks the
promotion-is-one-step idea. Option C won because it gives a single
shared mental model with type-specific norms layered on top, and the
promotion mechanic falls out for free.

### `size:` is a **living** field

Treat it like `status:` — agents are expected to keep it accurate.
Updates happen:

- At **design time** — `/design` sets it as part of the initial spec.
- During **authoring/refinement** — if scope grows, the spec writer
  bumps it.
- **Mid-delivery** — if the delivery lead discovers the spec is bigger
  than declared, it bumps and surfaces (this is where most drift gets
  caught in practice).

The declared `size:` is the **intent**. `hero estimate` is the
**measurement**. Drift is the difference between them; that's the
signal worth surfacing.

### Nudge intensity (advise, never block)

Even `giant` does not block. It's a **super-strong recommendation**
the user can override — friction climbs but the user always wins. Same
shape as a loud linter warning.

| Tier | Design-time | On scope-up edit | Mid-delivery |
|---|---|---|---|
| `large` | soft ask once | quiet | mention in handoff |
| `x-large` | strong ask; recommend `/compose` or `/split` | re-ask if scope grew | flag each delivery session |
| `giant` | super-strong rec; require acknowledgement | re-recommend each time | strong rec every session until split or scope shrinks |

Acknowledgement at `giant` is a one-shot: the user says "yes I know,
proceed anyway" and the spec records that acknowledgement in
frontmatter (`size_ack: giant` or similar) so the nudge stops nagging
at design-time. Mid-delivery still surfaces because that's where the
spec is actually feeling the size.

### Drift detection

Two flavors:

- **Leaf drift** (feature/bug/enhancement): declared `size:` vs
  `hero estimate` computed bucket. If declared `medium` and computed
  `large`, that's drift — surface it.
- **Container drift** (epic/initiative): declared `size:` vs
  **aggregated child rollup**. Rule: **declared ≥ rollup**. If you
  declared the initiative `medium` and its children sum to `large`,
  the declared size is stale — bump it.

Both surface via `hero check` and `hero_warnings`. Neither blocks
delivery; both nudge.

### Tracker-aware bidirectional sync (NON-destructive)

`size:` is **always set locally**, regardless of tracker presence.
The local field is the **AI-decision authority** — agents read it
to choose nudge intensity, regardless of what the tracker shows.

When a tracker is configured (`hero.json: tracker.type != "none"`),
mapping lives in `.hero/hero.json`:

```jsonc
"tracker": {
  "type": "jira",
  // ...
  "size_mapping": {
    "field": "story_points",     // tracker field name
    "thresholds": {              // tracker value → local tier
      "trivial": [0, 1],
      "small":   [2, 2],
      "medium":  [3, 5],
      "large":   [8, 8],
      "x-large": [13, 13],
      "giant":   [20, null]
    },
    "container_field": "epic_label"   // optional, for epic/initiative
  }
}
```

- Leaf-tier defaults shipped: Jira `story_points` with the bands
  shown above; Linear `estimate` with similar bands; GitHub `size/*`
  labels.
- Container-tier maps to Jira epic / Linear project / GitHub
  milestone attributes (often free-text or label).

Sync behavior:

- `hero sync pull`: seed local `size:` from tracker **only if absent**;
  if present, surface drift as a warning (never auto-resolve).
- `hero sync push`: write local `size:` to tracker when mapping is
  configured; **never silently overwrite** a tracker value a human
  just set — surface conflict, ask the user.

### Tracker-aware nudge tuning

Promotion-nudge threshold depends on **what the tracker can do for
hierarchy**:

| Tracker capability | Nudge aggressiveness | Promotion action offered |
|---|---|---|
| No tracker (`type: "none"`) | **Most aggressive** — `large` always asks, `x-large` strongly recommends, `giant` super-strong rec | Local promotion only (turn this spec into an initiative with children, all in `.hero/`) |
| Tracker without strong hierarchy (basic GitHub issues) | Same aggressiveness as no-tracker | Local promotion only — Hero handles parent/child locally, tracker just sees child issues |
| Tracker with strong hierarchy (Jira epics, Linear projects, GitHub sub-issues) | **Less aggressive** — higher threshold; soft nudge starts at `x-large`, strong at `giant` | Promotion offers to **create the parent in the tracker too** so the human team's view stays coherent |

Capability detection lives in the tracker adapter — each adapter
declares `supports_hierarchy: true|false` and the nudge layer reads
that.

## Acceptance Criteria

- THE SYSTEM SHALL accept `size:` as a frontmatter field on specs of
  type `feature`, `bug`, `epic`, and `initiative` with values
  `trivial | small | medium | large | x-large | giant`. (Enhancement
  coverage is added when the `enhancement` spec-type itself ships;
  not in scope here.)
- IF a spec carries `size:` with a value outside the ladder THEN THE
  SYSTEM SHALL reject the spec at load time with a clear error naming
  the field and allowed values.
- WHEN `/design` produces a new spec of a sized type THE SYSTEM SHALL
  stamp `size:` in the frontmatter based on the design conversation
  (or `medium` as a safe default when undetermined).
- WHEN `hero estimate <slug>` runs THE SYSTEM SHALL print both the
  **declared** `size:` (from frontmatter) and the **computed** bucket
  side by side, and flag drift when they differ.
- WHEN `hero size <slug>` runs without args THE SYSTEM SHALL print
  the current declared size for that spec.
- WHEN `hero size <slug> <tier>` runs THE SYSTEM SHALL update the
  frontmatter `size:` field non-destructively and re-emit the spec.
- WHEN `hero size --check` runs THE SYSTEM SHALL list every spec with
  size drift (leaf or container) and exit non-zero if any are found,
  for CI integration.
- WHEN an agent loads a spec of type `feature`, `bug`, or
  `enhancement` with declared `size: large` THE SYSTEM SHALL emit a
  soft suggestion to consider promotion at design time only.
- WHEN an agent loads such a spec with declared `size: x-large` THE
  SYSTEM SHALL emit a strong promotion recommendation at design time
  and on every delivery-session pickup.
- WHEN an agent loads such a spec with declared `size: giant` THE
  SYSTEM SHALL emit a super-strong promotion recommendation and
  require an explicit `size_ack: giant` acknowledgement in frontmatter
  before suppressing the design-time nudge (mid-delivery nudge
  continues regardless).
- WHEN an agent computes a container spec's declared size against its
  child rollup THE SYSTEM SHALL flag drift if declared < rollup and
  recommend bumping declared.
- WHILE `tracker.type` in `hero.json` is `"none"` THE SYSTEM SHALL
  use the most aggressive nudge schedule (large → soft, x-large →
  strong, giant → super-strong) and offer only local promotion.
- WHERE the configured tracker adapter declares
  `supports_hierarchy: true` THE SYSTEM SHALL raise the nudge
  threshold (soft at x-large, strong at giant) and offer to create
  the parent issue in the tracker as part of promotion.
- WHEN `hero sync pull` runs against a tracker with `size_mapping`
  configured AND a spec has no local `size:` THE SYSTEM SHALL seed
  the local field from the tracker's value.
- IF `hero sync pull` finds both a local `size:` and a tracker value
  that map to different tiers THEN THE SYSTEM SHALL surface the
  conflict and SHALL NOT auto-resolve.
- WHEN `hero sync push` runs with `size_mapping` configured THE
  SYSTEM SHALL write the local tier to the mapped tracker field,
  unless a human-modified tracker value would be overwritten, in
  which case it SHALL surface the conflict and stop. **(DONE for
  the create path: Jira `customfield_*`, Linear `estimate`, and
  GitHub `size/*` label are emitted by `CreateIssue` when the spec
  carries `size:` and the mapping resolves; nothing to overwrite,
  so the planner isn't invoked. The in-place update path (e.g.
  Jira PUT `/issue/{id}` for an existing story-points value) is a
  follow-up — it needs `UpdateSize` on the Tracker interface plus
  per-adapter wiring, and the planner already guards overwrites
  via `PlanSizePush`.)**
- THE SYSTEM SHALL include size-drift items in `hero check` output
  and in `hero_warnings` MCP tool results.
- THE SYSTEM SHALL document the size ladder, per-type bands, and
  nudge schedule in `core/spec-types/README.md` (or equivalent
  central doc) so the contract is visible without reading code.

## Files to Touch

### Schema (core/spec-types)

- `core/spec-types/feature.md` — add `size` to optional frontmatter
  with full enum + description; document per-type band (normal:
  trivial..medium; nudge: large+).
- `core/spec-types/bug.md` — same.
- `core/spec-types/enhancement.md` — same (if exists; otherwise
  defer until enhancement type ships).
- `core/spec-types/epic.md` — add `size` and `size_ack`;
  document container band (normal: small..x-large; promote at giant)
  and child-rollup expectation.
- `core/spec-types/initiative.md` — same as epic; document that
  `giant` initiatives should be `/compose`-d into phases.
- `core/spec-types/README.md` (create if missing, or pick existing
  central doc) — add a "Sizing" section with the ladder table, the
  per-type band table, and the nudge schedule.

### Validator / loader

- `internal/spec/spec.go` — add `Size string` and `SizeAck string` to
  the `Spec` struct; teach `parseFrontmatter` to read them; add a
  `validateSize(v string) error` helper for the 6-tier enum;
  surface invalid values as load-time errors.

### CLI

- `internal/cli/cost.go` — add `effortGiant = "giant"` constant; bump
  `bucketFromPoints` to emit it above the existing x-large threshold
  (calibrate so it triggers around ~2x x-large). Augment
  `estimateSpec` / output to print declared-vs-computed and flag
  drift; teach `printEstimateTable` to render a drift column.
- `internal/cli/size.go` (new) — implement `hero size <slug>` (get),
  `hero size <slug> <tier>` (set), `hero size --check` (drift
  report, exit non-zero on findings).
- `internal/cli/check.go` (or wherever `hero check` lives) — include
  size-drift findings in the health summary.

### Drift + rollup

- `internal/snapshot/rollup.go` — extend the existing
  initiative→children rollup to aggregate child `size:` values into a
  computed container size; expose for the drift check. **(slice 3:
  added `RollupChildSizes`, `ContainerDrift`, `BuildParentMap`, and
  the tier→midpoint sum-and-rebucket aggregation per spec rules.)**
- `internal/cli/cost.go` (drift helpers) — `LeafDrift(spec) *Drift`
  and `ContainerDrift(spec, rollup) *Drift`. **(slice 2: `LeafDrift`;
  slice 3: `ContainerDrift` lives in `internal/snapshot` to keep the
  snapshot package callable from MCP without an import cycle.)**
- `internal/sizing/sizing.go` (new, slice 3) — shared backend so the
  CLI and MCP surfaces compute drift identically. Holds
  `CollectDrift`, `EstimateSpec`, `BucketFromPoints`, and the effort-
  tier constants previously duplicated between cli/cost.go and the
  CLI-only `costEstimate` struct.

### MCP surface

- `internal/serve/mcp_tools.go` — `hero_warnings` includes size-drift
  entries; `hero_read_spec` returns declared size; consider a
  dedicated `hero_size` MCP tool mirroring the CLI.

### Tracker mapping

- `internal/config/config.go` — **(slice 5)** added
  `TrackerConfig.SizeMapping` + `SizeMappingConfig{Field, Thresholds,
  ContainerField}` struct; load-time validation rejects bad mappings
  when `tracker.type != "none"`.
- `internal/tracker/tracker.go` — **(slice 5)** extended `Tracker`
  interface with `SupportsHierarchy()`, `MapSize`, `ReverseMapSize`;
  threaded `SizeMapping` through `New` / `NewWithJiraConfig`.
- `internal/tracker/size_mapping.go` — **(slice 5, new)** per-adapter
  defaults (Jira `story_points`, Linear `estimate`, GitHub `size/*`
  labels); `mapSizeWith` / `reverseMapSizeWith` core; `ExtractTrackerSize`
  for issue inspection; `PlanSizePull` / `PlanSizePush` for the
  non-destructive sync planner; `TypeSupportsHierarchy` token-free
  capability lookup.
- `internal/tracker/jira.go` / `linear.go` / `github.go` —
  **(slice 5)** added `configuredSizeMapping` field; `SupportsHierarchy`,
  `MapSize`, `ReverseMapSize`, `sizeMapping` methods (delegating to
  shared helpers).
- `internal/cli/pull.go` — **(slice 5)** wires `PlanSizePull` into
  `runPull`; seeds local `size:` when absent; surfaces conflicts as
  warnings; never overwrites.
- `internal/cli/sync.go` — **(slice 5)** wires `PlanSizePush` into
  `runSync` for the `sync spec` path; warns on conflict; surfaces a
  hint when a clean push would apply (write itself is the per-tracker
  push path's job, not this command).
- `internal/cli/size.go` — **(slice 5)** prints the tracker-capability
  header in `hero size --check` (Option B from the spec); exports
  `WorkspaceTrackerCapability` for the agent surface.
- `internal/sizing/sizing.go` — **(slice 5)** added
  `TrackerCapability` struct + `NudgeRegime()` so the spec-sizing
  skill and CLI surface report the same regime label.
- `domains/engineering/skills/spec-sizing/SKILL.md` — **(slice 5)**
  fixed two stale `hero estimate` references to the actual
  `hero sprint estimate` subcommand (caught by the markdown drift
  test once slice 4 landed; opportunistic fix).

### Agent guidance

- `.claude/skills/spec-sizing/SKILL.md` (new) — defines the ladder,
  the per-type bands, the nudge intensities, the acknowledgement
  protocol, and how delivery leads should phrase the nudge. Both
  feature-delivery-lead and platform-delivery-lead load this skill.
  **(slice 4: landed; ~259 lines; paste-ready phrasing per tier and
  tracker regime.)**
- `.claude/agents/feature-delivery-lead.md` — add a step in the
  delivery loop: "load spec-sizing, check declared size + drift,
  surface nudge per schedule." **(slice 4: added step 4d in delivery
  loop and design-time skill-load in the design phase.)**
- `.claude/agents/platform-delivery-lead.md` — same. **(slice 4:
  added step 3b in delivery loop and design-time skill-load.)**
- `.claude/skills/spec-format/SKILL.md` — document `size:` as a
  living field alongside `status:`; show an example; explain when
  the spec writer is expected to bump it. **(slice 4: added `size`
  and `size_ack` frontmatter rows, a "living field" subsection, and
  a brief example; cross-references `spec-sizing` for the ladder.)**
- `.claude/commands/design.md` — **(slice 4)** add design-time
  `size:` stamping instruction alongside the `domain:` stamping
  paragraph; references `spec-sizing`.

## Implementation Notes

- **Build in slices.** Slice 1: schema + validator + frontmatter
  parsing (no behavior change yet — declarations land cleanly).
  Slice 2: `hero estimate` augmentation + the new `giant` tier in
  `bucketFromPoints` + `hero size` CLI. Slice 3: drift detection
  (leaf + container) + `hero check` + `hero_warnings` integration.
  Slice 4: spec-sizing skill + delivery-lead wiring (the nudge
  actually fires). Slice 5: tracker mapping + sync push/pull.
- The existing `effortTrivial..effortXLarge` constants in
  `cost.go` already cover 5 of the 6 tiers — adding `giant` is
  a low-risk extension, not a rename. Keep the existing strings
  exactly as-is to avoid breaking the calibration cache and
  `hero_velocity` history.
- The nudge text itself lives in the skill so it can be tuned
  without a code change. Delivery leads quote from the skill
  rather than hard-coding wording.
- `size_ack: giant` is the only acknowledgement value that
  matters today; design future-proof with a free string, but
  initially only consume `giant`.
- Drift reports should be **rate-limited** in `hero check` output —
  group all leaf drifts and all container drifts into two summary
  lines with counts and a hint, plus full detail on demand
  (`hero size --check`).

## Out of Scope

- **Auto-splitting.** The system never splits a spec automatically.
  It can recommend `/split` and `/compose` and pre-populate suggested
  child slugs, but the actual decomposition is a human/agent action.
- **ML-based estimation.** The computed bucket stays heuristic
  (file/word/section counts). No model-based effort prediction in
  this spec.
- **Velocity-aware sizing.** Adjusting the tier-to-effort mapping
  based on observed team velocity is a future enhancement; for now
  the calibration is the same `cost.go` heuristic everyone already
  uses.
- **Per-engineer size norms.** Sizing is project-level. We don't
  carry "this engineer's `small`" individually.
- **New spec types.** This spec adds the field to existing types
  (feature/bug/enhancement/epic/initiative). It doesn't introduce
  enhancement as a new type if it doesn't already exist — defer to
  the type's own spec.
- **Tracker fields beyond size.** Bidirectional mapping for other
  frontmatter fields (priority, severity, etc.) is out of scope —
  this spec only wires size.

## Risks

- **Nudge fatigue.** If `large` fires a soft nudge on every session
  pickup, engineers will tune it out. Mitigation: design-time-only
  for `large`; only `x-large`/`giant` nag mid-delivery; track ack
  state so we don't re-ask on the same session.
- **Tracker-mapping calibration drift.** Default Jira/Linear/GitHub
  bands will be wrong for many teams. Mitigation: ship sensible
  defaults but make the mapping fully overrideable in `hero.json`;
  surface conflicts loudly so users notice mismatches early.
- **Estimate calibration shift when `giant` is introduced.** The
  existing `bucketFromPoints` has tuned thresholds; adding a tier
  above x-large could re-shuffle bucket distributions if the upper
  threshold is set wrong. Mitigation: ship `giant` with a high
  threshold (~2x x-large) so existing x-large specs don't all
  rebucket; gather telemetry before adjusting.
- **Container-rollup correctness on partial initiatives.** If a
  child has no declared `size:`, the rollup must fall back to
  computed; if it's missing both, treat as unknown rather than
  zero (which would silently understate container size).
- **Promotion suggestion friction.** "Promote to initiative" sounds
  cheap; in practice it means creating a parent spec, re-parenting
  the original, and potentially writing child specs. The nudge
  should make the cost visible — "promotion creates a new
  initiative spec and converts this one into N child specs" —
  rather than implying it's a single keystroke.
- **Acknowledgement bypass.** A user who genuinely needs to ship a
  `giant` spec without splitting (incident response, time pressure)
  needs a fast escape hatch. `size_ack: giant` is that escape. Make
  sure it's documented so users don't feel trapped.

## Validation

- Unit tests in `internal/spec/spec_test.go` for `size:` parsing,
  enum validation, and `SizeAck` parsing.
- Unit tests in `internal/cli/cost_test.go` for `giant` bucketing
  and the declared-vs-computed drift helper.
- Unit tests for `internal/snapshot/rollup.go` covering child
  rollup with missing sizes (treat as unknown, not zero).
- Integration test: scaffold an initiative with three medium
  children, verify container drift surfaces when declared is
  `small`.
- Integration test: tracker sync round-trip — set local `large`,
  push to a fake adapter mapped to `story_points: 8`, pull back,
  confirm no drift; then mutate tracker value, pull again, confirm
  conflict is surfaced.
- Manual: run `/design` on a deliberately huge feature, confirm
  the spec writer stamps `x-large`/`giant` and the delivery lead
  asks about promotion.
- Manual: run `hero size --check` against this workspace, confirm
  it identifies actually-drifted specs (this workspace has plenty
  of large in-flight features — should produce real findings).
- Run `hero check` after wiring; confirm size-drift lines appear
  in the health summary without drowning out other findings.
