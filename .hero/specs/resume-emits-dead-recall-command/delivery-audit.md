# Delivery audit — resume-emits-dead-recall-command

**Audited:** `git diff HEAD~1 HEAD` (bdabdb97)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] Truncated sections emit `hero search <topic>`, not `hero recall` — `internal/digest/digest.go:930`; `TestMarkdown_RendersDigDeeperHintWhenTruncated` (`internal/digest/digest_test.go:190`). `hero search --help` shows `Usage: hero search <query>`; `hero search "feature-or-area"` from a fresh build runs and returns results (exit 0).
- [✓] No `hero recall` in emitted/generated output — `rg 'hero recall'` outside `.hero/` hits only the negative assertion in `digest_test.go:193-195`. `.hero/QUEUE.md` (generated) still contains the string, but only as quoted spec titles/kickoff text for this spec and `generated-command-refs-validated`, not as an emitted command hint.
- [✓] Test asserts the live name and fails if the dead name returns — `digest_test.go:190-196` asserts both. `go test -count=1 ./internal/digest/` passes.

## Changes
- [✓] 1. Emission site uses `hero search %s` — `internal/digest/digest.go:930`; invocation shape checked against the built binary's usage line.
- [✓] 2. Test corrected — `internal/digest/digest_test.go:190-196`.
- [~] 3. `sectionRecallTopic` rename — not renamed (`digest.go:929,973,975`). The spec allows leaving it but says to "note it"; the ledger has no row or note for it.
- [✓] 4. Sibling sweep — independently confirmed there are no other code or docs emission sites. The ledger has no row for this; covered by AC#2's note.

## Open items (if any)
- None marked PARTIAL/SKIPPED/BLOCKED. Changes 3 and 4 are missing from the ledger (see notes).

## Audit notes
- The ledger's Changes table has 2 rows and the spec lists 4. Change 3 (helper rename decision) went unrecorded. Keeping the name is within the spec's allowance, so this is a ledger-completeness issue, not a delivery gap.
- Boundaries respected: no general command-ref gate, no other surface audit, no enum changes. The diff touches only the digest file, its test, and the spec.
