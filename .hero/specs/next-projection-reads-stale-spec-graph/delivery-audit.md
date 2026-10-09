# Delivery audit — next-projection-reads-stale-spec-graph

**Audited:** `git diff 4658b18e..8042ee53` (branch fix/next-projection-stale-graph, checked out at 8042ee53). Re-audit after fixes; first pass covered 4658b18e..93646d9c and returned HOLD.
**Verdict:** SHIP
**Surface:** noteworthy

Executed vs read: EXECUTED = `go vet ./...` (clean); `go test ./internal/projection ./internal/spec ./internal/workmodel ./internal/serve ./internal/cli -count=1` (all ok, cli 96 s); the CI drift-gate recipe on this worktree at 8042ee53 (delete graph.db, `hero scan`, `hero next checkpoint --quiet`, `git diff -I'^updated: ' --exit-code -- .hero/NEXT.md`: no drift); falsification of every new test against the earlier code (first pass: projection, spec, serve, checkpoint reconcile; second pass: hook-payload and next-project tests); warm vs fresh vs `graph reingest` comparison and checkpoint timing (first pass). READ ONLY = git-history root-cause evidence, SQL/Go ranking parity, consumer greps, goroutine-leak and timing reasoning. Not executed: a held-open-stdin repro of the real binary (the sandbox refused the construct); the coordinator reports reproducing it with a goroutine dump and the unit test covers the same path.

## Acceptance criteria
- [✓] AC-1 checkpoint/`hero next project` project current status: `internal/cli/checkpoint.go:292` and `reconcileCheckpointSpecGraph`, `internal/cli/next_project.go:71`. `TestCheckpointProjectsCurrentSpecStatus` fails with the 4658b18e checkpoint.go/next_project.go (executed). `TestNextProjectProjectsCurrentSpecStatus` fails when next_project.go is reverted to 93646d9c (executed), closing the first-pass test gap.
- [✓] AC-2 priority ranking in Next (both slots) and Context to carry: `internal/projection/projection.go` `priorityRank`/`priorityRankSQL` (SQL CASE and Go switch agree: lower+trim, p0/critical=0 .. p3/low=3, else 9). `TestNextMD_PriorityConventionsRankTogether`, `TestNextMD_CriticalBugInSlot2` fail on the old projection.go (executed). `TestSpecPropsCreatedAndCompleted` fails on the old graph_ingest.go (executed). The spec text is now correct that `hero_work` Suggested needed no change.
- [✓] AC-3 Just finished from completed_at (max 5): `recentlyCompleted`; `TestNextMD_JustFinishedListsRecentlyCompleted` fails on old code (executed).
- [✓] AC-4 hero_handoff body-only `markdown` plus `updated_at` and `repo`: `internal/serve/mcp_tools_read_contract.go` `handoffBody`; `TestToolHandoff`; golden additive-only passes. Shape now documented in the tool description (`mcp_tools_def.go:478`), `web/docs/src/cli/server-and-mcp.md:58` and an amendment in the archived read-contract-v1 spec.
- [✓] AC-5 no hang on silent stdin: `readHookPayload` (`internal/cli/next_compact_handoff.go`). `TestResolveSessionContextDoesNotBlockOnSilentStdin` FAILS against the 93646d9c file (hits the 5 s guard; executed). `TestResolveSessionContextReadsPromptPayload` passes.

## Changes
- [✓] 1. checkpoint reconcile, next project reconcile, tests: present, falsified.
- [✓] 2. projection, graph_ingest props and hash: present, falsified. The `polish.go` and `polish_test.go` changes are reverted (diff to 4658b18e is empty for them).
- [✓] 3. hero_handoff shape, code, tests, golden, tool description, docs, archived-spec amendment.
- [✓] 4. `readHookPayload` and `hook_payload_test.go`.

## Prior blockers
- B1 committed NEXT.md stale: RESOLVED. At 8042ee53 a fresh graph + `hero scan` + `hero next checkpoint --quiet` leaves `.hero/NEXT.md` byte-identical apart from `updated:` (executed). Caveat for the closing gate: `hero spec verify`/completion flips this spec's status, which can change NEXT.md again; regenerate and re-check after that flip (the pre-commit hook must use the built binary, not a stale PATH one).
- B2 stale hero_handoff docs: RESOLVED (see AC-4).

## Open items
- None. No PARTIAL/SKIPPED/BLOCKED ledger rows.

## Audit notes
1. **Root cause** (unchanged from first pass, confirmed): no spec-write path ever refreshed the spec subgraph; `git log -G'spec\.WriteGraph\('` shows only signature/key changes and the move of the blocked heal into `reconcileSpecGraph`; that heal was used only by `hero why`/`hero blocked`; the NEXT.md projection never reconciled. "Something broke" is NOT true; this was a long-standing gap.
2. **Determinism** sound (first pass, executed): fresh scan == warm rerun == after `graph reingest`; `created` and `completed_at` come only from authored frontmatter; no clock/mtime dependence in the new sections. Re-confirmed by the clean drift-gate run at 8042ee53.
3. **Hang fix scrutiny.**
   - Goroutine leak: acceptable. All callers of `resolveSessionContext` (checkpoint.go:102, next_compact_handoff.go:102/120, resolveSessionID) are short-lived CLI processes, so a goroutine parked in `Read` dies with the process; `hero serve` does not use it. The result channel is buffered (size 1) so a late reader never blocks on send, and the panic path also sends on that channel.
   - 1 s budget: reasonable. Harnesses write the payload at spawn, so it normally arrives in milliseconds. The cost is a fixed 1 s delay per checkpoint when stdin is open and silent (IDE/agent-run git hook), versus an unbounded hang before. Trade-off: on a pathologically loaded machine a payload arriving after 1 s is dropped, which loses only the best-effort auto UserAsk and falls back to the registry lookback. Non-blocking.
   - Other stdin reads on hook paths: the only unbounded hook-payload read was `resolveSessionContext`; `grep` shows no other `os.Stdin`/`stdin.Read` on checkpoint, compact-handoff or session-id paths. Remaining `InOrStdin` users (connect --token-stdin, focus, export, code-host broker) are explicit user commands, not hooks.
4. **hashSpec props / ContentHash**: one-time rewrite of every spec node (one history version each); only consumer of node ContentHash is the upsert dedupe (`internal/graph/node.go:179`). Acceptable.
5. **Performance** (first pass, executed): warm checkpoint 0.7 to 2.1 s, full scan about 4 s, no concern.
6. **hero_handoff within schema v1**: additive field plus a narrowing of `markdown`; no in-repo consumer relied on verbatim content; docs and archived contract now say so.
7. **Nits (non-blocking).** The original doc comment for `resolveSessionContext` ("TranscriptPath is populated only when...") now sits above the new `hookPayloadWait` const instead of directly above its function. `go-tests.log` (216 lines) and an empty `vet.log` remain committed under the planning dir. `graph.Open` still sets no `busy_timeout`, so concurrent hero processes get an immediate SQLITE_BUSY that `reconcileSpecGraph` swallows (a stale projection for that run, not a hang); out of scope.

## Tests
- `go vet ./...`: clean.
- Five-package `go test -count=1`: all ok at 8042ee53.
- Falsification: first pass (projection 3, spec 2, serve build fail, checkpoint reconcile 1) and second pass (hook-payload test fails at the 5 s guard on old code; next-project test fails without the reconcile call). The reverted `TestSuggestedRanksPLevels` is gone, as intended.
