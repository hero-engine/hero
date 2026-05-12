---
title: Active Context Management — Design Constraints and Layering Rationale
type: note
created: 2026-05-01
tags: [context, curator, harness, claude-code, mcp, breadcrumbs, cache, architecture]
horizon: now
---

# Active Context Management — Design Constraints and Layering Rationale

Captured during design of `active-context-management` spec. Records the load-bearing constraints and decisions so future work doesn't re-derive them.

## The harness-additive constraint (load-bearing)

As of 2026-05, Claude Code is **purely additive**. Confirmed mechanisms:

- `UserPromptSubmit` / `SessionStart` hooks can **inject** context, not remove
- `PreToolUse` / `PostToolUse` hooks can **observe and block**, not redact tool results
- No API/setting caps tool result size before context ingestion
- No hook can rewrite past turns
- Subtractive primitives are limited to `/compact` (manual, coarse) and reactive auto-compaction

The Anthropic Messages API has context editing primitives, but they are exposed for SDK-direct use — **not surfaced through Claude Code**.

**Implication:** any context management system that ships on Claude Code today must be additive. You can inject a curated working set; you cannot delete the noise. Architectures must absorb future subtractive primitives without rewrite.

## Re-derivability heuristic (Hero principle)

If a piece of context is cheap to re-fetch (a single tool call, a single file read), do not carry it in context — replace it with a breadcrumb pointer and rehydrate on demand. Only carry what is expensive to re-derive (decisions, in-session insights, synthesis across sources).

This generalizes beyond curators:
- Tool response design — return summaries with `ref_id`, expand on request
- Sub-agent fences — child results return as one-sentence summaries; full work stays in child
- Memory promotion — only pull KB-worthy findings into long-term memory, not raw outputs

## Cache strategy: stable prefix, curated tail

Claude prompt caching has a 5-minute TTL and rewards stable prefixes. Aggressive mid-session rewriting causes cache misses that often exceed curation savings.

**Default rule:** never rewrite cached content speculatively. The Curator appends an injected working-set to recent turns; cached prefix stays untouched.

**Reset points are opt-in.** At deliberate task boundaries (spec close, phase shift, user-triggered), accept a cache miss in exchange for a sharper baseline. Do not attempt automatic resets without measurement that the savings beat the miss.

This is non-obvious — without measurement, "rewrite for clarity" feels free but isn't.

## Four-layer architecture (canonical for any context system)

1. **Ledger** — passive metadata on every context entry (source, tokens, re-derivability cost, decay, pin state). Cheap, append-only, the substrate for everything else.
2. **Curator** — active sharpening between turns (not within, to respect cache). Hybrid: rule-based for routine, model-call (Haiku) for synthesis at milestones.
3. **Reconstitution** — single tool to rehydrate breadcrumbs. Eviction is reversible.
4. **Promotion** — lift session findings to KB/memory. Avoids re-derivation across sessions. Integrates with existing `capture` skill.

## Cost guard for any model-in-the-loop curation

If the curator spends more tokens than it saves, it loses. Track `curator_tokens_in / tokens_evicted_equivalent`. If curator spend exceeds 10% of estimated savings, throttle Haiku calls and fall back to rule-based mode.

This guard applies to any synthesis-during-session feature, not just the curator.

## v1/v2 layering as a delivery discipline

When harness limits force a workaround, design the workaround so the proper primitive slots in without rewrite. For context management:

- v1 Curator emits **eviction recommendations** as additive injected context
- v2 Curator emits **eviction actions** through subtractive primitives
- Same Ledger, same breadcrumbs, same Reconstitution — the action layer changes, not the architecture

Pattern generalizes: when blocked by a harness gap, build the inert form of the eventual primitive (recommendation, log, advisory) so the active form is a swap, not a rebuild.

## Related work

- `context-injection` — cold-start context (already shipped); active management is the mid-session complement
- `hero-context-pipe` — output format control; orthogonal but composable
- `notes/memory-tools-and-community-patterns` — community landscape (claude-mem progressive disclosure pattern echoed here as two-tier MCP responses)
