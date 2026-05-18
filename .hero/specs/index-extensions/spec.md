---
title: Index Extensions — Relations, Tags, Type Filtering, Graph Queries
slug: index-extensions
type: feature
status: completed
tags: [index, search, sqlite, graph, relations]
created: 2026-04-12
relations:
  - target: hero-v2-system-design
    kind: parent
horizon: now
---

## Goal

Extend the SQLite index to support spec relationships, claims, convention scope matching, and enriched search filters (type, status, tag, date). Add `hero graph` for relationship visualization.

## What Was Built

**New index tables:**
- `convention_scopes(spec_slug, scope_glob)` — maps conventions to file glob patterns for context injection
- `spec_relations(from_slug, to_slug, relation)` — parent/child/depends-on/supersedes/related links
- `claims(spec_slug, claimed_by, claimed_at)` — active spec claims

**Extended `specs` table** — added `type`, `status`, `created_at`, `tags`, `claimed_by`, `tracker_id`, `url` columns.

**`hero search` filters** — `--type`, `--status`, `--tag`, `--since` flags added. All combinable.

**`hero graph <slug>`** — shows spec relationships: parent initiative, related specs, superseded decisions, child specs.

**`hero context imports --files`** — uses `convention_scopes` for glob matching, queries `specs.files_touched` for past work in the same files.

**`hero conflicts`** — queries for in-flight specs with overlapping `files_touched`.

## Changes

- `internal/index/index.go` — new tables, extended specs table, all new queries
- `internal/cli/graph.go` — `hero graph` command
- `internal/cli/search.go` — type/status/tag/since filter flags
- `internal/cli/context.go` — convention scope matching queries
- `internal/cli/conflicts.go` — overlapping files_touched query
