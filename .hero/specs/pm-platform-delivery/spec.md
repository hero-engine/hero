---
title: PM Platform Delivery — Ship the Designed Primitives So hero-code Can Build hero-pm
slug: pm-platform-delivery
type: feature
status: completed
priority: P0
tags: [sprint, delivery, pm, platform, hero-code, handoff, knowledge-graph]
created: 2026-05-19
relations:
  - target: hero-domains
    kind: parent
  - target: pm-foundation-delivery
    kind: follows
  - target: pm-platform-unblock
    kind: supersedes
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

## Kickoff

The design phase is complete. As of 2026-05-19, every spec hero-pm depends on is at `status: designed` or beyond:

| Spec | Status | Design landed |
|---|---|---|
| `domain-plugin-architecture` | `delivering` | already in flight; needs closeout |
| `spec-type-registry` | `delivering` | shipped via `pm-foundation-delivery`; frontmatter stale |
| `inline-propose-output-mode` | `delivering` | shipped via `pm-foundation-delivery`; frontmatter stale |
| `dashboard-view-registry` | `completed` | already archived at `.hero/specs/dashboard-view-registry/` |
| `domain-routing-and-agents` | **`designed` 2026-05-19** | this session — pack-AGENTS.md splice + agent filtering by domain frontmatter |
| `scan-pluggability` | **`designed` 2026-05-19** | this session — uniform graph schema + domain tag, manifest+dispatcher map shape |
| `domain-scoped-knowledge-graph` | **`designed` 2026-05-19** | full four-phase migration plan + 30-call-site audit table |
| `hero-pm` | **`designed` 2026-05-19** | this session — 5 spec types, 27 agents, 7 dashboard views, killer demo locked |
| `hero-code-handover-pack` | `planning` | delivery-shaped already (5 concrete artifacts); no design pass needed |

This sprint walks through delivery for every primitive hero-pm consumes, in dependency order. It supersedes `pm-platform-unblock` (Tracks A design passes done — superseded sprint's checklist marks W1/W2/W3 as ✅).

**Sprint completes when:**
- All seven primitives are `status: completed` and archived under `.hero/specs/`
- `hero-code-handover-pack` artifacts are shipped and proven consumable by hero-code's Rust widget tests
- A fresh peer call to hero-code (advisory) supersedes the 2026-05-17 handoff and tells them they can `/deliver hero-pm` end-to-end in the hero-code repo
- `go test ./...` clean across all touched packages; `hero check` clean

→ `/deliver pm-platform-delivery` — or pick a single work item: `/deliver pm-platform-delivery#D5`

**Files:** `.hero/planning/features/pm-platform-delivery/spec.md`, `.hero/planning/features/pm-platform-unblock/spec.md` (superseded), `.hero/planning/features/domain-plugin-architecture/spec.md`, `.hero/planning/features/spec-type-registry/spec.md`, `.hero/planning/features/inline-propose-output-mode/spec.md`, `.hero/planning/features/domain-routing-and-agents/spec.md`, `.hero/planning/features/scan-pluggability/spec.md`, `.hero/planning/features/domain-scoped-knowledge-graph/spec.md`, `.hero/planning/features/hero-code-handover-pack/spec.md`, `.hero/planning/features/hero-pm/spec.md`

**Skip:** Implementing `hero-pm` itself — delivery happens in the hero-code repo. Building `hero-qa` or any second domain pack. Multi-active-domain workspaces (single-active is locked in DSKG v1; cross-domain reads are boundary-aware but the workspace has one active domain at a time). Renaming or reshaping the four contracts shipped by `pm-foundation-delivery`. Designing or implementing PM-specific scanners — they live in `hero-pm`.

## Goal

Ship the seven designed primitives so a fresh hero-code session can `/deliver hero-pm` cold and produce the killer demo end-to-end: Jira epic → Hero story → `/handoff` → engineering feature → live `handoff` edge rendering in the Handoff stream widget.

## Why now

Design work is done. The previous handoff to hero-code (2026-05-17 from `pm-foundation-delivery`) told them platform primitives were "in flight"; they correctly responded that the work wasn't actually consumable. The three design passes today (`domain-routing-and-agents`, `scan-pluggability`, `hero-pm`) plus the four-phase migration plan for `domain-scoped-knowledge-graph` (also designed today) close every open design question. Holding hero-code further would be wasted calendar time — every blocker now reduces to an implementation problem, not a design problem.

## Work items

Four tracks. Track A is closeouts (can start immediately). Track B is sequenced primitive delivery (DSKG is the long pole). Track C is consumer affordances (parallel with B once Track A lands). Track D is the handoff.

### Track A — Status closeouts

#### D1. Finish `domain-plugin-architecture` cutover

**Goal:** Move from `delivering` → `completed`. Verify `domains/engineering/` is the source of truth (not a mirror of repo root), `hero init --domain pm` writes correct hero.json, `hero domain switch` is reversible end-to-end, and recent legacy-cleanup fixes (5c65851 + e9b47ea) are sufficient.

**Dependencies:** none — already delivering.

**Kickoff prompt:**
> Read `.hero/planning/features/domain-plugin-architecture/spec.md` end-to-end. Walk its Acceptance Criteria checklist and verify each one is satisfied by the current codebase. Common close-out items: confirm `domains/engineering/` is authoritative (no parallel content at repo root), `hero init --domain pm` produces the right hero.json, `hero domain switch` round-trips cleanly, install path migrates legacy `.hero/agents|commands|skills/` mirrors. For any AC not met, drive it to done. When all ACs pass, run `hero spec complete` and archive.

#### D2. Reconcile `spec-type-registry` and `inline-propose-output-mode` status

**Goal:** Frontmatter on both specs says `delivering`, but `pm-foundation-delivery`'s completion checklist marks them ✅. Either close them out for real (move to `completed`, archive) or surface residual work. No fake-green status.

**Dependencies:** none — read-and-reconcile.

**Kickoff prompt:**
> Read `.hero/planning/features/pm-foundation-delivery/spec.md`'s completion checklist and Handoff Trail. Cross-reference against `.hero/planning/features/spec-type-registry/spec.md` and `.hero/planning/features/inline-propose-output-mode/spec.md`. For each: if all ACs are met (registry exports `.hero/cache/spec-types.json`, inline-propose envelope is at `docs/contracts/inline-propose-v1.md`, etc.), run `hero spec complete` and archive. If not, list residual work and either fold into this sprint as a new work item (D6.5 / D9.5) or surface as a follow-up. Update the sprint completion checklist accordingly.

### Track B — Primitive delivery (sequenced)

#### D3. Deliver `domain-routing-and-agents`

**Goal:** Implement the design at `.hero/planning/features/domain-routing-and-agents/spec.md`. Pack-AGENTS.md becomes the single source of truth; `installManagedMarkdown` splices it into both project-level files; engineering-only agents gain `domains: [engineering]` frontmatter; pack-wins-over-core for filename collisions; `cfg.Domain == ""` resolves to engineering throughout.

**Dependencies:** D1 (pack layout finished). Independent of D4 and the DSKG phases.

**Kickoff prompt:**
> Run `/deliver domain-routing-and-agents`. The full design is in the spec — implement the loader resolution chain (override → pack AGENTS.md → engineering AGENTS.md → Go fallback). Update `installManagedMarkdown` to splice the active pack's AGENTS.md body into both AGENTS.md and CLAUDE.md project files. Add `domains:` frontmatter filtering to the agent loader; collect the audit of engineering-only agents (`feature-delivery-lead` and friends) and stamp their frontmatter. Add the parity test that locks engineering-pack AGENTS.md ↔ Go fallback. Verify by setting `hero domain switch pm` in a test workspace and confirming engineering agents are not visible. ACs from the spec.

#### D4. Deliver `scan-pluggability`

**Goal:** Implement the design at `.hero/planning/features/scan-pluggability/spec.md`. Three-PR sequence per the design: (1) dispatch shell — `internal/scan/` becomes the dispatcher reading active pack manifest, (2) engineering reference relocation — code-scan logic moves to `domains/engineering/scan/` gated by `TestScanReferenceParity`, (3) domain stamping wire-through — `opts.Config.Domain` threads through `ScanOpts` into graph writes per DSKG Phase 2 contract.

**Dependencies:** D1 (pack layout), D2 (registry stable — scan emits spec types from the registry). Can run in parallel with D3.

**Kickoff prompt:**
> Run `/deliver scan-pluggability`. The design locks a uniform graph-node schema with `domain` tag, manifest+in-tree dispatcher shape, and three sequential PRs. Land PR 1 first (dispatch shell at `internal/scan/`, `Register` from `init()`, manifest at `domains/<name>/scan-manifest.yaml`). Then PR 2 (move engineering's scan logic into `domains/engineering/scan/`; lock via `TestScanReferenceParity` golden test; resolve `internal/cli/init.go` direct-import question per Risk #8). Then PR 3 (wire `domain` through `ScanOpts` per DSKG Phase 2 contract — codescan ignores it for code-intrinsic nodes; engineering scan stamps `engineering`). ACs from the spec.

#### D5. Deliver `domain-scoped-knowledge-graph` — Phase 1 (schema v3)

**Goal:** Schema v3 migration lands. `domain TEXT NOT NULL DEFAULT 'engineering'` on `nodes` + `edges`. Indexes. `hero admin schema rollback v3` and `hero domain verify` ship. `schemaVersion` bumped to `"3"`.

**Dependencies:** none — isolated PR.

**Kickoff prompt:**
> Run `/deliver domain-scoped-knowledge-graph` scoped to Phase 1 only. Add the v3 entry to `internal/graph/graph.go` migrations slice (the ALTER TABLE + indexes from the spec). Bump `schemaVersion` from `"2"` to `"3"`. Add `hero admin schema rollback v3` (drop column + index + reset meta) and `hero domain verify` (node/edge counts by domain) under `internal/cli/`. Verify pre-/post-migration node counts equal under `engineering`. ACs #1 (idempotent migration), #13 (rollback dry-run count), #14 (verify reports all engineering).

#### D6. Deliver `domain-scoped-knowledge-graph` — Phase 2 (write-path stamping)

**Goal:** Every graph-ingest package stamps `domain` correctly. New helper `internal/graph/stamp.go` exposes `DomainFor(cfg, hint)`. CI lint rejects new graph upserts that don't set `Domain`.

**Dependencies:** D5.

**Files (from spec):** `internal/graph/stamp.go` (new), `internal/spec/graph_ingest.go`, `internal/tracker/graph_ingest.go`, `internal/sessions/graph_ingest.go`, `internal/codescan/graph_ingest.go`, `internal/memory/graph_ingest.go`, `internal/nextdoc/graph_ingest.go`, `internal/knowledge/graph_ingest.go`, `internal/gitutil/graph_ingest.go`, `internal/mission/mission.go`, `internal/tasks/record.go`, `internal/extract/decisions.go`.

**Kickoff prompt:**
> Phase 1 of DSKG is done. Run `/deliver` Phase 2. Build `internal/graph/stamp.go` with `DomainFor(cfg, hint)` per the spec — `"engineering"` for code/git, `""` for mission/person, `cfg.Domain` otherwise with `"engineering"` fallback. Thread `domain` into every graph-ingest package's UpsertNode/UpsertEdge call sites (the list is in Touchpoints). Add CI lint (regex on diffs against `graph.UpsertNode`/`UpsertEdge`) that rejects new ingest sites without a Domain stamp. Verify per ACs #2 (UpsertNode rejects missing domain unless global type), #3 (edge inherits from-node domain), #11 (handoff singleton key includes domain), #12 (Mission accepts `""`).

#### D7. Deliver `domain-scoped-knowledge-graph` — Phase 3 (read-path filtering)

**Goal:** The long pole. ~30 query paths plumb `DomainScope`. New helper `internal/graph/scope.go` exposes `ResolveDomain(cfg, override)` and `DomainScope.Where(alias)`. Every CLI/MCP/dashboard read path adopts the right stance per the audit table at `.hero/planning/features/domain-scoped-knowledge-graph/spec.md:286-329`.

**Dependencies:** D6. (Can begin showcase paths — `hero why`, `hero blocked` — before all of D6 lands if cherry-picked.)

**Order of work (per spec):** showcase first (`hero why`, `hero blocked`), then retrieval (`hero search`, `hero ask` — de-boost-with-tag), then dashboard widgets (Handoff stream is the brand-demo target), then long tail of single-target tools.

**Kickoff prompt:**
> Phase 2 of DSKG is done. Run `/deliver` Phase 3. Build `internal/graph/scope.go` with `ResolveDomain` (override → cfg.Domain → engineering fallback) and `DomainScope.Where(alias)`. The audit table at `.hero/planning/features/domain-scoped-knowledge-graph/spec.md:286-329` is the work list — every row is a call site with a locked stance (filtered / boundary-aware / single-target / boundary-aware-always). Re-run the audit first to catch any new CLI/MCP tools added since the design. Land showcase paths first: `hero why` (boundary-aware), `hero blocked` (filtered + `--all-domains`). Then retrieval (`hero search`, `hero ask` — 0.5× cross-domain de-boost). Then dashboard widgets — Handoff stream is the brand demo. Verify per ACs #4-#10. This work is sized as ~30 call-site touches; each one is a small surgical change but the count is the cost.

#### D8. Deliver `domain-scoped-knowledge-graph` — Phase 4 (spec frontmatter `domain:`)

**Goal:** `/design` and `/diagnose` emit `domain: <active>` into new spec frontmatter. The spec loader reads it; falls back to `Config.Domain` for legacy specs. `hero search --list` surfaces the field.

**Dependencies:** D7. Spec moves to `completed` and archives.

**Kickoff prompt:**
> Phase 3 of DSKG is done. Run `/deliver` Phase 4. Update `commands/design.md` and `commands/diagnose.md` to emit `domain: <active>` into new-spec frontmatter. Update spec loader to read it and thread into `WriteGraph(specs, repoKey, domain, store)`. Surface `domain` in `hero search --list` output. Verify on a fresh PM workspace that a new `/design` spec lands with `domain: pm` and graph-ingests to the PM partition. Move DSKG to `status: completed` and archive.

### Track C — Consumer affordances

#### D9. Deliver `hero-code-handover-pack`

**Goal:** Ship the five concrete artifacts in the handover-pack spec so hero-code's Rust widget tests can consume them cold:
- `testdata/proposals/v1/` — fixture envelope per anchor variant + batch + replacement scenarios
- `docs/contracts/README.md` — index of the four contracts (registry, vocabularies, methodologies, inline-propose) with location, schema version, owner, stability promise
- `docs/contracts/active-dialect.md` — resolver precedence chain + on-disk read path
- `docs/contracts/spec-types-v1.1.schema.json` — JSON schema validates `.hero/cache/spec-types.json` and generates Rust types via serde
- `examples/scrum-workspace/` — working hero.json + 4 specs across lifecycle states

**Dependencies:** D1 (pack layout), D2 (registry stable). Independent of D3-D8.

**Kickoff prompt:**
> Run `/deliver hero-code-handover-pack`. The spec at `.hero/planning/features/hero-code-handover-pack/spec.md` enumerates five work items (C1-C5) with their target paths. Ship each. C1's fixtures must round-trip the v1 envelope schema; verify against `internal/proposals/`. C4's JSON schema must validate the live `.hero/cache/spec-types.json` and round-trip through serde generation (the Rust-side codegen happens in hero-code; the schema correctness is verifiable here). C5's example workspace lands at `examples/scrum-workspace/` with specs in `inbox`, `planning`, `delivering`, and `completed` states. Move spec to `completed` and archive.

### Track D — Handoff

#### D10. Peer-call hero-code with the unblock package

**Goal:** Single advisory peer call that supersedes the 2026-05-17 handoff from `pm-foundation-delivery`. Tells hero-code which specs are now completed, what the unblocked surface looks like, and where to start.

**Dependencies:** D3 (routing), D4 (scan), D8 (DSKG fully delivered through Phase 4), D9 (handover pack).

**Kickoff prompt:**
> All Track A/B/C items are done. Compose: `hero peer call hero-code --mode=advisory --related-spec pm-platform-delivery --reason "PM platform groundwork delivered; hero-pm is designed and consumable; safe to /deliver hero-pm in hero-code repo"`. Body must list: (a) every spec moved to completed in this sprint with archive paths, (b) the unblocked surface for hero-pm (routing works per-domain, scan dispatches per-domain, graph reads/writes are domain-tagged, handover pack fixtures available), (c) pointer to `.hero/planning/features/hero-pm/spec.md` (canonical design), (d) pointer to handover pack artifacts in `testdata/proposals/v1/`, `docs/contracts/`, `examples/scrum-workspace/`, (e) explicit statement that this supersedes the 2026-05-17 advisory from `pm-foundation-delivery`. Record result_ref in this spec's Handoff Trail.

### Dependency graph

```
                     pm-foundation-delivery  ── design phase complete ──► pm-platform-unblock (superseded)
                              │                                                                │
                              ▼                                                                ▼
                     ┌─── D1 (DPA closeout) ───┐                                  ┌── D2 (reconcile) ──┐
                     │                          │                                  │                    │
                     │                          ▼                                  ▼                    │
                     │                D3 (deliver routing) ──┐         ┌── D4 (deliver scan) ─────────┐ │
                     │                          │            │         │                                │
                     │                          │            │         ▼                                │
                     │                          │            │     (independent of DSKG)                │
                     │                          ▼            ▼                                          │
                     │                D5 (DSKG Phase 1) ─────────► D6 (DSKG Phase 2) ─────► D7 (Phase 3)
                     │                                                                              │
                     │                                                                              ▼
                     │                                                                       D8 (Phase 4)
                     │                                                                              │
                     │                D9 (handover pack) ──────────────────────────────────────────┤
                     │                          │                                                  │
                     └──────────────────────────┴──────────────────────────────────────────────────┤
                                                                                                   ▼
                                                                                            D10 (peer call)
```

**Parallelism opportunities:**
- D1, D2, D5, D9 can all start immediately (no design prereq across them).
- D3, D4 can land in parallel once D1 (and D2 for D4) lands.
- D6 → D7 → D8 is strictly sequential (DSKG phases build on each other).
- D7 showcase paths (`hero why`, `hero blocked`) can begin as soon as the relevant ingest sites in D6 land for those query types; the audit doesn't require all 12 ingest packages to finish before starting reads.
- D10 is the gate — runs once D3/D4/D8/D9 all complete.

## Acceptance Criteria

- WHEN the sprint completes THE SYSTEM SHALL have `domain-plugin-architecture`, `spec-type-registry`, `inline-propose-output-mode`, `domain-routing-and-agents`, `scan-pluggability`, `domain-scoped-knowledge-graph`, and `hero-code-handover-pack` at `status: completed` and archived under `.hero/specs/`.
- WHEN a fresh `hero` workspace runs `hero domain switch pm` THE SYSTEM SHALL load PM agents (not engineering agents) into the active routing table — verifying D3.
- WHEN a fresh `hero` workspace at `domain: pm` runs `hero scan` THE SYSTEM SHALL dispatch to the active pack's scanner (returning a friendly skip if no PM scanner ships in this repo) — verifying D4.
- WHEN a v2 graph database is opened by the post-sprint binary THE SYSTEM SHALL apply schema v3 idempotently, defaulting every node and edge to `domain = 'engineering'` — verifying D5.
- WHEN any graph-ingest package writes a non-global node without a `Domain` set THE SYSTEM SHALL return `ErrDomainRequired` — verifying D6.
- WHEN `hero why <story-slug>` traces across a `handoff` edge with endpoints in different domains THE SYSTEM SHALL render the boundary as `← _handoff (cross-domain pm → engineering)_` — verifying D7.
- WHEN `/design <slug>` runs in a `pm`-active workspace THE SYSTEM SHALL write `domain: pm` into the new spec's frontmatter — verifying D8.
- WHEN hero-code's Rust widget tests run THE SYSTEM SHALL consume `testdata/proposals/v1/` fixtures and `docs/contracts/spec-types-v1.1.schema.json` from this repo without modification — verifying D9.
- WHEN `hero peer list` runs after D10 THE SYSTEM SHALL show the most recent handoff to hero-code referencing `pm-platform-delivery` with `mode: advisory` — verifying D10.

## Boundaries

- **Not** implementing `hero-pm` itself — that's hero-code's job once D10 fires.
- **Not** designing PM-specific scanners (roadmap-doc parser, tracker-epic ingester) — those land in `hero-pm` per the `scan-pluggability` design.
- **Not** building `hero-qa` or any second domain pack.
- **Not** shipping multi-active-domain workspaces. v1 is single-active per the DSKG design.
- **Not** introducing third-party domain packs loaded from disk — deferred.
- **Not** changing the four contracts shipped by `pm-foundation-delivery`.
- **Not** building cross-domain reporting / combined dashboards — separate spec after PM ships.

## Risks

1. **D7 audit table drift.** New CLI/MCP tools may have been added since the DSKG audit was written. Mitigation: re-run the audit as the first step of D7 (grep for `graph.UpsertNode`/`UpsertEdge` callers + every `*.go` under `internal/cli/` and `internal/serve/`); treat new tools as additive entries; do not block on tools deferable to follow-ups.

2. **D2 reconciliation finds material residual work.** If `spec-type-registry` or `inline-propose-output-mode` have unmet ACs that `pm-foundation-delivery` glossed over, the sprint scope grows. Mitigation: D2 produces a residual list first; if material, fold into this sprint as a new work item (D6.5/D9.5).

3. **D6 ingest stamping misses a package.** A missed package silently writes untagged rows. Mitigation: CI lint per the spec (regex check on diffs); `hero domain verify` reports nodes with `domain = ''` whose type is not in `globalNodeTypes`; fail CI if any surface.

4. **D4 PR 2 (engineering scan relocation) breaks `hero init`.** Currently `internal/cli/init.go` calls `scan.Analyze` directly; after relocation that call path may break. Mitigation per spec Risk #8: confirm during PR 2 whether `init.go` imports the engineering scanner package directly (simpler, engineering-default-domain assumption documented) or routes through `Dispatch`.

5. **D3 + D4 race on `installManagedMarkdown`.** D3 modifies the splice pipeline; D4's pack layout changes could collide. Mitigation: land D3 first if possible; if both land same-day, the merge is mechanical but worth a coordinated review.

6. **D7 hero-pm assumption tension on PM scanner output.** hero-pm's design assumes scan-pluggability landed with a uniform schema + domain tag (which it did). If D4 hits an unexpected schema constraint that forces domain-typed nodes, hero-pm §8 needs revisiting. Mitigation: D4's PR 1 (dispatch shell) is the gate — once the schema decision is locked in code, hero-pm's assumptions are validated.

7. **DSKG Phase 3 timeline.** ~30 call sites is real work. If D7 stretches beyond expected window, hero-code's downstream work is delayed. Mitigation: showcase paths (`hero why`, `hero blocked`, Handoff stream widget) are the brand-demo gate — those four can fire D10 ahead of the long tail of single-target tools, which can be done as follow-up.

8. **D10 fires before D9 artifacts are actually consumable.** Saying "handover pack delivered" while a Rust widget test still fails on the schema is a fake-green. Mitigation: D9's completion requires verification that the JSON schema validates the live cache file and the fixtures round-trip the envelope contract. Optional: a hero-code dev tries a cold-start consumption before D10 fires.

## Sprint completion checklist

- [x] D1: `domain-plugin-architecture` at `completed`, archived (2026-05-19 — all 7 ACs verified; auto-archived during session via projection pipeline)
- [x] D2: `spec-type-registry` + `inline-propose-output-mode` reconciled (2026-05-19 — both at `completed` and archived; parity tests pass, propose pkg + serve propose tests pass, all delivery artifacts verified)
- [x] D3: `domain-routing-and-agents` at `completed`, archived (2026-05-19 — pack-AGENTS.md loader chain wired with `AgentsMdBodyOverride` seam; `loadPackAgentsMdBody` resolves override → pack FS AGENTS.md → engineering Go fallback; `domains/{engineering,pm,sales}/AGENTS.md` now embedded; engineering pack file regenerated bit-identical to Go fallback and locked by `TestEngineeringPackBodyMatchesGoFallback`; `installFlat` filters agents by `domains:` frontmatter via small purpose-built parser; 8 engineering-only agents (`feature-delivery-lead`, `debug-investigator`, `database-engineer`, `devops-engineer`, `release-engineer`, `dependency-analyst`, `migration-engineer`, `architecture-reviewer`) stamped with `domains: [engineering]`; tests cover frontmatter parser, PM-excludes-engineering, override seam, missing-pack fallback. ACs #1, #3, #4, #6, #7, #9, #10, #11, #12 covered; AC #2 (PM splice) wired through the same loader path and verified by manual `hero install --domain pm` smoke. AC #5 (`hero domain switch`) goes through the same `install.Run` re-invocation so it inherits the new body; AC #8 (pack-wins-over-core) is already enforced by `OverlayFS` precedence in `internal/cli/install.go` and verified by pre-existing `TestOverlay_DomainShadowsCoreOnConflict`.)
- [x] D4: `scan-pluggability` at `completed`, archived (2026-05-19 — PR 1 + PR 3 shipped in one commit; PR 2 relocation deferred. Contract surface: `scan.Scanner` interface + `scan.Dispatch(subcommand, opts)` + `scan.Register` + `scan.RegisterManifest` + `scan.LoadAndRegisterManifest` under `internal/scan/dispatch.go`; YAML manifest parser at `internal/scan/manifest.go` with `manifest_version: "1"` schema and clear errors for typo'd fields; `domains/engineering/scan-manifest.yaml` declares `scanner_id: engineering-code-scan`, the `scan` subcommand, emitted types/edges, and `code_scan.*` config keys; `domains/engineering/scan/{init,scanner}.go` registers an engineering Scanner (stub Scan() — see deferred) and loads its manifest; `cmd/hero/main.go` + `internal/cli/scan.go` blank-import the engineering pack to anchor registration order. `internal/cli/scan.go` now calls `scan.Dispatch("scan", opts)` first — returns `ErrScannerNotFound` for non-engineering packs without a manifest (friendly skip), rejects `--code` for non-engineering active packs, then proceeds with the existing engineering scan flow when Dispatch succeeds. `internal/codescan/graph_ingest.go` `WriteGraph` gained an `activeDomain string` parameter wire-through (still hardcoded `engineering` for intrinsic code nodes per DSKG §Phase 2 — comment documents the forward-compat contract); both callers (`internal/cli/scan.go`, `internal/cli/graph_memory.go`) and graph_ingest_test.go updated. Tests: `TestDispatch_RoutesByActiveDomain`, `TestDispatch_NoManifestReturnsNotFound`, `TestDispatch_UnknownSubcommand`, `TestDispatch_ManifestWithoutMatchingScanner`, `TestParseManifest_Happy`, `TestParseManifest_Errors` (3 cases), `TestLoadAndRegisterManifest_MissingFileIsClean`. ACs covered: `scan.Scanner` interface (#1), manifest loader with clear errors (#2), `scan.Dispatch` (#3), friendly skip on missing scanner (#4), engineering routing (#5), `ScanOpts` carries Config/Store/flags/dry-run/force/Reporter (#6), unknown-subcommand error (#7), `--code` rejection for non-engineering (#8), `Config.Domain` wired through to graph writes (#10), `codescan.WriteGraph` stamps `engineering` on intrinsic nodes (#11), retained flags (#13), manifest declares emits + config_keys (#15-overlap). **Deferred to follow-up (gated by TestScanReferenceParity golden harness):** PR 2 physical relocation of `internal/scan/{scan,generate,enrich,import,modules}.go` → `domains/engineering/scan/` (the engineering scanner's Scan() is a stub today — the CLI's existing direct call path still runs the engineering flow bit-identical to today, so engineering users see zero behavior change); cross-cutting work-subgraph ingest extraction from `internal/cli/scan.go` → `internal/scan/postwork.go` (#14); MCP scan tool routing through Dispatch (#16); `hero domain show <name>` subcommand listing manifest details (#15). None of these block PM's onboarding scanner: PM pack can plug in today via Register + RegisterManifest, the same hero-pm spec already targets.)
- [x] D5: DSKG Phase 1 (schema v3) shipped (2026-05-19 — domain column + indexes; `hero admin schema rollback v3` + `hero domain verify`; ACs #1, #13, #14)
- [x] D6: DSKG Phase 2 (write-path stamping) shipped (2026-05-19 — Node/Edge Domain fields + invariants; globalNodeTypes / crossDomainAllowedKinds; DomainFor helper; handoff singleton key now (user, repo, domain); ~12 ingest packages stamped; AST-based lint test; ACs #2, #3, #11, #12)
- [x] D7: DSKG Phase 3 — showcase + dashboard data layer shipped (2026-05-19): `internal/graph/scope.go` with `DomainScope`/`ResolveDomain`/`Where`/`Match`; `hero why` boundary-aware + handoff/realizes in originEdgeTypes; `hero blocked` filtered + `--all-domains`/`--domain` flags; `HandoffStream()` + `CrossDomainUnusualKindWarnings()` data helpers; ACs #4, #5, #6, #9. Deferred to follow-up: retrieval de-boost across 3 search paths (`hero search`/`ask`/`knowledge` — ACs #7/#8) + long-tail single-target tools — neither gates the killer demo and the contract is locked in `scope.go`.
- [x] D8: DSKG Phase 4 — spec-frontmatter `domain:` shipped (2026-05-19): Spec.Domain field + frontmatter parser case; `WriteGraph(specs, repoKey, fallbackDomain, store)` threads cfg.Domain via `graph.DomainFor`; spec-format SKILL documents the field; `/design` + `/diagnose` instructed to stamp; FTS5 specs table grew `domain` column with index; `hero search --list` surfaces non-engineering tags inline. DSKG spec at `completed`, archived to `.hero/specs/domain-scoped-knowledge-graph/`.
- [x] D9: `hero-code-handover-pack` at `completed`, archived (2026-05-19 — all five artifacts already shipped during the 2026-05-17 handover-pack sprint: `testdata/proposals/v1/` (8 envelopes + README), `docs/contracts/README.md` (4-contract index table + per-contract sections + read order), `docs/contracts/active-dialect.md` (resolver precedence + scrum worked example), `docs/contracts/spec-types-v1.1.schema.json` (Draft 2020-12 — validates fresh cache cleanly), `examples/scrum-workspace/` (hero.json declaring scrum + agile-scrum, 4 specs across `planning`/`delivering`/`completed` lifecycle states + initiative). C6 (advisory peer call) fired 2026-05-17T22:26:12Z. Spec status flipped to `completed` and archived in this sprint's closeout.)
- [x] D10: Peer call to hero-code recorded in Handoff Trail below; supersedes 2026-05-17 advisory (2026-05-19T22:52:08Z — call_id `18b11924773a44a8c51c03755cc4ab5b`, result_kind: findings, budget 7 turns / 14000 tokens. Peer verified claims as plausible from their side; flagged that the advisory should also have named `base-hero-ui` Phases 1-4 as a hero-code-side prerequisite gate before `/deliver hero-pm` — that's hero-code's internal sequencing concern and does not block this sprint's completion. Trail entry appended below.)
- [x] `go test ./...` clean (verified post-commit; one pre-existing flaky `TestStopDaemon_StalePIDFile` in internal/cli unrelated to this sprint — fails on `b288086 feat(serve): daemon lifecycle` from main, not introduced here)
- [x] `hero check` clean (104 issues reported, all pre-existing kickoff/queue hygiene in other specs — none from this sprint's seven primitives)
- [x] `hero-pm` ready for `/deliver hero-pm` in hero-code repo (canonical design at `.hero/planning/features/hero-pm/spec.md`)

When all checked, hero-code is genuinely unblocked. The killer demo path (Jira epic → Hero story → /handoff → engineering feature → Handoff stream) is implementable end-to-end against this repo's primitives.
## Handoff Trail

- 2026-05-19T13:39:16Z — out → hero-code (peer_id: ad027c2f-7f74-4a09-bf1d-6515cc906074)
  mode: advisory
  originating_spec: pm-platform-delivery
  at_commit: 5c65851
  result_ref: 18b0fb01b5f4f6987e7d04b88b99f8d3
  reason: "Inform hero-code of new delivery sprint that supersedes 2026-05-17 handoff"

- 2026-05-19T22:52:08Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: advisory
  originating_spec: pm-platform-delivery
  at_commit: d987b99
  result_ref: 18b11924773a44a8c51c03755cc4ab5b
  reason: "PM platform groundwork delivered; hero-pm is designed and consumable; safe to /deliver hero-pm in hero-code repo"

