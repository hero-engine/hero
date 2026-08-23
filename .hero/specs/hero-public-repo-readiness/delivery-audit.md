# Delivery audit — hero-public-repo-readiness

**Audited:** supplied no-index delivery snapshots plus `git diff c5b9d060 -- README.md .hero/planning/initiatives/hero-marketing/hero-public-repo-readiness/spec.md`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] AC-1: report public-exposure risks with disposition evidence — `exposure-audit.md:44-135` records proprietary history, machine-local databases, credential-shape dispositions, commit identities, absolute paths, and internal endpoints; `exposure-audit.md:164-178` reconciles large, generated, and third-party material. The scanner emits only redacted evidence and exact fingerprints (`scripts/public-readiness-scan.sh:34-52,55-87`).
- [✓] AC-2: provide current contribution, security, conduct, support, issue, and pull-request surfaces with valid destinations — `CONTRIBUTING.md:3-63`, `SECURITY.md:12-39`, `CODE_OF_CONDUCT.md:20-32`, and `SUPPORT.md:6-26` define truthful current/future routes; `.github/ISSUE_TEMPLATE/bug.yml:1-67`, `.github/ISSUE_TEMPLATE/feature.yml:1-38`, `.github/ISSUE_TEMPLATE/config.yml:1`, and `.github/PULL_REQUEST_TEMPLATE.md:1-15` provide the GitHub contribution surfaces. The supplied evidence says issue-form YAML parsed and all relative policy links resolved.
- [✓] AC-3: hide or label anonymous source routes while the repository is private — `README.md:273-282`, `CONTRIBUTING.md:3-12`, `SECURITY.md:18-25`, and `SUPPORT.md:14-26` state that source, issue, and private-reporting routes are not active until their gates complete; `exposure-audit.md:137-149` records zero direct private-repository URLs across the public-facing surfaces.
- [✓] AC-4: block public readiness when proprietary Code/Cloud content is detected and do not grant it — `scripts/public-readiness-scan.sh:95-108,125-138` makes Cloud source paths blocking findings in current and reachable-history scans; `exposure-audit.md:46-64,151-162` distinguishes 45 historical source blobs from name-only contracts, assigns remediation, preserves Hero Code/Cloud as proprietary, and blocks visibility.
- [✓] AC-5: leave visibility unchanged without explicit approval — `repository-settings-checklist.md:5-20,56-70` records unchanged/unverified host settings, retains the visibility HOLD, and requires explicit owner approval last; `exposure-audit.md:7-15,180-192` confirms no visibility or history mutation occurred and keeps exposure blocked.

## Changes

- [✓] Audit tracked content and reachable history — `scripts/public-readiness-scan.sh:17-162` scans tracked files and named reachable blobs, redacts matched content, applies exact baselines, and exits `2` on blockers; `scripts/test-public-readiness-scan.sh:14-69` asserts historical proprietary/session detection, output redaction, exact baseline acceptance, changed-match reblocking, and a clean-repository exit.
- [✓] Add minimum policy and GitHub contribution surfaces — the four root policies, two issue forms, issue configuration, and pull-request template are present in the delivery diff and consistently preserve private/no-license and proprietary-product boundaries.
- [✓] Repair root navigation and gate private source routes — `README.md:263-282` links documentation and every root policy while stating that the preparation neither activates public routes nor grants redistribution rights.
- [✓] Prepare the host-settings checklist — `repository-settings-checklist.md:12-54` covers repository identity/homepage, issues/discussions, default branch, branch protection and required checks, vulnerability reporting, secret scanning, push protection, dependency alerts, and Actions permissions without applying them.
- [✓] Reconcile owner/licensing boundaries and fail on proprietary material — `exposure-audit.md:151-178` consumes the completed licensing boundary, keeps Sprout separate under MIT, excludes Hero Code/Cloud, and preserves required grant/notice gates; `exposure-audit.md:180-192` fails readiness on the remaining historical blockers.

## Audit notes

- The delivery is complete as a readiness and gating artifact; it does not make the repository ready to expose. Publication remains blocked by 45 reachable proprietary Cloud source blobs and three machine-local session databases (`exposure-audit.md:46-77`).
- Before visibility changes, the owner must also resolve or explicitly accept the redacted personal/internal-data findings, close licensing and host-setting gates, and approve the final launch sequence (`exposure-audit.md:107-135,180-185`; `repository-settings-checklist.md:56-70`).
- Validation evidence is substantive: the scanner self-test passed; the 26,027-row authoritative history scan exited `2` for the intended 48 blockers; issue-form YAML and policy links passed; direct private-source URLs were absent; anonymous repository/docs checks returned `404`/`200`; `git diff --check`, `go test ./...`, and `hero drift hero-public-repo-readiness` passed.
