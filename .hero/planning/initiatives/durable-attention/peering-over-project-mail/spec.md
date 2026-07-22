---
title: "Peering over Project Mail — Async First, Execution Optional"
slug: peering-over-project-mail
type: feature
status: planning
domain: engineering
priority: medium
size: large
horizon: next
created: 2026-07-20
parent: durable-attention
depends-on: [project-mail-triage-and-provenance]
tags: [peering, mail, handoff, compatibility]
---

# Peering over Project Mail — Async First, Execution Optional

## Context

Hero peering currently bundles three concepts: project communication, accepted
work transfer, and model execution. `internal/peering.Call` spawns a configured
CLI (default `claude`), parses a result fence, and changes spec status for
spec-out. `internal/peering.Handoff` writes a receiver spec before the receiver
accepts it and moves the origin spec to `handed_off`.

Project Mail provides the missing light primitive. Peering can retain its useful
identity, manifest, routing, provenance, and user-facing vocabulary while
making communication asynchronous and receiver-authorized.

## Goal

Route peer calls and handoff requests through durable Project Mail, remove model
spawning from the core path, and make receiver promotion the only operation that
creates work in another project.

## Kickoff

Refactor `hero peer call` and new handoffs into adapters over Project Mail.
Preserve peer resolution, manifests, and legacy trail/status readers, but stop
writing receiver specs or launching `claude`. Make calls async by default with
optional reply polling, keep old history visible, update installed peering
guidance, and supersede `peer-call-multi-cli` only after validation passes.

## Problem

The current synchronous call fails when a particular harness is missing,
unauthenticated, or incompatible with raw-terminal behavior. The current
handoff grants the sender write authority over the receiver's roadmap. Both are
heavier than the basic need: leave a durable, contextual request for another
project and let its owner or agent handle it later.

## Design

### Peer call becomes Mail request/reply

Keep the entry point:

```text
hero peer call <alias> --mode advisory|spec-out [--related-spec ...]
```

It now sends a Project Mail envelope whose kind is `peer.advisory` or
`peer.spec_out`, with the composed prompt, reason, mode, budgets as advisory
metadata, origin project, related spec provenance, origin commit, and peer
manifest version. The result is a queued message/thread ID. It never changes a
spec status.

Default behavior returns immediately. `--wait <duration>` polls the sender's
mailbox for a response in the same thread and exits successfully when one
arrives or with a structured `pending` result at timeout. Polling does not start
a watcher or launch a harness. A receiver-side user, open harness session, or
future external automation may inspect Mail and reply through normal commands.
`--prompt-file`/stdin is added; the positional prompt remains accepted for CLI
compatibility.

The legacy `--dry-run` renders the normalized Mail envelope. Budget flags remain
accepted and serialized as non-enforced hints so scripts do not fail, but Hero
does not enforce model turns/tokens when it is not the runtime.

### Handoff becomes work-transfer request

Keep `hero handoff <slug> <peer-alias>`, but send `peer.work_transfer` Mail with
the origin spec's stable identity, title/type, reason, source project/commit,
and a bounded textual design summary. It does not copy the spec file, write the
receiver repository, or change the origin spec status.

The receiver accepts through `hero handoff receive <message-id> [--type ...]`,
an ergonomic adapter over Mail promotion. It creates/reuses Intake and the
receiver-owned planning spec through the triage service, then replies in-thread
with the authoritative artifact/navigation reference. Repeating receive is
idempotent. Rejecting/dismissing sends an optional response but creates no work.

`hero handoff status [slug|message-id]` combines new Mail request state with the
existing legacy status/trail view. New requests use Mail state labels
`queued | read | accepted | dismissed | responded`; they do not write
`handed_off`, `awaiting_peer`, or `handed_back` into spec frontmatter.

### Runtime boundary

Hero core no longer selects or executes a model CLI for peer communication.
Remove `exec.Command`, auth-signature detection, result-fence parsing, and
`peering.subagent` from the active call path. Existing configuration keys decode
without failure and produce a one-time deprecation warning when present; they
are ignored for new Mail calls. Automatic responders belong to a harness or
future opt-in worker consuming Mail, with its own authority and confirmation
policy.

### Compatibility and history

- Continue parsing `contracts/peering` call/handoff records, `received_from`,
  peer-call artifacts, and Handoff Trail blocks.
- Keep reconciliation for already-existing `awaiting_peer`, `handed_off`, and
  `handed_back` specs until those carriers complete; do not create new carriers.
- Preserve `hero handoff accept <slug>` for legacy handed-back specs. The new
  receiver command is `hero handoff receive <message-id>` to avoid ambiguity.
- Extend trail entries with optional Mail message/thread IDs and transport
  `project_mail`; additive fields do not invalidate historical entries.
- Update commands, skills, AGENTS generation, and docs together so no installed
  guidance promises synchronous model execution or receiver-side writes.

After implementation and compatibility tests, run
`hero supersede peer-call-multi-cli --by peering-over-project-mail`. Do not edit
the old spec's frontmatter by hand.

## Changes

1. Replace the active implementation in `internal/peering/peercall.go` with a
   Mail request/reply adapter; move any legacy parsing needed solely for history
   to `internal/peering/legacy.go`.
2. Refactor `internal/peering/handoff.go` to compose work-transfer Mail and add
   receiver promotion/reply in `internal/peering/receive.go`.
3. Update `contracts/peering/peercall.go`, `handoff.go`, and `trail.go` only with
   additive Mail transport/message/thread/result-reference fields; use
   `contracts/attention` for the envelope itself.
4. Update `internal/cli/peer.go` for async output, `--wait`, stdin/prompt files,
   JSON results, and deprecated budget hint behavior.
5. Update `internal/cli/handoff.go` for request delivery, `receive`, combined
   status, and preservation of legacy `accept`.
6. Keep legacy reconciliation/read support in `internal/peering/resolve.go` and
   status views, but prevent new Mail flows from writing peering-only spec
   statuses.
7. Remove active subprocess/result-fence/auth-detection tests and replace them
   with Mail delivery, polling, timeout, receive, reply, idempotency, and
   no-receiver-tree-write tests.
8. Update `domains/engineering/skills/cross-repo-peering/SKILL.md`,
   `domains/engineering/commands/peer.md`, generated AGENTS content in
   `internal/install/agents_md.go`, CLI help, and peering documentation.
9. Add a config deprecation test proving existing `peering.subagent` keys still
   load but cannot cause execution.
10. At the final delivery gate, invoke the supported `hero supersede` command
    for `peer-call-multi-cli` and include its projected files in the commit.

## Acceptance Criteria

- **AC-1:** WHEN `hero peer call` is invoked in advisory or spec-out mode, THE SYSTEM SHALL deliver one typed Project Mail request and return its message/thread identity without launching a model.
- **AC-2:** WHEN `--wait` is used, THE SYSTEM SHALL return the first valid same-thread response or a structured pending timeout without creating a watcher or mutating specs.
- **AC-3:** WHEN `hero handoff` is invoked, THE SYSTEM SHALL send one work-transfer request and SHALL NOT write the receiver repository or change the origin spec status.
- **AC-4:** WHEN the receiver explicitly runs `hero handoff receive`, THE SYSTEM SHALL promote through the Mail/Intake authority, create or reuse one receiver-owned spec, and reply with its authoritative reference.
- **AC-5:** WHEN peer call or receive is retried with the same idempotency identity, THE SYSTEM SHALL return the original Mail/artifact result without duplicates.
- **AC-6:** WHEN legacy handed-off, awaiting-peer, handed-back, received-from, trail, or peer-call artifacts are inspected, THE SYSTEM SHALL preserve their existing status and provenance interpretation.
- **AC-7:** WHEN `hero handoff accept <slug>` is used for a legacy handed-back spec, THE SYSTEM SHALL preserve the existing acceptance behavior.
- **AC-8:** WHEN old `peering.subagent` configuration is loaded, THE SYSTEM SHALL tolerate it, warn that it is ignored, and SHALL NOT execute the configured command.
- **AC-9:** WHILE no model CLI or Hero Serve process is installed or running, THE SYSTEM SHALL still send, receive, inspect, and reply to local peer Mail successfully.
- **AC-10:** WHEN Hero installs peering guidance for any supported harness, THE SYSTEM SHALL describe async Mail, explicit receiver promotion, and optional external response handling without promising core-spawned execution.
- **AC-11:** WHEN delivery validation passes, THE SYSTEM SHALL supersede `peer-call-multi-cli` through `hero supersede` and retain its historical graph/provenance.

## Boundaries

- No always-on responder, automatic model trigger, cloud transport, full peer
  delivery runtime, or implementation of every harness CLI profile.
- No changes to session-level `/handoff`, NEXT.md, or task continuation.
- No deletion or rewriting of legacy specs/trails/statuses.
- No automatic promotion of advisory/spec-out/work-transfer requests.

## Risks

- Changing default peer-call semantics is user-visible; CLI help, JSON status,
  deprecation notes, and release notes must be explicit.
- Old automation may expect a synchronous result. `--wait` preserves waiting
  shape but intentionally cannot guarantee a responder.
- Legacy reconciliation must remain until old carriers drain; tests must separate
  reading old state from creating new state.
- Work-transfer summary truncation can omit context; provenance points back to
  the origin spec/project/commit, and receiver promotion remains explicit.

## Validation

- `go test ./internal/peering/... ./contracts/peering/... ./internal/cli/...`
- End-to-end two-project tests for advisory, spec-out, work transfer, receive,
  dismiss, reply, timeout, and exact replay with no model CLI on `PATH`.
- Git-status assertions proving send/call/handoff do not mutate the receiver
  checkout and receive is the first receiver work-tree mutation.
- Legacy fixture tests for every prior status, trail, artifact, and
  `received_from` form.
- Installer golden tests for all six targets and repository search proving
  active peering code no longer invokes `os/exec` or parses result fences.
- `hero supersede peer-call-multi-cli --by peering-over-project-mail`, followed
  by `hero check`, `hero index --if-stale -q`, and `go test ./...`.
