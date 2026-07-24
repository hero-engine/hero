# Delivery audit — attention-mcp-action-tools

**Audited:** `git diff 9a3795d` plus all implementation/test files listed by `git status --short`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: resolved Mail send creates once and returns an authoritative delivery — `internal/serve/mcp_tools_mail.go:13-40`; `TestMCPMailSendReplyAreTypedIdempotentAndInert` decodes the receipt and verifies replay identity at `internal/serve/mcp_tools_mail_test.go:118-149`.
- [✓] AC-2: Mail reply preserves the authoritative thread and stable sender target — `internal/attention/mail/service.go:130-210`; the end-to-end assertion checks thread, `in_reply_to`, and recipient peer at `internal/serve/mcp_tools_mail_test.go:173-197`.
- [✓] AC-3: explicit-user Focus create preserves prompt, lifecycle, optional project, and authoritative source — `internal/serve/mcp_tools_focus.go:41-91`; bound, unbound, and maximum-length-key creation are asserted at `internal/serve/mcp_tools_focus_test.go:79-172`. The operation-scoped key is a bounded SHA-256 derivation at `internal/serve/mcp_tools_attention_direct.go:42-45`.
- [✓] AC-4: exact replay returns the original and changed payloads conflict — Mail store replay comparison is atomic at `internal/attention/mail/store.go:92-118`; Focus store replay comparison is atomic at `internal/attention/focus/store.go:87-121`; MCP tests assert replay IDs/revisions and `idempotency_conflict`.
- [✓] AC-5: malformed identity, version, pairing, lifecycle, provenance, and limits fail before mutation with field-specific errors — validators are in `contracts/attention/direct_action.go:62-125`; direct contract coverage is at `contracts/attention/contract_test.go:93-151`; service/MCP tests cover recipient, project, and thread guards.
- [✓] AC-6: non-user direct actions are permission-denied before mutation — all requests share the user-only guard at `contracts/attention/direct_action.go:173-183`; Focus denial and unchanged store state are asserted at `internal/serve/mcp_tools_focus_test.go:119-131`; the tool description routes model work to `hero_focus_suggest`.
- [✓] AC-7: MCP definitions publish canonical policy metadata and standard hints — definitions derive annotations and `_meta` from `OperationPolicy` at `internal/serve/mcp_tools_def.go:716-735`; `TestDirectAttentionDefinitionsUseCanonicalPolicyMetadata` asserts every field at `internal/serve/mcp_tools_attention_direct_test.go:16-40`.
- [✓] AC-8: profiles require explicit inclusion and filtered dispatch remains blocked — the dispatcher gate is at `internal/serve/mcp_dispatch.go:99-130`; profile inclusion/exclusion and filtered-call rejection are asserted at `internal/serve/mcp_tools_attention_direct_test.go:61-95`; the exact inventory is asserted at `internal/serve/mcp_test.go:388-475`.
- [✓] AC-9: unavailable authorities return structured `unavailable` results — Focus resolution distinguishes typed missing and unavailable errors at `internal/attention/focus/project.go:18-22,130-176` and maps them at `internal/serve/mcp_tools_focus.go:59-70`; Mail service and handler normalization are at `internal/attention/mail/service.go:86-210` and `internal/serve/mcp_tools_mail.go:73-96`. Focus and malformed-Mail authority tests assert `unavailable` at `internal/serve/mcp_tools_focus_test.go:174-186` and `internal/serve/mcp_tools_mail_test.go:209-220`.
- [✓] AC-10: Mail content remains inert structured data and direct handlers avoid storage/process boundaries — the end-to-end test stores a tool-shaped body byte-for-byte without dispatch at `internal/serve/mcp_tools_mail_test.go:125-170`; `TestDirectAttentionHandlersDoNotAccessStoreFilesOrProcesses` guards the three handler files at `internal/serve/mcp_tools_attention_direct_test.go:42-59`.

## Changes
- [✓] Add direct-action DTOs, validators, fixture types, and error codes — `contracts/attention/direct_action.go`; `contracts/attention/action.go:6-15`.
- [✓] Add schema/fixture, manifest entry, and advertised checksum — `contracts/attention/schema/v1/direct-action.schema.json`; `contracts/attention/testdata/v1/direct-actions.json`; `contracts/attention/testdata/v1/manifest.json`; checksum pin at `internal/serve/api_attention.go:10`.
- [✓] Add stable Mail identity guards and provenance — `internal/attention/mail/service.go:26-44,86-210`; service assertions at `internal/attention/mail/service_test.go:100-142`.
- [✓] Add registry-derived MCP annotations and metadata — `internal/serve/mcp_protocol.go:91-108`; `internal/serve/mcp_tools_def.go:716-735`.
- [✓] Implement and dispatch Mail send/reply through Mail service — `internal/serve/mcp_dispatch.go:73-77`; `internal/serve/mcp_tools_mail.go:13-96`.
- [✓] Implement and dispatch idempotent Focus create through registry resolution and `CreateOrGet` — `internal/serve/mcp_dispatch.go:73`; `internal/serve/mcp_tools_focus.go:41-91`; bounded origin-key derivation at `internal/serve/mcp_tools_attention_direct.go:42-45`.
- [✓] Add shared structured result/error marshaling — `internal/serve/mcp_tools_attention_direct.go:11-45`.
- [✓] Extend contract, service, inventory, profile, handler, parity, unavailable-authority, limit, and static boundary tests — `contracts/attention/contract_test.go:93-151`; `internal/serve/mcp_tools_attention_direct_test.go:16-95`; `internal/serve/mcp_tools_focus_test.go:79-186`; `internal/serve/mcp_tools_mail_test.go:118-220`; `internal/serve/mcp_test.go:388-475`.

## Audit notes
- None.
