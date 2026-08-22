# Delivery audit — qa-public-pack

**Audited:** scoped worktree diff and untracked files for `domains/qa`, QA embedding/tests, `qa-public-pack`, and the `hero-qa` boundary amendment  
**Verdict:** SHIP  
**Surface:** clean  
**Confidence:** high

## Acceptance criteria

- [✓] Expose embedded offline `qa` through `DomainFS` and `AvailableDomains` — `content.go:42-43,75-89,145-165,282-287`; focused content tests pass uncached and the fresh-binary QA initialization/install smoke succeeded.
- [✓] Ship substantive routing, mission, agents, skills, commands, and spec types without placeholders — direct review found operational content in `domains/qa/AGENTS.md`, `mission.md`, 23 agents, 31 skills, 18 commands, and five spec types; exact inventory, minimum-content, placeholder, and frontmatter checks pass.
- [✓] Cover planning, authoring, charters, issue/flake triage, regression, gates, seams, and scrubbing — the corresponding routers under `domains/qa/commands/` contain concrete inputs, decisions, evidence expectations, and safety boundaries.
- [✓] Span coordination, strategy, authoring, investigation, curation, review, readiness, handoff, and hygiene — the exact 23-agent roster at `qa_content_test.go:12-20` and the reviewed agent bodies cover every named responsibility.
- [✓] Declare canonical QA artifacts with lifecycle and evidence expectations — all five records under `domains/qa/spec-types/` parse through the real registry; `TestLoad_QAIncludesOwnedTypesWithoutShadowingCore` confirms QA ownership and Core preservation.
- [✓] Primary QA includes Core plus QA and excludes engineering-only content — `internal/install/qa_overlay_install_test.go:9-80` passes for all targets; the independent Codex smoke produced 27 Core+QA agents and no `feature-delivery-lead`.
- [✓] Render native agents, skills, and commands for all seven harnesses — `TestQAOverlay_AllTargetsRenderNativeCoreAndQASurfaces` passes uncached for Claude, Codex, Cursor, Copilot, OpenCode, Grok, and Generic.
- [✓] Fail whenever a QA agent, skill, command, or spec-type reference names a missing artifact — `validateAllArtifactReferences` at `content_test.go:687-758` walks every non-README QA Markdown file, resolves every capability-shaped backtick against combined QA/Core agents, commands, skills, and spec types, and validates `ref(...)` targets in spec-type frontmatter. It covers the previously missed freeform agent-to-agent and secondary command references as well as `ref(...)`/`list[ref(...)]`. `TestQAReferenceValidationRejectsFreeformAndSpecTypeTargets` at `qa_content_test.go:84-98` proves both failure classes identify the referring file and missing target; it passes uncached.
- [✓] Detect removed advertised capabilities and reintroduced placeholders — exact inventories at `qa_content_test.go:12-44,78-81` catch removals/additions, while `validateRequiredPackFiles` catches empty/undersized files and conventional TODO/TBD/stub/placeholder markers.
- [✓] Avoid proprietary clients, hosted state, and connector requirements — scoped source changes add no proprietary code or dependency; reviewed integration guidance explicitly keeps local workflows available when connector data is absent or unavailable.

## Changes

- [✓] Add the public QA body under `domains/qa/` — routing body, mission, 23 agents, 31 skills, 18 commands, and five spec types are present and substantive.
- [✓] Embed and advertise QA in `content.go` — QA content, domain resolution, spec-type resolution, and advertised-domain registration are wired.
- [✓] Add integrity, reference, registry, and seven-harness tests — `qa_content_test.go`, `internal/install/qa_overlay_install_test.go`, `internal/spectypes/loader_test.go`, QA frontmatter cases, and content-wide positive/negative reference validation exist and pass.
- [✓] Record the public/private boundary in `hero-qa` — `.hero/planning/features/hero-qa/spec.md:49-58` assigns offline practitioner content to this pack and leaves proprietary views, hosted history, and connector implementations open.

## Exercise and excellence claims

- [✓] Exercise-the-feature — independently built a fresh binary, initialized an isolated Git workspace with `hero init --domain qa`, installed project-scoped Codex content, observed 27 agent files and 79 command/skill files, found QA routing and artifacts, and confirmed the engineering delivery lead was absent.
- [✓] Excellence Bar — the pack is substantive, locally useful, registry-valid, native across all supported harnesses, bounded from proprietary surfaces, and protected by exact inventory plus content-wide reference validation.

## Tests and drift

- Fresh uncached focused QA content, inventory, content-wide positive/negative reference, spec-type presence, and QA frontmatter tests passed.
- Fresh uncached `TestLoad_QAIncludesOwnedTypesWithoutShadowingCore` passed.
- Fresh uncached seven-target `TestQAOverlay_AllTargetsRenderNativeCoreAndQASurfaces` passed.
- The implementer supplied a fresh passing `go test ./... -count=1`; the cold re-audit did not redundantly rerun the broad suite.
- `hero drift qa-public-pack` reports 10/10 criteria linked and no spec/code drift. It also emits the existing workspace/render-version warning (`v0.29.0-22-g430d67b-dirty` workspace, `v0.32.0-2-ge09f3ee-dirty` binary); this did not affect focused validation or the fresh-binary smoke.

## Open items

None.

## Audit notes

- All ten ledger rows and all four Changes rows have concrete evidence; no DONE row was downgraded.
- The scoped implementation respects the public/private boundary. Unrelated worktree changes were not audited or modified.
