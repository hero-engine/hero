# Delivery audit — followup-epic-parity

**Audited:** `git show 27add446` (worktree at 27add446)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1: an epic is modelled like an initiative.
  - `IsContainer` (`internal/workmodel/model.go:265`) is used in `IsWorkItem` (decision parents), `buildItem` (progress), `Lane` (started) and `Revision` (children). `Designed` adds `"epic"`.
  - `next.go:58` and `next.go:86` already treated `epic` as a container.
  - `TestEpicParityAndContainerVerify` passes. It covers progress 3/1 from a declared stub plus parent-edge children, the in_progress lane, Drive, a decision under the epic as an item, and Compose plus `LaneNone` for a childless epic.
- [✓] AC-2: `verify: null` for initiatives and epics, and every polish entry has a next step — `VerifyOf` returns nil for containers (`model.go:389`). The same test asserts verify is nil for `big`, `init` and `doneinit`, and that every `Polish` entry has a non-nil `Next`. `weak_verify` (`polish.go:63`) is gated on `Verify != nil`, so containers can no longer produce it.

## Changes
- [✓] `internal/workmodel/model.go`, `revision.go`: `IsContainer` applied.
- [✓] `internal/workmodel/next_test.go`: `TestEpicParityAndContainerVerify`.
- [~] `.hero/specs/hero-read-contract/read-contract-v1/spec.md` amendment — present (verify-null sentence plus the 2026-10-07 amendment entry). The ledger note "hero-harness notified" has **no evidence found**. `hero_mail_list` shows the hero-harness thread with replies for ack, proposal, landed-121f850e and v0.35.0-released, and none for this amendment. `.hero/peer-calls` has nothing either.

## Open items
- None in the ledger.

## Audit notes
- Consistency: every container distinction in `internal/workmodel` (`model.go`, `next.go`, `revision.go`, `polish.go`) now agrees. `next.go` and `Designed` use string `case` lists instead of `IsContainer`, which is equivalent but not the single predicate the Excellence note claims.
- Additivity: `internal/serve/testdata/read_contract_v1.golden` is unchanged in this commit and already contains `hero_work.items[].verify:null` and `hero_spec.item.verify:null` (from decision fixtures). `go test ./internal/serve -run ReadContractV1 -count=1 -race` passes. The change is additive at the schema level. Semantically, a consumer that read initiative `verify.state` will now get `null`, which is why the peer notification matters.
- No serve-level fixture exercises an initiative or epic `verify: null`. Coverage is at workmodel level only.
- `go test ./internal/workmodel -count=1 -race` passes.
