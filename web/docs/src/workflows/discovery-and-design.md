# Discovery & Design

Hero's discovery and design commands help you explore what to build, produce
actionable specs, and make architectural decisions — all before writing a single
line of code.

---

## `/discover` — Brainstorm & Prioritize

Starts an interactive product discovery session with the **product-ideator**
agent. Use it when you're exploring what to build next.

The agent walks through product context, goals, and pain points, then proposes
and evaluates feature ideas to produce a prioritized list of work items ready
for `/design`.

```bash
# Open-ended exploration
/discover

# Scoped to a direction
/discover improve onboarding experience for self-serve customers

# Competitive analysis angle
/discover what features are we missing compared to Linear and Shortcut
```

!!! tip "Discovery → Design pipeline"
    Items from `/discover` are formatted as inputs to `/design`. Pick the
    top-priority item and run `/design` to produce a full spec.

---

## `/design` — Produce a Spec

Routes your request to the appropriate delivery lead who produces a complete
spec and saves it to `.hero/planning/`.

Hero automatically selects the right lead:

| Work type | Agent |
|---|---|
| Product features, user-facing enhancements | `feature-delivery-lead` |
| Migrations, refactors, platform changes | `platform-delivery-lead` |

```bash
# Feature spec
/design add CSV export to the analytics dashboard

# Platform spec
/design migrate session storage from Redis to PostgreSQL

# From a tracker issue
/design PROJ-1234
```

The delivery lead coordinates with architect agents as needed, produces a spec
with goals, changes, acceptance criteria, and a validation plan, then saves it
to `.hero/planning/{slug}/spec.md`.

!!! info "Auto-capture"
    When `knowledge.auto_capture` is enabled (the default), Hero silently
    captures architectural decisions and constraints discovered during design
    to `.hero/knowledge/`.

---

## `/compose` — Multi-Spec Initiatives

Breaks a large initiative into sequenced, independently deliverable specs with
a coordinated delivery plan.

The workflow has two phases:

1. **product-ideator** decomposes the initiative into concrete work items with
   dependencies and priorities
2. The appropriate **delivery lead** produces an initiative spec at
   `.hero/planning/initiatives/{slug}/spec.md` containing child spec stubs

```bash
# Product initiative
/compose rebuild the permissions system to support team-based access control

# Platform initiative
/compose migrate from monolith to service-oriented architecture
```

Each child spec stub is ready to be fleshed out with an individual `/design`
pass when the team is ready to start that piece.

!!! example "Initiative structure"
    ```
    .hero/planning/initiatives/team-permissions/
    ├── spec.md              # Initiative summary, goals, delivery order
    ├── team-permissions-rbac/
    │   └── spec.md          # Child: RBAC model
    ├── team-permissions-ui/
    │   └── spec.md          # Child: Management UI
    └── team-permissions-api/
        └── spec.md          # Child: API endpoints
    ```

---

## `/mock` — HTML Prototypes

Generates self-contained HTML mockups from a spec or free-text description
using the **ui-designer** agent. No external dependencies — just a single
`index.html` with embedded CSS and inline JS.

```bash
# From a spec
/mock team-permissions-ui

# From a description
/mock a dashboard showing real-time sprint progress with burndown chart

# Iterate on an existing mockup
/mock team-permissions-ui --iterate make the sidebar collapsible
```

Mockups are saved to `.hero/mocks/{slug}/index.html` and include responsive
layouts, modern CSS (flexbox/grid), and basic interactivity (tabs, modals,
dropdowns).

!!! tip "Load the skill"
    The `html-mockup-generation` skill is loaded automatically to give the
    ui-designer detailed guidance on design patterns and component libraries.

---

## `/decide` — Architectural Decision Records

Evaluates architectural decisions with structured tradeoff analysis and produces
a decision spec. Routes to **architecture-reviewer** and optionally involves
domain-specific architects:

| Concern | Additional agent |
|---|---|
| Existing system constraints | `brownfield-architect` |
| New system or subsystem design | `greenfield-architect` |

```bash
# Technology choice
/decide should we use SQLite or PostgreSQL for the knowledge base

# Architecture pattern
/decide event sourcing vs CRUD for the audit log

# Migration strategy
/decide big-bang vs strangler-fig for the API v2 migration
```

The decision spec is saved to `.hero/decisions/{slug}/spec.md` with the
recommendation, rationale, tradeoff analysis, and consequences.

---

## `/split` — Break Down Large Specs

Decomposes a spec that's too large for a single delivery into smaller,
independently deliverable child specs. Routes to the appropriate delivery lead.

```bash
# Split by slug
/split team-permissions

# Split a platform spec
/split database-migration
```

**Decomposition principles:**

- Each child should take no more than 1–2 days to deliver
- Prefer vertical slices (feature/user flow) over horizontal slices (layer)
- Children reference their parent via `relations` in frontmatter
- The parent spec is updated with a `## Child Specs` section

!!! note "Recursive splitting"
    If a child spec is still too large, run `/split` again on that child.
