---
title: "hero verify gates the status flip — agents don't touch status: completed"
type: decision
status: accepted
created: 2026-06-06
related_specs:
  - delivery-gate-enforcement
  - delivery-completion-discipline
---

# hero verify gates the status flip

## Decision

`hero verify <slug>` is the only path to `status: completed`. Agents call
verify; verify checks gates (ledger, audit, coverage, tests); verify flips
the status and archives. Agents do not edit `status: completed` directly.

## Context

The previous design had agents flip status first, then call verify, which
rubber-stamped the archive. This meant verification was cosmetic — the
status was already flipped, so verify had nothing to gate.

The `delivery-completion-discipline` spec (2026-05-22) tried to fix this
with more instructions: tighter ledger rules, mandatory audit, exercise-
the-feature gates. But all enforcement was in agent instructions, not
tooling. Agents can (and do) skip steps, and the CLI doesn't catch it.

## Rationale

Instruction-only fixes for agent-honesty problems are circular. The
tooling must be the enforcement point. By inverting the flow — verify
gates the flip instead of rubber-stamping after it — we make it physically
impossible to mark a spec completed without passing the checks.

## Alternatives considered

1. **`hero_completion_score` MCP tool** — a score derived from a dishonest
   ledger doesn't help. Rejected in favor of independent audit + tooling gate.

2. **Pre-commit hook that checks verify output** — too late in the flow.
   The gate should fire during delivery, not at commit time.

3. **Keep the honor system but add monitoring** — monitoring catches
   problems after the fact. The gate prevents them. Gate > monitor for
   this failure mode.
