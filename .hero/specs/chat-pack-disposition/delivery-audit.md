# Delivery audit — chat-pack-disposition

**Audited:** uncommitted working tree — `git diff --stat -- content.go content_test.go domains/chat/ .hero/planning/initiatives/content-remediation/spec.md .hero/planning/initiatives/content-remediation/chat-pack-disposition/spec.md` (plus `domains/chat/AGENTS.md`, untracked)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC1 — content.go documents chat as non-installable, names build.rs as consumer — `content.go:8-20` (package comment) and `content.go:280-289` (`clientEmbeddedDomains` var doc)
- [✓] AC2 — `DomainFS("chat")` returns error, asserted by test — `content_test.go:413-421`, `TestDomainFS_ChatIsClientEmbedded`; re-run: PASS. Error text confirmed at `content.go:82` (`domain %q not found — available domains: %v`) — mentions "chat" via `%q` and lists `AvailableDomains()`.
- [✓] AC3 — unlisted `domains/` dir fails the content test suite — `content_test.go:435-461`, `TestDomainsDirectory_AllEntriesAccounted`; re-run: PASS on current tree. Independently created a throwaway `domains/_audit_stray_fixture/` directory and re-ran the test — it failed with `domains/_audit_stray_fixture is neither in AvailableDomains() nor clientEmbeddedDomains — ...`. Removed the fixture and re-ran — PASS again. `domains/` is clean (`git status --porcelain -- domains/` shows no stray fixture left behind).
- [✓] AC4 — `AvailableDomains()` unchanged (engineering, sales, pm) — `content.go:277-279`, byte-identical to pre-delivery; `TestAvailableDomains` re-run: PASS.
- [✓] AC5 — ask-corpus.md search step is capability-phrased, not tool-identifier-phrased — `domains/chat/commands/ask-corpus.md:11-14`: "whatever semantic-search capability the session exposes, falling back to a plain text/grep search" / "the session's file-read capability." Old `semantic_search`/`read_file` literals removed. No other lines in the file changed (diff is a 2-line→4-line reword of step 1 only).
- [✓] AC6 — space.md SpaceStore instruction scoped to hero-code (GPUI), generic path is default — `domains/chat/commands/space.md:14-17`: default step is "surface a brief summary the user can paste into their client's new-space (or new-thread) dialog"; `SpaceStore` API use is explicitly conditioned on "In the hero-code (GPUI) client specifically." Only step 3 changed; rest of file identical.
- [✓] AC7 — `domains/chat/AGENTS.md` ships, routes only the 6 commands — new file confirmed on disk. Routing table lists exactly `/ask-corpus`, `/capture`, `/discover`, `/note`, `/space`, `/why` — a 1:1 match against `ls domains/chat/commands/*.md` (no extra, no missing). Heading structure: `# Hero Chat` (H1) + `### Natural Language Routing` (H3) — matches the spec's requirement to use `###` (not `##`), consistent with `domains/engineering/AGENTS.md`'s section-heading depth. No relative links found (`grep` for `](./`, `](../`, `](/` returns nothing).
- [✓] AC8 — `go test ./...` passes, parity test unmodified — re-ran `go test ./...`: all packages `ok`, no `FAIL` lines, exit 0 (unrelated knowledge-surfacing WIP packages also compiled/passed, as expected). `content_parity_test.go` has zero diff and zero git-status entry — confirmed untouched. Re-ran targeted set `TestDomainFS_ChatIsClientEmbedded|TestDomainsDirectory_AllEntriesAccounted|TestDomainPacks_NoUnannotatedCoreShadows|TestAvailableDomains` — all 4 PASS (including subtests engineering/sales/pm under `TestDomainPacks_NoUnannotatedCoreShadows`).

## Changes

- [✓] C1 — content.go documents the intentional exclusion — package comment (`content.go:8-20`) names hero-code's `crates/hero-core/build.rs` as the build-time consumer; `clientEmbeddedDomains` map (`content.go:280-289`) with `"chat": true` and a comment naming the consumer and citing the spec.
- [✓] C2 — new test: DomainFS("chat") errors + domains/ taxonomy allowlist — both tests present in `content_test.go`, re-run and independently fixture-verified (see AC3).
- [✓] C3 — ask-corpus.md capability language — step 1 rewritten, verified above.
- [✓] C4 — space.md SpaceStore scoped — step 3 rewritten, verified above.
- [✓] C5 — domains/chat/AGENTS.md minimal routing spine — new file, verified structure/content above.
- [✓] C6 — close the audit loop in content-remediation tracking — `.hero/planning/initiatives/content-remediation/spec.md` gains a "Wave 1 progress" section (new, ~17 lines) naming F9, F29, and the routing S3 finding as resolved by this delivery, and records all five wave-1 children (including this one) as delivered as of 2026-07-08. Cross-checked: `pm-pack-phantom-surfaces` and `sales-pack-reality-sync` both exist under `.hero/specs/` (archived), and `core-commands-domain-neutral` / `delivery-gate-consistency` both appear in recent merge commits (`git log`) — the "all five delivered" claim is consistent with repo state, not asserted in isolation.

## Open items (if any)

None. The Completion Ledger has no PARTIAL/SKIPPED/BLOCKED rows — all 14 rows (C1-C6, AC1-AC8) are DONE, and every one had concrete, independently-reproducible evidence.

## Audit notes

- **Boundaries honored.** Confirmed by direct inspection of `content.go`: no new `go:embed` directive for chat (grep shows only engineering/sales/pm/core embeds), no `DomainFS` case for `"chat"` (switch statement only has engineering/sales/pm, falls through to the not-found error), `AvailableDomains()` unchanged, `content_parity_test.go` has no diff, and `grep -rn "core_fork" domains/chat/` returns nothing — no annotations added to chat files. This matches every item in the spec's Boundaries section.
- **Path citations in content.go** (`content.go:17`, `content.go:289`) point to `.hero/specs/chat-pack-disposition/spec.md`, which does not exist yet — the spec currently lives at `.hero/planning/initiatives/content-remediation/chat-pack-disposition/spec.md` (`status: planning`). This is not a defect: sibling completed specs in this initiative (e.g. `pm-pack-phantom-surfaces`, `core-commands-domain-neutral`) archive to the flat `.hero/specs/{slug}/spec.md` form on completion, so the citation is a correct forward-reference to where this spec will land once it archives, not a broken link today. Flagging only so the orchestrator is aware the link is currently dead until archival.
- Diff is tightly scoped to the six files/dirs named in the spec's own "Files" list plus the two initiative spec.md files recording the ledger/progress — no scope drift.
- Fixture cleanup verified: no stray file or directory left under `domains/` after the negative-case test.
