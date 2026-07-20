# Hero Code contract response — Durable Attention v1

Paste this into the Hero Code task that owns
`durable-attention-consumer`.

## Response

Hero has accepted the consumer architecture and recorded its requested
snapshot/row fields, advertised action descriptors, structured errors,
promotion results, user-global access, schemas, manifest/checksums, and golden
fixtures in the upstream `durable-attention` initiative and its four blocking
children.

Use these proposed v1 decisions until a child `/design` pass produces a more
specific contract:

1. **`do_next`**
   - Hero atomically accepts the suggestion into a durable Personal Focus item
     in `today`.
   - The authoritative result returns the Focus row and a launch intent.
   - Desktop session creation is a separate Hero Code effect. A failed or
     cancelled launch leaves the Focus item in Today for safe retry; Hero and
     Hero Code do not pretend storage plus window/session creation is one atomic
     transaction.

2. **Mail `add_to_today`**
   - The action idempotently creates a separately-owned Personal Focus item
     linked to the Mail source.
   - It does not turn Mail into a Focus lifecycle and does not imply Mail
     acknowledgement or dismissal.
   - The authoritative result returns the created/existing Focus source ID and
     updated projection.

3. **Focus-to-session correlation**
   - Persisted correlation is not required for v1.
   - If observability requires it later, use a typed launch context/source
     reference. Never overload `Session.specSlug`, and never infer completion
     from harness todo state.

4. **Global client boundary**
   - Hero will publish exactly one boundary usable before a project is open.
   - `durable-attention-contracts` must inspect current Serve/MCP/runtime
     ownership before selecting it. Hero Code should keep transport behind
     `AttentionClient` and must not implement both MCP and HTTP/SSE.

5. **Refresh versus streaming**
   - V1 guarantees an authoritative versioned snapshot, revision/change token,
     explicit refresh after mutations, and refresh on mount, foreground,
     reconnect, and user retry.
   - A streaming event cursor is optional, not a hard v1 dependency. Ordering,
     replay windows, cursor expiry, duplicates, gap detection, and recovery
     would require a durable service/runtime and materially enlarge the
     lightweight mailbox.
   - Revise `durable-attention-consumer` so its core UI/store/launch delivery
     works from snapshot refresh alone. Keep the event subscription behind the
     client protocol as a later capability if Hero publishes it.

The terminology migration is accepted in principle:

- existing Advisor `FocusItem` becomes Work Recommendation;
- existing `NeedsAttention` becomes Backlog Health;
- durable **Focus Item** and combined **Attention** remain reserved for the new
  Hero contracts.

Please update the consumer spec's dependency statement and AC-12/refresh design
to make streaming optional. Report whether that reduces the declared `large`
scope or whether the terminology migration and cross-project launch coordinator
still justify `large`.
