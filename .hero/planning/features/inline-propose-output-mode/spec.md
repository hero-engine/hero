---
title: Inline-Propose Output Mode — Agents Propose into the Artifact Pane
type: feature
status: designed
priority: P0
tags: [platform, domains, agents, dashboard, ui, registry, contract]
created: 2026-05-16
relations:
  - target: hero-domains
    kind: parent
  - target: hero-pm
    kind: blocks
  - target: hero-code
    kind: cross-repo-consumer
depends-on:
  - domain-plugin-architecture
  - domain-routing-and-agents
  - dashboard-view-registry
horizon: next
smoke: deferred
---

## Context

The Hero PM domain pack (`hero-pm/spec.md`) locks an inline-propose UX
pattern as the canonical agent-output shape: drafted AC, refined PRD
sections, prioritization changes, template proposals all appear *in
the artifact pane*, marked proposed, with accept / edit / reject
affordances. The chat shows only a lifecycle line; the artifact stays
source of truth. Mockup
`.hero/planning/features/hero-pm/mockups/08-inline-proposal.html` is
the visual source of truth.

Engineering agents today support only **write-to-disk**. The PM agent
roster (13 P0 agents) plus every contextual button in
`agent-pack-design.md` §G assumes inline-propose exists. `hero-code`
(the Rust dashboard) is **blocked on this design + delivery** — its
accept / edit / reject widget has no contract to honor until the
proposal wire shape and emission contract are specified.

The two PM commands that already accept `--inline-propose` as a flag
(`domains/pm/commands/refine.md`, `domains/pm/commands/prd.md`) and the
two agents that declare inline-propose support
(`domains/pm/agents/story-writer.md`, `prd-author.md`) are written
against the contract this spec pins down.

## Goal

Define and deliver the cross-language platform contract that lets any
agent emit a structured **proposal** targeting an artifact pane
instead of writing to disk. The contract has four parts:

1. **Proposal payload schema** — versioned JSON envelope an agent emits.
2. **Transport** — how the payload reaches the dashboard, and how
   accept / edit / reject actions reach the spec file on disk.
3. **Agent contract** — how `--inline-propose` mode is plumbed from
   slash command → command router → agent prompt, and what the agent
   prompt instruction says.
4. **View widget contract** — the dotted-border block, "proposed by
   `<agent>`" badge, three affordances, lifecycle log line, and the
   bulk-accept / bulk-reject ergonomics.

"Done" means a `story-writer` agent run with `--inline-propose` can
emit four AC bullets, the Rust dashboard renders them on the open
artifact pane with the locked visual treatment, and the user can
accept some, edit some, reject some — with each outcome committing
correctly to the spec file on disk and a lifecycle line landing in
the chat scroll.

## Resolved decisions

These were the six open questions from the design brief. Resolved:

### 1. Storage default — **transient session state** held by the daemon

Proposals live in-memory in the `hero serve` process, scoped to a
session. They are **not** written to a sidecar file and **not** spliced
into the spec file as draft blocks. On dashboard reconnect within the
same daemon process, pending proposals are re-served from memory; on
daemon restart, pending proposals are lost.

**Why.** The mockup shows transient-feeling behavior; v1 scope
explicitly defers a "saved drafts tray." Sidecar files churn the spec
parser, complicate lint, and force a migration if the schema shifts.
Marked draft blocks in the spec file pollute the canonical artifact
and break the "artifact is source of truth, proposals are the trace"
invariant. Transient lets us ship the contract without committing to a
storage format we'd retrofit later.

**Forward-compat hook.** The proposal envelope carries an explicit
`persistence: "transient"` field. v2 can introduce
`persistence: "sidecar"` without a wire-format break; parsers that
don't recognize it must reject the proposal with a structured error.
The daemon's in-memory store is the single point of behavior change
when persistence modes expand.

**Risk acknowledged.** Daemon crash loses pending proposals. Acceptable
for v1 — the agent run is cheap to repeat. If users complain, the
sidecar mode is the migration path.

### 2. Idempotency under repeat-propose — **per-target replacement, per-anchor**

When an agent re-runs in the same session and proposes against the
same `target_anchor`, the new proposal **replaces** any pending
proposals at that anchor by the same `origin.agent`. Proposals from a
*different* agent at the same anchor stack. Proposals from the same
agent at *different* anchors stack.

**Replacement rule.** If a pending proposal `P1` has
`target_anchor.kind = "section"`, `target_anchor.path = "acceptance-criteria"`,
`origin.agent = "story-writer"`, and a new run emits `P2` with the
same triple, the daemon drops `P1` and serves `P2`. The dashboard
receives a `proposal.replaced` event keyed by the new proposal id
plus the dropped id; the widget re-renders the AC region atomically.

**Why.** Stacking by default produces a confusing duplicate-bullets
state in the artifact. Replace-by-default makes "Refine again" a
clean overwrite. Cross-agent stacking is preserved because two
different agents proposing on the same section is a legitimate
multi-perspective state the user should see.

### 3. Edit affordance fidelity — **inline markdown editor, pre-filled**

"Edit" before accept opens an inline markdown editor inside the
proposed block, pre-filled with the proposal's `proposed_content`. The
editor is the same component the artifact's regular section editor
uses, sized to the proposed block. Submit converts the proposal to
`state: "edited"` (yellow border, "edited by `<user>` · proposed by
`<agent>`" badge per the mockup) without committing to disk; a
second click on "Accept edit" commits the edited content.

**Why.** Mockup shows the edited yellow-border state as a first-class
intermediate. A single-line input is too thin for AC clauses with
inline code spans; spawning the full artifact editor is heavy and
loses the proposed-block framing. Reusing the section editor inline
gets us slash, mentions, markdown, and inline code with no new
component.

**Cross-language note.** The edit affordance is **dashboard-side
only** — it does not round-trip through the Go daemon while editing.
The edited content posts back to the daemon on commit.

### 4. Cross-domain propose — **wire shape permits it; v1 scope excludes it**

The envelope carries `origin.domain` (e.g. `"pm"`, `"engineering"`)
and `target.domain` independently. v1 enforces
`origin.domain == target.domain` at the daemon — proposals violating
this are rejected with a structured error. v2 lifts the restriction
once `domain-scoped-knowledge-graph` (#6) ships and cross-domain edge
semantics are settled.

**Why.** The wire shape costs nothing now and would cost a versioned
break later. The restriction lives in the daemon, not in the wire,
so v2 is a server-side toggle.

### 5. Persistence of dismissed proposals — **per-session dismissal log; not cross-session**

When the user rejects a proposal, the daemon records the rejection in
the session's in-memory dismissal log: `(origin.agent, target_anchor,
content_hash)`. Subsequent runs by the same agent in the same session
that would emit a proposal matching that triple **suppress emission**;
the agent prompt receives a `dismissed_proposals` context block listing
the dismissals so it can adapt.

Dismissals do **not** persist across daemon restarts (consistent with
proposal transience). On restart the agent has no memory of prior
rejections and will re-propose.

**Why.** The agent loop ("did I already propose this and get
rejected?") matters within a session — without it, "Refine again"
re-emits the same rejected bullet. Persisting across sessions is a
separate concern: it requires a stable spec-attached store, which is
the same retrofit as the sidecar in #1. Bundle the two: when
persistence lands, dismissals persist too.

### 6. Lifecycle log line format — **pinned templates**

System events written to the chat scroll use the existing
`sys-event.inline-event` styling (see mockup, `.sys-event.inline-event`
class) and one of the following templates:

| Event | Template | Example |
|---|---|---|
| Drafted (proposals emitted) | `` `<agent>` drafted <N> <kind> → proposed inline in artifact (<anchor-hint>) `` | `` `story-writer` drafted 4 AC → proposed inline in artifact (acceptance-criteria) `` |
| Single accept | `` <user> accepted #<N> (<short-summary>) · <relative-time> `` | `Marcus accepted #1 (non-HTTPS) · just now` |
| Single edit | `` <user> edited #<N> (<short-summary>) · <relative-time> `` | `Marcus edited #2 (added synthetic-span wording) · 12s ago` |
| Single reject | `` <user> rejected #<N> (<short-summary>) · <relative-time> `` | `Marcus rejected #3 (out of scope) · 4s ago` |
| Summary (proposal set fully resolved) | `` `<agent>` drafted <N> <kind> → <A> accepted, <E> edited, <R> rejected `` | `` `story-writer` drafted 4 AC → 2 accepted, 1 edited, 1 rejected `` |

The drafted line emits immediately on proposal arrival. Per-proposal
accept/edit/reject lines emit on each user action. The summary line
emits when the last pending proposal in a batch is resolved (and
replaces or supersedes the drafted line in collapsed views). Templates
are produced by the daemon, not the agent — agents do not write to the
chat scroll directly.

## Design

### Storage model

All pending-proposal state lives in the `hero serve` daemon's
in-process memory. Two stores:

- `pendingProposals`: `map[sessionID]map[proposalID]Proposal` — current
  pending proposals scoped to a session.
- `dismissals`: `map[sessionID][]Dismissal` — rejected proposal
  fingerprints per session.

Both stores are flushed on daemon shutdown. No filesystem
representation; no `.hero/` write.

### Lifecycle

1. **Emit.** Agent run with `--inline-propose` produces one or more
   proposal envelopes on stdout (one JSON object per line, prefixed
   `HERO-PROPOSAL:` — see Transport).
2. **Ingest.** The daemon's proposal sink (a goroutine reading agent
   stdout for the session) parses lines, applies the per-anchor
   replacement rule, stores accepted-into-pending proposals, and
   fans out via SSE to subscribed dashboard clients.
3. **Render.** Dashboard subscribes to the session's SSE event stream;
   on `proposal.emitted` it renders a `<proposed-block>` widget in the
   target anchor on the open artifact pane.
4. **Act.** User clicks Accept / Edit / Reject. Dashboard POSTs to
   the daemon's action endpoint (see Transport). Daemon validates
   the action, applies the disk write (accept / edit) or drops the
   proposal (reject), updates state, emits a `proposal.resolved` SSE
   event, and writes the lifecycle log line to the session chat
   stream.
5. **Resolve.** When the last pending proposal in a batch is
   resolved, the daemon emits a `proposal.batch_resolved` event and
   the chat scroll renders the summary line.

### Agent contract

When an agent runs with `--inline-propose` in the invocation
environment (passed by the command router via `HERO_OUTPUT_MODE=inline-propose`
env var, plus `HERO_PROPOSE_TARGET_SPEC=<slug>`,
`HERO_PROPOSE_SESSION=<session-id>`):

- The agent's system prompt includes the **inline-propose addendum**
  (see "Agent system prompt addendum" below).
- Instead of editing the spec file directly with Write/Edit, the
  agent emits one or more **proposal envelopes** on stdout, one JSON
  object per line, each prefixed with the literal token
  `HERO-PROPOSAL: ` followed by the JSON.
- The agent does not write to the chat scroll about *what* it
  proposed — the daemon owns the lifecycle log line. The agent may
  emit a single conversational follow-up describing *that* it
  proposed (e.g. "Proposed 3 AC inline — accept/edit/reject each
  in the artifact pane").
- If the agent cannot determine a target anchor, it emits an
  envelope with `target_anchor.kind: "free-position"` and a
  `target_anchor.hint` string (e.g. "after AC list") and lets the
  dashboard render at the closest match.

The agent contract is **language-agnostic** — it's a stdout protocol,
not a Go or Rust API. Any agent runner (Claude Code, future agents)
can implement it.

### View widget contract

Mockup `08-inline-proposal.html` is the visual source of truth. The
contract Rust must honor:

- **Container.** Dotted-border block, `1.5px dashed
  var(--hero-blue-500)`, light-blue gradient fill, 8px radius. Three
  state variants:
  - **Proposed** (pending): blue dotted border, blue badge.
  - **Edited**: orange dotted border (`var(--warning)`), orange
    badge "edited by `<user>` · proposed by `<agent>`".
  - **Accepted**: solid green border, green badge, 0.85 opacity, no
    action row.
- **Badge.** "proposed by `<agent>`" with a pulsing dot. CSS class
  `proposed-badge` per mockup. Positioned `-10px top, 12px left`
  absolute on the container.
- **Three affordances.** Accept (green text, primary), Edit (default
  text), Reject (muted text). Keyboard hints: `⏎ accept · E edit · ⌫
  reject`. Buttons fire the action POST (see Transport).
- **Edit state transitions** in-place to an editable textarea with
  the proposal content pre-filled; the action row swaps to "Accept
  edit / Continue editing / Revert". On accept-edit the block
  transitions through edited (yellow) → accepted (green) → fades.
- **Bulk affordances.** When a proposal batch has 2+ pending
  proposals, the artifact's **bottom strip** surfaces an "Accept all
  proposals" button (see mockup line 387–390). The button hits a
  daemon endpoint that accepts every pending proposal in the batch
  atomically. A symmetric "Reject all pending" appears only when
  some proposals in the batch have already been accepted or edited
  (avoid an accidental nuke of the whole batch).
- **Lifecycle log lines.** Render as `<sys-event class="inline-event">`
  in the chat scroll. The daemon emits the text; the dashboard
  styles it.

The contract is enforced by integration tests in hero-code (Rust)
that render each state from a fixture envelope.

### Multi-proposal semantics

A single agent run can emit N proposals in one batch (e.g.
`story-writer` drafts 4 AC at once). All proposals in a batch share a
`batch_id` (UUID generated by the daemon on first envelope of a run,
or carried by the agent if it generates batch-level IDs).

- **Per-proposal actions** are independent: accept #1, edit #2,
  reject #3, leave #4 pending — all valid.
- **Bulk accept** iterates the pending set in declaration order and
  applies each accept; ordering matters because two proposals
  targeting the same list anchor accumulate in emission order.
- **Bulk reject** drops every pending proposal in the batch and
  records dismissals for each.
- **Partial state and re-run.** If the user accepts #1 and #2, then
  triggers "Refine again", the new run's replacement rule applies
  to *pending* proposals only — accepted ones are already on disk
  and immutable from the proposal layer's view.

## Cross-language contract

This is the load-bearing section. The Rust dashboard in `hero-code`
implements every endpoint and renders every event documented here.
Schema is versioned; v1 is the only version that exists. Breaking
changes bump to v2 and the daemon serves both for a deprecation
window.

### Proposal envelope (v1)

Emitted by the agent on stdout, one per line, prefixed
`HERO-PROPOSAL: `. JSON shape:

```json
{
  "schema_version": "v1",
  "proposal_id": "0193b3f0-9c14-7c20-9234-2a4d8b1f0e91",
  "batch_id": "0193b3f0-9c14-7c20-9234-2a4d8b1f0e90",
  "session_id": "sess-2026-05-16-14-32-01",
  "persistence": "transient",
  "origin": {
    "agent": "story-writer",
    "domain": "pm",
    "skill_chain": ["ac-writing", "ears-format"],
    "command": "/refine",
    "emitted_at": "2026-05-16T14:33:12.018Z"
  },
  "target": {
    "spec_slug": "story-2849",
    "spec_type": "story",
    "domain": "pm"
  },
  "target_anchor": {
    "kind": "list-item-append",
    "section_path": "acceptance-criteria",
    "hint": "after AC list"
  },
  "proposed_content": {
    "format": "markdown",
    "kind": "ac-bullet",
    "body": "WHEN an outgoing HTTP request is to a non-HTTPS endpoint THE SYSTEM SHALL still inject the `traceparent` header (no security restriction — W3C spec applies to all schemes)."
  },
  "rationale": "Existing AC don't cover non-HTTPS; W3C spec is scheme-agnostic.",
  "supersedes": []
}
```

**Field semantics.**

| Field | Type | Required | Notes |
|---|---|---|---|
| `schema_version` | string | yes | `"v1"`. Parsers reject unknown versions. |
| `proposal_id` | UUIDv7 string | yes | Globally unique. Used as the correlation key for accept/edit/reject. |
| `batch_id` | UUIDv7 string | yes | Shared across all proposals emitted in the same agent run. |
| `session_id` | string | yes | The Hero session the agent is running under. Daemon validates the session exists. |
| `persistence` | enum `"transient"` | yes | v1 accepts only `"transient"`. Forward-compat for `"sidecar"` in v2. |
| `origin.agent` | string | yes | Agent name (e.g. `story-writer`). Drives the badge text and the dismissal fingerprint. |
| `origin.domain` | string | yes | Domain pack (`"pm"`, `"engineering"`). Used for cross-domain rejection in v1. |
| `origin.skill_chain` | string[] | no | Skills the agent had loaded. Audit-only. |
| `origin.command` | string | no | The slash command that invoked the agent. Shown as "via `<command>`" in the chat. |
| `origin.emitted_at` | RFC3339 string | yes | Agent's wall clock at emission. |
| `target.spec_slug` | string | yes | Slug of the target spec. |
| `target.spec_type` | string | yes | Type (`story`, `prd`, `feature`, etc.). |
| `target.domain` | string | yes | Domain of the target spec. v1: must equal `origin.domain`. |
| `target_anchor.kind` | enum | yes | One of: `frontmatter-field`, `section-replace`, `section-append`, `list-item-append`, `list-item-replace`, `list-item-insert-after`, `free-position`. |
| `target_anchor.section_path` | string | conditional | Heading slug path, e.g. `acceptance-criteria` or `risks/regression`. Required for section/list anchors. |
| `target_anchor.field_name` | string | conditional | Frontmatter field name. Required for `frontmatter-field`. |
| `target_anchor.list_item_index` | int | conditional | Zero-based index. Required for `list-item-replace`/`list-item-insert-after`. |
| `target_anchor.hint` | string | no | Human-readable hint for `free-position` or when section_path can't be resolved exactly. |
| `proposed_content.format` | enum | yes | `"markdown"` (v1 default). Forward-compat for `"structured"`. |
| `proposed_content.kind` | string | yes | What kind of content this is (`ac-bullet`, `section`, `field-value`, `ordering`, etc.). Drives widget rendering hints. |
| `proposed_content.body` | string | yes | The content. Markdown for `format: "markdown"`. |
| `rationale` | string | no | Optional short explanation. Surfaced in a tooltip on the proposed block. |
| `supersedes` | UUIDv7[] | no | Proposal ids this one explicitly replaces (used when an agent self-corrects mid-batch). |

### Transport

Three transport paths, all over the existing `hero serve` HTTP daemon
on localhost:7437. No new server framework; this extends the existing
REST + SSE surface.

**1. Agent → daemon (proposal ingestion).**

The daemon spawns agent subprocesses on the agent runner's behalf
(or, in the case of Claude Code agents, the agent runner forwards
its stdout to the daemon's `POST /api/{project}/sessions/{session_id}/proposals/ingest`
endpoint). The ingest endpoint accepts NDJSON: one proposal envelope
per line, body unbounded streaming. Auth: same bearer token as the
rest of the API.

Alternative path (for cases where the daemon doesn't own the
agent subprocess): the agent writes proposals to its stdout with
the `HERO-PROPOSAL: ` prefix; an agent-runner shim (`hero agent run`,
a new subcommand) tails the stdout, extracts prefixed lines,
re-posts them to the ingest endpoint. The shim is the standard path
for Claude Code agents.

**2. Daemon → dashboard (proposal fan-out).**

The dashboard subscribes to the existing SSE event stream
(`GET /api/events?session=<session-id>`). Three new event types:

| Event | Payload |
|---|---|
| `proposal.emitted` | The full proposal envelope. |
| `proposal.replaced` | `{new_proposal_id, dropped_proposal_id}`. |
| `proposal.resolved` | `{proposal_id, outcome: "accepted"\|"edited"\|"rejected", edited_content?: string}`. |
| `proposal.batch_resolved` | `{batch_id, accepted: int, edited: int, rejected: int}`. |
| `proposal.lifecycle_log` | `{batch_id?, proposal_id?, template, rendered_text}`. |

The dashboard maintains a per-session proposal cache keyed by
`proposal_id`; on reconnect it re-requests the current state via
`GET /api/{project}/sessions/{session_id}/proposals` (returns the
pending set).

**3. Dashboard → daemon (action commits).**

Three POST endpoints, all under `/api/{project}/sessions/{session_id}/proposals/{proposal_id}/`:

| Endpoint | Body | Effect |
|---|---|---|
| `POST .../accept` | `{}` | Apply the proposal's `proposed_content` at `target_anchor` in the spec file on disk. Remove from pending. Emit `proposal.resolved` (outcome: accepted) and the per-proposal lifecycle log line. |
| `POST .../edit-accept` | `{edited_content: string}` | Same as accept, but use `edited_content` instead of the original `proposed_content.body`. Emit `proposal.resolved` (outcome: edited). |
| `POST .../reject` | `{reason?: string}` | Drop the proposal; record a dismissal `(origin.agent, target_anchor, content_hash)`. Emit `proposal.resolved` (outcome: rejected). |

Bulk endpoints (POST `/api/{project}/sessions/{session_id}/proposals/batches/{batch_id}/accept-all`
and `.../reject-pending`) iterate the pending set in declaration order
and apply the same actions atomically. A bulk action failure aborts
the bulk operation; partial state is preserved and reported in the
response (`{accepted: [ids], failed: [{id, error}]}`).

**Atomicity and conflict handling for disk writes.**

- Each `accept`/`edit-accept` action takes a file lock on the target
  spec, reads the current file, applies the anchor-targeted edit,
  writes the result atomically (write-temp + rename), releases the
  lock.
- If the spec file mtime has advanced since the proposal was emitted
  (user manually edited the spec), the action returns HTTP 409 with
  `{error: "spec_modified", reload_required: true}`. The dashboard
  refreshes the artifact pane and re-evaluates whether the proposal
  is still applicable (the anchor may no longer exist).

### Agent system prompt addendum

When `HERO_OUTPUT_MODE=inline-propose` is set, the agent loader
prepends this addendum to the agent's system prompt:

```
## Output mode: inline-propose

You are running in inline-propose mode. Do NOT edit the target spec
file directly. Instead, emit one or more structured proposals on
stdout, one per line, each prefixed with the literal token
"HERO-PROPOSAL: " followed by a JSON envelope.

Envelope schema (v1):
- proposal_id: generate a UUIDv7
- batch_id: generate one UUIDv7 for the run; reuse for every
  proposal in this run
- session_id: "${HERO_PROPOSE_SESSION}"
- target.spec_slug: "${HERO_PROPOSE_TARGET_SPEC}"
- target_anchor: choose the most specific anchor you can identify
  in the target spec (frontmatter-field, section-append,
  list-item-append, etc.)
- proposed_content.body: the markdown content you would have written
- rationale: optional one-sentence explanation

If you already proposed something in this session that was rejected,
you will see it in the dismissed_proposals context block. Do not
re-propose those.

The chat scroll will receive a lifecycle log line automatically —
do NOT narrate what you proposed in your conversational output. A
single short follow-up ("Proposed N <kind> inline") is fine.
```

The addendum is loaded from `domains/<active>/agent-prompts/inline-propose-addendum.md`
so domains can override the wording without core changes; the default
lives in `domains/_platform/agent-prompts/inline-propose-addendum.md`.

### Command router thread-through

The slash command parser already accepts arbitrary `--flag` arguments
on commands; threading `--inline-propose` through to the agent loader
requires:

1. The command registry's `Invoke` step inspects the parsed flag set;
   if `--inline-propose` is present, it sets the agent runner's
   environment to include `HERO_OUTPUT_MODE=inline-propose`,
   `HERO_PROPOSE_TARGET_SPEC=<slug-derived-from-args-or-context>`,
   `HERO_PROPOSE_SESSION=<current-session-id>`.
2. Contextual buttons defined by the view registry declare their
   button manifest with `{command: "/refine", args: ["--section",
   "ac"], inline_propose: true}`. The dashboard, on click, hits the
   daemon's `POST /api/{project}/sessions/{session_id}/invoke`
   endpoint with the manifest; the daemon constructs the agent run
   with the right env.

## Changes

Implementation is Go-side except where noted. No Rust code lands in
this repo (hero-code consumes the contract).

1. **`internal/proposals/` (new package)** — in-memory proposal store
   and dismissal log. Types: `Proposal`, `Dismissal`, `Batch`,
   `Anchor`. Functions: `Store.Ingest(envelope)`,
   `Store.Resolve(id, outcome, edited?)`,
   `Store.Pending(sessionID)`, `Store.RecordDismissal(...)`,
   `Store.MatchingDismissals(agent, anchor)`. Per-session mutex; no
   filesystem.

2. **`internal/proposals/envelope.go` (new)** — proposal envelope
   JSON struct definitions with `schema_version: "v1"` validation.
   Reject unknown versions with structured error.
   `ParseProposalLine(line []byte) (Envelope, error)` for stdout
   line parsing. Round-trip tests.

3. **`internal/proposals/apply.go` (new)** — anchor-targeted spec
   file edits. One function per anchor kind: `applyFrontmatterField`,
   `applySectionReplace`, `applySectionAppend`,
   `applyListItemAppend`, `applyListItemReplace`,
   `applyListItemInsertAfter`, `applyFreePosition`. Each takes the
   spec markdown, applies the edit, returns the new markdown plus a
   structured diff for logging. File-lock + atomic write happens in
   the caller.

4. **`internal/cli/agent_run.go` (new)** — `hero agent run <agent>`
   subcommand. The shim that wraps a Claude Code agent invocation,
   tails its stdout for `HERO-PROPOSAL: ` lines, posts each to the
   daemon's ingest endpoint. Forwards non-prefixed stdout to the
   parent process. Honors `HERO_OUTPUT_MODE`, `HERO_PROPOSE_SESSION`,
   `HERO_PROPOSE_TARGET_SPEC` env vars (passes through to the agent
   subprocess).

5. **`internal/serve/proposals_routes.go` (new)** — HTTP handlers for
   the five new endpoints:
   - `POST /api/{project}/sessions/{session_id}/proposals/ingest` (NDJSON)
   - `GET /api/{project}/sessions/{session_id}/proposals` (pending set)
   - `POST /api/{project}/sessions/{session_id}/proposals/{proposal_id}/accept`
   - `POST /api/{project}/sessions/{session_id}/proposals/{proposal_id}/edit-accept`
   - `POST /api/{project}/sessions/{session_id}/proposals/{proposal_id}/reject`
   - `POST /api/{project}/sessions/{session_id}/proposals/batches/{batch_id}/accept-all`
   - `POST /api/{project}/sessions/{session_id}/proposals/batches/{batch_id}/reject-pending`
   - `POST /api/{project}/sessions/{session_id}/invoke` (contextual-button entry; constructs the agent run)
   All routes auth-token-gated.

6. **`internal/serve/events.go` (extend)** — register five new SSE
   event types (`proposal.emitted`, `proposal.replaced`,
   `proposal.resolved`, `proposal.batch_resolved`,
   `proposal.lifecycle_log`). Fan-out from the proposal store on
   state changes.

7. **`internal/serve/lifecycle_log.go` (new)** — renders the six
   pinned lifecycle log templates. Single source of truth for log
   line text; consumed by the proposal resolver and emitted via the
   SSE event stream.

8. **`internal/cli/commands/router.go` (extend)** — when a parsed
   slash command has `--inline-propose`, set the agent run's env to
   the three `HERO_PROPOSE_*` vars and require the daemon to be
   reachable (the contract depends on the daemon for ingest).
   Without the daemon, fall back to a clear error: "inline-propose
   requires `hero serve`."

9. **`domains/_platform/agent-prompts/inline-propose-addendum.md`
   (new)** — the system-prompt addendum text. Loaded by the agent
   loader (`domain-routing-and-agents` primitive) when
   `HERO_OUTPUT_MODE=inline-propose`.

10. **`internal/agent/loader.go` (extend; lands as part of
    `domain-routing-and-agents`, depended on)** — when
    `HERO_OUTPUT_MODE=inline-propose` is set, append the addendum to
    the agent's system prompt and prepend a `dismissed_proposals`
    block populated from `Store.MatchingDismissals(...)` for the
    current agent and session.

11. **`docs/contracts/inline-propose-v1.md` (new)** — public-facing
    contract document for hero-code (and future client
    implementers). Includes the envelope schema, the endpoint list,
    the SSE event types, and the lifecycle log templates.
    hero-code's Rust integration tests link to this doc.

12. **Tests** —
    - Unit: envelope round-trip, anchor application for each anchor
      kind, replacement rule (same-agent-same-anchor replace),
      stacking rule (different-agent-same-anchor stack), dismissal
      suppression.
    - Integration: spin up `hero serve`, invoke a fake agent that
      emits a known proposal batch, exercise accept/edit/reject via
      the HTTP endpoints, assert the spec file on disk matches the
      expected after-state, assert the SSE event stream emits the
      expected events in order, assert lifecycle log lines render
      with the right template.
    - E2E (deferred to hero-code): the Rust widget renders each
      state from a fixture envelope. Owned by hero-code; this repo
      ships the fixture JSON in `testdata/proposals/`.

13. **`hero.json` config schema (extend)** — add an
    `inline_propose:` block under the workspace config. Fields:
    `enabled` (bool, default true), `default_persistence` (enum,
    v1: only `"transient"`). Forward-compat for `"sidecar"` later.

## Acceptance Criteria

- WHEN an agent run with `HERO_OUTPUT_MODE=inline-propose` emits a
  valid v1 proposal envelope on stdout prefixed `HERO-PROPOSAL: `
  THE SYSTEM SHALL ingest it into the daemon's pending set and emit
  a `proposal.emitted` SSE event within 100ms.
- WHEN the dashboard POSTs to the accept endpoint for a pending
  proposal THE SYSTEM SHALL apply the proposal's content at the
  target anchor in the spec file on disk atomically and emit a
  `proposal.resolved` event with outcome `"accepted"`.
- WHEN the dashboard POSTs to the edit-accept endpoint with
  `edited_content` THE SYSTEM SHALL apply the edited content
  (not the original `proposed_content.body`) at the target anchor
  and emit a `proposal.resolved` event with outcome `"edited"`.
- WHEN the dashboard POSTs to the reject endpoint THE SYSTEM SHALL
  drop the proposal from pending, record a dismissal fingerprint
  for the current session, and emit a `proposal.resolved` event
  with outcome `"rejected"`.
- WHEN an agent emits a proposal whose `(origin.agent,
  target_anchor)` matches a pending proposal in the same session
  THE SYSTEM SHALL drop the prior proposal and emit a
  `proposal.replaced` event.
- WHEN an agent emits a proposal whose `(origin.agent,
  target_anchor, content_hash)` matches a recorded dismissal in the
  same session THE SYSTEM SHALL suppress emission and surface the
  match in the `dismissed_proposals` context block on the agent's
  next prompt.
- IF a proposal envelope has `schema_version` other than `"v1"`
  THEN THE SYSTEM SHALL reject the envelope with a structured error
  and not enter the pending set.
- IF a proposal envelope has `origin.domain != target.domain` in
  v1 THEN THE SYSTEM SHALL reject the envelope with error code
  `"cross_domain_unsupported"`.
- IF the spec file mtime has advanced since proposal emission
  THEN THE SYSTEM SHALL return HTTP 409 from the accept endpoint
  with `{error: "spec_modified", reload_required: true}`.
- WHEN the last pending proposal in a batch is resolved THE SYSTEM
  SHALL emit a `proposal.batch_resolved` event with accept/edit/
  reject counts and a `proposal.lifecycle_log` event carrying the
  summary template.
- WHILE the daemon is shut down THE SYSTEM SHALL discard all
  pending proposals and all dismissal fingerprints (v1 transient
  semantics).
- WHERE `inline_propose.enabled` IS FALSE in `hero.json` THE SYSTEM
  SHALL reject `--inline-propose` invocations with a clear error
  pointing to the config.
- THE SYSTEM SHALL emit lifecycle log lines using the pinned
  templates from the lifecycle log table in this spec.

## Boundaries

- **Not** designing how the Rust widget is implemented internally
  (state machine, component framework choice). hero-code owns the
  view-side implementation against the contract.
- **Not** persisting proposals across daemon restarts in v1.
  Sidecar persistence is a v2 retrofit gated by user demand.
- **Not** supporting cross-domain proposals in v1. The wire shape
  permits it; the daemon enforces same-domain.
- **Not** supporting concurrent authors proposing on the same
  artifact. v1 assumes one author per session, consistent with
  hero-pm's single-author scope.
- **Not** a generic "review diffs" surface for arbitrary file
  edits. Proposals target *artifact specs*, not source files.
- **Not** plumbing inline-propose through `/deliver`, `/diagnose`,
  or other write-to-disk commands. Those continue to write directly.
- **Not** introducing a new framework for agent transport. The
  contract extends the existing HTTP daemon + SSE event stream.

## Risks

- **Stdout prefix collision.** If an agent's prose ever contains a
  line starting with `HERO-PROPOSAL: ` the shim will mis-parse.
  Mitigation: the shim requires the line to be valid JSON after the
  prefix and to validate against the envelope schema; invalid lines
  log a warning and pass through as conversational output. A
  malicious agent could still inject; trust boundary is the agent.
- **SSE event ordering across reconnects.** If the dashboard
  reconnects mid-batch, it must reconcile the pending set against
  events received post-reconnect. Mitigation: the dashboard
  re-requests `GET /api/{project}/sessions/{session_id}/proposals`
  on every reconnect; events received during reconnect have
  monotonic `event_id` for de-duplication.
- **Anchor resolution drift.** A `section-append` anchor targeting
  `acceptance-criteria` fails if the section was renamed since
  emission. Mitigation: the `applyXxx` functions return a structured
  error; the accept endpoint returns HTTP 409 with
  `{error: "anchor_not_found"}` and the dashboard renders the
  proposal as "stale" with a re-target affordance.
- **Daemon-required friction.** Inline-propose requires `hero serve`
  running. CLI-only sessions get a clear error. Acceptable in v1
  because the UX target is the dashboard.
- **Bulk-accept anchor order interactions.** Two proposals
  targeting `list-item-append` at the same section accumulate in
  emission order. If the agent emitted them in semantically wrong
  order, bulk-accept commits them wrong. Mitigation: the agent
  prompt addendum advises ordering bullets in intended final order
  per anchor.
- **File-lock contention with `hero index` / `hero check`.** Long
  reads on the spec corpus could block proposal applies. Mitigation:
  the accept handler uses a short read-then-write critical section,
  not a long-held lock across the whole spec corpus.
- **Cross-language schema drift.** Rust adds a field client-side,
  Go ignores it. Mitigation: schema versioning is strict — unknown
  fields are accepted on ingest (server is liberal) but the
  fixture-based hero-code integration test asserts the documented
  envelope round-trips without loss. The public contract doc is
  the single source of truth.

## Validation

- **Unit tests.** Cover envelope parse, anchor application for each
  kind, replacement rule, stacking rule, dismissal suppression,
  schema-version rejection, cross-domain rejection.
- **Integration tests.** Spin `hero serve` in a temp workspace.
  Run a fixture agent (a Go binary that emits canned proposals via
  the shim). Exercise accept/edit/reject endpoints. Assert: spec
  file on disk matches expected, SSE event stream emits expected
  event sequence, lifecycle log lines render with pinned templates.
- **Contract fixtures.** Ship `testdata/proposals/v1/*.json` —
  one per envelope variant (each anchor kind, edited state,
  replacement scenario, dismissal scenario). hero-code's Rust
  widget tests consume these to verify the cross-language contract.
- **Hero-code handoff.** Once delivery completes, hand off to
  hero-code via `hero handoff inline-propose-output-mode hero-code`
  with the contract doc, fixtures, and a smoke checklist.

## Kickoff

Run `/deliver inline-propose-output-mode`. Confirm the three
dependency primitives (`domain-plugin-architecture`,
`domain-routing-and-agents`, `dashboard-view-registry`) are in a
state where their integration points are accessible — at minimum,
`domains/_platform/` exists for the agent prompt addendum and the
agent loader has the hook for env-conditional prompt augmentation.
If those primitives haven't landed, deliver the daemon-side pieces
(envelope, store, routes, lifecycle log, shim) standalone first and
stub the agent-prompt integration as a follow-up.

→ `/deliver inline-propose-output-mode`

**Files:** .hero/planning/features/inline-propose-output-mode/spec.md, docs/contracts/inline-propose-v1.md (new), internal/proposals/ (new), internal/serve/proposals_routes.go (new), internal/cli/agent_run.go (new), domains/_platform/agent-prompts/inline-propose-addendum.md (new), testdata/proposals/v1/ (new), .hero/planning/features/hero-pm/spec.md (consumer), .hero/planning/features/hero-pm/mockups/08-inline-proposal.html (visual source of truth)
**Skip:** sidecar persistence (v2), cross-domain proposals (v2), multi-author concurrent proposals (Hero Cloud), proposals on arbitrary source files (out of scope), inline-propose for `/deliver` and `/diagnose` (continue write-to-disk).
