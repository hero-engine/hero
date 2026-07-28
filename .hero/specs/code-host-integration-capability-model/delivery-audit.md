# Delivery audit — code-host-integration-capability-model

**Audited:** `git diff 3b68f46...ccec849`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1 GitHub/GitLab code-host bindings preserve stable IDs — provider eligibility and role validation are implemented in `internal/config/integrations.go:46` and `internal/config/integrations.go:400`; `TestIntegrationCapabilitiesAllowGitHubDualRole` and `TestGitLabCodeHostResolvesBeforeAdapterExists` assert the resolved IDs.
- [✓] AC-2 One connection serves tracker and code-host roles with one credential identity — both projections share `IntegrationConnection` at `internal/config/integrations.go:92`; `TestIntegrationCapabilitiesAllowGitHubDualRole`, `TestCodeHostConnectionUsesExistingCredentialResolution`, and `TestNonInteractiveConnectUpgradesExistingGitHubToDualCapability` cover shared ID, stable-ID credential resolution, and dual-role persistence.
- [✓] AC-3 Tracker/docs-only providers reject `roles.code-host` with safe detail — `internal/config/integrations.go:400` reports the exact role path, ID, provider, declared capabilities, and provider-supported capabilities; `TestCodeHostRoleRejectsTrackerOnlyProviders` covers Jira, Linear, and Confluence.
- [✓] AC-4 Explicit eligible `connection_id` wins over all selectors — `internal/config/integrations.go:796`; `TestResolveCodeHostExplicitIDOverridesAllSelectors` asserts selection against conflicting default, delivery, and code-host bindings.
- [✓] AC-5 Absent explicit ID selects only `roles.code-host` and otherwise returns a typed missing-selection error — `internal/config/integrations.go:790`; `TestOmittedCapabilitiesPreserveLegacyTrackerOnly` asserts no fallback to default/delivery and checks `code_host_role_missing`.
- [✓] AC-6 Existing credential precedence is reused — `IntegrationConnection.ResolveToken` and `runtimeIntegrationConnection` reuse literal `Secret`, `token_env`, and stable-ID credentials at `internal/config/integrations.go:107` and `internal/config/integrations.go:754`; `TestIntegrationOverlayAndProvenance`, `TestApplyCredentialsStableIDAndAmbiguousLegacy`, and `TestCodeHostConnectionUsesExistingCredentialResolution` exercise those paths.
- [✓] AC-7 Credentials remain `config.Secret` and are redacted — `internal/config/integrations.go:94` retains `Secret`, whose generic formatting and JSON serialization redact at `internal/config/integrations.go:174`; focused config/CLI tests and the release-shaped connect/list exercise assert no credential disclosure.
- [✓] AC-8 Repository scopes are bounded, unique, provider-valid, and preserve the default project — validation and projection are implemented at `internal/config/integrations.go:535`, `internal/config/integrations.go:600`, and `internal/config/integrations.go:767`; `TestConnectionCapabilityValidationAndRepositoryBounds` explicitly covers empty and 101-item arrays, duplicates, invalid names, and GitHub exact owner/repository shape, while `TestGitLabCodeHostResolvesBeforeAdapterExists` covers nested GitLab namespaces and the default project.
- [✓] AC-9 Existing tracker/docs behavior remains compatible — tracker selection filters by tracker capability at `internal/config/integrations.go:695` and `internal/config/integrations.go:726`; the supplied uncached focused suite, full repository suite, and vet all passed.
- [✓] AC-10 GitLab resolves credential-safe metadata without claiming PR operations — GitLab eligibility is declared at `internal/config/integrations.go:48`; `TestGitLabCodeHostResolvesBeforeAdapterExists` asserts its ID, provider, and repository metadata, and the diff adds no adapter or runtime-operation claim.
- [✓] AC-11 Omitted capabilities infer legacy tracker/docs behavior only — `internal/config/integrations.go:64`; `TestOmittedCapabilitiesPreserveLegacyTrackerOnly` asserts legacy GitHub remains tracker-only and fails implicit and explicit code-host selection.
- [✓] AC-12 Code-host-only connections are excluded from tracker ambiguity and delivery selection — tracker enumeration filters with `SupportsCapability(CapabilityTracker)` at `internal/config/integrations.go:726`; `TestCodeHostOnlyConnectionDoesNotCreateTrackerAmbiguity` asserts both implicit tracker selection and explicit rejection of the code-host-only connection.

## Changes

- [✓] Provider and connection capability declarations — `internal/config/integrations.go:33` adds typed capabilities, provider metadata, legacy inference, and validation; `internal/config/config.go:1597` preserves declarations during committed saves.
- [✓] Register and validate `roles.code-host` — `internal/config/integrations.go:54`, `internal/config/integrations.go:400`, and `TestCodeHostRoleRejectsTrackerOnlyProviders`.
- [✓] Add repository-scope settings — `internal/config/integrations.go:474`, `internal/config/integrations.go:600`, and `internal/config/integrations.go:767`, with focused bounds and provider-shape tests.
- [✓] Extract credential-safe runtime connection — shared `IntegrationConnection` and `ResolveToken` are at `internal/config/integrations.go:92`; tracker and code-host are typed projections over it.
- [✓] Add `ResolveCodeHostConnection` and typed failures — `internal/config/integrations.go:151` and `internal/config/integrations.go:790`; focused tests cover explicit/role selection, wrong capability, missing selection, and missing credentials.
- [✓] Extend connect/status behavior — `internal/cli/connect.go:123` reports capabilities and credential readiness, while `internal/cli/connect.go:206` validates role eligibility before persistence and leaves default unchanged for code-host; CLI tests cover create, reject-before-write, dual-capability upgrade, and token redaction.
- [✓] Add config, credential, connect, and compatibility tests — focused assertions in `internal/config/integrations_test.go:349` and `internal/cli/connect_integrations_test.go:113` cover all new branches, including empty and excessive repository arrays; supplied focused and full suites passed uncached.

## Audit notes

- None.
