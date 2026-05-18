---
title: "Configurable Content Paths — point at existing dirs instead of rendering copies"
slug: configurable-content-paths
type: feature
status: completed
status_verified: "2026-05-11 by go test ./internal/install/... -count=1 — TestConfigurableContentPaths_* (4 cases) pass plus full install suite. Dogfooded on hero's own repo: hero.json content block points at agents/commands/skills at root; install produces .claude/{agents,commands,skills} symlinks directly to source dirs with ZERO rendering into .hero/. Second-run install: 'Installed 0 files'. Source skills migrated to <name>/SKILL.md nested layout (89 dirs across skills/, domains/engineering/skills/, core/skills/) so direct symlink works without flat-to-nested conversion."
priority: P1
tags: [install, config, dogfood, content, paths, hero-on-hero]
created: 2026-05-11
relations:
  - target: single-source-install
    kind: parent
  - target: single-source-install-p2-canonical-tree
    kind: follows
  - target: configurable-workspace-location
    kind: related
horizon: now
mission_alignment: |
  Hero working on hero is the cleanest dogfood signal we have. Right now
  hero-on-hero duplicates: source content lives at `agents/`, `commands/`,
  `skills/` (embedded into the binary), and `hero install` renders that
  SAME content into `.hero/agents/`, `.hero/commands/`, `.hero/skills/`
  on every install. The duplication is gitignored but it's still a smell
  — the install is doing work that's logically a no-op. Making the
  canonical path a config knob lets hero point at its own source
  directories directly, eliminating the redundant copies and making the
  "single source of truth" promise structurally true on hero's own
  workspace.
principles_check: |
  Serves "it just works" (#1) — install becomes more idempotent and less
  wasteful. Serves "the floor rises for everyone" (#4) — any other
  tool-developer who wants to dogfood their tool on itself gets the same
  clean experience hero gets. Risks "it just works" if the config knob
  becomes another confusing decision; mitigated by sensible defaults
  (today's behavior unchanged when the new fields are absent) and by
  hero-on-hero being the only common case for non-default values.
---

## Goal

`hero.json` gains a `content` field that lets a project declare where
its canonical agent/command/skill directories live. When set, `hero
install` uses the declared paths as canonical instead of materializing
copies into `.hero/{agents,commands,skills}/`. Harness directories
symlink to whatever the config points at.

Default behavior is unchanged: a project with no `content` field gets
the P2 canonical-tree behavior (rendered copies in `.hero/`). The
escape hatch exists for the rare case where a project already has
its content laid out somewhere — chiefly: hero itself, where the
embedded source lives at the repo root.

## Problem

Today, `hero install` on hero's own repo:

1. Reads embedded source from `agents/`, `commands/`, `skills/` (these
   are the `//go:embed`-ed paths and the true source of truth for what
   hero distributes).
2. Renders that source into `.hero/agents/`, `.hero/commands/`,
   `.hero/skills/` (the P2 canonical tree).
3. Creates symlinks `.claude/agents → ../.hero/agents` etc.

Steps 1→2 produce byte-identical copies — the canonical tree in `.hero/`
is just a shadow of `agents/` at the root. They're gitignored, but they
exist on disk after every install and the install does the copy work
every time `--force` is passed.

The duplication makes the install logic harder to reason about ("which
copy is real?"), slightly slower than necessary, and means a user
editing `agents/foo.md` in hero's repo sees the change in the embed but
NOT in the rendered `.hero/agents/foo.md` until `hero install` runs
again. That's exactly the kind of stale-content drift the
single-source-install initiative exists to prevent.

For non-hero projects this doesn't come up: they don't have source
content at the root; `.hero/` is genuinely canonical.

## Design

### `hero.json` schema addition

```json
{
  "content": {
    "agents_path": "agents",
    "commands_path": "commands",
    "skills_path": "skills"
  }
}
```

All three fields are optional. Each is a path relative to the project
root. When set, that path is the canonical source for the
corresponding content kind; when absent, the default
(`.hero/agents/`, `.hero/commands/`, `.hero/skills/`) is used.

For hero's own repo, `.hero/hero.json` gets:

```json
{
  "content": {
    "agents_path": "agents",
    "commands_path": "commands",
    "skills_path": "skills"
  }
}
```

For every other project, the field stays absent and behavior is
identical to P2 today.

### Resolution logic

`CanonicalDirs(targetDir)` becomes `CanonicalDirs(targetDir, cfg)` and
returns the configured paths if set, falling back to the
`.hero/<kind>` defaults:

```go
func CanonicalDirs(targetDir string, cfg *config.Config) (agents, commands, skills string) {
    base := filepath.Join(targetDir, ".hero")
    agents = filepath.Join(base, "agents")
    commands = filepath.Join(base, "commands")
    skills = filepath.Join(base, "skills")

    if cfg != nil && cfg.Content != nil {
        if cfg.Content.AgentsPath != "" {
            agents = filepath.Join(targetDir, cfg.Content.AgentsPath)
        }
        if cfg.Content.CommandsPath != "" {
            commands = filepath.Join(targetDir, cfg.Content.CommandsPath)
        }
        if cfg.Content.SkillsPath != "" {
            skills = filepath.Join(targetDir, cfg.Content.SkillsPath)
        }
    }
    return
}
```

### Skip-render-when-config-points-elsewhere

`installCanonical` writes into the resolved canonical paths only when
the resolved paths are inside `.hero/`. If the user has pointed
`content.agents_path` at e.g. `agents/` at the repo root, that path
is presumed to already exist (with content) — `installCanonical`
treats it as a no-op for that kind. The user is asserting "this is
my source already; don't materialize anything."

```go
func installCanonical(opts Options, result *Result) error {
    if opts.Mode != ModeProject || opts.TargetDir == "" {
        return nil
    }
    cfg, _ := config.Load(opts.TargetDir)
    agentsDir, commandsDir, skillsDir := CanonicalDirs(opts.TargetDir, cfg)

    // Only materialize a kind when its canonical path lives under .hero/.
    // External paths (config-pointed) are assumed pre-populated by the user.
    heroBase := filepath.Join(opts.TargetDir, ".hero")
    if strings.HasPrefix(agentsDir, heroBase) {
        if err := installFlat(opts, result, "agents", agentsDir); err != nil { ... }
    }
    if strings.HasPrefix(commandsDir, heroBase) {
        if err := installFlat(opts, result, "commands", commandsDir); err != nil { ... }
    }
    if strings.HasPrefix(skillsDir, heroBase) {
        if err := installSkillsNested(opts, result, skillsDir); err != nil { ... }
    }
    return nil
}
```

### Skills layout caveat

`installSkillsNested` writes `.hero/skills/<name>/SKILL.md` per the
Anthropic format. Hero's own `skills/` at the repo root is currently
laid out as flat `*.md` files (legacy source layout).

If a user points `content.skills_path` at a flat-layout directory,
Claude Code's Skill loader won't see those skills via the symlink
(it requires the `<name>/SKILL.md` directory layout).

Two acceptable resolutions:

1. **Migrate hero's source `skills/` to the SKILL.md layout** as a
   one-time refactor. `skills/foo.md` → `skills/foo/SKILL.md`. The
   embed paths update accordingly. Hero's own dogfood then has zero
   duplication. **Preferred.**

2. **Document that flat layouts are still rendered** (i.e., when
   `content.skills_path` points at a flat layout, hero falls back to
   rendering into `.hero/skills/` even with the override). Slightly
   weaker dogfood story.

P1 of this feature delivers the config knob + symlink behavior. The
hero source-layout migration to SKILL.md format is a tiny sibling
delivery that lands in the same change.

### Harness install flow

Unchanged structurally. Per-target installers call
`linkOrRenderDir(opts, result, kind, canonicalDir, harnessDir, ...)`.
The `canonicalDir` value flows through from `CanonicalDirs(targetDir,
cfg)`. Symlink target becomes the configured path (e.g.,
`.claude/agents → ../agents`) when the override is set.

### Install state

`install-state.json` gets a new field per target tracking the
resolved canonical paths:

```json
{
  "targets": {
    "claude": {
      "mode": "symlink",
      "canonical_paths": {
        "agents": "agents",
        "commands": "commands",
        "skills": "skills"
      }
    }
  }
}
```

Useful for `hero verify-install` (P4) to detect when a project's
config has changed without re-running install.

## Acceptance Criteria

- WHEN `hero.json` declares `content.agents_path` THE SYSTEM SHALL
  treat that path (relative to project root) as the canonical agents
  directory, and harness install symlinks target it directly
- WHEN a `content.*_path` field is absent THE SYSTEM SHALL fall back
  to the default `.hero/<kind>/` location (P2 behavior unchanged)
- WHEN a configured canonical path is OUTSIDE the `.hero/` workspace
  THE SYSTEM SHALL NOT render embedded content into it (the path is
  assumed pre-populated by the project)
- WHEN a configured canonical path is INSIDE `.hero/` THE SYSTEM
  SHALL materialize embedded content into it on install (same as the
  default behavior)
- WHEN a harness target installs in a project with overrides THE
  SYSTEM SHALL create symlinks pointing at the configured paths,
  computing relative-path targets correctly (e.g., `.claude/agents
  → ../agents`)
- WHEN hero installs against its own repo with `content.agents_path:
  "agents"` etc. THE SYSTEM SHALL produce `.claude/agents` →
  `../agents` directly, with no rendering into `.hero/agents/`, and
  no duplicate copies of agent files on disk
- THE SYSTEM SHALL record the resolved canonical paths per target in
  `install-state.json` so subsequent operations can detect config
  changes
- WHEN the configured path doesn't exist at install time THE SYSTEM
  SHALL exit with a clear error: `content.agents_path points at <X>
  but that directory doesn't exist; either create it or remove the
  override`
- THE SYSTEM SHALL preserve the SKILL.md directory layout requirement
  for skills — if `content.skills_path` points at a flat-layout
  directory, Hero either migrates the layout (preferred) or falls
  back to rendering into `.hero/skills/` with a warning

## Changes

- `internal/config/config.go` — new `ContentConfig` struct with
  `AgentsPath`, `CommandsPath`, `SkillsPath`; add `Content
  *ContentConfig` to `Config`
- `internal/install/canonical.go` — `CanonicalDirs` takes a config;
  `installCanonical` skips paths outside `.hero/`
- `internal/install/install.go` — load config in `Run()`, plumb to
  `CanonicalDirs` calls
- `internal/install/target_*.go` — call `CanonicalDirs(opts.TargetDir,
  cfg)` instead of `CanonicalDirs(opts.TargetDir)`
- `internal/install/state.go` — `TargetState` gains `CanonicalPaths
  map[string]string`
- `internal/install/configurable_paths_test.go` (new) — table-driven
  tests: default behavior unchanged, overrides resolve correctly,
  outside-`.hero/` paths skipped during materialization, symlinks
  use relative targets, error on missing configured path,
  install-state records configured paths
- Hero's own `.hero/hero.json` — add `content.*_path` entries
  pointing at `agents/`, `commands/`, `skills/`
- Hero's own `skills/` — migrate flat layout to `<name>/SKILL.md`
  directory layout. Update `//go:embed` paths in `content.go`
  accordingly. (Sibling deliverable; ships together.)
- Documentation: `docs/cli/install.md` (or wherever install is
  documented) describes when to use `content.*_path` overrides and
  the hero-on-hero example.

## Boundaries

- **Not in scope:** changing the default canonical layout for typical
  projects. The override is opt-in; absent fields preserve P2
  behavior exactly.
- **Not in scope:** absolute paths or paths outside the project root.
  Only project-relative paths are supported (security boundary —
  install shouldn't symlink at arbitrary filesystem locations).
- **Not in scope:** per-target path overrides. The config applies
  globally; every harness target's content dirs resolve to the same
  configured canonical paths.
- **Not in scope:** migrating non-hero projects to use overrides.
  This is mostly a hero-dogfood feature; other projects don't need
  it.
- **Not in scope:** changing the `domain/<x>/...` or `core/...` embed
  paths inside hero's source. Those remain the source of truth for
  the binary; the override just lets hero's repo also be a Hero
  workspace without duplicating content.

## Mission Fit

> "Does this make the next agent session start smarter than the last
> one ended — and does it raise the floor for everyone?"

Yes for hero-on-hero specifically: editing `agents/foo.md` in hero's
repo immediately becomes visible to Claude Code via the symlink
`.claude/agents/foo.md → ../agents/foo.md`. No "render step in
between" gap; no stale-rendered-copy footgun. The same pattern
unlocks "tool developers dogfooding their tool on itself" for any
future tool the Hero ecosystem grows — they get a clean development
loop without separate source vs. canonical trees.
