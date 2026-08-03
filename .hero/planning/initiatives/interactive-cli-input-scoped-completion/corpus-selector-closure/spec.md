---
title: "Corpus Selector Closure"
slug: corpus-selector-closure
type: feature
status: planning
created: 2026-08-03
domain: engineering
size: medium
priority: high
parent: interactive-cli-input-scoped-completion
depends-on: [prompt-and-tty-contract-closure]
relates-to: [selector-spec-pickers, cli-input-classification]
tags: [cli, selector, prompt, corpus]
---

# Corpus Selector Closure

## Context

The donor branch adds local-corpus selectors but refuses to show them above 25
candidates. Hero itself has far more open specs, so the implementation is safe
from runaway output but does not deliver the original human-facing outcome.

## Goal

Selectively port the original selector targets and make every one useful against
the full local corpus through bounded interaction, without changing explicit,
non-TTY, JSON, or machine-driven command paths.

## Design brief

The `/design` pass must cover exactly these existing targets:

- `score`, `verify`, `spec move`, `supersede`, and `size`
- `skill show`, `skill run`, `skill edit`, `skill rm`, and `skill log`
- `handoff` and `handoff accept`

It must choose the smallest filtering, paging, or equivalent interaction that
keeps more than 25 local candidates reachable without dumping an unbounded
list. It must specify empty, single-item, large-corpus, cancellation,
invalid-choice, explicit-argument, non-TTY, and machine-mode behavior.

## Boundaries

- Do not discover or add selector targets during this work.
- Do not add network-backed selection, including team-user lookup.
- Do not reshape the command tree or absorb `cli-surface-consolidation`.
- Do not build a TUI framework, form engine, fuzzy retrieval subsystem, or
  exact-slug retrieval feature.
- Do not modify connect or setup behavior.

## Acceptance targets

- Every listed target can reach any valid local candidate in a corpus larger
  than 25 entries.
- Cancellation exits non-zero and performs no mutation.
- Empty corpora preserve an actionable error and never show an empty picker.
- Supplied arguments and all non-interactive modes show no picker and do not
  hang.
- The implementation is bounded and testable without new UI infrastructure.

## Kickoff

Design the smallest real-scale selector completion after the prompt foundation
verifies. Keep the target list frozen to the commands named here. Make all
candidates reachable beyond 25 entries and define empty, cancel, invalid,
explicit, non-TTY, JSON, and machine cases. Do not add a TUI framework, network
lookup, new command targets, surface consolidation, or unrelated retrieval and
branch cleanup.

→ `/design corpus-selector-closure`
