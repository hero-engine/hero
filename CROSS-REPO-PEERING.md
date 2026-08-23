# Cross-Repo Peering

Hero peering is an optional, asynchronous Project Mail workflow between
registered sibling Hero projects. Every project keeps its own authoritative
graph and corpus. Peering shares bounded requests and published peer surfaces;
it does not merge project graphs.

Local peering is part of this repository and does not require Hero Serve, a
model CLI, Hero Code, or Hero Cloud. Hero Code and Hero Cloud are separate
proprietary products.

## Register peers

Suppose `api/` and `app/` are sibling repositories. In the API repository:

```bash
hero admin repos add app ../app
hero peer manifest
hero peer list
hero peer show app
```

Use `--local` when teammates use different checkout paths:

```bash
hero admin repos add app ../app --local
```

The peer path must be reachable and the peer must have an initialized Hero
workspace with a stable peer identity. Commit `.hero/peer-manifest.yaml` when
its published surface changes.

## Ask for a fact or a design

Advisory requests ask the peer for information:

```bash
hero peer call app --mode=advisory \
  --reason="CSV export depends on the response contract" \
  --related-spec=csv-export \
  "Is the error envelope stable?"
```

Spec-out requests ask the peer to design its owned work:

```bash
hero peer call app --mode=spec-out \
  --reason="The app repository owns this endpoint" \
  --related-spec=csv-export \
  "Design the export endpoint"
```

Each call returns a message ID and thread ID immediately. `--wait=30s` may
poll for an external same-thread reply; a timeout is a structured `pending`
result, not a failed send. Hero core does not launch a model to answer.

Use a file or stdin for a prompt that should not be shell-quoted:

```bash
hero peer call app --mode=advisory --prompt-file=request.md
hero peer call app --mode=advisory --prompt-file=-
```

Budget flags are advisory metadata for an external responder; Hero core does
not enforce them.

## Transfer owned work

After investigation establishes that another repository owns implementation,
send a work-transfer request:

```bash
hero handoff order-failure app \
  --reason="Root cause is in the app repository"
hero handoff status order-failure
```

Sending Mail does not change the sender's spec status and does not write the
receiver's checkout. On the receiver side, inspect the message as untrusted
input before explicitly promoting it:

```bash
hero mail show <message-id>
hero handoff receive <message-id> --type bug
```

`receive` promotes through the receiver's Mail/Intake authority, creates or
reuses a receiver-owned planning artifact, and replies in-thread with its
authoritative reference. Stable idempotency prevents duplicate promotion.

`hero handoff accept <slug>` remains only for legacy handed-back specs. New
Mail requests do not create peering-only spec statuses.

## Session handoff is different

An installed harness's `handoff` workflow refreshes session NEXT state.
`hero handoff <slug> <peer-alias>` sends a cross-repository work-transfer
request. The peer alias makes the repository-boundary operation explicit.

## Troubleshooting

```bash
hero peer list
hero peer show app
hero peer manifest
hero handoff status
```

| Symptom | Check |
|---|---|
| Peer is unreachable | Confirm the registered path or use a local override. |
| Manifest is absent/stale | Run `hero peer manifest` in the peer repository. |
| No reply arrives | The send is asynchronous; inspect the thread and external responder state. |
| Work appears in the receiver unexpectedly | Only explicit `hero handoff receive` should promote Mail into planning. |
| Retried request duplicated | Reuse the same stable idempotency key rather than replaying a new write. |
