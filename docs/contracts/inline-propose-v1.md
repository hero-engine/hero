# Inline-Propose Contract — v1.0

**Status:** stable
**Contract version:** `1.0`
**Owner:** hero (Go) — produces; hero-code (Rust dashboard) — consumes.

This document is the **load-bearing cross-language contract** for
inline-propose. It pins the wire shapes (envelope, REST surface, SSE
events) so the Rust dashboard can render proposals without further
coupling to the Hero Go implementation.

Schema-version semantics: additive-only changes within `1.x`; a
breaking change bumps to `2.0`. The daemon advertises the supported
schema versions on each envelope it relays (`schema_version` field).

---

## 1. Wire transport (stdout → daemon)

Agents running under `--inline-propose` emit **structured proposals on
stdout** as NDJSON lines prefixed by the literal token
`HERO-PROPOSAL: `. Everything else on stdout is forwarded as agent
chatter and not parsed.

```
HERO-PROPOSAL: {"schema_version":"1.0","batch_id":"b-abc123",...}
```

Lines that begin with the prefix but are not valid JSON, or that
parse but fail envelope validation, are logged to stderr (`hero
agent propose-shim: invalid envelope: <reason>`) and dropped — they
do not abort the agent.

The shim (`hero agent propose-shim`) reads child stdout line-by-line,
strips the prefix, parses the JSON, and POSTs each envelope to the
daemon at:

```
POST http://127.0.0.1:7437/api/{project}/sessions/{session_id}/proposals/ingest
Content-Type: application/json
```

Non-prefixed stdout passes through to the shim's own stdout
unchanged; agent stderr is forwarded verbatim. The shim's exit code
mirrors the agent's exit code.

---

## 2. Envelope schema

A single proposal envelope. One envelope = one independently
accept/reject-able unit. Batches are expressed by sharing a `batch_id`
across multiple envelopes.

```json
{
  "schema_version": "1.0",
  "proposal_id": "p-7c1f9c",
  "batch_id": "b-abc123",
  "session_id": "sess-2026-05-17-a3f",
  "agent": "story-writer",
  "skill_chain": ["pm/story-writer", "core/acceptance-criteria"],
  "target": {
    "spec_slug": "csv-export",
    "spec_path": ".hero/planning/features/csv-export/spec.md",
    "anchor": {
      "kind": "section",
      "value": "acceptance_criteria",
      "position": "append"
    }
  },
  "content": {
    "format": "markdown",
    "body": "- THE SYSTEM SHALL emit a UTF-8 BOM when `--bom` is passed."
  },
  "rationale": "Excel still defaults to system codepage without a BOM.",
  "emitted_at": "2026-05-17T14:22:09Z"
}
```

### Field definitions

| Field | Type | Required | Notes |
|---|---|---|---|
| `schema_version` | string | yes | Semver, currently `"1.0"` |
| `proposal_id` | string | yes | Unique within session; agent-minted (`p-` + 6 hex) |
| `batch_id` | string | yes | Groups proposals that should bulk-accept together; agent-minted (`b-` + 6 hex). A solo proposal uses a fresh batch_id with one member. |
| `session_id` | string | yes | Caller session; the agent inherits from `HERO_SESSION_ID` env var, set by the shim |
| `agent` | string | yes | Origin agent slug (e.g. `story-writer`) |
| `skill_chain` | array<string> | optional | Skills invoked to produce the proposal; informational |
| `target.spec_slug` | string | yes | Slug of the spec to modify |
| `target.spec_path` | string | optional | Hint for the daemon; daemon resolves authoritatively from `spec_slug` |
| `target.anchor.kind` | enum | yes | `frontmatter` \| `section` \| `heading` \| `list_item` \| `free` |
| `target.anchor.value` | string | yes | Anchor identifier — frontmatter field name, section heading slug, list-item id, or free-form position label |
| `target.anchor.position` | enum | optional, default `replace` | `replace` \| `append` \| `prepend` \| `before` \| `after` |
| `content.format` | enum | yes | `markdown` \| `text` \| `yaml` (for frontmatter targets) |
| `content.body` | string | yes | The proposed content; the daemon stores it as-is and does not interpret |
| `rationale` | string | optional | Human-readable explanation; displayed in dashboard |
| `emitted_at` | RFC3339 string | optional | Daemon stamps if missing |

### Anchor scoping (Decision 2 — per-anchor replacement)

The daemon's proposal store keys on
`(session_id, target.spec_slug, target.anchor.kind, target.anchor.value, agent)`.
When an envelope arrives with a key that already has a pending
proposal, the new envelope **replaces** the existing pending proposal
(and emits `proposal_emitted` again; the old proposal is dropped
silently). This makes "agent re-runs in the same session" idempotent
on a per-anchor basis, scoped to the same agent.

Two different agents proposing on the same anchor stack — both
proposals are surfaced; the user resolves.

---

## 3. REST surface

All endpoints are scoped to `/api/{project}/sessions/{session_id}/proposals`.
All responses are JSON.

### `POST /api/{project}/sessions/{session_id}/proposals/ingest`

Body: a single envelope (per §2). On success returns `{"proposal_id": "..."}`
with HTTP 202. Validation failures return 400 with `{"error": "..."}`.

### `GET /api/{project}/sessions/{session_id}/proposals`

Lists pending proposals for a session.

Query parameters:
- `spec_slug` — filter to one spec
- `batch_id` — filter to one batch
- `agent` — filter to one agent

Response: `{"proposals": [<envelope>...], "count": N}`.

### `POST /api/{project}/sessions/{session_id}/proposals/{proposal_id}/accept`

Body: optional `{"by": "user"}`. Removes the proposal from the store
and emits `proposal_accepted`. The actual disk write is the caller's
responsibility (dashboard) — the daemon only manages the lifecycle.

### `POST /api/{project}/sessions/{session_id}/proposals/{proposal_id}/edit-accept`

Body: `{"edited_body": "...", "by": "user"}`. Removes the proposal,
emits `proposal_edited` carrying the edited body in the event payload.

### `POST /api/{project}/sessions/{session_id}/proposals/{proposal_id}/reject`

Body: optional `{"by": "user", "reason": "..."}`. Removes the proposal,
emits `proposal_rejected`.

### `POST /api/{project}/sessions/{session_id}/proposals/{proposal_id}/dismiss`

Body: optional `{"by": "user"}`. Same as reject but emits
`proposal_dismissed` (used by the dashboard when the user closes the
artifact without acting; semantically distinct in lifecycle logs).

### Bulk variants — `POST /api/{project}/sessions/{session_id}/proposals/bulk/{action}`

Where `{action}` is `accept` | `reject` | `dismiss`.

Body: `{"batch_id": "b-abc123", "by": "user"}`. Applies the action to
every pending proposal sharing that `batch_id` in the session.
Response: `{"applied": [<proposal_id>...], "count": N}`.

`bulk/edit-accept` is intentionally **not** offered — editing is per
proposal.

---

## 4. SSE events on `/api/events`

Five new event types extend the existing SSE stream. Each event
includes a `payload` field carrying the relevant envelope (for
`proposal_emitted`) or a lifecycle record (for the others).

| Event type | When | Payload shape |
|---|---|---|
| `proposal_emitted` | Ingest accepted | The full envelope (§2) |
| `proposal_accepted` | Accept endpoint | `{proposal_id, batch_id, spec_slug, anchor, agent, by}` |
| `proposal_edited` | Edit-accept endpoint | `{proposal_id, batch_id, spec_slug, anchor, agent, by, edited_body}` |
| `proposal_rejected` | Reject endpoint | `{proposal_id, batch_id, spec_slug, anchor, agent, by, reason?}` |
| `proposal_dismissed` | Dismiss endpoint | `{proposal_id, batch_id, spec_slug, anchor, agent, by}` |

SSE frame layout:

```
event: proposal_emitted
data: {"project":"hero","session_id":"sess-...","payload":{...envelope...},"timestamp":"2026-05-17T14:22:09Z"}

```

`event:` carries the type, `data:` carries the JSON body.

---

## 5. Lifecycle log line format

When a batch closes (all proposals in a batch have been accepted,
edited, rejected, or dismissed), the daemon emits one summary
lifecycle log line to its own stderr in the shape:

```
<agent> drafted N <noun> → A accepted, E edited, R rejected, D dismissed
```

Examples:

```
story-writer drafted 4 proposals → 3 accepted, 1 edited, 0 rejected, 0 dismissed
prd-author drafted 2 proposals → 0 accepted, 0 edited, 2 rejected, 0 dismissed
```

The noun defaults to `proposals`; the dashboard may localize per
artifact type (`AC`, `sections`, etc.) when rendering its own chat
log line.

---

## 6. Session and storage semantics

- **Transient session state** (Decision 1). Proposals live in
  process memory keyed by session. The daemon does not persist
  proposals across restarts. If the daemon restarts mid-session,
  pending proposals are lost; the dashboard reconnects to SSE and
  starts fresh.
- **Per-anchor replacement scoped to same agent** (Decision 2). See §2.
- **No cross-session visibility.** Proposals in session A are not
  visible to session B even within the same project.

---

## 7. Versioning & compatibility

- Hero (Go) always advertises `schema_version` on every envelope it
  emits and on every SSE event payload.
- A consumer (hero-code) MUST tolerate **unknown additive fields**
  without erroring.
- A consumer that needs a field added in `1.x` MUST gate on the
  presence of the field, not the schema version.
- A breaking change (renamed/removed field, semantics change) bumps
  to `2.0`. The daemon will then emit both `1.0` and `2.0` envelopes
  on the SSE stream for a deprecation window.
