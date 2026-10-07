---
title: "Read contract conformance — golden schemas, peer fixture, docs"
slug: read-contract-conformance
type: feature
status: completed
priority: low
domain: engineering
size: small
created: 2026-10-06
parent: hero-read-contract
depends-on: [hero-spec-handoff-tools, hero-work-tool, polish-and-suggested]
completed_at: 2026-10-07T00:01:00Z
---

# Read contract conformance

## Goal

Lock v1 so clients can depend on it. Give clients real fixture data and docs.

## Kickoff

Three pieces:
- a golden additive-only schema test over all three tools;
- `scripts/export-read-contract-fixture.py` to dump real replies for client fixture servers;
- docs in `web/docs/src/cli/server-and-mcp.md`.

Then reply to hero-harness with the Hero commit to pin. Test with `go test ./internal/serve -run ReadContractV1`.

## Design

- `internal/serve/read_contract_schema_test.go` flattens each tool's reply into `path:type` entries.
  - Every line in `testdata/read_contract_v1.golden` must still be produced, so a removed, renamed or retyped field fails.
  - New paths are allowed and are added with `-update-read-contract`.
  - The fixture workspace exercises every shape: nulls, polish, suggested with Explore, relations, acs and tracker.
- The export script runs a real `hero mcp`. It writes `hero_work.json`, `hero_handoff.json`, `hero_spec/<slug>.json` per item, and `manifest.json` (version, revision, source commit).

## Acceptance Criteria

- **AC-1:** IF a change removes or retypes any field recorded in the v1 golden schema of `hero_work`, `hero_spec` or `hero_handoff` THEN THE SYSTEM SHALL fail the test suite, while new fields SHALL be allowed.
- **AC-2:** WHEN `scripts/export-read-contract-fixture.py OUT_DIR` runs against a Hero project THE SYSTEM SHALL write the three tools' replies (one `hero_spec` file per `hero_work` item) and a manifest, using a real `hero mcp`.
- **AC-3:** THE SYSTEM SHALL document the three tools, the lane, verify and next-step vocabulary, revision and watch-glob semantics, the additive-only policy, and the export script.

## Changes

1. `internal/serve/read_contract_schema_test.go` and `testdata/read_contract_v1.golden`.
2. `scripts/export-read-contract-fixture.py`.
3. `web/docs/src/cli/server-and-mcp.md`.
4. A slug-collision fix found by the export: `workmodel.NewCorpus` ranks work specs above knowledge entries above intakes.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: additive-only golden | DONE | Round 1 HOLD fixed: the fixture now covers an audited, sized, verified item (`verify.audit:string`, `size:string`, `next:null`), a decision child (`verify:null`), a `blocks` edge and initiative children (element shapes), an empty handoff (`updated_at:null`), and a separate thin-backlog call for Explore (`suggested[].slug:null`). The golden grew 151 → 182 lines, purely additive (31 added, 0 removed). While enriching, the guard itself caught the vanished `slug:null`. Falsified: renaming `lane` fails with exactly `hero_work.items[].lane:string` and `hero_spec.item.lane:string` |
| 2 | AC-2: fixture export | DONE | Ran on this repo: 513 `hero_spec` files plus work, handoff and manifest (`fixture-manifest.json`, `real-exercise.log`). The first run found an explainer sharing a feature's slug; fixed in `NewCorpus`, with a regression case in `TestRound2AuditCases` |
| 3 | AC-3: docs | DONE | New "Read contract v1" section. Round 1: the wording now says the tools never write under `watch_globs` (runtime index/graph files may refresh). Docs unit tests and `mkdocs build --strict` pass (`docs-build.log`) |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | golden schema test | DONE | — |
| 2 | export script | DONE | Round 1: the manifest key is now `project_commit` (the exported project's HEAD), not a Hero source commit |
| 3 | docs | DONE | — |
| 4 | slug ranking fix | DONE | `workmodel/model.go` `slugRank` |

### Exercise-the-feature check

- [x] Exported the real fixture from this repo through `hero mcp` (all 513 items).

### Excellence Bar self-check

- [x] Yes. The contract is machine-guarded, and the export doubles as a whole-corpus smoke test (it found a real bug).
