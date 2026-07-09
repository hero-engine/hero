---
description: Run a workspace health check — convention compliance, stale specs, and project hygiene.
---
Perform a workspace health check. For quick, automated checks, suggest the user run `hero check` from the terminal. For deeper analysis, route to the appropriate agent.

Determine the check type from the request:

- **General health** (no specific focus) — report on:
  - Specs sitting in `planning/` without activity
  - Conventions that may have drifted
  - Decisions that reference outdated context
  - Suggest `hero check` for the CLI-based quick version

- **Convention compliance** (`/check conventions`) — delegate to the install's
  architecture/review agent if one is installed (engineering installs ship
  `architecture-reviewer`); otherwise perform the review directly as the
  session agent:
  - Load active conventions from `.hero/knowledge/conventions/`
  - Assess whether the codebase follows each convention's scope and rules
  - Report violations and drift

- **Drift** (`/check drift`) — run `hero drift --in-flight` to detect spec-vs-code divergence:
  - Missing or renamed files listed in a spec's `## Changes`
  - Acceptance criteria with no related code changes
  - Boundary violations (spec says "does not touch X" but X was modified)

- **Stale specs** (`/check stale`) — scan the planning folders:
  - Identify specs in `planning/` that have not been touched recently
  - Flag specs with no corresponding branch or recent commits
  - Suggest next actions (deliver, update, or archive)

- **Dependencies** (`/check deps`) — delegate to a dependency-analysis agent
  if one is installed (engineering ships `dependency-analyst`); otherwise
  run the audit directly as the session agent:
  - Audit dependency health, vulnerabilities, and license concerns

For general and stale checks, perform the analysis directly. For convention compliance and dependency audits, delegate to the specialist agent if the install ships one, otherwise perform the analysis directly.

Check focus: $ARGUMENTS
