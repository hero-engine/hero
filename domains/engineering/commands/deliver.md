---
description: Execute a spec — implement the planned changes, validate, and complete the work item.
---
Route this delivery request to the `feature-delivery-lead` agent for execution.

Be the `feature-delivery-lead` agent. Load the `context-injection` skill before starting.

**Initiative guard.** If the requested target resolves to a `type: initiative`,
do NOT deliver one child. An initiative is a parent — running the whole thing
autonomously is `/drive`, not `/deliver`. Offer the upgrade: "That's an
initiative — `/deliver` works one spec at a time. Want to `/drive` the whole
thing (autonomous, pauses when it needs you) instead?" Let the user pick;
never silently deliver a single child. (`hero spec deliver` enforces this at
the CLI layer too.)

**Before starting work**, emit a `hero next ask` capturing what the user
asked for. This preserves session intent across compaction — see the
`next-handoff-emit` skill for the full pattern (ask / suggest / reflection).

## Definition of done

A delivery is **not** finished until `hero spec verify <slug>` passes — and
verify requires the cold audit (step 6 below) to have run first. Until that
gate passes the work is in-flight, not delivered: do not report "done," do not
suggest next steps, and do not leave the spec in `planning`/`delivering` with
the audit unrun.

The closing gates run in the **same turn** as the implementation — not a
follow-up the user triggers later. Catching yourself about to say "the
audit still needs to run" is the signal to **run the gate now**, not yield.

This holds in **every** mode, including default supervised: the
persistence rule (`agent-reliability`) applies regardless of mode, and
"pause at handoffs" does not include the closing gates — they're part of
finishing, not a decision to surface.

## Delivery modes

Parse the mode from the user's invocation. If no mode is specified, default
to **supervised**.

| Flag | Behavior |
|---|---|
| `--supervised` (default) | Pause at handoffs, surface decisions, ask before destructive actions. The closing gates are not handoffs — see Definition of done above. |
| `--autopilot` | Run to completion without intermediate confirmations. Stop only on test failure, drift warning, boundary violation, or any non-`DONE` Completion Ledger item (PARTIAL, SKIPPED, BLOCKED). |
| `--dry-run` | Run analysis and planning. Produce a delivery plan (file list, agent assignments, estimated changes) but write NO code. |

### Autopilot mode

Suppress confirmation prompts during the delivery loop. Check `hero drift`,
run tests, validate the Completion Ledger, and: **PARTIAL** loops back to
the engineer (not a halt — only escalate after a second pass with a
written obstacle); **SKIPPED / BLOCKED** halt and surface; **test
failure / drift warning / boundary violation** halt and surface.
`--halt-on` (comma-separated: `drift`, `test`, `boundary`, `lint`,
`ledger`) narrows this to only the listed conditions.

Persistence rule (`agent-reliability`) applies: don't yield between
phases unless a true blocker fires. PARTIAL is not a blocker; SKIPPED
and BLOCKED are.

### Dry-run mode

Produce a plan at `.hero/planning/features/<slug>/plan.md`:

```markdown
# Delivery plan — <slug>
Generated: <timestamp>
Mode: dry-run

## Tasks (sequenced)
1. [specialist] description → target-file (~N lines)
2. ...

## Risks
- boundary checks, drift baseline, potential conflicts

## Estimated complexity
~N lines, N specialist handoffs, ~N minutes wall time
```

Do NOT write any source files. The plan file is the only output.

## Multi-spec mode

Covers both "fix the researched bugs" / "deliver the 5 planned features"
(ad hoc batch) and "deliver these while I'm away" / "queue these up" /
a slug list with `--autopilot` (queue variant). Follow the
`batch-discipline` skill for concurrency, per-item isolation, the
after-completion verification loop, and the summary-table format; the
rules below are what's specific to delivery.

1. **Find candidates.** Batch: `hero search --list --type bug --status planning`
   (or appropriate type/status), filtered to specs with a `## Suggested Fix
   Approach` or `## Changes` section. Queue: validate the given slugs are
   ready (have Changes, pass the quality gate) and sort by dependency order
   via `hero_sequence`.
2. **One confirmation before starting**, naming the full ordered list and
   the halt policy (test failure, boundary violation, non-`DONE` ledger item).
3. **Work through them in order — never parallelize** (overlapping-code
   conflict risk). Per spec: implement, test/build, require and validate a
   **Completion Ledger** (`completion-ledger` skill), commit atomically
   (one commit per spec, referencing slug/tracker ID), post to tracker if
   configured.
4. **PARTIAL rows**: loop the engineer to finish before moving to the next
   spec — never advance with PARTIAL unhandled. **SKIPPED / BLOCKED rows or
   a test/drift failure**: log it, skip the spec (do NOT flip to
   `completed`), continue with the next.
5. Run `hero spec verify <slug>` once a spec's ledger is fully `DONE` —
   this is what flips status and archives it.
6. At the end, summarize: delivered (`DONE`), halted (SKIPPED/BLOCKED, for
   user judgment), failed (with reasons). PARTIAL should not appear —
   it should have been resolved in step 4.

## Single spec mode

For a single spec delivery, follow the standard pre-flight and delivery workflow as the `feature-delivery-lead` agent.

**Pre-flight — check for mockups.** Before delegating to the engineer, check whether `.hero/mocks/{slug}/` exists for the spec being delivered. If it does, list the file paths inside it in the kickoff context handed to the engineer (e.g. "Mockups available for this spec: `.hero/mocks/{slug}/index.html` — read if a visual reference helps."). Don't render or quote the HTML; let the engineer open it if useful. The same check applies in multi-spec mode — surface mockup paths in the kickoff for each spec.

At the end of the delivery loop:

1. **Check for drift** — verify the implementation matches the spec. Address
   any divergence before moving on.
2. **Verify and expand test coverage** — every acceptance criterion needs
   a corresponding test; a bug fix needs a regression test reproducing
   the original failure. Cover adjacent code you touched too, matching
   the project's existing patterns — leave coverage better than you
   found it.
3. **Run the test suite** — make sure everything passes, including the
   tests you just wrote and any existing tests for the affected area.
4. **Link tests to criteria** — connect each new test back to the
   acceptance criterion it verifies so regressions are traceable.
5. **Validate the Completion Ledger** (contract: `completion-ledger`
   skill) before flipping spec status:
   - Confirm every acceptance criterion and every Changes item has a row;
     cross-check each `DONE` against code/test evidence on disk and
     downgrade performative claims.
   - **PARTIAL is not an acceptable end state** — send it back to the
     engineer with explicit instructions to finish; only surface to the
     user after a second pass still returns PARTIAL with a named,
     specific obstacle.
   - **SKIPPED / BLOCKED** are legitimate human-judgment escalations —
     surface with the row, the engineer's reason, and a recommendation.
   - Do NOT flip to `completed` while any row remains non-`DONE`.
6. **Cold audit pass** — once the ledger is fully `DONE` (or non-`DONE`
   rows have explicit user sign-off), spawn a **fresh** subagent with the
   `delivery-audit` skill loaded — the lead cannot grade its own homework.
   Hand it ONLY artifacts on disk: spec path, diff command
   (`git diff <base>...HEAD`), the Completion Ledger verbatim, and test
   evidence. The skill defines the report format, the
   `<AUDIT_VERDICT>` / `<AUDIT_HEADLINE>` / `<AUDIT_HIGHLIGHTS>` return
   blocks, and the signal-preservation rule (headline always shown,
   highlights conditional) — do not restate it here.

   Route on the verdict:
   - **HOLD** → do NOT flip to `completed`. Route the specific concerns
     back to the engineer, re-validate, re-audit. **Bounded retry:** stop
     after 2 engineer passes on the same row and escalate to the user
     with a concrete recommendation instead of grinding indefinitely.
   - **SHIP** → quote the full `<AUDIT_HEADLINE>` in your final response
     (plus `<AUDIT_HIGHLIGHTS>` if noteworthy). Proceed to step 7.

   Do not skip this step in supervised mode.
7. **Refresh the kickoff** — when status flips (planning → delivering,
   delivering → completed) or after meaningful chunks land, rewrite the
   spec's `## Kickoff` section so "Pick up at:" reflects *now*, not
   "two commits ago." Follow the `kickoff-prompt` skill. After mutating
   the spec, run `hero index --if-stale -q` so the new state surfaces
   in `hero search` / `hero list` / `hero_*` MCP tools immediately.
   The pre-commit hook will refresh `.hero/QUEUE.md` automatically; if
   you're not committing right away, run `hero queue write -q` to
   refresh the snapshot manually.
8. **Suggest what's next** — end your final response with a single
   concrete "Next up" recommendation, named specifically — not an option
   list, not "let me know." E.g. "Next up: deliver `<dep-slug>` — it's
   the only remaining blocker on `<parent-slug>`." Use `hero_kickoff` /
   `hero_pulse` if uncertain. Emit via `next-handoff-emit` so it lands in
   `.hero/NEXT.md` and the next session resumes with it visible.

`hero spec verify <slug>` (see Definition of done above) is the only path
to `completed` — it checks four gates (ledger, audit report, test
coverage, build), writes AC pass statuses to the knowledge graph,
auto-completes parent initiatives when all children land, and archives
the spec. **Do not edit `status: completed` directly.** If verify returns
FAIL, read the gate failures, route them back to the engineer, and
re-run. Flags: `--skip-tests` (tests already ran this turn), `--json`
(structured output), `--force` (logged human-judgment override that
bypasses failed gates).

### Spec-to-PR linking

After completing delivery, if the work was committed on a branch with an
open PR (check with `gh pr view` or equivalent), annotate the PR:

1. Add a comment linking back to the spec: "Delivered from Hero spec: `<slug>` — <title>"
2. If the spec has a `tracker_id`, mention it in the PR description
3. If a tracker is configured, use `hero_claim` to update the tracker issue
   with the PR link

This closes the spec→code→PR chain so reviewers can trace requirements.

Spec or work item: $ARGUMENTS
