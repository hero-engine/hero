# Agents, skills, and domain composition

Agents are bounded roles; skills are instruction sets loaded for the work at
hand. Exact inventories vary with the installed domain composition and harness,
so narrative pages describe roles rather than freezing mutable counts.

## Engineering setup

The default project receives Core plus the **Engineering** setup. It includes:

- delivery leads for features and platform work;
- architecture, design, pull-request, security, and roadmap reviewers;
- implementation specialists for API, database, integration, migration,
  performance, operations, release, and documentation work;
- investigation and functional QA roles;
- focused code-health scrubbers;
- project context, conventions, and session warm-start roles.

This setup is **shipped**. Agent execution depends on the active harness and its
permissions. Read-only and write-capable roles follow the boundaries declared
by the installed content; do not infer safety from a role name alone.

## Skills

The implementation agent detects the stack and loads only the relevant
language, framework, testing, and architecture guidance. Workflow skills define
Hero operations such as design, diagnosis, delivery, review, and capture.

Codex and Grok render workflow commands as command skills because they do not
use Claude's command-file surface. That makes per-target skill totals different
even when the underlying workflow inventory is the same.

## Optional domain setups

Focused PM, QA, and Sales setups are **optional** and maturity-bounded. Select
the intended primary domain or supported extension composition, then re-run
installation. The default Engineering setup already contains lightweight PM
and QA assistance used within coding workflows; installing a focused pack is a
different composition choice, not a prerequisite for normal delivery.

## Inspect the actual install

```bash
hero doctor
hero domain
```

`hero doctor` reports expected-versus-installed content for each target at the
running revision. Use that derived output when an exact command, agent, or skill
inventory matters.
