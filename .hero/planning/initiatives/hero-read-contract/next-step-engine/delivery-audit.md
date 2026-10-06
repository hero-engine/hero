# Delivery audit — next-step-engine

**Audited:** `git show 3f75cfd6`. The files in scope are `internal/workmodel/next.go`, `next_test.go`, `core/commands/verify.md`, `domains/engineering/routing.md`, and the `internal/cli/testdata/prompt_baseline/install_candidate_walk.*`. Audited against the amended `read-contract-v1`.
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria
- [✗] AC-1: every table row, including the nulls.
  - **Fixture rows:** `TestNextStepTable` passes all 17 fixture rows, and each matches the contract's action, label, command, phase and enabled/reason. This includes Verify only within `recently_done`, null for old work, null for verified work, and null for superseded work.
  - **Real-corpus failure:** `ApplyNext` (next.go:34) re-resolves each item's spec by slug through `NewCorpus`, and a promoted intake shadows its promoted spec. The live `delivering` feature `mail-b7ca19966ac5041e6ff604dd` (lane `in_progress`) gets `design / Design / /design …` instead of `deliver / Continue`. The engine computed it from the intake spec. I confirmed this with an independent `Build`+`ApplyNext` probe at 3f75cfd6.
  - **Untested rows:** `in-review`, `awaiting_peer`, `handed_back`, an accepted decision (null), rejected and merged (null), and an initiative with children that has not started (Drive, "Planning"/ready).
- [✓] AC-2: disabled with "waits on". `gateOnDeps` sets `enabled=false`, `reason "waits on <slugs>"` and state `waiting`. The `blocked` and `blockedbug` cases test this.
- [~] AC-3: invariants.
  - Met: slash-only commands, `[]` extras, no Deliver on finished work, and no "Delivered" phase without passed. `TestNextStepInvariants` covers these.
  - **Not met:** "never Diagnose and Deliver together." A diagnosed bug's primary is `action: "deliver"`, and its extra is `{action: "diagnose", label: "Challenge diagnosis", command: "/challenge …"}` (next.go:98).
  - The invariant test was written around this: it ignores a `diagnose` extra unless its label is "Diagnose" (next_test.go, `e.Action == ActionDiagnose && e.Label == "Diagnose"`).
  - A client checking the requester's rule on `action` sees both. The requester's `extras.action` is a free `string`, so a distinct value such as `challenge` is available without breaking the enum.
  - On the real corpus, 1 item emits this pair.
- [✓] AC-4: `/verify` for every target. `core/commands/verify.md` and the routing rows are present. The auditor ran `hero install` for each of the 8 targets into temp repos, and each rendered the workflow:
  - claude: `.claude/commands/verify.md`
  - codex: `.agents/skills/command-verify/SKILL.md`
  - opencode: `.opencode/commands/verify.md`
  - cursor: `.cursor/rules/commands/verify.md`
  - copilot: `.github/prompts/commands/verify.prompt.md`
  - generic: `.ai/commands/verify.md`
  - grok: `.grok/skills/command-verify/SKILL.md`
  - deepseek: `.dsh/skills/command-verify/SKILL.md`

  The installed CLAUDE.md and AGENTS.md route `/verify`.

## Changes
- [✓] `internal/workmodel/next.go` + `next_test.go`: present, but see AC-1 and AC-3.
- [✓] `core/commands/verify.md`. Step 3 requires a **fresh**, independent `delivery-audit` reviewer that is handed only on-disk artifacts, and it stops if the harness cannot spawn one. Step 4 forbids `--force` unless the user explicitly asks, and forbids hand-editing `status:`.
- [~] `domains/engineering/routing.md` and the regenerated `domains/engineering/AGENTS.md`, plus the baselines.
  - The routing rows and baselines are in the diff (`Installed 123 files`, plus the `commands/verify.md` dest).
  - **`domains/engineering/AGENTS.md` is not in the commit and has no diff.** The ledger says it was regenerated, but the regeneration changed nothing. `TestEngineeringPackBodyMatchesGoFallback` and `TestEngineeringAgentsMdRosterComplete` pass, so no regeneration was needed. The ledger claim is inaccurate but harmless.

## Open items
- None in the ledger. Every row is DONE.

## Audit notes
- **Blocker 1 (AC-1):** shared with work-item-model. Fix the slug index so promoted intakes don't shadow work specs. Better still, have `ApplyNext` use the spec that `Build` already used rather than looking it up again by slug. Add a duplicate-slug fixture.
- **Blocker 2 (AC-3):** give the Challenge extra a non-`diagnose` action, and remove the label-based carve-out from `TestNextStepInvariants`.
- **/verify vs the gate (not a blocker, decide):**
  - `verify.md` step 2 stops unless every ledger row is `DONE`, but `hero spec verify` accepts signed-off SKIPPED/BLOCKED rows.
  - The amended contract marks such specs `partial`, so a correctly gated spec with a signed-off skip shows Verify for 14 days, and `/verify` dead-ends at step 2 telling the user to run `/deliver`.
  - Also, for an already-archived completed spec, `hero spec verify` short-circuits with "already completed and archived" PASS (`internal/cli/verify.go:113`). In that case `/verify` effectively only (re)runs the audit, which `verify.md` does not say.
- **Contract ambiguities:**
  - A diagnosed bug gets no `Check against the code` extra, though the contract lists that extra for "Ready". The bug phase is "Diagnosed".
  - `handed_back` maps to Continue, though the next-step table does not list it.
  - The handed-off reason is always "handed off to a peer". The originator spec carries no parsed peer alias, so "when known" never applies.
- **Re-run (auditor):**
  - `go test ./internal/workmodel ./internal/install ./internal/cli -count=1` passes.
  - `go vet` is clean.
  - `tests.log` covers only `./internal/workmodel`. There was no logged evidence for the install/cli claims before this re-run.
- **Real-corpus distribution:** null 408, Design 63, Deliver 12, Continue 8, Drive 10, Compose 6, Diagnose 2, Verify 2, Fix 1, Decide 1.
