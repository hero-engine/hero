# Delivery audit — layered-integration-configuration

**Audited:** working tree diff for the spec-named implementation and documentation files, plus untracked implementation/tests and `test-evidence.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC1 shared project settings plus local auth produce one effective integration with field provenance — `internal/config/integrations.go:87`, `internal/config/integrations_test.go:11`.
- [✓] AC2 a complete local-only integration can define its connection and selectors — `internal/cli/connect.go:265`, `internal/cli/connect_integrations_test.go:66`.
- [✓] AC3 local false, zero, empty-string, and null replace/delete semantics — `internal/config/integrations.go:195`, `internal/config/integrations_test.go:37`, `internal/config/integrations_test.go:46`.
- [✓] AC4 dangling selectors identify both selector and missing connection paths — `internal/config/integrations.go:239`, `internal/config/integrations_test.go:46`.
- [✓] AC5 selection is explicit ID, then role, then default without map-order fallback — `internal/config/integrations.go:365`, `internal/cli/sync.go:64`, `internal/config/integrations_test.go:79`.
- [✓] AC6 stable IDs support multiple same-provider connections — `internal/config/integrations.go:17`, `internal/config/integrations_test.go:79`, `internal/config/credentials_test.go:150`.
- [✓] AC7 committed literal credentials are rejected without echo — `internal/config/integrations.go:147`, `internal/config/integrations_test.go:55`.
- [✓] AC8 committed writes exclude local/global/effective tracker and Confluence credentials — `internal/config/config.go:1541`, `internal/config/config.go:1566`, `internal/config/integrations_test.go:118`. The regression loads distinct local canaries into both runtime adapters, mutates, calls generic `Save`, and asserts neither secrets nor derived legacy blocks reach `hero.json`.
- [✓] AC9 local patches preserve unrelated keys/connections and enforce mode 0600 — `internal/config/integrations.go:427`, `internal/config/integrations_test.go:94`.
- [✓] AC10 legacy tracker/Confluence remains readable with one warning and no rewrite — `internal/config/config.go:1401`, `internal/config/config.go:1437`; no load-time writer is present.
- [✓] AC11 explicit migrator — DESCOPE-APPROVED by the recorded user decision; no implementation required.
- [✓] AC12 canonical/legacy conflicts report both paths — `internal/config/integrations.go:105`, `internal/config/integrations_test.go:73`.
- [✓] AC13 unknown legacy/canonical fields and misspelled/provider-inapplicable settings fail validation — `internal/config/integrations.go:114`, `internal/config/integrations.go:217`, `internal/config/integrations.go:267`, `internal/config/integrations_test.go:63`.
- [✓] AC14 noninteractive connect accepts deterministic non-secret fields and protected stdin, with no raw token flag — `internal/cli/connect.go:67`, `internal/cli/connect.go:201`, `internal/cli/connect_integrations_test.go:66`.
- [✓] AC15 interactive secrets use one stable reader and no-echo TTY input without echoed fallback — `internal/cli/connect.go:78`, secure-input implementation in `internal/cli/connect.go`, and `internal/cli/connect_integrations_test.go:95`.
- [✓] AC16 list/status reports sorted identity, selection, provenance, readiness, credential source, and verification without secret material — `internal/cli/connect.go:123`, `internal/cli/connect_integrations_test.go:14`; persisted E2E evidence confirms redacted human/JSON behavior.
- [✓] AC17 diagnostics name supported credential sources and a runnable status/recovery command — `internal/config/integrations.go:423`, `internal/cli/connect.go:223`, `internal/config/integrations_test.go:63`.
- [✓] AC18 connect aliases share flags and examples use canonical `hero sync import` — `internal/cli/connect_alias.go`, `internal/cli/connect_integrations_test.go:58`, `web/docs/src/workflows/sprint-and-planning.md:57`.
- [✓] AC19 guidance propagation covers opencode, cursor, claude, copilot, codex, and generic targets — `internal/install/operational_guidance.go:28`, `internal/install/harness_native_test.go:91`; focused parity evidence is PASS.

## Changes
- [✓] Change 1 canonical models/parser/overlay/strict validation — `internal/config/integrations.go`; provider-specific allowed/required fields and JSON value types for GitHub, Jira, Linear, GitLab, and Confluence are exercised by `TestProviderSettingsValidationIsSpecificAndTypeStrict`.
- [✓] Change 2 layer/runtime APIs and persistence boundaries — `internal/config/config.go:1541`, `internal/config/integrations.go`, `internal/config/credentials.go`; generic-save tracker/Confluence canary regression directly closes resolved-secret propagation.
- [✓] Change 3 schema documentation/examples — `web/docs/src/configuration/hero-json.md`, `web/docs/src/configuration/tracker-setup.md`.
- [✓] Change 4 legacy normalization and migrator — DESCOPE-APPROVED; narrow read-only compatibility and conflict handling remain.
- [✓] Change 5 consumer integration selection — `internal/config/integrations.go:60`, `internal/cli/sync.go:64`, `internal/cli/tracker_ops.go`, `internal/cli/sync_import.go`, `internal/cli/sync_push.go`.
- [✓] Change 6 shared connect workflow — `internal/cli/connect.go:67`, `internal/cli/connect.go:201`, protected interactive input in `internal/cli/connect.go`.
- [✓] Change 7 truthful list/status and repaired failures — `internal/cli/connect.go:123`, `internal/config/integrations.go:423`.
- [✓] Change 8 docs/generated command parity — updated configuration/workflow docs and shared operational guidance; all-target propagation evidence is PASS.

## Open items
- None. The ledger contains no PARTIAL, SKIPPED, or BLOCKED rows; AC11/Change 4 are user-approved scope removal.

## Audit notes
- The prior AC8 blocker is resolved by a centralized committed-copy boundary and a direct tracker-plus-Confluence `Load` → mutate → generic `Save` canary regression.
- The prior Change 1 blocker is resolved by provider-specific allowed/required settings and type validation across all five providers, with negative tests for inapplicable keys, nulls, wrong types, empty required strings, and missing required settings.
- Persisted evidence records focused packages, full `go test ./...`, harness parity, formatting/diff checks, drift, and a fresh-workspace CLI exercise as PASS. This artifact-only audit did not rerun experiments.
