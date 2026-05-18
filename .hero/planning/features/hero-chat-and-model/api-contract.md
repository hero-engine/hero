# Hero Chat — API Contract (v1)

Wire contract between the hero serve Go core and the `command-bar.js` UI
island. All endpoints are mounted under `/api/chat/` on the hero serve
HTTP listener. User identity is resolved server-side via
`session.UserID(req)` (cookie / header / OS user); the island does not
send a user id.

All request and response bodies are JSON. Timestamps are RFC 3339.
`conversation_id` is an opaque string minted by the server on the first
turn for a scope.

## Endpoints

### `GET /api/chat/capability`

Returns the current adapter capability snapshot. Used by the island to
decide whether to render the empty-state CTA and which adapter chip to
show.

Response: `200 OK`

```json
{
  "adapters": [
    {
      "id": "mcp-session-abc",
      "adapter": "hero-code",
      "version": "0.4.1",
      "kinds": ["interactive", "headless"],
      "connected_at": "2026-05-17T10:00:00Z",
      "last_seen":   "2026-05-17T10:05:12Z"
    }
  ],
  "interactive":    "mcp-session-abc",
  "headless":       "mcp-session-abc",
  "user_preferred": ""
}
```

When no adapter is connected, `adapters` is an empty array and both
`interactive` and `headless` are empty strings.

### `POST /api/chat/turn`

Submit a chat turn. Server creates the conversation if needed, persists
the user message, dispatches the turn (runner-free slash inline OR via
adapter), and returns the conversation id + the SSE topic the island
should subscribe to for streamed events.

Request:

```json
{
  "prompt":           "/ask what specs do we have",
  "conversation_id":  "",
  "page_scope":       "global",
  "context": {
    "page":      { "pack": "engineering", "home": "work", "view": "work-roadmap" },
    "artifact":  { "kind": "spec", "slug": "per-feature-smoke-coverage" },
    "selection": { "text": "...", "range": "spec.md:L42-L58" },
    "workspace": "/abs/path/to/repo"
  }
}
```

- `prompt` is required. Leading `/<word>` is parsed as a slash invocation.
- `conversation_id` empty → server creates a new conversation for
  `(user, page_scope)`.
- `page_scope` is the persistence key. Recommended values:
  - `"global"` for the ⌘K command bar
  - `"page:<pack>/<home>/<view>:<artifact_kind>:<artifact_slug>"` for
    per-page chat (empty trailing segments allowed)
- `context` is optional; passed through to the dispatched adapter.

Response: `200 OK`

```json
{
  "conversation_id": "01HXYZ...",
  "sse_topic":       "chat.01HXYZ..."
}
```

If the prompt is a runner-free slash (`/ask`, `/note`, `/scheduled`) the
slash handler is invoked synchronously inside hero serve and emits
events on the bus before this response returns. The island should
subscribe to the `sse_topic` _before_ POSTing the turn whenever it
wants to render incremental tokens for adapter-dispatched turns; for
runner-free slashes the events will already be in flight when the
response arrives, so the island should treat the bus subscription as
the source of truth and also accept that early events may arrive in
its connect-event backlog.

When the prompt is not a runner-free slash AND no interactive adapter
is connected, the server emits a single `chat.error` event on the
topic and returns 200 with the conversation id. The error payload
includes a CTA the island renders:

```json
{
  "type": "chat.error",
  "payload": {
    "code":    "no_adapter",
    "message": "No Hero adapter is connected. Install hero-code to run agent workflows.",
    "link":    "https://heroengine.ai/install/hero-code"
  }
}
```

### `GET /api/chat/history?scope=<page_scope>&limit=50`

Returns the most recent turns for the user in the given page scope,
oldest-first. Used by the island when mounting a chat surface to
rehydrate.

Response: `200 OK`

```json
{
  "scope": "global",
  "turns": [
    { "role": "user",      "content": "/ask what specs do we have" },
    { "role": "assistant", "content": "There are 17 specs..." }
  ]
}
```

`limit` defaults to 50 and is capped at 100.

### `POST /api/chat/preference`

Set the user's preferred interactive adapter type.

Request:

```json
{ "interactive_adapter": "hero-code" }
```

Empty string clears the preference. The preference is keyed on
`adapter` (e.g. `hero-code`, `claude-code-bridge`) — NOT on the
per-connection `id`, because adapter connections come and go.

Response: `204 No Content`

### `POST /api/chat/clear`

Clear a conversation (delete all messages; the conversation row stays
for FK simplicity).

Request:

```json
{ "conversation_id": "01HXYZ..." }
```

Response: `204 No Content`

## SSE event stream

The island subscribes to the existing `/api/events` SSE endpoint. Each
event published by chat carries an `EventType` of `chat.<sub>` where
`<sub>` is one of the canonical chat event types below, and a payload
that includes `conversation_id`.

The island MAY filter by `topic == "chat." + conversation_id` (the
server publishes with this prefix in the event `Type` field), or it
MAY consume all events and filter client-side on
`payload.conversation_id`.

| Event type           | Payload fields                                    |
| -------------------- | -------------------------------------------------- |
| `chat.token`         | `{ conversation_id, text }`                        |
| `chat.tool_call`     | `{ conversation_id, name, args }`                  |
| `chat.tool_result`   | `{ conversation_id, name, preview }`               |
| `chat.error`         | `{ conversation_id, code, message, link? }`        |
| `chat.cost`          | `{ conversation_id, usd, runner }`                 |
| `chat.done`          | `{ conversation_id, usd, outcome? }`               |

`outcome` is a free-form object the adapter populates (e.g.
`{ spec_slug, pr_url }`). Runner-free slashes use `outcome` to report
artifacts written (e.g. `{ "file": ".hero/knowledge/notes/abc.md" }`).

`chat.error.code` values currently emitted by the server:

- `no_adapter` — no interactive-capable adapter connected
- `slash_unknown` — `/foo` matches no registered slash
- `slash_failed` — runner-free slash handler returned an error
- adapter-defined codes are passed through verbatim

`chat.done` is always the last event for a turn. The island should
treat any later events with the same `conversation_id` as belonging to
the next turn.
