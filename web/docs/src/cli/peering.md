# Cross-repository peering CLI

Peering is an **optional**, asynchronous Project Mail workflow between
registered sibling Hero projects. Each project keeps one authoritative graph.

## Setup and inspection

```bash
hero admin repos add app ../app
hero admin repos add web ../web --local
hero peer manifest
hero peer list
hero peer show app
```

The peer must be reachable at its registered path and expose a current
`.hero/peer-manifest.yaml`.

## Advisory and spec-out requests

```bash
hero peer call app --mode=advisory "Is the error envelope stable?" \
  --reason "CSV export client depends on it" --related-spec csv-export

hero peer call app --mode=spec-out "Design the export endpoint" \
  --reason "API work belongs to app" --related-spec csv-export
```

These commands enqueue Project Mail. They do not launch a receiver-side model
or write the receiver tree synchronously. The receiver reads, promotes, and
replies through its own workflow.

## Work transfer

```bash
hero handoff order-failure app --reason "Root cause is in the API"
hero handoff status order-failure

# In the receiver workspace, after inspecting the Mail request:
hero handoff receive <message-id>
```

Sending does not mutate either spec tree. `receive` is the explicit
receiver-owned promotion through Intake. Use `hero handoff accept <slug>` only
for a legacy handed-back spec.

## Session handoff is different

`/handoff` inside a supported harness refreshes session NEXT state. `hero
handoff <spec> <alias>` transfers work across a repository boundary. Naming a
peer alias selects the cross-repository meaning.
