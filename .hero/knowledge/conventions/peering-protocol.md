---
title: Cross-Repo Peering Protocol
type: convention
status: active
created: 2026-05-15
scope:
  - "contracts/peering/**"
  - "internal/peering/**"
  - "internal/cli/handoff.go"
  - "internal/cli/peer.go"
tags: [peering, cross-repo, conventions, handoff, project-mail, local-first]
---

# Cross-Repo Peering Protocol

## Rule

Peering uses durable Project Mail. A sender may address a typed request to a
configured peer, but it may not launch a model in that workspace, write its
checkout, or place work on its roadmap. Receiver promotion is explicit.

| Request | Mail kind | Effect at send time |
|---|---|---|
| Advisory | `peer.advisory` | One durable question |
| Spec-out | `peer.spec_out` | One durable design request |
| Work transfer | `peer.work_transfer` | One durable transfer request |

`hero peer call` is async by default and returns message/thread identity.
`--wait` polls for the first valid same-thread response and returns `pending` at
timeout. Polling never starts a watcher. Prompt files/stdin are supported.
Budget flags are non-enforced hints for an optional external responder.

`hero handoff <slug> <alias>` sends a bounded summary and stable provenance. It
does not change the origin spec status or write the receiver tree. The receiver
reviews the request and runs `hero handoff receive <message-id>` to promote
through Project Mail and Intake. Promotion creates or reuses one receiver-owned
artifact and replies with its authoritative reference. Replay is idempotent.
Dismissal creates no work.

## Identity and provenance

`peer_id` is the stable join key; aliases are local display names. Mail carries
origin peer/spec/commit, reason, mode, manifest version, idempotency identity,
and typed kind. Optional `transport`, `message_id`, `thread_id`, and
`result_ref` fields are additive on historical peering contracts and trails.

## Compatibility

Continue reading old peer-call artifacts, `received_from`, Handoff Trail blocks,
and the statuses `handed_off`, `awaiting_peer`, and `handed_back`. Reconciliation
may finish existing carriers but new Mail flows never create them.
`hero handoff accept <slug>` remains for legacy handed-back specs.

Old `peering.subagent` configuration must decode, produce a deprecation warning,
and remain ignored. Automatic responders belong to an opt-in harness or worker
with its own authority policy; Hero core is transport and provenance only.

## Install requirement

Author guidance once under the engineering domain and verify it renders for all
six targets: `opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic`.
Every installed instruction must say async Mail, explicit receiver promotion,
and optional external response handling. No installed surface may promise
core-spawned execution or receiver-side writes.
