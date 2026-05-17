---
description: Open or reconcile the roadmap — navigate the board, drill into an item, or reconcile against live engineering delivery.
---
Route this roadmap request based on arguments.

## Modes

- **No arguments** → navigate to the Roadmap board singleton (the layered Horizon view). Default mode is presentation; no agent runs.
- **An initiative slug** → open that item's detail view (PRD link, linked intake, evidence, linked child specs/epics with their current `owner` and `status`).
- **`--reconcile` flag** → invoke `roadmap-curator` to walk the graph and update item statuses against live delivery.

## Reconcile mode

`roadmap-curator` does the following under the unified type model:

1. Walks the graph for each `initiative` with linked child specs and epics. For each child spec, reads `owner`, `status`, and the most-recent `owner_history` row.
2. Updates statuses based on what's actually true:
   - `shipped` — all child specs `completed`; the most recent `owner_history` rows show engineering close-out.
   - `in-flight` — at least one child spec is engineering-owned and `delivering`.
   - `stale-now` — the item is on the `Now` horizon (`kind: now`) but has no in-flight delivery for N days (default 14).
   - `committed-without-delivery` — committed but no child spec has flipped to engineering yet. Flag for PM attention.
3. Writes a changelog of every status change with rationale (which `owner_history` rows or child statuses drove the decision).

The curator never invents engineering state — it only reads the graph. If the graph says nothing, the curator says nothing.

## Output

- Navigate modes: open the appropriate view; no agent log.
- Reconcile mode: a changelog (rendered to chat as a short table) of what moved and why, plus updated frontmatter on each initiative spec. Log a `hero event` for the reconciliation pass.

A roadmap that lies is a failure mode (see [PM mission anti-patterns](../mission.md)). Reconcile weekly at minimum.

Request: $ARGUMENTS
