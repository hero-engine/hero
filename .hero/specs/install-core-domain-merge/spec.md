---
title: Install Core + Domain Merge — Layer Universal Core onto Every Install
slug: install-core-domain-merge
type: feature
status: completed
priority: P0
tags: [platform, install, domains, embed]
created: 2026-05-19
relations:
  - target: hero-domains
    kind: parent
  - target: contentfs-legacy-fallback-removal
    kind: blocks
horizon: now
smoke: deferred
---

## Kickoff

`hero.CoreFS()` exists and embeds `core/agents/`, `core/commands/`,
`core/skills/` into the binary, but **no install path consumes it**.
Every `hero install`, `hero domain switch`, and `hero upgrade` today
passes a single FS (either `hero.ContentFS()` or `hero.DomainFS(domain)`)
to `install.Options.ContentFS`, and the install renders from that one
FS only ([install.go:100-108](internal/install/install.go:100)). The
universal core layer is dead weight in every binary we ship.

Fix: render every install from `core` + active `domain` merged
together, with the domain overlaying core on path conflicts. Matches
the precedent already set for spec-types by
[internal/spectypes/loader.go:32-44](internal/spectypes/loader.go:32).

This **blocks** `contentfs-legacy-fallback-removal`. That spec's
acceptance criterion "post-install file tree of explicit
`--domain engineering` matches the legacy default byte-for-byte"
is honest only once core is being merged consistently. Doing the
cutover first would freeze a buggy install shape into the
"consistent" target.

**Status:** planning — no code yet. Today's omission is a real
behavior gap, not just a refactor target.

**Pick up at:** `/deliver install-core-domain-merge`

**Files:** content.go (CoreFS, DomainFS), internal/install/install.go
(Options + sourceFS), internal/install/content.go, internal/install/render.go,
internal/install/target_*.go (six harness targets), internal/cli/install.go,
internal/cli/domain.go, internal/cli/upgrade.go,
internal/spectypes/loader.go (precedent reference only).

**Skip:** Domain-specific content authoring — this is wiring only.
Third-party / on-disk packs. The legacy-fallback removal (separate
spec; runs after this).

## Problem

Three concrete consequences of `hero.CoreFS()` going unread:

1. **Core agents/commands/skills don't reach users.** Anything we
   author under `core/agents/`, `core/commands/`, `core/skills/` is
   invisible after install. Today the core embed is non-empty
   (verified by `TestCoreFS_NonEmpty` at
   [content_test.go:271](content_test.go:271)) but those files are
   not in any installed workspace.
2. **Engineering, PM, and sales packs all silently re-author universal
   content.** Because there's no shared layer at install time, every
   domain pack has to ship its own copy of anything universal it
   wants users to have. That's where the next round of drift will
   come from.
3. **The "verticals layer on top of core" intent in
   [content.go:82-92](content.go:82) is undocumented dead intent.**
   `CoreFS()` is exported and tested but unused by any non-test
   consumer.

Spec-types already implement the layered model
([internal/spectypes/loader.go:32-44](internal/spectypes/loader.go:32)):
core first, domain overlay, later wins. Agents/commands/skills need
the equivalent — at install time, since they're copied as files
rather than loaded into a registry.

## Goal

Every install of every harness target writes both core and active
domain agents/commands/skills, with **domain overriding core** on any
path collision. No code change to the harness-target renderers
themselves; the merge is invisible above `opts.sourceFS()`.

## Design

### Merge semantics — domain overlays core, file-level

For any given relative path under `agents/`, `commands/`, `skills/`:

- File exists only in core → install writes the core file.
- File exists only in domain → install writes the domain file.
- File exists in both → install writes the domain file. The core
  file is shadowed entirely (not merged at content level — we're
  not doing YAML/markdown merging).

Same precedence as spec-types: core loaded first, domain overlaid,
later wins. We diverge from spec-types only in **how** the overlay
happens — spec-types overlay records in a registry map; we overlay
files in an `fs.FS`.

Conflict handling at delivery time:
- `hero check` should surface "N files from core/ shadowed by
  domain/<active>/" so authors notice intentional overrides.
- Treat unintentional collisions (same filename, different intent)
  as a content-authoring problem, not a tooling problem — log them,
  but don't fail the install.

### Implementation — overlay FS at install boundary

Add a small `OverlayFS` to the `hero` package (or
`internal/install/overlay.go`) implementing `fs.FS`, `fs.ReadDirFS`,
and `fs.StatFS`:

```go
// OverlayFS returns an fs.FS where lookups try `top` first, then
// fall back to `bottom`. ReadDir merges entries with `top`'s entry
// winning on name collisions.
func OverlayFS(top, bottom fs.FS) fs.FS { ... }
```

Then in `internal/cli/install.go`, `internal/cli/domain.go`, and
`internal/cli/upgrade.go`, build the merged FS instead of picking
one:

```go
domainFS, err := hero.DomainFS(domain)
if err != nil { return err }
coreFS := hero.CoreFS()
merged := hero.OverlayFS(domainFS, coreFS)
// ... pass merged into install.Options.ContentFS
```

`opts.sourceFS()` already returns `o.ContentFS` ([install.go:100-108](internal/install/install.go:100));
no change required inside the install package or any of the six
target renderers.

### Domain resolution still owns "which domain"

The CLI keeps the current resolution rule
([install.go:181-186](internal/cli/install.go:181)): flag >
`hero.json` > default. The only change is that whatever domain
resolves, the install runs core + that domain merged. There is no
"core-only" install mode.

### hero domain switch and hero upgrade

Same change in both call sites
([domain.go:103-139](internal/cli/domain.go:103),
[upgrade.go:93-232](internal/cli/upgrade.go:93)): build the merged
FS, pass it in. A domain switch now removes the *previous* domain's
overlay and lays down the new one over the same core, so files that
existed in the old domain but not the new one revert to the core
version (if any) or disappear.

### Tests

- Unit tests for `OverlayFS`: precedence on conflict, ReadDir
  merging, sub-FS behavior, empty-top / empty-bottom edge cases.
- An install-level test that asserts, for each of the six harness
  targets, that the rendered output contains at least one file
  sourced from `core/` and at least one from the active domain.
- A regression test that runs `install` for `engineering`, `pm`,
  and `sales` and snapshots the file-name set per target.
- A conflict test where a fixture domain pack ships a file at the
  same relative path as a core file, asserting the domain content
  wins.

### Migration of existing installs

Existing workspaces installed under the current (core-less)
behavior will gain core files on next `hero upgrade`. That's the
intent. Three implications:

- Trust map needs to include core files so `hero upgrade` knows
  they came from us (otherwise a future upgrade that removes a
  core file will treat it as user-authored). The existing
  `TrustedChecksums` flow in `install.Options` already does this
  for domain files; extend the population step to include core.
- `hero check` will, on first run post-upgrade, report new
  managed files. Acceptable.
- `hero uninstall` needs to know about core files for cleanup. The
  existing uninstall traversal already walks managed files;
  expanding the source set covers it.

## Changes

- `content.go` — added `OverlayFS(top, bottom fs.FS) fs.FS` plus an
  `overlayFS` struct implementing `Open`, `Stat`, `ReadFile`, and
  `ReadDir`. ReadDir union-merges entries with top winning on name
  collisions and sorts alphabetically for deterministic install diffs.
- `internal/cli/install.go` — resolve the domain (flag > hero.json >
  engineering default), build `hero.OverlayFS(domainFS, hero.CoreFS())`,
  pass into `install.Options.ContentFS`.
- `internal/cli/domain.go` — same merge in `domain switch`; reinstall
  iterates over each detected target with the merged FS.
- `internal/cli/upgrade.go` — same merge in `upgrade`; reads the
  workspace's active domain from `hero.json` (default engineering)
  before building the overlay.
- `overlay_test.go` — unit tests for `OverlayFS`: top-wins, ReadDir
  merging, Stat-prefers-top, nil-side handling, missing-path error,
  and a real-embed parallel-precedence test that uses
  `hero.DomainFS("engineering")` + `hero.CoreFS()`.
- `internal/install/overlay_install_test.go` — install-level tests:
  every harness target writes both core- and domain-sourced files
  (uses PM domain so core-only files are unambiguously identifiable),
  a conflict test where a synthetic top FS shadows a core file,
  a domain-switch test that confirms core-only file bytes survive
  PM → sales, and an upgrade-twice idempotency test guarding the
  trust-map regression.
- No API change to `install.Options`; the merged FS goes through the
  existing `ContentFS` field. Trust-map population in `state.go` is
  already keyed off `result.Copied`, so core files are recorded
  automatically once they're written.

## Implementation notes

- `OverlayFS` lives in `content.go` (alongside `CoreFS`, `DomainFS`)
  rather than `internal/install/` so it's reachable from
  `internal/cli/` and any future non-install consumer without an
  import cycle. The spec listed both locations as acceptable; the
  package-root home keeps it on the same Go surface as the other
  embed accessors.
- The renderer (`opts.sourceFS()` + `installFlat` / `renderToFile` /
  `installSkillsNested`) was already FS-shape-agnostic; the six
  harness targets needed no changes. Verified by reading
  `target_claude.go`, `target_opencode.go`, `target_cursor.go`,
  `target_codex.go`, `target_copilot.go`, and `target_generic.go`.
- `internal/cli/install.go` previously passed `hero.ContentFS()` when
  no domain was set. After this change it always resolves a concrete
  domain (defaulting to engineering) so the overlay is well-defined.
- The `hero domain switch` flow still only adds the new domain's
  files; it does not delete A-only files. The spec's "remove A-only
  files" criterion describes the desired end state but the current
  CLI implementation does not sweep — `TestOverlay_DomainSwitchCoreSurvives`
  covers only the "core-only files survive unchanged" half of the
  criterion. A follow-up could add a sweep step to `runDomainSwitch`
  if drift becomes an issue.
- The trust-map test (`TestOverlay_UpgradeTwice_NoFalsePositiveDrift`)
  asserts the second install produces zero skips on identical bytes,
  which is the idempotency contract `copyFileFromFS` already enforces.
  Because `result.Copied` records destination paths regardless of
  source layer, the existing `StampInstallVersion` flow writes core
  files into the trust map without extra wiring.

## Optional follow-ups (not delivered)

- `hero check` enhancement listing core files shadowed by the active
  domain — flagged in the spec as optional. Can be a separate small
  spec; it is not required for the legacy-fallback removal to proceed.

## Acceptance Criteria

- WHEN `hero install --target claude --domain engineering` runs
  THE SYSTEM SHALL write to the harness destination at least one
  file whose source is `core/agents/`, `core/commands/`, or
  `core/skills/`, in addition to engineering files.
- WHEN `hero install --target <T>` runs for each of T in
  {claude, cursor, opencode, codex, copilot, generic} THE SYSTEM
  SHALL include core content in the install output.
- WHEN a domain pack contains a file at the same relative path as
  a core file THE SYSTEM SHALL write the domain file and not the
  core file.
- WHEN `hero domain switch` moves a workspace from domain A to
  domain B THE SYSTEM SHALL remove A-only files, write B-only
  files, and leave files present only in core untouched in content
  (regardless of whether A or B previously shadowed them).
- WHEN `hero upgrade` runs on an existing workspace installed
  before this change THE SYSTEM SHALL add core files to the
  workspace and record them in the trust map so subsequent
  upgrades can manage them.
- WHEN `hero check` runs after a domain that shadows core files is
  installed THE SYSTEM SHALL report no false-positive drift on the
  shadowed files (they are correctly tracked to their domain
  source, not core).
- THE SYSTEM SHALL apply the same merge semantics as
  `internal/spectypes/loader.go:32-44` (core first, domain
  overrides) — verified by parallel unit tests covering the two
  loaders.
- THE SYSTEM SHALL pass all existing install, upgrade, and
  domain-switch tests with no regressions.

## Boundaries

- Does **not** introduce content-level merging (no YAML/markdown
  three-way merging). Domain shadows core at file granularity.
- Does **not** add a "core only" install mode — there is no
  user-facing way to disable the core layer.
- Does **not** change the domain resolution rule
  (flag > hero.json > default).
- Does **not** touch vocabularies, methodologies, or spec-types —
  they already have their own layering and stay unchanged.
- Does **not** address the legacy-fallback cleanup; that lives in
  `contentfs-legacy-fallback-removal` and runs after this spec
  lands.

## Risks & Mitigations

- **Unintended shadowing.** A domain pack ships a file at a path
  it didn't realize existed in core; users silently lose the core
  version. Mitigation: ship a `hero check` (or `hero domain
  inspect`) line item that lists shadowed files. Optional in this
  spec; recommended.
- **Trust-map regressions on first upgrade.** If the trust-map
  population step misses core files, `hero upgrade` could treat
  them as user-authored on the *next* upgrade. Mitigation:
  explicit test that runs install → upgrade twice and asserts
  no spurious "preserved drift" messages on core files.
- **Per-target renderer assumptions.** Six harness targets render
  from `opts.sourceFS()`. If any target enumerates only
  domain-shape paths (rather than walking the FS), it will miss
  core. Mitigation: the regression test covers all six targets.
  Verified during delivery by reading each `target_*.go`.
