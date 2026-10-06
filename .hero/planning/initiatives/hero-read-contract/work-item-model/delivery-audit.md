# Delivery audit — work-item-model

**Audited:** `git diff 3f75cfd6...df250e73`, which covers `internal/workmodel/model.go`, the tests, and the amended `read-contract-v1`. This is re-audit round 3.
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: one item per work spec, knowledge excluded, path order. Tested by `TestBuildIncludesOnlyWorkSpecsInPathOrder`. A real `hero_work` call returned 513 items with no duplicate slugs.
- [✓] AC-2: lanes by first-match rules, with RecentDays and unmet deps. Tested by `TestLanes`, `TestRecentDaysWindow` and `TestEdgeCasesFromAudit`. `TestRound2AuditCases` adds `watcher` → `designed`, which checks that a dependency resolves to the live spec and not to the intake that shares its slug.
- [✗] AC-3: verify state and audit, "matching `hero spec verify` Gate 1".
  - `ledgerAllDone` (model.go:367) accepts any SKIPPED/BLOCKED row whose `ParseLedger` row has `SignedOff == true`.
  - Gate 1 (`internal/cli/verify.go:218-221`, `checkLedger`) first calls `ledger.ResolveSigners(knownSigners(...))`. That clears the sign-off for any signer that is not a git author and not in `ledger.signers`, so the gate fails closed.
  - The model skips this step, so it is fail-open.
  - **Real-corpus proof:** `token-efficiency-pass` gets verify `passed` from the model, so its next step is null and clients show Delivered. Gate 1 rejects its rows 10 and 19, whose "signers" are free text such as `"explicitly lowest-priority/optional per this spec's own text …"`.
  - This contradicts the amended contract ("rows that `hero spec verify` would not accept" → partial) and the requester's rule "only canonical delivery projects Delivered".
  - `TestRound2AuditCases` uses a real git-author signer (`chet-bellows`), so it cannot catch this.
- [✓] AC-4: normalization and progress. No change since round 2.
- [✓] AC-5: revision. `NewCorpus` (model.go:117-128) now lets a non-intake spec replace a promoted intake that shares its slug; in every other collision the first spec seen wins.
  - On the real corpus, `mail-b7ca19966ac5041e6ff604dd` now resolves to the delivering feature: `hero_work` and `hero_spec` both report `in_progress` / Continue.
  - Ten consecutive real `hero_work` calls returned an identical revision.

## Changes
- [✓] `internal/workmodel/model.go`: intake-safe `NewCorpus`, and signed-off rows in `ledgerAllDone`. The latter is incomplete, see AC-3.
- [✓] `internal/workmodel/next_test.go`: `TestRound2AuditCases` covers the intake collision, the signed-off pass, and the previously untested table rows.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Blocker:** `ledgerAllDone` must resolve signers as Gate 1 does: git authors and emails (user part included) plus `ledger.signers` from `hero.json`, via `LedgerResult.ResolveSigners`. Without that, verify `passed` is fail-open. The open bug `ledger-signoff-substring-match-fails-open` is in the same area. Add a test whose signer is free text.
- **Your question about the two untested minor cases:**
  - A finished spec with non-DONE rows, and an unfinished spec with a stale audit.
  - They are **not blocking** on their own. Both paths are one-line conditions I traced by reading the code.
  - The AC-3 defect above sits in the first path, though. Its fix should come with the free-text-signer test and a plain unsigned-SKIPPED → partial test.
- **Re-run (auditor):** at a clean df250e73 worktree:
  - `go test ./internal/workmodel ./internal/serve ./internal/nextdoc ./internal/install ./internal/cli -count=1` passes.
  - `go vet` is clean, and `gofmt` reports nothing in the new files.
- My first local build accidentally included uncommitted `polish-and-suggested` work from the working tree. All results above come from a clean worktree at df250e73.
