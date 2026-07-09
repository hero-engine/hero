# Delivery audit — drive-progressive-design

**Audited:** uncommitted working tree — `git diff -- internal/drive/ internal/cli/goal_test.go domains/engineering/skills/drive/SKILL.md scripts/drive/stop-hook.sh` + untracked `internal/drive/stage.go`, `internal/drive/stage_test.go`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC#1 — undesigned child (no `## Acceptance Criteria`) → `needs-design` + `action: design`, never delivery — `stage.go:41` returns `StageNeedsDesign` on empty AC section regardless of score; `check.go` sets `res.Kickoff` only when `res.Action == ActionDeliver`, so a design action carries no kickoff. Tests `TestCheckRoutesDesignForUndesignedChild` (asserts `action=design`, `kickoff==""`), `TestChildStageClassification`.
- [✓] AC#2 — designed + scored child → `action: deliver` + kickoff — `stage.go:47` `StageReadyDeliver`; `check.go` sets kickoff from `nextI.spec.Kickoff()`. Test `TestCheckRoutesDeliverForDesignedChild`.
- [✓] AC#3 — not `done` while any intended child unspecced/undesigned — `Check` seeds `Verdict:"done"`, moves only `StageDone` children to Completed, all others to Remaining; non-empty Remaining short-circuits the done return. A `nil`-spec declared child is `StageNeedsScaffold` → stays Remaining. Test `TestCheckNoShortCircuitOnDeclaredButUnscaffoldedChild`.
- [✓] AC#4 — routine design proceeds autonomously (no pause) — design routes through `NeedsMe` with `NextScore:-1` (score detector deferred), yielding `continue` + `action:design`, not a pause. Tests `TestDryRunSurfacesAction` (step1 = design, no pause), `TestCheckRoutesDesignForUndesignedChild`.
- [~] AC#5 — design fork → `DesignFork` pause — pause *capability* present: `DesignFork` category exists (from `needs-me-predicate`) and `SKILL.md` instructs the design step to pause on a genuine fork. Automated in-design fork *detection* is NOT wired (no detector fires `DesignFork` inside `Check`; `NextScore:-1` is the same dormant posture as the other deferred detectors). Ledger marks this "DONE (capability)" and states the detection is a future refinement — honest, not overclaimed.
- [✓] AC#6 — intended set = union(parent relations, declared table) — `buildIntended` (`check.go`) unions `Children()` (parent relations, the spine) with `declaredChildSlugs()` (`stage.go:66`, parses `[slug](slug/spec.md)` links); a declared slug with no on-disk spec attaches `spec==nil` → needs-scaffold. Tests `TestDeclaredChildSlugs` (external `../` link ignored), `TestCheckNoShortCircuitOnDeclaredButUnscaffoldedChild`.
- [✓] AC#7 — every intended child designed + verify PASS → `done` — done-logic runs over the full intended set; `completedSet` reflects verify-gated `completed`/`superseded` status. Tests `TestCheckDoneWhenAllChildrenCompleted` (pre-existing, still green), `TestDryRunSurfacesAction` (ends in `done`).

## Changes

- [✓] Stage classifier + child-table parser — `internal/drive/stage.go` (new): `Stage` enum, `ChildStage`, `ActionForStage`, `DesignReadyThreshold=40`, `declaredChildSlugs`.
- [✓] Check/DryRun rewrite (intended-set, stage, action) — `internal/drive/check.go`: `buildIntended`, `intended` type with `stage()`/`ready()`, `scoreFn` (real `score.Score` default), `Action` added to `CheckResult` and `DryStep`; `step()` removed and folded into `Check`.
- [✓] Skill + Stop-hook act on `action` — `SKILL.md` routes `action: design`→`/design`, `action: deliver`→`/deliver`, design-fork note; `scripts/drive/stop-hook.sh` parses `action`/`next_spec` and emits `/design` vs `/deliver` continuation reason.
- [✓] Tests + designed-child fixture — `internal/drive/stage_test.go` (new), `check_test.go` (`mkStub`, pinned `scoreFn`, new Check tests), `internal/cli/goal_test.go` (`goalChildA` given AC + Design + Test Plan so it classifies ready-to-deliver).

## Open items

- AC#5 — `~` partial by design — engineer's reason: automated in-design fork DETECTION deferred as a future refinement, consistent with the other dormant detectors from `needs-me-predicate`. **Concrete.** The pause capability and skill instruction are in place; only the automated trigger is deferred. Not a soft skip.

## Audit notes

- **Real scorer is wired in production.** Default `scoreFn` (check.go:79–84) calls `score.Score(s, score.DefaultConfig()).Score`. The `scoreFn = func(*spec.Spec) int { return 100 }` override lives only in `check_test.go`'s `init()` — a test-determinism aid so Check-level tests key off AC-presence structure, not the real scorer's verdict on tiny synthetic fixtures. `stage_test.go` exercises `ChildStage` with explicit score arguments (including the `score < 40` → needs-design path and `score == -1` → skip path), so the threshold logic is covered with the override absent. This is not hiding a real-scorer problem.
- **Action reaches the harness end-to-end.** `CheckResult.Action`/`DryStep.Action` carry `json:"action,omitempty"`; `goal.go` `goalEmitJSON` JSON-encodes the whole struct; `stop-hook.sh` reads `"action"` and branches to `/design` vs `/deliver`. The full contract chain is intact.
- **Scope is clean** — every change lands in the spec's named files (stage.go, check.go, SKILL.md, stop-hook.sh, the three test files). No drift.
- **Build/vet/test all green** (see below).
