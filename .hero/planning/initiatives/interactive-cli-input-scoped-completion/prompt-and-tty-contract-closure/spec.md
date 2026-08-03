---
title: "Prompt and TTY Contract Closure"
slug: prompt-and-tty-contract-closure
type: feature
status: planning
created: 2026-08-03
domain: engineering
size: medium
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

## Design brief

The `/design` pass must make these items concrete:

1. Port `internal/cli/prompt` and the original migrations across `connect.go`,
   `install.go`, `install_satellites.go`, `users.go`, `skill.go`, `handoff.go`,
   `export.go`, `note.go`, and `brief.go` without importing later side quests.
2. Reconcile `new.go`'s injectable scanner with the shared authority while
   preserving the explicit `--interactive` behavior.
3. Keep input and output terminal predicates separate and use real terminal
   checks rather than `ModeCharDevice` alone.
4. Design platform-specific secure secret input that works on Windows and Unix
   and never accepts echoed fallback input.
5. Define the prompt-policy matrix for closed stdin, a live non-TTY pipe, TTY,
   explicit values, `--json`, and every NEVER-PROMPT command.
6. Preserve exactly two behavior corrections here: no silent `opencode` target
   on non-TTY install, and no echoed password fallback.

## Boundaries

- Do not add setup prompts or selector behavior; the two adoption children own
  those surfaces.
- Do not redesign `new --interactive` or add guided `hero init`.
- Do not introduce a form/schema engine.
- Do not port unrelated uninstall, index, alias, timeout, or invocation-lint work.

## Acceptance targets

- One prompt/TTY authority and no competing interactive reader remain.
- Every original prompt site is Cobra-stream-testable.
- Closed and live non-TTY input fail promptly without a prompt or hang.
- Secure secret entry has runtime-valid platform seams, not compile-only proof.
- Input/output predicates, JSON suppression, and NEVER-PROMPT behavior are
  covered by standing tests.

## Kickoff

Design the clean foundation port for `interactive-cli-input-scoped-completion`.
Use the old branch as evidence, not as the base. Close the Windows secret,
remaining-reader, terminal-classification, and live-pipe gaps while preserving
the original opt-in and machine-path behavior. Name the exact owned files and
tests; exclude all setup, selector, index, init, alias, timeout, and invocation
guard work.

→ `/design prompt-and-tty-contract-closure`
