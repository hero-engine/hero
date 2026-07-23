# Delivery audit — deferred-work-suggestion-contract

**Audited:** `git diff HEAD -- . ':(exclude).hero/planning/initiatives/durable-attention/spec.md'` plus `git diff --no-index /dev/null` for the new skill, suggestion, project-registry, and MCP Focus files supplied by the orchestrator
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Structured proposals persist pending without creating Focus — `internal/attention/suggestion/store.go:64`, `internal/attention/suggestion/service.go:71`, and `TestProposalPersistsPrivatelyWithoutCreatingFocusAndReplays` verify private persistence and an empty Focus store; `TestMCPFocusSuggestionToolsAreStructuredAndConsentBounded` verifies the MCP boundary.
- [✓] Same proposal key and normalized payload replay once — `internal/attention/suggestion/store.go:75` returns the existing record only when `proposalEqual` matches; service and CLI replay assertions verify the original ID and JSON.
- [✓] Today and Later create or reuse one source-linked Focus row — `internal/attention/suggestion/service.go:177` selects the lifecycle and `internal/attention/suggestion/service.go:181` uses deterministic origin key `deferred_suggestion:<id>`; `TestAcceptTodayLaterDoNextAndDismiss` asserts state, replay, and one Focus row.
- [✓] Do Next accepts into Today and returns launch intent without session creation — `internal/attention/suggestion/service.go:171` validates launch resolution, `internal/attention/suggestion/service.go:177` retains Today, and `internal/attention/suggestion/service.go:210` builds the intent; service, CLI, and MCP tests assert exact prompt/project/path.
- [✓] Failed or cancelled launch leaves accepted Focus in Today — acceptance is committed before the launch intent is returned at `internal/attention/suggestion/service.go:181-197`; `TestAcceptTodayLaterDoNextAndDismiss` verifies the durable Today row after the response, with no rollback or session mutation path.
- [✓] Dismiss creates no Focus and returns authoritative dismissed proposal — the dismiss branch at `internal/attention/suggestion/service.go:160` only updates the proposal; the dismiss subtest verifies dismissed state and zero Focus rows.
- [✓] Stale, unsupported, missing, expired, and invalid actions return structured errors without commitment — `internal/attention/suggestion/service.go:121-158` validates and classifies action failures; `TestActionErrorsMakeNoCommitment`, CLI assertions, and the MCP structured-error assertion cover every named class and verify zero Focus rows.
- [✓] All six harnesses receive identical consent guidance without hooks — `TestAllTargetsInstallIdenticalDeferredWorkConsentGuidance` checks native OpenCode, Cursor, Claude, Copilot, Codex, and generic paths, byte equality, and required consent phrases; the diff adds no target-specific hook.
- [✓] Required current work is never auto-converted — `domains/engineering/skills/deferred-work-suggestions/SKILL.md:11-18` excludes required steps, acceptance criteria, ledgers, and harness todos, while the implementation exposes explicit CLI/MCP invocation only and adds no watcher.
- [✓] Consumers receive structured records and advertised actions without prose parsing — `contracts/attention/suggestion.go:3`, `internal/attention/suggestion/service.go:41-57`, and `internal/attention/suggestion/service.go:262-269` provide state, revision, DTOs, and action descriptors; CLI JSON and all three MCP tools have structured tests.

## Changes
- [✓] Add suggestion store/service and persistence tests — new `internal/attention/suggestion/store.go`, `service.go`, and `service_test.go` cover pending storage, replay, actions, expiry, retention, permissions, and recovery.
- [✓] Extend Focus CLI with suggest/list/action and stable JSON — `internal/cli/focus.go` adds `focus suggest`, `focus suggestions`, and `focus suggestion`, reads prompt/reason bodies from files or stdin, and supports JSON; `internal/cli/focus_test.go` covers invocation, replay, and errors.
- [✓] Add three MCP suggestion tools and dispatch — `internal/serve/mcp_tools_def.go`, `internal/serve/mcp_dispatch.go`, and new `internal/serve/mcp_tools_focus.go` define and dispatch structured suggest/list/action operations; `internal/serve/mcp_tools_focus_test.go` verifies definitions, dispatch, results, decimal revisions, and errors.
- [✓] Add canonical harness skill and Skills Reference — new `domains/engineering/skills/deferred-work-suggestions/SKILL.md` defines the consent contract; `domains/engineering/AGENTS.md` and `internal/install/agents_md.go` advertise it.
- [✓] Verify all six harness target installations — `internal/install/content_test.go:54` verifies native paths, identical bytes, and semantic consent boundaries across every supported target.
- [✓] Add Focus-create / receipt-commit failure recovery — `TestReceiptWriteFailureRecoversIdempotently` injects a proposal receipt failure after Focus creation and proves retry leaves exactly one Focus row and completes the receipt.
- [✓] Document advisory/ledger boundary — `domains/engineering/skills/deferred-work-suggestions/SKILL.md:20-38` states suggestions are advisory, require user acceptance, are not Completion Ledger output, and cannot replace required work.

## Open items (if any)
- None.

## Audit notes
- None.
