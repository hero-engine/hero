# Delivery audit — spec-size-and-promotion-nudge

**Audited:** `git diff HEAD~5..HEAD` (commits fc982e7, 985e95d, 705130f, 7d5dc1a, 3d0b942)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC#1 — `size:` accepted on feature/bug/epic/initiative — `internal/spec/spec.go:94,100,143,390,467` (validSizes, validateSize, Size field, Parse wiring); tests `TestParseSize`, `TestParseSize_AllValidValuesRoundTrip` (`internal/spec/spec_test.go:689,724`). AC was correctly narrowed to drop `enhancement` (spec-type doesn't ship today) — narrowing recorded inline in the spec.
- [✓] AC#2 — invalid `size:` rejected at load time — `internal/spec/spec.go:100-110`; test `TestParseSize_InvalidValueErrors` (`spec_test.go:747`) asserts error includes field name, bad value, and full enum.
- [✓] AC#3 — `/design` stamps `size:` — `domains/engineering/commands/design.md:8` adds explicit "Stamp `size:` in frontmatter" paragraph; `feature-delivery-lead.md:45` and `platform-delivery-lead.md:43` load `spec-sizing` skill at design time.
- [✓] AC#4 — `hero estimate` prints declared vs computed + drift — `internal/cli/cost.go:80-92,374-385` (`costEstimate.Declared/Drift`, `LeafDrift`, side-by-side print); JSON mode at `cost.go:411-413`; table view at `cost.go:436-462`. Tests `TestLeafDrift_DriftSignals` and `TestBucketFromPoints` (`cost_test.go:283,318`) cover giant emission + all four drift signal combos.
- [✓] AC#5 — `hero size <slug>` prints declared — `internal/cli/size.go:95` (`runSizeGet`); tests `TestSize_Get_Unset`, `TestSize_Get_Declared` (`size_test.go:10,32`).
- [✓] AC#6 — `hero size <slug> <tier>` non-destructive update — `size.go:110` (`runSizeSet` uses `spec.SetFrontmatterField`); tests `TestSize_Set_NewField`, `TestSize_Set_UpdateExisting`, `TestSize_Set_RejectsInvalidTier` (`size_test.go:55,93,130`).
- [✓] AC#7 — `hero size --check` exits non-zero on drift — `size.go:142` (`runSizeCheck`); covers leaf (slice 2) and container (slice 3); tests `TestSize_Check_FindsDrift`, `TestSizeCheckCmd_LeafAndContainerDrift` (`size_test.go:190`, `size_drift_test.go:104`).
- [✓] AC#8/#9/#10 — large soft / x-large strong / giant super-strong + `size_ack` — `domains/engineering/skills/spec-sizing/SKILL.md:60-82,99-188` (nudge tables, paste-ready phrasing per tier, `size_ack: giant` protocol); delivery leads load the skill via `feature-delivery-lead.md:104` / `platform-delivery-lead.md:67` step 4d/3b.
- [✓] AC#11 — container drift declared < rollup — `internal/snapshot/rollup.go:626,669,681,738` (`RollupChildSizes`, `ContainerDriftReport`, `ContainerDrift`, `BuildParentMap`); tests `TestContainerDrift_DeclaredBelowRollup` + 5 sibling tests (`size_drift_test.go:92-194`); uses midpoint-sum-and-rebucket per Implementation Notes.
- [✓] AC#12 — `tracker.type: none` → most-aggressive regime + local promotion — `internal/sizing/sizing.go:206,222` (`TrackerCapability`, `NudgeRegime`); `internal/cli/size.go:190-208` (`printTrackerCapability`, `WorkspaceTrackerCapability`); skill phrasing at `spec-sizing/SKILL.md:93-95`.
- [✓] AC#13 — `supports_hierarchy: true` raises threshold + offers parent creation — `internal/tracker/tracker.go:96-102` (`SupportsHierarchy()` on interface); Jira/Linear true, GitHub false (`size_mapping.go::TypeSupportsHierarchy`); test `TestSupportsHierarchy_Defaults`, `TestTypeSupportsHierarchy` (`size_mapping_test.go:18,34`); "create parent in tracker" lives as guidance only (skill `SKILL.md:95,214`) — no code path implements it, ledger noted this is not exercised live.
- [✓] AC#14 — `sync pull` seeds local from tracker when absent — `internal/cli/pull.go:67-83`; planner `internal/tracker/size_mapping.go::PlanSizePull`; test `TestPlanSizePull_SeedLocal` (`size_mapping_test.go:188`).
- [✓] AC#15 — `sync pull` surfaces conflict — `pull.go:81` (warning emit), planner case `SizeSyncConflict`; test `TestPlanSizePull_Conflict` (`size_mapping_test.go:209`).
- [~] AC#16 — `sync push` writes to tracker non-destructively — **PARTIAL**: `internal/cli/sync.go:106-123` invokes `PlanSizePush` and surfaces the plan as a hint/conflict warning, but no path actually writes the size to the tracker. `Tracker` interface (`tracker.go:67-118`) has no `UpdateSize`/`UpdateField` method — only `UpdateStatus`/`AddComment`/`CreateIssue`. Per `TestPlanSizePush_ConflictNonDestructive` (`size_mapping_test.go:259`) the non-destructive contract is tested at planner level. Engineer's stated reason is concrete but slightly incomplete — see Audit notes below.
- [✓] AC#17 — `hero check` + `hero_warnings` include size drift — `internal/cli/check.go:357-362` (rate-limited summary lines + hint); `internal/serve/mcp_tools.go:2588-2614` (per-spec warnings with copy-paste actions); tests `TestCheckSizeDriftSummary_RateLimited`, `TestToolWarnings_SizeDriftLeafAndContainer` (`size_drift_test.go:12`, `mcp_size_drift_test.go:14`).
- [✓] AC#18 — ladder documented centrally — `domains/engineering/skills/spec-sizing/SKILL.md` is the central doc (259 lines: ladder, per-type bands, nudge schedule, tracker tuning, `size_ack` protocol). `core/skills/spec-format/SKILL.md` + `domains/engineering/skills/spec-format/SKILL.md` add the frontmatter rows + "size: a living field" subsection with cross-reference. Per-type bands also appear inline in `core/spec-types/{feature,bug,epic,initiative}.md` field descriptions. No standalone `README.md` was created — the spec said "or equivalent central doc," which is honored.

## Changes (Files to Touch)

- [✓] `core/spec-types/{feature,bug,epic,initiative}.md` — size + size_ack added with per-type band copy (slice 1).
- [✓] `internal/spec/spec.go` — Size/SizeAck struct fields, parseFrontmatter reads, validateSize enforces enum.
- [✓] `internal/cli/cost.go` — effortGiant constant, bucketFromPoints emits giant at 40+ points, declared/computed/drift columns, LeafDrift helper.
- [✓] `internal/cli/size.go` — new file, get/set/--check with tracker-capability header.
- [✓] `internal/cli/check.go` — rate-limited drift summary added at lines 357-362.
- [✓] `internal/snapshot/rollup.go` — RollupChildSizes, ContainerDrift, BuildParentMap.
- [✓] `internal/sizing/sizing.go` — new package with CollectDrift, EstimateSpec, BucketFromPoints, TrackerCapability.
- [✓] `internal/serve/mcp_tools.go` — hero_warnings emits per-spec size-drift entries at lines 2588-2614.
- [✓] `internal/config/config.go` — TrackerConfig.SizeMapping + Validate().
- [✓] `internal/tracker/tracker.go` — interface extended with SupportsHierarchy/MapSize/ReverseMapSize.
- [✓] `internal/tracker/size_mapping.go` — new file, 476 lines: per-adapter defaults, PlanSizePull/PlanSizePush, ExtractTrackerSize.
- [✓] `internal/tracker/{jira,linear,github}.go` — configuredSizeMapping field + delegating methods.
- [✓] `internal/cli/{pull,sync}.go` — wired into runPull/runSync paths.
- [✓] `domains/engineering/skills/spec-sizing/SKILL.md` — new 259-line skill, the central doc.
- [✓] `domains/engineering/skills/spec-format/SKILL.md` + `core/skills/spec-format/SKILL.md` — size/size_ack rows + living-field subsection.
- [✓] `domains/engineering/commands/design.md` — design-time `size:` stamping paragraph.
- [✓] `domains/engineering/agents/{feature,platform}-delivery-lead.md` — design-phase skill load + delivery-loop nudge step.

## Open items

- AC#16 — **PARTIAL** — engineer's stated reason: "existing `hero sync spec` path is status/comment-based, not field-level; per-tracker push paths get the write call in a follow-up. Non-destructive contract honored." Assessment: **concrete** — the `Tracker` interface genuinely lacks a field-write method (no `UpdateSize`, no `UpdateField`); the existing `UpdateStatus` is single-purpose and would need extending to push the size. The planner is wired, conflict-detection is tested, the contract isn't violated. Follow-up work to add a per-tracker `UpdateSize` (Jira PUT to `/issue/{id}`, GitHub PATCH labels, Linear mutation) is real but out of scope for slice 5 as scoped. **Slight gap in framing**: `CreateIssue` on Jira/GitHub/Linear already writes field-level payloads — size could plausibly have been added to the initial create payload as a one-line change (e.g. `payload["fields"]["customfield_xxx"] = points` in `jira.go:362`, or appending `size/<tier>` to the GitHub labels array in `github.go:58`). That would have honored AC#16 on the create path while leaving the update path for follow-up. Not blocking; downgrade-worthy framing only.

## Audit notes

- Build is clean (`go build ./...` no output).
- Targeted test run for touched packages (`internal/{spec,cli,sizing,snapshot,tracker,config,serve}`) — **all pass**.
- One stale `hero estimate` comment reference in `internal/sizing/sizing.go:4` survived the slice-5 rename (no functional impact; markdown drift test doesn't cover Go source). Cosmetic.
- Harness/canonical question (slice 4 lessons-learned): `.claude/skills`, `.claude/agents`, `.claude/commands` are all gitignored (`/.gitignore:62-64`). The canonical `domains/engineering/` files are correct and committed; the harness `.claude/skills/spec-sizing/SKILL.md` is a slice-4 carbon copy that did NOT get the slice-5 `hero estimate → hero sprint estimate` fix applied (still has 2 stale references at lines 23, 130). Because harness files are local-only views regenerated from canonicals by tooling, this is **not a delivery defect** — it just shows a fresh clone or re-install would pick up the correct content. Slices 2/3/5 only touched Go code + canonical skill paths; nothing else fell into the slice-4 trap.
- Container-drift rollup uses midpoint-sum-and-rebucket (`rollup.go:680-708`) per spec Implementation Notes ("10 smalls naturally roll up larger than 1 small") — the design choice is documented in the commit message and the spec was updated to reference it. Test `TestRollupChildSizes_UnknownChildIsIndeterminate` confirms the "missing both → unknown" safety rule.
- `hero_warnings` (`mcp_tools.go:2596-2614`) emits **per-spec** drift entries with `hero size <slug> <tier>` actions while `hero check` summarizes — this two-surface pattern matches the spec's Implementation Notes on rate-limiting drift output.
- Scope discipline good: diff is contained to spec docs, schema, validator, CLI, MCP surface, tracker mapping, and skill docs. No drive-by refactors. The only AC narrowing (drop `enhancement`) is documented inline in the spec and tied to a defensible reason (the type doesn't ship today).
