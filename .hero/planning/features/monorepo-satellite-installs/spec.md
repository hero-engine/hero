---
title: "Monorepo Satellite Installs — One Workspace, Many Subfolder Entry Points"
slug: monorepo-satellite-installs
type: feature
status: delivering
priority: high
horizon: now
tags: [install, monorepo, harness-integration, scoping, team-config]
relations:
  - target: configurable-workspace-location
    kind: related
  - target: cloud-mcp
    kind: related
---

# Monorepo Satellite Installs — One Workspace, Many Subfolder Entry Points

## Problem

Real-world repositories converge. Teams that started with three repos — an app, an engine, a vendor library — end up merging them into one. After the merge there is one git repo, one CI pipeline, one source of truth, and *N* leftover `.hero/` directories scattered across subfolders. Concretely, example codebase today has:

```
example codebase/
├── .hero/                                  # root install
├── example codebase-engines/
│   ├── example codebase-mlx/.hero/                 # leftover from when this was its own repo
│   └── example codebase-cuda/.hero/                # leftover from when this was its own repo
└── example codebase-vendor/                        # not a hero project, but looks like one
```

Three things are broken at once:

1. **Fragmented graph and history.** Specs that span subprojects ("add CUDA fallback to MLX engine") cannot reference each other. Knowledge captured against `mlx` is invisible to a session in `cuda`. Recap, feed, why-traversal — all islanded. The corpus that's supposed to compound across sessions does not compound across the repo.

2. **Subfolder chats degrade silently.** When a developer `cd example codebase-engines/example codebase-mlx && claude`, what they actually get is a degraded session. Claude Code (and Codex, opencode, etc.) walk up the directory tree to find `CLAUDE.md` and `.mcp.json`, but they do **not** walk up for `.claude/agents/`, `.claude/commands/`, or `.claude/skills/`. Those load from cwd only. A subfolder chat opens with no slash commands, no Hero subagents, no skills — just the MCP server. The user does not see this; nothing tells them their session is half-equipped.

3. **No story for which folders are subprojects.** Hero already has subproject detection in `internal/scan/modules.go`, but nothing decides whether a detected folder *should* get a hero presence. Vendored deps look identical to first-party engines from the build-file signature.

The naive fix — duplicate the install in every subfolder — is what example codebase currently has. It produces drift the moment anything changes at root, multiplies graphs, and confuses tools that walk up looking for the workspace.

The right fix has three parts: (a) one real install at the root, (b) thin satellite entries in subprojects so chats opened there pick up everything that walks up *and* everything that doesn't, (c) a team-shared declaration of which subprojects are real and which are not.

## Goal

Make a Hero workspace **rooted once and reachable from any subfolder** within a monorepo, so that:

- A developer who opens a chat in any declared subproject gets the same harness experience (agents, commands, skills, MCP) as if they opened the repo root.
- Specs created in any subfolder land in the single root `.hero/` and are auto-stamped with a scope identifying the subproject.
- The set of valid subprojects is a team decision committed to git, while the per-machine set of materialized symlinks is local-only.
- Upgrades, repairs, and drift detection treat satellites as first-class managed state.

**Mission-fit.** This raises the floor: a junior dev who opens a chat in the wrong folder still gets the full Hero experience instead of a silently broken one. And it makes the next session smarter: specs across the entire monorepo accumulate in one graph, so traversals like `hero why` and `hero blocked` can cross subproject boundaries.

The non-goal is to support truly separate repositories with shared knowledge — that is what `hero repos` already does and is unrelated. This spec is for the case of **one git repo, multiple internal scopes**.

## Design

### 1. The two install modes

Hero installs are already parameterized by harness target (`claude`, `codex`, `opencode`, `cursor`, `copilot`, `generic`). This spec adds a second axis: **mode**.

- **Root mode** — the workspace lives here. Creates `.hero/`, full `<harness>/` directories with copied agents/commands/skills, settings files, and MCP configuration. This is what `hero install` does today.
- **Satellite mode** — a workspace lives at some ancestor directory. Creates symlinks in `<harness>/` pointing to that root's installed content, plus a marker file. Does **not** create `.hero/`. Does **not** create `.mcp.json` (the harness walks up to find root's).

Mode selection is automatic and based on cwd:

| State of cwd and ancestors | Action |
|---|---|
| `cwd/.hero/` exists | Root mode — install/upgrade in place |
| Some ancestor has `.hero/`, cwd does not | Satellite mode — symlink to that ancestor for installed targets |
| Neither cwd nor any ancestor has `.hero/` | Root mode — fresh install at cwd |

A new `--root` flag forces root mode at cwd even when an ancestor already has `.hero/`, for the rare case of intentionally nested workspaces. This is a destructive disclaimer prompt, not a casual flag — nesting workspaces is supported but discouraged.

### 2. What a satellite contains

For each harness target installed at root (e.g. `claude`, `codex`), the satellite gets:

```
example codebase-engines/example codebase-mlx/
├── .claude/
│   ├── agents     -> ../../../.claude/agents     (relative symlink)
│   ├── commands   -> ../../../.claude/commands   (relative symlink)
│   ├── skills     -> ../../../.claude/skills     (relative symlink)
│   └── settings.json                              (NOT symlinked — see below)
├── .codex/
│   └── ... same pattern
├── CLAUDE.md      (small marker file)
└── .hero-satellite (machine-readable marker)
```

**What is and is not symlinked, and why:**

- `<harness>/agents`, `commands`, `skills` are **subdirectory symlinks**. Any change at root reflects everywhere immediately. Relative symlinks survive moving the repo and survive checkout on different machines.
- `<harness>/settings.json` is **not symlinked**. Per-target settings may diverge per subproject (different model preferences, different permission allow-lists). If a satellite needs settings, they are project-shared via git in the subfolder, not derived from root.
- `<harness>/settings.local.json` is **not symlinked and not created**. It is per-user and gitignored.
- `.mcp.json` is **not created**. Harnesses walk up the tree to find root's. Creating one in the satellite would either duplicate the MCP server or shadow root's with a half-broken copy.
- `CLAUDE.md` (and the equivalent for other harnesses) is a **small generated marker**, not a copy. Contents:
  ```markdown
  Hero workspace at `../..` (relative to this folder).
  Specs and knowledge land at the root, scoped to `example codebase-engines/example codebase-mlx`.
  ```
  Harnesses already walk up for the *real* `CLAUDE.md` at root, so this satellite-local one is purely a hint surfaced inside the chat — it's the only way the user discovers their chat is correctly wired.
- `.hero-satellite` is a tiny machine-readable file: `{"root": "../..", "scope": "example codebase-engines/example codebase-mlx", "version": "<hero-version>"}`. The `hero` CLI reads this to know it is in a satellite without re-walking the tree.

### 3. Root walk-up for the `hero` CLI

The `hero` CLI must, in every command, locate the workspace. Today some commands assume cwd; this spec standardizes:

```
1. If cwd has .hero/, that is the workspace.
2. If cwd has .hero-satellite, read root from it.
3. Otherwise walk up: first ancestor with .hero/ wins.
4. If no ancestor has .hero/, error with "no Hero workspace found".
```

This walk happens once per command at startup and the resolved root + scope are passed through the command's context. No command implementation should call `os.Getwd` for workspace purposes after this point.

**Scope auto-stamping.** When the resolved workspace is satellite-or-walked-up, Hero computes the subfolder's path *relative to root* and offers it as the default scope for any spec/knowledge/note created in this run. Examples:

- `cd example codebase-engines/example codebase-mlx && hero design "fix tokenizer crash"` → spec written to root `.hero/planning/.../spec.md` with `scope: example codebase-engines/example codebase-mlx` in frontmatter.
- `cd example codebase-app/src/components && hero capture` → knowledge written to root `.hero/knowledge/...` with `scope: example codebase-app`.

The scope is **the longest declared subproject prefix of cwd-relative-to-root that appears in `subprojects.json`**, not the raw relative path. This means a developer working in `example codebase-app/src/components/feed/` gets `scope: example codebase-app`, not `scope: example codebase-app/src/components/feed`.

If cwd is not under any declared subproject, scope defaults to `<root>` (the workspace itself, no narrowing).

### 4. `.hero/subprojects.json` — the team config

A new file at `<root>/.hero/subprojects.json`, **committed to git**, declaring the canonical scope taxonomy for the workspace:

```json
{
  "subprojects": [
    {
      "path": "example codebase-app",
      "scope": "example codebase-app",
      "description": "User-facing application"
    },
    {
      "path": "example codebase-engines/example codebase-mlx",
      "scope": "example codebase-engines/example codebase-mlx",
      "description": "Apple Silicon engine"
    },
    {
      "path": "example codebase-engines/example codebase-cuda",
      "scope": "example codebase-engines/example codebase-cuda",
      "description": "NVIDIA engine"
    }
  ],
  "excluded": [
    "example codebase-vendor"
  ]
}
```

- `subprojects[]` lists folders that ARE valid subprojects. Each has a `path` (relative to root) and a `scope` identifier.
- `excluded[]` lists folders that look like subprojects (build files present) but should never be offered. This is the negative cache so re-running install doesn't re-prompt about `example codebase-vendor`.

When a developer clones the repo and runs `hero install`, this file is the source of truth. No prompts — the team already decided.

### 5. `.hero/satellites.local.json` — per-machine state

A new file at `<root>/.hero/satellites.local.json`, **gitignored**, tracking what has actually been materialized on this machine:

```json
{
  "version": 1,
  "satellites": [
    {
      "path": "example codebase-engines/example codebase-mlx",
      "targets": ["claude", "codex"],
      "installed_at": "2026-05-05T10:14:00Z"
    },
    {
      "path": "example codebase-app",
      "targets": ["claude"],
      "installed_at": "2026-05-05T10:14:00Z"
    }
  ]
}
```

This is the manifest that `hero install --repair`, `hero check`, and `hero uninstall` consult to find existing symlinks. Without it, repair would have to walk the entire tree.

### 6. The install flow

Single command, three contexts:

#### 6a. Fresh root install on a monorepo

```
$ cd example codebase/
$ hero install --target=claude
[ root install runs as today ]

Detecting subprojects...
  Found 4 candidate folders with build files:
    example codebase-app           (package.json)                    propose? [y/N/a/s/q/?]
    example codebase-engines/example codebase-mlx  (go.mod, existing .hero/) propose? [y/N/a/s/q/?]
    example codebase-engines/example codebase-cuda (go.mod, existing .hero/) propose? [y/N/a/s/q/?]
    example codebase-vendor        (package.json)                    propose? [y/N/a/s/q/?]
```

- `y` — add to `subprojects.json`, materialize satellite for the harness target(s) installed at root.
- `N` (default) — skip this run, but ask again next install. Not added anywhere.
- `a` — yes to all remaining (rare; implies user trusts the detection).
- `s` — skip all remaining for this run.
- `q` — quit prompting.
- `?` — show help including the option `x` to add to `excluded[]` (permanent skip).

After the walkthrough, two files are written: `subprojects.json` (committed-to-git, contains all `y` decisions) and `satellites.local.json` (gitignored, contains the materialized satellite entries).

The detector reasons: presence of `.hero/` or build files (`go.mod`, `package.json`, `Cargo.toml`, etc.) plus a "looks like a hero subproject" hint when an existing `.hero/` is found inside.

#### 6b. Satellite install in a subfolder

```
$ cd example codebase/example codebase-engines/example codebase-mlx
$ hero install --target=claude

Detected Hero workspace at /Users/.../example codebase
This is a subfolder of an existing workspace. Installing as satellite.

Subproject status: NOT declared in subprojects.json
  Add `example codebase-engines/example codebase-mlx` as a subproject? [y/N]
  (this writes to .hero/subprojects.json — commit it to share with your team)
```

If `y`: writes the entry to `subprojects.json`, materializes symlinks for `claude`, updates `satellites.local.json`. The user still has to commit the subprojects.json change for it to take effect for teammates.

If `N`: materializes symlinks anyway (this user clearly wants Hero in this folder), updates `satellites.local.json` only. The next teammate to clone won't get auto-satellited here, which is the right outcome — one user's preference is not a team decision.

#### 6c. Re-install after pulling new subprojects.json

```
$ git pull
$ hero install
Reconciling satellites against .hero/subprojects.json...
  example codebase-engines/example codebase-foo   declared, not materialized   create? [Y/n]
  example codebase-app                    declared, materialized       ok
  example codebase-engines/example codebase-mlx   declared, materialized       ok
```

Default `Y` for re-creating declared satellites because the team already decided. The user just needs to confirm they want them locally. `--yes` skips the prompt entirely, `--no` does the audit without changes.

### 7. Pre-existing full install in a subproject

This is the example codebase-today state. When `hero install` runs at the root and finds a subfolder with its own `.hero/`:

```
Found existing Hero workspace at example codebase-engines/example codebase-mlx/.hero/
This appears to be a leftover from a previous standalone repository.

  Convert to satellite? [y/N]
  (this will:
    - move the existing specs/knowledge to root .hero/ under scope `example codebase-engines/example codebase-mlx`
    - delete example codebase-engines/example codebase-mlx/.hero/
    - create satellite symlinks for installed harness targets)

  Or keep as nested workspace? [n] (rare; warns and continues)
```

The conversion is a structural migration: spec files move under `.hero/planning/.../<subproject>/...` if there is no collision, or get a scope-prefix in the file frontmatter if there is. Knowledge gets the same treatment. Events from the sub-`.hero/events.log` are appended to root with a `migrated_from` annotation. Graph state is rebuilt from scratch via `hero index` after the move; this is cheaper than trying to merge graph DBs.

This is destructive, prompts explicitly, and has a `--dry-run` mode that prints exactly what would move.

### 8. Upgrade and repair

`hero upgrade` (the CLI binary upgrade flow) and `hero install --repair` share the satellite reconciliation logic:

1. Re-stamp root agents/commands/skills (this is what upgrade does today).
2. For each entry in `satellites.local.json`:
   - Verify the satellite folder still exists (else drop from manifest).
   - Verify each symlink in `<harness>/` resolves to the current root path. Repair if broken.
   - If a new harness target was added at root since this satellite was created, prompt: "you installed `codex` at root since this satellite was created — extend `example codebase-engines/example codebase-mlx` satellite to codex too? [Y/n]"
3. For each entry in `subprojects.json` not in `satellites.local.json`, prompt: "declared subproject not materialized locally — create? [Y/n]"
4. For folders newly detected by the subproject scanner, surface as candidates only (never auto-add).

`hero check` runs the same logic in dry-run mode and reports drift.

### 9. Windows fallback

Windows symlinks require either developer mode enabled or admin rights. Hero detects this at install time:

- If the OS supports symlinks and the user has them: full satellite as described.
- If symlinks fail: fall back to **degraded satellite** — write the `CLAUDE.md` marker and `.hero-satellite` file, but **do not** copy or fake the agents/commands/skills directories. Then show:

  ```
  Symlinks unavailable on this Windows machine. Hero installed a marker
  at this folder telling chats opened here to use the workspace at
  ../.. — but slash commands and subagents will only be available
  when you open the workspace root directly.

  To enable full satellite support, enable Windows Developer Mode and
  run `hero install --repair`.
  ```

The "open root" doctrine is the explicit fallback. The marker ensures the user knows their chat is degraded; we do not silently produce a half-broken install.

Junction points are not used — they only work for absolute paths and break checkout portability, which is the whole point of the satellite design.

### 10. Cloud sync

Cloud syncs **scopes**, not satellites.

- `subprojects.json` is in git, so it ends up on the cloud the same way any source file does. The cloud graph derives the available scope taxonomy from this file.
- `satellites.local.json` is per-machine and never syncs. The cloud has no opinion on which symlinks exist on which laptop.
- Specs and knowledge created with `scope: example codebase-engines/example codebase-mlx` carry that scope to the cloud graph. Dashboard surfaces (filter, group-by, search) read this scope dimension.

The cloud-mcp federation story is unaffected: a satellite is a local materialization detail, invisible above the workspace boundary.

### Design decisions

**Why "satellite" and not "shim", "alias", or "subproject install"?** "Shim" implies translation/proxying, which isn't what this is. "Alias" suggests CLI command aliases, which is unrelated. "Subproject install" implies a real install in the subproject, which is exactly what we are *not* doing. "Satellite" captures it: a small, self-aware secondary entity that orbits a primary, contains no independent authority, and is materialized when needed. It also makes the inverse operation (`hero install --root`) read sensibly.

**Why split `subprojects.json` (team) from `satellites.local.json` (machine)?** They answer different questions. "What folders ARE subprojects of this monorepo?" is a project fact — same answer for everyone, belongs in git, code-reviewable. "What symlinks have been materialized on this laptop?" is a deployment artifact — different per user, per OS, per harness preference, irrelevant to teammates. Conflating them would either pollute git with per-machine noise or hide team decisions in gitignored files.

**Why subdirectory symlinks (`agents/`, `commands/`, `skills/`) instead of symlinking `<harness>/` itself?** Because `settings.json` and `settings.local.json` need to live alongside agents/commands/skills but follow different rules — settings may diverge per subproject, and `settings.local.json` is per-machine gitignored. If we symlinked the whole `.claude/` directory, those files would all share a single source of truth at root, which breaks both team and personal use. Symlinking only the always-shared subdirectories preserves the per-folder flexibility.

**Why no `.mcp.json` in satellites?** Harnesses walk up the directory tree to find `.mcp.json` — that is documented behavior. Creating a satellite copy means either (a) duplicating the MCP server registration (two servers spawned per session), (b) shadowing root's with a possibly-stale copy, or (c) maintaining a sync invariant we don't need. Relying on walk-up is simplest and matches how harnesses are designed.

**Why generate a satellite-local `CLAUDE.md` when the harness walks up to find root's anyway?** Because users need to know their chat is correctly wired. Without any local marker, opening a chat in a subfolder looks identical to a misconfigured project — there's no signal that "this works because of a satellite." A small marker that says "Hero workspace at ../.., scope is X" makes the wiring legible inside the model's context, which is the only place the user is going to look.

**Why default to `N` in the per-folder install prompt instead of `Y`?** Subproject auto-detection nets in vendored deps, sample apps, and false positives (any folder with a `package.json`). Defaulting `y` would force users to explicitly skip every false-positive on every re-run. Defaulting `N` makes the silent-default safe; the `a` shortcut covers the "I already know, all of these are real" case.

**Why does the prompt distinguish "skip" (`N`) from "exclude" (`x`)?** Skipping leaves the door open for future install runs to re-prompt — appropriate for "not yet" cases (new subproject still being scaffolded). Excluding is permanent and lands in `excluded[]` in `subprojects.json` — appropriate for `example codebase-vendor` and similar. Conflating them either silently re-prompts forever or silently makes "no" permanent. Two options, two outcomes.

**Why migrate existing sub-`.hero/` directories to satellites instead of leaving them?** Three subprojects with three `.hero/` directories means three graphs, three event logs, three `recap` outputs, three `why` traversals. The mission is one corpus that compounds — fragmenting it across nominal scopes inside one repo is the bug we are fixing. Leaving them as nested workspaces is supported but discouraged via the prompt path; the satellite path is the recommended migration.

**Why does `hero install --root` exist if nesting is discouraged?** Because there are real cases where someone genuinely wants a separate Hero workspace inside a larger repo (for example, a research subdirectory with very different conventions that wants a fresh corpus). Forbidding it would be wrong. Surfacing a destructive-disclaimer prompt is the right guardrail.

**Why does `hero install` walk the parent for `.hero/` instead of always operating at cwd?** Because the natural failure mode otherwise is "I forgot I was in a subfolder, I ran `hero install`, now I have a second workspace I didn't want." The walk-up makes the safe behavior the default and keeps the dangerous behavior (`--root`) explicit.

**Why store `.hero-satellite` as JSON when we already have a marker via the symlinks themselves?** Because the symlinks are the implementation detail; `.hero-satellite` is the *declaration*. Tooling needs a way to ask "is this a satellite?" without walking the tree to verify symlinks resolve. It also lets the file include the resolved root path (important when there are multiple `.hero/` candidates up the tree) and the scope, so `hero` CLI commands can answer "where do I write?" in O(1) from cwd.

**Why does upgrade re-prompt for new harness targets per existing satellite, instead of automatically extending?** Auto-extending would silently change which harnesses see Hero at each subproject. The satellite owner may have intentionally chosen "this folder gets Claude only because that's what I use here." The prompt is one keypress on the common case (`Y`) and prevents the rare-but-painful surprise.

**Why per-project event logs even though specs land at the root?** This is a non-decision: there is *one* event log at the root, because there is one workspace. Sub-`.hero/events.log` files only existed in the legacy nested-workspace state; once migrated, all events flow to the single root log with `scope` annotations.

**Why doesn't the cloud track satellites.local.json?** Because they are per-machine artifacts. The cloud's job is to sync the corpus (specs, knowledge, events with scope) and to power cross-machine surfaces. Symlink state doesn't appear on any of those surfaces; tracking it would be data without a consumer.

## Acceptance Criteria

- WHEN `hero install` is run AND cwd has a `.hero/` directory THE SYSTEM SHALL operate in root mode, modifying the workspace at cwd.
- WHEN `hero install` is run AND cwd does not have `.hero/` AND an ancestor directory has `.hero/` THE SYSTEM SHALL operate in satellite mode against that ancestor as the workspace root.
- WHEN `hero install` is run AND neither cwd nor any ancestor has `.hero/` THE SYSTEM SHALL operate in root mode, creating the workspace at cwd.
- WHEN `hero install --root` is run THE SYSTEM SHALL operate in root mode regardless of ancestor state, after displaying a destructive-action prompt warning that this will create a nested workspace.
- WHEN a satellite is materialized for harness target `<t>` THE SYSTEM SHALL create relative subdirectory symlinks `<t>/agents`, `<t>/commands`, `<t>/skills` pointing into the root workspace's installed `<t>/` directory.
- THE SYSTEM SHALL NOT symlink `<t>/settings.json` into a satellite, and SHALL NOT create `<t>/settings.local.json` in a satellite.
- THE SYSTEM SHALL NOT create `.mcp.json` in a satellite folder.
- THE SYSTEM SHALL NOT create a `.hero/` directory in a satellite folder.
- WHEN a satellite is materialized THE SYSTEM SHALL write a `.hero-satellite` JSON file at the satellite folder containing the relative path to root, the scope identifier, and the hero version that materialized it.
- WHEN a satellite is materialized THE SYSTEM SHALL write a per-harness marker file (e.g. `CLAUDE.md`) at the satellite folder containing a short note that the workspace lives at the resolved root and the active scope identifier.
- WHEN any `hero` CLI command starts THE SYSTEM SHALL resolve the workspace root by checking cwd for `.hero/`, then `.hero-satellite`, then walking up to the first ancestor with `.hero/`, and SHALL fail with a clear error if no workspace is found.
- WHEN a `hero` CLI command runs in a satellite or in a subfolder under root THE SYSTEM SHALL compute the active scope as the longest declared subproject prefix of cwd-relative-to-root present in `.hero/subprojects.json`, defaulting to the workspace itself if no prefix matches.
- WHEN a spec, knowledge entry, or note is created THE SYSTEM SHALL stamp the active scope into the artifact's frontmatter and SHALL write the artifact under the root `.hero/` regardless of the cwd it was invoked from.
- WHEN `hero install` runs in root mode AND `.hero/subprojects.json` does not yet exist THE SYSTEM SHALL detect candidate subproject folders via build-file signatures and existing `.hero/` directories, walk through each candidate one-by-one with options yes / no / yes-to-all / skip-all / quit / exclude / help, and persist all decisions to `.hero/subprojects.json` (yes entries to `subprojects[]`, exclude entries to `excluded[]`).
- WHEN `hero install` runs in root mode AND `.hero/subprojects.json` already exists THE SYSTEM SHALL reconcile materialized satellites against declared subprojects: declared-but-not-materialized entries SHALL be offered for creation (default yes), materialized-but-not-declared entries SHALL be flagged, and detected-but-undeclared candidate folders SHALL be surfaced for one-by-one decision.
- WHEN `hero install` runs in satellite mode in a subfolder not yet declared in `.hero/subprojects.json` THE SYSTEM SHALL prompt the user once whether to add the subfolder to `subprojects.json`, materialize symlinks regardless of the answer, and update `.hero/satellites.local.json` accordingly.
- WHEN a satellite is materialized or removed THE SYSTEM SHALL update `.hero/satellites.local.json` to reflect the change, including the satellite path, the harness targets covered, and a timestamp.
- THE SYSTEM SHALL persist `.hero/subprojects.json` as a tracked, committable file and SHALL persist `.hero/satellites.local.json` as a gitignored file.
- WHEN `hero install` (or `hero upgrade`) runs and a new harness target has been installed at root since a satellite was last touched THE SYSTEM SHALL prompt per existing satellite whether to extend it to the new target.
- WHEN `hero install --repair` is run THE SYSTEM SHALL verify each satellite in `.hero/satellites.local.json` exists, repair broken symlinks, drop manifest entries whose folder no longer exists, and reconcile against `.hero/subprojects.json`.
- WHEN `hero check` is run THE SYSTEM SHALL report satellite drift (broken symlinks, missing satellite folders, declared-but-not-materialized subprojects) without making changes.
- WHEN `hero uninstall` is run at the root workspace THE SYSTEM SHALL remove all materialized satellite trees listed in `.hero/satellites.local.json` in addition to the root install.
- WHEN `hero install` finds an existing `.hero/` directory inside a subfolder of an existing root workspace THE SYSTEM SHALL prompt whether to convert it to a satellite, and if accepted SHALL move specs and knowledge under root with scope-stamped frontmatter, append the sub-events.log to root with a `migrated_from` annotation, delete the legacy `.hero/`, and materialize the satellite.
- IF the host OS does not support relative symlinks (or the current user lacks privilege to create them) THEN THE SYSTEM SHALL fall back to writing only the marker files (`.hero-satellite` and per-harness markers), SHALL NOT create copies of agents/commands/skills, AND SHALL print a message instructing the user to enable symlink support or open the workspace root directly.
- WHERE Windows symlink fallback is in effect THE SYSTEM SHALL still update `.hero/satellites.local.json` to record the degraded satellite, and `hero install --repair` SHALL re-attempt full materialization on subsequent runs.
- THE SYSTEM SHALL NOT use NTFS junction points as a fallback for symlinks.

## Changes

### New files

- `internal/install/satellite.go` — satellite materialization, marker file writing, symlink creation with relative path resolution, OS capability detection.
- `internal/install/satellite_repair.go` — repair, drift detection, manifest reconciliation against `subprojects.json`.
- `internal/install/satellite_migrate.go` — convert nested `.hero/` to satellite, including spec/knowledge/event migration.
- `internal/install/subprojects.go` — read/write `.hero/subprojects.json`, manage declared-vs-excluded sets.
- `internal/install/satellites_local.go` — read/write `.hero/satellites.local.json`.
- `internal/workspace/locate.go` — single source of truth for resolving workspace root from cwd (walk-up + `.hero-satellite` reading + scope computation).
- `internal/workspace/scope.go` — compute active scope from cwd, root, and declared subprojects.
- `internal/cli/install_satellites.go` — `hero install satellites` subcommand for explicit re-prompting/reconciliation.
- `<repo>/.hero/subprojects.json` — committed team config (created on first satellite-aware install).
- `<repo>/.hero/satellites.local.json` — gitignored per-machine manifest.
- `<satellite>/.hero-satellite` — per-satellite marker file.

### Modified files

- `internal/install/install.go` — branch on root vs satellite mode at entry; satellite path delegates to `satellite.go`. Add `--root` and `--repair` flags. Add post-install subproject-walkthrough prompts in root mode.
- `internal/install/mcp.go` — no `.mcp.json` writes in satellite mode (it walks up); ensure root-mode `.mcp.json` correctly resolves when invoked from a satellite cwd.
- `internal/cli/install.go` — surface `--root`, `--repair`, and the `satellites` subcommand; thread mode resolution through.
- `internal/cli/uninstall.go` — read `satellites.local.json` and remove satellite trees alongside root.
- `internal/cli/check.go` — call satellite repair in dry-run mode and report drift in the existing health report.
- `internal/cli/root.go` — wire workspace-root resolution into the persistent pre-run hook so every command starts with `ctx.Workspace` populated.
- `internal/cli/design.go`, `deliver.go`, `diagnose.go`, `note.go`, `capture.go` — read active scope from context and stamp it into created artifacts; write artifacts under root regardless of cwd.
- `internal/scan/modules.go` — extend subproject detection to surface "is `.hero/` present here?" as a hint alongside build-file signatures.
- `internal/serve/mcp.go` — accept active scope in tool calls so MCP-driven artifact creation also stamps scope.
- `internal/refs/refs.go` — open the ref store at the resolved workspace root, not at cwd.
- `internal/digest/digest.go`, `internal/recap/recap.go`, `internal/feed/feed.go` — accept and surface scope when filtering / grouping (read-only changes; supports queries like "recap for scope `example codebase-engines/example codebase-mlx`").
- `internal/graph/schema.go` — `scope` is a property on existing spec / knowledge / event nodes (already string-typed metadata; this spec formalizes it as a recognized facet).
- `internal/graph/ingest.go` — ingest scope from artifact frontmatter and stamp it on graph nodes.
- `.gitignore` template — add `.hero/satellites.local.json`.
- `cmd/hero/main.go` (or equivalent) — ensure `--repair` and `satellites` reach `cli.install`.

## Phasing

### Phase 1 — Workspace walk-up and scope plumbing
The foundation. `internal/workspace/locate.go` and `scope.go`, the `.hero-satellite` marker format, walk-up wired into root pre-run, scope computation from cwd. CLI commands updated to read root from context. No satellite materialization yet; this phase makes "running `hero` from a subfolder finds the right workspace" true. Acceptance: `cd subfolder && hero status` prints the root workspace and the computed scope.

### Phase 2 — Satellite materialization (POSIX)
Implement `internal/install/satellite.go`: subdirectory symlink creation, marker file writing, harness-target detection from root install. `hero install` in a subfolder of an existing workspace produces a working satellite (Claude target only as the smoke-test path). `satellites.local.json` reads and writes. Acceptance: opening a chat in a satellite subfolder shows the same slash commands and subagents as opening the root.

### Phase 3 — `subprojects.json` and the install walkthrough
Team-config file format, the candidate-detection prompt with y/N/a/s/q/x/? options, persistence. Reconciliation between `subprojects.json` and `satellites.local.json` on re-run. Cross-target satellite extension prompt on upgrade. Acceptance: a teammate clones the repo, runs `hero install --target=claude`, and the declared subprojects are auto-materialized without prompting.

### Phase 4 — Migration of nested workspaces
The conversion path for example codebase-today: `hero install` detects nested `.hero/` directories under root, prompts to convert, migrates specs/knowledge/events with scope-stamping, and materializes satellites. Includes `--dry-run` and `--keep-nested` for the rare opt-out. Acceptance: example codebase ships with one `.hero/` at root, two satellite folders, all original specs reachable under their new scope.

### Phase 5 — Multi-target satellites and repair
Extend Phase 2 from claude-only to all installed harness targets. `hero install --repair` and `hero check` drift detection wired in. `hero uninstall` cleans satellite trees. Acceptance: upgrading hero on a machine with three satellites and two harness targets repairs all six symlink trees and reports any drift.

### Phase 6 — Windows fallback and degraded satellites
OS capability detection, marker-only fallback, repair-time re-attempt. Documentation of the "open root" doctrine as the explicit fallback. Acceptance: a Windows user without Developer Mode runs `hero install` in a subfolder and gets a clear marker-and-message install plus a `--repair` path forward; the root workspace continues to work normally for them when opened directly.
