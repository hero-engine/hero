# Delivery evidence — layered-integration-configuration

Date: 2026-07-15

## Automated validation

- Baseline before implementation: `go test ./internal/config ./internal/tracker ./internal/wiki` passed; `internal/cli` baseline was blocked inside the restricted sandbox by loopback-listener tests, then passed when run with its required test permissions.
- Focused final: `go test ./internal/config ./internal/tracker ./internal/wiki ./internal/cli ./internal/install` — PASS.
- Full suite: `go test ./...` — PASS, including `internal/cloud` (75.983s) and every repository package.
- Harness parity: `go test ./internal/install -run TestHarnessNative_DoctorRoutingGuidanceAllTargets -count=1` — PASS for opencode, cursor, claude, copilot, codex, and generic across all domain packs.
- Formatting/static diff: `gofmt` on all touched Go files and `git diff --check` — PASS.
- Stale tracker-import examples: `rg 'hero import'` across the touched CLI/config/workflow guidance returned no matches.
- Drift: `go run ./cmd/hero drift layered-integration-configuration` — PASS, `19/19 acceptance criteria have related code changes`, `No drift detected`.

## Cold-audit remediation

- Save boundary: `TestLoadMutateSaveNeverCommitsEffectiveTrackerOrConfluenceSecrets` loads canonical shared settings plus distinct local tracker and Confluence canary tokens, confirms both runtime adapters receive them, mutates the effective config, calls the generic `Config.Save`, and proves the committed file contains neither canary, no token/redaction placeholder, and no derived legacy `tracker`/`confluence` block. The saved file reloads successfully. `Config.forCommittedSave` centralizes this boundary for every existing `Load` → mutate → `Save` call site.
- Provider schemas: `TestProviderSettingsValidationIsSpecificAndTypeStrict` covers Jira, GitHub, Linear, GitLab, and Confluence. It rejects provider-inapplicable keys, null/wrong-typed values, empty required strings, and missing required settings with exact connection/setting paths.
- Post-remediation validation: `go test ./internal/config ./internal/cli ./internal/tracker ./internal/wiki` — PASS; `go test ./...` — PASS; rebuilt CLI loaded the representative split-layer Jira workspace and emitted the same redacted provenance/readiness JSON.

## Exercise the feature

A fresh workspace at `/tmp/hero-integration-e2e.tbNVqF` was exercised with a newly built CLI. The secret was generated directly into protected stdin and never placed in argv or command output.

1. `hero connect jira --integration-id jira-delivery --project MORPH --base-url https://jira.invalid --user-email dev@example.invalid --token-stdin --no-verify --json` returned a redacted ready JSON result.
2. `hero connect --list --json` reported `jira-delivery`, provider, default/role selection, credential source/readiness, verification state, and provenance without token bytes or a token-derived mask.
3. File inspection showed `.hero/hero.json` mode `0644` and `.hero/hero.local.json` mode `0600`; the committed file contained only settings while the local file held auth.
4. `hero sync --integration jira-delivery import --dry-run` selected the canonical Jira adapter and reached its search request. It stopped only at the intentionally nonexistent `.invalid` DNS endpoint; diagnostics contained no secret.
5. Automated all-provider exercises cover GitHub, Jira, Linear, GitLab, and Confluence complete local-only definitions.

## Completion ledger — Acceptance Criteria

| # | Status | Evidence |
|---|---|---|
| AC1 shared settings + local auth | DONE | `TestIntegrationOverlayAndProvenance`; `ResolveIntegrationDocuments` records field paths and source files. |
| AC2 complete local-only integration | DONE | `TestNonInteractiveConnectAllProvidersLocalOnly`; resolver handles missing committed file. |
| AC3 false/0/empty/null semantics | DONE | RFC 7396-style `mergePatch`; overlay and null-deletion tests, raw JSON preserves scalar presence. |
| AC4 dangling selector paths | DONE | `TestIntegrationNullDeletesAndDanglingSelector` asserts selector and connection paths. |
| AC5 explicit → role → default selection | DONE | `ResolvedIntegrations.Select`, `Config.SelectTracker`, sync persistent `--integration`; `TestStableIDsAndExplicitSelection`. |
| AC6 multiple stable IDs/same provider | DONE | stable-ID map plus two-Jira selection/credential ambiguity tests. |
| AC7 reject committed literal credential | DONE | `integrationNode`, `ValidateCommittedIntegrations`, `TestCommittedSecretRejectedWithoutEcho`. |
| AC8 committed writers never serialize resolved secrets | DONE | target-layer patches plus centralized `Config.forCommittedSave`; direct tracker+Confluence load-local-canary→mutate→Save regression proves no secret or derived legacy block reaches committed config. |
| AC9 local writes preserve unrelated keys/connections and 0600 | DONE | atomic `PatchLocalIntegrations`; `TestPatchLocalIntegrationsPreservesKeysAndMode`. |
| AC10 legacy runtime remains readable, warns once, no rewrite | DONE | existing legacy merge/runtime path retained; `warnLegacyIntegrations` uses `sync.Once`; no legacy writer/load rewrite added. |
| AC11 explicit migrator | DESCOPE-APPROVED | User decision: “we don't need a migrator - there isn't anyone that needs migration but me”. Migrator/check/dry-run/backups were removed. |
| AC12 legacy/canonical conflict | DONE | `hasLegacyIntegration` conflict reports `$.integrations` and `$.tracker/$.confluence`; regression test. |
| AC13 unknown fields fail | DONE | strict canonical decoder/provider-setting validation and exact `$.tracker.jira` recovery test. |
| AC14 protected noninteractive connect | DONE | provider-general flags, `--token-stdin`, no token argv flag; all-provider and canary-output tests. |
| AC15 interactive no-echo/stable input | DONE | one `connectInput` reader; `term.ReadPassword` on `/dev/tty`; no echoed fallback; stable multiline reader test. |
| AC16 truthful redacted list/status | DONE | sorted human/JSON rows include identity, selection, per-path sources, readiness, credential source, verification; canary negative test and E2E output. |
| AC17 actionable credential diagnostics | DONE | `ResolveToken` and `CredentialGuidance` name local/global/env sources and `hero connect --list`; unknown nesting supplies runnable connect command. |
| AC18 alias parity/canonical import examples | DONE | shared flag registration/service and alias test; CLI/docs use `hero sync import`. |
| AC19 six harness targets | DONE | shared operational guidance contributor plus all-six-target tripwire test PASS. |

## Completion ledger — Changes

| Change | Status | Evidence |
|---|---|---|
| 1. Canonical models/parser/overlay/validation | DONE | `internal/config/integrations.go`; provider-owned allowlists, required fields, and JSON type validation across all five providers in `TestProviderSettingsValidationIsSpecificAndTypeStrict`. |
| 2. Layer/runtime APIs and persistence boundaries | DONE | distinct raw/effective provenance, stable-ID credentials, atomic layer patches, centralized secret-free generic save copy, direct tracker+Confluence save-boundary canary regression. |
| 3. Schema documentation/examples | DONE | `hero-json.md`, `tracker-setup.md`: partial/local/multiple/roles/deletion/precedence examples. |
| 4. Legacy normalization and migrator | DESCOPE-APPROVED | User decision quoted above; narrow read-only compatibility retained, no auto-rewrite. |
| 5. Consumer integration selection | DONE | canonical configs normalize into existing tracker/wiki adapters; sync import/spec/Jira/push/tracker ops honor persistent `--integration`; stable-ID global credential compatibility fails closed on ambiguity. |
| 6. Shared connect workflow | DONE | canonical interactive and noninteractive persistence, protected stdin, secure TTY, local-only/global/shared modes, machine result. |
| 7. Truthful list/status and repaired failures | DONE | effective sorted human/JSON status, provenance/readiness, no masks; provider-aware credential and unknown-path guidance. |
| 8. Docs/generated command parity | DONE | canonical import/config/connect docs and shared six-target install guidance guarded by propagation test. |

No ledger row is PARTIAL, SKIPPED, or BLOCKED.
