---
title: Cloud Spec Sync
slug: cloud-sync
type: feature
status: completed
tags: [cloud, sync, cli]
created: 2026-04-12
parent: hero-cloud
depends-on: [cloud-api, cloud-auth]
horizon: now
---

## Goal

Add `hero sync --cloud` to push spec metadata from the local CLI to Hero Cloud,
and pull aggregated cross-repo views back down. Sync is explicit (not automatic),
respects .gitignore, and only transmits metadata — not full spec body content
by default.

## Design

### What Gets Synced

By default, sync pushes **metadata only**:
- Slug, title, type, status
- Tags, claimed_by, tracker_id
- Relations (parent, depends-on, etc.)
- Section headings (not content)
- Files touched list
- Created/modified timestamps

Full spec body sync is opt-in via `hero sync --cloud --full` for teams that
want cross-repo spec search over full content.

### Sync Protocol

1. CLI computes a manifest of all specs with content hashes
2. CLI POSTs manifest to `POST /api/v1/orgs/:org/repos/:repo/sync`
3. Server diffs against stored state, returns list of specs needing update
4. CLI pushes changed specs
5. Server stores and indexes
6. Server returns aggregated summary (total specs across org, changes since last sync)

### Conflict Handling

Specs are git-committed, so the source of truth is always the local repo.
Cloud sync is one-way (push). The cloud stores the most recently synced state.
If two users sync the same repo, the latest push wins — git handles the
actual merge conflict.

### CLI Changes

```
hero sync --cloud           # push metadata to cloud
hero sync --cloud --full    # push full spec bodies
hero sync --cloud --status  # show sync status without pushing
```

### First-Time Setup

```
hero login                  # authenticate
hero sync --cloud           # auto-detects org from git remote, or prompts
```

## Changes

- CLI: `internal/cli/sync.go` — add `--cloud` flag and cloud sync logic
- CLI: `internal/cloud/client.go` — HTTP client for cloud API
- CLI: `internal/cloud/manifest.go` — spec manifest computation
- Cloud service: sync endpoint handler

## Acceptance Criteria

- `hero sync --cloud` pushes spec metadata to cloud
- Only metadata is sent by default (not full body content)
- `--full` flag enables full body sync
- Sync is idempotent — re-running with no changes is a no-op
- First sync auto-detects org from git remote URL
- Sync reports number of specs pushed/updated/unchanged
- `--status` shows last sync time and pending changes without pushing
