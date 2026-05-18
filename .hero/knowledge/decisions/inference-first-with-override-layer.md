---
title: Inference-First With Override Layer — Declarative Files Are Deltas, Not Sources
type: decision
status: proposed
created: 2026-05-18
tags: [inference, declarative, configuration, zero-config, override, principle]
relations:
  - target: project-snapshot
    kind: decided-in
---

# Inference-First With Override Layer — Declarative Files Are Deltas, Not Sources

## Decision

When Hero needs a piece of metadata about a project (the surface
list, the release model, owner assignments, stack composition,
anything that *could* be inferred from the repo), the **default
source is inference from the existing repo**. Declarative files
(`*.yaml`, `hero.json` blocks, frontmatter fields) exist as an
**override layer** that the user creates lazily only when
inference is wrong.

The override file carries **deltas only** — renames, excludes,
additions, per-item field overrides. Never the full set. A fresh
Hero project gets working metadata with no human authoring.

## Context

Earlier project-snapshot drafts treated `.hero/surfaces.yaml` as
the source of truth: users would author the full surface list at
project boot. The user pushed back during design Q&A: "Hero
should infer; the YAML is for fixing mistakes."

This pattern generalized to release modeling (priority chain
with frontmatter / tracker / git tags / graceful hide rather
than a single required field) and surfaces broadly as a
principle worth naming.

## Rationale

Three reasons inference-first wins:

1. **Authoring friction is real.** Asking the user to write a
   full declarative file at project boot is exactly the chore
   that motivated NEXT.md's projected mode. Repeating the
   mistake on a different artifact is unforced.
2. **Inference is high-confidence for existence.** "Which
   surfaces are there" is a structural question — directory
   shape, package manifests, naming hints answer it
   well. "What is this surface's *intent*" is the harder
   semantic question, and that's exactly where overrides earn
   their keep.
3. **Override-as-delta keeps the file tiny.** A delta-only
   file is short, easy to review, easy to keep in sync with
   inference, and contains only the genuinely
   user-contributed knowledge. The full-list-as-source pattern
   forces the file to grow with the project and drift behind
   reality.

## Shape

The pattern, generalized:

```yaml
version: 1
# Inference produces the full set. Edit only when inference is wrong.
renames:           # rewrite an inferred id
  - from: <id>
    to: <id>
ignore:            # drop an inferred item entirely
  - id: <id>
additions:         # declare something inference missed
  - id: <id>
    ...full fields...
overrides:         # win per field for an inferred item
  - id: <id>
    <field>: <value>
```

The projector merges:
**detected ∪ added − ignored**, with overrides winning per field.

## Companion: graceful degradation on missing signal

Inference-first pairs naturally with **graceful degradation**:
when no signal exists for a particular aspect, the artifact
*hides* the unsupported dimension rather than demanding
configuration. Examples from project-snapshot:

- No release signal anywhere → "% to next milestone" column
  hidden + footnote.
- No surface signal at all (impossible in practice but for
  completeness) → single implicit `project` surface covering
  the repo.

The pair "inference + override + graceful degradation" replaces
the older "declare everything up front or it doesn't work"
pattern.

## Tripwires

- A future feature adds a required declarative file at project
  boot. Push back: can inference produce a working default?
- A feature adds a single required field and treats absence as
  an error rather than degrading gracefully. Push back: would
  a priority chain with a graceful hide work better?
- An override file accumulates the *full* entity set rather
  than deltas. Push back: this is the failure mode the pattern
  is designed to avoid.

## Application beyond surfaces

Candidates for inference-first treatment:

- **Stack detection.** Already inferred; codify the override
  shape if a user needs to force a specific stack.
- **Test runner.** Currently inferred per stack; same delta
  pattern would apply if a user needs a custom runner.
- **Initiative grouping.** Currently declared via frontmatter;
  could be inferred from spec naming / tag patterns with
  override for explicit groupings.
- **Owner / RACI.** Inferable from git blame + recent commits;
  override for cases where reality differs from blame.

Each candidate should adopt the pattern when added; the
override-file shape above is the canonical template.
