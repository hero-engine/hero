# Delivery audit — attention-lifecycle-read-awareness

**Audited:** `git diff 9a3795d` plus lifecycle-specific untracked implementation and test files listed by `git status --short`; completed sibling `attention-mcp-action-tools` changes were excluded from lifecycle attribution
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: omitted limit returns at most 8 rows while preserving full counts and revision — defaulting and single-snapshot compaction are at `internal/serve/mcp_tools_attention.go:14-36`; the 22-row real-handler assertions are at `internal/serve/api_attention_test.go:187-234`.
- [✓] AC-2: limits 1–20 produce coherent window metadata and invalid values return field-specific validation — parsing/range enforcement is at `internal/serve/mcp_tools_attention.go:39-56`; omitted, 1, 20, zero, negative, fractional, and 21 are asserted at `internal/serve/api_attention_test.go:171-225`.
- [✓] AC-3: compact rows remove bodies and Mail summaries and UTF-8-safely cap other summaries at 240 bytes — `internal/attention/projection/compact.go:22-30,46-54`; redaction, byte bound, UTF-8 validity, and recognizable private Mail content are asserted by `TestCompactBoundsRowsAndRemovesBodiesWithoutMutatingSource` and `TestAttentionMCPLimitValidationAndUnavailableAreStructured`.
- [✓] AC-4: awareness uses one existing projection snapshot, preserves order/revision, does not mutate the source, and performs no action — the adapter invokes `Service.Snapshot` once at `internal/serve/mcp_tools_attention.go:22-36`; the pure copy is at `internal/attention/projection/compact.go:17-43`; immutability and zero Mail action calls are asserted at `internal/attention/projection/compact_test.go:11-52` and `internal/serve/api_attention_test.go:232-234`.
- [✓] AC-5: successful zero-total state is empty while unavailable authority remains a structured error — state derivation is at `internal/attention/projection/compact.go:33-42`; service failures are wrapped as `unavailable` at `internal/serve/mcp_tools_attention.go:22-34`; both outcomes are decoded separately at `internal/serve/api_attention_test.go:236-268`.
- [✓] AC-6: fresh/resumed sessions require exactly one advertised bounded read after normal context — canonical resume wording is at `core/commands/resume.md:22-30`; reusable guidance is at `domains/engineering/skills/attention-lifecycle-awareness/SKILL.md:33-38`; all six installed forms are asserted at `internal/install/attention_guidance_test.go:10-61`.
- [✓] AC-7: successful mutations retain their authoritative result, refresh at most once, and never replay a write — root guidance is at `internal/install/attention_guidance.go:7-11`; the detailed reusable rule is at `domains/engineering/skills/attention-lifecycle-awareness/SKILL.md:40-48`; six-target root and skill propagation is covered by `TestAttentionLifecycleGuidanceReachesAllHarnessNativeSurfaces`.
- [✓] AC-8: failed post-mutation refresh makes the prior timestamp/revision stale, never current or empty — `domains/engineering/skills/attention-lifecycle-awareness/SKILL.md:15-24,46-48`; the essential root rule is at `internal/install/attention_guidance.go:9`; installed state vocabulary is pinned in `internal/install/attention_guidance_test.go:30-42`.
- [✓] AC-9: recap contributes nothing for unchanged/irrelevant Attention and never polls solely for recap — `domains/engineering/skills/attention-lifecycle-awareness/SKILL.md:50-60`; self-contained root prohibition is at `internal/install/attention_guidance.go:11`; all six root and skill surfaces are asserted by the installer matrix.
- [✓] AC-10: OpenCode, Cursor, Claude, Copilot, Codex, and Generic receive native root, skill, and resume surfaces without hook dependence — the exact 18 target paths and required markers are exercised at `internal/install/attention_guidance_test.go:10-61`; engineering-only root selection is wired at `internal/install/agents_md.go:208-224` and excluded from the PM domain at `internal/install/attention_guidance_test.go:64-72`.

## Changes
- [✓] Add additive snapshot state/window contract and coherence validation — `contracts/attention/projection.go:32-54`; `contracts/attention/validate.go:405-427`; `contracts/attention/schema/v1/attention-snapshot.schema.json:10-12`.
- [✓] Add pure deterministic compact projection — `internal/attention/projection/compact.go`; focused behavior and immutability coverage is in `internal/attention/projection/compact_test.go`.
- [✓] Bound `hero_attention_snapshot` with structured validation — `internal/serve/mcp_tools_attention.go:14-56`; handler-level coverage is in `internal/serve/api_attention_test.go:171-268`.
- [✓] Publish discoverable MCP input and bounded behavior — the snapshot tool description and optional `limit` schema are at `internal/serve/mcp_tools_def.go:11-17`.
- [✓] Add canonical lifecycle-awareness skill — `domains/engineering/skills/attention-lifecycle-awareness/SKILL.md`.
- [✓] Integrate bounded awareness into resume — `core/commands/resume.md:20-30`.
- [✓] Add engineering-only managed root contributor — `internal/install/attention_guidance.go`; contributor ordering and domain selection are at `internal/install/agents_md.go:208-224`.
- [✓] Add runtime, contract, compatibility, documentation, and six-target validation — `contracts/attention/contract_test.go:407-438`; `internal/serve/api_attention_test.go:118-268`; `internal/install/attention_guidance_test.go`; `contracts/attention/testdata/v1/HERO-CODE-HANDOFF.md`; `README.md`; `GETTING-STARTED.md`. The focused four-package suite and `go test ./...` both pass.

## Audit notes
- None.
