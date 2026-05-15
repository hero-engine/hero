---
title: "Cross-Repo Peering — Conventions Travel, Specs Hand Off, Heroes Call Each Other"
type: feature
status: delivering
priority: high
tags: [federation, peering, conventions, handoff, sync-peer-call, local-first]
created: 2026-05-15
relations:
  - target: hero-cloud-split
    kind: related
  - target: graph-memory-federation
    kind: related
  - target: cross-spec-awareness
    kind: related
  - target: agent-outposts
    kind: related
horizon: now
---

# Cross-Repo Peering — Conventions Travel, Specs Hand Off, Heroes Call Each Other

## Goal

Make sibling Hero workspaces work as a *tag team*: when a session in
repo A needs something from repo B's surface, it can ask B's Hero
*directly* (sync peer call), or hand the work off to B (async
handoff). And when A genuinely keeps the work but must respect B's
surface, B's conventions can travel into A's session.

Done means:

- A repo can declare which sibling repos it *peers* with (already
  partially done via `hero repos`), each with a stable **peer UUID**
  minted at `hero init`. The manifest, handoff records, and trail
  entries reference peer by UUID; local aliases (`hero repos`) are
  the human display form. Two machines can disagree on aliases
  without breaking peering.
- A session in A can issue a **sync peer call** to B — `hero peer
  call <alias> --mode=<advisory|spec-out|full> "<prompt>"` — that
  spins up a peer-side subagent and returns a structured result.
  - *Advisory*: B investigates, returns findings, writes nothing.
  - *Spec-out*: B designs a spec on its side with its native
    conventions baked in, returns the slug. This is the headline
    answer to "should A load B's conventions or ask B to design?"
  - *Full delivery*: B designs *and* delivers, returns commit/PR.
    Gated by approval + budget. v2.
- A spec can be **handed off** asynchronously with `hero handoff
  <slug> <peer>`. The originator's spec moves through a real
  lifecycle (`handed_off` → `awaiting_peer` → `handed_back` or
  `completed`) and the spec carries a `## Handoff Trail` recording
  every event. The receiving workspace gets a scaffolded spec
  stamped with provenance, and the trail is walkable from either
  side.
- Repos can declare which contract symbols they own. When a session
  in A edits files that import a peer-owned contract symbol,
  `/resume` and `hero context` **passively surface** a one-liner
  about the peer, the convention, and last change. No blocking, no
  prompts — just signal. Contract imports are the boundary
  *detector* powering the seamlessness goal.
- Convention loading across the boundary (`hero relevant --peer`) is
  still supported as a *fallback* for the case where work
  legitimately stays in A and must respect B's surface — e.g., a
  serializer in A producing a payload B parses.
- Everything above works **on a developer laptop with no cloud**,
  reading the peer's `.hero/` directory directly across a sibling
  checkout. The cloud-hosted case (graph-memory-federation) layers
  on top — same primitives, different transport.
- The design composes with `hero-cloud-split`'s contracts package:
  any new shapes (peer manifest, handoff record, peer call request /
  result, peer events) live in `contracts/peering/` with their own
  `PeeringContractsVersion` evolving independently of the main
  `ContractsVersion`.

## Kickoff

Three-tier ladder for sibling Hero workspaces: sync peer call (ask
B's Hero now), async handoff (pass the spec to B), or convention
import (load B's rules into A). Backed by a stable peer UUID, a
handoff trail recorded on every spec, and passive contract-import
surfacing.

**Status:** planning — design is locked, phasing sequenced, ready to
move toward `/deliver`. Existing `hero repos` foundation is the
launch point.

**Pick up at:** Phase 0 — mint peer UUIDs at `hero init`, plumb the
UUID through `hero.json`, upgrade `CrossRepoResolver` to dual-key on
UUID (canonical) and alias (display). Phase 1 follows immediately
with the handoff lifecycle and trail records.

→ `.hero/planning/features/cross-repo-peering/spec.md`

**Files:** `internal/cli/repos.go`, `internal/cli/init.go`,
`internal/spec/resolve.go`, `internal/config/config.go:1137-1184`,
`contracts/peering/` (new),
`.hero/planning/features/hero-cloud-split/spec.md` (contracts seam),
`.hero/planning/features/graph-memory-federation/spec.md` (cloud).
**Skip:** machine-to-machine peering across hosts, auto-trigger
boundary modes, full-delivery sync peer call — all v2+.

## Context

Today the user runs three sibling Hero workspaces: a Go backend
("app"), a web client, and a desktop client. Each has its own
`.hero/` with its own conventions, decisions, and specs. The CLIs
already know about each other in a thin way: `hero repos add` /
`hero repos scan` registers sibling aliases (`internal/cli/repos.go`,
`internal/config/config.go:45,1137-1184`), and `relations` in a spec
can reference a peer slug as `<alias>/<slug>` which `CrossRepoResolver`
(`internal/spec/resolve.go`) can read across the boundary.

But three gaps make peering feel inert in real work:

1. **Conventions don't travel — and aren't always the right answer.**
   When a session in the client touches the backend's API surface,
   the backend's conventions (error shapes, auth flow, naming) are
   *not* loaded. The client's Hero produces a change that runs
   face-first into the backend's expectations. The "load B's
   conventions" fix is leaky — A's session has a partial snapshot of
   B (conventions, no decisions, no broader patterns).
2. **Spec handoff is manual ceremony.** When the user concludes
   "this is really a backend job," the path is: write a spec on the
   backend by hand, paste in context from the client's session, hope
   the backend's Hero notices. There's no first-class handoff,
   no provenance, no return signal.
3. **No way to ask the peer's Hero *right now*.** Sometimes A
   doesn't need to hand off and doesn't want a stale convention
   snapshot — A wants a *live answer* from B about B's code. Today
   that means a human switching repos and starting a new session.

Two related streams of work touch this space and need to be honored:

- **`hero-cloud-split`** (delivering, Phase 0 just landed) is
  carving out a `contracts/` package as the only legal seam between
  the CLI repo and the cloud repo. That seam is itself an instance
  of cross-repo coordination — and it's the natural home for any
  wire shapes this spec defines.
- **`graph-memory-federation`** (planning) is the cloud-side answer
  to cross-repo signal: per-repo team graphs, unit-level join
  graphs, cross-repo edges in a server-hosted store. That answer is
  the right one when a cloud is present. It is *not* the right one
  for a solo dev with three sibling checkouts on a laptop.

Out-of-scope but orbital: `cross-spec-awareness` (intra-repo spec
dependency awareness) and `agent-outposts` (operable external systems
with scoped creds).

## Problem

The user said it plainly:

> "The client wants to make the changes on the app or the app wants
> to make the changes on the client. App equals back end. And they
> don't load each other's conventions to make sure that the changes
> are being done the way I would like them."
>
> "If my client suspects that there is backend work that needs to be
> done to fix the desktop client I would love the hero on my backend
> repo to see that spec and know it was raised by another hero and
> to investigate."

Three distinct failures:

**A. Convention starvation across the boundary** when A keeps the
work. A change in repo X that affects repo Y's surface is made
without Y's conventions in context. Y's reviewer finds it
inconsistent.

**B. Handoff has no first-class shape** when the work should move.
Today's `relations` is a one-way edge from A's spec body — B doesn't
see it, there's no lifecycle status, no provenance, no return
signal.

**C. No live peer query** when A wants an authoritative answer from
B without committing to a full handoff. The user often *just wants
to know*: "is there an existing endpoint for this?", "does this
change break you?", "what's your convention for X?"

Together these failures produce parallel work, neither hero aware of
the other's standards or in-flight tasks, and a human ferrying
context between them. The compounding promise of Hero — *the next
session starts smarter than the last one ended* — stops at the repo
boundary.

## Design

The design has five components, anchored by a three-tier interaction
ladder. The original A-or-B fork ("load conventions vs. handoff") is
**resolved by the sync peer call spec-out mode** — B designs the
spec on B's side, B's conventions kick in natively and persist as a
durable artifact. Convention import drops to a fallback.

### Three-tier ladder of interaction modes

When work in A touches B's surface, the session picks one of three
modes by increasing autonomy and durability:

| Mode | Writes? | Returns | Use case |
|---|---|---|---|
| **Sync peer call: advisory** | No | Findings text | "Does this break you?", "What's your convention?", "Is there an existing endpoint?" |
| **Sync peer call: spec-out** | Spec on B | Peer spec slug + initial handoff state | "I need backend work for this — design it on your side with your rules." |
| **Async handoff** | Spec on B (scaffolded) | Peer spec slug | "Here's an investigation I already did; pick it up when you're ready." |
| **Sync peer call: full delivery** (v2) | Spec + code on B | Commit/PR ref | "Just do it." Gated by approval prompt + budget. |
| **Convention import (fallback)** | No | Conventions in A's context | "Work stays in A but must respect B's surface — e.g. a serializer in A producing B's payload." |

Spec-out is the headline answer to the original fork: B's
conventions kick in natively *because B's Hero designs the spec*,
and the result persists as a real spec on B's side, walkable across
sessions. Async handoff handles the "I already did the work, take it
from here" case. Advisory handles the "I just need a fact" case.
Convention import handles the case where neither party should hand
off.

### 1. Peer identity: stable UUIDs minted at `hero init`

Every Hero workspace gets a UUID generated at `hero init` (or on
first invocation after upgrade for existing workspaces), stored in
`hero.json` as `peer_id`. This UUID is the **canonical identifier**
for the workspace across all peering operations.

```json
{
  "name": "app",
  "peer_id": "9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55",
  "repos": { ... },
  "peering": { ... }
}
```

Rules:

- Local aliases (`hero repos add`) remain the human-readable handle
  for display.
- The manifest, handoff trail records, peer call request envelopes,
  and any peer-routing artifacts reference peer by UUID, not alias.
- Resolution flow: UUID → look up local alias → display to human.
  Two machines can disagree on aliases ("app" on one, "backend" on
  another) without breaking peering.
- Migration: existing workspaces without a `peer_id` get one minted
  on first `hero` invocation after upgrade. A migration entry is
  written to the events log so the moment of identity assignment is
  recoverable. See Risks: identity stability.

`hero repos` extends to record the peer's UUID alongside the alias
and path. When `hero repos scan` discovers a new sibling, it reads
the sibling's `hero.json:peer_id` and records both.

### 2. Sync peer call — first-class third mode

A new top-level interaction: a session in A spawns a subagent
session in B's workspace, with B's full Hero context loaded, runs a
prompt, and returns a structured result. Calling shape:

```
hero peer call <alias> --mode=<advisory|spec-out|full> "<prompt>"
                       [--budget-turns N] [--budget-tokens N]
                       [--related-spec <slug>] [--reason <text>]
```

#### 2a. Advisory mode (v1)

- B's workspace spins up a subagent with full Hero context
  (conventions, decisions, code knowledge).
- The subagent investigates, returns findings as a structured
  result.
- **Never writes**, ever. No spec, no commit, no event log entry on
  B's side beyond the call record.
- Use cases: API change impact ("does this break you?"), convention
  lookup ("what's your error envelope?"), existing-code probe ("is
  there an endpoint for this?").

#### 2b. Spec-out mode (v1)

- B's workspace runs its `/design` flow (or equivalent) under the
  subagent.
- B's conventions, decisions, knowledge bake into the design
  naturally — the same way they would if a human ran `/design` on
  B's side.
- A spec is **saved to B's `.hero/planning/`** with status
  `planning` and a `received_from` block referencing A.
- Returned to A: the peer spec slug and a handoff trail entry.
- Pair with handoff: from B's side the spec is `planning` and
  `received_from` is set; from A's side the originating spec moves
  to `awaiting_peer` and the handoff trail records the spec-out
  call.
- **This is the headline answer to the convention-loading fork.**
  B's rules persist across sessions because they're baked into a
  real spec on B's side.

#### 2c. Full delivery mode (v2 — out of scope for v1)

- B designs and delivers. Returns a commit SHA or PR URL.
- Gated by an explicit approval prompt on A's side at call time, and
  by `--budget-turns` / `--budget-tokens` ceilings to prevent
  runaway peer work.
- Deferred to v2.

#### Call result shape

The call result is a structured block written into A's session
context (and, if `--related-spec` is set, appended to that spec's
`## Handoff Trail`):

```yaml
peer_call:
  call_id: 01HX...                    # ULID
  peer_id: 9c1c2f3e-...               # UUID
  peer_alias_display: app             # at time of call, for human reading
  mode: spec-out
  prompt: "Design the fix for error envelope mismatch we saw in client."
  result:
    kind: spec-ref                    # one of: findings | spec-ref | commit-ref
    spec_slug: error-envelope-mismatch
    peer_status: planning
  budget_consumed:
    turns: 12
    tokens: 4823
  at: 2026-05-15T14:00:00Z
  at_commit: 3176736
```

### 3. Handoff lifecycle — statuses, trail, async drop

Handoffs are a first-class operation with a real lifecycle. The
spec's **frontmatter status** moves through defined states, and a
**`## Handoff Trail`** section records every event so the spec
itself carries the audit trail.

#### Statuses (extend `internal/spec/spec.go`)

From the **originator's view**:

- `handed_off` — "I gave this to peer B" (initial transition).
- `awaiting_peer` — active state, "B has it, I'm waiting." This is
  the steady state while B works.
- `handed_back` — "B did their part (or bounced), ball's in my court
  again." Triggered when the peer's spec status reaches `completed`
  or the peer explicitly bounces.

From the **receiver's view**: the spec uses the existing lifecycle
(`planning` → `in-review` → `delivering` → `completed`). The
provenance lives in the `received_from:` frontmatter block and in
the `## Handoff Trail` reciprocal entry.

#### Handoff Trail section

Every spec involved in a handoff or peer call carries a
`## Handoff Trail` section. Each entry is one event:

```markdown
## Handoff Trail

- 2026-05-15T14:00:00Z — out → app (peer_id: 9c1c2f3e-...)
  mode: async-drop
  originating_spec: order-failure-error-display
  peer_spec: app/error-envelope-mismatch
  at_commit: 3176736
  reason: "Symptom is in the client, root cause is the API response shape."

- 2026-05-15T16:23:00Z — in ← app (peer_id: 9c1c2f3e-...)
  mode: handed-back
  peer_spec: app/error-envelope-mismatch (status: completed)
  result_ref: commit 4427cec
  reason: "Fixed in API layer. Verify error display in client."
```

Directions: `out` (this side initiated) / `in` (this side received).
Modes: `advisory`, `spec-out`, `async-drop`, `full-delivery`,
`handed-back`. The receiving peer's spec gets a **reciprocal trail
entry** referencing the origin spec and `peer_id`.

#### Cross-repo resolver

The resolver upgrades to walk the trail in either direction:

- Given an originator spec, list all peer spec refs it has handed
  to, and surface their current status.
- Given a receiver spec, walk back through `received_from` to find
  the originator and surface its status (so the receiver knows if
  A is still waiting).

Uses `peer_id` as the join key. Aliases are resolved to display
form per side.

#### Async drop ("handoff") command

```
hero handoff <slug> <peer-alias> [--title <new-title>] [--type <type>]
             [--reason <reason>]
```

- Updates the originator's spec: status → `handed_off`, appends a
  trail entry (mode: `async-drop`).
- Creates a scaffolded spec on the peer side at
  `<peer-path>/.hero/planning/<type>/<slug>/spec.md` with status
  `planning`, a `received_from` block, and a reciprocal trail entry.
- Emits `peer.handoff.sent` on originator and
  `peer.handoff.received` on receiver. Both events feed `hero
  recap`, `hero feed`, the graph.
- Slug collision on the peer side appends `-2`, `-3`, ... and
  reports the chosen slug.

#### Status return signal

When the peer's spec status reaches `completed`, the originator's
spec **does not auto-complete** — the originator must verify the
symptom resolved. But:

- A trail entry is appended automatically (mode: `handed-back`,
  `result_ref` pointing at the peer's commit / PR / completion
  event).
- The originator's status moves from `awaiting_peer` to
  `handed_back`.
- `hero status` surfaces it: "spec X was handed off to Y, which is
  now `completed` — verify and close on your side."

This requires reading the peer's spec status at status-render time,
done via the upgraded `CrossRepoResolver`. No new persistence on
the originator's side beyond the trail entry.

### 4. Contract metadata — surface, don't enforce

The manifest's `contracts:` section is consumed as a **context
signal**, not a guardrail. Three-step ladder:

#### v1 — Passive surfacing

When the current session edits files that import a peer-owned
contract symbol, `/resume` and `hero context` add a one-liner:

> "You're touching `contracts/events.Envelope` — owned by peer `app`
> (peer_id: 9c1c2f3e-...). Convention: error-envelope. Last changed:
> commit 4427cec."

No blocking, no prompts, no auto-call. Just signal.

**Key insight: contract imports are the boundary detector.** "This
file imports a peer-owned contract symbol" is the cleanest signal
that a change crosses a repo boundary — better than path heuristics
or commit-message scanning. v1 wires the detector to peer contract
imports as the primary mechanism for the seamlessness goal.

#### v2 — Boundary nudge

When the touch looks **structural** (changing a contract's shape,
not just consuming it), `hero nudge` suggests an advisory call. User
can accept, ignore, or silence per-spec.

Heuristic for "structural": the changed file edits a type
declaration, function signature, or schema field belonging to the
peer-owned symbol — vs. merely consuming it.

#### v3 — Auto-trigger

Contract-shape edits auto-open an advisory call (or escalate to
spec-out). Defer unless dogfooding shows the v2 nudge is friction.

#### Manifest contract section

```yaml
contracts:
  package: github.com/hero-engine/hero/contracts
  version: 3                          # main ContractsVersion (informational)
  shapes:
    - kind: event-envelope
      go_symbol: contracts/events.Envelope
      convention: error-envelope
    - kind: spec-frontmatter
      go_symbol: contracts/spec.Frontmatter
      convention: spec-format
```

The Go import scanner walks the consuming repo for any file that
imports a symbol named in any peer's manifest `contracts:` section.
Match → surface signal.

### 5. Peer convention seam — fallback, not headline

Each Hero workspace publishes a small machine-readable manifest of
the conventions it considers *peer-relevant* — conventions governing
its public surface (API shapes, error envelopes, auth flow) as
distinct from internal style guides.

This was originally the headline; **spec-out demoted it to a
fallback** for the case where work legitimately stays in A but must
respect B's surface (e.g., a serializer in A producing a payload B
parses, where designing a spec on B would be wrong).

**File: `<repo>/.hero/peer-manifest.yaml`** (generated by `hero
index`):

```yaml
schema: 1
repo:
  peer_id: 9c1c2f3e-4a8b-4f9d-9a0e-7e1f0d8c3a55   # canonical
  name: app                       # local short name (from hero.json)
  display: "Hero Backend"
  scope_hint: backend
conventions:
  - slug: error-envelope
    title: "Standard error envelope shape"
    surface: [http-response, sse-event]
    path: .hero/knowledge/conventions/error-envelope.md
    digest: sha256:abcd...
  - slug: auth-bearer-token
    title: "Bearer token format and rotation"
    surface: [http-request, header]
    path: .hero/knowledge/conventions/auth-bearer-token.md
    digest: sha256:...
contracts:
  package: github.com/hero-engine/hero/contracts
  version: 3
  shapes:
    - kind: event-envelope
      go_symbol: contracts/events.Envelope
      convention: error-envelope
```

Only conventions explicitly marked **peer-surface** appear — via a
`peer: true` flag in the convention's frontmatter, or via a glob in
`hero.json: { "peering": { "publish_conventions": [...] } }`.
Default publish set: **empty**. Principle of least authority applied
to context.

Consumer reads:

```
hero relevant --peer app                    # all peer-surface conventions
hero relevant --peer app --surface http-response
hero relevant src/api/orders.ts --peer app  # default behavior plus peer
```

Resolution: read from the peer's `.hero/` via the path the existing
`hero repos` registry resolves to. If unreachable, fail with a clear
"manifest missing — run `hero index` in `<peer-path>`" message.

### 6. Transport — local-first, cloud-augments

The hot constraint: this must work *with no cloud*. Three sibling
repos on a laptop, no login, no org_id. The cloud case (graph-
memory-federation) layers on top.

**Local (v1, default):**

- All cross-repo reads go through `hero repos`-resolved paths
  (`internal/config/config.go:1137-1184`).
- The peer's `.hero/` directory on local disk is the source of
  truth.
- Writes happen *only* in the workspace that owns the file. When
  `hero handoff` or a spec-out call runs, the two writes (receiver
  spec, then originator status) are local-filesystem writes from
  the same process. No daemon, no network.
- Sync peer call: A's process spawns a subagent that operates inside
  B's workspace directory. Implementation detail: subagent invoked
  via a peer-call shim that sets CWD to B's path, loads B's Hero
  context, runs the prompt, returns structured result over stdout.

**What about a peer on a different machine?** Out of scope for v1.
If the peer isn't on this disk, use cloud federation. v1 reports
"peer not reachable; mirror via cloud or wait." No SSH-into-peer
mode, no fancy daemon.

**Cloud (v2+, additive):**

When cloud is configured (`hero login` done, `cloud.org_id` set):

- Peer manifests published to the cloud (graph-memory-federation
  push API).
- Handoffs and peer calls against a peer not on local disk but in
  the same cloud org go through a server-mediated path: events
  routed by `peer_id`.
- Same data shapes (`contracts/peering/`); transport differs, model
  doesn't.

**Contracts location:** Per `hero-cloud-split`'s rule, anything
flowing between processes lives in `contracts/`. The peering
package:

```
contracts/peering/
  manifest.go     // PeerManifest, ConventionEntry, ContractEntry
  handoff.go      // HandoffRecord, HandoffStatus, ReceivedFrom, TrailEntry
  peercall.go     // PeerCallRequest, PeerCallResult, PeerCallMode
  events.go       // peer.handoff.* and peer.call.* event envelopes
  version.go      // PeeringContractsVersion (separate from ContractsVersion)
```

`PeeringContractsVersion` evolves **independently** of the main
`ContractsVersion` from `hero-cloud-split`. Bake into the package
design and surface in `hero check`.

### 7. CLI surface

**New:**

```
hero peer manifest [--out <path>]
    Generate this repo's peer-manifest.yaml. Wired into `hero index`.

hero peer list
    Show all configured peers (alias, peer_id, reachable y/n,
    manifest present y/n, peer-conventions count, in-flight handoffs,
    in-flight peer calls).

hero peer show <alias>
    Detail view: manifest contents, peer_id, the peer's own peer
    list (reciprocity check), last index time.

hero peer call <alias> --mode=<advisory|spec-out|full> "<prompt>"
               [--budget-turns N] [--budget-tokens N]
               [--related-spec <slug>] [--reason <text>]
    Spawn a subagent in <alias>'s workspace, run <prompt> in the
    chosen mode, return the structured result. Appends a trail
    entry to --related-spec if set.

hero handoff <slug> <peer-alias> [--title <new-title>] [--type <type>]
             [--reason <reason>]
    Async drop. Routes the local spec to <peer-alias>, updates
    statuses, writes both sides' trail entries.

hero handoff status [<slug>]
    Show handoff state for one spec or all in-flight handoffs.
    Walks the trail in either direction via CrossRepoResolver.

hero handoff accept <slug>
    Receive a handed_back spec — moves originator status from
    handed_back to delivering (or whatever phase the user picks).
```

**Extended:**

```
hero relevant ... [--peer <alias>] [--surface <tag>]
    Adds peer-surface conventions. Repeatable: --peer app --peer shared-libs.

hero queue
    Gains an "Incoming Handoffs" section listing specs received from
    peers (status=planning, received_from set), distinct from
    local-origin work.

hero status
    Surfaces handed-off specs whose peer-side status has reached
    `completed` (status: handed_back on this side).

hero context
hero resume
    Both gain the contract-import passive-surfacing one-liner when
    edited files import peer-owned contract symbols.

hero init
    Mints a peer_id UUID and writes it to hero.json.
```

**Already in place, no change:**

- `hero repos` — the registry. Consumed by new commands.
- `CrossRepoResolver` — upgraded to dual-key on UUID + alias.
- `relations` machinery — accepts new relation kinds without
  changes.

### Tradeoffs and alternatives considered

**Alternative: convention loading as the primary path.** Demoted to
fallback. Spec-out delivers the same benefit (B's rules govern the
work) with persistence and compounding for B's future sessions.

**Alternative: a daemon watching sibling `.hero/` dirs.** Rejected
for v1. Daemon brings lifecycle bugs; the same outcome is achieved
by writing the receiver's spec at handoff time and trusting the
peer's `hero queue` at session start.

**Alternative: handoffs via git (peer pulls a branch with the new
spec).** Rejected. The receiving spec is small; git is the wrong
granularity.

**Alternative: one shared `.hero/` across all sibling repos.**
Rejected (also rejected by `hero-cloud-split`). Muddies spec
ownership and breaks tracker integration.

**Alternative: aliases as canonical identity.** Rejected. Aliases
diverge across machines — UUID is the only stable join key.

### Solo-developer pilot gate (Phase 3+ blocker)

Before any work goes into Phase 3+, **dogfood the full handoff +
sync peer call flow with all three repos owned by the same
operator**. If `hero queue` + `/resume` at session-switch time
feels heavy, address ergonomics before adding more modes. This
isn't an open question — it's a Phase 3 entry gate.

## Acceptance Criteria

### Peer identity (Phase 0)

- WHEN `hero init` is run THE SYSTEM SHALL mint a new UUID and
  write it to `hero.json` as `peer_id`.
- WHEN any `hero` command is run in a workspace whose `hero.json`
  lacks `peer_id` THE SYSTEM SHALL mint a UUID, write it to
  `hero.json`, and emit a `workspace.peer_id_minted` event.
- THE SYSTEM SHALL treat `peer_id` as the canonical identifier for
  all peering operations (manifest, handoff records, trail entries,
  peer call envelopes) and SHALL never use the local alias as a
  join key across workspaces.
- WHEN `hero repos scan` discovers a sibling THE SYSTEM SHALL read
  the sibling's `hero.json:peer_id` and record both the alias and
  UUID in the local registry.
- WHEN displaying a peer reference to a human THE SYSTEM SHALL
  resolve `peer_id` to the local alias and show the alias, never
  the bare UUID, except in diagnostic output.

### Peer manifest (Phase 0–1)

- THE SYSTEM SHALL generate `<repo>/.hero/peer-manifest.yaml`
  listing conventions explicitly marked peer-surface (via
  convention frontmatter or `hero.json` glob), each with slug,
  title, surface tags, path, and content digest, plus the
  workspace's `peer_id`.
- WHEN `hero index` runs THE SYSTEM SHALL regenerate
  `peer-manifest.yaml` if the underlying conventions or peering
  config have changed.
- THE SYSTEM SHALL default to publishing *zero* conventions in the
  peer manifest unless they are explicitly marked peer-surface.
- THE SYSTEM SHALL include a `contracts:` section in the manifest
  listing Go symbols this repo owns, with their convention slug
  for cross-reference.

### Handoff lifecycle (Phase 1)

- WHEN `hero handoff <slug> <peer> [--reason ...]` is run THE
  SYSTEM SHALL update the local spec's status to `handed_off`,
  append a `## Handoff Trail` entry (direction: out, mode:
  async-drop, peer_id, at_commit, reason), and SHALL create a
  scaffolded spec at `<peer-path>/.hero/planning/<type>/<slug>/spec.md`
  with status `planning`, a `received_from` block, and a reciprocal
  trail entry (direction: in).
- WHEN a spec is in status `handed_off` and its peer-side
  counterpart's status becomes `delivering` THE SYSTEM SHALL move
  the local status to `awaiting_peer` and append a trail entry.
- WHEN a spec is in status `awaiting_peer` and its peer-side
  counterpart's status becomes `completed` THE SYSTEM SHALL move
  the local status to `handed_back`, append a trail entry with
  `result_ref` (commit SHA or PR URL if available), and surface
  the transition in `hero status`.
- WHEN `hero handoff accept <slug>` is run on a `handed_back` spec
  THE SYSTEM SHALL prompt the user for the next phase
  (`delivering` or `in-review`), update the status, and append a
  trail entry.
- WHEN `hero handoff` runs THE SYSTEM SHALL emit a
  `peer.handoff.sent` event in the originator's `events.log` and a
  `peer.handoff.received` event in the receiver's `events.log`.
- IF the peer-side scaffold slug collides with an existing peer
  spec THEN THE SYSTEM SHALL append a numeric suffix and report
  the chosen slug.
- IF `<peer>` is not configured in `hero repos` THEN THE SYSTEM
  SHALL fail with a clear error listing configured peer aliases.
- WHEN `hero queue` is run in a repo that has received handoffs
  THE SYSTEM SHALL display an "Incoming Handoffs" section listing
  specs with `received_from` set.
- WHILE a spec is in status `handed_off` or `awaiting_peer` THE
  SYSTEM SHALL exclude it from the active-delivering-specs
  invariant in `spec-status-integrity`.
- THE SYSTEM SHALL persist every cross-workspace event (handoff
  send/receive, peer call) as a trail entry on the relevant spec(s)
  and SHALL never rely on the events log alone for audit
  reconstruction.

### Sync peer call — advisory (Phase 2)

- WHEN `hero peer call <alias> --mode=advisory "<prompt>"` is run
  THE SYSTEM SHALL spawn a subagent session in `<alias>`'s
  workspace with full Hero context loaded, run the prompt, and
  return a structured result containing findings text, mode,
  peer_id, call_id, and budget consumed.
- THE SYSTEM SHALL NEVER write any spec, code, or non-event-log
  state on the peer side during an advisory call.
- WHEN `--related-spec <slug>` is provided THE SYSTEM SHALL append
  a trail entry to that spec recording the call (mode: advisory,
  peer_id, call_id, prompt summary, result summary).
- IF the peer is unreachable (directory missing, manifest absent)
  THEN THE SYSTEM SHALL fail with a clear error and SHALL NOT
  partially write any state.

### Sync peer call — spec-out (Phase 2)

- WHEN `hero peer call <alias> --mode=spec-out "<prompt>"` is run
  THE SYSTEM SHALL spawn a subagent in `<alias>`'s workspace that
  runs the peer's `/design` flow with the prompt, saves the
  resulting spec to `<peer-path>/.hero/planning/<type>/<slug>/spec.md`
  with status `planning` and a `received_from` block, and returns
  the peer spec slug plus a trail entry to A's session.
- WHEN `--related-spec <slug>` is provided on a spec-out call THE
  SYSTEM SHALL move that spec's status to `awaiting_peer` and
  append a trail entry (mode: spec-out, peer_spec: `<alias>/<slug>`).
- THE SYSTEM SHALL apply the same handoff lifecycle to spec-out
  results as to async handoffs: peer-side completion drives
  originator status from `awaiting_peer` to `handed_back`.

### Contract import surfacing (Phase 3)

- WHEN `/resume` or `hero context` runs THE SYSTEM SHALL scan
  recently changed files for imports of Go symbols listed in any
  configured peer's manifest `contracts:` section.
- IF any changed file imports a peer-owned contract symbol THEN
  THE SYSTEM SHALL surface a one-line signal naming the symbol,
  peer alias (resolved from peer_id), governing convention slug,
  and last-changed commit. THE SYSTEM SHALL NOT block, prompt, or
  auto-trigger any peer action.
- THE SYSTEM SHALL produce no signal when the only matches are
  non-contract files (e.g., test fixtures, generated code) — the
  match must be a real import of a contract symbol.

### Peer convention loading — fallback (Phase 2)

- WHEN `hero relevant --peer <alias>` is run THE SYSTEM SHALL
  resolve `<alias>` via the existing `hero repos` registry
  (peer_id-keyed), read `<peer-path>/.hero/peer-manifest.yaml`,
  and include the listed conventions in the relevant-context
  output.
- IF `<peer-path>/.hero/peer-manifest.yaml` is missing or
  unreadable THEN THE SYSTEM SHALL print a clear error pointing
  the user at `hero index` in the peer repo and exit non-zero.

### Cross-repo resolution (Phase 0)

- THE SYSTEM SHALL upgrade `CrossRepoResolver` to dual-key on
  peer_id (canonical) and alias (display), with peer_id as the
  join key for trail entries and `received_from` references.
- THE SYSTEM SHALL walk the `## Handoff Trail` of a given spec to
  resolve all peer-side counterparts, in either direction
  (originator → receivers, receiver → originator).

### Cloud (Phase 4, gated)

- WHERE cloud is configured (`cloud.org_id` set and credentials
  present) THE SYSTEM SHALL publish the peer manifest and handoff
  events through the cloud graph push API in addition to the
  local writes.
- THE SYSTEM SHALL route handoffs and peer calls by `peer_id`
  through the cloud when the target is not a local-disk sibling.

### Contracts package (Phase 0)

- THE SYSTEM SHALL place the `PeerManifest`, `HandoffRecord`,
  `PeerCallRequest`, `PeerCallResult`, `TrailEntry`, and peer event
  envelope Go types under `contracts/peering/` so both the CLI and
  a future cloud server consume identical wire shapes.
- THE SYSTEM SHALL maintain `PeeringContractsVersion` in
  `contracts/peering/version.go` and SHALL evolve it independently
  of the main `ContractsVersion` from `hero-cloud-split`.

### Safety

- THE SYSTEM SHALL never write secrets, credentials, or
  `outposts.enc` content into the peer manifest.
- THE SYSTEM SHALL fail spec-out and async-handoff writes
  atomically — receiver spec first, then originator status — and
  surface a recoverable error via `hero check` if the second write
  fails.

## Changes

### New files

1. `contracts/peering/manifest.go` — `PeerManifest`,
   `ConventionEntry`, `ContractEntry`.
2. `contracts/peering/handoff.go` — `HandoffRecord`,
   `ReceivedFrom`, `HandoffStatus`, `TrailEntry`,
   `TrailDirection`, `TrailMode`.
3. `contracts/peering/peercall.go` — `PeerCallRequest`,
   `PeerCallResult`, `PeerCallMode`, `BudgetSpec`,
   `BudgetConsumed`.
4. `contracts/peering/events.go` — event envelopes for
   `peer.handoff.sent`, `peer.handoff.received`,
   `peer.handoff.bounced`, `peer.call.invoked`,
   `peer.call.completed`.
5. `contracts/peering/version.go` — `PeeringContractsVersion`.
6. `internal/peering/identity.go` — `MintPeerID()`, migration helper
   for existing workspaces.
7. `internal/peering/manifest.go` — generate the peer manifest from
   local conventions + `hero.json` peering config. Called from
   `hero index`.
8. `internal/peering/handoff.go` — `Handoff(originSlug, peerAlias,
   reason)`; performs two-side write, emits events, appends trail
   entries.
9. `internal/peering/peercall.go` — `Call(peerAlias, mode, prompt,
   budget, relatedSpec)`; spawns the peer-side subagent, handles
   the three modes, returns the structured result.
10. `internal/peering/resolve.go` — peer manifest reader, trail
    walker, peer_id ↔ alias resolver. Wraps `os.ReadFile` + YAML.
11. `internal/peering/contract_imports.go` — Go import scanner that
    detects edited files importing peer-owned contract symbols.
12. `internal/peering/trail.go` — read and write `## Handoff Trail`
    section in a spec markdown file.
13. `internal/cli/peer.go` — `hero peer manifest|list|show|call`
    subcommand tree.
14. `internal/cli/handoff.go` — `hero handoff` and `hero handoff
    status|accept` subcommand tree.

### Modified files

15. `internal/cli/init.go` — mint `peer_id` UUID on `hero init` and
    write to `hero.json`. Migration: mint on first invocation when
    missing.
16. `internal/config/config.go` — add `PeerID` field to the loaded
    config struct; add optional `Peering` config block
    (publish-conventions globs, display name); update `hero.json`
    write path.
17. `internal/cli/repos.go` — record peer_id on scan/add; expose
    peer_id in `hero repos` output.
18. `internal/spec/spec.go` — add `HandedOff`, `AwaitingPeer`,
    `HandedBack` to the status enum; add `Handoff`,
    `ReceivedFrom`, `HandoffTrail` frontmatter fields and parsing;
    extend YAML loader.
19. `internal/spec/resolve.go` — dual-key `CrossRepoResolver` on
    peer_id + alias; register `handed-off-to` / `handed-off-from`
    relation kinds; add trail-walk methods.
20. `internal/cli/relevant.go` — add `--peer` and `--surface`
    flags; consult peer manifests when present.
21. `internal/cli/queue.go` — add "Incoming Handoffs" section.
22. `internal/cli/status.go` — surface peer-side completion of
    handed-off specs.
23. `internal/cli/context.go` and `internal/cli/resume.go` (or
    equivalent) — add contract-import passive surfacing
    one-liner.
24. `internal/index/index.go` — call into `internal/peering` during
    indexing to regenerate the peer manifest.
25. `internal/events/log.go` — register new `peer.handoff.*` and
    `peer.call.*` event kinds.
26. `internal/integrity/...` (per `spec-status-integrity` spec) —
    treat `handed_off` and `awaiting_peer` as in-flight-elsewhere,
    not contenders for active-delivering claims.
27. `commands/handoff.md` (new slash command),
    `commands/peer-call.md` (new), update `commands/diagnose.md`,
    `commands/design.md`, `commands/deliver.md` with handoff +
    peer-call routing notes.
28. `agents/feature-delivery-lead.md` — add "Peer handoff routing"
    and "Sync peer call" sections.
29. `skills/convention-writing.md` — add a "peer-surface"
    subsection.
30. `.hero/knowledge/conventions/peering-protocol.md` (new) —
    documents the three-tier ladder, manifest, handoff lifecycle,
    trail format, local-vs-cloud transport.

### Cloud-side (additive, Phase 4)

31. `cloud/api/handlers/peering.go` — server-mediated handoff and
    peer-call receive paths for the multi-machine case.
32. `cloud/store/peering.go` — persistence for handoff records and
    call records when cloud is the transport.

## Boundaries

- **Not redesigning `hero repos`.** Consumes the existing registry.
- **Not solving peer-to-peer conflict resolution** between divergent
  convention edits. Covered by `graph-conflict-detection` and
  future knowledge-federation work.
- **Not building cross-machine local peering.** SSH-into-peer or
  push-over-LAN is out. Different machine → cloud.
- **Not building auto-trigger boundary modes** (v3). Phase 3 ships
  passive surfacing only.
- **Not building full-delivery sync peer call** in v1. v2 with
  approval prompts + budgets.
- **Not changing tracker integration.** Each repo's hero workspace
  points at its own tracker config. A handoff to a peer with a
  different tracker produces a peer-side spec without
  `tracker_id` — the peer's user creates the tracker issue if they
  want one.
- **Not designing the cloud server side.** That's
  `graph-memory-federation` and a follow-on cloud spec. v1
  specifies the wire shapes (in `contracts/peering/`) and the
  client behavior.
- **Not solving Community Edition / licensing** for any of this.
  v1 primitives are local-only.

## Risks

- **Identity stability.** A workspace's `peer_id` must be stable
  across renames, moves, and re-clones. Mitigation: written to
  `hero.json` (under source control); minting is one-time and
  surfaced in the events log. If a developer clones the same
  workspace twice and edits both, peer_id will collide — surfaced
  by `hero check` as a duplicate-identity warning.
- **Manifest staleness.** If `peer-manifest.yaml` isn't
  regenerated, peer-conventions loaded into a session are wrong.
  Mitigation: tie generation to `hero index`; add a "last
  regenerated" timestamp; warn on stale.
- **Provenance forgery.** Anyone with write access to a peer's
  `.hero/` could create a spec with a fake `received_from`. v1
  trusts local filesystem; cloud mode can sign events (reuse
  graph-memory-federation's GitHub App identity). Documented limit.
- **Two-write atomicity.** Receiver-first write order means: if
  originator-side update fails, a peer spec exists with
  `received_from` pointing at an originator that still shows the
  pre-handoff status. Surfaced by `hero check` as a recoverable
  inconsistency; re-running `hero handoff` is idempotent.
- **Peer write permission.** `hero handoff` and spec-out calls
  write into the peer's `.hero/`. Pre-check writability in `hero
  peer list`; fail with a clear "peer path not writable" error.
- **Subagent isolation.** Sync peer calls spawn subagents that
  could in principle do unbounded work in advisory mode. Mitigation:
  hard turn/token budgets (`--budget-turns`, `--budget-tokens`)
  default-limited to small values; advisory mode also enforces
  no-write at the filesystem boundary, not just by prompt.
- **Convention quality drift on peer side.** Peer conventions
  loaded into a session may be vague or outdated. Mitigation: the
  convention-writing skill quality bar; `hero check` flags
  peer-surface conventions that fail the bar.
- **Slug collisions on receive.** Suffix-append loses semantic
  naming. Mitigation: user can rename the peer spec post-creation;
  `--title` can derive a hand-picked slug.
- **Solo-developer ergonomics.** All three repos one operator —
  context-switching cost. Mitigation: the Phase 3+ pilot gate
  documented above. If `hero queue` + `/resume` is heavy, fix
  ergonomics before adding more modes.
- **Composition with `hero-cloud-split`.** `PeeringContractsVersion`
  evolves separately from main `ContractsVersion` to avoid
  double-bumps. Documented in the package.
- **Status sprawl.** Three new statuses (`handed_off`,
  `awaiting_peer`, `handed_back`) is a real bloat. Mitigation:
  each maps to a distinct user-visible state (gave it / waiting /
  ball's back); the trail keeps the narrative. Coordinate with
  `spec-status-integrity` so invariants stay enforceable.

## Validation

- **Unit:** golden tests for manifest generation against a fixture
  repo with marked + unmarked conventions; tests for `Handoff()`
  covering happy path, slug collision, missing peer, unwritable
  peer; round-trip parse of `Handoff`, `ReceivedFrom`, and trail
  entries; peer_id mint idempotence; `Call()` mode dispatch.
- **Integration:** a fixture with two sibling `.hero/` workspaces;
  run `hero handoff` end-to-end and verify both filesystems'
  content + emitted events + trail entries on both sides; run
  `hero peer call --mode=advisory` and `--mode=spec-out` and
  verify the spec-out call creates the peer spec with correct
  provenance.
- **Smoke:** three sibling fixtures (`app`, `client`, `desktop`)
  with mock conventions; exercise `hero peer manifest`, `hero
  relevant --peer`, `hero peer call` (both modes), `hero handoff`,
  `hero queue` showing incoming handoffs, status return signal
  after staged peer-side completion, contract-import surfacing on
  edited files.
- **Drift:** `hero drift cross-repo-peering` must show expected
  file matches in `## Changes` before delivery is declared
  complete.
- **Manual verification on the user's actual three-repo setup**
  (Phase 3 pilot gate): starting in `client`, run an advisory peer
  call, a spec-out call, and an async handoff against `app`.
  Switch to `app`, observe incoming queue entries and the spec
  with received_from. Complete one peer spec. Switch back to
  `client`, see status flip to `handed_back`. Assess whether the
  switch cost feels like a tag-team or like ceremony.

## Resolved Decisions

These were open at draft-time and are now locked. Captured in
`.hero/knowledge/decisions/`:

- **Spec handoff as primary cross-repo path; convention import is
  fallback.** (`spec-handoff-as-primary-over-convention-import.md`)
- **Cross-repo peering is local-first; cloud augments.**
  (`cross-repo-peering-local-first.md`)
- **Peer manifest uses an explicit publish boundary.**
  (`peer-manifest-publish-boundary.md`)
- **Peer identity is a stable UUID minted at `hero init`.** (new:
  `peer-uuid-identity.md`)
- **Sync peer call is a first-class third mode with three tiers
  (advisory, spec-out, full).** (new:
  `sync-peer-call-three-tier-ladder.md`)
- **Contract metadata surfaces, does not enforce, in v1.** (new:
  `contract-metadata-surface-dont-enforce.md`)
- **`PeeringContractsVersion` evolves independently of main
  `ContractsVersion`.** (encoded in `contracts/peering/version.go`.)
- **All three handoff statuses ship in v1** (`handed_off`,
  `awaiting_peer`, `handed_back`) because the spec carries the
  handoff trail and each status maps to a distinct user-visible
  state.
- **Spec-out resolves the convention-loading fork.** Convention
  import remains for the genuine "work stays in A" case.

## Open Questions

Small enough to discover during delivery, not blocking.

- **Subagent invocation API for sync peer call.** How exactly does
  A spawn a subagent in B's workspace? Likely path: shell-out
  `hero peer-call-server <prompt> --mode=...` in B's cwd, parse
  structured stdout. Lock during Phase 2.
- **Budget defaults for advisory and spec-out.** Pick reasonable
  values during Phase 2 implementation. Suggest 20 turns / 50k
  tokens for advisory; 50 turns / 150k tokens for spec-out.
- **Trail entry format: YAML block list vs. structured prose.**
  Spec drafts a YAML-ish list; final format chosen during
  implementation. Constraint: must round-trip parse cleanly.
- **`hero handoff accept` UX for handed_back specs.** Whether to
  default the next status to `delivering` or prompt. Likely
  prompt; lock at Phase 1 polish.
- **Contract import scanner scope.** Whole repo or just changed
  files? v1 scans changed files (cheap, signal-rich); revisit if
  we miss imports living in untouched files.

## Phasing

### Phase 0 — Identity + contracts package + resolver upgrade

Mint `peer_id` UUIDs at `hero init`, write to `hero.json`, migrate
existing workspaces. Stand up `contracts/peering/` with
`PeerManifest`, `HandoffRecord`, `PeerCallRequest`, trail types, and
`PeeringContractsVersion`. Upgrade `CrossRepoResolver` to dual-key
on peer_id + alias. Wire `hero repos scan` to capture peer_id.

Exit: every workspace has a stable UUID; `CrossRepoResolver` resolves
by UUID with alias display; `contracts/peering/` compiles and is
consumed by both code paths.

### Phase 1 — Handoff lifecycle + trail + async drop

Add `handed_off`, `awaiting_peer`, `handed_back` statuses to the
spec model. Implement `hero handoff`, `hero handoff status`, `hero
handoff accept`. Emit `peer.handoff.*` events. Add `## Handoff
Trail` section read/write. Add `hero queue` incoming-handoffs
section. Coordinate with `spec-status-integrity` so the new
statuses are excluded from active-delivering claims. Add peer
manifest generation tied to `hero index`.

Exit: the user can hand a synthetic spec across the three local
repos and see it on the receiver's queue with full provenance and a
walkable trail in both directions; status return signal fires when
the peer's spec is marked completed.

### Phase 2 — Sync peer call: advisory + spec-out

Implement `hero peer call --mode=advisory` (no-write subagent in
peer cwd, structured result return). Implement `hero peer call
--mode=spec-out` (subagent runs peer's design flow, writes spec on
peer side, returns slug, integrates with handoff lifecycle).
Implement budget enforcement. Add the convention-loading fallback:
`hero relevant --peer <alias>`.

Exit: from a session in `client`, the user can ask `app`'s Hero a
question (advisory) and get a structured answer; can ask `app`'s
Hero to design a spec (spec-out) and have it appear in `app`'s
planning queue tied to the calling spec via the handoff trail.

### Phase 3 — Contract import passive surfacing + pilot gate

Implement the Go import scanner. Wire passive surfacing into
`/resume` and `hero context`. **Pilot gate:** dogfood the full
ladder on the user's three real repos. If ergonomics are heavy,
fix before Phase 4.

Exit: a session editing a file that imports a peer-owned contract
symbol sees the one-line signal. Dogfood report confirms the flow
is tag-team-shaped, not ceremony.

### Phase 4 (gated) — Cloud transport + boundary nudge

Mirror handoff and peer call events through the cloud graph push
API for peers not on local disk. Add `hero nudge` boundary
suggestion when contract-shape edits are detected (the v2 step in
the three-step contract-metadata ladder).

Exit: a peer in the same cloud org but on a different machine
receives a handoff via cloud and round-trips; boundary nudges
appear on structural contract edits.

### Deferred (not in this spec)

- **Auto-trigger boundary modes** (contract-shape edits auto-open
  an advisory or spec-out call). Defer until Phase 4 nudge UX
  proves not-friction.
- **Full-delivery sync peer call** with approval + budget.
- **Machine-to-machine peering** for different operators or
  different machines without cloud.
