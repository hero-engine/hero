# Delivery audit — followup-audit-staleness-git

**Audited:** `git diff 27add446...3925c865` on top of `git show 27add446` (clean detached worktree at 3925c865), round 2
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: committed + clean, audit committed at/after spec → accepted despite newer spec mtime — `committedAuditIsCurrent` (`internal/spec/audit.go`), consulted only when mtimes say stale. `TestAuditStalenessUsesCommitOrderForCommittedFiles` passes.
- [✓] AC-2: spec changed after its audit stays stale — the uncommitted-edit and committed-later cases are tested and pass. The same-commit pair is now an explicit, documented decision: the Fix section, the code comment on `committedAuditIsCurrent`, and the "committed together" test case. AC-2 as written covers "uncommitted edit, or committed later", which is satisfied.
- [✓] AC-3: archived specs exempt, completed-in-planning keeps the check — `auditCutoff` now requires `IsFinished()` and a path containing `/.hero/specs/`. Tests: `TestArchivedSpecIsExemptFromStaleness`, and the hand-flipped completed case in `planning/`, which stays stale. The auditor's round-1 probe, re-run at 3925c865:
  - hand-flipped `completed` in planning with an uncommitted edit after the audit: `found=false stale=true` (was `found=true stale=false` at 27add446);
  - `superseded` in planning, same edit: `stale=true`;
  - archived completed spec with an absolute path: `found=true stale=false`.

## Changes
- [✓] `internal/spec/audit.go`: `auditCutoff` (archived-only exemption), `committedAuditIsCurrent` (same-commit decision documented), `lastCommitTime`.
- [✓] `internal/spec/audit_git_test.go`: hand-flipped, same-commit and archived cases. All assert on `Found`/`Stale`.

## Open items
- None. All ledger rows are DONE.

## Audit notes
- Known accepted trade-off (spec decision, not a defect): if a spec is edited after its audit and both are then committed in the same commit, the audit is accepted. The re-run probe confirms `found=true stale=false`. Users should know Gate 2 cannot catch that ordering once it is committed.
- The archived check matches the literal `/.hero/specs/`. A relative `s.Path`, or a workspace whose configured `Folder` is not `.hero` (`config.HeroDir`), gets no exemption. The probe with a relative archived path gave `stale=true`. This fails safe: `hero spec verify` returns early for archived specs anyway. The only effect would be a stale-looking audit in workmodel reads. `spec.Discover` is given absolute paths in practice (`workspace.Locate` uses `filepath.Abs`).
- Existing #14 tests (`TestFindAuditReport_Stale`, `_NotStaleWhenReportIsNewer`) and all other `FindAuditReport*` tests pass. `go test ./internal/spec ./internal/serve ./internal/workmodel -count=1 -race` passes, which matches `go-tests.log`.
