---
title: "Cold-start digest emits a dead `hero recall` command — and a test enforces it"
slug: resume-emits-dead-recall-command
type: bug
status: planning
domain: engineering
priority: high
size: trivial
horizon: now
created: 2026-07-14
parent: hero-self-consistency
tags: [digest, cold-start, dead-command, self-consistency]
---

# Cold-start digest emits a dead `hero recall` command — and a test enforces it

## Goal

The cold-start digest names a command that exists. `hero recall` is replaced with `hero search` at the one emission site, and the test that currently asserts the dead string is corrected to assert the live one.

## Kickoff

The digest tells a fresh agent session to run `hero recall <topic>`. That command doesn't exist — it was renamed to `hero search`. A passing test pins the lie in place.

**Status:** planning — two-line fix, fully diagnosed, no investigation needed.

**Pick up at:** change the format string at `internal/digest/digest.go:930` to `hero search`, then fix the assertion at `internal/digest/digest_test.go:190` that requires "hero recall". Check `sectionRecallTopic` naming while there.

→ `rg -n "hero recall|sectionRecallTopic" internal/digest/`

**Files:** `internal/digest/digest.go:930`, `internal/digest/digest_test.go:190`

## Issue

Found during the `hero-self-consistency` scoping pass, 2026-07-14. Not reported externally — which is itself the point: this is emitted into every truncated cold-start digest and no human reads digests critically.

## Investigation

`internal/digest/digest.go:930` renders the truncation hint:

```go
topic := sectionRecallTopic(sec.Title)
fmt.Fprintf(&s, "_…+%d more — `hero recall %s` to dig deeper_\n", sec.Truncated, topic)
```

`rg -n 'Use:\s*"recall' internal/cli/` returns nothing — no Cobra command named `recall` is registered. The command was renamed to `hero search`. The digest was not updated.

`internal/digest/digest_test.go:190` asserts the dead string:

```go
if !strings.Contains(md, "hero recall") {
    t.Errorf("expected 'hero recall' hint in truncated output:\n%s", md)
}
```

### Root cause

A rename (`recall` → `search`) updated the command registry but not the digest emission site. The test was written against the emitted string rather than against command existence, so it locked in the stale name and the rename passed CI green. The symptom is a dead command ref; the cause is that no gate connects generated output to the command registry — which is exactly what sibling child `generated-command-refs-validated` fixes permanently.

### Severity

Low blast radius, high embarrassment, disproportionate cost. Every truncated digest section instructs a fresh agent session to run a command that will fail. The agent burns a turn discovering it, and the digest is Hero's first impression on every cold start. There is no workaround — the string is hardcoded. Caused entirely by our own code.

## Changes

1. Fix the emission site in `internal/digest/digest.go:930`
   - Replace `hero recall %s` with `hero search %s` in the `Fprintf` format string.
   - Verify `hero search <topic>` is the correct invocation shape for a topic argument — confirm against the Cobra registration, not from memory.
2. Fix the test that enforces the lie in `internal/digest/digest_test.go:190`
   - Assert on `hero search` instead of `hero recall`.
   - Keep the assertion on emitted output; asserting registry resolution is `generated-command-refs-validated`'s job, not this fix's.
3. Check the `sectionRecallTopic` helper name in `internal/digest/digest.go`
   - Rename to `sectionSearchTopic` **only if** it is local to the digest package and has no external callers. Verify with `rg`. If renaming widens the diff, leave it and note it — this spec is `trivial` and should stay that way.
4. Sweep for sibling occurrences
   - `rg -n 'hero recall' internal/ core/ docs/` and fix any additional emission sites found.
   - The brief counts 3 occurrences of 1 dead ref; confirm the sweep matches that count before closing.

## Acceptance Criteria

- WHEN the digest truncates a section THE SYSTEM SHALL emit `hero search <topic>` and not `hero recall <topic>`.
- THE SYSTEM SHALL contain no occurrence of the string `hero recall` in emitted or generated output.
- WHEN the digest test suite runs THE SYSTEM SHALL assert the presence of the live command name, and SHALL fail if the dead name returns.

## Boundaries

- Do not build the general command-ref gate here. That is `generated-command-refs-validated` (sibling child #3), which uses this exact case as its regression test.
- Do not audit other generated surfaces for other dead refs. Ship this in one sitting.
- Do not touch the type or status enums.

## Risks

- Low. The only real risk is scope creep via the `sectionRecallTopic` rename — bail out of the rename if it reaches beyond the digest package.
- If the sweep in change 4 finds occurrences outside `internal/digest/`, they may live in an installed instruction file, which makes the change harness-facing and pulls in tripwire `harness-changes-cover-all-targets`. Check before assuming the fix is local.

## Validation

- `go test ./internal/digest/` passes with the corrected assertion.
- `rg -n 'hero recall' internal/ core/ docs/` returns zero hits.
- Manually render a truncated digest and confirm the emitted hint names a command that `hero help` lists.
