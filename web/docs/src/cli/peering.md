# Cross-Repo Peering

When you run sibling Hero workspaces — backend, web client, desktop
client — they can work as a tag team. A session in one repo can ask
another's Hero a question, hand off a spec, or pull peer-surface
conventions into context. Provenance travels with every operation.

## Setup

```bash
hero init                                # mints a stable peer_id UUID
hero admin repos add app ../app          # register a sibling peer
hero admin repos add web ../web --local  # register only for this machine (writes to hero.local.json)
hero peer manifest                       # regenerate this workspace's peer-manifest.yaml
```

Registration is one-time per developer machine. The peer manifest lives
at `.hero/peer-manifest.yaml` and is committed.

## Inspect

```bash
hero peer list                           # configured peers + reachability + reciprocity
hero peer show app                       # manifest contents, in-flight handoffs
```

## Sync Peer Calls

A peer call spawns a subagent inside the peer workspace and returns
the result to your session.

```bash
hero peer call app --mode=advisory "What's your error envelope?"
hero peer call app --mode=spec-out "Add CSV export endpoint" --reason "API change owned by app"
hero peer call app --mode=advisory "..." --related-spec csv-export
```

Modes:

| Mode | Writes on peer? | Pick when |
|---|---|---|
| `advisory` | Nothing | You need a fact: convention, schema, ownership, "does this break you?" |
| `spec-out` | Spec on peer | The work is really the peer's. Their Hero designs the spec natively. |

## Async Handoff

When you have already done the investigation and want to drop the spec
on the peer's queue:

```bash
hero handoff order-failure app --reason "Root cause is the API"
hero handoff csv-export web --title "CSV export client wiring" --type feature
hero handoff status                      # all in-flight handoffs
hero handoff status order-failure        # one spec
hero handoff accept order-failure        # pick up a handed-back spec on this side
```

The receiving workspace gets a scaffolded spec stamped with
`received_from` provenance. The sender's spec status moves to
`handed_off` and a Handoff Trail entry is appended.

## Passive Boundary Detection

```bash
hero context imports                     # scan git-dirty files
hero context imports --files src/a.go,src/b.go
```

Scans changed files for Go imports of contract symbols listed in any
configured peer's manifest. When a match is found, prints a one-line
signal naming the symbol, the owning peer, the governing convention,
and the last-changed commit on the peer side. Passive — never blocks,
prompts, or auto-triggers a peer call.

## Disambiguating `handoff`

Two distinct commands share the verb:

| Command | Scope |
|---|---|
| `/handoff` (slash) | Session-level — force-refresh NEXT.md before switching tools or compaction. |
| `hero handoff <spec> <alias>` (CLI) | Cross-repo — drop a spec on a peer workspace. |

If a peer alias is named, it's the CLI command. Otherwise it's the
slash command.
