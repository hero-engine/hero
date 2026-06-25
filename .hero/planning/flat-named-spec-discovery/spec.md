---
title: "Flat-named spec files are invisible to discovery — verify can't resolve initiative children"
slug: flat-named-spec-discovery
type: bug
status: delivering
priority: high
severity: high
size: medium
domain: engineering
tags: [spec-discovery, slug-resolution, verify, initiative-children, archive]
---

# Flat-named spec files are invisible to discovery

## Symptom

In a downstream repo (hydra), `hero spec verify f-15-disk-resident-acid-buffer-pool`
fails to resolve the spec "even by path", even though the fully-authored,
delivered spec sits on disk. The user could not formally verify/close out a
delivered initiative child and logged it as a Hero closeout blocker.

## Root cause

`spec.Discover` (`internal/spec/spec.go`) recognizes exactly two on-disk spec
shapes:

- single-file: `<dir>/spec.md`
- three-file:  `<dir>/requirements.md`

The single-file branch is hard-gated by basename:

```go
if info.Name() != "spec.md" {
    return nil
}
```

Specs authored as a **flat `<slug>.md` file** are therefore never loaded. This
layout is used in practice for **initiative children stored as sibling files**:

```
.hero/planning/initiatives/make-hydra-fast/
  spec.md                                   # the initiative (discovered)
  f-15-disk-resident-acid-buffer-pool.md    # child (NEVER discovered)
  f-01-…md … f-17-…md                        # 17 children, all invisible
```

It is also used by Hero's **own** workspace (`.hero/planning/cev2-*.md`,
`.hero/planning/portable-routing-rules.md`).

Because the child is never discovered:

1. `hero spec verify <slug>` resolves purely against `spec.Discover`
   (`verify.go`), so the slug matches nothing. There is no path-input code path
   — verify takes `cobra.ExactArgs(1)` and treats the arg only as a slug —
   which is why "even by path" fails.
2. `ResolveOrHint`'s initiative-child detection then **misfires**: it parses the
   parent's children table, sees the slug listed, and emits
   *"…hasn't been designed into its own spec yet — run /design to materialize
   it"* — actively wrong for a spec that is fully authored and delivered. (This
   is the incident the `spec-slug-resolution-fragmentation` knowledge entry
   anticipated, but for an *unmaterialized* child; here the child IS
   materialized, just as a flat file.)

## Second-order bug (archive)

Making the child merely *discoverable* is not enough: once verify resolves it
and gates pass, it archives via `moveToSpecs` (`internal/cli/complete.go`),
which assumes the spec owns its directory:

- `slugDir = filepath.Base(filepath.Dir(path))` → for the flat child this is
  `make-hydra-fast` (the **initiative** slug), so the child would be
  mis-archived to `specs/make-hydra-fast/spec.md`.
- `moveSiblingArtifacts(specDir, …)` would drag the **entire initiative
  directory** (all 17 children + the initiative spec.md) into the destination.

So the fix must cover discovery **and** archive.

## Fix

1. **Discovery** — in `spec.Discover`, also load a flat `<name>.md` file as a
   spec when its frontmatter **explicitly** declares a work-spec `type:`
   (feature, bug, initiative, or any non-knowledge work type). The *explicit*
   requirement is load-bearing: untyped artifacts (audit reports, retros,
   `next/*`, peer-calls) would otherwise default to `feature`/`initiative` via
   `typeFromPath` and be slurped in. Knowledge entries (convention, decision,
   rule, external, context, note, tripwire, explainer) and meta files
   (`mission`, `retro`) are excluded, preserving today's behavior for them.

2. **Archive** — in `moveToSpecs`, detect a flat-file spec (basename is not
   `spec.md`) and handle it with a dedicated path: move only that single file to
   `specs/<spec-slug>/spec.md`, with the slug taken from the parsed spec rather
   than the parent directory name; skip `moveSiblingArtifacts` and
   `removeEmptyParents` (the directory is shared with siblings and not owned by
   this spec).

## Acceptance Criteria

- AC-1: `spec.Discover` loads a flat `<slug>.md` file with explicit
  `type: feature` (or bug/initiative) sibling to an initiative `spec.md`.
- AC-2: `spec.Discover` does NOT load untyped artifacts (delivery-audit.md,
  retro.md, `next/*.md`, peer-calls) or knowledge entries (`type: decision`,
  `type: convention`) that happen to be flat `.md` files.
- AC-3: `ResolveOrHint(<flat-child-slug>, specs)` returns the spec (not the
  "hasn't been designed yet" hint) once the flat child is discovered.
- AC-4: Archiving a flat-file spec moves only that file to
  `specs/<spec-slug>/spec.md` and leaves its initiative-directory siblings in
  place.
- AC-5: `go build ./... && go test ./internal/spec/... ./internal/cli/...`
  passes.

## Completion Ledger

| AC | Status | Note |
|----|--------|------|
| AC-1 | DONE | `Discover` flat-file branch; `TestDiscoverFlatNamedSpec` |
| AC-2 | DONE | explicit-type gate; `TestDiscoverIgnoresNonSpecFlatFiles` |
| AC-3 | DONE | covered by discovery; `ResolveOrHint` unchanged, resolves on exact match |
| AC-4 | DONE | `moveToSpecs` flat-file branch; `TestMoveToSpecsFlatFile` |
| AC-5 | DONE | build + spec/cli tests green |

- [x] exercise-the-feature: discovery + archive covered by new unit tests against synthetic initiative-child layouts.

## Changes

| # | Change | Status |
|---|--------|--------|
| 1 | `internal/spec/spec.go`: flat-file discovery branch + `isDiscoverableFlatSpec` helper | DONE |
| 2 | `internal/cli/complete.go`: `moveToSpecs` flat-file archive branch | DONE |
| 3 | Tests for discovery, exclusion, and flat-file archive | DONE |

## Kickoff

Fix: flat-named spec files (e.g. initiative children stored as
`<slug>.md` siblings of an initiative `spec.md`) are invisible to
`spec.Discover`, so `hero spec verify <slug>` can't resolve them and
`ResolveOrHint` misfires with a "hasn't been designed yet" hint. Two-part fix:
(1) `spec.Discover` loads flat `.md` files that explicitly declare a work-spec
`type:`; (2) `moveToSpecs` archives a flat-file spec to `specs/<slug>/spec.md`
without dragging shared initiative-directory siblings. See root cause and AC in
this spec. Verify with `go test ./internal/spec/... ./internal/cli/...`.
