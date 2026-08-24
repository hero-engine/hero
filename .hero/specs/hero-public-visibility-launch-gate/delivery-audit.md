# Delivery audit — hero-public-visibility-launch-gate

**Audited:** `git diff 2f58af9f2eb9adf709c04cba4ad498094ddb4153` at `34e4e4fce57d90ecc5cb1408b16d6b0d5fed9ed0`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] Leave visibility and release unchanged without explicit approval — the superseded pre-rewrite gate record shows anonymous source/release state remained unavailable and publication halted when the history scan found 55 blocking paths; the final launch record documents the later bounded owner approval (`spec.md:66-71`, `spec.md:93-113`).
- [✓] Halt unless DNS, deployment revision, Apache licensing, inventory, exposure scan, and candidate validation are green — the failed history scan exercised the halt path; the supplied delivery evidence records the final full current/history scan exit 0, production revision checks, Apache gate, test suites, and candidate checksum validation. Candidate provenance binds revision `34e4e4fc...`, tree `c047c329...`, five targets, and `publication_status: unpublished` (`.build/release-candidate/v0.34.0/provenance.json`).
- [✓] Expose only Hero and publish the sanitized v0.34 revision and five-target production artifacts — annotated tag `v0.34.0` points to `34e4e4fce57d90ecc5cb1408b16d6b0d5fed9ed0`; `.goreleaser.yaml:11-41` excludes Windows ARM64 and requires license/notices/checksums; downloaded release metadata lists exactly five archives plus checksums, license, notices, CycloneDX SBOM, and published provenance; published provenance identifies the same revision/tree, tag, successful workflow, and production hashes.
- [✓] Complete anonymous clone, install, docs, landing, source, issue, support, security, and artifact-integrity journeys — the supplied evidence records credential-free clone and public-route crawls, five passing SHA-256 rows, matching production landing/docs revisions, and a clean Darwin ARM64 archive completing version, `init --target codex`, `status`, and `check`; the downloaded release and smoke directories retain the exercised artifacts.
- [✓] Do not change Sprout, Hero Code, or Hero Cloud — the supplied anonymous API evidence records Hero Code and Hero Cloud as 404/private and Sprout as the separate public MIT repository; public boundary checks remain encoded in `internal/cli/public_docs_check.go:380-405`.

## Changes

- [✓] Re-run exposure, licensing, candidate, DNS, deployment, source-link, support, and security gates — supplied evidence records the full exposure scan exit 0, Go/docs/landing/release suites passing, `goreleaser check`, and anonymous destination and exact-revision checks.
- [✓] Record approval boundary and exact resolved launch outputs — `spec.md:64-76`, `docs/releases/v0.34.0-launch-checklist.md:59-68`, candidate provenance, and published provenance record the bounded authorization, sibling exclusions, source revision/tree, five production hashes, host/security state, and private archive.
- [✓] Publish the public repository, annotated tag, host settings, and five-target release with trust artifacts — the tag object identifies Chet Bellows and revision `34e4e4fc...`; downloaded release metadata and provenance record the successful Actions run and exactly ten expected assets; `.goreleaser.yaml:23-41` enforces the supported target and archive contract.
- [✓] Enable and verify public destinations and anonymous journeys — public source-link requirements are enforced by `internal/cli/public_docs_check.go:386-405` and `web/landing/scripts/landing_build.py:341-363`; the supplied external evidence records 200 responses, matching revision markers, package-channel hashes, and archive exercise.
- [✓] Capture immutable evidence and follow the rollback/incident contract — the immutable annotated tag and published provenance bind the release to the source tree and successful workflow; the supplied evidence records the recoverable private archive, mirror, all-refs bundle, metadata, and checksum manifest, while `docs/releases/v0.34.0-launch-checklist.md:70-85` retains the post-publication rollback rules. The mismatched extra target was reconciled before completion and the final ten-asset contract is evidenced.

## Audit notes

- None.
