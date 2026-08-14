# Delivery audit — hero-desktop-release-artifact-contract

**Audited:** `git diff --cached 62978643500d604b79d071eabb5ef6c32a54226e -- .goreleaser.yaml .github/workflows/release.yml internal/releasecontract/release_contract_test.go .hero/planning/bugs/hero-desktop-release-artifact-contract/spec.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — `.github/workflows/release.yml:40-62` selects the Darwin ARM64 executable, stages it as `hero`, writes `hero.sha256`, and uploads the directory as `hero-darwin-arm64`; the supplied staging exercise used the snapshot output.
- [✓] AC-2 — `.github/workflows/release.yml:43-50` removes empty results, requires a candidate count of exactly one, exits before staging on any other count, and selects the sole candidate only after the guard.
- [✓] AC-3 — `.goreleaser.yaml:23-24` stamps `main.version=v{{.Version}}`; supplied snapshot evidence reports `hero version v0.31.2-next`, matching the source-ref shape used by Hero Code.
- [✓] AC-4 — the snapshot evidence records six archives, and the on-disk `dist/` inventory contains Darwin, Linux, and Windows archives for both amd64 and arm64; the supplied executable output retains the `v` prefix.
- [✓] AC-5 — `.github/workflows/release.yml:18` uses checkout v6 and lines 56-62 use upload-artifact v4 with `if-no-files-found: error` and 90-day retention.
- [✓] AC-6 — `internal/releasecontract/release_contract_test.go:18-30` pins the declared contract values, lines 32-37 enforce them against repository files, and lines 39-67 mutate every required value plus the two explicit legacy regressions and require each mutation to fail validation.
- [✓] AC-7 — supplied Hero Code `validate-release-inputs.sh` evidence passed against the staged Darwin ARM64 binary for architecture, macOS 15 deployment floor, executable mode, and `v0.31.2-next` version.

## Changes

- [✓] Stamp `v{{.Version}}` in GoReleaser — `.goreleaser.yaml:23-24`.
- [✓] Stage and upload the Hero Code artifact — `.github/workflows/release.yml:18,40-62` upgrades checkout, enforces exact-one selection, stages the raw executable and checksum, and uploads the stable artifact.
- [✓] Add the release producer contract guard — `internal/releasecontract/release_contract_test.go:1-107` validates repository files and proves each guard is falsifiable.

## Audit notes

- No performative DONE rows, partial delivery, skipped work, or out-of-scope implementation were found.
- The first credentialed tag-triggered workflow remains a rollout event, as the spec states; local evidence covers configuration, packaging, staging, and the unchanged Hero Code consumer validator without weakening its provenance checks.
