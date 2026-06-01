# Delivery audit — size-drift-actionable-output

**Audited:** `git diff HEAD` (uncommitted working tree)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Leaf-drift row prints second indented line with paste-ready `hero size <slug> <computed>` — `internal/cli/size.go:217-223`; live run shows e.g. `→ 'hero size roadmap-review large' to bump declared`.
- [✓] Container-unset emits `hero size <slug> <rollup>` to acknowledge + `/compose <slug>` to phase — `size.go:235-243`; live row for `hero-platform` matches.
- [✓] Container-low emits `hero size <slug> <rollup>` to bump declared + `/compose <slug>` to phase children — `sizing.go:306-308`; `TestSuggestedAction_Mapping/container-low` asserts exact strings.
- [✓] Leaf-down routes to `/split` alternative — `sizing.go:300-302`; `TestSize_Check_LeafDownEmitsSplitHint` and live row for `spec-size-and-promotion-nudge`.
- [✓] No template placeholders survive in output — `TestSize_Check_FindsDrift` asserts `<slug>`, `<tier>`, `%s` are absent; `TestSuggestedAction_Mapping` same belt-and-suspenders check on both clauses.
- [✓] `hero size --check` non-zero exit prints failure error exactly once — `size.go:43` (`SilenceErrors: true` on `sizeCmd` only); `cmd/hero/main.go` untouched; `TestSize_Check_ErrorPrintsOnce` regression; live run shows single stderr line `size drift found in 16 spec(s)` and exit code 1.
- [✓] Non-zero exit preserved when drift found — `size.go:249` returns `fmt.Errorf("size drift found in %d spec(s)", total)`; live exit code 1 confirmed.
- [✓] Footer `Run '/roadmap-review' to triage interactively.` after count summary — `size.go:248` literal match; live output matches verbatim.
- [✓] `hero_warnings` size-drift entries include alternative as second clause — `internal/serve/mcp_tools.go:2598-2600, 2620-2622` use `", or %s"` format; `TestToolWarnings_SizeDriftLeafAndContainer` asserts both `grown beyond intent` (leaf-up alt) and `'/compose init-drifted'` (container alt).
- [✓] `hero_warnings` substitutes computed/rollup tier — `mcp_tools.go:2599` interpolates `d.Bucket`/`d.Rollup` directly; test asserts no literal `<tier>` survives.
- [✓] Indeterminate container keeps single-clause form, no inline `→` line in CLI — `size.go:229-234` `if d.Indeterminate { continue }` skips the suggestion print; `mcp_tools.go:2607-2613` keeps existing single-clause form before the `kind`/`SuggestedAction` block.
- [✓] Tracker-capability header at top unchanged — `printTrackerCapability` (`size.go:256-264`) not touched in diff; live output shows the existing one-line header.
- [✓] `--check` JSON contract preserved — no JSON output path exists for `--check` (confirmed via `grep -i json` on `size.go`), so trivially unchanged.

## Changes

- [✓] `internal/sizing/sizing.go` — `DriftKind` + 4 constants (`DriftKindLeafUp/Down/ContainerUnset/ContainerLow`), `driftSizeTierOrder`, `ClassifyLeafDriftKind`, `SuggestedAction` all present at `sizing.go:230-311`.
- [✓] `internal/cli/size.go` — `SuggestedAction` wired into `runSizeCheck` print loop (lines 217-243), indeterminate-container skip (229-234), footer `Run '/roadmap-review'…` (248), `SilenceErrors: true` flag flip with inline Option C comment (38-43).
- [✓] `internal/serve/mcp_tools.go` — leaf and non-indeterminate container `hero_warnings` entries now carry `Run <primary>, or <alternative>` (lines 2598-2600, 2620-2622); indeterminate kept single-clause (2607-2613); no new MCP fields added.
- [✓] Tests — `TestSuggestedAction_Mapping` (table across 4 kinds), `TestClassifyLeafDriftKind`, `TestSize_Check_FindsDrift` (extended), `TestSize_Check_NoDrift_QuietFooter`, `TestSize_Check_LeafDownEmitsSplitHint`, `TestSize_Check_ErrorPrintsOnce`, extended `TestToolWarnings_SizeDriftLeafAndContainer`. All pass: `go test ./internal/sizing/... ./internal/cli/... ./internal/serve/...` clean.

## Open items

- None marked PARTIAL/SKIPPED/BLOCKED by the engineer.

## Audit notes

- **Scope drift — `hero size --ack` flag landed in this delivery.** The diff to `internal/cli/size.go` adds the `--ack <tier> <slug>` flag (`size.go:77`), `runSizeAck` function (`size.go:166-186`), validation (`size.go:92-94, 102-106`), updated `Use`/`Short`/`Long` text, plus six new `TestSize_Ack_*` tests in `size_test.go:398-549`. None of this appears in the spec's Goal, Approach, Changes, or Acceptance Criteria. The orchestrator prompt explicitly stated `--ack` was intended for a **follow-up spec** (and that the SKILL/agent reverts protect the `--ack` fallback until that follow-up ships). It shipped here instead. Net ~150 lines outside spec scope. Functional and tested, but not authorized by this spec. Worth surfacing before the spec flips to completed — either retroactively expand this spec's Changes section, or carve out a separate `hero-size-ack` micro-spec to receive credit for the work.
- **Naming caveats are sound.** `ClassifyLeafDriftKind` exported because both `internal/cli/size.go` and `internal/serve/mcp_tools.go` consume it (confirmed via grep). `DriftKindLeafUp` etc. carry the `Kind` infix because `Estimate.Drift` (`sizing.go:41`) is a bool field that would collide with a plain `DriftLeafUp` constant in package-local readers. Both choices match standard Go practice.
- **Live binary verification.** Built `cmd/hero` HEAD-of-worktree; ran `hero size --check` against this workspace — produced 16 drift rows (6 leaf, 10 container) with correct inline hints, footer printed exactly once, single stderr line, exit 1. Ran on a fresh `hero init` workspace — `No size drift detected.`, no footer, exit 0.
- **Indeterminate path lightly exercised.** No indeterminate-container fixtures exist in this workspace's data, but the code path is straightforward (`continue` in CLI loop, `continue` after single-clause append in MCP) and unit-test coverage gaps here are minor. Not blocking.
