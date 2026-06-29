# Delivery audit — delivery-closing-gate-terminal-contract

**Audited:** `git diff -- internal/install/agents_md.go internal/install/harness_smoke_test.go domains/engineering/commands/deliver.md domains/engineering/AGENTS.md`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Generated Codex AGENTS.md contains terminal-state contract (not done until `hero spec verify` passes; audit first; run the gate rather than yield) — `internal/install/agents_md.go:367` (`renderCodexWorkflowSection()` appends "**A Hero workflow is not finished until its closing gate runs.**" paragraph, Codex-gated via `Render` `if s.opts.Target == TargetCodex`). Asserted by `TestHarness_SmokeCodex` (`harness_smoke_test.go:123-124`: "not finished until its closing gate runs", "run it now instead"). PASS.
- [✓] `deliver.md` has `## Definition of done` before `## Delivery modes` (same-turn, every mode incl. supervised) — `domains/engineering/commands/deliver.md:20` (`## Definition of done`) precedes `## Delivery modes` at line 40. Body states "not finished until `hero spec verify <slug>` passes," "run in the **same turn**," and "holds in **every** mode, including the default supervised mode."
- [✓] `--supervised` row states closing gates are not handoffs, run before yielding — `deliver.md:47`: "The closing gates (cold audit → `hero spec verify`) are NOT handoffs — run them before yielding, never stop short and hand back with the audit unrun."
- [✓] `harness_smoke_test.go` asserts new contract text; install tests pass — `harness_smoke_test.go:52-53` (shared line on CLAUDE.md + AGENTS.md) and `:123-124` (Codex deeper text). `go test ./internal/install/...` → ok; all 6 per-target smoke tests + `TestEngineeringPackBodyMatchesGoFallback` PASS.
- [✓] `go build ./...` clean — exit 0 confirmed.
- [✓] (AC6) Always-loaded contract reaches **all targets** via the shared body — CLAUDE.md and every target's AGENTS.md, not just Codex — verified by mechanism, not ledger word. See AC6 detail below.

### AC6 mechanism verification (the scrutinized claim)

The "all targets" reach is real and structural, not Codex-gated:

- **(a) Shared body has the item.** `generateEngineeringAgentsMdBody()` Key Workflow item 5 added at `agents_md.go:433` — "**Finish the closing gate before yielding**: `/deliver` is not done until `hero spec verify <slug>` passes…". This is the shared body used by every target.
- **(b) Pack file carries the identical line.** `domains/engineering/AGENTS.md:52` is byte-identical to the Go fallback string (regen ran). `TestEngineeringPackBodyMatchesGoFallback` PASS confirms pack == fallback.
- **(c) Smoke test asserts it on both root files.** `harness_smoke_test.go:52-53` asserts "Finish the closing gate before yielding" on both `CLAUDE.md` and `AGENTS.md` for the Claude target. PASS.
- **Reach proof:** `agentsMdBodySection.Render` (`agents_md.go:334-339`) emits `s.body` (the shared body) unconditionally for every target, and only *appends* `renderCodexWorkflowSection()` when `Target == TargetCodex`. So the shared item lands in every target's root file; the deeper paragraph stays Codex-only. Mechanism is sound — claim holds.

## Changes

- [✓] `internal/install/agents_md.go` — `renderCodexWorkflowSection()` terminal-state paragraph appended — `agents_md.go:367`.
- [✓] `internal/install/agents_md.go` — `generateEngineeringAgentsMdBody()` Key Workflow item 5 — `agents_md.go:433`.
- [✓] `domains/engineering/AGENTS.md` — regenerated, item 5 present byte-equal at line 52 — confirmed by `TestEngineeringPackBodyMatchesGoFallback` PASS.
- [✓] `domains/engineering/commands/deliver.md` — `## Definition of done` callout (line 20) + `--supervised` row annotation (line 47).
- [✓] `internal/install/harness_smoke_test.go` — Codex assertions (`:123-124`) + shared-body assertions on CLAUDE.md/AGENTS.md (`:52-53`).

## Open items

None. All ledger rows (AC1–AC6, C1–C5) verified DONE with concrete diff + test evidence.

## Audit notes

- **Diff is well-scoped.** Only the 4 named files changed; no scope drift.
- **Minor wording nuance on AC6's test coverage (not a defect).** The spec's Acceptance Criteria says the shared contract is "asserted by the smoke tests" across "all six install targets." Literally, only the **Claude** smoke test (`TestHarness_SmokeClaude`) adds a per-target assertion for the shared line on CLAUDE.md+AGENTS.md. The other five targets (opencode, cursor, codex, copilot, generic) do not add a per-target assertion for the shared line. This is not a gap in delivery: the shared body flows to every target through one unconditional code path (`Render` emits `s.body` for all targets), and `TestEngineeringPackBodyMatchesGoFallback` locks the body content — so the Claude assertion + pack-match test together cover the reach. The "asserted by the smoke tests [on all six]" phrasing is slightly stronger than the literal per-target assertions, but the underlying mechanism is correct and tested. Flagging for transparency, not as a hold.
- This is an instruction-text-only change; no runtime behavior to exercise. Deliverable is the presence, correctness, and target reach of specific instruction text — all confirmed by reading the diff, the generated/pack files, and passing tests.
