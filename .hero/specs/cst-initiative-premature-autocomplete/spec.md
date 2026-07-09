---
title: "Initiative auto-completes when its only materialized child is delivered, ignoring unbuilt declared children"
type: bug
status: completed
slug: cst-initiative-premature-autocomplete
domain: engineering
priority: high
severity: high
created: 2026-06-23
tags: [verify, auto-complete, initiative, lifecycle, data-integrity, cold-start]
parent: cold-start-trust-hardening
completed_at: 2026-06-23T22:29:43Z
---

# Initiative auto-completes when its only materialized child is delivered

## Problem

When `hero spec verify` completes a child spec, it tries to auto-complete the parent initiative if "all children are delivered." But it counts only **materialized** children — spec files that exist on disk and declare a `parent` relation back to the initiative. An initiative that declares many children (in its `child:` frontmatter list or `## Children` table) but has only *some* materialized as spec files will **auto-complete and archive the moment all of its materialized children are done**, even when most declared children were never built.

**Reproduced live (this session):** the `cold-start-trust-hardening` initiative declares 10 child stubs. Only one — `cst-verify-lifecycle-scoping` — had been materialized and delivered. Verifying that one bug printed:

```
Initiative "cold-start-trust-hardening" auto-completed — all children delivered
```

and moved the initiative from `.hero/planning/initiatives/` to `.hero/specs/`, flipping its status to `completed`. Nine declared children had not been written yet. The initiative had to be manually restored to `planning`.

**Impact:** silent, incorrect data mutation. An in-progress initiative is marked done and archived out of the planning surface (so it drops out of `hero queue`/active work) on the basis of a single delivered child. Recoverable by hand, but misleading and easy to miss — exactly the silent-wrong-behavior class this initiative targets.

## Root cause

**Classification:** logic error — incomplete invariant. Child enumeration is bottom-up only; the parent's declared child roster is never consulted.

`autoCompleteParentIfReady` (`internal/cli/verify.go:624-675`):

```go
allDone := true
childCount := 0
for _, s := range allSpecs {                 // every spec on disk
    for _, r := range s.Relations {
        if (r.Kind == "parent" || r.Kind == "child-of") &&
            normalizeVerifyParentTarget(r.Target) == parentSlug {
            childCount++
            if s.Status != spec.StatusCompleted {
                allDone = false
            }
            break
        }
    }
}
if childCount == 0 || !allDone {
    continue
}
// ... completeAndArchive(parent.Path, ...)
```

`childCount` is the number of **materialized** specs that declare `parent` back to the initiative. With one materialized child that is now `completed`, `childCount == 1` and `allDone == true` → the initiative is completed and archived. The parent's own declared roster (how many children it *intends* to have) is never read, so "all materialized children done" is mistaken for "initiative done."

### Compounding factor — fragmented, partly-unparsed child rosters

Even a fix that tries to read the parent's declared children must contend with **three different declaration formats**, one of which doesn't parse:

1. Frontmatter `child:` list — top-level shorthand parsed by `parseList(val)` on the **same-line** value only (`internal/spec/spec.go:507-510`). A block-style list (`child:` then newline `- item` lines) yields an **empty** value → **zero** child relations. (The `cold-start-trust-hardening` initiative uses exactly this block style, so its declared roster is invisible to the relation graph.)
2. Frontmatter `relations:` block with `kind: child`.
3. A markdown `## Children` table (the format the spec-not-found-hint feature reads — see `verifyInitiativeWithChild` fixture).

So the declared-children roster is split across formats and partly unparsed. A correct fix needs one reliable source of truth for "what children does this initiative declare."

### Test gap

`TestVerify_UnmaterializedInitiativeChild` (`internal/cli/verify_test.go:774`) sounds like it covers this, but it does **not** — it exercises the *spec-not-found hint* (verifying an unmaterialized child slug returns a helpful "owned by <initiative>, run /design" error). It never calls `autoCompleteParentIfReady`. There is **no** test for "an initiative with unbuilt declared children must not auto-complete when its materialized children finish."

## Suggested Fix Approach

In `autoCompleteParentIfReady`, before calling `completeAndArchive`, reconcile materialized-completed children against the parent's **declared** child roster:

- Resolve the parent's declared children from a single, reliable source. Prefer reusing whatever roster the spec-not-found-hint feature already parses (it clearly reads the initiative's children today). Unify with the frontmatter `child:` list — and fix the block-style `child:` parse gap (`spec.go:507`) so a newline list is recognized, or normalize all three formats through one helper.
- Auto-complete **only if** every declared child slug resolves to a materialized spec whose status is `completed`. If any declared child is unmaterialized or not completed, skip auto-complete.
- Keep the existing bottom-up `parent`-relation scan as a secondary signal, but the declared roster is authoritative for "is the initiative fully delivered."
- Safety fallback: if the parent declares **no** children in any format, do **not** auto-complete on the strength of incidental bottom-up children (current behavior auto-completes here — that is the risky path). Err toward leaving the initiative open; a human/`hero` can complete it explicitly.

This is surgical to `verify.go` plus a small declared-children resolver; it does not change the delivery gates themselves.

## Acceptance Criteria

- AC-1: THE SYSTEM SHALL NOT auto-complete an initiative when any child it declares (via `child:` list, `relations:` child, or `## Children` table) is unmaterialized or not `completed`.
- AC-2: THE SYSTEM SHALL still auto-complete an initiative when **every** declared child is materialized and `completed` (no regression to the happy path).
- AC-3: THE SYSTEM SHALL recognize a block-style frontmatter `child:` list (newline `- item`) as declared children, not silently drop it.
- AC-4: WHEN an initiative declares no children in any format, THE SYSTEM SHALL NOT auto-complete it from incidental bottom-up `parent` relations.
- AC-5: A regression test SHALL cover: an initiative declaring ≥2 children with only 1 materialized+completed — verifying that child MUST NOT auto-complete or archive the parent.

## Changes

- `internal/cli/verify.go` — reconcile `autoCompleteParentIfReady` against the parent's declared child roster; add a declared-children resolver (or reuse the existing one).
- `internal/spec/spec.go` — recognize block-style `child:` lists (if that path is chosen for the roster).
- `internal/cli/verify_test.go` — regression test for premature auto-complete with unbuilt declared children.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | No auto-complete while any declared child is unmaterialized/incomplete | DONE | declared-roster gate in `autoCompleteParentIfReady`; test `TestVerify_InitiativeNotCompletedWithUnbuiltChildren` |
| 2 | Still auto-completes when all declared children complete (no regression) | DONE | gate only triggers when `declaredCount > 0`; `TestVerify_InitiativeAutoComplete` stays green |
| 3 | Block-style `child:` list recognized as declared children | DONE | done in `cst-relation-frontmatter-fail-loud` (`parseScalarListBlock`); relied on here |
| 4 | No declared roster → never auto-complete from bottom-up | SKIPPED | deliberately narrowed: kept bottom-up for no-roster initiatives to avoid regressing existing flows; the stricter rule was not taken [signed-off] |
| 5 | Regression test covers premature auto-complete | DONE | `TestVerify_InitiativeNotCompletedWithUnbuiltChildren` |

### Changes

| # | Changes item | Status | Note |
|---|---|---|---|
| 1 | `internal/cli/verify.go` — declared-roster gate | DONE | reconciles materialized-completed children against the parent's declared roster |
| 2 | `internal/cli/verify_test.go` — regression test | DONE | `TestVerify_InitiativeNotCompletedWithUnbuiltChildren` |

### Exercise-the-feature check

- [x] Exercised: dogfooded against the real `cold-start-trust-hardening` initiative — completing this very spec leaves the initiative open (8 declared children unbuilt) instead of wrongly archiving it. Full suite green.

### Excellence Bar self-check

- [x] yes — surgical, composes with the relation-parsing fix, happy-path preserved. AC-4 narrowed with explicit sign-off to avoid regressing no-roster initiatives.

## Kickoff

**Pick up at:** `autoCompleteParentIfReady` in `internal/cli/verify.go:624`.

Cold-start prompt:
> Fix `hero spec verify` auto-completing an initiative when only some of its declared children are built. In `internal/cli/verify.go`, `autoCompleteParentIfReady` (line 624) counts only materialized specs that declare `parent` back to the initiative, so completing the single built child of a 10-child initiative archives the whole initiative. Before `completeAndArchive`, resolve the parent's *declared* child roster (the spec-not-found-hint feature already parses an initiative's children — reuse that source; also recognize the block-style frontmatter `child:` list, which `spec.go:507` currently drops because `parseList` only reads the same-line value). Auto-complete only if every declared child is a materialized `completed` spec; if the parent declares no children at all, do not auto-complete from incidental bottom-up relations. Add a regression test in `internal/cli/verify_test.go`: an initiative declaring 2 children with only 1 materialized+completed must not auto-complete. Verify: `go build ./... && go test ./internal/cli/ -run TestVerify`.

Reproduced live: completing `cst-verify-lifecycle-scoping` auto-completed and archived `cold-start-trust-hardening` (10 declared children, 1 built). Part of the `cold-start-trust-hardening` initiative.
