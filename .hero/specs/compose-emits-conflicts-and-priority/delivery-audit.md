# Delivery audit — compose-emits-conflicts-and-priority

**Audited:** `git diff -- core/ domains/` (working tree, 5 markdown files) + `go build ./...`
**Verdict:** SHIP
**Surface:** clean

Content/instruction-only delivery (Hero skill/command/agent markdown). No code, no
test suite — audited for correctness of the *instructions* and the single-canonical
anti-drift invariant, not for code behavior.

## Acceptance criteria

- [✓] Stamp `priority:` on every child stub via the mapping table — mapping table in
  `core/skills/spec-format/SKILL.md:369-374`; "Stamp every child … All-or-nothing"
  rule at `:376-378`; threaded into `compose.md`, both delivery leads, ideator.
- [✓] `bug` children also stamp `severity:` — rule at `spec-format/SKILL.md:385-388`;
  echoed in `compose.md`, feature-lead, platform-lead.
- [✓] Overlap seam → reciprocal `conflicts-with` on both children, outbound-only reason
  inline — two-sided YAML example `spec-format/SKILL.md:393-402`; rationale "judge
  honors own (outbound) … does not scan inbound … Author both edges; do not simplify
  to one" at `:404-410`. All four surfaces restate "one edge each / outbound only".
- [✓] Keep Wave/overlap prose + sync invariant (no orphan prose/relation) — blockquote
  invariant `spec-format/SKILL.md:417-425`; reinforced in `compose.md` and both leads.
- [✓] Contract documented in shared `core/skills/spec-format/SKILL.md` — new top-level
  `## Child-stub authoring contract` section `:353-440`.
- [✓] `/design` materialize preserves `priority`/`severity` + `conflicts-with` —
  "Preserve on materialize" rule `spec-format/SKILL.md:427-436`; both delivery leads
  name preserve-on-materialize as source-of-truth rule.
- [✓] No Go change required — `go build ./...` exit 0; `git diff --name-only` under
  `core/`/`domains/` is markdown-only.

## Changes

- [✓] `core/skills/spec-format/SKILL.md` — new "Child-stub authoring contract" section
  (mapping table, stamp-every-child, tiebreak-not-ordering, severity-for-bugs,
  reciprocal conflicts-with + outbound reason, sync invariant, preserve-on-materialize)
  + 3 frontmatter-table rows (`conflicts-with`/`priority`/`severity`). +91 lines.
- [✓] `domains/engineering/commands/compose.md` — "Structured signals on every child
  stub" subsection; points to spec-format contract, does not restate the table. +23.
- [✓] `domains/engineering/agents/product-ideator.md` — extended "Final output" with a
  concrete `priority` value per item and an overlap/seam callout naming both items.
  Correctly frames ideator as *flagging* the seam, delivery lead as *emitting* the
  relation. +2 lines.
- [✓] `domains/engineering/agents/feature-delivery-lead.md` — paragraph in the
  spec-composition path: stamp priority/severity + emit reciprocal conflicts-with,
  references the contract. +2 lines.
- [✓] `domains/engineering/agents/platform-delivery-lead.md` — same, with platform seam
  examples (schema-before-code, dual-write, shared-migration). +2 lines.

## Audit notes

- **Single-canonical-table invariant holds.** `grep "Foundational anchor" core/ domains/`
  returns exactly one hit — `core/skills/spec-format/SKILL.md:373`. The other four
  surfaces mention `conflicts-with` but reference the contract by name; none restate
  the mapping table. This is the spec's core anti-drift design and it was honored.
- **YAML example is parser-valid.** The two-sided `relations:` block
  (`- target: … / kind: conflicts-with`) matches `parseRelationsBlock`
  (`internal/spec/spec.go:642-770`) exactly. `conflicts-with` is an accepted relation
  kind (`spec.go:613`). Verified, not assumed.
- **Judgment call — 3 added frontmatter rows are correct, not scope creep.** The
  engineer added `conflicts-with`/`priority`/`severity` rows to the existing frontmatter
  table (slightly beyond the literal ask). All three fields genuinely parse:
  `priority:`→`s.Priority` and `severity:`→`s.Severity` (`spec.go:504-507`),
  `conflicts-with`→relation edge (`spec.go:613,625`). The rows document real, parsed
  fields and improve table completeness — coherence-improving within the named file and
  within Change item 1's scope. Not flagged.
- **Boundary held: zero Go.** `git diff --name-only` shows only the 5 named `.md` files
  under `core/`/`domains/` (plus pre-existing `.hero/*` session state, out of scope).
  `go build ./...` green — the spec's hard "zero Go change" boundary is intact.
- All six required contract rules present: (a) mapping table, (b) stamp-every-child
  all-or-nothing, (c) severity-for-bugs, (d) reciprocal conflicts-with + outbound reason,
  (e) prose⇄relation sync invariant, (f) preserve-on-materialize. Plus the
  tiebreak-not-ordering rule. No downgrades.
