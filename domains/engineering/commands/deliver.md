---
description: Execute a spec — implement the planned changes, validate, and complete the work item.
---
Route this delivery request to the `feature-delivery-lead` agent for execution.

Be the `feature-delivery-lead` agent. Load the `context-injection` skill before starting.

**Before starting work**, emit a `hero next ask` capturing what the user
asked for. This preserves session intent across compaction — see the
`next-handoff-emit` skill for the full pattern (ask / suggest / reflection).

## Delivery modes

Parse the mode from the user's invocation. If no mode is specified, default
to **supervised**.

| Flag | Behavior |
|---|---|
| `--supervised` (default) | Pause at handoffs, surface decisions, ask before destructive actions. Current behavior. |
| `--autopilot` | Run to completion without intermediate confirmations. Stop only on test failure, drift warning, boundary violation, or any non-`DONE` Completion Ledger item (PARTIAL, SKIPPED, BLOCKED). |
| `--dry-run` | Run analysis and planning. Produce a delivery plan (file list, agent assignments, estimated changes) but write NO code. |

### Autopilot mode

Suppress confirmation prompts during the delivery loop. Check `hero drift`
after implementation, run tests if a runner is configured, validate the
engineer's **Completion Ledger** against the spec, and halt on any of:
test failure, drift warning, boundary violation, or any non-`DONE` ledger
item (PARTIAL, SKIPPED, BLOCKED). If `--halt-on` is provided
(comma-separated: `drift`, `test`, `boundary`, `lint`, `ledger`), only halt
on those specific conditions.

Persistence rule (`agent-reliability` — "Persistence on continuous tasks")
applies: do not yield between phases unless a true blocker fires. A
non-`DONE` ledger row IS a true blocker — halt and surface it. "This is
taking a while" is NOT.

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

## Batch mode

If the user asks to fix/deliver multiple specs (e.g. "fix the researched bugs", "deliver the 5 planned features"):

1. Run `hero search --list --type bug --status planning` (or the appropriate type/status) to find candidate specs
2. Filter to specs that have a `## Suggested Fix Approach` or `## Changes` section — these are the ones with a plan ready to execute
3. Present the list to the user for confirmation: "Found N specs ready to deliver: [list]. Proceeding with all of them sequentially."
4. **Work through them sequentially, one at a time.** Each fix:
   - Read the spec and its fix plan
   - Implement the fix
   - Run tests / build to verify
   - Require a **Completion Ledger** from the engineer covering every acceptance criterion and Changes item (see `engineer.md` — "Closing output"). Validate it against the spec.
   - Commit with a message referencing the spec slug and tracker ID
   - Only set `status: completed` if the ledger is fully `DONE`. If any row is PARTIAL / SKIPPED / BLOCKED, halt and surface — do not flip status without sign-off. `hero spec verify` or the async runner will auto-archive completed specs to specs/.
   - Post results to tracker if configured
5. **One commit per fix** — each fix is atomic and independently revertable
6. If a fix fails tests or creates problems, skip it, note the issue in the spec, and move to the next one
7. At the end, summarize: N fixed, N skipped (with reasons)

**Do not parallelize fixes** — they may touch overlapping code and create conflicts. Sequential delivery ensures each fix builds on a clean, tested codebase.

## Queue mode

When the user says "deliver these while I'm away", "queue these up", or
provides a list of slugs with `--autopilot`:

1. Validate all specs are ready (have Changes section, pass quality gate)
2. Sort by dependency order using `hero_sequence`
3. Show the queue and get one confirmation: "Will deliver N specs in order:
   [list]. Mode: autopilot. I'll halt on test failures, boundary violations,
   or non-`DONE` Completion Ledger items."
4. Execute sequentially using the autopilot flow for each
5. Between each spec: run `hero drift`, run tests, validate the Completion
   Ledger, commit atomically
6. If one fails or its ledger has non-`DONE` rows: log the failure / open
   items, skip the spec (do NOT flip to `completed`), continue with the next
7. At the end: summary of delivered (fully `DONE`), partially delivered
   (with non-`DONE` ledger rows for follow-up), and skipped (with reasons)

## Single spec mode

For a single spec delivery, follow the standard pre-flight and delivery workflow as the `feature-delivery-lead` agent.

**Pre-flight — check for mockups.** Before delegating to the engineer, check whether `.hero/mocks/{slug}/` exists for the spec being delivered. If it does, list the file paths inside it in the kickoff context handed to the engineer (e.g. "Mockups available for this spec: `.hero/mocks/{slug}/index.html` — read if a visual reference helps."). Don't render or quote the HTML; let the engineer open it if useful. The same check applies in batch and queue modes — surface mockup paths in the kickoff for each spec.

At the end of the delivery loop:

1. **Check for drift** — verify the implementation matches the spec. Address
   any divergence before moving on.
2. **Verify and expand test coverage** — confirm every acceptance criterion
   has a corresponding test. When fixing a bug, add a regression test that
   reproduces the original failure. Look at the code you touched and the
   code around it — if adjacent functions or modules lack test coverage,
   add tests for them too. Match the project's existing test patterns and
   framework. The goal is to leave coverage better than you found it.
3. **Run the test suite** — make sure everything passes, including the
   tests you just wrote and any existing tests for the affected area.
4. **Link tests to criteria** — connect each new test back to the
   acceptance criterion it verifies so regressions are traceable.
5. **Validate the Completion Ledger** — the engineer's closing artifact
   (see `engineer.md` — "Closing output") enumerates every acceptance
   criterion and every `## Changes` item with a `DONE` / `PARTIAL` /
   `SKIPPED` / `BLOCKED` status. Before flipping spec status:
   - Confirm every acceptance criterion and every Changes item has a row.
   - Cross-check each `DONE` against actual code and test evidence on disk.
     Challenge performative `DONE` marks; downgrade rows that lack evidence.
   - For user-visible behavior, confirm the Exercise-the-feature check is
     populated — unit tests alone are not sufficient evidence.
   - **If any row is `PARTIAL` / `SKIPPED` / `BLOCKED`, do NOT flip to
     `completed`** without explicit user sign-off. In autopilot mode this
     halts the run; in supervised mode, surface the open rows and ask.
6. **Refresh the kickoff** — when status flips (planning → delivering,
   delivering → completed) or after meaningful chunks land, rewrite the
   spec's `## Kickoff` section so "Pick up at:" reflects *now*, not
   "two commits ago." Follow the `kickoff-prompt` skill. After mutating
   the spec, run `hero index --if-stale -q` so the new state surfaces
   in `hero search` / `hero list` / `hero_*` MCP tools immediately.
   The pre-commit hook will refresh `.hero/QUEUE.md` automatically; if
   you're not committing right away, run `hero queue write -q` to
   refresh the snapshot manually.

When delivery is complete and the Completion Ledger is fully `DONE` (or
non-`DONE` rows have explicit user sign-off), set the spec's
`status: completed` in the frontmatter and run `hero spec verify <slug>` —
verify auto-archives a completed spec to `.hero/specs/<slug>/`, so you
don't need a separate `hero spec complete` step. The async runner does the
same auto-archive at the tail of every successful agent delivery.

The Completion Ledger replaces the older "implementation summary" pattern.
Soft prose summaries that gloss skipped or partial work are explicitly
out — the ledger is the artifact of record.

### Spec-to-PR linking

After completing delivery, if the work was committed on a branch with an
open PR (check with `gh pr view` or equivalent), annotate the PR:

1. Add a comment linking back to the spec: "Delivered from Hero spec: `<slug>` — <title>"
2. If the spec has a `tracker_id`, mention it in the PR description
3. If a tracker is configured, use `hero_claim` to update the tracker issue
   with the PR link

This closes the spec→code→PR chain so reviewers can trace requirements.

Spec or work item: $ARGUMENTS
