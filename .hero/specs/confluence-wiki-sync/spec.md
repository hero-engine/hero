---
title: Confluence Wiki Sync — Publish Hero Knowledge to Confluence
type: feature
status: completed
milestone: v0.2
tags: [confluence, wiki, sync, publish, documentation]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: related
horizon: now
---

## Goal

Teams that live in Atlassian's ecosystem (Jira + Confluence) should be able to publish Hero's knowledge base to Confluence so that the engineering spec layer is visible across the org — not just to people who know to look in `.hero/`. This spec defines `hero sync confluence`: a command that pushes Hero conventions, decisions, specs, and knowledge notes to a configured Confluence space.

This is the Atlassian-native equivalent of the GitHub Wiki sync target already referenced in the v2 system design.

## Context

The v2 system design already defines a `sync.target: github-wiki` config stub. Confluence is the second logical sync target, particularly for teams already using Jira integration. Both share the same conceptual model: Hero is the source of truth for engineering specs, and the wiki is a read-friendly published view for the broader org.

## Design

### Command Surface

```bash
# Publish everything to Confluence
hero sync confluence

# Publish specific sections
hero sync confluence --section conventions
hero sync confluence --section decisions
hero sync confluence --section specs
hero sync confluence --section knowledge

# Preview without writing
hero sync confluence --dry-run

# Watch mode: auto-publish on Hero file changes
hero sync confluence --watch
```

### What Gets Published

| Hero Source | Confluence Destination |
|---|---|
| `.hero/conventions/` | Space → `Conventions` page tree |
| `.hero/decisions/` | Space → `Decisions` page tree |
| `.hero/specs/<slug>/spec.md` (completed) | Space → `Specs` → `<title>` |
| `.hero/planning/features/<slug>/spec.md` | Space → `Planning` → `<title>` |
| `.hero/knowledge/notes/<slug>/spec.md` | Space → `Knowledge` → `<title>` |
| `hero dashboard` output | Space → `Engineering Dashboard` (auto-updated) |

Only `completed` specs are published by default. Planning specs are opt-in (`--include-planning`).

### Page Structure in Confluence

```
[Hero Space Root]
├── Overview          ← generated summary: spec counts, recent completions
├── Conventions
│   ├── Code Style
│   ├── Testing
│   └── API Design
├── Decisions
│   ├── ADR-001: Use PostgreSQL
│   └── ADR-002: JWT over sessions
├── Specs
│   ├── hero-serve-daemon
│   ├── bulk-issue-import
│   └── sprint-planner
├── Planning
│   └── (planning specs, if --include-planning)
└── Knowledge
    ├── buddy-model-architecture
    └── memory-tools-and-community-patterns
```

### Markdown → Confluence Storage Format

Confluence uses its own XML-based storage format ("XHTML Storage Format"). The sync command converts Hero's markdown to Confluence storage format:

- Standard markdown → equivalent Confluence markup
- Code blocks → Confluence `<ac:structured-macro name="code">` with language highlighting
- Frontmatter → Confluence page labels (tags, type, status)
- Internal Hero links (spec slugs) → Confluence page links within the space
- Tables → Confluence table markup

The conversion uses a lightweight internal renderer — no external dependencies.

### Sync Strategy

**Page identity:** Pages are identified by a `heroSlug` label applied on creation. On subsequent syncs, the command queries for pages with that label and updates them in place (no duplicate pages).

**Deletion:** By default, pages are never deleted — they're marked with a `[Archived]` prefix and a notice if the source spec is removed. Explicit deletion requires `--delete-removed`.

**Conflict detection:** If a Confluence page has been manually edited since last sync (Confluence `version.number` changed), the sync warns and skips that page unless `--force` is passed.

### Authentication

```bash
# Environment variables
CONFLUENCE_URL=https://yourorg.atlassian.net
CONFLUENCE_EMAIL=user@example.com
CONFLUENCE_TOKEN=<api-token>  # same token as Jira if using Atlassian Cloud

# Or in hero.json (references env vars)
{
  "confluence": {
    "url": "${CONFLUENCE_URL}",
    "email": "${CONFLUENCE_EMAIL}",
    "token": "${CONFLUENCE_TOKEN}",
    "space_key": "ENG"
  }
}
```

Atlassian Cloud uses the same API token for both Jira and Confluence. On-premise Confluence Server/DC uses a personal access token (PAT) with `Authorization: Bearer` instead.

### Configuration

```json
{
  "sync": {
    "target": "confluence"
  },
  "confluence": {
    "url": "https://yourorg.atlassian.net",
    "email": "${CONFLUENCE_EMAIL}",
    "token": "${CONFLUENCE_TOKEN}",
    "space_key": "ENG",
    "root_page_title": "Hero Engineering Specs",
    "include_planning": false,
    "include_status": ["completed", "delivering"],
    "delete_removed": false,
    "auto_sync": false
  }
}
```

`auto_sync: true` causes `hero sync confluence` to run automatically after `hero complete <slug>` — keeping the wiki always current without manual invocation.

### Dashboard Page

In addition to individual specs, the sync publishes a live `Engineering Dashboard` page summarizing:
- Total spec counts by status
- Recently completed specs (last 30 days)
- Active deliveries and their owners
- Open planning specs by priority

This page is always regenerated on each sync, never manually edited.

## Changes

- `internal/sync/confluence.go` — Confluence REST API client (pages, labels, spaces)
- `internal/sync/confluence_render.go` — markdown to Confluence storage format converter
- `internal/cli/sync.go` — `hero sync confluence` subcommand (extends existing sync command)
- `internal/config/config.go` — `confluence` config section

## Acceptance Criteria

- `hero sync confluence` publishes all completed specs to the configured Confluence space
- Pages are organized into the defined page tree structure
- Re-running sync updates existing pages (no duplicates)
- Manually edited Confluence pages are warned about and skipped (unless `--force`)
- `--dry-run` shows what would be created/updated without making API calls
- Frontmatter tags become Confluence labels
- Code blocks render with syntax highlighting in Confluence
- Auth uses env vars consistent with Jira integration
- `auto_sync: true` triggers sync automatically on `hero complete`

## Boundaries

- Does **not** import content from Confluence back into Hero — sync is one-directional (Hero → Confluence)
- Does **not** sync source code, test files, or non-Hero content
- Does **not** manage Confluence space creation — the space must already exist
- Does **not** sync real-time (watch mode is a polling fallback, not a webhook listener)
