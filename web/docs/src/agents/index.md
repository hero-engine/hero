# Agents Reference

Hero delegates work to **34 specialized agents**, each with a focused role and constrained set of tools. Commands orchestrate agents — you rarely invoke them directly.

Agents are defined as markdown files in the `agents/` directory and loaded at runtime based on the active command workflow.

---

## Orchestration

Lead agents that coordinate other agents to complete complex workflows.

| Agent | Role |
|---|---|
| `feature-delivery-lead` | Coordinate agents to design features and diagnose bugs; produces spec documents |
| `platform-delivery-lead` | Coordinate agents for migrations, refactors, platform changes, and scaling work |
| `product-ideator` | Explore product direction, brainstorm features, produce prioritized work items |
| `session-primer` | Load session context — what's in progress, active conventions, decisions, and watch-outs. Read-only. |

## Architecture

System design and architectural review.

| Agent | Role |
|---|---|
| `brownfield-architect` | Design minimal, scale-ready changes that fit an existing codebase |
| `greenfield-architect` | Design new products/systems with simple starting architectures and clean scale path |
| `architecture-reviewer` | Review architecture for overengineering, scale dead ends, and operational risk |

## Engineering

Implementation agents that write and validate code.

| Agent | Role |
|---|---|
| `engineer` | Execute approved specs into minimal, correct, tested code changes |
| `database-engineer` | Design and implement schema, query, migration, and data workflow changes |
| `api-engineer` | Design and implement APIs with strong contract discipline and compatibility awareness |
| `integration-engineer` | Implement and harden external integrations, webhooks, and system boundaries |
| `performance-engineer` | Investigate and improve performance bottlenecks with practical optimization tradeoffs |
| `migration-engineer` | Plan and execute data/API/library migrations with rollback and zero-downtime strategies |
| `test-architect` | Design test strategies — what tests are needed, where boundaries fall, coverage ROI |

## Design

Visual design and prototyping.

| Agent | Role |
|---|---|
| `ui-designer` | Design and generate visual UI mockups as self-contained HTML prototypes |

## Review

Quality assurance and code review.

| Agent | Role |
|---|---|
| `functional-qa-engineer` | Validate behavior against requirements, identify regressions, strengthen coverage |
| `pr-reviewer` | Review PRs for bugs, regressions, missing tests, and overcomplicated design |
| `debug-investigator` | Reproduce issues, trace code flows, narrow hypotheses, identify definitive root cause |
| `security-reviewer` | Review code for auth, data exposure, input handling, and security risk |
| `dependency-analyst` | Evaluate library choices, dependency health, license compatibility, vulnerability exposure |
| `design-reviewer` | Review spec designs for completeness, feasibility, and consistency before delivery |

## Operations

Release, deployment, documentation, and project management.

| Agent | Role |
|---|---|
| `release-engineer` | Prepare and validate releases, versioning, changelogs, and deployment readiness |
| `devops-engineer` | Improve CI/CD, deployment, environment, and operational setup |
| `documentation-engineer` | Create and update technical documentation reflecting how the system actually works |
| `project-context-builder` | Analyze a codebase and create/improve project instruction files (e.g. AGENTS.md) |
| `issue-tracker` | Maintain local issue queue reports from the tracking system |
| `convention-author` | Analyze codebase patterns and produce convention specs with examples and anti-patterns |

## Code Health

Specialized scrubber agents for targeted codebase cleanup.

| Agent | Role |
|---|---|
| `deadcode-scrubber` | Find and remove unused functions, types, constants, imports, unreachable code |
| `dedup-scrubber` | Find and consolidate duplicated code; apply DRY without adding indirection |
| `type-scrubber` | Replace weak types (`any`, `interface{}`, `unknown`) with strong types; consolidate duplicates |
| `defensive-scrubber` | Remove unnecessary error-swallowing, panic recovery, and fallbacks that hide bugs |
| `legacy-scrubber` | Find and remove deprecated, legacy, and fallback code |
| `dependency-scrubber` | Untangle circular dependencies, reduce coupling, simplify import graphs |
| `comment-scrubber` | Remove AI slop, narrating comments, stubs, and misleading documentation |

---

## How agents work

Each agent definition includes:

- **Role description** — what the agent is responsible for
- **Tool permissions** — which tools the agent can use (read-only for reviewers, read-write for engineers)
- **Workflow instructions** — step-by-step process the agent follows
- **Output format** — what the agent produces (specs, code changes, reports)

Agents run within the context of the AI coding tool's model. They inherit the project's knowledge base, active conventions, and session context loaded by `/resume`.

!!! info "Agent delegation"
    Orchestration agents (delivery leads, product ideator) delegate to other agents. For example, `/design` activates the `feature-delivery-lead`, which may delegate to `brownfield-architect`, `api-engineer`, and `test-architect` depending on the feature scope.
