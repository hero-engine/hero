---
title: Web Surfaces Restructure — Make Docs and Landing Peers Under web/
slug: web-surfaces-restructure
type: initiative
status: completed
priority: P1
horizon: now
updated: 2026-05-15
---

# Web Surfaces Restructure — Make Docs and Landing Peers Under web/

## Goal

Make every public web surface a peer under `web/` so the `hero-docs` and
`hero-landing` Workers (and future surfaces — blog, cloud dashboard, status
page) share a uniform shape: config + source + build output + deploy artifact,
all self-contained per surface. After this initiative lands, every web surface
is `web/<name>/` containing its own `wrangler.toml`, its own source tree, its
own build output (gitignored if generated, committed if static), and nothing
else. The root no longer hosts surface-specific config.

## Kickoff

Restructures the repo so docs and landing are peer surfaces under `web/`.

**Status:** completed — `/docs`, `/mkdocs.yml`, `/wrangler.toml` moved into
`web/docs/`; mkdocs `--strict` build passes; both wranglers validate by
inspection; CLAUDE.md + docs workflow + cross-refs updated.

**Pick up at:** post-merge, exercise the docs CI path (workflow_dispatch on
`.github/workflows/docs.yml`) once to confirm GitHub Pages publishes from
`web/docs/` without `paths:` filter drift.

→ `gh workflow run docs.yml -f deploy=false`

**Files:** web/docs/mkdocs.yml, web/docs/wrangler.toml, web/landing/wrangler.toml, .github/workflows/docs.yml, .gitignore

## Context

Hero today has two public web surfaces deployed via Cloudflare Workers:

- **`hero-docs`** — mkdocs Material site, sources at `/docs/`, config at
  `/mkdocs.yml`, deploy at `/wrangler.toml`, build output to `/site/`
  (gitignored).
- **`hero-landing`** — static HTML at `web/landing/site/`, config at
  `web/landing/wrangler.toml`, committed to the repo (no build step).

The shapes don't match. `/docs/` and `/mkdocs.yml` live at the repo root by
mkdocs convention; the landing already lives at `web/landing/` because it was
introduced after the docs and bypassed the convention deliberately. The
collision shows up in `.gitignore`: `site/` is globally ignored (for mkdocs
output), then `!web/landing/site/` re-exempts the committed landing tree. The
exception is a smell — it says the layout is fighting itself.

More surfaces are imminent (blog, cloud dashboard, status page). Each one
will multiply the friction. The cost of the move compounds with attention:
right now there is zero external link footprint (Hero just shipped v0.8, no
HN/SEO yet, no `heroengine.ai` live deploy). Doing it before launch keeps
the migration cost near-zero.

## Problem

Two surfaces today, name collision on `site/`, awkward `!web/landing/site/`
gitignore exception, future surfaces (blog, dashboard, status) are imminent.
The convention "mkdocs lives at the root" is real but cheap to override with
`docs_dir: src`. The longer the asymmetry stays, the more files reference
the root layout (workflows, READMEs, agent guides), and the more expensive
the move becomes.

## Approach

Treat `web/<name>/` as the unit. Each surface is self-contained:

```
web/
  docs/
    mkdocs.yml          (with docs_dir: src)
    wrangler.toml       (directory = "./site")
    src/                (all the markdown that was in /docs/)
    site/               (mkdocs build output, gitignored)
  landing/              (unchanged location)
    wrangler.toml
    README.md
    site/               (committed static HTML)
```

Root loses: `/docs/`, `/mkdocs.yml`, `/wrangler.toml`, and the `/site/`
gitignore entry. Workers keep their names; DNS and custom domains are
untouched. Published docs URL paths are preserved (mkdocs `nav` is unchanged,
only the source directory name shifts from `docs/` to `src/`).

### Acceptance criteria (EARS)

- THE SYSTEM SHALL place all public web surfaces under `web/<name>/`.
- THE SYSTEM SHALL keep each surface self-contained: config, sources, build
  output, deploy artifact all under `web/<name>/`.
- WHEN `mkdocs build` runs from `web/docs/` THE SYSTEM SHALL produce a
  complete site at `web/docs/site/` without `--strict` failures.
- WHEN `wrangler` validates either Worker THE SYSTEM SHALL find its config
  and asset directory relative to `web/<name>/`.
- THE SYSTEM SHALL preserve every existing markdown page, asset, stylesheet,
  and internal link in the docs surface.
- IF an internal markdown cross-link or asset path is broken by the move
  THEN THE SYSTEM SHALL repair it in the same commit.
- THE SYSTEM SHALL update `CLAUDE.md` "Project Structure" to reflect the new
  layout.
- THE SYSTEM SHALL update `.github/workflows/docs.yml` to operate against
  the new paths.
- IF the root `/docs`, `/mkdocs.yml`, `/wrangler.toml`, or `/site` gitignore
  entries remain THEN delivery is incomplete.

### Decisions

- Docs source dir is `src/` (not `pages/`, `content/`). Matches the
  generic-web-surface vocabulary; `pages/` reads like Next.js-specific.
- Landing stays at `web/landing/site/` unchanged. Only the docs surface moves.
- mkdocs config gains `docs_dir: src` to override the default.
- Workers keep their names (`hero-docs`, `hero-landing`). No DNS impact.

## Changes

Atomic migration. One commit at the end. Use `git mv` so file history follows.

1. **Move `/docs/` → `/web/docs/src/`** via `git mv` (preserve history for
   every markdown file, asset, stylesheet).
2. **Move `/mkdocs.yml` → `/web/docs/mkdocs.yml`** via `git mv` and add
   `docs_dir: src` to the config. Leave `site_dir` implicit (default `site`
   relative to config file). Check `theme.logo`, `theme.favicon`, and
   `extra_css` — they're already relative to `docs_dir` so no edit needed.
3. **Move `/wrangler.toml` → `/web/docs/wrangler.toml`** via `git mv`.
   `directory = "./site"` stays correct relative to the new location.
4. **Edit `.gitignore`**:
   - Remove `site/`
   - Remove `!web/landing/site/`
   - Add `web/docs/site/`
5. **Edit `.github/workflows/docs.yml`**:
   - `paths:` filter → `web/docs/**`
   - Build step → `cd web/docs && mkdocs build --strict`
   - Deploy step → `cd web/docs && mkdocs gh-deploy --force`
   - `pip install -r requirements-docs.txt` stays at root (the requirements
     file is not a surface artifact).
6. **Edit `CLAUDE.md` "Project Structure"** — add the `web/` tree showing
   docs and landing as peers.
7. **Edit cross-references**:
   - `web/landing/site/index.html` header comment: `docs/stylesheets/brand.css`
     → `web/docs/src/stylesheets/brand.css`
   - `web/landing/README.md`: `docs/` deploy pattern reference → `web/docs/`
   - `.hero/mocks/hero-landing-page/index.html`: `../../../docs/assets/favicon.svg`
     → `../../../web/docs/src/assets/favicon.svg`
   - `.hero/planning/features/hero-landing-page/spec.md`: any `docs/` or
     `docs/assets/` source references → `web/docs/src/` equivalents
   - `docs/project-structure.md` (now `web/docs/src/project-structure.md`):
     if it self-describes the docs path, update it
8. **Verify build**:
   - `cd web/docs && mkdocs build --strict` succeeds
   - `web/docs/site/index.html` exists
   - `wrangler` finds config in each `web/<name>/` (deferred unless wrangler
     CLI is locally available)
9. **Run `hero index --if-stale -q` and `hero queue write -q`** after the
   spec lands so the workspace catalogs the new initiative.

## Boundaries

Out of scope:

- Docs URL structure changes — published paths stay identical.
- Domain or DNS changes.
- Worker renames.
- Production deploy. This is a structural move; deploys flow through the
  existing `vars.DEPLOY_DOCS` / `vars.DEPLOY_LANDING` gates after merge.
- Touching `/web/landing/` source. Only the docs surface moves.
- Refactoring unrelated files.

## Risks

- **Broken internal links**: mkdocs `--strict` catches cross-link breakage.
  All internal links are markdown-relative within the `src/` tree, so they
  move with the files.
- **Asset path drift**: `theme.logo: assets/logo.svg` is relative to
  `docs_dir` so it follows the move. Stylesheet links in `extra_css` are
  also relative to `docs_dir`.
- **CI path filter miss**: the `paths:` trigger on `.github/workflows/docs.yml`
  needs to update with the move, otherwise docs CI won't fire on docs
  changes. Step 5 handles this.
- **Stale spec references**: landing spec mentions the docs path; we update
  in the same commit so they don't drift.
- **History loss**: mitigated by `git mv` semantics where possible.

## Validation

- `cd web/docs && mkdocs build --strict` exits 0.
- `web/docs/site/index.html` exists and renders a working nav.
- `.gitignore` no longer contains `site/` or `!web/landing/site/`; instead
  contains `web/docs/site/`.
- `grep -r "/docs/\|docs/assets\|docs_dir" --include="*.md" --include="*.yml"
  --include="*.html"` from the repo root shows no stale source references
  (external URLs containing `/docs/` are fine).
- `.github/workflows/docs.yml` paths filter is `web/docs/**`.
- `CLAUDE.md` Project Structure section shows `web/docs/` and `web/landing/`
  as peers.
- Both `wrangler.toml` files parse and point at `./site` from their own
  directory.
