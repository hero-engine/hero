---
title: Hero Agent Run Validation — First-Use Smoke Test for Headless Delivery
slug: hero-agent-run-validation
type: feature
status: planning
tags: [validation, smoke, agent-run, async-delivery, dogfood]
created: 2026-04-30
relations:
  - target: async-delivery
    kind: validates
horizon: now
---

## Kickoff

First-time end-to-end smoke of `hero agent run deliver <slug>` against a real spec, in a real worktree, with real API credentials. Confirms async-delivery works as advertised before relying on it for bigger work.

**Status:** planning — written 2026-04-30, awaiting credential setup and a test slot.

**Pick up at:** ensure `ANTHROPIC_API_KEY` is set (or `hero login` completed), pick a small spec from `hero queue`, run with `--budget 2` first, observe.

→ `hero agent run deliver <small-spec-slug> --budget 2`

**Files:** [.hero/specs/async-delivery/spec.md](.hero/specs/async-delivery/spec.md), [internal/cli/agent_run.go](internal/cli/agent_run.go) (if exists), `hero agent run --help`

## Goal

Run `hero agent run deliver` end-to-end once, on a deliberately small spec, in a controlled budget, and confirm:

- It authenticates and connects to the LLM provider.
- It reads the spec, executes the deliver loop, and produces real file mutations.
- It commits its work (or surfaces what would be committed).
- The output is reviewable — logs, transcript, cost tally, exit status — without having sat through it live.

This is a *validation*, not a feature. The deliverable is **confidence and a runbook entry**, not new code.

## Problem

`async-delivery` shipped (status: completed). I have not yet personally fired `hero agent run` against a real spec end-to-end. Until I do, every "→ Want me to fire this headless via `hero agent run`?" offer in the recap rule is theoretical. The first real run is a discovery exercise: what does the output look like, what does failure look like, where do branches/PRs land, what does cost feel like?

Doing this on a small spec (one with narrow Changes, clear AC, low blast radius) buys the lessons cheap. Doing it on `kickoff-prompts-queue` first would conflate "is `agent run` working?" with "is this big spec deliverable headless?" — bad signal.

## Design

### Pre-flight checklist

1. Credentials available — either `export ANTHROPIC_API_KEY=…` or `hero login` completed.
2. Worktree clean (no uncommitted work that could be clobbered).
3. Branch is one I'm okay with the agent committing to (or a fresh branch).
4. `hero queue` shows at least one ready spec that's small.
5. Budget cap set: `--budget 2` (≈2 USD; raise only after the first run lands).

### Pick the test spec

Criteria for the first-run target:

- **Status `planning` or `delivering`** — something the agent can actually pick up.
- **Narrow `## Changes` list** — 1–3 files modified, no cross-cutting refactors.
- **Clear EARS-style acceptance criteria** — agent has unambiguous targets.
- **Low blast radius** — touching docs / skills / a single CLI subcommand, not core graph code.
- **Self-contained** — no waiting on humans, no external dependencies.

Candidates to scan for: any spec where the design is fully nailed, AC is concrete, and the implementation is essentially mechanical. If none of the current 49 planning specs fit, write a tiny throwaway spec for the test (e.g., "add `hero version --json` flag").

### Run command

```
hero agent run deliver <slug> --budget 2 --max-turns 30
```

`--max-turns 30` is well under the default 100 and prevents runaway loops on unexpected dead-ends. Bump up after the first successful run.

Optional: `--dry-run` first to confirm the spec parses and the system prompt assembles cleanly (already validated for `kickoff-prompts-queue`; redo for the chosen test spec).

### What to observe

- **Authentication** — does the run start, or fail with a key error like the first attempt did?
- **Tool registrations** — `hero agent run --dry-run` reports tool count. Watch how the agent uses them.
- **File mutations** — at run end, `git status` and `git log` show what landed.
- **Branch/PR** — does the agent push to a branch and open a PR? Or does it just leave commits on the current branch? (`async-delivery` mentions PRs; confirm the actual behavior.)
- **Cost** — final token / dollar tally vs. the $2 cap.
- **Failure modes** — if it gets stuck, what does it do? Loop? Exit? Surface the error?
- **Transcript / log** — can I read what it did after the fact, or only watch live?

### Capture findings

After the run:

- Append a short note to `.hero/knowledge/notes/hero-agent-run-first-use.md` with: chosen spec, command run, duration, cost, what worked, what surprised, what to fix in `async-delivery` follow-ups.
- If meaningful gaps surface (e.g., transcript missing, cost surprises), open follow-up specs (`/design`).
- If the run succeeded cleanly, this spec flips to `completed` and the recap rule's "→ fire headless via `hero agent run`" offer becomes lived experience, not theory.

### What's explicitly NOT in this spec

- **Not a feature.** No code changes. This is a runbook + validation pass.
- **Not a benchmark / load test.** One spec, one run, modest budget. Performance/scale belongs to a separate effort.
- **Not building tooling around `agent run`.** Observations may surface bugs to fix; those become their own specs.
- **Not gated on `kickoff-prompts-queue`** — the validation should land before that big spec ships, not depend on it.

## Changes

- `.hero/knowledge/notes/hero-agent-run-first-use.md` (new) — written *after* the run with the captured findings. Not pre-written.
- (Conditional) Follow-up bug/feature specs for any gaps surfaced.
- This spec flips `status: planning` → `completed` once the run lands and findings are captured.

## Acceptance Criteria

- WHEN `ANTHROPIC_API_KEY` is set or `hero login` is completed AND `hero agent run deliver <chosen-slug> --budget 2 --max-turns 30` is invoked THE SYSTEM SHALL run the agent loop to completion or to a clean error.
- THE SYSTEM SHALL not exceed the $2 budget cap during the test run.
- THE SYSTEM SHALL produce reviewable output (stdout log, file mutations, commit history, cost tally) sufficient to assess success without having watched live.
- WHEN the run completes successfully THE SYSTEM SHALL leave the chosen spec's acceptance criteria substantially advanced (status changed, code changes landed, or both).
- WHEN the run completes (success or failure) THE SYSTEM SHALL produce a knowledge note at `.hero/knowledge/notes/hero-agent-run-first-use.md` capturing what worked, what surprised, and concrete follow-ups.
- IF the chosen test spec is too ambitious or no suitable candidate exists THEN THE SYSTEM SHALL substitute a deliberately small throwaway spec for the validation rather than skip the run.
- IF the run reveals a bug in `hero agent run` or `async-delivery` THEN THE SYSTEM SHALL open a follow-up bug spec via `/diagnose` rather than fix inline.

## Boundaries

- Does **not** validate every `hero agent run` flag combination — single-spec, single-run, baseline only.
- Does **not** validate `--all` or `--match` batch modes — those are separate validation passes.
- Does **not** test failure-recovery, retry, or partial-state cleanup behaviors beyond what naturally surfaces.
- Does **not** validate cloud / remote agent execution — only the local `hero agent run` path.
- Does **not** block delivery of `kickoff-prompts-queue` or `agent-end-of-turn-recap`. They proceed interactively if this validation hasn't run yet.
- Does **not** require the chosen test spec to be one of the user's "important" specs — pick something the user is okay with the agent touching cold.

## Mission Fit

> "Does this make the next agent session start smarter than the last one ended — and does it raise the floor for everyone, not just the senior dev who already knows what to ask?"

Indirect fit. This spec is a *validation* of an existing capability (`async-delivery`), not a new mission-direct artifact. But the lived experience it produces directly enables the recap rule's headless-spin-off offer, which *does* raise the floor: a non-expert who hits "yes" on a spin-off offer needs the underlying machinery to work first time. Validating that here is what makes the recap rule's promise real.
