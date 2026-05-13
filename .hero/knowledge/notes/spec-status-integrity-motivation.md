---
title: Spec-Status Integrity — Evidence From pre-commit-auto-stage-next
type: note
status: active
tags: [spec-integrity, retro, process, ac-tracking]
created: 2026-05-12
relations:
  - target: spec-status-integrity
    kind: motivates
  - target: pre-commit-auto-stage-next
    kind: evidence-from
---

# Spec-Status Integrity — Real-World Motivator

The in-flight `spec-status-integrity` feature is about graph-verifying delivery claims: a spec marked `completed` should actually have all its acceptance criteria shipped. The `pre-commit-auto-stage-next` retro (2026-05-12) is a concrete evidence point that this isn't theoretical.

## What happened

`pre-commit-auto-stage-next` was marked `status: completed` in v0.4. Eleven ACs were declared. Ten shipped — the substantial engineering: pre-commit hook plumbing, marker-block lifecycle, `hero check` integration, `hero scan` auto-install, escape-hatch preservation. One didn't:

> **AC #11:** THE SYSTEM SHALL include CLAUDE.md guidance telling agents to include hero workspace files in commits as a backstop for repos without the hook.

The "Changes" section also called it out explicitly as a literal one-line edit to the Hero-managed CLAUDE.md. It never landed in `internal/install/agents_md.go` (the canonical template source). Spec was marked done anyway.

The gap surfaced ~6 months later when an agent in a worktree without the pre-commit hook installed defaulted to treating `.hero/NEXT.md` / `.hero/next/<user>.md` as "session scratch" and proposed skipping them on commit — exactly the Layer 3 failure mode the missing backstop was meant to prevent. The user's pushback ("we did a whole spec on the Next system… why is this still happening?") was the audit trigger.

## Why this is the integrity-feature use case

- **The AC was unambiguous.** Not a soft "consider" or a discretionary deliverable. A `SHALL`.
- **The miss was small and late.** Last item on the Changes list, single line of agent-readable prose. The kind of thing a status-transition checklist would catch but a casual review wouldn't.
- **The cost compounded over time.** Every new clone or worktree without the hook installed inherited the gap. The fix is one line; the missed sessions are gone.
- **No human malice or neglect** — the substantial work *did* ship. This is the structural failure mode integrity checks exist for: bookkeeping drift between "AC defined" and "code merged."

## What a `spec-status-integrity` check would have caught

Concretely, at the moment someone proposed flipping the spec to `completed`, an integrity check could have:

1. Parsed AC bullets from the spec.
2. Cross-referenced the Changes section's named files against the diff that closed the spec.
3. Flagged: "spec lists `CLAUDE.md` in Changes but no diff to that file or to the template that generates it; AC #11 references CLAUDE.md guidance — verify before marking completed."

That's a graph + filesystem check, not an AI judgment call. Cheap. Catches this exact class of slip.

## Fix commit

Today's `da246f4 feat(install): instruct agents to include .hero/NEXT.md and .hero/next/*.md in commits` closed AC #11 by adding the rule to `internal/install/agents_md.go` so every harness picks it up on next install.
