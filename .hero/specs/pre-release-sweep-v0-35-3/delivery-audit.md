# Delivery audit — pre-release-sweep-v0-35-3

**Audited:** `git diff 277e342d..8d5c20b3` (commits 12cb5290, e30d5080, ed729fa5, 012bd22d, 8d5c20b3), worktree at 8d5c20b3
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: `hero why --edges` starts from the promoted spec, using the same resolution as `hero why`. `internal/cli/brief.go:489-496` now calls `traversal.ResolveTarget` (`internal/traversal/why.go:114`), the same function `Why` uses (`why.go:90`). `TestWhyEdgesPromotedSpecWinsSlugTie` (`internal/cli/why_edges_test.go`) inserts the intake first and makes it the later-ingested node. **Falsified:** with the 277e342d `brief.go` restored, the test fails 3/3 runs with `"# Intake title `promoted` (Intake)"`.
- [✓] AC-2: `open_followups` uses the lane's completion time. `internal/workmodel/polish.go:78` calls `completionTime(done)`, defined at `polish.go:109-116`. `TestOpenFollowupsUseLaneCompletionFallback` (`polish_test.go:95`) covers a spec with no `completed_at` whose mtime is 48 h old. **Falsified:** with the 277e342d `polish.go` and `model.go` restored, it fails with "follow-up on an mtime-fallback recently_done item was not reported".
- [✓] AC-3: Archived classification no longer depends on the hero folder's name. `Discover` stamps `Archived` with a prefix check against `Join(heroDir,"specs")+sep` (`internal/spec/spec.go:1328-1331`). `auditCutoff` reads the flag (`internal/spec/audit.go:235`). `TestArchivedSpecIsExemptFromStaleness` (`audit_git_test.go:100`) uses the custom folder `work/` inside a repo under `planning/specs/`, with spec folders named `planning` and `specs`. **Falsified at 8d5c20b3:** with the old literal `strings.Contains(..., "/.hero/specs/")` restored, `y` and `planning` fail with `Stale:true` on all 3 runs (`-count=3`). With the real code, the test passes on all 3 runs. The fixture also defeats a looser `/specs/` substring heuristic, because the repo root contains `/specs/` and `z` must stay stale.
- [✓] AC-4: Each Fixes-item-4 test gap now has a test. `TestReadContractSweepGaps` (`internal/serve/mcp_tools_read_contract_test.go:291`) covers CRLF stripping, a knowledge note that is not a child, recorded pass/fail AC states and initiative `verify` null. `TestToolWorkWritesNothing` (`:243`) seeds and watches all six `WatchGlobs` (`mcp_tools_read_contract.go:230-237`). **Falsified:** dropping the `slugRank` filter in `workmodel.NewCorpus` (`model.go:137`) fails the sweep test with "a knowledge note pointing at an initiative must not be a child". Injecting a write into `toolWork` fails the no-write test for each injection: a `hero.json` rewrite, a `specs/done/spec.md` rewrite, an mtime-only touch of `NEXT.md`, and deletion of `next/chet.md`.
- [✓] AC-5: The first-audit notes are closed. `Lane` uses `completionTime` (`model.go:336`). `runWhyEdges` returns `ResolveTarget`'s error unchanged (`brief.go:492-494`), and `ResolveTarget` keeps the "no node" text only for `sql.ErrNoRows` (`why.go:129-133`). The no-write test compares content plus mtime and checks deletion. The sweep test checks `IsError` and the unmarshal error on every tool result, and asserts that `crlf` is a child, which shows the children list is computed. No test exercises the DB-error pass-through; the change is verified by reading the code (see notes).

## Changes
- [✓] 1. `ResolveTarget` exported and `why_federation_test.go` renamed accordingly. `runWhyEdges` uses it; new `why_edges_test.go`.
- [✓] 2. `completionTime` added in `polish.go`; `polish_test.go` test added.
- [✓] 3. `Spec.Archived` added with `json:"-"` (`spec.go:268-270`) and stamped by `Discover`. `auditCutoff` uses it. `audit_git_test.go` was rewritten. In 8d5c20b3 the fixture sets each audit's mtime explicitly instead of sleeping.
- [✓] 4. `TestReadContractSweepGaps` added; `TestToolWorkWritesNothing` covers all six globs. `snapshotTree` now records file content, not `rune(size)`.
- [✓] 5. `Lane` uses `completionTime`; `--edges` passes errors through; serve tests strengthened.

## AC-3 design checks
- **Production callers of `spec.FindAuditReport`:**
  - `cli/verify.go:316` (`checkAudit`): the spec comes from `Discover(cfg.HeroDir(...))` (`verify.go:89`). An archived completed spec returns early at `verify.go:113`.
  - `cli/complete.go:226`: reached only when `!isAlreadyInSpecsDir`, so only for unarchived specs. The spec is hand-parsed, which fails safe.
  - `workmodel/model.go:389` (`VerifyOf`) and `workmodel/revision.go:47`: reached only through `Build`/`BuildOne` from `serve` `toolWork`/`toolSpec`, both using `spec.Discover(s.heroDir)`.
  - No other callers exist.
- **Three-file layout:** `Archived` is stamped in a loop after the walk, over every spec found, including `ParseThreeFile` results (their virtual `Path` is `<dir>/spec.md`). A probe confirmed `specs/b/requirements.md` gets `archived=true`.
- **heroDir forms, probed against real `Discover`:** absolute, trailing slash, `./real/` relative, an uncleaned `..` path, and a symlinked root with a trailing slash all classify correctly. `filepath.Walk` builds child paths with `Join`, which cleans them, so they match the `Join`-built prefix. A symlinked heroDir **without** a trailing slash finds zero specs because `Walk` uses `Lstat` on the root. That is pre-existing `Discover` behavior, not introduced or worsened here.
- **Serialization:** `Spec` is never JSON-, gob- or YAML-encoded wholesale, and no `DeepEqual`/`cmp.Diff` compares `Spec` values, so `json:"-"` cannot change any output or golden test.

## Open items (if any)
- None. Every ledger row is DONE.

## Audit notes
- **No blocking defects found.** `go vet ./...` is clean at 012bd22d. `go test ./internal/cli ./internal/traversal ./internal/workmodel ./internal/spec ./internal/serve -count=1` passes at 012bd22d. At 8d5c20b3, `go vet ./internal/spec` is clean and `go test ./internal/spec -count=1` passes. The worktree was restored to clean after every falsification.
- **8d5c20b3 (test-only):** `TestArchivedSpecIsExemptFromStaleness` now writes each audit and spec together and backdates the audit's mtime by one hour with `os.Chtimes`, replacing the 10 ms sleep. All four fixtures (`y`, `planning`, `z`, `specs`) still have an audit older than their spec. The commit also refreshes `go-tests.log`; only timings changed, and every package is still `ok`. No production code changed.
- **Observation, not a defect:** at 012bd22d, the archive exemption has no user-visible effect in any production path. `hero spec verify` returns early for archived completed specs. `VerifyOf` already accepts a stale audit for a finished spec (`model.go:393`). `revision.go` uses only `audit.Path`. AC-3 is therefore proven at the `spec` package level, which is the only level where it shows. It guards future callers.
- **Non-blocking test nit:** the DB-error pass-through in `--edges` has no test. The change is a one-line `return err`, so reading the code is sufficient.
- **Resolved in 8d5c20b3:** the 10 ms mtime sleep in `TestArchivedSpecIsExemptFromStaleness` was a flake risk on coarse-mtime filesystems. Each audit's mtime is now backdated by one hour with `os.Chtimes`, so the result no longer depends on timestamp resolution.
- **Non-blocking test nit:** the initiative `verify: null` assertion unmarshals into a fresh struct, so it cannot tell an omitted key from `null`. The `json:"verify"` tag has no `omitempty` (`model.go:54`), so `null` is guaranteed by the type.
- **"By design" list:** I accept every entry. Each names a concrete reason or a prior recorded decision: shutdown refusal, committer-time ordering with an mtime fallback for uncommitted work, the same-commit pair from `followup-audit-staleness-git`, handoff decisions from `read-contract-v1`, and Aha! as a documented limitation (`ErrAhaAdapterNotImplemented`). I did not re-measure the signer-lookup figure (0.2–0.3 s on 513 specs); it is taken as stated.
