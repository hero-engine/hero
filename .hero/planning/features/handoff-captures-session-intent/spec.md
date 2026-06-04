---
title: "Handoff captures session intent, not just the last message"
slug: handoff-captures-session-intent
type: feature
status: planning
priority: high
domain: engineering
created: 2026-06-04
origin: session
size: small
relates-to:
  - next-auto-emit-user-ask
  - resume-brief-surfaces-handoff
  - handoff-one-call-simplification
---

# Handoff captures session intent, not just the last message

## Context

Hero recently shipped auto-emit of `UserAsk` at end-of-turn
(`next-auto-emit-user-ask`). The Stop-hook checkpoint
(`internal/cli/checkpoint.go` `autoEmitUserAsk`) reads the transcript and
records the user's **last** message as the handoff "Last user ask" via
`lastUserAskFromTranscript` (`internal/cli/next_compact_handoff.go`). This
killed handoff drift — the agent no longer has to remember to run
`hero next ask`.

But a content-quality eval this session found a real ceiling. An independent
judge evaluating a *reconstructed* handoff flagged that **the last message is
often a mid-conversation refinement, not the session's headline intent.**

Concrete example from the eval. A session whose goal was *"add rate limiting
to the login endpoint to stop credential-stuffing"* ended with the user
saying *"cap it at 5 a minute and make sure it doesn't blow up under load."*
Auto-emit captured the tail (`cap it at 5 a minute…`) as "Last user ask." A
fresh session reading only that line gets the tuned detail but loses:

- **the WHY** — credential-stuffing defense (the original framing), and
- **the causal thread** — the "doesn't blow up under load" line was a reaction
  to a redis-races discovery mid-session.

Judge verdict: *"capturing only the last message loses the original framing
and the causal thread."*

So the handoff should surface the session's **intent / goal**, not only the
latest user utterance. This is a genuine design question — what is the right
signal for "the goal," and how should the handoff capture and present it —
which is why this is a `/design` spec, not a bug fix.

### What exists today (investigation)

- **`internal/cli/next_compact_handoff.go`**
  - `scanUserAskFromTranscript(path, wantLast bool)` — shared bounded scanner
    (64 KiB / 1000 lines, truncates at `compactHandoffKickoffCap` = 400).
  - `firstUserAskFromTranscript(path)` — returns the **first** user record.
    Already used by `compact-handoff`'s "Original kickoff" section.
  - `lastUserAskFromTranscript(path)` — returns the **last** user record.
    Added by `next-auto-emit-user-ask`; called only by `autoEmitUserAsk`.
- **`internal/cli/checkpoint.go`** — `autoEmitUserAsk(stdin)` is the wire
  point. Best-effort, never fails the Stop hook. Calls
  `lastUserAskFromTranscript`, then `handoff.RecordAsk` on the
  `(user, repo, domain)` singleton.
- **`internal/handoff/handoff.go`** — `UserAsk` is a **singleton per
  `(user, repo, domain)`** keyed by `singletonKey(user, domain)`. There is no
  second variant. `NodeUserAsk`, `RecordAsk`, `LatestAsk` are the surface.
  Critically: **the manual `hero next ask` override and the auto-emit write to
  the *same* singleton row** — they supersede each other, so today there is no
  way to hold "the goal" and "the latest" simultaneously.
- **`internal/cli/next_handoff.go`** — `runNextAsk` is the manual override; it
  calls the same `handoff.RecordAsk` singleton.
- **`internal/projection/user_handoff.go`** — `UserHandoffMD` renders
  `## Last user ask` from `handoff.LatestAsk`, with a `stalenessNote`.
- **`internal/digest/digest.go`** — `handoffSection` ("Where you left off")
  renders `Last ask:` from `handoff.LatestAsk`. This is the resume-brief
  surface (`resume-brief-surfaces-handoff`).
- **`skills/next-handoff-emit/SKILL.md`** — the emission contract; documents
  auto-emit as making `hero next ask` a manual override.

## Goal

A fresh session reading the handoff should know the session's **goal** (the
durable intent — *why* the work is happening) alongside the **latest** user
message (the current refinement). The goal is captured **automatically** — no
new discipline tax — by reading the transcript's opening user intent, and is
surfaced as a distinct line ("Goal:" / "Latest:") in both the digest resume
brief and the per-user handoff file. A manual override remains available as a
quality ceiling but is never required. Done means: in the rate-limiting
example, a cold session sees both *"add rate limiting to the login endpoint to
stop credential-stuffing"* (goal) and *"cap it at 5 a minute…"* (latest).

## Design exploration

The task framed four directions. Weighed honestly:

### Option 1 — First + last (capture both transcript ends)

Capture the **first** user message (`firstUserAskFromTranscript`, already
exists) as the goal and the **last** (`lastUserAskFromTranscript`) as the
latest refinement. Surface both.

- **Pro:** lowest cost. The scanner, both extractors, and the truncation cap
  all exist. The auto path stays fully automatic.
- **Con (the honest hard part):** *"first message of the transcript" is not
  reliably the headline intent.* A Claude Code transcript can:
  - be a **continuation/compact** of a prior session — the first record may be
    injected resume context or a stale opener, not a fresh goal;
  - **span multiple unrelated topics** — the user pivoted twice and the first
    ask is two topics ago;
  - open with **throat-clearing** ("hey", "can you look at this repo?") that
    carries no intent.
  - The bounded 64 KiB read also means "first" is the first user record *in
    the first 64 KiB*, which for a long session is genuinely the opening — good
    — but for a compacted session is whatever the harness wrote first.

  So "first" is a **noisy proxy** for intent. It is right often enough to beat
  "nothing," but it will sometimes surface a stale or off-topic opener as
  "Goal:", which is worse than silence if presented with false confidence.

### Option 2 — Pinned / sticky intent

A goal that **persists across turns** until explicitly changed, distinct from
the volatile per-turn last-ask. Two sub-shapes:
- **2a — same node, different singleton.** Make `hero next ask` (manual) set a
  *sticky* intent on a separate singleton key while auto-emit keeps writing the
  *volatile* latest. Requires distinguishing the two `UserAsk` writers.
- **2b — new node type.** A `SessionGoal` (or `UserGoal`) node, singleton per
  `(user, repo, domain)`, set once and sticky.

- **Pro:** semantically correct — intent *is* sticky; it shouldn't churn every
  turn. Solves the "first message is stale" problem because the goal is
  whatever was last *deliberately* set, not a transcript artifact.
- **Con:** if the sticky goal is **only** set manually, it reintroduces the
  discipline tax the whole `next-auto-emit` effort removed — most sessions
  would have an empty goal. That violates the mission ("the right context …
  without anyone asking"). Sticky is only viable if it *also* has an automatic
  default.

### Option 3 — Distilled intent (model judgment)

The agent emits a one-line distilled goal (e.g. via `hero next ask` with a
paraphrase). Highest quality — the model can read the whole arc and name the
real goal, ignoring throat-clearing and pivots.

- **Pro:** best signal by far. Handles every Option-1 failure mode.
- **Con:** it is exactly the **discipline path** this work is trying to reduce.
  As a *default* it fails the mission. It belongs as an **optional ceiling**,
  not the floor.

### Option 4 — Window / thread (last N user messages)

Capture the last N user messages so the refinement keeps its local context.

- **Pro:** preserves the causal thread better than a single line.
- **Con:** doesn't actually surface the **goal** — it surfaces more of the
  *tail*. In the example, the last 3 messages are all credential-stuffing
  *refinements*; none of them restate the original framing. More tokens, same
  blind spot. Doesn't solve the stated problem.

### Option 5 — Embeddings-based smart selector (local `hero-embed-v1`)

Hero already ships a local 256-dim embedding model (`internal/embeddings/`,
`hero-embed-v1`, `CosineSimilarity`) — semantic similarity/retrieval, **no
generation**. Use it to *select* a better goal message than "first": embed the
session's user messages and pick the one nearest the session centroid (the
most on-theme message), and/or detect a topic *pivot* (a later message far from
the opener → the goal changed mid-session). This is a deterministic,
no-generation **selector** that sits between Option 1 (positional, dumb) and
Option 3 (generative, expensive).

- **Pro:** smarter than first-message at near-zero marginal cost (the model and
  cosine sim already exist); catches the "session pivoted to a new goal" case
  that Option 1 misses; runs locally, no LLM call, no discipline tax.
- **Con — honest scoping:** `hero-embed-v1` is a lightweight averaged-token
  embedding (256-dim, bag-of-tokens), so it resolves *topic-level* similarity,
  not subtle intent nuance. It can pick the most-central or detect a big pivot;
  it will NOT reliably distinguish "the goal" from "a closely-related
  refinement." And it still **selects a verbatim message** — it cannot
  *synthesize* a clean goal line (that's Option 3 only). Adds an embed step to
  the Stop path (bounded, but more than the zero-cost first-message read).
- **Verdict:** a legitimate **optional middle tier**, not the floor. Ship
  Option 1 first; layer Option 5 in if real-world use shows the first-message
  proxy is too noisy. Keep it behind the same toggle as Option 3 (see Tiering).

## Recommended Design

**Primary: Option 1 (first + last) as the automatic floor, with a thin
Option-2 sticky-override hook layered on top — and explicit "possibly the
opener, not a stated goal" honesty in how the goal line is presented.**

Rationale: the mission demands an **automatic default**, which rules out 2-manual
and 3 as the floor. Option 4 doesn't address the goal blind spot. Option 1 is
the only automatic option that surfaces *original framing* at near-zero cost
(both extractors already exist). Its weakness — "first ≠ goal" — is mitigated
two ways: (a) a manual sticky override that, when set, *wins* over the
auto-derived first message; (b) presenting the auto-derived goal with honest
framing so a stale opener is read as "the session opened with…", not asserted
as "the goal is…".

Concretely:

**Capture (where).** In `autoEmitUserAsk` (`checkpoint.go`), in addition to the
existing last-message `RecordAsk`, also read `firstUserAskFromTranscript` and
record it as a **second, distinct singleton: a session-goal node**. Use a new
node type `NodeSessionGoal = "SessionGoal"` in the `handoff` package, singleton
per `(user, repo, domain)` (same keying as `UserAsk`), with a
`source: "auto-first"` prop so the renderer and the override path can tell
auto-derived from manually-set.

- The goal node is written **only if no manually-set goal exists yet for this
  singleton**, OR the existing goal node is also `source: "auto-first"`. This
  makes auto-first **self-refreshing within a session** (each turn re-reads the
  same transcript opener — idempotent via content hash) but **never clobbers a
  manual override**. A `source: "manual"` goal is sticky and survives every
  subsequent checkpoint.
- Guarded exactly like the existing last-ask emit: empty first → no-op; any
  error → stderr + return. Never fails the Stop hook.

**Store (what).** New in `internal/handoff/handoff.go`:
- `NodeSessionGoal = "SessionGoal"` constant.
- `SessionGoal` struct (`User, Domain, Text, Source, SessionID, UpdatedAt`).
- `RecordGoal(store, repoKey, goal)` — upsert singleton; mirrors `RecordAsk`.
  Empty text clears. The `Source` prop ("auto-first" | "manual") is persisted.
- `LatestGoal(store, user, repoKey, domain)` — mirrors `LatestAsk`.
- `RecordGoal` must **not** overwrite a `source:"manual"` row with an
  `source:"auto-first"` write. Enforce in `RecordGoal`: when the incoming
  source is `auto-first`, read the current row; if it exists and is `manual`,
  return nil (no-op). Manual writes always win.

**Override (the ceiling).** Extend `hero next ask`
(`internal/cli/next_handoff.go` `runNextAsk`) — or add a sibling
`hero next goal` — to set the **manual** goal (`source:"manual"`). Decision:
add a dedicated **`hero next goal "<text>"`** subcommand rather than overloading
`ask`, because `ask` is semantically "the latest prompt" and the auto-emit now
owns it; conflating the two would make the override ambiguous. `next goal` with
no arg prints the current goal (read path), matching the `ask`/`suggest`
pattern. Skill doc updates make clear `next goal` is *optional* — fire it only
to correct a wrong auto-derived opener.

**Surface (how it renders).** Two surfaces, both gated on a non-empty goal:

1. **Digest resume brief** — `internal/digest/digest.go` `handoffSection`
   ("Where you left off"). Add a `Goal:` line **above** the existing
   `Last ask:` line, sourced from `handoff.LatestGoal`. Relabel the existing
   line `Latest:` (it currently reads `Last ask:`) so the pair reads as
   goal-then-refinement. When the goal node is `source:"auto-first"`, prefix
   the value with a soft framing so a stale opener isn't asserted as fact —
   e.g. `Goal (session opened with): …`. When `source:"manual"`, render plain
   `Goal: …`. Suppress the `Goal:` line entirely when the goal text equals the
   latest text (single-message sessions) to avoid a duplicate.

2. **Per-user handoff file** — `internal/projection/user_handoff.go`
   `UserHandoffMD`. Add a `## Session goal` section **above** `## Last user
   ask`, sourced from `handoff.LatestGoal`, with the same auto-vs-manual
   framing. Reuse the existing `stalenessNote` so a goal older than recent
   commits is flagged. Keep `## Last user ask` exactly as-is (it now reads as
   the refinement). Omit the goal section when empty or equal to the last ask.

**Why a new node type and not a second `UserAsk` variant.** `UserAsk` is a
strict singleton — auto-emit and manual `ask` already collide on it by design
(latest wins). Reusing it for the goal would force goal and latest to
supersede each other, which is the exact bug. A distinct `SessionGoal`
singleton is the minimal structural change that lets both coexist, and it
reuses every pattern in the `handoff` package (keying, upsert, scan, render).

### Honesty about what this does and doesn't fix

- It **reliably** surfaces the original framing when the session opened with a
  clear intent and stayed on-topic — the common case, and the eval's case.
- It **degrades gracefully** when "first" is noise: the soft framing
  ("session opened with") prevents asserting a wrong goal, and the manual
  `next goal` override is a one-liner fix. It does **not** magically detect
  pivots or compaction-injected openers — that needs Option 3 (model
  judgment), which stays optional.
- It does **not** distill or summarize. The goal line is a verbatim (truncated)
  user message, same as the last-ask line.

### Tiering decision (maintainer, 2026-06-04)

Goal capture is a **three-tier ladder**; this spec delivers Tier 1 now and
leaves the higher tiers as opt-in follow-ups the user can choose to enable:

| Tier | What | Cost | Status |
|------|------|------|--------|
| **1 — first-message** | Auto-capture the transcript opener as `SessionGoal` (Option 1) | zero (extractor exists) | **This spec — ships now, default-on** |
| **2 — embeddings select** | Pick the most-on-theme message / detect pivots via local `hero-embed-v1` (Option 5) | low (local embed, no LLM) | Optional follow-up, **user-toggleable** |
| **3 — model-distilled** | Agent emits a synthesized one-line goal (Option 3) | discipline/LLM step | Optional ceiling, **user-toggleable** |

The maintainer's call: **ship Tier 1, and make Tiers 2/3 a setting the user
opts into** (e.g. `next.goal_capture: "first" | "embed" | "distill"` in
`hero.json`, default `"first"`) rather than forcing the higher-cost paths on
everyone. The sticky `hero next goal` manual override (in the Recommended
Design) works at every tier. Tiers 2 and 3 are **out of scope for this spec** —
file them as follow-ups (`handoff-goal-embed-selector`,
`handoff-goal-model-distill`) if/when Tier 1's first-message proxy proves too
noisy in real use. The config knob (the `goal_capture` mode field + its default)
*should* be introduced by this spec even though only `"first"` is implemented,
so enabling a higher tier later is a config flip, not a schema change.

## Acceptance Criteria

- WHEN the Stop-hook checkpoint runs and the transcript has a readable first
  user message THE SYSTEM SHALL record a `SessionGoal` singleton with
  `source: "auto-first"` for the `(user, repo, domain)` triple.
- WHEN a `SessionGoal` with `source: "manual"` already exists THE SYSTEM SHALL
  NOT overwrite it with an auto-first write.
- WHEN the user runs `hero next goal "<text>"` THE SYSTEM SHALL record a
  `SessionGoal` with `source: "manual"` that supersedes any auto-first goal.
- WHEN `hero next goal` is run with no argument THE SYSTEM SHALL print the
  current session goal (or a "none recorded" hint).
- WHEN the digest resume brief renders and a session goal exists THE SYSTEM
  SHALL show a `Goal:` line above the latest-ask line in "Where you left off".
- WHERE the session goal is `source: "auto-first"` THE SYSTEM SHALL frame the
  goal line as the session opener (e.g. "session opened with") rather than
  asserting it as a stated goal.
- IF the goal text equals the latest-ask text THEN THE SYSTEM SHALL omit the
  goal line to avoid duplication.
- WHEN the per-user handoff file is projected and a goal exists THE SYSTEM
  SHALL render a `## Session goal` section above `## Last user ask`.
- IF the transcript is missing, unreadable, or has no first user message THEN
  THE SYSTEM SHALL leave any existing goal untouched and never fail the Stop
  hook.
- THE SYSTEM SHALL keep the existing last-ask auto-emit behavior unchanged.

## Approach / Changes

1. **`internal/handoff/handoff.go`** — add the `SessionGoal` node type.
   - Add `NodeSessionGoal = "SessionGoal"` to the node-type constants.
   - Add `SessionGoal` struct: `User, Domain, Text, Source, SessionID,
     UpdatedAt`.
   - Add `RecordGoal(store, repoKey, SessionGoal)` — mirror `RecordAsk`:
     singleton key `singletonKey(user, domain)`, empty text clears via
     `InvalidateNode`. Persist `source` in props. **Guard:** when incoming
     `Source == "auto-first"`, read the current row via
     `scanLatestSingleton`; if it exists and its `source` prop is `"manual"`,
     return nil (no clobber).
   - Add `LatestGoal(store, user, repoKey, domain)` — mirror `LatestAsk`;
     add `goalFromNode` to populate `Source` from props.
   - Content hash includes text + source so an auto-first re-write with the
     same opener is a no-op (no churn).

2. **`internal/cli/checkpoint.go`** — extend `autoEmitUserAsk`.
   - After the existing `lastUserAskFromTranscript` / `RecordAsk` block, read
     `first := firstUserAskFromTranscript(ctx.TranscriptPath)`.
   - If non-empty, call `handoff.RecordGoal(store, repoKey, handoff.SessionGoal{
     User: user, Domain: domain, Text: first, Source: "auto-first",
     SessionID: ctx.SessionID})`. Same best-effort error handling (stderr +
     continue). Reuse the already-open `store`, `user`, `repoKey`, `domain`.

3. **`internal/cli/next_handoff.go`** — add `hero next goal`.
   - Add `nextGoalCmd` (Use: `goal [text]`), register it in `next.go`'s
     `AddCommand` and in the `init()` flag loop alongside ask/suggest.
   - `runNextGoal`: with args → `handoff.RecordGoal(..., Source: "manual")`;
     no args → `handoff.LatestGoal` read path via `emitField` (mirror
     `runNextAsk`).

4. **`internal/cli/next.go`** — register `nextGoalCmd` (line ~82, alongside
   `nextAskCmd`).

5. **`internal/projection/user_handoff.go`** — add the `## Session goal`
   section in `UserHandoffMD`, above `## Last user ask`. Source from
   `handoff.LatestGoal`. Apply auto-vs-manual framing; reuse `stalenessNote`;
   omit when empty or equal to the last ask. Add a `## Session goal` parse case
   to the round-trip ingest (`handoff.IngestUserFile`) so the goal survives
   cross-machine push/pull like the other singletons.

6. **`internal/digest/digest.go`** — in `handoffSection`, prepend a `Goal:`
   line from `handoff.LatestGoal` (best-effort, same `logSkip` pattern),
   relabel `Last ask:` → `Latest:`, apply the auto/manual framing, and the
   equal-text suppression.

7. **`skills/next-handoff-emit/SKILL.md`** — document the auto-emitted session
   goal and the optional `hero next goal` override (when to fire: only to
   correct a wrong auto-derived opener). Add it to the cadence table and the
   "reading state" list. Keep the framing: goal is automatic, override is a
   ceiling.

## Test Plan

- **`internal/handoff/handoff_test.go`**
  - `RecordGoal` + `LatestGoal` round-trip; empty text clears.
  - Auto-first does **not** clobber an existing manual goal; manual **does**
    supersede an auto-first goal.
  - Singleton supersession: two auto-first writes with different text → latest
    wins; same text → no new row (content-hash stable).
- **`internal/cli/checkpoint_test.go`** (mirror the existing auto-emit tests)
  - Stop-hook payload with a multi-message transcript records both the
    last-ask `UserAsk` and the first-message `SessionGoal`.
  - Missing/empty/malformed transcript leaves goal untouched, exits 0.
  - A pre-existing manual goal is preserved across a checkpoint.
- **`internal/cli/next_handoff_test.go`** — `hero next goal "x"` sets manual
  goal; bare `hero next goal` prints it; read path agrees with `LatestGoal`.
- **`internal/projection/user_handoff_test.go`** — `## Session goal` renders
  above `## Last user ask`; auto-first framing vs manual framing; section
  omitted when empty or equal to last ask; staleness note appears.
- **`internal/digest/digest_test.go`** — `Goal:` line renders above `Latest:`
  in "Where you left off"; auto framing; equal-text suppression; empty goal →
  no Goal line; `User==""` → section unaffected.
- **End-to-end (eval regression):** feed a fixture transcript matching the
  rate-limiting example; assert the rendered handoff contains both the
  credential-stuffing opener (as Goal) and the "cap it at 5 a minute" tail (as
  Latest). This is the guardrail the eval finding demands.

## Boundaries

- **Out of scope: model-distilled intent (Option 3).** The agent emitting a
  hand-crafted one-line goal stays an **optional ceiling** via `hero next goal`
  — it is not wired as an automatic default and no skill change *requires* it.
- **Out of scope: pivot / topic-shift detection.** Detecting that the user
  changed topics mid-session (so "first" is stale) needs model judgment; the
  soft framing + manual override are the v1 mitigation. No heuristic
  topic-segmentation.
- **Out of scope: the last-N-messages window (Option 4).** Not pursued; it
  doesn't surface the goal.
- **Out of scope: compaction-aware first-message selection.** We do not try to
  detect and skip harness-injected resume context at the head of a compacted
  transcript. Documented as a known limitation; the override covers it.
- **Out of scope: changing `UserAsk` semantics or the existing auto-emit.**
  Last-ask behavior is untouched; the goal is purely additive.
- **Out of scope: Cloud/federation-specific behavior.** `SessionGoal` rides the
  same graph + round-trip-ingest path as the other handoff singletons; no new
  sync work.

## Risks

- **"First message ≠ goal" (the core risk).** Mitigated by (a) soft framing
  ("session opened with") so the line is never asserted as a definitive goal,
  and (b) the manual `hero next goal` override. Residual: a confidently-wrong
  opener can still mildly mislead. Acceptable — strictly better than the
  current state (no goal at all), and the framing caps the downside.
- **Multi-session / compacted transcripts.** The first record in a compacted
  transcript may be injected resume context, not a user goal. The 64 KiB
  bounded read means "first" is "first in the window." Mitigation: the
  extractor already filters to `role == user` records; harness-injected
  *system/context* blocks are skipped. A user-role resume echo could still slip
  through — covered by the override and flagged as a known limitation.
- **Intent is fuzzy.** "The goal" is inherently judgment. We deliberately pick a
  cheap, automatic proxy (first user message) over an absent or
  discipline-gated one, and provide the manual ceiling for when the proxy is
  wrong. We do not claim to *infer* intent.
- **Singleton collision regression.** The whole design hinges on `SessionGoal`
  being a *separate* singleton from `UserAsk`. A keying bug would silently make
  goal and latest clobber each other. The handoff unit tests for non-clobber
  and supersession are the guard.
- **Surface duplication.** If goal == latest (single-message session), naive
  rendering shows the same line twice. The equal-text suppression AC + tests
  guard this.
- **Stop-hook cost.** Adds one extra transcript scan (`first`) and one extra
  upsert per checkpoint. The scan is already bounded and the buffer is already
  read once; cost is negligible and fully inside the existing best-effort
  guard.

## Kickoff

Captures the session's GOAL (first user message) alongside the latest ask, so
a fresh session knows the *why*, not just the last refinement.

**Status:** planning — spec just landed, no code yet.

**Pick up at:** start in `internal/handoff/handoff.go` — add the `SessionGoal`
node type (`RecordGoal`/`LatestGoal`, singleton per user/repo/domain) with the
auto-first-never-clobbers-manual guard. Then wire `firstUserAskFromTranscript`
into `autoEmitUserAsk` (`checkpoint.go`). Surfaces (digest + user_handoff)
come after the capture+store path is green.

→ `.hero/planning/features/handoff-captures-session-intent/spec.md`

**Files:** `internal/handoff/handoff.go`, `internal/cli/checkpoint.go:100`, `internal/cli/next_compact_handoff.go:580`, `internal/projection/user_handoff.go:64`, `internal/digest/digest.go:313`
**Skip:** model-distilled goal as the default (stays optional via `hero next goal`); last-N-window (Option 4 — doesn't surface the goal); reusing the `UserAsk` singleton (goal and latest must not clobber each other).
