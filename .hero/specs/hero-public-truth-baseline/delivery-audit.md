# Delivery audit — hero-public-truth-baseline

**Audited:** `git diff --cached 75ea3cb1 -- .hero/planning/initiatives/hero-marketing/hero-public-truth-baseline internal/config/public_example_test.go internal/config/testdata/public-hero.json`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: Inventory every public claim with owner and source location — `public-claim-registry.yaml:13` contains 34 claim records with surfaces and owners; `public-surface-inventory.md:7` maps five root guides, all 35 hosted Markdown pages, and landing/exposure surfaces. The provided source-coverage validation mapped 35/35 hosted Markdown pages.
- [✓] AC-2: Shipped claims carry executable or delivered evidence sufficient to reproduce them — shipped registry rows cite Class A implementation/runtime authorities or Class B delivered evidence. The unresolved two-system outcome and Sprout consumed-version boundary are no longer classified as shipped (`public-claim-registry.yaml:47`, `public-claim-registry.yaml:458`).
- [✓] AC-3: Optional capabilities state prerequisites — `public-claim-registry.yaml:334`, `public-claim-registry.yaml:349`, `public-claim-registry.yaml:365`, and `public-claim-registry.yaml:411` name provider, credentials, repository registration, consent, or composition prerequisites for code-host, tracker, peering, and domain-pack claims.
- [✓] AC-4: Unresolved or planning-only claims are bounded — the two-system outcome and Sprout consumed-version boundary are `preview` with Class D evidence and bounded prerequisites (`public-claim-registry.yaml:47`, `public-claim-registry.yaml:458`); external providers and Apache status are planned/prohibited, while deployment, DNS, and headless-runtime claims carry preview boundaries.
- [✓] AC-5: Provide authoritative replacements and executable validation for every P0 audit row — `p0-correction-packet.md:7` through `p0-correction-packet.md:211` cover all seven P0 rows; the satellite/topology selector names and passed real `internal/install` tests (`p0-correction-packet.md:35`, `p0-correction-packet.md:92`), and `internal/config/public_example_test.go:9` asserts the canonical fixture through `config.Load`.
- [✓] AC-6: Every registered claim has one owner and resolution state — all 34 registry records contain singular `owner`, allowed `resolution`, and `last_verified` fields; the provided Ruby schema validation parsed every row without a required-field or enum failure.

## Changes

- [✓] Materialize an exhaustive claim registry — `public-claim-registry.yaml:1` defines the evidence, availability, and resolution schema for 34 claims; `public-surface-inventory.md:7` maps public sources and downstream owners.
- [✓] Resolve the seven-row P0 correction packet — `p0-correction-packet.md:7` through `p0-correction-packet.md:211` provide affected locations, exact replacements, owners, observed stale behavior, and executable checks for every P0 row.
- [✓] Derive command, agent, skill, MCP, and target inventories — `p0-correction-packet.md:213` records revision-scoped totals derived from install/runtime authorities; `internal/install/inventory_test.go:53` exercises all seven targets and `internal/serve/mcp_test.go:389` derives the MCP total from the runtime registries.
- [✓] Classify capability maturity and prerequisites — the registry covers the named capability families, and unresolved/planning evidence is bounded as preview or planned rather than shipped.
- [✓] Record the repository licensing boundary — `public-claim-registry.yaml:443` prohibits the Apache claim pending its grant gate; `public-claim-registry.yaml:458` bounds the unverified Sprout consumed-version claim; and `public-claim-registry.yaml:472` records the proprietary repository exclusion.

## Audit notes

- None. The audited diff is scoped to the spec artifacts and decoder-backed fixture.
