# Commands Reference

Hero installs slash command definitions into the user's harness. The
commands route natural-language work into repeatable workflows and
delegate to the appropriate agents and skills.

Current installed command definitions: **28**.

```text
/design auth flow for OAuth2 providers
```

Use `/hero <request>` when the intent is unclear. In Hero-aware projects,
plain natural-language asks are routed to these same workflows.

---

## Session and Context

| Command | Description |
|---|---|
| `/resume` | Load a graph-backed session brief: mission, active work, recent changes, blockers, and dead ends. |
| `/handoff` | Refresh NEXT state before switching tools or hitting context limits. |
| `/hero` | Route a natural-language request to the right workflow. |
| `/peer` | Cross-repo peering — list, inspect, call, or hand off to a sibling Hero workspace. |
| `/why <target>` | Trace where a spec, AC, file, or commit came from. |
| `/blocked` | List open features blocked by dependencies or failing criteria. |

## Core Engineering

| Command | Description |
|---|---|
| `/discover` | Explore product direction and possible work. |
| `/design` | Produce a feature, platform, or documentation spec. |
| `/diagnose` | Investigate a bug and produce a fix spec. |
| `/challenge` | Push back on a diagnosis — re-examine the root cause with new context. |
| `/deliver` | Implement and validate an approved spec. |
| `/review` | Review PRs, current changes, security, architecture, tests, or specs. |
| `/scrub` | Remove quality issues such as dead code, weak types, duplication, stale comments, and legacy cruft. |

## Planning

| Command | Description |
|---|---|
| `/compose` | Break an initiative into sequenced child specs. |
| `/split` | Decompose a large spec into smaller deliverable specs. |
| `/sprint` | Select and sequence work for an iteration. |
| `/import` | Import tracker issues into local spec scaffolds. |
| `/release` | Assess release readiness and operational risk. |
| `/retro` | Compare a completed spec to what shipped and capture learnings. |

## Knowledge and Standards

| Command | Description |
|---|---|
| `/convention` | Capture a reusable team convention. |
| `/decide` | Record a decision and tradeoff analysis. |
| `/note` | Capture a conversation, brainstorm, or observation. |
| `/capture` | Extract learnings from the current session. |
| `/docs` | Create or update technical documentation. |
| `/scan` | Detect stack, ingest project state, and seed the graph. |
| `/check` | Run workspace health checks. |

## Design Support

| Command | Description |
|---|---|
| `/mock` | Generate or serve an HTML prototype. |

---

## Examples

```text
/resume
/discover onboarding improvements for new teams
/design add CSV export for user data
/diagnose login times out after 30 seconds
/deliver .hero/planning/features/csv-export/spec.md
/review PR #42
/scrub stale docs and comments
/handoff
```

---

## Notes

- **Slash commands run inside your AI tool only.** They are not `hero <name>`
  terminal commands. Only a subset have CLI equivalents: `/check`, `/deliver`,
  `/design`, `/diagnose`, `/docs`, `/handoff`, `/import`, `/note`, `/scan`,
  `/sprint`, `/why`. All other slash commands (e.g. `/discover`, `/convention`,
  `/review`, `/mock`) exist only in the AI tool.
- `/prime` remains in older installed content as a session-start helper,
  but `/resume` is the current graph-backed warm-start workflow.
- CLI spec operations live under `hero spec ...`; for example
  `hero spec new`, `hero spec claim`, `hero spec verify`, and
  `hero spec complete`.
- Tracker operations live under `hero sync ...`; for example
  `hero sync connect`, `hero sync import`, `hero sync pull`, and
  `hero sync comment`.
- Headless work lives under `hero agent ...`; for example
  `hero agent run`, `hero agent jobs`, and `hero agent approve`.
