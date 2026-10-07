# Delivery audit — hero-work-tool

**Audited:** commit df250e73 plus the later changes in `git diff df250e73 8597d000 -- internal/serve/mcp_tools_read_contract.go`: `Signers` passed through, and `polish`/`suggested` wired by the sibling spec. This is re-confirmation round 4, built and run in a clean worktree at 8597d000.
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: HeroWork shape. Tested by `TestToolWorkShape`, which now asserts that `polish` and `suggested` are lists and that suggested ends with Explore when the backlog is thin. A real call returned schema_version 1 and 513 items, with polish and suggested filled by `polish-and-suggested`.
- [✓] AC-2: the revision ignores `generated_at` and tracks content. Tested by `TestToolWorkRevision`. Ten real calls returned the single revision `6efdc906d18c54f1`.
- [✓] AC-3: `recent_days` validation. Tested by `TestToolWorkRecentDays`.
- [✓] AC-4: read-only, with nothing written under the watch globs. `readOnlyHint` is true. The auditor's snapshot of all six globs was unchanged before and after the calls.
- [✓] AC-5: latency and stability.
  - The ledger now cites direct timings: first call 243 ms, median 276 ms, max 306 ms, one revision (`real-exercise.log`).
  - The auditor measured independently at 8597d000: 0.19–0.25 s over 10 calls, one revision. This now includes the `git log` signer lookup on every call. Well under 2 s.

## Changes
- [✓] `toolWork`, `workRevision`, `WatchGlobs`, `HeroWork`.
- [✓] Registration.
- [✓] Tests.
- [✓] Round 4: `Signers: s.ledgerSigners()`.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- None. The round-3 evidence gap (no direct latency figures) is closed, and the inherited sign-off defect is fixed in work-item-model.
