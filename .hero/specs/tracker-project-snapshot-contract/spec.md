---
title: "Tracker project snapshot contract"
slug: tracker-project-snapshot-contract
type: feature
status: completed
priority: high
size: medium
tags: [tracker, integration, sprint, roadmap]
completed_at: 2026-07-22T20:30:24Z
---

# Tracker project snapshot contract

## Kickoff

Deliver a versioned, provider-neutral, read-only project snapshot that lets Hero consumers render tracker-native sprint and schedule truth without embedding Jira APIs or credentials. Reuse the existing sprint loaders, keep remote work bounded and explicit, and emit completeness/error metadata rather than inventing data.

## Goal

Expose the configured tracker project, board, active/future iterations, and their item memberships through one JSON contract suitable for background UI hydration.

## Design

- Add `tracker-project-snapshot/v1` with project, board, iteration, item, generated-at, completeness, truncation, stale, and error fields.
- Add `hero sync project-snapshot [--board <id-or-name>]`.
- Jira enumerates active and future sprints from the selected project board, then uses a schedule-only issue loader that requests summary/type/status/assignee and paginates membership without hydrating descriptions or custom fields.
- The command joins tracker IDs to existing local Hero slugs without mutating specs.
- No descriptions, comments, changelogs, or attachments are included.
- Unsupported providers fail explicitly; consumers retain their last good snapshot.

## Acceptance Criteria

- The JSON is versioned and deterministic.
- Jira active/future sprint identity, dates, native status, normalized category, assignee, rank, and membership are represented.
- More than 100 issues are paginated rather than silently truncated.
- Existing local specs are linked by tracker ID when available.
- Tests cover contract JSON, board/sprint loading, pagination, and CLI output.

## Completion Ledger

Delivered 2026-07-22 as `tracker-project-snapshot/v1` and `hero sync project-snapshot`.

### Acceptance Criteria

| Criterion | Status | Evidence |
|---|---|---|
| Versioned deterministic JSON | DONE | `contracts/trackerproject/contract.go`; CLI output test decodes v1. |
| Jira board, active/future sprint, and membership truth | DONE | `jiraSprintLoader.LoadProjectSnapshot`, `resolveBoard`, and `listBoardSprints`. |
| Pagination beyond 100 issues | DONE | Dedicated membership loader follows `nextPageToken`; fixture covers two pages. |
| Local specs joined by tracker ID | DONE | `sync_project_snapshot.go` builds the tracker ID/slug index before JSON output. |
| Scheduling snapshot excludes full detail | DONE | Field request is exactly `summary,issuetype,status,assignee`; no description/custom fields/comments/attachments. |
| Existing contracts remain compatible | DONE | Full `go test ./...` passes. |

### Exercise-the-feature check

- [x] Contract JSON, board discovery, sprint discovery, issue pagination, normalization, and CLI output were exercised by focused tests.
- [x] Full `go test ./...` passed.
