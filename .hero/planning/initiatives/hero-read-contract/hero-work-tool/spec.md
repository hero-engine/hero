---
title: "hero_work MCP tool — the work picture with lanes and next steps"
slug: hero-work-tool
type: feature
status: delivering
priority: high
domain: engineering
size: small
created: 2026-10-06
parent: hero-read-contract
depends-on: [next-step-engine]
relations:
  - target: hero-spec-handoff-tools
    kind: conflicts-with
---

# hero_work

## Goal

The main read of the contract: every work item with lane, verify state and next step, plus a content revision and the paths whose change means "read again".

## Kickoff

Add read-only MCP tool `hero_work {recent_days?: integer = 14}` → `HeroWork {schema_version: 1, revision, generated_at, hero_version, watch_globs, items, polish: [], suggested: []}`. Items come from `workmodel.Build` plus `ApplyNext`. `polish` and `suggested` are filled by `polish-and-suggested`. Test with `go test ./internal/serve -run ToolWork`.

## Design

- `toolWork` in `internal/serve/mcp_tools_read_contract.go`, registered `safetyRead`.
- `recent_days` must be a positive integer; it is validated.
- `WatchGlobs` is exported: `.hero/planning/**`, `.hero/specs/**`, `.hero/knowledge/**`, `.hero/NEXT.md`, `.hero/next/**`, `.hero/hero.json`.
- `revision` is 16 hex characters of SHA-256 over the schema version, hero version, each `slug=item revision`, and polish/suggested. It excludes `generated_at`, so it moves only with content.
- `items` is never null.

## Acceptance Criteria

- **AC-1:** WHEN `hero_work` is called THE SYSTEM SHALL return `HeroWork` with `schema_version: 1`, `revision`, `generated_at`, `hero_version`, `watch_globs`, every work item with its next step, and `polish` / `suggested` as lists.
- **AC-2:** THE SYSTEM SHALL keep `revision` identical across calls whose content is unchanged (regardless of `generated_at`), and change it when any item changes.
- **AC-3:** IF `recent_days` is not a positive integer THEN THE SYSTEM SHALL return an error naming it.
- **AC-4:** THE SYSTEM SHALL register `hero_work` read-only, and a call SHALL write nothing under its watch globs.
- **AC-5:** WHEN `hero_work` runs on this repository (~513 specs) THE SYSTEM SHALL answer well under the contract's 2 s p95 budget, with an identical revision across repeated calls.

## Changes

1. `internal/serve/mcp_tools_read_contract.go`: `HeroWork`, `PolishItem`, `SuggestedItem`, `WatchGlobs`, `toolWork`, `workRevision`.
2. `mcp_dispatch.go` and `mcp_tools_def.go` registration; tool count 67→68.
3. Tests in `mcp_tools_read_contract_test.go`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: HeroWork shape | DONE | `TestToolWorkShape`: header, watch_globs, lanes, next command, knowledge excluded, `polish: []`, `suggested: []` |
| 2 | AC-2: revision stable vs content | DONE | `TestToolWorkRevision`: the same revision with `generated_at` an hour later; it changes after a spec edit |
| 3 | AC-3: recent_days validation | DONE | `TestToolWorkRecentDays`: 0, -1, 1.5 and "7" are rejected; 30 is accepted |
| 4 | AC-4: read-only, no writes | DONE | `TestToolWorkWritesNothing` |
| 5 | AC-5: real-repo latency and stability | DONE | `real-exercise.log`: 10 calls in one `hero mcp` session take first 243 ms, median 276 ms and max 306 ms, with one revision. That is well under the 2 s budget, and matches the auditor's independent 0.24–0.40 s |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | toolWork | DONE | — |
| 2 | registration | DONE | `safetyRead` → `readOnlyHint` |
| 3 | tests | DONE | 4 tests |

### Exercise-the-feature check

- [x] Real `hero mcp` over stdio, called 5 times on this repo (`real-exercise.log`).

### Excellence Bar self-check

- [x] Yes. A thin tool over the shared model, with a content-only revision.
