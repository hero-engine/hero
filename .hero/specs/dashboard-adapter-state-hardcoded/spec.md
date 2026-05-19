---
title: Dashboard adapter state is hardcoded — "via hero-code" chip lies, panels disagree
slug: dashboard-adapter-state-hardcoded
type: bug
status: completed
severity: high
root_cause_class: code
priority: high
tags: [dashboard, serve, chat-adapter, ui-state]
created: 2026-05-19
---

# Dashboard adapter state is hardcoded — "via hero-code" chip lies, panels disagree

## Issue

Reporter: 277887514+chet-bellows@users.noreply.github.com — observed live in browser on 2026-05-19 against
`hero serve` running locally on this workspace.

Symptoms observed on Now (`/now`) and Work (`/work`):

- The top-nav header shows a green `· via hero-code` adapter chip with the
  title "Chat adapter: hero-code (connected)" on every page render.
- At the same time, the `/now` Quick-launch section renders the "Install
  hero-code" empty-state panel as if no adapter were connected.
- The `/work` page renders the inline chat-input in its **disabled** state
  with placeholder "Connect a chat adapter to enable" plus a "Connect
  adapter →" link, again contradicting the header chip.
- The same disabled chat-input renders on the four non-Now homes (Work,
  Knowledge, People & ROI, Agents).

The two halves of the UI cannot both be right. Either an adapter is
connected (chip is honest, install panel + disabled input are stale) or
it isn't (chip is a lie).

## Investigation

### What's actually wired

| Component | Reads from | Lives at |
|---|---|---|
| Top-nav adapter chip | **Nothing — hardcoded HTML** | `internal/serve/shell/templates/top-nav.html:23-25` |
| Now install-panel + chat-input enable/disable decision | `chat.Resolve(deps.ChatRegistry, "").Interactive` | `internal/serve/pages/now/page.go:402-418` (`resolveAdapterState`) |
| Now chat-input `Disabled` field | **Never set** — `buildChatInput()` always returns `Disabled:false` | `internal/serve/pages/now/page.go:385-393` |
| Work / Knowledge / People / Agents chat-input disabled flag | `Server.chatInteractiveConnected()` → `chat.Resolve(...).Interactive != ""` | `internal/serve/server.go:774-786` + `internal/serve/pages/work/page.go:114-142` |

### End-to-end flow at request time

1. User hits `GET /now`.
2. `internal/serve/pages/now/page.go::handle` runs `buildPage`.
3. `resolveAdapterState(h.deps)` at `page.go:402` calls
   `chat.Resolve(deps.ChatRegistry, "").Interactive`. Because no adapter
   is registered in this workspace (hero.json has no `chat.headless`
   block — see `internal/serve/server.go:586-606`), `cap.Interactive` is
   the empty string, so `noAdapter = true` and the "Install hero-code"
   empty-state CTA payload is returned.
4. `buildChatInput()` at `page.go:385` ignores adapter state entirely and
   returns `shell.ChatInput{Variant:"hero", Placeholder:"Tell Hero what to
   do next…"}` with `Disabled` unset.
5. The Now template `quicklaunch.html` renders both the install banner
   (because `.NoAdapter` is true) **and** an enabled-looking chat input.
6. The Work handler at `page.go:114-130` builds its inline chat-input,
   sees `isChatDisabled(probe) == true`, and sets `in.Disabled=true,
   in.Placeholder="Connect a chat adapter to enable", in.ConnectHref=
   "/settings/chat"`.
7. The shell renders the page using the **same** `top-nav.html` template
   on every page, with its hardcoded `via hero-code` chip and the
   tooltip "Chat adapter: hero-code (connected)".

### Why the chip is hardcoded

`internal/serve/shell/templates/top-nav.html:23-25`:

```html
<span class="adapter-chip" title="Chat adapter: hero-code (connected)">
  <span class="adapter-dot"></span>via hero-code
</span>
```

No template variable. No conditional. The chip ships as static HTML
inside the shell's top-nav partial, so it appears on every shell-
rendered page regardless of actual adapter state. The `shell.Chrome`
struct that drives the rest of the top-nav (`{{ with .Workspace }}`,
`{{ range .Tabs }}`, etc.) has no field for adapter state.

### Why the workspace shows no adapter today

`internal/serve/server.go:590-606` registers a hero-code adapter
**only if** `hero.json` has a `chat.headless.endpoint` configured and
the endpoint is reachable. This workspace's `.hero/hero.json` has no
`chat` block at all — confirmed by reading the file. So
`chat.Resolve(...)` returns an empty Capability and every probe
correctly reports "no adapter".

Verified live:

```bash
$ jq -r '.chat // "no chat block"' .hero/hero.json
no chat block
```

### Reproduction (no special state required)

1. `cd` to any hero workspace whose `.hero/hero.json` has no
   `chat.headless.endpoint` configured (e.g. this one).
2. `hero serve` → open `/now` and `/work` in a browser.
3. Header shows green `· via hero-code` chip on both pages.
4. `/now` body shows the "Install hero-code" install panel.
5. `/work` body shows the inline chat-input in disabled state with
   "Connect a chat adapter to enable" placeholder.
6. `curl localhost:<port>/api/chat/capability` returns
   `{"adapters":[], "interactive":"", "headless":"", ...}` —
   confirming nothing is registered, despite the chip.

### Root cause

The shell's top-nav adapter chip is **static HTML**. Page bodies fetch
adapter state correctly through `chat.Resolve`. The two never meet —
the chip never receives the registry probe result.

Secondary defect in the same flow: `buildChatInput()` on Now never
honours adapter state. The Now Quick-launch chat-input is always
rendered enabled, so a user can type into a chat box that will be
dispatched to nothing while a sibling banner tells them they must
install hero-code first. The Now and Work disabled-state contracts
disagree.

### Severity

**High.** Affects every page of the dashboard on every workspace that
hasn't configured a chat endpoint (the out-of-box experience for
solo-mode hero). Users cannot tell whether their adapter is connected,
and at least one widget will lie to them on every render. No data
loss — purely UI deception — but it actively undermines trust in the
rest of the dashboard's reporting.

Caused by our codebase. Workaround: ignore the header chip; trust the
page body. That requires the user to know which widget to believe.

## Code Flow (End to End)

1. `internal/serve/server.go:580-606` — server boot probes `hero.json`
   for `chat.headless.endpoint`. Without one, no adapter is registered.
2. `internal/serve/server.go:632-648` — `buildShellRouter` constructs a
   `shell.Router` whose chrome carries `workspace`, `branch`, `userName`,
   `version`. **No adapter state is plumbed into the shell.**
3. `internal/serve/shell/templates/top-nav.html:22-25` — top-nav
   partial renders a fixed `<span class="adapter-chip">via hero-code</span>`
   on every page.
4. `internal/serve/pages/now/page.go:402-418` — Now handler computes
   `(noAdapter, emptyState)` via `chat.Resolve(...).Interactive == ""`
   and passes them to the Quick-launch template.
5. `internal/serve/pages/now/page.go:385-393` — `buildChatInput()`
   builds Now's chat-input WITHOUT consulting adapter state; the input
   is always `Disabled:false`.
6. `internal/serve/pages/now/templates/quicklaunch.html:13-16` — when
   `.NoAdapter` is true, the install empty-state banner renders above
   the always-enabled chat-input.
7. `internal/serve/pages/work/page.go:114-142` — Work handler's
   `chatInputFor` calls `isChatDisabled(probe)`; when no adapter is
   registered, sets `Disabled=true, Placeholder="Connect a chat adapter
   to enable", ConnectHref="/settings/chat"`.
8. `internal/serve/shell/templates/chat-input.html:7-34` — when
   `Disabled` is true with `ConnectHref` set, the partial renders the
   muted input, hides the submit button, and appends "Connect adapter
   →" alongside the context chips.

## Key Files

### Shell chrome (header chip)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/shell/templates/top-nav.html` | 22–25 | Hardcoded `via hero-code` chip |
| `internal/serve/shell/types.go` | (whole) | `shell.Chrome` struct — has no adapter field today |
| `internal/serve/shell/shell.go` | (router build) | Where Chrome is composed; adapter probe needs to land here |

### Now page (install panel + chat-input)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/pages/now/page.go` | 385–418 | `buildChatInput` ignores adapter state; `resolveAdapterState` correctly reads it |
| `internal/serve/pages/now/templates/quicklaunch.html` | 12–20 | Install banner + chat-input rendered together |

### Work / Knowledge / People / Agents (disabled chat-input)
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/server.go` | 774–786 | `chatInteractiveConnected` probe wired into four homes |
| `internal/serve/pages/work/page.go` | 114–142 | Disabled-state chat-input builder; mirror in knowledge/people/agentspage |
| `internal/serve/shell/templates/chat-input.html` | 7–34 | Disabled-state rendering |

### Adapter resolution
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/chat/capability.go` | 26–71 | `chat.Resolve` — the single source of truth |
| `internal/serve/server.go` | 590–606 | Boot-time hero-code registration |
| `internal/serve/chat/registry.go` | (whole) | Adapter registry |

## Secondary Defects

1. **Now chat-input is never disabled even when no adapter is connected.**
   `buildChatInput()` at `pages/now/page.go:385` never sets `Disabled`.
   Pairs the install banner with a chat box that looks usable. Different
   contract from the four other homes (which all DO disable). Either
   Now should disable like the rest, or all five homes should keep the
   input enabled (and let the dispatcher surface the error). Pick one;
   right now Now is the inconsistent one.

2. **The chip tooltip claims "(connected)" unconditionally.** Even if a
   future fix flips the chip's text dynamically, the title attribute
   needs the same treatment.

3. **`/api/chat/capability` payload is exposed for islands** (per the
   `Capability` struct comment) but no front-end client subscribes to
   adapter connect/disconnect events to update the chip. The SSE
   plumbing exists for other refreshes (`/api/now/quicklaunch`); the
   chip is just not on the wire.

## Notes

- The user's spec corpus already contains `hero-chat-and-model` and
  `chat-runner-pseudo-stream` (both completed) — the adapter substrate
  is delivered. The dashboard chrome never picked up the wiring.
- Once the chip is reactive, the Work-page disabled-input state copy
  ("Connect a chat adapter to enable") is fine as-is. The bug is
  purely the chrome lying — the body widgets read the right signal.
- `command-bar.js:677` has its own muted `cmd-adapter-chip` for the
  command bar — that one DOES seem to read state, so it's not in scope.

## Acceptance Criteria

- THE SYSTEM SHALL render the top-nav adapter chip from
  `chat.Resolve(...).Interactive` (or an equivalent registry probe), not
  from static HTML.
- WHEN no interactive chat adapter is registered THE SYSTEM SHALL render
  the chip with a muted "no adapter" label (mirroring
  `command-bar.js`'s muted chip) AND keep the install panel and the
  disabled chat-input visible.
- WHEN at least one interactive chat adapter is registered THE SYSTEM
  SHALL render the chip with the connected adapter's display name
  (e.g. "via hero-code") AND hide the install panel on Now AND enable
  the chat-input on every home.
- THE SYSTEM SHALL apply the same Disabled contract for the chat-input
  on Now as on the four other homes — both honour
  `chat.Resolve(...).Interactive == ""`.
- WHEN an adapter registers or de-registers at runtime THE SYSTEM SHALL
  push an SSE event that updates the chip without a full page reload
  (mount on the same channel that already drives `/api/now/quicklaunch`).

## Goal

The top-nav chip, the Now install panel, and every page's chat-input
all derive their visible state from the same chat-adapter registry
probe. No hardcoded "connected" claims; no surface disagrees with any
other surface on whether the workspace has a working adapter.

## Boundaries

- Not in scope: adding new adapter types, changing the registry
  protocol, or building a settings UI at `/settings/chat`. The existing
  empty-state CTA can keep linking there until a separate spec lands
  the page.
- Not in scope: fixing the broader user-identity attribution bugs (see
  `dashboard-user-identity-os-env-mismatch`).
- Not in scope: emitting delivery-complete events on `hero spec
  complete` (see `dashboard-delivery-events-never-emitted`).

## Risks

- Reactive top-nav chip needs to subscribe to an adapter-state SSE
  channel; if that channel only exists conceptually today, this spec
  may need to land the publisher on `chatRegistry.Register/Unregister`.
- Disabling the Now chat-input symmetrically with the other homes
  changes a long-standing UX choice — confirm with the hero-now-home
  spec author before flipping.
- Template-set differences across pages mean the chat-input partial is
  parsed in two places (`internal/serve/pages/now/page.go::
  loadTemplatesFor` uses its own funcs map). Verify the disabled
  contract still renders correctly on Now.

## Validation

1. With `hero serve` running and no chat block in hero.json:
   - Chip reads muted "no adapter" (or similar) on every page.
   - `/now` body still shows install panel + Now chat-input disabled.
   - `/work`, `/knowledge`, `/people`, `/agents` chat-input remains
     disabled with the same Connect-adapter affordance.
   - `curl /api/chat/capability` returns empty.
2. Configure `chat.headless.endpoint` in `hero.json` pointing at a
   reachable hero-code:
   - Chip reads "via hero-code" with green dot.
   - `/now` body hides the install panel; chat-input is enabled.
   - `/work` chat-input is enabled with the page-context chips
     intact and no Connect-adapter link.
3. Stop hero-code; verify the chip flips to muted within ~1s via SSE
   without a page reload.
4. Regression tests:
   - Add a shell test that renders `top-nav.html` with both adapter-
     connected and adapter-absent chrome data, asserting visible chip
     differs.
   - Add a Now-page render test pinning the Now chat-input's
     `Disabled` field to the adapter-state probe.

## Changes

- `internal/serve/shell/types.go` — added `AdapterState` struct and `Chrome.Adapter` field. Single source of truth the chip template reads on every render.
- `internal/serve/shell/shell.go` — added `Router.adapterProbe` field, `SetAdapterProbe` setter, and `resolveAdapter` helper. `buildChrome` calls the probe per-request so the chip reflects current registry state on every navigation.
- `internal/serve/shell/templates/top-nav.html` — chip now renders from `.Adapter.Connected` + `.Adapter.DisplayName` with a muted "no adapter" branch when disconnected. No more hardcoded "via hero-code" string.
- `internal/serve/shell/static/shell.css` — added `.adapter-chip.muted` styling (mirrors the command-bar muted-chip treatment).
- `internal/serve/server.go` — added `Server.shellAdapterState` probe (via `chat.Resolve` — same source as `chatInteractiveConnected`) plus `lookupAdapterDisplayName` helper. Wired with `r.SetAdapterProbe(s.shellAdapterState)` right after `shell.New`.
- `internal/serve/pages/now/page.go` — `buildChatInput` now takes a `noAdapter` bool and flips into the disabled/Connect-adapter state symmetric with the four other homes. Both call sites (initial render + `renderQuickLaunch` SSE fragment) updated.
- `internal/serve/shell/shell_render_test.go` — regression tests for connected and disconnected chip rendering.
- `internal/serve/shell/shell_test.go` — `TestBuildChrome_AdapterProbe` covers nil/connected/reset probe flows.
- `internal/serve/pages/now/page_test.go` — `TestBuildChatInput_DisablesWhenNoAdapter` and `..._EnabledWhenAdapterConnected` pin the new symmetric contract.

## Scope notes

AC #5 (SSE-driven live chip update without page reload) is partially met: the chip refreshes on every page render and on every `/api/now/quicklaunch` fragment refresh, which already runs on the existing SSE channel. A registry-driven publish path (live update on adapter `Register`/`Unregister` without nav) would require a new pub-sub channel from `chat.Registry` and is left as a follow-on enhancement — the core "chrome lies" bug is resolved by the per-render probe.

## Recap

The shell's top-nav adapter chip is hardcoded HTML; page-body widgets
correctly read `chat.Resolve(...)` but the chip never does. On a fresh
workspace with no chat endpoint configured the chip claims
"via hero-code (connected)" while the Now install panel, the Work
chat-input, and `/api/chat/capability` all agree there's no adapter.
Single-component fix in the shell, plus a small symmetry fix to make
Now's chat-input honour the same probe the other four homes use.
