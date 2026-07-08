---
title: Delivery Gate Consistency — One Owner for the Ledger Contract and the Verify-Gated Close
slug: delivery-gate-consistency
type: enhancement
status: planning
priority: P1
size: medium
domain: engineering
created: 2026-07-06
tags: [content, delivery-gates, completion-ledger, verify, agents, commands, audit-remediation]
relations:
  - target: content-remediation
    kind: parent
  - target: hero-content-audit
    kind: related
---

# Delivery Gate Consistency — One Owner for the Ledger Contract and the Verify-Gated Close

## Context

The delivery doctrine — a spec reaches `completed` only through `hero spec
verify`, which checks four gates (Completion Ledger, delivery audit, coverage,
build/tests) — is declared mandatory by `domains/engineering/commands/deliver.md`
("**`hero spec verify` is the only path to `completed`**"), by
`domains/engineering/AGENTS.md` rule 5, and by the test-locked Go fallback in
`internal/install/agents_md.go`. The engine enforces it too:
`internal/cli/complete.go:60-67` rejects `hero spec complete` on a work spec
and redirects to verify.

Three shipped files contradict or undermine that doctrine
([[hero-content-audit]] findings-agents.md S1/S2, findings-commands.md F8):

1. **`domains/engineering/agents/platform-delivery-lead.md`** is a stale fork
   of feature-delivery-lead's delivery procedure. Its step 13 (line 79)
   instructs "move the spec from `planning/` to `specs/` and update its status
   to `completed`" — hand-editing past all four gates. It has no Completion
   Ledger validation, no cold-audit pass, no verify gate, and no delivery
   modes. Any platform-scoped `/deliver` (routed there by `design.md:16`,
   `compose.md:8`, `split.md:8`, `core/commands/retro.md:8`) bypasses the
   entire closing sequence.
2. **`domains/engineering/commands/sprint.md`** execute mode step 4c (line 48)
   instructs `hero spec complete <spec-path>` directly after each autopilot
   delivery — an instruction the CLI itself now rejects for work specs.
   Execute mode also duplicates `/drive` (which post-dates it) with weaker
   guardrails.
3. **The Completion Ledger format** — the contract `hero spec verify` Gate 1
   machine-parses — lives as ~450 words inside one agent's body
   (`domains/engineering/agents/engineer.md:105-169`) but is consumed by
   feature-delivery-lead (step 17), deliver.md (twice), the delivery-audit
   skill, and the Gate 1 parser (`internal/spec/ledger.go`,
   `internal/cli/verify.go:checkLedger`). Cross-referencing an agent file for
   a format contract is how the platform fork rotted; the same drift already
   produced two documentation-vs-parser mismatches (see Approach).

Post-`177e8a1` (content-dedup-resync), core/ is the single master for
previously duplicated files and `content_parity_test.go` gates any
domain-pack shadow of a core path. feature-delivery-lead.md and
platform-delivery-lead.md were not part of that dedup set and still live only
in `domains/engineering/agents/`. Verified against the current tree: `hero
spec verify --help` shows the four gates plus `--skip-tests/--force/--json`;
`hero verify` (no `spec`) is **not** a command; `hero spec complete` documents
no gates of its own.

## Goal

The delivery doctrine is stated once and cited everywhere else. The
Completion Ledger contract lives in a single skill that matches what Gate 1
actually parses; engineer.md, both delivery leads, deliver.md,
delivery-audit, and agent-reliability cite it instead of restating or
forking it. platform-delivery-lead can no longer instruct a gate bypass, and
sprint.md no longer instructs `hero spec complete`. A repo-wide grep finds no
shipped engineering-pack instruction to hand-edit `status: completed` or to
complete a work spec outside `hero spec verify`.

## Kickoff

Makes the four-gate delivery close (`hero spec verify`) consistent across the
engineering pack: extracts the Completion Ledger contract to a core skill,
rewrites platform-delivery-lead as a thin delta on feature-delivery-lead, and
deletes sprint.md's gate-bypassing execute mode in favor of `/drive`.

**Status:** planning — spec authored from audit findings; no edits yet.

**Pick up at:** Change 1 — create `core/skills/completion-ledger/SKILL.md`
from `engineer.md:105-169`, reconciled against the Gate 1 parser.

→ `/deliver delivery-gate-consistency`

**Files:** `domains/engineering/agents/engineer.md:105-169`,
`domains/engineering/agents/platform-delivery-lead.md:61-80`,
`domains/engineering/commands/sprint.md:33-60`, `internal/spec/ledger.go`,
`internal/cli/verify.go:207`
**Skip:** compressing feature-delivery-lead/deliver.md verbosity — that is
[[token-efficiency-pass]]'s job (wave 3, depends on this extraction).

## Approach

### 1. Extract the ledger contract to `core/skills/completion-ledger` (S2 fix)

**Placement: `core/skills/`, not `domains/engineering/skills/`.** Three
reasons: (a) the contract's machine consumer is domain-agnostic engine code —
`internal/spec/ledger.go` + Gate 1 in `internal/cli/verify.go` run in any
workspace, whatever pack is installed; (b)
`core/skills/agent-reliability/SKILL.md:41` already points at the format
("see the engineering pack's `engineer.md` for the format; packs without an
engineer agent account for the same items explicitly in prose") — a core →
pack reference that dangles on pm/sales installs (install merges core + one
domain only). Core placement turns it into a core → core reference that
resolves everywhere while keeping the prose fallback for packs without an
engineer. (c) Post-dedup doctrine says core is the single master;
`content_parity_test.go` only constrains domain shadows of core paths, so a
new core skill is clean.

**The skill must match the machine contract**, verified against
`internal/spec/ledger.go` and `checkLedger` in `internal/cli/verify.go`:

- Section shape the parser finds: a `## Completion Ledger` section
  (case-insensitive key), with `### Acceptance Criteria` and `### Changes`
  sub-tables (3 or 4 columns; 4-column = `# | Summary | Status | Note`),
  plus `### Exercise-the-feature check` and `### Excellence Bar self-check`
  checkbox blocks (matched on "exercise"/"excellence" in the header).
- Statuses: `DONE` / `PARTIAL` / `SKIPPED` / `BLOCKED` (case-insensitive,
  bold/backticks tolerated). Anything else parses as `UNKNOWN` and **fails
  Gate 1** — a real reason not to invent statuses.
- **`[signed-off]` in the Note cell** is the machine-readable sign-off that
  lets a `SKIPPED`/`BLOCKED` row pass Gate 1. This is currently documented
  **nowhere** in shipped content — engineer.md says sign-off is required but
  never says how the machine reads it. The skill documents it.
- **Exercise-the-feature is advisory at the gate.** `checkLedger` emits
  `ADVISORY:` details for a missing/bare check and does not fail.
  engineer.md:137-139 claims "a bare `[x]` with no description will fail
  Gate 1" — false today. The skill states the true contract (advisory at the
  gate, mandatory by convention; the delivery lead still requires it for
  user-visible `DONE` rows) rather than perpetuating the overstatement.
  Hardening the gate in Go is a possible follow-up, not this spec (see
  Boundaries).
- Gate 1 pass condition, stated plainly: ledger present + every AC row and
  every Changes row `DONE` or signed-off `SKIPPED`/`BLOCKED`.

The skill carries **both sides of the contract**: the engineer's authoring
format (tables, status definitions, honesty rules, optional preamble — moved
from engineer.md) and the validator's rules (evidence bar for `DONE`,
PARTIAL-is-not-an-end-state loop-back, SKIPPED/BLOCKED escalation, never
flip `status:` by hand — `hero spec verify` is the only path). Every current
consumer then cites it.

### 2. platform-delivery-lead: thin delta, not fold-in (S1 fix)

**Recommendation: keep the agent, delete its forked delivery procedure.**
The audit's MERGE options were (a) thin delta on the shared procedure or
(b) fold into feature-delivery-lead with a platform mode. Thin delta wins:

- Routing surface stays stable: `design.md`, `compose.md`, `split.md`,
  `core/commands/retro.md`, plus `roadmap-review` and `migration-safety`
  skills and the README roster all name `platform-delivery-lead`. Fold-in
  means editing 7+ referencing files, shrinking the roster (README counts
  are checked by `hero docs check`), and losing the distinct trigger
  description harness agent-pickers select on.
- The drift mechanism was the **fork**, not the second name. A delta with
  zero procedural steps has nothing to rot — the platform file will contain
  emphases, not a copy of the pipeline.

Concretely, platform-delivery-lead.md keeps its frontmatter, identity,
design-phase content (platform spec-quality rules: ordered migration steps,
rollback in Risks) and platform-specific Rules/Default output. Its delivery
phase is replaced by a citation block: load the same skills
(`context-injection`, `agent-reliability`), then **follow
feature-delivery-lead's "Delivery phase" verbatim — same modes
(supervised/autopilot/dry-run), same steps 1-21, same Completion Ledger
validation (per the `completion-ledger` skill), same cold audit, same
`hero spec verify` close** — followed by 3-4 platform delta bullets that
modulate emphasis, not procedure (sequence to minimize migration/rollout
risk; always involve migration-engineer on rollback-risky work;
brownfield-architect before structural change). Cross-agent file references
are established practice (feature-delivery-lead cites engineer.md today);
both files install side by side.

### 3. sprint.md: delete execute mode in favor of `/drive` (F8 fix)

**Recommendation: delete, don't patch.** Fixing step 4c to `hero spec
verify` would leave a redundant third autonomous loop: `/deliver
--autopilot` already runs verify as its own closing gate (deliver.md "MUST
run" block), so sprint's post-delivery completion step is dead weight even
when corrected — and the surrounding loop duplicates `/drive`, which
post-dates it and does strictly more (progressive design of unscaffolded
children, deterministic `needs_me` pauses via `hero goal --check`, hard
caps, resume-on-disk). Sprint execute only operates on initiatives ("Load
the initiative and list its child specs") — exactly `/drive`'s target.
Replace the section with a short routing note: initiative-shaped sprint →
`/drive <initiative>`; ad-hoc list of ready specs → `/deliver` queue mode
("deliver these while I'm away"). Planning mode (steps 1-7 + principles) is
distinct and stays untouched.

### 4. Sweep the residual contradictions in the doctrine's own files

Gate-machinery-only edits (no verbosity work): feature-delivery-lead's
Session wrap-up step 1 says "Fully complete → `completed` (and use `hero
spec complete`)" (line 164) — contradicting its own step 19 and the CLI
redirect; step 19 names a nonexistent `hero verify` (line 136); deliver.md
batch mode says "Only set `status: completed` if the ledger is fully `DONE`"
(line 105) where the verify gate is the actual mechanism. All are repointed
at verify / the new skill.

## Changes

1. **Create `core/skills/completion-ledger/SKILL.md`** — single owner of the
   Completion Ledger contract. Frontmatter: `name: completion-ledger`,
   description ("Format and validation contract for the Completion Ledger —
   the closing artifact hero spec verify Gate 1 parses"),
   `compatibility: opencode`, `metadata: {audience: engineer,
   delivery-leads, auditors; purpose: delivery-gate-contract}`. Body:
   - Format block moved from `engineer.md:113-144` (both tables,
     exercise/excellence checks, optional preamble), status definitions
     (from lines 146-151), honesty rules (from lines 153-158).
   - New **"What Gate 1 actually parses"** subsection matching
     `internal/spec/ledger.go` + `internal/cli/verify.go:checkLedger`:
     section/sub-header names, 3-vs-4-column tolerance, the
     `UNKNOWN`-status failure mode, the `[signed-off]` Note annotation for
     SKIPPED/BLOCKED rows, and the corrected advisory status of the
     exercise check (drop the false "bare `[x]` fails Gate 1" claim).
   - New **"Validating a ledger"** subsection (the lead's side): DONE
     evidence bar, PARTIAL loop-back (not an end state; chase, don't ask),
     SKIPPED/BLOCKED escalation with `[signed-off]`, and the closing rule:
     `hero spec verify <slug>` is the only path to `completed` — never
     hand-edit `status:` or run `hero spec complete` on a work spec.
2. **Edit `domains/engineering/agents/engineer.md`** — replace the body of
   "## Closing output — the Completion Ledger" (lines 105-169) with a
   compressed section (~80-100 words): the mandate sentence (ledger is the
   mandatory final artifact), "load the `completion-ledger` skill and follow
   its format exactly — `hero spec verify` Gate 1 machine-parses it", and
   the two behavioral non-negotiables (every AC + Changes item gets a row;
   honest non-`DONE` beats performative `DONE`). Line 22's always-load list
   is unchanged (the skill is loaded at closing time, not startup).
3. **Edit `domains/engineering/agents/platform-delivery-lead.md`** — thin
   delta per Approach §2: keep frontmatter/identity/design phase/spec
   quality rules; replace "## Delivery phase" steps 1-14 (lines 61-80) with
   the shared-procedure citation block + platform delta bullets; step 13's
   hand-edit instruction is deleted with the fork. Keep "## Rules" and
   "## Default output".
4. **Edit `domains/engineering/commands/sprint.md`** — delete "## Execute
   mode" and "**Safety rails:**" (lines 33-59); insert "## Executing the
   sprint" (~5 lines): initiative → `/drive <initiative>` (autonomous,
   needs_me pauses); ad-hoc spec list → `/deliver` queue mode. No
   `hero spec complete` reference remains.
5. **Edit `domains/engineering/agents/feature-delivery-lead.md`** —
   gate-machinery only: (a) step 17 line 124: "(see `engineer.md` —
   'Closing output')" → "(see the `completion-ledger` skill)"; (b) step 19
   line 136: "`hero verify` checks four gates" → "`hero spec verify` checks
   four gates"; (c) Session wrap-up step 1 line 164: "Fully complete →
   `completed` (and use `hero spec complete`)" → "Fully complete → run
   `hero spec verify <slug>` (step 19) — never hand-edit `completed` or use
   `hero spec complete` on a work spec". No compression of steps 17-18
   prose.
6. **Edit `domains/engineering/commands/deliver.md`** — gate-machinery only:
   (a) batch step 4 line 103 and single-spec step 5 lines 156-158: repoint
   "(see `engineer.md` — 'Closing output')" → the `completion-ledger` skill;
   (b) batch step 4 line 105: "Only set `status: completed` if the ledger is
   fully `DONE`" → "Run `hero spec verify <slug>` once the ledger is fully
   `DONE`" (keep the PARTIAL/SKIPPED/BLOCKED routing sentence and the
   auto-archive note). No other deliver.md changes.
7. **Edit `core/skills/agent-reliability/SKILL.md`** line 41 — repoint the
   scoped sentence: "see the engineering pack's `engineer.md` for the
   format" → "see the `completion-ledger` skill for the format"; keep the
   "packs without an engineer agent account for the same items explicitly
   in prose" caveat verbatim.
8. **Edit `domains/engineering/skills/delivery-audit/SKILL.md`** line 38 —
   append a one-line pointer to input 3 ("Ledger — the engineer's Completion
   Ledger, pasted verbatim"): "(format contract: `completion-ledger`
   skill)" so the cold auditor can judge ledger validity against the owner.

## Boundaries

- **No verbosity/compression work** in feature-delivery-lead.md or
  deliver.md beyond the exact line edits above — the audit's S2
  token-efficiency findings on those files (step 4d sizing prose, step 18
  audit restatement, deliver.md's triple definition-of-done, batch/queue
  merge) belong to [[token-efficiency-pass]] (wave 3), which cuts against
  this spec's extracted skill. Surgical edits here keep that diff clean.
- **No engine changes.** `internal/spec/ledger.go`, `internal/cli/verify.go`,
  and `internal/cli/complete.go` are read-only inputs — the skill conforms
  to the parser, not vice versa. Hardening the advisory exercise check into
  a blocking gate is a separate engine decision if wanted.
- **`domains/engineering/AGENTS.md` and `internal/install/agents_md.go`**
  are untouched — rule 5 is already a correct one-paragraph citation of the
  doctrine, and the identity test makes any edit a lockstep dual-edit owned
  by [[routing-file-completeness]] / [[harness-agnosticism-sweep]].
- **`domains/sales/commands/debrief.md:66`** (`hero spec complete <slug>` on
  a deal spec) is sales-pack reality-sync territory
  ([[sales-pack-reality-sync]]), not engineering gate doctrine.
- **Other platform-delivery-lead findings** ({hero_folder} placeholder S3,
  frontmatter `domains:` inertness) ride along only where the delta rewrite
  naturally removes the text; no dedicated fixes.
- **No roster changes**: platform-delivery-lead keeps its name, description,
  and permission block — README counts and the 6 external references stay
  valid.

## Risks

- **Thin delta under-specifies if the citation is vague.** A
  platform-delivery-lead session must actually open feature-delivery-lead.md
  to get the procedure. Mitigation: the citation block names the exact
  section ("Delivery phase", steps 1-21) and file, and restates the four-gate
  close in one sentence so even a lazy read can't miss the gate.
- **Skill/parser mismatch survives extraction.** The whole point is matching
  the machine contract; a paraphrase that drifts from `ledger.go` recreates
  the problem. Mitigation: Validation step 3 checks the skill's format
  example against `internal/spec/ledger_test.go` fixtures and `checkLedger`
  behavior line by line.
- **`hero docs check` count drift.** Adding a core skill may change counts
  that README/GETTING-STARTED claim. Mitigation: run `hero docs check`
  during delivery and update the numeric claims if flagged.
- **Wave conflicts.** token-efficiency-pass (wave 3) edits the same two big
  files; harness-agnosticism-sweep (wave 2) touches deliver.md's
  "general-purpose subagent" phrasing (F23). This spec's edits are
  line-scoped and land first (wave 1), so later diffs rebase cleanly —
  but delivery should re-verify cited line numbers against HEAD before
  editing (the audit evidence predates some churn).
- **Muscle-memory break on `/sprint execute`.** Users who used it lose the
  verb. The replacement pointer keeps the phrase discoverable and routes to
  a strictly safer loop.

## Acceptance Criteria

- THE SYSTEM SHALL state the Completion Ledger format contract in exactly one file, `core/skills/completion-ledger/SKILL.md`, with `engineer.md`, `feature-delivery-lead.md`, `deliver.md`, `delivery-audit/SKILL.md`, and `agent-reliability/SKILL.md` citing it and no shipped file restating the table/status format.
- THE SYSTEM SHALL document in that skill the `[signed-off]` Note annotation, the UNKNOWN-status failure mode, and the advisory-at-the-gate status of the exercise-the-feature check, each matching `internal/spec/ledger.go` and `checkLedger` in `internal/cli/verify.go`.
- WHEN a delivery is coordinated by `platform-delivery-lead` THE SYSTEM SHALL direct it through the identical closing sequence as `feature-delivery-lead` (ledger validation → cold audit → `hero spec verify`), with no instruction to hand-edit `status: completed` or move the spec to `specs/`.
- WHEN `/sprint` is invoked with execute/run/go arguments THE SYSTEM SHALL route initiative execution to `/drive` and ad-hoc spec lists to `/deliver` queue mode instead of running a bespoke loop.
- THE SYSTEM SHALL contain no engineering-pack or core content instructing `hero spec complete` on a work spec or hand-editing `status: completed` (prohibition mentions in `diagnose.md` and `debug-investigator.md` excepted).
- WHEN `hero install` runs for any available domain THE SYSTEM SHALL install the `completion-ledger` skill, so `agent-reliability`'s format reference resolves on pm and sales installs.
- IF `go test ./...` runs after delivery THEN THE SYSTEM SHALL pass, including `content_parity_test.go` and the AGENTS.md identity test.

## Validation

1. `go test ./...` — parity gate (new core skill, no domain shadow) and
   AGENTS.md identity test (untouched files) both green.
2. Greps return empty:
   `rg -n "hero spec complete" core/ domains/engineering/ --glob '*.md'`
   (excluding the two prohibition mentions);
   `rg -n "engineer.md.*Closing output" core/ domains/`;
   `rg -n "update its status to .completed.|and use .hero spec complete." domains/engineering/agents/`;
   `rg -n "hero verify " domains/engineering/agents/feature-delivery-lead.md`.
3. Contract cross-check: paste the skill's format example into a scratch
   spec and confirm `ParseLedger` semantics against
   `internal/spec/ledger_test.go` fixtures — table columns, statuses,
   `[signed-off]`, exercise/excellence detection all as documented.
4. `hero install --target claude --domain pm` (scratch dir): the
   `completion-ledger` skill installs and `agent-reliability`'s reference
   resolves; repeat for `--domain engineering`.
5. `hero docs check` — no numeric-claim drift from the added skill (fix
   README/GETTING-STARTED counts if flagged).
6. Manual read of the new platform-delivery-lead delivery phase: citation
   names file + section + steps, four-gate close restated in one sentence,
   delta bullets contain no procedural steps.
