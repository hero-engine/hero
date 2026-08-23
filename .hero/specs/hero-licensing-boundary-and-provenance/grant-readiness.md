# Apache-2.0 Grant Readiness

Audit date: 2026-08-23

## Result

**Hero-owned content authority: clear.** The sole owner authorized preparing this repository for Apache-2.0.

**CLI third-party compatibility: clear with notice work.** No incompatible release-binary dependency or bundled model asset was found. The exact Sprout version now referenced by Hero contains its MIT license.

**Current generated docs output: blocked.** Material copies unused MPL-1.1 Lunr Languages files and an LGPL-3.0 Wordcut bundle into the English site. The hosted-docs remediation must remove them from the deployed output or satisfy their reciprocal obligations; they are not cleared as notices-only assets.

**Final grant gate: not yet armed.** The following mechanical preconditions belong to later initiative children and must be complete before the license is added or a release is published.

## Required before adding the grant

1. Add the canonical Apache License 2.0 text as root `LICENSE` only after the explicit license approval gate.
2. Add a distributable third-party notices artifact covering the release-binary dependencies and the embedded Minish Lab model. Include it in every GoReleaser archive.
3. Ensure the root documentation states that third-party components retain their own licenses and that Hero Code and Hero Cloud are proprietary products outside this repository's grant.
4. Keep Sprout identified as a separate MIT project; do not imply it is Apache-2.0 through Hero.
5. Confirm `go-licenses` has no unresolved external package in the release closure. `modernc.org/mathutil` must remain the documented manual BSD-3-Clause resolution unless the scanner recognizes it.

## Required before hosted-docs deployment

1. Bound or lock `requirements-docs.txt`; the current `>=` ranges can change the dependency and license set without a repository diff.
2. Rebuild the site from that bounded environment and remove the unreferenced MPL-1.1 Lunr Languages files and LGPL-3.0 Wordcut bundle from the English output, or satisfy their reciprocal obligations in full. Do not classify them as notices-only dependencies.
3. Publish a third-party attribution page for Material for MkDocs, Lunr, TinySegmenter, Font Awesome's CC BY icon, Pictogrammers icons, and the externally loaded fonts. If the reciprocal search bundles remain, publish their required licenses and corresponding-source path as well.
4. Remove current public-copy references to Hero being MIT-licensed. The target license is Apache-2.0, and the claim must not appear before the explicit grant gate succeeds.

## Required before v0.34 release

1. Regenerate license reports for the actual Darwin, Linux, and Windows release targets.
2. Verify GoReleaser archives contain `LICENSE` and the third-party notices artifact after those files are approved and added.
3. Verify Homebrew and Scoop metadata say `Apache-2.0`. This repository's release configuration has already been corrected from `MIT` to `Apache-2.0` during this audit.
4. Produce a snapshot build and inspect its file inventory; do not publish a tag or GitHub release during preparation.

## Fail-closed rule

The final license/publication gate fails if any redistributed component has an Unknown, incompatible, unfulfilled reciprocal, or unverified license; if the notice artifact is missing from binary archives; if generated site dependencies are unbounded; or if public copy collapses Hero, Sprout, Hero Code, and Hero Cloud into one license boundary. The current generated docs output is blocked by the copied MPL-1.1/LGPL-3.0 search assets until they are removed or their obligations are fulfilled.
