# Web UI Homes

`hero serve` boots a local HTTP daemon (default
`http://localhost:7437`) that exposes a small set of top-level **homes**
in the browser. Each home is a route owned by a package under
`internal/serve/pages/<name>/` and registered on the shared shell
router. Open the daemon and the top nav lists exactly the routes below.

The homes:

- [`/now`](#now-now) — what's happening right now
- [`/rollup`](#rollup-rollup) — projected shape of the project
- [`/work`](#work-work) — the spec corpus
- [`/knowledge`](#knowledge-knowledge) — the knowledge base
- [`/agents`](#agents-agents) — agent sessions and automation
- [`/people`](#people-people) — operator pulse and handoffs

Plus two pieces of global chrome that don't live at a route of their
own — the ⌘K command overlay and the inline chat dispatcher.

## Now — `/now`

The homepage. `/now` is the operator's primary entry point: what's
in flight, what was just touched, what's waiting on you, and the chat
input you'd usually reach for first. Registered in
`internal/serve/pages/now/page.go` (`Slug: "now"`).

## Rollup — `/rollup`

A projected snapshot of the project's shape — surfaces, the spec
table, initiatives, and the archives timeline. Aimed at leads and
anyone trying to see the whole thing at a glance rather than the
current sprint. Landed in v0.10 alongside the `hero snapshot`
projector. Registered in `internal/serve/pages/rollup/page.go`
(`Slug: "rollup"`).

!!! note "Previously `/project`"
    This home was originally mounted at `/project` and moved to `/rollup`.
    The `/project` slot is now owned by the per-project **section** page
    (`internal/serve/projectpage`, `Slug: "project"`) — an operator view
    of a single registered project rather than the whole-workspace rollup.

## Work — `/work`

The spec corpus view. Engineers come here to triage what's in the
sprint, see throughput and quality views, and drill into individual
specs via `/work/spec/{slug}`. Registered in
`internal/serve/pages/work/page.go` (`Slug: "work"`).

## Knowledge — `/knowledge`

The knowledge base — conventions, decisions, context, and ingested
notes. The home of "what does this project already know?" Detail
routes like `/knowledge/{slug}`, `/knowledge/why`, and
`/knowledge/search` exist but aren't enumerated here. Registered in
`internal/serve/pages/knowledge/page.go` (`Slug: "knowledge"`).

## Agents — `/agents`

Live agent sessions, proposals waiting on a human, scheduled tasks,
and automation wiring. For operators who need to see what the AI
agents are doing and what they want them to do next. Registered in
`internal/serve/pages/agentspage/page.go` (`Slug: "agents"`).

## People — `/people`

Operator pulse, profiles, handoffs, and ROI overview. For leads who
care about who's on what, what's been handed off, and where the
team's time is going. Registered in
`internal/serve/pages/people/page.go` (`Slug: "people"`).

## Global chrome

Two pieces of UI ride along on every home rather than living at a
route of their own. Both are still settling — treat them as present
and useful, not yet contract.

- **⌘K command overlay.** A global command palette island registered
  in `internal/serve/shell/static/islands/command-bar.js`. Press
  ⌘K (or Ctrl+K) from any home to open it.
- **Chat dispatcher.** Inline chat input mounted on most homes,
  backed by handlers under `/api/chat/*` (registered in
  `internal/serve/chat/api.go`). Behavior and endpoint shape are
  in flux.

## Still settling

Detailed UI walkthroughs land once the homes settle past polish-pass.
The inventory above names what's there; how each home behaves in
depth is the subject of a follow-up spec.
