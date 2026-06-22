---
title: "Helpful spec-not-found errors for hero spec verify"
slug: verify-slug-resolution-hints
type: enhancement
status: completed
domain: ""
size: small
priority: medium
created: 2026-06-21
relates-to: [spec-lifecycle-hygiene-breakdown]
tags: [cli, dx, error-messages, verify]
completed_at: 2026-06-22T01:03:26Z
---
# Helpful spec-not-found errors for hero spec verify

## Goal
When `hero spec verify <slug>` can't resolve a slug to a discoverable spec, the
error becomes diagnostic and directional instead of a dead end. Specifically:
a case-only mismatch resolves silently; an unmaterialized initiative-child slug
reports its owning initiative and directs the user to `/design`; a near-miss
suggests the closest real slug ("did you mean …?"); and only a genuine no-signal
miss falls back to today's bare message. "Done" = the four branches above are
implemented behind one shared helper in `internal/spec`, wired into
`runVerify`, with table-driven tests covering each branch.

## Kickoff

Make `hero spec verify <slug>`'s "not found" error helpful — detect
initiative-child slugs, suggest near-misses, match case-insensitively — instead
of dead-ending the user.

**Status:** delivered — `ResolveOrHint` shipped in `internal/spec/resolve_hint.go`
and wired into `runVerify`. All 7 ACs DONE, cold audit verdict SHIP, tests green.

**Pick up at:** nothing required — delivery complete. Optional follow-up (noted in
Approach, not delivered): adopt `ResolveOrHint` in the other genuinely
slug-resolving commands — `loadSpecBySlug` (`internal/cli/size.go`),
`findSpecBySlugOrPath` (`internal/cli/drift.go`), `findSpecBySlug`
(`internal/cli/claim.go`). Separately, `TestMarkdownInvocationsResolveAgainstRootCmd`
fails on a clean tree (pre-existing, unrelated markdown-drift check) — worth its
own diagnose.

→ `.hero/specs/verify-slug-resolution-hints/spec.md`

**Files shipped:** `internal/spec/resolve_hint.go` (new), `internal/spec/resolve_hint_test.go` (new), `internal/cli/verify.go`, `internal/cli/verify_test.go`.

## Problem
`hero spec verify <slug>` resolves a spec by walking `spec.Discover(heroDir)` and
matching `s.Slug == args[0]` exactly (`internal/cli/verify.go:86-92`). On no match
it returns a bare `fmt.Errorf("spec %q not found", args[0])` (line 94). That
message carries zero diagnostic signal.

In a real incident, a downstream project delivering an initiative had a model
pass an initiative-child slug (e.g. `R-01`) that had **never been materialized as
its own spec** — the child existed only as a row in the initiative's
`## Children` table, not as a real `spec.md` on disk. Discover correctly found no
matching spec, and the bare "not found" gave the model nothing to act on. It
confabulated a phantom tool limitation ("`hero spec verify` doesn't support
initiative-child slugs") and rationalized skipping the spec lifecycle entirely.

The root cause is **not** a missing resolution feature. Slug resolution is flat
and correct: any materialized child spec.md IS findable by its slug. The defect
is purely that the failure path is informationally empty — it neither tells the
user the slug exists as an unmaterialized child nor offers a near-miss
suggestion. This spec turns that dead end into a next action.

## Acceptance Criteria
- WHEN a slug differs from a discoverable spec's slug only by letter case THE SYSTEM SHALL resolve to that spec and verify it (no error, no prompt).
- WHEN a slug does not resolve to any spec AND it appears as the first-column entry of some initiative spec's `## Children` table THE SYSTEM SHALL report the owning initiative slug and direct the user to run `/design` to materialize the child before verifying.
- WHEN a slug does not resolve exactly AND a discoverable slug is within edit distance 2 (or differs only by case) THE SYSTEM SHALL print "no spec `<slug>` found — did you mean `<closest>`?" naming up to 3 closest slugs.
- IF a slug matches both an initiative-child row and a near-miss spec THEN THE SYSTEM SHALL prefer the initiative-child message (it is the more specific, more actionable signal).
- IF a slug resolves to none of the above (no case-insensitive hit, no initiative-child row, no near-miss within threshold) THEN THE SYSTEM SHALL fall back to the existing `spec %q not found` message unchanged.
- THE SYSTEM SHALL tolerate `## Children` heading variants (e.g. `## Children — six features`), tables with or without leading/trailing pipes, and first-column entries written as either a bare slug or a markdown link `[slug](path)`.
- THE SYSTEM SHALL implement the resolution-and-hint logic in a single shared helper in `internal/spec` so the behavior is unit-testable independently of the CLI command.

## Approach/Design

### Shape: one shared helper in `internal/spec`
Add a helper to `internal/spec` (e.g. `ResolveOrHint`) with this shape:

```go
// ResolveOrHint finds the spec whose slug matches `slug`. On an exact or
// case-insensitive match it returns (spec, ""). On no match it returns
// (nil, hint) where hint is a human-readable next-action string, or
// (nil, "") when no signal applies (caller falls back to its bare message).
func ResolveOrHint(slug string, specs []*Spec) (*Spec, string)
```

Living in `internal/spec` (not `internal/cli`) keeps it unit-testable with
constructed `[]*Spec` fixtures and lets other slug-resolving commands adopt it
later without an import cycle.

### Resolution order (in the helper)
1. **Exact match** — `s.Slug == slug`. Return `(s, "")`. Preserves today's fast path.
2. **Case-insensitive match** — `strings.EqualFold(s.Slug, slug)`. Return `(s, "")` — resolve silently, no prompt. (Satisfies AC-1.)
3. **Initiative-child detection** — for each spec where `s.Type == TypeInitiative`, read its children-section body and scan the first column for a slug matching `slug` (case-insensitive). On a hit, return `(nil, hint)` where hint names the owning initiative and points to `/design`. Example hint:
   > `R-01` is listed as a child of initiative `retrieval-quality` but hasn't been designed into its own spec yet — run `/design` to materialize it before verifying.
4. **Fuzzy "did you mean"** — compute Levenshtein distance from `slug` to every discoverable `s.Slug`; collect those with distance ≤ 2 (case-only diffs already handled in step 2, but keep them eligible here as distance 0–1 for robustness), sort ascending, take up to 3. Return `(nil, hint)`:
   > no spec `R-01` found — did you mean `r-01-foo`?
   With multiple suggestions: `… did you mean `r-01-foo`, `r-01-bar`?`
5. **No signal** — return `(nil, "")`. Caller emits the unchanged bare message.

Order matters: initiative-child (step 3) precedes fuzzy (step 4) so the more
actionable message wins on a tie (AC-4).

### Children-table parsing — reuse, don't reinvent
`internal/spec/ledger.go` already has `splitTableRow(line string) []string`
(line 163) and `parseTable(body string) []LedgerRow` (line 101). `splitTableRow`
is the right reusable primitive: it strips leading/trailing pipes and trims
cells, exactly what child-row parsing needs. **Do not hand-roll pipe-splitting.**
`parseTable` itself is ledger-specific (it sniffs for `Status`/`#`/`Criterion`
headers), so don't call it directly for children — instead reuse `splitTableRow`
to extract the first cell of each non-separator table row.

Section access: `parseSections` (`internal/spec/spec.go:869`) lowercases each
`## ` heading and stores the raw body in `s.Sections[key]`. The canonical key is
`children`, but the heading-variant `## Children — six features` keys as
`children — six features`. So look up the children body by **prefix-matching**
`s.Sections` keys against `"children"` (any key that `== "children"` or
`strings.HasPrefix(key, "children")`), not by exact key.

First-column normalization: a child cell may be a bare slug
(`configurable-reranking`) or a markdown link (`[project-charter](../../…)`).
Strip markdown-link syntax to the link text before comparing — e.g. extract the
`[text]` portion, fall back to the raw cell. Compare case-insensitively.

### Wiring into the CLI
In `runVerify` (`internal/cli/verify.go`), replace the loop + bare error
(lines 86–95) with a call to the helper:

```go
target, hint := spec.ResolveOrHint(args[0], specs)
if target == nil {
    if hint != "" {
        return fmt.Errorf("%s", hint)
    }
    return fmt.Errorf("spec %q not found", args[0])
}
```

The fallback line is byte-for-byte today's message, so the no-signal path is a
strict no-op behavior change.

### About `hero complete` — corrected scope note
The design brief assumed `hero complete` shares verify's resolve-by-slug
pattern. It does not. `runComplete` (`internal/cli/complete.go:45-50`) takes a
**spec path** (`Use: "complete <spec-path>"`), parses it with `spec.ParseFile`,
and for any work spec actively redirects the caller to `hero spec verify`
(lines 61–67). There is no slug-resolution branch to harden there, and adding
one would contradict its path-based contract. **`complete` is therefore out of
scope** — see Out of Scope. The verify path is where the incident actually
occurred and is the sole integration point. If a future spec wants this helper
in the genuinely slug-resolving commands, the natural candidates are
`loadSpecBySlug` (`internal/cli/size.go:335`), `findSpecBySlugOrPath`
(`internal/cli/drift.go:101`), and `findSpecBySlug` (`internal/cli/claim.go:323`)
— noted here as a follow-up, not delivered now.

## Changes
- `internal/spec/resolve_hint.go` (new) — `ResolveOrHint(slug, specs)` helper plus
  `childrenSectionBody`, `childTableHasSlug`, `normalizeCell`, `fuzzySuggestions`,
  and a self-contained `levenshtein`. Reuses `splitTableRow` for table cells.
- `internal/spec/resolve_hint_test.go` (new) — table-driven coverage of all
  branches (exact, case-only, initiative-child, heading variant + markdown-link
  row, pipe tolerance, fuzzy single/multiple, child-beats-fuzzy tie, no-signal,
  short-slug guard, Levenshtein).
- `internal/cli/verify.go` — `runVerify` not-found branch now calls
  `spec.ResolveOrHint`; the bare `spec %q not found` fallback is unchanged.
- `internal/cli/verify_test.go` — added `TestVerify_UnmaterializedInitiativeChild`
  and `TestVerify_NoSignalBareMessage`.

## Completion Ledger

| # | Item | Status | Evidence |
|---|------|--------|----------|
| AC-1 | Case-only mismatch resolves & verifies, no prompt | DONE | `internal/spec/resolve_hint.go:34-38` (`strings.EqualFold`); test `TestResolveOrHint/case-only_mismatch_resolves_silently` (`internal/spec/resolve_hint_test.go`). |
| AC-2 | Unmaterialized initiative-child reports owning initiative + `/design` | DONE | `resolve_hint.go:41-55`; live exercise: `spec verify configurable-reranking` → child hint (see Exercise check); tests `TestResolveOrHint/unmaterialized_initiative_child`, `TestVerify_UnmaterializedInitiativeChild`. |
| AC-3 | Near-miss (dist ≤2) prints "did you mean", ≤3 slugs | DONE | `resolve_hint.go:58-65` + `fuzzySuggestions`/`levenshtein`; live exercise on `verify-slug-resolution-hint` typo; tests `.../fuzzy_near-miss_single_suggestion`, `.../multiple_suggestions_capped_at_3`. |
| AC-4 | Initiative-child preferred over near-miss on tie | DONE | Step 3 precedes step 4 in `ResolveOrHint`; test `.../initiative-child_beats_fuzzy_on_tie`. |
| AC-5 | No-signal falls back to unchanged `spec %q not found` | DONE | `internal/cli/verify.go` fallback byte-for-byte unchanged; live exercise on `zzz-nonexistent-slug-xyz`; test `TestVerify_NoSignalBareMessage` asserts exact string. |
| AC-6 | Tolerate heading/pipe variants & bare-slug or `[slug](path)` cells | DONE | `childrenSectionBody` (prefix-match), `splitTableRow` reuse, `normalizeCell`; tests `.../children_heading_variant_with_markdown-link_row`, `.../leading_trailing-pipe_tolerance`. |
| AC-7 | Logic in a single shared `internal/spec` helper, unit-testable | DONE | `internal/spec/resolve_hint.go`; table-driven `resolve_hint_test.go` with constructed `[]*Spec` fixtures. |
| C-1 | Wire `ResolveOrHint` into `runVerify` not-found branch | DONE | `internal/cli/verify.go` (4 insertions, 7 deletions); fallback unchanged. |
| C-2 | Reuse `splitTableRow`, local Levenshtein (avoid `internal/triage`→`internal/spec` cycle) | DONE | `resolve_hint.go` imports nothing new beyond `fmt`/`sort`/`strings`; `splitTableRow` reused from `ledger.go`. |

### Exercise-the-feature check

- [x] Exercised end-to-end via `go run . spec verify <slug>` against this repo:
  - `configurable-reranking` (real unmaterialized child of `retrieval-quality`) → ``configurable-reranking` is listed as a child of initiative `retrieval-quality` but hasn't been designed into its own spec yet — run /design to materialize it before verifying.`` (exit 1)
  - `verify-slug-resolution-hint` (typo) → ``no spec `verify-slug-resolution-hint` found — did you mean `verify-slug-resolution-hints`?``
  - `zzz-nonexistent-slug-xyz` → `spec "zzz-nonexistent-slug-xyz" not found` (bare fallback, unchanged)
- Test evidence: `go build ./...` clean; `go test ./internal/spec/...` green; all 20 `TestVerify*` CLI tests green. Full-suite has one pre-existing, unrelated failure (`TestMarkdownInvocationsResolveAgainstRootCmd`) confirmed failing on a clean tree — not introduced here.

## Out of Scope
- **No change to the flat slug model.** Slugs stay flat; this spec does not add hierarchical or namespaced slug resolution.
- **No change to `spec.Discover`** — its walk and discovery semantics are correct and untouched.
- **No change to the status lifecycle** — planning/delivering/completed transitions, gates, and archival are unaffected.
- **Not a cross-repo resolution feature.** Initiative-child detection is local to the current workspace's discovered specs; no peer lookups.
- **`hero complete` is not modified** — it resolves by path and redirects work specs to verify (see Approach). No slug resolver is retrofitted onto it.
- **No auto-materialization.** The initiative-child message *directs* the user to `/design`; it does not create the child spec.
- **No new flags or output formats** on `hero spec verify`. The `--json` path is unaffected (the helper only changes the human error string on the not-found branch).

## Risks
- **Levenshtein false positives** — too loose a threshold suggests noise. Cap at distance 2 and at most 3 suggestions; very short slugs (≤3 chars) at distance 2 can over-match, so the engineer may additionally require the closest distance to be strictly less than the slug length. Validate against real slugs in tests.
- **Children-table heading drift** — initiatives in the wild use both `## Children` and `## Children — …`. The prefix-match on the section key covers the documented variants; an exotic heading (`## Child specs`) would be missed, which is acceptable (falls through to fuzzy or bare message — never worse than today).
- **Markdown-link first column** — failing to strip `[slug](path)` would make detection silently miss linked children. Covered by an explicit AC and a test fixture using the `get-back-on-track` linked-row format.
- **Performance** — fuzzy matching is O(specs × slug length); negligible at workspace scale (hundreds of specs) and only runs on the cold error path, never on a successful resolve.

## Validation
New table-driven tests in `internal/spec` (e.g. `resolve_test.go`) covering the
helper directly with constructed `[]*Spec` fixtures:

- **Exact match** → returns the spec, empty hint.
- **Case-only mismatch** (`Retrieval-Quality` vs `retrieval-quality`) → resolves to the spec, empty hint.
- **Unmaterialized initiative child** — fixture: one initiative spec with a `## Children` table containing `configurable-reranking`, no such child spec on disk; verifying `configurable-reranking` → nil spec, hint names the initiative and mentions `/design`.
- **Children heading variant** — same as above but heading is `## Children — six features` and the row is a markdown link `[project-charter](...)`; detection still fires.
- **Leading/trailing-pipe tolerance** — child table rows written with and without outer pipes both parse.
- **Fuzzy near-miss** — fixture with `r-01-foo`; verifying `R-01` → nil spec, hint "did you mean `r-01-foo`?".
- **Multiple suggestions** — two within distance 2 → both named, ≤3 total.
- **Initiative-child beats fuzzy** — slug that is both a child row and within edit distance of another spec → child message wins.
- **No signal** — unrelated slug with no near-miss and no child row → nil spec, empty hint (caller falls back to bare message).

CLI-level check (existing `internal/cli` test patterns): run `runVerify` against a
fixture workspace with an unmaterialized child slug and assert the returned error
string contains the initiative name and `/design`; assert the no-signal case
still returns exactly `spec "<slug>" not found`.

Run `go test ./internal/spec/... ./internal/cli/...` and confirm green. Manually
verify against this repo: `hero spec verify configurable-reranking` (a real
unmaterialized child of `retrieval-quality`) should now print the initiative-child
hint rather than the bare not-found message.
