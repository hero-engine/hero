---
title: "Hero Root Documentation Truth Repair"
slug: hero-root-docs-remediation
type: feature
status: completed
domain: engineering
size: medium
priority: high
horizon: now
created: 2026-08-04
tags: [documentation, onboarding, configuration, truth]
parent: hero-marketing
depends-on: [hero-public-truth-baseline, hero-positioning]
relations:
  - target: hero-public-docs-drift-guard
    kind: conflicts-with
  - target: hero-public-repo-readiness
    kind: conflicts-with
delivery_method: manual
completed_at: 2026-08-23T17:50:29Z
---

# Hero Root Documentation Truth Repair

## Goal

Make repository-level onboarding and reference documentation executable, current, and consistent with the memory-first, two-system positioning and shipped-surface inventory.

## Kickoff

Repairs the highest-risk public instructions before broader hosted-doc and landing publication.

**Status:** delivering — root-guide remediation and validation are complete;
the mandatory independent cold delivery audit is next.

**Pick up at:** cold-audit the on-disk spec, root-guide diff, Completion Ledger,
and validation evidence. Do not add a license, deploy, or mutate visibility.

→ `.hero/planning/initiatives/hero-marketing/hero-root-docs-remediation/spec.md`

**Files:** `README.md`, `GETTING-STARTED.md`, `MCP-SETUP.md`, `CROSS-REPO-PEERING.md`, `TEAM-SERVER.md`, root metadata and generated public guidance
**Skip:** `web/docs/src/` and landing publication.

## Changes

1. Replace nonexistent satellite/list/add and invalid repair examples with the canonical one-root satellite workflow; explicitly prevent nested `hero init` guidance.
2. Make `hero spec verify` the normal closing path, correct the Go prerequisite, remove the dead `hero verify-install` command, and align all root quickstarts with current help.
3. Correct configuration, harness-native workflow, peering, team, repository-tree, install, and capability-maturity statements from the baseline.
   - Explain Hero's durable project-memory system before the spec-and-agent delivery system, then show how verified delivery compounds the memory available to later sessions.
   - Keep project memory and delivery workflow visually and conceptually distinct enough that Hero does not read like another spec kit.
   - Describe this `hero` repository as the only Apache-2.0 candidate, Sprout as separately MIT-licensed, and `hero-code`/`hero-cloud` as proprietary.
4. Remove hand-maintained inventory counts from narrative pages or source them from generated reference data.
5. Exercise changed commands and configurations in disposable workspaces and record evidence.

## Acceptance Criteria

- **AC-1:** WHEN a user follows any root quickstart THE SYSTEM SHALL complete it against the current CLI without undocumented positional behavior.
- **AC-2:** THE SYSTEM SHALL document one root `.hero` corpus with thin satellite harness trees and SHALL NOT prescribe nested project initialization.
- **AC-3:** THE SYSTEM SHALL present `hero spec verify` as the normal evidence-backed closing gate and accurately bound any manual completion path.
- **AC-4:** WHEN root documentation includes configuration THE SYSTEM SHALL load its examples through the production decoder.
- **AC-5:** WHEN Hero workflow surfaces differ by harness THE SYSTEM SHALL use harness-native terminology rather than claiming universal slash commands.
- **AC-6:** THE SYSTEM SHALL resolve every root-doc claim assigned to this child in the claim registry with reproducible evidence.
- **AC-7:** WHEN a reader encounters the root product overview THE SYSTEM SHALL identify durable project memory as Hero's primary distinction and the spec-and-agent loop as its connected execution system.

## Boundaries

- No hosted-doc page edits, landing design, release campaign, or product compatibility shim for stale docs.
- No mutable counts in narrative copy unless they are generated from the canonical registry.
- No `LICENSE`, visibility, or release mutation and no open-source claim for `hero-code` or `hero-cloud`.

## Validation

- Execute root quickstarts in clean temporary workspaces and decode every complete JSON example.
- Run documentation invocation checks, link checks, `git diff --check`, Hero lint/score, and index refresh.

## Completion Ledger

Rewrote the five repository-root guides around durable project memory first and
verified delivery second. The guides now use current CLI contracts, explain
harness-native workflow rendering, preserve one root corpus in monorepos, and
bound optional integrations and preview execution paths. The documentation,
implementation, Go, testing, reliability, and Completion Ledger guidance was
applied.

Validation performed: full Go test suite; production-decoder tests for every
marked root `hero.json` example; Cobra-backed invocation drift across all five
root guides; link and JSON/TOML checks; mutable-inventory and P0/P1 claim
assertions; satellite and verification regression tests; disposable
single-project, monorepo, two-peer, local Serve, preview team-worker, and
headless dry-run exercises; exact tracker-pull path parsing; regressions against
slug-only pull, secret-bearing auth argv, and an unshipped approval bridge; Hero
lint/score/index; and diff whitespace review.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Root quickstarts execute against the current CLI | DONE | Disposable workspaces exercised init/install/status/scan/domain setup, a valid `spec deliver --manual` transition, inspection gates, one-root satellites, two-peer Mail/handoff/promotion, local Serve health, preview team workers, and headless dry-run. `hero sync pull .hero/planning/features/csv-export/spec.md` resolved the exact spec path and reached the expected missing-integration gate. Exercises caught and corrected delivery mode, Kickoff, diff-path, and sync-pull-path semantics. |
| 2 | One root corpus with thin satellites; no nested init | DONE | `README.md` and `GETTING-STARTED.md` use `hero install satellites`, `--yes`, `--repair`, and `--migrate-nested`; explicitly prohibit nested `hero init`. Materialization/legacy detection tests and the disposable monorepo exercise passed without creating `services/auth/.hero`. |
| 3 | `hero spec verify` is the normal close | DONE | README and Getting Started describe the Completion Ledger, independent cold audit, build/test gate, and one `hero spec verify <slug>` close. `hero spec complete` is bounded as a non-normal administrative path. Verify regression tests passed. |
| 4 | Root configuration examples load through production decoder | DONE | `internal/config/root_docs_examples_test.go` extracts every `<!-- hero-config -->` block and loads it through `config.Load`; both root examples passed alongside the canonical public fixture. All other root JSON/TOML examples parse. |
| 5 | Harness-native terminology | DONE | README and Getting Started distinguish Claude command files, Codex/Grok command skills, and other target-native surfaces; they direct readers to natural language instead of promising universal slash commands. |
| 6 | Resolve every assigned root-doc registry claim | DONE | The reconciliation below covers product framing, install/topology, verification, config, prerequisites, inventory, MCP, layout, peering, maturity, domains, and repository/license boundaries. Reproducible claim assertions, Cobra invocation drift, spec-path enforcement, secret-argv rejection, and split-store approval-boundary checks passed. |
| 7 | Lead the product overview with durable project memory | DONE | README and Getting Started open with memory, explain verified delivery separately, and label the reinforcing cross-tool outcome preview until public continuity proof exists. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Replace obsolete satellite and repair examples | DONE | All root guides use the flag-based one-root satellite workflow and scoped project/satellite repair commands. |
| 2 | Correct verification, Go, health, and quickstart commands | DONE | Normal close is verify-only; source builds require Go 1.26.4 while binaries require no Go; `hero doctor`/`hero check` replace the dead install-health command; delivery mode, diff-path, and tracker sync-pull path behavior are explicit. |
| 3 | Correct config, harness, peering, team, layout, install, and maturity claims | DONE | Rewrites cover supported targets, decoder-backed config, asynchronous one-graph-per-project peering, shipped local Serve, preview team/headless execution, current repository layout, and bounded optional integrations. Team auth uses process-manager-injected `HERO_AUTH_TOKEN` without secret argv; direct runner and queue-worker stores are explicitly separate, with no claimed approval pause/resume bridge. |
| 4 | Remove mutable narrative inventories | DONE | Hand-maintained command/agent/skill/MCP counts and name rosters were removed. Exact install inventory routes to `hero doctor`; exact filtered MCP inventory routes to runtime `tools/list`. A regression test rejects new narrative counts. |
| 5 | Exercise commands/configuration and record evidence | DONE | Clean-workspace exercises and targeted/full automated checks are recorded above and in the exercise section, including exact tracker path parsing to the expected missing-integration boundary and regressions for the three cold-audit HOLD conditions. |

### Root claim reconciliation

| Claim IDs | Resolution |
|---|---|
| `product-memory-system`, `product-delivery-system`, `product-two-system-loop` | Memory leads; delivery is separate and connected; the cross-tool outcome remains preview. |
| `install-satellite-subcommands`, `install-bare-repair`, `satellite-workspace-architecture` | Replaced nonexistent positional forms and unscoped repair; documented one root corpus and thin satellites. |
| `verify-closes-delivery`, `cold-audit-and-verify` | One verify call is the evidence-backed close after Ledger, audit, coverage, and build/test gates. |
| `go-prerequisite`, `verify-install-command` | Corrected source-build Go version and replaced the dead command by intent with doctor/check. |
| `public-config-example` | Root examples are marker-extracted and decoder-tested. |
| `installed-command-count`, `installed-agent-count`, `installed-skill-count`, `runtime-mcp-tool-count` | Removed narrative totals and rosters; directed exact inventory to runtime/generated authorities. |
| `harness-native-workflows`, `supported-install-targets` | Documented current targets and their native surface differences without universal slash claims. |
| `repository-layout-cloud`, `apache-license-status`, `proprietary-repository-boundary`, `sprout-license-boundary` | Removed the nonexistent tree; stated the future grant is not active; kept Hero Code/Cloud proprietary and Sprout separate MIT. |
| `cross-repo-peering` | Documented one graph per project, manifests, asynchronous Mail, and explicit receiver promotion. |
| `headless-runtime`, `hero-serve-intelligence` | Local Serve is shipped; team/headless paths are preview and require auth/network/provider/execution validation. Direct runner jobs and queue-worker jobs use separate stores; no end-to-end approval pause/resume path is claimed. |
| `engineering-default-pack`, `optional-domain-packs` | Engineering is the default with lightweight PM/QA help; focused PM/QA/Sales setups are optional and maturity-bounded. |

### Exercise-the-feature check

- [x] A built current `hero` binary completed the single-project quickstart in a disposable Git workspace, including Codex installation, status, scan, domain inspection/composition, a valid manual delivery transition, and headless dry-run.
- [x] The documented tracker refresh used the exact spec path, passed path resolution, and stopped at the expected missing tracker-integration boundary; a regression rejects slug-only root-doc `sync pull` examples.
- [x] A disposable Go monorepo initialized Hero only at root, materialized and repaired a detected satellite, printed the nested-migration plan, and retained no nested `.hero` directory.
- [x] Two disposable Hero repositories registered reciprocal peers, generated manifests, sent advisory Mail and a work transfer, listed the receiver inbox, inspected the untrusted message, and promoted it through `hero handoff receive`.
- [x] Local Serve and preview team mode started on isolated ports, returned health/status, ran workers, exposed documented administration/job surfaces, and shut down cleanly. Regression checks reject secret-bearing auth argv and the unshipped approval bridge while requiring process-manager `HERO_AUTH_TOKEN` guidance.
- [x] Complete root Hero configuration examples loaded through the production decoder; root links, JSON/TOML, Cobra invocations, and P0/P1 assertions passed.

### Excellence Bar self-check

Yes — the root guides now start with the actual differentiated product, teach a
copyable path that current commands accept, put prerequisites and action
boundaries beside optional/preview capabilities, and fail closed on mutable
inventories, license state, and public maturity claims.
