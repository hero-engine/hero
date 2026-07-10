---
title: /scan post-scan enrichment loop is unbounded
slug: scan-enrichment-unbounded-loop
type: bug
status: completed
completed_at: 2026-06-23
severity: high
created: 2026-05-12
tags: [scan, commands, enrichment, opencode]
---
# /scan post-scan enrichment loop is unbounded

## Problem

The `/scan` slash command's step 4 ("Read each generated entry and enrich it using guidance from the `project-context-generation` skill") has no bound, no stop condition, and no guidance on which entries to skip. On any non-trivial project the host model enters a long sequential rewrite of every generated knowledge entry — including code-intelligence files that are regenerated on every `hero scan` and that the model should not be hand-editing.

Reporter observed this in opencode on a Groovy/Spock + Vitest/Playwright project. After `hero scan` completed and wrote 7+ entries plus an AGENTS.md stub, the model entered the enrichment loop and was last seen reasoning about: "Investigating glob issues / I need to figure out why glob isn't showing up properly, and I'm wondering if maybe the .json path is hidden", "Should we include local config for optional dev workflow?", "Need update AGENTS nonmanaged top maybe user asked enrich". opencode's UI doesn't stream slow Read+Edit progress well, so the wandering loop presented as a hang.

## Expected Behavior

The model should enrich a small, fixed set of high-value stubs (overview, stack, conventions, decisions, project structure — up to ~5 entries), respect that code-intelligence files under `.hero/knowledge/code/` are auto-regenerated and should not be hand-edited, and stop with a brief report of what changed.

## Root Cause

`commands/scan.md` step 4 said, verbatim:

> Read each generated entry and enrich it using guidance from the `project-context-generation` skill

"Each" + no stop condition + no skip rules = the model interprets it as "rewrite all of them, and chase whatever feels suspicious along the way." There's no harness-side bound either — the harness just streams the model's tool calls.

## Fix

Rewrote step 4 in `commands/scan.md` (and the synced copy `core/commands/scan.md`):

- Cap enrichment to at most five entries (highest-value: overview, stack, project-structure, conventions, decisions).
- Explicitly exclude `.hero/knowledge/code/` files — they're regenerated on every scan and should not be hand-edited.
- Explicit stop: "report what you changed and what you intentionally left alone, and end the workflow."
- Explicit anti-tangent: "Do not chase tangents about glob configs, local dev tooling, or AGENTS.md formatting unless the user asks."

## Changes

- `commands/scan.md`: bounded step 4 with cap, skip rules, and stop condition.
- `core/commands/scan.md`: synced to match (root-level `commands/` is what the default install ships; `core/commands/` must stay in sync per the dogfood content-path override).

## Verification

- `commands/scan.md` and `core/commands/scan.md` are byte-identical.
- Fresh `hero install` writes the updated `commands/scan.md` to `.hero/commands/scan.md`.

## Kickoff

Resume work on the `/scan` enrichment-loop fix. Read this spec and `commands/scan.md`. The fix is in place; remaining work is to validate the new step 4 wording in real opencode/codex sessions (does the model actually stop after 5 entries? does it still reach for `.hero/knowledge/code/` files?) and, if needed, tighten the language further. Consider also extending the same bounded-enrichment pattern to other "read N things and enrich each" command surfaces.

## Resolution (2026-06-23)

**Verified already fixed** (status was stale). core/commands/scan.md step 4 caps enrichment to the five highest-value entries, excludes .hero/knowledge/code/, and stops explicitly.
