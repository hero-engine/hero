# Delivery audit — symlinked-hero-dir-walks-nothing

**Audited:** `git diff fbf1ad31..98002cc0` (commits 7296acae, 4da8ef83, ccc18beb, 37a35fd1, 98002cc0), with focus on `git diff 37a35fd1..98002cc0`. Worktree checked out at 98002cc0.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 Discover through a symlinked hero folder, `Archived` stamped — `internal/spec/spec.go:1254` uses `fsutil.Walk`; `TestDiscoverFollowsSymlinkedHeroDir` (earlier auditor falsified it against the old walker). Suite passes.
- [✓] AC-2 watch snapshot / spec-source validation / last-touched — `internal/watch/watch.go:69`, `internal/embeddings/chunker.go:90`, `internal/serve/projectpage/data/identity.go:137` all call `fsutil`; tests `TestScan_followsSymlinkedHeroDir`, `TestValidateSpecSourcesFollowsSymlinkedHeroDir`, `TestLastTouchedAtFollowsSymlinkedHeroDir`. Pass.
- [✓] AC-3 plain or broken-link root behaves like `filepath.Walk` — `resolveRoot` returns `(root,false)` for plain dirs, for a failed `Stat`, and for a failed `EvalSymlinks`; `TestWalkPlainRootUnchanged`, `TestWalkBrokenSymlinkRootReportsLikeFilepathWalk`. One narrow change from 37a35fd1: see note 5 (symlink to a file).
- [✓] AC-4 `hero list` end-to-end — the ledger says it was run with a built binary. Not re-run in this audit; the Discover tests cover the same path.
- [✓] AC-5 nested symlinked `.hero` listed; link to a file or dangling link not listed — `internal/install/satellite_detect.go:198-244`; `TestFindNestedHeroDirsFollowsSymlinkedHero`.
- [✓] AC-6 migration refuses a linked nested `.hero` and continues with the others — `internal/install/satellite_migrate.go:66` guard (`!info.IsDir() && mode&(Symlink|Irregular)`), `ErrLinkedNestedHero`; `TestMigrationRefusesLinkedNestedHero`, `TestMigrateNestedApplySkipsLinkedHero`. Pass.
- [✓] AC-7 Windows junctions treated like symlinks; non-link reparse directories stay plain — implemented at `internal/fsutil/walk.go:50-70` and `internal/install/satellite_detect.go:235`, with the guard at `satellite_migrate.go:66`. Evidence is **stub tests on darwin plus reading the stdlib**, not a Windows run (see notes 1 and 2).

## Changes
- [✓] 1 `internal/fsutil/walk.go` + `walk_test.go` — new package with `Walk` and `WalkDir`.
- [✓] 2 `spec.Discover` + `discover_symlink_test.go` — present in the diff.
- [✓] 3 `watch.Scan` + `watch_test.go` — present in the diff.
- [✓] 4 `validateSpecSources` + `chunker_test.go` — present in the diff.
- [✓] 5 `lastTouchedAt` + `identity_test.go` — present in the diff.
- [✓] 6 `FindNestedHeroDirs` / `isSymlinkToDir` + test — present, and `isSymlinkToDir` now accepts `ModeIrregular`.
- [✓] 7 Migration guard (`ErrLinkedNestedHero`, `PlanMigration`, CLI skip-and-continue) + tests.
- [✓] 8 Junctions — `resolveRoot` with swappable `lstat`/`stat`/`readlink`/`evalSymlinks`; linked iff `Lstat` is not a dir AND mode has Symlink|Irregular AND `Stat` is a dir; symlinks go through `EvalSymlinks`, junctions through `Readlink` then `EvalSymlinks`. The migration guard and `isSymlinkToDir` are updated, with tests.

## Open items (if any)
- None. The Windows-only part of AC-7 was not executed on Windows. The user chose to verify it by reading because there is no Windows runner. That is a recorded, concrete, user-accepted limitation, not a skip.

## Audit notes
Executed in this audit, on darwin/arm64 with Go 1.27.1:
- `go vet ./...`: clean.
- `GOOS=windows go vet ./internal/fsutil ./internal/install`: clean.
- `GOOS=windows go build ./...`: clean.
- `go test ./internal/fsutil ./internal/spec ./internal/watch ./internal/embeddings ./internal/serve/... ./internal/install ./internal/cli -count=1`: all ok. `internal/cli` took 85s.
- `go test -race ./internal/fsutil`: ok. The stub vars are package globals, but no test uses `t.Parallel`.
- Falsification 1: I restored the 37a35fd1 `walk.go` (plus the four stub vars, and `os.Lstat` replaced by `lstat` so the new tests compile). `TestWalkDescendsJunctionRoot` FAILS with "resolveRoot(junction) = ..., not linked". I then restored 98002cc0 and it passes.
- Falsification 2: I restored the 37a35fd1 `satellite_detect.go`. `TestIsSymlinkToDirRecognisesJunction` FAILS with "junction to dir: isSymlinkToDir = false, want true". I then restored 98002cc0 and it passes.
- The worktree is clean apart from this report.

Verified only by reading (nothing was run on Windows). The stdlib is Go 1.27.1, `$GOROOT/src`:
1. **Lstat on a junction.** `os/types_windows.go` `fileStat.mode()`. The tag `IO_REPARSE_TAG_MOUNT_POINT` has the name-surrogate bit (0x20000000), so `isReparseTagNameSurrogate()` is true and `ModeDir` is not set. In the reparse-tag switch only `SYMLINK`, `AF_UNIX` and `DEDUP` are special-cased; mount points hit `default` and get `ModeIrregular`. So a junction is `ModeIrregular` without `ModeDir`, as the spec says. A real directory with a non-surrogate tag (such as a OneDrive placeholder) keeps `ModeDir` and also gets `ModeIrregular`. The `info.IsDir()` guard in `resolveRoot`, `isSymlinkToDir` and the migration guard excludes that case.
2. **Readlink on a junction.** `file_windows.go` `readReparseLinkHandle` handles `IO_REPARSE_TAG_MOUNT_POINT` and passes the path through `normaliseLinkPath`. That strips the `\??\` prefix: `\??\C:\x` becomes `C:\x`, and `\??\UNC\s\x` becomes `\\s\x`. A `\??\Volume{guid}\` target becomes `\\?\Volume{guid}\...` (default godebug). That form is also usable, because `EvalSymlinks` then fails and `resolveRoot` falls back to it unchanged. Junction targets are always absolute, so no relative-path handling is needed.
3. **EvalSymlinks and junctions.** `path/filepath/symlink.go` `walkSymlinks` follows only `ModeSymlink`. `symlink_windows.go` adds only case normalisation. A junction is therefore returned unresolved, which is why the junction branch uses `Readlink`.
4. **WalkDir DirEntry for a junction.** `os/dir_windows.go` `dirEntry.Type()` is `de.fs.Mode().Type()`. Directory entries carry the reparse tag (`EaSize`/`Reserved0` in `newFileStatFromWin32finddata`), so a junction's entry reports `ModeIrregular` without `ModeDir`. That matches what `isSymlinkToDir` tests. When `winsymlink=0`, `modePreGo1_23` reports junctions as `ModeSymlink`. That goes through the symlink branch and `EvalSymlinks` (which won't follow a junction), so it falls back to "not linked". That is the pre-1.23 compatibility mode and is contrived. It is not a blocker.

Darwin and linux can never take the `ModeIrregular` branch. The only non-Windows stdlib setter is `os/stat_solaris.go:42` (unknown file types such as doors). No `.hero` takes that form. Behaviour on darwin and linux differs from 37a35fd1 only as in note 5.

Notes (non-blocking):
- **Note 5, symlink to a file.** A symlink root that points to a file used to count as linked, because 37a35fd1 only checked `EvalSymlinks`. The walk then visited the target file under the root's name. Now `Stat` must report a directory, so it falls through to plain `filepath.Walk(root)`, which reports the link itself. That matches AC-3 ("behave exactly like `filepath.Walk`") and is more correct. No hero-dir caller is affected.
- **Note 6, a junction to another junction or symlink.** `resolveRoot` runs `EvalSymlinks` on the `Readlink` result. That resolves a symlink target but not a junction target. Walking the first level still works, because the OS resolves the path. This is contrived.
- **Note 7, stub fidelity.** The stub tests assert what the stdlib *reads* as. They prove the logic is wired to `ModeIrregular`/`Readlink`, not that Windows returns those values. Reading the stdlib (notes 1-4) covers that gap.
- **Note 8, log churn.** `go-tests.log` and `vet.log` are committed in the planning dir and `go-tests.log` changed by 212 lines in this delta. This is churn only, not a defect.
- **Note 9, comment wording.** The `Walk` doc comment in `walk.go` runs past the usual wrap width. This is a style nit.

No defects found. Surface is `noteworthy` only because the Windows part is verified by reading and stubs, not by execution.
