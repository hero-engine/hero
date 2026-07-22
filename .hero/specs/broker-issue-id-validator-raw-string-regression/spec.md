---
title: "Tracker broker issue IDs are rejected by a raw-string validator typo"
slug: broker-issue-id-validator-raw-string-regression
type: bug
status: completed
domain: engineering
size: small
priority: high
severity: high
root_cause_class: code
created: 2026-07-21
tags: [tracker, broker, jira, github, gitlab, linear, validation, security, regression]
relations:
  - target: brokered-tracker-agent-access
    kind: related
completed_at: 2026-07-22T01:00:36Z
---

# Tracker broker issue IDs are rejected by a raw-string validator typo

## Kickoff

Fixes the shared tracker-broker issue-ID guard so ordinary provider IDs work
while real path and control characters remain blocked.

**Status:** planning — v0.28.2 root cause and the inverse control-character gap are confirmed; no product code has changed.

**Pick up at:** correct `validateBrokerIssueID`, then add table-driven valid/invalid cases and an exact CLI regression for `ACME-103` with `detail: evidence`.

→ `/deliver broker-issue-id-validator-raw-string-regression`

**Files:** `internal/tracker/broker.go:124`, `internal/tracker/broker.go:201`, `internal/tracker/broker_test.go:40`, `internal/cli/tracker_broker_test.go:30`
**Skip:** do not loosen path protections, change provider adapters, or version the `tracker-broker/v1` contract.

## Summary

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — the released `get_issue` capability rejects common provider-native IDs, blocking the packaged Hero Code consumer, while the same typo fails to enforce the intended CR/LF/NUL safety check. |
| **Ease of Fix** | easy — the defect is one string-literal choice in a shared validator plus missing positive and negative regression cases. |
| **Caused by our codebase?** | Yes — Hero v0.28.2 introduced the malformed validator and insufficient test matrix in commit `7981763`. |
| **Needs more research?** | No — the failing predicate, execution path, affected provider classes, inverse safety gap, and test omission are confirmed from the tagged source and focused execution. The separately reported `MORPH-297` result is not explained by this predicate and is recorded below. |

### Background

Hero v0.28.2 added `tracker-broker/v1`, including a shared `get_issue`
operation used by the in-process broker, JSON CLI, and MCP surfaces. Exact
packaged release qualification from Hero Code reproduced this request:

```bash
printf '{"connection_id":"fixture-jira","issue_id":"ACME-103","detail":"evidence"}' | <v0.28.2 hero> tracker broker get_issue
```

Instead of fetching the Jira issue, the command returns a structured
`invalid_issue_id` error with message `issue_id contains path or control
characters`. `ACME-103` is a normal Jira key.

### Analysis

`validateBrokerIssueID` passes a raw string to `strings.ContainsAny`:

```go
strings.ContainsAny(issueID, `/\\?#\r\n\x00`)
```

Go raw strings do not interpret escape sequences. The character set therefore
contains literal lowercase `r`, `n`, `x`, and digit `0`, along with slashes,
question mark, hash, and backslashes. It does not contain actual carriage
return, line feed, or NUL characters. The result is symmetric breakage:

- valid IDs containing `r`, `n`, `x`, or `0` are rejected;
- actual embedded CR, LF, and NUL characters are not rejected by this guard.

Focused execution of the exact predicate produced:

```text
"ACME-103" => true
"MORPH-297" => false
"jira-key" => true
"ENG-10" => true
"ENG-12\n3" => false
"ENG-12\x003" => false
```

The currently passing Jira broker test uses `PCS-1339`, which happens not to
contain any accidentally forbidden literal. The unsafe-input test covers path
confusion but not actual CR, LF, or NUL. Both sides of the typo therefore
escaped the original delivery gate.

### Root Cause

**Classification: `code`.** The broker's issue-ID validator uses a raw Go
string where an interpreted string was required. `strings.ContainsAny` is
behaving correctly; it receives the wrong rune set. The resulting error is not
caused by Jira data, connection configuration, credential resolution, Hero
Code decoding, or the provider adapter.

### Source

- `internal/tracker/broker.go:124-190` — shared `Broker.GetIssue` flow.
- `internal/tracker/broker.go:201-210` — defective issue-ID validator.
- `internal/cli/tracker_broker.go:19-61` — JSON CLI adapter that exposes the
  failure to the packaged consumer.
- `internal/serve/mcp_tools_tracker.go:11-17` — MCP sibling using the same
  `Broker.GetIssue` method.
- `internal/tracker/broker_test.go:40-70,194-218` — incomplete positive and
  unsafe-ID coverage.
- Commit `7981763` — introduced the broker, validator, tests, and release
  contract together; tag `v0.28.2` contains the same lines.

### Fix Direction

Keep the existing provider-independent policy and correct its representation:
reject `/`, `\\`, `?`, `#`, actual CR, actual LF, actual NUL, and `..`; do not
reject the literal characters `r`, `n`, `x`, or `0`. Preserve the existing
numeric-only rule after GitHub/GitLab full-project IDs are split.

Add table-driven unit coverage that explicitly pairs valid IDs containing each
previously accidental rune with invalid IDs containing each real path/control
character. Add an end-to-end broker/CLI regression using `ACME-103` and
`detail: evidence` so the exact released consumer path cannot regress behind a
passing `PCS-1339` example. The shared helper change automatically repairs the
in-process, CLI, and MCP surfaces without a contract or adapter change.

The 2026-07-21 `hero anchor` check found no conflicting tripwire. The active
harness-propagation tripwire is not triggered because this fix changes runtime
broker code and tests, not harness-installed instructions or content.

---

## Issue

Release-qualification report from the packaged Hero Code consumer; no external
tracker ticket is linked and the local workspace has no matching open bug spec.

- Affected release: exact tag `v0.28.2`
- Contract: `tracker-broker/v1`
- Operation: `get_issue`
- Provider: Jira in the reproduced request; the validator is shared by all
  providers
- Detail mode: `evidence`
- Observed stable error code: `invalid_issue_id`
- Observed message: `issue_id contains path or control characters`

## Problem Statement

### Expected

Given a configured `fixture-jira` connection, `ACME-103` is accepted as a
provider-native issue ID. The broker reaches Jira's evidence adapter and emits
the usual bounded, credential-redacted `tracker-broker/v1` result or a genuine
provider error.

IDs containing path separators, URL delimiters, traversal, CR, LF, or NUL are
rejected as `invalid_issue_id` before any provider request.

### Actual

The validator rejects `ACME-103` locally because its numeric portion contains
`0`. No Jira request occurs. Conversely, embedded CR/LF/NUL do not match the
validator's supposed control-character set; downstream behavior then varies by
adapter. For example, normalized Jira URL construction may fail later in
`http.NewRequest`, while Jira evidence path-escapes the value and Linear sends
it as a GraphQL variable. That is later and provider-dependent rather than the
promised shared broker rejection.

### Minimal reproduction

1. Use the exact tagged v0.28.2 Hero binary in a workspace with a valid Jira
   connection named `fixture-jira`.
2. Run:

   ```bash
   printf '{"connection_id":"fixture-jira","issue_id":"ACME-103","detail":"evidence"}' | hero tracker broker get_issue
   ```

3. Observe a JSON response whose `error.code` is `invalid_issue_id` and whose
   message says the ID contains path or control characters.
4. Confirm no provider request was made.

## Environment Details

- Repository inspected: `/Users/developer/projects/hero-engine/repository/hero`
- Tagged source: `v0.28.2`
- Current branch: `main`; tagged and current broker/CLI files are identical
- Observation date: 2026-07-21
- Packaged consumer: Hero Code release qualification
- Local tracker connection: none; the externally supplied exact packaged
  reproduction is accepted as evidence, and the predicate was independently
  exercised without network or credentials
- Local Hero CLI on `PATH`: v0.25.1, schema-compatible with this workspace;
  used only for status/search/anchor/index operations, not as evidence about
  v0.28.2 runtime behavior

---

## Root Cause Analysis

### Confirmed findings

1. `v0.28.2:internal/tracker/broker.go:201-203` uses
   `strings.ContainsAny(issueID, `/\\?#\r\n\x00`)` with a raw string.
2. Raw strings preserve the backslashes, so the set includes literal `r`, `n`,
   `x`, and `0` and excludes real CR, LF, and NUL.
3. `ACME-103` contains `0`; direct evaluation returns `true` and therefore
   takes the `invalid_issue_id` branch.
4. The exact predicate returns `true` for `jira-key` and `ENG-10`, proving the
   impact is not Jira-specific.
5. The exact predicate returns `false` for IDs containing embedded LF and NUL,
   proving the intended control-character check is ineffective.
6. `Broker.GetIssue` resolves the connection and tracker first, then invokes
   this validator before either `GetIssueEvidence` or `GetIssue`.
7. The evidence/normalized choice does not cause the failure; both modes pass
   through the same validator.
8. The CLI adapter only decodes the request, invokes `Broker.GetIssue`, and
   encodes the response. It does not modify the issue ID.
9. The MCP `hero_tracker_get_issue` handler invokes the same method, so it has
   the same defect even though the reproduced surface is CLI.
10. GitHub and GitLab full-project forms are split before validation; their
    numeric issue portion is then subject to the same accidental `0` rejection.
11. Linear IDs are passed as GraphQL variables after the same shared guard and
    are also affected when they contain an accidentally forbidden rune.
12. `TestBrokerGetIssueAcceptsFullJiraKeyWithoutProjectConstraint` uses
    `PCS-1339`, a positive example that does not exercise any accidental rune.
13. `TestBrokerRejectsIssueIDPathConfusionBeforeProviderRequest` tests
    traversal/URL/path shapes but not actual CR, LF, or NUL.
14. Focused existing broker tests pass unchanged, confirming the current suite
    does not detect either half of the defect.
15. The original completion ledger states that a release-candidate binary
    fetched `ACME-101`; that ID also contains `0` and cannot pass the committed
    validator. The claim is therefore not reproducible from the shipped source
    and must not be reused as validation evidence for this repair.
16. The direct packaged fixture succeeds for broker `search` and `request`,
    isolating the release failure to the `get_issue` path and its issue-ID
    validator rather than connection resolution, credentials, or general
    broker transport.

### Load-bearing claim ledger

| Claim | Grounding | Confidence |
|------|-----------|------------|
| The v0.28.2 CLI returns `invalid_issue_id` for `ACME-103` | Exact packaged consumer reproduction supplied by release qualification | confirmed/reproduced externally |
| The validator rejects `ACME-103` because of `0` | Tagged source plus direct predicate execution | confirmed/read and executed |
| The validator also rejects literal `r`, `n`, and `x` | Raw-string semantics plus direct predicate execution | confirmed/read and executed |
| The validator fails to match actual CR/LF/NUL | Tagged source plus direct LF/NUL predicate execution | confirmed/read and executed |
| The failure occurs before Jira evidence retrieval | Complete CLI → broker → validator → adapter trace | confirmed/read |
| CLI, MCP, and in-process calls share the defect | Both adapters invoke `Broker.GetIssue` directly | confirmed/read |
| A local validator/test-only fix is sufficient | Provider adapters receive the unchanged ID only after the shared guard | confirmed/read |
| The same predicate explains `MORPH-297` | It does not: uppercase `MORPH-297` contains none of the accidental runes | disproved for this predicate |

### Root cause classification

- **Class:** `code`
- **Severity:** `high`
- **Blast radius:** every v0.28.2 `get_issue` surface and every configured
  provider; exact affected IDs depend on their characters
- **Frequency:** common — numeric issue identifiers regularly contain `0`, and
  lowercase provider-native IDs can contain `r`, `n`, or `x`
- **Workaround:** callers can use provider-specific `request`/search paths where
  available, but that bypasses the normalized/evidence capability and is not an
  acceptable packaged-consumer contract

---

## Code Flow (End to End)

1. `cmd/hero/main.go:22` — the tagged binary enters Cobra's CLI executor.
2. `internal/cli/root.go:132` — the root registers the `tracker` command.
3. `internal/cli/tracker_broker.go:19-34` — `tracker broker get_issue` finds
   the workspace root, creates a `Broker`, decodes stdin into
   `trackerbroker.GetIssueRequest`, and calls `Broker.GetIssue` unchanged.
4. `internal/cli/tracker_broker.go:84-97` — strict JSON decoding accepts exactly
   one object and preserves `connection_id`, `issue_id`, and `detail`.
5. `internal/tracker/broker.go:124-135` — `Broker.GetIssue` rejects an empty ID,
   defaults/validates detail, and leaves non-empty `ACME-103` intact.
6. `internal/tracker/broker.go:62-87,136-141` — the broker loads workspace
   config, selects `fixture-jira`, resolves its credential, and constructs the
   Jira tracker through `NewWithJiraConfig`.
7. `internal/config/integrations.go:496-544` — stable connection selection
   resolves explicit `fixture-jira`; it does not inspect or alter the issue ID.
8. `internal/tracker/broker.go:148-169` — the broker trims surrounding
   whitespace and applies only GitHub/GitLab full-project splitting; Jira
   `ACME-103` stays unchanged.
9. `internal/tracker/broker.go:170-172,201-203` — the malformed raw-string
   character set matches `0`, returns the path/control error, and the broker
   emits stable code `invalid_issue_id`.
10. `internal/tracker/broker.go:173-190` — unreachable in the reproduction:
    evidence mode would call `EvidenceTracker.GetIssueEvidence`; normalized
    mode would call `Tracker.GetIssue`.
11. `internal/tracker/jira.go:666-727` — the expected evidence path fetches the
    full Jira issue and paginated comments only after validation succeeds.
12. `internal/cli/tracker_broker.go:59-61` — the CLI JSON-encodes the structured
    failure and returns normally; no provider request has occurred.

---

## Key Files

### Shared broker and contract

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/tracker/broker.go` | 124-210 | Complete `get_issue` path and defective shared validator. |
| `contracts/trackerbroker/contract.go` | 10-104 | Stable operation, request, response, detail, and error-code contract; no schema change is required. |
| `internal/config/integrations.go` | 35-65, 496-544 | Connection and credential resolution that precede validation. |

### Entry points and provider destination

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/tracker_broker.go` | 19-61, 84-105 | Exact reproduced JSON CLI entry and response encoding. |
| `internal/serve/mcp_tools_tracker.go` | 11-17 | MCP sibling proving the defect is shared across surfaces. |
| `internal/tracker/tracker.go` | 57-61, 260-280 | Evidence capability and provider construction. |
| `internal/tracker/jira.go` | 627-727 | Normalized and evidence destinations reached after the guard. |

### Tests and release evidence

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/tracker/broker_test.go` | 40-70 | Positive Jira test uses an ID that misses the accidental character set. |
| `internal/tracker/broker_test.go` | 168-218 | Full-project and unsafe-ID cases omit numeric `0` and actual controls. |
| `internal/cli/tracker_broker_test.go` | 12-46 | CLI coverage tests fixture output and malformed input, not a successful `get_issue` request. |
| `.hero/specs/brokered-tracker-agent-access/spec.md` | 1-157 | Original contract, validation plan, and now-contradicted release exercise claim. |

## Secondary Defects

1. **The intended safety check is inverted for controls.** Actual CR, LF, and
   NUL are allowed by the shared validator. Downstream URL/GraphQL behavior is
   provider- and detail-mode-dependent, violating the contract's promise of a
   consistent pre-provider rejection.
2. **The regression suite samples the wrong equivalence classes.** The one
   positive Jira key avoids every accidental rune, and the negative suite tests
   path confusion without testing the control characters named by the error.
   There is no command-level successful `get_issue` test.
3. **The original release exercise record is internally inconsistent.** Its
   claimed `ACME-101` success cannot occur with the committed predicate. This
   is an evidence-quality defect in the completed spec/audit trail; the fix
   must rely on executable assertions rather than another prose-only claim.

## Notes

- `MORPH-297`, exactly as uppercase text, does not match the accidental raw
  character set. Direct predicate execution returns `false`, and the existing
  `PCS-1339` broker test confirms ordinary uppercase keys without `0` can reach
  Jira. If an exact v0.28.2 package still rejects `MORPH-297` with the same
  error after this fix, capture its full response, binary hash/version, and
  request bytes as a separate defect; do not broaden this diagnosis without
  evidence.
- The raw string at `internal/tracker/broker.go:152`/`:160` for project-name
  checks is not the same bug: it intentionally names only literal `:`, `@`, and
  backslash characters and contains no escape-looking control sequences.
- No tracker credential, live Jira call, or external write was needed for this
  diagnosis.
- The working tree already contained unrelated Jira/evidence and Hero planning
  changes. This diagnosis adds only this spec and does not modify or normalize
  those files.

---

## Goal

Restore `tracker-broker/v1 get_issue` for valid provider-native IDs containing
`r`, `n`, `x`, or `0`, including the packaged `ACME-103` evidence request,
while making the shared guard reliably reject actual path, traversal, CR, LF,
and NUL inputs before any provider request.

## Changes

1. Correct the shared issue-ID character check in
   `internal/tracker/broker.go`.
   - Use an interpreted character set or an equally explicit predicate whose
     tested runes are `/`, `\\`, `?`, `#`, `\r`, `\n`, and `\x00`.
   - Preserve trimming, `..` rejection, GitHub/GitLab project splitting, and
     their numeric issue-number validation.
   - Keep the existing stable `invalid_issue_id` code and v1 response shape.
2. Add a table-driven validator regression in
   `internal/tracker/broker_test.go`.
   - Accept Jira keys `MORPH-297`, `ACME-103`, and `ZERO-10`, plus
     GitHub/GitLab numeric IDs containing `0`.
   - Reject separate cases for slash, backslash, question mark, hash, actual
     CR, actual LF, actual NUL, and `..` traversal.
   - Assert unsafe IDs fail before any provider handler is called.
3. Add an end-to-end successful broker regression in
   `internal/tracker/broker_test.go` for Jira `ACME-103` with evidence detail.
   - Assert the expected escaped Jira issue and comment paths are reached.
   - Assert the response contains structured evidence and no
     `invalid_issue_id` error.
4. Extend `internal/cli/tracker_broker_test.go` with a command-level regression
   matching the released JSON shape.
   - Use an isolated temporary workspace/config and local test server; never a
     real token or network service.
   - Feed the exact `connection_id`/`issue_id`/`detail` request through stdin,
     decode stdout as `trackerbroker.Response`, and assert the provider request
     occurs and the structured result succeeds.
   - Keep this test at the CLI boundary so future command wiring or packaging
     changes cannot hide behind direct `Broker` coverage.

## Acceptance Criteria

- **AC-1:** WHEN `get_issue` receives a valid provider-native ID containing lowercase `r`, `n`, `x`, or digit `0` THE SYSTEM SHALL pass validation and dispatch to the selected provider.
- **AC-2:** WHEN the v0.28.2 reproduction payload requests Jira `ACME-103` with `detail: evidence` THE SYSTEM SHALL return structured evidence instead of `invalid_issue_id`.
- **AC-3:** IF an issue ID contains `/`, `\\`, `?`, `#`, `..`, actual CR, actual LF, or actual NUL THEN THE SYSTEM SHALL return `invalid_issue_id` before any provider request.
- **AC-4:** WHEN GitHub or GitLab receives a configured-project numeric issue ID containing `0` THE SYSTEM SHALL accept it, while non-numeric issue numbers SHALL remain rejected.
- **AC-5:** WHEN callers use the in-process, CLI, or MCP `get_issue` surface THE SYSTEM SHALL apply the same corrected validator through shared `Broker.GetIssue` behavior.
- **AC-6:** THE SYSTEM SHALL preserve the `tracker-broker/v1` request/response schema, stable error codes, effect classification, credential redaction, bounds, and connection-selection behavior.
- **AC-7:** WHEN focused broker and CLI tests run THE SYSTEM SHALL executable-test both the valid accidental-rune class and the actual unsafe-control class, including the exact `ACME-103` CLI request shape.

## Boundaries

- Do not change `tracker-broker/v1`, add an error code, or modify the consumer
  fixture schema; this is an implementation repair within the existing
  contract.
- Do not alter Jira, GitHub, GitLab, or Linear adapter URL/query construction.
- Do not accept paths, URLs, fragments, traversal, or control characters as
  issue IDs.
- Do not redesign provider-native ID syntax beyond the current rules.
- Do not fold a separately reproducible `MORPH-297` package failure into this
  fix unless new evidence traces it to the same code.
- Do not edit the completed `brokered-tracker-agent-access` spec merely to make
  its historical completion claim read as if this regression never shipped.

## Risks

- Replacing the raw literal with another incorrectly escaped string could fix
  false positives while preserving the control-character gap; direct rune
  tables are mandatory.
- Testing only Jira would miss GitHub/GitLab numeric IDs containing `0` and the
  shared Linear path.
- A command-level test that mocks `Broker` rather than using an isolated
  connection/provider path would not cover the released failure.
- Global Cobra stdin/stdout or process working-directory state can leak between
  CLI tests; use existing cleanup patterns and avoid parallelizing that test if
  it changes process-wide state.

## Validation

1. Run focused validator and broker regressions:

   ```bash
   go test ./internal/tracker -run 'Test(BrokerGetIssue|ValidateBrokerIssueID)' -count=1
   ```

2. Run the command-level regression:

   ```bash
   go test ./internal/cli -run 'TestTrackerBrokerCommand' -count=1
   ```

3. Run both affected packages in full:

   ```bash
   go test ./internal/tracker ./internal/cli -count=1
   ```

4. Build/tag an exact candidate and repeat the release-qualification pipe with
   the isolated `fixture-jira` server. Assert `error` is absent, evidence is
   present, and the expected provider endpoints were called.
5. Send actual embedded CR, LF, and NUL cases through the direct broker and CLI
   paths. Assert `invalid_issue_id` and zero provider calls.
6. Run `go test ./...`, then cold-audit and verify the spec before release.

---

## Recap

Hero v0.28.2 used a raw string for an escape-sequence character set, causing
valid IDs containing `r`, `n`, `x`, or `0` to be rejected and actual CR/LF/NUL
to evade the shared guard. The repair is small but high priority: correct the
shared predicate and lock both halves down with direct, provider, and exact CLI
regressions.

## Completion Ledger

Implementation used the existing Go broker/CLI architecture and preserved the
`tracker-broker/v1` contract. Validation included focused broker and CLI tests,
full affected-package execution, and an exact-v0.28.2 source qualification
binary against the local Jira fixture. The affected-package run also surfaced
one unrelated pre-existing failure in the uncommitted `jira_adf_test.go`
changes; the broker and CLI regressions themselves are green.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Valid IDs containing `r`, `n`, `x`, or `0` dispatch | DONE | `internal/tracker/broker.go:201` uses actual control runes; `TestValidateBrokerIssueIDAcceptsOrdinaryProviderIDsAndRejectsPathControls` covers every formerly accidental rune. |
| 2 | `ACME-103` evidence returns structured evidence | DONE | `TestBrokerGetIssueEvidenceAcceptsACME103` and `TestTrackerBrokerCommandGetsACME103EvidenceThroughConfiguredConnection` both exercise `detail: evidence`. |
| 3 | Path, traversal, CR, LF, and NUL fail before provider | DONE | Validator, `Broker.GetIssue`, and JSON CLI tests cover actual CR/LF/NUL; broker and CLI provider-call counters remain unchanged for unsafe requests. |
| 4 | GitHub/GitLab numeric IDs containing `0` remain valid | DONE | Validator table accepts `10` for both providers and rejects `not-10`. |
| 5 | In-process, CLI, and MCP share the corrected guard | DONE | The fix is in shared `Broker.GetIssue`; direct broker and CLI boundary tests pass, while MCP continues to call that same method. |
| 6 | Preserve v1 schema, effects, redaction, bounds, selection | DONE | No contract or adapter code changed; exact qualification reports `tracker-broker/v1`, `effect: read`, and no credential canary. |
| 7 | Focused tests cover valid accidental runes and unsafe controls | DONE | Focused `internal/tracker` and `internal/cli` commands pass, including the exact `ACME-103` JSON request. |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Correct shared issue-ID character check | DONE | `internal/tracker/broker.go` now uses an interpreted string containing actual CR/LF/NUL runes. |
| 2 | Add table-driven validator regression | DONE | `internal/tracker/broker_test.go` covers ordinary Jira/Linear IDs, numeric GitHub/GitLab IDs, every unsafe rune, and traversal; broker/CLI boundary cases prove actual controls never reach the provider. |
| 3 | Add successful `ACME-103` evidence regression | DONE | Direct broker test asserts issue/comment endpoints and structured evidence. |
| 4 | Add command-level exact JSON regression | DONE | CLI test creates an isolated connection, injects a canary credential internally, verifies evidence success, and asserts the canary is absent from output. |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: an exact v0.28.2 source export with only this fix built as `/private/tmp/hero-v0.28.2-issuefix-bin`; `tracker broker get_issue` fetched normalized `ACME-103` from the local Jira fixture with `error: null`, `effect: read`, description present, and the credential canary absent. The evidence variant passed the corrected validator and reached the fixture's unimplemented comments endpoint; full evidence succeeds in both direct-broker and CLI-boundary tests using a complete local Jira test server.

### Excellence Bar self-check

- [x] Yes — the repair is one precise shared-guard change, with executable positive, negative, provider, evidence, and CLI-boundary coverage and no contract weakening.
