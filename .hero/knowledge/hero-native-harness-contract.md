---
title: "Hero-Native Harness Contract — Producer/Consumer Asymmetry"
type: convention
tags: [harness, install, architecture, abstraction, a Hero-native client, contract]
created: 2026-05-11
relations:
  - target: single-source-install
    kind: derived-from
  - target: harness-instruction-file-survey
    kind: motivated-by
---

## Summary

Hero produces; harnesses consume. The dependency goes one way. A
Hero-native harness (a Hero-native client, future Hero plugins for VS Code or
JetBrains, hero-cli-tui, etc.) reads canonical paths under `.hero/`
and respects user-authored content. Hero never gains
harness-specific knowledge in return.

## The asymmetry

```
        ┌──────────────┐
        │     Hero     │  ← produces canonical content
        │  (workflow)  │     in .hero/, AGENTS.md, etc.
        └──────┬───────┘
               │  (reads)
   ┌───────────┼───────────┐
   ▼           ▼           ▼
┌──────┐  ┌──────────┐  ┌────────┐
│hero- │  │ hero-cli │  │ future │   ← consumers
│code  │  │   -tui   │  │ plugin │
└──────┘  └──────────┘  └────────┘
```

- **Hero** has no consumer-specific code. There is no `hero install
  --target a Hero-native client` or per-consumer flag. Hero produces a
  conventionally-structured `.hero/` tree and AGENTS.md; that's it.
- **Consumers** have Hero-specific code. They know how to read
  `.hero/agents/`, parse SKILL.md, follow the AGENTS.md priority
  order, etc.

Reversing this asymmetry — making Hero aware of each consumer — would
mean every new harness requires Hero changes to enable. The ecosystem
couldn't grow independently. With the contract going one way, anyone
can write a new Hero-native consumer without touching Hero.

## Consumer rules (what a Hero-native harness must do)

1. **Discover Hero by `.hero/` presence.** No marker file, no
   registration, no install flag. If `.hero/` exists at the
   workspace root, Hero is in use.

2. **Read canonical content from canonical paths:**
   - `.hero/agents/` — agent definitions
   - `.hero/commands/` — command definitions
   - `.hero/skills/` — skill definitions (SKILL.md format)
   - `AGENTS.md` (workspace root) — primary instructions
   - `.hero/specs/`, `.hero/knowledge/`, `.hero/planning/` —
     workflow context (read-only)

3. **AGENTS.md priority order:**
   ```
   1. AGENTS.md            (root — canonical under P1)
   2. .hero/memory.md      (user override)
   3. CLAUDE.md            (user-authored supplementary; read-only)
   4. .hero/AGENTS.md      (legacy)
   5. .hero/CLAUDE.md      (legacy)
   ```

4. **User-authored harness files are supplementary, never primary.**
   A user-authored `CLAUDE.md` exists because the user wanted
   Claude-specific guidance. A Hero-native consumer reads it for
   context, but never replaces AGENTS.md primacy with it and never
   writes to it.

5. **Provide an embedded baseline** so a fresh project without
   `.hero/` still works. a Hero-native client does this via `build.rs`
   compiling Hero content into the binary. Presence of `.hero/`
   overlays project content on top of the baseline.

6. **Store consumer-specific overrides in consumer-owned locations**,
   never in `.hero/`. Settings to disable a skill, override an
   agent, customize a command — those live in the consumer's own
   settings store (`~/.config/<harness>/`, a settings UI, etc.).
   Hero's canonical tree stays canonical.

7. **Never write to canonical content.** If a user wants to
   permanently customize an agent, the harness's UI writes through
   to `.hero/agents/<name>.md` directly. No private mirror, no
   shadow tree.

## Producer rules (what Hero must do)

1. **Maintain the canonical tree** as the single source of truth.
2. **Never add consumer-specific awareness** to Hero core. No
   per-consumer install target, no per-consumer flags in `hero
   install`, no consumer detection logic in Hero CLI.
3. **Respect user content absolutely** (P1 principle): anything
   outside Hero markers in AGENTS.md, any user-authored harness
   file, any user-customized canonical content — inviolate.
4. **Document the contract.** This file. Future Hero versions can
   reshape Hero internals freely, but the contract surface (paths,
   filenames, frontmatter shapes) is stable.

## Why the consumer-side rules exist

- **Rule 1 (discovery via `.hero/`)** removes the need for any
  setup step. A user installs a Hero-native client, opens a project, and it
  Just Works if Hero is in use.
- **Rule 2 (canonical paths)** prevents per-harness content trees
  from re-emerging. The single-source-install initiative's whole
  point: one tree, every consumer reads it.
- **Rule 3 (priority order)** keeps consumer behavior consistent
  across the ecosystem. Two Hero-native harnesses should produce
  the same primary instructions for the same project.
- **Rule 4 (user files supplementary)** preserves the user's right
  to author harness-specific content without that content becoming
  authoritative across all consumers. CLAUDE.md is for Claude;
  AGENTS.md is for everyone.
- **Rule 5 (embedded baseline)** prevents the empty-project
  failure mode. A user opening a Hero-native client in a brand-new repo gets
  agents and skills immediately; if they later install Hero, the
  project content overlays.
- **Rule 6 (own settings location)** prevents consumer preferences
  from polluting `.hero/`. A user disabling a skill in a Hero-native client
  shouldn't affect what the Hero CLI shows or what other consumers
  see.
- **Rule 7 (no shadow trees)** prevents the original problem this
  whole initiative exists to solve: duplicated content that drifts
  over time.

## Implementation status (May 2026)

- **a Hero-native client** already implements most of this:
  - Reads canonical paths (with the `.ai/`/`.agents/` legacy that
    will be deprecated in favor of `.hero/{agents,commands,skills}/`).
  - Has embedded baseline via `build.rs`.
  - Reads AGENTS.md and CLAUDE.md as project memory (read-only).
  - Stores its own settings in a Hero-native client-owned locations.
  - Pending alignment: priority order should prefer **root**
    AGENTS.md over `.hero/memory.md`; user-content discovery should
    move from `.ai/`/`.agents/` to canonical `.hero/` paths.
- **Hero CLI** does not yet implement P1/P2 — it still installs
  per-harness physical copies. The contract above describes the
  target state.

## See also

- `.hero/planning/initiatives/single-source-install/spec.md`
- `.hero/knowledge/harness-instruction-file-survey.md`
- `.hero/planning/features/single-source-install-p1-agents-md/spec.md`
- `.hero/planning/features/single-source-install-p2-canonical-tree/spec.md`
- `../a Hero-native client/src/project_memory.rs` (consumer reference
  implementation)
- `../a Hero-native client/src/hero/mod.rs` (consumer registry pattern)
