---
title: "Acceptance Criteria Are Machine-Checkable — Human Attention Is for Judgment, Not Labor"
slug: machine-checkable-acs
type: feature
status: planning
domain: engineering
priority: high
size: small
horizon: now
created: 2026-07-14
relations:
  - target: hero-self-consistency
    kind: related
  - target: spec-state-axes
    kind: related
  - target: living-contract
    kind: related
---

# Acceptance Criteria Are Machine-Checkable — Human Attention Is for Judgment, Not Labor

## Goal

Establish one authoring rule, propagated to all six install targets: **an acceptance criterion must be checkable by the model itself, with the tools it has.** If checking it requires a human to go do work and report back, it is not an acceptance criterion — it is an observation, and it goes in a `## Unchecked` section that creates no obligation and gates nothing. Forward-only, guidance-only, no code changes.

## Kickoff

Stops specs from writing acceptance criteria that only a human can check — which the model then holds its own work hostage to, and the human rubber-stamps.

**Status:** planning — spec just landed, no code yet. Guidance-only; no Go changes.

**Pick up at:** write the rule into `core/skills/spec-format/SKILL.md` (new `### The model-checkable rule` subsection under "Acceptance Criteria and EARS"), then thread the one-line pointer through the two delivery leads and `/design` + `/diagnose`.

→ `.hero/planning/features/machine-checkable-acs/spec.md`

**Files:** `core/skills/spec-format/SKILL.md`, `domains/engineering/agents/feature-delivery-lead.md`, `domains/engineering/commands/design.md`, `internal/spec/acceptance.go:52`
**Skip:** no `PENDING` ledger status, no Gate 1 change, no lint rule, no rewrite of the 250 existing specs — all rejected below as machinery for a channel that shouldn't exist.

## Problem

The loop this deletes, in the maintainer's own words:

> the human says spec this up → the model writes AC with a human test → the model does all the work → then says "ok it's all done except the part where the human needs to go verify it" → the human says "ok i verified it" (they didn't really) → it's kinda pointless

It is **worse than pointless — it is negative value**:

- The model invents an obligation the human never asked for.
- The model holds its own work hostage to that invented obligation.
- The human rubber-stamps it to release the hostage.
- `internal/cli/verify.go:596-601` (`recordLedgerToGraph`) converts that rubber stamp into `acceptance.RunResult{Status: "pass"}` — writing it into the graph **as evidence**.

Two false attestations, zero information, one permanent false provenance record.

**Not writing the AC is strictly better than writing it.** A list entry reading "nobody ever checked the export button renders" beats both, because it is the only artifact in the loop that is true.

### Hero already agrees with this rule everywhere else

`internal/drive/needsme.go:52-73` defines eleven `needs_me` PauseCategories: `DesignFork` ("≥2 viable approaches with material tradeoffs"), `AmbiguousPick`, `Underspecified`, `Irreversible`, `VerifyStuck`, `Blocked`, `HardCap`, `SeamCollision`, `SeamDetected`, `Supervised`, `Unknown`.

**Every one is a fork or a question.** That taxonomy already encodes the principle: human attention is for judgment, not labor.

The manual-QA AC is the **only** place in Hero that asks a human for labor rather than judgment. It is a fourth interaction channel nobody designed, it bypasses the `needs_me` taxonomy, and it asks for the one thing that taxonomy never contemplated. `/drive` got this right; the AC path never got the memo. This spec makes the AC path agree with `/drive`.

### Why the gate makes it worse

- `internal/cli/verify.go:282-286` — any non-DONE AC row fails Gate 1.
- `internal/cli/complete.go:57-66` — `hero spec complete` refuses work specs, redirecting to `hero spec verify`.

So one un-eyeballed manual AC means the spec cannot close. Today's only escapes:

1. rubber-stamp `DONE` (a lie that becomes graph evidence),
2. `SKIPPED` + `[signed-off]` (`verify.go:227-234`),
3. `--force` (bypasses every gate, including the real ones).

All three are bad. The ledger vocabulary (`internal/spec/ledger.go:11-15`) is `DONE | PARTIAL | SKIPPED | BLOCKED | UNKNOWN` — nothing means *"a human might glance at this, and it isn't holding anything up."* Rather than add that word, this spec removes the need for it.

### Scale of the residue

**250 of 284 completed specs have no executable evidence linked** (5 have `verified_by:`, 31 have a real smoke script). Some fraction of those 250 is this ceremony, stamped. `internal/acceptance/record.go:33` records AC statuses as `pass | fail | skip`, where `skip` is a no-op (`record.go:196`).

`.hero/specs/living-contract/spec.md:261` named this exact gap ("whether a test *actually* proves the criterion is a human/agent judgment") and scoped it out as a non-goal. This spec picks up the narrow, tractable half of it.

## Approach

### 1. The mechanical test

Stated so a delivery lead can apply it at AC-writing time without a judgment call.

> **Write the check as an imperative sentence whose subject is "I".**
> **If the sentence needs a different subject, it is not an acceptance criterion.**

To apply it, name two things **at authoring time**:

1. **The actuator** — what *I* run or drive to exercise this. `go test ./internal/spec`, `curl localhost:8080/health`, `hero install --target codex`, "launch the app, click Export, screenshot." If the sentence naming the actuator has a human as its subject ("the user opens…", "someone checks…", "the reviewer confirms…"), it fails.
2. **The observable** — what output, artifact, or state *I* read that separates pass from fail. Exit code, matched string, a row in the DB, pixels in a screenshot I captured. If the observable is a person's opinion or perception, it fails.

If you cannot name **both, right now, while writing the AC**, it is not an AC. You will not be able to check it later either.

**The bar is checkable-by-the-model-at-delivery-time — not "has a linked re-runnable test."**

This distinction is load-bearing; conflating the two turns this spec into a 250-spec migration, which is explicitly out of scope.

- **One-shot is fine.** Launching the app, clicking through a flow once, and capturing a screenshot *is* checking. The evidence can be ephemeral. Automation is not required.
- **No test infrastructure is required.** "There's no Playwright harness for this" does not make an AC unverifiable — it makes it manual work for the model, which is exactly the work the model should be doing.
- The rule turns on **model capability**, not on repo tooling maturity.

Consequently the honest failures are narrow: the check needs physical hardware the model cannot reach, production credentials it must not hold, a third party it cannot contact, a human's aesthetic/perceptual judgment that no observable can settle, or — the fifth category — a **temporal property**.

### The temporal failure (added after testing this rule against a GUI corpus)

A criterion whose observable exists only *across frames* — smoothness, flicker, jank, dropped frames, animation continuity — is not checkable by screenshot, because **a screenshot is a point sample and the property lives in the sequence**. Naming an actuator is not enough; the observable must be capturable in a single state read, or by an instrument the model can run (Instruments trace, frame-time assertion, signpost log).

This category is invisible from a text-output CLI, which has no time axis. It was found by testing the rule against `hero-code`'s SwiftUI app, where the distribution proved bimodal: ~0% reclassification across 2,935 desktop AC bullets, but ~89% in a small cluster of scroll/motion bug specs. `hero-code`'s `chat-scroll-breathing-room` is the exemplar — 8 of 9 ACs unchecked, and the single completed one is the only machine-checkable one ("Build and all 15 `ChatScrollFollowPolicyTests` pass").

Apply the same precision forcing-function the rule already uses. Rewrite the temporal AC into either:

- **the frame-time budget an instrument can assert** — `hero-code`'s `hero-app-platform-v2/spec.md:358` ("render each token within 16ms of receipt") already does this correctly and *is* checkable; or
- **the end-state the animation reaches** — `chat-scroll-breathing-room` AC-7 ("scrolls to show the new user message with breathing room") is checkable as written.

The irreducible perceptual residue — "no *visible* flicker" — goes to `## Unchecked`.

That last category deserves naming: **"looks good" is not unverifiable, it is unfalsifiable.** It has no pass/fail state at all. It is not an AC, and it is not really an observation either — it is a design question. If it arises during work, it is a `DesignFork` and belongs in `/drive`'s `needs_me` channel. If it arises at authoring time, rewrite it into the structural assertion underneath it (see the worked example) and put the perceptual residue in `## Unchecked`.

### 2. Where unverifiable observations go: `## Unchecked` in the spec

A top-level `## Unchecked` section, placed after `## Acceptance Criteria`.

Each entry states **what wasn't checked** and **why the model couldn't check it**. Optionally, what would make it checkable later.

Rendered as a blockquote here rather than a fenced block, because
`parseSections` (`internal/spec/spec.go:1082`) scans for `## ` with no
code-fence tracking and trims leading whitespace first — so a literal
heading inside a fence forges a phantom section and truncates the one
above it. See `spec-sections-parser-ignores-code-fences`.

> **`## Unchecked`**
>
> Observations, not criteria. Nothing here gates completion. Nobody is
> obligated to act on any of it.
>
> - **Whether the "Weak matches" section reads as visually distinct to a
>   user.** I can assert the section exists, is titled, and sits beneath
>   Gaps. Whether that separation *reads* as distinct is a perceptual
>   judgment with no observable I can capture.

**Why this surface, over the alternatives:**

| Option | Verdict |
|---|---|
| **`## Unchecked` in the spec** | **Chosen.** Travels with the spec, so provenance is intact. Searchable via `hero search`. No new subsystem, no new command, no schema. **Provably inert** — verified at `internal/spec/acceptance.go:52`, `ParseAcceptanceCriteria` only reads sections whose lowercased heading *starts with* `acceptance criteria`, so `## Unchecked` yields zero criteria, produces zero ledger rows, and Gate 1 has nothing to trip on. The no-op is a property of existing code, not something we build. |
| New `PENDING` ledger status | Rejected. Machinery to administer a channel that shouldn't exist. If no unverifiable AC is written, there is nothing to hold pending. |
| A tracker issue per item | Rejected. Creates an obligation and an assignee — precisely the invented obligation this spec deletes. |
| A knowledge-base entry | Rejected. Divorces the observation from the spec that produced it; loses provenance; nobody reads it. |
| Reuse `## Risks` | Rejected. `Risks` means "this might go wrong during implementation." An unchecked item is not a risk — it is a known absence of information. Merging them destroys both signals. |
| Reuse `## Validation` | Rejected, and instructively so. `Validation` means "here is how to verify this," which is the labor framing wearing a different hat. The corpus proves it: the manual-QA ceremony has mostly pooled in `## Validation`, not in AC bullets (e.g. `.hero/specs/knowledge-export-cli/spec.md:157`). Routing residue there reinforces the defect. |

### 3. Boundary with `needs_me` — the two must not collide

| | `## Unchecked` | `needs_me` (`/drive`) |
|---|---|---|
| **When** | Authoring time | During work |
| **What** | An observation nobody checked | A fork or question only a human can settle |
| **Obligation** | **None.** Gates nothing, pauses nothing, assigns nobody. | **Pauses the run** until the human decides. |
| **Asks for** | Nothing | Judgment |

This spec governs **only what gets written into the AC set at authoring time.** A fork or question arising *during work* still goes to `/drive`'s existing `needs_me` channel — unchanged.

**The rule that keeps them from colliding: never route an `## Unchecked` entry into `needs_me`.** An unchecked observation is not a pending question. Escalating one would re-create the exact labor channel this spec deletes, smuggled in through the judgment door. `needs_me` is for decisions; `## Unchecked` is for absences. If an unchecked item turns out to hide a real decision, it was misfiled — rewrite it as the fork it is.

### 4. Guidance, not enforcement

**Default to guidance. No linter.** The case against enforcement is not "it's expensive" — it is that enforcement cannot work here:

- Any validator would pattern-match phrasing ("manually verify", "the user should", "confirm that"). That is heuristic, false-positive-prone, and **defeated by rewording** — which trains authors to launder bad ACs into passing prose. That is worse than no rule.
- The rule constrains the **author's reasoning** ("can I name the actuator and the observable?"), not the AC's surface text. A checkable AC and an uncheckable one can be lexically identical. There is no non-heuristic signal to match on.
- The gate already has nothing to trip on once the rule is followed (see the inertness argument above). Enforcement would be a second gate guarding a door that isn't there.

Guidance is the correct instrument. If a cheap, non-heuristic enforcement signal is discovered later, it is a separate spec.

### 5. Harness propagation — all six targets

The tripwire `harness-changes-cover-all-targets` [high] is live and directly governs this spec. Following its own "Instead" guidance:

- **Author once, harness-agnostic.** The authoritative rule goes in `core/skills/spec-format/SKILL.md` — a pack-level skill. Every install target materializes the same skill tree from the same source, so `opencode | cursor | claude | copilot | codex | generic` all receive identical rule text with no per-target authoring.
- **Native root instruction file per target.** No new root-file content is required by this spec — the rule lives in the skill, which each target's skill tree carries. If a pointer line is added to the Hero-managed block, it renders once and lands in `CLAUDE.md` for `claude` and `AGENTS.md` for the other five. **No symlink/import shim.**
- **Self-contained and imperative.** The rule text is written as a directive a hookless harness acts on by instruction alone. It is stated in the skill body, not gated on any hook. Claude's Stop/PreCompact hooks are irrelevant to this change — nothing about the rule depends on them.
- **Verify per-target propagation before done.** `hero install --target <t>` into a scratch dir for each of the six, then grep the materialized `spec-format` skill for the rule heading. This is AC-6, and I can run all six myself.

## Changes

1. **`core/skills/spec-format/SKILL.md`** — the rule's authoritative home.
   - Add `### The model-checkable rule` as a subsection under the existing "Acceptance Criteria and EARS" section, stating the "subject is I" test, the actuator/observable pair, and the explicit "checkable-at-delivery-time ≠ has a linked test" clarification.
   - Add `### Unchecked observations` documenting the `## Unchecked` section: purpose, placement (after `## Acceptance Criteria`), entry shape (what wasn't checked + why), and the standing statement that it gates nothing and obligates nobody.
   - Add `## Unchecked` to the feature and bug spec templates as an optional section, with a one-line comment that it is omitted entirely when empty.
   - Add one line to the "Quality bar" list: every AC passes the model-checkable test; anything that doesn't is in `## Unchecked`.

2. **`domains/engineering/agents/feature-delivery-lead.md`** — AC-writing behavior.
   - In the design-phase section, add the rule as a numbered step: when writing acceptance criteria, apply the model-checkable test to each candidate; route failures to `## Unchecked` rather than writing them as ACs.
   - Add the `needs_me` boundary as an explicit one-liner so the lead does not escalate unchecked items into pauses.

3. **`domains/engineering/agents/platform-delivery-lead.md`** — same two additions. It authors specs on the same path and must not diverge.

4. **`domains/engineering/commands/design.md`** — add the rule to the spec-authoring step, referencing `spec-format` as the source of truth rather than restating the full test (avoid drift between the two).

5. **`domains/engineering/commands/diagnose.md`** — same pointer, on the fix-plan AC-writing step.

6. **`.hero/knowledge/`** — no change. The rule is skill content, not a convention spec; adding a parallel convention entry would create two sources of truth.

Note the deliberate shape of this list: **one file carries the rule, four files point at it.** No file restates it.

## Acceptance Criteria

- **AC-1:** WHEN a reader opens `core/skills/spec-format/SKILL.md` THE SYSTEM SHALL contain a `### The model-checkable rule` subsection stating the "subject is I" test and naming both the actuator and the observable as required at authoring time
- **AC-2:** WHEN a reader opens `core/skills/spec-format/SKILL.md` THE SYSTEM SHALL document the `## Unchecked` section, its placement after `## Acceptance Criteria`, and the statement that it gates nothing
- **AC-3:** WHEN a reader opens `core/skills/spec-format/SKILL.md` THE SYSTEM SHALL state that the bar is checkable-by-the-model-at-delivery-time and explicitly NOT "has a linked re-runnable test"
- **AC-4:** THE SYSTEM SHALL reference the model-checkable rule from `feature-delivery-lead.md`, `platform-delivery-lead.md`, `design.md`, and `diagnose.md`, with the full rule text appearing only in `spec-format`
- **AC-5:** WHEN a spec containing a `## Unchecked` section is parsed by `internal/spec.ParseAcceptanceCriteria` THE SYSTEM SHALL return zero criteria from that section, and `hero spec verify` SHALL NOT emit a ledger row for any of its entries
- **AC-6:** WHEN `hero install --target <t>` runs for each of `opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic` into a scratch workspace THE SYSTEM SHALL materialize the `spec-format` skill containing the model-checkable rule in every one of the six
- **AC-7:** WHEN `internal/spec.ParseAcceptanceCriteria` parses this spec THE SYSTEM SHALL return exactly seven criteria with the stable IDs `AC-1` through `AC-7`

Each of these names an actuator I run (open the file and grep it; `go test`; `hero install`) and an observable I read (matched text; returned slice length and IDs; materialized file contents). AC-5 is the one that matters most — it is the claim the entire "no code changes needed" design rests on, and it is a `go test` away from proven or refuted.

Note on AC-7's phrasing: it asserts addressability, not EARS classification. `hero spec lint` reports these seven as `freeform` — not because they aren't EARS, but because of a confirmed upstream defect described in Risks. Writing an AC against the lint output would be asserting something I know to be misreported.

## Unchecked

Observations, not criteria. Nothing here gates completion. Nobody is obligated to act on any of it.

- **Whether delivery leads actually apply the rule when authoring future specs.** This is the spec's central behavioral claim and I cannot check it. The actuator would be "author many specs across many future sessions and measure the AC quality distribution"; there is no observable I can capture at delivery time. Writing it as an AC would require a human to go read future specs and report back — which is the precise defect this spec exists to delete. **This entry is the spec obeying its own rule on the one criterion it would most like to claim.**
- **Whether the rule's phrasing is clear enough to prevent judgment calls in practice.** I can confirm the text exists and says what it says. Whether it lands unambiguously with a model reading it cold in six months is not observable from here. First contradictory reading is the signal to sharpen it.
- **Whether the fraction of the 250 evidence-free completed specs attributable to this ceremony is large or small.** Determining it means reading 250 specs and judging intent per stamped row. Not checked, not blocking, and forward-only scope makes it moot for this delivery.

## Worked example — before / after

**Before** (real, `.hero/specs/coverage-heuristic-fix/spec.md:206-208`):

> - WHEN the text report contains weak matches THE SYSTEM SHALL display them in a separate "Weak matches" section beneath Gaps so they are **visually distinct from strong coverage**

The clause `so they are visually distinct from strong coverage` fails the test. Name the actuator: *the user looks at it.* Wrong subject. Name the observable: *whether it seems distinct to them.* A perception, not a state. There is no pass/fail here — this is unfalsifiable, not merely unverified.

**After** — the structural assertion underneath it, which is fully checkable:

> - WHEN the text report contains weak matches THE SYSTEM SHALL display them under a `Weak matches` heading positioned after the `Gaps` section, with strong matches under no such heading

Actuator: run `hero coverage` on a fixture with one weak and one strong match. Observable: the heading string is present in stdout and its byte offset is greater than that of `Gaps`. That is a unit test.

**The residue** that lands on `## Unchecked`:

> - **Whether the "Weak matches" section reads as visually distinct to a user.** I can assert the section exists, is titled, and sits beneath Gaps. Whether that separation *reads* as distinct is a perceptual judgment with no observable I can capture.

Note what happened: the AC got **stronger**, not weaker. Applying the rule forced the vague intent ("distinct") into a concrete structural claim ("this heading, after that one"). The uncheckable part didn't disappear — it went where it can't lie. That is the pattern to expect: most violating ACs are not impossible to check, they are **imprecisely written**, and the rule is a precision forcing-function.

**Secondary case — misaddressed labor** (`.hero/specs/knowledge-export-cli/spec.md:157-163`). Five numbered `Manually verify from a temporary workspace` steps: create sample files, run `hero export knowledge /tmp/...`, confirm relative paths, re-run and confirm conflict behavior. **Every one of these is something the model can just do.** Nothing is unverifiable; the steps were addressed to a human for no reason. Under the rule these do not go to `## Unchecked` — they become ACs and the model runs them. This is the more common failure and the cheaper win.

## Boundaries — explicit non-goals

- **No `PENDING` ledger status, no new AC state, no Gate 1 surgery, no blocking-policy knob.** All unnecessary: if no unverifiable AC is written, the gate has nothing to trip on. Earlier drafts proposed all of it — that was machinery to administer a channel that shouldn't exist.
- **No inbox or dashboard surgery.** `LoadInbox` (`internal/serve/pages/now/data/inbox.go:32-49`) already has five sources. A cross-spec view of unchecked items is a later query against a corpus that does not exist yet. Build the corpus first.
- **No rewrite of the 250 existing specs.** Forward-only. History stays as it is, including the stamped rows. Rewriting them would itself be a mass re-attestation with no new information — the same defect at scale.
- **Not the self-attestation defect.** `recordLedgerToGraph` (`internal/cli/verify.go:596-601`) turning a checkbox into `Status: "pass"` is real, separate, and lives in `hero-self-consistency` territory. This spec removes the worst *occasion* for that mechanism to produce a false record; it does not touch the mechanism. A model that stamps `DONE` on a machine-checkable AC it never ran is still lying — this spec just stops handing it a category of AC where lying is the *expected* path.
- **No lint rule or validator.** Argued above under "Guidance, not enforcement."
- **No change to `needs_me`, `/drive`, or the pause taxonomy.** They are the model this spec is conforming to, not a target of it.
- **Go-engine install targets only — the rule will NOT reach `hero-code`, and AC-6 will not notice.** AC-6 verifies six *harness targets*; it verifies zero *engine implementations*. `hero-code` is not a Hero client — it is a second Hero implementation in Rust (`crates/hero-cli`, `hero-core`, `hero-gtk`) that vendors its own `spec-format` skill, already drifted 130 lines from this one (365 vs 495), with zero `## Unchecked` sections in its 663 specs. `hero install` from this engine does not feed it. So AC-6 can pass green while the rule is entirely absent from the codebase with the highest concentration of temporal ACs — the exact category this spec added a failure mode for. This is a live cross-implementation propagation gap, tracked separately; do not let AC-6's green be read as coverage.

## Risks

- **The rule gets read as "only write ACs that have automated tests."** This is the main failure mode and it would be expensive — it implies a 250-spec migration and would push authors to *drop* legitimate ACs they can verify by hand. Mitigation: the "checkable ≠ has a linked test" clarification is called out as its own AC (AC-3), stated in the rule's first paragraph, and reinforced by the one-shot/screenshot examples.
- **`## Unchecked` becomes a dumping ground** for ACs the model finds inconvenient. The rule asks "can I check this?", not "do I want to?" — and the honest answer is almost always yes. Mitigation: the entry format demands *why the model couldn't check it*, which is hard to write dishonestly; "there's no test harness" is not a valid reason and the skill says so explicitly.
- **Drift between the rule text in `spec-format` and the four pointers.** Mitigation is structural: the pointers do not restate the rule. If they only reference, they cannot drift.
- **Six-target propagation regresses silently** if the skill tree layout changes. Mitigation: AC-6 checks all six explicitly rather than assuming pack-level authoring implies coverage — assuming it is exactly what the tripwire exists to catch.
- **Confirmed upstream defect: AC-N labels and EARS classification are mutually exclusive.** Verified in this repo at design time:

  | AC form | `ClassifyCriterion` | `ParseAcceptanceCriteria` |
  |---|---|---|
  | `- **AC-1:** WHEN …` | `freeform` | 1 criterion |
  | `- WHEN …` | `event` | 0 criteria |

  Cause: `ClassifyCriterion` (`internal/spec/spec.go:1424`) matches EARS via `strings.HasPrefix(upper, "WHEN ")` and never strips the `**AC-N:**` label, while `parseACBlock` (`internal/spec/acceptance.go:76`) requires that label to emit an addressable criterion. An author cannot satisfy both consumers today: label your ACs and they are machine-addressable but always report as `freeform`; omit the label and they classify as EARS but yield zero addressable criteria — and therefore zero coverage rows and zero graph nodes.

  **This is out of scope here and must not be fixed in this spec** — it is `hero-self-consistency` territory (related). It matters to this spec only because the rule tells authors to write checkable ACs, and the corpus's `freeform` ratios are currently a misleading signal for anyone judging AC quality. This spec chooses addressability over classification and says so. Flagged for separate work.

## Validation

- `go test ./internal/spec` — covers AC-5 (the `## Unchecked` inertness claim). Add a case to `internal/spec/acceptance_test.go` asserting a spec with both `## Acceptance Criteria` and `## Unchecked` returns only the former's criteria.
- `hero install --target <t>` into a scratch workspace for each of the six targets, grepping the materialized `spec-format` skill for `The model-checkable rule` — covers AC-6.
- `go test ./internal/spec` asserting `ParseAcceptanceCriteria` on this spec returns seven criteria with IDs `AC-1`..`AC-7` — covers AC-7.
- Grep assertions on the five edited markdown files — covers AC-1 through AC-4.

Every step above is one I run. There are no manual verification steps in this spec, which is the minimum bar for a spec whose entire thesis is that manual verification steps do not belong in specs.

## Mission fit

> "Does this make the next agent session start smarter than the last one ended, and does it raise the floor for everyone?"

**Yes, directly.** It stops the corpus accumulating `pass` records whose provenance is a human saying "sure" to a question the model invented. A corpus with false evidence in it is worse than a smaller honest one — the next session cannot tell which `pass` records mean anything, so it must discount all of them.

This spec removes false evidence rather than adding true evidence. That is the cheaper half of the same goal, and it raises the floor specifically for the person who *doesn't* already know which stamped rows to distrust.
