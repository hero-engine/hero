# Delivery audit — hero-landing-message-refresh

**Audited:** `git diff b514e8083b16edbe0e1cf6d464d4362b15189ad9 -- .github/workflows/landing.yml web/landing web/docs/src/index.md web/docs/src/assets/logo.svg web/docs/src/assets/favicon.svg internal/serve/shell/static/favicon.svg .hero/marketing/positioning.md .hero/planning/initiatives/hero-marketing/hero-landing-message-refresh/spec.md`, plus direct inspection of untracked landing assets and scripts
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 — memory is the primary promise and verified delivery is the execution layer — `web/landing/site/index.html:316-384`; the canonical tagline also appears in `web/docs/src/index.md:3` and `.hero/marketing/positioning.md:62`.
- [✓] AC-2 — claimed outcomes are linked or labeled illustrative — the terminal is visibly labeled at `web/landing/site/index.html:329-347`, and each proof link has a matching tracked docs page.
- [✓] AC-3 — stale and unsupported public claims are absent — `web/landing/scripts/landing_build.py:50-60,334-353`; the negative regression at `web/landing/scripts/test_landing_build.py:115-145` exercises the rejection path.
- [✓] AC-4 — capability maturity and prerequisites are adjacent — `web/landing/site/index.html:447-470`, including the bounded headless-runtime language at `:464-466`.
- [✓] AC-5 — proof links target tracked docs and artifacts identify their exact input bytes — `web/landing/scripts/landing_build.py:146-210` derives commit, source-tree digest, dirty state, and composite revision; `:425-463` validates the artifact identity. The dirty-worktree build produced the expected commit plus digest identity.
- [✓] AC-6 — the scoped local artifact checks pass — `web/landing/scripts/landing_build.py:280-470` validates structure, responsive/accessibility signals, assets, links, claims, and build identity; independent local HTTP checks returned 200 for all five assigned paths. The spec explicitly leaves production and browser proof to the launch gate.
- [✓] AC-7 — the artifact and launch-gated deployment path are prepared without crossing the launch gate — `.github/workflows/landing.yml:35-77` orders tests, source validation, build, artifact validation, and upload before the manual approved deploy; `web/landing/wrangler.toml:5` serves only `dist/`.
- [✓] AC-8 — the primary hierarchy establishes memory before workflow mechanics — `web/landing/site/index.html:317-404` precedes the delivery section beginning at `:408` and distinguishes Hero from another spec kit at `:359`.

## Changes

- [✓] Lead with memory, then connected verified delivery — `web/landing/site/index.html:316-434`.
- [✓] Remove stale proof and derive exact build identity — `web/landing/scripts/landing_build.py:146-234`; clean explicit builds bind to HEAD, dirty local builds use `<commit>+worktree:<digest>`, and invalid explicit identities fail.
- [✓] Label abbreviated output illustrative — `web/landing/site/index.html:329-347`.
- [✓] Separate shipped, optional, preview, and planned capability claims — `web/landing/site/index.html:447-470`.
- [✓] Route proof to docs and publish a trustworthy revision marker — six destinations have tracked source pages; `web/landing/site/revision.json` and rendered HTML share the validated composite revision.
- [✓] Prepare canonical metadata and a launch-gated deployment path — `web/landing/site/index.html:6-19`, `.github/workflows/landing.yml:59-77`, `web/landing/README.md`, and `web/landing/wrangler.toml`. No deployment, DNS, visibility, or license mutation appears in the diff.
- [✓] Add landing-specific regression coverage — eight tests cover claims, structure, responsive/accessibility signals, links, canonical assets, clean and dirty identities, explicit-revision rejection, digest mismatch, and unresolved placeholders (`web/landing/scripts/test_landing_build.py:34-205`).

## Open items

- None.

## Audit notes

- The canonical lowercase-h paths and color are geometrically identical to `hero-code` commit `e4077c12:assets/icons/hero-logo.svg`; all normalized SVG copies share SHA-256 `05d31357...`, and the obsolete bolt path is absent. The 1731×909 social PNG visibly uses the lowercase-h mark and exact tagline and matches locked SHA-256 `4b5fc06f...`.
- Independent checks passed: eight landing tests, tracked-source validation, dirty-worktree build, built-artifact validation, workflow YAML parsing, `git diff --check`, and local HTTP 200 responses for `/`, `/revision.json`, `/hero-logo.svg`, `/og-image.png`, and `/sitemap.xml`.
- The dirty build emitted `b514e8083b16edbe0e1cf6d464d4362b15189ad9+worktree:2cd139d1f00f7ab8a2e21fbd9f36460faae0cb0735546a0e69dc9a4d83bbfc9c`; an explicit CI identity against the same dirty source failed closed.
- The duplicate sitemap directive is removed, and the Completion Ledger now enumerates the supporting shared logo/favicon changes.
