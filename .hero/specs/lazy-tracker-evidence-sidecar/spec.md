---
title: "Lazy Tracker Evidence Sidecar — Explicit Full-Details Provenance"
slug: lazy-tracker-evidence-sidecar
type: enhancement
status: completed
domain: engineering
surface: hero-engine-tracker
parent: tracker-source-fidelity-and-evidence
size: medium
priority: high
created: 2026-07-21
depends-on: [jira-adf-description-fidelity-loss]
tags: [tracker, evidence, provenance, cache, jira, attachments, mcp, hero-code]
relations:
  - target: jira-adf-description-fidelity-loss
    kind: conflicts-with
delivery_method: manual
completed_at: 2026-07-22T01:41:23Z
---

# Lazy Tracker Evidence Sidecar — Explicit Full-Details Provenance

## Context

Hero already has the raw capability needed for deep issue inspection:

- `tracker.IssueEvidence` contains a normalized issue plus provider raw fields, field names, changelog, paginated comments, attachment metadata, and explicit omissions.
- Jira implements `EvidenceTracker.GetIssueEvidence` and same-origin attachment download.
- `hero sync evidence <slug>` explicitly fetches that envelope and downloads attachments into `.hero/cache/tracker-evidence/<slug>/attachments`.
- brokered `get_issue(detail=evidence)` safely exposes evidence for arbitrary issue IDs without sharing credentials.

What is missing is durable, per-spec loaded state. The command's JSON is transient and the global cache is regenerable; nothing beside a spec says which provider revision was loaded, whether the local private payload still matches it, or whether Hero Code can reuse it without a second full fetch.

This child follows the canonical ADF renderer because persisted `IssueEvidence.Normalized.Description` and evidence comment text must already have one trustworthy representation. It has a reciprocal `conflicts-with` seam with that bug because both change `IssueEvidence`, Jira evidence normalization, and shared tracker/evidence tests; `depends-on` provides delivery order and the mutex prevents accidental concurrent pickup.

## Goal

On an explicit full-details request for a tracker-linked spec, fetch or reuse current Jira evidence, persist the complete sensitive payload and attachments in an ignored adjacent directory, atomically publish a compact non-sensitive adjacent manifest, and return one versioned structured status through in-process, CLI, and MCP surfaces—without eager/background loading or provider-specific cache logic in clients.

## Kickoff

Adds explicit, lazy full-ticket evidence loading beside a spec, with a committed safe manifest and a private validated payload reused across Hero and Hero Code.

**Status:** planning — the storage/status contract is designed and waits on canonical Jira ADF normalization.

**Pick up at:** define `tracker-evidence/v1`, build the shared loader/cache validator, then adapt `hero sync evidence` and MCP to it.

→ `/deliver lazy-tracker-evidence-sidecar`

**Files:** `contracts/trackerevidence/`, `internal/tracker/evidence_store.go`, `internal/cli/sync_evidence.go`, `internal/serve/mcp_tools_tracker.go`, `internal/cli/init.go`
**Skip:** no eager/background fetch, no committed evidence payload, and no provider stubs that pretend unsupported evidence is empty success.

## Problem

Manifest existence alone cannot prove that evidence is usable: the ignored payload may be missing on another clone, corrupt, from a different provider/issue, written by an incompatible format, or stale relative to the tracker's native update timestamp. Conversely, refetching the full issue, all comment pages, and every attachment on each card expansion wastes requests and causes private files and committed metadata to churn.

The persistence layer also cannot live only in the CLI. Hero Code needs the same answer through MCP, and future in-process consumers must not reimplement provider selection, timestamp checks, hashing, file permissions, attachment path safety, atomic publication, or unsupported-provider behavior.

## Design

### Versioned shared contract

Add public `contracts/trackerevidence` types with `Version = "tracker-evidence/v1"`, an embedded consumer fixture, and one service request:

```json
{
  "spec_slug": "morph-297-bulk-start-of-discovered-vms-none-starts",
  "connection_id": "jira-main",
  "include_attachments": true,
  "force_refresh": false
}
```

`connection_id` is optional and follows existing integration selection: omission succeeds only when the spec/config resolves one unambiguous tracker connection. `include_attachments` defaults true to preserve `hero sync evidence`; `force_refresh` bypasses a valid cache but remains an explicit foreground action.

The shared result is bounded metadata, never the evidence body:

```json
{
  "version": "tracker-evidence/v1",
  "status": "current",
  "provider": "jira",
  "connection_id": "jira-main",
  "spec_slug": "morph-297-bulk-start-of-discovered-vms-none-starts",
  "issue_id": "MORPH-297",
  "tracker_updated_at": "2026-07-20T17:42:19.123-0600",
  "content_sha256": "<64 lowercase hex>",
  "manifest_path": ".hero/planning/bugs/.../tracker-evidence.json",
  "evidence_path": ".hero/planning/bugs/.../.tracker-evidence/evidence.json",
  "attachment_count": 2,
  "omission_count": 0,
  "cache_hit": true,
  "error": null
}
```

Allowed statuses are `fetched`, `refreshed`, `current`, `unsupported`, and `unavailable`. Errors use stable codes (`spec_not_found`, `tracker_unlinked`, `ambiguous_connection`, `unsupported_provider`, `provider_unavailable`, `invalid_manifest`, `payload_missing`, `payload_corrupt`, `cancelled`, `write_failed`) plus safe message and retryability. `unsupported` and `unavailable` are structured results, not empty evidence. Responses and logs exclude raw fields, comments, filenames, URLs, attachment bytes, tokens, auth headers, and child environment values.

### Adjacent storage contract

For a folder spec at `<spec-dir>/spec.md`, publish:

```text
<spec-dir>/
  spec.md
  tracker-evidence.json              # compact, non-sensitive, tracked
  .tracker-evidence/                 # ignored, mode 0700
    evidence.json                    # complete IssueEvidence, mode 0600
    attachments/                     # mode 0700
      <safe generated filename>      # mode 0600
```

The tracked `tracker-evidence.json` is an allowlisted v1 manifest containing only:

- contract version;
- provider;
- issue ID;
- exact provider-native `tracker_updated_at`;
- snapshot `content_sha256`;
- workspace-relative evidence path;
- attachment and omission counts;
- source retrieval timestamp from the stored envelope.

It must not contain connection credentials, auth material, raw fields, comments, changelog, title/description, people, URLs, attachment names, local absolute paths, or errors. Connection ID is returned in the transient status but omitted from the committed manifest.

The snapshot hash covers the exact `evidence.json` bytes plus every attachment's relative generated path and bytes in lexical path order, domain-separated by `tracker-evidence/v1`. Attachment filenames on disk use stable safe generated names (attachment ID/hash plus a sanitized extension), while the original filename remains only inside ignored evidence JSON. All persisted `LocalPath` values are workspace-relative and path traversal is rejected.

Add `.hero/**/.tracker-evidence/` to Hero's managed root `.gitignore` block and verify init/upgrade idempotency. The manifest is intentionally not ignored so loaded provenance travels; the sensitive payload never stages.

### Explicit load and cache-current algorithm

Implement one context-aware service in the engine (not Cobra or MCP):

1. Resolve the spec, tracker ID, and selected connection. Reject missing/unlinked/ambiguous state before filesystem writes.
2. Require the provider to implement `EvidenceTracker`. In v1 only Jira is supported; other providers return `unsupported`/`unsupported_provider` and create nothing.
3. Fetch the provider's normalized issue metadata once to obtain the exact `UpdatedAt`. This is a cheap foreground freshness check, not a full evidence fetch.
4. If not forced, read the manifest and private payload. Classify `current` only when all five checks pass: contract version, provider, issue ID, non-empty exact `tracker_updated_at`, and recomputed whole-snapshot SHA-256. Required attachment files must also exist for an attachment-inclusive request.
5. On a valid hit, return `current` without calling `GetIssueEvidence`, downloading attachments, or rewriting any file. Repeated hits therefore leave both manifest mtime/content and payload untouched.
6. Otherwise call `GetIssueEvidence`, download requested attachments through the provider's existing safe method, persist the snapshot, and return `fetched` when no prior manifest existed or `refreshed` when replacing stale state.
7. If `UpdatedAt` is missing, malformed, or empty, never claim `current`; refetch on each explicit load and report the provider value unchanged in status/manifest when available.
8. If the freshness check or refetch is unavailable, retain the last validated snapshot, return `unavailable` with its paths/hash and `cache_hit=false`, and do not present it as current.

The arbitrary-ID broker operation `get_issue(detail=evidence)` remains transient because it has no required local spec directory. Only the explicit per-spec loader persists a sidecar.

### Atomic and private writes

Build a complete candidate in a restrictive sibling temporary directory. Serialize stable indented JSON, close files, and calculate the whole-snapshot hash before publication. Publish the private directory with a backup/rollback sequence under a per-spec process lock, then write `tracker-evidence.json.tmp` and atomically rename the manifest last as the commit marker. On any failure before manifest commit, restore the prior private directory and manifest; on startup/load, clean or recover stale temp/backup directories deterministically.

Use `0700` for private/temp/backup directories and `0600` for evidence/attachments/manifests before rename (the tracked manifest may later follow repository permissions, but Hero creates it restrictively). Never use caller filenames as paths. Cancellation must stop network/download/write work, remove temporary state, and preserve the last committed snapshot.

### Surface adapters and compatibility

- **In-process:** expose one loader interface returning the versioned status and, for trusted internal callers, a method to open/decode the validated evidence snapshot.
- **CLI:** refactor `hero sync evidence <slug>` to use the service. Default output remains the existing full `IssueEvidence` JSON for compatibility, loaded from the validated adjacent snapshot; add `--status` for the versioned status envelope and retain `--no-attachments`. Add `--force` for explicit refresh.
- **MCP:** add `hero_tracker_load_evidence` with the shared request shape and status-only response. Register it in definitions and dispatch; do not return the private payload through this status tool.
- **Fixture:** embed a `tracker-evidence/v1` `current` result and matching safe manifest fixture under `contracts/trackerevidence/testdata/v1/` for Hero Code decoding.

No background watcher, `hero serve` startup, broad import, normal broker get/search, spec discovery, queue rendering, or card listing calls the loader.

## Changes

1. Define the shared status/manifest contract and consumer fixtures.
   - Add `contracts/trackerevidence/contract.go`, validation/JSON tests, and v1 fixture files.
2. Add the shared evidence loader/store.
   - Reuse `IssueEvidence`, `EvidenceTracker`, existing integration resolution, Jira comment pagination, and same-origin attachment download.
   - Implement five-part freshness validation, whole-snapshot hashing, safe paths, restrictive atomic publication/rollback, cancellation, and private snapshot decoding.
3. Refactor CLI evidence loading without breaking default output.
   - Update `internal/cli/sync_evidence.go` and tests for `--status`, `--force`, `--no-attachments`, adjacent paths, cache hits, and compatibility JSON.
4. Add the MCP adapter.
   - Update `internal/serve/mcp_tools_def.go`, `mcp_dispatch.go`, and `mcp_tools_tracker.go`; test exact request decoding, statuses, and absence of evidence/secrets.
5. Add and validate the managed ignore rule.
   - Update `internal/cli/init.go` plus init/upgrade/gitignore tests to ignore private sidecars at every spec depth while keeping manifests trackable.
6. Add end-to-end and failure coverage.
   - Exercise first load, repeated hit, source update, force refresh, missing/corrupt payload, format/provider/issue mismatch, attachment change, unsupported providers, unavailable tracker with prior snapshot, cancellation, atomic failures, permissions, and concurrent loads.

## Acceptance Criteria

- **AC-1:** WHEN an explicit full-details load targets a Jira-linked spec with no current sidecar THE SYSTEM SHALL fetch full `IssueEvidence`, paginate comments, optionally download attachments, and publish the adjacent private snapshot plus safe manifest.
- **AC-2:** WHEN contract version, provider, issue ID, exact non-empty `tracker_updated_at`, and recomputed snapshot hash all match THE SYSTEM SHALL return `current` without a full evidence fetch, attachment download, or file rewrite.
- **AC-3:** IF any freshness component mismatches or a required private file is missing/corrupt THEN THE SYSTEM SHALL refetch explicitly and return `refreshed` only after an atomically complete replacement is committed.
- **AC-4:** WHEN repeated explicit loads hit a current cache THE SYSTEM SHALL leave manifest and payload bytes and modification times unchanged.
- **AC-5:** THE SYSTEM SHALL keep full evidence, raw ADF, comments, changelog, URLs, people, filenames, and attachment bytes only in the ignored private sidecar and SHALL publish only the manifest allowlist.
- **AC-6:** THE SYSTEM SHALL create private directories with mode `0700`, files with mode `0600`, safe generated paths, and atomic manifest-last publication with rollback on failure or cancellation.
- **AC-7:** WHEN a load includes attachments THE SYSTEM SHALL include their sorted paths and bytes in `content_sha256`, record download omissions explicitly inside private evidence, and never treat missing required files as current.
- **AC-8:** IF the provider-native update timestamp is missing or malformed THEN THE SYSTEM SHALL never claim the cache is current and SHALL retain the exact provider value without substituting load time, file mtime, or local clock time.
- **AC-9:** IF the tracker is unavailable while a previously validated snapshot exists THEN THE SYSTEM SHALL preserve it and return structured `unavailable` status without representing stale evidence as current.
- **AC-10:** IF the selected provider does not implement full evidence THEN THE SYSTEM SHALL return `unsupported` with code `unsupported_provider` and SHALL NOT create a manifest or private directory.
- **AC-11:** WHILE no explicit full-details load is requested THE SYSTEM SHALL perform no evidence fetch, comment pagination, attachment download, manifest write, or private sidecar creation from import, refresh, serve, discovery, queue, normal broker, or background paths.
- **AC-12:** THE SYSTEM SHALL return equivalent `tracker-evidence/v1` status through the shared in-process service, `hero sync evidence --status`, and `hero_tracker_load_evidence` MCP tool.
- **AC-13:** WHEN `hero sync evidence <slug>` is used without `--status` THE SYSTEM SHALL preserve its existing full `IssueEvidence` JSON behavior while sourcing it from the validated adjacent snapshot.
- **AC-14:** THE SYSTEM SHALL provide Hero Code fixtures for every success status plus unsupported, unavailable, missing, corrupt, cancelled, and validation errors, with unknown-field-tolerant versioned JSON.
- **AC-15:** THE SYSTEM SHALL preserve existing brokered arbitrary-ID evidence reads, Jira authentication, same-origin attachment controls, comment pagination, normalized/raw evidence semantics, and credential non-disclosure.

## Boundaries

- No eager/background/full-details fetch during imports, refreshes, server startup, browsing, search, queue projection, or card rendering.
- No raw evidence or attachment content in Git, `spec.md`, manifest, status responses, logs, diagnostics, or test artifacts containing live data.
- No persistence for arbitrary issue IDs without a local linked spec; broker `detail=evidence` remains transient.
- No GitHub, GitLab, or Linear evidence implementation in v1. They return explicit unsupported status rather than fabricated empty evidence.
- No evidence-database, daemon, TTL scheduler, eviction policy, cloud upload, encryption/key management, or cross-machine private payload sync.
- No ADF conversion changes in this child; it consumes the completed canonical renderer.

## Rollout and compatibility

The contract is additive and versioned `tracker-evidence/v1`. The existing CLI's default JSON shape remains available; structured status is opt-in through `--status` and is the MCP/in-process consumer contract. The old `.hero/cache/tracker-evidence` directory is ignored and left in place; the first explicit load writes the adjacent v1 snapshot without migration.

Hero-managed gitignore updates on init/upgrade. A manifest may travel to another clone without its private sidecar; hash/path validation reports `payload_missing`/`unavailable` until the user explicitly loads evidence there. Hero Code must treat `status`, contract version, and error code as authoritative and must not infer availability from the manifest alone.

## Risks

- Evidence may contain sensitive customer or employee data. A field allowlist for manifest/status plus ignore and permission tests are mandatory.
- Multi-file replacement can expose partial state after interruption. The lock, candidate directory, rollback, manifest-last commit, and recovery tests are core behavior, not polish.
- Jira attachment sets may change without a distinct attachment timestamp; using the issue's native `updated` value is the provider contract, while force refresh remains the escape hatch.
- A missing provider timestamp prevents safe cache hits and can increase explicit-load traffic; correctness wins over inventing freshness.
- Concurrent Hero/CLI/MCP requests can race. Per-spec locking and post-lock revalidation must collapse duplicate full fetches.

## Validation

1. Run `go test ./contracts/trackerevidence ./internal/tracker ./internal/cli ./internal/serve -count=1` and focused `go test -race` for loader/concurrency packages.
2. Use failure-injection filesystem/provider tests for every atomic stage, cancellation point, hash mismatch, missing attachment, and stale temp/backup recovery; assert prior committed state survives.
3. Use `git check-ignore` against nested initiative/bug/feature paths to prove `.tracker-evidence/` is ignored and `tracker-evidence.json` is not.
4. Decode every embedded Hero Code fixture and assert status responses contain no raw evidence, URLs, identities, filenames, or credential canaries.
5. Exercise an isolated Jira-like server: first load returns `fetched`, second returns `current` without comment/attachment calls or mtime changes, changed `updated` returns `refreshed`, and default CLI still prints the stored `IssueEvidence` envelope.
6. Run `go test ./...`, `go vet ./...`, `hero docs check`, `hero spec lint lazy-tracker-evidence-sidecar`, cold audit, and `hero spec verify lazy-tracker-evidence-sidecar`.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | First explicit Jira load publishes sidecar | DONE | `EvidenceLoader.Load` fetches `IssueEvidence`, paginated Jira comments and optional attachments, then publishes adjacent state; loader and Jira adapter suites pass. |
| 2 | Five-part valid cache returns current | DONE | Snapshot validation checks version/provider/issue/timestamp/hash and the cache-hit test proves no full fetch or download. |
| 3 | Invalid cache refetches atomically | DONE | Payload, attachment, version, provider, issue and timestamp mismatches all refetch and replace only after publication succeeds. |
| 4 | Cache hits never churn files | DONE | Test asserts exact bytes and mtimes are unchanged on `current`. |
| 5 | Sensitive payload remains private | DONE | Manifest/status allowlists and credential-canary tests exclude evidence, identities, URLs, filenames and auth; private path is ignored. |
| 6 | Restrictive atomic writes + rollback | DONE | Candidate/backup/manifest-last flow uses 0700/0600; injected backup/candidate/manifest failures, mid-publication cancellation, and crash recovery preserve committed state. |
| 7 | Hash covers attachments and omissions | DONE | Domain-separated hash includes exact evidence bytes plus sorted attachment paths/bytes; missing attachment and corruption force refresh. |
| 8 | Missing timestamp never becomes current | DONE | Invalid provider-native timestamps are retained verbatim and force every explicit load to refresh. |
| 9 | Provider unavailable preserves stale snapshot | DONE | Safe `unavailable` result retains validated hash/paths without claiming a cache hit or changing files. |
| 10 | Unsupported provider creates nothing | DONE | Non-Jira providers return `unsupported_provider` before credential or adapter access and leave no sidecar. |
| 11 | No explicit load means no fetch/write | DONE | Loader construction exists only in the explicit CLI and MCP handlers; import, refresh, serve startup, discovery, queue and broker paths are unchanged. |
| 12 | In-process/CLI/MCP status parity | DONE | All adapters use `contracts/trackerevidence.Request` and `Status`; focused CLI/MCP tests pass. |
| 13 | Default CLI evidence JSON remains compatible | DONE | Default command decodes the validated snapshot and prints the existing `IssueEvidence`; `--status` is opt-in. |
| 14 | Hero Code fixtures cover status/errors | DONE | Embedded fixture contains every v1 state/error and is available from the released binary with `hero tracker contract tracker-evidence`. |
| 15 | Existing broker/security/evidence behavior preserved | DONE | Full `go test ./...`, focused race tests, vet, docs and lint pass, including existing Jira/broker security tests. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Shared contract + consumer fixtures | DONE | Added `contracts/trackerevidence`, schema tests, fixtures and binary fixture output. |
| 2 | Shared loader/store | DONE | Added provider-neutral loader, freshness validator, private atomic store and trusted snapshot reader. |
| 3 | CLI compatibility + status adapters | DONE | Shared service now backs default output, `--status`, `--force`, `--no-attachments` and connection selection. |
| 4 | MCP adapter | DONE | Registered `hero_tracker_load_evidence` with shared status-only request/response behavior. |
| 5 | Managed ignore rule | DONE | Added recursive private-sidecar rule to root and managed gitignore plus idempotency tests. |
| 6 | End-to-end/failure coverage | DONE | Added real Jira-like CLI exercise plus cache, corruption, attachment-byte, omission, timestamp, in-flight cancellation, publication-stage, concurrency and surface tests. |

### Exercise-the-feature check

- [x] `go run ./cmd/hero tracker contract tracker-evidence` emits the released-binary consumer fixture.
- [x] Focused service/CLI/MCP suites exercise fetched, current, refreshed, unsupported and unavailable outcomes.
- [x] An isolated authenticated Jira-like server exercises first load, no-churn cache hit, source update, comments, attachment download, and legacy CLI JSON.
- [x] `git check-ignore -v --no-index` proves nested `.tracker-evidence/evidence.json` is ignored while adjacent `tracker-evidence.json` is trackable.

### Excellence Bar self-check

Yes — the explicit-load behavior is shared rather than adapter-duplicated, privacy and atomicity failure paths are exercised, CLI compatibility is retained, the consumer contract ships from the binary, and the full repository validation suite passes. Cold audit and Hero verification remain the independent closing gates.

## Handoff Trail

- 2026-07-22T01:54:10Z — out → hero-code (peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3)
  mode: async-drop
  originating_spec: lazy-tracker-evidence-sidecar
  peer_spec: hero-code/lazy-tracker-evidence-sidecar
  at_commit: 3f675c6
