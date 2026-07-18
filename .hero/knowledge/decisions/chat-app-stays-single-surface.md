---
title: "Chat App Stays a Single Simple Surface — No Domain-Switching or Mode Packs"
type: decision
status: accepted
created: 2026-07-18
tags: [chat, domains, product-scope, hero-code, architecture, simplicity]
relations:
  - target: chat-canonical-research
    kind: related
---

# Chat App Stays a Single Simple Surface — No Domain-Switching or Mode Packs

## Decision

The standalone chat app stays **one simple conversational surface**. We will
**not** add domain/mode switching to it, **not** graduate research into its own
domain, and **not** build a lightweight `code` domain as a selectable mode.
Research capabilities remain **baked into the baseline chat pack** (as delivered
by `chat-canonical-research`). There is one surface, not a mode picker.

## Context

A design exploration (2026-07-18) proposed:
- graduating the research apparatus out of baseline chat into its own `research`
  domain,
- authoring a lean `code` domain (a coding-craft mode), and
- letting the chat app **switch** between chat / research / code modes.

Planning artifacts were drafted (a `research-graduates-to-its-own-domain`
decision, a `research-domain-pack` spec, and a `code-domain-pack` initiative with
three child specs). **None were committed.** All were abandoned and deleted when
this decision was made.

## Why we rejected it

hero-code — which owns the chat client — made the decisive argument: **adding
domain-switching plus a research mode plus a code mode to what is meant to be a
basic chat app just turns it into a second hero-code.** It rebuilds the full
multi-surface, multi-mode, specialized-view complexity in the one product whose
entire value proposition is being lightweight and simple.

The complexity is *justified* in hero-code (the full IDE-grade experience:
specialized views, mode switching, the complete engineering pack). It is exactly
what the simple chat app should **not** grow. Optimizing the pack architecture
(clean domain layering, a code mode "like Codex/Claude Code") lost sight of the
product point: the chat app's job is to stay basic.

## What this means

- **No domain switcher** in the chat app.
- **No separate `research` or `code` domain packs** for the chat app.
- **Research stays as shipped** — baked into the baseline chat pack
  (`chat-canonical-research`), one surface, no switching.
- The full multi-mode, multi-pack, specialized-view experience remains
  **hero-code's** job. The chat app stays simple.

## Consequences

- Reverses the earlier same-session "research graduates to its own domain"
  direction (never committed; deleted).
- `chat-canonical-research` stands as delivered and shipped; nothing to unwind
  there.
- If the baseline chat's research workflow ever feels too heavy for "basic," that
  is a **trim-in-place** conversation about baseline chat — not a reason to
  reintroduce domain graduation or mode-switching.
- General principle to carry forward: **do not push chat-app-side complexity that
  duplicates hero-code.** When a feature would give the lightweight app a second
  mode/surface/pack, that is a signal it belongs in hero-code, not the chat app.
