---
title: "hero_work MCP tool — the work picture with lanes and next steps"
slug: hero-work-tool
type: feature
status: planning
priority: high
domain: engineering
created: 2026-10-06
parent: hero-read-contract
depends-on: [next-step-engine]
relations:
  - target: hero-spec-handoff-tools
    kind: conflicts-with
---

# hero_work

## Kickoff

Add read-only MCP tool `hero_work {recent_days?: number = 14}` → `HeroWork`, with fields:
- `schema_version`
- `revision`, which changes whenever any field changes
- `generated_at`
- `hero_version`
- `watch_globs`
- `items` (every work spec as `WorkItem` with `next`)

`polish` and `suggested` start empty; `polish-and-suggested` fills them. It must not write under `watch_globs`. Measure latency on this repo.

→ `/design hero-work-tool`

## Scope

- Registered in `internal/serve/mcp_tools_def.go` / `mcp_tools.go` (overlap with `hero-spec-handoff-tools`).
- Include the revision short-circuit and a latency budget well under the peer's 10 s timeout.
