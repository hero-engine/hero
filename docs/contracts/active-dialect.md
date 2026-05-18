# Active Workspace Dialect — Read Contract

How a consumer (the hero-code Rust dashboard, an external integration, or another Hero client) reads the **active dialect** of a workspace — the vocabulary that controls display names and the methodology that controls behavior — and renders artifact names correctly.

Contract version: v1. Audience: consumers reading `hero.json` and the embedded `core/` corpus.

## §1. What the dialect is

The dialect has two orthogonal layers:

- **Vocabulary** — display names only. Under `agile-scrum`, a `feature` artifact renders as `Story`; under `shape-up`, the same artifact renders as `Pitch`. Vocabulary changes nothing about how Hero stores, validates, or transitions specs — only what the user sees.
- **Methodology** — structural behavior. Lifecycle states and transitions, time-box requirements (e.g. sprint duration), estimation field (`points` vs `appetite`), rituals, rollups (velocity, burndown). Methodology changes how the system operates.

A workspace may declare **neither** (engineering default — bare type literals, kanban behavior), **one**, or **both**. Setting one without the other is supported: methodology infers a paired vocabulary via `aligned_vocabulary`, and vocabulary infers nothing about methodology.

## §2. Where to read the active dialect

Authoritative source: `hero.json` at the workspace root. Relevant fields on the top-level `Config`:

| Field | Type | Purpose |
|---|---|---|
| `vocabulary` | `string` | Explicit preset name: `default`, `agile-scrum`, `shape-up`, `kanban`, `jira`, `linear`. Empty → infer. |
| `vocabulary_overrides` | `map[string]string` | Per-key tweaks on the resolved vocabulary (see §4). |
| `methodology` | `string` | Explicit profile name: `scrum`, `kanban`, `shape-up`, `waterfall`, `scrumban`. Empty → infer. |
| `methodology_overrides` | `map[string]string` | Per-key tweaks on the resolved methodology (time-box duration, estimation field, etc.). |
| `tracker.type` | `string` | Inference fallback. Used only when `vocabulary` and `methodology` are empty. |
| `pm.presets.delivery` | `string` | Inference fallback: `sprint` → agile-scrum/scrum, `cycle` → shape-up, `flow`/`continuous` → kanban. |
| `domain` | `string` | Active domain pack (`engineering`, `pm`, `sales`, …). Affects which spec-types overlay loads — orthogonal to vocabulary/methodology selection. |

Read order: load `hero.json`, then merge `hero.local.json` on top. `hero.local.json` is a per-user, gitignored file that overrides any field on the top-level `Config` — including the four dialect fields. Override semantics for the dialect layer:

- `vocabulary` (scalar) — non-empty local value replaces the base value.
- `methodology` (scalar) — non-empty local value replaces the base value.
- `vocabulary_overrides` (map) — entry-by-entry merge: local entries replace base entries on key collision; non-colliding base keys are preserved.
- `methodology_overrides` (map) — entry-by-entry merge with the same semantics as `vocabulary_overrides`.

An absent or empty `hero.local.json`, or one that omits these fields, leaves the base `hero.json` values untouched. The merge is implemented in `internal/config/config.go::MergeLocal`; consumers reading these files directly must replicate the same precedence (local wins on scalars; entry-by-entry merge with local-wins-on-collision for the override maps) to stay compatible.

The preset corpora live under `core/vocabularies/<name>.yaml` and `core/methodologies/<name>.yaml`. These ship embedded in the Hero binary (`vocabulary.CoreFS()`, `methodology.CoreFS()`); a consumer reading the YAML directly from a Hero checkout gets the same content.

## §3. Resolver precedence chain

### Methodology (`internal/methodology/resolver.go`)

1. **Explicit** — `cfg.Methodology` if non-empty.
2. **Tracker-inferred** — `tracker.type == "jira"` → `scrum`. No other trackers carry a strong methodology signal today.
3. **Delivery-preset-inferred** — `pm.presets.delivery`: `sprint` → `scrum`, `cycle` → `shape-up`, `flow`/`continuous` → `kanban`.
4. **Default** — `kanban` (least-opinionated baseline).

### Vocabulary (`internal/vocabulary/resolver.go` + caller wrapping)

The vocabulary resolver itself implements three steps; methodology-derivation happens in the **caller** that wraps it (see `internal/cli/vocab.go::activeVocab`, `internal/serve/vocab.go`, `internal/install/dialect.go`). The effective end-to-end chain a consumer must replicate is:

1. **Explicit** — `cfg.Vocabulary` if non-empty.
2. **Methodology-derived** — if `cfg.Methodology` is set and `cfg.Vocabulary` is empty, resolve the methodology and read its `aligned_vocabulary` field (`methodology.DeriveVocabularyName`). E.g. `scrum.yaml` declares `aligned_vocabulary: agile-scrum`.
3. **Tracker-inferred** — match `tracker.type` against each vocabulary's `auto_select: [{tracker: <name>}]` rule.
4. **Delivery-preset-inferred** — match `pm.presets.delivery` against each vocabulary's `auto_select: [{delivery_preset: <value>}]` rule.
5. **Default** — the `default` preset.

For the full `auto_select` schema — allowed keys and values, match semantics within and across presets, and an authoring example — see [`vocabulary-auto-select.md`](./vocabulary-auto-select.md).

After picking, `vocabulary_overrides` (and `methodology_overrides`) are merged onto a clone of the chosen base. The resolver warns when overrides exceed 10 keys — a signal the workspace should author its own preset instead.

**Important implementation note.** The bare `vocabulary.Resolve(cfg, vocabs)` call alone does **not** perform step 2 (methodology derivation). It runs steps 1, 3, 4, 5 only. Internal CLI/serve/install paths inject the derived name onto a `cfg` copy before calling `Resolve` — that is what produces the documented end-to-end chain. A consumer calling Hero in-process must use the same wrapping pattern, or call `methodology.DeriveVocabularyName` explicitly and set `cfg.Vocabulary` before invoking the vocabulary resolver. A consumer reading from disk independently must implement step 2 themselves.

## §4. Display map

Each vocabulary YAML declares display strings at two granularities:

```yaml
# core/vocabularies/agile-scrum.yaml (excerpt)
types:
  spec: Story
  epic: Epic
  roadmap-item: Theme
kinds:
  spec.feature: Story
  spec.bug: Bug
  spec.refactor: Tech Debt Story
sections:
  acceptance_criteria: "Acceptance Criteria"
```

Resolution order in `Vocabulary.Display(typeName, kind)`:

1. `kinds["<type>.<kind>"]` — most specific.
2. `types["<type>"]` — type-level fallback when no kind override exists.
3. Literal `kind` — fall through when kind is non-empty.
4. Literal `type` — fall through when kind is empty.

Hero's flat spec-frontmatter type literals (`feature`, `bug`, `chore`, …) are stored as top-level types but render through the canonical `(type=spec, kind=<literal>)` pair. The CLI bridges this in `canonicalize()` (`internal/cli/vocab.go`); a consumer reading specs directly should apply the same mapping: `feature`/`bug`/`chore`/`refactor`/`perf`/`infra`/`security`/`ux` → `(spec, <literal>)`; `initiative` → `(roadmap-item, "")`; everything else passes through with empty kind.

`vocabulary_overrides` keys layer onto the resolved map:

| Key shape | Target |
|---|---|
| `types.<type>` | `Types[<type>]` |
| `kinds.<type>.<kind>` | `Kinds["<type>.<kind>"]` |
| `sections.<canonical>` | `Sections[<canonical>]` |
| `lifecycle.<type>.<status>` | `Lifecycle[<type>][<status>]` |

## §5. Worked example

`hero.json` declares only:

```json
{ "methodology": "scrum" }
```

Resolution:

1. **Methodology resolves to `scrum`** (step 1, explicit).
2. **Vocabulary is empty** → fall through to step 2 (methodology-derived). Read `scrum.yaml` → `aligned_vocabulary: agile-scrum`. **Vocabulary resolves to `agile-scrum`.**
3. A spec with frontmatter `type: feature` canonicalizes to `(type=spec, kind=feature)`. Look up `kinds["spec.feature"]` in `agile-scrum.yaml` → **renders as `Story`**.
4. A spec with `type: refactor` canonicalizes to `(spec, refactor)` → `kinds["spec.refactor"]` → **renders as `Tech Debt Story`**.
5. A spec with `type: feature, kind: tech-debt` (a free-form kind not in the vocab) → `kinds["spec.tech-debt"]` missing → fall back to `types["spec"]` → **renders as `Story`**.

A workspace with `{}` (no methodology, no vocabulary): consumers should render bare type literals. The CLI's `activeVocab` returns `nil` in this case; `Vocabulary.Display(nil, ...)` returns the literal unchanged.

## §6. Read paths summary

| Want | Read | Apply |
|---|---|---|
| Active methodology | `hero.json` → `methodology` → fall through chain (§3) | Load `core/methodologies/<name>.yaml`, merge `methodology_overrides`. |
| Active vocabulary | `hero.json` → `vocabulary` → derive from methodology → tracker → delivery preset → `default` | Load `core/vocabularies/<name>.yaml`, merge `vocabulary_overrides`. |
| Display name for a `(type, kind)` | Resolved vocabulary | `kinds["<type>.<kind>"]` → `types["<type>"]` → literal. |
| Section heading | Resolved vocabulary | `sections["<canonical>"]` → title-cased literal. |
| Lifecycle states | Resolved methodology | `lifecycle["<type>"].states` ordered list. |
| Time-box defaults | Resolved methodology | `time_boxes[]` filtered by `level`. |
| Estimation field | Resolved methodology | `estimation["<type>"].required_field`. |
| Is any dialect layer active? | `cfg.Vocabulary != "" \|\| cfg.Methodology != ""` | If false, render bare type literals. |

## §7. Stability — what changes vs what holds

**Stable within v1:**

- Field names on `Config` (`vocabulary`, `vocabulary_overrides`, `methodology`, `methodology_overrides`, `pm.presets.delivery`, `tracker.type`).
- YAML top-level field names on vocabulary/methodology presets (`name`, `display_name`, `types`, `kinds`, `sections`, `lifecycle`, `time_boxes`, `estimation`, `aligned_vocabulary`, `auto_select`).
- The resolver precedence chain (steps and order).
- The `Display` resolution order (kinds → types → literal).

**Additive within v1 (consumers must tolerate without breaking):**

- New top-level fields on vocabulary or methodology YAMLs.
- New presets (e.g. a `kanban-lean` vocabulary) and new methodology profiles.
- New `kinds` / `types` / `sections` entries inside an existing preset.
- New `auto_select` rule key shapes.
- New override key prefixes (consumers parsing overrides should ignore unrecognized prefixes — that's exactly what the in-process resolvers do, logging and skipping).

**Breaking (bumps the contract version):**

- Renaming a shipped preset (`agile-scrum` → `scrum-stories`).
- Reordering the resolver precedence chain.
- Removing a field or changing its type.
- Changing the `(type, kind)` canonicalization mapping for the engineering flat literals.

**Consumer rule of thumb.** Read what you understand; ignore what you don't. Never fail on an unknown field. When in doubt about a preset's identity, prefer reading `display_name` over hardcoding `name`.
