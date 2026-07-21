---
title: "Brokered tracker agent access"
slug: brokered-tracker-agent-access
type: feature
status: completed
domain: engineering
priority: high
size: x-large
created: 2026-07-21
tags: [tracker, broker, credentials, mcp, hero-code, security]
relations:
  - target: tracker-diagnosis-full-ticket-evidence
    kind: related
  - target: integration-config-uses-stable-ids
    kind: related
delivery_method: manual
completed_at: 2026-07-21T23:09:03Z
---

# Brokered tracker agent access

## Goal

Let trusted Hero clients perform bounded tracker reads and writes through a configured connection while the tracker credential remains exclusively inside Hero. Publish one versioned contract shared by the in-process service, CLI, and MCP surfaces, plus a golden consumer fixture Hero Code can validate against the released binary.

## Kickoff

Deliver the approved `brokered-tracker-agent-access` engine contract end to end.

**Start here:** read this spec, then inspect `internal/config/integrations.go`, `internal/tracker`, `internal/cli`, and `internal/serve` before changing code.

**Deliver:** a single broker service with `get_issue`, `search`, `request`, and `cli`; thin CLI and MCP adapters; security and bounds tests; an embedded consumer fixture; cold audit; `hero spec verify`; and a tagged Hero release.

**Preserve:** all existing tracker sync, import, evidence, comment, attachment, pagination, and custom-field behavior.

**Do not:** expose credentials, invoke a shell, accept arbitrary origins, reinterpret provider-native search text, or infer a connection from default/role bindings when multiple tracker connections exist.

→ `.hero/planning/features/brokered-tracker-agent-access/spec.md`

## Problem

Hero already owns configured tracker credentials and can use them for narrow sync and evidence workflows, but clients such as Hero Code cannot safely perform broader tracker work. They either lack the operation, must copy a credential into their own process, or fall back to harness-specific behavior. Existing tracker methods are also project-oriented: Jira and GitHub raw search paths may add the configured project, and some single-issue callers begin from a local spec instead of a provider-native ID.

The missing product boundary is a small, explicit credential broker. It must preserve provider-native intent, enforce the configured tracker origin, classify effects, bound every output, and keep credentials out of all caller-visible and parent-process surfaces.

## Design

### Versioned contract

Add a provider-neutral `tracker-broker/v1` contract containing operation requests, a common response envelope, stable error codes, effect classification (`read`, `write_idempotent`, `write_non_idempotent`), truncation metadata, duration, status/exit codes, pagination cursors, and operation-specific structured results. The same Go types and broker service back direct in-process calls, JSON CLI output, and MCP tools.

Every operation accepts optional `connection_id`. An explicit ID selects only that connection. When omitted, Hero selects only when exactly one configured tracker connection exists; `integrations.default` and role bindings do not resolve ambiguity. Legacy single-tracker configuration remains compatible under a stable synthetic connection ID.

### Operations

1. `get_issue(issue_id, detail)` accepts provider-native identifiers directly. `normalized` returns the provider-neutral issue; `evidence` uses the optional complete-evidence capability and returns an explicit unsupported error where absent. It never requires or resolves a local spec.
2. `search(query, limit, cursor)` sends the query text unchanged to the provider-specific broad-search path. Jira uses its existing search pagination and field parsing without injecting the configured project. Limits and opaque cursors bind pagination to provider, connection, and query.
3. `request(method, relative_path, query, headers, body, output_limit)` resolves a strictly relative path against the selected connection's configured API origin, injects auth internally, blocks credential-bearing caller headers and cross-origin redirects, executes once, and returns a bounded redacted body with an effect derived from the HTTP method.
4. `cli(executable, arguments, stdin, output_limit)` is available only for provider-declared executable identities with documented child-environment credential mapping and host binding. It executes exact argv with no shell, strips ambient credential variables, injects the selected credential only into the child environment, rejects unsafe/auth/config/host-override arguments and credential literals, executes once, and returns bounded redacted streams with a provider-declared effect.

Jira broad work is fully available through `request`; no Jira CLI is advertised until a CLI supports a documented, non-persistent child-only credential contract. GitHub and GitLab may advertise their native CLIs using their documented token and host environment variables. Unsupported provider/operation combinations return a structured capability error rather than weakening the security contract.

### Security and bounds

Reject absolute and scheme-relative URLs, userinfo, fragments, backslashes/control characters, host-bearing paths, cross-origin redirects, auth/cookie/api-key headers, unknown executables, executable paths, shell escapes, host overrides, credential-display/config commands, and literal configured credentials. Redact exact, encoded, and authorization-wrapped credential forms from bodies, streams, errors, and diagnostics. Never mutate the parent environment.

Use explicit maximum item, byte, redirect, and duration bounds. Cancellation propagates to HTTP requests and child processes. Non-idempotent operations are attempted at most once and are never automatically retried.

## Changes

1. Add the versioned broker contract, stable errors/effects, and embedded Hero Code consumer fixture.
2. Add unambiguous tracker-connection resolution and credential-safe provider metadata without changing existing delivery selection.
3. Add the in-process broker service and provider-specific full-key/broad-search adapters, reusing Jira pagination and parsing.
4. Add hardened same-origin HTTP request brokerage with internal auth, redirect controls, redaction, effect classification, cancellation, and bounds.
5. Add provider-declared exact-argv CLI brokerage with child-only credentials, host binding, policy validation, redaction, cancellation, and bounds.
6. Add `hero tracker` JSON commands and equivalent MCP tools as thin adapters over the same broker service.
7. Add focused contract, config, provider, broker, CLI, MCP, security, and consumer-fixture tests plus release-facing documentation.

## Acceptance Criteria

- AC-1: WHEN a caller requests a full provider-native issue ID, THE SYSTEM SHALL fetch it directly without requiring a local spec or constraining it to the configured project.
- AC-2: WHEN a caller submits a provider-native search query, THE SYSTEM SHALL preserve the query text exactly and return an explicitly bounded page with an opaque continuation cursor.
- AC-3: WHEN Jira search targets projects outside the configured project, THE SYSTEM SHALL reuse Jira pagination and parsing without injecting a configured-project clause.
- AC-4: WHEN exactly one tracker connection is configured and `connection_id` is omitted, THE SYSTEM SHALL select it; IF zero or multiple tracker connections exist, THEN THE SYSTEM SHALL return a stable selection error without using defaults or roles.
- AC-5: WHEN a brokered HTTP request is valid, THE SYSTEM SHALL resolve only a relative path against the configured origin, inject authentication internally, execute once, and return a bounded redacted response with an effect classification.
- AC-6: IF a brokered HTTP request contains an absolute URL, scheme-relative path, userinfo, fragment, host confusion, caller auth/cookie headers, or a cross-origin redirect, THEN THE SYSTEM SHALL reject it before sending credentials to that origin.
- AC-7: WHEN a declared provider CLI operation is valid, THE SYSTEM SHALL execute its exact argv directly without a shell and SHALL inject credentials only into the child environment with provider-declared host binding and effect classification.
- AC-8: IF a CLI request uses an unknown/path-qualified executable, auth/config/credential-display command, host override, shell escape, or credential literal, THEN THE SYSTEM SHALL reject it without starting a child process.
- AC-9: THE SYSTEM SHALL return the same stable `tracker-broker/v1` JSON contract from in-process, CLI, and MCP entry points, including provider, connection ID, effect, status or exit code, bounded output, truncation, duration, cursor, and structured error fields.
- AC-10: THE SYSTEM SHALL NOT expose a credential through argv, response payloads, stdout/stderr, errors, logs, diagnostics, fixtures, parent environment mutation, or failed-operation artifacts.
- AC-11: WHEN callers cancel work or configured byte, item, redirect, or time limits are reached, THE SYSTEM SHALL stop predictably and report bounded structured state.
- AC-12: WHEN an operation is classified non-idempotent, THE SYSTEM SHALL make at most one provider attempt and SHALL NOT automatically retry it.
- AC-13: WHEN existing sync, import, evidence, comment, attachment, pagination, deduplication, or custom-field tests run, THE SYSTEM SHALL preserve their behavior and compatibility.
- AC-14: WHEN Hero Code validates the released binary, THE SYSTEM SHALL provide a versioned golden consumer fixture covering every operation, effects, errors, truncation, and cursors without any real credential.

## Validation

- Exercise direct full-key Jira lookup and evidence lookup without a local spec.
- Assert Jira receives byte-for-byte cross-project JQL and that native continuation tokens round-trip through opaque bound cursors.
- Cover same-origin requests, lookalike hosts, absolute/scheme-relative/userinfo/fragment inputs, auth headers, same-origin redirects, and cross-origin redirects with credential-canary servers.
- Execute a fake allowed CLI to capture argv and child environment; prove exact argv, no shell, child-only token/host variables, stripped ambient credentials, cancellation, and bounded streams.
- Run credential-canary tests across success, provider errors, redirects, truncation, malformed output, command failures, and diagnostics.
- Cover item, byte, redirect, and time limits, cursor mismatch, cancellation, and single-attempt non-idempotent HTTP/CLI writes.
- Decode the embedded fixture from both the Go contract and released-binary CLI surface.
- Run focused and full `go test ./...`, race-sensitive tests where practical, `hero docs check`, cold delivery audit, and `hero spec verify brokered-tracker-agent-access`.

## Boundaries

- This is a credential broker for configured tracker connections, not a general outbound network proxy or arbitrary command runner.
- It does not add credential migration, secret export, OAuth refresh, tracker configuration mutation, or local-spec discovery.
- It does not redesign Jira pagination, `nextPageToken`, custom-field discovery, Severity, sync/import, evidence, comment, or attachment behavior.
- It does not add Hero Code UI; Hero Code owns the dependent desktop tools under its `brokered-tracker-agent-access` initiative.

## Compatibility Decision for Hero Code

Hero Code consumes `tracker-broker/v1`, should always send `connection_id` once the user has selected one, must treat `effect` as authoritative, must use `next_cursor` opaquely, and must render structured broker errors without parsing prose. Absence of `connection_id` is convenience-only for a truly single-tracker workspace. Provider capabilities are explicit: a missing CLI capability is not permission to invoke a provider executable directly.

## Completion Ledger

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Full native issue ID, no local spec/project constraint | DONE | `Broker.GetIssue` accepts Jira/Linear native IDs and cross-project `owner/repo#id` GitHub/GitLab IDs directly; the release-candidate binary fetched `ACME-101` while the configured project was deliberately `CONFIGURED`. |
| 2 | Exact native search plus bounded cursor | DONE | Provider page adapters preserve the exact query, enforce 1–100 items, and bind opaque cursors to contract version/provider/connection/query hash. |
| 3 | Cross-project Jira search parity | DONE | Jira's existing parser and `nextPageToken` loop now share `searchIssuesPage`; broker tests assert byte-for-byte cross-project JQL and native cursor round-trip, and the release-candidate binary returned a bounded cross-project page. |
| 4 | Unambiguous connection selection only | DONE | `Config.ResolveTrackerConnection` ignores default/roles, selects exactly one tracker, rejects zero/multiple/confluence selections, and preserves legacy single-tracker compatibility as `legacy`. |
| 5 | Same-origin internally authenticated HTTP | DONE | `Broker.Request` resolves only against the configured origin, injects provider auth after validation, follows bounded same-origin redirects, classifies effects, and returns status plus bounded redacted body. |
| 6 | Reject URL/header/redirect confusion | DONE | Tests cover absolute/scheme-relative/userinfo/fragment/backslash paths, Host/forwarded/auth headers, lookalike hosts, cross-origin redirects, non-idempotent redirects, and the five-redirect bound. |
| 7 | Exact-argv child-only credential CLI | DONE | `exec.CommandContext` receives exact argv with no shell; provider declarations bind `gh`/`glab`, token and host env, effect, isolated cwd/config, disabled prompts/pagers, and stripped ambient tracker credentials. |
| 8 | Reject unsafe CLI shapes | DONE | Tests reject path/unknown executables, auth/config/alias/extension/secret commands, host/URL/header overrides, shell/repo execution, filesystem arguments, and raw/encoded credential literals before lookup. |
| 9 | One v1 contract across all surfaces | DONE | `contracts/trackerbroker` owns the request/response types used directly by `Broker`, `hero tracker broker`, and all four `hero_tracker_*` MCP tools; CLI/MCP tests decode the same envelope. |
| 10 | Credential never exposed | DONE | `config.Secret`, child-only env construction, isolated CLI state, bounded lookahead redaction, and canary tests cover raw/encoded/basic-auth forms in bodies, stdout, stderr, errors, redirects, and truncation boundaries without parent env mutation. |
| 11 | Cancellation and bounds | DONE | Search items, query/header/body/stdin/argv, output bytes, redirects, errors, credentials, and 30-second execution are bounded; HTTP and child-process cancellation tests pass. |
| 12 | Non-idempotent attempt once | DONE | Hero has no broker retry loop; POST failure and 307 redirect tests prove one provider attempt and no redirected second write. CLI launches one process only. |
| 13 | Existing tracker compatibility | DONE | Existing Jira pagination now delegates to its one-page primitive without changing callers; focused adapter packages, full `go test ./...`, and focused race tests all pass. |
| 14 | Released consumer fixture | DONE | `contracts/trackerbroker/testdata/v1/consumer-fixture.json` is embedded and emitted by `hero tracker contract`; the release-candidate binary decoded every operation/error/truncation example. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Contract and consumer fixture | DONE | Added `contracts/trackerbroker` v1 types, stable effects/errors, embedded golden fixture, and fixture validation. |
| 2 | Connection resolution | DONE | Added explicit/unambiguous broker selection with secret-preserving metadata and legacy compatibility. |
| 3 | Broker and broad provider adapters | DONE | Added the in-process broker, direct full IDs, Jira/GitHub/GitLab/Linear broad page adapters, and opaque cursor binding. |
| 4 | HTTP brokerage | DONE | Added strict relative-origin resolution, internal provider auth, header/redirect defenses, cancellation, effects, bounds, and redaction. |
| 5 | CLI brokerage | DONE | Added provider-declared `gh`/`glab` execution with exact argv, isolated state/cwd, child-only credentials, host binding, policy validation, effects, and bounded streams. |
| 6 | CLI and MCP adapters | DONE | Added `hero tracker broker`, `hero tracker contract`, and four MCP tools as thin shared-service adapters. |
| 7 | Tests and release docs | DONE | Added contract/config/provider/security/CLI/MCP tests, `docs/contracts/tracker-broker-v1.md`, contract index updates, and MCP count update. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: a freshly built `/private/tmp/hero-broker-v1` loaded split committed/local Jira configuration, fetched full-key `ACME-101`, ran cross-project paginated JQL, performed a generic same-origin request against the isolated mock tracker, and emitted the embedded consumer fixture.

### Excellence Bar self-check

- [x] Yes — credentials remain inside Hero, every capability is explicit and bounded, non-idempotent work is single-attempt, and all surfaces share one contract and implementation.
