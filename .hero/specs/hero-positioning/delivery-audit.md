# Delivery audit — hero-positioning

**Audited:** `HEAD 53617502` plus the current positioning delivery diff
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] **AC-1 — memory first, verified delivery second.** The standalone position defines Hero as a project memory and delivery system, makes durable memory the headline, and makes verified delivery the connected execution layer (`.hero/marketing/positioning.md:9-26,58-88`).
- [✓] **AC-2 — every proof point links to a baseline claim, evidence class, and availability.** Exact-match validation found all 18 unique registry claim IDs referenced by the positioning source. The proof register and inline claims carry the recorded class and availability (`.hero/marketing/positioning.md:70-110,127,145-147,201`; `.hero/specs/hero-public-truth-baseline/public-claim-registry.yaml:14-485`). Sprout is now linked as `sprout-license-boundary`, Class A, shipped, with evidence in the archived licensing-boundary artifacts (`public-claim-registry.yaml:458-472`).
- [✓] **AC-3 — one lead audience.** AI-native engineers and hands-on technical leads in long-lived codebases are the sole lead audience; engineering leads, platform engineers, and multi-repository maintainers are secondary, while PM/QA/Sales are expansion (`.hero/marketing/positioning.md:28-44`).
- [✓] **AC-4 — “Correct your AI once” remains test-only.** The source prohibits publication and defines a revision-tied, cross-harness, cold-session threshold of ten successful runs across at least two harness pairings (`.hero/marketing/positioning.md:185-197`). The cohort test is explicitly a downstream publication experiment, not a gate for this authority (`.hero/planning/initiatives/hero-marketing/hero-positioning/spec.md:82-87`).
- [✓] **AC-5 — complete messaging toolkit and exact boilerplates.** Preferred vocabulary, prohibited claims, jobs, objections, fair comparisons, taglines, and all three boilerplates are present (`.hero/marketing/positioning.md:46-56,129-213`). Fresh whitespace counts are exactly 25, 50, and 150 words (`:203-213`).
- [✓] **AC-6 — PM and Sales remain expansion.** Engineering is the wedge; focused PM/QA/Sales setups are optional, maturity-bounded expansion paths (`.hero/marketing/positioning.md:42-44,211-213`).
- [✓] **AC-7 — two-system clarity.** The one-sentence position, numbered system model, reinforcing loop, and spec-kit objection make the source independently legible as two reinforcing systems rather than merely a spec kit (`.hero/marketing/positioning.md:9-26,159-161`).

## Changes

- [✓] **Rewrite the public messaging authority.** The authority contains the category, outcome, audience hierarchy, jobs, proof, boundaries, objections, boilerplate, and downstream surface contract (`.hero/marketing/positioning.md:1-233`). Its status is canonical but publication-gated, while the spec remains `delivering`; neither artifact prematurely claims approval (`.hero/marketing/positioning.md:3-7`; `.hero/planning/initiatives/hero-marketing/hero-positioning/spec.md:1-5,23-29`).
- [✓] **Define the messaging house and proof pillars.** The roof and three pillars preserve the memory-first hierarchy and map proof to the baseline (`.hero/marketing/positioning.md:58-110`). Supporting evidence uses the corrected paths `internal/cli/code_host_broker.go` and `internal/attention/mail/` (`.hero/specs/hero-public-truth-baseline/public-claim-registry.yaml:334-344,365-374`).
- [✓] **Define vocabulary and prohibited claims.** Required harness, peering, maturity, licensing, proprietary-product, readiness, output, and mutable-count boundaries are present (`.hero/marketing/positioning.md:120-150`). The truth audit and registry consistently record seven install targets: opencode, cursor, claude, copilot, codex, generic, and grok (`.hero/planning/initiatives/hero-marketing/content-truth-audit.md:59`; `public-claim-registry.yaml:242-270`).
- [✓] **Produce candidate taglines and exact boilerplate.** Candidate status, publication threshold, and exact 25/50/150-word blocks are present (`.hero/marketing/positioning.md:185-213`).
- [✓] **Write fair category comparison guidance.** The source covers coding harnesses, rule files, wikis, trackers, and spec frameworks, and defers named competitor claims pending current research (`.hero/marketing/positioning.md:175-183`).

## Open items

- None. No PARTIAL, SKIPPED, or BLOCKED rows appear in the Completion Ledger.

## Audit notes

- Existing supplied validation evidence remains green: full `go test ./...`, lint 7/7, score 95/A, index, registry-reference validation, and exact boilerplate word-count validation.
