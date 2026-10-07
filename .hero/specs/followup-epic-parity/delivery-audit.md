# Delivery audit — followup-epic-parity

**Audited:** `git diff 27add446...3925c865` on top of `git show 27add446` (clean detached worktree at 3925c865), round 2
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: an epic is modelled like an initiative — `IsContainer` (`internal/workmodel/model.go`) is used in:
  - `IsWorkItem` (decision parents), `buildItem` (progress), `Lane` (started), `VerifyOf`;
  - `Revision` (`revision.go`);
  - `NextFor`, both the finished and the Compose/Drive branches (`next.go`).
  `Designed` lists the same two types with a pointer comment to `IsContainer`. `TestEpicParityAndContainerVerify` passes.
- [✓] AC-2: `verify: null` for containers, and every polish entry has a next step — `VerifyOf` returns nil for containers. The same test asserts nil verify for `big`, `init` and `doneinit`, and a non-nil `Next` on every `Polish` entry.

## Changes
- [✓] `internal/workmodel/model.go`, `revision.go`, `next.go`: `IsContainer` applied.
- [✓] `internal/workmodel/next_test.go`: `TestEpicParityAndContainerVerify`.
- [✓] `.hero/specs/hero-read-contract/read-contract-v1/spec.md` amendment — present. The notification claim is now corrected and verified: `hero_mail_thread_show` on thread `mail_2a0ea87cdef292b91b0035db` shows a `reply_succeeded` event `reply-mail_2a0ea87c-epic-container-verify-amendment` with `source_id` `mail_1204c44a924697e2adb2475b`, applied at 2026-10-07T02:04:36Z. The body of the outbound reply cannot be read from this side; the event confirms it was sent.

## Open items
- None.

## Audit notes
- None. `internal/serve/testdata/read_contract_v1.golden` is unchanged and already allows `verify:null` for `hero_work.items[]` and `hero_spec.item`. `go test ./internal/serve -run ReadContractV1 -count=1 -race` passes, so the change is additive within v1. `go test ./internal/workmodel -count=1 -race` passes.
