# Delivery audit — doctor-routing-guidance-all-packs

**Audited:** `git diff main~1...HEAD` on branch `fix/agent-hero-version-schema-confusion` (HEAD `1cfcf41`)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Guidance authored in exactly one place — only non-test `.go` source is `internal/install/operational_guidance.go:28` (const `heroOperationalGuidance`). Repo-wide grep for the marker returns only that file (+ test files + the prior `agent-hero-version-schema-confusion` spec, not a render source). 0 occurrences in `agents_md.go`.
- [✓] Renders for all 4 domains × 6 targets — `TestHarnessNative_DoctorRoutingGuidanceAllTargets` runs 24 real subtests ({engineering,pm,sales,chat} × {claude,codex,opencode,cursor,copilot,generic}), all PASS; asserts both the guidance markers and the `## Hero Binary & MCP Surface` heading via `mustContain`, so dropping the section fails the exact domain/target. Wired at `agents_md.go:216` in `defaultSections`, shared by AGENTS.md + CLAUDE.md callers.
- [✓] Removed from engineering body + mirror — paragraph deleted from `generateEngineeringAgentsMdBody` (`agents_md.go`, former line 675) and from `domains/engineering/AGENTS.md` (diff shows the 2-line removal, nothing else). `TestEngineeringBodyOmitsOperationalGuidance` asserts absence in both the Go fallback body and the on-disk mirror — PASS.
- [✓] Future/unknown domain inherits automatically — `TestHarnessNative_OperationalGuidanceFallbackDomain` installs `--domain widgets` (no own AGENTS.md → engineering Go fallback body, which no longer carries the paragraph) and asserts the heading + guidance are present — PASS. Presence can only come from the shared section.
- [✓] Byte-stability tests green — `TestHarnessNative_SameManagedBody` and `TestEngineeringPackBodyMatchesGoFallback` both PASS (uncached, `-count=1`).

## Changes
- [✓] New `internal/install/operational_guidance.go` — `operationalGuidanceSection` implementing `managed.SectionContributor` (stable ID `install:hero-operational-guidance`, title `Hero Binary & MCP Surface`, Render returns the verbatim v0.25.0 paragraph). Confirmed in diff.
- [✓] New `internal/install/operational_guidance_test.go` — unit-tests SectionID, SectionTitle, Render content, and that Render embeds NO heading. Confirmed in diff; PASS.
- [✓] `internal/install/agents_md.go` — `defaultSections` inserts `newHeroOperationalGuidanceSection()` between the pack body and the snapshot pointer; engineering-body paragraph removed. Confirmed in diff.
- [✓] `internal/install/harness_native_test.go` — matrix test repointed to all-domain × all-target; 2 new tests (omit-body, fallback-domain). Confirmed in diff.
- [✓] `domains/engineering/AGENTS.md` — regenerated mirror, only the guidance paragraph removed. Confirmed in diff (net -2 lines).

## Open items
None. No PARTIAL / SKIPPED / BLOCKED rows in the ledger.

## Audit notes
- **Exactly-once / no double-render verified empirically.** Built the binary and ran real `hero install project <tmp> --target claude --domain <d>`: engineering → marker count 1, heading count 1; pm → 1 / 1 (previously 0 — pm body never carried the marker, confirmed by `grep domains/*/AGENTS.md`); sales → 1 / 1. No double-render for engineering.
- **Heading ownership is correct.** `internal/managed/region.go:154-158` emits a single `## <SectionTitle>` above each section body; the section body itself carries no heading (guarded by the render test). No double-emit.
- **Chat caveat disclosed and reasonable.** `chat` is non-installable (no `DomainFS` case); the matrix test exercises it through the real `AgentsMdBodyOverride` seam feeding the on-disk `domains/chat/AGENTS.md` into the same `defaultSections` pipeline. This proves the shared section renders on top of the chat body; the only unexercised path is the embed pipeline chat doesn't have. Honest and adequate.
- **Minor (not a defect):** the matrix test asserts presence (`mustContain`), not occurrence count, so it alone would not catch a hypothetical double-render. The exactly-once property is nonetheless verified — indirectly by `TestEngineeringBodyOmitsOperationalGuidance` (kills the duplicate-source path) and directly by the empirical install counts above.
- **Scope clean.** Code diff is confined to the spec's named files. The additional `.hero/{NEXT,QUEUE,SNAPSHOT}.md` + `next/chet-bellows.md` + `spec.md` changes are Hero projection/handoff files expected to travel with commits per project rules — not scope drift.

## Re-run results (uncached)
- `go build ./cmd/hero/` — OK (exit 0)
- `go test ./internal/install/... ./internal/snapshot/...` — ok, all pass
- `go vet ./internal/install/...` — clean (exit 0)
- Probe tests `-count=1 -v` — `DoctorRoutingGuidanceAllTargets` (24 subtests), `OperationalGuidanceSection_Render`, `SameManagedBody`, `EngineeringPackBodyMatchesGoFallback`, `EngineeringBodyOmitsOperationalGuidance`, `OperationalGuidanceFallbackDomain` — all PASS.
