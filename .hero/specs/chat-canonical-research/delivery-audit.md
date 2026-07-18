# Delivery audit — chat-canonical-research

**Audited:** `git diff 145c4a8` (tracked) + untracked `domains/chat/{agents,skills}` and `commands/research.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 Six existing commands unchanged — `git diff 145c4a8 --stat` on the six command files is empty (no behavioral or wording change).
- [✓] AC-2 `/research` command complete — `domains/chat/commands/research.md` has `description:` frontmatter and the ordered body: restate `$ARGUMENTS` → plan → pause-for-approval → controlled source set → search rounds w/ per-round summaries → source evaluation → cited synthesis → `plan → round → evaluation → synthesis → report` checkpoints (lines 16-39).
- [✓] AC-3 Plan pauses before search — research.md step 2 ("Emit … `plan` checkpoint and **wait**") and `research-workflow/SKILL.md` Phase 1 (lines 46-65) both mark pre-search approval mandatory.
- [✓] AC-4 Interrupt → partial report banner — `research-workflow/SKILL.md` "Interrupt safety" (lines 134-146) specifies the "Incomplete — stopped after round K" banner + partial contract; echoed in research.md:41-44 and researcher.md:63-65.
- [✓] AC-5 Exactly three agents, name==stem — `domains/chat/agents/` = `researcher.md`, `document-analyst.md`, `data-analyst.md`; each `name:` matches its stem; `TestChatPack/agents` asserts frontmatter and passes.
- [✓] AC-6 Exactly five hidden skills — `domains/chat/skills/` = research-workflow, source-evaluation, evidence-and-citation, document-analysis, data-analysis; each `SKILL.md` has non-empty `description:`; `TestChatPack/skills` passes.
- [✓] AC-7 AGENTS.md 7 rows + hidden note + NL-intents line + one prose rule — routing table has exactly 7 command rows; "Hidden agents & skills" note; "These stay natural-language intents" section; one "Writing prose for other people" section; no write command/writer agent.
- [✓] AC-8 Routing bidirectional — `TestChatPack/commands_and_routing` builds shipped set from disk, asserts `== want` (7 files), extracts routing rows from `|`-prefixed AGENTS.md lines via regex, asserts `routed == shipped`. Non-vacuous, both directions. Passes.
- [✓] AC-9 Go content test — `go test . -run TestChatPack -v` PASS (agents/skills/commands_and_routing subtests all green).
- [✓] AC-10 Skip when `domains/` absent — `content_test.go` guards `os.Stat(chatRoot)` → `os.IsNotExist` → `t.Skip`, mirroring `TestDomainsDirectory_AllEntriesAccounted`.
- [✓] AC-11 Client-embedded invariants — `grep chat content.go` shows chat only in comments + `clientEmbeddedDomains["chat"]=true` (line 295); not in any `go:embed`, `AvailableDomains()`, or `DomainFS` case. `TestDomainFS_ChatIsClientEmbedded` + `TestDomainsDirectory_AllEntriesAccounted` PASS.
- [✓] AC-12 No client-private sole path — every agent + research.md + research-workflow state the "describe abstractly, name a client only as an aside" rule; the one client-private mention (hero-code GPUI approval card, researcher.md:72) is an explicit aside, not a sole path.
- [✓] AC-13 content.go comment-only — `git diff content.go` touches only `//` header comment lines (10-21); `go build ./...` clean.
- [✓] AC-14 Summarize/compare/explain/brainstorm stay NL intents — AGENTS.md "These stay natural-language intents" section; no such command or agent shipped.

## Changes
- [✓] 1 Add `commands/research.md` — present, thin orchestration, delegates to `researcher` where subagents exist else inline.
- [✓] 2 Add three agents — researcher/document-analyst/data-analyst, standard agent frontmatter shape, each loads its skills.
- [✓] 3 Add five hidden skills — all present; research-workflow is the authoritative checkpoint/interrupt contract.
- [✓] 4 Rewrite AGENTS.md — /research row added, six kept, "no agents/skills" paragraph replaced, NL-intents line, one prose rule, client-embedded framing kept.
- [✓] 5 Add chat content test — `TestChatPack` + `assertStringSetsEqual` helper; skips cleanly when `domains/` absent.
- [✓] 6 Update content.go header comment — comment-only.

## Open items
None. No PARTIAL / SKIPPED / BLOCKED rows in the ledger.

## Audit notes
- All test evidence re-run independently: `go test . -run 'TestChatPack|TestDomainFS_ChatIsClientEmbedded|TestDomainsDirectory_AllEntriesAccounted' -v` all PASS; `go build ./...` and `go vet .` exit 0; `hero spec lint chat-canonical-research` → 14/14 EARS (1 event, 11 ubiquitous, 2 unwanted).
- Challenge points cleared: (1) no sole-path client-private dependency — client symbols appear only as asides; (2) exactly three agents, zero summarizer/comparer/explainer/writer; (3) exactly one prose rule, gated to human-facing prose, no write command/agent; (4) content.go change is comment-only, chat stays out of embed/AvailableDomains/DomainFS and in clientEmbeddedDomains; (5) routing table has exactly seven rows and the test enforces a real bidirectional match; (6) no performative ledger rows — every DONE maps to on-disk evidence.
- Pre-existing client-agnostic smells in the untouched `ask-corpus.md`/`space.md` are out of scope for this spec (the six commands were deliberately left unchanged) and are not a finding against this delivery.
- Diff is well-scoped: only the spec's named files were touched.
