# Delivery audit — cli-successor-test-contract-reconciliation

**Audited:** `git diff 9a28b90...d67b06c`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: real-terminal `skill save` prompts for and persists both fields, with empty and unterminated guards — `internal/cli/prompt_adoption_test.go:270` asserts both prompt bytes and the saved title through a PTY; `TestSkillSaveEmptyNameRejected` and `TestSkillSaveUnterminatedTerminalInputWritesNothing` cover empty input and terminal EOF without a file.
- [✓] AC-2: non-terminal `skill save` refuses without output, reads, or mutation — `TestSkillSaveNonTTYFailsFastAndWritesNothing` asserts the exact error, empty output, all input bytes unread, and an empty skills directory at `internal/cli/prompt_adoption_test.go:294`.
- [✓] AC-3: terminal handoff aliases parse and non-terminal input defaults silently — `TestPromptNextStatusAnswersAtATerminal` covers all eight supported answers through PTYs; `TestPromptNextStatusNonTTYTakesTheDeliveringDefault` asserts `delivering`, no output, and unread input at `internal/cli/prompt_adoption_test.go:494` and `internal/cli/prompt_adoption_test.go:527`.
- [✓] AC-4: the four GitHub connect closed/pipe baselines match provider-first failure and current help — only the four named fixtures changed, each now records `repository is required` and the current `--project` help; `TestPromptSiteBaseline` compares the complete subprocess bytes, and `TestInteractiveConnectFailsBeforeALiveNonTTYPipeCanBeRead` proves the live pipe is not read.
- [✓] AC-5: the generic descriptor remains absent — `TestGenericFieldDescriptorRemainsAbsent` requires zero `collectFields` consumers and no `promptfield.go` at `internal/cli/selector_test.go:1726`; selector/setup guards retain direct prompt primitives, while `internal/cli/connect.go:640` retains private `collectConnectFields`.
- [✓] AC-6: no absent donor-only release-note path is required — the diff removes `TestSanctionedBreaksAreDocumented`; executable sanctioned-break tests remain in `internal/cli/prompt_sanctioned_breaks_test.go`, and the CLI package passes.
- [✓] AC-7: the isolated CLI package passes — archived evidence at `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md:72` records `go test -count=1 -timeout 10m ./internal/cli` passing in 83.292s after the terminal EOF assertions were added.
- [✓] AC-8: full suite, race, vet, native, and Windows gates pass — `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md:74` through line 80 records the green matrix; supplemental post-`d67b06c` evidence records `go test -count=1 -timeout 10m ./...` exiting zero with `internal/cli` in 131.405s.
- [✓] AC-9: the false historical PASS is explicitly corrected without deletion — the original row remains at `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md:46`, and the dated correction plus fresh results are appended at line 59.
- [✓] AC-10: production Go files remain unchanged — `git diff --name-status 9a28b90...d67b06c` contains only four `_test.go` files among Go changes; all other changes are four text fixtures and Hero spec, evidence, or generated projections.

## Changes

- [✓] Reconcile skill-save and handoff terminal tests — `internal/cli/prompt_adoption_test.go:270-378` and `internal/cli/prompt_adoption_test.go:494-580` exercise the delivered PTY and non-TTY stream classes with state, output, error, and unread-byte assertions.
- [✓] Refresh four superseded connect fixtures — the diff changes exactly `connect_github_{repo,secret}_prompt.{closed,pipe}.txt`, each by the provider-first error and current help bytes only.
- [✓] Enforce zero generic descriptor — `internal/cli/selector_test.go:1697-1747` and `internal/cli/prompt_setup_commands_test.go:809-840` encode absence, direct primitives, and private connect ownership.
- [✓] Remove the invalid donor release-note assertion — `internal/cli/prompt_policy_test.go` deletes only the nonexistent-doc test while retained behavior and byte-level help coverage pass.
- [✓] Append a dated correction to closing evidence — `.hero/specs/interactive-cli-acceptance-and-merge-gate/acceptance-evidence.md:59-83` preserves the unsupported historical row and records the corrected validation matrix.

## Open items

- None.

## Audit notes

- None.
