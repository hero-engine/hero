---
title: Peer Identity is a Stable UUID Minted at `hero init`
type: decision
status: proposed
created: 2026-05-15
tags: [peering, identity, hero-init, decision]
relations:
  - target: cross-repo-peering
    kind: decided-in
---

# Peer Identity is a Stable UUID Minted at `hero init`

## Decision

Every Hero workspace gets a UUID generated at `hero init` (or on
first invocation after upgrade for existing workspaces), stored in
`hero.json` as `peer_id`. The UUID is the **canonical identifier**
across all cross-repo peering artifacts: manifests, handoff trail
records, peer call envelopes, cloud-routed events.

Local aliases (`hero repos`) remain the human-readable handle for
display only. Two machines can disagree on aliases — one calls a
repo "app", another calls it "backend" — without breaking peering.

## Context

The original draft of the cross-repo-peering spec listed peer name
canonical form as an open question: local alias, the
`hero.json:name` value, the first-commit SHA, or a UUID.

The risk of using aliases or local names as the join key: two
developers (or one developer with two checkouts of the same sibling
graph) can call the same repo by different aliases. Trail entries
written on one side stop resolving on the other. Cloud federation
breaks. The bug is silent until the names diverge.

First-commit SHA is stable but fragile: rebasing the initial commit
or importing the repo from elsewhere breaks it.
`hero.json:name` collides easily across personal forks.

## Options considered

1. **Local alias as canonical.** Use whatever the user typed in
   `hero repos add`.
   - Pros: nothing to mint.
   - Cons: silently breaks when machines disagree.

2. **First-commit SHA.** Use the SHA of the repo's first commit.
   - Pros: derivable, no mint.
   - Cons: rebases / re-clones / repo imports break it; not
     under Hero's control.

3. **`hero.json:name`.** Use the human-readable name from
   `hero.json`.
   - Pros: already exists.
   - Cons: collisions across forks; can be renamed.

4. **UUID minted at `hero init`, stored in `hero.json`.**
   - Pros: stable, unambiguous, under Hero's control, survives
     renames and re-clones (because it's in the tracked
     `hero.json`).
   - Cons: requires a migration step for existing workspaces;
     duplicate-clone scenario (one developer clones the same
     workspace twice) creates colliding peer_ids.

## Decision

Option 4. UUID minted at `hero init`, stored in `hero.json` as
`peer_id`. Migration mints on first invocation when missing and
writes a `workspace.peer_id_minted` event for traceability.

## Consequences

- `hero init` writes `peer_id` to `hero.json`.
- A first-invocation migration mints `peer_id` for workspaces that
  predate this feature.
- All peering data structures (manifest, handoff trail, peer call
  request/result envelopes) key on `peer_id`.
- `hero repos scan` reads the sibling's `peer_id` and records both
  the alias and UUID. `CrossRepoResolver` dual-keys on both, with
  peer_id as the canonical join.
- `hero check` warns on duplicate `peer_id` (e.g., a workspace
  cloned and edited in two places).
- Display layer always resolves UUID → local alias before showing
  to humans. Bare UUIDs only appear in diagnostic output.
- This is a hard rule: any code path that uses an alias as a
  cross-workspace join key is a bug.
