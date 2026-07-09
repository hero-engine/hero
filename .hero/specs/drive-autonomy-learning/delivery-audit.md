# Delivery audit — drive-autonomy-learning

**Audited:** `git diff -- internal/drive/check.go internal/drive/check_test.go internal/cli/goal.go internal/cli/goal_test.go internal/cli/helpers_test.go internal/feed/feed.go internal/serve/mcp_tools.go` + untracked `internal/drive/trust.go`, `internal/drive/trust_test.go`
**Verdict:** SHIP
**Surface:** noteworthy

## Dormancy claim — VERIFIED HONEST

The ledger discloses the learning is "armed but dormant": no promotable pause
fires through the live loop in v1 because the promotable-category detectors
(DesignFork / Underspecified / AmbiguousPick) are deferred. This is accurate.

Traced through `drive.Check` → `step` (`internal/drive/check.go:131-157`): `step`
constructs `RunContext` with only `ActionClassified: true`, `NextScore: -1`,
`Blocked: blocked`, `Promoted: promoted`. It never sets `DesignFork`,
`AmbiguousPick`, `VerifyVerdict`, or a non-negative `NextScore`. In
`NeedsMe` (`internal/drive/needsme.go:160-177`) the three promotable branches each
gate on a field `step` leaves zero:
- DesignFork → `ctx.DesignFork` (always false)
- Underspecified → `ctx.NextScore >= 0 && < threshold` (NextScore is -1 → branch skipped)
- AmbiguousPick → `ctx.AmbiguousPick` (always false)

So the only verdicts the live loop can produce are: `Supervised` (supervised
mode), `Blocked` (unmet dep), `continue`, or `done`. Supervised and Blocked are
both non-promotable. The disclosure is truthful — it is not hiding a broken
feature; the machinery is wired but cannot be reached end-to-end until the
deferred detectors set those RunContext fields.

## Acceptance criteria

- [✓] AC1 — K consecutive approved → promote — `Promotions.RecordOutcome` (`trust.go:80-100`, promote at `Streak >= PromoteThreshold=3`); `TestPromotionStateMachine` (`trust_test.go:5-49`); CLI path `runGoalResolve` records `OutcomeApproved` (`goal.go:152`), `TestGoalAnswerRecordsOutcomeAndTrust`.
- [✓] AC2 — edit/redirect → demote + resume pausing — `RecordOutcome` resets streak + clears `Promoted` on Edited/Redirected (`trust.go:96-99`); `TestPromotionStateMachine:24-27`; `--redirect` records `OutcomeRedirected` and re-pauses (`goal.go:148-160`).
- [✓] AC3 — NEVER promote Irreversible / hard-cap — `Promotable()` gate in both `RecordOutcome` (`trust.go:81`) and `IsPromoted` (`trust.go:105`); `maybePromoted` re-checks `cat.Promotable()` (`needsme.go:187`); `TestGuardrailCategoriesNeverPromote` covers Irreversible/HardCap/Unknown/VerifyStuck/Blocked (never tracked, never promoted); predicate-level `TestNeedsMeAutonomousPromotionProceeds:142-147` confirms a promoted Blocked still pauses.
- [✓] AC4 — supervised/guided ignore promotions (autonomous only) — `maybePromoted` requires `mode == Autonomous` (`needsme.go:187`); `TestNeedsMeAutonomousPromotionProceeds:134-140` asserts autonomous proceeds while guided pauses on the same promoted Underspecified.
- [✓] AC5 — promotions inspectable + resettable — `hero goal --trust` returns `PromotedList()` + categories (`goal.go:79-83`), `--untrust <cat>` calls `Reset` + Save (`goal.go:85-95`); `TestGoalAnswerRecordsOutcomeAndTrust` exercises trust→untrust→trust round-trip.
- [✓] AC6 — record each pause outcome as a feed event — `emitDriveOutcomeEvent` appends `drive.pause_outcome` (`goal.go:189-197`); type registered in `feed.ValidTypes` (`feed.go:31`); test asserts `events.log` contains `drive.pause_outcome` (`goal_test.go:170-172`).

## Changes

- [✓] `internal/drive/trust.go` (NEW) — `Promotions` per-user store at `.hero/drive/trust/<user>.json`, `RecordOutcome`, `IsPromoted`, `Reset`, `PromotedList`, `LoadPromotions`/`Save`. Matches spec.
- [✓] `internal/drive/check.go` — `Check`/`DryRun`/`step` thread a `promoted func(PauseCategory) bool` hook into `RunContext.Promoted`. Confirmed in diff.
- [✓] `internal/cli/goal.go` — `--redirect`/`--trust`/`--untrust` flags added; `--answer`/`--redirect` route through `runGoalResolve` which records the outcome + emits the event. Confirmed.
- [✓] `internal/feed/feed.go` — `drive.pause_outcome` registered. Confirmed.
- [✓] `internal/serve/mcp_tools.go` — `Check`/`DryRun` call sites updated to pass `nil` (MCP path intentionally has no promotion hook). Confirmed.
- [✓] Tests — `trust_test.go` (NEW), `goal_test.go` (+`TestGoalAnswerRecordsOutcomeAndTrust`), `check_test.go`/`helpers_test.go` call-site + flag-reset updates. Confirmed.

## Open items

None. The "dormant" status is a deferred-dependency disclosure, not an
incomplete row in this spec — the deferral belongs to `hero-goal-command`
(the detectors), and every AC of *this* spec is independently delivered.

## Audit notes

- The CLI test seeds a `DesignFork` pause directly into the ledger JSON
  (`goal_test.go:160`) rather than producing it through `Check`. This is the
  correct way to exercise the CLI resolution path given the live detectors are
  dormant — it is not a cheat; it isolates the code-under-test (outcome
  recording + trust + event) from the deferred detector.
- MCP `toolGoal` passes `nil` for the promotion hook (`mcp_tools.go:1274,1276`),
  so promotions only apply on the `hero goal` CLI path. Not an AC requirement;
  noting as scope fact.
- Build clean (`go build ./...` rc=0), `go vet` clean on all four packages,
  `go test` PASS on drive/cli/feed/serve. New tests run (not just compile):
  `TestPromotionStateMachine`, `TestGuardrailCategoriesNeverPromote`,
  `TestGoalAnswerRecordsOutcomeAndTrust` all PASS verbosely.
- Surface is `noteworthy` solely because the user should understand the
  shipped-but-dormant status — no performative rows, no soft skips, no
  downgrades.
