---
title: "Basic Chat Sheds Research — Extract the Research Apparatus to a Dormant Hero Research Seed"
slug: chat-sheds-research-to-seed
type: decision
status: accepted
priority: P1
domain: engineering
created: 2026-07-18
tags: [chat, domains, product-scope, hero-code, research, client-embedded, simplicity, decision]
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: hero-chat-swift-app
  call_id: 18c370942a72d9a8e2b54779ba784fd7
  mode: spec-out
  handed_off_at: 2026-07-18T16:54:19Z
  at_commit: b44e9e87
  reason: "Lock the revised Hero Chat boundary before more client work is designed; supersede the conflicting Research-baked-into-Chat conclusion while preserving the no-domain-switcher and no-lightweight-Code conclusions."
relations:
  - target: chat-app-stays-single-surface
    kind: supersedes
  - target: chat-canonical-research
    kind: related
  - target: chat-slim-to-basic-research-seed
    kind: related
---

# Basic Chat Sheds Research — Extract the Research Apparatus to a Dormant Hero Research Seed

## Decision

**Baseline Hero Chat is basic Chat. Research does not live in it.**

The guided-research apparatus that `chat-canonical-research` added to the
baseline chat pack — the `/research` command, the `researcher` /
`document-analyst` / `data-analyst` agents, the five research/analysis skills,
and the machine-readable plan → round → evaluation → synthesis → report
checkpoint/interrupt contract — is **removed from `domains/chat`** and
**preserved verbatim, dormant, outside `domains/`** as seed material for a
possible future Hero Research product.

This decision **supersedes `chat-app-stays-single-surface`**. Two of that
decision's three conclusions are re-adopted here **unchanged**; only its
"research stays baked into baseline chat" conclusion is reversed.

Re-adopted from `chat-app-stays-single-surface` (still in force):

1. **No domain switcher in the chat app.** The standalone chat app remains one
   simple conversational surface, not a mode picker.
2. **No lightweight `code` domain** authored as a selectable chat-app mode. The
   full multi-mode, specialized-view experience stays hero-code's job.

Reversed:

3. ~~Research stays baked into the baseline chat pack~~ → **Basic Chat does not
   own research.** The research apparatus is extracted to a dormant seed.

## Context

`chat-app-stays-single-surface` (accepted 2026-07-18) resolved a design debate
by keeping the chat app simple: no domain switching, no `research` domain, no
`code` domain — and, as a fourth clause, keeping research *baked into baseline
chat* "as shipped" by `chat-canonical-research`.

That fourth clause did not survive contact with the product intent. A basic chat
app that ships a guided-research workflow — a reviewable plan the user must
approve, controlled-source rounds, a source ledger, source-evaluation passes, an
evidence-synthesis progress model, checkpoint state, and interrupt-safe partial
reports — is no longer basic. It is a research *product* wearing a chat app's
clothes, and it forces the client (hero-code) to build plan-approval, progress,
and interrupt UI for a surface whose entire value proposition is staying light.

The same-session correction already began on disk: commit `04a0b5d` ("trim
research apparatus — keep chat simple") removed `/research`, the three agents,
and the five skills from `domains/chat`, returning it to a commands-only
conversational pack with light, natural research *habits* (cite sources, don't
fabricate, look things up when asked, read what the user shares) in `AGENTS.md`.
But that trim **deleted** the extracted content rather than preserving it, and it
left `chat-app-stays-single-surface` still asserting — as an accepted decision —
that "research stays baked into the baseline chat pack." The corpus therefore
holds an accepted decision that contradicts the tree. This decision closes that
gap and records the preservation intent the ad-hoc trim skipped.

## Why

- **"Basic chat" and "guided research product" are different products.** Bolting
  the second onto the first reproduces exactly the "second hero-code" failure
  mode that `chat-app-stays-single-surface` itself warned against — just via a
  workflow instead of a mode picker.
- **The research work is genuinely good and worth keeping.** The doctrine
  authored in `chat-canonical-research` (plan-first with a hard approval pause,
  controlled sources, honest round loop, per-source evaluation, cited synthesis
  with contradiction-surfacing, interrupt-safe partial reports) is high quality.
  Deleting it outright — as `04a0b5d` did — throws away product-seed value. A
  future Hero Research app or a `research` domain that `Extends: chat` could lift
  it directly. So it is **preserved, not discarded**.
- **Preserve, but keep it dormant and un-stageable.** hero-code's
  `crates/hero-core/build.rs` stages content by iterating **every** directory
  under `domains/` and copying each pack's `agents/`, `skills/`, and `commands/`.
  Any research content left anywhere under `domains/` — inside `domains/chat/…`
  or as a new `domains/research/…` pack — would be re-exposed to a client at
  build time with no Go or build.rs change. The seed must therefore live
  **entirely outside `domains/`**, where neither build.rs (which reads only
  `domains/`) nor Go's `go:embed` set (which enumerates specific `domains/*`)
  can reach it. The implementation spec names the exact path.

## What this means

- `domains/chat` is **basic Chat**: the six original commands
  (`ask-corpus`, `capture`, `discover`, `note`, `space`, `why`), a light
  `AGENTS.md` with soft research *habits* and the single conditional
  natural-writing rule — no `/research`, no researcher lifecycle, no plan
  approval, no controlled-source rounds, no source ledger, no source-evaluation
  workflow, no evidence-synthesis progress, no checkpoint state, no
  interrupt-safe partial reports, no report/paper authoring.
- The extracted research apparatus is **preserved verbatim in a dormant seed
  location outside `domains/`**, documented as future Hero Research material and
  guaranteed un-stageable by the directory-iterating build.
- **No domain switcher** and **no lightweight `code` domain** — carried unchanged
  from `chat-app-stays-single-surface`.
- `chat-canonical-research` stands as **completed and historically correct**.
  Nothing about its delivery record is reopened, failed, or rewritten; this
  decision changes the *forward* boundary, not the past.

## Consequences

- `chat-app-stays-single-surface` is marked **superseded** and points forward to
  this decision. Its surviving conclusions are not orphaned — they are re-adopted
  verbatim above, so exactly one accepted decision now governs the chat boundary.
- hero-code should **not** build plan-approval, progress, or interrupt UI for
  chat mode; that contract left baseline chat with `04a0b5d` and is now formally
  dormant. (An advisory reversal notice already went to hero-code at that commit;
  this decision is the durable record.)
- The implementation is one planning feature — `chat-slim-to-basic-research-seed`
  — which finalizes the slim and, crucially, **recovers** the research content
  `04a0b5d` deleted (from commit `3a09d27`) into the dormant seed. This decision
  does not itself move code.
- If Hero Research is ever built, the seed is the starting point — a graduation
  (`research` domain that `Extends: chat`, or a standalone Hero Research app),
  **not** a reason to re-bake research into baseline chat. Reviving it is a new
  decision, not a silent restoration.

## Provenance

Received via `hero peer call` **spec-out** mode (call_id
`18c370942a72d9a8e2b54779ba784fd7`) from peer `hero-code`
(peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`), originating from that repo's
`hero-chat-swift-app` initiative. hero-code owns the client (loading and
presentation); this repo owns the canonical, client-agnostic Chat pack content
and the governing decision.
