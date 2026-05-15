# Review & Quality

Hero provides specialized review and code quality commands that are entirely
**read-only** — they analyze and report but never modify code unless explicitly
asked.

---

## `/review` — Multi-Perspective Reviews

Routes review requests to the appropriate specialist agent based on the type
of review needed.

### Review agents

| Agent | Focus |
|---|---|
| `pr-reviewer` | Pull request diffs — correctness, style, test coverage |
| `architecture-reviewer` | System design, coupling, scalability concerns |
| `design-reviewer` | Spec quality, completeness, feasibility |
| `security-reviewer` | Vulnerabilities, auth issues, data exposure |
| `functional-qa-engineer` | Functional correctness, edge cases, user flows |
| `test-architect` | Test strategy, coverage gaps, test design |
| `dependency-analyst` | Dependency health, vulnerabilities, license risks |

```bash
# PR review (most common)
/review PR #42

# Design review of a spec
/review design team-permissions

# Security audit
/review security the authentication module

# Architecture review
/review architecture the event processing pipeline
```

If the review type is ambiguous, Hero asks for clarification. When a spec slug
is provided, the agent loads the spec for additional context.

!!! info "Read-only"
    All review agents produce reports and recommendations only. They do not
    modify files, create branches, or push commits.

---

## `/scrub` — Codebase Health

Scans the codebase for quality issues across seven concern areas, each handled
by a dedicated scrubber agent.

### Scrubber agents

| Concern | Agent | What it finds |
|---|---|---|
| `dead-code` | `deadcode-scrubber` | Unreachable functions, unused exports, orphaned files |
| `duplication` | `dedup-scrubber` | Copy-pasted logic, near-duplicate functions |
| `types` | `type-scrubber` | Weak types, `any` casts, missing type annotations |
| `defensive` | `defensive-scrubber` | Unnecessary nil checks, dead error branches, over-guarding |
| `legacy` | `legacy-scrubber` | Deprecated APIs, outdated patterns, stale workarounds |
| `dependencies` | `dependency-scrubber` | Unused deps, version drift, known vulnerabilities |
| `comments` | `comment-scrubber` | Stale comments, TODO archaeology, misleading docs |

### Workflow

Each scrubber follows the same four-step process:

1. **Research** — scan the codebase for issues in its concern area
2. **Assess** — rank findings by severity and confidence
3. **Implement** — fix high-confidence issues automatically
4. **Verify** — run the build and tests to confirm nothing broke

```bash
# Scrub a single concern
/scrub dead-code

# Scrub everything
/scrub all

# Scrub types in a specific area
/scrub types in the API layer
```

!!! note "Pre-flight"
    `/scrub` loads the `code-scrub` and `stack-detection` skills, then
    verifies the build passes before starting. If the build is already broken,
    it stops and reports the issue.

### Post-scrub

After all scrubbers complete:

- A final build and test run confirms overall health
- A summary lists all changes by concern area
- Learnings are captured to `.hero/knowledge/` if auto-capture is enabled

!!! example "Scrub summary"
    ```
    Scrub complete — 14 issues found, 11 fixed automatically.

    dead-code:    3 removed (2 unused functions, 1 orphaned test helper)
    duplication:  2 consolidated (shared validation logic)
    types:        4 strengthened (any → concrete types)
    defensive:    2 removed (impossible nil checks)
    comments:     3 updated (stale TODOs from 2024)
    dependencies: 0 issues
    legacy:       0 issues

    Build: ✓  Tests: 142 passed, 0 failed
    ```
