---
title: "Generated Command Refs Validated — Every `hero <subcommand>` Hero Emits Must Exist"
slug: generated-command-refs-validated
type: feature
status: planning
domain: engineering
priority: high
size: small
horizon: now
created: 2026-07-14
parent: hero-self-consistency
tags: [validation, cobra, generated-output, harness, self-consistency]
---

# Generated Command Refs Validated — Every `hero <subcommand>` Hero Emits Must Exist

## Goal

Every `hero <subcommand>` reference in Hero's generated output and installed instruction files resolves against the Cobra command registry, enforced by a test. Renaming a command without updating the text that names it fails the build instead of shipping silently.

## Kickoff

Hero's digest shipped a `hero recall` reference for however long — the command was renamed to `hero search` and nothing noticed. This extracts every command ref Hero emits and asserts it resolves against Cobra.

**Status:** planning — independent of the other children; can start immediately.

**Pick up at:** build the Cobra registry walker first (flatten all registered command paths), then the extractor. Use the `hero recall` case as the regression test — it must catch it.

→ `rg -n 'hero [a-z-]+' internal/digest/ core/ --only-matching | sort -u | head -40`

**Files:** `internal/digest/digest.go:930`, `internal/cli/root.go`, `internal/install/agents_md.go`, `internal/cli/install.go:115`
**Skip:** don't build general claim-checking (relations resolve, statuses accurate) — command refs only.

## Context

Parent initiative: `hero-self-consistency`. This child kills finding (C) permanently.

`internal/digest/digest.go:930` emits ``hero recall <topic>` to dig deeper`. `hero recall` does not exist — it was renamed to `hero search`. Worse, `internal/digest/digest_test.go:190` asserts the dead string is present, so a passing test enforces the lie.

Sibling child `resume-emits-dead-recall-command` (#1) fixes that instance. **This spec makes the class impossible.** The instance fix is cheap; without a gate, the next rename reintroduces it.

This child is **genuinely independent** — it reads the Cobra registry only, touching neither the type enum (#2) nor the boundary wiring (#4). It can be delivered in parallel with everything.

## Approach

Two halves, both small:

1. **Enumerate what exists.** Walk the Cobra command tree from the root command and flatten every registered command path (`search`, `check validate`, `peer call`, …). This is the ground truth for "does `hero X` exist?" and requires no new data — Cobra already knows.
2. **Extract what Hero claims exists.** Scan generated output and installed instruction files for `hero <subcommand>` references, then assert each resolves against the tree.

Design notes:

- **Extract, don't hardcode.** A list of known commands maintained by hand is a fourth source of truth and would recreate the problem this initiative exists to fix.
- **Handle multi-word subcommands.** `hero check validate` and `hero peer call` are real; a naive first-token match would pass `hero check bogus`. Resolve the longest path that matches, and flag unresolvable remainders.
- **Expect false positives and design the escape hatch up front.** Prose legitimately contains command-shaped strings that aren't invocations, including intentionally-stale examples. The repo already uses `<!-- drift-test:ignore -->` markers for exactly this (see the `kickoff-prompt` skill's worked examples, which deliberately document a phantom `hero nudge` command). **Reuse that existing marker convention rather than inventing a second one.**
- **Fail the build, don't warn.** Unlike #4's 781 uncalibrated issues, this check has a small, sharp, unambiguous surface: a named command either exists or it does not. There is no calibration question here, so this one errors.

## Changes

1. Add the Cobra registry walker
   - Flatten the command tree from the root into a set of full command paths.
   - Include aliases if Cobra registers any — an alias is a real, resolvable name.
2. Add the command-ref extractor
   - Match `hero <subcommand>` (and multi-word paths) in scanned content.
   - Resolve longest-match-first so `hero check validate` resolves as a unit and `hero check bogus` fails.
   - Honor the existing `<!-- drift-test:ignore -->` marker convention for intentional exceptions.
3. Scan generated output
   - `internal/digest/` — the digest is the highest-value surface; it is Hero's first impression on every cold start.
   - Any other package that emits user- or agent-facing text naming commands. Enumerate with `rg` rather than assuming the digest is the only one.
4. Scan installed instruction files — **all six targets**
   - `hero install` writes across six targets: `opencode | cursor | claude | copilot | codex | generic` (`internal/cli/install.go:115`).
   - `claude` → `CLAUDE.md`; **all others → `AGENTS.md`** (`internal/install/agents_md.go`).
   - The scan must cover **all six**, not just Claude — see Risks.
   - Cover the propagated command/agent/skill content that `hero install` writes, not only the root instruction file.
5. Add the regression test
   - Assert the gate catches the `hero recall` case: feed it the pre-fix digest string and require a failure.
   - This is the acceptance test for the whole spec — if it doesn't catch `recall`, the gate doesn't work.
6. Wire into the test suite
   - Runs in CI as an ordinary `go test`. No new infrastructure.

## Acceptance Criteria

- WHEN Hero's generated output names a `hero <subcommand>` THE SYSTEM SHALL resolve that reference against the Cobra command registry.
- WHEN an installed instruction file names a `hero <subcommand>` THE SYSTEM SHALL resolve that reference against the Cobra command registry, for all six install targets.
- IF a command reference does not resolve THEN THE SYSTEM SHALL fail the test suite and name the offending file, line, and reference.
- WHEN the gate is run against the pre-fix digest string ``hero recall <topic>`` THE SYSTEM SHALL fail — the regression test for finding (C).
- WHERE a command-shaped string is intentionally stale THE SYSTEM SHALL honor the existing `<!-- drift-test:ignore -->` marker and pass.
- THE SYSTEM SHALL resolve multi-word command paths such as `hero check validate` as a unit rather than matching only the first token.

## Boundaries

- **Command refs only.** Not general claim-checking. Spec-relations resolution, referenced-status accuracy, and file-path existence are explicitly out — the parent initiative defers them as a class. This is the tiny high-value slice.
- Do not fix the `hero recall` instance here — that is child #1, shipping standalone. This spec provides the gate; #1 provides the fix. If #1 lands first the regression test still uses the historical string.
- Do not validate flags or arguments, only command names.
- Do not touch the type/status enums (#2) or boundary wiring (#4).

## Risks

- **Tripwire `harness-changes-cover-all-targets` [high] is LIVE for this spec.** It validates installed instruction files, and `hero install` writes across six targets (`opencode | cursor | claude | copilot | codex | generic`). `claude` → `CLAUDE.md`, all others → `AGENTS.md`. Covering only Claude is the exact failure the tripwire exists to catch. Cover all six.
- **False positives are the main failure mode.** Prose contains command-shaped strings. If the gate is noisy it gets disabled and the initiative ships a check nobody runs — the precise anti-pattern finding (D) is about. The `drift-test:ignore` escape hatch must work before the gate is required.
- Multi-word command paths are the subtle correctness trap. A first-token match would pass `hero check bogus` and give false confidence.
- If the extractor must scan the six install targets' *rendered* output rather than their source templates, this spec grows. Verify which surface is authoritative early; if it means rendering all six targets, `size:` may need to move to `medium`.

## Validation

- `go test ./...` passes with the gate active.
- The regression test fails when fed `hero recall <topic>` and passes when fed `hero search <topic>` — verify both directions.
- Temporarily rename a real command and confirm the gate catches every reference to the old name.
- Confirm the gate scans all six install targets: add a bogus `hero notacommand` ref to a non-Claude (`AGENTS.md`) target and verify it fails. **Testing only the Claude path would leave the tripwire's exact failure mode uncovered.**
- Confirm `<!-- drift-test:ignore -->` suppresses a known-intentional stale ref (the `kickoff-prompt` skill's `hero nudge` examples are a live test case).
