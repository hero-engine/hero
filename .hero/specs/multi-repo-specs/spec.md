---
title: Multi-Repo Spec Awareness — Cross-Repository References and Drift
slug: multi-repo-specs
type: feature
status: completed
tags: [specs, drift, multi-repo, config, context]
created: 2026-04-22
priority: P1
relations:
  - target: hero-killer-features
    kind: parent
  - target: spec-drift-detection
    kind: depends-on
  - target: cross-spec-awareness
    kind: related
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

## Goal

Allow specs to declare dependencies on specs in other repositories and have
Hero resolve, validate, and report on those cross-repo references. When repo
A's spec depends on repo B's spec, `hero drift` flags changes in repo B that
could affect repo A. `hero context` pulls in cross-repo conventions and
decisions when the file being edited has cross-repo dependencies.

## Problem

Hero manages specs within a single repository. Real projects span many repos --
the user has 20+ adjacent repos today and manually adds agent notes pointing at
other repo paths to communicate cross-repo dependencies. There is no structured
way to express "this spec depends on the session-tokens spec in auth-service,"
no tooling to detect when an upstream spec changes, and no way for
`hero context` to surface conventions or decisions from a dependency repo.

Drift that originates across repo boundaries is invisible: repo B ships a
breaking change to its session-tokens spec, and repo A's consuming spec has no
idea until a human notices at integration time.

## Design

### Cross-repo relation syntax

Relations gain an optional repo prefix separated by `/`:

```yaml
relations:
  - target: auth-service/session-tokens
    kind: depends-on
  - target: api-gateway/rate-limiting
    kind: related
```

A target without a `/` is a local spec (current behavior, unchanged). A target
with a `/` is `<repo-alias>/<spec-slug>`, resolved via the `repos` map in
`hero.json`.

### `hero.json` repos configuration

```json
{
  "repos": {
    "auth-service": "../auth-service",
    "api-gateway": "../api-gateway",
    "shared-libs": "/Users/dev/projects/shared-libs"
  }
}
```

Paths are relative to the project root (where `hero.json` lives) or absolute.
Each value must point to a directory containing its own `.hero/` folder. Hero
validates repo paths at parse time and reports clear errors for missing or
misconfigured entries.

### Cross-repo spec resolution

`internal/spec/spec.go` gains a resolver that, given a cross-repo relation
target like `auth-service/session-tokens`:

1. Looks up `auth-service` in the `repos` map.
2. Resolves the path to an absolute directory.
3. Walks `<resolved-path>/.hero/planning/` to find a spec with slug
   `session-tokens`.
4. Parses and returns the remote spec (read-only -- no writes to other repos).

Resolution is lazy and cached per CLI invocation. If a repo alias is not
configured or the path is invalid, Hero emits a warning and continues (graceful
degradation, not a hard error).

### Cross-repo drift detection

`hero drift` gains awareness of cross-repo `depends-on` relations:

```
hero drift <slug>              # includes cross-repo dependency checks
hero drift --cross-repo        # only show cross-repo drift signals
```

New drift signals specific to cross-repo:

| Signal | Detection | Example |
|---|---|---|
| **Upstream spec changed** | Remote spec's `ModifiedAt` is newer than local spec's `ModifiedAt` | auth-service/session-tokens was updated after this spec was last touched |
| **Upstream status changed** | Remote spec moved to `superseded` or `completed` | Dependency was superseded, local spec may need updating |
| **Upstream boundary conflict** | Remote spec's boundaries contradict local spec's assumptions | auth-service says "does not support refresh tokens", local spec assumes refresh tokens |
| **Missing upstream** | Remote spec slug not found in the configured repo | Dependency was deleted or renamed |

Output integrates with the existing drift report:

```
csv-export (status: delivering)
  ...existing local drift signals...
  Cross-repo dependencies:
  ⚠ auth-service/session-tokens was modified 3 days after this spec
      → review upstream changes for breaking impact
  ⚠ api-gateway/rate-limiting status changed to "superseded"
      → upstream spec was superseded, check replacement
  ✓ shared-libs/error-handling — no changes detected
```

### Cross-repo context enrichment

`hero context` gains the ability to pull in relevant specs, conventions, and
decisions from dependency repos when the file being edited has cross-repo
dependencies:

1. For each cross-repo `depends-on` relation on the active spec, resolve the
   remote spec.
2. Include the remote spec's Goal and Acceptance Criteria sections in the
   context payload (trimmed to avoid context budget bloat).
3. If the remote repo has conventions or decisions relevant to the interface
   surface (detected by tag or section overlap), include those too.

This is additive -- cross-repo context appends to, never replaces, local
context.

### Validation

`hero check` gains a cross-repo validation step:

- All repo aliases referenced in spec relations exist in the `repos` map.
- All configured repo paths are accessible and contain `.hero/` folders.
- All cross-repo relation targets resolve to existing specs.

Warnings, not errors -- a temporarily unavailable repo should not block local
work.

## Changes

- `internal/config/config.go` — add `Repos map[string]string` field to `Config`; validation for repo paths
- `internal/spec/spec.go` — parse cross-repo relation targets (split on `/`), add `Repo` field to `Relation` struct
- `internal/spec/resolve.go` — new file: cross-repo spec resolver with lazy loading and per-invocation cache
- `internal/drift/drift.go` — cross-repo drift signals: upstream modified, status changed, missing upstream
- `internal/cli/drift.go` — `--cross-repo` flag, cross-repo resolution plumbing
- `internal/context/format.go` — cross-repo context enrichment: pull Goal/Criteria from dependency specs
- `internal/cli/check.go` — cross-repo validation step: repo paths, relation targets
- `internal/serve/mcp.go` — extend `hero_drift` and `hero_context` MCP tools with cross-repo data
- `hero.json` schema documentation — document `repos` key

## Acceptance Criteria

- WHEN a spec's relation target contains a `/` separator THE SYSTEM SHALL parse it as `<repo-alias>/<spec-slug>` and resolve the spec from the configured repo path
- WHEN `hero.json` contains a `repos` map and `hero drift` runs against a spec with cross-repo `depends-on` relations THE SYSTEM SHALL check each upstream spec for modifications newer than the local spec and flag changes as warnings
- WHEN an upstream spec's status has changed to `superseded` or `completed` since the local spec was last modified THE SYSTEM SHALL flag the status change in the drift report with the new status value
- WHEN a cross-repo relation target references a repo alias not present in `hero.json` `repos` THE SYSTEM SHALL emit a warning naming the missing alias and continue processing remaining relations
- WHEN a cross-repo relation target references a spec slug that does not exist in the resolved repo THE SYSTEM SHALL flag the missing upstream spec in the drift report and exit non-zero
- WHEN `hero context` runs for a file whose active spec has cross-repo `depends-on` relations THE SYSTEM SHALL include the upstream spec's Goal and Acceptance Criteria sections in the context payload
- WHEN `hero check` runs and the project has cross-repo relations THE SYSTEM SHALL validate that all referenced repo paths are accessible and all relation targets resolve to existing specs
- WHILE a configured repo path is inaccessible (missing directory or no `.hero/` folder) THE SYSTEM SHALL log a warning for that repo and continue resolving other repos without failing the command

## Boundaries

- Does **not** require repos to be in the same git repository -- sibling directories, monorepo subdirectories, and absolute paths all work
- Does **not** sync or write specs between repos -- all cross-repo access is read-only
- Does **not** require a running server -- resolution is file-based using configured paths
- Does **not** auto-discover repos -- all repo aliases must be explicitly configured in `hero.json`
- Does **not** clone or fetch remote repos -- repos must already exist on the local filesystem
- Does **not** track cross-repo git history -- drift detection compares spec metadata (timestamps, status), not commit logs across repos
