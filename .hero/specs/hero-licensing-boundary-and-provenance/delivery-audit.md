# Delivery audit — hero-licensing-boundary-and-provenance

**Audited:** `git diff --cached` (index versus `HEAD`)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1: identify every included and excluded repository/material category — `repository-boundary.md:13-37` defines the tracked Hero grant, third-party treatment, local-state exclusion, separate Sprout boundary, and proprietary Hero Code/Hero Cloud exclusions.
- [✓] AC-2: record source, license, compatibility, and obligations for redistributed dependencies/assets — `third-party-inventory.md:9-97` reconciles the cross-platform release closure, embedded model, hosted-doc build and generated bundles, external fonts/icons, and tracked assets; `internal/embeddings/defaultmodel/SOURCE.md:3-23` records the model revision, lineage, transformation, notices, and exact hashes.
- [✓] AC-3: record sole-owner authorization without contributor-rights reopening — `repository-boundary.md:7-9` records the owner's preparation authority, expressly avoids contributor-by-contributor review, and leaves the grant/publication mutations separately gated.
- [✓] AC-4: fail closed on unresolved provenance or compatibility — `grant-readiness.md:11-39` blocks the current generated docs output on MPL-1.1/LGPL-3.0 bundles and blocks the final gate on unknown, incompatible, unfulfilled-reciprocal, or unverified licenses, missing notices, unbounded docs dependencies, and blurred product boundaries.
- [✓] AC-5: consume a Sprout archive containing MIT while keeping Code/Cloud proprietary — `go.mod:7` and `go.sum:3-4` select immutable Sprout commit `cd3f0c4a2208`; its exact module ZIP contains the MIT `LICENSE`; `repository-boundary.md:32-34` excludes Hero Code and Hero Cloud and keeps Sprout separate.

## Changes

- [✓] Repository-boundary report — `repository-boundary.md:13-41` defines the exact repository/product rule and explicit exclusions.
- [✓] Exact Sprout licensing — `go.mod:7`, `go.sum:3-4`, and the inspected module ZIP identify the licensed artifact; comparing both ZIPs after stripping module-version prefixes found the new `LICENSE` as the only content difference.
- [✓] Dependency and asset inventory — `third-party-inventory.md:9-97` includes Darwin/Linux/Windows binary dependencies, Go runtime, model lineage, docs packages, generated search/icon/font assets, and visual/vendored-file reconciliation.
- [✓] Owner authority — `repository-boundary.md:7-9` records preparation authority without contributor-theater or premature grant/publication approval.
- [✓] Grant/notice preconditions — `grant-readiness.md:15-39` provides separate grant, hosted-doc, and release preconditions plus a fail-closed rule; `.goreleaser.yaml:64,82` corrects future Homebrew/Scoop metadata while publication remains gated.

## Open items

- Current generated documentation output — BLOCKED — Material 9.7.7 copies unused MPL-1.1 Lunr Languages files and an LGPL-3.0 Wordcut bundle into the English site — concrete: remove them from bounded regenerated output or fulfill the reciprocal obligations before deployment (`grant-readiness.md:23-28`).
- Final Apache grant/publication — BLOCKED — root Apache license, distributable third-party notices, archive verification, bounded docs dependencies, and final approval are intentionally deferred to later gated work — concrete (`grant-readiness.md:13-35`).

## Audit notes

- All five DONE acceptance rows and all five DONE change rows have staged implementation evidence; no performative row was found.
- The embedded model hashes reproduce from pinned Hugging Face revision `bf8b056651a2c21b8d2565580b8569da283cab23`; the packet also records the MIT BAAI/FlagEmbedding base-model lineage and captures both MIT notices.
- Cross-target `go list -deps` reconciliation found the Windows-only `inconshreveable/mousetrap` dependency, and the final staged inventory includes it.
- Validation passed: `go test ./...`, `go mod verify`, targeted mock-tracker tests, `goreleaser check`, `git diff --cached --check`, and Darwin/Linux/Windows dependency-closure comparison.
- The packet is shippable as a readiness artifact. It does not authorize adding the root Apache license, publishing a release, changing repository visibility, or deploying the currently blocked generated docs output.
