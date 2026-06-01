---
title: Jira Deep Integration — Bidirectional Sync Beyond Import
slug: jira-deep-integration
type: feature
status: completed
milestone: v0.2
tags: [jira, tracker, sync, bidirectional, sprint]
created: 2026-04-12
relations:
  - target: bulk-issue-import
    kind: extends
  - target: sprint-from-tracker
    kind: dependency-of
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

The existing `hero import jira` command is a one-shot import: it pulls issues and creates spec stubs. That's a start, but teams with deep Jira workflows need more. This spec defines what "deep" Jira integration means for Hero: bidirectional sync, sprint loading, epic hierarchy, status propagation, and rich field mapping.

This is not a replacement for Jira — it's a bridge that lets Hero and Jira coexist as complementary layers. Jira owns project management; Hero owns the engineering spec layer.

## Current State

`hero import jira` (from `bulk-issue-import`) already:
- Authenticates via env vars (`JIRA_URL`, `JIRA_TOKEN`, `JIRA_EMAIL`)
- Queries issues with JQL filters
- Creates spec stubs with `source` and `tracker_id` frontmatter
- Deduplicates by `source` URL

What it does NOT do:
- Load sprint-level batches (handled by `sprint-from-tracker` for the sprint command surface; this spec handles the Jira adapter depth)
- Sync status changes back to Jira when specs move to `completed`
- Import epic hierarchy and map to Hero initiatives
- Import linked issues as Hero spec relations
- Import Jira components as Hero tags/convention scopes
- Sync comments or accept new comments from Hero

## Design

### Enhanced Field Mapping

Expand the Jira adapter's field mapping to cover more Jira issue fields:

| Jira Field | Hero Frontmatter |
|---|---|
| `summary` | `title` |
| `description` | Body `## Goal` section |
| `issuetype.name` | `type` (story→feature, bug→bug, task→feature, epic→initiative) |
| `status.name` | `status` (mapped via status map below) |
| `priority.name` | `priority` |
| `assignee.displayName` | `claimed_by` |
| `reporter.displayName` | `author` |
| `labels` | `tags` (merged with sprint tag) |
| `components` | `tags` (component names as tags) |
| `fixVersions` | `tags` (version names as tags) |
| `sprint.name` | `sprint` |
| `parent.key` | `relations` (kind: parent) |
| `issuelinks` | `relations` (kind depends on link type) |
| `acceptanceCriteria` (custom) | Body `## Acceptance Criteria` section |
| `story_points` | `effort` |
| `duedate` | `due` |

**Status mapping** (configurable in `hero.json`):

```json
{
  "jira": {
    "status_map": {
      "To Do": "planning",
      "In Progress": "delivering",
      "In Review": "delivering",
      "Done": "completed",
      "Blocked": "blocked"
    }
  }
}
```

### Epic and Initiative Hierarchy

Jira epics map to Hero initiatives. When importing an epic:
1. Create `planning/initiatives/<slug>/spec.md` with type `initiative`
2. Query all issues belonging to that epic
3. Create spec stubs for each child issue
4. Add `relations` frontmatter to the initiative listing each child with `kind: child`
5. Add `relations` frontmatter to each child pointing to the initiative with `kind: parent`

For deeply nested hierarchies (Jira Advanced Roadmaps `parent_link` field), walk up the parent chain and create the full initiative tree.

### Linked Issues as Hero Relations

Jira issue links (`blocks`, `is blocked by`, `relates to`, `duplicates`, `clones`) map to Hero spec relations:

| Jira Link Type | Hero Relation Kind |
|---|---|
| `blocks` | `blocks` |
| `is blocked by` | `blocked-by` |
| `relates to` | `related` |
| `duplicates` | `duplicate-of` |
| `clones` | `derived-from` |

### Status Writeback (Push to Jira)

When a Hero spec transitions to `completed`, optionally push that status change back to Jira:

```bash
hero sync jira --push-status
```

Or configure auto-push in `hero.json`:
```json
{
  "jira": {
    "push_status_on_complete": true
  }
}
```

When auto-push is enabled, `hero complete <slug>` transitions the corresponding Jira issue to the configured "done" status via the Jira Transitions API.

This requires a `JIRA_TOKEN` with write permissions. Read-only tokens skip writeback silently (log a warning).

### Sprint Loading (Jira Adapter Side)

The `sprint-from-tracker` spec defines the `hero sprint load` command surface. This spec defines the Jira adapter's implementation for sprint queries:

```
GET /rest/agile/1.0/board/{boardId}/sprint?state=active
GET /rest/agile/1.0/sprint/{sprintId}/issue
```

JQL fallback for installations without Agile API access:
```
JQL: sprint = "Sprint 42" ORDER BY rank ASC
```

The adapter resolves board names to board IDs automatically:
```bash
hero sprint load --tracker jira --board "Engineering"
# resolves "Engineering" → boardId=42 → active sprint → issues
```

### Comments as Knowledge Notes (Optional)

When importing an issue with Jira comments, optionally append a `## Discussion` section to the spec stub with key comments (configurable: all comments, or only comments containing specified keywords like "decision", "blocker", "approach"):

```json
{
  "jira": {
    "import_comments": "decisions",
    "comment_keywords": ["decision", "blocker", "approach", "why"]
  }
}
```

### Configuration

All Jira config in `hero.json` under `jira`:

```json
{
  "jira": {
    "url": "https://yourorg.atlassian.net",
    "email": "${JIRA_EMAIL}",
    "token": "${JIRA_TOKEN}",
    "default_project": "ENG",
    "status_map": { "To Do": "planning", "Done": "completed" },
    "push_status_on_complete": false,
    "import_comments": "none",
    "custom_fields": {
      "acceptance_criteria": "customfield_10016",
      "story_points": "customfield_10028"
    }
  }
}
```

Custom field IDs vary per Jira installation. Hero auto-discovers them via the fields API on first run and caches in `hero.json`.

## Changes

- `internal/tracker/jira.go` — expanded field mapping, epic hierarchy walking, sprint queries, status writeback
- `internal/tracker/jira_fields.go` — custom field discovery and caching
- `internal/cli/sync.go` — `hero sync jira` command with `--push-status` flag
- `internal/config/config.go` — `jira` config section with status map and options
- `commands/sprint.md` — document the Jira sprint loading flow

## Acceptance Criteria

- `hero import jira` maps all documented Jira fields to Hero frontmatter
- Epic issues create initiative specs with child relations to their stories
- Jira issue links (blocks, relates-to, etc.) create Hero spec relations
- `hero sync jira --push-status` transitions Jira issues to Done when Hero specs complete
- Status map is configurable in `hero.json`
- Sprint queries work via both Agile API and JQL fallback
- Custom field IDs are auto-discovered and cached on first run
- All operations require only the existing env var auth pattern

## Boundaries

- Does **not** create Jira issues from Hero specs (Hero reads Jira; it does not own issue creation)
- Does **not** sync spec body text back to Jira descriptions (one-way: Jira description → Hero spec on import)
- Does **not** manage Jira boards, sprints, or workflows — Hero never modifies Jira project configuration
- Status writeback is opt-in and disabled by default
