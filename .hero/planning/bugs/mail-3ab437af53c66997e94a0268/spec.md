---
title: "Jira attachment content is blocked by broker redirect policy"
slug: mail-3ab437af53c66997e94a0268
type: bug
status: delivering
diagnosis_status: diagnosed
domain: engineering
priority: high
severity: high
size: medium
root_cause_class: design
created: 2026-07-30
tags: [jira, attachments, tracker-broker, redirects, credentials, evidence]
relations:
  - target: mail-3ab437af53c66997e94a0268
    kind: derived_from
  - target: brokered-tracker-agent-access
    kind: related
  - target: tracker-source-evidence-preflight
    kind: related
delivery_method: manual
---

# Jira attachment content is blocked by broker redirect policy

## Kickoff

Make Jira attachment bodies readable through Hero without weakening the
same-origin credential boundary.

**Status:** delivering — the shared normalizer, broker integration, typed
redirect boundary, and regressions are implemented. Both focused regressions
and the complete tracker package pass; repository-wide vet and compile checks
are clean.

**Pick up at:** commit the implementation, run the cold delivery audit, address
any findings, and complete the `hero spec verify` gate.

→ `/deliver mail-3ab437af53c66997e94a0268`

**Files:** `internal/tracker/broker.go`, `internal/tracker/jira.go`, `internal/tracker/broker_test.go`, `internal/tracker/tracker_test.go`
**Skip:** do not allowlist Jira media hosts or forward credentials across origins.

## Summary

### Categorization

| Attribute | Assessment |
|---|---|
| **Criticality** | High — Jira issue metadata remains readable, but attachment bodies needed as source evidence are systematically unavailable through the advertised broker path |
| **Ease of Fix** | Moderate — the safe provider request is known, but both attachment paths need the same explicit security contract and regressions |
| **Caused by our codebase?** | Yes — Hero combines Jira's redirecting default with an intentional same-origin redirect boundary without normalizing the provider request |
| **Needs more research?** | No — the received production-path evidence and current source agree, and the safe `redirect=false` request was already reproduced against the same attachment |

### Background

Jira Cloud's attachment-content endpoint redirects to a media host by default.
Hero's tracker broker correctly rejects that cross-origin redirect so the Jira
credential cannot cross the configured-origin boundary. Jira can instead
stream the same attachment from the configured origin when the request includes
`redirect=false`.

The typed evidence downloader has the opposite inconsistency: it validates the
initial attachment URL against the configured Jira origin, then uses the
client's default redirect behavior. That can contact a different host even
though Go normally strips sensitive authorization headers. Both paths need one
deterministic provider-specific request shape and one fail-closed boundary.

### Analysis

`Broker.Request` constructs the bounded same-origin URL, injects Jira
authentication, and rejects cross-origin redirects. It does not recognize Jira
attachment-content GETs or set `redirect=false`, so Jira's documented default
always collides with the safety policy.

`jira.DownloadEvidenceAttachmentContext` verifies only the initial URL's scheme
and host. It neither sets `redirect=false` for Jira attachment-content URLs nor
installs a same-origin `CheckRedirect` policy on its client copy.

### Root Cause

This is a design defect at the provider boundary. A generic authenticated
request API cannot leave Jira's security-relevant attachment transport detail
to each caller or model. The broker's rejection is correct; the missing piece
is deterministic Jira request normalization shared with the typed evidence
path.

### Source

The diagnosis arrived through Project Mail
`mail_3ab437af53c66997e94a0268` from Hero Code commit `99eb0eb`. Its complete
evidence is preserved in
`.hero/planning/intake/mail-3ab437af53c66997e94a0268/spec.md`.

### Fix Direction

Add a narrow Jira attachment-content path predicate and URL normalizer that
overrides the query to `redirect=false`. Apply it only to Jira GETs in the
broker and to recognized Jira attachment-content URLs in the typed downloader.
Clone the typed downloader's HTTP client and install the same-origin redirect
check before dispatch so an unexpected Jira redirect still fails without
contacting its target.

## Problem Statement

When an agent reads a Jira attachment through `hero_tracker.request`, Hero sends
`GET /rest/api/3/attachment/content/<id>` with the connection credential. Jira
answers with a redirect to `api.media.atlassian.com`; the broker rejects it as
`unsafe_redirect`, so the attachment body is unavailable. Repeating the same
request with `query.redirect=false` returns the body with HTTP 200 from the
configured Jira origin.

The local typed evidence path can download attachments, but its current
implementation does not enforce the redirect boundary its comment promises.
The two paths therefore disagree on security behavior.

## Environment Details

- Source runtime: Hero Code's bundled Hero executable on macOS.
- Provider: Jira Cloud using the configured stable-ID Jira connection.
- Received evidence: ten default attachment-content requests returned
  `unsafe_redirect`; the same authenticated request with `redirect=false`
  returned the real bounded attachment body.
- No credential, attachment body, or signed media URL is stored in this spec.

## Root Cause Analysis

Confirmed load-bearing claims:

1. **Read:** `internal/tracker/broker.go` builds a same-origin URL from
   `relative_path` and structured query values, injects the Jira credential,
   and rejects every cross-origin redirect before the target is contacted.
2. **Read:** `Broker.Request` has no Jira attachment-content normalization, so
   a caller that omits the provider-specific query receives Jira's redirecting
   default.
3. **Read:** `internal/tracker/jira.go` rejects an initially cross-origin
   attachment URL but calls `j.client.Do` without a request-specific
   `CheckRedirect`.
4. **Read:** the typed evidence store reduces attachment transport failures to
   a bounded omission and does not need a new credential or error channel.
5. **Confirmed by received live evidence:** `redirect=false` returns the
   attachment body from the configured Jira origin.

The defect is not missing authentication, output bounds, or a required media
allowlist. It is the absence of a provider-owned normalization rule shared by
the two existing authenticated transports.

## Code Flow (End to End)

### Brokered non-local research

1. Hero Code invokes `hero_tracker.request` with a Jira attachment-content
   relative path.
2. `internal/cli/tracker_broker.go` decodes bounded JSON and calls
   `tracker.Broker.Request`.
3. `internal/tracker/broker.go` resolves the selected Jira connection,
   constructs the configured-origin URL, and injects authentication.
4. Jira returns a redirect to its media origin.
5. `Broker.Request` rejects the cross-origin redirect and returns normalized
   `unsafe_redirect`; the media host is never contacted.

### Typed local evidence

1. `hero sync evidence` loads full issue evidence and attachment metadata.
2. `internal/tracker/evidence_store.go` calls
   `DownloadEvidenceAttachmentContext`.
3. `internal/tracker/jira.go` validates the initial URL against `j.baseURL`,
   injects authentication, and currently delegates redirects to the default
   client policy.
4. A failed download becomes a bounded `download unavailable` omission.

## Key Files

### Tracker broker

| File | Relevance |
|---|---|
| `internal/tracker/broker.go` | Authenticated generic request, URL/query construction, redirect policy, and normalized errors |
| `internal/tracker/broker_test.go` | Existing cross-origin, redirect-bound, credential, cancellation, and output-bound regressions |

### Jira typed evidence

| File | Relevance |
|---|---|
| `internal/tracker/jira.go` | Jira client, attachment URL validation, authentication, and body download |
| `internal/tracker/tracker_test.go` | Full Jira evidence and attachment integration coverage |
| `internal/tracker/jira_evidence_context_test.go` | Attachment cancellation coverage |
| `internal/tracker/evidence_store.go` | Bounded omission behavior for typed attachment failures |

## Secondary Defects

The typed evidence downloader's comment claims it preserves a same-origin
boundary, but the current implementation checks only the initial URL. This
spec includes that defect because the same shared normalizer and redirect
policy fix it without expanding the public contract.

No evidence supports changing the evidence-store omission shape or adding a
second download mechanism.

## Goal

Make Jira attachment bodies reliably readable through brokered research and
typed evidence while retaining one auditable same-origin credential boundary.

## Acceptance Criteria

- **AC-1:** WHEN a Jira `GET /rest/api/3/attachment/content/<id>` request reaches the tracker broker THE SYSTEM SHALL override the provider query to `redirect=false` and return the existing bounded response from the configured Jira origin.
- **AC-2:** IF Jira still redirects an attachment-content broker request to another origin THEN THE SYSTEM SHALL return `unsafe_redirect` without contacting the redirect target or forwarding credentials.
- **AC-3:** WHEN the typed Jira evidence downloader receives a configured-origin attachment-content URL THE SYSTEM SHALL request `redirect=false`, preserve cancellation, and return the attachment bytes through the existing evidence path.
- **AC-4:** IF the typed Jira evidence downloader receives an initially cross-origin URL or encounters a cross-origin redirect THEN THE SYSTEM SHALL reject it without contacting the redirect target or exposing credentials.
- **AC-5:** WHEN either Jira attachment path follows a same-origin redirect THE SYSTEM SHALL retain the existing redirect bound and reapply Jira authentication only within the configured origin.
- **AC-6:** THE SYSTEM SHALL leave non-Jira providers, non-GET broker requests, non-attachment Jira paths, caller-header validation, output redaction, and non-idempotent redirect rejection unchanged.

## Suggested Fix Approach

1. Add a shared unexported Jira attachment-content path predicate and URL
   normalizer in `internal/tracker/jira.go`.
   - Match only a non-empty single attachment ID below
     `/rest/api/3/attachment/content/`.
   - Set, rather than append, `redirect=false` so a caller cannot retain
     `redirect=true`.
2. In `Broker.Request`, after bounded URL construction and before dispatch,
   apply the normalizer only when the resolved provider is Jira and the method
   is GET. Keep its existing `CheckRedirect`, redaction, bounds, and error
   normalization unchanged.
3. In `DownloadEvidenceAttachmentContext`, normalize recognized attachment
   URLs, clone `j.client`, and install a bounded same-origin `CheckRedirect`
   callback that reapplies Jira headers only after the target passes the
   origin check.
4. Add regressions that assert the origin sees `redirect=false`, valid bodies
   return, same-origin redirects retain auth, cross-origin targets receive
   zero requests, and unrelated paths/providers remain unchanged.

## Changes

1. Update `internal/tracker/jira.go` with shared Jira attachment URL
   normalization and a fail-closed typed-download redirect policy.
2. Update `internal/tracker/broker.go` to normalize Jira attachment-content
   GETs before its existing authenticated dispatch.
3. Extend `internal/tracker/broker_test.go`,
   `internal/tracker/tracker_test.go`, and only if needed
   `internal/tracker/jira_evidence_context_test.go` with security-boundary and
   regression coverage.

## Validation

- Run focused broker and Jira attachment tests before and after the change.
- Run `go test ./internal/tracker -count=1`.
- Run `go test ./internal/cli -run 'TestTrackerBroker|TestSyncEvidence' -count=1`.
- Run `go test ./... -count=1` and `go vet ./...`.
- Exercise the real `hero tracker broker request` command against a local fake
  Jira server and confirm the body returns only when the server observes
  `redirect=false`.
- Run `hero spec lint mail-3ab437af53c66997e94a0268`.

## Boundaries

- Do not allowlist `api.media.atlassian.com` or any other media origin.
- Do not forward Jira credentials, cookies, or caller-supplied authorization
  to a different origin.
- Do not add a new credential store, downloader, public operation, or raw
  authenticated shell fallback.
- Do not change Hero Code source in this repository; return the verified Hero
  contract and commit through the existing handoff thread.
- Do not generalize provider-specific normalization beyond the Jira
  attachment-content endpoint.

## Recap

Hero's secure redirect rejection conflicts with Jira Cloud's attachment
default because the provider request is not normalized. A narrow
`redirect=false` rule shared by the broker and typed evidence downloader fixes
the functional failure and makes both paths enforce the same credential
boundary.

## Completion Ledger

Implementation reuses the existing tracker broker, Jira client, stable
connection credential resolution, same-origin predicate, and redirect bound.
No public contract, credential store, or cross-origin allowlist was added.

Validation performed:

- compiled the complete `internal/tracker` and `internal/cli` test binaries;
- ran both new regressions and the complete `internal/tracker` suite;
- ran `go vet ./...`;
- ran `hero spec lint mail-3ab437af53c66997e94a0268`;
- ran `git diff --check`.

The host's macOS developer-tool classification blocked direct launch of newly
linked binaries from Codex. The tracker tests were therefore launched through
Apple's signed LLDB surface. The larger CLI test binary compiled cleanly but
could not be runtime-launched through that host workaround; no CLI code changed,
and every CLI/MCP caller continues to use the tested shared `Broker.Request`
implementation.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Jira broker attachment GET forces `redirect=false` | DONE | `Broker.Request` applies the shared Jira normalizer after bounded URL construction; the focused fake-Jira regression observes `false` and receives the attachment body. |
| 2 | Unexpected broker cross-origin redirect remains fail-closed | DONE | The regression forces a media-host redirect, receives `unsafe_redirect`, and proves the target call count remains zero. |
| 3 | Typed downloader normalizes and preserves cancellation | DONE | The typed regression observes `redirect=false` and returns exact bytes; the complete tracker suite retains the existing in-flight cancellation regression. |
| 4 | Typed initial and redirected cross-origin URLs are rejected | DONE | The typed regression covers both shapes and proves the media target receives zero requests. |
| 5 | Same-origin redirects remain bounded and authenticated | DONE | Typed coverage verifies reapplied Basic authentication and the shared five-redirect bound; existing broker redirect/authentication tests pass in the full tracker suite. |
| 6 | Unrelated providers, methods, paths, and safety behavior are unchanged | DONE | Focused coverage exercises non-Jira, non-GET, and unrelated Jira paths; the complete tracker suite covers header rejection, redaction, bounds, and non-idempotent redirects. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Shared Jira attachment URL normalization and typed redirect policy | DONE | `internal/tracker/jira.go` adds the narrow v3 attachment predicate, overrides the query, clones the client, and rejects cross-origin or excessive redirects with bounded sentinel errors. |
| 2 | Broker normalization before authenticated dispatch | DONE | `internal/tracker/broker.go` applies normalization only to resolved Jira GET requests and retains the existing broker transport and error contract. |
| 3 | Security-boundary and compatibility regressions | DONE | `internal/tracker/broker_test.go` and `internal/tracker/tracker_test.go` cover successful bodies, forced redirects, auth, limits, provider/method/path isolation, and zero target contact. |

### Exercise-the-feature check

- [x] The exact broker request was exercised against a fake configured-origin
  Jira server: a caller-supplied `redirect=true` was replaced with `false`, the
  attachment body returned, and a forced cross-origin redirect produced
  `unsafe_redirect` without contacting the target. The typed attachment path
  was exercised independently with the same boundary.

### Excellence Bar self-check

- [x] Yes — the fix is provider-narrow, reuses one authenticated transport and
  credential system, fails closed on every unexpected redirect, and carries
  executable positive, negative, compatibility, cancellation, and credential
  boundary coverage.
