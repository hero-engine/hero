---
title: "Hero Governance — Classification, Policy-Filtered Retrieval, Agent Identity, Audit-by-Construction"
slug: hero-governance
type: feature
status: planning
priority: P0
tags: [governance, security, contracts, policy, audit, multi-tenancy, enterprise]
created: 2026-05-15
relations:
  - target: agent-outposts
    kind: related
  - target: tenant-isolation-rls
    kind: related
  - target: cloud-admin
    kind: related
  - target: cross-org-intelligence
    kind: related
  - target: hero-cloud
    kind: parent
horizon: now
---

# Hero Governance — Classification, Policy-Filtered Retrieval, Agent Identity, Audit-by-Construction

## Goal

Hero gains a single, layered governance model that controls what enters the
corpus (ingress), what comes back out of it (egress), who is asking (agent
identity), and what was in effect when a decision was made (policy-as-graph,
audit-by-construction). The **vocabulary** — classification levels, subjects,
principals, scopes, policy nodes, audit events, and the retrieval API
signature — lives in the OSS **hero contracts package** and is the same on a
laptop, in Community Edition, in multi-tenant cloud, and in an air-gapped
enterprise install. The **enforcement engine** that uses that vocabulary to
actually decide and audit lives in `hero-cloud` and is the moat.

Done means: every node in the graph can carry a classification and subjects;
every retrieval flows through one filter function that takes a principal +
scope + classification policy and returns the allowed view; every retrieval
emits a structured audit event referencing the policy version in force; agent
sessions are first-class principals with their own scoped tokens; the same
code paths run in all four runtime profiles (solo CLI, Community Edition,
Cloud, Enterprise on-prem) with feature-flag-style degradation rather than
forked implementations.

## Kickoff

Foundational governance model for Hero — classifications, subjects, policies,
agent identity, and a single retrieval filter — split between the OSS
contracts package (vocabulary + API shape) and the paid enforcement engine.

**Status:** planning — spec just landed; this is the foundation for the
hero-cloud repo split and the Community Edition packaging specs.

**Pick up at:** lock the contracts-package surface first: write the Go
interfaces for `Classification`, `Subject`, `Principal`, `Scope`,
`PolicyNode`, `AuditEvent`, and the `Retriever` signature. Land those in
a new `contracts/governance/` directory before any enforcement work.

→ `.hero/planning/features/hero-governance/spec.md`

**Files:** `contracts/governance/` (new), `internal/graph/node.go`,
`internal/retrieval/`, `.hero/planning/features/agent-outposts/spec.md`,
`.hero/planning/features/tenant-isolation-rls/spec.md`
**Skip:** designing the admin UI, SSO/SAML wiring, specific PII regex
catalogs, and LLM-call wrapping mechanics — all are out of scope here.

## Problem

Hero's mission is to make every agent session start smarter than the last
one ended by compounding the team's knowledge into a corpus and injecting
it back automatically. The corpus is exactly the kind of thing that makes
governance non-optional the moment more than one person is involved:

- It contains decisions, conversations, PII pulled in via tracker imports,
  diagnostic notes that mention customer names, credentials accidentally
  pasted into a chat the agent then captured, and architectural choices
  that competitors would pay to read.
- The corpus is *queried by stateless agents on every turn*. The retrieval
  path is the only thing between sensitive content and an LLM context
  window. There is no human reviewer in the loop most of the time.
- The same graph types must work for a solo dev on a laptop with no server,
  a five-person team running Community Edition, a multi-tenant SaaS
  customer, and a regulated-industry on-prem deployment with customer-held
  keys. Forking the data model per profile is unacceptable.
- We are about to split `hero-cloud` into its own repo. The line we draw
  between OSS contracts and paid enforcement *now* determines what a
  third party can reimplement later, and where the moat actually lives.

The status quo has none of this. `graph_nodes` has no classification
column. Retrieval has no central filter — every callsite re-implements
"what should this user see," badly or not at all. Agents impersonate the
user with no separate identity. There is no audit of what was returned to
whom. Policies, where they exist at all, live in code.

If we ship Hero Cloud to a team without solving this, the first serious
customer evaluation will end with a security questionnaire we cannot
answer. If we solve it badly — by hardcoding enforcement into ad-hoc spots
in the codebase — every cloud feature thereafter has to remember to plug
in, and one will forget.

**Mission-fit.** Governance is what lets the *floor rise for everyone*
rather than only the senior dev who already knows what to ask. A junior
on the Acme account can safely query the full corpus because the
restricted Acme-only nodes will only return to people with Acme scope —
nobody has to remember to ask carefully. Gated access enables broader
contribution without leaking what shouldn't leak.

## Design

Five interlocking concerns. Each has a **vocabulary layer** (OSS contracts
package, frozen across runtime profiles) and an **enforcement layer**
(paid engine in `hero-cloud`, present-but-constrained in CE, degenerate
no-op in solo CLI). Be ruthless about which side of that line each thing
sits on.

### 1. Classification & Subjects

**Classification** is a single ordered enum on every graph node:

```
public < internal < confidential < restricted
```

These are *defaults*; an organization can extend the set via a policy
node that registers additional levels with an ordinal position
(e.g. `secret` between `confidential` and `restricted`). The ordering
matters because egress decisions are "max classification in the result
set vs. caller's clearance" comparisons.

**Subjects** are structured tags describing what a node is *about* —
not who can see it. A subject is a typed pair:

```go
type Subject struct {
    Type  string // open-ended: "customer", "employee", "system",
                 // "repo", "incident", "contract", ...
    ID    string // stable identifier within Type
    Label string // human-readable, optional
}
```

A node carries zero or more subjects. Subjects let policies match on
patterns like "anything with `subject{type:customer, id:acme}` is
restricted to principals with `scope:customer/acme`" — far more
expressive than flat tags and the foundation for "least-privilege by
default." We define the *shape* and leave the *type vocabulary* open;
the contracts package ships a small starter set (`customer`, `employee`,
`system`, `repo`, `tracker`) and orgs add their own.

**Defaults by node kind** (the contracts package ships these as the
canonical defaults; orgs override via policy):

| Node kind            | Default classification | Subject inheritance |
|----------------------|------------------------|---------------------|
| `note`               | `internal`             | inherits from session principal |
| `spec`               | `internal`             | inherits from repo |
| `decision`           | `internal`             | inherits from repo + linked specs |
| `convention`         | `internal`             | inherits from repo |
| `tracker_import`     | match source ticket visibility (public ticket → `internal`; restricted ticket → `confidential`) | inherits from ticket project + assignees |
| `enrichment` / `llm_output` | **max of input nodes** | union of input subjects |
| `conversation`       | `internal`             | inherits from session principal |
| `event` (retrieval/audit) | `restricted`      | self |
| `policy_node`        | `restricted`           | self |

The "enrichment inherits max" rule is non-negotiable — it is the only
way to prevent classification laundering through LLM summarization.

**Backward compatibility.** Existing graphs have nodes without a
classification field. The rule:

1. Schema migration adds the column nullable, indexed.
2. On read, `NULL` is treated as `internal` (the *team* default), not
   `public`. Solo CLI nodes treated as `private` for that user.
3. A background backfill job rewrites NULLs to `internal` (or the
   per-kind default in the table above) and records the change as a
   classification-event so it shows up in `hero why`.
4. Before any *team mode* (CE, Cloud, on-prem) is enabled on a given
   workspace, the backfill must reach 100% — gated by `hero check`.
5. No code path is allowed to write a new node with NULL classification
   once the schema migration ships.

**Lives in contracts package:** the `Classification` enum + extension
mechanism, `Subject` type, the per-kind default table, the inheritance
rules for enrichments, the NULL-on-read fallback rule.

**Lives in enforcement engine:** policy-driven overrides, the actual
classification of any given node (data, not interface), the backfill
worker.

### 2. Ingress Filtering

Ingress filtering controls what is allowed *into* the graph in the
first place. It runs on the CLI side, before anything leaves the
developer's machine, governed by org policy that the CLI fetches from
the server and caches locally.

**Trigger points** (every place data enters the graph):

- `hero scan` reads of repository files
- File watchers / capture-on-save flows
- `hero note` user input
- Tracker import streams
- Conversation capture from harnesses
- Outpost action events (from `agent-outposts`)

**Policy types**:

- **Path exclusion** — paths/globs never read at all (`.env*`,
  `**/secrets/*`, `**/*.pem`, anything matching `.gitignore` by default).
- **Content redaction** — regex/pattern rules that mask matching
  substrings on capture (e.g. credit card numbers, AWS keys). The
  specific regex catalog is **out of scope** for this spec; this defines
  the redaction-rule *shape* and the *plumbing*.
- **Classification minimum** — "anything from path X gets at least
  classification Y on ingest."
- **Subject auto-tagging** — "any node ingested under `src/customers/acme/`
  gets `subject{customer, acme}`."
- **Hard block** — refuse to capture; emit a local-only redaction event
  so the user knows something was suppressed.

**Policy delivery**:

- Policies are graph nodes (see §5) replicated to the CLI via the same
  sync path as specs, scoped to the user's org.
- CLI caches the active policy bundle at `~/.hero/policy/<org>/bundle.json`
  with a content hash and a `valid_through` timestamp.
- On each capture operation the CLI checks: is the cached bundle past
  its refresh window (default 1h)? If yes, fetch async; use the cached
  bundle for *this* operation. If the bundle is past its `valid_through`
  hard expiry (default 24h) and refresh fails, *fall back to deny-by-default*
  for any capture above `internal` classification — capture continues for
  routine `internal` work; sensitive paths get blocked. This is the
  failure mode an enterprise customer will ask about.
- Solo CLI: no server, no bundle, no enforcement other than the OSS
  defaults baked into the binary (the `.env*` / `.pem` / `.gitignore`
  exclusions, which are non-optional).

**Lives in contracts package:** the rule-shape enum (`PathExclude`,
`Redact`, `MinClassification`, `AutoSubject`, `Block`), the bundle
file format, the deny-by-default expiry rule, the OSS-default
exclusion list.

**Lives in enforcement engine:** policy authoring, bundle distribution,
the per-org regex catalog, redaction-event aggregation server-side.

### 3. Egress Filtering & Role-Aware Retrieval

This is the hot path. Every read against the graph — from a user CLI
command, from the MCP server, from an agent session, from an enrichment
worker — flows through one function:

```go
// In contracts/governance/retriever.go
type Retriever interface {
    // Filter takes a candidate node set (the result of an unfiltered
    // graph query) and returns the subset the caller is allowed to see,
    // plus an audit token that MUST be emitted alongside any use of
    // the result.
    Filter(ctx context.Context, q Query, candidates []NodeRef) (
        allowed []NodeRef,
        decisions []NodeDecision, // per-node allow/deny + reason
        auditToken AuditToken,
        err error,
    )
}

type Query struct {
    Principal     Principal
    Scope         Scope
    Purpose       Purpose // "user_view", "agent_context",
                         // "llm_egress", "enrichment_input", ...
    RequestedAt   time.Time
}

type NodeDecision struct {
    NodeID         string
    Allowed        bool
    Reason         string  // matched policy rule name + version
    Classification Classification
    Subjects       []Subject
}
```

**Algorithm** (concretely enough that an engineer can implement it):

1. Resolve the caller's effective clearance — the **maximum** classification
   they are permitted to *see* — by intersecting principal-clearance with
   scope-clearance.
2. Resolve the caller's effective subject scope — the set of subject
   predicates they may view (e.g. `customer:acme`, `customer:beta`, plus
   "all non-subject-restricted").
3. For each candidate node:
   - If `node.classification > effective_clearance` → deny.
   - If `node.subjects` contains any subject not in caller's subject scope
     AND that subject type is configured as *restricting* (a policy flag)
     → deny.
   - Otherwise allow.
4. Construct an `AuditToken` that names: the query ID, the policy node
   version in force, the principal ID, the scope, the purpose, the
   matched-but-denied node count (not IDs — those become an information
   leak in their own right) and the allowed node IDs.
5. Return `(allowed, decisions, auditToken)`. The audit event itself is
   emitted by the calling layer when it commits to using the result —
   not eagerly — so that "fetched but never used" doesn't pollute the
   audit log. The token is required to be consumed within a short
   window (e.g. 5 minutes) or it expires.

**No admin escape hatch.** There is no "su" mode, no internal-only
unfiltered API. Admin operations that legitimately need broader access
are modeled as principals with broader *scope*, which still flows
through `Filter` and still produces audit events. The "I am the
support engineer reading a customer's data to debug an incident" path
must produce an audit trail just like a normal query.

**Performance.** `Filter` is on every read. Three rules:

- Policy evaluation must be O(1) per node — no per-query DB lookups.
  Policies are compiled to an in-memory matcher per org, refreshed on
  policy-node updates.
- Audit tokens are constructed in-memory and emitted via a buffered
  async writer (bounded queue; backpressure surfaces as a soft-fail
  warning on the read path but does *not* block the read).
- The DB layer (`tenant-isolation-rls`) provides the org-level floor;
  `Filter` provides the user-level ceiling. Both must hold for a row
  to escape the database.

**Lives in contracts package:** the `Retriever` interface, `Query`,
`NodeDecision`, `AuditToken`, `Purpose` enum, the algorithm description
above as a normative reference.

**Lives in enforcement engine:** the actual matcher compilation, the
audit emitter, the principal/scope resolution against the user
directory.

### 4. Agent Identity & Scoped Credentials

AI agents are principals in their own right, not impersonators of
their user. An agent token names:

```go
// contracts/governance/principal.go
type AgentToken struct {
    AgentID         string         // stable, registered ID
    OnBehalfOf      UserID         // the human who issued this token
    Issuer          string         // server URL / "local"
    ReadScope       Scope          // what this agent may retrieve
    WriteScope      Scope          // what this agent may write into the graph
    EgressClearance Classification // max classification this agent may
                                   // include in an LLM-context egress
    NotBefore       time.Time
    NotAfter        time.Time      // hard expiry; required
    SessionID       string         // links this token to one session
    Capabilities    []string       // optional: "outpost:prod-api", etc.
}

type Scope struct {
    Repos         []string  // empty = no repo limit
    Subjects      []Subject // empty = no subject limit
    Classification Classification // max read clearance
    Kinds         []string  // node kinds; empty = all
}
```

**Issuance**: a user runs `hero agent token --scope <descriptor>` which
calls the enforcement engine (or, in solo CLI, mints a local-only token
with `scope=private`). The token is a signed JWT — signed by the org's
issuer key in cloud/CE/enterprise; signed by a local keychain-bound key
in solo CLI. Tokens are short-lived (default 1h, configurable up to a
ceiling set by org policy).

**Carriage**: agents present the token on every retrieval call. The
MCP server, the CLI commands, and any agent SDK accept the token via a
standard env var (`HERO_AGENT_TOKEN`) or header. The token replaces
the user's session token for the duration of the agent run.

**Revocation**: token IDs are tracked in a revocation list cached on
the server side; revocation propagates within a refresh window (default
60 seconds — short because agent runs are short). Solo CLI revocation
is "delete the keychain entry."

**Audit tagging**: every audit event from a query made with an agent
token records `principal_kind=agent`, the `AgentID`, the `OnBehalfOf`
user, and the `SessionID`. This is how a future session can answer
"what did agent X touch on behalf of user Y last Tuesday" — the
question Hero exists to make answerable.

**Integration with `agent-outposts`**: outpost capabilities listed in
`Capabilities` allow the agent to operate against named external
systems; the outpost-events feature already specifies the structured
event shape; we add the agent token's `AgentID`/`SessionID` to those
events so the audit story is end-to-end.

**Lives in contracts package:** `AgentToken`, `Scope`, the JWT claim
shape, the env-var/header carriage convention, the revocation-list
schema.

**Lives in enforcement engine:** token issuance, signing keys,
revocation propagation, scope-descriptor parsing UX.

### 5. Policy-as-Graph & Audit-by-Construction

Policies are not config files; they are first-class graph nodes.

```go
// contracts/governance/policy.go
type PolicyNode struct {
    ID          string
    OrgID       string
    Version     int                // monotonic per OrgID
    Effective   time.Time
    Supersedes  string             // prior PolicyNode ID
    Rules       []Rule             // ingress + egress + classification rules
    Authors     []UserID
    ReviewedBy  []UserID           // mirrors spec/PR review
    Classification Classification  // policy nodes are themselves
                                   // classified (default: restricted)
}

type Rule struct {
    ID       string
    Kind     RuleKind        // ingress|egress|classification|subject_scope
    Match    Matcher         // path glob / subject pattern / classification level
    Action   Action          // allow|deny|redact|reclassify|require_egress
    Reason   string          // human-readable; required
}
```

Properties of policy-as-graph:

- Policies are versioned. A new policy supersedes the prior version
  via the `supersedes` field; both remain in the graph forever.
- Policies are reviewable like specs — author + reviewer chain, diffs
  between versions, `hero why <policy-id>` works just like for any
  other node.
- "What policy was in effect at time T?" is a graph query.
- A policy node is itself classified (default `restricted`) and
  subject to retrieval filtering — only authorized principals can
  *see* the policy, although the *effects* of the policy apply to
  everyone.

**Audit-by-construction**:

```go
// contracts/governance/audit.go
type AuditEvent struct {
    EventID         string
    OrgID           string
    OccurredAt      time.Time
    Principal       PrincipalRef
    AgentSessionID  string   // empty if direct user query
    Purpose         Purpose
    PolicyVersion   int      // *which* policy was in force
    PolicyNodeID    string
    AllowedNodeIDs  []string
    DeniedCount     int      // not IDs — leak prevention
    EgressTarget    string   // "local", "llm:claude-3-5", "user:cli", ...
    EgressClassMax  Classification // max class in the allowed set
    DecisionReason  string
}
```

- Every successful retrieval emits one `AuditEvent`.
- Audit events themselves are graph nodes (kind `event`, classification
  `restricted`).
- Retention: solo CLI keeps local audit events for 7 days; CE caps at
  30 days; cloud/enterprise are policy-configurable with a hard floor
  (e.g. 90 days for cloud, customer-set for on-prem).
- Audit events are *append-only* — there is no DELETE path through the
  enforcement engine. (DBA-level deletion exists; doing it leaves a
  hole in the event chain by design.)

**LLM egress as a first-class purpose.** When an agent constructs
context for an LLM call, that retrieval has `Purpose: llm_egress` and
an explicit `EgressTarget` (e.g. `llm:claude-3-5-sonnet`,
`llm:self-hosted-llama-70b`). Org policy can match on the target:
*"anything `restricted` cannot egress to `llm:external/*`"*. The
filter returns a denial, the agent receives an empty (or
classification-reduced) context, and the audit event records the
denial. The agent's options at that point are: prompt the user to
unblock, use a self-hosted-target token, or fail the call. This is
the enterprise-grade selling point.

**Lives in contracts package:** `PolicyNode`, `Rule`, `RuleKind`,
`Matcher`, `Action`, `AuditEvent`, `Purpose`, the append-only and
versioning rules, the LLM-egress purpose convention.

**Lives in enforcement engine:** policy compilation, audit writers,
retention enforcement, alerting on policy violations.

### Runtime profile matrix

The same code paths run in all four profiles. Differences are *flags
on a single binary tree*, not forks.

| Concern | Solo CLI | Community Edition | Cloud (SaaS) | Enterprise on-prem |
|---|---|---|---|---|
| Classification enum present | yes (defaults only) | yes | yes + extensions | yes + extensions |
| Subjects present | yes | yes | yes | yes |
| All nodes classified `private` | yes | no (defaults) | no | no |
| Ingress policy bundle | OSS-defaults baked in | local config file | server-distributed | server-distributed, customer-keys |
| `Retriever.Filter` runs | yes — single-principal degenerate | yes | yes | yes |
| Agent tokens | local keychain-signed | server-signed, no SSO | server-signed + SSO | server-signed, customer signing keys allowed |
| Audit log | local file, 7-day | server, 30-day cap | server, 90-day default | server, customer-set retention, optional export to customer SIEM |
| LLM egress policy | OSS defaults (block on `.env`-derived content) | basic rules from config | full policy node UI | full + customer-controlled |
| Admin UI for policies | n/a | file-based only | full UI (separate spec) | full UI |
| Cross-org intelligence | n/a | n/a (org_id pinned to 1) | gated by classification | typically disabled |

**No data-model migration between profiles.** A solo CLI graph copied
into a CE server has nodes classified `private`; the CE migration
relabels them to `internal` on first server-side enrichment, with the
relabel itself audited.

### OSS/paid boundary and competitive risk

The contracts package is OSS-licensable. A third party could
reimplement the enforcement engine against the same contracts and ship
a competing product. We accept this risk for three reasons:

1. The contracts are *narrow on purpose* — they define vocabulary and
   API shape; they do not define the policy authoring UX, the policy
   compilation, the audit emission pipeline, the SSO integration, or
   the retention enforcement. Those are the moat.
2. We *want* third parties to build tools that speak Hero's governance
   language — exporters, dashboards, integrations — without writing a
   second "kind of governance" that fragments the ecosystem.
3. The audit-by-construction property is enforced by the *enforcement
   engine*. A third party can ship an engine that emits audit events;
   they can also ship one that doesn't. Our defense against that is
   the cloud-side audit *retention* and *queryability*, plus the fact
   that policy-as-graph means a customer can verify their own audit
   chain against their policy history. Customers buying for compliance
   reasons will not choose an engine that does not produce audits.

**Mitigation in contracts**: the `AuditToken` returned by `Filter` is
*required* by the contract for any downstream egress. A third-party
engine that returns nil tokens or skips audit emission is detectable
by the calling layers (which can assert), and we ship lints
(`hero check`) that verify a known-good engine is in use in any
non-solo-CLI profile.

## Acceptance Criteria

- THE SYSTEM SHALL define `Classification`, `Subject`, `Principal`,
  `Scope`, `PolicyNode`, `Rule`, `AuditEvent`, `AgentToken`, `Query`,
  `NodeDecision`, `AuditToken`, `Purpose`, and `Retriever` in
  `contracts/governance/` with stable, documented signatures.
- THE SYSTEM SHALL store a `classification` and `subjects` value on
  every graph node and reject writes that omit `classification` once
  the schema migration is in force.
- WHEN a graph migration encounters a node without a classification
  THE SYSTEM SHALL treat it as `internal` on read and enqueue it for
  backfill to the per-kind default.
- WHEN any code path retrieves graph nodes for any consumer (user,
  agent, enrichment worker, MCP server) THE SYSTEM SHALL invoke
  `Retriever.Filter` and use only the returned `allowed` set.
- WHEN `Retriever.Filter` returns a non-empty `allowed` set THE SYSTEM
  SHALL emit an `AuditEvent` referencing the policy version in force,
  the principal, the purpose, and the allowed node IDs, before the
  caller acts on the result.
- WHEN an enrichment node is created from input nodes THE SYSTEM SHALL
  set its classification to the maximum of the input classifications
  and the union of input subjects.
- WHILE a CLI is running without server contact THE SYSTEM SHALL
  enforce only the OSS-baked-in ingress defaults (`.env*`, `*.pem`,
  `.gitignore` paths) and tag all new nodes as classification
  `private`.
- WHILE an ingress policy bundle is past its `valid_through` hard
  expiry and refresh has failed THE SYSTEM SHALL fall back to
  deny-by-default for any capture targeted to classification above
  `internal`.
- WHEN an agent presents an `AgentToken` THE SYSTEM SHALL use the
  token's read scope, write scope, and `EgressClearance` for all
  subsequent operations, and SHALL tag every emitted audit event with
  the token's `AgentID`, `OnBehalfOf` user, and `SessionID`.
- IF an `AgentToken` is expired or appears on the revocation list
  THEN THE SYSTEM SHALL reject the operation with a structured error
  and emit an audit event with `Allowed=false`.
- WHEN a retrieval is invoked with `Purpose: llm_egress` and the
  effective egress policy denies the target THE SYSTEM SHALL return an
  empty or classification-reduced allowed set and SHALL emit an audit
  event recording the denial.
- WHERE the runtime profile is `community-edition` IS ENABLED THE
  SYSTEM SHALL pin `org_id` to 1, disable cross-org features, and cap
  audit retention at 30 days, using the same code paths as the cloud
  profile.
- WHERE the runtime profile is `enterprise-on-prem` IS ENABLED THE
  SYSTEM SHALL accept customer-supplied JWT signing keys and
  customer-set audit retention values without code changes.
- THE SYSTEM SHALL store policies as graph nodes that are versioned,
  supersede prior versions, and are themselves classified
  (default `restricted`).
- THE SYSTEM SHALL provide a query path that answers "what policy was
  in effect at time T for org O" by selecting the policy node whose
  effective range includes T.
- THE SYSTEM SHALL never expose an internal-only retrieval API that
  bypasses `Retriever.Filter`, including for administrative or
  support operations.
- THE SYSTEM SHALL emit audit events as append-only graph nodes of
  kind `event` with classification `restricted`.
- WHEN the audit emission queue is at capacity THE SYSTEM SHALL log a
  soft-fail warning on the read path but SHALL NOT block the read.
  (Deferred audit emission is tracked and reconciled by the engine.)
- `hero check` SHALL verify that the active enforcement engine
  produces audit events for filtered reads in non-solo-CLI profiles
  and SHALL fail the health check if it does not.
- A node copied from a solo CLI graph into a CE server SHALL retain
  its classification and SHALL be relabeled from `private` to the
  per-kind default only via an audited transition event.

## Risks

- **Retrieval performance regression.** `Filter` runs on every read.
  If policy compilation or audit emission is slow, the whole product
  feels slow. Mitigation: O(1) per-node matchers, in-memory compiled
  policies, async audit emission with bounded queue, benchmarks
  required on the delivery spec for the enforcement engine.
- **Migration of existing graphs.** Millions of nodes need
  classification. Wrong default leaks too much; conservative default
  hides too much. Mitigation: per-kind default table; `internal` on
  NULL read; backfill gate before team-mode enable.
- **Audit log itself becomes the leak.** Audit events name node IDs
  and principals; an attacker who can read the audit log can map
  classifications. Mitigation: audit events are themselves
  `restricted`; denied-node counts are bare integers, not IDs.
- **Contracts package is too narrow / too wide.** Too narrow and the
  enforcement engine has to bypass it; too wide and we leak moat
  surface. Mitigation: review the contracts surface in `cloud-admin`
  and `hero-community-edition` follow-on specs before locking 1.0.
- **Third-party engine that skips audits.** A competitor implements
  `Retriever` and never emits events. Mitigation: `hero check`
  verification, customers buy us for the audited engine, not the
  contracts.
- **Agent token sprawl.** Short-lived tokens are good for security
  and bad for UX if every agent run prompts for a new one.
  Mitigation: default 1h expiry with auto-refresh inside an active
  user session; agent SDK handles refresh transparently.
- **Classification creep.** Org admins keep adding levels until the
  set is unmanageable. Mitigation: the contracts limit the ordinal
  range (e.g. 8 levels max); extensions require a policy node, which
  itself shows up in reviews.
- **NULL-on-read leak window.** Between schema migration and backfill
  completion, NULL-classified nodes are read as `internal`. A node
  that *should be* `restricted` is briefly readable by anyone with
  internal clearance. Mitigation: pre-migration audit categorizes
  high-risk node kinds (tracker_import from private projects, etc.)
  for direct classification rather than NULL-then-backfill.

## Out of Scope

- Admin UI for authoring policies (handled in `cloud-admin` follow-on).
- SSO/SAML integration details (handled in `cloud-admin` and a
  dedicated `cloud-auth` follow-on).
- Specific PII regex catalogs and redaction-rule contents (handled by
  a dedicated policy-content spec, post-foundation).
- LLM-call wrapping mechanics — how the agent SDK actually intercepts
  context construction and routes to self-hosted vs. external models.
  This spec defines the *policy hook point* (the
  `Purpose: llm_egress` retrieval) but not the wrapping implementation.
- The complete subject vocabulary. We ship a starter set; orgs
  extend.
- Cross-org intelligence policies (handled in `cross-org-intelligence`).
- Tenant-level row isolation in Postgres (handled in
  `tenant-isolation-rls` — `Filter` is *above* RLS in the stack).
- Outpost credential storage and rotation (handled in `agent-outposts`
  — this spec only links agent tokens to outpost capabilities).
- The actual repo split mechanics (handled by the follow-on
  hero-cloud-repo-split spec — this spec defines the contracts package
  that lives at the seam).
- Community Edition packaging (handled by the follow-on
  hero-community-edition spec — this spec defines the runtime profile
  flag and the constraints that profile imposes).
- A cryptographic chain-of-custody / tamper-evident audit log
  (e.g. Merkle-anchored). Useful for some enterprise tiers; not
  required for the foundation.

## Open Questions

- **JWT vs. opaque tokens for agent identity.** JWT lets the CLI
  validate locally without round-tripping; opaque tokens give the
  server immediate revocation. Current draft picks JWT + short
  expiry + revocation cache, but enterprise customers may push back.
- **Subject type registry.** Should subject types be free strings
  (engineering simplicity, drift risk) or registered in the contracts
  package with extension via policy node (consistency, friction)?
  Current draft leans free-string-with-recommended-set; revisit before
  contracts 1.0.
- **Policy node language.** Are rules expressed as a typed struct
  (`Rule{Kind, Match, Action}`) or a DSL (e.g. Rego, Cedar)?
  Typed struct is simpler and matches Hero's spec-driven vibe; a DSL
  is more flexible for complex orgs. Current draft is typed struct;
  a future spec may layer a DSL on top.
- **Where does scope resolution against a user directory live?**
  Reading "user U has scope S" requires hitting the user/role
  store. That store is part of the enforcement engine but the
  contract assumes a `Principal` arrives at `Filter` already resolved.
  Need a clean seam — likely a `PrincipalResolver` interface in the
  contracts package, implemented in the engine.
- **What's the budget for audit-event volume?** A loaded session can
  trigger hundreds of retrievals. At full scale, audit volume could
  dwarf primary data. Need a back-of-envelope sizing before the
  enforcement-engine delivery spec.
- **Versioning of the contracts package itself.** Breaking changes to
  governance contracts are *very* expensive — every downstream tool
  breaks. SemVer is mandatory; the rule for what counts as a breaking
  change (adding a field? adding an enum value? changing default
  behavior?) needs to be written down in a separate convention spec.
