---
title: "Single-Source Install P2 — Canonical .hero Tree, Mode-Aware Harness Install"
type: feature
status: completed
status_verified: "2026-05-11 by go test ./internal/install/... -count=1 — all 30+ tests pass; dogfooded on hero's own repo (`make bootstrap`): .hero/{agents,commands,skills}/ materialized, .claude/{agents,commands,skills} symlinks created, install-state.json records mode=symlink, idempotent re-run produces no log lines, multi-target test verifies all harnesses share single canonical tree."
priority: P0
tags: [install, upgrade, harness, symlinks, config-redirect, drift-detection]
created: 2026-05-11
relations:
  - target: single-source-install
    kind: parent
  - target: single-source-install-p1-agents-md
    kind: follows
  - target: harness-instruction-file-survey
    kind: motivated-by
horizon: now
---

## Goal

Agent, command, and skill content lives in **one** canonical
filesystem location — `.hero/agents/`, `.hero/commands/`,
`.hero/skills/` — regardless of how many harnesses are installed.
Each harness reads the canonical content via whichever mechanism it
supports best: config-redirect, directory symlink, or (only when no
better option is available) rendered copies with drift detection.
`hero install --target X` selects the mode automatically based on
harness capability and host filesystem capability.

## Problem

Today, every harness target writes a physical copy of agent/command/
skill files into a harness-specific directory:

- `.claude/agents/`, `.claude/commands/`, `.claude/skills/`
- `.opencode/agent/`, `.opencode/command/`, `.opencode/skills/`
  (note opencode's singular naming for agent/command)
- `.openhands/skills/`
- `.codex/skills/`
- `.cursor/rules/`
- etc.

The same engineer.md gets written N times. Each install/upgrade
runs at a different moment; copies drift. Paperboy today: 14 of 34
agents differ between `.claude/agents/` and `.opencode/agents/`;
`.claude/skills/` has 44 files while `.opencode/skills/` has 79.

Hero's `targetLayouts` registry (`internal/install/satellite.go:57`)
already understands which subdirectories each harness uses, and the
satellite system already creates directory symlinks for sub-roots
(`SymlinkedDirs = ["agents", "commands", "skills"]`). The pattern
works — it just doesn't apply to root installs.

## Design

### Canonical layout

```
.hero/
├── agents/                 # canonical agent definitions
├── commands/               # canonical command definitions
├── skills/                 # canonical skill definitions (SKILL.md format)
├── ... (existing .hero contents)
```

A file lives in exactly one place. `.hero/agents/engineer.md` is the
source of truth; nothing else holds another copy.

### Per-harness install modes

For each harness target, Hero picks the cleanest mode that works:

| Harness | Preferred mode | Fallback chain |
|---|---|---|
| **opencode** | config-redirect via `opencode.json` `instructions: [".hero/AGENTS.md", ".hero/rules/*.md"]`; agent/command dirs via symlink (`opencode.json` has no agentsPath knob, but supports following symlinks per community reports) | symlink → rendered |
| **Claude Code** | directory symlinks: `.claude/agents → ../.hero/agents`, similarly for commands, skills, rules | rendered copies on Windows without Developer Mode |
| **Codex** | config-redirect via `.codex/config.toml` `model_instructions_file = ".hero/AGENTS.md"` and `skills.config[].path` | symlink → rendered |
| **Cursor** | symlink `.cursor/rules → ../.hero/skills` (if `.md` accepted) OR render `.mdc` copies (if extension mismatch matters — verify empirically during implementation) | rendered |
| **Copilot** | symlink `.github/copilot-instructions.md → ../../AGENTS.md`; `.github/instructions/` dir symlink to `.hero/` content if applicable | rendered |
| **Aider** | config-redirect via `.aider.conf.yml` `read: [".hero/AGENTS.md", ".hero/rules/*.md"]` | n/a (config-redirect always works) |
| **Cline** | rendered copies — symlinks broken (issue #3092 still open) | n/a — drift detection mandatory |
| **OpenHands** | symlink `.openhands/skills → ../.hero/skills` | rendered |
| **Windsurf, Amp, Junie, Roo, goose, Gemini CLI, Zed, Warp, Factory** | AGENTS.md at root (handled by P1) covers most; specific agent/command dirs by symlink if the harness uses them | rendered |

### Mode selection algorithm

```
for each installed harness target T:
    capability = T.preferred_mode   # config-redirect | symlink | rendered
    if capability == config-redirect:
        materialize via harness config file (idempotent merge)
    elif capability == symlink:
        if host supports symlinks (POSIX OR Windows with Developer Mode):
            create directory symlink: <harness-dir> → <.hero canonical>
        else:
            fall back to rendered copies + register drift detection
    elif capability == rendered:
        render copies into harness dir, register drift detection
```

The decision is deterministic per target per host. Logged in
`.hero/install-state.json` so subsequent `hero install` runs know
what was done.

### Host symlink capability detection

On first install (or via `hero install --probe`):

1. Attempt to create a test symlink in a temp directory.
2. If it succeeds, mark host as `symlinks_supported: true`.
3. If it fails (Windows without Developer Mode, restricted FS),
   mark `symlinks_supported: false`. All targets default to
   rendered-copy mode.
4. Result stored in `.hero/install-state.json` and refreshed if
   the user runs `hero install --probe` again (e.g., after enabling
   Developer Mode).

### Harness-specific shim files (config-redirect mode)

For harnesses that use config files, Hero writes (or updates) a
managed block within the harness config, similar to the AGENTS.md
managed-region pattern from P1.

**`opencode.json`** (JSON managed regions are awkward, so Hero owns
specific top-level keys it can detect):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "instructions": [".hero/AGENTS.md", ".hero/rules/*.md"],
  "hero": {
    "managed": true,
    "version": "0.7.1"
  }
  // user content elsewhere
}
```

Hero owns `instructions` (rewrites on upgrade) and `hero.*`. User-set
top-level keys are preserved. If `instructions` exists and isn't
Hero-managed, Hero detects this via the absence of `hero.managed:
true` and refuses to overwrite without `--force-managed`.

**`.codex/config.toml`** uses TOML comment markers (consistent with
existing `internal/install/mcp.go:247` pattern):

```toml
# hero:managed
model_instructions_file = ".hero/AGENTS.md"
project_doc_fallback_filenames = [".hero/AGENTS.md", "AGENTS.md"]

[[skills.config]]
path = ".hero/skills"
# end:hero:managed

# user content below this line preserved
```

**`.aider.conf.yml`** (YAML — uses key-namespace ownership):

```yaml
# hero:managed-start
read:
  - .hero/AGENTS.md
  - .hero/rules
# hero:managed-end

# user content preserved
auto-commit: true
```

For all three: re-running `hero install` against an already-correct
config is a no-op.

### Rendered-copy fallback + drift detection

When rendered-copy mode is used (Cline always; Windows-no-symlinks;
any harness flagged as symlink-broken), Hero:

1. Copies canonical content into the harness dir.
2. Writes a `.hero-rendered.json` manifest in the harness dir
   tracking: source path, destination path, content hash, hero
   version, timestamp.
3. Registers the dir for drift detection.

`hero verify-install` (delivered in Phase 4 but stubbed here) reads
the manifest, recomputes hashes, and reports drift. `hero install`
also auto-detects and offers to re-render. This way, rendered-copy
mode has the same correctness guarantees as symlink mode — drift
becomes visible and fixable, not silently corrosive.

### Idempotency contract

`hero install` and `hero upgrade` after this feature lands:

1. **Compute desired state** from harness targets installed +
   canonical content in `.hero/`.
2. **Compute actual state** by reading the filesystem and existing
   harness config files.
3. **Diff** and produce an operation list.
4. **Execute** only operations in the diff. If diff is empty, exit
   with "no changes" and zero filesystem writes.
5. **Log** the operation list to stdout in human-readable form and
   to `.hero/install-state.json` for next-time comparison.

This produces the property that running `hero install` against a
correctly-installed project is **always** a no-op, regardless of
how many times it's run.

### Interaction with the satellite system

The existing satellite system in `internal/install/satellite.go`
already implements directory symlinks (`SymlinkedDirs`) for sub-root
folders pointing at the workspace `.hero/`. This feature **extends
that pattern to the project root**:

- Phase 2's symlink mode reuses the satellite-symlink helpers.
- `LayoutFor`, `DetectInstalledTargets`, and `targetLayouts` get
  extended to track per-target mode (config-redirect / symlink /
  rendered) and config-file paths.
- Satellites continue to work as today: they symlink into the
  project root's `.hero/`, which now happens to be the canonical
  source the project root's harness dirs *also* reference.

This is a coherent extension, not a redesign.

## Acceptance Criteria

- THE SYSTEM SHALL store agent, command, and skill content in
  `.hero/agents/`, `.hero/commands/`, and `.hero/skills/`
  respectively, with exactly one copy per file in the project
- THE SYSTEM SHALL store each skill in canonical as a directory
  containing a `SKILL.md` file (e.g.
  `.hero/skills/context-injection/SKILL.md`), not as a flat
  `.md` file at `.hero/skills/<name>.md` — Claude Code's skill
  loader only registers skills laid out as `<name>/SKILL.md`, so
  any flat-file skill is silently invisible to the `Skill` tool
  even when the symlink and file content are otherwise correct
- WHEN an agent or command definition references a skill by name
  (e.g. "Load the `context-injection` skill before starting") AND
  that skill exists in canonical THE SYSTEM SHALL ensure every
  installed harness's skill-invocation mechanism resolves the
  name to the canonical SKILL.md — verified by a post-install
  smoke test per harness that enumerates skill references in
  agent/command files and asserts each is invocable
- WHEN `hero install --target opencode` runs on a host where the
  user has opencode installed THE SYSTEM SHALL update `opencode.json`
  to include `.hero/AGENTS.md` and `.hero/rules/*.md` in its
  `instructions` array, and SHALL NOT copy agent/command files into
  `.opencode/` directories
- WHEN `hero install --target claude` runs on a host that supports
  symlinks THE SYSTEM SHALL create directory symlinks
  `.claude/agents → ../.hero/agents`, `.claude/commands →
  ../.hero/commands`, and `.claude/skills → ../.hero/skills`,
  without creating physical copies
- WHEN `hero install --target claude` runs on a host without symlink
  support THE SYSTEM SHALL render copies of agents/commands/skills
  into `.claude/{agents,commands,skills}/` and write a manifest
  enabling drift detection
- WHEN `hero install --target cline` runs THE SYSTEM SHALL render
  copies into `.clinerules/` (because symlinks are known broken for
  Cline) and write a drift-detection manifest
- WHEN `hero install` runs on a project whose harness shims are
  already correctly configured THE SYSTEM SHALL produce zero
  filesystem writes and report "no changes" — verified by a test
  that runs install twice and asserts the second run's diff is empty
- WHEN `hero install --probe` runs THE SYSTEM SHALL determine
  whether the host supports symlinks by attempting a test symlink
  in a tempdir, and record the result in `.hero/install-state.json`
- WHEN a harness config file (`opencode.json`, `.codex/config.toml`,
  `.aider.conf.yml`) exists with user-authored content outside Hero's
  managed region THE SYSTEM SHALL preserve that content verbatim
  through every install and upgrade
- WHEN a harness config file's Hero-managed region is at a stale
  version THE SYSTEM SHALL regenerate just the managed region on
  `hero upgrade`, preserving user content outside it
- IF a harness config file's Hero-managed keys exist without the
  Hero-managed marker (suggesting user hand-edited) THEN THE SYSTEM
  SHALL refuse to overwrite without `--force-managed` and print a
  diagnostic message
- WHEN a harness target is removed (`hero uninstall --target X`)
  THE SYSTEM SHALL remove only that target's symlinks, rendered
  copies, and managed config entries, leaving `.hero/` canonical
  content and other harness targets untouched
- THE SYSTEM SHALL record per-target install mode in
  `.hero/install-state.json` so subsequent operations can detect
  capability changes (e.g., user enabled Developer Mode on Windows)

## Changes

- `internal/install/canonical.go` (new) — manage canonical
  `.hero/{agents,commands,skills}/` content; replace the per-target
  rendering paths currently in `install.go`
- `internal/install/mode.go` (new) — per-target install mode
  resolver (config-redirect / symlink / rendered); host capability
  probe; `.hero/install-state.json` reader/writer
- `internal/install/symlinks.go` (new) — directory symlink
  primitives shared by satellite system and root install (extract
  from `satellite.go` where appropriate)
- `internal/install/rendered.go` (new) — rendered-copy mode with
  manifest writing; stub `verify-install` hook for Phase 4
- `internal/install/configs/opencode.go` (new) — `opencode.json`
  managed-region merge
- `internal/install/configs/codex.go` (new) — `.codex/config.toml`
  managed-region merge (extend existing pattern from
  `internal/install/mcp.go:247`)
- `internal/install/configs/aider.go` (new) — `.aider.conf.yml`
  managed-region merge
- `internal/install/satellite.go` — extend `TargetLayout` to track
  preferred install mode and config-file path; update `targetLayouts`
  registry per harness-survey findings
- `internal/install/install.go` — replace per-target rendering with
  mode-aware install; preserve existing target-validation and
  flag-handling
- `internal/install/install_test.go` — extensive new coverage:
  config-redirect mode, symlink mode, rendered-copy fallback,
  idempotency (run twice, second run is no-op), uninstall isolation
- `internal/install/skill_smoke_test.go` (new) — per-harness
  post-install smoke test: enumerates every skill name referenced
  by agent/command files (e.g. `Load the \`X\` skill`), resolves
  to the canonical path through each installed harness's
  resolution mechanism, and asserts the SKILL.md exists and is
  reachable. Guards against the flat-file regression where
  `.claude/skills/context-injection.md` is present but Claude
  Code's `Skill` tool reports "Unknown skill"
- `internal/cli/install.go` — `--probe` flag; updated help text;
  improved output reporting
- `internal/cli/uninstall.go` — ensure target removal isolates by
  target
- `docs/cli/install.md`, `docs/cli/upgrade.md` — document modes,
  drift detection, manifest

## Boundaries

- **Not in scope:** the `--migrate` operation for legacy
  drifted-copies state — that's Phase 3.
- **Not in scope:** the full `hero verify-install` command — Phase
  4. This phase writes the manifest; reading and acting on it ships
  in P4.
- **Not in scope:** changing what agents/commands/skills are
  installed — strictly a packaging change.
- **Not in scope:** changing the AGENTS.md model (handled in P1).
  This phase assumes P1 has shipped and AGENTS.md exists at root.
- **Not in scope:** monorepo satellite installs (already work via
  the existing satellite system, which this phase extends but does
  not redesign).
- **Not in scope:** dynamic switching between modes on the fly
  (e.g., user enabling Developer Mode mid-session). Mode is locked
  per install; `hero install --probe` or explicit `hero install
  --refresh` re-evaluates.

## Mission Fit

> "Does this make the next agent session start smarter than the
> last one ended — and does it raise the floor for everyone?"

Yes — and structurally. Today, the floor is "as smart as the last
time some specific harness's install ran." After P2, every harness
in a project reads from the same canonical content updated atomically
by Hero. A team where Alice uses Claude Code and Bob uses opencode
sees the same agents and skills — automatically, without each of
them remembering to run `hero install --refresh` for their harness.
The floor rises particularly for multi-harness teams (which is the
direction the ecosystem is moving), and the maintenance tax becomes
constant rather than scaling with harness count.
