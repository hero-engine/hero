# Delivery audit — agent-hero-version-schema-confusion

**Audited:** `git diff main...HEAD` @ `7dba572` (branch `fix/agent-hero-version-schema-confusion`)
**Verdict:** SHIP
**Surface:** clean

Cold audit — auditor had no part in the delivery. Build re-run, all suites re-run, live `hero doctor` exercised. All three bundled concerns (engine Spec A, engine Spec B, harness Spec C) landed with real, asserting tests.

## Acceptance criteria (spec Test Plan items 1–5)
- [✓] **graph mismatch table test** — `internal/graph/mismatch_test.go`. `TestCheckSchemaMismatch` asserts binary-older = hard error containing `os.Executable()`, both schemas, `hero doctor`, and NOT `hero upgrade`; binary-newer = warn+continue; and the double-digit `("9","10")` case takes the warn branch. `TestSchemaLess` explicitly asserts `"9"<"10"` true and `"10">"9"` false — both FAIL under the old lexical compare, so the fix is genuinely guarded.
- [✓] **`hero doctor` test** — `internal/cli/doctor_test.go` `TestBuildDoctorReport`: asserts exe path, binary version, binary schema, graph schema, all three verdicts (agree/older/newer), the PATH-divergence flag firing, and graceful no-workspace degradation.
- [✓] **MCP initialize stamping** — `internal/serve/mcp_test.go` `TestMCP_Initialize_StampsSchema`: drives `initialize`, decodes the result, asserts `ServerInfo.Schema` == compiled schema and `ServerInfo.GraphSchema` == graph schema. Real assertions, not just import.
- [✓] **MCP dedup/supersede** — `internal/serve/mcp_singleton_test.go`: 5 tests — supersede-live-incumbent, stale-pidfile-as-free, reconnect-ends-with-new-server, distinct-clients-don't-collide, and an explicit `TestMCPSingleton_OrphanWatchdogStillFires` regression guard proving orphan reaping is intact.
- [✓] **Harness propagation per-target** — `internal/install/harness_native_test.go` `TestHarnessNative_DoctorRoutingGuidanceAllTargets`: renders all six targets (claude→CLAUDE.md, codex/opencode/cursor/copilot/generic→AGENTS.md), asserts three guidance substrings in each via `mustContain`. Would FAIL naming any dropped target. Verified running (all six subtests PASS).

## Changes (spec Fix Approach 1–4)
- [✓] **Fix 1 — self-locating schema-mismatch error + `hero doctor`** — `internal/graph/graph.go:315-366` (`checkSchemaMismatch`, `schemaLess`) + new `internal/cli/doctor.go` (registered `root.go:94`, version-check-skipped `root.go:41`). Direction correct: graph-newer→warn+continue (returns `""`,nil), binary-newer→hard error. Both messages print `os.Executable()`, both schemas, point at `hero doctor`, and state `hero upgrade` "will NOT help". Numeric compare via `strconv.Atoi`, not lexical.
- [✓] **Fix 2 — stamp version+schema on MCP surface** — `internal/serve/mcp_protocol.go` (`MCPServerInfo.Schema`/`GraphSchema`, additive + `omitempty`), `mcp.go` (reads graph schema at construction via `ReadSchemaVersion`, non-migrating), `mcp_lifecycle.go:143` stamps both on `handleInitialize`. Backward-compatible.
- [✓] **Fix 3 — MCP dedup/supersede per client+workspace** — new `internal/serve/mcp_singleton.go`; wired in `mcp_lifecycle.go:46` gated by `s.input == os.Stdin` (bytes.Buffer tests never touch the pidfile). Keyed by (heroDir, parent pid); distinct clients get distinct pidfiles; supersede signals only the recorded incumbent pid; stale holder detected via `IsProcessAlive` seam; ownership-checked release (only removes if `rec.PID == self`) so a superseded daemon can't delete its successor's file.
- [✓] **Fix 4 — routing guidance, all six targets** — authored once in `internal/install/agents_md.go:675` (`generateEngineeringAgentsMdBody`, the sole AGENTS body generator → feeds all six targets), mirrored to checked-in `domains/engineering/AGENTS.md:175`. Content: prefer MCP over bare `hero`, run `hero doctor` on mismatch, do NOT confabulate a migration story or run `hero upgrade`.

## Build & tests (re-run by auditor)
- `go build ./cmd/hero/` — clean.
- `go test ./internal/graph/... ./internal/serve/... ./internal/install/...` — all pass.
- `go test ./internal/cli/` — one failure, `TestMarkdownInvocationsResolveAgainstRootCmd`, on `web/docs/src/releases/index.md:213` (`hero verify`) and `:519` (`hero peer --peer`). That file is NOT in this diff; the failing invocations reference commands this diff does not touch. Pre-existing release-notes doc drift, unrelated. It is the ONLY cli failure.
- Live `hero doctor` run: prints running-binary path/schema, PATH resolution with the divergence WARNING when the running binary differs from PATH's `hero` (the Defect-2 signal, observed firing), workspace graph schema, and a correct verdict.

## Open items
- **Open Repro Step** (spec §"Open Repro Step") — capturing the literal Codex `db=X binary=Y` text from inside a Codex session — is explicitly a repro-only, non-code item that cannot be reproduced from this repo. Disclosed in the spec; not a delivery gap. Defect 2 was never a code fix (PATH is the caller's; the fix is detection + loud failure, delivered via doctor's PATH-divergence warning).

## Audit notes
- Spec recommended splitting into three specs (A/B/C); the engineer delivered all three on one branch with full coverage. Legitimate — nothing was narrowed or skipped.
- Two numeric-aware schema comparators now exist (`graph.schemaLess`, `version.CompareVersions` used by `doctorVerdict`). Both are correct at double digits (verified `CompareVersions` parses parts to ints). Minor duplication, not a defect — noted for future consolidation only.
- Diff scope is clean: engine + harness code/tests, the new spec.md, and the projected `.hero/NEXT.md|QUEUE.md|SNAPSHOT.md|next/*.md` handoff files (expected to travel with commits per project rule). No stray or unexplained changes.
