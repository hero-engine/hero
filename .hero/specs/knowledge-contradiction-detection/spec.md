---
title: Knowledge Contradiction Detection — Surface Conflicts in the Knowledge Base
slug: knowledge-contradiction-detection
type: feature
status: completed
milestone: v0.2
tags: [knowledge, conventions, decisions, quality, contradiction]
created: 2026-04-12
relations:
  - target: hero-serve-daemon
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Hero's knowledge base (conventions, decisions, knowledge notes) is append-only today. Teams add new conventions and decisions as they learn, but there's no mechanism to detect when a new entry contradicts an existing one. Over time, this creates silent conflicts: a new convention says "use `slog`" but an older convention says "use `zap`". An agent following both gets inconsistent guidance, or picks one arbitrarily.

This spec defines knowledge contradiction detection: a system that identifies when new knowledge conflicts with existing knowledge, surfaces those conflicts for human resolution, and marks stale entries to prevent them from misleading agents.

## Context

The gap is identified in the `memory-tools-and-community-patterns` note: Supermemory's key differentiator over standard RAG is a memory graph that resolves contradictions — when a new memory conflicts with an existing one, it merges/updates/marks stale rather than appending. Hero's knowledge base has the same append-only limitation.

The insight: detecting contradictions doesn't require a graph database. At Hero's scale (tens to hundreds of entries), an LLM can perform contradiction detection reliably in a single pass. The detection runs on `hero check` and on knowledge base writes, not continuously.

## Design

### When Detection Runs

1. **On `hero capture` / `hero note`** — when a new convention or decision is written, check it against existing entries of the same type
2. **On `hero check`** — the workspace health check includes a "knowledge consistency" check that scans for conflicts
3. **On demand** — `hero knowledge check` explicit command

### Detection Method

Contradiction detection uses an LLM pass over the relevant knowledge entries. For each new or changed entry, the system:

1. Loads all existing entries of the same type and scope (e.g., all conventions scoped to `*.go`)
2. Constructs a contradiction-detection prompt
3. The LLM identifies any conflicting entries
4. Conflicts are recorded as a `contradictions.json` index alongside the knowledge base

This is intentionally LLM-mediated — semantic similarity alone is insufficient. "Use `slog`" and "Use `zap`" are not string-similar, but they're semantically contradictory for the same scope. An LLM identifies this correctly.

### Conflict Types

| Type | Description | Example |
|---|---|---|
| `direct` | Two entries prescribe opposite behaviors | "Use tabs" vs "Use spaces" |
| `superseded` | New entry is a policy update that replaces an older one | "Use JWT (ADR-015)" after "Use sessions (ADR-007)" |
| `scope-overlap` | Two entries conflict for an overlapping scope | "All Go code: use zap" + "Backend Go: use slog" |
| `implicit` | Two entries are not literally contradictory but produce inconsistent behavior together | Depends on LLM judgment |

### `contradictions.json` Index

Conflicts are recorded in `.hero/knowledge/contradictions.json`:

```json
{
  "conflicts": [
    {
      "id": "conflict-001",
      "type": "direct",
      "status": "unresolved",
      "entries": [
        {
          "path": ".hero/conventions/logging/spec.md",
          "excerpt": "Use zap for all structured logging"
        },
        {
          "path": ".hero/conventions/backend-logging/spec.md",
          "excerpt": "Use slog for all Go logging (stdlib preferred)"
        }
      ],
      "detected_at": "2026-04-14",
      "summary": "Two conventions prescribe different logging libraries for Go code",
      "suggestion": "Decide on one logging library and deprecate the other convention"
    }
  ]
}
```

### `hero check` Integration

The workspace health check gains a new check: `knowledge-consistency`. On `hero check`:

```
Knowledge consistency... WARN
  2 unresolved conflicts detected:
  - conventions/logging vs conventions/backend-logging (direct conflict: logging library)
  - decisions/adr-007 vs decisions/adr-015 (superseded: auth approach)
  Run `hero knowledge check` for details and resolution options.
```

Unresolved conflicts show as warnings, not errors — they don't block delivery but they surface the problem.

### `hero knowledge check` Command

```bash
# Show all detected conflicts
hero knowledge check

# Re-run detection (slow — makes LLM calls)
hero knowledge check --scan

# Show conflicts for a specific type
hero knowledge check --type conventions
hero knowledge check --type decisions

# Resolve a conflict
hero knowledge check --resolve conflict-001
```

**`--scan`** re-runs LLM-mediated detection over all entries. This is the slow path (seconds to minutes depending on knowledge base size). The default (no `--scan`) reads from the cached `contradictions.json`.

**`--resolve`** opens an interactive resolution flow:
1. Shows both conflicting entries
2. Offers options: keep A, keep B, merge, mark one deprecated, or mark conflict as "known/intentional"
3. On resolution: updates frontmatter of the deprecated entry with `status: deprecated` and `superseded_by: <slug>`; updates `contradictions.json`

### Stale/Deprecated Entries

When a conflict is resolved by marking one entry deprecated:

```markdown
---
title: Logging Convention — Use Zap
type: convention
status: deprecated
superseded_by: backend-logging
deprecated_at: 2026-04-14
deprecated_reason: Replaced by backend-logging convention (slog preferred)
---
```

Hero agents are instructed (via AGENTS.md / hero context) to ignore entries with `status: deprecated`. The context injection system (`hero context`) already filters by status — deprecated conventions don't reach the model.

### False Positive Management

Not all detected conflicts are real. Some entries cover different scopes that look contradictory but aren't. `hero knowledge check --resolve` includes a "mark as intentional" option:

```json
{
  "id": "conflict-001",
  "status": "intentional",
  "resolution_note": "Different logging libraries for different service tiers — intentional"
}
```

Intentional conflicts are not shown in `hero check` output and not re-scanned.

### Detection Scope

Contradiction detection runs within entry types — conventions vs. conventions, decisions vs. decisions, notes vs. notes. Cross-type detection (e.g., a note contradicting a convention) is out of scope for v0.2 — the signal-to-noise ratio is too low.

Within each type, detection is scoped by the `scope` frontmatter field: a convention scoped to `*.ts` is not checked against one scoped to `*.go`. Overlapping scopes (both `*` and `*.go`) are checked.

### On-Write Detection (Fast Path)

When `hero capture` or `hero note` writes a new entry, a lightweight on-write check runs:

1. Load existing entries of the same type and scope
2. Check if entry count is below threshold (default: 50 entries) — if above, skip on-write and defer to `hero check`
3. Run LLM contradiction check for the new entry only (not a full scan)
4. If a conflict is detected, warn immediately: `Warning: new convention may conflict with conventions/logging — run 'hero knowledge check' to review`

This provides fast feedback without expensive full scans on every write.

## Changes

- `internal/knowledge/contradiction.go` — contradiction detection logic (LLM pass, type definitions, index management)
- `internal/knowledge/contradictions_index.go` — `contradictions.json` reader/writer
- `internal/cli/knowledge.go` — `hero knowledge check` command with scan and resolve subcommands
- `internal/check/checks.go` — `knowledge-consistency` check added to `hero check`
- `internal/capture/capture.go` — on-write detection trigger for `hero capture` and `hero note`

## Acceptance Criteria

- `hero check` reports unresolved knowledge conflicts as warnings
- `hero knowledge check` lists all conflicts with summaries and affected entries
- `hero knowledge check --scan` runs LLM detection and updates `contradictions.json`
- `hero knowledge check --resolve <id>` walks through interactive resolution
- Resolved conflicts update frontmatter with `status: deprecated` and `superseded_by`
- Deprecated entries are excluded from `hero context` output
- Intentional conflicts are suppressed from `hero check` output
- On-write detection warns immediately when a new entry may conflict with existing ones
- False positives can be marked as "intentional" without deleting either entry

## Boundaries

- Detection is **LLM-mediated** — not string similarity; requires an AI model call for `--scan`
- Does **not** auto-resolve conflicts — human decision required for all resolution
- Does **not** detect cross-type conflicts (convention vs. decision) in v0.2
- Does **not** continuously scan — detection is on-write (lightweight) and on-demand (full scan)
- `contradictions.json` is a cache — it can be deleted and rebuilt with `--scan`
