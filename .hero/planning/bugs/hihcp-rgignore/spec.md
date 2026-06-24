---
title: "Add .rgignore to hero-code Repo"
slug: hihcp-rgignore
type: bug
status: handed_off
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

## Handoff Trail

- 2026-06-24T18:01:15Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: hihcp-rgignore
  peer_spec: hero-code/hihcp-rgignore
  at_commit: 2f774b7
  reason: "Title literally scopes to the hero-code repo (.build/DerivedData rg timeouts). The hero CLI is Go with no such artifacts."

