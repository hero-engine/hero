---
title: "On-Demand Feature Synthesizer — hero synthesize <slugs>"
type: feature
status: planning
slug: fks-on-demand-synthesizer
domain: engineering
parent: feature-knowledge-synthesis
priority: medium
size: medium
created: 2026-06-23
tags: [knowledge, synthesis, cli, mcp, generation]
kind: new
relations:
  - target: fks-feature-knowledge-artifact
    kind: depends-on
---

# On-Demand Feature Synthesizer

## Goal

Ship the synthesis engine: given an explicit set of spec slugs, read them and the
git diff across their delivery window and emit one `explainer` knowledge
entry in the `fks-feature-knowledge-artifact` shape. Exposed as
`hero synthesize <slugs…>` and an MCP tool. This is the spine of the initiative —
manually invoked, zero detection risk — and it exists first so we learn what a
*good* feature explainer looks like before automating the trigger (#3/#4) or the
upkeep (#5).

## Kickoff

Build `hero synthesize <slug…>`: read the named specs + the git diff over their
delivery window, synthesize one feature knowledge entry in the
`fks-feature-knowledge-artifact` template, and write it to
`.hero/knowledge/explainers/<slug>/spec.md` with provenance frontmatter
(`synthesized_from`, `last_synthesized`). Add the matching MCP tool. Then run
`hero index`. Depends on `fks-feature-knowledge-artifact` landing the artifact
contract. Parent initiative: `feature-knowledge-synthesis`. Look at how existing
knowledge writes happen (`hero note`, `auto-knowledge-capture` skill) and how
delivery windows are derived from spec timestamps + git history.

## Problem

The originating pain — "a feature shipped across many specs and there's no
how-it-works summary" — is solvable *today* for any known feature if we have an
engine that turns a spec cluster into an explainer. Building that engine first,
on an explicit slug list, has two payoffs:

1. **Immediate value, zero risk.** A human who knows a feature shipped points the
   tool at its specs and gets the doc — no boundary inference, no false
   positives.
2. **It de-risks everything downstream.** Cluster detection (#3) and the trust
   handshake (#4) only matter if the generated doc is good. We can't calibrate
   "good" without the generator. Build it, run it on real shipped features (e.g.
   `cold-start-trust-hardening`), and tune the template/prompt against reality.

## Design

### Surface
- `hero synthesize <slug> [<slug> …]` — explicit spec slugs.
- Optional `--out <path>` override; default `.hero/knowledge/explainers/<slug>/spec.md`
  where `<slug>` derives from the dominant spec / initiative slug.
- MCP tool mirroring the CLI for in-session synthesis.

### Inputs the engine assembles
1. The full text of each named spec (via the spec reader).
2. The **git diff across the delivery window** — bounded by the specs'
   created/completed timestamps and the commits referencing them — so the
   explainer reflects what was actually built, not just what was planned.
3. Existing `decision` entries referenced by the specs, for the **Related
   decisions** links (link, don't restate).

### Output
- One `explainer` knowledge entry in the #1 template, with `synthesized_from`
  set to the input slugs and `last_synthesized` to today.
- Followed by `hero index` so it's immediately searchable.

### What this spec does NOT decide
- *Which* specs form a cluster — caller supplies them explicitly. (#3 infers.)
- Whether to auto-run — always manual here. (#4 automates.)
- How to update an existing entry — out of scope; first write only. (#5 amends.)

## Acceptance Criteria

- WHEN a user runs `hero synthesize <slug…>` with valid spec slugs THE SYSTEM
  SHALL write one `explainer` knowledge entry in the
  `fks-feature-knowledge-artifact` template populated from those specs.
- THE SYSTEM SHALL set `synthesized_from` to the input slugs and
  `last_synthesized` to the run date in the entry's frontmatter.
- WHEN synthesizing THE SYSTEM SHALL incorporate the git diff across the specs'
  delivery window, not the spec prose alone, so the explainer reflects shipped
  code.
- WHERE a named spec references existing `decision` entries THE SYSTEM SHALL link
  them in **Related decisions** rather than restating their content.
- IF a supplied slug resolves to no spec THEN THE SYSTEM SHALL fail loud, naming
  the unresolved slug, and write nothing.
- WHEN synthesis completes THE SYSTEM SHALL run `hero index` so the entry is
  searchable, and print the written path.
- THE SYSTEM SHALL expose an MCP tool equivalent to the CLI command.

## Boundaries / Out of scope

- No cluster inference, no auto-trigger, no amendment of existing entries.
- Overwriting an existing entry at the target path: out of scope here (first-write
  semantics); #5 owns update/amend.
- No trust handshake — explicit invocation is the authorization.

## Dependencies

- **Depends on `fks-feature-knowledge-artifact`** — writes into that artifact's
  template and frontmatter contract.
- Downstream: `fks-cluster-detection` (#3) calls this engine with inferred slug
  sets; `fks-living-doc-amendment` (#5) extends it from first-write to amend.

## Notes

Use a real shipped feature as the calibration target — `cold-start-trust-
hardening` (10 child specs, rich progress log) is an ideal first synthesis to
judge output quality against.
