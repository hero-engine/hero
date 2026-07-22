# Delivery audit — jira-adf-description-fidelity-loss

**Audited:** `git diff --cached -- README.md internal/tracker/jira_adf.go internal/tracker/jira_adf_test.go internal/tracker/testdata/jira/morph-297-description.json internal/tracker/testdata/jira/morph-297-description.md internal/tracker/jira.go internal/tracker/jira_fields.go internal/cli/sync_import_test.go internal/cli/sync_push_merge_test.go .hero/planning/initiatives/tracker-source-fidelity-and-evidence/jira-adf-description-fidelity-loss/spec.md` (plus the spec-required staged `internal/tracker/sprint.go` change, which the supplied command omits)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: Nested lists preserve source order, starting ordinal, and nesting — `internal/tracker/jira_adf.go:156`; exact nested-list output is asserted by `TestADFToMarkdown_MORPH297Fixture`.
- [✓] AC-2: Text marks render deterministically — fixed-order code/strong/emphasis/strike/link rendering and titled links are implemented at `internal/tracker/jira_adf.go:322` and tested in both source orders at `internal/tracker/jira_adf_test.go:50`.
- [✓] AC-3: Block nodes preserve meaning — headings, blockquotes, panel allowlisting/fallback, rules, exact `<br>\n`, and safe language fences are implemented at `internal/tracker/jira_adf.go:80` and asserted by the golden and focused table tests.
- [✓] AC-4: Inline semantic nodes remain readable — mention/status/emoji/card/date/media representations and fallbacks are implemented at `internal/tracker/jira_adf.go:245` and asserted at `internal/tracker/jira_adf_test.go:70`.
- [✓] AC-5: Plain JSON strings pass through byte-for-byte — `internal/tracker/jira_adf.go:61`; asserted at `internal/tracker/jira_adf_test.go:29`.
- [✓] AC-6: Unknown nodes retain descendants or leaf text — recursive fallback is at `internal/tracker/jira_adf.go:135`; container and leaf retention are fixture/compatibility tested.
- [✓] AC-7: Missing, invalid, and partially malformed input is best-effort — per-child tolerant decoding is at `internal/tracker/jira_adf.go:18`; malformed sibling preservation is asserted at `internal/tracker/jira_adf_test.go:90`.
- [✓] AC-8: MORPH-297 repeatedly equals the exact golden — `TestADFToMarkdown_MORPH297Fixture` performs ten exact renders at `internal/tracker/jira_adf_test.go:13`.
- [✓] AC-9: Core issue read surfaces return identical golden bytes — two-page `ListIssues`/`Search`, `GetIssue`, and evidence normalization are asserted by `TestJiraADFMarkdown_IsIdenticalAcrossReadSurfaces`.
- [✓] AC-10: Evidence comments, sprint items, and `GetFields` use canonical bytes while raw evidence remains retained — assertions are at `internal/tracker/jira_adf_test.go:169`; direct wiring is at `internal/tracker/jira.go:768`, `internal/tracker/jira_fields.go:120`, and `internal/tracker/sprint.go:575`.
- [✓] AC-11: Initial generated content and baseline preserve the golden — generation is asserted at `internal/cli/sync_import_test.go:292`; baseline equality is asserted at `internal/cli/sync_push_merge_test.go:375`.
- [✓] AC-12: Refresh repairs untouched placeholders and preserves authored content — exact repair/baseline and authored-body tests are at `internal/cli/sync_import_test.go:446` and `internal/cli/sync_import_test.go:511`.
- [✓] AC-13: Canonical equality emits no patch and renderer upgrades never push flattened text — both cases are asserted at `internal/cli/sync_push_merge_test.go:407` and `internal/cli/sync_push_merge_test.go:440`.
- [✓] AC-14: Exactly one inbound implementation remains — all audited call sites invoke `jiraADFToMarkdown`; both shallow canonicalizers are deleted.
- [✓] AC-15: Existing tracker/write behavior is preserved — outbound `textToADF` is unchanged; tracker/CLI suites, focused race tests, vet, and staged diff checks pass.

## Changes

- [✓] Canonical renderer and MORPH-297 fixtures — recursive tolerant rendering, exact goldens, supported-node/mark cases, missing attributes, malformed children, fence growth, escaping, panel fallback, and repeated output are covered.
- [✓] Route all inbound Jira paths — issue, evidence comment, custom-field fallback, field diff, and sprint paths directly use `jiraADFToMarkdown`; both shallow implementations are removed.
- [✓] Adapter parity tests — the isolated server proves exact bytes across the named surfaces, including two-page list/search behavior and raw evidence retention.
- [✓] Import/refresh/merge parity tests — generation, baseline, placeholder repair, authored-body preservation, canonical equality, and stale-flattened regression cases are asserted.
- [✓] Compatibility/repair documentation — `README.md:263` documents explicit repair and the no-migrator/no-authored-overwrite policy.

## Audit notes

- No open items, performative DONE rows, or scope drift found in the audited delivery surface.
- Independent focused tracker/CLI tests and `git diff --cached --check` pass; the supplied full package, expanded race, and vet evidence also passes.
