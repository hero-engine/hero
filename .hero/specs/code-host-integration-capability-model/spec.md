---
title: "Code-host integration capability model"
slug: code-host-integration-capability-model
type: feature
status: completed
domain: engineering
priority: critical
size: medium
created: 2026-07-27
parent: hero-code-host-broker-capabilities
relates-to:
  - integration-config-uses-stable-ids
  - layered-integration-configuration
  - brokered-tracker-agent-access
tags: [integrations, code-host, credentials, capabilities, github, gitlab]
delivery_method: manual
completed_at: 2026-07-28T00:28:43Z
---

# Code-host integration capability model

## Context

Hero's canonical `integrations.connections` map already separates stable
connection identity from provider type, overlays committed and local
configuration by ID, and resolves credentials from local, global, or
environment-backed sources. Its role registry currently knows `delivery`,
`roadmap`, and `docs`, and provider validation is still tracker-oriented.

GitHub and GitLab are multi-capability providers: the same account/repository
connection can back issue tracking and code hosting. Jira and Linear are
tracker-only. This child adds that distinction to the existing model before a
PR contract or adapter can accidentally equate provider with role.

## Goal

Add a first-class `code-host` integration role and provider capability
declarations that reuse Hero's stable IDs, layered settings, `config.Secret`,
and credential precedence. Code-host selection must be explicit and
provider-valid without changing delivery-tracker selection or creating another
credential system.

## Kickoff

Teach the existing integration registry which providers can host repositories
and resolve `roles.code-host` independently from tracker delivery.

**Status:** delivering — capability declarations, role validation, typed
runtime projections, connect behavior, and compatibility tests are implemented.

**Pick up at:** validate the Completion Ledger and cold audit, then run
`hero spec verify code-host-integration-capability-model`.

→ `/deliver code-host-integration-capability-model`

**Files:** `internal/config/integrations.go`, `internal/config/credentials.go`, `internal/config/integrations_test.go`, `internal/cli/connect.go`
**Skip:** do not add a code-host token field/store or fall back from `code-host` to `delivery`.

## Problem

The current maps in `internal/config/integrations.go` answer only “is this a
known provider?” and “is this a known role?” They cannot express that GitHub
can serve both issues and PRs while Jira serves only issues. Adding
`roles.code-host` to the role allowlist alone would permit nonsensical Jira
bindings, while creating a separate code-host connection schema would duplicate
stable IDs, layering, credential precedence, redaction, and connect UX.

`TrackerConnection` also packages credential-safe metadata under a
tracker-specific name. Reusing it directly would leak tracker concepts into the
new domain; reimplementing it would risk different secret resolution.

## Approach

Introduce a central provider declaration with semantic capability constants:
`tracker`, `code-host`, and `docs`. GitHub and GitLab permit tracker plus
code-host eligibility; Jira and Linear permit tracker; Confluence permits docs.
Each connection declares the subset it serves in an optional top-level
`capabilities` array. Omitted capabilities infer the legacy behavior
(`tracker` for non-Confluence and `docs` for Confluence), so existing GitHub
tracker connections remain tracker-only until a user deliberately adds
`code-host`. Eligibility means a role binding is meaningful, not that the
current binary implements every operation. Runtime adapter capabilities remain
a separate broker response.

Add `code-host` to the role registry and validate each role target against the
connection's declared/inferred capabilities and its provider's permitted set.
Preserve the same provider settings object.
For GitHub/GitLab, `settings.project` remains the default `owner/repository` or
`namespace/project`; optional bounded `settings.repositories` may add explicit
repository scopes without changing the connection or credential identity.

Extract the common secret-bearing connection metadata and token resolution into
an integration-level runtime type. `TrackerConnection` and the new
`CodeHostConnection` become typed projections over that shared runtime
connection. Both reveal the token only inside their adapter boundary.

Code-host resolution order is:

1. explicit `connection_id`, when supplied;
2. `integrations.roles.code-host`;
3. otherwise a structured missing-selection error.

It never consults `integrations.default`, `roles.delivery`, provider names,
ambient `gh`/`glab` authentication, or map iteration. The same stable
connection ID may appear under both delivery and code-host roles.

## Changes

1. Add provider and connection capability declarations in
   `internal/config/integrations.go`.
   - Replace the boolean provider map with provider metadata while preserving
     the current accepted provider set and provider-specific settings schemas.
   - Declare GitHub/GitLab as tracker+code-host, Jira/Linear as tracker, and
     Confluence as docs.
   - Add optional `IntegrationConfig.Capabilities`; omitted values infer only
     the provider's legacy capability, while explicit values must be unique and
     supported by the provider.
   - Keep static eligibility separate from runtime implemented operations.
2. Register and validate `integrations.roles.code-host`.
   - Reject a role whose selected provider lacks the declared capability.
   - Include exact JSON paths, connection ID, provider, and supported
     capabilities in errors without including auth metadata.
   - Permit one connection ID to satisfy multiple compatible roles.
3. Add optional repository-scope settings for code-host-capable providers.
   - Preserve `settings.project` as the default repository.
   - Validate `settings.repositories` as a bounded, unique array of canonical
     provider repository names; do not infer account-wide scope.
4. Extract a credential-safe integration runtime connection.
   - Reuse `IntegrationAuth`, `Secret`, `ApplyCredentialsStrict`, stable global
     credential keys, token-env precedence, and redaction behavior.
   - Make existing tracker resolution delegate to the shared primitive and
     filter only tracker-capable connections, so code-host-only GitHub/GitLab
     connections do not create tracker ambiguity.
5. Add `ResolveCodeHostConnection` and code-host metadata.
   - Select explicit ID then `roles.code-host`, validate provider eligibility,
     expose host/default/configured repositories, and retain the token as
     `config.Secret`.
   - Return typed selection/configuration/credential failures suitable for
     later normalization rather than requiring message parsing.
6. Extend connect/status configuration behavior in `internal/cli/connect.go`.
   - Accept `--role code-host` for GitHub/GitLab.
   - Reject it for Jira/Linear/Confluence before persistence.
   - Continue using protected stdin/local/global/env credential sources and
     never print capability-derived credential information.
7. Add focused config, credential, connect, and compatibility tests.

## Acceptance Criteria

- **AC-1:** WHEN a GitHub or GitLab connection declares `code-host` and is assigned `roles.code-host` THE SYSTEM SHALL accept the binding and preserve the stable connection ID.
- **AC-2:** WHEN one GitHub or GitLab connection explicitly declares both `tracker` and `code-host` and is assigned both roles THE SYSTEM SHALL reuse one connection definition and one resolved credential.
- **AC-3:** IF Jira, Linear, or Confluence is assigned `roles.code-host` THEN THE SYSTEM SHALL reject the configuration with the exact role path, connection ID, provider, and supported capabilities.
- **AC-4:** WHEN code-host resolution receives an explicit eligible `connection_id` THE SYSTEM SHALL select it regardless of delivery/default bindings.
- **AC-5:** WHEN `connection_id` is absent THE SYSTEM SHALL select only `roles.code-host`; IF that role is absent THEN it SHALL return a structured missing-selection error without using `default`, `delivery`, provider name, or map order.
- **AC-6:** THE SYSTEM SHALL resolve code-host credentials through the existing local literal, stable-ID global credential, and `token_env` precedence without adding a code-host credential field, file, or store.
- **AC-7:** THE SYSTEM SHALL retain credentials as `config.Secret` until the provider adapter boundary and SHALL redact them from generic formatting, JSON, status, errors, and tests.
- **AC-8:** WHEN `settings.repositories` is present THE SYSTEM SHALL accept only a bounded unique set of provider-valid repository names and preserve `settings.project` as the default repository.
- **AC-9:** WHEN existing tracker and docs configuration, selection, connect, credential, and broker tests run THE SYSTEM SHALL preserve their behavior and legacy compatibility.
- **AC-10:** WHEN GitLab is selected as code host before a runtime adapter exists THE SYSTEM SHALL preserve the valid role binding and resolve credential-safe connection metadata without claiming any implemented PR operation.
- **AC-11:** WHEN an existing connection omits `capabilities` THE SYSTEM SHALL infer its legacy tracker/docs capability only; a GitHub tracker SHALL NOT silently become selectable as a code host.
- **AC-12:** WHEN a GitHub or GitLab connection declares only `code-host` THE SYSTEM SHALL exclude it from tracker-broker ambiguity and delivery selection.

## Boundaries

- No PR DTOs, API calls, broker operations, MCP tools, or Swift changes.
- No new provider implementation and no GitLab merge-request adapter.
- No general role/plugin framework beyond the existing provider registry.
- No account-wide repository discovery or implicit organization scope.
- No credential migration, export, OAuth, Keychain, or new secret persistence.

## Risks

- Static provider eligibility could be mistaken for runtime support. Name and
  test the two layers separately.
- Adding `repositories` to a provider schema must not make existing
  tracker-only GitHub/GitLab configs invalid or change `settings.project`.
- Extracting runtime connection metadata can regress the shipped tracker broker
  if selection semantics move with it. Keep tracker resolution tests byte- and
  behavior-compatible.
- Stable-ID global credential fallback is ambiguous when several IDs share a
  legacy provider/project key. Preserve the existing fail-closed behavior.

## Validation

- Table-test every provider × declared/inferred capability × role combination
  and dual-role reuse.
- Test explicit code-host selection, role selection, missing role, invalid
  provider, dangling role, multiple eligible connections, and local overlays.
- Test committed settings plus local literal auth, global stable-ID auth,
  `token_env`, missing auth, and canary redaction.
- Test repository arrays for duplicates, invalid names, excessive items, empty
  values, and compatibility with the default project.
- Run `go test ./internal/config ./internal/cli ./internal/tracker`, then
  `go test ./...` and `go vet ./...`.

## Completion Ledger

Implementation commits: `84c90c4`, `fb41993`, `ccec849`.

Validation performed:

- `go test ./internal/config ./internal/cli ./internal/tracker -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- release-shaped local binary exercise using `hero connect github --role
  code-host --token-stdin --local-only --no-verify --json`, followed by
  `hero connect --list --json`
- `hero drift code-host-integration-capability-model --since 3b68f46`

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | GitHub/GitLab code-host binding preserves stable ID | DONE | `internal/config/integrations.go:375`, `internal/config/integrations_test.go:349` |
| 2 | One connection serves tracker and code-host roles | DONE | `internal/config/integrations.go:64`, `internal/config/integrations_test.go:349` |
| 3 | Tracker/docs-only providers reject code-host role | DONE | `internal/config/integrations.go:397`, `internal/config/integrations_test.go:383` |
| 4 | Explicit eligible code-host connection wins | DONE | `internal/config/integrations.go:789`, `TestResolveCodeHostExplicitIDOverridesAllSelectors` |
| 5 | Missing ID resolves only roles.code-host | DONE | `internal/config/integrations.go:779`, `internal/config/integrations_test.go:404` |
| 6 | Existing credential precedence is reused | DONE | `internal/config/integrations.go:94`, `internal/config/integrations_test.go:502` |
| 7 | Credentials remain Secret and redacted | DONE | `internal/config/integrations.go:94`, `internal/config/integrations_test.go:502` |
| 8 | Repository scopes are bounded, unique, and provider-valid | DONE | `internal/config/integrations.go:481`, `TestConnectionCapabilityValidationAndRepositoryBounds` covers GitHub exact owner/repo and GitLab nested namespace shapes |
| 9 | Existing tracker/docs behavior remains compatible | DONE | Full uncached repository suite passed; focused tracker suite passed |
| 10 | GitLab binding resolves metadata without claiming operations | DONE | `internal/config/integrations.go:46`, `TestGitLabCodeHostResolvesBeforeAdapterExists`; operation claims remain absent until the broker children |
| 11 | Omitted capabilities infer legacy behavior only | DONE | `internal/config/integrations.go:64`, `internal/config/integrations_test.go:404` |
| 12 | Code-host-only connection is excluded from tracker selection | DONE | `internal/config/integrations.go:719`, `internal/config/integrations_test.go:434` |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Provider and connection capability declarations | DONE | `internal/config/integrations.go:33` adds typed registry, inference, validation, and repository bounds |
| 2 | Register and validate roles.code-host | DONE | `internal/config/integrations.go:53`, `internal/config/integrations.go:397` |
| 3 | Add repository-scope settings | DONE | `internal/config/integrations.go:481`, `internal/config/integrations.go:750` |
| 4 | Extract credential-safe runtime connection | DONE | `internal/config/integrations.go:93` shared by typed tracker/code-host projections |
| 5 | Add ResolveCodeHostConnection | DONE | `internal/config/integrations.go:789` with typed selection, capability, and credential failures; `TestCodeHostMissingCredentialReturnsTypedError` |
| 6 | Extend connect/status behavior | DONE | `internal/cli/connect.go:127`, `internal/cli/connect.go:206`, CLI tests at `internal/cli/connect_integrations_test.go:113` |
| 7 | Add config, credential, connect, compatibility tests | DONE | New focused tests in `internal/config/integrations_test.go:349` and `internal/cli/connect_integrations_test.go:113` |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end with a release-shaped
  local Hero binary. Code-host connect returned capability `code-host`, kept
  `default` empty, selected `roles.code-host`, reported local credential
  readiness, and emitted no credential.

### Excellence Bar self-check

Yes — the change makes capability selection explicit without creating a second
connection or credential system, preserves legacy tracker behavior under the
full test suite, and exercises the actual CLI boundary.
