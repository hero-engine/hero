# Initiative acceptance evidence — interactive CLI successor

This map joins the parent and final-gate criteria to executable evidence. The
test names are intentionally stable search keys; commands and results are
recorded in the final gate Completion Ledger.

| Criterion | Executable or inspected evidence |
|---|---|
| Parent AC-1 | prompt_contract_test, prompt_adoption_test, prompt_streams_test, and the scope map show one prompt package with no legacy readers. |
| Parent AC-2 | internal/cli/prompt/prompt_test.go exercises distinct stream predicates, pipes, files, PTYs, and /dev/null. |
| Parent AC-3 | TestPromptSitesReturnBeforeLivePipeEOF and TestInteractiveConnectFailsBeforeALiveNonTTYPipeCanBeRead keep writers open and prove completion before EOF. |
| Parent AC-4 | TestSecretUsesOnlyTheProtectedTerminal, TestOpenWindowsConsoleFilesUsesTheProtectedConsoleHandles, TestOpenWindowsConsoleFilesClosesInputWhenOutputCannotOpen, and the Windows compile test exercise the secure-secret seam. |
| Parent AC-5 | TestPromptSiteBaseline and TestSuppliedSetupCompatibilityBaseline compare command, status, stdout, and stderr byte-for-byte; prompt_sanctioned_breaks_test isolates the four sanctioned corrections. |
| Parent AC-6 | prompt_json_test and TestInteractiveConnectNeverPromptsUnderJSON exercise JSON under a live PTY. |
| Parent AC-7 | prompt_policy_test inventories NEVER-PROMPT paths; setup primitive tests prevent indirect descriptor reuse. |
| Parent AC-8 | TestBothPathsPersistIdenticalState, TestRoleFlagConnectIsAcceptedByTheCodeHostResolver, and TestInteractiveConnectIsAcceptedByTheCodeHostResolver test the effective resolver. |
| Parent AC-9 | TestUninstallTargetPickerOffersAllSixTargets, TestInstallAndUninstallPickersEnumerateIdenticalTargets, the existing four uninstall tests, and TestUninstallCopilotAndGenericRoundTripsPreserveAdjacentUserFilesAndAGENTS cover all six targets and manifest preservation. |
| Parent AC-10 | selector_test covers 0, 1, 25, 26, and 250 candidates, filtering, exact match, stable ordering, and outside-first-25 selection. |
| Parent AC-11 | TestPickerCancellationAndInvalidFinalChoiceDoNotScore proves nonzero cancellation and no mutation. |
| Parent AC-12 | selector_test covers supplied, non-TTY, JSON, invalid, and frozen-target cases for every adopter. |
| Parent AC-13 | scope-provenance.md covers 229 production hunks plus four test-only portable PTY hunks and records no boundary categories. |
| Parent AC-14 | Full suite, focused race, vet, native build, Windows cross-build, platform-secret seam, and live-pipe commands are recorded below and in the ledger; final audit/verify remain root-owned gate actions. |
| Gate AC-1 to AC-4 | scope-provenance.md and donor-branch-disposition.md supply the hunk and donor classifications and retain the donor branch. |
| Gate AC-5 to AC-8 | Prompt, connect, uninstall, setup, and selector tests above provide the required matrices. |
| Gate AC-9 | The command matrix in the ledger is the reproducible result set. |
| Gate AC-10 | Completion Ledger is complete except for the deliberately root-owned cold-audit/verify step. |
| Gate AC-11 | The scope scan found no production defect. The gate made no production edit. |
| Gate AC-12 | Parent Progress and the final gate kickoff are updated to match the archived child status and pending root gates. |

## Platform and liveness evidence

- The live-pipe tests use an open io.Pipe writer and bounded completion
  assertions; they are not closed-reader substitutes.
- The local runtime seam test executes the Windows handle acquisition contract
  through its injectable opener. A Windows build additionally compiles the
  platform implementation and its dedicated test file. No claim of a local
  Windows console runtime is made from a cross-build alone.

## Validation record

All commands below exited zero on the clean successor after the evidence maps
were written.

| Command | Result |
|---|---|
| go test -count=1 -timeout 10m ./... | PASS — full repository suite. |
| go test -race -count=1 ./internal/cli ./internal/cli/prompt | PASS — affected CLI and prompt packages. |
| go vet ./... | PASS. |
| go build ./... | PASS — native build. |
| GOOS=windows GOARCH=amd64 go build ./... | PASS — Windows cross-build. |
| GOOS=windows GOARCH=amd64 go test -c ./internal/cli/prompt | PASS — Windows secret adapter and test compilation. |
| go test -count=1 ./internal/cli -run Test(PromptSitesReturnBeforeLivePipeEOF|InteractiveConnectFailsBeforeALiveNonTTYPipeCanBeRead|SuppliedSetupCompatibilityBaseline|UninstallCopilotAndGenericRoundTripsPreserveAdjacentUserFilesAndAGENTS)$ | PASS — live-pipe, compatibility, and manifest-round-trip evidence. |
| go test -count=1 ./internal/cli/prompt -run Test(OpenWindowsConsoleFilesUsesTheProtectedConsoleHandles|OpenWindowsConsoleFilesClosesInputWhenOutputCannotOpen|SecretUsesOnlyTheProtectedTerminal)$ | PASS — executed protected-console seam. |
| hero spec lint interactive-cli-acceptance-and-merge-gate | PASS — 12 of 12 EARS criteria. |
| hero spec score interactive-cli-acceptance-and-merge-gate | PASS — 95/100, grade A. |
| hero drift interactive-cli-acceptance-and-merge-gate | PASS — no drift detected. |
| git diff --check | PASS. |

## Correction — 2026-08-03

The original full-suite PASS row above was not supported by the checked-in
successor tree: stale cross-child prompt tests, four superseded connect golden
fixtures, and a rejected generic-descriptor assertion were already present.
`cli-successor-test-contract-reconciliation` preserves that historical row and
corrects the merge evidence rather than silently rewriting it.

After the test-contract-only repair, with no production `.go` change, the
following commands exited zero outside the desktop network sandbox:

| Command | Fresh result |
|---|---|
| go test -count=1 -timeout 10m ./internal/cli | PASS — `internal/cli` in 83.292s after adding the cold-audit-requested terminal EOF assertions. |
| go test -count=1 -timeout 2m ./internal/cli -run '^(TestSkillSave\|TestPromptNextStatus)' | PASS — complete affected PTY/non-TTY regression group. |
| go test -count=1 -timeout 10m ./... | PASS — post-repair full repository suite; `internal/cli` in 131.405s after `d67b06c`. |
| go test -race -count=1 -timeout 10m ./internal/cli ./internal/cli/prompt | PASS — CLI in 137.457s; prompt in 1.284s. |
| go vet ./... | PASS. |
| go build ./... | PASS. |
| GOOS=windows GOARCH=amd64 go build ./... | PASS. |
| GOOS=windows GOARCH=amd64 go test -c ./internal/cli/prompt | PASS. |
| git diff --check | PASS. |

The initial sandbox failures in callback-listener tests were separately traced
to denied loopback binds and are not counted as product evidence.
