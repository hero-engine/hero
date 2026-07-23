---
description: Send asynchronous cross-repo questions and work-transfer requests over Project Mail.
---

# /peer

Load the `cross-repo-peering` skill, then:

1. Run `hero peer list` before targeting an alias.
2. Choose `advisory`, `spec-out`, or work transfer from the user's intent.
3. Compose a focused prompt with `--related-spec` and `--reason` when relevant.
4. Send with `hero peer call` or `hero handoff`; report the message/thread ID.
5. Do not wait unless the user requests it. `--wait` only polls Project Mail and
   may return `pending`; it never launches a model.
6. Explain that receiver work begins only after
   `hero handoff receive <message-id>`. Do not claim a receiver spec exists
   before that explicit promotion.

For list/show requests, use `hero peer list` or `hero peer show <alias>`.
Preserve legacy `hero handoff status` and `hero handoff accept <slug>` usage.
External automatic responders are optional harness/worker behavior, not Hero
core behavior.
