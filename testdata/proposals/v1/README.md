# Inline-Propose v1.0 — Test Fixtures

Canonical JSON fixtures for the inline-propose envelope contract pinned at
[`docs/contracts/inline-propose-v1.md`](../../../docs/contracts/inline-propose-v1.md).

These fixtures are the load-bearing handoff from hero (Go, producer) to
hero-code (Rust dashboard, consumer). They exist so hero-code's widget
tests can deserialize known-good envelopes and exercise rendering without
having to spin up a live hero daemon.

## Index

| Fixture | Shape | Exercises |
|---|---|---|
| `single-ac-append.json` | one envelope | Canonical case: `anchor.kind=section`, `value=acceptance_criteria`, `position=append`, markdown body in EARS form. The base case every widget must render correctly. |
| `frontmatter-field-replace.json` | one envelope | `anchor.kind=frontmatter` with a YAML scalar body. Exercises the YAML content-format branch and the frontmatter-replace renderer. |
| `section-replace.json` | one envelope | `anchor.kind=section` with `position=replace` and a multi-paragraph body. Exercises full-section overwrite rendering. |
| `heading-before.json` | one envelope | `anchor.kind=heading` with `position=before`. Exercises inserting a brand-new section ahead of an existing heading. |
| `list-item-after.json` | one envelope | `anchor.kind=list_item` with a stable item id (`ac-3`) and `position=after`. Exercises in-list insertion using a list-item anchor. |
| `free-position.json` | one envelope | `anchor.kind=free` with a hint string (`"end of file"`) and no `position`. Exercises the "anchor unknown, hint provided" fallback. |
| `batch-multi-ac.json` | array of four envelopes | Four envelopes sharing one `batch_id` from the same agent against the same spec — the "agent drafts 4 AC at once" UX. Exercises batch rendering and the bulk accept/reject widgets. |
| `replacement-scenario.json` | array of two envelopes | Two envelopes from the same agent against the same `(spec_slug, anchor.kind, anchor.value)` tuple with distinct `proposal_id`s and bodies. Per the contract's Decision 2 (per-anchor replacement scoped to same agent), the second envelope replaces the first in the daemon's store. Exercises a consumer's idempotency / dedupe-on-anchor logic. |

## Envelope validity rules

Every envelope in this directory MUST satisfy
[`docs/contracts/inline-propose-v1.md`](../../../docs/contracts/inline-propose-v1.md) §2:

- `schema_version` is exactly `"1.0"`.
- Required fields are present: `proposal_id`, `batch_id`, `session_id`,
  `agent`, `target.spec_slug`, `target.anchor.kind`, `target.anchor.value`,
  `content.format`, `content.body`.
- `target.anchor.kind` is one of: `frontmatter` | `section` | `heading` |
  `list_item` | `free`.
- `target.anchor.position`, when present, is one of: `replace` | `append` |
  `prepend` | `before` | `after`.
- `content.format` is one of: `markdown` | `text` | `yaml`.
- `proposal_id` matches `p-` + 6 hex; `batch_id` matches `b-` + 6 hex;
  `session_id` matches `sess-YYYY-MM-DD-` + 3 hex.
- `emitted_at`, when present, is RFC3339.

Optional fields (`skill_chain`, `target.spec_path`, `rationale`,
`emitted_at`, `target.anchor.position`) appear on most fixtures so consumers
can exercise both the present and absent branches; `free-position.json`
deliberately omits `target.anchor.position` to cover the default-omitted
case.

## How hero-code's tests consume these

The expected consumption path on the Rust side:

```rust
// pseudo-code; widget-test layer
let path = workspace.join("testdata/proposals/v1/single-ac-append.json");
let raw = std::fs::read_to_string(&path)?;
let envelope: ProposalEnvelope = serde_json::from_str(&raw)?;
widget::render(&envelope);
```

For array fixtures (`batch-multi-ac.json`, `replacement-scenario.json`),
deserialize as `Vec<ProposalEnvelope>` and feed each envelope through the
ingest path (or batch renderer) in order.

The serde types should be derived from the schema fields documented in the
contract; treat unknown fields as additive-tolerant per §7 of the contract.

## Maintenance

- Adding a fixture: pick one anchor variant or scenario the existing set
  doesn't cover, name it descriptively, validate with `jq '.' <file>`,
  and add a row to the index table above.
- Changing a fixture: only acceptable if the contract changes. Schema
  bumps land in the contract document first; fixtures move under
  `testdata/proposals/v2/` (etc.) when the major version bumps.
- These fixtures are static. They are not generated, and they are not a
  complete session trace — they are minimum-viable known-good envelopes
  per scenario.
