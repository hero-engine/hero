# Delivery audit — hero-root-docs-remediation

**Audited:** `git diff HEAD -- README.md GETTING-STARTED.md MCP-SETUP.md CROSS-REPO-PEERING.md TEAM-SERVER.md internal/cli/docs_check_test.go internal/cli/markdown_drift_test.go internal/config/root_docs_examples_test.go .hero/planning/initiatives/hero-marketing/hero-root-docs-remediation/spec.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — root quickstarts execute against the current CLI — all documented invocations resolve against the Cobra tree, disposable workflows exercised the principal project/satellite/peering/Serve/headless paths, and `GETTING-STARTED.md:223-225` now gives `sync pull` the required spec path. The disposable pull reached the expected missing-integration gate, while `internal/cli/docs_check_test.go:56-68` rejects future slug-only root examples.
- [✓] AC-2 — one root corpus with thin satellites and no nested initialization — `README.md:141-168` and `GETTING-STARTED.md:162-185` prescribe the implemented flag-based satellite workflow, explicitly prohibit nested `hero init`, and match the supplied monorepo exercise without a nested `.hero` directory.
- [✓] AC-3 — `hero spec verify` is the normal evidence-backed close — `README.md:105-139` and `GETTING-STARTED.md:123-160` present one verify close and bound `spec complete` as non-normal; the supplied verify regressions cover the Ledger, audit, coverage, and build/test gates.
- [✓] AC-4 — marked root configuration examples load through the production decoder — `internal/config/root_docs_examples_test.go:10-39` extracts exactly two marked examples and calls `config.Load`; the examples are at `GETTING-STARTED.md:250-282` and `MCP-SETUP.md:92-111`.
- [✓] AC-5 — harness-native workflow terminology — `README.md:60-64` and `GETTING-STARTED.md:61-76` distinguish Claude command files, Codex/Grok command skills, and other targets' native surfaces without promising universal slash commands.
- [✓] AC-6 — every assigned root claim is resolved with reproducible evidence — the five guides reconcile the assigned product, onboarding, verification, configuration, inventory, MCP, peering, maturity, domain, repository, and licensing claims. `TEAM-SERVER.md:60-67` uses process-manager-injected `HERO_AUTH_TOKEN` without secret argv, and `TEAM-SERVER.md:89-110` accurately bounds direct runner and queue jobs as separate stores with no shipped approval pause/resume bridge. `internal/cli/docs_check_test.go:70-83` guards those boundaries.
- [✓] AC-7 — memory-first product overview with a distinct connected delivery system — `README.md:3-14` and `GETTING-STARTED.md:3-9` lead with durable project memory, describe delivery separately, and bound the reinforcing cross-tool outcome as preview.

## Changes

- [✓] Change 1 — replace obsolete satellite/list/add and repair guidance — root onboarding uses `hero install satellites`, `--yes`, `--repair`, and plan-only `--migrate-nested`, with scoped project repair and one root corpus.
- [✓] Change 2 — correct verification, Go, health, and root quickstarts — the guides use verify-only normal close, Go 1.26.4 for source builds, doctor/check for health, a spec path for `hero diff`, and a spec path for normal tracker pull.
- [✓] Change 3 — correct configuration, harness, peering, team, layout, install, and maturity claims — the rewrites use decoder-backed configuration, harness-native terminology, asynchronous receiver-owned peering, shipped local Serve, preview team/headless boundaries, non-argv auth, current repository layout, and exact repository/license separation.
- [✓] Change 4 — remove mutable narrative inventories — hand-maintained command/agent/skill/MCP totals and rosters are absent; `internal/cli/docs_check_test.go:43-54` rejects their reintroduction in the inventory-bearing guides.
- [✓] Change 5 — exercise commands/configuration and record evidence — the supplied evidence covers clean single-project and monorepo journeys, peer Mail/handoff/receive, local Serve, preview workers, headless dry-run, exact tracker-path parsing, production config decoding, links, JSON/TOML, invocation drift, stale-claim assertions, full Go tests, Hero lint/score/index, and diff whitespace.

## Open items

- None.

## Audit notes

- None.
