# Vocabulary `auto_select` — Authoring Contract

How a `core/vocabularies/<name>.yaml` preset declares the conditions under
which it should be picked automatically as the workspace's active
vocabulary, and how the resolver reads those rules.

Audience: anyone authoring a new vocabulary preset — for a custom
methodology, a tracker integration, or a domain-specific dialect.

Authoritative source: `internal/vocabulary/vocabulary.go`
(`AutoSelectRule` struct, `rawVocabulary.AutoSelect` YAML binding) and
`internal/vocabulary/resolver.go` (`pickName`, `findAutoSelect`). This
doc covers vocabulary auto-select only — methodology selection uses a
different, hardcoded chain (see `active-dialect.md` §3).

## §1. What `auto_select` does

A vocabulary preset's `auto_select` block lets the preset opt in to
being chosen automatically when the workspace has not pinned a
vocabulary explicitly. The block is consulted by `vocabulary.Resolve`
during **step 3 (tracker-inferred)** and **step 4
(delivery-preset-inferred)** of the precedence chain documented in
`active-dialect.md` §3.

The chain a consumer must replicate, in order:

1. Explicit `cfg.Vocabulary`.
2. Methodology-derived (`cfg.Methodology` → its `aligned_vocabulary`).
3. **Tracker-inferred** — match against `auto_select: [{tracker: …}]`.
4. **Delivery-preset-inferred** — match against
   `auto_select: [{delivery_preset: …}]`.
5. `default` preset.

Omitting the `auto_select:` block entirely is how a preset says "never
auto-select me; require an explicit `vocabulary:` setting." That is the
status of `core/vocabularies/default.yaml` today.

## §2. Rule shape

The YAML schema is a list of single-entry maps. Each map is one rule;
each rule has exactly one recognized condition key.

```yaml
auto_select:
  - <key>: <value>
  - <key>: <value>
```

Decoded into Go as `[]AutoSelectRule` where each rule holds a
`Condition map[string]string`. Rule entries with empty maps are dropped
on load (`normalize` in `vocabulary.go`).

**Recognized condition keys** (anything else is silently ignored by
`findAutoSelect`):

| Key | Allowed values | Source of allowed values | Selection step |
|---|---|---|---|
| `tracker` | `github`, `jira`, `linear`, `none` | `TrackerConfig.Type` (`internal/config/config.go`, line 661) | Step 3 |
| `delivery_preset` | `sprint`, `cycle`, `continuous`, `flow` | `PMPresets.Delivery` (`internal/config/config.go`, lines 168–174) | Step 4 |

Notes on allowed values:

- `tracker: none` is technically loadable, but step 3 short-circuits
  when `cfg.Tracker.Type` is empty **or** `"none"`. Declaring
  `tracker: none` will never match.
- For `delivery_preset`, only the four strings above are produced by
  the config layer today. Hero-pm's authoritative list lives in the
  doc comment on `PMPresets.Delivery`. A value like `continuous` is
  accepted by the resolver but no shipped preset declares it (see §6).
- Value matching is **case-insensitive** (`strings.EqualFold` in
  `findAutoSelect`). `tracker: JIRA` and `tracker: jira` are
  equivalent.

## §3. Match semantics

### Within one preset: any rule matches (OR)

Multiple rules in one preset's `auto_select:` list are treated as OR.
The first rule whose key/value matches the workspace's signal wins
selection of that preset. Per the doc comment on `Vocabulary.AutoSelect`:

> any matching rule causes the vocabulary to be selected

So a preset that wants to fire for either a Jira tracker or a
sprint-cadence workspace declares both:

```yaml
auto_select:
  - tracker: jira
  - delivery_preset: sprint
```

### Across the precedence chain: step 3 beats step 4

If a workspace has both a tracker and a delivery preset configured, the
tracker-inferred match (step 3) is checked first. Only if no preset
matches the tracker does the resolver fall through to delivery-preset
matching (step 4). This is hardcoded in `pickName`.

### Across presets: non-deterministic tie-break

If two presets both declare a rule that matches the same workspace
signal, the winner depends on Go map iteration order. `findAutoSelect`
ranges over the `vocabs` map and returns the first match it finds —
the iteration order of a Go map is randomized per process.

This is a real implementation detail, not a guarantee. In practice the
six shipped presets are disjoint (see §6), so collisions only arise
when a custom preset re-declares an existing rule. **Authoring rule:
do not write an `auto_select` rule that duplicates a shipped preset's
condition.** Run `hero status` in a representative workspace and
confirm the expected preset wins before relying on the resolution.

### Never-auto-select

Omit the `auto_select:` block entirely. A preset with no block will
only be chosen when `cfg.Vocabulary` names it explicitly (step 1) or
when a methodology's `aligned_vocabulary:` points at it (step 2). The
`default` preset works this way.

## §4. Worked authoring example

Goal: a `team-foo` preset that auto-selects whenever a workspace uses
GitHub Issues with a shape-up-style `delivery_preset: cycle`.

The file lives at `core/vocabularies/team-foo.yaml`. The `name:` field
must match the filename stem or the loader skips it
(`loadInto` in `vocabulary.go`). `types:` (or `kinds:`) must be
non-empty for the same reason.

```yaml
# core/vocabularies/team-foo.yaml
name: team-foo
display_name: Team Foo Dialect
description: GitHub + 6-week cycles dialect for the Foo team.

auto_select:
  - tracker: github
  - delivery_preset: cycle

types:
  spec: Ticket
  epic: Initiative
  roadmap-item: Bet

kinds:
  spec.bug: Defect
  spec.feature: Enhancement

sections:
  acceptance_criteria: "Done When"
```

With this preset shipped, a `hero.json` of `{}` plus
`{"tracker": {"type": "github"}, "pm": {"presets": {"delivery": "cycle"}}}`
resolves the vocabulary to `team-foo` (step 3, tracker-inferred).
A `hero.json` with `vocabulary: agile-scrum` overrides this and
resolves to `agile-scrum` (step 1, explicit).

Caveat: no shipped preset currently declares
`auto_select: [{tracker: github}]`, so the example above lights up
step 3 cleanly. If a future shipped preset adds `tracker: github`,
the two will collide non-deterministically (see §3) — pick a more
specific condition or remove one of the rules at that point.

## §5. Authoring checklist

Before merging a new preset, check:

- The `name:` field matches the filename stem exactly (loader skips
  the file otherwise).
- `types:` or `kinds:` is non-empty (loader skips empty presets).
- Every `auto_select` rule key is `tracker` or `delivery_preset` —
  any other key is silently ignored at match time.
- Every `auto_select` value is one of the allowed values in §2 for
  that key. Other values load but never match.
- No `auto_select` rule duplicates a condition already declared by a
  shipped preset (see §6) unless you intend to replace it — and even
  then, prefer explicit `vocabulary:` selection over relying on
  map-iteration order.
- A representative workspace resolves to your preset via
  `hero status` (or `vocabulary.Resolve` in tests) before you ship.
- Display map (`types`, `kinds`, `sections`) covers every canonical
  type the preset is expected to render. Fallback is the literal
  type/kind string (`Vocabulary.Display` resolution order in
  `vocabulary.go`).

## §6. Pointers to source and existing presets

Code:

- `internal/vocabulary/vocabulary.go` — `Vocabulary.AutoSelect`,
  `AutoSelectRule`, `rawVocabulary.AutoSelect` YAML binding,
  `normalize` (drops empty-map rules).
- `internal/vocabulary/resolver.go` — `pickName` (precedence chain
  ordering), `findAutoSelect` (per-rule matching, case-insensitive,
  map-iteration tie-break).
- `internal/config/config.go` — `TrackerConfig.Type` (line 661) and
  `PMPresets.Delivery` (lines 168–174) for the canonical value sets.

Shipped presets and their `auto_select` blocks (as of v1):

| Preset | `auto_select` |
|---|---|
| `default` | (none — never auto-selects) |
| `agile-scrum` | `[{delivery_preset: sprint}]` |
| `kanban` | `[{delivery_preset: flow}]` |
| `shape-up` | `[{delivery_preset: cycle}]` |
| `jira` | `[{tracker: jira}]` |
| `linear` | `[{tracker: linear}]` |

No shipped preset declares `tracker: github`, `tracker: none`, or
`delivery_preset: continuous` today. A workspace with
`pm.presets.delivery: continuous` falls through step 4 and lands on
the `default` preset.

See also: `active-dialect.md` for the full resolver chain end-to-end,
including methodology derivation (step 2) and the `vocabulary_overrides`
merge that runs after a preset is chosen.
