# Delivery audit — landing-docs-content-refresh

**Audited:** `git diff HEAD` (uncommitted working tree against HEAD `4e0daa5`)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria

- [✓] Cross-repo peering card replaces Tripwire card in "What we're building next" — `web/landing/site/index.html:900-906`; `<span class="soon-label">Shipped</span>` and `<h3>Your projects know each other</h3>` present; correct copy matches spec verbatim.
- [✓] Peering card shows label `Shipped` — `web/landing/site/index.html:900`; prior `In progress` label on the Tripwire card replaced with `Shipped`.
- [✓] "Knowledge compounds" why-card includes offline semantic search sentence — `web/landing/site/index.html:801`; exact spec text appended after the "corpus stays" sentence, in the correct card (confirmed by grep context showing `<h3>Knowledge compounds</h3>` above).
- [✓] `delivery-and-debugging.md` describes `hero verify` as delivery gate with command shown — `web/docs/src/workflows/delivery-and-debugging.md:40-67`; new section "The verify gate" added; command shown in bash block (`hero verify <slug>`, with `--skip-tests` and `--json` variants); `required checkpoint` wording present; warning block added.
- [~] `installation.md` has monorepo/multi-workspace section — `web/docs/src/getting-started/installation.md:90-116`; "Monorepo setup" section present with `hero install project . --target claude` command and satellite concept explained. **Spec also required "Cross-reference the main install steps so it's findable" — no such cross-reference is present.** The section stands alone at the bottom of the file. Minor gap; the section is still discoverable by scrolling, and the install steps are above it in the same file.
- [✓] `knowledge-base.md` explains tripwire guardrails as institutional knowledge — `web/docs/src/concepts/knowledge-base.md:73-102`; "Tripwire guardrails" subsection added with `forbidden:` marker concept; spec-scoring/sizing note added in "Spec scoring and sizing" sub-subsection.
- [✓] No new MIT mentions in landing page — `grep MIT web/landing/site/index.html` returns 0 results; the diff shows only removals (meta description, eyebrow span, footer text, MIT license link in footer).

## Changes

- [✓] `web/landing/site/index.html` — Tripwire card replaced with cross-repo peering card (label, title, copy); semantic search sentence appended to Knowledge compounds card; MIT mentions removed from meta description, eyebrow, footer paragraph, footer link, and footer copyright line.
- [✓] `web/docs/src/workflows/delivery-and-debugging.md` — Delivery workflow step 3 updated from "Validation" to "Verify gate"; new "The verify gate" section added with command block and warning admonition.
- [~] `web/docs/src/getting-started/installation.md` — "Monorepo setup" section added with satellite install concept and code examples; missing the explicit cross-reference to main install steps called out in the spec.
- [✓] `web/docs/src/concepts/knowledge-base.md` — "Tripwire guardrails" section and "Spec scoring and sizing" sub-section added.

## Open items

None blocking. One minor gap noted below.

## Audit notes

1. **Cross-reference gap (minor, non-blocking):** The spec explicitly required `installation.md` to "Cross-reference the main install steps so it's findable." The Monorepo section is at the bottom of the file immediately before `## Next steps` with no link or note pointing readers to the preceding Quick install / platform install sections above. The section is still in the right file and will be reached by anyone reading top-to-bottom, but the explicit cross-reference requirement is unmet. Downgraded the `installation.md` Changes row to `~`. This is a soft gap — the AC as stated ("find a monorepo or multi-workspace section explaining `hero install --target` for subfolder workspaces") **is** met; only the Approach description's sub-requirement is missed.

2. **Bonus MIT removals in scope:** The diff removes MIT mentions beyond what the spec targeted (meta `<description>`, eyebrow badge, footer paragraph, MIT license footer link, copyright line). These are consistent with the spirit of the "No new MIT mentions" AC and with the prior-pass license removal mentioned in the spec. Not a concern — this is additive correctness.

3. **Hero-managed files in diff:** `.hero/NEXT.md`, `.hero/SNAPSHOT.md`, `.hero/next/chet-bellows.md`, `.hero/QUEUE.md` are timestamp refreshes and session turnover — not spec changes. Out of scope, as expected.

4. **No tests:** Changes are content-only (HTML, Markdown). No automated test suite exists for static copy. Verification was visual/grep-based. Acceptable for this change type.
