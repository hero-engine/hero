# Delivery audit — deepseek-harness-install-target

**Audited:** `git diff HEAD` over the supplied installer, CLI, tests, documentation and tripwire paths, plus the six new implementation/test/compatibility files read directly. Existing changes were distinguished using `/tmp/hero-deepseek-preexisting.patch`.
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 native layout and target selection — `target_deepseek.go`, `install.go`; `TestDeepSeekNativeLayoutAndInventory`, eight-target picker tests and snapshots.
- [✓] AC-2 native skill rendering and accurate role limits — `render.go`, `agents_md.go`; `TestDeepSeekRenderingYAML` and all three native compatibility logs validate loader acceptance and role invocation flags.
- [✓] AC-3 explicit portable MCP overlay and workspace binding — `mcp_deepseek.go`, CLI workspace preflight; `TestDeepSeekInstallJSONWorkspace`, `TestDeepSeekWorkspaceOverlayRefresh`, real MCP initialize/list/status and explicit-root execution.
- [✓] AC-4 global native paths — `DeepSeekHome`, global manifest handling; `TestDeepSeekGlobalOwnershipAndHome` and `TestDeepSeekGlobalRefreshPruneAndDryRun` verify home handling, isolation and ownership.
- [✓] AC-5 repeat install, pruning and preservation — `deepseek_state.go`, installer preflight; namespace/pruning, collision, global refresh/dry-run and workspace collision tests assert resulting filesystem behavior.
- [✓] AC-6 inventory and doctor — inventory counts canonical/workflow/role skills and names missing output. Doctor renders missing DeepSeek paths even with full numeric counts, marks the target incomplete, and labels activation unverified. `TestDoctorDeepSeekNamesMissingArtifactsDespiteFullCounts` asserts missing role and overlay paths, repair verdict and activation limits; targeted regression passes.
- [✓] AC-7 satellites — skills-only layout and parent-overlay guidance in `satellite.go`; `TestDeepSeekSatelliteLayout`, satellite coverage matrix, native Git-root discovery assertion.
- [✓] AC-8 owned uninstall and shared content — `uninstallDeepSeek`, ownership check in `removeHeroFiles`; ownership/dry-run, modified overlay, shared Codex instructions/state and workspace overlay removal tests.
- [✓] AC-9 coexistence and seven-target regression — mixed guidance aggregation, sibling target registry; both-order Codex/Claude/Grok tests plus native, contract, composition, routing, PM and QA matrices; cross-discovery docs and real precedence assertion.
- [✓] AC-10 real native compatibility — `scripts/deepseek-compatibility.mjs` imports the pinned native loaders/Cordis/MCP plugin, and engineering/PM/QA logs record 121/133/106 loaded skills and 86 MCP tools each, tool execution and zero model calls.

## Changes
- [✓] 1 target, native rendering and global manifest — new target/state files and renderer, dispatch and collision checks.
- [✓] 2 instructions and contracts — mixed-target guidance, skills contract, canonical engineering instructions, eighth-target tripwire and parity coverage.
- [✓] 3 MCP overlay — new YAML adapter, target registration, workspace error propagation and safe removal.
- [✓] 4 lifecycle integration — state, upgrade selection, satellites, uninstall, doctor missing-path reporting and exclusions implemented and covered by focused tests.
- [✓] 5 verification — focused installer/CLI tests, expanded target matrices and snapshots, opt-in real native runner.
- [✓] 6 documentation — all seven named docs cover native paths, role limits, activation, home resolution, cleanup, discovery and runtime selection. Both MCP setup docs accurately state that nonblank `DSH_HOME` retains its whitespace.

## Validation evidence
Read existing evidence only; no experiments were run by this auditor. Installer tests pass (11.140s), focused CLI tests pass (0.640s), full CLI retry passes (84.872s), and the clean full-suite retry log reports CLI passing in 92.557s with no FAIL lines. After the doctor correction, focused doctor/DeepSeek regressions pass (0.651s, `/tmp/hero-deepseek-doctor-tests.log`). Native engineering/PM/QA checks pass at `477b4f420553e8a52c2fbccc464d7561b239c443`. `make build`, `go vet ./...`, 28 docs tests and strict MkDocs build pass in the supplied evidence.

## Audit notes
No open delivery gaps remain. The ledger has all required rows and concrete end-to-end exercise evidence. Initial missing-path reporting and home-resolution wording findings were corrected and rechecked before this verdict. Existing unrelated doctor/upgrade/inventory and Aha changes were excluded from scope assessment using the captured baseline.
