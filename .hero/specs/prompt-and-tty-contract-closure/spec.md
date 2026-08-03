---
title: "Prompt and TTY Contract Closure"
slug: prompt-and-tty-contract-closure
type: feature
status: completed
created: 2026-08-03
domain: engineering
size: large
priority: critical
parent: interactive-cli-input-scoped-completion
relates-to:
  - cli-prompt-package-core
  - cli-prompt-package-adoption
  - cli-input-classification
tags: [cli, prompt, tty, security, platform]
delivery_method: manual
completed_at: 2026-08-03T20:44:30Z
---

# Prompt and TTY Contract Closure

## Context

The donor branch contains the original shared prompt package and most prompt-site
migrations, but its acceptance proof is incomplete. Secret entry hardcodes
`/dev/tty`, the remaining `new.go` scanner weakens the one-authority claim, and
closed-reader tests do not prove that a live non-TTY pipe cannot hang.

## Goal

Selectively port and close the original shared prompt foundation so every
interactive reader follows one cross-platform stream, TTY, secret, JSON, and
NEVER-PROMPT contract without changing `new --interactive`'s opt-in semantics.

## Approach

Port the donor prompt package and all original migrations as one foundation.
Splitting core from adoption would intentionally leave competing readers and
predicates between deliveries, recreating the condition this work removes.

`internal/cli/prompt` owns `Prompt`, `Choice`, `Confirm`, `Secret`,
`IsInputTTY`, and `IsOutputTTY`. Input/output checks use `term.IsTerminal` on
the actual Cobra streams and remain distinct. Callers decide whether missing
input is an error before invoking a primitive; primitives do not guess command
policy.

Secure input uses platform files behind one package-private abstraction:

- Unix opens `/dev/tty` for protected input and label output.
- Windows opens console input/output handles (`CONIN$` / `CONOUT$`) and calls
  `term.ReadPassword` on the input handle.
- Failure to acquire a protected terminal returns `prompt.ErrNoTTY`; there is
  no echoed fallback on either platform.

All ordinary input flows through `cmd.InOrStdin()`. Remove `connectInput`,
`newStdin`, `install.isTerminal`, `note.hasPipedInput`, and
`exportIsTerminal`. `new --interactive` remains explicitly opt-in; only its
reader plumbing changes. `brief` migrates its rendering check to
`IsOutputTTY(cmd.OutOrStdout())`, never the input predicate.

## Changes

1. Add `internal/cli/prompt/prompt.go` plus platform-specific secret-terminal
   files and tests.
   - Port the four primitives and two predicates from the donor branch.
   - Replace the Unix-only `Secret` body with the platform abstraction above.
   - Keep `ErrNoTTY` stable so commands can name their supported automation path.
2. Port the original prompt-site migrations in these files:
   `internal/cli/connect.go`, `internal/cli/install.go`,
   `internal/cli/install_satellites.go`, `internal/cli/users.go`,
   `internal/cli/skill.go`, `internal/cli/handoff.go`,
   `internal/cli/export.go`, `internal/cli/note.go`, and
   `internal/cli/brief.go`.
   - Use Cobra input/output streams for every ordinary prompt.
   - Gate before reading: missing value plus real input TTY, with JSON and
     NEVER-PROMPT suppression where applicable.
   - Delete every replaced reader and TTY helper after its final caller moves.
3. Reconcile `internal/cli/new.go` and its tests.
   - Remove the package-level `newStdin` injection variable.
   - Pass `cmd.InOrStdin()` to the interactive helper/scanner.
   - Preserve the explicit `--interactive` requirement and existing defaults.
4. Preserve exactly two deliberate compatibility corrections.
   - Non-TTY `hero install project` without `--target` returns the existing
     actionable error instead of silently choosing `opencode`.
   - `hero admin users add/passwd` without a protected terminal returns an
     actionable error instead of reading an echoed password.
5. Port and strengthen the donor regression fixtures and standing policy tests.
   - Record stdout, stderr, and exit status for supplied, closed, non-TTY, and
     TTY-like paths at every migrated site.
   - Add an open `io.Pipe` liveness harness that keeps the writer open and
     asserts the command returns before the deadline without reading.
   - Enumerate all NEVER-PROMPT command paths and all prompting commands that
     expose `--json`.
   - Add platform tests for the secret-terminal opener and Windows CI runtime or
     an equivalent console-handle seam; cross-compilation alone is insufficient.

### Delivered files

- `internal/cli/prompt/prompt.go`, `secret_terminal*.go` — shared primitives,
  stream predicates, and protected Unix/Windows terminal adapters.
- `internal/cli/connect.go`, `internal/cli/install.go`, `internal/cli/install_satellites.go`, `internal/cli/users.go`, `internal/cli/skill.go`, `internal/cli/handoff.go`, `internal/cli/export.go`, `internal/cli/note.go`, `internal/cli/brief.go`, and `internal/cli/new.go` — Cobra-stream prompt migration and non-TTY guards.
- `internal/cli/prompt_adoption_test.go`, `internal/cli/prompt_baseline_test.go`, `internal/cli/prompt_contract_test.go`, `internal/cli/prompt_json_test.go`, `internal/cli/prompt_policy_test.go`, `internal/cli/prompt_sanctioned_breaks_test.go`, `internal/cli/prompt_streams_test.go`, `internal/cli/note_stdin_test.go`, `internal/cli/new_test.go`, `internal/cli/e2e_test.go`, `internal/cli/connect_integrations_test.go`, and `internal/cli/testdata/prompt_baseline/` — stream, policy, JSON, baseline, PTY, and open-pipe regression coverage.
- `internal/ptytest/` — portable PTY support for real-terminal test cases.

## Boundaries

- Do not add setup prompts or selector behavior; the two adoption children own
  those surfaces.
- Do not redesign `new --interactive` or add guided `hero init`.
- Do not introduce a form/schema engine.
- Do not port unrelated uninstall, index, alias, timeout, or invocation-lint work.

## Risks

1. `Secret` cannot be made testable by accepting an arbitrary reader; that
   would reintroduce the insecure pipe path. Test the terminal acquisition seam.
2. `brief` asks about output, not input. Using `IsInputTTY` changes rendering.
3. Byte-at-a-time reads avoid per-prompt buffer read-ahead. Reintroducing a new
   `bufio.Reader` per prompt can swallow the next answer.
4. A golden baseline can normalize a regression. The two sanctioned corrections
   must be asserted separately and everything else compared byte-for-byte.
5. Size is `large`: this consolidates the original large core and medium
   adoption plus Windows and `new.go` closure. Keep it one child because a
   half-migrated prompt authority is not a safe deliverable state.

## Acceptance Criteria

- **AC-1:** THE SYSTEM SHALL expose `Prompt`, `Choice`, `Confirm`, `Secret`, `IsInputTTY`, and `IsOutputTTY` only through `internal/cli/prompt` for interactive CLI input.
- **AC-2:** THE SYSTEM SHALL retain distinct input- and output-terminal predicates based on the actual stream and SHALL NOT treat `/dev/null` as a terminal.
- **AC-3:** THE SYSTEM SHALL remove `connectInput`, `newStdin`, `install.isTerminal`, `note.hasPipedInput`, `exportIsTerminal`, and every other replaced interactive reader or TTY helper.
- **AC-4:** WHEN `new --interactive` runs THE SYSTEM SHALL read through `cmd.InOrStdin()` while preserving its explicit opt-in semantics.
- **AC-5:** WHEN required input is missing and stdin is closed or is a live non-TTY pipe THE SYSTEM SHALL return promptly with the command's existing actionable error and SHALL NOT prompt or wait for EOF.
- **AC-6:** WHEN a protected secret is requested on Unix or Windows THE SYSTEM SHALL read it without echo through the platform terminal implementation.
- **AC-7:** IF no protected terminal is available THEN THE SYSTEM SHALL return `ErrNoTTY`, SHALL NOT read an echoed stream, and SHALL direct the caller to its supported automation mechanism.
- **AC-8:** WHERE `--json` IS ENABLED THE SYSTEM SHALL NOT prompt at any migrated site.
- **AC-9:** THE SYSTEM SHALL NOT prompt on any command classified NEVER-PROMPT.
- **AC-10:** WHEN all required flags or arguments are supplied THE SYSTEM SHALL preserve baseline stdout, stderr, exit status, and mutation behavior except for AC-11 and AC-12.
- **AC-11:** WHEN `hero install project` has no TTY and no `--target` THE SYSTEM SHALL fail instead of selecting `opencode`.
- **AC-12:** WHEN user password entry has no protected terminal THE SYSTEM SHALL fail instead of accepting echoed input.
- **AC-13:** THE SYSTEM SHALL drive every migrated ordinary prompt through Cobra-configured streams in tests.

## Validation

- Run focused prompt package and prompt-site tests, including the live-open-pipe
  liveness harness and `/dev/null` classification.
- Run Unix terminal/PTY tests and Windows console-seam runtime tests; also
  cross-build `GOOS=windows`.
- Run the standing NEVER-PROMPT and JSON matrices.
- Run `go test -count=1 ./internal/cli/...`, `go test -race ./internal/cli/...`,
  `go vet ./...`, and `go build ./...`.
- Falsify AC-5 and AC-6 against the donor code: connect must hang/consume the
  live pipe and Windows secret acquisition must fail before the closure.

## Kickoff

Closes the CLI's shared prompt, TTY, secret-input, and Cobra-stream contract.

**Status:** delivering — implementation and validation are complete; cold audit and Hero verification remain.

**Pick up at:** review the completion ledger against the prompt package and migration tests, then run the cold audit and `hero spec verify`.

→ `hero spec verify prompt-and-tty-contract-closure --skip-tests`

**Files:** `internal/cli/prompt/prompt.go`, `internal/cli/prompt/secret_terminal_windows.go`, `internal/cli/prompt_policy_test.go`, `internal/cli/prompt_streams_test.go`
**Skip:** setup behavior, selectors, init, and unrelated donor changes.

## Completion Ledger

Stack detected: Go. Loaded `stack-detection`, `go-stack`,
`implementation-principles`, `testing-and-validation`, `agent-reliability`,
`security-review`, and `completion-ledger`. Validation: focused prompt/stream
tests, baseline regeneration and comparison, `go test -count=1
./internal/cli/...`, `go test -race ./internal/cli/...`, `go vet ./...`,
`go build ./...`, `GOOS=windows GOARCH=amd64 go build ./...`, and compilation
of the Windows console-seam test binary.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Shared prompt API is the only interactive input authority | DONE | `internal/cli/prompt/prompt.go` exports `Prompt`, `Choice`, `Confirm`, `Secret`, `IsInputTTY`, and `IsOutputTTY`; `prompt_contract_test.go` pins signatures. |
| 2 | Input/output TTY checks are distinct and reject `/dev/null` | DONE | `internal/cli/prompt/prompt.go` uses `term.IsTerminal` on each actual stream; `prompt_test.go` exercises PTYs, pipes, files, and `/dev/null`. |
| 3 | Replaced readers and TTY helpers are removed | DONE | `connectInput`, `newStdin`, `isTerminal`, `hasPipedInput`, and `exportIsTerminal` are absent; `prompt_adoption_test.go` holds the structural guard. |
| 4 | `new --interactive` reads `cmd.InOrStdin()` and remains opt-in | DONE | `internal/cli/new.go` passes Cobra input to `collectInteractiveInputs`; `new_test.go` drives real PTY Cobra input. |
| 5 | Closed and live non-TTY input returns promptly without prompting | DONE | Non-TTY guards cover migrated required-input sites; `TestPromptSitesReturnBeforeLivePipeEOF` keeps a pipe writer open and proves each site returns before deadline. |
| 6 | Unix/Windows protected secret input is non-echoed | DONE | `secret_terminal_unix.go` reads `/dev/tty`; the Windows adapter reads `CONIN$` with `term.ReadPassword` and calls portable `openWindowsConsoleFiles`; locally executed seam tests verify `CONIN$`/`CONOUT$` names and input-handle cleanup. |
| 7 | No protected terminal yields `ErrNoTTY` and no echoed fallback | DONE | `Secret` maps acquisition/read failure to `ErrNoTTY`; `secret_terminal_test.go`, `users.go`, and sanctioned-break tests cover refusal and alternatives. |
| 8 | JSON paths never prompt | DONE | `connect.go` routes `connectJSON` through the non-interactive path; `prompt_json_test.go` runs missing-value install and connect JSON invocations under a PTY and asserts neither blocks or emits a prompt. |
| 9 | NEVER-PROMPT commands do not prompt | DONE | `prompt_policy_test.go` enumerates and structurally checks all NEVER-PROMPT families. |
| 10 | Fully supplied inputs preserve baseline behavior | DONE | `prompt_baseline_test.go` records stdout, stderr, status, and supplied-input fixtures; regenerated fixtures compare byte-for-byte. |
| 11 | Non-TTY install project without target fails | DONE | `install.go` rejects missing `--target` before a non-TTY read; `prompt_sanctioned_breaks_test.go` verifies no opencode install. |
| 12 | Password entry without protected terminal fails | DONE | `users.go` wraps `ErrNoTTY` with the `--password` path; sanctioned-break and stream tests reject echoed input. |
| 13 | Ordinary prompts are driven through Cobra streams in tests | DONE | `prompt_streams_test.go`, `new_test.go`, and `connect_integrations_test.go` exercise `SetIn`/`SetOut` stream plumbing. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add shared prompt package and platform secret files | DONE | `internal/cli/prompt/` contains primitives, predicates, Unix `/dev/tty`, Windows `CONIN$`/`CONOUT$`, and acquisition seams/tests. |
| 2 | Port prompt-site migrations | DONE | `connect.go`, `install.go`, `install_satellites.go`, `users.go`, `skill.go`, `handoff.go`, `export.go`, `note.go`, and `brief.go` use shared stream/predicate rules. |
| 3 | Reconcile `new.go` and tests | DONE | `newStdin` removed; interactive scanner receives `cmd.InOrStdin()`; PTY Cobra-stream tests preserve explicit `--interactive`. |
| 4 | Preserve two compatibility corrections | DONE | Install target and protected-password failures are covered by `prompt_sanctioned_breaks_test.go`. |
| 5 | Strengthen fixtures and policy tests | DONE | Baselines capture supplied/closed/pipe/TTY paths; JSON (including connect), NEVER-PROMPT, executable Windows console seam, and live-open-pipe harnesses are present. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: `go test -count=1 ./internal/cli -run '^TestPromptSiteBaseline$'` rebuilt the CLI subprocess and compared supplied, closed, pipe, and TTY fixtures; `TestPromptSitesReturnBeforeLivePipeEOF` exercised every migrated site against an open pipe.

### Excellence Bar self-check

Honest answer to "would a senior engineer who cares about this codebase be proud to ship this?" — yes; the prompt authority, platform secret boundary, stream ownership, and liveness failures are all explicit and covered.
