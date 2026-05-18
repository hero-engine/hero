---
title: Hero Chat — Dispatch to hero-code (Required), Optional IDE Adapter, ⌘K Command Bar
type: feature
status: completed
tags: [serve, surface, chat, dispatch, adapter, mcp, command-bar, hero-code, ide-bridge]
created: 2026-05-17
relations:
  - target: hero-surface-architecture
    kind: parent
  - target: hero-surface-shell
    kind: depends-on
  - target: hero-surface-deployment-and-rendering
    kind: depends-on
  - target: hero-now-home
    kind: relates-to
  - target: hero-agents-home
    kind: relates-to
  - target: hero-runner
    kind: relates-to
horizon: now
---

## Context

Every home in the surface assumes "chat works." The Now home has a
big "Tell Hero what to do next" input. Every page has a ⌘K bar in
the top nav. Each home spec defers to "the chat" without saying
how a user prompt gets routed to a model and a response back to
the page. That gap is the single point of failure for the entire
surface — every home consumes it.

**Hero serve does not run inference.** It is a dispatcher. Every
chat invocation routes to a **Hero adapter** — a process that can
listen for dispatch events, run the agent loop, and stream events
back. Hero adapters live in one of two places:

- **hero-code** — the canonical, always-available adapter. The
  sibling runner binary (Go today, Rust on its roadmap) holds API
  keys, executes agent loops, and exposes itself as a Hero adapter
  out of the box. Required for headless work (scheduled agents,
  automations) and the default for interactive chat. **"Requires
  hero-code" is an honest stance** — solo users install it
  alongside hero; teams run a shared endpoint.
- **In-IDE Hero adapter (optional, aspirational)** — a Hero
  plugin/skill/extension running inside an IDE harness (Claude
  Code, Cursor, Codex) that listens for dispatches and injects
  them into the harness's own agent loop. When this works, the
  user keeps their familiar IDE chat — hero serve just drives.
  Feasibility varies per harness:
  - **Claude Code** is the most promising because of its
    well-established skills and hooks system. Hero already
    registers skills here (the existing `mcp__hero__*` tools); an
    additional skill could subscribe to dispatch events.
  - **Cursor / Codex** are TBD. We ship the dispatch protocol;
    plugin viability is a per-harness conversation.

The previous draft treated the IDE as a peer backend and required
every harness to implement a `hero_chat` MCP tool. That was
over-engineered. **One abstraction (Hero adapter), one wire
protocol, multiple homes.** hero-code is the home that always
works.

Hero already has the substrate this spec needs:

- MCP server (`internal/serve/mcp*.go`) — already exposes Hero
  tools to clients. The dispatch protocol piggy-backs on the same
  MCP plumbing; adapters connect as MCP clients with an additional
  capability declaration.
- `internal/runner/` exists in this repo today as the seed of
  hero-code. This spec consumes hero-code as an external sibling;
  it does NOT extend the runner package.

The merge-with-hero-code question is out of scope. If it ever
happens (one install, two processes), the dispatcher boundary is
the same. This architecture is robust to that decision.

## Goal

Hero serve's chat input — anywhere it appears (Now's big input,
each page's contextual chat, the ⌘K command bar) — dispatches
prompts to a registered Hero adapter and streams responses back
into the page. The default and recommended adapter is hero-code.
If an in-IDE Hero adapter is also connected, the user can pick it
for interactive turns. If no adapter is connected, `/ask` and
`/note` still work locally; everything else shows a clear
"Install hero-code →" affordance.

When this lands:

- A user with hero-code installed and reachable → chat in any
  browser tab streams through hero-code; all slashes, scheduled
  agents, and automations work. This is the **complete, supported
  baseline**.
- A user with hero-code AND a working in-IDE Hero adapter (e.g.,
  Claude Code with the Hero dispatch skill) → can choose to route
  interactive chat into their IDE; scheduled work still uses
  hero-code. Backend chip in the chat UI shows which adapter is
  active.
- A user with neither → `/ask` and `/note` work; other slashes
  and the free-text chat show "Install hero-code to enable agent
  workflows →".
- A scheduled run fires when hero-code is unreachable → marked
  `deferred`, surfaced in the Agents home, fires when hero-code
  reconnects (per the per-automation `miss_policy`).

## Approach

### The Hero adapter abstraction

A Hero adapter is any process that:

1. Connects to hero serve via the dispatch protocol (extends MCP)
2. Declares its identity, kind (`hero-code` | `ide-bridge`), and
   capabilities (`interactive`, `headless`, both)
3. Listens for dispatch events from hero serve
4. Runs the requested agent work in its own loop
5. Streams `chat.*` events back to hero serve

hero-code is one adapter. An in-IDE Hero plugin is another. They
implement the same protocol. Hero serve doesn't care which one is
on the other end — it just dispatches.

### Adapter registration

The protocol extends MCP with a single new initialize-time
declaration:

```json
{
  "capabilities": {
    "hero_dispatch": {
      "kinds": ["interactive", "headless"],
      "adapter": "hero-code",
      "version": "0.4.1",
      "session_id": "..."
    }
  }
}
```

- `kinds` — which dispatch types this adapter can handle
- `adapter` — `hero-code`, `claude-code-bridge`, `cursor-bridge`,
  `codex-bridge`, etc. Free-form string; hero serve uses it for
  display only.
- `session_id` — adapter's session identifier, used so hero serve
  can correlate streamed events back to the originating dispatch.

A single hero serve instance can have multiple adapters connected
simultaneously (a user might run hero-code locally AND have a
Claude Code Hero bridge active). Hero serve picks one per
dispatch based on capability + user preference.

### Capability resolver

```go
type ChatCapability struct {
    Adapters      []AdapterInfo  // all connected adapters
    Interactive   string         // adapter id chosen for interactive; "" if none
    Headless      string         // adapter id chosen for headless; "" if none
    UserPreferred string         // optional override stored per-user
}

type AdapterInfo struct {
    ID           string    // unique connection id
    Adapter      string    // "hero-code", "claude-code-bridge", ...
    Version      string
    Kinds        []string  // ["interactive"], ["headless"], or both
    ConnectedAt  time.Time
    LastSeen     time.Time
}
```

Selection rules:

- **Interactive**: if user has a preference and that adapter is
  connected → use it. Else first connected adapter that supports
  `interactive`, prefer `hero-code` over IDE bridges (predictable
  default).
- **Headless**: first connected adapter that supports `headless`.
  In practice this is always hero-code; IDE bridges may or may
  not declare `headless` (they shouldn't — IDEs aren't on when a
  cron fires).
- **`/ask` and `/note`**: never dispatch; execute inside hero
  serve.

The `/api/chat/capability` endpoint returns the current state for
the UI.

### Dispatch wire contract

When hero serve dispatches:

```json
{
  "kind": "interactive",
  "conversation_id": "...",
  "prompt": "...",
  "context": {
    "page":     { "pack": "engineering", "home": "work", "view": "work-roadmap" },
    "artifact": { "kind": "spec", "slug": "per-feature-smoke-coverage" },
    "selection":{ "text": "...", "range": "spec.md:L42-L58" },
    "workspace": "/abs/path/to/repo"
  },
  "history": [ ... ],
  "slash":   { "name": "design", "args": "..." }
}
```

The adapter responds with a stream of events on the same MCP
session:

| Event | When |
|---|---|
| `chat.token` | incremental text chunk |
| `chat.tool_call` | adapter agent is calling a Hero (or other) tool |
| `chat.tool_result` | tool returned; brief preview |
| `chat.error` | failure, rate-limit, budget, or adapter-internal error |
| `chat.cost` | running cost ticker (optional, per-turn or final) |
| `chat.done` | turn complete; payload includes total cost and outcome ref (PR url, spec slug, etc.) |

Hero serve normalizes these into SSE events the chat UI consumes.
The UI is adapter-agnostic.

### What hero-code is

hero-code is a sibling runner. Its full spec lives in that repo.
For this spec's purposes, hero-code:

- Connects to hero serve as a Hero adapter (`adapter: hero-code`,
  `kinds: ["interactive", "headless"]`)
- Holds provider credentials (Anthropic, OpenAI, Azure, …) — never
  hero serve
- Runs the same agent workflows (`/design`, `/diagnose`,
  `/deliver`, etc.) the Hero CLI today implements via
  `internal/runner/`
- Implements the credential broker (per-user budgets, shared org
  keys) that previously lived in `hero-team-server`. The broker is
  hero-code's concern, not hero serve's.
- Can run locally (solo) or as a service (team / cloud)
- Reachable via a configured endpoint:
  ```json
  { "chat": { "headless": { "endpoint": "unix:///run/hero-code.sock" } } }
  ```
  When omitted, hero serve probes a sensible default (e.g.,
  `~/.hero/hero-code.sock` for a local install).

### What an in-IDE Hero adapter is (optional)

A small piece of code running inside an IDE that:

- Subscribes to dispatch events from hero serve (over a local
  socket, websocket, or MCP server-initiated request)
- Translates a dispatch into a synthetic prompt for the IDE's
  agent loop — as if the user had typed it
- Streams the IDE's response back to hero serve as `chat.*`
  events

**Claude Code is the realistic v1 target** because:
- Hero already integrates with Claude Code as an MCP server
  (today the user has `mcp__hero__*` tools and the `claude-api`
  skill available)
- Claude Code's skills and hooks systems can subscribe to
  external events and dispatch prompts into the active session
- A `hero-dispatch-listener` skill could be the entire integration

For Cursor and Codex, the spec ships the wire protocol and we
treat plugin authoring as a separate, per-harness effort.

### Routing examples

| Source | Connected adapters | Routes to |
|---|---|---|
| Interactive chat | hero-code only | hero-code |
| Interactive chat | hero-code + claude-code-bridge | hero-code (default), claude-code-bridge if user preferred |
| Interactive chat | claude-code-bridge only | claude-code-bridge |
| Interactive chat | none | `/ask` and `/note` work; everything else CTA |
| Scheduled run | hero-code reachable | hero-code |
| Scheduled run | hero-code unreachable | apply `miss_policy` (deferred-queue / skip / require) |
| Automation trigger | hero-code reachable | hero-code |
| Automation trigger | hero-code unreachable | apply per-automation `miss_policy` |

IDE bridges never receive headless dispatches by design — the IDE
may not be running when a cron fires. If a Claude Code bridge
declares `kinds: ["interactive", "headless"]`, hero serve still
prefers hero-code for headless.

### Deferred / queued scheduled work

When a scheduled agent or automation fires and no adapter
supporting `headless` is reachable, the run enters one of three
states per the automation's `miss_policy`:

| Policy | Behavior |
|---|---|
| `queue` (default) | Recorded as `deferred`. Fires when hero-code reconnects, in original order. Surfaced in Agents → Scheduled with a one-click "Run now." |
| `skip` | Logged as `missed`; next occurrence proceeds normally. |
| `require` | Marked `missed` AND a notification fires (Slack, email, or Hero feed) so a human can intervene. |

Time-sensitive automations can carry an `sla`:

```yaml
trigger: { type: tracker, event: issue_created, filter: { priority: Critical } }
action:  { command: diagnose, ... }
miss_policy: require
sla: "15m"
```

The Agents home Scheduled view shows per-run status (`next`,
`running`, `deferred`, `missed`, `done`) and exposes policy
controls inline.

### Slash commands inside chat

| Slash | Action | Needs an adapter? |
|---|---|---|
| `/design <thing>` | Design workflow | yes |
| `/diagnose <bug>` | Diagnose workflow | yes |
| `/deliver <slug>` | Deliver workflow against a spec | yes |
| `/ask <query>` | Read-only Q&A over indexed corpus | **no** |
| `/note <text>` | Capture a note to `.hero/knowledge/` | **no** |
| `/scheduled "..."` | Convert input into a scheduled-agent definition | no (writes a YAML file) |

`/ask`, `/note`, and `/scheduled` are runner-free — they execute
inside hero serve directly. This is part of why the empty-state
chat isn't fully disabled: those three slashes remain available
when no adapter is connected.

When the user types `/` at the start of input, a palette renders
inline showing available slashes filtered by subsequent
keystrokes — same UX pattern as Linear / GitHub / Notion.

### The ⌘K command bar

⌘K (or `Ctrl+K`) opens a centered overlay above the current page.
The overlay is **one input with three progressive modes**:

1. **Search** (default on open) — types map to unified search
   results (specs, knowledge, people, sessions, code symbols). No
   adapter required.
2. **Chat** — activates when the user presses `?` at the start,
   hits `Enter` on a query that doesn't match a result, or
   presses `→` to "ask Hero about your query."
3. **Slash** — activates when the user types `/` at the start;
   palette opens.

The bar carries the current page + active artifact as context
automatically. A small chip below the input shows what's
attached: `Context: Spec · per-feature-smoke-coverage` with an ✕
to remove.

The overlay is the **single chat affordance** across the surface.
The big "Tell Hero what to do next" input on Now is a teaching
surface — visually identical, same backend, but always visible.
Both share state.

### Streaming, persistence, cost

- **Streaming** — adapter `chat.*` events are normalized and
  republished on the existing SSE bus. The chat UI subscribes to
  `/api/events?topic=chat.<conversation_id>`. Heartbeat every 15s;
  reconnect resumes from last event id.
- **Persistence** — per-page chat threads persist in SQLite as
  `chat_messages` keyed by `(user, workspace, page_id,
  artifact_id)`. The ⌘K chat has a separate top-level thread per
  workspace. Conversations hold ~50 turns hot; older turns roll
  off to a `chat_archive` table.
- **Cost** — hero serve does NOT bill, ever. Adapters report cost
  in `chat.cost` and `chat.done` events. The chat UI shows a
  ticker beneath the input: `$0.04 this conversation · $0.42
  today via hero-code`. Budgets live on the adapter — if hero-code
  declines a turn due to broker policy, it emits `chat.error`
  with a budget reason, and the UI shows a "Manage budget in
  hero-code →" link.

### Setup empty state

When no adapter is connected:

```
Hero needs hero-code (or a Hero IDE adapter) to run agent work.
/ask and /note still work right here.

[ Install hero-code (recommended) ]      [ Already running it elsewhere → ]

Using Claude Code, Cursor, or Codex with the Hero IDE adapter?
Make sure it's running and connected — it'll appear here automatically.
```

`Install hero-code` opens a modal with the install snippet
(`brew install hero-engine/tap/hero-code` or curl install) plus a
note explaining what it does and the default endpoint hero serve
expects.

### Configuration

`hero.json` gets a new top-level `chat` block, all optional:

```json
{
  "chat": {
    "preferred_interactive": "hero-code",
    "headless": {
      "endpoint": "unix:///run/hero-code.sock",
      "fallback_endpoint": "https://hero-code.team.internal"
    }
  }
}
```

- `preferred_interactive` — adapter id (e.g., `hero-code`,
  `claude-code-bridge`). Defaults to `hero-code` if multiple
  adapters are present.
- `chat.headless.endpoint` — where hero-code listens. Omitted →
  hero serve probes `~/.hero/hero-code.sock` first, then the
  installer's default.

No API keys, no model names, no provider strings. Those live in
hero-code.

### Edition variations

- **local**: solo developer. hero-code installed locally OR a
  remote endpoint configured. Optional IDE adapter for in-IDE
  chat continuity.
- **team**: team server. Each user runs their own IDE adapter (if
  any). The team runs a shared hero-code endpoint. Auth token
  attributes spend to the calling user via the broker.
- **cloud**: hosted hero-code endpoint. SSO identity passed
  through.
- **enterprise**: + customer-held keys inside the customer's own
  hero-code deployment; + signed audit chain entries for every
  dispatch and adapter-reported tool call.

The "shared credential broker" that previously lived in
`hero-team-server` becomes **hero-code's responsibility**. Hero
serve never touches credentials. The Agents home's Credentials
view becomes a window into hero-code's broker state.

## Changes

**Delivered in the Go-core sprint (this commit):**

- `internal/serve/chat/adapter.go` — `HeroAdapter` interface, `Kind`, `AdapterInfo`
- `internal/serve/chat/registry.go` — connected-adapter registry with `ByKind` / `PreferHeroCode`
- `internal/serve/chat/capability.go` — capability resolver (user pref + hero-code tiebreak; IDE bridges blocked from headless)
- `internal/serve/chat/protocol.go` — `DispatchRequest`, `Event`, event constructors
- `internal/serve/chat/stream.go` — `Streamer` republishes events on the bus as `chat.<sub>` with per-conversation topic
- `internal/serve/chat/slash.go` — slash registry + `ParseSlash`, default slashes
- `internal/serve/chat/slash_ask.go` — runner-free `/ask` over the retrieval layer
- `internal/serve/chat/slash_note.go` — runner-free `/note` writes `.hero/knowledge/notes/<slug>.md`
- `internal/serve/chat/slash_scheduled.go` — runner-free `/scheduled` writes `.hero/automations/<slug>.yaml`
- `internal/serve/chat/env.go` — `resolveHeroDir` helper
- `internal/serve/chat/persistence.go` + `schema.sql` — SQLite-backed conversation + message + preference store
- `internal/serve/chat/herocode.go` — `HeroCodeAdapter` probe (client side only — server side lives in hero-code)
- `internal/serve/chat/api.go` — `/api/chat/capability`, `/api/chat/turn`, `/api/chat/history`, `/api/chat/preference`, `/api/chat/clear` handlers + dispatcher
- `internal/serve/chat/*_test.go` — registry / capability / slash / persistence / stream / api tests + the import-boundary AST guard
- `internal/serve/mcp_protocol.go` — `HeroDispatchCapability` added to `MCPCapabilities`
- `internal/serve/mcp.go` — `MCPServer.SetChatRegistry` + `chatRegistry` field
- `internal/serve/mcp_lifecycle.go` — `tryRegisterDispatchClient` parses `hero_dispatch` on initialize; `mcpClientAdapter` stub
- `internal/serve/server.go` — `initChat` constructs the registry/store/api; `busAdapter` bridges chat events onto the serve event bus; chat routes mounted under `/api/chat/*` in the top mux
- `internal/config/config.go` — `ChatConfig` and `ChatHeadlessConfig`
- `.hero/planning/features/hero-chat-and-model/api-contract.md` — wire contract for the JS island engineer

**Still to land (per the spec; deferred to follow-up sprints):**

1. **Chat dispatcher** — `internal/serve/chat/dispatch.go`,
   `chat/adapter.go`, `chat/capability.go`. Defines the
   `HeroAdapter` interface; resolves capability; routes dispatches.
   No inference code.
2. **Adapter registry** — `internal/serve/chat/registry.go`.
   Tracks all connected adapters keyed by session id; updates on
   MCP connect/disconnect events.
3. **Dispatch protocol** — `internal/serve/chat/protocol.go`.
   Wire format types (request envelope, event stream); MCP
   capability declaration parsing.
4. **MCP handshake extension** — `internal/serve/mcp_protocol.go`.
   Parse `capabilities.hero_dispatch` on initialize; expose
   adapters to the registry. Existing handshake unchanged for
   clients that don't declare it.
5. **SSE bridge** — `internal/serve/chat/stream.go`. Normalizes
   adapter events into the published `chat.*` SSE event types.
6. **Slash command registry** — `internal/serve/chat/slash.go`.
   Static map of slashes; marks which execute inside hero serve
   (`/ask`, `/note`, `/scheduled`) and which dispatch. Slash
   `Register()` is the pack extension point.
7. **Runner-free slash implementations** —
   - `chat/slash_ask.go` — wraps the existing `hero ask`
     retrieval pipeline (no inference).
   - `chat/slash_note.go` — writes a knowledge entry to
     `.hero/knowledge/`.
   - `chat/slash_scheduled.go` — writes a YAML automation
     definition under `.hero/automations/` for the user to
     review.
8. **Deferred-run store** — `internal/serve/chat/deferred.go`
   + SQLite tables `deferred_runs`, `missed_runs`. The Agents
   Scheduled view consumes these via existing API.
9. **API endpoints** — `internal/serve/api_chat.go`:
   - `GET /api/chat/capability` — current adapter state
   - `POST /api/chat/turn` — submit a turn (returns conversation
     id + SSE topic)
   - `GET /api/chat/history?page=…&artifact=…` — page-scoped
     history
   - `POST /api/chat/preference` — set preferred adapter
   - `POST /api/chat/clear` — clear a conversation
   - `POST /api/chat/deferred/:id/run` — kick a deferred run
10. **⌘K overlay** — `internal/serve/islands/command-bar.js`.
    Only chat-related island; everything else is template + SSE.
11. **Inline chat input fragment** —
    `internal/serve/templates/chat-input.html`. Shared by Now's
    big input, per-page contextual chat, and ⌘K.
12. **Conversation persistence** — SQLite migrations adding
    `chat_messages`, `chat_conversations`, `chat_archive`.
13. **Settings page** — `/settings/chat` view. Form for editing
    `chat.headless.endpoint` and `preferred_interactive`. Read-
    only display of currently connected adapters with last-seen
    timestamps. NO model selection, NO API key entry.
14. **Empty-state notice fragment** — shown above the input when
    no adapter is connected. Install snippet and IDE-adapter
    note.
15. **Slash palette** — inline UI hydrated from
    `command-bar.js`; filters as the user types.
16. **Now home wire-up** — Now's view template uses the shared
    chat-input fragment.
17. **Cost ticker fragment** — shared between inline chat and
    ⌘K; subscribes to `chat.cost` events.
18. **Audit log hook (enterprise)** — `chat/audit.go`, compile-
    time gated. Writes every dispatch + adapter-reported tool
    call to the signed audit chain.
19. **Claude Code Hero-dispatch skill (separate work, separate
    repo)** — published alongside this spec but owned by the
    Claude Code Hero integration. Documents the dispatch
    protocol from the adapter side; serves as the reference
    implementation of an IDE bridge.

## Boundaries

- **Not in scope:** running inference inside hero serve. No API
  keys, no provider SDKs, no token accounting. Build-time check:
  `internal/serve/chat/*` cannot import `internal/runner/*`.
- **Not in scope:** hero-code's internal design (provider
  abstraction, credential broker, sandbox). This spec consumes
  hero-code as a sibling via the adapter protocol.
- **Not in scope:** the Cursor or Codex IDE adapters. The
  dispatch protocol enables them; building them is per-harness
  work tracked separately.
- **Not in scope:** the workflows themselves (`/design`,
  `/deliver`, `/diagnose`) — they exist; adapters execute them.
- **Not in scope:** merging hero with hero-code. Separate
  decision. This architecture works either way.
- **Not in scope:** chat for PM / QA packs — they consume this
  surface and may register additional slashes; the dispatch
  logic is identical.
- **Not in scope:** voice or multimodal input.
- **Not in scope:** cross-user shared chat. Team coordination
  happens through People feed + handoffs, not chat threads.

## Acceptance Criteria

- WHEN hero-code is connected as an adapter THE SYSTEM SHALL
  expose it as the default interactive backend AND the default
  headless backend.
- WHEN no adapter is connected THE SYSTEM SHALL render the
  empty-state notice above the chat input AND keep `/ask`,
  `/note`, and `/scheduled` slashes functional.
- WHEN multiple interactive-capable adapters are connected AND no
  user preference is set THE SYSTEM SHALL default to
  `adapter: hero-code`.
- WHEN the user submits a chat turn THE SYSTEM SHALL dispatch it
  to the selected adapter and stream `chat.*` events back via
  SSE.
- WHEN the user types `/` at the start of the input THE SYSTEM
  SHALL render the slash palette filtered by subsequent
  keystrokes.
- WHEN a slash resolves to `/ask`, `/note`, or `/scheduled` THE
  SYSTEM SHALL execute it inside hero serve without dispatching
  to any adapter.
- WHEN any other slash is selected THE SYSTEM SHALL dispatch to
  the active interactive adapter with the rest of the input as
  arguments.
- WHEN the user presses ⌘K on any page THE SYSTEM SHALL open the
  command bar overlay centered above the current view.
- WHEN ⌘K opens on a page with an active artifact THE SYSTEM
  SHALL attach the artifact as context and display it as a
  removable chip.
- WHEN a scheduled agent or automation fires AND hero-code is
  reachable THE SYSTEM SHALL dispatch the run to hero-code.
- WHEN a scheduled run fires AND no headless-capable adapter is
  reachable AND `miss_policy` is `queue` THE SYSTEM SHALL record
  the run as `deferred` and surface it in the Agents Scheduled
  view.
- WHEN a deferred run exists AND any headless-capable adapter
  connects THE SYSTEM SHALL fire deferred runs in order.
- WHEN a scheduled run's `miss_policy` is `require` AND its
  `sla` passes without firing THE SYSTEM SHALL emit a
  `chat.error` and a feed notification.
- WHILE a turn is in flight THE SYSTEM SHALL show a cancel
  affordance that aborts the adapter stream cleanly.
- IF the adapter emits `chat.error` with a budget-exceeded
  reason THEN THE SYSTEM SHALL render the error inline with a
  "Manage budget in <adapter> →" link.
- WHERE the edition is `enterprise` THE SYSTEM SHALL append
  every dispatch and reported tool call to the signed audit
  chain.
- THE SYSTEM SHALL persist conversation history keyed by user,
  workspace, page, and artifact, and SHALL load page-scoped
  history on chat surface mount.
- THE SYSTEM SHALL emit identical `chat.*` SSE event types
  regardless of which adapter served the turn.
- THE SYSTEM SHALL NOT load any provider SDK, API key
  configuration, or inference code path.

## Risks

- **hero-code dependency for first-run.** A user installing hero
  alone gets a half-functional chat. Mitigation: `hero install`
  proactively offers to install hero-code; the empty-state CTA is
  direct and welcoming; `/ask` and `/note` remain useful.
- **IDE adapter feasibility uncertain.** Claude Code's skills/
  hooks system is the most promising path but unproven for this
  specific use case (event-driven prompt injection into the
  active session). Mitigation: ship the dispatch protocol and the
  hero-code adapter first; treat the Claude Code bridge as a
  spike with a fallback plan (use hero-code, screenshot the
  result back).
- **Multiple adapters connected with overlapping kinds.**
  Selection rules must be deterministic and visible to the user.
  Mitigation: rules above, plus the backend chip in the chat UI
  is always visible.
- **Streaming reliability.** SSE over long-lived connections can
  drop on flaky networks. Mitigation: 15s heartbeat; resume from
  last event id; clear connection-status indicator.
- **Cost transparency split.** Costs come from adapters; hero
  serve aggregates without enforcing. Risk that a user thinks
  hero serve is enforcing a budget that's actually adapter
  policy. Mitigation: ticker always names the adapter; budget-
  exceeded link routes to the adapter's controls.
- **Context bloat.** Attaching full page state on every turn
  could blow context budgets. Mitigation: 4KB envelope cap; large
  artifacts are summarized; adapter decides whether to fetch
  more.
- **Slash collisions.** PM and QA packs will want their own
  slashes. Mitigation: namespacing (`/pm:design-story`); pack
  registration validated at startup.

## Validation

- Manual: with no adapter connected, verify empty-state notice
  renders AND `/ask` + `/note` work end-to-end.
- Manual: install hero-code locally; verify it auto-registers as
  an adapter; submit a chat turn; verify streaming.
- Manual: fire a scheduled run; verify it dispatches to
  hero-code; verify the Agents Scheduled view updates.
- Manual: stop hero-code; fire a scheduled run; verify it lands
  in `deferred`; restart hero-code; verify deferred run fires.
- Manual: connect a Claude Code Hero-dispatch bridge (when
  available); verify it appears as a second adapter; switch
  preference; verify interactive turns route to the bridge while
  scheduled work stays on hero-code.
- Manual: press ⌘K on each of the 5 homes; verify overlay opens
  with current page context attached.
- Manual: type `/de` in chat input; verify palette shows
  `/design`, `/deliver`, `/diagnose`; arrow keys select.
- Test: capability resolver — table-driven cases for various
  adapter combos + preferences.
- Test: dispatcher routing — given a request and a capability,
  the right adapter is invoked.
- Test: SSE event normalization — events from any adapter
  produce byte-identical `chat.*` events.
- Test: `/ask`, `/note`, `/scheduled` execute without an adapter
  available.
- Test: deferred-run lifecycle — fire with no adapter; reconnect;
  verify execution; verify `skip` and `require` policies.
- Test: NO import of `internal/runner/*` from
  `internal/serve/chat/*` — enforced as a build-time check.
- Visual: ⌘K overlay matches the locked web-app grammar.

## Kickoff

**Status: dispatcher + UI overlay delivered 2026-05-17.** The chat
surface is live in hero serve. Every home page can now embed the
chat-input fragment and the ⌘K overlay is bound on every route.

**What works today:**
- `GET /api/chat/capability` — returns connected adapters + chosen
  interactive/headless (currently `interactive: ""` since hero-code's
  server side isn't wired yet)
- `POST /api/chat/turn` — accepts a prompt + context, dispatches
  runner-free slashes inline (`/ask`, `/note`, `/scheduled`), routes
  adapter-required prompts to the selected adapter (emits
  `chat.error` with install CTA when no adapter)
- `GET /api/chat/history` and `POST /api/chat/clear` — page-scoped
  conversation persistence (SQLite at `~/.hero/chat.db`)
- `POST /api/chat/preference` — per-user adapter preference
- ⌘K overlay (search / chat / slash modes + empty-state CTA + adapter
  chip + cost ticker), 1713-line vanilla JS island at
  `internal/serve/shell/static/islands/command-bar.js`
- MCP `hero_dispatch` capability declaration on initialize; clients
  that declare it auto-register as adapters
- API contract published at
  [api-contract.md](api-contract.md)

**Pick up at: drive home specs.** Each home spec
([hero-now-home](../hero-now-home/spec.md),
[hero-work-home](../hero-work-home/spec.md),
[hero-knowledge-home](../hero-knowledge-home/spec.md),
[hero-agents-home](../hero-agents-home/spec.md),
[hero-people-and-roi-home](../hero-people-and-roi-home/spec.md))
can now replace its shell stub with real content — chat is
available, ⌘K works, slash dispatch works.

**Follow-up specs to file separately:**

- **hero-code adapter wire-up** — the client side
  (`HeroCodeAdapter.Stream`) is a stub awaiting hero-code's
  server-side adapter contract. Per the cross-repo peering call, this
  is a joint design + build with the hero-code repo.
- **MCP-client dispatch back-channel** — when a harness MCP client
  declares `hero_dispatch` capability on initialize, the server
  registers it but `Stream` currently no-ops with
  `adapter_not_wired`. The path that calls back into the client's
  `hero_chat` tool needs the reverse-direction MCP request flow.
- **Streamed-token persistence** — today the assistant turn is
  persisted as an outcome marker; the spec calls for token-by-token
  concatenation into the message store. Small follow-up.
- **Deferred-run queue + `miss_policy`** — surfaced in Agents home's
  Scheduled view; the queue itself is a follow-up.
- **`/settings/chat` page** — adapter preference + headless endpoint
  config form. Deferred to a tiny follow-up spec.
- **Cost ticker `today_cost` field** — capability response should
  surface today's total spend; the island reads it opportunistically.
  Trivial follow-up.
- **Conversation history write-side cap** — today the read returns
  last 50 turns of any-size conversation; write-side cap at 100 is a
  small tightening.
- **Enterprise audit-log hook** — compile-time gated; separate sprint.
- **Claude Code Hero-dispatch bridge spike** — separate spec, separate
  repo, owned by Claude Code's Hero integration. Optional v2 path.

**"Requires hero-code" is the honest baseline.** The dispatch
protocol is general enough to admit IDE adapters; we ship the
hero-code adapter first because it always works. An IDE adapter is
a wonderful enhancement when feasible, not a P0 dependency.

## Handoff Trail

- 2026-05-18T00:26:05Z — out → hero-code (peer_id: ad027c2f-7f74-4a09-bf1d-6515cc906074)
  mode: advisory
  originating_spec: hero-chat-and-model
  at_commit: 2952923
  result_ref: 18b08110963ad1f82e1cdb89046be394
  reason: "kickoff readiness check for hero-surface-architecture"

