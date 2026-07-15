# Delivery Plan — Hero Doesn't Lie (`hero-self-consistency`)

Companion to `spec.md`. Sequencing, gates, and the decisions that constrain how this ships.

## Sequence

```
#1 resume-emits-dead-recall-command  (trivial, today, standalone)
      |
#2 spec-contract-enums-unified ──┐   #3 generated-command-refs-validated (independent, parallel)
      |                          |
#4 wire-checks-to-boundaries ◄───┘
      |
#5 spec-state-axes  (large, already designed — parallel from day 1)
```

| # | Slug | Type | Size | Depends on | Can start |
|---|---|---|---|---|---|
| 1 | `resume-emits-dead-recall-command` | bug | trivial | — | now |
| 2 | `spec-contract-enums-unified` | feature | medium | — | now |
| 3 | `generated-command-refs-validated` | feature | small | — | now |
| 4 | `wire-checks-to-boundaries` | feature | medium | **#2 (hard)** | after #2 |
| 5 | `spec-state-axes` | feature | large | — | now (parallel) |

Four of five children can start immediately. Only #4 waits.

## Phases and gates

### Phase 0 — Ship the embarrassment (today)

**#1 only.** Standalone, `trivial`, no dependencies, no coordination.

**Gate:** `rg -n 'hero recall' internal/ core/ docs/` returns zero hits, and the digest test asserts the live command name. Ship it and move on — this phase should not take a day.

### Phase 1 — Fix the contract, gate the refs (parallel)

**#2 and #3 run concurrently.** They share no files: #2 touches the type/status enums, #3 reads the Cobra registry.

**Gate for #2:** `hero check validate` reports 0 `invalid type` and 0 `invalid status`; the anti-drift test fails in both directions when the registry and `core/spec-types/` diverge; a fate decision is recorded for each of the 7 currently-invalid types.

**Gate for #3:** the regression test fails when fed `hero recall <topic>`; the gate scans all six install targets, verified by planting a bogus ref in a non-Claude (`AGENTS.md`) target; the `drift-test:ignore` escape hatch demonstrably suppresses an intentional stale ref.

**#2's gate is not "the enums are merged."** It is "the 7 type-fates are decided." Merging enums without deciding what happens to `context`×114 and `enhancement`×15 just moves the disagreement.

### Phase 2 — Wire the gate (after #2)

**#4 only.** Hard-blocked on #2.

**Gate:** every validator issue class is classified warn or error with recorded rationale; `hero check` on this repo reports 0 error-severity issues and a defensible warn list; the commit hook blocks on a broken type; a `completed` spec naming a nonexistent file errors; the `internal/reconcile/` audit result is documented either way.

**#4 does not begin with wiring.** It begins with the warn-vs-error policy. If the first commit in #4 touches a hook, the spec is being delivered wrong.

### Parallel track — #5

`spec-state-axes` is already designed at `.hero/planning/features/spec-state-axes/spec.md` and runs on its own track from day 1. It is **adopted by reference**: not restubbed, not redesigned, not moved.

**Coordination point:** #2 and #5 both touch `regressed`. #2 only adds it to the valid status set; #5 decides whether verification health belongs in `status` at all. Agree the split before either lands. Similarly, #4's `internal/reconcile/` audit looks for the same axis collapse #5 corrects in `internal/integrity/regression.go` — if the reconciler has it too, decide which spec owns the fix rather than both attempting it.

## The hard dependency: #2 → #4

Not procedural. Substantive.

`wire-checks-to-boundaries` makes the validator run at the commit boundary. `spec-contract-enums-unified` fixes the fact that the validator's type contract disagrees with two other definitions of the same contract — including one other Go enum in the same binary.

Wiring first would gate every commit in the repo against a definition **this initiative has already established is wrong in three directions**. It would reject the 114 `context` specs that `internal/triage/structural.go` accepts, and the 9 `handed_off` specs that `internal/peering/handoff.go:212` itself wrote. A gate enforcing a contract the product doesn't believe in gets disabled immediately — and then the initiative has shipped finding (D) instead of fixing it.

Order matters here in a way it usually doesn't. Do not "start #4 early to parallelize."

## Forbidden measures

Two numbers are excluded from this initiative's reporting. This is not pedantry — an initiative titled "Hero Doesn't Lie" that ships an inaccurate claim about itself has failed on its own terms before it starts.

### Forbidden: "invalid statuses 3 → 0"

**The real number is 14** — `handed_off`×9, `handoff`×2, `designed`×2, `delivered`×1.

And the shape is wrong, not just the magnitude: **11 of the 14 are fixed by correcting the validator, not the corpus.** `internal/peering/handoff.go:212` writes `handed_off` through a shipped, CLAUDE.md-documented feature; `internal/cli/validate.go` simply omits it from `validStatuses`. Those specs are not malformed — the validator is. Reporting "3 → 0" would misstate both the count and the causation, and would frame a validator bug as corpus mess.

**Report instead:** `invalid status` 14 → 0, and separately, "statuses written by one path and rejected by another: 2 → 0" (`handed_off`, `regressed`).

### Forbidden: "`hero check validate` 1019 → 0"

**781 of the 1019 are uncalibrated policy, not defects** — 504 `file not found` + 277 `missing smoke`. A planning spec whose `files:` names code that doesn't exist yet is **correct**; that is what a planning spec is for.

Driving 1019 to 0 would mean either mutilating 781 correct specs or suppressing a check wholesale. Both are worse than the status quo. #4 must set the warn-vs-error policy *first*; only then is there a meaningful number, and it will be "0 error-severity issues," not "0 issues."

**Report instead:** the countable measures in `spec.md` → Measures. Each is a real count of a real disagreement, verified in code.

## Reassignment record

Two findings surfaced during scoping that belong to `spec-prioritization` (P0, planning, parent `get-back-on-track`), which already owns the `horizon` field and specs auto-demote at line 28. Recorded here so they are not lost, and explicitly **not** children of this initiative — attention/horizon work is out of scope by design.

1. **Deterministic horizon rules.** `superseded` and `handed_off` should never be `now`; a missing horizon means untriaged, not `now`. 26 of 91 planning specs lack a horizon; with the 23 explicit ones, **54% of the backlog claims current attention**.

2. **Mechanical-write provenance + dormancy ranking.** Hero's bulk maintenance writes corrupt its own freshness signal by **19x** — 7 specs report 3 days fresh by naive git mtime against a true dormancy of 57 days, because `chore(hero): backfill created: on 42 specs` touched 41 files in one commit. `events.log` is the honest substrate: it carries actor provenance (`"agent":"human/chet-bellows"`) and is immune to mechanical pollution by construction. But it is a completion ledger — 126 `delivery_complete` events against 3 `spec_created` — so closing that coverage gap is the real work, not adopting the file. **Any dormancy ranking built on git mtime would be silently wrong in the direction that pins stale work to `now`**, which is the worst available failure for a prioritization feature.

`get-back-on-track` receives both. This initiative refines `hero-killer-features` and relates to `get-back-on-track` — it does not extend or absorb either.

## Sizing

Initiative declared `large`. Per the `spec-sizing` per-type bands, `large` is **normal** for an initiative — no promotion nudge fires, and no `size_ack` is needed.

This is deliberate. Two `giant` P0 initiatives are already in flight; this one slots beside them rather than competing. Child sizes (`trivial`/`medium`/`small`/`medium`/`large`) are all within normal bands for their types — the `large` on #5 sits at the soft-nudge line for a feature, but #5 is already designed and its scope is settled, so the nudge is moot.

**Watch for drift on #2 and #4** — both are `medium` on the assumption that their central decisions (7 type-fates; warn-vs-error policy) are made once and applied mechanically. If either turns into per-spec judgment across 100+ specs, bump `size:` via `hero size <slug> large` rather than absorbing it silently. Silent absorption is how a `medium` becomes a two-week spec that nobody re-scoped.

## Authoring note — an in-class finding

While writing this initiative's frontmatter, the brief's specified shape turned out to be unparseable, and the discovery is an instance of the initiative's own thesis rather than a footnote:

- `internal/spec/spec.go:649` parses `relations:` **only** as a YAML list of objects (`- target: x` / `kind: y`). `applyRelField` (line 930) accepts only the keys `target` and `kind`/`type`.
- The **map** shape — `relations:` with nested `depends-on: foo` — parses to **zero relations**, silently. Verified empirically against `spec.Parse`.
- **`spec-state-axes` (child #5) uses the map shape**, so all four of its declared relations (`depends-on: acceptance-criteria-graph`, `refines: spec-status-integrity`, `relates-to: [living-contract, tripwire-system, status-reconciliation, spec-prioritization]`) currently parse to nothing. Its edges do not exist in the graph.
- A top-level `refines:` key is **silently dropped** — `refines` appears nowhere in the Go source and is not in the shorthand key list at `internal/spec/spec.go:619`. It survives only as a free-form `kind:` string inside a `relations:` list, which is what this initiative's frontmatter uses.

This is finding (A)/(B)'s pattern in a third place: **Hero writes frontmatter Hero cannot read, and says nothing.** It is not scoped into this initiative — no child was added, per the validated scope — but it is a live candidate for the general claim-checking work the parent spec defers, and `spec-state-axes`'s dropped relations are worth fixing independently regardless.
