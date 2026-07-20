# Delivery audit — progressive-design-ac-recap

**Audited:** `git diff -- domains/engineering/commands/design.md internal/install/harness_smoke_test.go`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Large contracts use a total count and 2–5 concrete coverage themes instead of one row per criterion — `domains/engineering/commands/design.md:38`
- [✓] Current-loop changes name up to five ACs or summarize larger deltas by count and theme — `domains/engineering/commands/design.md:38`
- [✓] Initial contracts are labeled without enumerating every new criterion as a delta — `domains/engineering/commands/design.md:38`
- [✓] Attention-worthy exceptions name the affected AC and issue — `domains/engineering/commands/design.md:40`
- [✓] Short or explicitly requested contracts retain the compact AC table — `domains/engineering/commands/design.md:42`
- [✓] Every closing links the full contract and preserves score and next-step guidance — `domains/engineering/commands/design.md:40`
- [✓] The progressive contract propagates to all six native command surfaces — `internal/install/harness_smoke_test.go:182`; `TestHarness_DesignClosingUsesProgressiveACDisclosureForAllTargets`
- [✓] No client foldout, configuration, persistence, or verification changes are required — the scoped diff changes only canonical workflow prose and install propagation coverage.

## Changes
- [✓] Update the canonical design closing contract — `domains/engineering/commands/design.md:38` adds bounded coverage and delta summaries, exception visibility, selective expansion, and the full-contract link requirement.
- [✓] Add all-six-target propagation coverage — `internal/install/harness_smoke_test.go:182` installs the real engineering overlay for each supported target and asserts the new contract is present and the obsolete unconditional-table mandate is absent.

## Open items (if any)
- None.

## Audit notes
- Focused test evidence reports `TestHarness_DesignClosingUsesProgressiveACDisclosureForAllTargets` passed for all six targets; the full `internal/install` package and `git diff --check` also passed.
- The exercise record reports practical generic and Codex installs completed and their installed command surfaces were inspected for both the progressive contract and obsolete-rule absence.
