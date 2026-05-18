---
title: "Bulk Issue Import — Pull Issues from GitHub, Jira, and Linear"
slug: bulk-issue-import
type: feature
status: completed
tags: [cli, tracker, import]
created: 2026-04-12
horizon: now
---

## Goal

The `hero import` command pulls issues from external trackers (GitHub, Jira, Linear) and creates Hero spec stubs, enabling teams to adopt Hero without abandoning their existing issue tracking workflow.

## Design

The import system uses a pluggable tracker interface. Each tracker adapter knows how to:

- Authenticate (via environment variables or config)
- Query issues with filters (label, milestone, status, assignee)
- Map tracker fields to Hero spec frontmatter (title, status, tags, priority)

Imported issues become draft specs with a `source` frontmatter field linking back to the original issue. Duplicate detection uses the source URL to avoid re-importing the same issue.

### Supported Trackers

- **GitHub** — issues and pull requests via the GitHub API
- **Jira** — issues via the Jira REST API with JQL support
- **Linear** — issues via the Linear GraphQL API

### Usage

```
hero import github --repo owner/repo --label bug
hero import jira --project PROJ --status "To Do"
hero import linear --team engineering --state backlog
```

## Changes

- `internal/cli/import.go` — `hero import` command with subcommands per tracker
- `internal/cli/import_test.go` — tests for import command, flag parsing, deduplication
- `internal/tracker/tracker.go` — common tracker interface and shared logic
- `internal/tracker/github.go` — GitHub issue import adapter
- `internal/tracker/jira.go` — Jira issue import adapter
- `internal/tracker/linear.go` — Linear issue import adapter

## Acceptance Criteria

- `hero import github` pulls issues from a GitHub repository and creates spec stubs
- `hero import jira` pulls issues from Jira with JQL filters
- `hero import linear` pulls issues from Linear with team/state filters
- Imported specs include a `source` field linking to the original issue
- Re-running import skips already-imported issues (deduplication by source URL)
- Each tracker authenticates via environment variables
- Tests cover import logic, deduplication, and field mapping
