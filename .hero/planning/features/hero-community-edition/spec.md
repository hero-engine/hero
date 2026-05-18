---
title: "Hero Community Edition — Self-Host Team Build that Demonstrates the Power without Eroding the Moat"
slug: hero-community-edition
type: feature
status: planning
priority: medium
tags: [community-edition, packaging, distribution, self-host, governance, moat, adoption]
created: 2026-05-15
relations:
  - target: hero-governance
    kind: depends-on
  - target: hero-cloud-split
    kind: depends-on
  - target: hero-cloud
    kind: related
  - target: hero-team-server
    kind: related
  - target: hero-distribution
    kind: related
  - target: cloud-admin
    kind: related
  - target: cross-org-intelligence
    kind: related
horizon: next
---

# Hero Community Edition — Self-Host Team Build that Demonstrates the Power without Eroding the Moat

## Goal

Ship a free, self-hostable build of `hero-cloud` — **Hero Community
Edition (CE)** — that lets a 2–8 person team feel the real Hero compounding
loop across people, while leaving enough headroom that the paid Cloud and
Enterprise tiers remain the obvious upgrade. CE is a *build target* of the
`hero-cloud` codebase, not a fork: same source tree, same governance code
paths, capped and feature-flagged at runtime. A team should be able to go
from `docker run` to a first agent session with shared context in under
30 minutes, and should think *"this is amazing — we need the real version
the moment we add a second team or care about audit retention"* — never
*"this is hobbled."*

Done means: a single `hero-cloud` binary boots in CE mode behind one env
var; the CE profile from `hero-governance` is wired end-to-end (org_id
pinned, 30-day audit, no SSO); a team of up to 8 users can sync specs,
notes, decisions, and scans against the server; agent context inheritance
works *across developers* on the team; capped/excluded features show a
clear "Cloud only" affordance with a one-screen comparison rather than a
404; CE ships from `hero-cloud` CI as a docker image plus a single binary,
on every Cloud release.

## Kickoff

Decide the IN/OUT line for Hero CE, pin the v1 caps (8 users, 30-day
audit, single org, no SSO, local auth, file-based policy config), and
specify how CE is built and shipped from the `hero-cloud` repo without a
fork. The user's framing — *"a CE with a lot missing but really letting a
team see the true power is pretty impactful"* — anchors every cut: the
demonstration surface must be unmistakably the real Hero loop across
people.

**Status:** planning — depends on `hero-governance` landing its CE
profile and on `hero-cloud-split` defining the build/release pipeline.

**Pick up at:** confirm the IN/OUT lists in §Design with stakeholders,
then write the CE profile boot path (env-var detection → cap loader →
feature gating layer) and the upgrade-affordance convention before any
code lands. The build/distribution work follows once the boot path
exists.

→ `.hero/planning/features/hero-community-edition/spec.md`

**Files:** `cmd/hero-cloud/` (CE boot path, future), `internal/edition/`
(new — edition flag, caps, feature gating, future),
`pkg/contracts/governance/` (CE profile referenced from governance spec),
`.hero/planning/features/hero-governance/spec.md`,
`.hero/planning/initiatives/hero-cloud-split/spec.md`,
`Dockerfile.ce` (new), `.github/workflows/release.yml` (extend with CE
artifacts).

**Skip:** designing the admin UI in detail (CE doesn't have one), writing
the exact docker compose file, enumerating every Cloud feature with a
CE/Cloud disposition (define the rule + 8–12 representative examples),
building the CE→Cloud migration tool (sketch the export shape, defer the
build), choosing a license key scheme (v1 is unkeyed with hardcoded
caps).

## Problem

Hero's mission is to make every session start smarter than the last one
ended, by capturing during work and injecting on the next turn. The
compounding gets dramatically more valuable the moment a *second person*
joins — the next developer on a project should start where the previous
developer left off, not from scratch. That requires a shared server.

Today, every potential user faces one of two on-ramps:

1. **Solo CLI** — works on a laptop, no server, no sharing. Powerful but
   the magic is one-person-deep.
2. **Hero Cloud (planned)** — multi-tenant SaaS, the full experience, but
   requires giving us the team's corpus.

Both miss a real audience and both leak a real opportunity:

- **Privacy-conscious teams (2–8 people) who cannot or will not use SaaS** —
  internal-tools teams, contractors under NDA, side-project groups,
  regulated-industry trial users. Without a self-host option they
  never get past solo CLI. They are also the highest-quality adoption
  seeds: they evangelize, they file good issues, they hire teammates
  who later push for Hero at their next company.
- **Teams evaluating Hero before committing to paid** — the eval *is* the
  conversion funnel. If the only way to try team mode is a sales call,
  the conversion never happens for the bottom 80% of teams. If trying
  team mode requires re-running every spec they have, the eval dies.

There is also a sibling problem from our own side: we are splitting
`hero` and `hero-cloud` into separate repos
(`.hero/planning/initiatives/hero-cloud-split/spec.md`). The same code
base needs to produce both the paid Cloud SaaS and the free CE build
without forking. Without an explicit edition concept in the codebase,
every new server feature gets shipped with no answer to "does this run
in CE?" — and by the time we ask, the feature is entangled.

We also need to be honest about the moat. CE is a giveaway; it must
demonstrate enough to drive upgrades while withholding enough to make
upgrades worth paying for. Get the line wrong on the free side and CE is
useless. Get the line wrong on the paid side and we erode the SaaS
business before it starts. The governance spec already defines the CE
*runtime profile* and what it lacks at the policy layer. This spec
extends that into the full product surface — what features, what caps,
what UX at the limits, and how it gets built and shipped.

**Mission-fit.** The most powerful Hero moment is *"the next session
starts where the last one ended"*. CE makes that moment cross-person
for the first time. A junior developer joining a project starts as
smart as the senior who onboarded them — without us hosting their data.
That is the demonstration. Everything else in CE is in service of that
moment or out of scope.

## Design

### Positioning thesis

CE is **the demonstration build**. Two audiences, one product:

1. **Self-host adopters** — teams who will run CE long-term because they
   cannot or will not use SaaS. They may never upgrade and that is fine;
   they are the seed corn.
2. **Evaluators** — teams trialing Hero before buying Cloud. CE is the
   only way to try team mode without a sales call. 30 minutes from
   `docker run` to a teammate joining is the bar.

Both audiences need the same product. The differentiation is in
*caps and UX at the limits*, not in feature presence. Every CE feature
that exists is the real implementation — never a degraded mock.

### What's IN CE (the "feel the power" surface)

The demonstration surface. These are the things that make a team say
*"oh, this changes how we work"*. CE has every one of them, capped only
by team size and retention.

1. **Shared team graph.** One server, one graph, all team members
   syncing the same corpus. Specs, decisions, conventions, scans, notes,
   spec-derived events. Backed by a single SQLite or Postgres volume on
   the host.
2. **Cross-developer spec workflow.** A teammate runs `/design auth-flow`
   on their laptop; the spec appears in everyone else's `hero status`
   and `hero list`. They can read it, comment on it, link from their
   own specs, claim it for delivery. Spec ownership and status are
   visible across the team.
3. **Cross-person agent context inheritance.** *The headline moment.*
   Developer A finishes a session enriching the graph with decisions
   and notes about the payments module. Developer B opens a fresh
   session against the payments module the next day. Developer B's
   first agent turn surfaces A's relevant work — not because B searched
   for it, but because the corpus injection treats team knowledge as
   first-class. This must work in CE on day one.
4. **Governance vocabulary, end-to-end.** Classification (private /
   internal / confidential / restricted), subjects, and the single
   `Retriever.Filter` path from `hero-governance` all run in CE. Egress
   policies, ingress rules, and audit events all work — capped only by
   the constraints listed in §What's OUT.
5. **Basic policy enforcement.** Org-level ingress defaults (`.env`,
   `*.pem`, `.gitignore`) are baked in. CE-extra rules — path
   exclusions, classification minimums for specific paths, subject
   auto-tagging — are declared in a file-based policy bundle (YAML at
   `/etc/hero-cloud/policy.yaml` or the equivalent mounted path). The
   *engine* that compiles and enforces those rules is the same one
   Cloud uses; CE just lacks the admin UI.
6. **Audit log with 30-day retention.** Every retrieval emits an audit
   event per the governance spec. CE caps retention at 30 days; events
   are append-only graph nodes, fully queryable via `hero why` and the
   audit-query API during the retention window.
7. **Self-host install in under 30 minutes.** `docker compose up` with a
   bundled compose file is the canonical path. The team admin gets an
   invite-token URL on first boot; team members run `hero trust ce
   https://server.example.com` and paste the token. No SSO, no
   directory integration; just invite tokens and a stable
   email-as-identity.
8. **Sync to and from the `hero` CLI.** Every CLI command that already
   works locally — `hero scan`, `hero note`, `hero status`, `hero list`,
   `hero why`, `hero search`, the full slash-command set — works
   against the CE server with no behavior change other than that the
   results now include teammates' contributions.
9. **GitHub integration via PAT.** Tracker import and outgoing PR
   linkage work against GitHub using a personal access token. The
   integration is the same one Cloud uses; CE includes it because
   GitHub is where most CE-target teams already live.
10. **Local model providers.** CE wires up to whatever local or
    third-party model the team configures (Claude, OpenAI, a
    self-hosted Llama, etc.) through the same model-routing layer
    Cloud uses. No proprietary cloud-side prompts; standard agent
    flows only.
11. **Backup and restore.** A `hero-cloud backup` and `hero-cloud
    restore` command produce and consume a graph snapshot + spec-tree
    tarball. The team is responsible for running them; CE prints a
    "you should set this up" line in the welcome message and links to
    docs.

If a CE-running team can do all 11 things and especially #3, the demo
has landed. Everything in §What's OUT is something they will not miss
until they scale past 8 people, until they want SSO, or until they
need cross-team intelligence or audit retention beyond 30 days.

### What's OUT of CE (the moat)

Explicit cuts, each defensible as either "they will not miss it at
team size 8" or "it is a clear upgrade signal."

1. **Multi-tenancy / multiple organizations.** CE is single-tenant;
   `org_id` is pinned to 1 (per the governance spec's CE profile). No
   org-switching UI, no cross-org queries. *Defense:* a single team
   does not need multiple orgs; the moment they do they have outgrown
   CE by definition.
2. **SSO / SAML / OIDC.** CE uses invite tokens and email-as-identity
   only. *Defense:* compliance and IT departments require SSO; if SSO
   matters to the team, they have outgrown CE.
3. **Cross-team / cross-org intelligence and federation.** Features
   under `cross-org-intelligence` and `graph-memory-federation` are
   stripped. *Defense:* the value of federation grows with the number
   of teams; one team does not need it.
4. **Admin UI for policies.** CE policies are file-based config; Cloud
   has a full policy authoring UI. *Defense:* teams of 8 with one or
   two admins can edit YAML; teams of 80 cannot. The UI is also the
   surface enterprise customers care most about.
5. **Long-term audit retention.** CE caps at 30 days; Cloud default is
   90 days with org-configurable extension; Enterprise allows
   customer-set retention with optional SIEM export. *Defense:*
   compliance audits care about retention; small teams do not.
6. **Granular RBAC.** CE has exactly two roles: `admin` and `member`.
   Cloud adds custom roles, fine-grained scopes, per-spec ACLs.
   *Defense:* 8-person teams do not need RBAC; 80-person ones do.
7. **Hosted operations.** Backups, upgrades, scale, uptime, monitoring
   are entirely the team's problem on CE. *Defense:* if the team
   doesn't want to operate it, that *is* the Cloud pitch.
8. **Cloud-managed AI features.** Proprietary prompts, premium model
   routing, cloud-side enrichment workers, cross-corpus retrieval
   models, and any model behavior that depends on Cloud infrastructure
   are Cloud-only. CE uses the open agent flows. *Defense:* the
   premium model routing is core IP; protecting it preserves the
   reason to upgrade.
9. **User cap: 8.** Hard cap. The 9th user invitation prompts an
   upgrade screen (see §UX at the limits). *Defense:* picked at 8
   rather than 5 because most "small team" scenarios are in the 4–7
   range and 8 leaves comfortable headroom; picked under 10 because
   that is the threshold where SSO and RBAC start to matter and where
   upgrade conversations should happen.
10. **Premium integrations.** Slack, Linear, Jira, PagerDuty, and any
    paid third-party connector are Cloud-only. CE supports the
    baseline set: GitHub via PAT, generic webhook outbound. *Defense:*
    integrations are a clear "add-on" surface customers are accustomed
    to paying for; the GitHub baseline ensures CE is not useless.
11. **Compliance features.** SOC 2 evidence collection, customer-held
    keys, audit export connectors, compliance reporting, retention
    policy proofs, signed-audit chains. All enterprise-only.
    *Defense:* compliance is the most expensive feature category to
    build and operate; customers who need it pay for it.
12. **Hosted dashboards.** The cloud dashboard UI (`cloud-dashboard`)
    is not in CE; CE users get the CLI surface only, plus a minimal
    read-only local web view that ships with the server (TBD whether
    even that is in v1). *Defense:* the dashboard is a major Cloud
    selling point; keeping it Cloud-only is intentional.

Graph node count is intentionally **uncapped** in v1. Capping graph
size would punish exactly the *demonstration* — a team that ran a
real `hero scan` would hit the limit on day one. Reconsider only if
we see CE installs being used as free production for orgs that should
be paying.

### The "feels real, not crippled" calibration — UX at the limits

Every cut needs a UX path. The principle: **discoverable upgrade,
never naggy, always a real reason**.

- **Soft cap features** (audit retention, integrations, admin UI):
  the feature appears in the product but shows a small "Cloud only"
  badge where the gated affordance would be. Hovering or running
  `hero why-cloud <feature>` prints one screen: what Cloud adds, why
  it matters, a `https://heroengine.ai/cloud` link. No modal, no
  blocking dialog.
- **Hard caps** (9th user, cross-org, SSO): the action attempt
  surfaces a single screen with three lines — what was attempted,
  the cap, what Cloud offers — and a link. No retry-with-payment flow
  inside CE; the team contacts us or signs up at
  `heroengine.ai/cloud`.
- **Welcome message** (first `hero` invocation pointed at a CE
  server) prints: edition, version, the 30-day audit retention note,
  the user-cap status (e.g. "3 of 8 seats used"), and links to docs
  and the GitHub Discussions community page. Once per workspace.
- **Convention**: introduce a single helper in the shared codebase —
  `edition.RequireCloud(featureName)` — that the calling site uses to
  emit a consistent "Cloud only" affordance both in CLI output and in
  API responses. New Cloud features that should be CE-gated wrap the
  feature entry point in this helper.
- **No nagging.** No periodic upgrade prompts, no startup nag, no
  email captures from CLI flows. The product is the marketing.

### License & enforcement

**v1 is unkeyed with hardcoded caps.** Reasons:

- No license-key infrastructure to build, distribute, or rotate.
- Self-host users hate keys; for a privacy-conscious audience, a key
  that phones home is a non-starter; an offline key is just friction.
- The cap is enforced by the code anyway; a determined patcher will
  bypass any scheme, and we are not solving for that.

Move to a keyed CE *only if* CE becomes a paid SKU later (e.g. a
"Hero Team" tier with raised caps). At that point the keyed build is
a different feature and gets its own spec.

**Telemetry: off by default, opt-in.** CE does not phone home. Reasons:

- The same privacy-conscious audience that wants self-host hates
  silent telemetry. Default-on breaks the trust pitch.
- We get richer signal from a post-install survey link in the
  welcome message and from GitHub Discussions activity than from
  anonymized event streams of low base-rate features.
- Reconsider once CE has > 100 installs and there is a concrete
  product question telemetry could answer.

Opt-in telemetry, when enabled, sends: edition, version, install age,
user count bucket (e.g. 1–2, 3–5, 6–8), and `hero check` failure
counts. No spec content, no node content, no user identities.

### Build & distribution

**Single binary, runtime-flagged edition.** The `hero-cloud` binary
detects edition at boot via `HERO_EDITION` env var (`ce` | `cloud` |
`enterprise`) defaulting to `cloud` for safety. Boot order:

1. Parse `HERO_EDITION`. Unknown values fail fast with a clear error.
2. Load the cap table for the edition (`internal/edition/caps.go`,
   compile-time data, tiny).
3. Wire the feature-gating layer — features check the active edition
   before activating. `RequireCloud` and friends call into this
   layer.
4. Boot governance with the matching profile (per
   `hero-governance`).

Only a small number of code paths are *compile-time* stripped from CE
builds: those that contain genuinely proprietary IP (premium model
routing prompts, SOC 2 evidence pipelines, signed-audit chains).
Build tags or a build-time codegen step removes those from the CE
binary, so a CE binary on disk cannot reveal them.

The rest of the codebase is one binary, one tree, one set of tests.
Every Cloud feature ships with a CE/Cloud disposition decision at
delivery time — recorded in the feature spec — answering "does this
run in CE?" and if so, "is it capped?". This rule is the operational
contract that prevents the fork.

**Distribution paths**:

- **Primary**: Docker image at a public registry (e.g.
  `ghcr.io/hero-engine/hero-cloud:ce-vX.Y.Z`) with a bundled
  `docker-compose.yml` published at `heroengine.ai/ce/docker`.
- **Secondary**: Single binary on GitHub Releases (Linux amd64/arm64,
  macOS arm64), with a `systemd` unit example in the docs.
- **Not v1**: Homebrew, apt, kubernetes operator. Add when there is
  measured demand.

**Versioning**: CE versions track Cloud versions exactly — same
release cadence, same version number, distinguished only by image
tag (`:ce-vX.Y.Z` vs `:cloud-vX.Y.Z`) and by the runtime edition
flag. No `-ce` suffix on the version itself; nothing to coordinate
between two repos because CE and Cloud are the same repo and the same
release.

### Operational realism

**Upgrade path from CE to Cloud.** Sketch only; the migration tool is
its own future spec:

- A team running CE who wants to move to Cloud runs `hero-cloud export
  --for-cloud-migration > snapshot.tar.gz`. The snapshot contains:
  the graph database dump, the spec tree (the same `.hero/specs/` and
  `.hero/planning/` trees the team has been editing), the policy
  bundle file, and a manifest naming users by email.
- We provide a Cloud-side import endpoint that consumes the tarball:
  provisions a fresh Cloud org, re-creates users by email (they
  receive Cloud invites), imports specs and graph, re-runs subject
  inheritance to apply Cloud-tier classification policies.
- Audit log does **not** migrate. Different retention model; bundling
  CE audit events into Cloud audit storage muddies provenance.
  Document the cutover date instead.
- The build of this migration tool is deferred. The export shape and
  this contract is what the CE spec commits to; the import side and
  the conflict-resolution rules belong to a follow-on.

**Backup story for CE.** The team is responsible. CE ships:

- `hero-cloud backup --to <path>` — produces a snapshot tarball
  (graph + specs + policy).
- `hero-cloud restore --from <path>` — restores into an empty
  install.
- Welcome message reminds the admin to schedule backups; docs include
  a cron example.

**Support story.** Community-only. The welcome message says so
explicitly:

> Hero CE is community-supported via GitHub Discussions. For SLA
> support, audit retention, SSO, and managed ops, see Hero Cloud at
> heroengine.ai/cloud.

This sets expectations correctly so paid-tier users do not feel
shorted by CE turnaround on their unrelated free-tier installs.

### Out-of-scope confirmations from governance

The `hero-governance` CE profile already commits CE to:

- `org_id` pinned to 1 (single-tenant).
- 30-day audit retention cap.
- No SSO (local-only auth).
- File-based policy authoring (no admin UI).
- Cross-org intelligence disabled.

This spec inherits those rather than redefining them.

## Acceptance Criteria

- THE SYSTEM SHALL produce a single `hero-cloud` binary that selects
  its edition at boot from the `HERO_EDITION` environment variable,
  accepting values `ce`, `cloud`, and `enterprise`, defaulting to
  `cloud`, and failing fast on unknown values.
- THE SYSTEM SHALL publish a Hero CE Docker image on every
  `hero-cloud` release, tagged `:ce-vX.Y.Z`, built from the same
  source tree as the Cloud image.
- THE SYSTEM SHALL provide a single `docker-compose.yml` example
  whose execution takes a new operator from zero to a running CE
  server in under 30 minutes on a standard development machine.
- WHILE `HERO_EDITION=ce` is set THE SYSTEM SHALL pin `org_id` to 1,
  cap user count at 8, cap audit retention at 30 days, disable
  cross-org features, and disable SSO providers, using the
  `hero-governance` CE runtime profile.
- WHEN a CE administrator attempts to invite a 9th user THE SYSTEM
  SHALL reject the invitation and SHALL display a single screen
  naming the cap, the Cloud feature that lifts it, and a link to
  `https://heroengine.ai/cloud`.
- WHEN a feature that is Cloud-only is invoked on a CE server THE
  SYSTEM SHALL call `edition.RequireCloud(featureName)` (or the
  language-equivalent helper) and SHALL return a consistent
  structured error containing the feature name and an upgrade link.
- WHEN a CE user runs the CLI against a CE server for the first time
  on a workspace THE SYSTEM SHALL print a welcome message naming the
  edition, version, current seat usage, audit retention cap, the
  community-support note, and the docs and discussions URLs.
- WHILE a CE server is running THE SYSTEM SHALL allow every CLI
  command that works in solo CLI mode (`hero scan`, `hero note`,
  `hero status`, `hero list`, `hero why`, `hero search`, the full
  slash-command set) to operate against the server with no behavior
  difference other than the inclusion of teammates' contributions.
- WHEN a developer using CE opens a fresh agent session against a
  module another teammate has previously enriched THE SYSTEM SHALL
  surface that teammate's relevant work in the developer's initial
  agent context via the standard retrieval-and-injection path, with
  no additional configuration.
- THE SYSTEM SHALL provide `hero-cloud backup --to <path>` and
  `hero-cloud restore --from <path>` commands that produce and
  consume a snapshot tarball containing the graph database, the spec
  tree, and the policy bundle.
- THE SYSTEM SHALL provide a `hero-cloud export --for-cloud-migration`
  command producing a tarball containing the graph dump, spec tree,
  policy bundle, and a user manifest of email-keyed identities, and
  SHALL exclude audit events from this export.
- WHILE `HERO_EDITION=ce` is set THE SYSTEM SHALL load policies from
  a file-based bundle (default `/etc/hero-cloud/policy.yaml` or the
  equivalent mounted path) and SHALL refuse to start if the bundle
  exists but is malformed.
- THE SYSTEM SHALL emit no outbound network traffic from a CE server
  beyond what is required to fulfill explicit user actions (model
  calls, tracker imports, configured webhooks), with telemetry
  defaulting off.
- WHEN telemetry is opt-in enabled on a CE install THE SYSTEM SHALL
  send only: edition, version, install age, user-count bucket, and
  `hero check` failure counts — no spec content, node content, or
  user identities.
- THE SYSTEM SHALL include CE in the standard release CI pipeline so
  that every `hero-cloud` release also produces and publishes a CE
  Docker image and a CE binary set without manual steps.
- THE SYSTEM SHALL document, on every spec for a server feature
  delivered after this spec lands, a CE/Cloud disposition decision
  answering "does this run in CE?" and if so, "is it capped, and
  how?".
- WHERE compile-time-stripped IP is present in the codebase (premium
  model routing, signed-audit chains, SOC 2 evidence pipelines) THE
  SYSTEM SHALL exclude those code paths from CE builds via build
  tags so that a CE binary does not contain the source or compiled
  artifacts of those features.
- WHEN a CE user requests support THE SYSTEM SHALL direct them to
  the community channel (GitHub Discussions) and SHALL NOT offer a
  paid-tier SLA path inside CE.

## Risks

- **Caps feel arbitrary.** Picking 8 vs 5 vs 10 users is judgement,
  not data. If 8 turns out to be too tight (legitimate "small team"
  feels constrained on day 3) we lose adopters; too loose and we
  erode upgrade pressure. *Mitigation:* ship 8; instrument the
  invite-9 rejection (locally, surfaced via `hero check`) so an
  operator who wants to argue for raising the cap has a number.
  Revisit at 6-month mark.
- **CE undercuts paid sales.** A team using CE successfully forever
  is a non-paying customer. *Mitigation:* the cuts in §What's OUT
  are deliberately what enterprise buyers ask for first (SSO,
  retention, RBAC, compliance, support). The likely outcome is that
  CE *creates* paid demand by surfacing teams who outgrow it, not
  consumes it. Track the upgrade conversion rate from CE installs.
- **Forks erode the moat anyway.** A third party patches the cap and
  redistributes a "CE+" build. *Mitigation:* we do not solve this
  technically. The license terms address it; the moat is the
  combination of Cloud-only features and Cloud-side operations, not
  the cap itself. A patched CE still lacks SSO, retention, RBAC, and
  hosted ops.
- **Edition entanglement.** New server features fail to declare a CE
  disposition; CE breaks silently or Cloud-only code leaks into CE
  builds. *Mitigation:* enforce the disposition decision via spec
  template + `hero check` rule that fails delivery if a server
  feature spec lacks the disposition section. Build CI runs both
  edition test suites on every PR.
- **Operational support load.** CE adopters file Cloud-style support
  requests, expect Cloud-style SLAs. *Mitigation:* welcome message
  + docs + Discussions categories make the support story crystal
  clear before the install. Reuse the `hero-community` spec's
  category structure.
- **Migration path mismatch.** A team running CE for a year produces
  a graph that does not cleanly import into Cloud (schema drift,
  policy-bundle incompatibilities, classification reinterpretation).
  *Mitigation:* the export shape is committed in this spec; the
  import side gets a dedicated spec before the first paying
  migration. Versioning the snapshot format from v1 means future
  Cloud can accept old CE exports.
- **"Free is bad UX" by default.** If CE is buggy or slow it
  poisons the Cloud trial story. *Mitigation:* CE ships the same
  binary as Cloud; the bar for CE quality is the bar for Cloud
  quality. CI runs the same test suite. No "CE-only" code branches
  that go untested.

## Out of Scope

- The detailed admin UI design — CE doesn't have one; Cloud's admin
  UI is in `cloud-admin` and follow-ons.
- The exact `docker-compose.yml` file contents — sketched here,
  written during delivery.
- The full CE→Cloud migration tool implementation — only the export
  shape is committed.
- Telemetry pipeline implementation — opt-in only; pipeline is a
  separate future spec if/when telemetry is needed.
- Pricing — this spec does not name a Cloud price, a "Team" middle
  tier, or any monetary number. Pricing is a marketing decision.
- License key infrastructure — v1 is unkeyed; a keyed CE belongs to
  a future paid-CE spec if that ever happens.
- Homebrew / apt / Kubernetes-operator distribution — Docker and
  binary only in v1.
- The minimal read-only local web view (TBD) — defer the decision
  to delivery; if it is not trivial it gets cut from v1.
- The CE/Cloud disposition tags for every existing Cloud feature
  spec — this spec commits to the rule going forward; back-tagging
  existing specs is a hygiene task tracked elsewhere.
- Any feature that depends on multi-tenant identity (cross-org
  intelligence, federation, multi-org search) — explicitly excluded
  from CE by governance.

## Open Questions

- **Postgres vs SQLite for CE.** SQLite is simpler to operate but
  caps concurrency; Postgres handles team-size loads better but
  requires a separate container. *Lean:* ship the docker-compose
  with both bundled (server + Postgres), and document a `single
  binary + SQLite` mode for the smallest teams. Decide before
  delivery.
- **Should CE release before Cloud or after?** Cloud is the revenue
  path so probably after, but CE's build infra needs to exist from
  day one so every Cloud feature has a CE disposition. *Lean:* the
  CE edition flag and gating layer land first, CE *artifacts* publish
  alongside the first public Cloud release.
- **Branding.** "Hero CE" vs "Hero Community" vs "Hero Self-Host".
  *Lean:* "Hero Community Edition" formally, "Hero CE" in code, docs,
  and image tags. Defer the customer-visible branding to marketing.
- **Read-only local web view in v1?** A bare local page showing
  current specs and the team's activity, served by the CE binary on
  the same port. *Lean:* not v1 unless trivial; CLI surface is
  enough to demo. Decide during delivery.
- **GitHub Enterprise.** CE users may want their GitHub PAT to point
  at GHE rather than github.com. *Lean:* support via the standard
  GitHub integration's base-URL config; trivial to enable, no extra
  product surface.
- **Cap interaction with agent sessions.** Are agent identities (from
  the governance spec) counted toward the 8-user cap? *Lean:* no —
  cap human seats, not agent principals. Agent tokens are scoped to
  a human, so unbounded agent identities still get audited to one of
  the 8 seats.
- **What happens to a CE install whose user count was raised before
  the cap shipped?** *Lean:* if upgrading from a pre-cap version, do
  not retroactively block existing users; surface a "you are
  over-cap; reduce to 8 to enable new invites" warning in `hero
  check`. Existing users keep working.
- **Should CE include the dashboard view registry?** The minimal
  dashboards may be valuable demonstration surface. *Lean:* let
  `cloud-dashboard` and `dashboard-view-registry` declare their CE
  disposition individually per the new rule, rather than blanket-
  excluding here.
