---
title: Hero Charter — Core
type: mission
scope: core
locked_at: 2026-04-28
locked_by: chet-bellows
version: 2
source: |
  Synthesized from buddy-model-architecture, hero-v2-system-design,
  graph-memory, hero-serve-daemon, context-injection, AGENTS.md, and
  the 2026-04-28 recovery-strategy conversation. Locked language drawn
  verbatim from the user where possible.
---

## Mission

**Hero is the sidekick brain for AI-augmented knowledge work.**

Whenever a stateless AI agent is in the loop on real work —
engineering, sales, research, support, design, anything — Hero
captures everything that happens during that work, structures it, and
injects it back automatically so every session starts as smart as
where the last one left off, the team grows together with every turn,
and the floor rises for everyone.

The user works in their **harness** — Claude Code, Cursor, Codex,
opencode, Aider, plain humans, CI bots. Hero rides shotgun, capturing
*everything you'd want to insert into a new person to make them
understand everything that has gone on*: knowledge, decisions,
changes, trials, failures, fixes. Then injects it back into the
harness, automatically, every session.

The model is stateless. It starts cold. It only knows what someone
thinks to inject. Hero's job is to make sure the right context —
including the stuff nobody told it — lands in the model's window at
the right moment, without anyone asking. The workflows on top are the
surface; the corpus that compounds across sessions, people, and time
is the substance.

## Two layers, one manifesto

Hero is built in two layers. The mission and principles below apply
unchanged to both.

| Layer | What it is |
|---|---|
| **Core Hero** | The engine and mechanisms. Capture, structure, project, inject, sync, compound. Domain-agnostic. The substrate every vertical rides on. |
| **Verticals** | Domain-specific toolkits riding on the core. Each adds spec types, agents, skills, commands, vocabulary tailored to one kind of work. |

**Hero Code** is the engineering vertical
([`domains/engineering/mission.md`](../domains/engineering/mission.md))
— PM + product + engineering + testing + release-readiness. Today its
"UI" is implicit (the harness — Claude Code, Cursor, opencode — plus
the Hero dashboard). Future verticals will have their own dedicated
apps.

**Roster (current and planned):**

| Vertical | Status | UI surface |
|---|---|---|
| **Hero Code** (engineering) | Active — building | The harness + Hero dashboard |
| **Hero Sales** | Scaffold + spec [`hero-sales`](planning/features/hero-sales/spec.md) | Dedicated rep/AE app (planned) |
| **Hero Finance** | On the horizon | Dedicated app (web/desktop) |
| **Hero Accounting** | On the horizon (may merge with Finance) | Dedicated app |
| **Hero Marketing** | On the horizon | Dedicated app |
| **Hero Management** | On the horizon | Dedicated app — org / team / planning view |
| Legal, Research, Support, Design, ... | Open territory | TBD per vertical |

A **vertical** is a complete product riding on core Hero. It includes:
domain spec types, agents, skills, commands, vocabulary, *and
optionally its own dedicated UI* (desktop or web app). All verticals
share core's engine: the same corpus mechanics, the same graph, the
same manifesto, the same sync layer, the same hive-mind topology
across the three modes. Each vertical brings its own vocabulary and
its own user surface; none touches the core.

This means Hero is a **platform** at the core layer, and each vertical
is a **product** built on it — same shape as how something like Office
ships Word/Excel/PowerPoint as distinct products on a shared
foundation. Cross-vertical context (a sales conversation that informs
an engineering decision; a finance constraint that shapes a marketing
plan) flows through the shared corpus automatically.

## Three modes

Hero ships in three deployment shapes. Mission and principles are the
same in all three; the topology differs.

**Local mode** (free, default). Works amazing by yourself. New
sessions start as smart as where the last one left off. Everything in
`.hero/`. Zero network. Most users start here and many stay.

**Team mode via Hero Cloud.** The team becomes a hive mind. Hero Cloud
is the central anchor where everyone's local corpus syncs. Dev A's
hard-won lesson becomes Dev B's starting context, automatically. Same
UX as local; the network is invisible.

**Self-hosted team mode.** For organizations that can't or won't use
Hero Cloud — security, compliance, data sovereignty — run your own
Hero Cloud locally. Same code, your data plane. The hive mind is
yours.

## Principles

These are the experience bar. Every spec, every commit is checked
against them. Every drift mode the v2 audit catalogued is a violation
of at least one.

### 1. It just works.

Zero ceremony. No command sequences. Sane defaults. If a user has to
remember to run *A then B then C*, we failed. The system does the
right thing without being told.

### 2. Natural language is the interface; tools are the escape hatch.

The default user describes intent in their own words; the system
routes. Flags, sub-verbs, JSON output exist for the hardcore
practitioner — they never drown the default experience.

### 3. Sessions start omniscient.

A new session — any tool, any machine — feels like the previous one
never ended. **We should never start another session having to explain
what we are trying to build, what our goals are, or what plans are in
motion.** Re-priming is a system failure.

### 4. Sessions end making everyone smarter.

Every interaction's residue — decisions tried, attempts that failed,
conventions discovered, acceptance criteria that flipped — becomes
future context automatically. Zero ephemera. The corpus is always
richer at session end than at session start.

### 5. Two audiences, one product: the model and the human.

The magic is for the model and the less-experienced human. The tools
and flags are for the practitioner. Practitioner surface never drowns
the magic. **We raise the floor; we don't gate the ceiling.**

## Vocabulary

**Preferred:**
- **sidekick brain** — what Hero is to the user/model team
- **harness** — the AI coding tool the user lives in (Claude Code,
  Cursor, opencode, etc.)
- **corpus** — the project knowledge Hero curates
- **spec** — the design-before-build artifact
- **workspace** — `.hero/` and what it contains
- **session** — one continuous stretch of work in a harness
- **injection** — automatic delivery of context into the model's window
- **compounding** — the mechanic by which every session leaves the
  next one smarter
- **vertical** — a domain-specific toolkit riding on core Hero
- **hive mind** — the team-mode experience: shared corpus, shared
  context, shared institutional memory

**Banned:**
- *AI assistant*, *copilot*, *agentic platform*, *AI ops* — generic
  marketing flatness; doesn't say what we are
- *productivity tool* — too small; we're substrate, not a utility
- *context broker* — internal jargon; users don't think this way

## Anti-patterns

The system must never become these. Listed roughly by how directly
each violates the mission.

### Work without a spec

Code shipped, design decided, architecture chosen, bug fixed —
without anything captured. Whatever was learned dies with the
session; the next session re-discovers it from scratch. Violates
principles #3 and #4 directly. The spec-first loop exists exactly to
prevent this.

### Specs marked "completed" without verifiable acceptance criteria

The corpus lying about itself. Every downstream injection inherits
the lie; agents make wrong decisions because they trust framing the
code doesn't back up. Fixed by `spec-status-integrity`.

### Required command sequencing

A logical user action — *"ingest everything"*, *"start a fresh
session"*, *"ship a spec"* — implemented as a sequence the user must
chain (`hero scan && hero extract && hero sync pull`). Sub-verbs that
organize related but independent actions (`hero spec new`, `hero spec
deliver`) are good — they match how the user thinks. Sub-verbs that
fragment a single user intent across multiple commands are not.
Violates #1.

### Hero competing with the harness's brain

The harness is where reasoning, decisions, and code-writing happen.
Hero is the corpus manager and context delivery system that feeds the
harness. We don't take the harness's job. Local models for substrate
work — better extraction, projection synthesis, entity resolution —
are fair game when they clearly serve the corpus better than
deterministic code can. Cloud-LLM calls in the binary's hot path are
not.

### Plumbing without the user-visible feature it enables

Schemas, queues, frontends, engines that ship in isolation while the
thing they were supposed to power doesn't exist. The dominant v2
drift pattern (per the audit). A feature isn't done until it
*delivers* something to the user or the model.

### Future plans drowning the now

It is right to capture marketing, positioning, sales, distribution,
launch, and other forward work before we forget it. It is wrong to
let those captures appear in the same view as actionable work,
drowning the now-relevant signal. Without a prioritization mechanism
that distinguishes *now / next / someday / parking-lot*, captured
plans become noise. Fixed by [`spec-prioritization`](planning/features/spec-prioritization/spec.md).

## Mission-fit test

Every new spec, every commit, every PR must answer this in one
sentence:

> **"Does this make the next agent session start smarter than the
> last one ended — and does it raise the floor for everyone, not just
> the senior dev who already knows what to ask?"**

If the answer is unclear, the work is probably mission-adjacent and
should be horizon-tagged accordingly (see `spec-prioritization`).

## Charter discipline

This file is **not edited casually**. Changes require:

1. A "Mission revision" PR with explicit before/after diff in the
   commit body and a justification paragraph
2. The previous version preserved in graph history (bitemporal
   `valid_to` invalidation, never overwrite)
3. Vocabulary changes: every existing spec using a deprecated term
   must be updated or marked deprecated in the same PR — no silent
   term drift

Read this file at the start of every session. It is the
highest-priority context any agent working on Hero should hold.
`project-charter` will eventually inject it automatically into every
agent context bundle; until that ships, agents read it via the
AGENTS.md prepend.
