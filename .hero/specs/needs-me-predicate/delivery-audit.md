# Delivery audit — needs-me-predicate

**Audited:** uncommitted working tree — `git diff -- internal/drive/ internal/spec/spec.go internal/spec/spec_test.go` plus untracked `internal/drive/needsme.go`, `internal/drive/needsme_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Irreversible action → `pause{Irreversible}` in every mode — `needsme.go:142-144` returns the Irreversible pause as the first hard guardrail, before the mode switch and before any taxonomy/promotion path. `TestNeedsMeIrreversibleAlwaysPausesEveryMode` (needsme_test.go:83-98) asserts pause + `CategoryIrreversible` across Supervised/Guided/Autonomous with `Promoted` returning true.
- [✓] `supervised` pauses at every boundary (parity) — `needsme.go:156-158` short-circuits to `pause(CategorySupervised)`. `TestNeedsMeSupervisedAlwaysPauses` (needsme_test.go:72-81) feeds a `base()` ctx that would proceed in Guided and asserts pause.
- [✓] Next spec below threshold → `pause{Underspecified}` — `needsme.go:171-174` (`NextScore >= 0 && NextScore < ScoreThreshold`). `TestNeedsMeTaxonomy/underspecified_pauses` (score 40 < 60). Companion `score_unknown_does_not_underspecify` confirms `NextScore=-1` does not spuriously pause.
- [✓] Verify failed N times → `pause{VerifyStuck}` (not infinite retry) — `needsme.go:161-164` (`VerifyFailCount >= VerifyStuckThreshold`). `TestNeedsMeTaxonomy/verify_fail_at_threshold` (count 2 == threshold 2 → stuck) and `verify_fail_under_threshold` (count 1 → proceeds as rework).
- [✓] Unclassifiable transition → pause (conservative) — `needsme.go:145-147` (`!ActionClassified` → `CategoryUnknown`). `TestNeedsMeUnknownActionPauses` (needsme_test.go:100-108) asserts pause + `CategoryUnknown`.
- [✓] verify-PASS → ready scored next child → proceed (guided/autonomous) — proceed-silently fall-through at `needsme.go:180`. `TestNeedsMeTaxonomy/{guided,autonomous}_proceeds_on_clean` assert Proceed on `base()`.

## Changes

- [✓] New `internal/drive` predicate package — `needsme.go` (192 LOC): `AutonomyMode` + `ParseMode`, `PauseCategory` (+ `Promotable`), `Decision`, `RunContext`, pure `NeedsMe()`. No I/O, no exec, no loop/CLI logic. All signals arrive via `RunContext`; caller owns score/verify/blocked lookups.
- [✓] `Autonomy` field + parse on the spec model — `internal/spec/spec.go:240-242` (field, defaults to supervised when empty) and `:560-561` (`case "autonomy"`). `TestParseAutonomyField` (spec_test.go) asserts `autonomy: autonomous` parses.
- [✓] Table-driven + property + hard-cap + promotion tests — `needsme_test.go`, 8 test funcs. Hard-cap and initiative-boundary covered by `TestNeedsMeHardCapAndBoundary`; promotion seam by `TestNeedsMeAutonomousPromotionProceeds` and `TestPromotableScope`.

## Open items

None. Every ledger row is DONE with matching code + asserting test.

## Audit notes

Adversarial checks the auditor was asked to run, verified by reading code (not trusting tests):

- **No scope creep.** `NeedsMe` is a pure function: zero I/O, zero loop driving, zero completion evaluation. `ParseMode` is pure string mapping. The spec.go change is a single field + one parse case. Nothing here belongs to the later loop/CLI/`--check` specs.
- **Hard guardrails enforced first, never relaxed.** `needsme.go:141-153` runs Irreversible → Unknown(unclassified) → InitiativeBoundary → HardCap, each with a direct `return pause(...)`, *before* the `mode == Supervised` check (line 156) and *before* any taxonomy or `maybePromoted` call. Mode and promotion are structurally unreachable for these four. `Promotable()` (lines 68-75) also explicitly excludes Irreversible, HardCap, Unknown, VerifyStuck, Blocked, so even the promotable path cannot relax them. `TestNeedsMeIrreversibleAlwaysPausesEveryMode` forces `Promoted=>true` in all modes and still pauses, confirming the structural guarantee.
- **Conservative-when-unknown holds.** A zero-value `RunContext` has `ActionClassified=false` → pauses `Unknown` (line 145). Unknown score (`NextScore<0`) is explicitly guarded out of the Underspecified trigger (line 171) so it neither false-pauses nor false-proceeds on that axis. Empty `VerifyVerdict` does not match `"FAIL"`. Nothing reaches the proceed fall-through without classified, scored, non-stuck, non-blocked signals.
- **Test-plan nuance (non-blocking).** The spec's Test Plan names a dedicated "property test: empty/unknown RunContext always pauses." There is no standalone property/fuzz test by that name; the zero-value path is instead covered concretely by `TestNeedsMeUnknownActionPauses` (ActionClassified=false is the zero value). The behavior is verified; only the test *style* differs from the plan. Not a delivery gap.

Build/vet/test evidence (run by auditor): `go build ./...` clean; `go vet ./internal/drive/ ./internal/spec/` clean; `go test ./internal/drive/ ./internal/spec/` both `ok`; `go test ./internal/drive/ -v` shows all 8 functions + 9 subtests PASS.
