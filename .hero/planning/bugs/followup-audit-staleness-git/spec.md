---
title: "Delivery-audit staleness misfires after checkout or clone because it compares file mtimes"
slug: followup-audit-staleness-git
type: bug
status: delivering
priority: P2
severity: moderate
root_cause_class: design
domain: engineering
size: small
created: 2026-10-07
tags: [verify, audit, gate2, git]
---

# Audit staleness: commit order beats mtimes

## Goal

Gate 2's staleness check (#14) should flag real post-audit changes, not mtime noise from `git checkout`, branch switches or clones, and should not apply to finished specs.

## Kickoff

`spec.FindAuditReport` marks a report stale when its file mtime predates the spec's. Checkouts and clones rewrite mtimes in arbitrary order, and `hero spec verify` itself rewrites finished specs after their audit. The fix:
- skip staleness for finished specs;
- when mtimes say stale and both files are committed and clean, decide by last-commit time.

Test with `go test ./internal/spec -run AuditStaleness`.

## Root Cause

`loadAuditReport` trusts `os.Stat` mtimes, which git does not preserve, and treats finished specs like in-flight ones.

## Fix

- `auditCutoff(s)` is zero for finished specs, so no staleness check applies to them.
- `committedAuditIsCurrent` runs only when mtimes say stale. If both files are committed and unmodified (`git status --porcelain` is empty), the audit is current unless the spec's last commit is newer than the audit's. Uncommitted or untracked files keep the mtime verdict, and non-git directories behave as before.

## Acceptance Criteria

- **AC-1:** WHEN both the spec and its audit are committed and unmodified, and the audit was committed at or after the spec, THE SYSTEM SHALL accept the audit even if the spec's mtime is newer.
- **AC-2:** IF the spec was changed after its audit (uncommitted edit, or committed later) THEN THE SYSTEM SHALL keep flagging the audit as stale.
- **AC-3:** WHEN the spec is finished THE SYSTEM SHALL not apply the staleness check.

## Changes

1. `internal/spec/audit.go`: `auditCutoff`, `committedAuditIsCurrent`, `lastCommitTime`.
2. `internal/spec/audit_git_test.go`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | AC-1: committed order wins over mtime | DONE | `TestAuditStalenessUsesCommitOrderForCommittedFiles`, checkout-style mtime bump. It fails on the old code with "flagged stale by mtime alone" |
| 2 | AC-2: real edits still stale | DONE | Same test, for an uncommitted edit and an edit committed after the audit |
| 3 | AC-3: finished specs exempt | DONE | Same test, with a completed spec newer than its audit |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | audit.go | DONE | Git runs only when mtimes say stale, so hot paths like `hero_work` keep their latency |
| 2 | test | DONE | Real temp git repo with fixed committer dates |

### Exercise-the-feature check

- [x] The test drives a real git repository through checkout-style mtime rewrites, uncommitted edits, later commits and completion. The existing #14 tests (non-git directories) still pass.

### Excellence Bar self-check

- [x] Yes. It fixes the false positives without weakening real staleness detection.
