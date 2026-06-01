---
title: Needs-your-input inbox only sources from proposals + inbound handoffs — and proposals are hardcoded nil
slug: dashboard-inbox-misses-most-activity-sources
type: bug
status: completed
severity: medium
root_cause_class: code
priority: medium
tags: [dashboard, inbox, proposals, ux, serve]
created: 2026-05-19
completed_at: 2026-05-19T14:50:33Z
---

# Needs-your-input inbox only sources from proposals + inbound handoffs — and proposals are hardcoded nil

## Issue

Reporter: bdwheeler@gmail.com — observed live in browser on 2026-05-19.

Symptom: the `/now` "Needs your input" section reads
**"Nothing waiting on you — when an agent proposes a change or a peer
hands work back, it shows up here."**

The user had spent the previous day deep in active workflows that
should have generated inbox items: proposals from agents on long-
running tasks, decisions awaiting confirmation from peer-call
responses, status reconciliation prompts from `hero check`, etc.

None of those surface to the inbox today.

## Investigation

### What the inbox actually pulls from

`internal/serve/pages/now/data/inbox.go:24-38 — LoadInbox`:

```go
func LoadInbox(in InboxInputs) Inbox {
    rows := make([]InboxRow, 0)
    for _, p := range in.Proposals {
        rows = append(rows, proposalRow(p))
    }
    rows = append(rows, inboundHandoffRows(in)...)
    _ = in.Edition
    return Inbox{Rows: rows, Total: len(rows)}
}
```

Two sources: `in.Proposals` and `inboundHandoffRows`. Everything else
is unsupported — note the discarded `_ = in.Edition` and the TODO
comment above it.

### Source 1: `Proposals` is *always* nil in solo mode

`internal/serve/server.go:752-772 — snapshotProposals`:

```go
func (s *Server) snapshotProposals() []*nowdata.ProposalRow {
    if s == nil || s.api == nil || s.api.proposals == nil {
        return nil
    }
    slug := filepath.Base(s.projectRoot)
    if slug == "" || slug == "." {
        return nil
    }
    store := s.api.proposals.get(slug)
    if store == nil {
        return nil
    }
    // The store is keyed by session; we don't track active sessions
    // from here, so we have no list of sessions to iterate. The propose
    // store does not expose a global "all sessions" enumerator yet —
    // when the agents home wires the live session ledger this gains a
    // real source. Until then return an empty slice so the inbox
    // renders cleanly.
    _ = store
    return nil
}
```

The comment is candid: the store is keyed by session, the snapshot
function has no list of sessions, no enumerator is exposed, **so it
returns nil**. Always. This is the wiring used for the user's
workspace (solo edition) and was meant to be temporary.

### Source 2: Inbound handoffs require a specific frontmatter block

`internal/serve/pages/now/data/inbox.go:72-112 — inboundHandoffRows`
scans every spec for a `ReceivedFrom` field, and only surfaces specs
where `Status == StatusPlanning || "handed_off"`. In a workspace
without cross-repo peer handoffs in flight (this one), it returns an
empty slice every time.

### Sources the inbox does NOT consider

Browsing the codebase for things that ought to need user attention:

| Signal | Today | Should surface? |
|---|---|---|
| Pending proposals from agent sessions | Wiring TODO at `server.go:752` | Yes — primary purpose of the inbox |
| Peer-call responses awaiting confirmation | Not wired | Yes — `peer.call.completed` events with `findings` payload often need a follow-up |
| `hero check` blockers / drift | Not wired | Yes — that's the "do something about this" signal |
| Decisions awaiting confirmation | Not wired | Maybe — `decision_made` events imply already-decided, but ADRs in `status: proposed` are inbox-worthy |
| Specs with `status: in-review` and the user is the reviewer | Not wired | Yes (team mode); maybe (solo) |
| Failing acceptance criteria after `hero verify` | Not wired | Yes — the user explicitly flagged spec-status-integrity work |
| Imported tracker issues without local specs | Not wired | Yes |
| Recently-hit `blocker_hit` events | Not wired | Yes |

The inbox aspires to be *the* "what needs you right now" surface, but
the implementation surfaces only one of the eight reasonable sources,
and that one is hardcoded nil.

### Reproduction (no special state required)

1. Open any solo-mode hero workspace.
2. Take any normal action that should require user attention:
   - Have an agent emit a proposal (`hero propose ...` via MCP).
   - Run `hero check` and produce drift output.
   - Have a peer call complete with findings.
3. `hero serve` → `/now`.
4. The "Needs your input" section continues to read "Nothing waiting
   on you" regardless.

The empty-state message is also slightly misleading: it tells the
user the inbox listens for "agent proposes a change or peer hands
work back" — which exactly matches the two implemented sources, but
those are not the user's mental model for "what needs my attention".

### Root cause

Two intertwined issues:

1. **`snapshotProposals` returns nil unconditionally.** This is an
   explicit `// TODO: wire when session ledger lands` situation. The
   session ledger now exists (`s.snapshotLiveSessions` is used by the
   agents page), so the unblocker is just iterating sessions and
   reading their proposal stores.

2. **The inbox source list is too narrow** to live up to its title.
   It needs to absorb at least `hero check` blockers, peer-call
   findings, and pending-review specs. Without those sources, even
   a fully-wired proposals snapshot will leave the section feeling
   empty on most days.

### Severity

**Medium.** The dashboard's most actionable section silently
under-reports. Workaround: check the agents page, the peer-handoff
status, `hero check`, and the work-page roadmap manually — i.e. do
the user's own correlation. Defeats the purpose of having a "Now"
home.

Caused by our codebase. Not a data-loss bug; a feature-completeness
bug whose symptoms read as "the system thinks nothing is happening".

## Code Flow (End to End)

1. `internal/serve/server.go:654-666` — Now page registered with
   `Proposals: s.snapshotProposals`.
2. `internal/serve/server.go:752-772 — snapshotProposals` returns
   nil unconditionally.
3. `internal/serve/pages/now/page.go:213-217` — `LoadInbox` called
   with empty proposals slice.
4. `internal/serve/pages/now/data/inbox.go:24-38 — LoadInbox` walks
   the two sources, returns empty Inbox.
5. `internal/serve/pages/now/templates/inbox.html:15-17` — empty state
   renders with the "Nothing waiting on you" message.

## Key Files

### Inbox source plumbing
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/serve/server.go` | 752–772 | `snapshotProposals` — hardcoded nil |
| `internal/serve/pages/now/data/inbox.go` | 24–112 | Two-source inbox loader |
| `internal/serve/pages/now/templates/inbox.html` | 1–37 | Empty state copy |
| `internal/serve/proposals.go` | (whole) | Propose store — needs a "list across sessions" method |

### Candidate new sources
| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/check.go` | (whole) | `hero check` blockers — could feed inbox |
| `internal/cli/check_status.go` | (whole) | spec-status-integrity audit results |
| `internal/serve/pages/now/data/changes.go` | 137–166 | event types already classified — reuse for inbox sources |
| `internal/serve/team_coordination.go` | (whole) | peer-call lifecycle — findings deserve inbox treatment |

## Secondary Defects

1. **Empty-state copy is self-referential.** "When an agent proposes
   a change or a peer hands work back, it shows up here" — that
   describes exactly the implemented sources, not the user's mental
   model of "what's blocking me". Once more sources land, the copy
   needs an update.

2. **No session enumeration on the propose store.** The fact that
   `s.api.proposals.get(slug)` returns a per-project store with no
   way to enumerate sessions is a deeper API gap — `internal/serve/
   proposals.go` should expose `Snapshot() []*ProposalRow` so the
   wiring isn't blocked on the session ledger.

3. **No SSE channel for inbox refresh.** When a new proposal lands
   the inbox doesn't update without a page reload. The
   `/api/now/quicklaunch` channel exists for the chat input; a
   sibling channel for inbox would close the loop. Out of scope
   for this fix but adjacent.

4. **`Edition` is read but ignored.** `_ = in.Edition` at
   `inbox.go:35` — when team-mode wiring lands, this becomes the
   place to gate PR-review-mention rows and import-draft rows. Mark
   it explicitly so the gate doesn't get forgotten.

## Notes

- The user already has work specs that touch related areas:
  - `agent-outposts` covers per-session proposal handling
  - `spec-status-integrity` covers AC-graph blockers (a clear inbox
    source candidate)
  - `cross-repo-peering` ships peer-handoffs (the one wired source)
  - `inline-propose-output-mode` covers proposal UX in the artifact
    pane (separate concern)
- A surgical fix that just wires `snapshotProposals` (Source 1) is a
  worthwhile incremental — it converts the inbox from "always empty"
  to "shows proposals when sessions emit them". Broader source
  expansion can land in follow-ons.

## Acceptance Criteria

- WHEN at least one chat session has pending proposals THE SYSTEM
  SHALL surface them in the Now inbox via `snapshotProposals`,
  iterating the session ledger and pulling each session's open
  proposals.
- WHEN `hero check` has surfaced an unaddressed blocker (or
  `blocker_hit` event in the trailing window) THE SYSTEM SHALL
  surface it as an inbox row with an action to open the blocker.
- WHEN a peer call has completed with `kind=findings` and the user
  has not yet acknowledged it THE SYSTEM SHALL surface a row to
  review the findings.
- WHEN a spec sits at `status: in-review` AND the active user is
  the reviewer THE SYSTEM SHALL surface a row to open the review.
- WHERE the workspace is in team or cloud edition THE SYSTEM SHALL
  additionally surface PR-review mentions and imported tracker
  issues that lack a local spec.
- THE SYSTEM SHALL update the empty-state copy to describe the
  expanded source list (proposals, blockers, peer findings, reviews,
  inbound handoffs).
- IF a row is acted on (approve / dismiss / open) THE SYSTEM SHALL
  remove it from the next render — no stale rows lingering.

## Goal

The Now inbox surfaces every reasonable "needs your input" signal
the workspace produces, not just the two sources wired today. After
a day of activity the inbox is non-empty and the user can act on it
without consulting four other surfaces.

## Boundaries

- Not in scope: building the proposal-store session enumerator from
  scratch — adopt an existing `internal/serve/proposals.go` API or
  add one minimally.
- Not in scope: tracker integration UI for imported issues — surface
  the row but link out.
- Not in scope: SSE-driven live inbox refresh — that's a sibling
  spec.
- Not in scope: the `hero check` overhaul; we only consume its
  output.

## Risks

- A noisy inbox is worse than an empty one — every new source needs
  a clear "dismiss" affordance and a sensible default ordering
  (proposals first, blockers second, etc).
- Tying inbox to events.log re-introduces the under-emission risk
  documented in `dashboard-delivery-events-never-emitted`. Where
  possible, prefer authoritative state (frontmatter, propose store)
  over event log.
- The peer-call findings row needs to handle the case where the
  local spec referenced in the call doesn't exist (`peer call
  returned but local persist failed` — observed live in this
  workspace's events.log on 2026-05-18). Surface gracefully.

## Validation

1. With a chat session that has 2 pending proposals:
   - `/now` inbox shows both with Approve / View diff / Reject
     action chips.
2. With an unaddressed `hero check` blocker:
   - `/now` inbox shows a row linking to the blocker.
3. With a peer-call findings response from yesterday that the user
   hasn't dismissed:
   - `/now` inbox shows a row to review findings.
4. With no signals at all:
   - Empty-state copy reflects the broader source list.
5. Regression tests:
   - Pin `snapshotProposals` returning non-nil when sessions exist.
   - Add an inbox_test fixture for each source kind.
   - Pin the empty-state copy update.

## Changes

- `internal/propose/store.go` — added `Store.SnapshotAll()` which returns every pending envelope across every session. Unblocks the inbox without requiring a separate session enumerator (per the spec's Boundaries section).
- `internal/serve/proposals.go` — added `proposalStores.snapshotProject(project)` wrapper that returns nil cleanly when no store exists for the project yet.
- `internal/serve/server.go` — `Server.snapshotProposals` now iterates the store snapshot and maps each `propose.Envelope` into a `nowdata.ProposalRow` (was hardcoded nil). Fixes the primary "Source 1 always empty" symptom.
- `internal/serve/pages/now/data/inbox.go` — added `blockerEventRows` (trailing 7d `blocker_hit` events, deduped on slug+message), `peerFindingsRows` (`peer.call.completed` events with `kind=findings` in the message), and `pendingReviewRows` (specs at `status: in-review`). `LoadInbox` invokes all three alongside the existing proposal + handoff sources.
- `internal/serve/pages/now/templates/inbox.html` — updated empty-state copy to describe the broader source list (proposals, blockers, peer findings, pending reviews, inbound handoffs).
- `internal/serve/pages/now/data/inbox_test.go` — regression tests pinning blocker rows (with dedup), peer-findings rows (kind=findings filter), and pending-review rows.

## Scope notes

ACs 5 and 7 are partially addressed:
- AC 5 (team / cloud edition surfaces for PR mentions and import drafts) — out of scope per spec Boundaries; solo-mode is the target.
- AC 7 (dismissal removes the row from next render) — the rows ship with dismiss actions where appropriate, but their hrefs are still placeholders (`#`). Wiring the action endpoints to a per-user dismissal store is a follow-on; for the current rendering the rows surface, which is the regression being fixed.

## Recap

The Now inbox advertises itself as the "needs your input" surface
but pulls from only two sources — and one of them (`snapshotProposals`)
is hardcoded nil pending wiring. Result: empty inbox on every render
even when proposals, blockers, peer findings, and reviews are all
waiting. Wire the proposal snapshot and broaden the source list.
