---
title: claimedByMatches "you"/"me" sentinels collide with real git identities
slug: claim-matches-sentinel-collision
type: bug
status: planning
severity: low
root_cause_class: code
priority: low
tags: [serve, dashboard, identity, plate]
created: 2026-05-19
relates-to: [dashboard-user-identity-os-env-mismatch]
---

# claimedByMatches "you"/"me" sentinels collide with real git identities

## Kickoff

A tiny matcher in the dashboard's plate treats the literal strings `"you"`
and `"me"` as "match the current viewer" regardless of identity. Now that
identity comes from `git config user.name`, a user whose git name is
literally `you` or `me` would cross-match other people's claims.

**Status:** planning — backlog spec carved out of `dashboard-user-identity-os-env-mismatch`. No code yet.

**Pick up at:** confirm there are no in-repo writers emitting `claimed_by: you` or `claimed_by: me` (only matcher-side sentinels exist today), then implement **Option C** — reject `you`/`me` as identity inputs in `gitutil.UserName()` normalization, and delete the `cb == "you" || cb == "me"` arm from `claimedByMatches`.

→ `internal/serve/pages/now/data/plate.go:60-70`

**Files:** `internal/serve/pages/now/data/plate.go`, `internal/gitutil/gitutil.go`, `internal/serve/pages/now/data/plate_test.go`

## Issue

Surfaced during delivery of `dashboard-user-identity-os-env-mismatch`
(commit `8d2872d`, 2026-05-19). Flagged in that spec under
"Secondary Defects #1" and explicitly deferred — gets its own ticket now
so it isn't lost.

No real user has hit this. Filed as low/code so it sits in the backlog
behind real work.

## Investigation

### The matcher

`internal/serve/pages/now/data/plate.go:60-70`:

```go
// claimedByMatches accepts a few common spellings of "claimed by me".
func claimedByMatches(claimedBy, user string) bool {
    if claimedBy == "" {
        return false
    }
    cb := strings.ToLower(strings.TrimSpace(claimedBy))
    u := strings.ToLower(strings.TrimSpace(user))
    if u == "" {
        return false
    }
    return cb == u || cb == "you" || cb == "me"
}
```

Any spec whose `claimed_by` frontmatter is the literal string `you` (or
`me`) matches *every* current user the matcher is asked about.

### Callers

`claimedByMatches` has exactly one caller:

- `internal/serve/pages/now/data/plate.go:38` — `LoadPlate` filters
  `spec.Discover()` results down to the active user's specs for the
  "On your plate" cards on `/now`.

So the blast radius is the plate widget on the dashboard. Nothing else
in `internal/serve/` calls this function.

### Writers (do any emit `you` / `me`?)

Grepped every `claimed_by` writer in the tree:

| File | Lines | What it writes |
|------|-------|----------------|
| `internal/cli/new.go` | 517, 785 | `fmt.Sprintf("claimed_by: %s\n", claimedBy)` from a resolved agent string |
| `internal/tracking/tracking.go` | 120 | `spec.SetFrontmatterField(..., "claimed_by", agent)` |
| `internal/index/index.go` | 1077, 1088 | SQLite mirror, value passed from above |
| `internal/spec/graph_ingest.go` | 252 | graph property mirror |

All writers route through `resolveAgent()` → `gitutil.UserName()` →
git config / `$USER` / `$USERNAME` / `"unknown"`. **No writer in the
current tree intentionally emits `you` or `me` as a sentinel value.**

The one place `"you"` appears as a literal is the *display* fallback at
`plate.go:90` — `safeStr(s.ClaimedBy, "you")` — which substitutes the
word "you" when rendering an unclaimed card's meta label. That's a
display string, not a stored sentinel, and it never round-trips back
through the matcher.

The matcher's `you`/`me` arms appear to be defensive code for a sentinel
convention that was never actually adopted.

### Root cause

`claimedByMatches` treats two ordinary English words as wildcard tokens
that match the current viewer. This was harmless when identity came
from `$USER` (no one's `$USER` is literally `you`). Now that identity
comes from `git config user.name`, which is user-settable to anything
including `you` or `me`, the wildcards can theoretically be triggered
by a real identity — causing one user's view to surface another user's
claimed specs.

### Severity

Low. The git names `you` and `me` are legal but vanishingly unlikely.
There is no known reproduction and no user report. Worth fixing as
hygiene, not as triage.

## Goal

`claimedByMatches` returns `true` only when `claimed_by` equals the
viewer's normalized identity. There are no wildcard identity strings
in the matcher, and the identity resolver refuses to produce `you` or
`me` as identity values so the contract is enforced on both ends.

## Approach

Three options considered:

- **Option A — explicit semantics.** Introduce dedicated tokens like
  `<self>` and `<unclaimed>`, update every writer + matcher together.
  Cleanest model, but no writer emits the old sentinels today, so this
  is paying for a refactor we don't need.
- **Option B — namespaced sentinels.** Rename to `__self__` / `__me__`
  so no real name collides. Same problem as A: no writer uses them.
- **Option C — remove the sentinel + reserve the names.** Drop the
  `cb == "you" || cb == "me"` arm from `claimedByMatches`. Add a
  reservation in `gitutil.UserName()` so an identity that normalizes
  to `you` or `me` falls through to the next ladder rung (with a
  diagnostic). Lowest cost, smallest surface area, matches the fact
  that the sentinels are dead code.

**Recommended: Option C.** It removes the bug without inventing new
contract surface, leaves the display fallback at `plate.go:90` alone
(that's just a UI label, not a matched value), and adds one tiny
invariant to the identity resolver that already owns this contract.

## Acceptance Criteria

- THE SYSTEM SHALL NOT treat the literal strings `"you"` or `"me"` as
  wildcard matches inside `internal/serve/pages/now/data/plate.go`'s
  `claimedByMatches`.
- WHEN comparing `claimed_by` to the active user THE SYSTEM SHALL match
  only on case-insensitive, whitespace-trimmed string equality.
- IF `gitutil.UserName()` resolves an identity that normalizes to
  `"you"` or `"me"` THEN THE SYSTEM SHALL skip that source and continue
  down the resolution ladder.
- WHERE `gitutil.UserName()` rejects a `"you"`/`"me"` value THE SYSTEM
  SHALL emit a one-line diagnostic identifying the rejected source so
  the user knows why their git config was ignored.
- THE SYSTEM SHALL preserve the display fallback at `plate.go:90`
  (`safeStr(s.ClaimedBy, "you")`) — the word "you" remains a valid UI
  label for an unclaimed card, just not a matched value.

## Changes

1. `internal/serve/pages/now/data/plate.go`
   - Delete the `cb == "you" || cb == "me"` arm in `claimedByMatches`
     (line 69). The function becomes a straight case-insensitive
     trimmed equality check.
   - Update the `// Match either by username or by "you" sentinel…`
     comment at lines 36-37 and the `// claimedByMatches accepts a few
     common spellings…` comment at line 59 to reflect the new contract.
   - Leave `safeStr(s.ClaimedBy, "you")` at line 90 untouched — that's
     display-only.
2. `internal/gitutil/gitutil.go`
   - In `UserName()`, after normalization at each ladder rung, treat a
     normalized value of `"you"` or `"me"` as if the rung produced an
     empty value and fall through to the next source.
   - Emit a single `log.Printf` (or whatever pattern the existing
     startup diagnostic uses — see commit `8d2872d`) noting which
     source produced the rejected value.
3. `internal/serve/pages/now/data/plate_test.go`
   - Add a regression test: a spec with `claimed_by: you` and a viewer
     whose normalized identity is `chet-bellows` no longer matches.
   - Add a regression test for the converse: a spec with
     `claimed_by: chet-bellows` and a viewer of `you` no longer
     matches (the viewer string is whatever fell out of the resolver
     after rejection, but this pins the matcher contract directly).
4. `internal/gitutil/gitutil_test.go`
   - Add a test: when `git config user.name` returns `you`, the
     resolver falls through to `$USER`.
   - Add a test: when both git config and `$USER` resolve to `me`,
     the resolver falls through to `$USERNAME` (then `unknown`).

## Boundaries

- **Out of scope:** other quirks of `claimedByMatches` — case folding
  policy, whitespace handling beyond `TrimSpace`, alias resolution,
  multiple identity matching. If any of those need work, they get
  their own spec.
- **Out of scope:** introducing structured identity types (an
  `Identity` struct with name + email + aliases). That was already
  flagged as deferred in `dashboard-user-identity-os-env-mismatch`.
- **Out of scope:** the display fallback `safeStr(s.ClaimedBy, "you")`
  at `plate.go:90`. It's a UI string, not a matched value, and
  changing it would shift the visual design of an unclaimed plate
  card — a separate concern.
- **Out of scope:** sweeping the codebase for other "sentinel string
  vs. real value" patterns. This spec is one matcher, one resolver.

## Risks

- Fixtures or tests anywhere in the tree that happen to set
  `claimed_by: you` expecting the matcher to wildcard will start
  failing. Search before changing: `rg "claimed_by:\s*(you|me)\b"`
  across `.hero/`, `internal/`, and `test*/` to catch any test
  fixtures that rely on the current behavior.
- A user whose actual git name is `You` or `Me` (after case folding)
  will now see a startup diagnostic and have their git identity
  ignored. The fallback ladder still produces *some* identity, so the
  dashboard keeps working — but the diagnostic should be clear enough
  that they understand why.

## Validation

1. Unit tests in `plate_test.go` and `gitutil_test.go` per Changes #3
   and #4 pass.
2. `rg 'cb == "you"|cb == "me"' internal/` returns no matches after
   the change.
3. Manual: set `git config user.name "you"` in a throwaway worktree,
   start `hero serve`, confirm the startup diagnostic prints and the
   dashboard renders with the next-ladder identity (e.g. `$USER`),
   not with `you`.
4. Full `go test ./internal/serve/... ./internal/gitutil/...` passes.
