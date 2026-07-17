---
description: Compose a standup update from intra-cycle graph changes — what moved since the last standup, read from the cross-domain graph, cut for the internal team.
---
Route to `stakeholder-communicator`, loading `stakeholder-communication` and `cross-domain-graph-query`.

## Scope

- **No arguments** → compose a standup for the current cycle since the last standup pass.
- **`--since <duration>`** (e.g. `--since 1d`, `--since 1w`) → the window to read graph changes over.
- **A cycle or initiative slug** → scope the update to that cycle's / initiative's movement.

## What lands

A standup update composed from **intra-cycle graph changes** — not a hand-maintained list. Read what moved from the cross-domain graph (`hero feed` / graph events) via `cross-domain-graph-query`:

- specs that advanced status
- handoffs (owner flips `pm → engineering`)
- hill-chart movement (uphill → downhill)
- blockers hit

Cut the update for the **internal team** audience — plain and specific, not a launch announcement. Where the update should persist, write it to `.hero/knowledge/notes/`.

## Output

- The standup update (the moved-since-last items, grouped for the team cut).
- A one-line log naming the window read and the count of items that moved.

Request: $ARGUMENTS
