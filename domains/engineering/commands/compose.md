---
description: Break a large initiative into sequenced specs with a coordinated delivery plan.
---
Route this initiative to `product-ideator` for scope and sequencing analysis, then to the appropriate delivery lead for spec planning.

Determine whether the initiative is primarily product or platform work:
- Product initiatives, user-facing overhauls, new capability suites → delegate planning to `feature-delivery-lead`
- Migrations, platform transitions, infrastructure overhauls → delegate planning to `platform-delivery-lead`

The workflow:
1. `product-ideator` breaks the initiative into concrete, sequenced work items with dependencies and priorities
2. The delivery lead produces an initiative spec at `.hero/planning/initiatives/{slug}/spec.md` containing:
   - Initiative summary and goals
   - Sequenced work items with dependency ordering
   - Child spec stubs for each work item (ready for individual `/design` passes)
   - Cross-cutting concerns and shared risks
   - Recommended delivery order

Each child spec stub should be actionable as input to `/design` when the team is ready to start that piece.

### Where child specs live

The initiative's children table may reference children as lightweight stubs
before they are designed. When `/design` **materializes** a child into a real
spec, give it its **own folder under the initiative**:
`.hero/planning/initiatives/{initiative-slug}/{child-slug}/spec.md`. The folder
keeps the child grouped with its initiative *and* gives it a home for its
companion artifacts (delivery audit, mocks, plan) — which the delivery gates
look for beside `spec.md`. See the `spec-format` skill's "Folder-per-spec is the
optimal layout" guidance.

A child may start as a flat `.../{initiative-slug}/{child-slug}.md` stub, but
promote it to its own folder before delivery so its audit and mocks co-locate
where `hero spec verify` expects them. Always stamp `slug:` in each child's
frontmatter — it is the authoritative identifier regardless of layout.

### Structured signals on every child stub

The sequencing analysis produces two decision-relevant signals the `/drive`
judge reads directly. Emit them as structured frontmatter on the stubs, not only
as prose — otherwise the judge can't act on them and both features sit inert:

- **Stamp `priority:` on every child** (and `severity:` on `bug`-type children)
  per the mapping in the `spec-format` skill's "Child-stub authoring contract".
  All-or-nothing — an unstamped child sinks below its stamped siblings in the
  judge, silently reordering the run.
- **For every overlap seam the initiative prose names, emit a reciprocal
  `conflicts-with`** relation on **both** named children (one edge each). The
  judge honors outbound edges only, so a one-sided edge protects one direction
  and silently no-ops the other.
- **Keep the Wave table and "in-flight overlap watch" prose** — it explains the
  *why*. But satisfy the **prose ⇄ relation sync invariant**: every named seam
  in prose has a matching reciprocal `conflicts-with`, and no `conflicts-with`
  lacks explaining prose (no orphan prose, no orphan relation).

The `spec-format` "Child-stub authoring contract" is the source of truth for the
priority mapping, the reciprocity rule, the sync invariant, and the
preserve-on-materialize rule — follow it; don't restate the mapping table here.

Initiative to decompose: $ARGUMENTS
