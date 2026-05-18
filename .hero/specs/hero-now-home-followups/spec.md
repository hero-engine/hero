---
title: Hero Now Home — Polish Follow-ups
type: feature
status: completed
tags: [serve, surface, now, home, polish, web-app]
created: 2026-05-17
relations:
  - target: hero-now-home
    kind: parent
  - target: hero-surface-shell
    kind: relates-to
  - target: hero-chat-and-model
    kind: relates-to
  - target: hero-agents-home
    kind: relates-to
horizon: now
---

## Context

The [hero-now-home](../../../specs/hero-now-home/spec.md) delivery
shipped four loose ends that the Now spec's kickoff explicitly named
for follow-up after the rest of the home fan-out landed. The fan-out
is now complete (`hero-work-home`, `hero-knowledge-home`,
`hero-agents-home`, `hero-people-and-roi-home` all in
[.hero/specs/](../../../specs/)), and the Agents home delivery
introduced the live session ledger that Now was waiting on.

This spec closes those four loose ends in a single coordinated pass.

## Goal

After delivery:

1. **Quick launch input uses the shell's `chat-input` fragment.**
   No hand-rolled markup. The ⌘K island can hydrate the input on
   Now exactly the same way it hydrates the overlay.
2. **Page-hero subhead refreshes via SSE** when its component counts
   (inbox count, running agent count, last-active timestamp) change.
   The shell emits the `event: hero` channel; Now's page wires a
   small client-side handler to swap the subhead's text in place.
3. **Quick launch shows the empty-state notice** above the input
   when no chat adapter is connected. The shell's
   `empty-state-notice` fragment lives directly above the
   `chat-input` fragment, conditionally rendered based on the
   capability resolver.
4. **Currently-running session block populates** from the shared
   session ledger that [hero-agents-home] introduced
   (`internal/serve/pages/agentspage/data/sessions.go::LoadSessions`).
   Now's `agents.go` consumes the ledger instead of returning
   `Running == nil`.

## Approach

### 1. Quick launch → shell chat-input fragment

Today `internal/serve/pages/now/templates/page.html` renders its own
`<input>` for Quick launch. Replace with:

```html
{{ template "chat-input" (chatInputArgs .) }}
```

where `chatInputArgs` is a small helper on the Now page handler that
builds a `shell.ChatInput` struct with:

- `Variant: "hero"` — the largest, 64px-tall variant
- `Placeholder: "Tell Hero what to do next…"`
- Page context attached (so the input dispatches with the current
  page + artifact attached, matching the chat-and-model contract)

The intent chips below (`/design` / `/diagnose` / `/deliver` /
`/review` / `/ask`) stay in Now's template — they're page content,
not chrome. The "Try:" recent-prompts line stays too.

The ⌘K island already knows how to hydrate elements with
`data-chat-input-variant` (per the chat-and-model JS island). Now's
hand-rolled input misses that hook today; the fragment swap fixes it.

### 2. Page-hero subhead SSE refresh

Add `event: hero` to the `/api/now/events` SSE channel in
`internal/serve/api/now.go`. The payload is the new subhead text
(e.g., `"2 need your input · 1 agent running · since Fri 4:12pm"`).

Two triggers republish on `event: hero`:

- Any `proposal.*` or `handoff.*` event that changes inbox count
- Any `session.*` event that changes running-agent count

Now's page template wraps the subhead in
`<span data-now-subhead>{{ .PageHero.Subhead }}</span>`. The
client-side SSE subscriber adds one branch:

```js
es.addEventListener('hero', (e) => {
  const el = document.querySelector('[data-now-subhead]');
  if (el) el.textContent = e.data;
});
```

Debounce remains 250ms per event-type, same pattern as the existing
section refreshes.

### 3. No-adapter empty-state notice

Above the `chat-input` fragment in Now's Quick launch section,
conditionally render the `empty-state-notice` fragment when the chat
capability resolves to no interactive adapter:

```html
{{ if .NoAdapter }}
  {{ template "empty-state-notice" .EmptyState }}
{{ end }}
{{ template "chat-input" (chatInputArgs .) }}
```

`NoAdapter` is computed on each render by calling
`chat.Resolve(...).Interactive == ""` from the Now route handler.
`EmptyState` carries:

- `Headline: "Hero needs hero-code (or a Hero IDE adapter) to run agent work."`
- `Body: "/ask and /note still work right here."`
- `PrimaryAction: { Label: "Install hero-code", Href: "https://heroengine.ai/install/hero-code" }`
- `GhostAction: { Label: "Already running it elsewhere →", Href: "/settings/chat" }`
- `FootNote: "Using Claude Code, Cursor, or Codex with the Hero IDE adapter? Make sure it's running and connected."`

When `NoAdapter` flips (adapter connects / disconnects), the page
hero SSE channel (from #2) doesn't suffice — capability change is
its own signal. Add an additional `event: capability` to the SSE
channel; on receipt, the client re-fetches the entire Quick launch
section fragment from a new `GET /api/now/quicklaunch` endpoint.

(Alternative: just re-fetch on page reload. Acceptable for v1 since
adapter connect/disconnect is rare — pick whichever the engineer
finds cleaner during delivery; document the choice.)

### 4. Currently-running session block via Agents ledger

Today `internal/serve/pages/now/data/agents.go::LoadAgents` always
returns `Running: nil` with `RunningCount: 0`, with a comment
explaining the live session store isn't wired.

The Agents home delivery introduced
`internal/serve/pages/agentspage/data/sessions.go::LoadSessions(in
SessionsInputs) Sessions` which surfaces a `[]SessionRow` of live
sessions. Now should consume this through a small shared abstraction:

**Option A (preferred):** Extract the session-listing logic out of
`agentspage/data/sessions.go` into a new neutral package
`internal/serve/sessions/` that both `agentspage` and `now` import.
Both pages render the same source of truth differently.

**Option B (cheaper):** Now imports `agentspage/data` directly and
calls `LoadSessions`, filtering to "Mine" + running-only.

Pick Option A if the extraction is one-file-clean (1–2 hours);
otherwise B with a TODO to refactor later. Document the choice in
the kickoff section.

Either way, Now's `LoadAgents` becomes:

```go
sessions := sessions.LoadLive(sessions.Inputs{
    UserID:  in.UserID,
    Scope:   "mine",
    Limit:   1, // primary card shows one
})
if len(sessions) > 0 {
    out.Running = &Running{
        Agent: sessions[0].Agent,
        Spec:  sessions[0].Spec,
        // ...
    }
    out.RunningCount = len(sessions)
}
```

Empty case (no running session) renders the existing empty state.

## Changes

Picked **Option B** for the shared session ledger — `now/data/types.go`
re-exports `agentspage/data.SessionRow` via a type alias so Now consumes
the same shape Agents writes, without churning a third package. A TODO
points to lifting it into `internal/serve/sessions/` if a third
consumer appears.

Files modified / created:

1. `internal/serve/pages/now/templates/page.html` — slimmed to template
   composition; Quick launch moved into a dedicated `quicklaunch.html`.
2. `internal/serve/pages/now/templates/quicklaunch.html` (new) —
   renders the shell-owned `chat-input` fragment via a template func
   helper, conditionally renders `empty-state-notice` above it, and
   keeps the intent chips + try-prompts as page content.
3. `internal/serve/pages/now/page.go` — adds `ChatRegistry` and
   `LiveSessions` to `Deps`; adds `buildChatInput`,
   `resolveAdapterState`, `subheadPlainText`, `renderQuickLaunch`,
   `QuickLaunchFragment`, and `SubheadText` helpers; exposes a
   `chatInput` / `emptyStateNotice` template FuncMap so Now templates
   can embed shell-owned fragments without merging template sets.
4. `internal/serve/pages/now/styles.go` — extends `nowScript`'s SSE
   subscriber with `hero` (textContent swap on
   `[data-page-hero-subhead]`) and `capability` (refetch
   `/api/now/quicklaunch` and outerHTML-swap `#now-quicklaunch`).
5. `internal/serve/pages/now/data/types.go` — adds `SessionRow` type
   alias re-exporting `agentspage/data.SessionRow`.
6. `internal/serve/pages/now/data/agents.go` — `AgentsInputs` gains
   `LiveSessions`; `LoadAgents` consumes the ledger to populate
   `Running` + `RunningCount` and filters non-live statuses.
7. `internal/serve/pages/now/import_test.go` — drops the chat ban so
   capability detection compiles; runner ban stays.
8. `internal/serve/shell/templates/page-hero.html` — wraps `{{ .Subhead }}`
   in `<span data-page-hero-subhead>` so every page-hero subhead is
   SSE-swappable.
9. `internal/serve/api/now.go` — adds `event: hero` (carries
   `now.SubheadText` payload), `event: capability` (signal only,
   triggers client refetch), the `/api/now/quicklaunch` fragment
   endpoint, and the new `sectionsForEventType` fan-out helper.
10. `internal/serve/server.go` — threads `ChatRegistry` +
    `LiveSessions` into both `nowpage.Deps` wiring sites
    (page-handler register + API handler construction).

Tests added / updated:

- `internal/serve/pages/now/page_test.go` — asserts
  `data-chat-input-variant="hero"`, `empty-state-notice`,
  `data-page-hero-subhead`, and `#now-quicklaunch` in the rendered
  body; adds `TestRegister_NoEmptyStateWhenAdapterConnected` that
  registers a fake adapter and asserts the empty-state notice does
  NOT render.
- `internal/serve/pages/now/data/agents_test.go` — adds
  `TestLoadAgents_PopulatesRunningFromLedger`,
  `TestLoadAgents_LedgerEmptyKeepsEmptyState`,
  `TestLoadAgents_LedgerFiltersDoneAndFailed`.
- `internal/serve/api/now_test.go` — adds
  `TestSectionsForEventType_FanOutHeroAndCapability`,
  `TestNowHandler_SSEEmitsHeroOnInboxEvent`,
  `TestNowHandler_SSEEmitsCapabilityOnAdapterEvent`,
  `TestNowHandler_QuickLaunchFragmentEndpoint`.

## Boundaries

- **No new chat dispatch logic.** Quick launch still hands submits
  off to the chat-and-model adapter; this spec only swaps the
  template + adds the no-adapter notice.
- **No new metric pipelines.** The page-hero subhead reuses
  existing inbox + agent counts; no new aggregator.
- **No new SSE bus.** Reuse the existing per-event-type debounce in
  `api/now.go`.
- **No PR/review work on the Agents home's session ledger.** This
  spec consumes whatever shape Agents shipped; refactoring (Option
  A) is in scope, redesigning is not.

## Acceptance Criteria

- WHEN the user opens `/now` THE SYSTEM SHALL render the Quick
  launch section with the shell's `chat-input` fragment (the
  rendered DOM SHALL include `data-chat-input-variant="hero"`).
- WHEN the user opens `/now` AND no chat adapter is connected THE
  SYSTEM SHALL render the `empty-state-notice` fragment immediately
  above the chat input.
- WHEN the user opens `/now` AND a chat adapter IS connected THE
  SYSTEM SHALL NOT render the empty-state-notice.
- WHEN the inbox count or running-agent count changes THE SYSTEM
  SHALL emit `event: hero` on the `/api/now/events` SSE channel
  with the new subhead text as the data payload.
- WHEN the client receives `event: hero` THE SYSTEM SHALL update the
  `[data-now-subhead]` element's text content without a page
  reload.
- WHEN a chat adapter connects or disconnects THE SYSTEM SHALL emit
  `event: capability` on the SSE channel; the client SHALL re-fetch
  `/api/now/quicklaunch` to update the empty-state state.
- WHEN a live agent session is running for the user THE SYSTEM
  SHALL render the Currently-running block on Now sourced from the
  shared session ledger (NOT a stub).
- WHEN no live session is running THE SYSTEM SHALL render the
  existing empty state on the Currently-running block.
- THE SYSTEM SHALL NOT break any existing acceptance criterion from
  the `hero-now-home` spec.

## Risks

- **Shared session ledger extraction (Option A) is bigger than it
  looks.** The Agents home's `sessions.go` is ~270 lines and may
  have agent-page-specific shaping baked in. If extraction balloons,
  fall back to Option B (direct import) with a TODO; document the
  decision in the kickoff. Either way, do NOT block Now from
  consuming live data.
- **SSE event proliferation.** Adding `hero` + `capability` brings
  the per-page SSE event types to 7 (inbox, plate, agents, changes,
  hero, capability, ping). If subscribers struggle with that breadth,
  consider collapsing related events client-side. Out of scope here
  — just keep an eye on subscriber complexity.
- **Empty-state vs adapter-connect race.** The page-render snapshot
  of `NoAdapter` may go stale within milliseconds. The
  `event: capability` re-fetch covers it; if the re-fetch flickers
  on rapid connect/disconnect, debounce client-side.

## Validation

- Manual: with no chat adapter configured, open `/now`. Verify the
  empty-state notice appears above the Quick launch input and the
  input still accepts `/ask` and `/note` (those execute server-side
  without an adapter).
- Manual: with a chat adapter (hero-code) reachable, open `/now`.
  Verify no empty-state notice; the input is normal; ⌘K opens the
  overlay and shows `via hero-code` in the footer chip.
- Manual: trigger an inbox change (e.g., approve a proposal via the
  CLI). Verify the page-hero subhead updates without a reload.
- Manual: start an agent session externally. Verify the
  Currently-running block on Now populates within a few seconds.
- Test: handler test asserts the chat-input fragment is rendered.
- Test: handler test asserts empty-state-notice appears/disappears
  based on a fake capability.
- Test: SSE test asserts `event: hero` and `event: capability` fire
  on appropriate triggers.
- Test: `agents_test.go` asserts Running populates from a fake
  ledger.

## Kickoff

**Status: delivered 2026-05-17.** All four fixes shipped. Now home
now uses the shell's `chat-input` fragment, renders the
`empty-state-notice` above it when no adapter is connected, refreshes
the page-hero subhead via SSE on `event: hero`, and populates the
Currently-running session block from the shared session ledger.

**Session ledger choice: Option B.** `now/data/types.go` re-exports
`agentspage/data.SessionRow` via a type alias. The agentspage shape
was already neutral enough that lifting it into a third package would
churn without clarity gains. TODO marker points to promoting it to
`internal/serve/sessions/` if a third consumer appears.

**Latent capability events.** The `event: capability` SSE branch
and the `/api/now/quicklaunch` fragment endpoint are wired and
correct, but the chat registry doesn't yet publish
`chat.adapter.added` / `chat.connected` / `chat.disconnected`
events to the bus. Until that wiring lands, adapter connect/disconnect
requires a page reload to flip the empty-state. Trivial follow-up
when chat-registry publishing is added.

**Pick up at:** the surface architecture initiative is now functionally
complete end-to-end. Remaining work is incremental polish + the home
specs' own follow-ups (live transcript streaming on Agents, real chart
data on People & ROI, etc.).
