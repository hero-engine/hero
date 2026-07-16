# Delivery audit — next-drift-gate-branch-line-drift (revision, re-audit)

**Audited:** `git diff HEAD -- internal/projection/` (working tree vs HEAD) + working-tree `.hero/NEXT.md`
**Verdict:** SHIP
**Surface:** noteworthy

> Re-audit of the REVISED fix. The prior audit HELD on a real blocker: the
> first implementation dropped BOTH `branch:` and `session:`, and `session:`
> is consumed (`graph_ingest` Session-node ingestion + `readSessionFromExistingNext`
> → `attemptsForSession` anchoring `## Tried and failed`). The revision narrows the
> fix to drop **only** `branch:` and **keep** `session:`. This audit verifies that
> the narrowing resolves the blocker without reopening the drift.

## Acceptance criteria

- [✓] **`session:` is preserved (prior blocker resolved).** `internal/projection/projection.go:87-94`
  still emits `session: %s` guarded by `opts.SessionID != ""` — only the `branch:` block
  was removed. The full handoff path is intact:
  - `readSessionFromExistingNext` (`internal/cli/next_project.go:93-113`) still reads
    `session:` from frontmatter and returns it.
  - `attemptsForSession` (`internal/projection/projection.go:273`) is still called at
    `projection.go:160` to render `## Tried and failed`.
  - `graph_ingest.go` `parseNext` still parses `session:` (`:145-146`) and the Session-node
    upsert (`:55-79`) still fires when `parsed.session != ""`.
  The prior audit's blocker (dropping `session:` silently empties the handoff section) is gone.

- [✓] **`branch:` is fully removed (drift source eliminated).** The `if opts.Branch != ""`
  emission block is gone, replaced by an explanatory comment (`projection.go:95-100`). No
  code path emits `branch:` into NEXT.md frontmatter. The rebuilt binary
  (`go build ./cmd/hero` → BUILD OK) produces frontmatter of only `updated:` + `repo:` in the
  commit/CI path, and `+ session:` only in a live session. Working-tree `.hero/NEXT.md`
  frontmatter is `updated:` + `repo:` only — `grep -c '^branch:' .hero/NEXT.md` → 0.

- [✓] **The drift gate would pass.** The delivery's working-tree `.hero/NEXT.md` was
  regenerated branch-free (the delivery removes the stale `branch:` line as a working-tree
  modification staged for the same commit — see Open items for the in-flight nuance).
  `git grep '^session:' -- .hero/NEXT.md` and `git grep '^branch:' -- .hero/NEXT.md` → none
  in the tracked tree. The remaining volatile field `updated:` is ignored by the gate
  (`.github/workflows/test.yml:57` `git diff --exit-code -I'^updated: '`). `session:` is NOT
  committed and cannot be committed through normal flows — see residual-risk note below.

- [✓] **Tests guard both behaviors.** `internal/projection/projection_test.go`:
  - `TestNextMD_HappyPath` (`:88-109`) passes `Branch: "main"` and asserts
    `strings.Contains(out, "branch:")` → `t.Error`. Re-adding branch emission fails it.
  - `TestNextMD_EmitsSessionButNotBranch` (`:481-504`) passes both `Branch` and `SessionID`,
    asserts `session: sess-xyz` **present** AND `branch:` **absent**. Dropping `session:`
    or re-adding `branch:` fails it. Genuine two-sided guard.

- [✓] **Full suite green under `-race`.** `go test -race -count=1 ./...` → EXIT 0, 86 packages
  `ok`, 0 `FAIL`, no `panic`/`DATA RACE`. Targeted `internal/projection`, `internal/cli`,
  `internal/nextdoc` each `ok` under `-race`.

## Changes

- [✓] `internal/projection/projection.go` `NextMD` — `branch:` emission removed (comment
  in its place); `session:` emission retained with an explanatory comment. Matches the
  revised approach (drop branch only).
- [✓] `internal/projection/projection_test.go` — `TestNextMD_HappyPath` flipped to assert
  branch absent; new `TestNextMD_EmitsSessionButNotBranch` added pinning the split.
- [✓] `.hero/NEXT.md` (working tree) — regenerated branch-free with the fixed binary.
- [—] `opts.Branch` and its two call-site assignments (`checkpoint.go:471`,
  `next_project.go:59`) left inert (surgical, as the spec permitted). `opts.SessionID`
  remains live (still consumed). Consistent with the revised ledger.

## Open items

- **Delivery not yet committed — commit the regenerated `.hero/NEXT.md` with the code.**
  HEAD's committed `.hero/NEXT.md` still carries a stale `branch: fix/parent-watchdog-race-ci-red`
  line; the working tree (this delivery) removed it but it is not yet committed. This is the
  expected in-flight state, not a defect — the NEXT.md regeneration must land in the SAME
  commit as the `projection.go` change (per the handoff-travels-with-commits rule). Once
  committed, committed == CI-regenerated (modulo `updated:`) and the gate goes green. — concrete

## Audit notes

- **Prior HOLD is fully resolved.** The blocker was a false safety claim ("nothing consumes
  `branch:`/`session:`") plus the resulting silent capability loss. The revision keeps
  `session:` — the actually-consumed line — so `graph_ingest` Session ingestion,
  `readSessionFromExistingNext`, and the `## Tried and failed` anchor all still function.
  Only `branch:` (a Session-node prop with no independent consumer, and the sole line that
  actually drifted) is dropped. This is the correct narrowing.

- **Residual drift risk is effectively nil (verified at the source, not assumed).** `session:`
  is stamped only when `opts.SessionID != ""`, and the commit/CI path never sets it:
  `writeProjectedNextMD` (`internal/cli/checkpoint.go:460-484`, invoked by the pre-commit
  hook / `hero next checkpoint` at `checkpoint.go:299`) constructs `NextMDOptions` with **no**
  `SessionID` field. CI runs exactly this path (`.github/workflows/test.yml:56`
  `./hero next checkpoint --quiet`), so CI regenerates NEXT.md with no `session:` line — and
  the committed file has none either. The only path that stamps `session:` is
  `hero next project` (`next_project.go`), gated on a `--session` flag or a read-back from an
  already-`session:`-bearing NEXT.md — neither present in CI. The one contrived way `session:`
  could land committed and drift: a developer explicitly runs `hero next project --session=X --write`
  AND commits with `--no-verify` to bypass the regenerating pre-commit hook. A normal commit
  self-heals (the hook re-projects via the no-SessionID path, dropping `session:`). Not a
  real-world flow; does not block.

- **The prior audit's remaining concern is retired by the revision.** The prior HOLD noted the
  Session-from-NEXT ingestion path would go "silently dead" if `session:` were dropped. By
  keeping `session:`, that live-session path (Session node + `attempted_in` edges) is preserved
  exactly as before. No capability was lost.

## What the drift fix got right

The primary objective — make the CI drift gate winnable — is delivered and verified: the
committer-branch vs CI-`main` divergence is removed at the source, the projection is
deterministic modulo `updated:` in the commit/CI path, and the fix no longer touches the
handoff-critical `session:` line. The revision is a clean, minimal resolution of the prior HOLD.
