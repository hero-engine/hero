---
title: `hero serve` routes inventory — one-paragraph map of the homes
type: feature
status: completed
severity: low
tags: [docs, serve, web-ui]
created: 2026-05-18
---

# `hero serve` routes inventory — one-paragraph map of the homes

## Kickoff

A minimal inventory page that names every web-UI home `hero serve`
exposes — `/now`, `/project`, `/work`, `/knowledge`, `/agents`, `/people`,
plus the ⌘K overlay and chat dispatcher — with one paragraph each.

**Status:** completed — `web/docs/src/serve/homes.md` landed with one
paragraph per home plus a "Global chrome" section flagging the ⌘K
overlay and chat dispatcher as still settling. Each route heading was
verified against the `RegisterHome(...)` call in the per-page source
before writing prose.

**Pick up at:** if the homes stop polish-pass churning, a follow-up
spec layers in detailed UI walkthroughs (view tabs, sub-routes,
SSE channel shapes, chat-dispatcher protocol) — all explicitly
deferred here.

→ `.hero/planning/features/hero-serve-routes-inventory/spec.md`

**Files touched:** `web/docs/src/serve/homes.md` (new),
`web/docs/mkdocs.yml` (nav: `Serve → Homes`),
`web/docs/src/cli/server-and-mcp.md` ("See also" pointer).

## Context

`hero serve` now exposes six top-level homes plus a global ⌘K command
overlay and a chat dispatcher. The shell registrations live in
`internal/serve/server.go` (the `buildShellRouter` method) and each home
is owned by a package under `internal/serve/pages/<name>/`. The current
docs only describe the API surface: `web/docs/src/cli/server-and-mcp.md`
covers `hero serve` flags, multi-project mode, MCP setup, and team mode
— it does not name any of the web routes a user lands on after starting
the daemon and opening the browser.

A user running `hero serve` and clicking the top-nav has no way to match
what they see to documentation. The homes are real and load-bearing
enough that this gap is worth closing — but the user has explicitly
flagged that the homes are still "polish-pass churning" (commits
`424bb36`, `9c4d61f`, `95f20d0`, `83c767c`, `9f988e5` are five separate
polish passes in a short window). Deep per-home behavioral docs are
**premature** until the surfaces settle.

The right doc for now is a minimal **routes inventory**: name the URL,
say one sentence about who it's for and what they'll find there, mark
the dispatcher pieces as still settling, and stop.

## Goal

A new `web/docs/src/serve/homes.md` page exists, is reachable from the
docs nav, and lists every top-level web route `hero serve` exposes with
a single-paragraph description of each. A reader running the daemon
locally can match every top-nav tab to a name and a one-line "this is
what it's for." The page explicitly defers detailed walkthroughs to a
later spec.

## Approach

Treat this as a navigation aid, not a manual. One short intro paragraph
naming the homes as a group, then one section per home with the route
in a heading and a single paragraph below it covering:

- What the home is (the noun the top-nav uses for it).
- Who it's for (operator / engineer / lead / etc.).
- What the user finds when they land there (in the most general terms
  — no per-section detail).

Verify every route against the actual `Register` call in the per-page
file before writing the heading. Don't infer routes from commit
messages; read the code.

The ⌘K overlay and chat dispatcher get their own short section because
they don't live at a route per se — they're chrome — and because the
user wants both flagged as "still settling."

## Changes

1. **New file `web/docs/src/serve/homes.md`.** Structure:
   - Intro paragraph naming `hero serve` and listing the homes as a
     bulleted route list (`/now`, `/project`, `/work`, `/knowledge`,
     `/agents`, `/people`).
   - One H2 section per home, with the route as the heading suffix:
     - `### Now — /now` (homepage; what's happening now in your
       workspace; primary entry for solo operators).
     - `### Project — /project` (project-snapshot home — surface, table,
       initiatives, archives timeline; landed in v0.10 via the project
       snapshot work).
     - `### Work — /work` (spec corpus view — sprint, throughput,
       quality views; ownership of `/work/spec/{slug}` detail routes).
     - `### Knowledge — /knowledge` (knowledge base entries —
     conventions, decisions, context, ingested notes).
     - `### Agents — /agents` (live agent sessions, proposals, automation
       wiring).
     - `### People — /people` (operator pulse, profiles, handoffs, ROI
       overview).
   - A short H2 **"Global chrome"** section covering:
     - The ⌘K overlay (mentioned briefly: a global command palette
       island registered in
       `internal/serve/shell/static/islands/command-bar.js`).
     - The chat dispatcher (mounted under `/api/chat/*` in
       `internal/serve/server.go`; surfaces inline on most homes).
   - A closing **"Still settling"** note: an explicit one-line
     deferral statement — *"Detailed UI walkthroughs land once the
     homes settle past polish-pass."*

2. **Update `web/docs/mkdocs.yml`.** Add a top-level `Serve` section
   to the nav (sibling to `CLI Reference`) with `- Homes: serve/homes.md`
   as its single child. Order it directly after `CLI Reference` so it's
   close to `Server & MCP`.

3. **Update `web/docs/src/cli/server-and-mcp.md`.** Add a one-line
   "See also" pointer near the HTTP Daemon section linking to
   `../serve/homes.md` for the route inventory.

## Boundaries

- **No UX walkthroughs.** No screenshots, no "click here to do X" guides,
  no annotated tours.
- **No component-level docs.** Do not describe view tabs, filter
  semantics, SSE channels, fragment endpoints, the proposal-store wiring,
  or how `chatInteractiveConnected` flips the inline chat input.
- **No behavior contracts beyond what's stable today.** Anything that
  was touched by a v0.10 polish-pass commit is still in flux — the
  inventory says it exists; it does not promise how it behaves.
- **No per-sub-route documentation.** `/work/spec/{slug}`,
  `/people/profiles`, `/project/snapshots/{date}`, etc. all exist; the
  inventory mentions detail routes only in passing and does not
  enumerate them.
- **No team-mode UI documentation.** `hero serve --team` exposes a
  different shape; documenting that surface is its own spec.
- **No `/_kitchen-sink` doc.** It's a dev-only fragment preview route.

A separate spec lands once the homes settle: that one covers behavior,
view tabs, sub-routes, and per-home workflows.

## Risks

- **Routes change before this page lands.** The polish-pass cadence is
  fast. Mitigation: keep each paragraph short enough that a route move
  is a one-line edit, and reference the registration site in the page
  source so a future reviewer can grep for it.
- **Mistaken inventory.** Listing a home that doesn't actually
  register, or missing one that does, is worse than no inventory.
  Mitigation: hard-link each heading to the `Register(...)` call site
  during review by name (file path in the spec's Changes section is
  already explicit).
- **Scope creep.** Reviewers may push to add "just one sentence on what
  the sprint tab does." Resist — the spec's Boundaries section exists
  specifically to make that easy to refuse.

## Validation

- Every route in the page resolves to a 200 when hitting
  `http://127.0.0.1:7437<route>` against a freshly built `hero serve`
  on this repo.
- Every route heading on the page has a matching `Slug:` value in the
  corresponding `internal/serve/pages/<home>/page.go` `RegisterHome`
  call (`now`, `project`, `work`, `knowledge`, `agents`, `people`).
- `mkdocs serve` from `web/docs/` renders the new section and page
  without errors and the nav shows `Serve → Homes`.
- The "Still settling" note is present and reads as a deferral, not a
  promise of future content.
- The page word count stays under ~600 words; if it's growing past
  that, the boundaries have failed.

## Related concerns

Documentation gaps spotted while scoping this spec — flagged but
**explicitly out of scope**:

- `hero snapshot` (v0.10, commit `5110b1a`) is the projector behind the
  `/project` home. Its CLI coverage on
  `web/docs/src/cli/search-and-context.md` should be audited; if thin,
  it deserves its own follow-up spec.
- The chat dispatcher endpoints (`/api/chat/capability`, `/api/chat/turn`,
  `/api/chat/history`, `/api/chat/preference`, `/api/chat/clear` —
  registered in `internal/serve/chat/api.go`) are completely
  undocumented. Out of scope here because the dispatcher is part of the
  "still settling" surface area.
- `hero serve --team` mode (job queue, worker pool, scheduled tasks,
  team auth, JWT secret) is mentioned in `cli/server-and-mcp.md` but
  not given a dedicated reference. Separate spec when team mode
  stabilizes.
