---
title: PM Platform Unblock — Finish the Groundwork hero-code Needs to Build hero-pm
slug: pm-platform-unblock
type: feature
status: superseded
priority: P0
tags: [sprint, delivery, pm, platform, hero-code, handoff, knowledge-graph, superseded]
created: 2026-05-19
superseded_by: pm-platform-delivery
relations:
  - target: hero-domains
    kind: parent
  - target: pm-foundation-delivery
    kind: follows
  - target: pm-platform-delivery
    kind: superseded-by
  - target: domain-plugin-architecture
    kind: delivers
  - target: domain-routing-and-agents
    kind: delivers
  - target: scan-pluggability
    kind: delivers
  - target: domain-scoped-knowledge-graph
    kind: delivers
  - target: hero-code-handover-pack
    kind: delivers
  - target: hero-pm
    kind: unblocks
horizon: now
smoke: deferred
---

## Superseded

This sprint was written 2026-05-19 to bundle design + delivery work for the seven primitives hero-pm depends on. The same day, the three remaining design passes (W1 `domain-routing-and-agents`, W2 `scan-pluggability`, W3 `hero-pm`) landed in-session, leaving only delivery work. The delivery work was lifted into a clean delivery-only sprint at [`pm-platform-delivery`](../pm-platform-delivery/spec.md) — that's the live sprint to walk through.

Treat the rest of this document as historical record of the design-track kickoff prompts. Track A (W1/W2/W3) is complete; Tracks B/C/D moved to `pm-platform-delivery` as D1-D10.

## Kickoff

`pm-foundation-delivery` shipped the cross-language **contracts** hero-code consumes (spec-type registry, vocabularies, methodologies, inline-propose envelope). It explicitly skipped the **graph and routing primitives** required to render the killer demo — a PM `Story` handing off to an engineering `Feature` and surfacing in the Handoff stream. Hero-code's repeated "we still have groundwork" responses point at exactly this gap, with `domain-scoped-knowledge-graph` (designed 2026-05-19, zero code) as the loudest blocker.

This sprint closes the remaining platform groundwork so a fresh hero-code session can `/deliver hero-pm` against a real graph, real routing, real scanners, and a complete handover pack — without another peer-call round trip into `hero` for missing primitives.

**Sprint completes when:**
- `domain-plugin-architecture` is fully delivered (not just "delivering") and the engineering pack lives at `domains/engineering/`
- `spec-type-registry` and `inline-propose-output-mode` frontmatter are reconciled (move to `completed` if `pm-foundation-delivery` actually finished them; surface residual work if not)
- `domain-routing-and-agents` is designed and delivered — active-domain `AGENTS.md` and agent loader work
- `scan-pluggability` is designed and delivered — `hero scan` dispatches to the active pack's scanner
- `domain-scoped-knowledge-graph` is delivered through all four phases (schema v3, write-path stamping, read-path filtering, spec-frontmatter `domain:` field)
- `hero-code-handover-pack` is delivered — `testdata/proposals/v1/`, `docs/contracts/README.md`, `docs/contracts/active-dialect.md`, `docs/contracts/spec-types-v1.1.schema.json`, `examples/scrum-workspace/`
- `hero-pm` has run through `/design` and moved from `planning` → `designed` with the canonical Changes section
- A fresh peer call to hero-code (advisory) hands over the unblocked platform with pointers to every artifact

Each work item below has a paste-ready kickoff so items can be picked up in fresh sessions independently where dependencies allow.

→ `/deliver pm-platform-unblock` (or pick a single work item: `/deliver pm-platform-unblock#W3`)

**Files:** `.hero/planning/features/pm-platform-unblock/spec.md`, `.hero/planning/features/domain-plugin-architecture/spec.md`, `.hero/planning/features/domain-routing-and-agents/spec.md`, `.hero/planning/features/scan-pluggability/spec.md`, `.hero/planning/features/domain-scoped-knowledge-graph/spec.md`, `.hero/planning/features/hero-code-handover-pack/spec.md`, `.hero/planning/features/hero-pm/spec.md`, `.hero/planning/features/hero-pm/handoff-to-hero-code.md`

**Skip:** Implementing `hero-pm` itself (that's hero-code's job once unblocked). Building `hero-qa` or any second domain pack. Multi-active-domain workspaces. Cross-org domain conventions / cloud federation rules. Renaming or re-shaping the four contracts shipped by `pm-foundation-delivery`.

## Goal

Get every platform primitive that `hero-pm` depends on from its current state (planning / designed / delivering) to **delivered and consumable**, plus get `hero-pm` itself to `designed` status, plus ship the handover artifacts hero-code's Rust tests need. End state: hero-code can `/deliver hero-pm` without needing another change in this repo.

## Why now

Hero-code is correct that platform groundwork is incomplete:

- `domain-scoped-knowledge-graph` was designed today (2026-05-19) but zero code has landed. Schema is still v2; no `domain` column on `nodes`/`edges`; no `internal/graph/scope.go`; no `internal/graph/stamp.go`.
- `domain-routing-and-agents` and `scan-pluggability` are stubs without designs — hero-code cannot route to PM agents or scan a PM workspace.
- `hero-code-handover-pack` lists five concrete artifacts hero-code needs (fixtures, contracts index, active-dialect doc, JSON schema, scrum workspace) — none shipped.
- `hero-pm` is stuck at `planning` with seven items in its `depends-on` list; the `/design` pass never ran.

`pm-foundation-delivery` shipped the contracts. This sprint ships the **runtime** behind those contracts — graph partitioning, routing, scanning, fixtures, and the design pass that ties it all together. After this, the parent `hero-domains` initiative can validate end-to-end via the PM domain.

## Work items

Three tracks. Track A unblocks Track B; Track C can run in parallel with both once W1 lands.

### Track A — Design passes (parallelizable; no Go code)

#### W1. `/design domain-routing-and-agents`

**Goal:** Produce the canonical design for domain-aware agent routing — active-domain `AGENTS.md`, agent loader that selects the active pack's table, natural-language routing scoped to the active domain. Single-active-domain v1; multi-domain deferred to `domain-scoped-knowledge-graph` read-path work.

**Dependencies:** `domain-plugin-architecture` (foundation primitive) — needs to be far enough along that `domains/engineering/` exists and the agent loader has a hook for env-conditional resolution.

**Kickoff prompt (paste in fresh session):**
> Read `.hero/planning/features/domain-routing-and-agents/spec.md` end-to-end, then read `.hero/planning/features/domain-plugin-architecture/spec.md` to understand the pack layout. Run `/design domain-routing-and-agents`. First job: enumerate every place the model consults `AGENTS.md` or the natural-language routing table today (grep for AGENTS.md and the routing table). Then design the loader that selects the active pack's table. Resolve open question #1 (multi-domain loader behavior) by picking a single-active-v1 stance and deferring multi-active to `domain-scoped-knowledge-graph`.

#### W2. `/design scan-pluggability`

**Goal:** Produce the canonical design for domain-aware `hero scan` — dispatch surface, scan output schema (do all domains emit the same node/edge types into the graph, or domain-typed nodes?), engineering's code scan as the reference impl under `domains/engineering/scan/`.

**Dependencies:** `domain-plugin-architecture` and `spec-type-registry` — scan output writes spec types from the registry; the pack layout must exist for the dispatch.

**Kickoff prompt:**
> Read `.hero/planning/features/scan-pluggability/spec.md` end-to-end. Run `/design scan-pluggability`. First decision: scan output schema (uniform node/edge types vs domain-typed). Then design the dispatch surface — `hero scan` reads active pack and runs its scanner. Engineering's code scan moves under `domains/engineering/scan/` as the reference impl. PM-specific scanners are explicitly out of scope (they live in `hero-pm`); this spec only ships the dispatch shape.

#### W3. `/design hero-pm`

**Goal:** Resolve hero-pm's remaining open questions and produce the canonical Changes section. Incorporate (don't re-derive) the four sibling files: `research-brief.md`, `mockup-brief.md`, `agent-pack-design.md`, `handoff-to-hero-code.md`. Move from `planning` → `designed`.

**Dependencies:** W1, W2 designs locked enough that hero-pm's "depends-on" can resolve. The four hero-pm sibling files are already authored.

**Kickoff prompt:**
> Read `.hero/planning/features/hero-pm/spec.md` and its four sibling files in this order: `research-brief.md`, `mockup-brief.md`, `agent-pack-design.md`, `handoff-to-hero-code.md`. Run `/design hero-pm`. Goal: resolve the open questions in the spec stub and produce the canonical Changes section. Reference (don't restate) the four siblings. The output is the final design hero-code reads to build the PM pack; it should be self-sufficient when combined with the siblings.

### Track B — Go implementation (sequenced)

#### W4. Finish `domain-plugin-architecture` cutover

**Goal:** Move `domain-plugin-architecture` from `delivering` → `completed`. Verify `domains/engineering/` exists, `hero init --domain` works, `hero domain switch` works, the engineering pack is the active default, and `hero.json`'s `domain` field is honored end-to-end.

**Dependencies:** Already delivering; this is the close-out.

**Kickoff prompt:**
> Read `.hero/planning/features/domain-plugin-architecture/spec.md`. Check the AC checklist for any item not yet satisfied. Drive each one to done. Common close-out items: ensure `domains/engineering/` is the source of truth (not a mirror of repo root), `hero init --domain pm` writes the correct hero.json, `hero domain switch` is reversible, and the install path migrates legacy `.hero/agents|commands|skills/` mirrors (note: 5c65851 + e9b47ea already handled some of this). Move spec to `completed` and archive via `hero spec complete`.

#### W5. Reconcile `spec-type-registry` and `inline-propose-output-mode` status

**Goal:** `pm-foundation-delivery`'s completion checklist marks both as ✅, but their frontmatter says `delivering`. Either close them out for real (move to `completed`, archive) or surface the residual work. No fake-green status.

**Dependencies:** `pm-foundation-delivery` itself — read its checklist + handoff trail to see what shipped.

**Kickoff prompt:**
> Read `.hero/planning/features/pm-foundation-delivery/spec.md`'s completion checklist and handoff trail. Then read `.hero/planning/features/spec-type-registry/spec.md` and `.hero/planning/features/inline-propose-output-mode/spec.md`. Cross-check ACs against what `pm-foundation-delivery` actually shipped. For each spec: if all ACs are met, run `hero spec complete` and archive. If not, list the residual work and either fold it into this sprint as a new work item or surface it as a follow-up.

#### W6. Deliver `domain-routing-and-agents`

**Goal:** Implement the design produced in W1. Active-domain `AGENTS.md`, agent loader, scoped routing table.

**Dependencies:** W1 (design), W4 (pack layout exists), W5 (registry stable).

**Kickoff prompt:**
> Read `.hero/planning/features/domain-routing-and-agents/spec.md` (now with full design after W1). Run `/deliver domain-routing-and-agents`. Implement the agent loader, the active-domain AGENTS.md resolution, and the scoped routing table. Verify by setting `hero domain switch pm` in a test workspace and confirming engineering agents (`feature-delivery-lead`, etc.) are not loaded.

#### W7. Deliver `scan-pluggability`

**Goal:** Implement the design from W2. `hero scan` dispatches to the active pack's scanner; engineering's code scan is the reference impl.

**Dependencies:** W2 (design), W4 (pack layout), W5 (registry stable). Can run in parallel with W6.

**Kickoff prompt:**
> Read `.hero/planning/features/scan-pluggability/spec.md` (now with full design after W2). Run `/deliver scan-pluggability`. Move engineering's existing scan logic under `domains/engineering/scan/`. Build the dispatch surface that reads active pack and runs the right scanner. Verify by running `hero scan` in an engineering workspace (unchanged behavior) and confirming the dispatch is in place for a future PM scanner.

#### W8. Deliver `domain-scoped-knowledge-graph` — Phase 1 (schema v3 migration)

**Goal:** Schema v3 lands. `domain TEXT NOT NULL DEFAULT 'engineering'` on `nodes` and `edges`. Indexes. `hero admin schema rollback v3` and `hero domain verify` ship.

**Dependencies:** None (this is one PR, isolated from the read/write path work).

**Files (per spec):** `internal/graph/graph.go`, `internal/graph/node.go`, `internal/graph/edge.go`, `internal/cli/admin*.go` (new subcommands).

**Kickoff prompt:**
> Read `.hero/planning/features/domain-scoped-knowledge-graph/spec.md` end-to-end. Run `/deliver domain-scoped-knowledge-graph` scoped to Phase 1 only. Add the v3 migration to `internal/graph/graph.go`'s migrations slice. Bump `schemaVersion` from "2" to "3". Add `hero admin schema rollback v3` and `hero domain verify` under `internal/cli/admin*.go`. Verify pre- and post-migration node counts are equal under `engineering`. ACs from the spec: AC #1 (idempotent schema migration), AC #13 (rollback dry-run count), AC #14 (verify reports all engineering).

#### W9. Deliver `domain-scoped-knowledge-graph` — Phase 2 (write-path stamping)

**Goal:** Every graph-ingest package stamps `domain` correctly. New helper `internal/graph/stamp.go` exposes `DomainFor(cfg, hint)`. CI lint rejects new graph upserts that don't set `Domain`.

**Dependencies:** W8.

**Files:** `internal/graph/stamp.go` (new), `internal/spec/graph_ingest.go`, `internal/tracker/graph_ingest.go`, `internal/sessions/graph_ingest.go`, `internal/codescan/graph_ingest.go`, `internal/memory/graph_ingest.go`, `internal/nextdoc/graph_ingest.go`, `internal/knowledge/graph_ingest.go`, `internal/gitutil/graph_ingest.go`, `internal/mission/mission.go`, `internal/tasks/record.go`, `internal/extract/decisions.go`.

**Kickoff prompt:**
> Phase 1 of `domain-scoped-knowledge-graph` is done. Now `/deliver` Phase 2. Build `internal/graph/stamp.go` with `DomainFor(cfg, hint)` per the spec (engineering for code/git, "" for mission/person, cfg.Domain otherwise). Thread `domain` into every graph-ingest package's UpsertNode/UpsertEdge call sites. Add a CI lint (regex on diffs is sufficient — the call sites are bounded) that rejects new ingest sites without a Domain stamp. Verify: ACs #2 (UpsertNode rejects missing domain unless global), #3 (edge inherits from-node domain), #11 (handoff singleton key includes domain), #12 (Mission accepts "").

#### W10. Deliver `domain-scoped-knowledge-graph` — Phase 3 (read-path filtering)

**Goal:** The long pole. ~30 query paths plumb `DomainScope` through. New helper `internal/graph/scope.go` exposes `ResolveDomain(cfg, override)` and `DomainScope.Where(alias)`. Every CLI/MCP/dashboard read path adopts the right stance per the audit table in the spec.

**Dependencies:** W9.

**Order of work (per spec):** `hero why` and `hero blocked` first (showcase), then `hero search` / `hero ask` (retrieval), then dashboard widgets, then long tail.

**Kickoff prompt:**
> Phase 2 of `domain-scoped-knowledge-graph` is done. Now `/deliver` Phase 3 — the long pole. Build `internal/graph/scope.go` with `ResolveDomain` and `DomainScope.Where` per the spec. The spec's audit table at `.hero/planning/features/domain-scoped-knowledge-graph/spec.md:286-329` is the work list — every row is a call site that must adopt a stance (filtered / boundary-aware / single-target). Land the showcase paths first: `hero why`, `hero blocked`. Then `hero search` / `hero ask` (de-boost-with-tag). Then dashboard widgets — Handoff stream is the brand-demo target. Verify per AC #4–10 in the spec.

#### W11. Deliver `domain-scoped-knowledge-graph` — Phase 4 (spec frontmatter `domain:`)

**Goal:** `/design` and `/diagnose` emit `domain: <active>` into new spec frontmatter. The spec loader reads it; falls back to `Config.Domain` for legacy specs. `hero search --list` surfaces the field.

**Dependencies:** W10.

**Kickoff prompt:**
> Phase 3 of `domain-scoped-knowledge-graph` is done. Now Phase 4 — spec-frontmatter `domain:` field. Update `commands/design.md` and `commands/diagnose.md` to emit `domain: <active>` into the new-spec frontmatter. Update the spec loader to read it and thread into `WriteGraph(specs, repoKey, domain, store)`. Confirm `hero search --list` surfaces the field. Move the spec to `completed`.

### Track C — Handover affordances (parallelizable with B once W4 lands)

#### W12. Deliver `hero-code-handover-pack`

**Goal:** Ship the five concrete artifacts in the handover-pack spec: inline-propose test fixtures, contracts README index, active-dialect resolver doc, spec-types JSON schema, scrum example workspace.

**Dependencies:** `pm-foundation-delivery` shipped the contracts; W4 confirms pack layout; W5 confirms registry stability.

**Kickoff prompt:**
> Read `.hero/planning/features/hero-code-handover-pack/spec.md` end-to-end. Run `/deliver hero-code-handover-pack`. Ship all five work items: C1 (testdata/proposals/v1/ fixtures), C2 (docs/contracts/README.md), C3 (docs/contracts/active-dialect.md), C4 (docs/contracts/spec-types-v1.1.schema.json validates `.hero/cache/spec-types.json` and generates Rust types via serde), C5 (examples/scrum-workspace/ with hero.json + 4 specs across lifecycle states). Each must be consumable cold by hero-code's Rust widget tests.

### Track D — Closeout

#### W13. Peer-call hero-code with the unblock package

**Goal:** Single advisory peer call summarizing what shipped, what hero-code can now build against, and the pointer set for kickoff.

**Dependencies:** W3 (hero-pm designed), W6 (routing), W7 (scan), W11 (DSKG fully delivered), W12 (handover pack).

**Kickoff prompt:**
> All Track A/B/C items are done. Compose a `hero peer call hero-code --mode=advisory --related-spec pm-platform-unblock --reason "PM platform groundwork complete; hero-pm is designed; safe to /deliver hero-pm"` whose body lists: (a) every spec moved to completed in this sprint, (b) the unblocked surface for hero-pm, (c) pointer to `hero-pm/spec.md` canonical Changes section, (d) pointer to `hero-code-handover-pack` artifacts. Record the result_ref in this spec's Handoff Trail.

### Dependency graph

```
                    pm-foundation-delivery (already shipped contracts)
                              │
                              ▼
          ┌─────────── W4 (DPA closeout) ───────────┐
          │                                          │
          │                                          ▼
          │                              W5 (reconcile registry +
          │                                  inline-propose status)
          │                                          │
          ▼                                          │
W1 (design routing) ─┐                              │
                     │                              │
W2 (design scan) ────┤                              │
                     │                              │
                     ▼                              │
              W3 (design hero-pm) ◀─────────────────┘
                     │
                     ▼
          ┌─── W6 (deliver routing) ───┐
          │                            │
          │   W7 (deliver scan) ───────┤
          │                            │
          │   W8 (DSKG Phase 1) ──────┐│
          │            │              ││
          │            ▼              ││
          │   W9 (DSKG Phase 2)       ││
          │            │              ││
          │            ▼              ││
          │   W10 (DSKG Phase 3)      ││
          │            │              ││
          │            ▼              ││
          │   W11 (DSKG Phase 4)      ││
          │                           ││
          │   W12 (handover pack) ────┤│
          │                           ││
          └───────────┬───────────────┘│
                      ▼                │
                W13 (peer call) ◀──────┘
```

W6/W7/W8/W12 are independent of each other once their prereqs land. W9 → W10 → W11 is strictly sequential (DSKG phases). W13 is the gate.

## Acceptance Criteria

- WHEN this sprint completes THE SYSTEM SHALL have `domain-plugin-architecture`, `spec-type-registry`, `inline-propose-output-mode`, `domain-routing-and-agents`, `scan-pluggability`, `domain-scoped-knowledge-graph`, and `hero-code-handover-pack` all at `status: completed` and archived under `.hero/specs/`.
- WHEN this sprint completes THE SYSTEM SHALL have `hero-pm` at `status: designed` with a canonical Changes section produced by the W3 design pass.
- WHEN `hero peer list` runs after W13 THE SYSTEM SHALL show the most recent handoff to hero-code referencing `pm-platform-unblock` with `mode: advisory`.
- WHEN a fresh `hero` workspace runs `hero domain switch pm` after this sprint THE SYSTEM SHALL load PM agents (not engineering agents) into the active routing table — verifying W6.
- WHEN a fresh `hero` workspace at `domain: pm` runs `hero scan` THE SYSTEM SHALL dispatch to the active pack's scanner — verifying W7.
- WHEN a v2 graph database is opened by a post-sprint binary THE SYSTEM SHALL apply schema v3 idempotently, defaulting every node and edge to `domain = 'engineering'` — verifying W8.
- WHEN `hero why <story-slug>` runs on a workspace with a cross-domain `handoff` edge THE SYSTEM SHALL render the boundary as `← _handoff (cross-domain pm → engineering)_` — verifying W10.
- WHEN hero-code's Rust widget tests run THE SYSTEM SHALL consume `testdata/proposals/v1/` fixtures and `docs/contracts/spec-types-v1.1.schema.json` from this repo without modification — verifying W12.

## Boundaries

- **Not** implementing `hero-pm` itself. That's hero-code's job. This sprint ends with hero-pm at `designed` and a peer call out.
- **Not** building `hero-qa` or any second domain pack.
- **Not** shipping multi-active-domain workspaces. `domain-scoped-knowledge-graph` v1 is single-active; multi-active is a future spec.
- **Not** renaming or re-shaping the four contracts shipped by `pm-foundation-delivery` — registry, vocabularies, methodologies, inline-propose envelope.
- **Not** designing PM-specific scanners (roadmap-doc parser, tracker-epic ingester) — those live in `hero-pm`.
- **Not** introducing third-party domain packs loaded from disk — deferred.
- **Not** cross-domain reporting / combined dashboards — separate spec after PM ships.

## Risks

1. **DSKG Phase 3 read-path scope creep.** The audit table in `domain-scoped-knowledge-graph/spec.md` enumerates ~30 call sites. New CLI/MCP tools added since the audit was written would need to be folded in. Mitigation: re-run the audit as the first step of W10; treat any new tools as additive entries to the table; do not block the sprint on tools deferred to a follow-up.
2. **W3 (design hero-pm) reveals primitives we missed.** The full design pass on hero-pm might surface a dependency that's not currently in the sprint. Mitigation: run W3 early (it's parallelizable with W1/W2/W4/W5); fold any new primitive into Track B as W6.5 / W7.5 / etc. before W13.
3. **W5 reconciliation finds more residual work than expected.** If `spec-type-registry` or `inline-propose-output-mode` have unmet ACs that `pm-foundation-delivery` glossed over, the sprint scope grows. Mitigation: W5 produces a residual list before the sprint commits; if material, decide whether to fold into this sprint or spin a separate closeout.
4. **DSKG Phase 2 ingest stamping misses a package.** Adding `domain` to every ingest is a sweep; a missed package silently writes untagged rows. Mitigation: CI lint per the spec (regex check on diffs against `graph.UpsertNode`/`UpsertEdge` call sites); `hero domain verify` reports nodes with `domain = ''` whose type is not in `globalNodeTypes` and fails CI if any surface.
5. **Hero-code starts work against incomplete state.** If hero-code sees the peer call from `pm-foundation-delivery` (already sent 2026-05-17) and begins building before this sprint completes, they hit the same blockers again. Mitigation: W13's peer call explicitly supersedes the pm-foundation-delivery handoff; until then, the existing reference materials (`handoff-to-hero-code.md`) are correctly marked as "wait for primitives".
6. **Sprint feels too big to land coherently.** 13 work items across 4 tracks is a lot. Mitigation: items can land in any order that respects the dependency graph; W4/W5/W8/W12 are immediately pickup-able with no design prereq; the sprint can ship in roughly two halves (Track A + W4/W5/W8/W12 first, then W6/W7/W9-11 second).

## Sprint completion checklist

- [x] W1: `domain-routing-and-agents` design produced (status: `designed` 2026-05-19)
- [x] W2: `scan-pluggability` design produced (status: `designed` 2026-05-19)
- [x] W3: `hero-pm` design produced with canonical Changes section (status: `designed` 2026-05-19)
- [ ] W4: `domain-plugin-architecture` closeout complete (status: `completed`, archived)
- [ ] W5: `spec-type-registry` and `inline-propose-output-mode` status reconciled (both `completed` and archived, or residual work folded into sprint)
- [ ] W6: `domain-routing-and-agents` delivered (status: `completed`, archived)
- [ ] W7: `scan-pluggability` delivered (status: `completed`, archived)
- [ ] W8: DSKG Phase 1 (schema v3 migration) shipped
- [ ] W9: DSKG Phase 2 (write-path stamping) shipped
- [ ] W10: DSKG Phase 3 (read-path filtering) shipped
- [ ] W11: DSKG Phase 4 (spec-frontmatter `domain:`) shipped; DSKG `completed`, archived
- [ ] W12: `hero-code-handover-pack` delivered (status: `completed`, archived); all five artifacts shipped and verified consumable
- [ ] W13: Peer call to hero-code recorded in Handoff Trail below
- [ ] `hero-pm` is `designed` and ready for a fresh `/deliver hero-pm` session in the hero-code repo
- [ ] `go test ./...` clean across all changes
- [ ] `hero check` clean

When all checked, hero-code is genuinely unblocked.

## Handoff Trail

_(populated as work items complete)_
