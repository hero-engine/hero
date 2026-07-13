---
title: "hero docs check reports actual:0 in the engine repo, and GETTING-STARTED.md counts are stale"
slug: hero-docs-check-engine-repo-misfire
type: bug
status: delivering
priority: medium
severity: low
domain: engineering
created: 2026-07-13
origin: session
root_cause_class: design
tags: [docs, hero-docs-check, engine-repo, getting-started, counts]
relates-to:
  - agent-hero-version-schema-confusion
---

# hero docs check reports actual:0 in the engine repo, and GETTING-STARTED.md counts are stale

## Summary

`hero docs check` is designed to validate an **installed** workspace (where a harness has `.claude/agents`, `.claude/commands`, `.claude/skills`, etc. populated by `hero install`). Run inside the **hero engine repo itself**, it reports `actual: 0` agents / commands / skills, because in this repo those live as **source content** under `core/` + `domains/*/` (and mirrored into `.claude/`, `.codex/`, `web/docs/site/`), not as an installed harness tree the checker counts. It then flags GETTING-STARTED.md's counts as mismatched against a bogus `0`.

Two distinct problems:
1. **Checker misfire** (the bug): in the engine repo, `hero docs check` counts nothing and produces meaningless `claims N, actual 0 ← MISMATCH` output. It cannot validate the repo that authors the content.
2. **Genuinely stale doc counts** (surfaced by, but independent of, the misfire): GETTING-STARTED.md claims "27 slash command definitions, 34 agents, and 45 skills." Actual canonical `core/` + `engineering` domain counts are ≈ **29 commands / 35 agents / 55 skills**. The doc was not touched since v0.24.1 and drifted as content was added.

Discovered during the v0.25.0 release readiness pre-flight (the release skill runs `hero docs check` first). It did **not** block the release — `hero doctor` / MCP work changed no agents/commands/skills — but it's a real papercut for every future release and for anyone trusting the doc counts.

## Impact
- Every release pre-flight in the engine repo produces a false-positive docs failure that a human has to recognize and wave past — exactly the kind of "tool cries wolf" that erodes the gate.
- GETTING-STARTED.md undercounts what the installer ships, so a reader forms a smaller mental model of Hero's surface than reality.

## Root Cause
`hero docs check`'s counting assumes the installed-harness layout and has no mode for the source layout of the engine repo (`core/` base + `domains/<domain>/` overlays, deduped by name across the active install set). Evidence gathered this session:
- Sources: `core/agents` (4), `core/commands` (15), `core/skills` (16); `domains/engineering/agents` (31), `commands` (14), `skills` (39). Combined ≈ 35 / 29 / 55.
- Raw `find` across all locations (core + domains + `.claude` + `.codex` + `web/docs/site`) yields 89 / 83 / 213 — inflated by mirrors/symlinks, which is why a naive recursive count is also wrong.
- The canonical number is the **deduped install manifest** (what `hero install` actually copies for a given domain), not a raw file count.

## Suggested Fix Approach
1. **Teach `hero docs check` the engine-repo/source layout.** Detect when run in the hero source repo (e.g. presence of `core/` + `domains/` + `.goreleaser.yaml`, or a `hero.json` marker) and count the **canonical install set** — the same set `hero install` would copy for the default/active domain, deduped by name — instead of an installed harness tree. Reuse the install manifest logic so "what the checker counts" and "what install copies" can never diverge. Locate the checker in `internal/` (grep for the `docs check` command / "Documentation freshness check" string) and the install-manifest enumeration (`internal/install/`, `internal/cli/install.go`).
2. **Refresh GETTING-STARTED.md (and README.md) counts** from the corrected canonical numbers. Confirm the exact figures by running the fixed checker — do not hardcode the ≈ estimates above. Verify README.md's reference tables/sections match too.
3. **Consider having the counts generated, not hand-maintained** — if the doc counts can be emitted by the same manifest source, drift stops recurring. Evaluate whether that's in scope or a stretch.

## Acceptance Criteria
- [ ] `hero docs check` run in the engine repo reports non-zero, correct canonical counts (matching the install manifest), and passes when the docs are accurate.
- [ ] GETTING-STARTED.md and README.md counts match the canonical numbers (verified by the fixed checker, not by hand-estimate).
- [ ] A test covers the engine-repo counting path so the checker can't silently regress to `actual: 0`.
- [ ] Release pre-flight (`hero docs check`) is green in the engine repo with accurate docs.

## Test Plan
- Unit test the counting function against a fixture representing the `core/` + `domains/<d>/` source layout; assert deduped canonical counts, not raw file counts.
- Assert the checker's count equals the install manifest's count for the default domain (guards the "checker vs install diverge" failure mode).
- Doc-content assertion (or golden) that GETTING-STARTED/README counts equal the manifest count.

## Notes
- Keep this scoped to counting + doc refresh. Do NOT fold in unrelated docs rewrites.
- Cross-check with the release pre-flight in the `release` skill, which is the caller that surfaced this.

## Completion Ledger

Delivered on branch `fix/agent-hero-version-schema-confusion`. `go build ./cmd/hero/` OK; `go test ./internal/cli/... ./internal/install/...` green; `go vet` clean. Canonical counts **derived** (not hardcoded) from the fixed enumeration: **35 agents / 29 commands / 55 skills** (engineering domain).

| Acceptance criterion | Status | Evidence |
|---|---|---|
| `hero docs check` in engine repo reports non-zero canonical counts (== install manifest), passes when docs accurate | DONE | Live run: engine-repo mode detected, 35/29/55, exit 0, "No issues found." Match-to-install proven by `TestEnumerateContent_MatchesInstalledFiles`. |
| GETTING-STARTED.md + README.md counts match canonical (checker-derived, not estimate) | DONE | `GETTING-STARTED.md:74` (34/27/45→35/29/55); `README.md:127-129` table (34/45→35/55, 28→29 cmds); verified by `TestDocCountsMatchManifest`. |
| Test covers engine-repo counting path (can't regress to actual:0) | DONE | `TestCanonicalCountsNonZero`, `TestIsEngineSourceRepo` (`internal/cli/docs_check_test.go`); `TestEnumerateContent_MatchesInstalledFiles` (`internal/install`). |
| Release pre-flight `hero docs check` green in engine repo | DONE | Exits 0 (was exit 1). |

**Root-of-fix:** extracted install's content selection into shared `selectFlatContent`/`selectSkillContent` (`internal/install/manifest.go`) consumed by BOTH `installFlat`/`installSkills*` AND a new `EnumerateContent` — checker and install cannot diverge by construction. Engine-repo signal: `core/` + `domains/` + `.goreleaser.yaml` at root.

**Changes:** new `internal/install/manifest.go` (+ `_test.go`); `internal/install/content.go` (install fns consume shared selectors, net line reduction, no behavior change); `internal/cli/docs_check.go` (engine-repo counting mode; installed-workspace path unchanged); new `internal/cli/docs_check_test.go`; `GETTING-STARTED.md`, `README.md` counts refreshed.

**Deliberate scope decisions (disclosed):**
- *Stretch (generated counts):* evaluated, deferred — the spec marked it optional; `TestDocCountsMatchManifest` provides the anti-drift guard without a codegen step.
- *MCP-tool counts* (GETTING-STARTED "41", README "42") left alone — out of this spec's agents/commands/skills scope; a real but separate inconsistency worth a follow-up.
- *README table* enforced by the Go test rather than the runtime claim-regex (regex doesn't match the table format); regex extension is a reasonable follow-up.

## Kickoff

`hero docs check` reports `actual: 0` agents/skills in the engine repo (it assumes an installed harness layout; sources here live under `core/` + `domains/*`), and GETTING-STARTED.md counts drifted (claims 34 agents/27 commands/45 skills; actual ≈ 35/29/55).

**Status:** planning — diagnosed during v0.25.0 release pre-flight; did not block that release.

**Pick up at:** find the `hero docs check` implementation (grep "Documentation freshness check" in `internal/`) and the install-manifest enumeration (`internal/install/`, `internal/cli/install.go`). Add an engine-repo/source-layout counting mode that counts the canonical deduped install set (reuse the manifest logic so checker and install can't diverge), then refresh GETTING-STARTED.md + README.md counts from the fixed checker's output. Add a test for the engine-repo counting path.

**Files:** `internal/install/`, `internal/cli/install.go`, `GETTING-STARTED.md`, `README.md`, wherever `docs check` is registered.
**Skip:** raw recursive `find` counts (89/83/213 — inflated by `.claude`/`.codex`/`web/docs/site` mirrors); the canonical number is the deduped install manifest.
