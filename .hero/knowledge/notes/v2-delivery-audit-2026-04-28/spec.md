---
title: V2 Delivery Audit — 2026-04-28
type: note
status: active
tags: [audit, v2, delivery-gaps, evidence, recovery]
created: 2026-04-28
relations:
  - target: get-back-on-track
    kind: evidence
  - target: recovery-strategy-conversation
    kind: companion
horizon: now
---

# V2 Delivery Audit — 2026-04-28

Three parallel Explore-agent audits comparing what the v2 graph-memory
work + cloud/team-server work + killer-features work *promised in the
specs* against what *actually shipped in the code*. Run on 2026-04-28
at the close of the recovery-strategy conversation. Companion to the
`recovery-strategy-conversation` note; this is the evidence base for
the `get-back-on-track` initiative.

## TL;DR

**Across all three slices, the same shape repeats: foundations ship;
the surface that delivers the mission doesn't.** Schemas, queues,
parsers, loaders, frontends, engines — all present. Traversal queries,
OAuth callbacks, automation triggers, notification integrations, write-
time validation, auto-capture — missing or stubbed. Approximately:

- **Graph memory + ingest:** ~60% of the v2 thesis
- **Cloud / team-server:** ~70% of jobs/queue/RLS, ~30% of user-facing
- **Killer-features / commands:** ~70% delivered solidly; key dishonest
  status fields and one architecturally-stranded feature

## Through-the-mission lens

If the mission is **"AI in the editor has the right context at the
right moment, knowledge that compounds"**, then audit findings
recategorize:

- **Mission-critical and broken:** master ingest, traversal queries,
  Tier-2 auto-extraction, cross-spec awareness, projections, write-time
  validation
- **Mission-adjacent (defer):** OAuth, dashboard data wiring,
  notifications integration, automations triggers, NATS, Stripe,
  launch-readiness
- **Process integrity (urgent):** lying status fields on `auto-capture`
  and `graph-schema-simplification`; the phased-plan checkmark table on
  graph-memory marks ✅ shipped on phases that are ~60%

## Audit 1 — Graph memory + ingest (phases 1–10)

### Status by spec

| Spec | Status | Headline gap |
|---|---|---|
| graph-memory (master) | ⚠️ Partial | Scan covers ~60% of promised sources |
| graph-memory-federation | ✅ Delivered | Schema contracts + scope/alias/traverse all present |
| graph-conflict-detection | ✅ Delivered (push side) | Pull-side `hero check conflicts` not wired to graph |
| graph-schema-simplification | 🔻 **Fake delivery** | Phase 7c commit claims it; schema unchanged |
| unified-search | ⚠️ Partial | Sibling-repo ingest works; `(type, key)` collision documented |
| cross-spec-awareness | ❌ Missing | Spec exists, zero code |
| cross-org-intelligence | ❌ Missing | Spec exists, zero code |
| institutional-memory | ❌ Missing | Spec exists, zero code |
| architectural-drift-detection | ❌ Missing | Spec exists, zero code |
| graph-memory-7c-live-test | ✅ Delivered | Live multi-dev sync verified |

### Critical findings

**🔥 `hero why` and `hero blocked` don't exist.** They were the
canonical "graph traversal beats grep" showcase queries from the v2
spec — the entire reason for the substrate. Never built.

**🔥 `hero scan` ingests ~60% of promised sources.** Missing from the
scan flow: notes-as-Note-nodes, tracker, memory files, opportunistic
team-server sync, Tier-2 extraction. Each became its own verb (`extract`,
`sprint load`, `sync graph push`) — sub-verb sprawl the v2 spec
*explicitly forbade*.

**🔥 Tier-2 extraction is opt-in only.** Notes/specs prose never reach
the graph automatically. The spec promised "Tier 1 is enough day one;
Tier 2 enriches over time without blocking anything" — reality requires
the user to remember to run `hero extract`.

**🔻 Schema simplification was claimed but never landed.** Phase 7c's
commit message says "conflict detection + schema simplification". The
audit found schema simplification is missing entirely — both client and
server are still bitemporal. The "simplification" referred to
conflict-detection logic, not the schema.

**Projections half-built.** NEXT.md projects from graph ✅. Code specs
bypass the graph (still come from codescan directly) ❌. MEMORY.md
projection doesn't exist ❌. Sprint section in brief is hand-coded.

## Audit 2 — Cloud / team-server / multi-dev

### Status by spec

| Spec | Status | Headline gap |
|---|---|---|
| hero-team-server | ⚠️ Partial | Job queue solid; OAuth + credential brokering missing |
| team-oauth | ❌ Missing | No browser OAuth flow; password-only auth |
| team-connect | ✅ Delivered | CLI + storage functional |
| team-notifications | ⚠️ Partial | Framework wired; never invoked from ApproveJob |
| team-knowledge-flywheel | 🔻 Deferred | No post-delivery automation |
| tenant-isolation-rls | ✅ Delivered | RLS in place + handler-threaded |
| client-id-user-scoping | ✅ Delivered | JWT claims + user-scoping |
| cloud-admin | 🔻 Incomplete | Org/user CRUD; billing stubs |
| cloud-billing | 🔻 Incomplete | Cost tracking; no Stripe |
| cloud-dashboard | ⚠️ Partial | API endpoints partial; UI lacks live data |
| cloud-dashboard-ui | ⚠️ Partial | Shell exists; SSE + approval buttons unwired |
| cloud-mcp | ❌ Missing | Cross-repo federation MCP not implemented |
| cloud-notifications | ⚠️ Partial | No email; Slack untested |
| cto-dashboard | 🔻 Incomplete | Frontend pages; backend endpoints missing |
| hero-dashboard-v2 | ⚠️ Partial | Pages present; no real-time updates |
| hero-runner | ✅ Delivered | Anthropic works; OpenAI stub; Azure missing |
| hero-automations | ✅ Loader; ❌ triggers | YAML loads; no Jira/webhook/cron/file-watcher |
| nats-event-bus | 🔻 Deferred | Not implemented |
| launch-readiness | ❌ Missing | No deploy guide; no load test; no runbook |

### Production-readiness scores (auditor-assigned)

- Auth: 4/10 (password-only blocks cloud tier)
- Job Queue & Workers: 8/10
- Notifications: 4/10 (framework + no integration)
- Dashboard: 3/10 (shell + no data)
- Automations: 5/10 (engine + no triggers)
- Multi-Provider Support: 5/10
- Deployment: 1/10

## Audit 3 — Killer features + command surface + lifecycle

### Status by spec

| Spec | Status | Headline |
|---|---|---|
| activity-digest (P0) | ✅ Delivered | `hero recap` works, MCP wired |
| impact-analysis (P0) | ✅ Delivered | `hero impact` with cross-repo |
| living-contract (P0) | ✅ Delivered | `verified_by:` parsing + link/check |
| spec-drift-detection | ✅ Delivered | 5 signals, cross-repo aware |
| spec-three-file-split | ✅ Delivered | Parser + discovery |
| nl-event-hooks | ✅ Delivered | `hero hook fire` shim |
| hero-ask | ✅ Delivered | Extractive Q&A + MCP |
| hero-search | ✅ Delivered | FTS5 + graph fallback |
| multi-repo-specs | ✅ Delivered | Resolver + cross-repo drift |
| learned-templates | ✅ Delivered | Pattern extraction |
| cost-calibration | ✅ Delivered | calibration.json model |
| environment-awareness (`hero ci`) | ✅ Delivered (untested live) | GH Actions provider scaffolded |
| hero pulse | ⚠️ Naming fracture | CLI registers as `hero status`; MCP is `hero_pulse` |
| domain-plugin-architecture | ⚠️ 80% | Directory + embed done; `hero domain` CLI missing |
| greenfield-scaffolding | ❌ Missing | Spec status `draft`; no flag, no template |
| **auto-capture learnings** | 🔻 **Fake delivery** | Spec says `completed`; **no implementation anywhere** |
| hero-prime (CLI) | ❌ Missing | Exists only as agent doc, not a binary command |

### Renames — all completed cleanly

`brief→resume`, `nudge→relevant`, `recall→search`, `ingest→import`,
`graph-impact→impact`, `suggestions→suggest`. No stale call sites
detected.

### MCP tool count

35 tools registered (README claims "25+" — verified). Audit-3 question:
*does this number serve principle #5 (don't drown the magic with
practitioner surface)?* Open question for the recovery work.

### Spec-format work — parsed but not validated

EARS criteria, three-file layout, quality scoring, drift: parsers
exist; validation is read-side only (drift detection is *retroactive*).
Author of a malformed spec gets no immediate feedback. This is one of
the things `spec-status-integrity` is meant to fix.

## Pattern catalog

Drift modes the audit revealed (each becomes a thing
`spec-status-integrity` must structurally prevent):

1. **Lying status field** — frontmatter `status: completed` with no
   code (`auto-capture`)
2. **Lying commit message** — commit claims feature X; only delivers
   feature Y (`graph-schema-simplification`)
3. **Phased plan checkmark fraud** — table marks all phases ✅; reality
   averages 60% (`graph-memory`)
4. **Sub-verb sprawl despite explicit ban** — every new ingest got its
   own verb (`hero scan` regression)
5. **Foundation-without-surface delivery** — engine loads YAML, no
   triggers; framework exists, no integration; UI renders, no
   endpoints (most of cloud)
6. **Spec exists, zero code** — `cross-spec-awareness`,
   `institutional-memory`, `architectural-drift-detection`,
   `cross-org-intelligence`, `greenfield-scaffolding`

Each drift mode maps to one or more violations of the experience
principles being formalized in `project-charter`.

## What's still working

The audit is not all bad. The team did real work:

- Federation schema + sync protocol (cleanest piece of v2)
- RLS isolation throughout cloud
- Job queue + worker pool + JWT
- The P0 trio (recap, impact, contract)
- `hero ask` / `hero search` / `hero drift` / `hero impact` /
  `hero recap` / `hero relevant` / `hero suggest` — all real, all
  shipped
- Renames threaded cleanly
- 35 MCP tools registered and dispatching

## How this audit was conducted

Three Explore subagents ran in parallel on 2026-04-28, each with a
focused brief covering one slice. Each was instructed to *not trust*
spec status fields and verify against code via grep + Read of
`internal/cli`, `internal/graph`, `cloud/`, etc. Full agent transcripts
in the session task store; this note is the consolidated synthesis.

## Recommended use

When evaluating any of the affected specs, read this note alongside
the spec — the spec's frontmatter and "what shipped" claims may not
match reality. Until `spec-status-integrity` lands and graph-verifies
delivery status, treat this audit as the authoritative state of v2.
