# What Is Hero?

Hero is a project memory and delivery system for AI-assisted engineering.

## Project memory is the headline

AI coding sessions routinely lose why a decision was made, which correction
matters, what already failed, and what evidence established the current state.
Hero keeps those artifacts in a repository-centered `.hero/` corpus that later
sessions can search, retrieve, and review.

The shipped memory system preserves intent, decisions, corrections,
conventions, evidence, failures, and current state. It requires a Hero workspace
and a supported harness or CLI entry point. It is project-owned context, not a
claim that a model remembers everything.

## Verified delivery is the execution layer

Hero's own spec system turns intent into bounded work. Specialized agents use
project context while implementing. Completion requires a Completion Ledger, a
fresh cold audit, and the configured build/test gates before
`hero spec verify <slug>` closes and archives the spec.

The delivery system is shipped with the Engineering setup. Agent execution
depends on the active harness and the project must provide testable validation.
Hero does not replace the coding agent that writes the code.

## How the systems reinforce one another

Memory informs design and delivery. Delivery can produce decisions, evidence,
corrections, and a current handoff for future sessions. The individual
components ship; the full repeatable cross-tool continuity outcome remains
**preview** until its public proof is complete.

## Harness-native workflows

Hero installs into the surfaces each target actually supports. Claude receives
command files; Codex and Grok receive command skills; other targets receive
their supported native workflow surfaces and root instructions. Natural
language routing works through those installed instructions. Do not assume a
workflow name is also a terminal subcommand.

Supported install targets are OpenCode, Cursor, Claude, GitHub Copilot, Codex,
Generic MCP, and Grok. Run `hero install --help` for the current target list.

## Scope and boundaries

- Hero's shipped delivery workflows use Hero specs. GitHub Spec Kit, OpenSpec,
  and other external spec providers are future direction, not shipped adapters.
- The default setup is Core plus Engineering, including lightweight PM and QA
  assistance used within coding workflows. Focused PM, QA, and Sales setups are
  optional and maturity-bounded.
- Hero is local-first. Optional trackers, code hosts, peering, team mode, and
  headless execution require their stated setup and consent.
- Hero Code and Hero Cloud are separate proprietary products. This repository
  does not contain either product's implementation.

## Next

- [Project memory](concepts/knowledge-base.md)
- [Verified delivery](concepts/core-loop.md)
- [Capability status and evidence](reference/capability-status.md)
