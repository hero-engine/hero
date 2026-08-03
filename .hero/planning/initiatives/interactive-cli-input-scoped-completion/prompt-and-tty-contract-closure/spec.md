---
title: "Prompt and TTY Contract Closure"
slug: prompt-and-tty-contract-closure
type: feature
status: planning
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
2. Port the original prompt-site migrations in `internal/cli/connect.go`,
   `install.go`, `install_satellites.go`, `users.go`, `skill.go`, `handoff.go`,
   `export.go`, `note.go`, and `brief.go`.
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

Design the clean foundation port for `interactive-cli-input-scoped-completion`.
Use the old branch as evidence, not as the base. Close the Windows secret,
remaining-reader, terminal-classification, and live-pipe gaps while porting the
complete original prompt-site set. Preserve `new --interactive`, JSON, and
machine paths. Keep secure terminal acquisition platform-specific; never add an
echoed test hook. Exclude setup behavior, selectors, index, init, aliases,
timeouts, invocation guards, and unrelated uninstall work.

→ `/design prompt-and-tty-contract-closure`
