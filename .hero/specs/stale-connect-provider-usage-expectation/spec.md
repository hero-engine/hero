---
title: "Connect non-terminal test omits Aha from expected usage"
slug: stale-connect-provider-usage-expectation
type: bug
status: completed
domain: engineering
created: 2026-09-24
priority: low
severity: low
root_cause_class: code
size: tiny
delivery_method: manual
completed_at: 2026-09-25T00:33:24Z
---

# Connect non-terminal test omits Aha from expected usage

## Goal

Restore the non-terminal connect regression test by updating its stale provider expectation.

## Kickoff

Fix the stale connect usage expectation left behind when Aha was added.

**Status:** completed — focused race tests, full Go suite, build, independent audit, and Hero verification passed.

**Pick up at:** no remaining work for this fix.

Delivery closed via `hero spec verify stale-connect-provider-usage-expectation --skip-tests`; tests had already passed in this turn.

**Files:** `internal/cli/prompt_setup_commands_test.go:132`, `internal/cli/connect.go:103`

## Summary

### Categorization

| Attribute | Assessment |
|---|---|
| Criticality | Low — false test failure; no demonstrated runtime defect |
| Ease of Fix | Easy — one expected-string change |
| Caused by our codebase? | Yes — outdated test fixture |
| Needs more research? | No — reproduced mismatch agrees with source |

### Background

The copied checkout's Go suite fails its non-terminal connect test. The test protects prompt suppression and refusal under piped or closed stdin.

### Analysis

Runtime usage includes `aha`; the expected substring retains the older provider list. Because the entire parenthesized list is compared, the new list cannot contain the old list.

### Root Cause

The `wantErr` constant at `internal/cli/prompt_setup_commands_test.go:133` was not updated alongside Aha support.

### Source

The test runs the shipped CLI with no provider and pipe/closed stdin. `runConnect` returns usage before prompting; the failing assertion compares that error with stale text.

### Fix Direction

Update the expected usage list to include Aha, preserving all behavioral assertions.

## Problem Statement

Reproduction: `go test ./internal/cli -run '^TestConnectProviderPickerStaysSilentWithoutATerminal$' -count=1`.

The parent session's focused reproduction, read from `/tmp/hero-aha-before.log`, failed both `pipe` and `closed` subtests at line 153. Actual: `Error: usage: hero connect <type>  (github, jira, linear, gitlab, aha, confluence)`. Expected substring omitted `aha`. No other assertion failed in that run.

## Environment Details

Local macOS checkout on a new machine; the affected file has a Unix build tag. No tracker issue is associated with this local regression. This is a test maintenance defect, not evidence of missing machine configuration.

## Root Cause Analysis

Confirmed: `connect.go:103` includes Aha in the non-terminal usage error. The empty interactive answer at line 111 returns the same list. The registry at line 504 contains Aha with its verification function; provider fields also exist. Removing Aha from runtime output would misrepresent implemented support.

Corpus search found the completed `interactive-setup-and-connect-closure` feature as the origin of these safety tests. Its prompt-preservation contract remains relevant and must not be weakened. The completed `codex-command-workflow-surface-split-brain` spec already records this same unrelated stale expectation at lines 370–371. Neither completed item is reopened; this spec addresses the remaining regression.

## Code Flow (End to End)

1. `internal/cli/prompt_setup_commands_test.go:132` starts the test and selects pipe/closed stdin cases.
2. `internal/cli/prompt_setup_commands_test.go:140` runs the binary with `connect` and no provider argument.
3. `internal/cli/connect.go:82` enters `runConnect`, loads credentials, and reaches the missing-provider branch.
4. `internal/cli/connect.go:102` detects non-terminal stdin and returns the usage error including Aha before invoking the picker.
5. `internal/cli/prompt_setup_commands_test.go:152` compares the extracted error line with the stale complete provider-list substring and fails.

## Key Files

### CLI and regression test

| File | Lines | Relevance |
|---|---|---|
| `internal/cli/prompt_setup_commands_test.go` | 132–162 | Failing expectation and preserved safety assertions |
| `internal/cli/connect.go` | 82–113, 491–510 | Missing-provider refusal and provider registry |

## Secondary Defects

None identified within this bounded failure.

## Suggested Fix Approach

File: `internal/cli/prompt_setup_commands_test.go`, `TestConnectProviderPickerStaysSilentWithoutATerminal`.

Before:

```go
const wantErr = "usage: hero connect <type>  (github, jira, linear, gitlab, confluence)"
```

After:

```go
const wantErr = "usage: hero connect <type>  (github, jira, linear, gitlab, aha, confluence)"
```

Why: matches current intentional provider support without removing any check for timeout, exit status, unexpected prompts, or consuming piped input. No production changes are needed. The Hero anchor check found no tripwire applicable to this test-only correction.

## Acceptance Criteria

- [ ] AC-1: The expected non-terminal usage list includes `aha`; both pipe and closed subtests pass with all existing behavioral assertions preserved.

## Test Plan

Existing coverage: `TestConnectProviderPickerStaysSilentWithoutATerminal`, `TestConnectProviderPickerOffersEveryProvider`, `TestConnectProviderOptionsComeFromTheProviderMap`, and other `TestConnectProvider*` cases in `internal/cli/prompt_setup_commands_test.go`.

Test changes needed: update only the one stale constant. Run `go test ./internal/cli -run '^TestConnectProvider' -count=1` to cover refusal, interactive selection, registry alignment, JSON mode, supplied arguments, and unknown choices.

Regression scope: test-only expected output; runtime behavior remains unchanged. Parent delivery may run the broader suite to verify the setup objective.

## Notes

Code intelligence overview, symbol search, dependencies, and hot-file lookups were consulted; symbol search did not locate `runConnect`, so direct source reads supplied the evidence. No source files were edited during diagnosis.

## Recap

The test accurately enforces non-terminal safety but expects an obsolete provider list. Adding Aha to that expected text is the smallest correction.

## Changes

1. Update only `wantErr` in `internal/cli/prompt_setup_commands_test.go` to include `aha` between `gitlab` and `confluence`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: Aha in expected usage; pipe/closed pass with assertions preserved | DONE | `internal/cli/prompt_setup_commands_test.go:133`; failing reproduction `/tmp/hero-aha-before.log`, passing provider suite `/tmp/hero-aha-after.log` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Update stale wantErr constant | DONE | One-line diff in `internal/cli/prompt_setup_commands_test.go:133`; no runtime or behavioral assertion changes |

### Exercise-the-feature check

- [x] `go test -race ./internal/cli -run '^TestConnectProvider' -count=1` passed all seven tests, including real CLI execution under pipe and closed stdin. `go build ./...` and `git diff --check` passed.

### Excellence Bar self-check

- [x] Yes — minimal correction preserves the independently asserted output contract and all non-terminal safety checks.

## Final validation

`go test ./...` passed (output: `/tmp/hero-aha-full-suite.log`). Hero verification passed; `--skip-tests` avoided repeating this completed run.
