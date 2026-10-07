# Delivery audit — next-step-engine

**Audited:** `git diff 3f75cfd6...df250e73`, which covers `internal/workmodel/next.go`, `next_test.go` and `core/commands/verify.md`, against the amended `read-contract-v1` (round-2 amendment). This is re-audit round 2.
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: every table row, including nulls.
  - **Tests:** `TestNextStepTable` has 17 rows. `TestRound2AuditCases` adds in-review and handed_back (Continue), awaiting_peer (disabled, "With peer"), and rejected, merged and an accepted decision (all null). It also adds an unstarted initiative with children (Drive, Planning/ready) and the intake-shadowed live feature (Continue).
  - **Real corpus, clean df250e73 build:** `mail-b7ca19966ac5041e6ff604dd` now returns Continue.
  - **Handed-off reason:** the amended table now says "handed off to a peer", which matches the code.
- [✓] AC-2: disabled with "waits on". The `blocked` and `blockedbug` cases test it.
- [✓] AC-3: invariants.
  - The diagnosed-bug extra uses `ActionChallenge` (next.go:101).
  - `TestNextStepInvariants` now flags any `diagnose` extra next to Deliver, with no label carve-out.
  - On the real corpus, the diagnosed bug `tracker-backed-diagnosis-publication-contract-broken` emits `{action: challenge}`.
- [✓] AC-4: `/verify` for every target.
  - `verify.md` step 2 now accepts the same signed-off rows as the gate. Step 4 explains the already-completed and archived case.
  - The independent-reviewer requirement and the "no `--force` unless the user asks" rule are unchanged.
  - Round 2 already confirmed that all 8 targets render the workflow.

## Changes
- [✓] `internal/workmodel/next.go`: the `challenge` action.
- [✓] `core/commands/verify.md`: signed-off rows and the already-completed case.
- [✓] Ledger change 3 is corrected: the pack `AGENTS.md` needed no regeneration. This matches round 2's finding.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Inherited from work-item-model (its HOLD):** the engine returns `next: null` for any verify `passed` item. The model currently reports `passed` for specs whose sign-offs Gate 1 rejects; `token-efficiency-pass` is one real case. So that item shows as Delivered instead of offering Verify. The fix belongs in `workmodel.ledgerAllDone`, not in the engine, and the engine needs no change.
- **Wording mismatch in `verify.md`:** step 2 describes the accepted sign-off as "structured", but does not mention that Gate 1 also requires the signer to be a known identity, meaning a git author or a `ledger.signers` entry. Mention it when the model fix lands.
- **Re-run (auditor):** at a clean df250e73 worktree, the workmodel, serve, nextdoc, install and cli tests pass, and `go vet` is clean.
