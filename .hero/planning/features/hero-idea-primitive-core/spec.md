---
title: "Intake primitive — Go engine recognition, committed-work predicate, CLI & MCP"
type: feature
status: in-review
priority: P1
claimed_by: bdwheeler
created: 2026-06-26
horizon: now
size: large
tags: [hero-core, cli, mcp, spec-types, intake, pre-commitment, predicate]
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero
  originator_slug: hero-idea-primitive-core
  handed_off_at: 2026-06-26T18:22:51Z
  at_commit: 6f9fc13
  reason: "Engine-side delivery of the 'idea' primitive (decision: hero-idea-primitive, recorded in hero-code). ALREADY SHIPPED in hero-code (Swift desktop app): idea SpecType + .hero/planning/ideas/ routing, intake-funnel projection from SpecStore, triage accept/reject/merge/promote persisted via SpecWriteService writing only FieldOwnership .local fields (intake_status, classifications, merged_into, promoted_to, intake_sources), and the terminal 'rejected' state. NEEDED in hero (core/CLI/MCP/workflows, owned by this peer): (1) spec_index walks .hero/planning/ideas/ and recognizes idea as pre-commitment; (2) one shared is_committed_work() predicate so status/queue/active/velocity/snapshot exclude ideas; (3) 'hero idea' capture/promote/reject/list verbs (promote = create roadmap spec + derived_from edge); (4) MCP: ideas in hero_search/hero_why, excluded from in-flight tools; (5) recognize intake_status: rejected in core. Full Goal/Approach/AC/Test-Plan/Changes/Risks are in the spec body."
relations:
  - { kind: implements, target_type: decision, target: hero-idea-primitive }
---

# Intake primitive — Go engine recognition, committed-work predicate, CLI & MCP

## Provenance

Handed off from peer `hero-code` (peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`)
as `hero-idea-primitive-core` — the engine-side delivery of the `idea`
pre-commitment primitive the Swift desktop app already ships.

**This spec re-grounds that handoff for the Go engine and maps it onto the
existing `intake` spec-type.** The original was written against a Rust core
(`crates/hero-core/...`) and asked for a new `idea` type. Neither premise holds
here: this repo is Go, and it already ships a structurally-identical
pre-commitment type called **`intake`** ([core/spec-types/intake.md](../../../../core/spec-types/intake.md)).

The reconciliation was confirmed by advisory peer-call to hero-code
(call `18bcc88ccdad7378b3600011124c181a`, artifact under `.hero/peer-calls/`):
the two repos **independently converged** on the same primitive under different
names — hero-code's author never had the engine `intake` type in view, and even
kept the `intake_*` field names while renaming only the type slug to `idea`.
hero-code endorses mapping onto `intake` engine-side and will file a follow-on
to rename/migrate the Swift `idea` → `intake`. **No parallel `idea` type is
added here** (that is the type-proliferation failure in Risk #4 below).

## Goal

Make `intake` a first-class **pre-commitment** primitive in the Go engine: a
captured signal that is indexed, searchable, and provenance-linked through the
spec graph and MCP, capturable/promotable/rejectable from the CLI — while
staying out of every committed-work rollup (status, queue, velocity, snapshot)
until it is promoted to a roadmap spec. The type declaration already exists in
the registry; the engine code does not yet model it, which both hides it and
risks leaking it into in-flight views.

## Problem

The original handoff's #1 item — "the spec-index walks only `planning/features`
+ `planning/initiatives`, so pre-commitment specs are invisible" — **does not
apply to the Go engine.** `spec.Discover` ([internal/spec/spec.go](../../../../internal/spec/spec.go))
does a full `filepath.Walk` of the entire `.hero/` tree and loads any `spec.md`
it finds. An `intake` spec under `.hero/planning/intake/<slug>/spec.md` is
already discovered. The real gaps are narrower and different:

1. **`intake` is unmodeled in code.** `spec.go`'s `Type` enum (the `Type*`
   consts) has no `TypeIntake`, and `typeFromPath` has no `/intake/` case
   (its fallback is `TypeFeature`). Frontmatter `type: intake` *is* honored
   (frontmatter wins; path only fills when `Type == ""`), but the value then
   matches neither `IsWorkSpec()` nor `IsKnowledge()` — it lands in a limbo
   category with undefined rollup behavior.

2. **The committed-work predicate already exists but is bypassed.**
   `Spec.IsWorkSpec()` (`spec.go`, returns `Type == Feature || Type == Bug`)
   is the predicate the handoff asks for. But ~18 rollup sites re-derive it
   inline with raw `s.Type != TypeFeature && s.Type != TypeBug` checks instead
   of calling it. Adding a third category (`intake`) to a deny-list world means
   touching all of them or leaking.

3. **In-flight rollups are deny-lists, so `intake` leaks.** `IsReady()`
   ([internal/spec/select.go](../../../../internal/spec/select.go)) gates the
   queue on `!IsKnowledge()`. `intake` is not knowledge, so it passes the gate
   and can appear in `hero queue`. `hero status` groups by `status`; an `intake`
   with `status: planning` would surface in the work "planning" section. The
   structural fix is to flip in-flight rollups from deny-list (`!IsKnowledge`)
   to allow-list (`IsWorkSpec`).

4. **No capture/transition CLI.** There is `hero note` but no `hero intake`.
   The only way an intake exists today is the Mac funnel or hand-authoring.

5. **MCP doesn't model the third category.** `hero_search`/`hero_why` are
   type-agnostic (good — intakes should appear), but `hero_queue` inherits the
   `IsReady` leak, and there is no way to filter to/away from intakes by intent.

## Approach

**1. Model `intake` in `spec.go`.** Add `TypeIntake Type = "intake"` to the
type consts and a `/intake/` case to `typeFromPath` (so a path-only intake
resolves correctly; frontmatter still wins when present). Add the intake
lifecycle statuses (`triaged`, `promoted`, `rejected`, `merged`) as `Status`
constants if not already modeled, matching [intake.md](../../../../core/spec-types/intake.md)'s
declared lifecycle (`planning → triaged → promoted | rejected | merged`).

**2. One pre-commitment predicate; flip in-flight rollups to allow-list.**
Add `Spec.IsPreCommitment()` (true for `TypeIntake`). Keep `IsWorkSpec()` as the
canonical "committed work" predicate (feature/bug). Then:
- Replace every inline `Type != Feature && Type != Bug` rollup guard with a call
  to `IsWorkSpec()` (allow-list) — see the Changes table for the full site list.
- Change `IsReady()` to gate on `IsWorkSpec()` (or `!IsKnowledge() &&
  !IsPreCommitment()`), so the queue admits only committed work.
- `hero status` routes intakes into their own pre-commitment section (mirroring
  how knowledge types are segregated at status.go), never the work status
  buckets.

Search, `hero_why`, the graph, and `hero list --type intake` include intakes
(provenance matters). Queue/status/velocity/snapshot exclude them.

**3. `hero intake` CLI verbs** (mirroring `hero note`'s structure in
[internal/cli/note.go](../../../../internal/cli/note.go)):
- `hero intake "<text>"` — capture; scaffold
  `.hero/planning/intake/<slug>/spec.md` with `type: intake`, `status: planning`.
- `hero intake promote <slug> [--type feature|bug]` — create the roadmap spec,
  write the provenance relation, set the intake `status: promoted` and record
  the promoted-to target. Reuses the existing create-spec write path.
- `hero intake reject <slug>` — set the terminal `status: rejected`.
- `hero intake list` — list intakes by status (excluded from `hero status`).

**4. Provenance edges.** On promote, write the `promotes_to` relation
(intake → roadmap spec) that [intake.md](../../../../core/spec-types/intake.md)
already declares, and a reverse `derived_from` on the promoted spec so
`hero_why` traverses promoted-spec → originating intake. Confirm which relation
kind `hero_why` already follows and reuse it; do not invent a parallel edge.

**5. MCP.** `hero_search`/`hero_why` already include all types — verify intakes
surface and `hero_why` resolves the provenance edge. `hero_queue` inherits the
`IsReady` allow-list fix from step 2 (no separate change). Add `intake` as a
recognized value for the `type` filter in `hero_search`/`hero_list`.

**Workflow loop (`/discover` → `/design`) is explicitly deferred** — it is the
fuzziest, largest slice (original Risk #3) and not required for intakes to be
usable end-to-end via the CLI. Tracked as a follow-on, not delivered here.

## Design / Data & State

- **Type** — `intake`, dir `.hero/planning/intake/`, already declared in the
  registry ([core/spec-types/intake.md](../../../../core/spec-types/intake.md)):
  category `work`, lifecycle `planning → triaged → promoted | rejected | merged`
  (terminals `promoted, rejected, merged` — note `rejected` is already present,
  closing the gap hero-code flagged). Engine code now mirrors this declaration.
- **Three categories, not two.** Engine specs partition into: **committed work**
  (`IsWorkSpec`: feature, bug), **knowledge** (`IsKnowledge`: convention,
  decision, rule, external, context, note, tripwire, explainer), and
  **pre-commitment** (`IsPreCommitment`: intake). Initiatives remain containers,
  outside `IsWorkSpec` as today — unchanged.
- **Registry `category` stays export-only.** Rollups consult the Go predicates,
  not the registry's `category` field (which is serialized to
  `.hero/cache/spec-types.json` for the integration layer). The predicate is the
  single source of truth in core, per the handoff's intent.
- **Provenance** — `promotes_to: <roadmap-slug>` on the intake, `derived_from:
  <intake-slug>` on the promoted spec. No new store; intakes live in the same
  spec graph.

## Acceptance Criteria

- WHEN core discovers a workspace containing `.hero/planning/intake/<slug>/spec.md`
  THE SYSTEM SHALL load it as `Type == TypeIntake` (frontmatter-driven), and it
  SHALL appear in `hero search` and `hero list --type intake`.
- WHEN any committed-work rollup is computed (`hero status` work sections,
  `hero queue`, `hero snapshot`, velocity, the synthesize detector) THE SYSTEM
  SHALL exclude `intake` specs by routing through `IsWorkSpec()` (allow-list),
  not an inline type check.
- WHEN an `intake` spec has `status: planning` THE SYSTEM SHALL NOT surface it in
  the `hero status` work "planning" bucket; it SHALL appear only in a
  pre-commitment listing.
- WHEN a user runs `hero intake "<text>"` THE SYSTEM SHALL scaffold a valid
  `type: intake`, `status: planning` spec under `.hero/planning/intake/` such
  that a subsequent `Discover` returns it as `TypeIntake`.
- WHEN a user runs `hero intake promote <slug>` THE SYSTEM SHALL create the
  roadmap spec, write the `promotes_to`/`derived_from` relations, and set the
  intake `status: promoted`, through the existing durable write path.
- WHEN a user runs `hero intake reject <slug>` THE SYSTEM SHALL set the terminal
  `status: rejected`, after which the intake SHALL NOT appear as actionable in
  any in-flight view.
- WHEN `hero_why` is asked about a promoted spec THE SYSTEM SHALL traverse the
  provenance relation back to the originating intake.
- WHEN `hero_queue` runs THE SYSTEM SHALL NOT return `intake` specs regardless of
  their status (no leak via the former `!IsKnowledge` gate).
- Regression: existing feature/bug/initiative/knowledge rollups SHALL be
  unchanged — the allow-list refactor is behavior-preserving for those types.

## Test Plan

- **Modeling** — unit-test `typeFromPath` returns `TypeIntake` for an
  `/intake/` path; `Parse` honors frontmatter `type: intake` over path.
- **Predicate** — table test `IsWorkSpec` / `IsKnowledge` / `IsPreCommitment`
  across all types (mutually exclusive, exhaustive); assert `intake` is
  pre-commitment only.
- **No-leak rollups** — temp workspace with an `intake` (status `planning`);
  assert absent from `hero queue`, the `hero status` work buckets, snapshot
  rollups, and the synthesize detector; assert present in `hero search` and
  `hero list --type intake`.
- **CLI** — `hero intake` capture → `Discover` returns it; `promote` → roadmap
  spec + `promotes_to`/`derived_from` + intake `status: promoted`; `reject` →
  terminal `rejected`.
- **Provenance** — `hero_why` on a promoted spec resolves back to the intake.
- **Regression** — golden tests for `hero status` / `hero queue` / snapshot on a
  fixture with features+bugs+knowledge (no intake) are byte-identical before and
  after the allow-list refactor.

## Changes

| File / area | Change | Est. |
|---|---|---|
| `internal/spec/spec.go` | Add `TypeIntake` + intake `Status` consts; `/intake/` case in `typeFromPath`; add `IsPreCommitment()`; keep `IsWorkSpec()` as committed-work predicate | M |
| `internal/spec/select.go` | `IsReady()` gate → `IsWorkSpec()` allow-list (was `!IsKnowledge()`); recognize `intake` in `Filter.Types` | S |
| `internal/cli/status.go` | Segregate `intake` into a pre-commitment section; route work buckets through `IsWorkSpec()` | M |
| `internal/cli/pipeline.go` (L83), `deliver.go` (L300), `synthesize/detect.go` (L93), `snapshot/rollup.go` (L486) | Replace inline `Type != Feature && Type != Bug` with `IsWorkSpec()` | M |
| `internal/cli/intake.go` (new) | `hero intake` capture / promote / reject / list (mirror `note.go`); register in `root.go` | L |
| `internal/serve/mcp_tools.go` | Verify `hero_search`/`hero_why` surface intakes + provenance; add `intake` to type filters; confirm `hero_queue` inherits the `IsReady` fix | M |
| `core/spec-types/intake.md` | Reconcile prose lifecycle (`new → triaged → linked`) with the authoritative frontmatter (`planning → triaged → promoted/rejected/merged`) — prose is stale | S |
| Tests | modeling + predicate + no-leak rollups + CLI + provenance + regression goldens | L |

## Risks

1. **Rollup leak.** A site that still uses an inline deny-list shows intakes as
   real work. *Mitigation:* the allow-list flip + the Explore-audited site list
   (deliver/pipeline/detect/rollup/status/select); regression goldens that fail
   if an intake appears in any work rollup.
2. **Allow-list refactor changes behavior for initiatives.** `IsWorkSpec()`
   excludes initiatives, which some deny-list sites implicitly included.
   *Mitigation:* audit each converted site for whether it intended initiatives;
   preserve current behavior with an explicit initiative branch where needed.
3. **Predicate drift vs. registry.** Go predicates and registry `category` could
   disagree. *Mitigation:* registry `category` stays export-only; the Go
   predicate is the single rollup authority. A test asserts every type is
   covered by exactly one of the three predicates.
4. **Type proliferation.** Adding a parallel `idea` type alongside `intake`
   would duplicate a primitive. *Mitigation (this spec's central decision):* map
   onto `intake`; do not add `idea`. hero-code files the Swift rename follow-on.
5. **Provenance edge mismatch.** `hero_why` may follow a relation kind the
   promote path doesn't write. *Mitigation:* confirm the kind `hero_why`
   traverses before wiring promote; reuse it.

## Follow-ons (not in this spec)

- **hero-code:** decision superseding `hero-idea-primitive`; Swift `SpecType.idea
  → .intake`, `planning/ideas/ → planning/intake/` migration.
- **hero:** `/discover` lands intakes; `/design` consumes/promotes an intake
  (the deferred workflow loop).

## Completion Ledger

| Acceptance criterion | Status | Evidence |
|---|---|---|
| Core indexes `.hero/planning/intake/**` as `TypeIntake` | DONE | `typeFromPath`/`Parse` recognize intake — `TestTypeFromPath`, `TestParseHonorsIntakeFrontmatterType` |
| Committed-work rollups exclude intake via `IsWorkSpec()` allow-list | DONE | select.go `IsReady`, status.go, deliver.go, pipeline.go, detect.go, snapshot/rollup.go routed through predicate; `TestIntakeNotReady` |
| `status: planning` intake never in the work "planning" bucket | DONE | status.go pre-commitment section — `TestIntakeAbsentFromStatusWorkBuckets` + e2e smoke |
| `hero intake "<text>"` scaffolds a valid intake | DONE | intake.go `runIntakeCapture` — `TestIntakeCaptureCreatesSpec` |
| `hero intake promote` creates roadmap spec + provenance + sets promoted | DONE | `runIntakePromote` — `TestIntakePromoteCreatesFeatureWithProvenance`, `TestIntakePromoteBugType` |
| `hero intake reject` sets terminal rejected | DONE | `runIntakeReject` — `TestIntakeReject` |
| `hero why` traverses provenance back to the intake | DONE | graph node + `derived_from` edge — `TestSpecWriteGraphIntakeProvenance` + e2e `hero why` shows `→ Intake` |
| `hero_queue` never returns intake | DONE | `IsReady` excludes pre-commitment — `TestIntakeNotReady` |
| Existing feature/bug/initiative/knowledge rollups unchanged | DONE | full `go test ./...` green; allow-list refactor behavior-preserving |

All AC DONE. Build clean, `go vet` clean, full suite green. Deferred (per spec, not in scope): the `/discover`→`/design` workflow loop, and hero-code's Swift `idea→intake` rename follow-on.
## Handoff Trail

- 2026-06-26T18:22:51Z — in ← hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: hero-idea-primitive-core
  peer_spec: hero/hero-idea-primitive-core
  at_commit: 6f9fc13
  reason: "Engine-side delivery of the 'idea' primitive — see received_from above."

- 2026-06-27T00:29:07Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: advisory
  originating_spec: hero-idea-primitive-core
  at_commit: 386ab1e
  result_ref: private-peer-result-excluded
  reason: "Pre-delivery reconciliation: handoff asks for an 'idea' type, but hero already ships a structurally-identical 'intake' type. Confirmed independent convergence; mapping onto intake."

- 2026-06-27T01:28:55Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: advisory
  originating_spec: hero-idea-primitive-core
  at_commit: 386ab1e
  result_ref: private-peer-result-excluded
  reason: "Notify originator: the engine-side slice of hero-idea-primitive-core is delivered (mapped onto the existing 'intake' type, not a new 'idea' type). Surfaces the Swift idea→intake rename/migration follow-on that hero-code owns."

