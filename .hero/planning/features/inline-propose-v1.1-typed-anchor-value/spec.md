---
title: Inline-propose v1.1 — typed anchor.value discriminator (gated on hero-code feedback)
type: feature
status: planning
priority: P3
horizon: next
created: 2026-05-17
tags: [inline-propose, contract, hero-code, anchor, schema-evolution, gated]
relations:
  - target: inline-propose-output-mode
    kind: extends
  - target: hero-code-handover-pack
    kind: surfaced-by
  - target: hero-code
    kind: cross-repo-consumer
gating_signal: hero-code reports concrete Rust serde typing pain on prose-shaped anchor.value
smoke: deferred
---

# Inline-propose v1.1 — typed anchor.value discriminator (gated on hero-code feedback)

## Status

**Filed for tracking; delivery deferred.** Do not pick this up until hero-code has built `src/proposed_block.rs` against the v1.0 fixtures (`testdata/proposals/v1/`) and surfaced concrete typing pain. Premature delivery risks designing the wrong discriminator.

The trigger to start work is **evidence**, not a calendar date:

- Hero-code files a peer call (advisory) describing serde deserialization friction on `anchor.value`, OR
- Hero-code's Rust integration tests pin the v1.0 fixtures and the resulting type definitions feel obviously wrong (multi-purpose `String` fields, runtime branching on `anchor.kind` to interpret `anchor.value`), OR
- A second consumer (CLI tool, third-party integration) reports the same shape pain.

Until then, the spec sits in `planning` with `priority: P3` and `horizon: next`. The follow-ups table in `hero-code-handover-pack` points here so the gap is tracked, not lost.

## Context

The v1.0 inline-propose contract at [docs/contracts/inline-propose-v1.md](../../../../docs/contracts/inline-propose-v1.md) declares the anchor envelope as:

```json
{
  "anchor": {
    "kind": "section",
    "value": "acceptance_criteria",
    "position": "append"
  }
}
```

Where `anchor.kind ∈ {frontmatter, section, heading, list_item, free}` and `anchor.value` is a single string whose **semantic interpretation depends on `kind`**:

| `kind` | `value` shape |
|---|---|
| `frontmatter` | YAML frontmatter field name (`priority`, `status`, ...) |
| `section` | section heading slug (`acceptance_criteria`, `risks`) |
| `heading` | heading slug to anchor adjacent (with `position: before`/`after`) |
| `list_item` | stable list-item identifier (`ac-3`, `risk-2`) |
| `free` | free-form position hint string (`"end of file"`, `"after AC list"`) |

The contract documents this in prose only. The JSON Schema treats `anchor.value` as an unconstrained `string`. A consumer building typed code (Rust, TypeScript, generated from the schema) gets a single `String` field with no per-kind interpretation — they must runtime-branch on `kind` to know what `value` means.

For hero-code specifically, this means `proposed_block.rs` will likely end up with shapes like:

```rust
struct Anchor {
    kind: AnchorKind,
    value: String,  // semantics opaque without inspecting `kind`
    position: Option<AnchorPosition>,
}
```

instead of the more idiomatic:

```rust
enum Anchor {
    Frontmatter { field: String },
    Section { slug: String, position: Position },
    Heading { slug: String, position: BeforeOrAfter },
    ListItem { id: String, position: BeforeOrAfter },
    Free { hint: String },
}
```

Whether that's actual friction or merely aesthetic depends on what hero-code is building **with** the envelope. The widget renders proposals — does the renderer care which kind it is at a typed level, or is "render the body at the anchor on the artifact pane" agnostic enough that runtime branching is fine? We don't know until they build it.

This was surfaced by the C1 agent (inline-propose fixtures) and C4 agent (JSON Schema) of `hero-code-handover-pack` as a future contract evolution worth tracking but not pre-emptively designing.

## Goal

Once hero-code surfaces concrete pain, ship a v1.1 additive evolution of the inline-propose contract that lets typed consumers produce per-kind anchor shapes without losing the v1.0 wire compatibility producers already implement.

The bar: hero-code can write idiomatic Rust without `match anchor.kind { ... runtime branch on value semantics ... }`, while a v1.0 producer (the current `internal/propose/` package) keeps emitting valid envelopes without code changes until they want to opt in.

## Design — three options to weigh when delivery starts

Premature design is the failure mode this spec is trying to avoid. The options below are sketches to remind future-us of the shape space; the final design fires after hero-code's evidence lands.

### Option A — discriminator field `anchor.value_shape`

Add an optional `anchor.value_shape` enum field whose value names the per-kind interpretation explicitly: `frontmatter_field | section_slug | heading_slug | list_item_id | free_text`. The discriminator is informational for typed consumers; the wire shape of `anchor.value` stays a string.

- Pro: smallest possible envelope change. v1.0 consumers ignore the field; v1.1 typed consumers gate codegen on it.
- Con: redundant — `value_shape` is a pure function of `kind`. Two sources of truth invite drift.
- Con: doesn't get us a discriminated union in Rust; it just makes the prose-typing legible.

### Option B — discriminated union: `anchor` becomes a typed shape

Replace the flat `{kind, value, position}` shape with a tagged union keyed on `kind`:

```json
{
  "anchor": {
    "kind": "section",
    "slug": "acceptance_criteria",
    "position": "append"
  }
}
```

vs

```json
{
  "anchor": {
    "kind": "list_item",
    "id": "ac-3",
    "position": "after"
  }
}
```

- Pro: idiomatic for typed consumers; the JSON Schema can use `oneOf` per `kind`.
- Con: breaking change for v1.0 producers. Daemon would need to accept both v1.0 (`value` field) and v1.1 (per-kind fields) and translate.
- Con: producer-side `internal/propose/` envelope construction gets more complex (one builder per kind instead of a generic anchor builder).

### Option C — keep v1.0; ship a Rust-side normalization helper

Treat the prose-typed `value` field as a producer/wire-format artifact and let consumers normalize to typed shapes themselves. We could ship a reference normalization helper (Go or vendored Rust) that hero-code uses to map `{kind: section, value: "acceptance_criteria"}` → `Anchor::Section{slug, position}` at parse time.

- Pro: zero wire change. v1.0 stays the canonical contract; the discrimination is a consumer-side concern.
- Con: every typed consumer rewrites the same normalization. Doesn't help the second consumer.
- Con: the contract document still has the prose-shape problem for new consumers.

**Recommended trigger when work starts:** start with Option A unless hero-code's pain is specifically about producing typed Rust at the schema-derivation step (which would justify Option B's wire change). Option C is the fallback if Options A and B both feel disproportionate to the actual friction.

## Acceptance Criteria

These activate when the spec moves out of gated state. Today they're aspirational.

- THE SYSTEM SHALL preserve v1.0 envelope wire compatibility — a v1.0 producer's envelopes keep flowing through the v1.1 daemon without producer-side changes.
- THE SYSTEM SHALL bump `schema_version` to `"1.1"` on envelopes that opt into the v1.1 shape; v1.0 envelopes keep emitting `"1.0"`.
- THE SYSTEM SHALL update `docs/contracts/inline-propose-v1.md` (or fork to `inline-propose-v1.1.md`, depending on the chosen design's scope) to document the new shape with worked examples.
- THE SYSTEM SHALL update `docs/contracts/spec-types-v1.1.schema.json`'s anchor sub-schema (or sibling) to validate both v1.0 and v1.1 envelopes.
- THE SYSTEM SHALL update or extend `testdata/proposals/v1/` to add a parallel `testdata/proposals/v1.1/` fixture corpus exercising the new shape per anchor kind.
- WHEN a v1.0 envelope arrives, THE SYSTEM SHALL relay it on the SSE stream unchanged (no implicit upgrade in v1.1 — would be a behavior change for existing consumers).
- WHEN a v1.1 envelope arrives, THE SYSTEM SHALL relay it on the SSE stream with the v1.1 shape preserved (no implicit downgrade either).

## Boundaries

- **Not** designing a v2.0 breaking version. v1.1 is strictly additive in wire format (even if it adds new fields or new sibling shapes).
- **Not** retrofitting v1.1 onto agents emitting v1.0. The Go side's `internal/propose/` keeps emitting v1.0 unless explicitly opted in via flag or config.
- **Not** changing the lifecycle log line format, the REST endpoints, the SSE event types, or the per-anchor replacement semantics. Only the anchor envelope shape is in scope.
- **Not** writing a Rust normalization helper from this repo. If we go with Option C, the helper ships in hero-code, not here.

## Validation

Activates when the spec moves out of gated state.

- Round-trip unit tests for both v1.0 and v1.1 envelopes through `internal/propose/`'s ingest path.
- JSON Schema validation: v1.0 fixtures still validate; new v1.1 fixtures validate; v1.1 fixtures targeted at v1.0 schema fail loudly.
- An e2e test that exercises a v1.0 producer + v1.1 consumer round-trip through the daemon.
- Hero-code-side: hero-code authors typed Rust against the new shape, ports their `src/proposed_block.rs` rendering, and confirms via peer call that the typing pain has been addressed.

## Gating workflow

1. Hero-code builds `src/proposed_block.rs` against the v1.0 fixtures.
2. If typed-shape pain materializes, hero-code fires a peer call (advisory) with specific evidence: which fields cause friction, what shape they'd prefer, what their use case is.
3. We move this spec from `planning` to `refined`, choose the design option (A/B/C above) based on the evidence, file a kickoff in this spec, and start delivery.
4. If hero-code ships `proposed_block.rs` without surfacing pain, the spec stays in `planning` indefinitely. After ~90 days of silence, downgrade to `horizon: someday` and archive into `.hero/specs/` as won't-fix-because-not-needed.

The point: **evidence drives delivery**, not pre-emptive design.

## Kickoff

> **Do not pick this up without evidence first.** Confirm hero-code has filed a peer call (or equivalent advisory) describing concrete typing pain on `anchor.value` in the v1.0 envelope. Read the evidence. Then re-read this spec's Design section — A/B/C are sketches, not commitments; pick the option that addresses the specific pain hero-code surfaced, not the one that sounds cleanest in the abstract. Update this spec with the chosen design (replace the Design section with a single locked option, retain A/B/C as "Rejected" subsections), then deliver per the Acceptance Criteria. If hero-code's pain is rendering-side rather than typing-side, push back — this spec is for typed-shape friction only.

## Related work

- Parent contract: [docs/contracts/inline-propose-v1.md](../../../../docs/contracts/inline-propose-v1.md)
- v1.0 fixture corpus: [testdata/proposals/v1/](../../../../testdata/proposals/v1/)
- Originally surfaced in: [hero-code-handover-pack](../hero-code-handover-pack/spec.md) §"Open contract gaps"
- Producer implementation: `internal/propose/envelope.go`
- JSON Schema for spec-types (related but distinct): [docs/contracts/spec-types-v1.1.schema.json](../../../../docs/contracts/spec-types-v1.1.schema.json)
