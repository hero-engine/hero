---
title: "Execution Plan — Local Finish + Cloud Launch"
type: plan
status: active
created: 2026-04-19
horizon: next
---

# Execution Plan

## Current State

### Shipped (v0.5.15+):
- Full spec-driven workflow: /design, /deliver, /diagnose, /compose, /review
- Tracker integration: Jira, GitHub, Linear (import, sync, post)
- Code intelligence suite: multi-language parsers (Go, JS/TS, Python, Java, Rust, C/C++, Ruby, Groovy), scanner, knowledge generator, search, MCP tools
- Config/env map, error pattern catalog, tree-sitter backend, deep mode enrichment
- 18 MCP tools, session primer, batch delivery mode
- Batch bug diagnosis via /diagnose (synchronous, tedious prompting required)

### Local work remaining:
- Update spec statuses for already-built features (config-env-map, deep-mode, error-pattern-catalog, tree-sitter)
- Test codescan on Morpheus (real validation of all quality fixes)
- API surface index (planning status, not yet built)
- Spec quality scoring (prerequisite for async)
- Hand delivery workflow (`hero deliver --manual`)

### Cloud work:
- Existing specs for auth, API, sync, dashboard, billing, admin, MCP, notifications
- New vision specs: governance layer, architectural drift, institutional memory, CTO dashboard, cross-org intelligence

---

## Phase 0: Housekeeping (1 day)

**Goal:** Clean up statuses, validate what we shipped.

- [ ] Update feature spec statuses: config-env-map → completed, error-pattern-catalog → completed, tree-sitter → completed, deep-mode → completed
- [ ] Run codescan on Morpheus, verify quality fixes work at scale
- [ ] Tag v0.5.16 with all current fixes

---

## Phase 1: Local Polish (1 week)

**Goal:** Finish the local features that make Hero solid before cloud work. These also feed into cloud.

### 1a. Spec Quality Scoring (2-3 days)
- Scoring engine: acceptance criteria, scope clarity, technical specificity, ambiguity, test strategy
- `hero score <slug>` CLI command
- `/deliver` warns on low-quality specs, blocks below configurable threshold
- `hero_score` MCP tool for agents
- **Why first:** prerequisite for async delivery, teaches users to write better specs

### 1b. Hand Delivery Workflow (1 day)
- `hero deliver --manual <slug>` — marks spec in-progress, gives developer the spec, waits for them to mark done
- Acceptance criteria runner: `hero verify <slug>` checks if the spec's criteria are met
- Status tracking: same lifecycle (draft → approved → in-progress → delivered) regardless of agent vs manual
- **Why now:** core philosophy, must work before governance layer cares about delivery method

### 1c. Batch Diagnosis Pipeline (1-2 days)
- `hero diagnose --batch` — works through all imported bug specs automatically
- Per-bug: investigate, produce fix spec, update status, move to next
- Progress reporting: `hero status` shows batch progress
- Rate limiting for tracker API calls
- Failure isolation: one bug failing doesn't block the rest
- **Why now:** removes the #1 daily pain point (tedious prompting for batch bugs)

### 1d. API Surface Index (1 day)
- Detect REST/GraphQL/gRPC endpoints, map to handlers
- Add to code knowledge output
- **Why now:** feeds into governance layer (scope drift detection needs to know what endpoints exist)

---

## Phase 2: Cloud Foundation (2-3 weeks)

**Goal:** Stand up the cloud service. Auth, API, sync working end-to-end.

### 2a. Project Setup
- Separate repo or monorepo with `cloud/` directory? (decision needed)
- Stack: Go API server, Postgres, minimal web framework
- Infrastructure: start with single-server deployment (fly.io or similar), scale later
- Per-org data isolation from day one

### 2b. Cloud Auth (cloud-auth spec exists)
- GitHub OAuth login (primary)
- Org creation, team management
- API key generation for CLI
- `hero login` CLI command

### 2c. Cloud API (cloud-api spec exists)
- REST API for spec CRUD, knowledge sync, status reporting
- Org-scoped endpoints
- Webhook support for events

### 2d. Cloud Sync (cloud-sync spec exists)
- `hero sync` pushes spec metadata + knowledge to cloud
- Pull aggregated views back
- Conflict handling: git is source of truth, cloud is read-mostly
- Incremental sync (checksums, like codescan)

---

## Phase 3: Async Delivery (2-3 weeks)

**Goal:** The "diagnose 30 bugs and go to lunch" workflow.

### 3a. Local Async (background process)
- `hero deliver --async <slug>` or `hero deliver --batch --async`
- Background process with progress file
- `hero status` polls progress
- Branch-per-spec isolation
- PR creation on completion
- **Start here:** no cloud dependency, validates the model

### 3b. Cloud Async
- `hero deliver --cloud <slug>` pushes spec to cloud for remote execution
- Cloud agent picks up, executes in sandboxed environment
- Progress via cloud API, CLI polls with `hero status --remote`
- Webhook notifications on completion
- **Depends on:** Phase 2 (cloud foundation)

### 3c. Batch Pipeline
- Full flow: `hero import` → `hero diagnose --batch --async` → review → approve → `hero deliver --batch --async`
- Dashboard shows pipeline status across all specs
- Mixed mode: some specs async-agent, some hand-delivered, unified view

---

## Phase 4: Governance Layer (2-3 weeks)

**Goal:** Enterprise sale. "Your developers are using AI. Hero makes sure that's a good thing."

### 4a. GitHub App — PR Spec Linkage
- GitHub App that checks PRs for spec references
- Advisory mode first (comments), enforcement mode later (blocks)
- Links PR to spec in cloud dashboard
- **This is the enterprise wedge.** Ship this early, even before full governance.

### 4b. Convention Compliance
- PR bot checks changes against .hero/knowledge/conventions
- Scope drift detection: PR touches files outside spec scope → flag
- Architectural constraint validation (from drift detection spec)

### 4c. Audit Trail
- Every spec lifecycle event logged: created, scored, approved, delivered, merged
- Who approved, who delivered (agent vs human), when
- Exportable for compliance (SOC2, etc.)

---

## Phase 5: Dashboard + Analytics (2-3 weeks)

**Goal:** The CTO sale. Prove AI is working.

### 5a. Cloud Dashboard (cloud-dashboard spec exists)
- Web UI: cross-repo spec status, team activity, in-flight work
- Knowledge search across repos
- Convention compliance overview

### 5b. Spec Analytics (CTO dashboard spec)
- Spec fidelity, rework rate, time-to-merge
- AI leverage metrics (agent vs manual delivery)
- Spec quality → delivery success correlation
- Team/repo comparisons

---

## Phase 6: Intelligence Layer (ongoing)

**Goal:** The moat. Knowledge that gets smarter over time.

### 6a. Cross-Spec Awareness
- Detect file overlap, API conflicts, shared patterns between specs
- Surface warnings during /deliver
- Suggest sequencing

### 6b. Institutional Memory
- Pattern mining from session activity
- Proactive warnings: "last 4 times someone modified this, they broke X"
- Auto-generated conventions from observed patterns

### 6c. Cross-Org Intelligence (long-term)
- Anonymized pattern aggregation across customers
- "Projects like yours typically need these conventions"
- Network effect moat

---

## Phase 7: Monetization + Enterprise (parallel with Phase 5-6)

### 7a. Billing (cloud-billing spec exists)
- Stripe integration
- Free / Team / Enterprise tiers
- Seat management

### 7b. Enterprise Features
- SSO/SAML
- Audit log export
- Self-hosted option
- Custom agent marketplace

---

## Sequencing Summary

```
NOW        Phase 0: Housekeeping (1 day)
Week 1     Phase 1: Local polish (spec scoring, hand delivery, batch diagnosis, API index)
Week 2-4   Phase 2: Cloud foundation (auth, API, sync)
Week 5-7   Phase 3: Async delivery (local first, then cloud)
Week 7-9   Phase 4: Governance layer (GitHub App, compliance, audit)
Week 9-11  Phase 5: Dashboard + analytics
Week 11+   Phase 6: Intelligence layer (ongoing)
Parallel   Phase 7: Monetization + enterprise
```

## Key Dependencies

- Spec quality scoring → async delivery (quality gate before handoff)
- Cloud auth + API → cloud sync → cloud async delivery
- Cloud sync → dashboard → analytics
- Governance layer needs GitHub App infrastructure
- Institutional memory needs enough session volume to mine patterns
- Cross-org intelligence needs enough customers to aggregate

## Key Decisions Needed

1. Cloud repo structure: monorepo or separate repo?
2. Cloud hosting: fly.io, AWS, GCP?
3. GitHub App vs GitHub Actions for governance?
4. Local async: background process vs daemon?
5. OpenClaw integration: build on it or build our own agent runtime?
