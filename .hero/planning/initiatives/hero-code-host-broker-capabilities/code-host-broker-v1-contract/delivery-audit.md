# Delivery audit — code-host-broker-v1-contract

**Audited:** `git diff 1e3fa83...fce0a50`
**Verdict:** HOLD
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1: exactly twenty authoritative operation policies.
- [✓] AC-2: repository-qualified PR identity.
- [✓] AC-3: distinct fork base/head identities, refs, and SHAs.
- [✗] AC-4: `Response.Result` was raw JSON without operation-specific schema validation.
- [✓] AC-5: partial result and section failures coexist.
- [✗] AC-6: cursor fingerprints were not tied to returned/requested wire cursors.
- [✓] AC-7: mutation policy material is required.
- [✓] AC-8: read/write/merge effect and consent classes are authoritative.
- [✗] AC-9: successful mutations could omit receipt/reconciliation and error retry mappings were not enforced.
- [✓] AC-10: closed error/retry inventories are published.
- [✗] AC-11: no unknown advertised capability was fixture-tested.
- [✗] AC-12: several published bounds were advertised but not enforced.
- [✓] AC-13: fixture and digest are byte-stable.
- [✗] AC-14: fixture mutation fields contained example text rather than redacted placeholders.

## Changes

- [✗] Canonical result validation and some bounds were incomplete.
- [✓] Single operation-policy registry.
- [✗] Cursor wire binding was incomplete.
- [✓] Mutation receipt and reconciliation types.
- [✗] DTO/nullability documentation was incomplete.
- [✗] Fixture state and redaction coverage was incomplete.
- [✗] Strict independent decoding, bounds, and meaningful fuzz coverage were incomplete.

## Required remediation

1. Strictly validate operation-specific result schemas and all DTO bounds.
2. Encode and validate opaque cursor envelopes against scope and query material.
3. Require successful mutation receipts/reconciliation and exact retry mappings.
4. Add unknown advertised capability and complete availability/completeness/page fixtures.
5. Enforce repository, item, diff, redirect, journal, rate-limit, and cursor bounds.
6. Replace fixture mutation text with explicit redacted sentinels.
7. Expand the field/nullability contract, independent decoder, boundary, and cursor fuzz tests.
