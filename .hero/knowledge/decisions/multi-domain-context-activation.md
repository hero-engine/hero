---
title: Multiple Domains Resolve by Context, Not by Union — hero-code Picks the Active Set
type: decision
status: superseded
superseded_by: hybrid-content-packs-and-workflow-providers
created: 2026-07-05
tags: [domains, hero-code, agents, skills, roster, context, architecture, decision]
relations:
  - target: domain-routing-and-agents
    kind: builds-on
  - target: hero-domains
    kind: related
  - target: content-dedup-resync
    kind: related
  - target: hero-content-audit
    kind: related
---

# Multiple Domains Resolve by Context, Not by Union — hero-code Picks the Active Set

> Superseded on 2026-08-22 by
> `hybrid-content-packs-and-workflow-providers`. The useful invariant
> remains: Hero does not union every available domain roster. The newer
> decision permits one primary domain plus explicitly enabled, bounded
> capability extensions such as PM and QA.

## Decision

When a product embeds multiple Hero domains (hero-code ships engineering,
chat, and pm), the model never sees a union roster. **At any moment the
loaded content set is exactly one domain + core ("domain + common") — the
engine's existing `OverlayFS(domain, core)` contract, unchanged.** What
becomes multi-domain is *activation*: **hero-code is responsible for
choosing the right domain per context** (which space/window/surface the
user is in, what's active) and loading that domain's agents, skills, and
commands. Domain selection moves from a process-wide constant to a
context-resolved value owned by the embedding client.

The Go engine stays as-is: single active domain per resolution,
`OverlayFS(one domain, core)`, `hero domain switch` for CLI workspaces.
No hero.json `domains: []` list, no N-layer overlay, no union install.

## Context

hero-code embeds hero content per-domain at build time
(`crates/hero-core/build.rs` stages engineering/chat/pm separately;
`embedded.rs::load_all_for(domain)`) and currently resolves ONE active
domain process-wide (`domains.rs`, `OnceLock` — the code itself flags
per-window state as the future need). The model-facing roster is
system-prompt Layer 1b (`hero_surface.rs`): commands + agents with
one-line descriptions, skills as names.

The original plan was to split domains into separate app experiences;
that was reversed in favor of one UI hosting multiple domains. The open
question was how rosters work then. Three families were considered:

1. **Union at once** — all enabled domains' content loaded together.
   Rejected: same-named commands collide *by design* (pm and chat each
   ship their own `/discover`; chat rewrites `/capture`, `/note`, `/why`),
   the roster balloons to ~58 agents / 94 skills / 72 commands (which is
   what forced the "lighter-weight roster" discussion), and priority-order
   resolution is silent and surprising.
2. **Namespaced union** (`/pm:discover`) — unambiguous but uglifies the
   dominant single-domain case and breaks muscle memory.
3. **Context-driven activation** (chosen) — the client knows what the user
   is doing (chat space vs code project vs PM space) and loads the one
   domain that fits, plus core. The union never materializes, so the
   collision and roster-weight problems dissolve rather than get solved.

The "lighter-weight roster" concern mostly evaporates with this decision:
per-context rosters are today's size, and hero-code already reverted a
names-only Layer 1b experiment because descriptions at decision time
measurably improve agent selection. A one-line availability hint for
*inactive* domains ("PM domain available") is the only lightweight
addition worth considering, so the model knows switching exists.

## Consequences

- **hero-code owns context→domain resolution.** Moving `ACTIVE` from
  process-wide `OnceLock` to per-context (per-window/space) state, the
  resolution rules (space type, project type, explicit user switch), and
  Layer 1b reassembly on context change are hero-code design work — to be
  specced natively in hero-code (its `domain-availability-model` planning
  item is the natural home).
- **Cross-domain needs are served by data, not rosters.** A PM-context
  session asking about engineering work goes through the shared knowledge
  graph and MCP tools (`hero_why`, `hero_search`, cross-domain graph
  query) or peer handoffs — not by loading the engineering pack. Content
  loads per-domain; knowledge is cross-domain.
- **The engine's single-domain contract is load-bearing.** Same-named
  files across domain packs remain legitimate per-domain specializations
  (pm `/discover` vs chat `/discover`) — they are alternatives selected by
  activation, never merged. The `content-dedup-resync` parity test's
  `core_fork:` annotation documents them; no cross-domain collision rules
  are needed.
- **Agents' `domains:` frontmatter stays a single-active-domain filter**
  (its only engine consumer). No new membership semantics required.
- **Mid-session domain switches** (user moves a conversation from a chat
  space to a code project) need defined semantics in hero-code — likely
  "new context = fresh Layer 1b at next assembly," not live mutation of a
  running conversation. Flagged for the hero-code spec, not decided here.
- **Reversibility:** if a real use case ever demands two domains truly
  simultaneous in one context, the namespaced-union option can be layered
  on later without unwinding this — activation-by-context remains the
  default and the union becomes the scoped exception.

## Status

Proposed — agreed in session 2026-07-05 (bdwheeler). hero-code-side design
spec to be requested via peer spec-out; engine-side: no work.
