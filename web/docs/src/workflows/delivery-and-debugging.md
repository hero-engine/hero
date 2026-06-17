# Delivery & Debugging

Hero's delivery and debugging commands turn specs into working code and
investigate bugs with structured evidence gathering.

---

## `/deliver` — Execute a Spec

Implements the planned changes from an approved spec. The delivery lead
generates context, selects the right engineering agents, and coordinates
implementation against the spec's acceptance criteria.

### Pre-flight checks

Before any code is written, `/deliver` runs a series of checks:

1. **Locates the spec** — by path, slug, tracker ID, or description
2. **Checks status** — stops if the spec is completed or superseded; warns if
   it's still in draft or proposed
3. **Syncs with tracker** — runs `hero sync pull` to get the latest status from
   Jira, GitHub, or Linear

!!! warning "Won't work on closed items"
    If the tracker issue is closed or the spec status is `completed` /
    `superseded`, `/deliver` stops immediately. This prevents wasted effort
    on already-resolved work.

### Delivery workflow

```
Spec → Delivery Lead → Engineering Agents → Verify Gate → Archive
```

1. The **delivery lead** reads the spec and generates implementation context,
   loading relevant conventions and history from `.hero/knowledge/`
2. **Engineering agents** implement the changes — Hero selects from specialists
   like `engineer`, `api-engineer`, `database-engineer`, `migration-engineer`,
   and others based on the work type
3. **Verify gate** — `hero spec verify` checks all acceptance criteria against the
   delivered state before the spec can move to `completed`. This is a hard
   checkpoint: a spec cannot be archived without passing it.
4. The completed spec moves from `.hero/planning/` to `.hero/specs/`
5. **Tracker sync** updates the issue status
6. **Knowledge capture** persists any learnings from the implementation

### The verify gate

`hero spec verify <slug>` is the required checkpoint between "implementation done"
and "spec closed." It checks four gates: completion ledger, audit report, test
coverage, and build. All must pass before the spec status flips to `completed`
and the spec is archived.

```bash
# Run the verify gate on a spec
hero spec verify csv-export-reports

# Skip re-running the test suite if tests just passed
hero spec verify csv-export-reports --skip-tests

# Structured output for CI or scripted checks
hero spec verify csv-export-reports --json
```

!!! warning "Don't skip the gate"
    Flipping `status: completed` manually in frontmatter bypasses the four-gate
    check. Use `hero spec verify` — it's the only path to `completed` that produces
    a durable audit record and triggers archiving.

```text
# Deliver by slug
/deliver team-permissions-rbac

# Deliver by tracker ID
/deliver PROJ-1234

# Deliver by description (Hero finds the matching spec)
/deliver the CSV export feature
```

!!! tip "Conventions matter"
    The delivery lead loads project conventions before handing off to engineers.
    Run `/convention` first to codify patterns if the codebase doesn't have
    conventions documented yet.

---

## `/diagnose` — Investigate Bugs

A two-phase workflow that traces bugs to their root cause with evidence, then
produces a fix spec. Designed for thorough investigation, not quick patches.

### Pre-flight checks

Same as `/deliver` — finds the spec, checks status, syncs with tracker.

### Phase 1: Deep Investigation

Delegated to the **debug-investigator** agent, who:

1. Traces the code path end-to-end
2. Identifies the root cause with file-and-line evidence
3. Writes an investigation report including:
    - Code flow analysis
    - Key files table with specific line references
    - Secondary defects discovered along the way

!!! danger "Quality gate"
    The investigation must pass a strict evidence gate before proceeding.
    The debug-investigator must provide concrete `file:line` references for
    every claim. Vague descriptions like "somewhere in the auth module" are
    rejected.

### Phase 2: Fix Planning

After the quality gate passes, the **delivery lead**:

1. Re-reads the source files cited in the investigation
2. Classifies the root cause (code / data / environment / user error /
   external / race condition / design flaw)
3. Writes a fix plan with before/after code diffs
4. Produces a test plan
5. Saves the fix spec
6. Posts a summary comment and attaches the report to the tracker

```text
# Diagnose by slug
/diagnose cart-total-rounding-bug

# Diagnose by tracker ID
/diagnose BUG-567

# Batch mode — diagnose multiple imported bugs
/diagnose 5 bugs
```

### Batch mode

When given a count (e.g., "5 bugs"), Hero selects from locally imported bug
specs and diagnoses them sequentially. It uses `hero search --list --type bug`
to find candidates — it never bulk-queries the tracker.

!!! example "Investigation report structure"
    ```markdown
    ## Investigation Report

    ### Code Flow
    1. Request enters at `api/handlers/cart.go:45`
    2. Calls `calculateTotal()` at `services/cart.go:112`
    3. Rounding applied at `services/cart.go:128` — uses `int()` truncation
       instead of `math.Round()`

    ### Key Files
    | File | Lines | Role |
    |---|---|---|
    | `services/cart.go` | 112–135 | Total calculation |
    | `models/cart.go` | 22–30 | Price type definition |

    ### Root Cause
    Integer truncation in price calculation loses fractional cents.

    ### Secondary Defects
    - Missing unit test for fractional price totals
    ```
