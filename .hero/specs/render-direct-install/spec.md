---
title: Render-Direct Install — Drop the .hero Canonical Mirror and Symlinks
type: feature
status: completed
priority: P0
completed: 2026-05-12
severity: high
created: 2026-05-12
tags: [install, upgrade, architecture, simplification, render, symlinks]
relations:
  - target: install-upgrade-contract-coverage
    kind: child
  - target: harness-install-paths-match-loaders
    kind: builds-on
  - target: single-source-install-p2-canonical-tree
    kind: supersedes
  - target: multi-harness-install-collision
    kind: builds-on
---

# Render-Direct Install — Drop the .hero Canonical Mirror and Symlinks

## Goal

Every install target renders content directly from the embedded
source to its harness-specific destination. No canonical-on-disk
mirror at `.hero/{agents,commands,skills}/`. No symlinks pointing
at it. `.hero/` becomes purely workspace metadata (planning,
specs, knowledge, NEXT.md, hero.json).

`hero install --target X` also refreshes any other detected
harnesses (auto-sync), preventing drift between install moments.
`hero upgrade` continues to walk all detected targets (unchanged
behavior).

## Kickoff

Architectural simplification — flipped from canonical-on-disk +
symlinks to render-direct per harness. `.hero/` is now reserved
for workspace metadata only (planning, specs, knowledge, NEXT.md,
hero.json). Every harness gets its content rendered directly from
the embedded `//go:embed` source to its documented destination.
Auto-sync keeps multi-harness projects at the same binary version.

**Status:** completed — full Go test suite passes, vet clean,
manual install smokes confirmed for every target (claude, codex,
copilot, opencode, cursor, generic), this repo's own state
migrated successfully from symlink-to-`../agents` layout to
rendered files.

**What landed:**
- Deleted `internal/install/canonical.go`, `migrate.go`, `verify.go`, and `internal/cli/verify_install.go`. Removed the legacy P2 architecture entirely.
- Simplified `internal/install/linking.go` to a 50-line symlink-probe-only file (probe stays because state.go records host capabilities; satellite materializer still uses symlinks).
- Rewrote all six `internal/install/target_*.go` to a uniform shape: `installFlat` / `installSkillsNested` / `renderToFile` from embedded source to harness destination. No mode-conditional branches.
- Added `internal/install/auto_sync.go` — `Options.AutoSyncTargets` makes install detect other installed harnesses and refresh them. CLI: `hero install --target X` sets it by default; `--only-target` opts out.
- Added `cleanupLegacyCanonicalSymlinks` in `internal/install/cleanup.go` — first install on the new architecture removes any leftover `.hero/{agents,commands,skills}/` mirror dirs and harness-dir symlinks pointing into `.hero/` or top-level canonical (Hero-authored bytes only; user-edited preserved with warning).
- Added `internal/install/verification_test.go` — Layer 1 routine verification with real parsers (`github.com/BurntSushi/toml`, `gopkg.in/yaml.v3`, `encoding/json`), round-trip semantic checks for Codex TOML, auto-sync coverage, and legacy cleanup migration coverage.
- Wrote `.hero/knowledge/decisions/render-direct-install.md` capturing the architectural decision and the criteria for revisiting.

**Validation done:**
- `go test ./...` — full suite passes (203+ specs, every install target's smoke + contract + migration tests).
- `go vet ./...` — clean.
- Manual install smoke per target in a throwaway dir:
  - Claude: 34 agents, 29 commands, 47 skills — all real files, no symlinks, no `.hero/agents/` mirror.
  - Codex: 34 TOML agents, zero markdown agents, 45 skills at `.agents/skills/`.
  - Copilot: 34 prompts at `.github/prompts/agents/`, 27 at `commands/`, 45 skills at `.github/skills/`, zero `.github/copilot/` content.
  - OpenCode, Cursor, Generic: clean rendered output at documented paths.
- Self-migration: this repo's `.claude/agents → ../agents` symlink was removed and replaced with a real directory of rendered files. No data lost (canonical lives in the binary now).

→ `hero spec complete .hero/planning/features/render-direct-install/spec.md`

**Why it matters:** the previous symlink architecture worked only
for same-format targets. Codex TOML and Copilot `.prompt.md`
forced rendering, making the architecture hybrid. Going
render-direct everywhere consistent removes the hybrid, the
Windows symlink fallback, the auto-migrate dir→symlink machinery,
and the conceptual overhead of "wait, is this a symlink?" while
preserving drift prevention via auto-sync. The dev-loop friction
(edit `agents/foo.md`, `go run ./cmd/hero install`) is acceptable
and addressable by a future `hero dev --watch` if it ever bites.

## Problem

Today's architecture has two install paths fighting each other:

- **Symlink path** (Claude/OpenCode/Cursor/Generic, all kinds;
  Codex/Copilot skills): canonical content materialized at
  `.hero/{agents,commands,skills}/`, harness dirs symlink to it.
- **Render path** (Codex agents → TOML, Copilot agents+commands
  → `.prompt.md`): rendered directly to harness dest, never goes
  through canonical.

The original justification (single-source-install-p2) was
"single canonical tree, every harness reads it" to prevent
install-time drift between harness copies. The justification
required uniformity to pay off. Codex's `.toml`-only loader and
Copilot's `.prompt.md` filename suffix forced us to render per
target — so we now carry both paths.

Costs of the hybrid:
1. **Two code paths** — `linkOrRenderDir` for symlinks +
   `renderToFile` for rendering, plus the auto-migrate helper
   `dirIsOnlyHeroAuthored` to handle dir→symlink migration.
2. **Windows symlink fallback** — `linkOrRenderDir` falls back to
   rendered copies when symlinks fail; another code path.
3. **IDE / file-tree surprise** — opening `.claude/agents/foo.md`
   in VS Code shows symlink behavior, not regular file. File
   watchers and git tools sometimes misbehave.
4. **Mental model overhead** — a reader of the install code has
   to know which (target × kind) cells symlink vs render.
5. **Cleanup machinery** — every install-path change has to
   handle "what if there's a legacy dir/symlink at the
   destination?"

The drift-prevention property symlinks provided is also
behaviorally achievable: have `hero install --target X` and
`hero upgrade` always refresh every detected harness. The user's
auto-sync proposal makes this explicit and structurally robust.

## Design

### Architectural layout (after)

```
project-root/
├── .hero/                         ← workspace metadata only
│   ├── planning/
│   ├── specs/
│   ├── knowledge/
│   ├── NEXT.md
│   ├── hero.json
│   └── (no agents/, commands/, skills/ here)
├── .claude/
│   ├── agents/engineer.md         ← rendered from embedded
│   ├── commands/design.md
│   ├── skills/spec-format/SKILL.md
│   ├── settings.json
│   └── CLAUDE.md (managed-block)
├── .opencode/
│   ├── agents/engineer.md
│   ├── commands/design.md
│   ├── skills/spec-format/SKILL.md
│   └── opencode.json
├── .codex/
│   ├── agents/engineer.toml       ← rendered TOML
│   ├── hooks.json
│   └── ... (already render-direct)
├── .agents/
│   └── skills/spec-format/SKILL.md  ← Codex cross-tool skills path
├── .github/
│   ├── copilot-instructions.md
│   ├── skills/spec-format/SKILL.md  ← Copilot skills
│   └── prompts/
│       ├── agents/engineer.prompt.md
│       └── commands/design.prompt.md
├── AGENTS.md
└── .cursor/rules/
    ├── agents/engineer.md
    ├── commands/design.md
    └── skills/spec-format.md
```

Each harness dir contains real files. No symlinks anywhere in
the install path. The harness sees what's documented as its
expected layout.

### Per-target installer shape (new uniform pattern)

```go
func runX(opts Options) (*Result, error) {
    destBase, err := resolveXPaths(opts)
    if err != nil { return nil, err }
    result := &Result{}

    // 1. Cleanup any legacy install artifacts (symlinks pointing at
    //    .hero/, or .hero/{agents,commands,skills}/ canonical
    //    materialization if this is the first render-direct install).
    if err := cleanupLegacyX(opts, result, destBase); err != nil {
        return nil, err
    }

    // 2. Render each content kind directly from embedded source.
    for kind, dest := range kindDests(opts, destBase) {
        if err := renderToFile(opts, result, kind, dest, xRenderer(kind)); err != nil {
            return nil, err
        }
    }

    // 3. Per-target extras (settings.json, hooks, AGENTS.md, MCP).
    return result, finalizeX(opts, result, destBase)
}
```

All six targets follow this shape. Differences live in:
- `resolveXPaths` (where the destBase is)
- `xRenderer(kind)` (what format to emit — markdown passthrough for most; TOML for Codex agents; `.prompt.md` for Copilot agents+commands)
- `finalizeX` (target-specific extras)

### Auto-sync on install

```go
// In install.Run, after the primary target install completes:
if opts.AutoSyncTargets {
    others := detectInstalledTargets(opts.TargetDir, excluding=opts.Target)
    for _, t := range others {
        sub := opts
        sub.Target = t
        sub.AutoSyncTargets = false  // prevent recursion
        sub.TrustedChecksums = trustedChecksumsFromInfo(loadVersionInfo(opts.TargetDir))
        if _, err := Run(sub); err != nil {
            fmt.Fprintf(os.Stderr, "warning: auto-sync %s failed: %v\n", t, err)
            // Continue; primary target install already succeeded
        }
    }
}
```

CLI wires:
- `hero install --target X` → sets `AutoSyncTargets: true`
- `hero install --target X --only-target` → opts out of auto-sync if a user explicitly wants single-target install
- `hero upgrade` → already walks detected targets; behavior unchanged

User-facing output:
```
$ hero install --target claude
Installed claude target.
Auto-syncing detected siblings: opencode, codex
  opencode: refreshed 47 files
  codex: refreshed 34 agents (TOML), 45 skills
Done.
```

### Migration of existing state

Existing Hero-installed projects have:
- `.hero/agents/`, `.hero/commands/`, `.hero/skills/` (canonical mirror)
- Harness dirs as symlinks: `.claude/agents → ../.hero/agents`, etc.

On the next `hero install` or `hero upgrade` after this spec
lands:

1. For each harness dest, if the path is a symlink whose target
   resolves into `.hero/`, remove the symlink. The render step
   that follows will create real files there.
2. After all per-target installs complete, remove
   `.hero/{agents,commands,skills}/` if they exist and their
   contents are detectably Hero-authored (existing `removeIfHeroAuthored`).
3. If any file under `.hero/{agents,commands,skills}/` is NOT
   Hero-authored (user-edited), leave it and emit a warning.

The migration is idempotent and runs every install — second
install after migration is a no-op since the symlinks and
canonical dirs are already gone.

### Hero-on-hero workflow

This repo has `agents/`, `commands/`, `skills/` at top level
(the `//go:embed` source) AND `.claude/agents → ../agents`
(symlink for live dev). After the flip:

- Top-level `agents/`, `commands/`, `skills/` remain (they're the
  embed source).
- `.claude/agents/` becomes a regular dir with rendered files
  (from embedded source).
- Dev workflow: edit `agents/foo.md` → `go run ./cmd/hero install --target claude .` → see in Claude Code.
- `go run` is fast enough (~3s) that this is acceptable friction.

If the friction becomes painful, add a future `hero dev --watch`
that mirrors top-level changes into `.claude/*` automatically.
Defer until needed.

### Code deletions / simplifications

**Deleted entirely:**
- `installCanonical` function
- `internal/install/canonical.go` (move `ResolveCanonicalDirs` to a smaller helper if anything still needs it; otherwise delete)
- `SkipCanonicalRender` option (no canonical to skip)
- `content.<kind>_path` overrides in `hero.json` config — hero-on-hero uses embedded source like everyone else
- The symlink branch in `linkOrRenderDir`
- `dirIsOnlyHeroAuthored` (it existed to gate auto-migrate from dir→symlink; we now go dir→dir, no special handling)

**Simplified:**
- `linkOrRenderDir` → rename to `renderDir`, body shrinks to "if exists and not force, skip; else render from embedded"
- `target_claude.go`, `target_opencode.go`, `target_cursor.go`, `target_generic.go` lose their `if opts.Mode == ModeProject { symlink } else { render }` branches
- `cleanup.go` keeps `removeIfHeroAuthored` (for legacy cleanup) but loses `dirIsOnlyHeroAuthored`

**Kept and reused:**
- Trust-checksum machinery (still useful for upgrade preserving user-edited files)
- `renderToFile` and the existing Codex TOML / Copilot `.prompt.md` renderers
- `installFlat` / `installSkillsNested` for the kinds that just pass markdown through

### Test layer plan

This delivery includes the Layer 1 verification tests we
discussed:

1. **Per (target × kind × scope) smoke tests** asserting
   file existence at the new (no-symlink) paths.
2. **Format-correctness tests** using real parsers
   (`github.com/BurntSushi/toml` for Codex agent TOML;
   `gopkg.in/yaml.v3` for YAML frontmatter; `encoding/json`
   for settings.json / opencode.json / hooks.json) — assert
   required fields per consuming harness's documented loader.
3. **Round-trip semantic checks** for rendered formats: render
   canonical agent → parse rendered TOML → assert body content
   matches input. Same for Copilot `.prompt.md`.
4. **Global-mode coverage** for the targets that support it
   (claude, opencode, codex) — extend installHarness with a
   `globalDir` field + controlled fake `HOME`.
5. **Migration tests**: fixture with `.hero/{agents,commands,skills}/`
   + symlinks at harness dests → run install → assert symlinks
   gone, real files at harness dests, canonical dirs cleaned up,
   user-edited content preserved with warning.
6. **Auto-sync test**: install target A in fresh dir; install
   target B → assert both A and B are at current binary version.
7. **Embedded-source integration test** in the root `hero`
   package: install.Run with `ContentFS = hero.ContentFS()`
   against a tempdir for each target; assert output. Catches
   "production install produces wrong bytes" even when fixture
   tests pass.

## Boundaries

- Do NOT touch the canonical source content at top-level
  `agents/`, `commands/`, `skills/`, `core/`, `domains/`. Those
  remain the `//go:embed` input.
- Do NOT change the satellite materializer
  ([internal/install/satellite.go](internal/install/satellite.go))
  in this spec. Satellites still use symlinks for monorepo
  subprojects (different concern). Surface as Followup if the
  satellite-vs-root inconsistency starts to bite.
- Do NOT remove `hero.json` `folder` config (which sets
  `.hero/` location). Only the `content.<kind>_path` overrides
  go away.
- Do NOT add new content kinds. Agents, commands, skills only.
- Do NOT introduce a build-side `hero dev --watch`. Defer until
  Hero-on-hero workflow friction is measured.

## Risks

- **Hero-on-hero dev loop friction.** Live-symlink convenience
  (`agents/foo.md` edits visible in `.claude/agents/foo.md`
  instantly) is gone. Mitigation: `go run ./cmd/hero install`
  is ~3s; acceptable. `hero dev --watch` is a future affordance
  if needed.
- **Migration removes symlinks user might have customized.**
  Mitigation: only remove symlinks that point at `.hero/` (Hero-managed
  targets); leave any other symlink in place. The trust-checksum
  + `removeIfHeroAuthored` machinery already handles user-edited
  files.
- **`hero install --target X` doing auto-sync changes
  semantics.** Mitigation: surface clearly in output
  ("Auto-syncing detected siblings: ..."); add `--only-target`
  flag for users who explicitly want single-target behavior.
- **Auto-sync recursion.** If A's sync triggers B's install
  which auto-syncs A... infinite loop. Mitigation:
  `AutoSyncTargets = false` is set on the recursive call
  (explicit in pseudocode above). One level deep, then stop.
- **Drift returns if auto-sync code is buggy.** Mitigation:
  Layer 1 tests include "install A, install B, both at current
  version" coverage. Catches auto-sync regressions.
- **Sunk-cost machinery from prior spec.** The `dirIsOnlyHeroAuthored`
  helper we just shipped becomes dead code. The trust-checksum
  helpers stay useful. Net: small deletion, no functional regression.
- **Other Hero-installed projects on your machine.** Anything
  outside this repo that was installed prior gets migrated on
  next install/upgrade against it. One-line warning + automatic
  cleanup.

## Validation

- `go test ./...` passes, including all new Layer 1 verification.
- `go vet ./...` clean.
- Manual smoke per target on a throwaway dir: every harness's
  expected files land at expected paths in expected formats.
- Manual migration smoke: take a fresh clone of this repo (which
  has `.hero/agents/` + symlinks today), run install — observe
  symlinks replaced with real files, `.hero/{agents,commands,skills}/`
  removed.
- Multi-harness drift test: install claude at v1, simulate binary
  upgrade, install opencode → assert claude is also refreshed
  via auto-sync; both at v2.

## Acceptance Criteria

- WHEN `hero install --target X` runs THE SYSTEM SHALL render
  X's content directly to its harness destination paths from the
  embedded source, with no `.hero/{agents,commands,skills}/`
  intermediate.
- WHEN `hero install --target X` runs AND other harnesses are
  detected as installed THE SYSTEM SHALL auto-sync those
  harnesses to the current binary version unless
  `--only-target` is set.
- WHEN `hero install` or `hero upgrade` runs against a project
  with legacy `.hero/{agents,commands,skills}/` canonical dirs
  + harness symlinks pointing at them THE SYSTEM SHALL remove
  the symlinks, render real files at the harness dests, AND
  remove the canonical dirs (Hero-authored bytes only;
  user-edited content preserved with a warning).
- IF a harness destination contains a symlink NOT pointing at
  `.hero/` THEN THE SYSTEM SHALL leave it untouched.
- WHEN every (target × kind × scope) smoke test runs THE SYSTEM
  SHALL produce destination files that parse cleanly with the
  consuming harness's documented format (real TOML / YAML / JSON
  parsers, not bespoke probes).
- WHEN rendered-format outputs (Codex TOML, Copilot `.prompt.md`)
  are round-tripped (canonical → render → re-parse) THE SYSTEM
  SHALL preserve the body content semantically equivalent to
  canonical.
- WHEN `Run` is invoked with `ContentFS = hero.ContentFS()` against
  any target THE SYSTEM SHALL produce the same destination files
  as fixture-driven tests.
- THE SYSTEM SHALL NOT install canonical content under `.hero/`
  for the kinds agents, commands, skills.
- THE SYSTEM SHALL preserve `.hero/`'s workspace metadata role
  (planning, specs, knowledge, NEXT.md, hero.json).
- THE SYSTEM SHALL NOT create symlinks for the agents, commands,
  or skills install destinations.

## Followups

- **`hero dev --watch`**: if the post-flip dev workflow for
  Hero-on-hero feels too slow, add a watcher that auto-re-renders
  top-level `agents/` changes into `.claude/agents/` (or any
  configured target).
- **Satellite materializer alignment**: today's satellite system
  uses symlinks for monorepo subprojects. After this flip, root
  install is render-direct and satellites are symlinked. Either
  bring satellites to render-direct too, or document why the two
  contexts differ. Out of scope here.
- **Real-harness verification** (parent initiative #6):
  unchanged. Layer 1 tests prove "Hero produces the right bytes";
  real-harness tests prove "consuming tools actually load them."
- **Architectural decision record**: add a knowledge doc at
  `.hero/knowledge/decisions/render-direct-install.md` capturing
  why we flipped from the P2 symlink architecture, so future
  contributors don't re-litigate.
