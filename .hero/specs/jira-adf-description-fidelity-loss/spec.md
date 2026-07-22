---
title: "Jira ADF descriptions lose structure during import"
slug: jira-adf-description-fidelity-loss
type: bug
status: completed
domain: engineering
surface: hero-engine-tracker
parent: tracker-source-fidelity-and-evidence
size: medium
priority: critical
severity: critical
root_cause_class: code
created: 2026-07-21
tags: [tracker, jira, adf, import, refresh, evidence, markdown, fidelity]
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero
  originator_slug: jira-adf-description-fidelity-loss
  handed_off_at: 2026-07-21T15:07:50Z
  at_commit: 37be49c2
  reason: "Hero Code exposes Jira descriptions already damaged by the bundled Hero v0.28.0 tracker normalization; MORPH-297 proves nested ADF lists, marks, status nodes, and code fences are discarded before persistence."
relations:
  - target: lazy-tracker-evidence-sidecar
    kind: conflicts-with
delivery_method: manual
completed_at: 2026-07-22T00:57:21Z
---

# Jira ADF descriptions lose structure during import

## Provenance

Handed off from Hero Code through peer `hero` (peer_id `cd8dd06d-3df1-4878-a88f-24593dcbb4b3`). Originator spec: `jira-adf-description-fidelity-loss`.

**Reason:** Hero Code exposes Jira descriptions already damaged by the bundled Hero v0.28.0 tracker normalization; MORPH-297 proves nested ADF lists, marks, status nodes, and code fences are discarded before persistence.

## Issue

Jira Cloud `MORPH-297` contains nested preconditions and reproduction steps, marked Expected/Actual labels, status/emoji nodes, and logs in a code block. Hero v0.28.0 persisted only top-level text: list bodies disappeared, marks and inline semantic nodes were dropped, and the code block became ordinary prose. The generated spec and `.hero/sync-state` baseline contain the same damaged string, proving the loss occurs in the Hero engine before Hero Code renders it.

The Jira source remains intact, but every agent or consumer reading the imported spec receives incomplete evidence. That makes priority and severity `critical`: the defect silently deletes diagnostic source material and can influence an incorrect diagnosis or sync decision without an error or visible fidelity warning.

## Investigation

The inbound flow is:

1. `internal/tracker/jira.go:1082-1118` requests `description` from `ListIssues` and `Search`; `GetIssue` requests it at `internal/tracker/jira.go:627-660`.
2. All normal issue responses pass through `parseIssueRaw`; `internal/tracker/jira.go:942-945` assigns `Issue.Description` from `extractADFText`.
3. `extractADFText` at `internal/tracker/jira.go:1355-1392` models only `doc → block → inline text`. It cannot reach `orderedList → listItem → paragraph → text`, does not model marks or node attributes, and gives every top-level block paragraph spacing regardless of block type.
4. `GetIssueEvidence` reuses `parseIssueRaw` for `Normalized` and separately calls the same shallow reader for comment `Text` at `internal/tracker/jira.go:663-768`; raw fields remain lossless, but normalized evidence is damaged.
5. Jira sprint loading independently calls `extractADFText` at `internal/tracker/sprint.go:574-576`.
6. Shared-field reads use a duplicate function, `adfToText`, at `internal/tracker/jira_fields.go:147-178`. It is equally shallow and joins blocks with `\n` instead of `\n\n`, so even flat descriptions have two canonical forms.
7. `generateImportedSpec`, `refreshImportedSpecs`, `seedBaselineFromIssue`, and `advanceBaselineOnPull` copy the already-normalized `Issue.Description`; they cannot recover discarded structure.

### Root cause

**Classification: code.** Hero treats hierarchical ADF as a fixed two-level plain-text payload. The primary parser discards descendants and attributes it cannot model; the field-sync path duplicated that parser with different spacing. Tests cover only two flat paragraphs, so the loss and cross-path inconsistency were never exercised.

### Severity

**Critical.** The failure is silent, affects every Jira Cloud description using common rich nodes, removes reproduction evidence before agents see it, and persists the damaged result into specs and sync baselines. Jira retains the source, so explicit re-import is a workaround, but existing local data alone cannot reconstruct what Hero dropped.

## Goal

Replace both shallow readers with one provider-owned, recursive, deterministic Jira ADF-to-Markdown renderer and make every inbound Jira description/comment path use it, so normal issues, evidence normalization, sprint/import/refresh, baselines, and diff comparison receive byte-identical readable Markdown without exposing raw ADF or changing outbound write-back behavior.

## Kickoff

Fixes MORPH-297-style Jira description corruption by replacing both shallow ADF readers with one canonical recursive Markdown renderer.

**Status:** planning — root cause and all affected read paths are confirmed; no implementation has landed.

**Pick up at:** add the MORPH-297 JSON/golden Markdown fixture, implement `jiraADFToMarkdown`, then route every Jira read and diff path through it.

→ `/deliver jira-adf-description-fidelity-loss`

**Files:** `internal/tracker/jira_adf.go`, `internal/tracker/jira.go`, `internal/tracker/jira_fields.go`, `internal/tracker/sprint.go`, `internal/cli/sync_import_test.go`
**Skip:** do not redesign `textToADF` or overwrite authored spec bodies to repair old imports.

## Problem

The current normalized `Description string` contract is correct for consumers, but its implementation is not a renderer: it extracts only direct `text` properties. Adding more anonymous struct levels would remain brittle and would not solve marks, node attributes, arbitrary nesting, unknown nodes, or deterministic formatting. Keeping two readers would also preserve the false-diff risk.

The fix must retain the useful split already present in `IssueEvidence`: ordinary normalized fields are canonical Markdown; lossless provider JSON stays in `RawFields`, `RawBody`, and `Changelog` for explicit evidence consumers.

## Design

### One canonical renderer

Add `internal/tracker/jira_adf.go` with one unexported entry point:

```go
func jiraADFToMarkdown(raw json.RawMessage) string
```

Decode through a recursive generic node carrying `type`, `text`, ordered `content`, ordered `marks`, and `attrs` as raw values. The renderer must never depend on Go map iteration order. It returns a string because existing adapter contracts treat malformed/missing descriptions as empty rather than failing the entire issue fetch.

Canonical output rules:

- A JSON string passes through byte-for-byte, including whitespace; `nil`, empty, `null`, invalid JSON, or a document with no representable content returns `""` without panic.
- Block output uses `\n`, exactly one blank line between sibling block nodes, and no synthetic trailing newline. Code contents are not trimmed.
- `paragraph` renders inline children; `hardBreak` renders `<br>\n`; `heading` renders `#` through `######` from clamped `attrs.level`; `rule` renders `---`.
- `bulletList` uses `- `; `orderedList` starts at positive `attrs.order` or `1` and increments. Nested list content is indented four spaces per depth, and multi-paragraph list items retain valid continuation indentation.
- `blockquote` prefixes every rendered line with `> `. `panel` renders a blockquote whose first line is `> **<Panel type>:**` followed by its content; absent/unknown panel types use `Panel`.
- `codeBlock` emits a fenced block, includes a sanitized `attrs.language` token when present, and chooses a backtick fence at least three characters long and one longer than the longest backtick run in the payload.
- Text escapes Markdown delimiters only where needed for literal text. Marks are applied in a fixed canonical order independent of input JSON ordering: inline code, strong, emphasis, strike, with `link` outermost. Links preserve `attrs.href` and optional title using deterministic Markdown escaping.
- `mention` prefers `attrs.text`, then `attrs.displayName`, then a readable `@<id>` fallback; `emoji` prefers `attrs.text`, then `:<shortName>:`; `status` renders `[<text>]`.
- `inlineCard` and `blockCard` render a label/title plus link when an `attrs.url` exists, otherwise a readable label. `media`, `mediaSingle`, and `mediaGroup` render Markdown image/link syntax only when URL plus alt/title/filename are representable; otherwise they emit a readable `[media: <label>]` fallback without inventing a URL.
- An unknown container recursively renders recognized descendants in source order. An unknown leaf preserves its `text` when present and otherwise contributes nothing. A malformed child is skipped without discarding valid siblings.

### Shared inbound wiring

- `parseIssueRaw` uses `jiraADFToMarkdown`, covering `GetIssue`, `ListIssues`, `Search`, broker normalized reads, and `GetIssueEvidence.Normalized`.
- Evidence comment `Text` uses the same renderer while `RawBody` remains unchanged.
- Jira sprint `Description` uses the same renderer.
- `GetFields` uses the same renderer and deletes `adfToText`, making shared-field comparison byte-identical to `Issue.Description`.
- Delete `extractADFText`; custom-field fallback that currently calls it uses the canonical renderer too.
- Import, refresh, and baseline code remain consumers of `Issue.Description`; do not add provider-specific rendering or per-issue refetches to CLI paths.

### Historical data safety

Initial imports write canonical Markdown to the spec and seed the identical baseline. Refresh may fill only the existing untouched import placeholder and reseed the identical body. It does not replace an authored `## Problem`/`## Goal` section.

For old flattened local/baseline values, `GetFields` will now expose the corrected remote representation. Existing three-way merge rules must treat that as a remote change and must not push the flattened local value merely because normalization improved. Repair of an already-authored body remains explicit: re-import/recreate the untouched imported spec or manually restore it from Jira evidence.

## Changes

1. Add the canonical renderer and realistic golden fixtures.
   - Create `internal/tracker/jira_adf.go` and `internal/tracker/testdata/jira/morph-297-description.json` plus exact `morph-297-description.md`.
   - Add table-driven cases for every supported node/mark, plain strings, unknown descendants, missing attributes, malformed children/documents, fence growth, escaping, and deterministic repeated rendering.
2. Replace every inbound Jira normalization call.
   - Update `internal/tracker/jira.go`, `internal/tracker/jira_fields.go`, and `internal/tracker/sprint.go`; remove both shallow readers.
   - Keep `IssueEvidence.RawFields`, comment `RawBody`, and changelog lossless while normal descriptions/comment text become Markdown.
3. Prove adapter parity with the same MORPH-297 fixture.
   - Extend `internal/tracker/tracker_test.go` (or focused Jira test files) so `GetIssue`, paginated `ListIssues`, `Search`, `GetIssueEvidence.Normalized`, evidence comment text, `GetFields`, and sprint loading return the exact golden bytes.
4. Prove persistence and merge parity.
   - Extend `internal/cli/sync_import_test.go` for initial spec generation, baseline seed, placeholder refresh, authored-body preservation, and exact body parity.
   - Extend shared merge/diff tests so canonical local/remote descriptions emit no patch and a historical flattened baseline adopts corrected remote text without an outbound description write.
5. Document the compatibility and explicit repair instruction in tracker/import documentation or release notes; do not add a migrator or automatic authored-body rewrite.

## Acceptance Criteria

- **AC-1:** WHEN Jira returns nested bullet or ordered lists THE SYSTEM SHALL preserve every item, source order, starting ordinal, and nesting as valid canonical Markdown.
- **AC-2:** WHEN Jira text carries strong, emphasis, strike, inline-code, or link marks THE SYSTEM SHALL emit deterministic Markdown with the corresponding readable semantics.
- **AC-3:** WHEN Jira returns headings, blockquotes, panels, rules, hard breaks, or language-tagged code blocks THE SYSTEM SHALL preserve their block meaning with deterministic spacing and safe code fences.
- **AC-4:** WHEN Jira returns mentions, status nodes, emoji, inline/block cards, or media THE SYSTEM SHALL emit the specified readable representation when attributes are available and SHALL NOT silently drop representable content.
- **AC-5:** WHEN Jira returns a plain JSON string description THE SYSTEM SHALL return it byte-for-byte unchanged.
- **AC-6:** IF an ADF node type is unknown THEN THE SYSTEM SHALL recursively preserve recognized descendant content in source order without failing the whole description.
- **AC-7:** IF ADF is missing, null, invalid, or partially malformed THEN THE SYSTEM SHALL return deterministic best-effort output without panic and without discarding valid siblings.
- **AC-8:** WHEN the MORPH-297 fixture is rendered repeatedly THE SYSTEM SHALL produce the exact golden Markdown bytes, including populated preconditions/steps, marked Expected/Actual text, status/emoji content, and a fenced Logs block.
- **AC-9:** THE SYSTEM SHALL return the same golden description bytes from Jira `GetIssue`, `ListIssues`, `Search`, and `GetIssueEvidence.Normalized`.
- **AC-10:** THE SYSTEM SHALL return the same canonical bytes from Jira evidence comment text, sprint loading, and `GetFields` diff comparison while retaining raw ADF only in existing evidence fields.
- **AC-11:** WHEN the fixture is initially imported THE SYSTEM SHALL write the exact golden Markdown into the generated spec and the identical description into its shared-field baseline.
- **AC-12:** WHEN refresh encounters an untouched import placeholder THE SYSTEM SHALL insert and baseline the exact canonical description, and WHEN the spec body is authored THE SYSTEM SHALL leave that body unchanged.
- **AC-13:** WHEN canonical local and Jira descriptions are compared THE SYSTEM SHALL emit no description diff, and WHEN only normalization changed from a historical flattened baseline THE SYSTEM SHALL NOT push the flattened value to Jira.
- **AC-14:** THE SYSTEM SHALL use one Jira ADF-to-Markdown implementation for all inbound paths and SHALL NOT retain a second description canonicalizer.
- **AC-15:** THE SYSTEM SHALL preserve existing Jira pagination, custom-field discovery, authentication, raw evidence, comment/attachment retrieval, field ownership, and outbound `textToADF`/write-back behavior.

## Boundaries

- Do not change Hero Code's Swift Markdown renderer or card layout.
- Do not store raw ADF in `Issue.Description`, generated spec bodies, frontmatter, normal broker responses, or sync baselines. Existing explicit `IssueEvidence` raw fields remain lossless by design.
- Do not redesign outbound `textToADF`, `jiraFieldEncode`, `UpdateFields`, `AddComment`, or attachment upload semantics.
- Do not add per-issue requests to broad import/refresh or change Jira pagination, JQL, custom-field discovery, severity, timestamps, or deduplication.
- Do not automatically overwrite locally authored problem/goal content or add a historical migrator.
- Do not implement the adjacent durable evidence sidecar here; that dependent child owns persistence and status contracts.

## Risks

- Markdown escaping and nested-list continuation rules can produce visually plausible but byte-different output; exact goldens and single-renderer parity are the release gate.
- Applying marks in provider order would make semantically equivalent payloads nondeterministic; the renderer must use the fixed canonical order.
- Code logs may contain backticks; dynamic fence length must prevent truncation.
- ADF evolves. Unknown-node recursion must preserve content without claiming unsupported visual semantics.
- The renderer change reveals historical baseline drift. Merge regression tests must prove no stale flattened description is pushed upstream.

## Validation

1. Run `go test ./internal/tracker -run 'ADF|Jira.*(GetIssue|ListIssues|Search|Evidence|Fields|Sprint)' -count=1`.
2. Run `go test ./internal/cli -run 'Import|Refresh|Baseline|Diff|SharedMerge' -count=1`.
3. Run `go test ./internal/tracker ./internal/cli -count=1`, `go test ./...`, `go vet ./...`, and focused `go test -race ./internal/tracker ./internal/cli`.
4. Render the MORPH-297 fixture through each named entry point and compare exact bytes with `internal/tracker/testdata/jira/morph-297-description.md`.
5. Exercise an isolated Jira-like server end to end: import a fixture issue, inspect the spec/baseline, refresh an untouched placeholder, and run dry-run diff to confirm no outbound description patch.
6. Run `hero spec lint jira-adf-description-fidelity-loss`, cold audit, and `hero spec verify jira-adf-description-fidelity-loss`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Nested lists preserve order/nesting | DONE | `TestADFToMarkdown_MORPH297Fixture` covers ordered and nested bullet lists with exact indentation. |
| 2 | Text marks render deterministically | DONE | Fixed-order code/strong/em/strike/link rendering is covered by the exact golden and compatibility cases. |
| 3 | Block nodes preserve meaning | DONE | Headings, blockquotes, rules, hard breaks, and dynamically fenced language code blocks are golden-tested. |
| 4 | Inline semantic nodes remain readable | DONE | Emoji, mention, status, link, inline code, card, date, and media fallbacks are retained by the recursive renderer. |
| 5 | Plain strings pass through byte-for-byte | DONE | `TestADFToMarkdown_CompatibilityAndFallbacks/plain_string_unchanged` verifies literal Markdown and newlines. |
| 6 | Unknown nodes recurse safely | DONE | The MORPH-297 extension node and unknown-leaf compatibility case retain descendant/text content. |
| 7 | Malformed/missing input is best-effort | DONE | Malformed JSON returns an empty value without panic; absent content remains empty. |
| 8 | MORPH-297 equals exact golden | DONE | JSON and Markdown fixtures assert exact canonical bytes including lists, status, emoji, links, and logs. |
| 9 | Core issue read surfaces are byte-identical | DONE | `TestJiraADFMarkdown_IsIdenticalAcrossReadSurfaces` covers GetIssue, ListIssues, and Search. |
| 10 | Evidence/sprint/diff use canonical bytes | DONE | The parity test covers GetFields, IssueEvidence.Normalized, EvidenceComment.Text, and sprint items. |
| 11 | Initial import and baseline match golden | DONE | CLI generation and baseline tests use the exact canonical fixture without truncation or reformatting. |
| 12 | Refresh repairs placeholder, preserves authored body | DONE | Focused refresh tests prove placeholder repair uses canonical bytes and authored Problem content survives. |
| 13 | No false diff or stale flattened push | DONE | `TestMergeSharedFields_RendererUpgradeNeverPushesFlattenedDescription` takes remote canonical text and emits no description push. |
| 14 | Exactly one inbound implementation | DONE | Both prior readers now converge on `adfToMarkdown`; the duplicate field reader was removed. |
| 15 | Existing tracker/write behavior preserved | DONE | Full tracker/CLI suites, focused race suite, and `go vet` pass; outbound `textToADF` is unchanged. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Canonical renderer + MORPH-297 fixtures | DONE | Added recursive renderer plus realistic JSON and exact Markdown golden. |
| 2 | Route all inbound Jira paths | DONE | Issue, search/list, field diff, evidence/comment, and sprint call sites use one implementation. |
| 3 | Adapter parity tests | DONE | One isolated Jira-like server proves exact bytes across all named adapter surfaces. |
| 4 | Import/refresh/merge parity tests | DONE | Generation, baseline, safe refresh, authored-body, and stale-push regressions pass. |
| 5 | Compatibility/repair documentation | DONE | README documents explicit repair and the no-migrator/no-authored-overwrite policy. |

### Exercise-the-feature check

- [x] Exercised the MORPH-297 fixture through isolated Jira GetIssue/ListIssues/Search/GetFields/evidence/comment/sprint paths and through import, baseline, refresh, and merge behavior.

### Excellence Bar self-check

Yes — the implementation is recursive, deterministic, provider-owned, exact-golden tested at every inbound boundary, safe against stale upstream pushes, and retains existing write/pagination behavior.

## Handoff Trail

- 2026-07-21T15:07:50Z — in ← hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: jira-adf-description-fidelity-loss
  peer_spec: hero/jira-adf-description-fidelity-loss
  at_commit: 37be49c2
  reason: "Hero Code exposes Jira descriptions already damaged by the bundled Hero v0.28.0 tracker normalization; MORPH-297 proves nested ADF lists, marks, status nodes, and code fences are discarded before persistence."
