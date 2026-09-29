---
title: "Completion Ledger sign-off gate fails open — any note mentioning [signed-off] self-approves"
slug: ledger-signoff-substring-match-fails-open
type: bug
status: delivering
domain: engineering
priority: high
severity: high
root_cause_class: code
tags: [verify, ledger, gate, governance, parsing, fails-open]
created: 2026-07-25
---

# Completion Ledger sign-off gate fails open — any note mentioning `[signed-off]` self-approves

## Kickoff

Paste into a fresh session to start delivery:

> Deliver `ledger-signoff-substring-match-fails-open`. `hero spec verify`
> Gate 1 decides whether a SKIPPED/BLOCKED ledger row carries human sign-off
> with a bare substring test — `strings.Contains(noteLower, "[signed-off]")`
> at `internal/spec/ledger.go:216`. A note that *denies* sign-off
> ("[signed-off] NOT yet given", "needs [signed-off] from the owner") contains
> the literal marker and therefore grants it. The gate fails open: the agent
> writing the ledger can self-approve its own descope while appearing to
> escalate. Replace the substring test with anchored parsing that only honors
> the marker as a standalone token, and add regression tests for the
> negative-sentence cases. Start by reading **Key Files**, then work the
> Acceptance Criteria in order. Close with the cold delivery audit and
> `hero spec verify`.

## Summary

`[signed-off]` in a Completion Ledger note is the machine-readable token that
lets a `SKIPPED` or `BLOCKED` row pass Gate 1 — it is the entire mechanism by
which a human authorizes shipping incomplete work. It is detected by substring
containment, so the token grants sign-off regardless of the sentence around it.

A note written specifically to withhold sign-off grants it instead.

## Issue

Found by a cold delivery audit during the delivery of
`graph-why-resolution-and-peer-spec-indexing`. That delivery deliberately
deferred one acceptance criterion and wrote the ledger row as:

```
| 3b | … | SKIPPED | … **Needs user sign-off — [signed-off] NOT yet given** |
```

The intent was for Gate 1 to fail and route the descope to the user. Gate 1
reported `8/8 AC rows DONE` and passed. Only the auditor's independent HOLD
verdict stopped the spec flipping to `completed` with the descope never
surfacing. Had the audit returned SHIP, incomplete work would have shipped
under a note that said, in English, that it was not approved.

This is the same failure shape as the bug the same session was fixing
(`initiative-autocomplete-ignores-declared-children`): a governance gate that
reads as enforced while being trivially defeated.

## Root Cause

`internal/spec/ledger.go:214-218`:

```go
// Check for signed-off annotation
noteLower := strings.ToLower(row.Note)
if strings.Contains(noteLower, "[signed-off]") || strings.Contains(noteLower, "[signed off]") {
    row.SignedOff = true
}
```

Containment carries no notion of the surrounding clause. Every one of these
grants sign-off today:

- `[signed-off] NOT yet given`
- `needs [signed-off] before this can ship`
- `blocked pending [signed-off]`
- `do not mark [signed-off] until the owner reviews`

The failure is asymmetric and in the dangerous direction: the mistake a
careful author is *most likely* to make — naming the marker while explaining
that it is absent — is exactly the mistake that grants approval.

## Key Files

- `internal/spec/ledger.go:214-218` — the substring test.
- `internal/spec/ledger.go` — `LedgerRow.SignedOff` and the Gate 1 pass
  computation that consumes it.
- `internal/cli/verify.go` — `checkLedger`, Gate 1's caller.
- `internal/spec/ledger_test.go` — existing ledger parse tests; no
  negative-sentence case.
- `.claude/skills/completion-ledger/SKILL.md` — documents `[signed-off]` as
  "the machine-readable sign-off"; wording may need to state the anchoring
  rule once parsing is strict.

## Goal

`[signed-off]` grants sign-off only when the note asserts it. A note that
mentions the marker while denying, requesting, or deferring sign-off must not
pass Gate 1. When the gate is unsure, it must fail closed.

## Suggested Fix Approach

1. **Anchor the marker.** Honor it only as a standalone annotation — e.g. the
   note begins with it, or it is delimited such that no negating or
   conditional words attach to it. Simplest robust rule: the marker must be
   the note's leading token (optionally followed by attribution, e.g.
   `[signed-off] bwheeler — accepted the descope`).
2. **Fail closed on negation.** Independently of anchoring, reject the marker
   when it is preceded or followed by a negating/conditional cue (`not`,
   `no`, `without`, `needs`, `pending`, `until`, `before`, `awaiting`,
   `require`). Belt-and-suspenders against a phrasing anchoring alone lets by.
3. **Say why in the gate output.** When a row carries the marker but it was
   not honored, Gate 1 should say so explicitly rather than reporting a bare
   "not signed-off" — otherwise an author who thought they signed off has no
   idea why the gate disagrees.
4. Keep `[signed off]` (unhyphenated) working under the same rules.

## Acceptance Criteria

- **AC-1:** WHEN a ledger note carries `[signed-off]` as its leading
  annotation THE SYSTEM SHALL treat the row as signed off.
- **AC-2:** WHEN a ledger note mentions `[signed-off]` in a negating,
  requesting, or conditional clause THE SYSTEM SHALL NOT treat the row as
  signed off, and `hero spec verify` Gate 1 SHALL fail for a `SKIPPED` or
  `BLOCKED` row so annotated.
- **AC-3:** WHEN the marker is present but not honored THE SYSTEM SHALL state
  in the gate output that the marker was found but rejected, and why.
- **AC-4:** THE SYSTEM SHALL apply identical rules to `[signed off]`
  (unhyphenated) and SHALL remain case-insensitive.
- **AC-5:** THE SYSTEM SHALL preserve existing behavior for ledgers whose
  sign-off notes are already well-formed (regression guard over the existing
  ledger corpus).

## Boundaries

- **In scope:** sign-off detection, the Gate 1 message, the skill wording, and
  regression tests.
- **Out of scope:** changing what `SKIPPED`/`BLOCKED` mean, who may sign off,
  or adding an out-of-band approval channel (e.g. a signature or a tracker
  approval) — those are worth considering but are a separate design.

## Risks

- **Retroactively invalidating a real sign-off.** A previously-passing ledger
  whose note phrases the marker mid-sentence would start failing. Mitigation:
  AC-5's regression pass over the existing corpus; anchoring rules chosen to
  accept the common well-formed shapes.
- **Over-strict anchoring frustrating authors.** Mitigation: AC-3's explicit
  gate message tells the author exactly how to phrase it.

## Validation

| Test | Asserts |
|---|---|
| `[signed-off] bwheeler — accepted` → signed off | AC-1 |
| `[signed-off] NOT yet given` → NOT signed off | AC-2 (the reported case) |
| `needs [signed-off] before shipping` → NOT signed off | AC-2 |
| `awaiting [signed off] from owner` → NOT signed off | AC-2, AC-4 |
| Gate 1 output names the rejected marker | AC-3 |
| Existing ledgers in `.hero/specs/` still parse as before | AC-5 regression |

## Completion Ledger

### Acceptance Criteria

| # | Criterion | Status | Note |
|---|---|---|---|
| 1 | Leading `[signed-off]` is honored | DONE | `parseSignOff` in `internal/spec/ledger.go` honors only the structured leading form `[signed-off] <who> — <why>` or `[signed-off: <who>] <why>` (signer ≤ 3 words, no denial words). `TestParseSignOff` structured cases. |
| 2 | Negating/requesting/conditional mention is not honored; Gate 1 fails | DONE | Any non-leading marker, and any leading marker followed by prose instead of a dash-terminated signer, fails closed. `TestParseSignOff` covers the four spec examples plus all seven bypasses from the first audit's HOLD; `TestVerify_DeniedSignOffFailsGate` reproduces the reported row end-to-end through `hero spec verify`. |
| 3 | Gate output says the marker was found but rejected, and why | DONE | `LedgerRow.SignOffRejected` + `spec.SignOffForm` surfaced by `checkLedger` for both AC and Changes rows (`internal/cli/verify.go`); asserted by `TestVerify_DeniedSignOffFailsGate`. |
| 4 | Same rules for `[signed off]`, case-insensitive | DONE | Shared `parseSignOff`; cases `[SIGNED OFF] Brian Wheeler - …` and `[signed off: dave@example.com]`. |
| 5 | Well-formed existing ledgers unchanged | SKIPPED | [signed-off] David Christiansen — chose the structured-marker design (option 1) in chat on 2026-09-29 knowing it un-honors the 7 legacy free-form sign-offs in 4 completed, archived specs (cst-initiative-premature-autocomplete, interactive-cli-acceptance-and-merge-gate, spec-completion-loop, token-efficiency-pass). No runtime effect: `SignedOff` is read only by Gate 1, and `verify` exits before Gate 1 for archived completed specs (`internal/cli/verify.go:113`). A lexical rule that preserved them could not also reject the audit's bypasses. |

### Changes

| # | Change | Status | Note |
|---|---|---|---|
| 1 | Structured sign-off parsing | DONE | `internal/spec/ledger.go` |
| 2 | Gate 1 rejection message (AC and Changes rows) | DONE | `internal/cli/verify.go` |
| 3 | Skill documents the structured form | DONE | `core/skills/completion-ledger/SKILL.md` |
| 4 | Regression tests | DONE | `internal/spec/ledger_test.go`, `internal/cli/verify_test.go` (existing fixtures migrated to the structured form) |

### Exercise-the-feature check

- [x] Exercised: `go test ./...` green; `TestVerify_DeniedSignOffFailsGate` runs the real `hero spec verify` against the exact row from the bug report and it fails Gate 1 naming the rejection and the required form. Scratch corpus pass (deleted) confirmed which archived rows change.

Out of scope, noted for follow-up: the signer is self-declared text, so an agent can still write `[signed-off] <someone> — …` itself; binding sign-off to a real identity is the out-of-band approval design the spec's Boundaries exclude. Two archived specs put the marker in the Status cell (`SKIPPED [signed-off]`), which parses as UNKNOWN — pre-existing and unaffected.
