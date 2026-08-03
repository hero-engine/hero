# Delivery audit — corpus-selector-closure

**Audited diff:** `git diff dfed3dc...268214c`
**Audited commit:** `268214cd55b059831984ad598f09fdb2657affdd`
**Verdict:** SHIP
**Surface:** clean
**Confidence:** high

## Acceptance criteria

- [x] **AC1 — corpora of 25 or fewer render directly through `Choice`.** `pickFromCorpus` takes the direct-choice branch at `len(candidates) <= pickerMax` (`internal/cli/selector.go:66-72`), with concrete one-candidate and exactly-25 coverage in `TestPickerRendersOneCandidate` and `TestPickerRendersExactlyAtTheCap` (`internal/cli/selector_test.go:299`, `:529`).
- [x] **AC2 — oversized local corpora remain fully reachable while rendered choices stay bounded.** The filter loop searches the complete corpus and only calls `Choice` after a result set reaches 25 or fewer (`internal/cli/selector.go:74-98`). `TestPickerFiltersAnOversizedCorpus` reaches an exact candidate outside the initial 25, and `TestPickerFiltersTwoHundredFiftyCandidatesToABoundedChoice` selects an outside-first-25 candidate from a 250-item corpus (`internal/cli/selector_test.go:381`, `:415`).
- [x] **AC3 — case-insensitive exact matches select immediately.** The `EqualFold` pass precedes substring choice rendering (`internal/cli/selector.go:82-85`), exercised by `TestPickerFiltersAnOversizedCorpus` (`internal/cli/selector_test.go:381`).
- [x] **AC4 — no-match filters report and retry without mutation.** Empty filtered sets print a diagnostic and continue the bounded retry loop (`internal/cli/selector.go:88-98`). `TestPickerRetriesNoMatchThenSelects` proves retry-and-select behavior, while the cancellation/invalid-selection mutation assertion guards the score boundary (`internal/cli/selector_test.go:446`, `:470`).
- [x] **AC5 — blank cancellation exits nonzero without mutation.** Blank filter and blank choice responses return `ErrSelectorCancelled` before command work begins (`internal/cli/selector.go:79-80`, `:101-109`). `TestPickerCancellationAndInvalidFinalChoiceDoNotScore` verifies both nonzero results and absence of score-state mutation (`internal/cli/selector_test.go:470`).
- [x] **AC6 — empty corpora retain the original missing-argument behavior.** `pickFromCorpus` returns the caller's original Cobra missing-argument error before displaying a picker (`internal/cli/selector.go:66-69`). This is exercised for score, skill, and handoff accept (`internal/cli/selector_test.go:253`, `:1135`, `:1572`).
- [x] **AC7 — supplied positional arguments bypass selector discovery and prompting.** The shared skill target returns a supplied value before discovery (`internal/cli/selector.go:176-185`), and each adopter follows the same missing-only selection shape. Supplied-value tests cover score, supersede, size, skill run, handoff, and handoff accept (`internal/cli/selector_test.go:187`, `:866`, `:989`, `:1159`, `:1391`, `:1593`).
- [x] **AC8 — non-TTY and JSON paths preserve strict argument validation.** `selectorArgs.rule` relaxes arity only when input is a TTY, JSON mode is off, and values are missing (`internal/cli/selector.go:45-61`). Non-TTY coverage spans every adopter family, and JSON coverage is present for score, verify, and handoff (`internal/cli/selector_test.go:204`, `:233`, `:612`, `:643`, `:718`, `:827`, `:938`, `:1082`, `:1466`, `:1516`, `:1624`).
- [x] **AC9 — only the frozen command set gained selectors.** The production diff is limited to the shared selector plus `score`, `verify`, `spec move`, `supersede`, `size`, the five specified skill verbs, `handoff`, and `handoff accept`. The adopter matrix is exercised in `internal/cli/selector_test.go`, including all five skill verbs through `TestSkillCommandsPromptAtATerminal` (`:1111`).
- [x] **AC10 — candidate sources are local and preserve established filtering/order.** Spec candidates use the existing active/subproject/recency selector, handed-back candidates retain that ordering, installed skills come from local discovery, and peer aliases are deterministically sorted (`internal/cli/selector.go:123-203`). `TestSpecPickerOrderingMatchesHeroList`, `TestSelectorFilteringIsStableAndScopeAware`, `TestSelectorCorpusResolutionIsLocal`, and `TestHandoffAcceptOffersOnlyHandedBackSpecs` cover those contracts (`internal/cli/selector_test.go:329`, `:500`, `:560`, `:1541`).

## Changes

- [x] `internal/cli/selector.go` provides the bounded, stable, local selector primitive and candidate resolvers.
- [x] `internal/cli/score.go` adopts missing-slug interactive selection without changing supplied/noninteractive behavior.
- [x] `internal/cli/verify.go` adopts missing-slug interactive selection without changing supplied/noninteractive behavior.
- [x] `internal/cli/spec_move.go` adopts missing-source selection while retaining required flag validation.
- [x] `internal/cli/supersede.go` independently selects either omitted positional while preserving scan/list/unset modes.
- [x] `internal/cli/size.go` selects only in the zero-argument interactive mode and preserves check/ack modes.
- [x] `internal/cli/skill.go` shares the selector across show/run/edit/rm/log and preserves supplied-name behavior.
- [x] `internal/cli/handoff.go` selects missing spec/peer arguments and restricts accept candidates to handed-back specs.
- [x] `internal/cli/selector_test.go` covers the adopter matrix, 0/1/25/26/250 corpus boundaries, outside-first-25 reachability, exact/no-match/cancel/invalid flows, stable ordering, scope, and frozen-command structure.

## Ledger and validation assessment

All 10 acceptance-criterion rows and all 9 Changes rows in the spec's Completion Ledger are `DONE`, and their evidence is supported by the audited implementation and test bodies. No `PARTIAL`, `SKIPPED`, or `BLOCKED` row remains.

The supplied evidence reports the focused selector/adopter matrix, full `internal/cli` normal and race suites, `go vet`, and build checks passing. `git diff --check dfed3dc...268214c` is clean. The unrelated dirty `.hero/drive/interactive-cli-input-scoped-completion.json` projection is outside the audited commit range and was not treated as delivery evidence.

## Audit conclusion

The implementation restores the initiative's originally intended interactive selectors on the frozen command set, bounds visible choices without truncating reachability, and preserves the legacy explicit-argument, non-TTY, and JSON contracts. No correctness defect, scope drift, or performative ledger evidence was found.
