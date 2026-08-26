# Delivery audit — mail-d9dae6ef23521c42d2b46cfd

**Audited:** current worktree against `4cb6c40f80b37fa6869329898d0e928658241b60` with spec-scoped `git diff`, untracked `internal/install/agent_purpose.go` and `internal/install/agent_purpose_test.go`, and the version-matched Hero Code extraction
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Every installable Core, Engineering, PM, and Sales descriptor declares exactly one purpose — `TestCanonicalAgentPurposesCoverDynamicInventory` covers all 70 non-README descriptors; the independent extraction audit also found exactly one allowed declaration in every file.
- [✓] Only the six portable values are accepted — `internal/install/agent_purpose.go:15-34,98-109` closes the vocabulary to design, diagnose, agent, draft, review, and assist; focused vocabulary tests pass.
- [✓] Missing, empty, duplicate, and unknown canonical purpose fails before installation — `Run` validates production `ContentFS` before any target work (`internal/install/install.go:154-163`), without content sniffing (`internal/install/agent_purpose.go:43-63`). `TestCanonicalAgentPurposeRunRejectsEntirelyMissingPackBeforeWrites` exercises the real boundary and proves no target directory is created; deprecated custom/test `SourceDir` remains explicitly lenient.
- [✓] All seven harness targets preserve their native contracts — `TestCanonicalAgentPurposeInstallContractsAllTargets` installs and validates Claude, OpenCode, Cursor, Copilot, Codex, Generic, and Grok output.
- [✓] Purpose remains descriptor-owned without a central role map — all 70 descriptor diffs add one literal `purpose:` field; runtime code contains only the typed vocabulary and frontmatter parser, and `ContentManifest` is unchanged.
- [✓] Version-matched extraction preserves purpose in hashed bytes — Hero Code's manifest and this checkout both identify `v0.33.0-27-g4cb6c40f-dirty`; all 70 manifest hashes match staged bytes, all staged descriptors are byte-identical to Hero source, and every staged file has exactly one allowed purpose.

## Changes

- [✓] Add purpose to Core, Engineering, PM, and Sales descriptors — exactly 70 descriptor files changed by one metadata line each; the audited distribution is 18 agent, 3 assist, 17 design, 10 diagnose, 7 draft, and 15 review.
- [✓] Add typed canonical frontmatter validation — the production `ContentFS` boundary now fails closed even when every required purpose is absent, while QA-only descriptors and custom `SourceDir` agents remain outside the contract.
- [✓] Add inventory, rejection, and seven-target tests — focused tests cover dynamic inventory, vocabulary, malformed metadata, the no-write `Run` boundary, delivery-lead classification, and every supported target.
- [~] Publish raw-source contract and regenerate Hero Code content — the version-matched extraction and 70-file hash proof are complete; only the explicitly parent-owned Project Mail reply remains.

## Open items

- Change 4 — PARTIAL — Project Mail reply remains with the parent cross-repository delivery — concrete and explicitly outside this audit's authority; it does not block the delivered Hero-owned code and extraction contract.

## Audit notes

- The `SourceDir`/`ContentFS` boundary is now consistent: production install/init/upgrade/domain callers use strictly validated `ContentFS`; deprecated third-party/custom/test sources remain on the lenient `SourceDir` path.
