# Cross-Repo Peering

Most real systems span more than one repository — a backend, a web client, a
mobile app, a shared design system. Each one can run its own Hero workspace,
and **peering** lets those sibling workspaces work as a tag team instead of in
isolation. A session in one repo can ask another repo's Hero a question, hand a
spec across the boundary, or notice when it's touching code another repo owns —
with provenance travelling along on every operation.

This page explains the *model*. For the exact commands, see the
[Cross-Repo Peering CLI reference](../cli/peering.md).

## The mental model: sibling workspaces

Peering is **not** a monorepo and **not** a shared database. Each repo keeps its
own graph, its own specs, and its own conventions. What peering adds is a thin,
explicit protocol for one workspace to reach another:

- Every workspace mints a stable **`peer_id`** (a UUID) at `hero init`.
- You **register** siblings once per machine (`hero admin repos add <alias> <path>`).
- Each workspace publishes a committed **peer manifest** (`.hero/peer-manifest.yaml`)
  describing what it exposes — contract symbols, conventions, ownership.

Because registration and the manifest are explicit, a peer only ever sees what
the other workspace chose to surface. Nothing is implicitly shared.

## Three ways to interact

When work touches a boundary, you pick the interaction that matches how much the
peer needs to do:

| Interaction | What happens | Reach for it when |
|---|---|---|
| **Advisory call** | Peer's Hero answers a question; writes nothing on the peer | You need a fact — their error envelope, a schema, who owns an endpoint, "does this change break you?" |
| **Spec-out call** | Peer's Hero designs a spec natively on its side | The work is really theirs — let their workspace own the design. |
| **Async handoff** | You drop an already-investigated spec on the peer's queue | You did the root-cause work and the fix belongs in their repo. |

Advisory and spec-out are *synchronous* peer calls — a subagent runs inside the
peer workspace and returns to your session. A handoff is *asynchronous* — the
receiving workspace picks it up later, with the spec stamped `received_from` so
its origin is never lost.

## Provenance travels

The reason peering is trustworthy is that every crossing is recorded. A handed-off
spec carries a `received_from` stamp; the sender's spec moves to `handed_off` and
gains a Handoff Trail entry. Peer-call findings are persisted with the trail of
which repo answered and why. You can always trace a decision back across the
repo boundary — the same [`hero why`](../cli/search-and-context.md) traversal that
works inside one repo.

## Passive boundary detection

Peering also works *without* you asking. As you edit, Hero can scan changed files
for imports of contract symbols listed in a peer's manifest and print a one-line
signal — which symbol, which peer owns it, the governing convention, and the last
commit on the peer side. It's passive by design: it never blocks a commit,
prompts, or auto-fires a peer call. It just makes sure you *know* when you've
wandered into territory another repo governs.

## Where to go next

- [Cross-Repo Peering CLI reference](../cli/peering.md) — every command, flag, and mode.
- The `/peer` slash command is the session-level front door that picks the right
  mode (advisory / spec-out / handoff / list / show) for you.
