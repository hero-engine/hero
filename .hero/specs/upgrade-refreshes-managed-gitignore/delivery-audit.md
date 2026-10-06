# Delivery audit — upgrade-refreshes-managed-gitignore

**Audited:** `git diff origin/main...021bc39c` (excluding `.hero/hero.json`)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: `hero upgrade` refreshes an existing managed block and keeps content outside the markers — `refreshManagedGitignoreIfPresent` (`internal/cli/init.go:561-572`) delegates to the existing `ensureManagedGitignoreBlock`; called on both completion paths of `runUpgrade` (`internal/cli/upgrade.go:210`, `internal/cli/upgrade.go:311`). `TestManagedGitignoreCoversMCPPidfilesAndUpgradeRefresh` asserts `user-entry` is preserved and `.hero/mcp-*.pid`, `.hero/mcp-*.pid.*`, `.hero/mcp-debug.log` are added to a stale block. Re-run by the auditor: PASS.
- [✓] AC-2: no managed block, or dry-run, writes nothing — same test asserts dry-run leaves bytes identical and a workspace without `.gitignore` gets none. The guard is in the helper (`!strings.Contains(..., gitignoreMarkerStart) || dryRun`).

## Changes
- [✓] `internal/cli/init.go`: `refreshManagedGitignoreIfPresent` — present, 13 lines.
- [✓] `internal/cli/upgrade.go`: called on both completion paths — two call sites, each beside the hook refresh, failures reported as a warning on stderr.
- [✓] `internal/cli/init_gitignore_test.go`: `TestManagedGitignoreCoversMCPPidfilesAndUpgradeRefresh` — present.

## Open items
- None.

## Audit notes
- No leftover duplicate pidfile logic: the diff's only `internal/` changes are the helper, the two upgrade call sites, and the test. No reaper, sweep, release, or watchdog code; mentions of pidfiles in the diff are in the spec text and comments only.
- The test drives the helper directly. The wiring into `runUpgrade` is verified by reading the diff, not by a test that runs `hero upgrade`. Low risk given the two-line call sites.
- The helper treats any `.gitignore` read error (not only "file missing") as "nothing to do" and returns nil, so an unreadable `.gitignore` is skipped silently instead of producing the warning. Minor; it does not affect either AC.
- Evidence: `go-tests.log` is the same full `go test ./...` run as the sibling spec in this commit (0 FAIL lines); `vet.log` empty. The auditor re-ran the targeted tests and `go vet` on `internal/cli`: pass.
