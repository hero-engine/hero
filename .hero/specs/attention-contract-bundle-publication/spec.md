---
title: "Attention Contract Bundle Publication — Immutable Cross-Repo Conformance Pin"
slug: attention-contract-bundle-publication
type: feature
status: completed
domain: engineering
priority: critical
size: medium
horizon: now
created: 2026-07-23
parent: conversational-attention-operability
depends-on:
  - attention-interaction-consent-contract
  - attention-mcp-action-tools
  - attention-lifecycle-read-awareness
  - attention-conversational-routes
tags: [attention, contracts, conformance, fixtures, publication, hero-code]
relations:
  - target: durable-attention-contracts
    kind: related
  - target: attention-read-model-v1
    kind: related
delivery_method: manual
completed_at: 2026-07-24T01:53:07Z
---

# Attention Contract Bundle Publication — Immutable Cross-Repo Conformance Pin

## Context

Hero now has a stable Attention v1 contract, interaction policy, direct Mail
and Focus MCP actions, typed results and errors, and bounded lifecycle read
semantics. The final conversational-routes child will add the complete phrase
corpus. Hero Code has already designed its native chat-loop consumer, but it
correctly refuses to implement against an uncommitted Hero checkout or a
partial fixture directory.

The current `contracts/attention/testdata/v1/manifest.json` checksums fixture
payloads only. It names schemas without checksumming or packaging them, does not
publish the model-facing MCP tool inventory and input schemas, and cannot prove
that every contract artifact or conversational case is present. The HTTP
contract endpoint advertises only that fixture-manifest checksum. This is
enough for the existing Durable Attention consumer but not enough to pin a
cross-repo conversational integration safely.

Hero needs one immutable, complete, machine-verifiable conformance bundle that
Hero Code can vendor as a unit. The bundle identifies its own manifest by
SHA-256; the consuming repository separately pins the clean Hero commit or
release containing that manifest. The manifest must not attempt to include its
own Git commit hash.

## Goal

Publish one deterministic Attention v1 conformance bundle containing every
schema, fixture, conversational route case, and model-facing MCP tool contract
needed by Hero Code, advertise its identity from Hero, and provide exact
consumer pinning instructions that do not depend on uncommitted paths or
implementation internals.

## Kickoff

Assemble the completed Attention contracts and conversational routes into one
immutable bundle Hero Code can vendor, checksum, decode, and pin.

**Status:** in review — deterministic bundle generation, runtime inventory,
consumer validation, cross-surface identity, and readiness gating are complete.

**Pick up at:** cold-audit the Completion Ledger, address any HOLD findings,
then run the delivery verification gate.

→ `/deliver attention-contract-bundle-publication`

**Files:** `contracts/attention/conformance/v1/`,
`contracts/attention/schema/v1/`, `contracts/attention/testdata/v1/`,
`internal/attention/conformance/builder.go`,
`cmd/attention-conformance/`, `internal/serve/mcp_tools_def.go`,
`internal/serve/api_attention.go`, `internal/serve/mcp_tools_attention_contract.go`
**Skip:** embedding a self-referential commit SHA, provisional Swift DTOs,
changing Attention storage, or creating another transport authority.

## Design

### One vendorable distribution directory

Publish a generated directory at
`contracts/attention/conformance/v1/` with this logical shape:

```text
conformance/v1/
  manifest.json
  schemas/
    *.schema.json
  fixtures/
    *.json
  HERO-CODE-HANDOFF.md
```

The canonical authoring sources remain `contracts/attention/schema/v1/`,
`contracts/attention/testdata/v1/`, the canonical interaction and route
registries, and the real MCP tool definitions. The distribution directory is
generated from those sources and must not become a second hand-edited contract.
Generation is deterministic and repository tests fail when checked-in output
is stale.

The bundle manifest has its own version and the Attention schema major version.
Every artifact entry includes:

- a bundle-relative path;
- an artifact kind such as `schema`, `fixture`, `route-corpus`, or
  `mcp-tool-inventory`;
- its media type and purpose;
- an exact SHA-256; and
- a schema reference when the artifact is JSON governed by one.

The manifest does not checksum itself. Consumers checksum the exact
`manifest.json` bytes and compare that value to Hero's advertised
`bundle_manifest_sha256`.

### Complete inventory, not selected examples

The generated bundle includes every published v1 JSON Schema and every
canonical fixture, including success, error, unknown-additive, empty,
unavailable, compact-window, interaction-policy, direct-action, suggestion,
promotion, and launch shapes.

It also includes:

1. a deterministic Attention MCP tool inventory generated from Hero's actual
   registered tool definitions, containing stable tool names, descriptions,
   complete JSON input schemas, and effect/permission annotations; and
2. the complete conversational route corpus from
   `attention-conversational-routes`, containing positive, ambiguous, negative,
   adversarial, unavailable, stale, and retry cases with expected disposition,
   operation, tool/action family, effect, consent class, required missing fact,
   and mutation count.

An inventory test walks the canonical schema and fixture directories and fails
for any eligible orphan or omitted file. A runtime parity test fails if the
published MCP inventory differs from the tools actually registered by Hero.

### Additive publication over the existing v1 endpoint

Keep the existing fixture manifest and
`fixture_manifest_sha256` advertisement for current consumers. Extend
`GET /api/attention/v1/contract` additively with:

- `bundle_version`;
- `bundle_manifest_sha256`;
- a stable repository-relative bundle path or release artifact name;
- the existing Attention `schema_version`; and
- compatibility language stating that unknown additive fields and open
  identifiers remain inert but decodable.

Expose the same bundle identity through the MCP contract/discovery surface used
by model clients so HTTP and MCP cannot advertise different contract
revisions. The endpoint advertises identity and compatibility metadata, not
fixture bodies.

The compiled advertised checksum is generated from the checked-in manifest or
otherwise parity-checked byte-for-byte. A stale constant must fail tests.

### Clean pin and handoff protocol

`HERO-CODE-HANDOFF.md` names:

- the exact bundle manifest SHA-256;
- how to validate every listed artifact before decoding;
- the Attention schema and bundle versions;
- the required compatibility behavior for unknown additive values;
- the runtime endpoint/tool inventory parity expectation; and
- the consumer workflow: vendor the entire directory, then record the clean
  Hero commit or release that contains the advertised manifest.

The bundle must not embed its containing Git commit because that would make the
commit self-referential. After Hero commits or releases the generated bundle,
Hero Code pins that external revision plus the already-published manifest hash
in its own compatibility suite.

Publication is not considered ready while the working tree contains
uncommitted contract changes, while route or tool parity checks fail, or while
the advertised bundle checksum differs from the generated manifest.

### Hero Code compatibility seam

Hero Code remains responsible for copying the published directory into its
Swift test fixtures and decoding it. Hero's side supplies a small consumer
compatibility check, independent of Hero internals, that:

- validates the manifest and all artifact hashes;
- decodes each JSON artifact using only the published shape;
- verifies unknown additive values do not gain behavior; and
- proves the route corpus references only tools and actions present in the
  published inventories.

No test imports Swift code or reaches into the sibling working tree. The exact
bundle directory and handoff are the stable cross-repo boundary.

## Changes

1. Define the versioned conformance bundle manifest schema and deterministic
   artifact ordering.
2. Add a deterministic builder that assembles generated schemas, fixtures,
   route corpus, MCP inventory, and handoff into
   `contracts/attention/conformance/v1/`.
3. Generate the Attention MCP tool inventory from the real registered tool
   definitions and add runtime parity tests.
4. Include the complete conversational route corpus and validate every route's
   operation, tool/action, consent, effect, and mutation expectation against
   the published registries.
5. Add completeness and checksum tests covering every bundle artifact plus
   orphan detection for canonical v1 schemas and fixtures.
6. Extend the HTTP and MCP contract advertisement additively with bundle
   version, exact manifest SHA-256, location, and compatibility metadata while
   retaining the fixture-manifest fields.
7. Publish a generated Hero Code handoff with whole-directory vendoring,
   validation, clean-revision pinning, and forward-compatibility instructions.
8. Add a consumer-style compatibility test that uses only the bundle and fails
   for missing, extra, stale, malformed, or internally inconsistent artifacts.

## Acceptance Criteria

- **AC-1:** WHEN the Attention conformance bundle is generated THE SYSTEM SHALL
  produce one deterministic `contracts/attention/conformance/v1/` directory
  containing its manifest, all published schemas, all required fixtures, the
  route corpus, the MCP tool inventory, and the Hero Code handoff.
- **AC-2:** WHEN `manifest.json` is read THE SYSTEM SHALL identify every
  machine-consumable bundle artifact by relative path, kind, purpose, media
  type, exact SHA-256, and applicable schema without listing the manifest
  itself as a hashed child.
- **AC-3:** WHEN canonical v1 schema or fixture files are added, removed, or
  changed THE SYSTEM SHALL fail completeness or staleness validation until the
  generated bundle and manifest exactly match the canonical inventory.
- **AC-4:** WHEN the published MCP inventory is generated THE SYSTEM SHALL
  contain `hero_mail_send`, `hero_mail_reply`, `hero_focus_create`,
  `hero_attention_snapshot`, and every Attention tool referenced by the route
  corpus with the same complete input schemas and annotations as Hero's runtime
  registry.
- **AC-5:** WHEN a conversational route case is bundled THE SYSTEM SHALL
  resolve its operation, tool/action family, effect, consent, target
  disposition, and expected mutation count against the canonical interaction
  and MCP registries; invalid or missing references SHALL fail generation.
- **AC-6:** WHEN a consumer validates the bundle THE SYSTEM SHALL reject a
  missing, extra, reordered-without-regeneration, malformed, or
  checksum-mismatched artifact before decoding compatibility assertions.
- **AC-7:** WHEN `GET /api/attention/v1/contract` or the MCP contract surface
  advertises Attention THE SYSTEM SHALL return the same bundle version and
  exact manifest SHA-256 while preserving the existing schema and fixture
  manifest fields additively.
- **AC-8:** WHEN unknown additive fields, operation IDs, effect values, action
  IDs, or error values appear in forward-compatibility fixtures THE SYSTEM
  SHALL require consumers to preserve or ignore them safely and SHALL NOT grant
  them executable behavior.
- **AC-9:** WHEN Hero Code follows the published handoff THE SYSTEM SHALL let it
  vendor the complete directory, validate all hashes, and record the clean Hero
  commit or release containing the advertised manifest without relying on an
  absolute path or uncommitted checkout.
- **AC-10:** IF Hero has uncommitted contract output, a stale generated bundle,
  an advertisement checksum mismatch, or a route/tool inventory mismatch THEN
  THE SYSTEM SHALL NOT claim the bundle is ready for Hero Code delivery.

## Boundaries

- No self-referential Git commit hash inside the bundle.
- No new Attention schema major version or breaking rename.
- No replacement of Hero Serve HTTP, MCP discovery, or the existing projection
  authority.
- No Swift DTOs, native cards, permission UI, or Hero Code implementation.
- No duplicate hand-authored schemas, fixtures, route cases, or MCP tool
  definitions in the distribution directory.
- No network fetch requirement at runtime; the bundle is a release/test
  artifact and the endpoint advertises only its identity.
- No implementation against a sibling repository's uncommitted state.

## Risks

- **Generated-copy drift:** a vendorable directory necessarily duplicates
  bytes from canonical sources. Deterministic regeneration and exact parity
  tests must make those copies output, never authority.
- **Incomplete inventory:** a manifest can be internally valid while omitting
  a new artifact. Canonical directory walks and runtime tool parity close that
  gap.
- **Self-reference:** embedding the containing commit SHA makes reproducible
  publication impossible. The consumer pins the revision externally.
- **Runtime/fixture drift:** MCP schemas can change without fixture updates.
  Generation from, or strict parity with, the real registry is required.
- **Compatibility overreach:** unknown additive values must decode without
  silently enabling UI or actions. The bundle includes inert forward cases.
- **Dirty publication:** a correct local manifest is not a stable cross-repo
  pin until committed or released. The handoff and readiness gate say so
  explicitly.

## Validation

- `go test ./contracts/attention ./internal/serve`
- `go test ./...`
- Regenerate the bundle twice and compare all output bytes for determinism.
- Walk canonical schema and fixture directories and prove no eligible file is
  omitted or orphaned.
- Hash every listed artifact and compare it with the manifest.
- Compare the generated MCP inventory with the real runtime tool registry.
- Validate every route corpus reference against the interaction policy,
  advertised row actions, and MCP inventory.
- Exercise HTTP and MCP contract advertisements and compare bundle identity.
- Run the consumer-style compatibility check using only a copied bundle
  directory.
- After commit or release, verify a clean checkout reproduces the same manifest
  SHA before handing the pin to Hero Code.

## Completion Ledger

Hero now generates one vendorable Attention v1 conformance directory from the
canonical schema/fixture trees and the MCP definitions actually returned by
`tools/list`. The checked-in bundle contains 37 files: its manifest, 35 hashed
machine-consumable artifacts, and a generated Hero Code handoff. Its current
manifest SHA-256 is
`bf5dc3524809dfdaf87935bbcfb28c0751f0493da7ed61eabb2fda3561598da5`.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Generate one deterministic complete v1 conformance directory | DONE | `go run ./cmd/attention-conformance` produces `contracts/attention/conformance/v1/` with the manifest, every schema and fixture, route corpus, 12-tool MCP inventory, and generated handoff; repeated-build byte comparison passes. |
| 2 | Manifest every machine-consumable artifact with identity metadata | DONE | The sorted 35-entry `manifest.json` records path, kind, purpose, media type, SHA-256, and applicable schema, while deliberately excluding itself and the human handoff from hashed children. |
| 3 | Detect canonical schema/fixture additions, removals, and stale output | DONE | The builder walks both canonical v1 directories, rejects fixtures omitted from the source fixture manifest, and `conformance.Check` compares every generated byte and file set against checked-in output. |
| 4 | Generate complete Attention MCP inventory from the real runtime registry | DONE | `AttentionToolDefinitions` filters and sorts the actual `tools/list` definitions; the bundle contains 12 Attention/Mail/Focus tools, including send, reply, create, snapshot, action, and contract, with full input schemas, annotations, and metadata. |
| 5 | Resolve bundled route operation/tool/action/effect/consent/mutation expectations | DONE | Generation invokes the complete canonical route validator before checking bundled interaction policies and the generated MCP inventory, covering dispositions, shape, trust, effect, consent, target resolution, and exact first-dispatch/retry mutation rules. |
| 6 | Reject missing, extra, stale, malformed, reordered, or inconsistent artifacts | DONE | `TestConsumerValidationRejectsMissingExtraStaleAndMalformedArtifacts` exercises a copied bundle and rejects every named corruption class, including checksum-valid invalid dispositions and route/inventory mismatches. |
| 7 | Advertise one bundle identity from HTTP and MCP while preserving fixture fields | DONE | HTTP `/api/attention/v1/contract` and MCP `hero_attention_contract` share the same metadata function and exact bundle hash/path/version; parity tests also retain and verify the existing fixture-manifest hash. |
| 8 | Keep unknown additive values inert but decodable | DONE | A copied-bundle test adds a checksum-valid future MCP tool and proves it remains decodable; a route attempting to execute its unknown operation is rejected by the stable v1 route contract. |
| 9 | Give Hero Code a whole-directory clean-pin protocol | DONE | Generated `HERO-CODE-HANDOFF.md` names the exact manifest hash, validation order, schema/bundle versions, runtime parity, forward compatibility, whole-directory vendoring, and external clean commit/release pin. |
| 10 | Refuse readiness for dirty, stale, or mismatched publication state | DONE | `--check` rejects stale output; checksum/runtime/route parity tests reject mismatches; `--release-ready` covers schemas, fixtures, output, generator, conformance validator, runtime registry, dispatch, handler, and advertisement sources, and currently refuses readiness because this delivery is uncommitted. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Define bundle manifest schema and deterministic ordering | DONE | Added `conformance-bundle-manifest.schema.json` and sorted manifest generation in `internal/attention/conformance/builder.go`. |
| 2 | Add deterministic builder and generated distribution | DONE | Added `cmd/attention-conformance`, in-memory build/write/check APIs, and the 37-file checked-in `conformance/v1` output. |
| 3 | Generate MCP inventory from runtime definitions | DONE | Added `AttentionToolDefinitions`, runtime-wide Attention annotations, `mcp-tool-inventory.schema.json`, and exact generated inventory parity. |
| 4 | Bundle and validate the route corpus | DONE | The builder includes `conversational-routes.json` and validates its operation/tool/action/effect/consent/mutation mappings before generating a manifest. |
| 5 | Add completeness, checksum, and orphan detection | DONE | Canonical directory walks, source-manifest membership, strict sorted paths, exact hashes, exact file-set validation, and stale-output comparison are all enforced. |
| 6 | Extend HTTP and MCP contract advertisement | DONE | Added bundle identity fields to HTTP plus the read-only `hero_attention_contract` MCP tool backed by the same metadata. |
| 7 | Publish generated Hero Code handoff | DONE | The bundle handoff is deterministically derived from the canonical v1 handoff and augmented with the exact bundle hash and clean-pin protocol. |
| 8 | Add consumer-only compatibility validation | DONE | Tests validate a copied bundle using only its manifest/artifacts and exercise missing, extra, checksum, malformed, reordered, checksum-valid semantic inconsistency, and decodable-but-inert unknown identifier behavior. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end:
  `go run ./cmd/attention-conformance --check` reproduced the exact checked-in
  bundle and manifest hash; HTTP and MCP returned the same identity;
  copied-bundle corruption tests all passed; `--release-ready` correctly
  refused the current uncommitted output; and `go test ./...` passed.

### Excellence Bar self-check

- [x] Yes — authority remains in canonical sources/runtime definitions,
  generated copies are deterministic and exhaustively checked, consumers can
  validate without Hero internals, and publication cannot honestly claim
  readiness before a clean external revision exists.
