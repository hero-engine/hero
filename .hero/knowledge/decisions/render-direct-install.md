---
title: Render-Direct Install Architecture
type: decision
status: accepted
created: 2026-05-12
tags: [install, upgrade, architecture, harness, decision]
relations:
  - target: render-direct-install
    kind: implemented-by
  - target: single-source-install
    kind: supersedes
  - target: multi-harness-install-collision
    kind: builds-on
---

# Render-Direct Install Architecture

## Decision

`hero install` writes content directly from the embedded source to
each harness's documented destination paths. There is NO
`.hero/{agents,commands,skills}/` canonical mirror and NO symlinks
between harness dirs and a canonical tree.

`.hero/` is reserved for workspace metadata: planning, specs,
knowledge, NEXT.md, hero.json, etc.

Drift across multi-harness installs is prevented behaviorally —
`hero install --target X` auto-syncs other detected harnesses,
and `hero upgrade` refreshes every detected target by default.

## Context

The previous architecture (single-source-install P2) materialized
`.hero/{agents,commands,skills}/` as a canonical content mirror
and symlinked each harness's content directories at it
(`.claude/agents → ../.hero/agents`, etc.). Justified by an
observed drift pattern: in real projects, install moments staggered
across harness targets produced divergent copies — the example
codebase project had 14 of 34 agents differing between Claude and
OpenCode trees.

That architecture worked for the same-format case (Claude,
OpenCode, Cursor, Generic — all consume Markdown agents). It
broke uniformity for Codex (TOML agents) and Copilot
(`.prompt.md` files), which the
`harness-install-paths-match-loaders` spec papered over with a
hybrid "symlink for same-format kinds; render for format-divergent
kinds" approach.

The hybrid carried two parallel install code paths plus auto-migrate
machinery for dir→symlink transitions, Windows fallback paths, and
canonical-vs-rendered semantics across every install command.

## Forces considered

**For keeping symlinks:**
- Structural drift prevention (the original P2 justification).
- Single edit propagates to every harness automatically — but Hero
  canonical content is binary-owned (`go:embed`) and refreshed on
  upgrade; users aren't supposed to edit it.
- Smaller file count in projects with many harnesses installed.
- Hero-on-hero dev loop: editing top-level `agents/foo.md` is
  visible in `.claude/agents/foo.md` via the symlink.

**Against symlinks:**
- Hybrid architecture: Codex TOML and Copilot `.prompt.md` already
  render-direct. The "everything symlinks to canonical" story is a
  lie.
- Two install code paths (`linkOrRenderDir` + `renderToFile`) to
  maintain.
- Windows symlink fallback machinery.
- Auto-migrate complexity for dir→symlink transitions (legacy
  cleanup helper).
- IDE / file-tree surprise: opening `.claude/agents/foo.md` in
  VS Code shows symlink behavior.
- Inconsistency with what users see: half their harness dirs are
  "indirect," the other half are real files.

**Render-direct + auto-sync mitigation:** the original drift
problem (different install moments across harnesses) is solved
behaviorally — install/upgrade refreshes every detected harness in
one go. The trust-checksum machinery preserves user-edited files
across upgrades.

## Decision rationale

The hybrid was a compromise. Codex and Copilot forced rendering
anyway. Going full render-direct removes the symlink half and ends
up with one consistent install pipeline. Auto-sync covers the
drift case. The simplification touches significant code (delete
`installCanonical`, simplify `linkOrRenderDir`, delete
`canonical.go`, retire `verify.go` and `migrate.go` for the legacy
P2 architecture) but the result is uniform, easier to reason
about, and works the same on Windows.

Hero-on-hero dev loop cost is acceptable (~3s extra per
agent-authoring iteration; `hero dev --watch` is a future
affordance if friction grows).

## Implementation notes

- **`hero install --target X`** invokes `install.Run`, which:
  1. Runs `cleanupLegacyCanonicalSymlinks` — removes
     `.hero/{agents,commands,skills}/` mirror dirs and any harness
     symlinks pointing at them, but only when contents match
     embedded canonical bytes (user-edited content is preserved
     with a warning).
  2. Invokes the per-target `runX` which renders directly to the
     harness destination via `installFlat` / `installSkillsNested`
     / `renderToFile` (for format-divergent renderers like Codex
     TOML and Copilot `.prompt.md`).
  3. Auto-syncs detected sibling harnesses unless `--only-target`
     is set.

- **Trust-checksum machinery** (`Options.TrustedChecksums`) lets
  upgrade refresh files installed at a prior version's bytes
  without `--force`, while still refusing to clobber user-edited
  content.

- **Satellite materializer** (monorepo subprojects) still uses
  symlinks. That's a different concern: subproject root → parent
  project root indirection is fundamentally a symlink shape. Root
  install going render-direct doesn't force satellites to.

## Revisit if

- Real-harness drift incidents observed in spite of auto-sync (auto-sync
  bug → re-introduces drift). Layer 1 verification tests catch this in
  CI; if a bug slips through, escalate to structural enforcement.
- Hero-on-hero dev loop friction grows past acceptable. Add
  `hero dev --watch` then; if that doesn't help, reconsider.
- A future install target needs a format that requires content
  source live on disk (none anticipated).

## Files touched

Deleted:
- `internal/install/canonical.go`
- `internal/install/canonical_test.go`
- `internal/install/migrate.go`
- `internal/install/migrate_test.go`
- `internal/install/verify.go`
- `internal/install/verify_test.go`
- `internal/cli/verify_install.go`

Significantly simplified:
- `internal/install/linking.go` — shrunk to just the symlink
  capability probe used by state recording.
- `internal/install/install.go` — `Run` no longer materializes
  canonical mirror; runs legacy cleanup + per-target installer +
  auto-sync.
- All six `internal/install/target_*.go` — uniform shape.

Added:
- `internal/install/auto_sync.go` — multi-harness auto-sync.
- `internal/install/verification_test.go` — Layer 1 routine
  verification with real parsers (TOML, YAML, JSON), round-trip
  semantic checks, multi-target auto-sync coverage, legacy cleanup
  coverage.

## Related

- Spec: `.hero/specs/render-direct-install/spec.md`
- Supersedes: `.hero/specs/single-source-install/spec.md` and its
  P2 / P3 / P4 children.
- Initiative: `install-upgrade-contract-coverage` (this is a child).
