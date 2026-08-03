---
title: "Interactive Setup and Connect Closure"
slug: interactive-setup-and-connect-closure
type: feature
status: planning
created: 2026-08-03
domain: engineering
size: medium
priority: high
parent: interactive-cli-input-scoped-completion
depends-on: [prompt-and-tty-contract-closure]
relates-to:
  - connect-writer-unification
  - prompt-adoption-setup-commands
  - connect-help-accuracy
  - uninstall-target-parity
tags: [cli, prompt, connect, setup, uninstall]
---

# Interactive Setup and Connect Closure

## Context

The donor branch implements the original guided setup surfaces and unifies
connect's writer, but missing connect fields can still consume or wait on
non-TTY input. The curated implementation must preserve the useful human path
without importing later uninstall/config cleanup.

## Goal

Selectively port and close the original setup experience: ask only for missing
values at a terminal, preserve fully supplied command behavior, make connect's
role semantics identical across entry paths, and keep install/uninstall target
support symmetrical.

## Design brief

The `/design` pass must constrain the work to:

1. Connect provider selection and missing fields, including one write path for
   role, capability, and default semantics and correct `--role` routing.
2. Install/uninstall target selection across exactly
   `opencode|cursor|claude|copilot|codex|generic`.
3. Missing-value prompts for `repos add`, `users add`, `users passwd`, `trust`,
   and the original satellite confirmation flow.
4. Accurate connect help for the implemented flags and interactive behavior.
5. Live-open-pipe tests for connect plus golden compatibility tests for fully
   supplied flag paths.
6. Resolver-level validation that both interactive and flag-driven
   `code-host` connections work.

## Boundaries

- Do not change prompt primitives or selector infrastructure.
- Do not add setup prompts beyond the literal list above.
- Keep only uninstall behavior necessary for six real targets; exclude shared
  root-block removal, Codex line-welding, and other later cleanup.
- Do not add guided `hero init`, network-backed pickers, or a form engine.

## Acceptance targets

- Prompts appear only for missing setup values and only on a TTY.
- Closed and live non-TTY connect input fail promptly without consuming fields.
- Explicit `--role code-host` and interactive equivalent persist resolver-valid
  role/capability/default state.
- Install and uninstall expose and perform all six target paths.
- Existing fully supplied invocations preserve baseline output and exit status
  except for the finite corrections named by the parent initiative.

## Kickoff

Design the bounded setup/connect port after the prompt contract verifies. Own
only connect, the literal setup command list, six-target parity, and connect
help. Add live-pipe and resolver-level evidence. Preserve all supplied-flag
paths. Do not grow the prompt layer, add commands, absorb unrelated uninstall
repairs, or import any init, index, alias, timeout, graph, or invocation-lint
work from the donor branch.

→ `/design interactive-setup-and-connect-closure`
