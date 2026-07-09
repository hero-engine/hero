---
title: "Compose emits priority + reciprocal conflicts-with on child stubs"
slug: compose-emits-conflicts-and-priority
type: enhancement
status: completed
size: small
priority: high
domain: engineering
tags: [compose, drive, authoring, relations, conflicts-with, sequencing]
created: 2026-07-09
relates-to: [priority-conflict-aware-drive-selection]
completed_at: 2026-07-09T21:56:47Z
---

# Compose emits priority + reciprocal conflicts-with on child stubs

## Context

The companion spec `priority-conflict-aware-drive-selection` (completed, now in
`.hero/specs/`) taught the `/drive` judge (`drive.Check`, `internal/drive/check.go`)
to **honor** two structured signals when picking the next child of an initiative:

1. **priority / severity** — used as a tiebreak among dependency-ready children
   (`rank`: `critical=0, high=1, medium=2, low=3, unset=99`), and
2. **`conflicts-with`** — a soft mutex: the judge won't start a child whose
   `conflicts-with` target is currently `delivering`, and pauses with a
   non-promotable `SeamCollision` category.

That was the **consumer** half. Nothing on the **authoring** side emits those
signals yet, so both features are largely inert:

- The priority tiebreak has no inputs unless a human hand-stamped `priority:` on
  a child. Children scaffolded by `/compose` today carry **no** priority, so they
  all rank `unset=99` and the judge falls straight back to slug order — exactly
  the behavior the companion spec set out to fix.
- The `conflicts-with` gate only fires on **hand-written** relations. `/compose`
  reasons about overlap **only in prose** today. Real initiatives (e.g.
  `.hero/planning/initiatives/content-remediation/spec.md`) carry a
  `| Wave | Child | Type | Size | Why this position |` table, a "Waves minimize
  same-file conflicts" note, and an "In-flight overlap watch" narrative — none of
  which the judge can read. So the seam guard the companion spec built sits idle
  on the exact initiatives it was designed to protect.

This spec closes the loop on the `/compose` authoring side: when compose sequences
children, it stamps `priority`/`severity` on every stub and emits **reciprocal**
`conflicts-with` relations for every overlap seam it already describes in prose —
turning machine-unreadable sequencing narrative into the structured inputs the
judge consumes.

**No Go change is required for the core work.** `conflicts-with` is already a
first-class accepted relation kind (parser at `internal/spec/spec.go:578`, edge
map at `internal/spec/graph_ingest.go:224`, both landed by the companion spec) and
`priority`/`severity` are existing frontmatter string fields
(`spec.Spec.Priority` / `.Severity`, values `critical|high|medium|low`). This is
authoring-instruction and shared-contract work.

## Goal

When `/compose` decomposes an initiative, every child stub it writes carries a
`priority:` stamp (and `severity:` where the child type warrants it) derived from
the sequencing analysis via an explicit, reproducible mapping, **and** every
overlap seam the compose narrative identifies is emitted as a **reciprocal**
`conflicts-with` relation on both named children — not only as a prose "overlap
watch" note. The prose Wave table and overlap narrative stay (they explain *why*);
the structured relations are their machine-actionable projection, authored
alongside and kept in sync so neither drifts. The child-stub authoring contract —
including a rule that `/design` **preserves** a stub's `priority`/`severity` and
`conflicts-with` when it materializes the stub into a full spec — is documented in
the shared `core/skills/spec-format/SKILL.md` so every domain's authoring
workflow inherits it. Done means: the judge's priority tiebreak and seam guard get
real inputs from ordinary `/compose` output, with zero new hand-stamping.

## Kickoff

Makes `/compose` stamp `priority` on every child stub and emit reciprocal
`conflicts-with` relations for the overlap seams it already writes as prose — so
the shipped `/drive` judge actually gets the signals it was built to read.

**Status:** planning — spec just landed, no code yet. Companion judge-side half
(`priority-conflict-aware-drive-selection`) is already shipped in `.hero/specs/`.

**Pick up at:** start with `core/skills/spec-format/SKILL.md` — add the
"Child-stub authoring contract" (priority mapping table, reciprocal-relation rule,
preserve-on-materialize rule). Then thread the same rules into `compose.md`,
`product-ideator.md`, and the two delivery-lead agents so authoring emits them.

→ `.hero/planning/features/compose-emits-conflicts-and-priority/spec.md`

**Files:** `core/skills/spec-format/SKILL.md`, `domains/engineering/commands/compose.md`, `domains/engineering/agents/product-ideator.md`, `domains/engineering/agents/feature-delivery-lead.md`, `domains/engineering/agents/platform-delivery-lead.md`
**Skip:** any Go change for the core work — `conflicts-with` and `priority` already parse. The `hero check` lint and file-overlap detection are Future, not this spec.

## Problem

The judge can only act on **structured, on-disk** state (it is a pure,
cold-startable function — determinism is a hard constraint of the companion spec).
Compose currently encodes its two most decision-relevant signals in forms the
judge cannot consume:

- **Sequencing preference** lives in a prose "Why this position" column and Wave
  ordering, never as `priority:`. Result: `unset=99` for every child, priority
  tiebreak inert.
- **Overlap seams** live in a prose "In-flight overlap watch" note, never as
  `conflicts-with` edges. Result: seam guard never fires on composed initiatives.

Both gaps degrade to the same failure the companion spec was fixing — the judge
picks the wrong next child, or starts a colliding one, and a human overrides the
run. The companion spec fixed the *reader*; the *writer* still speaks only prose.

## Approach

Option A (see `## Alternatives`): author the structured signals at `/compose`
time, by hand (LLM/human), deterministic and travelling with the spec, zero new Go.
Four coordinated authoring edits plus one shared contract.

### A1 — Priority stamping (reproducible mapping)

When compose sequences children it already produces a Wave/position rationale.
Map that rationale to concrete `priority:` values with a **fixed, reproducible
table** (documented once in `spec-format`, referenced by the authoring surfaces):

| Sequencing signal (from the compose analysis) | `priority:` |
|---|---|
| Foundational anchor — blocks the most siblings, or is safety- / security- / data-integrity-critical (typically Wave 1) | `critical` |
| On the critical path — early wave, one or more siblings depend on it | `high` |
| Standard work — middle wave, no special urgency, no dependents waiting | `medium` |
| Deferrable — late wave, independent polish, or explicitly "nice to have" | `low` |

Rules that make it reproducible and effective:

- **Stamp every child stub.** An unstamped child ranks `unset=99` and sinks below
  every stamped sibling — leaving *some* children unstamped silently reorders the
  judge. All-or-nothing: if compose stamps one, it stamps all.
- **Priority is a tiebreak, not an ordering.** Hard ordering stays in
  `depends-on`. `priority:` expresses *"when several children are dependency-ready
  at once, which do we want first."* The Wave position is the proxy for that.
- **`severity:` where the child type warrants it.** For `bug`-type children,
  additionally stamp `severity:` from the same impact reasoning (a Wave-1
  data-loss/correctness bug → `severity: critical`). The judge ranks severity as
  the second key, after priority. Feature/enhancement children need only
  `priority:`.

### A2 — Reciprocal `conflicts-with` emission

When the sequencing analysis identifies two children that touch the same code
region / "same-file conflict" / overlap seam, emit a `conflicts-with` relation
**on both children** — a hard rule, not a preference:

```yaml
# in child-a/spec.md stub frontmatter
relations:
  - target: child-b
    kind: conflicts-with
# in child-b/spec.md stub frontmatter
relations:
  - target: child-a
    kind: conflicts-with
```

**Reciprocity is required, not optional.** The companion judge honors a child's
**own (outbound)** `conflicts-with` relations and does **not** scan inbound edges
(v1 semantics, documented in `priority-conflict-aware-drive-selection`). A
one-sided relation only protects one direction: whichever child is being *selected*
must itself carry the edge, or the guard silently no-ops. Authoring both edges is
the only way to make the mutex symmetric. State this in the authoring instruction
verbatim.

### A3 — Keep the prose, keep it in sync

The Wave table and overlap narrative are **not** replaced — they explain the *why*
a machine can't infer. The structured relations are the machine-actionable
projection authored **alongside** the prose. The anti-drift rule:

> Every "overlap watch" / same-file / seam entry named in the initiative prose
> MUST have a corresponding reciprocal `conflicts-with` relation on the two named
> children. No orphan prose (a named seam with no relation) and no orphan relation
> (a `conflicts-with` edge with no prose explaining it).

This makes the prose and the relations a single fact expressed twice, deliberately
— reviewable by a human, executable by the judge — with a stated invariant that
ties them together so a future editor can't quietly drop one.

### A4 — Preserve-on-materialize contract (shared home)

The above rules describe how compose writes a **stub**. A stub is later
**materialized** into a full spec by `/design`
(`.hero/planning/initiatives/{init}/{child}/spec.md`). The materialize step MUST
**preserve** the stub's `priority`/`severity` frontmatter and its `conflicts-with`
relations — silently dropping them would re-inert the judge at exactly the moment
the child becomes real and deliverable. This is the **child-stub authoring
contract**, and its home is the shared `core/skills/spec-format/SKILL.md` (not an
engineering-domain file) so every domain's `/design` inherits it — PM, and future
packs, materialize stubs too. `spec-format` today has **no** conflicts-with /
priority / child-stub guidance (grep-confirmed empty), so this is net-new.

## Changes

1. **`core/skills/spec-format/SKILL.md` — add a "Child-stub authoring contract"
   section (the shared home; do this first).**
   - New top-level section documenting what an initiative child stub carries and
     how it is preserved. Include:
     - the **priority mapping table** from A1 (canonical copy — the authoring
       surfaces reference it rather than restating it, to prevent drift);
     - the **stamp-every-child** and **priority-is-a-tiebreak-not-ordering** rules;
     - the **`severity:` for bug-type children** rule;
     - the **reciprocal `conflicts-with`** hard rule from A2, with the two-sided
       YAML example and the one-line reason (judge honors outbound-only in v1);
     - the **prose ⇄ relation sync invariant** from A3 (no orphan prose, no orphan
       relation);
     - the **preserve-on-materialize** rule from A4 — `/design` MUST carry over a
       stub's `priority`/`severity` and `conflicts-with` when materializing.
   - Cross-reference the existing `relates-to`/`depends-on` frontmatter table and
     the `conflicts-with` relation kind so authors see it is a first-class kind.

2. **`domains/engineering/commands/compose.md` — instruct structured emission
   during decomposition.**
   - In the child-spec-stub authoring step (currently items 2 and "Where child
     specs live", lines 12–35), add explicit instructions to:
     - stamp `priority:` (and `severity:` for bug children) on **every** child
       stub per the `spec-format` mapping table;
     - for every overlap seam the initiative prose describes, emit a **reciprocal**
       `conflicts-with` relation on both named children;
     - keep the Wave table / "In-flight overlap watch" prose AND satisfy the
       prose ⇄ relation sync invariant.
   - Add a one-line pointer to the `spec-format` "Child-stub authoring contract"
     as the source of truth (don't restate the mapping table here).

3. **`domains/engineering/agents/product-ideator.md` — carry priority + seam
   signal into the sequencing output.**
   - The ideator already produces a "Prioritized feature list… ordered by
     recommended priority" and "identify dependencies and natural sequencing"
     (lines 22, 48–49). Extend its output contract so each sequenced item emits:
     - a concrete `priority` value (mapping its priority-position rationale to the
       `critical|high|medium|low` scale per `spec-format`), so the delivery lead
       can stamp it directly rather than re-deriving it;
     - an explicit **overlap/seam callout** naming any two items that touch the
       same code region — the raw material the delivery lead turns into reciprocal
       `conflicts-with` relations.

4. **`domains/engineering/agents/feature-delivery-lead.md` — emit the structured
   signals when writing initiative child stubs.**
   - In the compose/initiative authoring path (the `spec-composition` routing area
     around lines 45–46), add: when materializing the ideator's sequenced list
     into child stubs, stamp `priority`/`severity` per the mapping and emit
     reciprocal `conflicts-with` for each named seam, honoring the sync invariant.
     Reference the `spec-format` contract as the source of truth.

5. **`domains/engineering/agents/platform-delivery-lead.md` — same emission on the
   platform authoring path.**
   - Platform initiatives routinely produce multi-spec, multi-subsystem scopes
     with real overlap seams (schema-before-code, dual-write ordering). Add the
     same instruction as item 4 to the platform lead's compose/initiative path
     (around lines 45, 69), referencing the shared `spec-format` contract.

## Alternatives

The user chose **Option A** for this spec. The alternatives are recorded so the
follow-up is grounded, not silently pre-empted.

- **Option A (this spec) — compose-time hand-authored signals.** Deterministic,
  travels with the spec, zero new Go, reviewable prose ⇄ relation pair.
  **Honest limitation:** it depends on the LLM/human getting the seam analysis and
  priority mapping right *at authoring time* — which is the **same class of
  failure the companion judge spec was fixing** (a human forgetting the guardrail).
  If the author forgets to name a seam, no relation is emitted and the guard stays
  blind to it. Option A raises the floor (structured output becomes the default)
  but does not eliminate the forget-the-guardrail failure mode.

- **Option B — delivery-time automated detection.** Use the existing
  `hero conflicts` / `idx.FindConflicts` file-overlap engine (referenced in
  `.hero/planning/initiatives/concurrent-session-branching/spec.md` and
  `.hero/planning/initiatives/cold-start-trust-hardening/spec.md`) to detect
  overlaps from the actual `## Changes` file sets rather than from authored intent.
  Catches **undeclared** overlaps A misses; costs a Go integration and is
  non-deterministic w.r.t. authoring. **Not designed here.**

- **Option C (recommended NEXT increment) — both.** Authored intent (A) as the
  primary, reviewable signal, plus a `FindConflicts` backstop that flags
  *undeclared* file overlaps at judge / pre-flight time. C directly covers A's core
  weakness (the forgotten seam) with a machine detector, while keeping A's authored
  relations as the human-readable, deterministic default. Recommend C as the
  natural follow-up once A is in place; the `FindConflicts` machinery already
  exists, so the follow-up is grounded. **Not designed here — documented as
  follow-up (see `## Future`).**

## Future

- **Option C backstop** — a `FindConflicts`-based detector that flags undeclared
  file overlaps between initiative children at judge/pre-flight time, complementing
  A's authored relations. Grounded in the existing `idx.FindConflicts` engine.

- **`hero check` overlap-lint (the scoped optional decision — OUT of this spec).**
  A lint that warns when an initiative's prose mentions overlap/seam but no
  `conflicts-with` relations exist between its children.
  **Decision: OUT — moved to Future.** Reasoning: (1) it is **not** genuinely
  cheap — it needs a fuzzy prose-keyword heuristic (`overlap`/`seam`/`same-file`),
  child-relation graph traversal, a new warning type in the `hero check` runner,
  and tests; that is real Go + test surface, not a one-liner. (2) It is a **weak**
  safety net for Option A's core weakness: it only catches the case where prose
  *was* written but the relation was forgotten — it cannot catch the more likely
  failure where the author forgot **both** the prose and the relation. (3) The
  principled version of "catch the seam the author missed" is Option C's
  file-overlap detector, which subsumes the lint and doesn't rely on keyword
  matching. Keeping the lint OUT keeps this spec pure content/instruction work
  (`size: small`); the safety-net need is better served by scoping C. If a cheap
  win is still wanted before C, the lint can be its own trivial follow-up spec.

## Boundaries

- **No Go change in this spec.** `conflicts-with` and `priority`/`severity` already
  parse and form edges (companion spec). This is authoring-instruction + shared
  `spec-format` contract work only. If any change here seems to need Go, stop —
  it belongs in Option C or the deferred lint.
- **No change to the judge** (`internal/drive/*`). The reader is done and frozen;
  this spec only feeds it inputs. In particular, do not "fix" the outbound-only
  `conflicts-with` semantics — author reciprocal edges to match them instead.
- **No file-overlap / `FindConflicts` detection** (Option B/C) — Future.
- **No `hero check` lint** — Future (decision above).
- **No `wave` ordinal field.** The companion spec deferred it as redundant once
  priority lands; do not introduce it. Priority encodes the wave preference.
- **Don't rewrite existing initiative specs.** This changes how *new* `/compose`
  output is authored. Backfilling `content-remediation` or other in-flight
  initiatives with relations is optional cleanup, not part of this spec.

## Risks

- **Authoring drift (the core Option A risk).** The author names a seam in prose
  but forgets the reciprocal relation, or stamps priority on some children but not
  all. Mitigations: the stamp-every-child all-or-nothing rule, the prose ⇄
  relation sync invariant stated as a hard rule in `spec-format`, and (longer term)
  Option C. Accept that A raises the floor without eliminating the failure mode —
  state this honestly in the spec, as done in `## Alternatives`.
- **One-sided relation no-op.** If reciprocity isn't followed, the judge's
  outbound-only v1 semantics silently protect only one direction. The reciprocity
  rule must be stated as *required*, with the "why" (outbound-only) inline so a
  future author doesn't "simplify" it to one edge.
- **Mapping subjectivity.** The Wave→priority mapping is a heuristic; two authors
  could stamp differently. The fixed table narrows this, but it won't make it
  fully mechanical. That's acceptable — priority is a tiebreak, and slug remains
  the final deterministic key in the judge, so mis-stamps degrade gracefully
  rather than breaking determinism.
- **Materialize drop.** If `/design`'s materialize step doesn't honor the
  preserve rule, signals vanish when a stub becomes a full spec. The `spec-format`
  contract is the guard; note it explicitly in the child-stub section.
- **Instruction-only change is unverifiable by unit test.** This is doc/skill/
  agent-prompt content. Validation is by review + a worked example, not `go test`
  (see `## Validation`).

## Validation

This spec produces **no code**, so validation is by contract review and a worked
example, not a test suite:

- **Contract present and consistent.** `core/skills/spec-format/SKILL.md` contains
  a "Child-stub authoring contract" section carrying the mapping table, the
  reciprocity rule (with the outbound-only reason), the prose ⇄ relation sync
  invariant, and the preserve-on-materialize rule. The four authoring surfaces
  (`compose.md`, `product-ideator.md`, the two delivery-lead agents) reference it
  rather than restating the table (grep for a single canonical table).
- **Worked example / dry check.** Run `/compose` (or read a freshly composed
  initiative) against a small 2–3-child initiative with a known same-file seam;
  confirm: (a) every child stub has a `priority:` (and bug children a `severity:`);
  (b) the seam is expressed as a **reciprocal** `conflicts-with` on both children;
  (c) the Wave/overlap prose still exists and every prose seam has a matching
  relation (no orphan prose, no orphan relation).
- **End-to-end signal reaches the judge.** With such an initiative on disk, mark
  one seam child `delivering` and confirm `drive.Check` (the companion consumer)
  now selects by the stamped priority and pauses with `SeamCollision` on the
  conflicting child — proving the authored signals feed the shipped reader. This
  exercises the companion spec's already-passing code; it is an integration
  sanity check, not new test coverage.

## Acceptance Criteria

- WHEN `/compose` writes an initiative's child stubs THE SYSTEM SHALL stamp a
  `priority:` value on **every** child, derived from the sequencing analysis via
  the `spec-format` mapping table (`critical`/`high`/`medium`/`low`).
- WHERE a child stub is a `bug` THE SYSTEM SHALL additionally stamp `severity:`
  from the same impact reasoning.
- WHEN the sequencing analysis identifies two children that touch the same code
  region THE SYSTEM SHALL emit a **reciprocal** `conflicts-with` relation on
  **both** children (one edge on each), because the judge honors outbound-only in
  v1.
- THE SYSTEM SHALL keep the Wave table and overlap narrative prose, and SHALL
  maintain the sync invariant — every named seam in prose has a matching reciprocal
  `conflicts-with` relation, and no `conflicts-with` relation lacks explaining
  prose (no orphan prose, no orphan relation).
- THE SYSTEM SHALL document the child-stub authoring contract — priority mapping,
  reciprocity rule, sync invariant, and preserve-on-materialize rule — in the
  shared `core/skills/spec-format/SKILL.md` so every domain's authoring workflow
  inherits it.
- IF `/design` materializes a child stub into a full spec THEN THE SYSTEM SHALL
  preserve the stub's `priority`/`severity` frontmatter and its `conflicts-with`
  relations (it SHALL NOT silently drop them).
- THE SYSTEM SHALL require no Go change for the above — `conflicts-with` and
  `priority`/`severity` already parse and form edges.

## Completion Ledger

Delivered 2026-07-09. Content/instruction-only change across 5 markdown files under
`core/` + `domains/`; **zero Go** (`go build ./...` green no-op; `git diff --stat`
shows only `.md`). Validation is by contract review + consistency proof, not tests
(no runtime surface). Cold audit: see `delivery-audit.md`.

### Acceptance Criteria

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `/compose` stamps `priority:` on every child stub via the mapping table | DONE | canonical table `core/skills/spec-format/SKILL.md` "Child-stub authoring contract"; instruction in `compose.md`, both delivery leads |
| 2 | `bug` children also stamp `severity:` | DONE | severity rule in `spec-format/SKILL.md`; echoed in compose + leads |
| 3 | Overlap seam → reciprocal `conflicts-with` on both children (outbound-only reason inline) | DONE | two-sided YAML + outbound-only rationale in `spec-format/SKILL.md`; all four surfaces state "both children, one edge each" |
| 4 | Keep Wave/overlap prose + sync invariant (no orphan prose/relation) | DONE | sync-invariant blockquote in `spec-format/SKILL.md`; reinforced in `compose.md` + leads |
| 5 | Contract documented in shared `core/skills/spec-format/SKILL.md` | DONE | new top-level `## Child-stub authoring contract` section |
| 6 | `/design` materialize preserves `priority`/`severity` + `conflicts-with` | DONE | "Preserve on materialize" rule in `spec-format/SKILL.md`; referenced by both leads |
| 7 | No Go change required | DONE | `go build ./...` no-op green; diff is markdown-only under `core/`/`domains/` |

### Changes

| # | Change | Status | Evidence |
|---|---|---|---|
| 1 | `core/skills/spec-format/SKILL.md` — "Child-stub authoring contract" (canonical) | DONE | new section: priority table, stamp-every-child, tiebreak-not-ordering, severity-for-bugs, reciprocal `conflicts-with`, sync invariant, preserve-on-materialize; + 3 frontmatter rows (`conflicts-with`/`priority`/`severity`) |
| 2 | `domains/engineering/commands/compose.md` — instruct emission, point to contract | DONE | "Structured signals on every child stub" subsection; pointer to spec-format, no restated table |
| 3 | `domains/engineering/agents/product-ideator.md` — priority + seam callout in output | DONE | extended "Final output" with concrete `priority` value + overlap/seam callout |
| 4 | `domains/engineering/agents/feature-delivery-lead.md` — emit signals on child stubs | DONE | spec-composition path paragraph, references contract |
| 5 | `domains/engineering/agents/platform-delivery-lead.md` — same on platform path | DONE | platform-composition paragraph (schema-before-code, dual-write, shared-migration seams) |

### Exercise-the-feature check

- [x] Authoring-instruction content, no runtime — exercised via consistency proof:
  `go build ./...` green (no Go touched); `git diff --stat` → only `.md` under
  `core/`/`domains/`; `grep "Foundational anchor" core/ domains/` → single file
  (mapping table canonical); each of the four authoring surfaces references the
  contract by name without restating the table. `hero index` exit 0.
