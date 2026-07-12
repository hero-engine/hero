# Delivery audit — next-drift-gate-unwinnable

**Audited:** `git diff HEAD -- internal/ .github/` (uncommitted working tree)
**Verdict:** SHIP
**Surface:** noteworthy
**Confidence:** high

## Summary

The fix is delivered and correct. The churny `## Roadmap shape` size-drift
emission is removed from the NEXT.md projection, the `sizing` import is gone, the
build is clean, the dead `hero scan -q` flag is fixed, and the size-drift feature
still lives on its authoritative surfaces. `hero next checkpoint` is idempotent by
design (volatile-frontmatter-stripping writer). All six acceptance criteria are
satisfied.

Both HOLD blockers from the first audit pass were fixed and independently
re-verified:

1. **Vacuous regression guard — RESOLVED.** `TestNextMD_RoadmapShape_NeverEmitted`
   now sets `ActiveSpec: "drifted-leaf"`. Re-proved empirically (throwaway test,
   since removed): `sizing.AmbientDrift(heroDir, tmp, {ActiveSpec:"drifted-leaf"})`
   on the exact seed returns `Quiet=false, Count=1`, so a restored emission
   (`if !rep.Quiet && rep.Count > 0`) **WOULD** fire and the guard **WOULD** fail.
   The guard is now non-vacuous.

2. **Scope drift — RESOLVED.** The 5 scan-regenerated knowledge specs were
   reverted. `git status --porcelain` now shows only the 3 fix files
   (projection.go, projection_test.go, test.yml) + the projected
   NEXT.md/SNAPSHOT.md/next/chet-bellows.md + the new spec dir.

Surface remains **noteworthy** (not clean) only to carry the "reissued after
fixes" trail and the deferred-scrub note — no open blocker remains.

## Acceptance criteria

- [✓] **AC-1** gate green on correctly-committed repo — `writeProjectedFileIfSemanticChanged` (checkpoint.go:623) strips `updated:` via `normalizeUpdatedFrontmatter` before comparing; content-match → no write → committed timestamp preserved. Used by the checkpoint path (checkpoint.go:492).
- [✓] **AC-2** no corpus-derived count in NEXT.md — `## Roadmap shape` block removed (projection.go, now a NOTE comment); `grep` of `.hero/NEXT.md` for "size drift"/"Roadmap shape" → none.
- [✓] **AC-3** valid scan invocation — `test.yml:48` now `./hero scan >/dev/null 2>&1 || true`. Confirmed `hero scan` has only `--dry-run/--force/--code/--no-hooks` (scan.go:86-89); no `-q`/`--quiet` — the old flag was genuinely broken.
- [✓] **AC-4** still catches hook-less commits — preserved two ways: (a) architecturally — NEXT.md carries stable committed-spec-derived content (Next feature, Blocked-on dependency list, Context) reproducible on clean rebuild, so a stale/missing projection diverges; (b) the regression guard `TestNextMD_RoadmapShape_NeverEmitted` is now non-vacuous (`ActiveSpec` set → seed surfaces `Count=1` → a restored emission would fail the guard).
- [✓] **AC-5** timestamp-only run doesn't rewrite — same mechanism as AC-1; `normalizeUpdatedFrontmatter` placeholders the `updated:` line so a timestamp-only delta is a no-op write.
- [✓] **AC-6** build + tests — `go build ./...` clean; `go test -count=1 ./internal/projection/...` PASS; coordinator confirms `go test ./...` green (86 pkgs).

## Changes

- [✓] **internal/projection/projection.go** — `## Roadmap shape` emission removed, `sizing` import removed, replaced with an explanatory NOTE. No other projection section touched (`## Blocked on` header immediately follows). Build clean → no unused import.
- [✓] **internal/projection/projection_test.go** — 3 old roadmap tests (`_Emits`, `_OmittedWhenQuiet`, `_NoHeroDir`) collapsed into `TestNextMD_RoadmapShape_NeverEmitted`, which now seeds real drift AND sets `ActiveSpec: "drifted-leaf"` so the seed surfaces through `AmbientDrift` (non-vacuous — would fail if the emission were restored).
- [✓] **.github/workflows/test.yml** — dead `./hero scan -q` → `./hero scan >/dev/null 2>&1 || true`.

## Audit notes

### Non-vacuity re-verified (was the blocker)

`TestNextMD_RoadmapShape_NeverEmitted` (projection_test.go:206-243) seeds a
`drifted-leaf` feature (declared `size: trivial`, 10 files → real drift) and calls
`NextMD` with `ActiveSpec: "drifted-leaf"`. That fires Rule 1 of
`filterAmbientDrift` (ambient.go:202-206), so `AmbientDrift` returns
`Quiet=false, Count=1` — the exact condition the removed emission gated on. The
guard asserts `## Roadmap shape` and `size drift` are both absent; it passes only
because the emission is gone, and would fail if it were restored. Empirically
confirmed (`Quiet=false Count=1`).

### Feature still lives outside NEXT.md (fix correctly scoped)

`sizing.AmbientDrift` remains called by `hero size --check` (size.go:213), the
size-drift MCP tools (mcp_tools.go:432, mcp_tools.go:1008 — the latter populates
`pulse.AmbientSizeDrift` for the pulse MCP surface), and `hero check` via
`reportSizeDriftSummary` (check.go:475 → size.go). Removal was NEXT.md-only.

### Dead `NextMDOptions` fields — acceptable deferred scrub

`RoadmapRecencyDays`/`RoadmapStopNaggingHours` remain in the struct
(projection.go:57-63) and `HeroDir`/`ProjectRoot`/`ActiveSpec` are still assigned
by callers (checkpoint.go:486-487, next_project.go:65-66) — now inert but harmless
(valid Go, compiles clean, no behavior). The ledger defers their removal as a
scrub follow-on. Not a defect.

### Related follow-up

The coordinator spawned a separate task for the underlying `hero scan`
clobbers-`created:` bug that produced the earlier knowledge-spec churn. Out of
scope for this delivery.

## Open items

None. Both prior blockers resolved and re-verified. The only remaining acceptance
is the real-world one the ledger already flags: the `Test` GitHub Actions job
turning green on the next push to main (`gh run list --workflow=Test --branch main`).
