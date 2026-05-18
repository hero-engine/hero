---
title: Recent Activity Digest — Smart Recap for Session Start
slug: activity-digest
type: feature
status: completed
tags: [recap, agents, cold-start, git, mcp, dx]
created: 2026-04-22
priority: P0
relations:
  - target: hero-killer-features
    kind: parent
horizon: now
---

## Goal

Give agents and humans a meaningful answer to "what happened since
yesterday?" by grouping recent git activity by spec rather than by commit.
Ship as `hero recap`, an MCP tool, and auto-included context in
`hero prime` output so agents get it at session start without explicitly
asking.

## Problem

Agents read `.hero/NEXT.md` at session start, which captures the previous
session's *intent* — what was finished, what's next, what decisions carry
forward. But NEXT.md is one session's perspective. When multiple agents or
humans land work between sessions, the returning agent has no way to know
what changed besides parsing raw `git log` — which is noise. Fifty commits
scroll past; most touch files the agent doesn't care about; none say which
spec they belong to or whether a spec changed status.

`hero pulse` covers sprint-level narrative (done / in-flight / at-risk)
but doesn't answer the tactical question: "which specs had actual code
land in the last 24 hours, and what specifically changed?"

The gap: a time-windowed, spec-grouped activity summary that complements
NEXT.md (agent-written intent) with git-derived fact.

## Design

### `hero recap` command

```
hero recap                      # activity in the last 24 hours
hero recap --since 2d           # activity in the last 2 days
hero recap --since 2026-04-20   # activity since a specific date
hero recap --format json        # machine-readable output
```

Default human output:

```
Activity since 2026-04-21 09:00 (24h)

spec csv-export (delivering):
  3 commits — added streaming write path, fixed buffer overflow,
              updated acceptance criteria
  files: internal/export/csv.go, internal/export/csv_test.go

spec auth-flow (delivering → completed):
  status changed: delivering → completed
  2 commits — final OAuth token refresh, cleanup

convention error-handling:
  updated — added retry-with-backoff guidance

knowledge:
  new: decisions/use-sqlc-over-raw-sql.md
  modified: context/deployment-topology.md

unmatched (2 commits):
  ci: bump Go to 1.23, docs: fix typo in README
```

### Spec-to-commit mapping

For each commit in the time window:

1. Get the list of changed files from `git diff-tree`.
2. Look up each file against the `files_touched` index (SQLite) to find
   which spec(s) own it.
3. Group commits by spec slug; commits touching no spec go into an
   "unmatched" bucket.

This reuses the existing `FilesTouched` mapping in `internal/spec/` and
the `files_touched` table in `internal/index/`. No new indexing required.

### Status transitions

Compare the current spec status against the status recorded at the
`--since` boundary. The index stores spec metadata; a simple diff of
`status` fields between the two points surfaces transitions like
`delivering -> completed`.

### Knowledge entries

Scan `git diff-tree` output for paths under `.hero/planning/decisions/`,
`.hero/planning/context/`, and `.hero/planning/conventions/` to detect
new or modified knowledge entries in the time window.

### MCP tool — `hero_recap`

```json
{
  "name": "hero_recap",
  "description": "Generate a spec-grouped activity summary for a time window",
  "inputSchema": {
    "type": "object",
    "properties": {
      "since": {
        "type": "string",
        "description": "Duration (e.g. '24h', '2d') or ISO date. Default: 24h"
      },
      "format": {
        "type": "string",
        "enum": ["text", "json"],
        "description": "Output format. Default: text"
      }
    }
  }
}
```

Returns the same structured payload as `--format json`.

### Integration with `hero prime`

Update `commands/prime.md` to run `hero recap --since 24h` after
surfacing `.hero/NEXT.md` and before other priming output. This gives the
agent both intent (NEXT.md) and fact (recap) automatically. If recap
returns no activity, the section is omitted silently.

### Data structures

```go
type Recap struct {
    Since      time.Time       `json:"since"`
    Until      time.Time       `json:"until"`
    Specs      []SpecActivity  `json:"specs"`
    Knowledge  []KnowledgeEntry `json:"knowledge,omitempty"`
    Unmatched  []CommitSummary `json:"unmatched,omitempty"`
}

type SpecActivity struct {
    Slug         string          `json:"slug"`
    Type         string          `json:"type"`
    OldStatus    string          `json:"old_status,omitempty"`
    NewStatus    string          `json:"new_status"`
    Commits      []CommitSummary `json:"commits"`
    FilesTouched []string        `json:"files_touched"`
}

type CommitSummary struct {
    Hash    string `json:"hash"`
    Subject string `json:"subject"`
    Author  string `json:"author"`
    Date    string `json:"date"`
}

type KnowledgeEntry struct {
    Path   string `json:"path"`
    Action string `json:"action"` // "new" or "modified"
}
```

## Changes

- `internal/recap/recap.go` — core logic: time-window commit enumeration, spec grouping via `files_touched` index, status transition detection, knowledge entry scanning
- `internal/recap/recap_test.go` — table-driven tests for grouping, unmatched bucketing, status transitions, knowledge detection, edge cases (no commits, all unmatched)
- `internal/cli/recap.go` — `hero recap` command with `--since` and `--format` flags
- `internal/cli/root.go` — register `recapCmd`
- `internal/serve/mcp.go` — register `hero_recap` tool
- `commands/prime.md` — include recap output after NEXT.md, before other priming context

## Acceptance Criteria

- WHEN `hero recap` runs with no flags THE SYSTEM SHALL display a spec-grouped activity summary for the last 24 hours, with commits grouped under their owning spec slug
- WHEN `hero recap --since <duration>` is provided with a relative duration such as "2d" or "48h" THE SYSTEM SHALL include all commits within that time window
- WHEN `hero recap --since <date>` is provided with an ISO date THE SYSTEM SHALL include all commits from that date to now
- WHEN a commit's changed files match entries in the `files_touched` index THE SYSTEM SHALL group that commit under the corresponding spec; WHEN a commit matches no spec THE SYSTEM SHALL place it in an "unmatched" bucket
- WHEN a spec's status at the start of the time window differs from its current status THE SYSTEM SHALL display the status transition (e.g. "delivering -> completed")
- WHEN `hero recap --format json` is provided THE SYSTEM SHALL output a JSON object matching the `Recap` struct schema
- WHEN the `hero_recap` MCP tool is called THE SYSTEM SHALL return the same structured payload as `--format json`

## Boundaries

- Does **not** replace `.hero/NEXT.md` — NEXT.md is agent-written intent, recap is git-derived fact; they complement each other
- Does **not** persist recap data — computed on demand from git history and the spec index
- Does **not** require external services — all data comes from the local git repo and SQLite index
- Does **not** call an LLM — commit-to-spec mapping is mechanical via `files_touched`
- Does **not** modify any spec or knowledge file — strictly read-only
