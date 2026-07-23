# Cross-Repo Peering

Hero peering is asynchronous Project Mail between configured sibling projects.
It works locally without Hero Serve, a model CLI, or a daemon.

```bash
hero admin repos add app ../app
hero peer list
hero peer call app --mode=advisory "What is the current API envelope?"
hero peer call app --mode=spec-out --related-spec=my-client-change \
  --reason="the API project owns the contract" "Please design the API change"
hero handoff my-investigation app --reason="implementation belongs to app"
```

Each send returns a message ID and thread ID immediately. `peer call --wait=30s`
may poll for an external same-thread reply; timeout is a successful structured
`pending` result. `--prompt-file=<path>` or `--prompt-file=-` reads larger
prompts without shell quoting. Budget flags remain accepted as advisory
metadata, but Hero does not enforce them because it is not executing a model.

## Receiver-owned promotion

Calls and handoffs only deliver Mail. They do not write the receiver checkout
or change the sender's spec status. On the receiving side:

```bash
hero mail inbox
hero mail show <message-id>
hero handoff receive <message-id> --type feature
```

`receive` promotes through the Mail/Intake authority, creates or reuses one
receiver-owned planning artifact, and replies in-thread with the authoritative
reference. Retrying with the same identity is idempotent. A receiver may dismiss
the message instead, creating no work.

External harnesses or future workers may optionally inspect Mail and reply.
That response handling is outside Hero core and must carry its own authority and
confirmation policy.

## History and migration

`hero handoff status` retains the historical view of Handoff Trails,
`received_from`, and legacy `handed_off`, `awaiting_peer`, and `handed_back`
specs. `hero handoff accept <slug>` remains for those legacy handed-back
carriers. New Mail requests never create peering-only spec statuses.

Existing `peering.subagent` configuration still loads, emits a deprecation
warning, and is ignored. Hero core never executes the configured command.
