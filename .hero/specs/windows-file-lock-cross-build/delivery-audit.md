# Delivery audit — windows-file-lock-cross-build

**Audited:** `git diff 3712247..HEAD`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Windows amd64 and arm64 targets compile — `.github/workflows/test.yml:31` builds the full `cmd/hero` command for both targets; supplied evidence reports both builds and the six-target snapshot passed.
- [✓] Attention locks remain exclusive and blocking — `internal/filelock/lock.go:17`, `internal/filelock/lock_unix.go:12`, and `internal/filelock/lock_windows.go:14` implement the blocking exclusive path; `TestAcquireBlocksUntilRelease` and `TestTryAcquireReportsCrossProcessContentionAndRelease` exercise blocking, cross-process exclusion, and release.
- [✓] Concurrent Attention updates remain correct — `TestStoreConcurrentReplaceDoesNotLoseUpdate` and `TestStoreConcurrentReceiptUpdatesPreserveState` assert the stale-revision and no-lost-update invariants; supplied race-suite evidence reports both passing.
- [✓] Code refresh keeps immediate busy semantics — `internal/cli/scan.go:406` exits before state loading when `TryAcquire` reports busy, and `TestIncrementalCodeRefreshSkipsUnusableCacheAndBusyLock` asserts the contention skip; the platform backends use `LOCK_NB` and `LOCKFILE_FAIL_IMMEDIATELY`.
- [✓] Failed acquisition closes the opened file — `internal/filelock/lock.go:39` routes post-open failures through `closeAfterFailure`; `TestAcquireClosesFileAfterLockFailure` injects a lock failure, preserves its cause, and asserts the opened file returns `os.ErrClosed`.
- [✓] Close unlocks and reports unlock/close errors — `internal/filelock/lock.go:65` unlocks before closing and reports either or both errors; `TestCloseReportsUnlockErrorAndStillClosesFile` and `TestCloseReportsFileCloseError` directly assert both failure paths.
- [✓] Platform details live behind build tags — OS lock calls are confined to `internal/filelock/lock_unix.go` and `internal/filelock/lock_windows.go`; portable Attention and CLI files call the shared package.
- [✓] Full release snapshot succeeds — supplied evidence reports `goreleaser release --snapshot --clean`, six archives, `checksums.txt`, successful checksum verification, and a runnable Darwin artifact.

## Changes
- [✓] Add shared platform file-lock package — `internal/filelock/lock.go`, `internal/filelock/lock_unix.go`, and `internal/filelock/lock_windows.go` provide a concrete acquire/try-acquire/close contract using the existing `x/sys` module.
- [✓] Migrate Attention locks — Focus, Mail, and Suggestion now delegate to `internal/filelock` and add store-specific error context.
- [✓] Migrate code-refresh try-lock — `internal/cli/scan.go:565` preserves the `(lock, busy, error)` result and cache-local lock path.
- [✓] Add lock regression tests — `internal/filelock/lock_test.go` covers blocking, cross-process busy, release/reacquire, open and post-open acquisition failure, unlock failure, and close failure; existing Attention and refresh contention tests cover consumer invariants.
- [✓] Add Windows cross-build gate and rehearse release — `.github/workflows/test.yml:31` builds both Windows architectures; the supplied exercise evidence covers the exact GoReleaser snapshot.

## Audit notes
- No performative rows, partial items, soft skips, or scope drift found.
