---
title: "Synthesis Trust Handshake — Auto / Review / Off Autonomy"
type: feature
status: completed
slug: fks-trust-handshake
domain: engineering
parent: feature-knowledge-synthesis
priority: medium
size: medium
created: 2026-06-23
completed_at: 2026-06-23
tags: [knowledge, synthesis, autonomy, config, trust, explainer]
kind: new
relations:
  - target: fks-cluster-detection
    kind: depends-on
---

# Synthesis Trust Handshake

## Goal

Turn detected candidates (#3) into the autonomy the user asked for: a persisted
`auto | review | off` setting that decides whether a high-confidence cluster is
synthesized automatically, proposed for review, or ignored — with **low-confidence
candidates always routed through review regardless of mode**, so detection false
positives are never silently written. This is what makes auto-generation safe.

## Kickoff

Add a synthesis-autonomy setting to `hero.json` (`knowledge.explainer_synthesis:
auto | review | off`, default `review`, mirroring `knowledge.auto_capture`). Add a
policy that maps (candidate confidence, mode) → auto / review / skip, where
sub-threshold confidence forces review even in `auto`. Wire it into `hero
synthesize --detect` (annotate each candidate with its action) and add
`hero synthesize --auto` (synthesize the auto-eligible candidates, list the rest)
and `hero synthesize --set-mode <auto|review|off>` (persist the user's handshake
answer). Depends on #3. Parent: feature-knowledge-synthesis.

## Problem

#3 emits candidates; #2 synthesizes a named cluster. Nothing decides *whether* to
act on a candidate without a human naming it. The user's stated want (the
`/discover` session) was: generation happens automatically, but with a one-keystroke
handshake — "feature X shipped; keep doing this automatically / review each / turn
it off" — and the answer sticks. Two things make this safe rather than reckless:

1. **A persisted mode**, so the choice is made once, not re-litigated every time.
2. **A confidence floor on automation**: inferred clusters carry false-positive
   risk (e.g. the hub-file blob #3 had to guard against), so anything below a
   high-confidence bar is *always* proposed for review, even when the mode is
   `auto`. Only the safest candidates (completed-initiative children, strong
   inferred clusters) auto-synthesize.

## Design

### Setting
`knowledge.explainer_synthesis` in `hero.json`: `auto | review | off`, default
**`review`** (conservative — propose, don't write, until the user opts into auto).

### Policy
`Action(confidence, mode)`:
- `off` → **skip** everything.
- `review` → **review** everything (propose, never auto-write).
- `auto` → **auto** when `confidence >= autoConfidenceBar` (high-confidence only),
  else **review**. The bar keeps risky inferred clusters out of silent writes.

### Surfaces
- `hero synthesize --detect` annotates each candidate with its action under the
  current mode (`AUTO` / `REVIEW` / `SKIP`).
- `hero synthesize --auto` synthesizes the `AUTO` candidates (via #2's engine /
  scaffold) and lists the `REVIEW` ones for confirmation. The headless entry point
  for the eventual completion-gate trigger.
- `hero synthesize --set-mode <auto|review|off>` persists the user's handshake
  answer to `hero.json`.

### The handshake itself
The conversational prompt ("keep auto / review each / off") is an **agent
behavior**, documented for the harness: on detecting candidates, the agent shows
them and offers the three choices, then records the answer with `--set-mode`. The
CLI provides the policy, the annotation, and the persistence; the agent provides
the conversation. (Mirrors how `/retro` + `auto_capture` split machine policy from
agent action.)

## Acceptance Criteria

- THE SYSTEM SHALL read `knowledge.explainer_synthesis` (`auto|review|off`,
  default `review`) from `hero.json`.
- WHERE the mode is `off` THE SYSTEM SHALL skip all candidates.
- WHERE the mode is `review` THE SYSTEM SHALL mark every candidate for review and
  never auto-write.
- WHERE the mode is `auto` THE SYSTEM SHALL auto-synthesize only candidates at or
  above the confidence bar, and route the rest to review.
- WHEN a user runs `hero synthesize --set-mode <mode>` THE SYSTEM SHALL persist
  the mode to `hero.json` and confirm it.
- WHEN a user runs `hero synthesize --detect` THE SYSTEM SHALL annotate each
  candidate with its action under the current mode.
- WHEN a user runs `hero synthesize --auto` THE SYSTEM SHALL synthesize the
  auto-eligible candidates and list the review candidates without writing them.

## Boundaries / Out of scope

- No automatic firing on the initiative-completion gate yet — `--auto` is the
  manual/headless entry point; wiring it to the lifecycle transition can follow.
- No amendment of existing explainers — that's #5.
- The interactive prompt UI is an agent behavior; this spec ships the policy +
  persistence + surfaces it reads.

## Dependencies

- **Depends on `fks-cluster-detection` (#3)** — consumes its candidates.
- Uses #2's synthesis engine for the `--auto` path.

## Delivery notes

- 2026-06-23 — Delivered. `knowledge.explainer_synthesis` config field
  (`auto|review|off`, default `review`) on `KnowledgeConfig`. Policy in
  `internal/synthesize/policy.go`: `NormalizeMode` + `Action(confidence, mode)`
  with `autoConfidenceBar = 0.9` (sub-bar always routes to review, even in auto).
  CLI: `hero synthesize --set-mode <mode>` persists to hero.json;
  `--detect` annotates each candidate with `[AUTO|REVIEW|SKIP]` under the mode;
  `--auto` synthesizes the auto-eligible clusters and lists the review ones.
  Verified e2e: default review marks all REVIEW; auto promotes high-confidence
  (95%) candidates to AUTO. Policy + config tests green.
- The interactive "keep auto / review / off" prompt is the agent's job (it shows
  candidates and records the answer via `--set-mode`); the CLI ships the policy +
  persistence, mirroring the auto_capture split.
