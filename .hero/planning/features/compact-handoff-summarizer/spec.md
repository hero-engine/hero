---
title: "Compact Handoff Summarizer — LLM-Curated Middle Section for Resume Context"
slug: compact-handoff-summarizer
type: feature
status: draft
priority: medium
horizon: next
tags: [next, compaction, llm, ai-services, hero-cloud]
relations:
  - target: next-compact-handoff
    kind: depends-on
  - target: local-project-model
    kind: related
  - target: embeddings-index
    kind: related
created: 2026-05-21
---

# Compact Handoff Summarizer — LLM-Curated Middle Section for Resume Context

## Problem

The MVP compact handoff ([next-compact-handoff](../next-compact-handoff/spec.md)) ships a deterministic skeleton — active spec, session metadata, files touched, recent decisions, next action. That covers the **factual** axis: hard pointers, graph state, exact strings.

It does not cover the **comprehension** axis: what was actually decided in the conversation that hasn't been logged as a graph event yet, what ruled-out approaches the next session needs to know about, what subtle context the user provided that won't survive Claude Code's auto-generated conversation summary.

The deterministic skeleton can list "5 Decision events from the graph." It cannot say "the user clarified they want option B over option A because of the deployment constraint they mentioned three turns ago." Only a model that has read the transcript can produce that.

## Goal

Add an LLM-curated middle section to the compact handoff. A small, fast model reads the conversation transcript and produces a tight 300–600 token synthesis with three labeled segments — **Working on**, **Decided / ruled out**, **Up next** — that gets concatenated into the existing `additionalContext` payload between the deterministic header and the working-tree footer.

**Concrete success criteria:**

- After a Claude Code compaction, the injected `additionalContext` includes both the deterministic skeleton from the MVP *and* a model-generated middle section synthesizing what was just compacted.
- The summarizer completes within a 5s hard timeout. On timeout/error/missing-API, the handoff still ships with the skeleton-only path — no broken hooks.
- The architecture supports two backends from day one: **direct** (call the API with a user-provided key) and **hero-cloud** (call Hero's hosted summarization service, no user key required). Default selection is configurable; one is always picked.
- Adding future backends (local model, alternate provider) is an interface implementation, not a redesign.

## Design

### The AIServices seam

Introduce `internal/aiservices/` as the single seam for "Hero needs a model to do work." This is not a one-off helper for the summarizer — it's the architectural commitment for every future feature that needs comprehension (embeddings index queries, peer-call prompt composition, spec-relation auto-suggestion, the "what's helpful" engine).

```go
// Package aiservices is the single seam for Hero's AI-assisted features.
package aiservices

type Summarizer interface {
    Summarize(ctx context.Context, req SummarizeRequest) (SummarizeResponse, error)
}

type SummarizeRequest struct {
    Transcript    string            // raw transcript content
    SystemPrompt  string            // task-specific instruction
    MaxTokens     int               // hard cap on response
    Metadata      map[string]string // session_id, spec_slug, etc. (for hero-cloud telemetry)
}

type SummarizeResponse struct {
    Text         string
    Provider     string // "anthropic-direct", "hero-cloud", "openai-direct", ...
    Model        string
    LatencyMs    int
    InputTokens  int
    OutputTokens int
}
```

Two implementations in MVP:

1. **`directProvider`** — calls Anthropic API directly using `ANTHROPIC_API_KEY` (or OpenAI with `OPENAI_API_KEY`). Picks Haiku by default. ~$0.001 per call.
2. **`heroCloudProvider`** — calls Hero's hosted summarization endpoint. Hero-cloud picks the model/provider internally. Requires Hero account credentials (separate user-config story).

### Default backend selection

Order of resolution at first call:

1. If `hero.json` has `ai_services.summarizer.provider` set explicitly → use it.
2. Else if Hero is configured against hero-cloud (credentials present) → `hero-cloud`.
3. Else if `ANTHROPIC_API_KEY` is set → `direct` (anthropic).
4. Else if `OPENAI_API_KEY` is set → `direct` (openai).
5. Else → no summarizer; skip Part 2; skeleton-only handoff with a one-line note ("LLM summarizer skipped: configure hero-cloud or set ANTHROPIC_API_KEY").

`hero.json` override:

```json
{
  "ai_services": {
    "summarizer": {
      "provider": "hero-cloud | anthropic-direct | openai-direct | local | none",
      "model": "claude-haiku-4-5-20251001",
      "endpoint": "http://localhost:11434/v1",
      "timeout_ms": 5000
    }
  }
}
```

### Summarization prompt

The hook command `hero next compact-summarize` invokes the configured provider with:

**System prompt:**

> You are summarizing a coding-assistant conversation transcript for the model that will pick it up after a context compaction. Produce ONLY the three labeled sections below, in this exact order. No preamble, no closing. Total length ≤600 tokens.
>
> **Working on:** One sentence naming the specific thing this session is in the middle of.
>
> **Decided / ruled out:** Bulleted list of decisions made or approaches rejected in this conversation that aren't obvious from the spec or graph. Skip anything that's already recorded as a Decision event. If nothing meaningful: write "Nothing notable beyond what's already in the graph."
>
> **Up next:** The immediate next action the resuming session should take. Be concrete — a specific file edit, a specific verification, the next acceptance criterion. Not generic ("continue working on the feature").

**User content:** the transcript (last 32K tokens) plus a one-line postscript indicating the active spec slug (so the model has the framing).

### Transcript truncation

Long sessions produce long transcripts. The provider call truncates input to a configurable budget (default 32K tokens of transcript, keeping the most recent). Without this, costs grow unbounded on multi-hour sessions.

### Hard timeout + graceful fallback

The `hero next compact-handoff` MVP command grows a `--with-summary` flag (default true once this ships) that:

1. Spawns the summarizer in a goroutine with a 5s context.
2. If summarizer returns in time → splice its output between the deterministic header and the working-tree footer.
3. If timeout, error, or provider returns nothing → emit skeleton-only handoff with a one-line note (`> LLM summary unavailable (timeout|error|no-provider); using skeleton-only.`).

Compaction is a sensitive UX moment. Never block waiting on summarization.

### hero-cloud endpoint contract

For the `hero-cloud` provider, the HTTP request shape:

```
POST https://api.heroengine.ai/v1/summarize
Authorization: Bearer <hero account token>

{
  "task": "compact-handoff",
  "transcript": "<text>",
  "system_prompt": "<text>",
  "max_tokens": 600,
  "metadata": {
    "session_id": "abc123",
    "spec_slug": "auth-refactor",
    "project_id": "<hashed project identifier>"
  }
}
```

Response:

```
{
  "text": "<summary>",
  "model": "claude-haiku-4-5-20251001",
  "input_tokens": 24000,
  "output_tokens": 480
}
```

Hero-cloud handles provider selection, prompt tuning, A/B testing, and billing internally. The CLI knows none of that. This means hero-cloud can improve handoff quality across all users via server-side changes without binary updates.

**Privacy implication.** Transcripts leave the user's machine via Hero before reaching the LLM provider. This is an additional trust hop relative to the direct path. The MVP `hero hooks install` flow should surface this when picking `hero-cloud` as default ("Hero will send transcripts to api.heroengine.ai for compact handoff summarization. To keep transcripts off Hero servers, set provider=direct and configure an API key.").

### Telemetry (hero-cloud only)

Hero-cloud collects (per the user's account ToS): provider used, latency, input/output token counts, optionally a coarse rating from a follow-up "was this useful?" signal we may add later. Transcripts are NOT retained beyond the call duration except where the user opts in to training data sharing. This needs its own privacy policy / data-handling spec; called out as a dependency below.

## Out of scope

- Replacing the deterministic skeleton — the LLM section is additive.
- Embeddings index for retrieval-augmented summarization. Once [embeddings-index](../embeddings-index/spec.md) lands, the summarizer prompt can be augmented with retrieved chunks — but that's a follow-up, not part of this spec.
- Local model inference. Covered in [local-project-model](../local-project-model/spec.md), exploratory.
- Cross-session memory in hero-cloud (server-side knowledge of past compactions). Privacy-charged; out of scope.
- Adaptive summarization style based on past usefulness. Real feature, but needs feedback signal infrastructure that doesn't exist yet.
- Building hero-cloud itself. This spec assumes hero-cloud exists with a `POST /v1/summarize` endpoint; the dependency is called out below.

## Dependencies

- **[next-compact-handoff](../next-compact-handoff/spec.md)** must ship first. This spec splices content into its envelope.
- **hero-cloud** must have a summarization endpoint deployed and a stable contract (or the `hero-cloud` backend is deferred until it does, and only `direct` ships in the MVP).
- **Privacy / data-handling policy for hero-cloud** must be written and surfaced in the CLI before `hero-cloud` ships as a default backend.
- **`hero account` user-config story** (storing hero-cloud credentials) — likely already exists for peer auth; reuse the same.

## Risks

- **Hero-cloud is the long pole.** If `POST /v1/summarize` doesn't exist yet, the MVP ships with only the `direct` backend and adds `hero-cloud` later. That's fine — the architecture supports it.
- **LLM latency on compaction.** Compaction is already slow. Adding 1–5s for the LLM call extends it. Hard 5s timeout + skeleton fallback is the safety net. Worth measuring perceived UX impact and tuning.
- **Transcript size & cost.** Long transcripts truncated to 32K tokens by default. Cost ~$0.001–0.005 per call for direct/Haiku; hero-cloud absorbs the cost into subscription pricing. Need a per-day / per-session budget enforcement to prevent runaway costs on aberrant projects.
- **Privacy framing for hero-cloud.** Sending transcripts to Hero-operated infra is a real trust ask. Must be opt-in or at least clearly disclosed during `hero hooks install`. Mishandled, this turns a feature into a complaint.
- **Provider lock-in for hero-cloud users.** If users default to hero-cloud and Hero changes prompt/model, results shift under them. Versioned prompts + change-notification ("the compact handoff summarizer was upgraded; recent handoffs will read differently") helps.
- **Direct backend requires user API key.** The noob-prompter persona Hero serves may not have one. That's exactly why hero-cloud matters as a default — direct alone leaves that user with skeleton-only.
- **Summarizer hallucination.** A small model could invent decisions or up-next items not in the transcript. Mitigation: the prompt explicitly says "skip anything that's already recorded as a Decision event" and rewards "nothing notable" — and the deterministic header is always present as the trusted anchor.

## Acceptance criteria

- [ ] `internal/aiservices/` package exists with `Summarizer` interface and `direct` + `heroCloud` implementations.
- [ ] `hero next compact-summarize --transcript <path> --json` invokes the configured provider and returns the three-section summary as JSON.
- [ ] `hero next compact-handoff --json` (extending the MVP command) splices summarizer output into the envelope when available.
- [ ] Provider selection precedence works as specified: explicit config → hero-cloud → direct (anthropic) → direct (openai) → none/skeleton-only.
- [ ] 5s hard timeout. Timeout/error/no-provider path returns skeleton-only handoff with a visible one-line note.
- [ ] Transcript truncation to a configurable budget (default 32K tokens, most-recent kept).
- [ ] `hero.json` config knobs honored: `provider`, `model`, `endpoint`, `timeout_ms`.
- [ ] `hero hooks install --host=claude` surfaces the privacy implication when picking hero-cloud as default.
- [ ] Integration test: stub provider returns canned summary → handoff envelope contains the spliced sections.
- [ ] Integration test: stub provider times out → handoff envelope is skeleton-only with the note line.
- [ ] No-provider path verified: empty env + no hero-cloud config → skeleton-only without crash.

## Changes

- `internal/aiservices/` (new package) — `Summarizer` interface, `direct.go` (Anthropic + OpenAI), `herocloud.go`, `config.go` for provider selection.
- `internal/cli/next_compact_summarize.go` (new) — the standalone summarize subcommand.
- `internal/cli/next_compact_handoff.go` — splice summarizer output into MVP envelope.
- `internal/cli/next.go` — register `nextCompactSummarizeCmd`.
- `internal/config/ai_services.go` (new) — config schema for `ai_services.*` block.
- `internal/cli/hooks.go` — surface privacy disclosure on `--host=claude` install when defaulting to hero-cloud.
- Tests covering provider selection, timeout, truncation, splicing, no-provider fallback.

## Kickoff

Ship the AIServices seam and add an LLM-curated middle to the compact handoff. Start with the `direct` (Anthropic) backend so the feature is usable today; add `hero-cloud` once that endpoint exists.

Read first:
- [next-compact-handoff](../next-compact-handoff/spec.md) — the MVP this extends.
- The current hero-cloud peer manifest (`.hero/peer-manifest.yaml`) to learn what hero-cloud already exposes vs. what needs to be built.
- This spec's "Dependencies" section to understand what must be true before each backend can ship.

Then implement in order:

1. `internal/aiservices/` package skeleton — `Summarizer` interface, config loading, provider-selection precedence.
2. `direct` provider for Anthropic (Haiku). Manual smoke-test against a real transcript.
3. `hero next compact-summarize` subcommand exposing the provider directly (testable in isolation).
4. Splice into `hero next compact-handoff` with the 5s timeout + skeleton fallback.
5. `hero-cloud` provider — gated on the endpoint existing. If it doesn't, document the contract in this spec and ship the rest.
6. Privacy-disclosure surface in `hero hooks install --host=claude` when hero-cloud becomes the default.

The deliverable is reviewable in two cuts: (a) the seam + direct backend (smaller change, immediately useful for users with API keys), and (b) the hero-cloud backend (bigger surface, depends on infra availability). Land (a) standalone if (b) isn't ready.
