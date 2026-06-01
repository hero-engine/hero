---
title: "Version Tracking & Project Upgrade"
slug: version-tracking-upgrade
type: feature
status: completed
created: 2026-04-13
milestone: v0.2
tags: [init, install, upgrade, versioning]
horizon: now
completed_at: 2026-05-18T19:25:38Z
---

# Version Tracking & Project Upgrade

## Goal

Hero should know what version scaffolded a project, detect when the binary is newer, and provide a clean `hero upgrade` command to bring the workspace up to date. No more `install --force` as the only upgrade path.

## Background

Today there is no version tracking:

- `hero init` creates `.hero/` with no version stamp
- `hero install` copies agents/commands/skills but doesn't record what version was installed
- When a user updates the hero binary, there's no way to know their project workspace is stale
- The only way to re-sync agent/command/skill files is `hero install --force`, which is confusing and destructive (it overwrites customizations without diffing)

Users need:
1. A way to know their project is out of date
2. A non-destructive upgrade path that preserves customizations
3. Visibility into version info in status/serve output

## Design

### Version stamp

On `hero init` and `hero install`, write a `.hero/version.json` file:

```json
{
  "hero_version": "0.2.3",
  "initialized_at": "2026-04-13T10:30:00Z",
  "last_install": {
    "version": "0.2.3",
    "target": "opencode",
    "mode": "project",
    "timestamp": "2026-04-13T10:31:00Z"
  },
  "last_upgrade": null
}
```

This file is managed by Hero, not the user. It lives inside `.hero/` so it's part of the workspace.

### Mismatch detection

On every CLI command that loads config (`hero status`, `hero check`, `hero serve`, etc.), compare the binary version against `version.json`. If the binary is newer, print a one-line warning to stderr:

```
hero: workspace was created with v0.2.3, binary is v0.3.0 — run 'hero upgrade' to update
```

This is a warning, not a blocker. Everything keeps working.

### `hero upgrade` command

```
hero upgrade              Upgrade the current project's workspace
hero upgrade --dry-run    Show what would change without modifying anything
hero upgrade --force      Overwrite customized files (default: skip modified)
```

What `hero upgrade` does:

1. **Compare versions** — read `.hero/version.json`, compare against binary version
2. **Update agents/commands/skills** — same as install, but:
   - If a file hasn't been modified by the user (checksum matches last install), overwrite it
   - If a file has been modified (user customization), skip it and warn
   - `--force` overrides this and overwrites everything
3. **Run schema migrations** — if `hero.json` format changes between versions, apply safe additive migrations (add new keys with defaults, never remove)
4. **Update version stamp** — write the new version to `.hero/version.json`
5. **Print summary** — files updated, files skipped (customized), migrations applied

### Checksum tracking for smart upgrades

Extend `version.json` to track file checksums:

```json
{
  "hero_version": "0.2.3",
  "installed_files": {
    "agents/engineer.md": "sha256:abc123...",
    "commands/prime.md": "sha256:def456...",
    "skills/agent-reliability.md": "sha256:789abc..."
  }
}
```

On upgrade, for each file:
1. Compute current file's checksum
2. Compare against stored checksum from last install
3. If they match → file is unmodified → safe to overwrite
4. If they differ → user customized it → skip and warn (unless `--force`)

### Version display

- `hero status` — show "Hero v0.2.3 (workspace v0.2.1 — upgrade available)" in header
- `hero serve` — show version in startup banner and `/health` response
- Dashboard UI — show version in sidebar footer

## Changes

### Go files

- `internal/version/version.go` — new: `VersionInfo` struct, `ReadVersion()`, `WriteVersion()`, `CheckMismatch()`, checksum helpers
- `internal/version/version_test.go` — new: tests
- `internal/cli/upgrade.go` — new: `hero upgrade` command with `--dry-run` and `--force`
- `internal/cli/init.go` — stamp version on init
- `internal/cli/install.go` / `internal/install/install.go` — stamp version + file checksums on install
- `internal/cli/status.go` — show version info in output
- `internal/cli/root.go` — register upgrade command, add version check middleware
- `internal/serve/server.go` — include version info in startup banner
- `internal/serve/api.go` — include workspace version in `/health` response

### No breaking changes

- Workspaces without `version.json` are treated as "unknown version" — upgrade works, just can't do smart checksum diffing on the first run
- All new fields in `version.json` are optional

## Acceptance Criteria

1. `hero init` writes `.hero/version.json` with the current binary version
2. `hero install` updates `version.json` with install target, timestamp, and file checksums
3. Running any hero command with a newer binary prints a one-line upgrade warning to stderr
4. `hero upgrade` updates agents/commands/skills, preserving user customizations
5. `hero upgrade --dry-run` shows what would change without modifying files
6. `hero upgrade --force` overwrites even customized files
7. `hero status` shows version info including mismatch warning
8. Workspaces without `version.json` (pre-existing) work without errors
9. File checksums enable smart diff — only unmodified files get overwritten on upgrade
