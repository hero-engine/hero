---
title: "Corpus Selector Closure"
slug: corpus-selector-closure
type: feature
status: completed
created: 2026-08-03
domain: engineering
size: medium
priority: high
parent: interactive-cli-input-scoped-completion
depends-on: [prompt-and-tty-contract-closure]
relates-to: [selector-spec-pickers, cli-input-classification]
tags: [cli, selector, prompt, corpus]
delivery_method: manual
completed_at: 2026-08-03T21:01:25Z
---

# Corpus Selector Closure

## Context

The donor branch adds local-corpus selectors through
`internal/cli/selector.go`, but `pickFromCorpus` refuses to prompt above
`pickerMax = 25`. Hero itself has hundreds of open specs, so the spec selectors
are inert in the workspace that motivated them. The existing implementation is
useful evidence: it establishes local corpus resolvers, preserves Cobra arity
errors, suppresses pickers under `--json`, and keeps supplied arguments fast.
The completion must retain those properties while removing the hard usability
ceiling.

## Goal

Selectively port the original selector targets and make every one useful against
the full local corpus through bounded interaction, without changing explicit,
non-TTY, JSON, or machine-driven command paths.

## Approach

Port the donor selector architecture, including `selectorArgs`, shared local
corpus resolvers, and the missing-argument Cobra gates. Replace the current
"more than 25 means fail" branch with a two-stage picker:

1. At 25 or fewer candidates, continue to call `prompt.Choice` directly.
2. Above 25 candidates, ask for a case-insensitive substring filter through
   `prompt.Prompt`, report the number of matches, and repeat until the filtered
   set contains 1–25 candidates. Then call `prompt.Choice` over that set.

An exact candidate typed as the filter may be accepted immediately. A blank
filter cancels. A filter with no matches prints a bounded retry message and asks
again; it does not dump the corpus. The filter loop lives in
`internal/cli/selector.go`, not in the generic prompt package: corpus filtering
is selector policy, not a fifth prompt primitive.

All resolution remains local. Spec candidates retain `hero list`'s active-scope
filter and recency ordering; skill candidates retain installed-skill ordering;
handoff candidates remain restricted to locally registered peers or handed-back
specs as appropriate.

## Changes

- `internal/cli/selector.go` — local corpus resolvers, missing-argument gates,
  bounded filtering, cancellation errors, and stable selection helpers.
- `internal/cli/score.go` — selector adoption for an omitted spec target.
- `internal/cli/verify.go` — selector adoption while preserving JSON behavior.
- `internal/cli/spec_move.go` — selector adoption after destination validation.
- `internal/cli/supersede.go` — independent old/replacement spec selectors.
- `internal/cli/size.go` — zero-argument selector without changing `--check`.
- `internal/cli/skill.go` — shared installed-skill selection for five verbs.
- `internal/cli/handoff.go` — spec/peer and handed-back-spec selectors.
- `internal/cli/selector_test.go` — terminal adopter coverage and 0/1/25/26/250
  corpus, filtering, cancellation, scope, ordering, and handoff cases.

## Boundaries

- Do not discover or add selector targets during this work.
- Do not add network-backed selection, including team-user lookup.
- Do not reshape the command tree or absorb `cli-surface-consolidation`.
- Do not build a TUI framework, form engine, fuzzy retrieval subsystem, or
  exact-slug retrieval feature.
- Do not modify connect or setup behavior.

## Risks

1. A generic searchable-choice abstraction would expand the prompt package for
   one consumer. Keep the filter loop selector-specific.
2. Filtering can become an unbounded output path if it prints all matches.
   Always render at most 25 candidates.
3. Relaxed Cobra arity can accidentally make missing arguments succeed in
   non-TTY or JSON mode. Each command needs a negative-path assertion.
4. Handoff has two positional identifiers with different corpora. Preserve the
   partial-argument behavior rather than treating both as one list.

## Acceptance Criteria

- **AC-1:** WHEN a selector corpus contains 25 or fewer candidates THE SYSTEM SHALL render the existing bounded `prompt.Choice` interaction.
- **AC-2:** WHEN a selector corpus contains more than 25 candidates THE SYSTEM SHALL let the user narrow the full local corpus and reach any valid candidate without rendering more than 25 choices at once.
- **AC-3:** WHEN a filter exactly matches a candidate THE SYSTEM SHALL select that candidate without requiring an additional unbounded listing.
- **AC-4:** IF a filter matches no candidates THEN THE SYSTEM SHALL report the empty result and allow another bounded filter attempt without mutation.
- **AC-5:** IF the user cancels a selector THEN THE SYSTEM SHALL exit non-zero and SHALL NOT mutate workspace state.
- **AC-6:** IF the local corpus is empty THEN THE SYSTEM SHALL return the command's existing missing-argument error and SHALL NOT render an empty picker.
- **AC-7:** WHEN an explicit selector argument is supplied THE SYSTEM SHALL bypass corpus discovery and prompting.
- **AC-8:** WHEN stdin is non-TTY or `--json` is enabled THE SYSTEM SHALL preserve the pre-existing non-interactive error or operational mode and SHALL NOT prompt or hang.
- **AC-9:** THE SYSTEM SHALL add selectors only to `score`, `verify`, `spec move`, `supersede`, `size`, `skill show/run/edit/rm/log`, `handoff`, and `handoff accept`.
- **AC-10:** THE SYSTEM SHALL resolve selector candidates only from local workspace state and SHALL preserve each corpus's established filtering and ordering.

## Validation

- Run focused selector and adopting-command tests with race detection.
- Run the prompt-policy guard tests from `prompt-and-tty-contract-closure` to
  ensure selector adoption does not weaken NEVER-PROMPT or JSON behavior.
- Run `go test -count=1 ./internal/cli/...`, `go test -race ./internal/cli/...`,
  `go vet ./...`, and `go build ./...`.
- Falsify AC-2 against the donor implementation: the 26- and 250-candidate
  tests must fail on its hard-cap path before passing here.

## Kickoff

Makes local-corpus CLI selectors work at real workspace scale without changing
explicit or non-interactive command behavior.

**Status:** completed — the frozen selector set and 0/1/25/26/250 coverage
passed its independent SHIP audit and Hero verification.

**Pick up at:** consult the archived audit and Completion Ledger only when
investigating selector reachability, cancellation, or noninteractive behavior.

→ `.hero/specs/corpus-selector-closure/delivery-audit.md`

**Files:** `internal/cli/selector.go`, `internal/cli/selector_test.go`,
`internal/cli/handoff.go`, `internal/cli/supersede.go`
**Skip:** reopening delivery, TUI, fuzzy retrieval, network lookup, general
prompt changes, and selector targets outside the frozen list.

## Completion Ledger

Selector filtering is local-only and bounded at 25 choices. Validation passed:
`go test -count=1 ./internal/cli/...`, `go test -race ./internal/cli/...`,
`go vet ./...`, and `go build ./...`.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | ≤25 candidates render Choice | DONE | `internal/cli/selector.go` and `selector_test.go` cover 1 and 25 candidates. |
| 2 | >25 candidates narrow to bounded choice | DONE | 26- and 250-candidate terminal tests filter into at most 25 choices. |
| 3 | Exact filter selects immediately | DONE | `TestPickerFiltersAnOversizedCorpus` selects an exact 26th-corpus candidate. |
| 4 | No match reports and retries | DONE | `TestPickerRetriesNoMatchThenSelects` proves retry then selection. |
| 5 | Cancellation is non-mutating | DONE | `ErrSelectorCancelled` and cancellation/invalid score exercise stop before scoring. |
| 6 | Empty corpus keeps missing-argument error | DONE | score, skill, and handed-back empty-corpus tests cover no empty picker. |
| 7 | Supplied arguments bypass picker | DONE | adopter tests cover supplied score, size, skill, supersede, handoff, and accept paths. |
| 8 | Non-TTY and JSON do not prompt | DONE | selector gate plus adopter pipe/closed/JSON tests preserve Cobra errors and modes. |
| 9 | Only frozen command targets adopt selectors | DONE | `selector_test.go` exercises exactly score, verify, move, supersede, size, five skill verbs, handoff, and accept. |
| 10 | Local, ordered corpus resolution | DONE | `selector.go` uses local specs/skills/config; stable, active-scope, and handed-back tests pass. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `internal/cli/selector.go` | DONE | Shared local resolvers, selector gates, bounded filtering, and cancellation errors. |
| 2 | `internal/cli/score.go` | DONE | Omitted target selects locally; JSON still preserves Cobra arity. |
| 3 | `internal/cli/verify.go` | DONE | Omitted target selects from the already-discovered corpus. |
| 4 | `internal/cli/spec_move.go` | DONE | Destination validation still precedes source selection. |
| 5 | `internal/cli/supersede.go` | DONE | Old and replacement selectors preserve independent supplied values. |
| 6 | `internal/cli/size.go` | DONE | Zero-argument selector leaves `--check` operational. |
| 7 | `internal/cli/skill.go` | DONE | Five name-taking skill verbs share installed-skill selection. |
| 8 | `internal/cli/handoff.go` | DONE | Spec/peer and handed-back-only selectors retain partial argument behavior. |
| 9 | `internal/cli/selector_test.go` | DONE | PTY end-to-end coverage covers frozen adopters and corpus boundaries. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end through the built CLI in PTY tests: 26- and 250-candidate corpora narrowed and selected candidates outside the first 25, while cancellation and invalid choices exited non-zero before scoring.

### Excellence Bar self-check

Honest answer to "would a senior engineer who cares about this codebase be proud to ship this?" — yes; the change reuses local ordering/resolution rules, bounds every rendered choice list, and protects all explicit and machine-driven paths with focused end-to-end tests.
