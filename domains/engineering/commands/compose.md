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

Initiative to decompose: $ARGUMENTS
