# Tracker Evidence v1

`tracker-evidence/v1` is Hero's provider-neutral contract for explicitly
loading the complete tracker evidence linked to a local spec. Hero selects the
configured connection and injects its credential internally. Consumers receive
bounded status and provenance; they never receive credentials through this
contract.

## Consumer fixture

The released binary exposes the canonical fixture bundle:

```sh
hero tracker contract tracker-evidence
```

The same bytes are embedded by `contracts/trackerevidence.ConsumerFixture`.
Consumers must branch on `version`, `status`, and `error.code`, tolerate unknown
fields, and reject unsupported major versions.

## Request

```json
{
  "spec_slug": "morph-297",
  "connection_id": "jira-main",
  "include_attachments": true,
  "force_refresh": false
}
```

`spec_slug` is required. `connection_id` may be omitted only when exactly one
tracker connection is selectable. Attachments default to enabled. Force refresh
bypasses a valid snapshot but remains an explicit foreground operation.

## Status

Every in-process, CLI status, and MCP result uses the same envelope. Success
states are `fetched`, `refreshed`, and `current`; terminal capability and
availability states are `unsupported` and `unavailable`. The response includes
the provider, selected connection ID, spec and issue identity, exact native
tracker update timestamp, content hash, workspace-relative paths, counts, cache
hit flag, and an optional structured error.

The status intentionally excludes issue content, raw provider fields, comments,
changelog, people, URLs, attachment names and bytes, credentials, and auth
headers.

## Storage and freshness

An explicit load writes two adjacent artifacts beside `spec.md`:

```text
tracker-evidence.json              tracked allowlisted manifest
.tracker-evidence/evidence.json    ignored complete IssueEvidence
.tracker-evidence/attachments/     ignored downloaded attachments
```

The private directory is created with mode `0700`; files use `0600`. The
manifest contains only contract version, provider, issue ID, the exact native
`tracker_updated_at`, a whole-snapshot SHA-256, private evidence path, counts,
and the source retrieval timestamp.

A snapshot is `current` only when its version, provider, issue ID, non-empty
valid tracker timestamp, payload, required attachments, and whole-snapshot hash
all validate. Missing or malformed native timestamps are preserved exactly but
never produce a cache hit. Hero never substitutes import time, load time, local
file mtime, or the local clock for tracker activity.

Candidate private state is built restrictively and published under a per-spec
lock. The manifest is renamed last as the commit marker. Failed or cancelled
loads preserve the previous committed snapshot.

## Surfaces

- In-process: `tracker.NewEvidenceLoader(root).Load(ctx, request)`.
- CLI status: `hero sync evidence <slug> --status`; `--force` refreshes and
  `--no-attachments` skips attachment downloads.
- CLI compatibility: without `--status`, `hero sync evidence <slug>` still
  prints the complete legacy `IssueEvidence` JSON from the validated snapshot.
- MCP: `hero_tracker_load_evidence` accepts the request shape and returns status
  only.

Jira is the only full-evidence provider in v1. Other tracker providers return
`unsupported` with `unsupported_provider` and create no files. Normal import,
refresh, search, queue, discovery, server startup, and background paths never
trigger an evidence load.
