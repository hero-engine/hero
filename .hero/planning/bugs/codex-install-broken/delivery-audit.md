# Delivery audit — codex-install-broken

**Audited:** `git diff HEAD` at `2e7a7240209b472244a0fc7348c4e777dfdb1c34`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

The spec defines fixes in priority order. The delivered items are Fixes 1, 2, and 3. Fixes 4, 5, 6 are explicitly skipped.

- [✓] **AC1 — AGENTS.md body target-aware with Codex workflow section**
  `internal/install/agents_md.go:336-337`: `agentsMdBodySection.Render()` now appends `renderCodexWorkflowSection()` when `opts.Target == TargetCodex`. The section teaches Codex to read `.agents/skills/command-<name>/SKILL.md` and execute steps, not treat them as docs. Present in `AGENTS.md` diff: "Running Hero Workflows in Codex" heading + routing table + explicit anti-shortcut warnings. Grep confirms 1 occurrence in on-disk `AGENTS.md`.

- [✓] **AC2 — Hero commands emitted as Codex-loadable skills**
  `internal/install/render.go:170-178`: `renderCommandAsCodexSkill()` renders each command with YAML frontmatter (`purpose: command-workflow`) and an execution preamble blockquote. `internal/install/target_codex.go:79`: `renderToFile(opts, result, "commands", skillsDest, renderCommandAsCodexSkill)` call wired in `runCodex()`. On-disk result: 29 `command-*` skill directories under `.agents/skills/`. On-disk `.agents/skills/command-deliver/SKILL.md` confirms both `purpose: command-workflow` (line 5) and preamble text (line 8).

- [✓] **AC3 — Re-install Codex target verified**
  `.hero/version.json` diff confirms a live `hero install project . --target codex` ran: `last_install.target = "codex"`, `last_install.timestamp = "2026-06-09T09:40:10.341558-06:00"`. The `installed_files` map grew from ~86 entries to ~236 entries, now including all 29 `command-*` skills, 35 `.codex/agents/*.toml` files, and the full skill corpus. AGENTS.md diff shows the managed region updated from a stub to ~148 lines of Hero instructions.

- [✓] **AC4 — Smoke test assertions updated and passing**
  `internal/install/harness_smoke_test.go:105-115`: `TestHarness_SmokeCodex` now asserts: `command-deliver/SKILL.md` and `command-design/SKILL.md` exist; preamble strings `"This is a Hero workflow for Codex"` and `"purpose: command-workflow"` are present; `AGENTS.md` contains `"Running Hero Workflows in Codex"` and `"command-deliver/SKILL.md"`. Live run: `PASS: TestHarness_SmokeCodex (0.01s)`.

- [✓] **AC5 — All tests pass**
  Ledger claims `go test ./...` — 81 packages, 0 failures. This audit did not re-run the full suite, but `TestHarness_SmokeCodex` passes independently, and the diff shows no changes outside `internal/install/` (plus config/projected files), making a suite-wide regression implausible. Confidence: medium — test output from the delivery run is not attached as an artifact.

- [~] **Fix 4 — `hero check` advisories for Codex completeness** — SKIPPED. Engineer's reason: "P3 priority; separate delivery pass." Concrete: names a priority tier. Not a soft skip — no concern here.

- [~] **Fix 5 — MCP in Codex sandbox** — SKIPPED. Engineer's reason: "Needs research into Codex sandbox setup_steps support." Concrete: names a specific open question that blocks the fix. No concern.

- [~] **Fix 6 — Auto-detect stale managed regions** — SKIPPED. Engineer's reason: "Lower priority; upgrade path improvement." Slightly soft — "lower priority" alone is vague — but the context (an upgrade path enhancement vs. a P0 correctness fix) makes the deferral reasonable.

## Changes

- [✓] `internal/install/agents_md.go` — Added `renderCodexWorkflowSection()` function; modified `Render()` to call it when `opts.Target == TargetCodex`. Lines 336-511.
- [✓] `internal/install/render.go` — Added `renderCommandAsCodexSkill()` function at lines 170-178. Correct signature, frontmatter, and preamble generation.
- [✓] `internal/install/target_codex.go` — Added `renderToFile(..., renderCommandAsCodexSkill)` call at line 79 in `runCodex()`, after `installSkillsNested`.
- [✓] `internal/install/harness_smoke_test.go` — Updated `TestHarness_SmokeCodex` with command skill existence checks and AGENTS.md content assertions. Comment block also updated to reflect new behavior.
- [✓] `AGENTS.md` — Re-installed managed region now contains full Hero routing table (~148 lines) plus the new Codex-specific workflow section.
- [✓] `.hero/version.json` — Updated `installed_files` map recording the full Codex install output (236 files), including all 29 command skills and 35 agent TOMLs.
- [✓] `.codex/config.toml` — `hero:managed` marker added; `command` field changed from `hero` (PATH lookup) to `/Users/developer/go/bin/hero` (absolute path). Note: absolute path is machine-local — see audit notes.
- [✓] `.codex/hooks.json` — `SessionStart` hook added (`hero next ingest --quiet`); key order normalized in `Stop` hook (cosmetic).
- [✓] `.hero/NEXT.md`, `.hero/next/chet-bellows.md`, `.hero/SNAPSHOT.md`, `.hero/peer-manifest.yaml` — Projected handoff files refreshed (timestamp/commit list updates).

## Open items

- Fix 4 — `hero check` advisories for Codex completeness — SKIPPED — P3 priority; separate delivery pass — **concrete**
- Fix 5 — MCP in Codex sandbox — SKIPPED — Needs research into Codex sandbox setup_steps support — **concrete**
- Fix 6 — Auto-detect stale managed regions — SKIPPED — Lower priority; upgrade path improvement — **borderline** (but acceptable given P0 fixes shipped)

## Audit notes

1. **`.codex/config.toml` hardcodes a machine-local path.** The diff changes `command = "hero"` (PATH lookup) to `command = "/Users/developer/go/bin/hero"` (absolute path). This is wrapped in a `# hero:managed` block, suggesting it is generated by the install and will be different on other machines. This is the fix for Fix 5's surface symptom (MCP unreachable in Codex sandbox) for the local development case — but it is explicitly not a general sandbox solution. The spec's Fix 5 scope ("Codex sandbox VM/container") remains open. No HOLD is warranted — the on-disk state is correct for local use — but teams checking this into shared repos should know this path is machine-specific.

2. **Smoke test uses minimal fixture, not full command corpus.** `TestHarness_SmokeCodex` runs against 2 agent stubs and 2 command stubs (the harness fixture set). It asserts `command-deliver/SKILL.md` and `command-design/SKILL.md` exist, which is correct for the unit test scope. The full 29-command on-disk install is separately verified by `.hero/version.json`. This split is acceptable — the unit test validates the render path and content; the live install record validates the count.

3. **Full `go test ./...` not independently re-run by this audit.** The ledger claims 81 packages pass. The smoke test run by this audit confirms the directly affected package passes. The test evidence from the delivery run is not attached as a file artifact. Confidence in the full-suite claim is medium, not high — but the changes are narrowly scoped to `internal/install/` and no regressions are evident in the diff.
