---
title: "Add .rgignore to hero-code Repo"
slug: hihcp-rgignore
type: bug
status: planning
domain: engineering
size: trivial
priority: medium
created: 2026-06-09
tags: [hero-code, rg, performance, dx, p2]
parent: hero-in-hero-code-parity
---

# Add .rgignore to hero-code Repo

## Issue

`rg` (ripgrep) searches in hero-code scan heavyweight generated directories
(`.build/`, `build/`, `DerivedData/`) producing 10-second timeouts. The model's
code search is effectively broken for the majority of sessions. No `.rgignore`
file exists in the repo.

Parent initiative: `hero-in-hero-code-parity`.

## Scope -- design inputs for `/design`

Create a `.rgignore` file at the hero-code repo root with exclusions for:
- `.build/`
- `build/`
- `DerivedData/`
- `.hero/cache/`
- `.hero/sessions/`
- Any other heavyweight generated directories identified during implementation

Single-file change. No code modifications required.

**Files to touch:**
- `.rgignore` (new file at repo root)

## Boundaries

- Do not modify `.gitignore` -- this is specifically for `rg` search performance
- Do not exclude source directories or test fixtures

## Risks

- None significant. Worst case: an exclusion is too broad and hides a file the
  model needs to search. The fix is removing the line.

## Validation

- `rg` searches complete within 2 seconds (down from 10+)
- No rg timeout errors in model sessions
- Source files and test fixtures remain searchable
