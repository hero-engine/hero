# Delivery audit — inline-flow-relations-dropped

**Delivered via:** `/drive knowledge-surfacing` (guided), 2026-07-07
**Verdict:** SHIP
**Surface:** noteworthy

Frontmatter parser now accepts inline-flow relations. One-function fix +
reused `applyRelField`; no dependency added.

## Acceptance criteria

- [✓] **Inline-flow parses** — `TestParseRelations_InlineFlowAndMixed` asserts
  `- { target: x, kind: parent }` yields target+kind.
- [✓] **Mixed inline + block** — same test mixes both styles; all 3 parse.
- [✓] **content-remediation children resolve** — live: `hero goal
  content-remediation --check` went from `completed: None` to
  `completed: [pm-pack-phantom-surfaces, sales-pack-reality-sync]`, and the 2
  planning inline-flow children (`core-commands-domain-neutral`,
  `token-efficiency-pass`) now appear in `remaining` instead of vanishing. All 4
  previously-dropped children visible.
- [✓] **No regression** — `go test ./...` = ALL PASS.

## Fix

- `internal/spec/spec.go` — `parseRelationsBlock` detects a `{…}` payload after
  `- ` and routes to new `applyInlineRelation`, which strips braces, splits on
  `,`, and applies each pair via the existing `applyRelField`. Block style
  unchanged.
- `internal/spec/spec_test.go` — `TestParseRelations_InlineFlowAndMixed`.

## Notes

- Root cause: `applyRelField` did `strings.Cut(line, ":")` on the raw list-item
  text, so `{ target: x, kind: y }` yielded key `"{ target"` → dropped silently.
- Blast radius at time of fix: 4 specs (`sales-pack-reality-sync`,
  `pm-pack-phantom-surfaces`, `content-remediation/token-efficiency-pass`,
  `content-remediation/core-commands-domain-neutral`). No migration needed —
  the parser now accepts both styles.
- Surfaced while driving knowledge-surfacing: the initiative's own children were
  invisible until rewritten to block style. This fix means new specs can use
  either style safely.
