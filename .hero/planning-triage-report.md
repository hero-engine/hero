# Hero planning triage — what belongs where after the split

Survey date: 2026-05-16. Survey scope: every spec under
`.hero/planning/{features,initiatives,bugs}/*/spec.md` in the **hero**
repo. Archived specs in `.hero/specs/` were intentionally skipped.

## Summary

- Total specs surveyed: **108** (88 features + 14 initiatives + 6 bugs)
- Stays in hero: **70**
- Moves to hero-cloud: **18** (of which **0** are `status: delivering` — no immediate active-work disruption)
- Stays in hero as umbrella with hero-cloud children: **4**
- Belongs in hero-code: **0** (one possible flag — see notes)
- Unclear / needs user judgment: **6**

The split lands well: the most disruptive bucket (active `delivering`
work) is **entirely** CLI/local code. The only `delivering` specs that
touch server-side concerns are validation/test suites against an
already-running cloud (which logically still live in hero as test
harness owners). No `delivering` spec needs a mid-flight pause-and-migrate.

The real volume in "moves to hero-cloud" comes from the `cloud-*` and
`team-*` planning specs, plus the federation/sync server-side schema
work. Almost all of it is `planning` or `draft` — safe to move whenever.

A subtle wrinkle worth flagging up front: the `hero-platform` initiative
declares that **"Hero Cloud is hero-team-server hosted by us"** and
`hero-community-edition` is **"a build target of the `hero-cloud`
codebase, not a fork."** That means the server-side code powering
`hero serve --team`, `hero run`, `hero-automations`, `agent-outposts`,
and the team coordination layer all logically belong in the
`hero-cloud` repo even though their *CLI client-side affordances*
(`hero connect`, `hero login`, etc.) stay in hero. Several of those
specs are written today as if the server lives in hero. Classifying
them as "moves to hero-cloud" reflects the post-split intent.

---

## Moves to hero-cloud (clear cases — all `planning`/`draft`/`approved`)

Server-side / multi-tenant / SaaS / governance-enforcement specs that
clearly belong in the cloud repo:

- `cloud-admin` (planning) — SSO/SAML, audit logging, compliance controls, org policy. Cloud-tier admin features.
- `cloud-billing` (planning) — Stripe subscription billing, seats, invoices, feature gating by tier. Pure cloud-server concern.
- `cloud-dashboard` (planning) — Hosted web UI of team state across all repos in an org. Cloud product surface.
- `cloud-dashboard-ui` (approved) — Preact/HTM SPA embedded in the Go cloud backend. Lives next to the backend that serves it.
- `cloud-mcp` (planning) — Hosted MCP server federating knowledge across an org's repos. The flagship paid feature.
- `cloud-notifications` (planning) — Slack/email/dashboard notifications driven by the cloud server.
- `tenant-isolation-rls` (planning) — Postgres RLS migrations on the cloud DB schema. Pure server work.
- `graph-schema-simplification` (planning) — CockroachDB bitemporal-to-upsert schema redesign on the server.
- `nats-event-bus` (planning) — JetStream alongside CockroachDB on the cloud server.
- `hero-team-server` (planning) — Server-side job queue + approval gates + team coordination. Per `hero-platform`, this IS the cloud server in its self-hostable form.
- `hero-runner` (planning) — Headless agent execution via Claude API. Currently scoped as a `hero` CLI mode, but the headless runner is the execution engine that hero-cloud / CE hosts. Worth a user judgment call (see "Unclear"); leaning cloud.
- `hero-automations` (planning) — Declarative trigger-to-action engine that runs in `hero serve`. Belongs with the server.
- `hero-dashboard-v2` (planning) — Visual web UI served by `hero serve`. Same fate as `hero-team-server`: it's the server's UI.
- `team-connect` (planning) — CLI registration with team server. Half of this is CLI-side (`hero connect`) and stays in hero; the other half is server-side endpoints and lives in cloud. Mark as "split" if you want surgical, otherwise the design spec moves to cloud as the canonical home with a CLI-side child later.
- `team-oauth` (planning) — GitHub/Google SSO for team server auth. Server-side.
- `team-notifications` (planning) — Webhook alerts for job events from the team server. Server-side.
- `client-id-user-scoping` (planning) — Federation sync protocol identity bug. Server-side push/pull behavior; lives with the federation server.
- `agent-outposts` (planning) — Two-scope outpost registry with scoped credentials and audit-by-construction. Strongly governance/audit-shaped; depends on multi-tenant identity + audit infra that lives in cloud. Could be argued as hybrid; classifying as cloud since the audit/identity/scope machinery is cloud-domain. Worth user judgment.

## Moves to hero-cloud (needs care — currently `delivering`)

**None.** No `delivering` spec is clearly cloud-side. The closest case is
`graph-memory-7c-live-test` (live multi-dev sync validation against a
running cloud server) and `e2e-validation` family — but those are *test
harnesses* that exercise the cloud from the CLI's perspective. They stay
in hero. So the split has no active-delivery disruption risk.

## Stays in hero as umbrella with hero-cloud children

These are parent/design specs whose vocabulary or contract design lives
in hero, with enforcement/server-side children landing in hero-cloud:

- `hero-governance` (planning) — Explicitly umbrella: the *vocabulary* (classification, subjects, principals, scopes, audit events, retrieval API signature) lives in hero contracts; the *enforcement engine* lives in hero-cloud. The spec itself says this. Stays in hero as the design home.
- `hero-cloud` (initiative, in-progress) — The umbrella initiative for the cloud platform. The design *story* of Hero Cloud has historically lived in the hero repo as the product roadmap. Two valid options: (a) move the initiative spec to hero-cloud as that repo's top-level roadmap, or (b) keep here as the cross-repo product umbrella with implementation children landing in hero-cloud. Leaning (a) now that hero-cloud is its own repo. Worth user judgment.
- `pre-launch-hardening` (initiative, delivered) — Federation polish + security + observability. Children (`tenant-isolation-rls`, `client-id-user-scoping`, `unified-search`) span both repos. Already `delivered`, so this is archival; can either move with its children or stay as a historical record in hero. Recommend: stay in hero, mark as historical.
- `launch-readiness` (initiative, planning) — Telemetry, production deploy, public-use polish. Telemetry (`hero-telemetry`) is a CLI concern; production deploy is a cloud concern. Hybrid by construction. Stays in hero as the cross-cutting umbrella.

## Belongs in hero-code (if any)

**None obviously.** No spec talks about Rust workspaces, GPUI, or the
AI-native code editor surface.

One borderline mention worth flagging: the `hero-domains` initiative's
"Pick up at" line references `domains/engineering/` refactor as a hero
concern, but mentions handoff docs to hero-code for some PM artifacts.
That's coordination metadata, not a spec that belongs in hero-code.

## Unclear / needs user judgment

- `hero-runner` (planning) — Currently written as a `hero run` CLI mode that calls the Claude API directly. The headless API-driven runner is *also* the execution substrate for cloud-hosted agent work. Options: (a) split into "hero run CLI surface" (stays) and "runner execution engine" (moves to cloud); (b) keep whole in hero as a CLI feature that hero-cloud later embeds. Recommend: keep in hero for now since the spec ships a CLI command; revisit when cloud needs to embed the runner.
- `hero-team-server` (planning) — Sized as a `hero serve` upgrade in the hero CLI. Per `hero-platform`'s thesis, this *is* the server that hero-cloud hosts. Strongest case for moving to cloud, but the user may want it to stay in hero as the self-hostable server reference and have hero-cloud build *on* it. Recommend: move to cloud as the canonical home, since hero-community-edition already says CE is a build of the hero-cloud codebase.
- `agent-outposts` (planning) — Outpost registry with scoped credentials and audit-by-construction. Depends on cloud-shaped infra (multi-tenant identity, audit trail). Could be argued as a CLI feature (project-scoped outposts in a local workspace) or a cloud feature (global outposts with org-scoped credentials). Recommend: read the full spec with the user — the design likely covers both scopes and may want to live in hero as an umbrella with cloud children.
- `hero-cloud` (initiative) — See "umbrella" note above. Move the spec to hero-cloud as that repo's roadmap, or keep here as a hand-off pointer.
- `hero-platform` (initiative, planning) — Cross-repo by construction: defines `hero run` (CLI side) + `hero serve --team` (server side) + automations + dashboard + cloud relation. Hybrid umbrella. Stays in hero as the platform initiative, with implementation children routed to the right repo.
- `cross-repo-peering` (delivering) — Peer-to-peer between two `hero` CLIs (workspace A calls workspace B without going through cloud). Lives squarely in hero. Flagging only because the tags include `local-first` and reference `hero-cloud-split`. Classification: **stays in hero.** No action needed; listed here to make the call explicit.

## Stays in hero (the clear pile)

Brief list — these are CLI / local engine / contracts / spec workflow /
scan-index-search / satellite / documentation / landing / distribution
work. Listed for completeness, no rationale unless surprising.

**Features (planning/draft/completed):**
- acceptance-criteria-graph (completed)
- architectural-drift-detection (draft)
- codex-trust-nudge (completed)
- configurable-content-paths (completed)
- configurable-workspace-location (planning)
- core-vertical-layering (planning)
- coverage-heuristic-fix (completed)
- cross-repo-peering (delivering)
- cross-spec-awareness (draft)
- domain-plugin-architecture (planning)
- domain-routing-and-agents (planning)
- domain-scoped-knowledge-graph (planning)
- e2e-area-suites (planning)
- e2e-discovery (delivering)
- e2e-onboarding (planning)
- e2e-traversal (delivering)
- e2e-validation (delivering)
- graph-conflict-detection (delivering) — surfaces conflict signal to the user; the conflict detection algo runs on the client side
- graph-memory (planning) — substrate spec; cloud children may emerge, but the design is the hero contract
- graph-memory-7c-live-test (planning) — live validation harness; the test scripts live in hero
- graph-memory-federation (planning) — federation topology design; lives in hero contracts even though the server-side impl lives in cloud. Could arguably be umbrella; leaving in hero.
- greenfield-scaffolding (draft)
- hero-community (planning) — community surface (Discord/Discussions/contrib guide); CLI/repo concern
- hero-community-edition (planning) — defines a build *target* of hero-cloud, but the design home is hero. Could be argued as umbrella.
- hero-content-engine (planning) — blog/dev posts/case studies; lives with the project's public surface
- hero-demo-content (planning) — GIFs, screencasts; project marketing assets
- hero-distribution (planning) — Homebrew/install script/releases; CLI binary distribution
- hero-docs-site (planning) — public docs
- hero-landing-page (delivering) — public homepage
- hero-launch-playbook (planning) — launch sequence; project-level
- hero-pm (planning) — domain pack; ships under `domains/pm/` in hero
- hero-positioning (planning) — narrative/messaging
- hero-qa (planning) — domain pack
- hero-sales (planning) — domain pack
- hero-telemetry (planning) — opt-in CLI telemetry
- inline-propose-output-mode (planning) — agent output-mode contract; lives with the agent loader in hero
- institutional-memory (draft) — *flag*: tagged `[cloud, enterprise, billion-dollar, moat]` and describes proactive knowledge mining "across the org" — sounds cloud, but currently a vision draft. Could be cloud once the design firms up. Leaving in hero pending user judgment.
- lean-agent-profile (planning)
- master-ingest-restore (delivering)
- monorepo-satellite-installs (delivering)
- multi-domain-core (draft) — domain pack architecture; CLI/engine
- next-as-projection (planning)
- next-noop-writes (completed)
- per-feature-smoke-coverage (delivering)
- premise-interrogation (planning)
- project-charter (planning)
- satellite-corpus-integration (planning)
- satellite-harness-coverage (planning)
- satellite-scope-extras (planning)
- satellite-scope-finishers (planning)
- satellite-walkthrough-ux (planning)
- scan-pluggability (planning)
- single-source-install-p1-agents-md (planning)
- single-source-install-p2-canonical-tree (completed)
- single-source-install-p3-migrate (completed)
- single-source-install-p4-verify (completed)
- single-source-install-p5-json-api (completed)
- spec-prioritization (planning)
- spec-status-integrity (delivering)
- spec-type-registry (planning)
- synthesis-maintenance (planning)
- timely-briefs (planning)
- traversal-queries (delivering)
- tripwire-system (delivering)
- unified-retrieval-layer (delivering)
- unified-search (delivering)
- version-mismatch-severity (planning)
- dashboard-view-registry (planning) — *flag*: the dashboard itself is server-side, but the *registry contract* is engine-level. Borderline; leaving in hero with the contract.
- cto-dashboard (draft) — *flag*: tagged `[cloud, enterprise]`. Describes a cloud-rendered dashboard. Could move to cloud, but it's a vision draft. User judgment.
- cross-org-intelligence (draft) — *flag*: tagged `[cloud, network-effect]`. Vision draft for anonymized cross-org learning. Leaning cloud once it firms up.
- team-knowledge-flywheel (draft) — *flag*: explicitly describes cloud-org-level knowledge sync. Vision draft. Leaning cloud once it firms up.

**Initiatives (planning/active):**
- environment-awareness (planning)
- execution-plan (active) — cross-cutting plan; could be archived once split is complete
- get-back-on-track (planning)
- hero-domains (planning)
- hero-killer-features (planning)
- hero-marketing (planning)
- hero-team-experience (planning) — *flag*: defines the team layer that spans hero CLI + hero serve. Hybrid umbrella; recommend staying in hero as the cross-repo experience initiative.
- install-upgrade-contract-coverage (planning)
- single-source-install (planning)

**Bugs:**
- hero-workspace-not-self-describing (planning) — install-time AGENTS.md content drift; CLI/install code
- install-target-emits-both-claude-and-agents-md (planning) — `hero install --target claude` regression; CLI code
- restore-hero-feed-cli (completed) — CLI feed reader
- scan-enrichment-unbounded-loop (planning) — `/scan` slash command behavior in hero
- scan-test-detection-misses-spock-vitest (planning) — `internal/scan/scan.go` detector bug
- upgrade-multi-target (completed) — CLI upgrade behavior

---

## Recommended migration order

Move in small low-risk batches. Suggested order:

1. **Tagged `[cloud,...]` vision drafts** (lowest risk — `draft` status, no active work, name + tags telegraph intent): `cross-org-intelligence`, `team-knowledge-flywheel`, `institutional-memory`, `cto-dashboard`. Move these first; they're trial-balloons for the migration mechanics. Even if you later decide one stays, the round-trip is cheap.
2. **Explicitly-named `cloud-*` planning specs** (clear intent, all `planning`/`approved`, no in-flight work): `cloud-admin`, `cloud-billing`, `cloud-dashboard`, `cloud-dashboard-ui`, `cloud-mcp`, `cloud-notifications`. These are unambiguous and have no upstream/downstream churn in hero.
3. **Server-side infra planning specs** (federation/server schema): `tenant-isolation-rls`, `graph-schema-simplification`, `nats-event-bus`, `client-id-user-scoping`. All `planning`. These reference cloud DB schemas and have no CLI-side files.
4. **Team-server feature family** (the hero-platform thesis: this is the cloud server in self-hostable form): `team-oauth`, `team-notifications`, `hero-automations`, `hero-dashboard-v2`, `hero-team-server`. All `planning`. Decide as a coherent batch — they cite each other and share design DNA.
5. **`team-connect`** as a split or hero-cloud move with a CLI-side child later. Recommend doing this alongside batch 4.
6. **User-judgment specs** (`hero-runner`, `agent-outposts`, `hero-cloud` initiative, `hero-platform` initiative): pause for the user to weigh in before moving. These are the cross-cutting cases where the boundary is less obvious.
7. **`pre-launch-hardening`** (delivered): leave as historical record in hero, or move to hero-cloud as part of its history. Either is fine; recommend leaving since it's archival.

**Anti-pattern to avoid:** moving the `hero-governance` umbrella to
hero-cloud. That umbrella is explicit that the vocabulary belongs in
the OSS hero contracts package; the enforcement engine is what lives
in cloud. Spawn an enforcement-side child spec in hero-cloud and keep
the umbrella here.

**Verification per batch:** after each move, run `hero search` and
`hero status` in both repos to confirm cross-references are still
resolvable. Specs reference each other by slug — broken cross-repo
references are the most likely silent failure mode.
