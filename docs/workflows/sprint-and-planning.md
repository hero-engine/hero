# Sprint & Planning

Hero integrates with work trackers and provides sprint planning, retrospectives,
and issue management commands to keep delivery on track.

---

## `/sprint` — Sprint Planning

Plans a sprint by selecting and sequencing specs from a backlog or initiative.
Delegated to the **feature-delivery-lead**.

```bash
# Plan from an initiative
/sprint team-permissions

# Plan from the full backlog
/sprint

# Plan with capacity constraints
/sprint 2 engineers, 1 week
```

The delivery lead:

1. Loads the initiative or runs `hero check` / `hero dashboard` to assess
   workspace state
2. Identifies specs that are ready (dependencies met, not blocked)
3. Suggests scope based on complexity, dependencies, and priority
4. Verifies each spec has clear acceptance criteria and a Changes section
5. Produces a sprint plan at `.hero/knowledge/notes/sprint-{date}/spec.md`

The sprint plan includes a sprint goal, selected specs with delivery order,
dependency graph, risk items, and capacity allocation.

!!! tip "Planning principles"
    - Target 1–2 week sprints
    - Leave 20% buffer for unplanned work
    - Prioritize specs that unblock others
    - If a spec is too large, Hero suggests running `/split` first

### `hero sprint load`

Loads sprint data from your configured tracker (Jira or Linear) into local
specs.

```bash
# Load the current sprint from Jira
hero sprint load

# Load a specific sprint
hero sprint load "Sprint 24"
```

---

## `hero import` — Pull Issues from Trackers

Imports issues from Jira, GitHub, or Linear as spec scaffolds in
`.hero/planning/`. Each imported spec includes tracker-prefixed fields in
its YAML frontmatter (e.g., `jira_status`, `jira_priority`, `jira_assignee`).

```bash
# Import bugs from Jira
hero import --type bug

# Import a specific issue
hero import PROJ-1234

# Import from GitHub
hero import --source github --label "ready"
```

Imported specs are scaffolds — they contain the tracker metadata and
description but need a `/design` or `/diagnose` pass to become actionable.

!!! info "Local-first"
    Hero always checks locally imported specs before querying the tracker.
    When asked to work on multiple items, it selects from local specs using
    `hero search --list --type <type>`.

---

## `hero status` and `hero recap` — Sprint Narrative

Generates a narrative summary of the current sprint's progress.

```bash
hero status
hero recap --since 2w
```

Reports three categories:

- **Done** — specs completed since sprint start
- **In-flight** — specs currently being worked on
- **At-risk** — specs with blockers, stale activity, or missed targets

---

## `hero spec claim` / `hero spec claims` — Work Assignment

Assign and track ownership of specs.

```bash
# Claim a spec for yourself
hero spec claim team-permissions-rbac

# See all current claims
hero spec claims
```

---

## `/retro` — Post-Delivery Retrospective

Runs a retrospective comparing a completed spec against what was actually
implemented. Routes to the appropriate delivery lead based on work type.

```bash
# Retro on a completed spec
/retro team-permissions

# Retro on a platform migration
/retro database-migration
```

The delivery lead:

1. Reads the completed spec
2. Reviews git history to see what was actually delivered
3. Compares spec intent vs. delivered reality:
    - What matched the plan
    - What deviated and why
    - What was harder or easier than expected
4. Identifies learnings — convention updates, new decisions, estimation
   calibration insights

!!! info "Auto-capture"
    When `knowledge.auto_capture` is enabled, `/retro` doesn't just suggest
    conventions and decisions — it writes them directly to `.hero/knowledge/`
    and runs `hero index`. Novel learnings are persisted automatically.
