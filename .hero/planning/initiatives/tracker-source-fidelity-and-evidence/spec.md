---
title: "Tracker Source Fidelity — Canonical Jira Markdown and Lazy Evidence Provenance"
slug: tracker-source-fidelity-and-evidence
type: initiative
status: planning
domain: engineering
size: large
priority: critical
created: 2026-07-21
tags: [tracker, jira, adf, evidence, provenance, fidelity, hero-code]
child:
  - jira-adf-description-fidelity-loss
  - lazy-tracker-evidence-sidecar
---

# Tracker Source Fidelity — Canonical Jira Markdown and Lazy Evidence Provenance

## Goal

Make a loaded tracker issue a trustworthy, explainable source: Hero first converts every inbound Jira description to one canonical Markdown representation, then explicitly loads and persists full provider evidence beside the spec without fetching in the background, committing sensitive payloads, or changing tracker write-back ownership.

## Kickoff

Fixes Jira description corruption first, then adds an explicit, lazy full-evidence sidecar so Hero and Hero Code can tell exactly what source material has been loaded for a spec.

**Status:** planning — two implementation-ready children are sequenced; no code has landed.

**Pick up at:** deliver the canonical Jira ADF renderer and its MORPH-297 parity fixture before building the evidence cache on top of `IssueEvidence`.

→ `/deliver jira-adf-description-fidelity-loss`

**Files:** `internal/tracker/jira.go`, `internal/tracker/jira_fields.go`, `internal/tracker/sprint.go`, `internal/cli/sync_evidence.go`, `contracts/`
**Skip:** do not add eager/background evidence fetches, store raw ADF in normal descriptions, or redesign outbound Markdown-to-ADF writes.

## Context

Hero Code proved with Jira issue `MORPH-297` that Hero v0.28.0 discarded nested ADF lists, text marks, status nodes, and code-block structure before the desktop app saw the issue. The current engine has two shallow and inconsistent ADF readers: `extractADFText` supplies `Issue.Description` and sprint/evidence text, while `adfToText` supplies shared-field comparison. The same Jira document can therefore be both lossy and normalized differently by read surface.

Hero already has a second, adjacent capability: `IssueEvidence`, Jira's full-fields/changelog/comments retrieval, paginated comment loading, attachment download, `hero sync evidence <slug>`, and brokered `get_issue(detail=evidence)`. That capability is transient. It prints or returns the evidence and writes attachments under a regenerable global cache, but a spec has no compact declaration that full details were loaded, which provider revision they represent, or whether the private payload is still present and intact.

These are related but independently shippable concerns. Canonical normalization is a correctness prerequisite for evidence persistence because the sidecar's normalized issue and comment text must not capture the same corruption in a more durable location.

## Architecture

The initiative preserves two layers:

1. **Provider-owned canonical normalization.** One deterministic Jira ADF-to-Markdown renderer produces all ordinary description/comment strings used by `GetIssue`, `ListIssues`, `Search`, `GetIssueEvidence.Normalized`, Jira evidence comment text, sprint loading, import/refresh, baselines, and `GetFields` diff comparison. Raw ADF remains available only in the existing lossless evidence fields.
2. **Spec-owned lazy evidence materialization.** One shared evidence-loading service, invoked only by an explicit full-details request, validates or refreshes a private adjacent payload and atomically publishes a compact, non-sensitive manifest. In-process, CLI, and MCP surfaces report the same versioned status; they do not each implement caching rules.

Normal spec discovery and broad tracker refresh remain cheap. They neither require nor create full-evidence sidecars.

## Specs

| Wave | Child | Type | Priority | Size | Outcome |
|---:|---|---|---|---|---|
| 1 | `jira-adf-description-fidelity-loss` | bug | critical / severity critical | medium | One recursive, deterministic Jira ADF-to-Markdown renderer with exact parity across every inbound read and sync comparison path |
| 2 | `lazy-tracker-evidence-sidecar` | enhancement | high | medium | Explicit Jira-first full-evidence loading with validated private payloads, compact manifests, and one shared status contract |

## Dependencies and delivery order

```text
jira-adf-description-fidelity-loss
  → lazy-tracker-evidence-sidecar
```

Wave 1 is a hard dependency: evidence persistence must consume the corrected `IssueEvidence` shape and canonical normalized Markdown rather than freezing the current lossy result. Wave 2 may begin only after Wave 1 verifies adapter/import/refresh/diff parity.

## In-flight overlap watch

The children carry reciprocal `conflicts-with` relations because both touch `IssueEvidence`, Jira evidence parsing, and their tracker/CLI fixtures. The dependency prevents normal sequential delivery from colliding, while the soft mutex also protects manual or parallel pickup from editing the same seams concurrently. Every overlap named here is represented on both child specs; there are no prose-only seams.

## Changes

1. Deliver `jira-adf-description-fidelity-loss` and release the canonical inbound Jira representation.
2. Deliver `lazy-tracker-evidence-sidecar` against that representation, including the versioned status fixture Hero Code consumes.
3. Run each child's cold audit and verification, then publish the compatible Hero release and communicate the contract/version to Hero Code.

## Acceptance Criteria

- **AC-1:** WHEN Jira returns a rich ADF description THE SYSTEM SHALL preserve its readable hierarchy and semantics in canonical Markdown before any normal issue, import, refresh, sprint, evidence-normalized, baseline, or diff consumer receives it.
- **AC-2:** THE SYSTEM SHALL use one deterministic Jira description renderer across every inbound Jira path and SHALL NOT expose raw ADF through ordinary `Issue.Description` or canonical field values.
- **AC-3:** WHEN a user or client explicitly loads full tracker details for a linked spec THE SYSTEM SHALL persist a validated private evidence payload and publish a compact non-sensitive manifest beside that spec.
- **AC-4:** WHILE no explicit full-details load is requested THE SYSTEM SHALL NOT fetch evidence, paginate comments, download attachments, or create evidence sidecars during discovery, import, refresh, serve startup, or background work.
- **AC-5:** WHEN a cached evidence snapshot is checked THE SYSTEM SHALL classify it as current only when contract version, provider, issue ID, provider-native `tracker_updated_at`, and the recomputed payload hash all match.
- **AC-6:** THE SYSTEM SHALL expose one versioned evidence-load status contract through in-process, CLI, and MCP adapters, including a Hero Code consumer fixture.
- **AC-7:** IF a configured provider does not implement full evidence THEN THE SYSTEM SHALL return an explicit unsupported result without creating a manifest or private payload.
- **AC-8:** THE SYSTEM SHALL preserve existing tracker authentication, pagination, sync/import, broker, evidence, comment, attachment, field-ownership, and outbound write-back semantics except for the corrected inbound Markdown representation and additive lazy persistence.

## Boundaries

- No outbound Markdown-to-ADF redesign; `textToADF`, `jiraFieldEncode`, `UpdateFields`, comment creation, and attachment upload retain their current write semantics.
- No eager, automatic, scheduled, serve-start, card-list, import, or refresh evidence fetch.
- No raw ADF or full evidence embedded in `spec.md`, ordinary `Issue.Description`, the committed manifest, MCP status, logs, or diagnostics.
- No Hero Code UI implementation. Hero publishes the engine contract and fixture; Hero Code owns its explicit load gesture and presentation.
- No automatic overwrite of locally authored spec problem/goal bodies while repairing historical imports.
- Jira is the first supported full-evidence provider. GitHub, GitLab, and Linear return structured unsupported status until their adapters deliberately implement the capability.

## Rollout and compatibility

Wave 1 is an inbound normalization correction. Existing plain-string Jira Server/Data Center descriptions remain byte-identical, and existing outbound writes are unchanged. Already-corrupted local descriptions cannot be reconstructed from their local baseline; an explicit re-import/repair from Jira is required, and authored bodies remain protected.

Wave 2 is additive. `hero sync evidence <slug>` keeps its existing full `IssueEvidence` JSON behavior while using the shared loader and adjacent cache; a structured status flag/operation and MCP tool expose the new contract. Existing `.hero/cache/tracker-evidence` content is ignored as legacy regenerable cache and is not migrated. The adjacent committed manifest can travel across clones, but a missing or hash-mismatched private payload is reported as unavailable/stale until an explicit load refetches it.

Hero Code must consume the released contract version, treat status/error codes as authoritative, resolve returned paths relative to the workspace, and never infer freshness merely from manifest existence.

## Risks

- Canonical Markdown changes can make a historical flattened baseline differ from the corrected remote view. The existing three-way merge must take remote safely and must never push flattened local text solely because the renderer changed.
- Markdown list indentation, fence selection, and mark ordering must be exact and deterministic or parity tests will hide surface-specific drift behind visually similar output.
- Full Jira evidence may contain sensitive comments, URLs, identities, filenames, and attachments. Only the private ignored payload may contain it; the manifest and status contract require an allowlist.
- Cache replacement spans metadata and attachments. A manifest-last commit protocol, restrictive permissions, hash validation, and rollback/recovery tests are required to prevent partial snapshots from appearing current.
- A committed manifest may exist on a clone without its ignored payload. Consumers must distinguish declared/load history from locally available current evidence.

## Validation

1. Deliver and verify the ADF child with exact golden parity across renderer, adapter, evidence, sprint, import, refresh, baseline, and diff paths.
2. Deliver and verify the evidence child with first-load, cache-hit, source-update, format-version, provider/issue mismatch, hash-corruption, missing-payload, attachment, atomic-failure, permission, unsupported-provider, cancellation, and no-eager-fetch tests.
3. Run `go test ./internal/tracker ./internal/cli ./internal/serve ./contracts/...` and `go test ./...` after each child; run focused `go test -race` for the evidence store and concurrent/cache cases.
4. Run `hero spec lint`, `hero spec score`, the cold delivery audit, and `hero spec verify` for each child and finally for this initiative.
5. Build a release candidate and exercise the versioned Hero Code fixture against the released binary before tagging.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Rich ADF preserved before every consumer | PARTIAL | Planning complete; Wave 1 implementation and verification pending. |
| 2 | One renderer; no raw ADF in normal fields | PARTIAL | Planning complete; Wave 1 implementation and verification pending. |
| 3 | Explicit load persists private evidence + safe manifest | PARTIAL | Planning complete; Wave 2 implementation and verification pending. |
| 4 | No explicit request means no evidence fetch | PARTIAL | Planning complete; Wave 2 implementation and verification pending. |
| 5 | Five-part cache-current validation | PARTIAL | Planning complete; Wave 2 implementation and verification pending. |
| 6 | Shared versioned status + fixture | PARTIAL | Planning complete; Wave 2 implementation and verification pending. |
| 7 | Unsupported providers create nothing | PARTIAL | Planning complete; Wave 2 implementation and verification pending. |
| 8 | Existing tracker/write contracts preserved | PARTIAL | Cross-child regression verification pending. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Deliver canonical Jira normalization child | PARTIAL | Child is implementation-ready; no code has landed. |
| 2 | Deliver lazy evidence sidecar child | PARTIAL | Child is implementation-ready and blocked on Wave 1. |
| 3 | Audit, verify, release, coordinate consumer | PARTIAL | Runs after both children complete. |

### Exercise-the-feature check

- [ ] Cannot be exercised yet because this initiative is in planning and neither child has been delivered.

### Excellence Bar self-check

No — the design is ready, but production code, end-to-end exercise, cold audits, and release evidence are still required.
