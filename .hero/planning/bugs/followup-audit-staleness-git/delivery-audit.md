# Delivery audit — followup-audit-staleness-git

**Audited:** `git show 27add446` (worktree at 27add446)
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: committed + clean, audit committed at/after spec → accepted despite newer spec mtime — `internal/spec/audit.go:109` (`committedAuditIsCurrent` only consulted when mtimes say stale), `audit.go:244-255`; `TestAuditStalenessUsesCommitOrderForCommittedFiles` first block (passes).
- [~] AC-2: spec changed after its audit stays stale — uncommitted edit and edit committed in a later commit are both covered by the test and pass. **Gap:** a spec edited after its audit and then committed *in the same commit* as the audit is accepted (commit times equal, `auditAt >= specAt`). Reproduced with a scratch probe: mtime-only verdict `stale=true`; after `git add -A && git commit` the same files report `found=true stale=false`. That is a spec changed after its audit that is no longer flagged.
- [~] AC-3: finished specs exempt — implemented (`auditCutoff`, `audit.go:232-237`) and tested. **Gap:** the exemption is not limited to specs `hero spec verify` has already handled. `verify` returns early only for specs that are `completed` *and archived* (`internal/cli/verify.go:113`); a spec hand-flipped to `status: completed` while still in `.hero/planning/` runs Gate 2, and the exemption now skips staleness. Probe: a completed spec in planning with an uncommitted edit made after the audit reports `found=true stale=false`. The same weakening applies to the `autoArchiveIfCompletedOpt` guard (`internal/cli/complete.go:226`), which exists to catch agents that edit `status: completed` directly. Previously a stale report made `Found=false` and blocked the archive.

## Changes
- [✓] `internal/spec/audit.go`: `auditCutoff`, `committedAuditIsCurrent`, `lastCommitTime` — present as described. Git runs only on the mtime-stale path.
- [✓] `internal/spec/audit_git_test.go` — real temp repo, fixed committer dates, asserts on `Found`/`Stale`.

## Open items
- None in the ledger. All rows are DONE.

## Audit notes
- Behaviour checked against the questions asked:
  - The git check runs only when mtimes say stale (`audit.go:109`). It can only turn stale into current, never the reverse.
  - Uncommitted spec edit: stays stale, because porcelain output is non-empty.
  - Spec committed after the audit: stays stale.
  - Untracked audit: porcelain shows `??`, so the mtime verdict stands.
  - Gitignored audit: porcelain is empty, but `git log` output is empty, the parse fails and the mtime verdict stands.
  - Non-git directory: `git status` errors and the mtime verdict stands.
  - Subdirectories: `git -C <spec dir>` with absolute paths works. A relative `s.Path` would be resolved relative to the spec dir and fall back to the mtime verdict (fail-safe, not wrong).
- Truly stale audits that are now accepted: (1) same-commit edit-after-audit; (2) any finished-but-unarchived spec (hand-flipped `completed`, or `superseded`) going through `hero spec verify` Gate 2 or the auto-archive guard. Suggested direction, not prescribed: scope the finished exemption to specs already under `.hero/specs/`, or to specs whose status flip was made by verify. Decide explicitly whether same-commit should fall back to the mtime verdict.
- Commit time (`%ct`) is wall-clock, not topology. Rebase, cherry-pick or clock skew can reorder. Low risk.
- Every mtime-stale spec now spawns up to three `git` processes per `FindAuditReport` call. After a fresh clone that can be many specs on `hero_work`/workmodel reads. Not measured.
- Existing #14 tests pass: `TestFindAuditReport_Stale` and `TestFindAuditReport_NotStaleWhenReportIsNewer`, plus all other `FindAuditReport*` tests. `go test ./internal/spec -count=1 -race` passes.
