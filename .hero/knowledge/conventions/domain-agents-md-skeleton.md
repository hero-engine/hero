---
title: Domain AGENTS.md Skeleton — Heading Depth, Section Order, Path Placeholders
type: convention
status: active
created: 2026-07-08
scope:
  - "domains/*/AGENTS.md"
tags: [content-audit, routing, agents-md, domain-packs, install]
---

# Domain AGENTS.md Skeleton — Heading Depth, Section Order, Path Placeholders

## The rule

Every domain pack's `AGENTS.md` (`domains/<domain>/AGENTS.md`) follows one skeleton:

1. **Heading depth: `###`, never `##`.** The pack file has exactly one `#`
   H1 (the pack title). Every body section is `###`. No `##` heading
   appears anywhere below the H1.
2. **Canonical section order** (sections a pack doesn't need are simply
   omitted — this is an ordering rule, not a mandatory-section list):

   intro paragraph → Session Title/Start → Natural Language Routing →
   Key Workflow → Commands Reference → Agents Reference → Skills
   Reference → CLI Commands (opens with the shared disclaimer "These
   are run in the terminal, not as slash commands") → Project Structure
   → Important Rules → handoff-briefing + compaction-survival sections
   (these come last).

3. **Path rule.** Installed-content pointers (commands/agents/skills the
   pack ships) use `<harness>/…` placeholders. Workspace pointers
   (specs, knowledge, config the running instance writes) use `.hero/…`
   paths. Never a repository-relative markdown link or a hero-engine
   source-tree path — those 404 the moment they're installed into a
   user's project, because the pack's own source tree does not exist
   there.

## Why the heading depth matters (renderer rationale)

At install, `splitPackAgentsMd` (`internal/install/agents_md.go`) strips
the pack file's leading `# ` H1 line and returns the rest as the body.
The install orchestrator then re-emits that H1 text as an **H2** section
title inside `<!-- hero:managed-start -->` / `<!-- hero:managed-end -->`.

- A pack body written at `###` depth nests correctly one level under
  that installed H2 — every section stays inside Hero's managed region.
- A pack body written at `##` depth renders as a **sibling** of the
  installed H2, not a child of it — the section visually escapes the
  managed region in the user's file, even though the raw markers still
  wrap it.

This is why the skeleton is `###`, not "whatever reads nicely in the
source repo": the pack file is authored one level shallower than it
will render once installed.

## Why the section order matters

The three domain packs (engineering, pm, sales) evolved independently
and drifted into three different section shapes — divergence that
reads accidental, not domain-motivated. A shared skeleton means an
agent moving between an engineering session and a sales session finds
the same information in the same place (routing table first, then
workflow, then reference rosters, then CLI, then structure, then
rules, then handoff/compaction mechanics last), instead of having to
re-learn each pack's bespoke layout.

## Roster completeness

Every command, agent, and skill a pack installs should be named
somewhere in that pack's `AGENTS.md` — an entry no session can route
to isn't load-bearing, it's dead weight review can't catch. Packs with
a large roster (30+ commands, 30+ agents, 50+ skills — engineering
today) use **compact grouped one-liners** in the Commands/Agents/Skills
Reference sections (e.g. "Scrubbers: comment-scrubber, deadcode-scrubber,
…"), not one row per item — a sales-style per-row table at that scale
would blow the always-loaded token budget. Smaller rosters (sales
today) may use per-row tables if the total stays small.

## Dual-edit note (engineering only)

`domains/engineering/AGENTS.md` and `generateEngineeringAgentsMdBody`
(`internal/install/agents_md.go`) are held byte-identical by
`TestEngineeringPackBodyMatchesGoFallback`
(`internal/install/agents_md_test.go`). Never hand-edit the pack file
alone: change `generateEngineeringAgentsMdBody`, then regenerate with

```
HERO_REGEN_PACK_AGENTS=1 go test -run TestEngineeringPackBodyMatchesGoFallback ./internal/install/
```

pm and sales have no Go fallback and no parity test — their pack files
are edited directly.

## Enforcement

- `TestEngineeringPackBodyMatchesGoFallback` — engineering pack file
  and Go fallback stay byte-identical.
- `TestEngineeringAgentsMdRosterComplete` — every command/agent/skill
  shipped by an engineering install (pack + core) must be named in
  `domains/engineering/AGENTS.md`, or the test fails naming the missing
  entry. pm and sales rosters are manually maintained; the content
  audit is the backstop there.
- `grep -nE '^## ' domains/*/AGENTS.md` should return no matches (H1 is
  `# `; every section below it is `### `).
- `grep -nE '\]\((commands|agents|skills|spec-types|\.\./)' domains/pm/AGENTS.md domains/sales/AGENTS.md`
  should return no matches.

## Exceptions

None today. `domains/chat/AGENTS.md` does not exist (chat is not an
installable domain) so it is out of scope until that changes.
