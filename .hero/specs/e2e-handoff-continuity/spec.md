---
title: E2E Handoff Continuity — The Magic Guardrail
slug: e2e-handoff-continuity
type: feature
status: completed
priority: P1
domain: engineering
size: medium
created: 2026-06-03
origin: session
tags: [e2e, handoff, continuity, cross-machine, guardrail, next-md, projection]
relations:
  - target: handoff-one-call-simplification
    kind: enables
  - target: next-auto-emit-user-ask
    kind: relates-to
  - target: next-unconditional-commit-staging
    kind: relates-to
  - target: next-as-projection
    kind: relates-to
completed_at: 2026-06-04T06:04:07Z
---

# E2E Handoff Continuity — The Magic Guardrail

> The protected invariant for the entire handoff-simplification initiative. Nothing in
> `handoff-one-call-simplification` (Phase 1/2/3) ships unless this stays green.

## Why this exists

The maintainer's words: *"when things don't go wrong it's incredibly helpful and powerful — I
never have to think about prepping context — so whatever we do, we can't break the magic."*

The magic, mechanically: **finish a turn → context is captured automatically; start a fresh
session or move to another machine → the right context is already loaded.** This spec turns that
sentence into an executable test so the simplification work (dropping files, collapsing the load
path, deleting the migration gate) can proceed aggressively without ever gambling with the loop
the maintainer relies on.

Today's coverage is insufficient: `Test_writeCheckpoint_TeamMode_RoundTripIsIdempotent`
(`internal/cli/checkpoint_test.go:344`) proves project→ingest→re-project is byte-stable **on the
same machine with the same graph**. It does NOT prove the load-bearing case — **another machine,
where `graph.db` is gitignored and did NOT travel, reconstructs the context from the committed
files alone.** That cross-machine leap is the magic, and it is currently unguarded.

## The protected invariant (what "the magic" is, precisely)

> Given a turn's handoff state persisted on machine A and committed, a fresh session on machine B
> — which has ONLY the committed files and an EMPTY graph — loads and surfaces the same
> what-was-asked / what's-next / what-was-learned context, with zero manual prep on either side.

The two halves, both of which must hold:
1. **Same-machine, fresh session** (graph.db present): after checkpoint, the start-of-turn load
   surfaces the last ask + suggested-next + reflections.
2. **Cross-machine** (graph.db absent — the real test): only the committed `.hero/next/<user>.md`
   travels; ingest rehydrates the graph on B; the load surfaces A's context.

## Acceptance Criteria

- **AC-1 — Cross-machine continuity (the core guardrail).** Persist handoff state on a simulated
  machine A (seed UserAsk + NextSuggestion + SessionReflection, run the per-user projection,
  capture the committed file). Construct machine B as a SEPARATE workspace with an **empty graph**
  and ONLY the committed `.hero/next/<user>.md` present (no `graph.db`). Run the real ingest path
  (`handoff.IngestUserFile`), then the real start-of-turn load (`digest.Generate` /
  `hero resume`). Assert B's loaded briefing contains A's ask text, A's suggested-next text, and
  A's reflection. This FAILS if any future change breaks the file→graph→briefing reconstruction.
- **AC-2 — Same-machine fresh session.** After a checkpoint, the start-of-turn load surfaces the
  last ask and suggested-next from the graph (no file-body dependence — `hero resume` is the
  graph-digest path).
- **AC-3 — The federation file is travel-eligible.** The per-user `.hero/next/<user>.md` is NOT
  gitignored and IS written to the path that the staging hook stages. (Guards against a future
  change that drops it from tracking or moves it under a gitignored path — which would silently
  break travel.) The gitignored `*.local.md` and `graph.db` are confirmed NOT travel-eligible.
- **AC-4 — Round-trip idempotence preserved.** The cross-machine reconstruction is idempotent:
  ingest→project→ingest→project on B does not duplicate reflections or drop the agent suggestion
  (extends the existing same-machine idempotence assertion to the cross-machine path).
- **AC-5 — Runs in the default test suite AND as a named e2e suite.** The Go guardrail runs under
  `go test ./...` so it gates every change automatically (the cheap, always-on tripwire). A bash
  sibling `scripts/e2e/handoff.sh` drives the real `hero` binary across two sandbox repos for
  full-fidelity coverage, matching the existing `e2e-*` convention.
- **AC-6 — Extensible to the Phase-1 features.** The test is structured so that when
  `next-auto-emit-user-ask` lands, AC-1's "seed the graph manually" step can be replaced by
  "drive a Stop checkpoint with a transcript payload" and still assert the same outcome — i.e.
  the guardrail will then also prove auto-emit feeds the magic. Leave a clearly-marked seam.

## Design

### Primary: Go guardrail (`internal/cli` or `internal/handoff`)

Reuse the existing helpers seen in `Test_writeCheckpoint_TeamMode_RoundTripIsIdempotent`:
`newTestEnv`, `config.DefaultConfig`, `handoff.RecordAsk/RecordSuggestion/RecordReflection`,
`writeUserHandoffFile`, `handoff.IngestUserFile`, `normalizeUpdatedFrontmatter`. Add the
cross-machine leap the existing test lacks:

```
// Machine A: seed graph, project the per-user file, "commit" it (capture bytes).
envA := newTestEnv(t); seed ask/suggestion/reflection; writeUserHandoffFile(...)
committed := read(.hero/next/<user>.md)

// Machine B: brand-new workspace, EMPTY graph, drop in ONLY the committed file.
envB := newTestEnv(t)            // fresh graph.db, nothing seeded
write(envB/.hero/next/<user>.md, committed)
handoff.IngestUserFile(envB.store, ...)   // rehydrate from the traveled file

// Load on B exactly as a fresh session would.
brief := digest.Generate(envB.store, ...)  // the hero resume path (internal/cli/brief.go)
assert brief contains A's ask, A's suggestion, A's reflection
```

The crux vs. the existing test: **B never shares A's graph.** If a Phase-2 change (e.g. dropping
the per-user file, or moving content into a gitignored surface) breaks travel, AC-1 goes red.

Find the real start-of-turn load entry point before asserting: `digest.Generate`
(`internal/cli/brief.go:102`) is what `hero resume` uses; assert against its output (or the
`hero resume` command output) so the test tracks the *actual* path a fresh session takes, not a
reimplementation.

### Sibling: bash e2e suite (`scripts/e2e/handoff.sh`)

Follow the `lib.sh` convention (`e2e_init`, `e2e_step`/`e2e_assert`, `e2e_finish`). Two sandbox
dirs: A does `hero next ask`/`suggest` + `hero next checkpoint` + `git add`/`commit`; B is a fresh
`git clone` (or file copy of committed `.hero/` only, minus `graph.db`) where `hero next ingest`
+ `hero resume` must surface A's context. Map to AC ids so `hero ac record` picks it up like the
other e2e areas. Register it alongside `discovery.sh`/`traversal.sh`/`onboarding.sh`.

## Test Plan

The guardrail IS the test. Verification that the guardrail itself is honest:
1. Run it green on the current `main` (the magic works today → must pass).
2. **Mutation check:** temporarily break travel (e.g. point the ingest at the wrong path, or skip
   the ingest) and confirm AC-1 goes RED. A guardrail that can't fail is theater — prove it bites.
3. Confirm it runs in `go test ./...` and as `scripts/e2e/handoff.sh`.

## Delivered

- `internal/cli/handoff_continuity_test.go` — the Go guardrail. Tests:
  `Test_HandoffContinuity_CrossMachine` (AC-1), `_SameMachine` (AC-2),
  `_TravelEligibility` (AC-3, via real `git check-ignore`),
  `_CrossMachine_Idempotent` (AC-4), and `_CrossMachine_GuardrailBites`
  (the permanent mutation-check from Test Plan #2). Carries the
  `SEAM(next-auto-emit-user-ask)` marker for AC-6.
- `scripts/e2e/handoff.sh` — the bash sibling (AC-5). Drives the real
  `hero` binary across two sandbox repos: A captures + commits; B is a
  fresh `git clone` (graph.db gitignored → does not travel) that
  ingests and reconstructs A's ask + suggestion. Maps AC-1..AC-4.

**Implementation note — the real load surface for handoff content.**
The design sketch named `digest.Generate` (the `hero resume` brief) as
the thing to assert handoff content against. Reading the code shows the
brief structurally does NOT carry the UserAsk / NextSuggestion /
SessionReflection singletons — it surfaces Mission / In-flight /
Just-changed / Tried / Blocked / Nearby. The handoff content a fresh
session actually consumes surfaces through (1) the graph-query surface
the `hero next ask/suggest/reflection` commands read (`handoff.LatestAsk`
/ `projection.PickUserSuggestion` / `handoff.RecentReflections`) and
(2) the re-projected `.hero/next/<user>.md`. The Go guardrail asserts
reconstruction against BOTH of those genuine surfaces and ALSO runs
`digest.Generate` on the rehydrated B graph to prove the `hero resume`
load path executes end-to-end against the traveled-and-ingested state.
This keeps the guardrail honest — it tracks the path a fresh session
takes rather than a brief section that cannot carry this content today.

## Boundaries

- Does NOT test auto-emit yet (that feature isn't built) — but leaves the seam (AC-6) so it
  extends cleanly when `next-auto-emit-user-ask` lands.
- Does NOT assert on SNAPSHOT.md/QUEUE.md content — those are being dropped as handoff files; the
  magic does not depend on them, and the guardrail must not encode a dependency we're removing.
- Asserts on the per-user federation file + graph + resume briefing only — the irreducible core.

## Kickoff

You're building the executable guardrail that protects Hero's handoff "magic" — the one thing the
whole simplification initiative is forbidden to break. Read this spec, then
`handoff-one-call-simplification` for context. The magic = automatic capture + automatic load,
zero prep; the load-bearing invariant is **cross-machine reconstruction from committed files
alone**.

**Pick up at:** write the Go guardrail in `internal/cli` (mirror
`Test_writeCheckpoint_TeamMode_RoundTripIsIdempotent` at `checkpoint_test.go:344` for setup, then
add the machine-B-with-empty-graph leap). Assert against the real `hero resume` load path
(`digest.Generate`, `internal/cli/brief.go:102`). Then add `scripts/e2e/handoff.sh` following
`scripts/e2e/lib.sh`. **Prove the guardrail bites** (mutation check, Test Plan #2) before calling
it done — a green-no-matter-what test is worse than none.

→ `internal/cli/checkpoint_test.go:344`, `internal/cli/brief.go:102`, `internal/handoff/ingest.go:29`, `internal/projection/user_handoff.go:31`, `scripts/e2e/lib.sh`

## Recap

The handoff "magic" — finish a turn, context captured automatically; start fresh or on another
machine, context already loaded — is Hero's core value and the binding constraint on the
simplification work. Today only same-machine idempotence is tested; the load-bearing case
(another machine, empty graph, only committed files) is unguarded. This spec builds that
guardrail: a Go test that persists on A, reconstructs on a fresh-graph B from the committed file
alone, and asserts the briefing surfaces A's context — plus a `scripts/e2e/handoff.sh` sibling for
full-binary fidelity. It must be proven to bite (mutation check) and runs in `go test ./...` so it
gates every change. Nothing in the simplification ships unless this stays green.
