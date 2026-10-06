---
title: "hero_spec and hero_handoff MCP tools"
slug: hero-spec-handoff-tools
type: feature
status: planning
priority: high
domain: engineering
created: 2026-10-06
parent: hero-read-contract
depends-on: [next-step-engine]
relations:
  - target: hero-work-tool
    kind: conflicts-with
---

# hero_spec and hero_handoff

## Kickoff

Add two read-only MCP tools:
- **`hero_spec {slug}`** → `{ item: WorkItem, body (Markdown without frontmatter), relations { parent, children, depends_on, blocks, related } as Ref {slug,title,type,status}, acs [{id,text,state: pass|fail|unknown}] }`.
- **`hero_handoff {}`** → `{ markdown, updated_at }`, the `hero next` briefing.

Both are `readOnlyHint`, re-index when stale, and write nothing.

→ `/design hero-spec-handoff-tools`

## Scope

- **Reuse:** `hero_read_spec` parsing, `internal/acceptance` states, and the NEXT projection.
- **Overlap:** same files as `hero-work-tool` (`internal/serve/mcp_tools*.go`). Deliver one at a time.
