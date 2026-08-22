# Delivery audit — pm-public-pack

**Audited:** corrected working-tree delivery across `domains/pm/`, `content.go`, content/install/registry tests, and the two scoped specs

**Verdict:** SHIP
**Surface:** clean

## Findings

No meaningful findings.

## Acceptance criteria

- [✓] `pm` is embedded and advertised offline — `content.go` embeds the complete PM pack including `mission.md`; availability and `DomainFS` tests pass.
- [✓] PM content is substantive with no empty or placeholder capabilities — complete roster validation, parsed agent/skill metadata, semantic placeholder checks, and the long-placeholder negative fixture pass.
- [✓] Canonical feature/epic authoring plus PM-owned roadmap item — PM authoring roles target Core records; the runtime registry loads the schema-valid `roadmap-item` as PM-owned.
- [✓] Core-owned identities remain canonical without shadows — the focused registry test proves `feature`, `epic`, `intake`, and `prd` remain Core-owned while PM owns only `roadmap-item`.
- [✓] Primary PM install resolves Core + PM without engineering-only content — overlay/install coverage and runtime PM registry loading pass.
- [✓] Seven harness-native PM installs — Claude, Codex, Cursor, Copilot, OpenCode, Grok, and Generic each assert a PM agent, PM skill, and native PM command output.
- [✓] Missing PM references fail validation with the missing artifact identified — agent load declarations, command agent/skill routing, skill Cross-references, and spec-type `default_agents` are validated; negative fixtures name each missing source and target.
- [✓] Inventory detects removed advertised capabilities or reintroduced placeholders — all 30 agents, 19 commands, 51 skills, and PM-owned `roadmap-item` are enumerated; size and semantic TODO/TBD/stub/placeholder/coming-soon checks are enforced.
- [✓] No proprietary dependency added — scoped changes remain embedded public Markdown, local Go tests, and spec documentation.

## Changes

- [✓] Complete `domains/pm/` declarations — invalid domain shadows were removed, canonical Core ownership is documented, and loadable PM-owned `roadmap-item` remains.
- [✓] Preserve `content.go` public availability — PM remains advertised and its mission is embedded.
- [✓] Add content/install coverage — complete inventory, semantic placeholder validation, cross-artifact reference validation, runtime PM registry loading, and seven-target native outputs are covered.
- [✓] Record the separated public slice in `hero-pm` — private UI scope remains open and the public-pack boundary is recorded.

## Open items

- None. All Completion Ledger rows are supported by current implementation and tests.

## Audit notes

- Passed independently: focused PM content inventory, metadata, cross-reference, missing-reference, and semantic-placeholder tests with `-count=1`.
- Passed independently: `go test ./internal/spectypes -run TestLoad_PMIncludesOwnedTypesWithoutShadowingCore -count=1`.
- Passed independently: `go test ./internal/install -run TestOverlay_AllTargetsIncludeCoreAndDomain -count=1`.
- Independent inventory comparison confirms 30 agents, 19 commands, and 51 skills.
- Supplied evidence reports fresh `go test ./... -count=1` passing; focused results are consistent with that evidence.
