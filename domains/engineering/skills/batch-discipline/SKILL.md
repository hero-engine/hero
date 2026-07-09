---
name: batch-discipline
description: Generic protocol for running multiple specs or agents in a batch — concurrency limits, per-item isolation, the after-completion verification loop, and the summary-table format. Loaded by deliver.md's multi-spec mode and diagnose.md's parallel batch mode.
metadata:
  audience: delivery-leads, orchestrating-commands
  purpose: batch-protocol
---

## What I do

Define the one protocol every command uses when it fans out to multiple
specs or multiple parallel agents: how many to run, how each stays
isolated, what "done" means after they all return, and how to report the
result. Commands own their own selection logic (which specs, which
filters); this skill owns what happens once a batch is chosen.

## Concurrency discipline

- **Sequential by default.** Work through items one at a time unless the
  user explicitly asks for parallel execution ("run each in an agent",
  "diagnose all of them at once", "do these in parallel"). Sequential
  delivery means each item builds on a clean, already-tested state and
  avoids overlapping-file conflicts.
- **Parallel only on explicit ask**, and only for read/investigate/diagnose
  work that doesn't mutate shared code. Launch one subagent per item via
  the harness's delegation mechanism (e.g. Task agents).
- **Never parallelize when items may touch overlapping files or code** —
  that's a conflict risk regardless of what the user asked for. Say so and
  fall back to sequential.

## Per-item isolation

Each agent in a batch:

- Works only on its assigned spec/item — no cross-agent file access.
- Must write its findings/changes to disk (the spec file, the code) —
  never just report in chat. If an agent doesn't update the actual
  artifact, its item is incomplete regardless of what it says.
- Must not delete, move, or rename spec files. The only file operation
  allowed is editing the assigned item in place.
- If a tracker is configured and the item has a `tracker_id`, the agent
  posts its own summary/comment — don't batch tracker posts across items.

## After all agents complete

Once a batch (parallel or sequential) finishes, the orchestrator reviews
before reporting:

1. For each item, verify the artifact on disk was actually updated (spec
   file, code, tests) — don't trust the agent's self-report.
2. For each item with a `tracker_id`, verify the tracker post landed
   (check for an explicit "posted"/"attached" confirmation in the agent's
   output). Re-run any that missed it.
3. Re-run, individually, any item that failed to write its artifact or
   post to tracker — don't silently drop it from the summary.

## Summary table format

Report results in one table, not prose:

| Item | Spec/Location | Result | Status |
|---|---|---|---|
| `<id>` | `.hero/planning/.../spec.md` | [1-line summary] | done / needs-research / skipped |

## Failure handling

If an item fails (test failure, drift, tracker post fails), skip it, note
the reason next to it in the summary, and continue with the rest of the
batch — one failure should not block the others. At the end, summarize:
N completed, N skipped (with reasons), N needing follow-up.
