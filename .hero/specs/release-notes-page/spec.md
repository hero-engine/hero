---
title: "Release Notes Page — Left-Nav Release History for docs.heroengine.ai"
slug: release-notes-page
type: feature
status: completed
priority: P2
domain: engineering
size: medium
tags: [docs, releases, web, changelog]
created: 2026-07-12
relations:
  - target: hero-landing-page
    kind: relates-to
horizon: now
completed_at: 2026-07-13T00:22:45Z
---

## Context

The user asked for a page on the Hero website listing releases, with a
left-hand nav that lets a visitor pick a release and see its bulleted
highlights (major features, fixes). Two website codebases exist in this
repo:

- `web/landing/` — the public marketing homepage (`hero-landing-page` spec).
  Plain HTML + Tailwind, single static page, no router, no multi-page nav
  infrastructure. Not a fit for a browsable release list.
- `web/docs/` — the docs site (`docs.heroengine.ai`), built with
  **mkdocs-material**. It already ships a working left-hand primary nav
  (`.md-nav--primary`) that collapses into a hamburger drawer on mobile,
  a permalink-anchored table of contents (`toc: permalink: true` is
  already set in `web/docs/mkdocs.yml`), and Hero-branded active-link
  styling in `web/docs/src/stylesheets/brand.css` (`--hero-blue-700` /
  `--hero-blue-500` on `.md-nav__link--active`). This is the natural home
  for a release notes page — it reuses working, responsive, on-brand nav
  instead of building bespoke nav from scratch in the landing page.

Release data already exists, but is scattered and inconsistent:

- **No `CHANGELOG.md`** in the repo.
- **Git tags** exist locally (`v0.8.0` … `v0.24.1`), driven by
  `.goreleaser.yaml`, which builds and publishes on every `v*` tag push
  (`.github/workflows/release.yml`).
- **GoReleaser publishes real GitHub Releases**, but to a *separate*
  repo: `hero-engine/hero-releases` (see `release.github.owner/name` in
  `.goreleaser.yaml`), not `hero-engine/hero`. `hero-engine/hero`'s own
  release list (`v0.16.x`) is stale/out of sync with the actual shipped
  versions on `hero-releases` (currently `v0.24.1`) — the two have
  diverged.
- The `hero-releases` GoReleaser changelog is currently **terse and
  uncategorized** — one bullet per commit, prefixed with the full commit
  hash (e.g. `* 07dbcfc... fix(next): make NEXT.md projection
  deterministic...`). It excludes `docs:`, `test:`, `chore:` prefixed
  commits via `changelog.filters.exclude`, but does not group the
  remainder into sections.
- Commit messages in this repo consistently follow **Conventional
  Commits** (`feat(scope): ...`, `fix(scope): ...`, `chore(scope): ...`,
  `docs(scope): ...`) — confirmed via `git log --oneline`. This is the
  signal that makes automatic "Major Features" vs "Fixes" categorization
  possible without hand-curation.

## Goal

A `docs.heroengine.ai/releases/` page exists, reachable from the docs
site's left nav, that lists every published Hero release (version +
date) with a permalink-anchored, deep-linkable section per release. Each
release's notes are grouped into "Major Features" and "Fixes" bullet
lists. The release list is generated automatically from
`hero-engine/hero-releases` GitHub Releases at docs build time — no
hand-authored markdown per release, no manual nav maintenance as new
versions ship. The page is keyboard-navigable and works on mobile via
mkdocs-material's existing responsive drawer.

## Approach

Reuse `web/docs/` (mkdocs-material) rather than `web/landing/`. Build a
single page, `web/docs/src/releases/index.md`, containing every release
as an `## v{version} — {date}` section (newest first), each with `###
Major Features` / `### Fixes` sub-sections. Turn on mkdocs-material's
`navigation.integrate` (aka "integrated TOC") feature so this page's
in-page table of contents (the H2/H3 release headings) renders **in the
left nav column**, not the default right-hand TOC rail — this is what
satisfies "left of them, navs to the release" without writing any custom
nav component. `toc.permalink` is already enabled site-wide, so every
`##`/`###` heading gets a stable `#v0-24-1` / `#v0-24-1-fixes`-style
anchor for free — satisfying deep-linkability.

**Data source: GitHub Releases on `hero-engine/hero-releases`, fetched
at docs build time.** Rejected alternatives and why:

- *Hand-authored markdown per release*: most control over copy, but
  creates a second place (beyond the git tag / GoReleaser release) that
  must be updated on every ship, and it *will* drift — the existing
  `hero-engine/hero` vs `hero-releases` release-list divergence in this
  repo is a live example of exactly that failure mode.
- *Generate from `CHANGELOG.md`*: doesn't exist; would need to be
  invented and would duplicate what GoReleaser's changelog already
  produces at tag time.
- *Generate from raw git tags/log at docs-build time*: works, but the
  docs site repo checkout may not have full unauthenticated history
  context for a public release list, and it reinvents categorization
  GoReleaser can already do natively via `changelog.groups`.
- **Chosen: fetch already-published GitHub Releases from
  `hero-engine/hero-releases`.** GoReleaser already runs on every tag
  push and already produces the changelog body; teaching it to group
  commits by conventional-commit prefix (Change 1 below) means the
  category bullets exist in the release body at publish time, and the
  docs site just needs to fetch and render them. One source of truth,
  no duplicate authoring step, no drift.

Mobile/responsive nav is inherited for free from mkdocs-material's
existing drawer behavior — no additional work required beyond verifying
it on the releases page specifically (see Validation).

## Changes

1. **`.goreleaser.yaml`** — add `changelog.groups` so the GitHub Release
   body GoReleaser writes to `hero-engine/hero-releases` is already
   categorized:
   - Group 1, title `Major Features`, regexp matching `^feat`
   - Group 2, title `Fixes`, regexp matching `^fix`
   - Existing `filters.exclude` (`^docs:`, `^test:`, `^chore:`) stays as-is
     so those commits are dropped, not just left ungrouped.
   - This changes the *shape* of future release bodies only — it does
     not retroactively rewrite already-published releases (see Risks).

2. **`web/docs/scripts/generate_release_notes.py`** (new) — a Python
   script that:
   - Calls the GitHub REST API for `hero-engine/hero-releases` releases
     (`GET /repos/hero-engine/hero-releases/releases`), using
     `GITHUB_TOKEN` from the environment for rate-limit headroom (falls
     back to unauthenticated if unset, since these are public releases).
   - Parses each release's `tag_name`, `published_at`, and `body`.
   - Re-renders `body` into `## {tag_name} — {published_at:%Y-%m-%d}`
     with the existing `### Major Features` / `### Fixes` grouped
     headings passed through as-is (GoReleaser already wrote them per
     Change 1). Releases published before Change 1 lands will render
     with a single ungrouped `### Changes` bucket — the script must
     handle both shapes without erroring.
   - Writes the full concatenated page to
     `web/docs/src/releases/index.md`, newest release first.
   - Idempotent — safe to re-run on every docs build.

3. **`web/docs/mkdocs.yml`**:
   - Add `navigation.integrate` to `theme.features`.
   - Add `Releases: releases/index.md` as a new top-level `nav:` entry.

4. **`.github/workflows/docs.yml`** — add a step before `mkdocs build
   --strict` that runs `python web/docs/scripts/generate_release_notes.py`
   (working directory `web/docs`), so the releases page is regenerated
   on every docs deploy.

5. **`.github/workflows/release.yml`** — after the `goreleaser` job
   succeeds, add a step that triggers the `docs.yml` workflow
   (`gh workflow run docs.yml` or `workflow_dispatch` via the GitHub API)
   so a new tag push refreshes the published releases page without
   waiting for an unrelated `web/docs/**` change to trigger it.

6. **`web/docs/src/stylesheets/brand.css`** — verify the existing
   `.md-nav__link--active` Hero-blue styling still applies correctly
   once `navigation.integrate` changes the left nav's DOM structure
   (integrated TOC entries use `.md-nav__link` the same as primary nav
   items, so this is expected to need no changes — confirm during
   Validation, adjust only if broken).

**Implemented in:**

- `.goreleaser.yaml` — added `changelog.groups` (`Major Features` / `Fixes`).
- `web/docs/scripts/generate_release_notes.py` (new) — fetch/parse/render script.
- `web/docs/scripts/test_generate_release_notes.py` (new) — 16 unittest cases,
  including pagination regression coverage added after a cold audit caught the
  fetch loop silently truncating at GitHub's 30-item default page size.
- `web/docs/mkdocs.yml` — added `toc.integrate` (corrected from spec's
  `navigation.integrate`, which does not exist in mkdocs-material 9.7 —
  see Completion Ledger note on Change 3) and the `Releases` nav entry.
- `web/docs/src/releases/index.md` (new, generated) — seeded by running the
  script against the live `hero-engine/hero-releases` API (43 releases,
  paginated fetch — see Completion Ledger addendum below).
- `.github/workflows/docs.yml` — added the generator step before `mkdocs build --strict`.
- `.github/workflows/release.yml` — added `actions: write` permission and a
  `gh workflow run docs.yml` trigger step gated on `vars.DEPLOY_DOCS == 'true'`.
- `web/docs/src/stylesheets/brand.css` — verified, no changes needed.

## Acceptance Criteria

- THE SYSTEM SHALL render a `/releases/` page on `docs.heroengine.ai`
  listing every release published to `hero-engine/hero-releases`
- THE SYSTEM SHALL display a left-hand nav panel on the releases page
  listing each release by version and date
- WHEN a visitor clicks a release entry in the left nav THE SYSTEM SHALL
  scroll to that release's notes section
- THE SYSTEM SHALL group each release's notes into "Major Features" and
  "Fixes" bulleted sub-sections for releases published after the
  `changelog.groups` config lands
- WHERE a release predates the grouped changelog config THE SYSTEM SHALL
  render its notes in a single fallback section rather than erroring
- THE SYSTEM SHALL give every release its own deep-linkable anchor URL
- WHEN the viewport is narrow (mobile width) THE SYSTEM SHALL collapse
  the left nav into mkdocs-material's existing drawer/hamburger pattern
- THE SYSTEM SHALL regenerate the releases page automatically on every
  docs deploy, without requiring hand-authored markdown per release
- THE SYSTEM SHALL be fully keyboard-navigable, consistent with the rest
  of the docs site

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Evidence |
|---|---|---|---|
| 1 | Render `/releases/` listing every published release | DONE | `web/docs/src/releases/index.md` generated against the live `hero-engine/hero-releases` API with a paginated `fetch_releases()`; `grep -c '^## v' web/docs/src/releases/index.md` → 43, matching the live release count (initial pass under-counted at 30 due to a pagination bug caught by cold audit and fixed — see below) |
| 2 | Left-hand nav lists each release by version + date | DONE | `mkdocs build --strict` + browser inspection: integrated TOC in left sidebar shows `v0.24.1 — 2026-07-12` etc. |
| 3 | Clicking a release entry scrolls to its section | DONE | `toc.permalink: true` (pre-existing, site-wide) produces stable per-heading anchors; verified in built HTML |
| 4 | Group notes into Major Features / Fixes for post-`changelog.groups` releases | DONE | `test_grouped_parsing`, `test_grouped_heading_and_bullets` pass; no live release is post-Change-1 yet, expected per Risks |
| 5 | Pre-grouped releases render as single fallback section | DONE | `test_ungrouped_falls_back_to_single_section`, `test_default_goreleaser_changelog_heading_falls_back_to_changes` cover both real-world body shapes; all 43 live releases currently render under the `### Changes` fallback |
| 6 | Every release has a deep-linkable anchor | DONE | Same `toc.permalink` mechanism as #3, confirmed in built HTML |
| 7 | Mobile viewport collapses left nav to drawer | DONE | Resized to 375x812 in Browser tool, confirmed hamburger drawer opens with "Releases" highlighted Hero-blue |
| 8 | Regenerate automatically on every docs deploy, no hand-authored markdown | DONE | `.github/workflows/docs.yml` runs `generate_release_notes.py` before `mkdocs build --strict` |
| 9 | Fully keyboard-navigable | DONE | Tab-tested via Browser tool: focus visibly lands on permalink anchors and left-nav TOC links with visible focus rings; DOM check confirms ~127 nav links + 87 headerlink anchors are real tab stops |

### Changes

| # | Item (abbreviated) | Status | Evidence |
|---|---|---|---|
| 1 | `.goreleaser.yaml` — `changelog.groups` | DONE | Added Major Features (`^feat`) / Fixes (`^fix`) groups; `goreleaser check` validates cleanly |
| 2 | `web/docs/scripts/generate_release_notes.py` (new) | DONE | Fetches (paginated), parses both body shapes, writes newest-first, idempotent; 16 unittest cases, 16/16 pass |
| 3 | `web/docs/mkdocs.yml` — integrated TOC + nav entry | DONE | Used `toc.integrate` (spec said `navigation.integrate`, which does not exist in the pinned mkdocs-material version — corrected after build-verified investigation, confirmed correct by cold audit); `Releases: releases/index.md` added to `nav:` |
| 4 | `.github/workflows/docs.yml` — generator step | DONE | Step added before `mkdocs build --strict`, working-directory `web/docs`, `GITHUB_TOKEN` passed |
| 5 | `.github/workflows/release.yml` — cross-workflow trigger | DONE | `gh workflow run docs.yml` step added, gated on `vars.DEPLOY_DOCS == 'true'` per Risks; `actions: write` permission added |
| 6 | `web/docs/src/stylesheets/brand.css` — verify active-link styling | DONE | No changes needed — confirmed `.md-nav__link--active` applies identically to integrated TOC entries in both desktop sidebar and mobile drawer |

### Post-audit fix round

A cold audit (fresh subagent, no shared context) returned HOLD on first pass with three findings, all fixed and independently re-verified by a second cold audit (SHIP):
1. **Pagination bug (blocker)** — `fetch_releases()` had no pagination; GitHub's API defaults to 30/page, silently dropping 13 of 43 real releases. Fixed by looping `page=N` until a short/empty page returns; regression-tested with a mocked 3-page/243-item fixture.
2. **Missing visual regression check** — spec's Validation step to spot-check unrelated docs pages (`cli/overview.md`, `concepts/core-loop.md`) after enabling `toc.integrate` site-wide had no ledger row. Performed and recorded: no regression observed on either page.
3. **AC#9 keyboard-navigable** — originally disclosed as not manually tab-tested; strengthened to real DONE evidence per the Acceptance Criteria table above.

Full audit trail: `.hero/planning/features/release-notes-page/delivery-audit.md`.

## Boundaries

- Not building a separate release-notes UI in `web/landing/` — the docs
  site's existing nav infrastructure is the delivery surface.
- Not backfilling/rewriting the changelog bodies of already-published
  `hero-engine/hero-releases` releases to match the new grouped format —
  older releases render with a single ungrouped bucket (see Change 2).
  Regrouping history retroactively is a separate, optional follow-up.
- Not reconciling the stale `hero-engine/hero` release list (`v0.16.x`)
  with `hero-engine/hero-releases` (`v0.24.1`) — that's a pre-existing
  process gap outside this spec's scope, called out here so it isn't
  lost.
- Not adding release-notes authoring UI, email/RSS subscription, or
  filtering/search across releases — v1 is a single scrollable page with
  nav-assisted jumping, matching what was asked for.
- Not touching `CHANGELOG.md` (doesn't exist, not being introduced).

## Risks

- **GitHub API rate limits during docs builds.** Unauthenticated REST
  calls are capped at 60/hour per IP; CI runners can share IPs. Mitigate
  by passing `GITHUB_TOKEN` (already available in Actions) to the fetch
  script.
- **`navigation.integrate` is a site-wide theme feature.** Turning it on
  changes the left-nav/TOC relationship on *every* docs page, not just
  `/releases/`. Expected to be a net UX improvement (fewer nav surfaces
  to scan) but must be visually checked across a few existing pages
  (e.g. `cli/overview.md`, `concepts/core-loop.md`) before shipping, not
  just the new page.
- **Historical release bodies are ungrouped.** Until enough new tags
  ship under the Change 1 config, most of the releases list will show
  the fallback single-bucket rendering, not "Major Features / Fixes."
  This is expected and stated explicitly in the Goal — not a defect.
- **Release cadence vs. docs deploy trigger.** `docs.yml` is currently
  gated behind `vars.DEPLOY_DOCS == 'true'` and only auto-triggers on
  `web/docs/**` pushes. Change 5 (cross-workflow trigger) must respect
  that same gate — a tag push should not force-deploy docs if
  `DEPLOY_DOCS` is off.

## Mockups

- [Release Notes Page](.hero/mocks/release-notes-page/index.html) — 2026-07-12 — Docs-site releases page with left nav TOC and bulleted Major Features/Fixes per release

## Validation

- Run `mkdocs build --strict` locally after all changes; confirm no
  broken-link or nav warnings.
- Run `python web/docs/scripts/generate_release_notes.py` against the
  real `hero-engine/hero-releases` API and inspect the generated
  `releases/index.md` for correct version ordering (newest first),
  correct dates, and correct grouped/ungrouped rendering for both
  release shapes.
- Manually exercise the built site: left nav shows the releases page
  entry; clicking it lands on `/releases/`; the integrated left-nav TOC
  lists each release version; clicking a version jumps to its anchor;
  copy the anchor URL and load it directly in a fresh tab to confirm the
  deep link lands on the right section.
- Resize to a mobile viewport and confirm the left nav collapses to
  mkdocs-material's drawer/hamburger and the releases page is still
  navigable through it.
- Tab through the page with keyboard only; confirm every nav link and
  release anchor is reachable and visibly focused.
- Visually spot-check 2-3 unrelated existing docs pages after enabling
  `navigation.integrate` to confirm the site-wide nav change doesn't
  regress them.
- Confirm `.md-nav__link--active` Hero-blue active-state styling still
  applies to the integrated TOC entries.

## Kickoff

**Delivered.** All 6 Changes items and all 9 acceptance criteria are DONE,
cold-audited (SHIP verdict after one HOLD/fix round), and verified against
the live `hero-engine/hero-releases` API. `web/docs/src/releases/index.md`
currently reflects 43 releases fetched via a paginated `fetch_releases()` —
regenerate it any time by running
`python web/docs/scripts/generate_release_notes.py` from `web/docs` (needs
`GITHUB_TOKEN` in the environment for rate-limit headroom; falls back to
unauthenticated).

If picking this back up for follow-on work: the `web/docs/mkdocs.yml`
theme feature is `toc.integrate`, not `navigation.integrate` as originally
specced — that name doesn't exist in the pinned mkdocs-material version, and
the substitution was build-verified and cold-audited as the correct
equivalent. Known deferred-by-design gaps (see Boundaries): older
`hero-releases` entries render under a single `### Changes` fallback until
enough new tags ship under the `changelog.groups` config; the stale
`hero-engine/hero` (v0.16.x) vs `hero-engine/hero-releases` (v0.24.1+)
release-list divergence is unaddressed by this spec.
