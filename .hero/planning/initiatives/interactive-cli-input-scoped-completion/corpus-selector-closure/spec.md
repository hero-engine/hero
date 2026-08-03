---
title: "Corpus Selector Closure"
slug: corpus-selector-closure
type: feature
status: planning
created: 2026-08-03
domain: engineering
size: medium
priority: high
parent: interactive-cli-input-scoped-completion
depends-on: [prompt-and-tty-contract-closure]
relates-to: [selector-spec-pickers, cli-input-classification]
tags: [cli, selector, prompt, corpus]
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

1. Port and close the selector infrastructure in `internal/cli/selector.go`.
   - Preserve `selectorArgs`, `jsonModeOn`, local corpus resolvers, existing
     missing-argument errors, and the 25-item direct-choice display limit.
   - Replace the hard cap with the filter-to-bounded-choice flow above.
   - Make blank input return a typed cancellation/error path so callers exit
     non-zero without mutation.
   - Keep filtering case-insensitive and stable-order; do not add fuzzy ranking.
2. Port selector adoption for exactly these command paths:

- `score`, `verify`, `spec move`, `supersede`, and `size`
- `skill show`, `skill run`, `skill edit`, `skill rm`, and `skill log`
- `handoff` and `handoff accept`

   Update only `internal/cli/score.go`, `verify.go`, `spec_move.go`,
   `supersede.go`, `size.go`, `skill.go`, and `handoff.go`, plus their tests.
   Preserve the donor rule that commands with strict arity gate through
   `selectorArgs.rule`, while zero-argument modes such as `size --check` and
   `supersede --scan` gate in `RunE`.
3. Preserve non-interactive behavior.
   - Supplied arguments bypass discovery and prompting.
   - Non-TTY input and `--json` retain the command's prior Cobra/usage error.
   - Existing zero-argument operational modes continue without a picker.
   - Empty corpora return the existing missing-argument error without printing
     an empty choice.
4. Add focused selector tests in `internal/cli/selector_test.go` and command
   tests beside the adopting commands.
   - Cover 0, 1, 25, 26, and 250 candidates.
   - Prove a candidate outside the first 25 can be filtered and selected.
   - Cover exact-filter acceptance, no-match retry, cancellation, invalid final
     choice, stable ordering, active subproject filtering, and handed-back-only
     filtering.
   - Cover every adopting command's supplied-argument, non-TTY, and `--json`
     paths.

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

Design the smallest real-scale selector completion after the prompt foundation
verifies. Port the donor's local resolvers and Cobra gates, then replace its
hard refusal above 25 with selector-local substring filtering into a bounded
choice. Keep the target list frozen. Prove large, empty, cancel, invalid,
explicit, non-TTY, and JSON cases. Do not add a TUI, network lookup, fuzzy
retrieval, prompt primitive, or command target.

→ `/design corpus-selector-closure`
