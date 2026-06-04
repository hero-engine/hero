---
title: Resume Brief Surfaces Handoff — Close the Load Half of the Magic
slug: resume-brief-surfaces-handoff
type: feature
status: planning
priority: high
severity: high
size: small
domain: engineering
created: 2026-06-04
origin: session
root_cause_class: design
relates-to:
  - handoff-one-call-simplification
  - next-auto-emit-user-ask
  - e2e-handoff-continuity
  - next-as-projection
---

# Resume Brief Surfaces Handoff — Close the Load Half of the Magic

## Context

The handoff "magic" is a loop: **capture** at end of turn, **travel** via git, **load** at
start of turn. Phase 1 fixed capture ([next-auto-emit-user-ask](../../specs/next-auto-emit-user-ask/spec.md))
and travel ([next-unconditional-commit-staging](../../specs/next-unconditional-commit-staging/spec.md)).
This spec fixes the **load** half — the part you actually *see* at session start.

The [e2e-handoff-continuity](../../specs/e2e-handoff-continuity/spec.md) guardrail surfaced the
gap: **`hero resume` — the auto-fired session-start brief — does NOT surface the handoff
singletons** (last user ask, suggested next, recent reflections). It carries Who-you-are /
In-flight / Just-changed / Tried / Blocked / Mission / Acceptance / Nearby (the `digest`
section functions), but nothing that answers *"what did I last ask for, and what's the next
step?"* — the single most expensive thing for a fresh session to recover.

So today the auto-emitted UserAsk lands in the graph and renders into `.hero/next/<user>.md`,
but the brief the model is handed at `SessionStart` doesn't include it. Capture ✓, travel ✓,
but the greeting at the door is thin. This closes that.

This is a concrete slice of `handoff-one-call-simplification`'s Phase-3 "one start-of-turn
`resume` that emits the briefing (last ask, suggested-next, reflections...)".

## Root Cause

`internal/digest/digest.go` assembles the brief from section builders
(`whoYouAreSection`, `sprintSection`, `inFlightSection`, `justChangedSection`, `triedSection`,
`blockedSection`, `missionSection`, `acceptanceSection`, `nearbySection`). **None reads the
handoff package.** Worse, `digest.Options` (`digest.go:31-39`) carries `AuthorEmail` (for the
Person lookup) but NOT the `user` slug + `domain` that the handoff singletons are keyed by
(`handoff.LatestAsk(store, user, repoKey, domain)`, `handoff.go:210`). So the digest
*structurally cannot* read the handoff nodes today — the key isn't even threaded in.

The handoff read functions already exist and are correct:
- `handoff.LatestAsk(store, user, repoKey, domain)` — `internal/handoff/handoff.go:210`
- `handoff.LatestSuggestion(store, user, repoKey, domain)` — `handoff.go:226`
- `handoff.RecentReflections(store, user, repoKey, domain, limit)` — `handoff.go:245`

This is a wiring gap, not a missing capability — the same shape as the rest of this initiative.

## Goal

`hero resume` (and any `digest.Generate` consumer) surfaces a handoff section near the top of
the brief: the last user ask, the suggested-next prompt, and the most recent reflections for the
current user — so a fresh or cross-machine session opens already knowing what was asked and
what's next, with no manual `hero next` and no reading the file by hand.

## Acceptance Criteria

- WHEN `hero resume` runs for a user who has a recorded `UserAsk` THE SYSTEM SHALL include a
  handoff section in the brief showing that last ask.
- WHEN a `NextSuggestion` exists for the user THE SYSTEM SHALL show it as the suggested next step
  in the same section; WHEN reflections exist THE SYSTEM SHALL show the most recent N.
- THE SYSTEM SHALL place the handoff section near the TOP of the brief (it is the highest-value
  "where was I" context — above In-flight/Nearby), subject to the existing budget mechanism.
- WHEN the user has no handoff nodes (fresh repo) THE SYSTEM SHALL omit the section entirely (no
  empty/placeholder clutter in the brief).
- THE SYSTEM SHALL key the lookup by the same `(user, repo, domain)` triple the projection and
  auto-emit use, so the brief, the `.hero/next/<user>.md` file, and the graph all agree.
- THE SYSTEM SHALL degrade gracefully: a handoff-read error logs and is skipped, never failing
  `hero resume` (same best-effort contract as the other sections / the auto-emit path).
- THE SYSTEM SHALL surface the handoff content for the cross-machine case: after ingest on a
  fresh-graph machine B, `hero resume` on B shows machine A's ask/suggestion (this is the
  guardrail extension below).

## Approach

Wire the handoff key through `digest.Options`, add a `handoffSection`, and place it high.

1. **Thread user + domain into `digest.Options`** (`internal/digest/digest.go:31-39`). Add
   `User string` and `Domain string` fields. (Keep `AuthorEmail` — it powers the Person lookup
   and focus scoring; the handoff key is a separate concern.)

2. **Populate them in `hero resume`** (`internal/cli/brief.go:93-100`). `brief.go` is in package
   `cli`, so it can resolve the same way auto-emit does: load config, then
   `User: nextUserSlug(cfg)` and `Domain: graph.DomainFor(cfg, graph.IntrinsicActive)`. (Confirm
   `brief.go` already has or can cheaply load `cfg`; `runNextCheckpoint`'s `autoEmitUserAsk`
   shows the exact derivation to mirror.) **Confirmed: `brief.go` is the ONLY external caller of
   `digest.Generate`** (the MCP prime path uses `BuildPrimeContext`/`spec.Discover`, not the
   digest), so this is a single call site — no caller hunt. **Confirmed: no import cycle —
   `internal/handoff` does not import `internal/digest`**, so `digest`→`handoff` is safe.

3. **Add `handoffSection(store, opts, budget)`** (`internal/digest/digest.go`, beside the other
   section builders). If `opts.User == ""` return an empty section (omitted). Otherwise:
   - `handoff.LatestAsk(store, opts.User, opts.RepoKey, opts.Domain)` → "Last ask: …"
   - `handoff.LatestSuggestion(...)` → "Suggested next: …" (use the recorded suggestion; the
     mechanical floor `projection.PickUserSuggestion` is a nice-to-have but pulls an extra
     dependency — see Boundaries; prefer the handoff-package read here and leave the floor as a
     follow-up).
   - `handoff.RecentReflections(..., limit)` → "Recent: …" (small N, e.g. 3).
   - Any error → log + return empty section (best-effort).
   Mind the import direction: `digest` importing `internal/handoff` must not create a cycle
   (confirm `handoff` does not import `digest`). If a cycle exists, resolve by reading the nodes
   via the graph queries directly in digest, mirroring `handoff`'s SQL — but prefer reusing the
   handoff funcs.

4. **Place it near the top** in `Generate`'s section assembly and give it a budget slice in
   `planFor`/the section-budget fractions (`digest.go:72+`). It should sit just after
   Who-you-are (identity) and before In-flight — the "what was I doing" answer comes first. Keep
   it within the soft-budget machinery so it trims like everything else.

5. **Markdown + JSON both inherit it** automatically (it's a `BriefSection`), so `hero resume`,
   `--json`, and MCP callers all get it with no extra rendering code.

## Test Plan

1. **Section present when handoff exists (unit, `internal/digest`):** seed a graph with a
   UserAsk + NextSuggestion + reflections for `(user, repo, domain)`; call `digest.Generate`
   with `User`/`Domain` set; assert a handoff section exists with the ask/suggestion/reflection
   text, and that it is ordered above the In-flight section.
2. **Omitted when empty:** no handoff nodes (or `User == ""`) → no handoff section in the brief.
3. **Keyed correctly:** a node under a *different* user/domain does not leak into this user's
   brief (mirror the existing cross-repo/cross-user isolation tests).
4. **Best-effort:** force a handoff-read error → `Generate` still returns a brief (section
   skipped), no error propagated.
5. **`hero resume` integration (`internal/cli`):** seed handoff nodes, run the resume path,
   assert the rendered markdown contains the last ask. Reuse the brief test harness.
6. **Extend the continuity guardrail (`internal/cli/handoff_continuity_test.go`).** The guardrail
   currently asserts cross-machine reconstruction against the handoff graph queries + the
   re-projected file, and explicitly runs `digest.Generate` on B only to prove the load path
   executes (the design note records that the brief didn't carry the content). Now that it does,
   STRENGTHEN AC-1 (and the AutoEmit variant) to assert B's `digest.Generate` brief **contains**
   A's ask/suggestion — closing the gap the guardrail documented. Update the design-note comment
   in `handoff-one-call-simplification` accordingly.

Regression: the other brief sections, budget trimming, and `--json` output must be unaffected;
run the full `internal/digest` and `internal/cli` suites.

## Boundaries

- **NOT auto-deriving a *good* suggested-next.** Show the recorded `NextSuggestion`; if none,
  show nothing (or the mechanical `PickUserSuggestion` floor as an optional follow-up if the
  import direction is clean). No model-synthesized next-step here — same boundary as
  `next-auto-emit-user-ask`.
- **NOT changing the handoff node schema or the projection/file rendering** — only adding a
  consumer (the brief) of the existing nodes.
- **NOT redesigning the budget machinery** — the handoff section participates in it like any
  other section.
- **NOT touching SNAPSHOT/QUEUE** — separate Phase-2 concern.

## Risks

- **Import cycle (`digest` → `handoff`).** Confirm `handoff` doesn't import `digest`. If it does,
  read the nodes via graph SQL in digest instead of the handoff funcs. Low risk — `handoff` is a
  low-level package.
- **User/domain resolution in non-CLI callers.** `digest.Generate` may be called where the user
  slug isn't known (some MCP path). Threading `User: ""` there is a safe no-op (section omitted),
  so it degrades, not breaks. Enumerate callers and set what's known.
- **Budget pressure.** Adding a top section could push lower sections past budget. Give it a
  modest slice and confirm the soft-target trimming still produces a sane brief on a rich repo.

## Kickoff

Closes the *load* half of the handoff magic: make `hero resume` actually show the last user ask /
suggested-next / reflections that auto-emit now captures and staging now ships. Today the
`digest` brief structurally can't — `digest.Options` doesn't even carry the user/domain the
handoff nodes are keyed by.

**Status:** planning — diagnosis complete, read against source. No code yet.

**Pick up at:** (1) add `User`/`Domain` to `digest.Options` (`internal/digest/digest.go:31`);
(2) populate them in `hero resume` (`internal/cli/brief.go:93`) via `nextUserSlug(cfg)` +
`graph.DomainFor(cfg, graph.IntrinsicActive)` — mirror `autoEmitUserAsk` in `checkpoint.go`;
(3) add `handoffSection` beside the other section builders, reading `handoff.LatestAsk` /
`LatestSuggestion` / `RecentReflections`, omitted when `User==""` or empty, best-effort on error;
(4) place it just after Who-you-are and before In-flight, inside the budget machinery; (5)
extend the continuity guardrail to assert B's brief now contains A's ask/suggestion.

→ `internal/digest/digest.go:31` (Options), `internal/digest/digest.go:97` (Generate/section order),
`internal/cli/brief.go:93` (resume Options), `internal/handoff/handoff.go:210` (read funcs),
`internal/cli/handoff_continuity_test.go` (guardrail extension)

**Skip:** model-synthesized suggested-next (keep recorded value + optional mechanical floor);
SNAPSHOT/QUEUE; budget redesign.

## Recap

`hero resume` — the auto-fired session-start brief — doesn't surface the handoff singletons
(last ask / suggested-next / reflections), because the `digest` package has no handoff section
and `digest.Options` doesn't carry the `(user, domain)` key the nodes need. So Phase 1's
auto-captured, now-travelling ask never reaches the brief the model actually reads at session
start — capture ✓, travel ✓, load ✗. The fix threads user+domain into `digest.Options`, adds a
top-placed `handoffSection` reading the existing `handoff.Latest*` funcs (omitted when empty,
best-effort on error), and extends the continuity guardrail to assert the cross-machine brief now
carries A's context — closing the load half of the magic. Small, wiring-shaped, behind the
guardrail.
