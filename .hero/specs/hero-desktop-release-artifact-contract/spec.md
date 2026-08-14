---
title: "Hero release cannot supply a version-matched binary artifact to Hero Desktop"
slug: hero-desktop-release-artifact-contract
type: bug
status: completed
domain: engineering
size: small
priority: critical
severity: high
created: 2026-08-14
tags: [release, hero-code, artifact, goreleaser, desktop]
delivery_method: manual
completed_at: 2026-08-14T16:29:35Z
---

# Hero release cannot supply a version-matched binary artifact to Hero Desktop

## Goal

Make every tagged Hero release produce the exact macOS ARM64 executable Hero
Code's sealed-candidate workflow consumes, with one version identity shared by
the binary and the source tag.

## Kickoff

Repairs the Hero-to-Hero-Code release handoff: publish a raw Darwin ARM64
Actions artifact and stamp binaries with the `v`-prefixed release identity.

**Status:** delivering — implementation and the full release rehearsal are green.

**Pick up at:** cold-audit the three-file contract change, then run
`hero spec verify hero-desktop-release-artifact-contract`.

→ `/deliver hero-desktop-release-artifact-contract`

**Files:** `.goreleaser.yaml`, `.github/workflows/release.yml`, `internal/releasecontract/release_contract_test.go`
**Skip:** changing Hero Code's sealed-candidate inputs or migrating Homebrew packaging.

## Problem

Hero Code's sealed desktop workflow downloads an approved Hero executable by
Actions repository, run ID, artifact name, and SHA-256. Hero's release workflow
only asks GoReleaser to publish archives to `hero-engine/hero-releases`; it does
not upload a workflow artifact containing the raw executable. There is no run
ID and artifact name Hero Code can supply.

The same boundary has a second incompatibility. GoReleaser stamps
`{{.Version}}`, so a `v0.32.0` tag produces a binary reporting `0.32.0`. Hero
Code intentionally uses the approved binary's reported version as the source
checkout ref and requires source and binary identity to match. The source ref
is `v0.32.0`, not `0.32.0`, so the sealed workflow cannot check it out.

The release workflow also still uses `actions/checkout@v4`, while this
repository's test and smoke workflows have moved to `@v6` for the current
JavaScript runtime.

## Approach

Keep the consumer contract unchanged and repair the producer:

1. Stamp GoReleaser binaries as `v{{.Version}}`. Hero's development builds
   already use `git describe` and report a `v`-prefixed identity; tagged
   release binaries will now match their immutable source ref exactly.
2. After GoReleaser succeeds, resolve exactly one raw Darwin ARM64 `hero`
   executable from `dist/`, fail on zero or multiple candidates, copy it into a
   stable staging directory, record its SHA-256, and upload that directory as
   the stable Actions artifact `hero-darwin-arm64`.
3. Upgrade the release workflow to `actions/checkout@v6` and retain
   `actions/upload-artifact@v4` with explicit missing-file failure and bounded
   retention.
4. Add a repository structural guard that pins the artifact name, raw-binary
   selection/failure contract, action versions, and the `v`-prefixed ldflag.

The public GoReleaser archives and package-manager publication remain
unchanged. The raw Actions artifact is an additional private provenance input
for Hero Code.

## Acceptance Criteria

- **AC-1:** WHEN a tagged release completes GoReleaser THE SYSTEM SHALL upload an Actions artifact named `hero-darwin-arm64` containing exactly one raw executable named `hero` and its SHA-256 record.
- **AC-2:** IF the GoReleaser output contains zero or multiple Darwin ARM64 `hero` executables THEN THE SYSTEM SHALL fail before uploading an artifact.
- **AC-3:** WHEN a release is built from tag `vX.Y.Z` THE SYSTEM SHALL make the executable report `hero version vX.Y.Z`, matching the immutable source ref Hero Code checks out.
- **AC-4:** WHEN a snapshot is built THE SYSTEM SHALL preserve the `v` prefix in the snapshot's reported version and SHALL continue producing all six configured OS/architecture archives.
- **AC-5:** THE SYSTEM SHALL pin the release workflow to `actions/checkout@v6` and `actions/upload-artifact@v4`, fail on missing artifact files, and retain the artifact for a bounded period.
- **AC-6:** IF the artifact name, binary-selection guard, action versions, or GoReleaser version stamp drifts THEN a repository test SHALL fail with the violated contract.
- **AC-7:** WHEN Hero Code validates the generated Darwin ARM64 executable THE SYSTEM SHALL satisfy its architecture, macOS deployment-floor, executability, and reported-version checks.

## Changes

1. Update `.goreleaser.yaml` to stamp `v{{.Version}}` into `main.version`.
2. Update `.github/workflows/release.yml` to use checkout v6, stage exactly one
   Darwin ARM64 executable plus checksum, and upload stable artifact
   `hero-darwin-arm64` after GoReleaser succeeds.
3. Add `internal/releasecontract/release_contract_test.go` to guard the
   producer contract structurally without requiring GitHub credentials.

## Validation

- Run the new release-contract test and falsify each guarded value once before
  trusting green.
- Run `goreleaser check`.
- Run `goreleaser release --snapshot --clean`; verify six archives, inspect
  `dist/artifacts.json`, run the Darwin ARM64 executable, and record its SHA.
- Run Hero Code's `validate-release-inputs.sh` against that executable.
- Run `go test -race -count=1 ./...`, `go vet ./...`, and `go build ./...`.

## Rollout and rollback

Land this change before creating `v0.32.0`. The tag-triggered release remains
the first credentialed end-to-end execution. If artifact staging fails, no
Hero Code candidate can be sealed and the release job fails visibly. Rollback
is reverting this commit and continuing to pin Hero Code to the last approved
artifact; no schema or persistent data changes are involved.

## Boundaries

- Do not change Hero Code's immutable candidate inputs or provenance checks.
- Do not migrate the deprecated GoReleaser Homebrew pipe in this fix.
- Do not change public archive names, supported platforms, or release targets.
- Do not expose release tokens or checksums through command-line arguments.

## Completion Ledger

Implementation keeps the existing Hero Code consumer contract and repairs the
Hero release producer. GoReleaser snapshot, Hero Code's real release-input
validator, and the repository-wide race suite are green.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Upload stable artifact with one raw binary and SHA | DONE | `.github/workflows/release.yml` stages `hero` + `hero.sha256` and uploads `hero-darwin-arm64`; exercised locally against snapshot output. |
| 2 | Fail on zero or multiple Darwin ARM64 candidates | DONE | Candidate count is computed from `dist/` and must equal one before staging; structurally guarded and exercised with one real candidate. |
| 3 | Tagged binary reports the v-prefixed source ref | DONE | `.goreleaser.yaml` stamps `v{{.Version}}`; snapshot executable reports `hero version v0.31.2-next`. |
| 4 | Snapshot keeps v prefix and six archives | DONE | `goreleaser release --snapshot --clean` succeeded with six archives and v-prefixed binary output. |
| 5 | Current action runtimes and bounded failure policy | DONE | Release uses checkout v6/upload-artifact v4, `if-no-files-found: error`, and 90-day retention. |
| 6 | Structural drift fails repository tests | DONE | `internal/releasecontract/release_contract_test.go` pins every contract value and programmatically falsifies each guard. |
| 7 | Hero Code accepts the Darwin ARM64 executable | DONE | `validate-release-inputs.sh` passed for arm64, macOS 15 floor, executability, and `v0.31.2-next`. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Stamp `v{{.Version}}` in GoReleaser | DONE | `.goreleaser.yaml` updated and exercised through a complete snapshot. |
| 2 | Stage and upload Hero Code artifact | DONE | `.github/workflows/release.yml` selects exactly one binary, records SHA-256, uploads the stable artifact, and uses checkout v6. |
| 3 | Add release producer contract guard | DONE | New `internal/releasecontract` tests cover required and forbidden values, including mutation-based falsification. |

### Exercise-the-feature check

- [x] Ran `goreleaser release --snapshot --clean`; staged its real Darwin ARM64
  executable through the workflow's selection/copy/checksum path; observed
  `hero version v0.31.2-next`; produced six archives; and passed Hero Code's
  `validate-release-inputs.sh` against the staged binary.

### Excellence Bar self-check

Yes. The change repairs the producer rather than weakening Hero Code's immutable
consumer checks, fails closed on ambiguous artifacts, and carries a falsifiable
structural guard plus a real packaging exercise.
