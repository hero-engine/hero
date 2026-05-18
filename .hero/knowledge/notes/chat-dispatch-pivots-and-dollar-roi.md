---
title: Chat dispatch reframes + dollar / Net / ROI-multiple chain
type: note
status: active
created: 2026-05-17
tags: [hero-serve, chat, model, adapter, hero-code, roi, decision-history]
relates-to:
  - hero-chat-and-model
  - hero-people-and-roi-home
  - hero-surface-architecture
---

## What this captures

Two design pivots and one product insight from the Hero Surface
Architecture design session that would otherwise be lost. The
hero-chat-and-model spec is the third draft; the People & ROI home
spec carries the dollar chain as a "Money tab" in the metric strip.
This note traces how each landed.

## Pivot 1 — Local model in hero serve (rejected)

**First draft of `hero-chat-and-model`:** hero serve runs inference
itself. A `chat.model` block in `hero.json` lets a user add an API
key for a provider (Anthropic, OpenAI, Azure — the existing
`internal/runner/` plumbing). Direct provider calls from hero serve.
"Optional" framing: if no IDE harness is connected, fall back to the
local model.

**Rejected by the user**: *"we don't want to become an inference
engine here — unless you want to merge this all with hero-code now."*

This pushed the boundary out of hero serve entirely.

## Pivot 2 — IDE-harness-required (rejected)

**Second draft:** every chat invocation routes to an IDE harness
(Claude Code / Cursor / Codex) via a new `hero_chat` MCP tool the
harness must expose back. Hero serve becomes a pure router. The
empty-state CTA says "Connect your IDE."

**Problem:** required harness adoption work we don't control. Until
Claude Code, Cursor, and Codex each implement a `hero_chat` MCP
tool, harness mode is no-op even when an MCP client is connected.
That's months of dependency on three vendors before anything works.

The user noted: *"it would be great if hero inside a harness or IDE
— or hero-code — would be listening for or recognize the request
and process it. not sure if we can do it in others IDEs but killer
in hero-code — and a reason to say 'requires hero-code' if we want
and can't do it another way."*

That reframed the problem: instead of demanding every harness
implement a wire contract we control, lean on a runner we DO
control (hero-code) and treat IDE bridges as a bonus where feasible.

## Pivot 3 — Hero adapter abstraction (landed)

**Third draft (landed):** one abstraction — the Hero adapter — for
anything that can pick up a dispatch from hero serve and run the
agent loop. Adapters live in one of two places:

- **hero-code** (canonical) — the sibling runner. Always works.
  Handles interactive + headless. Required baseline. *"Requires
  hero-code" is honest, not apologetic.*
- **In-IDE Hero adapter** (optional, aspirational) — a plugin /
  skill inside an IDE that picks up dispatches and runs them in the
  IDE's own agent loop. Claude Code is the realistic v1 target via
  its skills + hooks system. Cursor / Codex are TBD.

Hero serve never holds an API key, never imports an LLM SDK, never
bills tokens. Build-time check enforces `internal/serve/chat/*`
cannot import `internal/runner/*`.

**Hero-code reality check** (via peer call to the sibling repo):
hero-code is bucket (c) — significant joint design work. It has MCP
client + orchestration primitives but no server-side adapter wire
surface yet. Binary is actually `hero-chat` in `crates/hero-cli`.
Adapter contract is a parallel build, not a "consume as-is."

## The merge question

Recurring during these pivots: should hero serve and hero-code
merge? Hero-code itself has suggested it (and is exploring Rust).

My take, captured here for posterity: **keep them separate, for
product reasons more than technical ones.**

- **Hero serve** = the surface. Read-mostly. Cheap to embed. Runs
  anywhere (laptop, dev box, cloud, server).
- **Hero-code** = the runner. Inference + tools + sandbox + cost.
  Heavier; genuinely benefits from Rust for concurrent tool
  execution and predictable latency.

Merge them and the surface drags an LLM SDK + sandbox + tool
runtime everywhere it ships; every runner change re-validates the
dashboard. Two binaries with a clean MCP contract between them is a
strength — they can scale and ship independently.

One scenario where I'd reconsider: if "install two things" kills
adoption. Then embed hero-code as a **subprocess** of hero serve
(one install, two processes, clean separation). Best of both worlds
without a code merge. The chat-and-model spec's architecture works
either way.

## The friend-with-Hero insight — dollar / Net / ROI multiple

User shared this during the People & ROI design conversation:

> *"a friend using hero asking this stuff — and he got it to tell
> him like how many hours of dev had been saved and dollar saved
> estimates etc — super interesting"*

That insight reshaped the People & ROI home's headline metric strip.

**Hours saved** is the substrate (computed from the saved-hours
estimator: agent edits × time-per-edit + auto-imports × triage-time-
saved + …). **Dollars saved** is the punchline. **Net value** and
**ROI multiple** close the loop on the budget conversation a
manager actually has.

The chain:

```
hours_saved(window)    — saved-hours estimator
dollars_saved(window)  = hours_saved × c_hourly_cost     (default $150/hr loaded)
hero_api_spend(window) = Σ adapter-reported cost
net_value(window)      = dollars_saved − hero_api_spend
roi_multiple(window)   = net_value ÷ hero_api_spend
```

The ROI Overview's **Money** metric tab renders all four as
equal-weight tiles — Hours and Dollars side by side as the headline
pair, Net and ROI multiple as the "so what" derivatives. Each
tile's sub-line shows the chain so a reader can verify the math
without opening the methodology modal (e.g., `≈340h × $150/hr
loaded` on the Dollars tile).

**Privacy guardrail**: no per-engineer dollar breakdown ever. Only
team-level totals and contributor share. The `c_hourly_cost`
coefficient is loaded-team-average, not per-person.

**Headline framing**: the ROI Overview page hero's subhead surfaces
the net value and ROI multiple together: *"Last 4 weeks · 142 specs
delivered · ~$49.9K net value · 44× ROI."* That's the line that
gets screenshotted into a budget review.

## What to remember

- Hero serve is a dispatcher. If a feature requires an LLM, it
  doesn't live here.
- "Requires hero-code" is honest. Don't apologize for it.
- Two binaries with a clean MCP contract beat one fat binary with
  unclear boundaries.
- The hero-code adapter wire contract is a parallel joint build —
  don't assume hero-code is "ready to consume as-is."
- Dollar / Net / ROI multiple is the bit that gets Hero adopted
  past the lead dev. Don't bury it. Don't apologize for it. Show
  the math so it's auditable.
- Configurable coefficients in `hero.json` (`roi.coefficients`)
  make the numbers calibratable per team — that's what makes them
  defensible to a CFO.
