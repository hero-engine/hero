# Delivery audit — eventslog-churn-fix-tracked-not-dirty

**Audited:** `git diff -- internal/cli/hook.go internal/cli/next_hooks.go internal/cli/next_hooks_test.go internal/cli/hooks_staging_integration_test.go` (working tree, uncommitted)
**Verdict:** SHIP
**Surface:** noteworthy
**Method:** cold — verified against code on disk, not the ledger. `go build ./...`, `go vet ./internal/cli/...`, `go test ./internal/cli/...` all run here and green (cli suite `ok`, 14.5s).

## Acceptance criteria
- [✓] Commit no longer appends post-commit event to events.log — `hook.go:94` `case "post-commit":` now contains only the `writeCheckpoint()` refresh; the `sha := gitRevParse("HEAD")` + `hooks.LogEvent(...post-commit...)` block is deleted (diff confirms 5-line removal). Guard test `TestPostCommitHook_DoesNotAppendEventsLog` (hooks_staging_integration_test.go:89) asserts events.log is not created after running the post-commit hook.
- [✓] `writeCheckpoint()` preserved — `hook.go:99` `_, _ = writeCheckpoint()` plus its comment present and unchanged in diff.
- [✓] Pre-commit stages events.log with the handoff files — `.hero/events.log` added to `handoffFilePaths` (next_hooks.go:53); staging loop at next_hooks.go:338 joins the same slice. Integration test `TestIntegration_DefaultInstall_StagesHandoffFiles` (next_hooks_test/hooks_staging_integration_test.go:157) now asserts `.hero/events.log` in the committed tree.
- [✓] events.log declared `merge=union` (single-source) — `updateGitAttributes` (next_hooks.go:618) iterates `handoffFilePaths`; not hand-edited. `TestUpdateGitAttributes_BindsAllFourPathsToUnion` adds `.hero/events.log` to its expected union set (next_hooks_test.go:35).
- [✓] events.log stays tracked (no gitignore) — no `.gitignore` change in diff anywhere; no `.gitignore` in repo references events.log. Boundary honored.
- [✓] pathspec + gitattributes stay in sync from single source, tests updated — both derived sites (staging loop next_hooks.go:338, gitattributes next_hooks.go:618) read `handoffFilePaths` and were NOT hand-edited; only the slice entry was added. `TestHandoffFileList_SingleSourceOfTruth` (next_hooks_test.go:181) iterates the slice and asserts every path appears in BOTH the staging body and the gitattributes block — genuinely covers the new entry.

## Changes
- [✓] 1. `hook.go` — remove post-commit append, keep writeCheckpoint — diff: 5 lines deleted (`sha` local + 4-line `LogEvent`); checkpoint + comment intact. Other hook cases (post-checkout:71, post-merge:82, prepare-commit-msg:101, pre-commit:106) untouched — verified by inspection; `eventsLogPath` still consumed by post-checkout (74) and post-merge (86), so no dangling reference.
- [✓] 2. `next_hooks.go` — `.hero/events.log` added as final slice entry (line 53) with a doc-comment note (lines 43-46) explaining it is append-only JSONL, union-safe because readers re-sort. Slice not otherwise widened.
- [✓] 3. `next_hooks_test.go` — union assertion + single-source test now cover events.log.
- [✓] 4. `hooks_staging_integration_test.go` — new Prong-1 guard test + `writeHandoffFiles` now writes an events.log fixture + default-install staging assert includes `.hero/events.log`.
- [✓] 5. Full internal/cli suite green — reproduced here: `go test ./internal/cli/...` → `ok`.

## Open items
- Disclosed deviation (engineer, ledger): no dedicated two-branch merge test. — SKIPPED — reason: `merge=union` is a git built-in; the code's only responsibility is emitting the `.gitattributes merge=union` directive, which IS test-covered (union test + single-source test). — Assessment: **concrete and acceptable.** A two-branch merge test would exercise git's own union driver, not this code. Not a gap.

## Audit notes
- **`gitRevParse` is now dead code.** After the Prong-1 deletion, `gitRevParse` (defined hook.go:209) has **zero callers** repo-wide (`grep -rn gitRevParse --include=*.go` returns only the definition). It compiles because Go does not flag unused package-level functions. The spec explicitly anticipated this and ruled removal out of scope ("if it is unused after this edit, that is out of scope to remove unless it produces a compile error"). So this is spec-compliant, not a defect — but it is a genuine residual (a newly-orphaned helper) worth a follow-up scrub. Flagged because the audit request specifically asked whether `gitRevParse` is still referenced: it is not.
- Scope is tight: exactly the four named files changed (plus the untracked spec dir). No drift into unrelated code.
- `internal/tracking/tracking.go` (`AppendEvent`) untouched — confirmed clean via `git status`.
- No test anywhere still asserts the old "post-commit writes an event" behavior; the only surviving `post-commit`+event references are the new guard test's comments/message, which assert the negative.
- Churn-eliminated claim verified by **code inspection + the passing unit guard test** (post-commit hook run → events.log not created), which is a more targeted proof than the engineer's /tmp scratch session. Did not re-run a scratch repo; the guard test supersedes it.
- Stale-but-harmless: test name `TestUpdateGitAttributes_BindsAllFourPathsToUnion` still says "Four" though the slice now holds six paths. Cosmetic; not a correctness issue.
