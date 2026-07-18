# Delivery audit — chat-slim-to-basic-research-seed

**Audited:** working tree vs `HEAD` (`git status --porcelain` + untracked `seeds/`), byte-identity vs `3a09d27`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 chat = six commands, no agents/ or skills/ dir — `ls domains/chat` shows only `AGENTS.md` + `commands/` (six files: ask-corpus, capture, discover, note, space, why); `TestChatPack/basic_chat_shape` asserts the six-set and stat-guards both dirs.
- [✓] AC-2 six commands behaviorally unchanged — no command file appears in `git diff`; only `AGENTS.md` and `content_test.go` are modified under the chat tree.
- [✓] AC-3 no `/research`-workflow advertising in chat — `grep -rin 'research|checkpoint|plan approval|/research' domains/chat` returns only soft-habit / negation lines in `AGENTS.md` (lines 25, 33, 35, 56, 62); no `/research` routing row.
- [✓] AC-4 seed 9 files byte-identical to `3a09d27` — per-file `diff <(git show 3a09d27:domains/chat/<p>) seeds/hero-research/<p>` clean for all nine (research.md, 3 agents, 5 SKILL.md).
- [✓] AC-5 seed outside `domains/`, un-stageable — all 10 seed files under `seeds/hero-research/`; `find seeds -type f | grep -c '^domains/'` = 0; none is git-tracked yet (untracked tree, invisible to a `domains/`-only stager regardless).
- [✓] AC-6 seed README (provenance/removal/dormancy/revival) — `seeds/hero-research/README.md` carries all four: provenance (chat-canonical-research), removal (chat-sheds-research-to-seed), dormancy (not staged/embedded/domain), revival = new decision.
- [✓] AC-7 chat client-embedded invariant intact — `content.go` unchanged; chat still `clientEmbeddedDomains["chat"]=true` (line 293), no `go:embed` chat line, no `DomainFS` case; `TestDomainFS_ChatIsClientEmbedded` + `TestDomainsDirectory_AllEntriesAccounted` green.
- [✓] AC-8 content test: shape + no agents/skills + seed frontmatter + skip — `TestChatPack` two subtests pass; skip loop stats both `domains/chat` and `seeds/hero-research` via `os.IsNotExist`.
- [✓] AC-9 test fails if `domains/chat/agents` or `skills` exists — empirically verified: creating `domains/chat/agents/` made `basic_chat_shape` FAIL at content_test.go:546 ("research must not be re-baked into basic chat"); non-vacuous. (temp dir removed, not committed.)
- [✓] AC-10 AGENTS.md soft habits + one prose rule, no `/research` row — one "Writing prose for other people" rule; habits are soft ("Ground factual claims", "Look things up", "Read what the user shares"); routing table has exactly the six rows.
- [✓] AC-11 chat-canonical-research unmodified (status + ledger) — status still `completed`, delivery ledger untouched. NOTE: the spec file IS modified in the working tree, but only append-only peer-call trail entries at the tail (not status, not ledger) — AC-11's verify condition holds.
- [✓] AC-12 content.go comment accurate — `git diff content.go` is empty; header comment already describes chat as commands-only client-embedded (no repair needed).

## Changes
- [✓] Recover 9 research files into `seeds/hero-research/` — all byte-for-byte vs `3a09d27`.
- [✓] Add `seeds/hero-research/README.md` — provenance + dormancy + revival, all present.
- [✓] Confirm `domains/chat` is basic Chat — verified; no residual research surface.
- [✓] Update `TestChatPack` (guard + seed validation) — `basic_chat_shape` + `research_seed` subtests; new `assertDescriptionFrontmatter` helper; reuses `assertAgentFrontmatter` (checks name==stem, content_test.go:125) and `assertSkillFrontmatter`.
- [✓] Confirm `content.go` header comment — accurate, comment-only, no change needed.

## Open items
- None blocking.

## Audit notes
- **Governance verified independently:** `chat-app-stays-single-surface` is `status: superseded`; `chat-sheds-research-to-seed` is `status: accepted`. Single accepted governor, as the spec requires.
- **AC-11 literal-vs-ledger nuance:** ledger row 11 states "not touched this delivery," but `.hero/specs/chat-canonical-research/spec.md` shows a working-tree modification — two appended outbound peer-call trail entries (advisory reversal notices to hero-code). This is trail bookkeeping, not a status/ledger edit, so AC-11's stated verify (no status/ledger change) still passes. Flagged only so the user knows the file is dirty.
- **Scope beyond named files:** working tree also carries modifications to `.hero/` handoff/decision/trail artifacts (NEXT.md, SNAPSHOT.md, events.log, the superseded decision, the sheds decision spec, two peer-call records). These are Hero governance bookkeeping consistent with the spec's Context, not code drift.
- **Build/test:** `go build ./...` clean, `go vet .` clean, full `go test .` passes, targeted `go test . -run 'TestChatPack|TestDomainFS_ChatIsClientEmbedded|TestDomainsDirectory_AllEntriesAccounted' -v` all green.
