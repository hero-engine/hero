# Delivery audit — hero-docs-check-engine-repo-misfire

**Audited:** `git show 2b35aca` (single delivery commit, branch `fix/agent-hero-version-schema-confusion`)
**Verdict:** SHIP
**Surface:** noteworthy

Cold audit — auditor did not observe the delivery. All findings from re-running the build/tests and reading the diff on disk.

## Re-run results
- `go build ./cmd/hero/` — OK
- `go test -count=1 ./internal/cli/... ./internal/install/...` — both green
- `go vet ./internal/cli/... ./internal/install/...` — clean
- `hero docs check` from repo root — exit 0, "No issues found.", counts 35 agents / 29 commands / 55 skills

## Acceptance criteria
- [✓] `hero docs check` in engine repo reports non-zero canonical counts (== install manifest), passes when docs accurate — re-ran live: engine-repo mode detected, 35/29/55, exit 0. Match-to-install proven by `TestEnumerateContent_MatchesInstalledFiles` (`internal/install/manifest_test.go:109`), which installs into temp dirs and asserts on-disk file counts == manifest — a real end-to-end assertion, not a shallow import.
- [✓] GETTING-STARTED.md + README.md counts match canonical — `GETTING-STARTED.md:74` (34/27/45→35/29/55), `README.md:127-129` table (34/28/45→35/29/55); enforced by `TestDocCountsMatchManifest` (`internal/cli/docs_check_test.go:50`). Numbers match the checker's live output exactly.
- [✓] Test covers engine-repo counting path (can't regress to actual:0) — `TestCanonicalCountsNonZero`, `TestIsEngineSourceRepo` (incl. negative case: installed-workspace layout not detected) in `internal/cli/docs_check_test.go`; all re-ran green.
- [✓] Release pre-flight `hero docs check` green in engine repo — re-confirmed exit 0.

## Changes
- [✓] New `internal/install/manifest.go` — `ContentManifest` + `EnumerateContent` + shared `selectFlatContent`/`selectSkillContent` selectors.
- [✓] `internal/install/content.go` refactored — `installFlat` (:42), `installSkillsNested` (:142), `installSkillsFlat` (:178) now call the SAME selectors `EnumerateContent` uses (manifest.go:41/45/49). Verified: no parallel walk. Full `internal/install` suite (incl. `TestHarnessNative_PerTargetFileSet`, `TestInstall_ExcludesContentReadmes`) passes, proving install still copies the same file sets — behavior-preserving.
- [✓] `internal/cli/docs_check.go` — engine-repo counting mode via `isEngineSourceRepo` (requires core/ + domains/ + .goreleaser.yaml all present). Installed-workspace branch (:68-73) is the original `countMDFiles`/`countSkillDirs` logic verbatim — untouched.
- [✓] New `internal/cli/docs_check_test.go`, `internal/install/manifest_test.go`.
- [✓] `GETTING-STARTED.md`, `README.md` counts refreshed.

## Probe findings
1. **Checker == install, no divergence:** CONFIRMED. Both route through `selectFlatContent`/`selectSkillContent`. `TestEnumerateContent_MatchesInstalledFiles` is real (installs to disk, compares counts).
2. **Counts derived, not fudged:** CONFIRMED. Independent cross-check of core+engineering deduped by basename yields 35/29/55 exactly. Dedup-by-name proven by `TestEnumerateContent_DedupesByName` (shared file counted once). READMEs excluded (`isContentReadme` + `TestEnumerateContent_SkipsReadmes`). Mirrors structurally excluded: the checker reads the embedded `hero.CoreFS()`/`hero.DomainFS()` (go:embed of `core/` and `domains/<domain>/` only — content.go:31/40), not a repo walk, so `.claude`/`.codex`/`web/docs/site` cannot be counted. Raw find (~89/83/213) is not reachable.
3. **Installed-workspace path unchanged:** CONFIRMED. else-branch is verbatim original. Detection requires all three markers; `TestIsEngineSourceRepo` asserts a `.claude/`-only workspace is NOT detected — no false-positive.
4. **content.go refactor behavior-preserving:** CONFIRMED. Full install suite green including native per-target file-set tests.
5. **Docs match:** CONFIRMED. 35/29/55 in both docs == checker output; Go test enforces.
6. **Ledger honesty:** Truthful (see Open items).

## Open items
- Stretch (generated counts) — SKIPPED — "spec marked it optional; `TestDocCountsMatchManifest` provides the anti-drift guard." Assessment: **concrete/legitimate.** Spec line 42 frames it as "Consider… Evaluate whether that's in scope or a stretch" — explicitly optional. The Go test closes the drift risk the codegen would have addressed. Not a dodge.
- MCP-tool counts left alone (GETTING-STARTED "41" vs README "42") — SKIPPED — "out of this spec's agents/commands/skills scope." Assessment: **concrete/truthful.** Genuinely outside the agents/commands/skills counting scope. Note the disclosed 41-vs-42 inconsistency is a real latent doc bug worth a follow-up.
- README table enforced by Go test, not runtime regex — disclosed. Assessment: **truthful, minor coverage nuance.** The runtime checker's regex (`(\d+)\s+commands` etc.) does not match the README table format ("Slash command definitions | 29") — confirmed: live run's README claim-validation section was empty. It also doesn't match GETTING-STARTED's "29 slash command definitions" prose (only "35 agents"/"55 skills" matched at runtime). So the commands count (29) in both docs is guarded only by `TestDocCountsMatchManifest`, not by `hero docs check` at runtime. Docs are accurate and the test enforces them, so the AC holds — but the runtime checker's coverage of doc phrasing is narrower than the ledger's framing implies (it's not just the README table).

## Audit notes
- No performative DONE rows. Every ledger claim maps to code + a real test that asserts on the new behavior.
- Diff is well-scoped to the spec's named files (single commit touches exactly manifest.go/content.go/docs_check.go + their tests + GETTING-STARTED/README + .hero projection files). No scope drift into product code.
- Verdict SHIP. Surfaced as noteworthy only for the disclosed scope calls (esp. the MCP 41-vs-42 doc inconsistency and the runtime-regex coverage nuance) — user may want a follow-up, none block this delivery.
