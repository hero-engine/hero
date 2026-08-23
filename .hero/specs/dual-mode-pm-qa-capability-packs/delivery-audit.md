# Delivery audit — dual-mode-pm-qa-capability-packs

**Audited:** final remediated uncommitted worktree at `96d834bc` (`git diff HEAD` plus untracked implementation files)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — Canonical composition config — `internal/domains/composition.go` and `internal/config/config.go` validate one primary plus ordered, deduplicated, role-compatible extensions; resolver/config tests cover valid and invalid shapes.
- [✓] AC-2 — Legacy compatibility — absent and scalar legacy configuration resolve in memory to equivalent primary-only compositions without network access or implicit rewrites; compatibility tests remain green.
- [✓] AC-3 — Dual-mode packages — installed and deferred primary/extension projections resolve from the same embedded `DomainFS` source (`composition_content.go:228-247,338-351,457-496`).
- [✓] AC-4 — Engineering essentials — the primary-only byte-compatibility test preserves the legacy Core + Engineering surface.
- [✓] AC-5 — Bounded roster — declared extension agents/commands are installed, deeper content is recorded in `DeferredEntries`, and `ResolveDeferredContent` plus `hero domain content` load enabled-pack bytes locally without adding them to the installed roster (`composition_content.go:251-281,357-365,441-517`; `internal/cli/domain.go:91-121`; deferred-content unit/CLI tests).
- [✓] AC-6 — Collision safety — composition independently claims stable IDs and rendered paths, returns actionable owners/IDs/paths on conflict, permits shared commands only through typed declarations, and resolves before lifecycle writes (`composition_content.go:109-125,251-324`; synthetic identity/path collision tests; lifecycle rollback tests).
- [✓] AC-7 — Shared command routers — typed descriptors carry artifact, lifecycle, intent, target, priority, role, and owner metadata; executable selection compares specificity/priority and returns a typed ambiguity error; all rendered harness routers carry the same contract.
- [✓] AC-8 — Spec-type amendments — Core declares the lifecycle extension point; QA's namespaced amendment is validated, applied to canonical feature lifecycle/status values, and exported without installing a shadow type (`core/spec-types/feature.md`; `internal/spectypes/loader.go`; applied/rejection/export tests).
- [✓] AC-9 — Artifact ownership — `hero graph node add --handler-owner` validates through `graph.DomainForHandler` at the mutation boundary; integration coverage creates a QA TestPlan, changes primary context, preserves stored `qa` provenance, and rejects disabled owners (`internal/cli/graph_node.go:97-149`; `internal/cli/graph_node_test.go:13-72`).
- [✓] AC-10 — Enabled-stack retrieval — domain provenance is carried through specs, graph projection, node index, FTS, and retrieval results. Scoped index/retrieval variants inject allowed-domain predicates and focus ordering before `ORDER BY`/`LIMIT` (`internal/index/index.go:822-1144`; `internal/retrieval/retrieval.go:335-742`). Search, list, and blocked use the composition scope; regression tests with 55 stronger disabled-domain hits and 25 newer disabled file hits prove enabled/explicit QA results cannot be starved (`internal/cli/domain_retrieval_test.go:106-138`; `internal/index/index_test.go:242-270`).
- [✓] AC-11 — Safe disable — install tests prove Hero-owned QA files are pruned while project-owned files and historical artifacts survive; default reads exclude disabled QA while explicit/all-domain reads retrieve it.
- [✓] AC-12 — Known collision reconciliation — shared commands require typed per-owner descriptors, canonical lifecycle additions use amendments, command-to-agent closure and collision inventory tests reject unexplained overlaps, and no `lite` content tree exists.
- [✓] AC-13 — Harness parity — the all-harness test iterates every advertised extension agent, command target, and resolved shared router across OpenCode, Cursor, Claude, Copilot, Codex, Generic, and Grok, including native command-skill placement.
- [✓] AC-14 — Local deterministic operation — composition, routing metadata, and deferred loading use embedded local content only; supplied offline/fresh-binary smoke is green.
- [✓] AC-15 — Standalone initialization — CLI tests and fresh-binary exercises verify bundled PM→Codex and QA→Claude primary workspaces without registry/network dependencies.
- [✓] AC-16 — Extension initialization — canonical `--with` parsing validates and persists repeatable/comma-separated ordered extensions without manual JSON edits.
- [✓] AC-17 — Init-and-install — multi-target initialization materializes every requested harness and preserves exact no-target guidance/compatibility.
- [✓] AC-18 — Atomic lifecycle commands — enable/disable/switch resolve before writes, reinstall all recorded/detected targets, persist config last, and restore prior rendered/config state on failure.
- [✓] AC-19 — Inspectable setup — domain show/list expose primary, extensions, roles, bundled readiness, and validation state; `hero domain content` exposes locally available deferred entries.

## Changes

- [✓] Add canonical composition and pack-role types — implemented and covered by resolver/role tests.
- [✓] Migrate configuration through a compatibility resolver — implemented with canonical writes and legacy read compatibility.
- [✓] Compose local pack content and an inspectable manifest — installed/deferred projections, stable ID/path validation, typed routers, and local deferred resolution are present.
- [✓] Compose spec-type registries and amendments — owner extension points are declared, validated, applied, and exported without shadows.
- [✓] Separate enabled retrieval, focus ranking, and routed ownership — graph scope/stamp contracts drive search, list, blocked, command selection, and handler-owned writes; filtering/ranking occurs before result caps.
- [✓] Add composed init, install, and lifecycle commands — standalone packs, extensions, repeated targets, inspection, reinstall, pruning, and rollback are covered.
- [✓] Route remaining primary-domain consumers through canonical config — check/docs/scan/upgrade/cache/dispatch and graph-writing consumers use `ResolveDomains`/`PrimaryDomain`.
- [✓] Add composition, compatibility, collision, graph, CLI, retrieval, and harness validation — tests cover all workspace modes, deferred local loading, identity/path conflicts, ambiguity, amendments, ownership, cap-safe retrieval, rollback/pruning, init UX, and seven-harness parity.

## Audit notes

- The former AC-10 cap-starvation defect is closed: plain search, filtered FTS, list, file search, node-index retrieval, and graph retrieval apply scope before fixed limits; direct CLI routes use the scoped variants.
- Fresh focused validation passed: `go test ./internal/cli ./internal/index ./internal/retrieval -run 'Test(SearchDomainScopeAppliesBeforeResultCap|SearchByFileDomainsFiltersBeforeLimit|SearchCompositionDefaultsAndFocusRanking|ListCompositionDefaultsFocusAndHistoricalOverrides|RetrieveCarriesProjectedGraphDomain)' -count=1`.
- Supplied evidence reports `go test ./... -count=1`, fresh build `/var/folders/vw/2hdr6jx55cd0bgr4sv9_ygzw0000gn/T/tmp.vBX2hL7Z5P/hero`, and PM/QA/composed smoke all green.
- All 19 acceptance criteria and all eight Changes rows have concrete implementation and behavioral evidence. Unrelated marketing, hero-sales, mock, and `CLAUDE.md` changes were excluded.
