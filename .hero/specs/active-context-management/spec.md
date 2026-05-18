---
title: Active Context Management — Hero-Native Curator for Lean, Sharp Sessions
slug: active-context-management
type: feature
status: planning
tags: [context, curator, mcp, hooks, breadcrumbs, sub-agents, cache, measurement]
created: 2026-05-01
relations:
  - target: context-injection
    kind: extends
  - target: hero-context-pipe
    kind: related
  - target: agent-cold-start
    kind: related
horizon: now
---

## Kickoff

Build a Hero-native system that actively manages session context — not just *injecting* the right things at the start (which `context-injection` already does), but *curating* what stays in the model's window throughout a session: dropping what's re-derivable, keeping what's load-bearing, and externalizing the rest to a scratchpad so context stays sharp across long runs.

**Status:** planning — design captured; no code changes yet. Implementation will land as a v1 (additive only, ships on Claude Code today) and v2 (subtractive primitives, when the harness or SDK exposes them).

**Pick up at:** decide curator implementation approach (rule-based vs Haiku-call vs hybrid), then scaffold the Ledger as a session-scoped record in `.hero/sessions/<id>/ledger.jsonl`.

## Goal

Make every session start sharp and stay sharp — no matter how long it runs — by treating context as a tracked, curated resource rather than an accumulating tape. Reduce tokens-per-completed-task without quality regression, and lay architecture that absorbs future harness primitives (surgical eviction, PostToolUse redaction, context editing API) without rewrite.

## Problem

Hero already wins the **cold-start** battle: `prime`, `recap`, and `context-injection` make sure the model doesn't start blind. But mid-session, context bloats. Every long task accumulates:

- **Stale tool results** that informed one decision and now just take up tokens
- **File reads** that have since been edited, but the old content sits in history
- **Sub-agent return summaries** that contain detail the parent doesn't need
- **Redundant lookups** (the same file read three times across a session)
- **One-shot context** (a single grep result that's load-bearing for one turn and dead weight after)

The current floor is *reactive*: when the harness hits its limit, auto-compaction summarizes the whole conversation. That's coarse, lossy, and timed by the model running out of room — not by what's actually useful.

The deeper problem is that the harness is **purely additive**. There is no API, hook, or setting that lets us surgically drop a turn, redact a tool result, or rewrite past content mid-session. Every workaround has to inject *more* to combat *more*. So the design must work within an additive constraint today and fold in subtractive primitives when they arrive.

## Mission Fit

Hero's mission is to make sure the right context lands in the model's window at the right moment, automatically. Cold-start is solved. **Active management is the next frontier.** This is not mission-adjacent — it is mission-core. It also raises the floor for every session, not just sessions run by experts who know to manually `/compact`. A junior dev shouldn't need to know what prompt caching is to get a sharp session.

## Design

### Four Layers

```
┌──────────────────────────────────────────────────────────┐
│  Promotion       Lift findings → KB/memory (cross-session)│
├──────────────────────────────────────────────────────────┤
│  Reconstitution  Rehydrate breadcrumbs on demand          │
├──────────────────────────────────────────────────────────┤
│  Curator         Active sharpening between turns          │
├──────────────────────────────────────────────────────────┤
│  Ledger          Passive metadata on every context entry  │
└──────────────────────────────────────────────────────────┘
```

#### 1. Ledger — passive bookkeeping

Every chunk that enters context gets recorded with metadata:

```jsonl
{"id":"ctx_4f1","source":"tool:hero_read_spec","slug":"hero-ask","turn":12,"tokens":1840,"rederive_cost":"cheap","decay":"linear-30","pinned":false,"created_at":"..."}
{"id":"ctx_4f2","source":"tool:read_file","path":"internal/api/auth.go","turn":13,"tokens":2200,"rederive_cost":"cheap","decay":"on-edit","pinned":false}
{"id":"ctx_4f3","source":"user:decision","note":"prefer Haiku for curator","turn":14,"tokens":40,"rederive_cost":"expensive","decay":"none","pinned":true}
```

Stored as `.hero/sessions/<session_id>/ledger.jsonl`. Append-only. Cheap. The Ledger is the substrate for every other layer — without it, the Curator is blind.

**Sources** include Hero MCP tool calls, file reads (via PostToolUse observation), user messages, sub-agent returns, and explicit pin/note actions.

**Re-derivability cost** is a 3-way tag:
- **cheap** — single tool call away (`read_file`, `hero_read_spec`, single grep)
- **medium** — multi-step recovery (a sub-agent investigation, a synthesis across multiple sources)
- **expensive** — non-derivable (a user decision, an in-session insight, a debugging conclusion)

**Decay** is a function or label that says when this entry stops being load-bearing:
- `none` — never decays (pinned facts)
- `on-edit` — invalidated when the underlying file changes
- `linear-N` — relevance decays over N turns
- `task-scope` — relevant until the current task closes

#### 2. Curator — active sharpening between turns

The Curator runs *between* turns (via a hook on turn boundary), not within them. Why between: respecting prompt caching means the cached prefix should stay stable. The Curator's output is **additive** — it injects a sharpened "current working set" as fresh context the next turn, not a rewrite of the past.

**Curator output shape:**

```
<hero-working-set turn="42" generated="...">
## Active facts
- User wants Haiku-call curator (pin ctx_4f3)
- Working in internal/api/auth.go (latest content via tool result ctx_4f2 turn 13 — refresh if uncertain)

## Recent decisions
- Curator runs at turn boundaries, not within turns

## Available breadcrumbs (expand via hero_expand <ref>)
- ref_a1: spec hero-ask full body (was ctx_4f1, last loaded turn 12)
- ref_a2: full content of internal/api/auth.go at turn 13

## Recommended evictions (for awareness)
- ctx_4f1 (1840 tokens) — same info available via ref_a1 if needed
</hero-working-set>
```

The model sees this each turn and treats it as the authoritative working set. Old verbatim content stays in history (we can't delete it) but the model is steered toward the curated version.

**Curator implementation: hybrid recommended.**
- **Rule-based** for the obvious wins (this is 80% of value):
  - Drop tool results older than N turns where `rederive_cost == cheap` AND not referenced since
  - Mark file reads as stale when the file is later edited
  - Collapse duplicate reads (same file read 3x → one breadcrumb)
- **Haiku call** for nuanced summarization at curation milestones:
  - Synthesize "active facts" from the last K turns
  - Identify implicit pins ("the user keeps coming back to X — pin it")
  - Extract decisions worth promoting
- **Cost guard:** Curator must spend less than its savings. Track `curator_tokens_in / tokens_evicted_equivalent` ratio; if curator spend exceeds 10% of savings, throttle.

#### 3. Reconstitution — rehydrate on demand

When a breadcrumb is needed back in context, a single tool call resolves it:

```
hero_expand <ref_id>
```

Behind the scenes:
- If KB-cached, return cached blob
- If not, re-run the original fetch (re-read the file, re-call the MCP tool)
- Update Ledger with reconstitution event

Reconstitution rate is a quality signal: high rate means the Curator was wrong, low rate means it's too conservative. Surfaced via `hero context --audit`.

#### 4. Promotion — lift to KB/memory

Some session findings are worth permanent capture. The Curator flags promotion candidates from the Haiku synthesis pass:
- Decisions made (architectural choices, pins, rejected alternatives)
- Patterns discovered (a convention emerged from three similar fixes)
- Surprising findings (a non-obvious bug pattern, a hidden constraint)

Integrates with the existing `capture` skill — Curator drafts a candidate, `capture` writes it to KB. Avoids re-deriving the same insight in future sessions.

### Two-Tier MCP Tool Responses

The single biggest concrete lever available today: **make every Hero MCP tool return a compact summary by default, with a `ref_id` for expansion.** No harness change required.

**Worked example — `hero_read_spec`:**

Today (verbose):
```json
{
  "spec_id": "hero-ask",
  "title": "...",
  "body": "<full markdown — 4000 tokens>",
  "frontmatter": {...}
}
```

After (two-tier):
```json
{
  "spec_id": "hero-ask",
  "title": "Hero Ask — Conversational Q&A Over the Corpus",
  "summary": "Adds `hero ask <query>` for natural-language Q&A over specs/KB/conventions. Status: planning. Key decisions: federation across local + cloud KB; cite sources inline.",
  "ref_id": "spec:hero-ask:full",
  "expand_via": "hero_expand"
}
```

The model gets the gist for free. If it needs the full body it asks via `hero_expand spec:hero-ask:full`. Apply the same pattern to:
- `hero_search` (results return titles + ref_ids; full body on expand)
- `hero_context` (summary of injected context with ref_ids per knowledge entry)
- `hero_why`, `hero_blocked`, `hero_feed`, `hero_recap`
- All read-side tools

**Eviction-by-default tools** flip the burden. Today every tool dumps everything because there's no way to ask later. With reconstitution available, tools return signal-rich summaries and let the model pull the rest.

### Externalized Scratchpad

The Curator maintains `.hero/sessions/<id>/working-set.md`. Loaded fresh each turn via the Curator's injected context (so it lives outside conversation history and stays current). This file is the externalized "what I currently know" — the model's actual working memory, edited between turns by the Curator.

### Curation Milestones

Heuristics that trigger a deeper curation pass (Haiku call, scratchpad rewrite):
- Spec status flips (`planning` → `delivering` → `completed`)
- Sub-agent returns
- User invokes `/compact`, `/handoff`, `capture`, or completes a task
- Ledger crosses configurable token threshold
- N turns elapsed since last milestone

Between milestones, only rule-based curation runs (cheap, no model call).

### Cache Strategy

**Default: stable prefix, curated tail.** The cached system prompt and early turns remain untouched. The Curator's working-set injection appends to recent context. This preserves cache hits and keeps cost predictable.

**Reset points: opt-in, not automatic.** At deliberate boundaries (task complete, phase shift, user-triggered) we accept a cache miss in exchange for a sharper baseline. Surfaced as `hero context --reset` and offered automatically at milestones with high confidence.

We do **not** rewrite cached content speculatively. The risk of cache-miss tax exceeding curation savings is too high without measurement to back it.

### v1 vs v2 — What Ships When

**v1 — buildable now on Claude Code:**

| Layer | v1 mechanism |
|---|---|
| Ledger | PostToolUse / SessionStart hooks observe and record; MCP tools self-report |
| Curator | UserPromptSubmit hook injects working-set; rule-based + optional Haiku call |
| Reconstitution | `hero_expand` MCP tool; KB cache substrate |
| Promotion | Hooks into existing `capture` skill |
| Two-tier tools | Hero MCP changes only — no harness dependency |
| Milestones | Rule-based detection in Hero hooks |
| Reset points | Programmatic `/compact` trigger from hook |

**v2 — when harness/SDK primitives land:**

| Capability | Replaces v1 workaround |
|---|---|
| Surgical eviction | Drop ledger entries from real context, not just recommend evictions |
| PostToolUse redaction | Truncate tool results before they enter context (no breadcrumb dance) |
| Past-turn rewriting | Curator can replace old verbatim content with breadcrumbs in-place |
| Context editing API (SDK-direct) | Use Anthropic's primitives directly when running headless |

The v1 architecture is intentionally compatible: the Ledger, Curator, and breadcrumb model all work the same; v2 just lets the Curator's eviction recommendations become eviction *actions*. No throwaway code.

## Open Design Questions

1. **Curator implementation — confirm hybrid.** Rule-based for routine, Haiku for milestones. Recommended default; revisit if Haiku call cost dominates savings on real sessions.
2. **User visibility — silent by default, observable on demand.** Curation actions logged to `.hero/sessions/<id>/curator.log`. Surfaced via `hero context --audit`. No interactive approval (would defeat the purpose), but every action is auditable.
3. **Hero feature vs harness feature — Hero-first.** We have the MCP integration surface, the KB substrate, and the existing primitives (`capture`, `context`, sub-agents). Any harness with Hero MCP installed gets the curator for free. Future-proofs us as harness primitives evolve. (Does **not** preclude a separate harness-native version later.)
4. **Session ID provenance — TBD.** Does the curator key off Hero's session concept or the harness's? Recommend Hero session keyed off `cwd + start time` until harnesses standardize a session-id convention.
5. **Ledger garbage collection — TBD.** When does a session's ledger get archived/pruned? Recommend keep last N sessions hot, archive older to KB.

## Acceptance Criteria

- WHEN a Hero MCP read-side tool returns a result THE SYSTEM SHALL return both a compact summary and a `ref_id` for expansion
- WHEN the model invokes `hero_expand <ref_id>` THE SYSTEM SHALL rehydrate and return the full content
- WHEN any context-bearing event occurs (tool call, file read, user message, sub-agent return) THE SYSTEM SHALL append an entry to the session Ledger with source, tokens, re-derivability cost, and decay metadata
- WHEN a turn boundary is reached THE SYSTEM SHALL emit a sharpened working-set context for the next turn via `UserPromptSubmit` injection
- WHEN a curation milestone is detected THE SYSTEM SHALL run a deeper synthesis pass (Haiku) and update the externalized scratchpad
- WHILE a session is active THE SYSTEM SHALL maintain `.hero/sessions/<id>/working-set.md` reflecting the current curated working set
- WHILE a session is active THE SYSTEM SHALL log every curation decision (eviction recommendation, breadcrumb creation, promotion candidate) for audit
- IF the Curator's reconstitution rate exceeds a configurable threshold (default 25%) THEN THE SYSTEM SHALL widen retention policy and log the tuning event
- IF the Curator's token spend exceeds 10% of estimated savings THEN THE SYSTEM SHALL throttle Haiku calls and fall back to rule-based mode
- IF a Ledger entry's re-derivability cost is `expensive` THEN THE SYSTEM SHALL NOT recommend its eviction
- WHERE harness subtractive primitives are available (v2) THE SYSTEM SHALL prefer surgical eviction over additive curation
- WHERE the user invokes `hero context --audit` THE SYSTEM SHALL display the current Ledger, working set, and recent curation decisions
- WHERE the user invokes `hero context --reset` THE SYSTEM SHALL trigger a deliberate cache-reset milestone with a fresh sharpened baseline
- THE SYSTEM SHALL preserve cache hits by default (no speculative rewriting of cached content)
- THE SYSTEM SHALL emit promotion candidates to the existing `capture` skill at task-close milestones

## Measurement Plan

Without measurement, this becomes vibes-based. Required metrics, recorded per session:

- **Tokens-per-completed-task** — baseline measurement before rollout, post-rollout comparison
- **Quality regression watch** — sample of representative tasks scored against pre-curator baseline (does the model still get the right answer with less context?)
- **Curator overhead** — wall-clock time and tokens spent on curation itself, as a percentage of session totals
- **Reconstitution rate** — fraction of breadcrumbs that get expanded back. Target: 5–20%. Below 5% = curator too conservative; above 25% = curator too aggressive.
- **Cache hit rate** — measure prompt cache hits per turn before and after; ensure curator doesn't tank caching
- **Promotion conversion** — fraction of promotion candidates that become actual KB entries

Surface a per-session summary via `hero pulse --context` so users can see the curator's effect in their own sessions.

## Out of Scope / Boundaries

- **Not a harness fork.** This works on top of Claude Code (and any harness with Hero MCP); we don't ship a forked harness in v1.
- **Not a replacement for `prime`/`recap`/`context-injection`.** Those handle cold-start; this handles mid-session. They compose.
- **Not a memory replacement.** Hero memory and KB are persistent across sessions; the curator is a session-scoped active layer feeding into them.
- **No interactive curation in v1.** Curator runs silently. Manual override exists (`hero context --pin`, `hero context --evict`) but no prompts mid-flow.
- **No model-side context manipulation in v1.** v1 stays additive; surgical eviction waits for v2.
- **Not implementing Anthropic's context editing API in v1.** That's a v2 path when running SDK-direct, not Claude Code.

## Changes (anticipated, will be confirmed at delivery)

- `internal/context/ledger.go` — Ledger append/query/persist
- `internal/context/curator.go` — rule-based curation engine
- `internal/context/synthesizer.go` — Haiku-call synthesis at milestones
- `internal/cli/expand.go` — `hero expand` CLI + MCP tool
- `internal/cli/context.go` — `--audit`, `--reset`, `--pin`, `--evict` flags
- `internal/mcp/*` — two-tier response shape across read-side tools
- `internal/hooks/*` — hook integrations (UserPromptSubmit, PostToolUse observation, SessionStart)
- `skills/active-context-management.md` — skill explaining the system, when to trust it, how to override
- `agents/*` — delivery-lead and engineer agents updated to recognize working-set injection
- Measurement instrumentation: `hero pulse --context`, session metrics in `.hero/sessions/<id>/metrics.json`
