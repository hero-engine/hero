---
title: Environment Awareness — CI/Deployment/Runtime Visibility
slug: environment-awareness
type: initiative
status: planning
tags: [ci, deployment, runtime, observability, agents, provider]
created: 2026-04-22
priority: P2
relations:
  - target: hero-killer-features
    kind: parent
horizon: next
---

## Goal

Give Hero read-only awareness of the environment surrounding the code — CI/CD
pipeline status, deployment state, and runtime health — so agents and humans
can make informed decisions without leaving the Hero workflow. Surface this
data through CLI commands, MCP tools, and the `hero prime` context pipeline.

## Problem

Hero currently has zero visibility into what happens after code leaves the
editor. An agent can write a perfect implementation, push it, and have no idea
that CI has been red for three days, the staging deployment is stuck on a bad
image, or the endpoint it just changed is throwing 500s in production. The
information exists — it lives in GitHub Actions, Grafana, Datadog, deploy
dashboards — but it's siloed away from the spec-driven workflow.

This creates three concrete failure modes:

1. **Blind delivery**: An agent delivers against a spec, CI fails on an
   unrelated flaky test, and nobody notices until a human checks the Actions
   tab hours later.
2. **Deploy confusion**: A spec is marked "completed" but the code hasn't
   actually reached production. There's no way to answer "is this live?" from
   within Hero.
3. **Missing feedback loop**: Runtime errors that relate to a recently
   delivered spec are invisible. The agent that wrote the code never learns
   that its change caused a regression.

Hero already has a provider abstraction for trackers (`internal/tracker/`)
that supports GitHub, Jira, and Linear through a common interface. The same
pattern applies here: different teams use different CI providers, deploy
strategies, and observability stacks, so the design needs a provider interface
from day one.

## Design

### Provider interface pattern

Follow the same shape as `internal/tracker/tracker.go`: define a `Provider`
interface with methods like `PipelineStatus()`, `DeploymentStatus()`, and
`RuntimeHealth()`, then implement concrete providers per service. Configuration
lives in `hero.json` alongside existing tracker config.

```
internal/environment/
  provider.go        — Provider interface + registry
  github_actions.go  — GitHub Actions provider (first implementation)
  gitlab_ci.go       — future
  datadog.go         — future
  grafana.go         — future
```

### hero.json configuration

```json
{
  "environment": {
    "ci": {
      "provider": "github-actions"
    },
    "deploy": {
      "provider": "git-tags",
      "environments": ["staging", "production"]
    },
    "runtime": {
      "provider": "datadog",
      "service": "my-api"
    }
  }
}
```

### Start with GitHub Actions

The tracker integration already has GitHub API access (`internal/tracker/github.go`).
Reuse the same authentication and client setup. GitHub Actions is the most
common CI for the target audience and lets us prove the provider interface
before adding others.

## Children

| Slug | Title | Priority | Summary |
|---|---|---|---|
| `ci-status` | CI pipeline status command and MCP tool | P1 | `hero ci` queries the CI provider for current pipeline status. "Last run: failed on test X in commit abc123." Exposed as `hero_ci` MCP tool. Start with GitHub Actions. |
| `deploy-status` | Deployment state visibility | P2 | `hero deploy status` shows what's deployed where — which commit/tag is live in each environment. Uses provider APIs or git tag conventions. |
| `runtime-health` | Runtime observability integration | P2 | Integration with Grafana, Datadog, or similar to surface "this endpoint is throwing 500s" alongside the spec that delivered it. |

## Sequencing

1. **ci-status** first — highest signal, smallest scope, reuses existing
   GitHub API auth. Proves the provider interface pattern.
2. **deploy-status** second — builds on the provider registry from ci-status.
   Git-tag-based provider can ship without any external API dependency.
3. **runtime-health** last — biggest integration surface, most variation
   across teams, depends on the provider pattern being proven by 1 and 2.

## Acceptance Criteria

- WHEN `hero.json` contains an `environment.ci` configuration block THE SYSTEM SHALL query the configured CI provider and display the current pipeline status for the active branch, including pass/fail state and failing step details
- WHEN an agent begins a delivery session and a CI provider is configured THE SYSTEM SHALL include recent CI status in the `hero prime` context output so the agent knows whether the pipeline is healthy before making changes
- WHEN multiple environment providers are configured (CI, deploy, runtime) THE SYSTEM SHALL surface each through a consistent provider interface, allowing teams to mix providers without code changes
- WHERE a team uses a CI provider not yet supported THE SYSTEM SHALL provide a documented provider interface that allows adding new providers without modifying existing commands or MCP tool schemas
- WHEN environment data is requested via MCP tools THE SYSTEM SHALL return structured JSON matching a stable schema so host tools can render status inline

## Boundaries

- Does **not** replace CI/CD tools — Hero is a read-only consumer of status, not a pipeline orchestrator
- Does **not** trigger deployments — no `hero deploy push` or `hero deploy rollback`; strictly read-only visibility
- Does **not** require self-hosting — all providers connect to existing SaaS APIs (GitHub, GitLab, Datadog, etc.)
- Does **not** store environment data persistently — status is fetched on demand, not synced to SQLite
- Does **not** require a cloud account — all environment commands work locally against provider APIs using existing credentials
