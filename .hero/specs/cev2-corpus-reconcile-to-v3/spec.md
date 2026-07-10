---
title: "Reconcile context-engine-v2 corpus to verified reality (post hero-code v3 audit)"
slug: cev2-corpus-reconcile-to-v3
type: bug
status: completed
completed_at: 2026-07-06
severity: medium
priority: medium
size: small
created: 2026-07-07
tags: [hero-code, context-engine, corpus, hygiene, supersession, peering, bookkeeping]
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  originator_slug: context-planning-corpus-hygiene
  call_id: 18bfebc275f83f803fb6a6e33ab7a7ea
  mode: spec-out
  at: 2026-07-07T05:46:37Z
  at_commit: 4a7e4d4
  reason: "context-engine-v3 (hero-code) supersedes context-engine-v2's remaining scope; 2026-07-06 corpus audit verified one v2 child implemented in hero-code but still marked planning on your side."
relations:
  - target: context-engine-v2
    kind: parent
  - target: cev2-widen-load-bearing-markers
    kind: reconciles
  - target: cev2-bash-output-supersede
    kind: reconciles
  - target: cev2-tool-input-compression
    kind: reconciles
  - target: cev2-context-engine-test-harness
    kind: reconciles
  - target: cev2-system-prompt-curator
    kind: reconciles
---

# Reconcile context-engine-v2 corpus to verified reality

## Kickoff

You are picking up a small bookkeeping chore. No code changes. Six spec
frontmatters need to be updated so the corpus matches what actually
shipped or was replaced in peer `hero-code`. Open this spec, then read
`.hero/knowledge/decisions/superseded-is-genealogy-not-status.md` (the
local supersession convention) before editing frontmatter. Then apply
the six edits enumerated under **Changes**. Verify with the checks in
**Acceptance**.

## Context

The `context-engine-v2` initiative (`.hero/planning/initiatives/context-engine-v2/spec.md`)
and its seven flat-named child specs at `.hero/planning/cev2-*.md` were
authored on 2026-06-09 to fix and optimize hero-code desktop context
curation. Two children shipped on our side and are already `completed`:

- `cev2-verbatim-turn-counting`
- `cev2-protect-compaction-summaries`

A cross-workspace peer-call audit on 2026-07-06
(`context-planning-corpus-hygiene` on peer `hero-code`) surfaced two
distinct forms of drift in the remaining five entries:

1. **One planning child actually shipped in hero-code.**
   `cev2-widen-load-bearing-markers` reads `status: planning` here but
   is implemented at
   `apps/hero-desktop-mac/Sources/HeroDesktop/Engine/ContextCurator.swift`
   in `looksLoadBearing()` — the marker set was widened 8 → 22 with
   `[cev2]`-tagged additions plus code-fence and pipe-table structural
   signals. A delivery audit already exists on our side at
   `.hero/planning/audits/cev2-widen-load-bearing-markers-audit.md`
   (verdict: SHIP, 46/46 tests, dated 2026-06-09). The spec status is
   simply stale — someone forgot to flip it after the audit landed.

2. **Four planning children and the initiative's remaining scope were
   replaced by a hero-code v3 initiative.** Peer `hero-code` opened
   `context-engine-v3` after the v2 delivery pass, restructured the
   remaining work, and now owns four child specs that carry the
   previously v2 scope:

   | v2 child on our side (planning)        | v3 child on hero-code (successor)     |
   |----------------------------------------|---------------------------------------|
   | `cev2-bash-output-supersede`           | `bash-output-supersede`               |
   | `cev2-tool-input-compression`          | `tool-input-compression`              |
   | `cev2-context-engine-test-harness`     | `context-fixture-test-harness`        |
   | `cev2-system-prompt-curator`           | `system-prompt-curation`              |

   `context-engine-v3` declares `supersedes: context-engine-v2`. The
   scope moved because the implementation lives natively in the
   hero-code Swift codebase — the v2 initiative was authored on our
   side but always intended to land on the other side of the peer
   boundary. Having four `planning` entries on our side that will
   never be worked here misleads retrieval and every cold-start `hero
   queue` render.

The two already-completed children (`cev2-verbatim-turn-counting`,
`cev2-protect-compaction-summaries`) stay untouched — they shipped
here and their status reflects reality.

## Convention

Local convention (`superseded-is-genealogy-not-status.md`):

> "Was this spec replaced by another?" is a **genealogy** fact,
> modeled as a dedicated frontmatter field (`superseded_by: <slug>`),
> not a value of the lifecycle `status:` enum.

The prescribed frontmatter shape is:

```yaml
superseded_by: <slug>
relations:
  - target: <slug>
    kind: superseded-by
```

The indexer derives the inverse `supersedes` edge on ingest. `status:`
is left at its lifecycle value (typically `completed`); the legacy
`StatusSuperseded` enum value remains valid for backward compatibility
but is not the preferred write.

**Cross-workspace wrinkle.** `superseded_by:` is a local slug field.
Our indexer does not resolve slugs across the peer boundary today — a
bare peer slug in `superseded_by:` won't dereference. To keep the
pointer machine-readable without inventing new plumbing:

- Set `superseded_by: <peer_slug>` (the hero-code slug) so any future
  cross-peer resolver has a stable pointer.
- Add a co-located `superseded_by_peer:` block mirroring the shape of
  `received_from:` (peer_id, peer_alias_display, peer_slug) so a
  human/reader can see this is a cross-workspace supersession, and so
  a future indexer pass can qualify the pointer.
- Use `status: superseded` as well (legacy enum). Rationale: bare
  `superseded_by:` alone does not remove a `planning` spec from
  `hero queue`; the audit's whole point is to stop these entries from
  masquerading as available work. Belt-and-suspenders: both fields.

This is the pattern below.

## Changes

Six frontmatter edits. **No body edits, no code changes, no touching
`.hero/planning/audits/*-audit.md`, no touching the two completed
children.**

### 1. Mark `cev2-widen-load-bearing-markers` as completed

File: `.hero/planning/cev2-widen-load-bearing-markers.md`

Replace `status: planning` with `status: completed` and add
`completed_at: 2026-06-09` (the audit date). Add a `delivered_in_peer:`
block pointing at hero-code, since delivery happened across the peer
boundary and not on our commit history. Result:

```yaml
---
title: "Widen looksLoadBearing() marker set"
slug: cev2-widen-load-bearing-markers
type: feature
status: completed
completed_at: 2026-06-09
priority: medium
size: small
parent: context-engine-v2
created: 2026-06-09
delivered_in_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  code_path: apps/hero-desktop-mac/Sources/HeroDesktop/Engine/ContextCurator.swift
  function: looksLoadBearing
  summary: "Marker set widened 8 → 22 with [cev2]-tagged additions plus code-fence and pipe-table structural signals."
  audit_local: .hero/planning/audits/cev2-widen-load-bearing-markers-audit.md
tags: [hero-code, swift, context-engine, curator, tagging, heuristics]
---
```

Leave the entire body untouched. The delivery audit at
`.hero/planning/audits/cev2-widen-load-bearing-markers-audit.md`
already carries the SHIP verdict; do not rewrite it.

### 2. Supersede `cev2-bash-output-supersede`

File: `.hero/planning/cev2-bash-output-supersede.md`

Change `status: planning` to `status: superseded`. Add supersession
frontmatter pointing at the hero-code v3 successor:

```yaml
status: superseded
superseded_by: bash-output-supersede
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: bash-output-supersede
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative; implementation is native Swift and lives in the peer workspace."
relations:
  - target: bash-output-supersede
    kind: superseded-by
```

Leave the body untouched.

### 3. Supersede `cev2-tool-input-compression`

File: `.hero/planning/cev2-tool-input-compression.md`

Same edit shape as (2). Pointer values:

```yaml
status: superseded
superseded_by: tool-input-compression
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: tool-input-compression
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative."
relations:
  - target: tool-input-compression
    kind: superseded-by
```

### 4. Supersede `cev2-context-engine-test-harness`

File: `.hero/planning/cev2-context-engine-test-harness.md`

Note the slug rename on the successor side (`context-fixture-test-harness`,
not `cev2-context-engine-test-harness`):

```yaml
status: superseded
superseded_by: context-fixture-test-harness
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: context-fixture-test-harness
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative; successor slug renamed to context-fixture-test-harness."
relations:
  - target: context-fixture-test-harness
    kind: superseded-by
```

### 5. Supersede `cev2-system-prompt-curator`

File: `.hero/planning/cev2-system-prompt-curator.md`

Note the successor slug is `system-prompt-curation` (verb → gerund):

```yaml
status: superseded
superseded_by: system-prompt-curation
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: system-prompt-curation
  successor_initiative: context-engine-v3
  reason: "Scope moved to hero-code's context-engine-v3 initiative; successor slug renamed to system-prompt-curation."
relations:
  - target: system-prompt-curation
    kind: superseded-by
```

The existing `depends-on: [cev2-context-engine-test-harness]` line can
stay — it was true at the time of authoring and now carries historical
value only. Do not delete it.

### 6. Supersede the initiative's remaining scope

File: `.hero/planning/initiatives/context-engine-v2/spec.md`

The initiative is partially shipped: two children (`cev2-verbatim-turn-counting`,
`cev2-protect-compaction-summaries`) landed here; one
(`cev2-widen-load-bearing-markers`) landed on hero-code; four moved to
`context-engine-v3` on hero-code. The remaining-scope pointer is at
the initiative level:

```yaml
status: superseded
superseded_by: context-engine-v3
superseded_by_peer:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  peer_slug: context-engine-v3
  reason: "Remaining scope (bash-output-supersede, tool-input-compression, test-harness, system-prompt-curator) migrated to hero-code's context-engine-v3 initiative. Three v2 children (cev2-verbatim-turn-counting, cev2-protect-compaction-summaries, cev2-widen-load-bearing-markers) remain completed on their original workspaces and are not superseded."
relations:
  - target: context-engine-v3
    kind: superseded-by
```

Add a short **Reconciliation** section to the **body** of the
initiative spec (just before the `## Progress` table, so the surviving
progress table stays intact). Text:

```markdown
## Reconciliation (2026-07-07)

Superseded by `context-engine-v3` on peer `hero-code`. Not all v2
scope moved — three children shipped and are frozen where they
delivered:

| Child                                | Outcome           | Where it shipped |
|--------------------------------------|-------------------|------------------|
| `cev2-verbatim-turn-counting`        | completed         | this workspace   |
| `cev2-protect-compaction-summaries`  | completed         | this workspace   |
| `cev2-widen-load-bearing-markers`    | completed         | peer `hero-code` |
| `cev2-bash-output-supersede`         | superseded → v3   | peer `hero-code` |
| `cev2-tool-input-compression`        | superseded → v3   | peer `hero-code` |
| `cev2-context-engine-test-harness`   | superseded → v3 (`context-fixture-test-harness`) | peer `hero-code` |
| `cev2-system-prompt-curator`         | superseded → v3 (`system-prompt-curation`) | peer `hero-code` |

Successor initiative: `context-engine-v3` on peer `hero-code`
(peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`). It declares
`supersedes: context-engine-v2`.
```

Do **not** rewrite the existing `## Progress` table — leave it as a
snapshot of the original planning state. The Reconciliation section is
additive.

## Boundaries

- **No code changes.** This spec touches spec frontmatter and (in one
  case) initiative body prose only. If a reviewer asks "did anything
  compile differently?", the answer is no.
- **Do not edit the two already-completed children.**
  `cev2-verbatim-turn-counting` and `cev2-protect-compaction-summaries`
  are correct as-is.
- **Do not delete any spec files.** Superseded specs stay on disk with
  a pointer, per `superseded-is-genealogy-not-status`. `hero why`
  traversals continue to reach them.
- **Do not edit the delivery audits.** Audit files under
  `.hero/planning/audits/cev2-*-audit.md` are historical records.
- **Do not touch peer-side files.** This spec's remit is the local
  corpus only. The `context-engine-v3` initiative on hero-code is
  managed by hero-code.
- **Do not invent a `superseded_by_peer` field consumer.** No indexer
  change ships with this spec; the field is a documentary pointer
  today and a machine-readable hook for the future. If the indexer
  does not recognize it, that is fine.

## Risks

- **`superseded_by:` referencing a peer slug will not resolve locally.**
  Our indexer treats `superseded_by:` as a local slug lookup, so on
  ingest it may log an unresolved-reference warning for each of the
  five entries. This is acceptable — the intent is documentary, and
  the `superseded_by_peer:` block provides the disambiguating context.
  If `hero check` fails hard on unresolved `superseded_by:` (unlikely
  — it is treated as a de-weight hint, not a hard reference), fall back
  to setting `status: superseded` alone and moving the peer slug into
  the `superseded_by_peer:` block only. Discover this by running
  `hero check` after the edits (see Acceptance).

- **The four `cev2-*` flat-named children are already affected by
  `flat-named-spec-discovery` (active bug).** They may be invisible to
  discovery today, so this edit may not visibly change `hero queue`
  output. That is not a reason to skip the edit — when the discovery
  bug is fixed, these entries should already be correctly marked.

- **`cev2-widen-load-bearing-markers` is being marked `completed`
  without a local commit trail.** Delivery happened in the peer
  workspace. The `delivered_in_peer:` block is the trail. `hero why`
  will not walk back to a commit SHA on this side; that is expected.

## Acceptance

Run these checks after applying the edits. All must pass.

1. **`hero read cev2-widen-load-bearing-markers` reports
   `status: completed`** with `completed_at: 2026-06-09` and a
   `delivered_in_peer:` block pointing at
   `apps/hero-desktop-mac/Sources/HeroDesktop/Engine/ContextCurator.swift`.

2. **Each of the four superseded children reports `status: superseded`
   with a `superseded_by:` pointer:**
   - `cev2-bash-output-supersede` → `bash-output-supersede`
   - `cev2-tool-input-compression` → `tool-input-compression`
   - `cev2-context-engine-test-harness` → `context-fixture-test-harness`
   - `cev2-system-prompt-curator` → `system-prompt-curation`

3. **The initiative `context-engine-v2` reports `status: superseded`
   with `superseded_by: context-engine-v3`** and carries a
   `## Reconciliation` section listing the seven children and where
   each landed.

4. **`hero queue` no longer includes any of the four superseded
   children.** (Assuming `flat-named-spec-discovery` does not blind
   this check; if it does, `grep -l "status: superseded"
   .hero/planning/cev2-*.md` returning the four expected files is a
   sufficient substitute.)

5. **The two untouched children remain untouched.** `grep -c "^status:
   completed" .hero/planning/cev2-verbatim-turn-counting.md` and
   `grep -c "^status: completed"
   .hero/planning/cev2-protect-compaction-summaries.md` each return 1;
   `git diff` shows no changes to either file.

6. **No files under `.hero/planning/audits/cev2-*-audit.md` are
   modified.**

7. **`hero check`** run after the edits does not surface a new hard
   failure. Warnings about unresolved cross-peer `superseded_by:`
   pointers are acceptable — record them as an entry in
   `.hero/knowledge/decisions/` (or note in this spec's completion
   record) if they appear, so the future cross-peer-resolver work has
   a discoverable input.

## Validation

- Read each of the six edited files after the change and confirm the
  frontmatter matches the shapes prescribed in **Changes**.
- `git diff --stat` should show six files touched (five under
  `.hero/planning/cev2-*.md`, one under
  `.hero/planning/initiatives/context-engine-v2/spec.md`), plus this
  spec itself.
- No other spec files should appear in the diff.

## Provenance

Designed in response to a `hero peer call --mode=spec-out` from peer
`hero-code` (peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`,
alias display `hero-code`), related to originator spec
`context-planning-corpus-hygiene`, call_id
`18bfebc275f83f803fb6a6e33ab7a7ea`, at `2026-07-07T05:46:37Z`, on
their commit `4a7e4d4`.
