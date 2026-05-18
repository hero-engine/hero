---
type: feature
status: completed
severity: medium-high
tags: [install, snapshot, agents-md, managed-region, refactor, drift]
relates-to: [snapshot-architecture, next-as-projection-architecture, cli-invocation-drift-test-markdown, next-noop-writes]
---
# AGENTS.md Managed-Region Consolidation

## Context

Today AGENTS.md (and its siblings CLAUDE.md and NEXT.md) can carry up to two independently-managed regions, each with its own marker pair and its own writer:

1. **Install generator** — `internal/install/agents_md.go` calls
   `installManagedMarkdown` → `RenderManagedRegion` (`internal/install/managed_region.go:145`)
   to produce the bulk of the file (Hero CLI guide, slash command map, etc.) inside
   `<!-- hero:managed-start v=<version> -->` … `<!-- hero:managed-end -->` markers.
2. **Snapshot pointer** — `internal/snapshot/pointers.go` (`EnsurePointer`, called from
   `internal/snapshot/projector.go:118`) appends a separate block bracketed by
   `<!-- >>> hero snapshot pointer (managed) >>> -->` …
   `<!-- <<< hero snapshot pointer (managed) <<< -->`, after the install region.

Today they don't collide because the snapshot block is always *appended* below the
install end marker, and `EnsurePointer` is a one-shot insert with content-match
detection. But the arrangement is fragile:

- A future install rewrite that blindly truncates content after the install end
  marker would wipe the snapshot pointer.
- A user manually editing AGENTS.md sees two unrelated "managed by hero" blocks
  with no orchestration between them, which is confusing.
- The pattern doesn't scale: a third subsystem wanting to inject content (say
  `hero peer` wanting to advertise registered peers) would add a third independent
  marker pair, deepening the same problem.
- The existing install primitive (`InsertManagedRegion`,
  `internal/install/managed_region.go:175`) already knows how to preserve outside
  content byte-for-byte. The snapshot pointer reinvents a weaker variant.

NEXT.md has only one writer today (snapshot pointer), so the consolidation reduces
to "use the shared primitive" rather than a merge — but it's worth folding into
the same orchestrator so the marker convention is consistent across all three
files.

Related decision specs `snapshot-architecture` and `next-as-projection-architecture`
are being scaffolded in parallel; the orchestrator design here should compose with
whatever shape those land on. Treat their slugs as forward references in
`relates-to`.

## Kickoff

Delivered. AGENTS.md / CLAUDE.md / NEXT.md now carry exactly one Hero-managed region, populated by an ordered list of `SectionContributor`s. Legacy two-block files migrate automatically on the next install or snapshot projection.

**Status:** completed.

**Where it landed:**
- New package `internal/managed/` owns the marker primitives (moved down from `internal/install`) plus the `Writer` orchestrator and inline `stripLegacySnapshotBlock` migration.
- `internal/install/managed_region.go` is now a thin re-export shim — public Go API (`install.RenderManagedRegion`, `install.InsertManagedRegion`, `install.FindManagedRegion`, `install.IsLegacyHeroStub`, `install.ManagedRegion`) preserved for in-package callers.
- `internal/install/agents_md.go` and `claude_md.go` route through `installManagedMarkdown` which now calls `managed.Writer{Sections: defaultSections(...)}.Write(...)`. Install body and snapshot pointer compose as two `SectionContributor`s.
- `internal/snapshot/pointers.go` exposes `NewPointerSection` and routes `EnsurePointer` through `managed.Writer` (single-contributor case for NEXT.md, fallback for AGENTS.md when install hasn't run).
- `internal/install/agents_md.go`'s `RenderAgentsMdBodyForDriftTest()` now returns the orchestrator-rendered body so the markdown drift test reflects production output.

**Pointer location change:** the snapshot pointer used to live at the *bottom* of AGENTS.md in its own marker pair. Post-consolidation it lives inside the single managed region under an H2 `## Project snapshot` heading.

**Skip:** adding a section contributor for `hero peer` or any other subsystem that doesn't write to AGENTS.md today.

## Goal

One marker-bounded "Hero managed" region per file (AGENTS.md, CLAUDE.md, NEXT.md),
populated by an ordered list of named section contributors. Existing files with the
legacy two-block layout migrate cleanly to the consolidated layout on the next
`hero init` / `hero install` run, without touching user content outside any marker
pair. Re-running the orchestrator is a no-op when nothing has changed.

Done means:

1. AGENTS.md / CLAUDE.md / NEXT.md emitted by a fresh install carry exactly one
   `hero:managed-start` / `hero:managed-end` pair, with named subsections inside.
2. An existing AGENTS.md carrying both the install block and the snapshot pointer
   block is rewritten to the new single-block form on the next install or
   snapshot projection run, with the old snapshot marker pair fully removed.
3. A unit test asserts the "exactly one managed-region marker pair per file"
   invariant and fails on regression.
4. User content above, below, and between any old marker pairs is preserved
   byte-for-byte across the migration.

## Approach

### Orchestrator design

Add a new package `internal/managed/` housing the orchestrator. It's used by both
`internal/install` and `internal/snapshot`, so it must not import either (no
upward dependencies). The naming mirrors existing single-purpose packages in
`internal/` (`internal/handoff`, `internal/recap`, `internal/snapshot`).

```go
// internal/managed/region.go — sketch, refine during implementation.

// Context is what contributors receive at render time. Kept small on
// purpose — anything load-bearing should be passed in by the orchestrator
// caller, not pulled from globals.
type Context struct {
    File          string // absolute path of the file being rendered
    HeroVersion   string // for the start-marker version stamp
    ProjectDir    string // project root (for resolving relative pointers)
}

// SectionContributor is one named section inside the single managed region.
type SectionContributor interface {
    SectionID() string             // stable id, used for ordering + dedup
    SectionTitle() string          // human-readable heading rendered inside
    Render(ctx Context) (string, error)
}

// ManagedRegion is the per-file aggregator. Sections are written in the
// order they appear in the slice — there is no auto-sort. The caller
// declares the canonical order in one place.
type ManagedRegion struct {
    File     string
    Sections []SectionContributor
}

func (m *ManagedRegion) Write(ctx Context) error // re-renders the whole block
```

The render path:

1. Iterate `Sections` in order; collect rendered bodies under their section
   titles. An empty body for a contributor is skipped (so a contributor can
   no-op without leaving an empty heading).
2. Concatenate into one body with a stable subsection delimiter (e.g. an
   `## <SectionTitle>` heading inside the region).
3. Hand the body to the existing install primitive
   (`install.RenderManagedRegion` + `install.InsertManagedRegion`) to wrap
   markers and splice into the file. This keeps the marker syntax,
   versioning, and outside-content preservation in one place.

### Caller composition

There is no global registry. Each call site that needs to write a file
constructs the `ManagedRegion` with the contributors it wants, in the
canonical order:

```go
// in install
mr := managed.ManagedRegion{
    File: agentsMdPath,
    Sections: []managed.SectionContributor{
        install.NewAgentsMdBodySection(opts), // current generateAgentsMdBody
        snapshot.NewPointerSection(projectDir),
    },
}
return mr.Write(ctx)
```

This avoids the singleton-registry problem (order surprises, init-order
races, hidden side effects). The canonical order lives in the install
flow; snapshot's `EnsurePointer` becomes a contributor exposed to the
install package, plus a thin wrapper that calls the orchestrator for the
snapshot-only file (NEXT.md).

Dependencies point one way: `install` and `snapshot` depend on `managed`;
`managed` depends on neither. The `install` package may import `snapshot`
to wire `NewPointerSection` into its section list — that import already
exists indirectly via the projector path, and the alternative (a global
registry inside `managed`) is worse.

### Migration

`internal/managed/migrate.go` (or sibling file in the same package) owns the
one-shot consolidation. On any `Write` call:

1. Read the existing file.
2. Detect old layout: presence of the install marker pair
   (`install.FindManagedRegion` returns `Present=true`) AND a snapshot
   marker pair (`<!-- >>> hero snapshot pointer (managed) >>> -->` …
   `<!-- <<< hero snapshot pointer (managed) <<< -->`).
3. If both pairs are present and the snapshot pair is *outside* the install
   region:
   a. Strip the snapshot pair (markers and enclosed content) from the file
      content. The contributor will re-emit the same content inside the
      consolidated region during this same write.
   b. Proceed with the normal render path. The install region (now the only
      surviving marker pair) is rewritten as the single consolidated region.
4. If only the install pair is present, or only the snapshot pair, treat as
   normal first-time write — `InsertManagedRegion` handles both cases.
5. If neither is present, create the consolidated region per the existing
   "first-time write" rules in `InsertManagedRegion`.

The migration runs inline in `ManagedRegion.Write`, not as a separate
command. It only mutates content inside known marker pairs; user content
outside any marker pair is never touched. It is idempotent — once the file
is in the single-block layout, step 2's detection returns false and the
write is a normal regenerate.

The legacy single-marker form (`<!-- hero:managed -->`) is already handled
by `install.FindManagedRegion` and does not need separate handling here.

### Design note: dependency direction (chosen during delivery)

**Decision:** Option A from the Risks section — move the marker primitives
(`FindManagedRegion`, `RenderManagedRegion`, `InsertManagedRegion`,
`IsLegacyHeroStub`, plus the `ManagedRegion` struct and marker constants)
down from `internal/install/managed_region.go` into the new
`internal/managed/` package. `internal/install` then depends on
`internal/managed`; `internal/managed` depends on neither install nor
snapshot.

**Why A over B/C:**
- Audit shows only 5 call sites for these primitives — all inside
  `internal/install` itself. The blast radius of the move is small.
- The orchestrator MUST be able to render and insert managed regions.
  Option B (orchestrator lives in `install`) re-creates the
  "install package owns more than install" problem this spec exists to
  fix.
- Option C (third package) is unnecessary indirection: only one consumer
  (the orchestrator) sits between the primitives and the install/snapshot
  use sites. The primitives and the orchestrator belong together as the
  "managed-region engine."
- `IsLegacyHeroStub` is install-specific in intent ("is this a Hero stub
  CLAUDE.md we can replace?") but mechanically it's a pure function over
  the marker primitives — moves cleanly without breaking the abstraction.

**Net dependency edges after move:**
- `internal/managed` → no internal deps
- `internal/install` → `internal/managed` + `internal/snapshot` (the
  latter exists indirectly via the snapshot section contributor wiring)
- `internal/snapshot` → `internal/managed`

### Marker convention

Keep the existing install markers (`<!-- hero:managed-start v=X -->` /
`<!-- hero:managed-end -->`). They are versioned, well-tested, and already
support outside-content preservation. The snapshot marker pair is removed
entirely by the migration — its content moves inside the install-style
markers as a named section.

Do **not** introduce a new marker phrasing. The point of the consolidation
is one canonical marker pair across the codebase; changing the phrasing
would be a separate concern and would force a second migration.

### Subsection layout inside the region

Inside the single region, contributors render under their `SectionTitle`
as an H2 heading (the body itself uses H3+). This gives a visible separator
in AGENTS.md so the user can see which subsystem owns which chunk. Section
order is fixed by the caller's slice: install body first, snapshot pointer
last.

## Changes

1. **New package `internal/managed/`**
   - `internal/managed/region.go` — `Context`, `SectionContributor`, `ManagedRegion`,
     and `Write` orchestrator. Delegates marker rendering to
     `install.RenderManagedRegion` / `install.InsertManagedRegion` (acceptable
     downward dep on `internal/install`'s primitive — see Risks).
   - `internal/managed/migrate.go` — detect-old-layout helper used by
     `Write`. Pure string ops, no I/O of its own.
   - `internal/managed/doc.go` — package-level overview comment mirroring the
     `internal/install/managed_region.go` header style.

2. **Refactor `internal/install/agents_md.go`**
   - Extract `generateAgentsMdBody` into a `SectionContributor`
     implementation (`agentsMdBodySection`). `SectionID()` returns
     `"install:agents-md-body"`, `SectionTitle()` returns the existing
     H2 heading ("Hero — Spec-Driven AI Engineering").
   - Drop the direct call to `RenderManagedRegion` from
     `installManagedMarkdown`; build a `managed.ManagedRegion` with the
     install body section + snapshot pointer section, and call `Write`.
   - Keep `installManagedSpec` and the existing dry-run / skip behaviour;
     just route the actual write through the orchestrator.

3. **Refactor `internal/snapshot/pointers.go`**
   - Replace `EnsurePointer` / `ensurePointerInFile` internals with a
     `SectionContributor` implementation (`pointerSection`) and a thin
     facade `EnsurePointer` that constructs a single-contributor
     `managed.ManagedRegion` and writes it. This keeps the existing
     public API for `projector.go` callers.
   - Remove the snapshot-specific marker constants
     (`pointerMarkerStart` / `pointerMarkerEnd`) — they exist only for
     migration detection, which moves to `internal/managed/migrate.go`.
   - Keep `PointerLine` and `buildPointerLine` — they remain the canonical
     pointer string the section contributor renders.

4. **Migration logic** (in `internal/managed/migrate.go`)
   - `detectLegacySnapshotBlock(content) (start, end int, found bool)` —
     locates the old snapshot marker pair by byte offset.
   - `stripLegacySnapshotBlock(content) string` — returns content with the
     old block removed, normalising seam whitespace (same rules as
     `install.InsertManagedRegion`).
   - Both are called from `ManagedRegion.Write` before the install
     primitive renders the new region.

5. **Wire callers**
   - `internal/snapshot/projector.go:118` (`EnsurePointer` call) — still
     calls the snapshot package's facade. No change at the call site, but
     the facade now goes through the orchestrator.
   - `internal/install/agents_md.go` `installAgentsMd` and the CLAUDE.md
     analog in `internal/install/claude_md.go` — both build their
     `ManagedRegion` with the same two contributors in the same order.
   - NEXT.md path (called from `internal/snapshot/projector.go`) builds a
     `ManagedRegion` with only the snapshot pointer contributor.

6. **Tests**
   - `internal/managed/region_test.go` — table-driven coverage of:
     orchestration with multiple sections, stable ordering, empty-body
     skip behaviour, idempotent re-write (second call produces no
     filesystem mutation), migration from the legacy two-block layout
     (assert exactly one marker pair afterward and that user content
     outside both old pairs is preserved byte-for-byte).
   - `internal/managed/migrate_test.go` — focused tests on the
     legacy-detection + strip helpers using fixture strings.
   - `internal/managed/marker_test.go` — the primitive-level tests
     (formerly inside `internal/install`) now live here as part of the
     primitive-down move (Option A in the dependency-direction design
     note). Renamed `ManagedRegion` references to `Region` to match the
     package's parse-struct name. Tests assert the same behavior as
     before.
   - `internal/snapshot/pointers_test.go` — update expectations: writes
     now produce the consolidated-region layout, not a standalone pointer
     block. Add a regression test for "AGENTS.md with both old blocks →
     one block after migration."
   - **Invariant test** — add an assertion (in
     `internal/managed/region_test.go` and, if it lands first, in the
     drift-test harness from spec `cli-invocation-drift-test-markdown`):
     for every rendered AGENTS.md / CLAUDE.md / NEXT.md fixture in the
     repo, exactly one `hero:managed-start` and one `hero:managed-end`
     marker is present. This catches accidental regression of the
     consolidation.

## Boundaries

- **No new section contributors.** This spec adds the orchestrator and
  migrates the two existing writers. Adding a `hero peer` contributor or
  any other subsystem is explicitly out of scope — mention them only as
  future extension points in code comments.
- **No marker rename.** The existing `hero:managed-start` /
  `hero:managed-end` markers stay. Changing marker phrasing is a separate
  concern with its own migration cost.
- **No change to `InsertManagedRegion` semantics.** The orchestrator
  reuses the primitive as-is. If a bug in outside-content preservation is
  found, file a separate bug; this spec doesn't touch those rules.
- **No standalone `hero migrate` command.** Migration runs inline on the
  next `hero init` / `hero install` / snapshot-projection invocation. A
  dedicated command would imply opt-in migration, which fights the
  "files in the wild get fixed quietly on next write" goal.
- **No change to the legacy single-marker form** (`<!-- hero:managed -->`).
  `install.FindManagedRegion` already handles it; the orchestrator
  inherits that behaviour.
- **No removal of `install/managed_region.go` primitives.** They remain
  the low-level marker engine; the new orchestrator sits on top.

## Risks

- **Behavior change: hand-authored pointer outside the managed region.**
  Pre-consolidation, `EnsurePointer` would skip writing if the user had
  manually inserted the canonical `PointerLine` text anywhere in the
  file. Post-consolidation, the orchestrator always emits the pointer
  inside the managed region; a user-authored copy outside the region
  would coexist (duplicate pointer line in the rendered file). This
  matches the broader "Hero owns content inside markers; user owns
  content outside" contract — a hand-authored copy outside the region
  is no longer treated as a sentinel that suppresses Hero's own
  emission. Rare in practice; if a user runs into duplication they can
  delete their hand-authored line.
- **Files-in-the-wild upgrade risk.** Users on v0.10.0 have AGENTS.md with
  the two-block layout. The migration only fires when `installAgentsMd` /
  `EnsurePointer` next runs. Acceptable for `hero install` / `hero init`
  (those are write paths users invoke deliberately), but **`EnsurePointer`
  is called from the snapshot projector on a read-adjacent code path**
  (`internal/snapshot/projector.go:118`). Confirm during implementation
  that the projector is itself only invoked from explicit write commands
  (`hero snapshot`, etc.) and not from an inadvertently-side-effecting
  read. If a read path does invoke it, gate the migration behind an
  explicit flag or move the migration call to an unambiguous write
  command.
- **Botched migration could wipe user content.** Mitigation: the migration
  only mutates byte ranges between detected marker pairs. Any content
  outside *all* marker pairs is preserved by `InsertManagedRegion`'s
  existing logic. Tests must cover: arbitrary user content before the
  install pair; user content between the install end and the snapshot
  start (the common case for a hand-edited file); user content after the
  snapshot end. Diff-the-bytes assertions, not string-equality on
  re-rendered output.
- **Package import direction.** `internal/managed` reusing
  `install.RenderManagedRegion` / `install.InsertManagedRegion` creates a
  `managed → install` dep. The reverse (`install → managed`) is also
  needed so install can build its `ManagedRegion`. To avoid the cycle,
  either:
  (a) move the low-level marker primitives from `internal/install` down
      into `internal/managed`, leaving `install` to depend on `managed`
      only (preferred — cleaner layering);
  (b) keep them in `install` and have `managed` not call them, duplicating
      the marker wrap logic (rejected — duplicates the bug-prone code).
  Pick (a) during implementation; flag here so it's not a surprise.
- **Section ordering regressions.** The order lives in caller code, not in
  the orchestrator. A future contributor inserted in the wrong position
  could subtly change AGENTS.md layout. Mitigate with a snapshot-style
  golden-file test for the rendered AGENTS.md body (already implied by
  `internal/install` test conventions).
- **Idempotency under partial migration.** If the migration runs on a file
  that's *almost* the new layout (e.g. snapshot block already removed but
  install region empty), the orchestrator must converge to the canonical
  layout, not oscillate. Tests should cover at least: (a) fresh write,
  (b) re-run on consolidated layout, (c) re-run on partially-migrated
  layout.
- **Interaction with `cli-invocation-drift-test-markdown`.** That spec
  intends to scan markdown surfaces (including rendered AGENTS.md) for
  drift. The consolidation reshuffles section headings inside the
  region; coordinate landing order so the drift test isn't reading stale
  layout fixtures. The invariant test ("exactly one marker pair per
  file") is the right shared assertion to land in whichever spec ships
  first.

## Validation

**Automated**

- `go test ./internal/managed/...` covers orchestration, ordering, empty
  skipping, idempotence, migration from two-block layout, and the
  exactly-one-marker-pair invariant.
- `go test ./internal/install/...` and `go test ./internal/snapshot/...`
  pass with updated expectations.
- Add a fixture-based regression test: a checked-in `testdata/` AGENTS.md
  in the legacy two-block layout, with deliberately weird whitespace and
  user content outside both pairs. Assert that running the orchestrator
  produces a byte-perfect expected output (also in `testdata/`) and that
  re-running it is a no-op.

**Manual smoke**

- On a fresh project: `hero init` creates AGENTS.md with one marker pair
  carrying the install body + snapshot pointer subsections.
- On a project upgraded from v0.10.0 (simulate by hand-crafting a
  two-block AGENTS.md): `hero install` rewrites it to the single-block
  layout; `git diff` shows only the expected marker-pair consolidation,
  no churn outside markers.
- `hero snapshot` against the upgraded project leaves AGENTS.md unchanged
  on the second run (idempotence).

**Operational**

- No new external dependencies; no new on-disk artefacts; no new commands.
- Release notes should call out the AGENTS.md / CLAUDE.md / NEXT.md
  layout change so users on v0.10.0 know to expect a one-time rewrite of
  those files on next `hero install`.
