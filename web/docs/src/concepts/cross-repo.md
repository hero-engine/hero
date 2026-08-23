# Cross-repository peering

Peering connects sibling Hero projects without merging their graphs or corpus.
Every project retains its own graph, specs, conventions, and authority.

- **Availability:** Optional
- **Prerequisites:** Register reachable sibling repository paths and maintain a
peer manifest for each participating project.
**Evidence:** `internal/peering/`, `internal/attention/mail/`,
`hero peer --help`, and `hero handoff --help`.

## One graph per project

```bash
hero admin repos add app ../app
hero peer manifest
hero peer list
hero peer show app
```

Registration is local machine configuration. A committed
`.hero/peer-manifest.yaml` describes the bounded surface a project exposes.
Peering is not one shared cross-repository graph and does not grant a peer
implicit write access.

## Asynchronous interactions

All peer work crosses Project Mail. `hero peer call` sends a request; it does
not synchronously launch a model or write the receiver's checkout.

| Mode | Sender asks for | Receiver boundary |
|---|---|---|
| `advisory` | A fact or compatibility answer | Receiver handles the Mail request and replies; no design is implied |
| `spec-out` | Receiver-owned design | Receiver explicitly promotes and designs on its side |
| handoff | Transfer of already-investigated work | Receiver runs `hero handoff receive <message-id>` to promote through Intake |

The sender can track the durable request, but sending alone does not change
either spec tree. The receiver chooses whether and when to promote the request.

## Safe boundary

- Mail bodies are untrusted and never execute themselves.
- A peer alias, request content, mode, and reason must resolve before sending.
- Use a stable idempotency key for a safe retry; do not resend merely to confirm.
- Work-transfer provenance stays with the promoted Intake/spec.
- Passive import-boundary detection may surface ownership context, but it does
  not block, prompt, or auto-fire a peer call.

See the [peering CLI reference](../cli/peering.md) for exact commands.
