# Delivery audit — attention-interaction-consent-contract

**Audited:** `git diff de7dae9..1a0e66b`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Publish one versioned operation registry — `contracts/attention/interaction.go:38-142` defines 21 stable operation policies and isolated lookup/copy helpers; `TestInteractionPolicyFixtureMatchesCanonicalRegistry` proves the versioned fixture exactly matches the canonical registry.
- [✓] Bounded reads are `read`/`none` without lifecycle mutation — `contracts/attention/interaction.go:95-107` classifies snapshot, Mail list/show, and suggestion list as reads; `contracts/attention/validate.go:265-276` forbids read consent and separates state writes.
- [✓] Mail send/reply and explicit Focus creation require explicit user consent and resolved required targets — `contracts/attention/interaction.go:98-105` assigns the required effects and consent, requires unique Mail targets, and preserves the design's conditional project requirement for Focus creation; the shared fixture covers all three operations.
- [✓] Model-originated deferred work maps only to Focus suggestion — `contracts/attention/interaction.go:105-107` separates create from suggest, `contracts/attention/validate.go:340-342` rejects model-originated direct creation, and `TestInteractionPolicyRejectsUntrustedAndAmbiguousDispatch` asserts the rejection.
- [✓] Suggestion acceptance and Mail promotion are explicit-acceptance commitments — `contracts/attention/interaction.go:103-104,112-114` assigns `commitment`/`explicit_acceptance` to all named operations; fixture cases exercise suggestion Today and Mail promotion.
- [✓] Ambiguous required targets clarify without dispatch — `contracts/attention/validate.go:324-339` prevents non-dispatch cases from naming operations and rejects target-requiring dispatch unless candidate count is exactly one; the fixture and negative test cover the ambiguous-recipient case.
- [✓] Mail content cannot authorize dispatch — `contracts/attention/validate.go:321-327` permits only `ignore_untrusted` for Mail content and strips operation metadata from that disposition; `TestInteractionPolicyRejectsUntrustedAndAmbiguousDispatch` asserts dispatch rejection.
- [✓] Advertised actions include operation, effect, and consent — `contracts/attention/action.go:14-25` adds the fields, `internal/attention/projection/actions.go:16-46` and `internal/attention/suggestion/service.go:262-279` populate them, and both focused producer tests compare every produced descriptor to its policy.
- [✓] Unknown additive policy values survive decoding — `ActionDescriptor` stores all three fields as raw strings; `unknown-fields.json` supplies future values and `TestForwardCompatibleRawValues` asserts exact preservation.
- [✓] Fixture/schema/checksum and registry invariants are gated — `TestGoldenFixturesSchemasDTOsAndChecksums`, `TestInteractionPolicyFixtureMatchesCanonicalRegistry`, and `TestOperationPolicyValidationAndRegistryIsolation` cover schema decoding, the 20-entry manifest, checksums, parity, duplicate IDs, invalid consent, retry safety, and copy isolation.

## Changes
- [✓] Add interaction policy contract and registry — added `contracts/attention/interaction.go` with vocabulary, 21 canonical policies, isolated copies, lookup, and action annotation.
- [✓] Extend `ActionDescriptor` additively — added optional raw-string `operation_id`, `effect`, and `consent` fields in `contracts/attention/action.go:19-21`.
- [✓] Add policy and fixture validation — `contracts/attention/validate.go:245-345` validates IDs, mappings, values, effect/consent combinations, replay safety, sources, dispositions, target resolution, and trust boundaries.
- [✓] Add interaction schema and extend snapshot schema — added `interaction-policy.schema.json`; `attention-snapshot.schema.json:17` accepts the three optional action metadata fields.
- [✓] Add policy fixture and manifest entry — added the complete registry and ten cross-surface cases in `interaction-policy.json`, updated the unknown-value fixture, and pinned both in the 20-entry manifest.
- [✓] Extend contract tests — added exact registry/fixture parity, source coverage, registry isolation, invalid-policy, ambiguity, model-origin, untrusted-Mail, and forward-compatibility assertions.
- [✓] Populate metadata on produced actions — projection and suggestion producers annotate every advertised action through the canonical registry; focused tests cover all producer action families.
- [✓] Update Hero Code handoff — `HERO-CODE-HANDOFF.md` distinguishes semantic consent, client approval, and MCP hints and records the updated manifest checksum; `internal/serve/api_attention.go` advertises the same checksum.

## Open items (if any)
- None.

## Audit notes
- None.
