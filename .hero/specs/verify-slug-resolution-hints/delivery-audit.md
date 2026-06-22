# Delivery audit — verify-slug-resolution-hints

**Audited:** `git diff HEAD -- internal/cli/verify.go internal/cli/verify_test.go` + untracked `internal/spec/resolve_hint{,_test}.go`
**Verdict:** SHIP
**Surface:** noteworthy

Cold audit. Auditor did not observe the work; every claim below was reproduced from artifacts on disk (build, tests, live CLI exercises).

## Acceptance criteria
- [✓] AC-1 Case-only mismatch resolves & verifies, no prompt — `internal/spec/resolve_hint.go:33-37` (`strings.EqualFold`); test `TestResolveOrHint/case-only_mismatch_resolves_silently` passes (uncached).
- [✓] AC-2 Unmaterialized initiative-child reports owning initiative + `/design` — `resolve_hint.go:40-54`; tests `TestResolveOrHint/unmaterialized_initiative_child`, `TestVerify_UnmaterializedInitiativeChild` pass; **live**: `go run ./cmd/hero spec verify configurable-reranking` prints the initiative-child hint naming `retrieval-quality` and `/design`.
- [✓] AC-3 Near-miss (dist ≤2) prints "did you mean", ≤3 slugs — `resolve_hint.go:56-63` + `fuzzySuggestions`/`levenshtein`; test `.../fuzzy_near-miss_single_suggestion` asserts exact string. Cap-at-3 behavior is correct in code (verified by reproduction: 4 in-range candidates → 3 emitted, 4th dropped) — but see Audit notes: the `multiple suggestions capped at 3` test does not actually assert the cap.
- [✓] AC-4 Initiative-child preferred over near-miss on tie — step 3 precedes step 4 in `ResolveOrHint` (`resolve_hint.go:40` before `:56`); test `.../initiative-child_beats_fuzzy_on_tie` genuinely exercises the order: the competing fuzzy spec `configurable-rerankinX` is Levenshtein distance 1 from the child slug (confirmed), so it WOULD match — the child path winning proves precedence.
- [✓] AC-5 No-signal falls back to unchanged `spec %q not found` — `internal/cli/verify.go` fallback line is byte-for-byte unchanged; test `TestVerify_NoSignalBareMessage` asserts exact string; **live**: `verify zzz-nonexistent-slug-xyz` prints exactly `spec "zzz-nonexistent-slug-xyz" not found`.
- [✓] AC-6 Tolerate heading/pipe variants & bare-slug or `[slug](path)` cells — `childrenSectionBody` prefix-match (`:72-79`), `splitTableRow` reuse (`:92`), `normalizeCell` (`:114-122`); separator rows guarded (`:101`); tests `.../children_heading_variant_with_markdown-link_row`, `.../leading-trailing-pipe_tolerance` pass.
- [✓] AC-7 Logic in a single shared `internal/spec` helper, unit-testable — `internal/spec/resolve_hint.go`; table-driven `resolve_hint_test.go` with constructed `[]*Spec` fixtures, no CLI dependency.

## Changes
- [✓] `internal/spec/resolve_hint.go` (new) — `ResolveOrHint` + `childrenSectionBody`, `childTableHasSlug`, `normalizeCell`, `fuzzySuggestions`, `levenshtein`. Confirmed on disk; reuses `splitTableRow` (ledger.go:163), imports only `fmt`/`sort`/`strings` (no new deps, no import cycle).
- [✓] `internal/spec/resolve_hint_test.go` (new) — table-driven coverage of all branches; `TestResolveOrHint`, `TestFuzzySuggestions_ShortSlugGuard`, `TestLevenshtein` all pass uncached.
- [✓] `internal/cli/verify.go` — not-found branch now calls `spec.ResolveOrHint` (diff: 4 insertions / 7 deletions); bare fallback unchanged. Confirmed in `git diff HEAD`.
- [✓] `internal/cli/verify_test.go` — added `TestVerify_UnmaterializedInitiativeChild`, `TestVerify_NoSignalBareMessage`; both pass.

## Out of Scope — adherence
- [✓] `spec.Discover` untouched — not in diff.
- [✓] Flat slug model untouched — helper is read-only over discovered specs.
- [✓] Status lifecycle untouched — not in diff.
- [✓] `hero complete` not modified — `internal/cli/complete.go` not in diff.
- [✓] No new flags / output formats — `verify.go` change is confined to the error string on the not-found branch.

## Open items
None. Every ledger row (AC-1..AC-7, C-1, C-2) is `✓` with reproduced evidence. No PARTIAL / SKIPPED / BLOCKED rows.

## Audit notes
- **Test under-asserts AC-3 cap (minor, non-blocking).** `TestResolveOrHint/multiple_suggestions_capped_at_3` is named for the cap but only asserts `hintSubstr: ["did you mean", "\`reports\`"]`. With 4 candidates all within distance 2, it would still pass if the cap were broken (all 4 emitted). The production cap IS correct — reproduced independently: input `report` yields exactly `` `reports`, `resort`, `deport` `` (3, `export` dropped). The in-test comments labeling distances (`export` "dist 2", `resort` "dist 2", `deport` "dist 2 — would be 4th, dropped") are also inaccurate (actual: reports=1, export=2, resort=1, deport=1); the dropped one is `export`, not `deport`. Behavior ships correctly; the test+comments are just weaker/looser than their name implies. Recommend tightening the assertion in a follow-up, not a HOLD.
- **Levenshtein is correct.** Two-row DP, rune-based (unicode-safe: `levenshtein("café","cafe")==1`), verified against `kitten/sitting==3` and empty-string edges.
- **Short-slug over-match guard is sound** — `resolve_hint.go:153` suppresses suggestions when a ≤3-char slug's closest match is not strictly nearer than its length; `TestFuzzySuggestions_ShortSlugGuard` covers both suppress and allow directions.
- **Edge cases clean** — empty input slug does not panic; `fuzzySuggestions` skips empty candidate slugs; separator rows (`---`) are filtered before slug comparison.
- **No AI-slop, no dead code.** All comments are accurate and load-bearing; every helper is exercised.
- **Working-tree noise (not this delivery).** The tree also carries unrelated changes under `internal/herotest/*` and `internal/cli/test.go` from a separate workstream — out of scope for this audit, correctly excluded from the delivery's footprint.
- **Pre-existing unrelated failure confirmed.** `TestMarkdownInvocationsResolveAgainstRootCmd` fails on a clean tree (`GETTING-STARTED.md:406` references `hero name`, an unknown subcommand). Not introduced by this delivery; flagged here only for completeness.

## Reproduction log
- `go build ./...` — clean (exit 0).
- `go test ./internal/spec/ -run 'TestResolveOrHint|TestFuzzySuggestions|TestLevenshtein' -v -count=1` — all green.
- `go test ./internal/cli/ -run TestVerify -v` — all `TestVerify*` green incl. both new tests.
- `go run ./cmd/hero spec verify configurable-reranking` → initiative-child hint (`retrieval-quality`, `/design`), exit 1.
- `go run ./cmd/hero spec verify zzz-nonexistent-slug-xyz` → exactly `spec "zzz-nonexistent-slug-xyz" not found`.
