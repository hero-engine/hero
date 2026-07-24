# Delivery audit — attention-contract-bundle-publication

**Audited:** `git diff HEAD -- <spec-scoped Attention publication paths>`, plus direct reads of the untracked generator, validator, generated bundle, schema, and MCP handler files
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — deterministic complete v1 conformance directory — `internal/attention/conformance/builder.go:53-167` assembles canonical schemas, fixtures, runtime MCP definitions, manifest, and handoff; `builder_test.go:12-38` compares repeated builds and checked-in bytes. The generated directory contains 37 files.
- [✓] AC-2 — complete sorted manifest metadata without self-hash — `internal/attention/conformance/builder.go:117-144` creates and sorts artifact records; `contracts/attention/conformance/v1/manifest.json` contains 35 unique sorted entries with path, kind, purpose, media type, SHA-256, and applicable schema, excluding the manifest and handoff.
- [✓] AC-3 — canonical additions, removals, and stale output fail validation — `internal/attention/conformance/builder.go:59-110` walks both canonical directories and rejects fixtures omitted from the source manifest; `builder.go:183-217` compares the exact generated file set and bytes.
- [✓] AC-4 — complete MCP inventory derives from the runtime registry — `internal/serve/mcp_tools_def.go:791-806` filters and sorts the same definitions returned by `tools/list`; `builder.go:112-130` generates and validates the inventory. `mcp-tools.json` contains 12 fully described Attention, Mail, and Focus tools including send, reply, create, snapshot, action, and contract.
- [✓] AC-5 — bundled routes resolve against canonical shape and policy — `internal/attention/conformance/builder.go:364-472` invokes `attention.ValidateConversationalRouteFixture` before checking runtime inventory and policy mappings; `contracts/attention/conversational_route.go:85-249` validates disposition, shape, trust, operation, surface, target resolution, effect, consent, and first/retry mutation counts.
- [✓] AC-6 — consumer validation rejects every named corruption class — `internal/attention/conformance/builder.go:220-288` validates exact checksums, JSON decoding, ordering, schemas, file-set completeness, and semantic route consistency; `builder_test.go:41-162` asserts missing, extra, checksum-mismatched, malformed, reordered, checksum-valid invalid-disposition, and checksum-valid route/inventory mismatch failures.
- [✓] AC-7 — HTTP and MCP advertise one identity additively — `internal/serve/api_attention.go:69-80` supplies shared fixture and bundle metadata; `mcp_tools_attention_contract.go:3-5` returns the same metadata; `api_attention_test.go:101-137` verifies both manifest hashes, path, and both surfaces.
- [✓] AC-8 — unknown additive identifiers decode without gaining execution — `contracts/attention/conformance/v1/fixtures/unknown-fields.json` carries future fields, source kind, action, effect, consent, and operation values; `builder_test.go:164-193` accepts a checksum-valid future tool but rejects a route that tries to execute its unknown operation.
- [✓] AC-9 — Hero Code receives an unambiguous whole-directory clean-pin protocol — `contracts/attention/conformance/v1/HERO-CODE-HANDOFF.md:42-65` names `fixtures/manifest.json` explicitly, then gives the root bundle manifest hash, validation order, versions, runtime parity, forward-compatibility rule, whole-directory vendoring, and external clean commit/release pin.
- [✓] AC-10 — readiness refuses dirty, stale, or mismatched publication state — `cmd/attention-conformance/main.go:16-26` covers canonical schemas/fixtures, generated output, generator, validator, runtime definitions, dispatch, MCP handler, and HTTP advertisement; `main.go:46-80` requires generated-byte parity and clean scoped Git status. `main_test.go:17-31` guards the expanded publication scope.

## Changes

- [✓] Define bundle manifest schema and deterministic ordering — `contracts/attention/schema/v1/conformance-bundle-manifest.schema.json` and `internal/attention/conformance/builder.go:132-144`.
- [✓] Add deterministic builder and generated distribution — `cmd/attention-conformance/main.go`, `internal/attention/conformance/builder.go`, and the checked-in 37-file `contracts/attention/conformance/v1/` directory.
- [✓] Generate MCP inventory from runtime definitions — `internal/serve/mcp_tools_def.go:791-806`, `builder.go:112-130`, and generated `mcp-tools.json`.
- [✓] Bundle and validate the route corpus — `internal/attention/conformance/builder.go:124-130,364-472` includes canonical route validation plus operation/tool/action/effect/consent/mutation parity.
- [✓] Add completeness, checksum, and orphan detection — `internal/attention/conformance/builder.go:59-110,183-288` and `builder_test.go:12-162`.
- [✓] Extend HTTP and MCP contract advertisement — `internal/serve/api_attention.go:69-80`, `internal/serve/mcp_tools_attention_contract.go`, `internal/serve/mcp_dispatch.go:25-94`, and `api_attention_test.go:101-137`.
- [✓] Publish generated Hero Code handoff — `internal/attention/conformance/builder.go:146-165` and generated `HERO-CODE-HANDOFF.md`.
- [✓] Add consumer-only compatibility validation — `internal/attention/conformance/builder_test.go:41-193` validates copied bundles and exercises both checksum-valid semantic inconsistency and decodable-but-inert unknown identifiers.
