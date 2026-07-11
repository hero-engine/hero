---
title: "Satellite Walkthrough UX — Vendor-Pattern Detection and Exclude-Parent Shortcut"
slug: satellite-walkthrough-ux
type: feature
status: planning
priority: medium
horizon: now
tags: [monorepo, satellites, ux, install]
relations:
  - target: monorepo-satellite-installs
    kind: parent
created: 2026-05-12
---

# Satellite Walkthrough UX — Vendor-Pattern Detection and Exclude-Parent Shortcut

## Problem

The first example codebase migration surfaced a real UX gap. Running `hero install satellites` proposed **42 candidate subprojects**, of which **40 were vendored crates** under `example codebase-vendor/`:

```
[3/42] example codebase-vendor/example codebase-candle           propose? n
[4/42] example codebase-vendor/example codebase-candle/candle-book   propose? n
[5/42] example codebase-vendor/example codebase-candle/candle-core   propose? n
…  (35 more siblings, all `n`)
```

Two failure modes:

1. **`example codebase-vendor` should have been auto-noise-listed.** The existing detector hard-codes `vendor` as a noise directory — but a folder named `example codebase-vendor` (or anything else with a `-vendor` suffix or `vendor-` prefix) doesn't match the exact-string check. So the walker descends into all 40 child crates, each with its own `Cargo.toml`, and every one gets surfaced as a candidate.

2. **There's no "exclude this whole subtree" shortcut mid-walk.** Once the user realizes "everything under `example codebase-vendor/` is junk," they have to either type `n` 40 times (will be re-prompted next install — annoying), or `x` 40 times (permanently excluded but still individually entered). The actual right answer — "exclude `example codebase-vendor/` as a parent and stop asking about anything under it" — requires quitting the walkthrough and hand-editing `subprojects.json`.

This is fixable. The walkthrough is the on-ramp; if it's painful in monorepos with vendor trees (which is most of them), satellite adoption stalls before it starts.

## Goal

Two small additions, both surface-level:

- **Auto-noise vendor-shaped names.** Folders matching `*-vendor` or `vendor-*` get treated like `vendor` already does — the walker skips them and their entire subtree. Conservative pattern, just covers the actual common case.
- **`X` (capital) option in the walkthrough = exclude parent.** When the user encounters a candidate they want to bulk-exclude *along with all its siblings*, hitting `X` adds the parent folder to `excluded[]` in `subprojects.json` (instead of the leaf), and the walker skips the rest of that subtree on the spot.

**Mission-fit.** Satellite installs are about reducing the friction of running Hero in real monorepos. Forcing a user through 40+ prompts on first install is exactly the kind of papercut that makes "raise the floor" feel like "punish the floor."

The non-goal is a full glob-pattern exclusion language. We're adding two narrow, predictable shortcuts; if `*-vendor` doesn't catch everything someone has, `X` covers it manually with one keystroke.

## Design

### 1. Auto-noise vendor-shaped names

Current detector (`internal/install/satellite_detect.go`):

```go
noise := map[string]bool{
    "node_modules": true,
    "vendor":       true,
    ".git":         true,
    // …
}
```

This is exact-name matching. Add a small prefix/suffix check alongside it:

```go
isVendorShaped := func(base string) bool {
    return strings.HasSuffix(base, "-vendor") ||
           strings.HasPrefix(base, "vendor-") ||
           base == "vendored"
}
```

Apply the check at the same point the noise-map check runs. Folders matching `*-vendor` / `vendor-*` / `vendored` → `filepath.SkipDir`, just like `vendor`. The walker doesn't descend; no children get surfaced.

Conservative on purpose. Not adding `external`, `third-party-*`, `deps`, etc. — those are real folder names some projects use for *first-party* code. The vendor patterns are the empirically-observed common case from example codebase and similar repos. Anything else can be handled by the manual `X` shortcut below.

**Why not also auto-noise based on a `.gitignore`-flagged directory?** Tempting, but `.gitignore` patterns are arbitrarily complex (negations, globs, comments) and parsing them correctly is its own dependency. Punt for now; if it ever matters, that's a follow-up.

### 2. `X` (exclude parent) walkthrough option

The walkthrough today has these options on each prompt:

```
y/yes        materialize satellite
n/no         skip — ask again next time (default)
a/all        yes to all remaining
s/skip-all   skip all remaining for this run
x/exclude    permanently exclude (writes excluded[] entry)
q/quit       stop prompting
?            help
```

Add capital `X` (parent-exclude):

```
X            permanently exclude this folder's parent — also skips
             all remaining candidates under that parent
```

Behavior on `X`:

1. Compute `parent = filepath.Dir(candidate.Path)`. If `parent == "."` (i.e. the candidate is at the top level), fall back to behaving like lowercase `x` (exclude the leaf only) and print a note explaining why.
2. Add the parent to `subprojects.json` `excluded[]` (idempotent — `AddExcluded` already de-dupes).
3. Filter the remaining candidate list to drop everything under that parent (or under any other path already excluded). Continue prompting.
4. Print the immediate consequence: `excluded example codebase-vendor (skipped 39 remaining candidates under it)`.

The remaining-list filter has to be done *in the walker loop*, not just at the next install — otherwise the user excludes `example codebase-vendor` on candidate 3/42 and still has to walk through 4/42 through 42/42 even though they're all under it.

Update the help text (`?` option) to include `X` with a clear "shotgun" flavor:

```
X / EXCLUDE   permanently exclude this folder's parent (and skip all
              remaining candidates under it). Use this when you hit
              a vendor or third-party tree.
```

### Design decisions

**Why prefix/suffix matching for vendor patterns instead of full glob?** Because the actual problem is "a folder named like a vendor tree wasn't recognized." Two patterns cover example codebase-vendor and reverse-cased equivalents (vendor-libs, vendor-third-party). A glob engine adds complexity that doesn't pay off until someone hits a case the prefix/suffix doesn't cover — at which point they should add a noise entry, not configure a glob.

**Why is `X` capital and `x` lowercase, not just renaming `x` to `Xself` and `X` to `Xparent`?** Because the existing `x` shortcut is shipped and people will have muscle memory. The capital version reads as "more aggressive" and pairs naturally — same family, bigger effect. Two-letter mnemonics like `Xs` and `Xp` are slower to type and less obvious at a glance.

**Why does `X` fall back to leaf-exclude behavior when the candidate is top-level (parent = `.`)?** Because excluding `.` would mean "exclude everything in the workspace from being a subproject," which is never the intent. The fallback turns `X` into `x` for that one case and tells the user why, so they don't think the keystroke silently failed.

**Why filter the remaining candidates immediately instead of letting the walker hit them and skip via `IsExcluded`?** Two reasons. First, walking 40 entries that all just print "(excluded)" wastes the user's attention. Second, the candidate list is built *before* the walk starts (via `DetectCandidates`), so excluded entries that were added mid-walk don't get re-filtered automatically — we have to do it inline.

**Why not warn when many candidates share a parent and offer "do you want to exclude the parent now?" up front?** Tempting but adds branching to the entry path. The `X` keystroke covers the same case reactively without adding pre-walk logic — and reactively is honestly better, because the user has already seen the candidates and knows which subtree to nuke. Up-front prompts for "looks like a vendor tree?" risks false positives on legitimate sub-project clusters.

**Why update `subprojects.json` immediately on `X` instead of batching writes at the end of the walk?** Because the existing walkthrough already saves on each `y` and `x` (via the `dirty` flag deferred at function exit). Keeping the same save model means a Ctrl-C mid-walk preserves all the decisions the user has actually made — which matches what people expect from an interactive flow.

## Acceptance Criteria

- WHEN the candidate detector encounters a directory whose base name matches `*-vendor`, `vendor-*`, or equals `vendored` THE SYSTEM SHALL skip the directory and its descendants from candidate proposal, the same way it currently skips `vendor`, `node_modules`, etc.
- THE SYSTEM SHALL accept `X` (capital) as a walkthrough option that permanently excludes the parent of the current candidate and skips all remaining candidates under that parent.
- WHEN the user selects `X` on a top-level candidate (parent equals `.`) THE SYSTEM SHALL behave like lowercase `x` (exclude the leaf only) and print a note explaining the fallback.
- WHEN the user selects `X` on a non-top-level candidate THE SYSTEM SHALL append the parent path to `excluded[]` in `.hero/subprojects.json`, drop remaining queued candidates that fall under the new exclusion, and report how many were dropped.
- THE SYSTEM SHALL update the walkthrough help text (the `?` option) to document the `X` option with a one-line description of its scope.

## Changes

### Modified files
- `internal/install/satellite_detect.go` — add the vendor-shaped check alongside the noise-map check.
- `internal/install/satellite_detect_test.go` — add tests covering `*-vendor`, `vendor-*`, `vendored`, and a non-matching control case.
- `internal/cli/install_satellites.go` — add `X` option in `walkCandidates`, plus help text update.

## Phasing

Single phase. Two surface tweaks, atomic.

## Kickoff

Resume by reading `.hero/planning/features/satellite-walkthrough-ux/spec.md`. Two small UX fixes for the satellite install walkthrough surfaced during the example codebase migration: auto-noise `*-vendor` / `vendor-*` / `vendored` directory names in the candidate detector, and add capital `X` as an "exclude parent" walkthrough option that drops all remaining candidates under the parent. Parent spec: monorepo-satellite-installs.
