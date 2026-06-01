# Delivery audit — roadmap-review-ambient-surfacing

**Audited:** `git diff HEAD` (uncommitted working tree, 17 modified + 2 new)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Helper `sizing.AmbientDrift(heroDir, projectRoot, opts) AmbientDriftReport` — `internal/sizing/ambient.go:74`. Report carries `Count / Hint / Quiet / Reason / LegacyExcluded`. Opts carry `ActiveSpec / RecencyDays / StopNaggingHours / Now`.
- [✓] OR-joined noise rules (active / recency / horizon:now unsized initiative) — `internal/sizing/ambient.go:202-220` (`matchesAmbientRules`). Tested by `TestAmbientDrift_ActiveSpecRuleFires`, `TestAmbientDrift_HighImpactInitiativeRuleFires`, `TestAmbientDrift_DriftFilteredOut_NoActiveNoRecent`.
- [✓] No-drift quiet return with `Reason: "no drift"` — `ambient.go:90, 110`. Tested `TestAmbientDrift_QuietWhenNoSpecs`.
- [✓] Stop-nagging suppression when newest session record within window AND count not greater than recorded — `ambient.go:115-125`. Tested `TestAmbientDrift_StopNagging_SuppressesWithinWindow`.
- [✓] Lift suppression when count exceeds `drift_count_at_exit` — `ambient.go:120-122`. Tested `TestAmbientDrift_StopNagging_LiftsWhenCountGrows`.
- [✓] Missing `drift_count_at_exit` is fully suppressive — `ambient.go:120` (the `ok && ...` guard means missing field → fall to the else branch → `Quiet: true`). Tested `TestAmbientDrift_StopNagging_MissingFieldIsSuppressive`.
- [~] Hint phrasing — engineer SHIPPED `"N specs have size drift — run /roadmap-review to triage"` (`ambient.go:138-143`). **Spec AC mandates `"⚠ N specs have size drift — …"` verbatim.** Engineer dropped the `⚠` prefix per project CLAUDE.md no-emoji-without-request rule. This is a documented judgment call — flagged for user to confirm.
- [✓] NEXT.md `## Roadmap shape` section between `## Next` and `## Blocked on` — `internal/projection/projection.go:139-152`. Tested `TestNextMD_RoadmapShape_Emits` (explicit placement assertion). Live NEXT.md confirms: `## Next` → `## Roadmap shape` → `## Blocked on` order.
- [✓] NEXT.md omits section entirely when quiet/zero — `projection.go:148` (guarded `if !rep.Quiet && rep.Count > 0`). Tested `TestNextMD_RoadmapShape_OmittedWhenQuiet`, `TestNextMD_RoadmapShape_NoHeroDir`.
- [✓] `hero_pulse` populates `PulseData.SizeDrift` and renders in text/markdown/JSON — `internal/serve/mcp_tools.go:980-993`; renderers at `internal/pulse/render.go:54-58` (text), `:152-155` (markdown), `:283-288` (JSON). Tested `TestMCP_ToolCall_Pulse_SizeDrift_PresentWhenInitiativeUnsized`.
- [✓] `hero_pulse` nil SizeDrift & no line when quiet/zero — `mcp_tools.go:988` (guarded). Renderers nil-check. Tested `TestMCP_ToolCall_Pulse_SizeDrift_AbsentWhenQuiet`.
- [✓] `hero_kickoff` passes slug as `ActiveSpec` and prepends hint — `mcp_tools.go:418-432`. Tested `TestMCP_ToolCall_Kickoff_SizeDriftPrefix` (asserts hint appears before `## drifted-feature` header).
- [✓] `hero_kickoff` body unchanged when quiet/zero — `mcp_tools.go:431` (driftPrefix only set when non-quiet). Tested `TestMCP_ToolCall_Kickoff_NoPrefixWhenQuiet`.
- [✓] Both delivery-lead agent docs extended with one sentence about `size_drift` field / workspace-wide hint — `domains/engineering/agents/feature-delivery-lead.md:106` (step 4d), `domains/engineering/agents/platform-delivery-lead.md:69` (step 3b). One-sentence inserts, not rewrites.
- [✓] No per-spec rows in any of the three ambient surfaces — confirmed by reading NEXT.md projection (just hint), pulse render (just hint), kickoff prefix (just hint). Renderers do not iterate drifted specs.
- [✓] Lens-agnostic "size drift" phrasing — `ambient.go:138-143` uses "size drift" only. No "sizing-lens" or "roadmap-shape concern" anywhere.
- [✓] Config defaults — `internal/config/config.go:118-148` defines `RoadmapConfig`. `AmbientRecencyDaysOrDefault()` returns 7; `StopNaggingHoursOrDefault()` returns 24; zero/nil falls through cleanly. Negative values rejected at load (`:1310-1320`).
- [✓] Missing session-records directory → no suppression — `ambient.go:251-256` (`os.ReadDir` error → `return false, 0, false` → no suppress). Implicitly exercised by `TestAmbientDrift_ActiveSpecRuleFires` (no session dir; fires normally).

## Changes (Files to Touch)

- [✓] `internal/sizing/ambient.go` (new, ~370 lines actual vs ~120 estimated — over but justified by adapter/mtime helpers).
- [✓] `internal/sizing/ambient_test.go` (new, ~446 lines, 9 test functions covering all branches).
- [✓] `internal/config/config.go` — `Roadmap *RoadmapConfig` field + OrDefault helpers + validation.
- [✓] `internal/projection/projection.go` — `NextMDOptions` gains `HeroDir / ProjectRoot / ActiveSpec / RoadmapRecencyDays / RoadmapStopNaggingHours`. Section emission between `## Next` and `## Blocked on`.
- [✓] `internal/projection/projection_test.go` — three new golden tests (emits / quiet / no-hero-dir).
- [✓] `internal/pulse/pulse.go` — `AmbientSizeDrift` type + `SizeDrift *AmbientSizeDrift` field on `PulseData`.
- [✓] `internal/pulse/render.go` — text/markdown/JSON renderers updated; text/markdown emit before the existing `Drift detected` block; JSON uses `size_drift,omitempty`.
- [✓] `internal/serve/mcp_tools.go` — `toolPulse` and `toolKickoff` both build `AmbientDriftOpts` from `cfg.Roadmap` and call `sizing.AmbientDrift`.
- [✓] `internal/serve/mcp_test.go` — four new test functions (pulse present/absent, kickoff prefix/no-prefix).
- [✓] `domains/engineering/agents/feature-delivery-lead.md` — one-sentence inline insert on step 4d.
- [✓] `domains/engineering/agents/platform-delivery-lead.md` — one-sentence inline insert on step 3b.
- [✓] `domains/engineering/skills/roadmap-review/SKILL.md` — new `## Ambient surfaces` subsection (~23 lines) documenting the three surfaces and the `drift_count_at_exit` convention.
- [✓] Canonical paths only — all `.md` edits under `domains/engineering/`. No `.claude/` writes in the diff.
- [✓] No new MCP tools.

## Open items

- **Hint phrasing — `⚠` prefix dropped (engineer judgment call).** Spec AC mandates verbatim `"⚠ N specs have size drift — …"`. Engineer dropped the warning-sign per CLAUDE.md no-emojis-without-request. Reason is concrete and consistent with workspace policy. User decision: keep the project convention or honor the spec literally and update the spec to remove `⚠`.
- **Scope creep — `hero size --check --summary` flag.** Not in the spec's `Files to Touch` list; not required by any AC. Adds `--summary` flag + mutual-exclusion validation on `internal/cli/size.go` (~24 added lines). The engineer then wired this new flag into BOTH delivery-lead docs as a co-equal path alongside the spec-mandated `hero_pulse` `size_drift` read. Justification (per ledger): a one-line CLI surface that mirrors the ambient hint, useful for delivery-lead pre-flight. **No test coverage** for the new flag — only live-exercise evidence in the ledger. Small, isolated, low-risk; flagging because it expands user-visible CLI surface area without spec authorization.

## Audit notes

- Build clean (`go build ./...` zero output).
- All touched packages green (`go test ./internal/sizing/ ./internal/projection/ ./internal/serve/ ./internal/config/`).
- Live NEXT.md (`.hero/NEXT.md`) carries the new section in the correct position with the engineer's chosen phrasing: `12 specs have size drift — run /roadmap-review to triage`. The 12-count is real workspace state.
- Spec status flip `ready → delivering` is mechanical (validator parity); not a delivery concern.
- Helper file size: spec estimated ~120 LOC, actual ~370 LOC. Driven by the git-mtime cache, frontmatter parser, and the containerLike adapter — not gold-plating, just real work the spec under-estimated.
- Rule 3 implementation uses `spec.TypeInitiative` only. Spec text says "initiative (or epic)"; this workspace has no `TypeEpic`, so the gap is moot. Worth noting if/when `epic` type is added.
- No TODOs, no commented-out code, no hedging in the diff.
- Delivery-lead inserts are appended sentences on existing lines (not rewrites of the surrounding paragraphs). Confirmed by reading the diff — adjacent text is byte-identical.
- The active-spec branch in NEXT.md regeneration on commit can't pass `ActiveSpec` (commit-time has no session context). Spec acknowledges this; recency + high-impact branches still fire. Matches the spec's "Active-spec detection is best-effort" risk note.
