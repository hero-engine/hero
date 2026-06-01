---
title: Size-Drift Actionable Output — Inline Next-Step and Dedupe Duplicate Error
slug: size-drift-actionable-output
type: feature
domain: engineering
status: ready
size: trivial
priority: P2
tags: [size, drift, cli, polish]
created: 2026-06-01
relations:
  - target: roadmap-shape
    kind: parent
---

## Goal

`hero size --check` is actionable on first read. Each drift row prints a
second indented line with the exact CLI/slash command the operator can
paste to resolve it (slug substituted in, no templates). The duplicate
`Error: size drift found in N spec(s)` / `size drift found in N spec(s)`
that prints back-to-back on non-zero exit collapses to a single line.
A closing pointer at `/roadmap-review` is added to the footer for
ambient triage. The `hero_warnings` MCP entries gain the same
alternative-action pointer (`/compose` or `/split`) so the model has
the same two paths the human sees.

## Context

Child of [`roadmap-shape`](../../initiatives/roadmap-shape/spec.md),
phase 2. Sibling of `multi-spec-design-routing`; independent of #1
(`roadmap-review`) but should land before #2
(`roadmap-review-ambient-surfacing`) so the row text is stable if #2
ever quotes an excerpt.

Slice 3 of the original `spec-size-and-promotion-nudge` work landed
the drift collector and the `hero_warnings` per-spec entries — those
entries already include `hero size <slug> <tier>` as the primary
action. What's missing is the *alternative* action (`/compose` or
`/split`) when the drift kind suggests promotion or splitting rather
than just bumping declared. This spec adds that.

The duplicate-error symptom is a `SilenceErrors: false` cobra command
returning an error AND `cmd/hero/main.go` printing the same error
again via `fmt.Fprintln(os.Stderr, err)` before `os.Exit(1)`. Two
paths, one error, double print. The user's stub suggested two options
(return nil + manual exit, or drop the explicit print). The cleanest
canonical-cobra fix is actually a **third option**: set
`SilenceErrors: true` on `sizeCmd` and keep everything else. main.go
already does the right thing for every command — sizeCmd just opted
back into cobra's print. See **Approach** for the rationale.

## Kickoff

Two surgical changes: inline a paste-ready next-step line under each
`hero size --check` drift row, and silence cobra's duplicate error
print on this command.

**Status:** ready — design done; no code yet.

**Pick up at:** add `sizing.SuggestedAction(declared, computed, kind)`
returning `(primary, alternative string)`, wire it into `runSizeCheck`
print loop (two lines per row), flip `sizeCmd.SilenceErrors = true`,
add the `/roadmap-review` footer, then extend the `hero_warnings`
entries in `mcp_tools.go` to include the alternative pointer.

→ `/deliver size-drift-actionable-output`

**Files:** `internal/cli/size.go:33,165-187`, `internal/sizing/sizing.go`, `internal/serve/mcp_tools.go:2588-2614`
**Skip:** new check categories, restructuring `--check` schema, JSON-output changes beyond adding the alternative-action field.

## Approach

**Shared suggestion helper.** Add a single function in
`internal/sizing/sizing.go` that maps `(declared, computed, kind)` to
`(primary, alternative string)`. Both surfaces (CLI print loop and
MCP warnings) consume it so the wording is identical and lives in
one place. Signature:

```go
// SuggestedAction returns the paste-ready primary action and an
// optional alternative for a drift row. Caller substitutes the slug
// into the returned strings (they include "%s" only where the slug
// goes — see DriftKind constants).
type DriftKind int
const (
    DriftLeafUp        DriftKind = iota // declared < computed
    DriftLeafDown                        // declared > computed
    DriftContainerUnset                  // initiative, declared empty
    DriftContainerLow                    // initiative, declared < rollup
)
func SuggestedAction(slug, declared, computed string, kind DriftKind) (primary, alternative string)
```

Returned strings substitute the slug directly (`hero size foo large`)
— no `%s` for the caller to fill. This is what "paste-ready" means in
the stub.

Mapping table:

| Kind                  | Primary                                | Alternative                                           |
|-----------------------|----------------------------------------|-------------------------------------------------------|
| `DriftLeafUp`         | `hero size <slug> <computed>` to bump declared | check whether the spec has grown beyond intent      |
| `DriftLeafDown`       | `hero size <slug> <computed>` to relax declared | `/split <slug>` if the spec is doing two things    |
| `DriftContainerUnset` | `hero size <slug> giant` to acknowledge | `/compose <slug>` to phase                            |
| `DriftContainerLow`   | `hero size <slug> <rollup>` to bump declared | `/compose <slug>` to phase children                  |

Both returns are formatted as one short clause each. The CLI joins
them with " or "; MCP emits them as separate fields (see below).

**CLI print loop change.** In `runSizeCheck` (size.go:165–181), after
each `fmt.Printf` row, classify the drift kind from the row's
declared/computed/rollup, call `SuggestedAction`, and print a second
indented line:

```
[leaf]      foo  declared: medium  computed: large
  → 'hero size foo large' to bump declared, or check whether the spec has grown beyond intent
[container] roadmap-shape  declared: (unset)  rollup: giant  (8 child(ren))
  → 'hero size roadmap-shape giant' to acknowledge, or '/compose roadmap-shape' to phase
```

Two-space indent + arrow + single quotes around CLI text. No ANSI
colors (out of scope per stub).

**Footer hint.** After the existing summary line, add one more line:

```
12 spec(s) with size drift  (2 leaf, 10 container).
Run '/roadmap-review' to triage interactively.
```

The footer hint is unconditional whenever drift was found. It's a
pointer to the surfacing layer the operator may not know exists.

**Cobra duplicate dedupe.** The duplicate prints because:
1. `sizeCmd.SilenceErrors = false` → cobra prints `Error: <err>` on
   non-zero RunE return.
2. `cmd/hero/main.go:23` prints `err` again via `fmt.Fprintln(os.Stderr, err)`.

**Pick option C, not A or B.** Flip `sizeCmd.SilenceErrors = true`.
Reasons:
- **Option A** (return nil + manual exit) requires `os.Exit(1)` inside
  `runSizeCheck`, which makes the function untestable end-to-end
  (the test harness can't catch `os.Exit`). Hard pass.
- **Option B** (drop the explicit print in main.go) is a global
  change — every other cobra command in the codebase relies on the
  current `SilenceErrors: false` + main.go's print as the canonical
  error path. Changing main.go silences error output for *every*
  command's non-cobra-printed errors. Wrong blast radius for a
  trivial spec.
- **Option C** (silence cobra on this one command) is local, leaves
  main.go's print as the single source, keeps `runSizeCheck`
  testable via `Execute`-style entry points, and is the standard
  cobra idiom when an outer wrapper handles error display.

Other cobra commands in the codebase that already follow the
`SilenceErrors: true` pattern serve as precedent — sizeCmd is the
outlier, not the rule. (Validation note: the design assumes other
commands silence; if grep shows they don't, the dedupe still works
because main.go's print is what users actually see — cobra's
`Error: ` prefix is the duplicate. Either way, flipping sizeCmd's
flag fixes the symptom locally.)

**MCP alternative-action field.** Today
`mcp_tools.go:2596-2613` emits a single sentence per drift:

```
**Size drift (leaf)** `foo`: declared `medium`, computed `large`. Run `hero size foo <tier>` to update or check whether scope grew.
```

Extend each entry to include both the primary and alternative
actions from `SuggestedAction`. Same flat-string shape — no new
keys, no JSON schema change. Format:

```
**Size drift (leaf)** `foo`: declared `medium`, computed `large`. Run `hero size foo large` to bump declared, or check whether the spec has grown beyond intent.
**Size drift (container)** `roadmap-shape`: declared `(unset)`, rollup `giant` (8 child(ren)). Run `hero size roadmap-shape giant` to acknowledge, or `/compose roadmap-shape` to phase.
```

The `<tier>` placeholder in the existing message becomes a concrete
tier (the computed/rollup value). This is the slug-substitution
requirement from the stub.

## Changes

1. **Add `SuggestedAction` helper in `internal/sizing/sizing.go`.**
   - New `DriftKind` int type + four constants
     (`DriftLeafUp`, `DriftLeafDown`, `DriftContainerUnset`,
     `DriftContainerLow`).
   - New `SuggestedAction(slug, declared, computed string, kind DriftKind) (primary, alternative string)`.
   - Slug substituted directly into returned strings; no `%s` left for
     callers. Strings match the mapping table in **Approach**.
   - Add a small helper `classifyLeafDrift(declared, computed string) DriftKind`
     that returns `DriftLeafUp` when computed > declared on the ladder,
     `DriftLeafDown` otherwise. (Container kinds are picked by the
     call site since they need to distinguish unset from low.)

2. **Wire helper into `runSizeCheck` print loop (`internal/cli/size.go`).**
   - After each `[leaf]` row print, classify and emit the indented
     `→ 'primary', or alternative` line.
   - After each `[container]` row print, same — distinguish
     `DriftContainerUnset` (declared empty) from `DriftContainerLow`
     (declared non-empty but below rollup) at the call site.
   - Skip the second-line print for indeterminate container rows
     (`d.Indeterminate == true`) — no actionable suggestion fits
     "we don't know the rollup."
   - After the existing total-summary line, print one more line:
     `Run '/roadmap-review' to triage interactively.`

3. **Silence cobra's duplicate error print (`internal/cli/size.go`).**
   - On `sizeCmd`, change `SilenceErrors: false` → `SilenceErrors: true`.
   - No other change. `main.go` already prints the error and exits
     non-zero — that single path becomes the only one users see.

4. **Extend MCP warnings entries (`internal/serve/mcp_tools.go`).**
   - In the `Size drift (leaf)` block (~line 2595–2599): replace the
     `<tier>` placeholder with the computed bucket, and append the
     alternative clause from `SuggestedAction`.
   - In the `Size drift (container)` non-indeterminate block
     (~line 2611–2613): append the alternative clause (`/compose <slug>`).
   - Indeterminate-container block stays as-is (no actionable
     alternative when rollup is unknown).
   - The format change is additive — no new fields, no schema break;
     `hero_warnings` consumers continue to see flat markdown strings.

## Acceptance Criteria

- WHEN `hero size --check` finds leaf drift THE SYSTEM SHALL print a second indented line under each `[leaf]` row containing the paste-ready `hero size <slug> <computed>` command with the actual slug substituted.
- WHEN `hero size --check` finds container drift with declared unset THE SYSTEM SHALL emit the inline hint `→ 'hero size <slug> <rollup>' to acknowledge, or '/compose <slug>' to phase`.
- WHEN `hero size --check` finds container drift with declared below rollup THE SYSTEM SHALL emit the inline hint `→ 'hero size <slug> <rollup>' to bump declared, or '/compose <slug>' to phase children`.
- WHEN leaf drift has declared > computed THE SYSTEM SHALL emit the alternative `'/split <slug>' if the spec is doing two things` rather than the "scope grew" alternative.
- THE SYSTEM SHALL substitute the actual slug into each inline hint — no `<slug>`, `%s`, or other template placeholders may appear in output.
- WHEN `hero size --check` exits non-zero due to drift THE SYSTEM SHALL print the failure error exactly once (not twice).
- THE SYSTEM SHALL keep `hero size --check` exit code non-zero whenever drift is found.
- WHEN drift is found THE SYSTEM SHALL print a closing line `Run '/roadmap-review' to triage interactively.` after the count summary.
- WHEN `hero_warnings` emits a size-drift entry whose drift kind has an alternative action (leaf-up, leaf-down, container-unset, container-low) THE SYSTEM SHALL include that alternative as a second clause in the entry.
- WHEN `hero_warnings` emits a size-drift entry THE SYSTEM SHALL substitute the computed/rollup tier into the primary action — no `<tier>` placeholder may remain.
- IF a container drift entry is indeterminate THEN THE SYSTEM SHALL keep the existing single-clause form (no alternative action) and not print a second line in CLI.
- THE SYSTEM SHALL preserve the existing tracker-capability header line at the top of `--check` output unchanged.
- THE SYSTEM SHALL preserve the `--check` JSON contract for any external consumer (no key renames or shape changes).

## Boundaries

Out of scope:

- New drift categories or row kinds beyond `[leaf]` / `[container]`.
- Restructuring `--check` output (column layout, table format,
  multi-column alignment).
- Per-row ANSI colorization.
- Changing the `hero_warnings` MCP shape from flat-string entries to
  structured objects with named action fields. Additive markdown only.
- JSON output mode for `hero size --check` (not a thing today; not
  added here).
- Refactoring `runSizeCheck` itself (extracting helpers, moving the
  print loop into a renderer struct, etc.).
- Touching `reportSizeDriftSummary` (consumed by `hero check`) —
  that path stays count-only.

## Risks

- **Slug edge cases in the paste-ready CLI text.** Slugs come from
  filenames and Hero validates them as kebab-case
  (`^[a-z0-9][a-z0-9-]*$`), so quoting in single quotes is safe.
  Confirm during delivery with a quick grep on `spec.Slug`
  validation. If special characters ever slip through (defensive
  posture), the worst case is the operator copy-pastes a slightly
  broken command — annoying but not corrupting.
- **Other cobra commands' `SilenceErrors` posture.** This spec flips
  `sizeCmd.SilenceErrors` to `true`, leaving other commands alone.
  If a future audit standardizes on `SilenceErrors: true` for all
  commands, sizeCmd is already there; if it goes the other way, this
  command becomes an outlier that someone will flag. Accept the
  asymmetry; the fix is local and reversible.
- **Footer hint timing.** `/roadmap-review` may not be implemented at
  delivery time. The slash command is the surfacing layer planned in
  the `roadmap-shape` initiative. Mention in delivery handoff if it's
  not yet shipped — the hint still reads correctly; it just points
  at a not-yet-built workflow. Acceptable.

## Validation

- `go test ./internal/sizing/... ./internal/cli/...` passes.
- `go test ./internal/serve/...` passes; `mcp_size_drift_test.go`
  updates assertions to expect the alternative-action clause.
- New unit tests for `SuggestedAction` covering all four
  `DriftKind` values with at least one slug each, asserting:
  - Slug substituted (no template chars in output).
  - Primary contains a `hero size` command for leaf-up / leaf-down
    / container-low; `hero size <slug> giant` for container-unset.
  - Alternative contains `/split` for leaf-down and `/compose` for
    both container kinds.
- Manual: create a leaf-drift spec, run `hero size --check`, confirm
  the second line renders correctly and the closing `Run '/roadmap-review'`
  line appears.
- Manual: confirm `echo $?` after `hero size --check` is still
  non-zero when drift is found, and that only one error line prints
  (no `Error:` prefixed duplicate).
- Manual: hit `hero_warnings` via MCP and confirm the leaf entry
  reads `Run 'hero size foo large' to bump declared, or check whether
  the spec has grown beyond intent` (and equivalent for the other
  three kinds).
