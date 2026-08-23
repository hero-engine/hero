# Proposed Apache-2.0 Repository Boundary

Audit date: 2026-08-23

## Owner authorization

The repository owner stated in this initiative that he owns 100% of Hero and authorizes preparing the `hero` repository for an Apache-2.0 grant. That statement is the authority for Hero-owned material in this repository. This packet does not reopen a contributor-by-contributor rights investigation.

The authorization is preparation authority only. Adding the license file and making the repository public remain explicit, separate approval gates.

## Included in the proposed grant

The proposed Apache-2.0 grant covers Hero-owned material tracked in `github.com/hero-engine/hero`, including:

| Category | Repository paths | Treatment |
|---|---|---|
| CLI and libraries | `cmd/`, `internal/`, `contracts/`, root Go sources | Hero-owned source; included |
| Product content | `core/`, `domains/` | Hero-authored agents, workflows, skills, spec types, methodologies, and vocabularies; included |
| Project tooling | `Makefile`, `scripts/`, `.github/`, `.goreleaser.yaml`, `tools/` | Hero-owned build, install, release, and maintenance material; included |
| Public documentation | `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CHANGELOG.md`, `web/docs/src/` | Hero-owned prose, examples, configuration, and site source; included |
| Public landing site | `web/landing/site/` | Hero-owned HTML, CSS, and SVG artwork; included |
| Hero planning corpus | tracked `.hero/` material | Hero-owned specifications, decisions, conventions, and projected state; included unless removed before publication for product or privacy reasons |
| Generated harness copies | tracked `.agents/`, `.claude/`, and related generated instructions | Generated from included Hero content; included, but public-readiness checks must identify machine-local configuration before visibility changes |
| Embedded model derivative | `internal/embeddings/defaultmodel/` | Included only under the upstream MIT terms recorded in the third-party inventory; not relicensed as exclusively Hero-owned |

The Apache grant applies to Hero-authored portions. Third-party material retains its own license and attribution, even when embedded in a Hero binary or generated site.

## Explicitly excluded

| Repository or material | Status | Reason |
|---|---|---|
| `hero-code` | Proprietary; excluded | Separate private product. No source, binary, documentation, or brand asset is granted by the Hero license. |
| `hero-cloud` | Proprietary; excluded | Separate private product. No source, service code, documentation, or deployment material is granted by the Hero license. |
| Sprout (`github.com/bdwheeler/sprout`) | Separate public MIT project | Consumed as a dependency; not owned by or relicensed through the Hero repository grant. |
| Go modules and Python packages | Third-party licenses | Referenced or redistributed under the licenses in `third-party-inventory.md`. |
| User or machine-local state | Not part of the repository grant | Ignored credentials, local configuration, caches, sessions, generated binaries, and untracked working files are outside the tracked publication set. |
| External names and trademarks | No trademark grant implied | Apache-2.0 covers copyright and patent terms, not ownership of third-party marks. |

## Repository-to-product rule

Public language may say “Hero is open source” only when it clearly means this `hero` CLI/repository. It must not imply that Hero Code, Hero Cloud, or Sprout are licensed under Hero's Apache-2.0 grant. Sprout may be described separately as MIT-licensed.
