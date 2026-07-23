---
name: cross-repo-peering
description: Route cross-repo questions and work-transfer requests through asynchronous Project Mail with receiver-owned promotion.
---

# Cross-repo peering

Peering is durable Project Mail, not model execution. Hero core sends typed
requests and returns a message/thread ID. It never launches a harness, writes
the receiver checkout, or changes a sender spec status.

## Pick the request kind

| Need | Command | Receiver effect |
|---|---|---|
| Ask a focused question | `hero peer call <alias> --mode=advisory "..."` | Mail only |
| Ask the peer to design work | `hero peer call <alias> --mode=spec-out "..."` | Mail only until receiver acts |
| Offer investigated work | `hero handoff <slug> <alias> --reason "..."` | Mail only until receiver acts |
| Inspect configured peers | `hero peer list` / `hero peer show <alias>` | Read-only |

Calls are asynchronous by default. Use `--wait <duration>` only when an
external responder may already be active; timeout returns structured `pending`.
`--prompt-file <path>` and `--prompt-file -` avoid shell quoting problems.
Budget flags are advisory metadata because Hero is not the runtime.

Compose a concrete prompt: name the exact question, include
`--related-spec <slug>` when applicable, and explain the reason. Do not ask the
peer for facts available locally.

## Receiver authority

An incoming `peer.work_transfer` creates no roadmap work by itself. The receiver
must explicitly run:

```text
hero handoff receive <message-id> [--type feature|bug]
```

This promotes through Project Mail and Intake, creates or reuses one
receiver-owned artifact, and replies in-thread with its reference. Repeating
the command with the same identity is idempotent. Dismissal creates no work.
Automatic response handling, if desired, belongs to an opt-in harness or worker
with its own authority policy.

## Compatibility

`hero handoff status` continues to show historical trails and legacy
`handed_off`, `awaiting_peer`, `handed_back`, and `received_from` carriers.
`hero handoff accept <slug>` remains the legacy handed-back acceptance command.
New Mail requests do not create those statuses.

Old `peering.subagent` configuration is tolerated, warned about once, and
ignored. Never promise core-spawned execution or receiver-side writes.
