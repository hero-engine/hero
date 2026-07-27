---
title: "Unix-only file locks break Windows release builds"
slug: windows-file-lock-cross-build
type: bug
status: delivering
domain: engineering
size: medium
priority: critical
severity: high
root_cause_class: code
created: 2026-07-27
tags: [release, windows, file-locking, attention, code-index]
delivery_method: manual
---

# Unix-only file locks break Windows release builds

## Summary

### Categorization

| Attribute | Assessment |
|---|---|
| **Criticality** | high — every tagged release builds Windows artifacts, and the release pipeline cannot complete. |
| **Ease of Fix** | moderate — four lock call sites need one portable contract without weakening serialization or busy-skip behavior. |
| **Caused by our codebase?** | Yes — portable packages directly import Unix-only lock APIs. |
| **Needs more research?** | No — the exact GoReleaser failure and every remaining direct file-lock call site are confirmed. |

### Background

The `v0.30.0` release rehearsal passed `go mod tidy` and `go test ./...`, then
failed while cross-compiling `windows/arm64`. The first compiler error came from
Focus, but Mail and Suggestions duplicate the same `syscall.Flock` pattern.
The code-index refresh lock also imports `golang.org/x/sys/unix` from the
portable `internal/cli` package, which would become the next Windows failure
after Attention compiles.

### Root Cause

Platform-specific operating-system calls live in files with no build
constraints. Go therefore compiles Unix-only symbols for Windows. The release
pipeline is the first gate that compiles the whole command for every supported
target; native macOS tests cannot detect the defect.

### Fix Direction

Factor a small internal file-lock primitive with build-tagged Unix and Windows
backends. Reuse it in Focus, Mail, Suggestions, and the code-refresh lock.
Preserve blocking exclusive locks for state mutations and nonblocking
try-lock/busy semantics for hook-driven code refresh.

## Kickoff

Makes Hero's existing file locks portable so the six-platform release build
can complete without weakening Attention serialization or hook busy-skip.

**Status:** delivering — portable locks and the CI cross-build gate are implemented; all release rehearsal gates pass.

**Pick up at:** cold-audit the delivery evidence, address any findings, then run
`hero spec verify windows-file-lock-cross-build`.

→ `.hero/planning/bugs/windows-file-lock-cross-build/spec.md`

**Files:** `internal/attention/focus/lock.go`, `internal/attention/mail/lock.go`, `internal/attention/suggestion/store.go`, `internal/cli/scan.go`, `.goreleaser.yaml`
**Skip:** disabling Windows artifacts or replacing locks with process-local mutexes.

## Problem Statement

Reproduction from clean `main` at `3712247`:

```text
$ goreleaser release --snapshot --clean
...
build failed: # github.com/hero-engine/hero/internal/attention/focus
internal/attention/focus/lock.go:22:20: undefined: syscall.Flock
internal/attention/focus/lock.go:22:47: undefined: syscall.LOCK_EX
internal/attention/focus/lock.go:30:23: undefined: syscall.Flock
internal/attention/focus/lock.go:30:55: undefined: syscall.LOCK_UN
target=windows_arm64_v8.0
```

Static sweep confirms the same unportable pattern in:

- `internal/attention/focus/lock.go`
- `internal/attention/mail/lock.go`
- `internal/attention/suggestion/store.go`
- `internal/cli/scan.go`

The first package failure masks the later ones. A fix that addresses only Focus
would still leave the tagged release red.

## Environment Details

- Host: Darwin arm64, Go 1.26.5, GoReleaser 2.15.2.
- Release targets: Darwin, Linux, and Windows on amd64 and arm64.
- Native full tests pass because Darwin provides `Flock`.
- GoReleaser runs tests natively before cross-compiling all six artifacts.

## Root Cause Analysis

The Attention stores use lock files to serialize read-modify-write operations
across processes. Their lock types are package-local duplicates backed directly
by `syscall.Flock`. `syscall.Flock`, `LOCK_EX`, and `LOCK_UN` are not defined by
the Windows standard-library target.

The incremental code-index coordinator has a different but related contract: it
uses `unix.Flock(... LOCK_EX|LOCK_NB)` to skip silently when another refresh
owns the lock. Importing `x/sys/unix` in untagged `internal/cli/scan.go` is also
not portable to Windows.

The fundamental defect is not an unavailable dependency or CI environment. It
is a code-boundary error: platform APIs were placed directly in portable
packages instead of behind build-tagged implementations.

## Code Flow (End to End)

1. A `v*` tag triggers `.github/workflows/release.yml`.
2. GoReleaser runs `go mod tidy` and `go test ./...` on Linux.
3. GoReleaser cross-compiles `./cmd/hero` for six targets from
   `.goreleaser.yaml`.
4. Windows compilation traverses the CLI and Attention dependency graph.
5. Untagged Focus, Mail, Suggestion, and code-refresh lock code references
   Unix-only symbols, so compilation stops before archives or checksums exist.

At runtime, the same locks protect:

1. Focus create/replace operations in `internal/attention/focus/store.go`.
2. Mail delivery and receipt updates in `internal/attention/mail/store.go`.
3. Suggestion create/action/cleanup operations in
   `internal/attention/suggestion/store.go`.
4. Incremental graph/index/embedding refresh in `internal/cli/scan.go`.

## Key Files

| File | Relevance |
|---|---|
| `internal/attention/focus/lock.go` | Blocking exclusive lock; first Windows compiler failure. |
| `internal/attention/mail/lock.go` | Duplicate blocking exclusive lock. |
| `internal/attention/suggestion/store.go` | Third embedded blocking exclusive lock. |
| `internal/cli/scan.go` | Nonblocking code-refresh try-lock and direct Unix import. |
| `internal/attention/focus/store_test.go` | Existing concurrent replace invariant. |
| `internal/attention/mail/store_test.go` | Existing concurrent receipt-update invariant. |
| `.goreleaser.yaml` | Authoritative six-target artifact gate. |

## Secondary Defects

The release workflow's native test job cannot detect target-specific compile
breakage before a tag because it runs only on Linux. This fix adds a portable
cross-build regression gate in repository tests; changing CI workflow policy is
not required to unblock this release.

## Goal

Produce the same blocking and nonblocking file-lock behavior on Unix and
Windows, remove every portable-package import of Unix-only lock APIs, and make
the complete GoReleaser snapshot succeed for all six release targets.

## Acceptance Criteria

- WHEN Hero builds for Windows amd64 or arm64 THE SYSTEM SHALL compile Focus,
   Mail, Suggestions, code-index refresh, and `cmd/hero` without Unix-only
   symbol errors.
- WHEN Focus, Mail, or Suggestions acquires a state lock THE SYSTEM SHALL hold
   an exclusive cross-process lock until `Close`, preserving the existing
   blocking serialization contract on Unix and Windows.
- WHEN an Attention state lock is contended THE SYSTEM SHALL preserve the existing no-lost-update and stale-revision behavior.
- WHEN code-index refresh finds its lock already held THE SYSTEM SHALL return
   the existing busy result immediately without mutating refresh state on Unix
   and Windows.
- WHEN a lock acquisition fails THE SYSTEM SHALL close the opened file and
   return the underlying failure without reporting ownership.
- WHEN a held lock closes THE SYSTEM SHALL attempt to unlock before closing
   the file and SHALL return a meaningful unlock or close error.
- THE SYSTEM SHALL centralize operating-system-specific file locking behind
   build-tagged implementations and SHALL leave no direct `syscall.Flock` or
   `x/sys/unix` lock dependency in portable Attention or CLI files.
- WHEN the release snapshot runs THE SYSTEM SHALL pass native tests and build
   archives plus checksums for all configured Darwin, Linux, and Windows
   targets.

## Changes

1. Add a small internal file-lock package with a common acquire/try-acquire/
   close contract and build-tagged Unix and Windows implementations.
   - Keep the API concrete and narrow; no interface or new external dependency.
   - Use the existing `golang.org/x/sys` module for platform calls.
2. Replace the duplicated Focus, Mail, and Suggestion lock implementations with
   the shared blocking exclusive lock while preserving package-level error
   context.
3. Replace `internal/cli/scan.go`'s direct Unix try-lock with the shared
   nonblocking path while preserving the `(lock, busy, error)` behavior.
4. Add focused lock regression tests for exclusive contention, try-lock busy
   behavior, release/reacquire, and failure cleanup where observable.
5. Add a Windows cross-build test or equivalent deterministic gate covering the
   full `cmd/hero` target, then run the exact GoReleaser snapshot rehearsal.

## Boundaries

- Do not remove Windows from the release matrix.
- Do not replace cross-process locks with `sync.Mutex`.
- Do not redesign Attention storage, revisions, or code-index orchestration.
- Do not change state locations, lock filenames, or public CLI/MCP contracts.
- Do not add a third-party locking module when `x/sys` already supplies the
  platform primitives.

## Risks

- Windows `LockFileEx` busy errors must map only true lock contention to the
  nonblocking busy result; other errors must remain failures.
- Refactoring three independent Attention locks can accidentally erase their
  useful operation-specific error context.
- Unlock and close both may fail; error precedence must be deterministic and
  must not leak the file handle.
- A compile-only Windows test proves target compatibility but cannot prove
  Windows runtime contention on a Darwin host. The platform implementation
  should stay thin, with shared contract tests plus the release cross-build.

## Validation

- Run focused tests for the new lock package and all three Attention stores.
- Run existing code-refresh lock and hook integration tests.
- Run `go test -race -count=1 ./...`.
- Run `GOOS=windows GOARCH=amd64 go build ./cmd/hero` and the arm64 equivalent.
- Run `goreleaser release --snapshot --clean` and verify six archives plus
  `checksums.txt`.
- Run `hero docs check`, `hero drift windows-file-lock-cross-build`, and
  `git diff --check`.

## Completion Ledger

Implemented one concrete cross-platform lock primitive with build-tagged Unix
and Windows backends. Focus, Mail, Suggestions, and incremental code refresh
now reuse it without changing storage paths or public contracts. Validation
passed: focused lock/Attention tests, the complete race-enabled suite, both
Windows command builds, and the exact six-target GoReleaser snapshot.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Windows targets compile | DONE | `lock_windows.go` uses `LockFileEx`; full `cmd/hero` builds produced valid Windows amd64 and arm64 PE32+ executables. |
| 2 | Attention locks remain exclusive and blocking | DONE | `internal/filelock/lock.go`, platform backends, and `TestAcquireBlocksUntilRelease` preserve blocking cross-process exclusivity. |
| 3 | Concurrent Attention updates remain correct | DONE | Existing Focus concurrent replace and Mail concurrent receipt tests pass in `go test -race -count=1 ./...`. |
| 4 | Code refresh keeps immediate busy semantics | DONE | `TryAcquire` maps only platform contention to `busy`; cross-process helper coverage and existing code-refresh contention tests pass. |
| 5 | Failed acquire closes file | DONE | `internal/filelock/lock.go` funnels permission/lock failures through `closeAfterFailure`; open failure propagation has regression coverage. |
| 6 | Close unlocks and reports errors | DONE | `Lock.Close` unlocks before close and preserves either or both errors with `errors.Join`; release/reacquire tests pass. |
| 7 | Platform details live behind build tags | DONE | OS calls exist only in `lock_unix.go` and `lock_windows.go`; portable Attention/CLI files contain no direct Flock/unix imports. |
| 8 | Full release snapshot succeeds | DONE | `goreleaser release --snapshot --clean` passed, creating six archives and `checksums.txt`; every checksum verified. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Add shared platform file-lock package | DONE | Added `internal/filelock/lock.go`, `lock_unix.go`, and `lock_windows.go` with blocking and try-acquire operations. |
| 2 | Migrate Attention locks | DONE | Focus, Mail, and Suggestion wrappers now reuse `internal/filelock` and retain operation-specific error context. |
| 3 | Migrate code-refresh try-lock | DONE | `internal/cli/scan.go` uses `TryAcquire` while preserving `(lock, busy, error)` and the cache-local path. |
| 4 | Add lock regression tests | DONE | Added blocking, cross-process busy, release/reacquire, and open-failure tests; existing store/refresh contention tests remain green. |
| 5 | Add Windows cross-build gate and rehearse release | DONE | `.github/workflows/test.yml` now builds both Windows architectures; local Windows builds and the complete snapshot passed. |

### Exercise-the-feature check

- [x] Ran `goreleaser release --snapshot --clean`; inspected Mach-O, ELF, and PE32+ binaries, ran the Darwin artifact's `--version`, and verified all six archive checksums.

### Excellence Bar self-check

Yes — the change removes duplicated platform leakage, preserves concurrency
semantics with direct tests, and moves Windows compatibility into ordinary CI.

## Notes

Load-bearing claims were read from the current source and reproduced against the
actual release command. No tracker evidence or external assumptions are
involved.

## Recap

Hero's new persistence and index-refresh locks work on Unix but are compiled
from portable packages, breaking every Windows release artifact. One
build-tagged lock seam plus a full-command Windows cross-build gate fixes the
root cause and prevents another masked target failure.
