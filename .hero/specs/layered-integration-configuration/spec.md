---
title: "Layered Integration Configuration — Shared Projects, Local Credentials, Multiple Providers"
slug: layered-integration-configuration
type: feature
status: completed
priority: high
horizon: now
domain: engineering
size: large
created: 2026-07-15
origin: session
tags: [config, integrations, trackers, jira, credentials, migration, cli, security]
relations:
  - target: jira-connection-onboarding-misleads-agents
    kind: supersedes
delivery_method: manual
completed_at: 2026-07-15T16:35:55Z
---

# Layered Integration Configuration — Shared Projects, Local Credentials, Multiple Providers

## Context

A fresh Jira-backed project exposed a mismatch between Hero's configuration model and the model users naturally infer. The committed file held the shared Jira fields while `.hero/hero.local.json` held `tracker.jira.token`; Hero silently discarded that provider-keyed local object because the runtime currently expects one flat `tracker` object selected by `tracker.type`. It then falsely reported that `token_env` was required. The investigation also found that `connect` is difficult to automate, can echo secrets, replaces unrelated local settings, can copy an effectively merged token into committed `hero.json`, omits local connections from `--list`, and publishes stale command examples.

The immediate bug and the schema problem are one platform change. Hero needs a single integration contract that can be split consistently across committed and local layers, can be entirely local when a user does not want an integration enabled for everyone, and can grow to Jira-for-delivery plus Aha/Productboard-for-roadmap without another breaking rewrite. This spec supersedes the narrower [Jira connection diagnosis](../../bugs/jira-connection-onboarding-misleads-agents/spec.md); its confirmed defects and evidence are retained there as provenance and are mandatory scope here.

The active `harness-changes-cover-all-targets` tripwire applies: command/config guidance must remain consistent across opencode, cursor, claude, copilot, codex, and generic install targets.

## Goal

Replace the singular flat tracker configuration with a validated, provider-neutral integration map shared by `.hero/hero.json` and `.hero/hero.local.json`. Project configuration may declare any non-secret subset and local configuration may complete it, override it, or declare an entirely local integration. Hero must select integrations explicitly by default and role, resolve credentials without leaking them, preserve both files losslessly, support deterministic automation, and retain narrow read-only compatibility for existing tracker/confluence configurations.

## Kickoff

Introduces one provider-neutral integration schema shared by `hero.json` and `hero.local.json`, with local overlays, safe credentials, explicit role selection, and narrow legacy reads.

**Status:** delivering — implementation and validation have landed; the delivery lead is completing the cold audit and verification gates.

**Pick up at:** audit the completion ledger and persisted test evidence, then run `hero spec verify`.

→ `/deliver layered-integration-configuration`

**Files:** `internal/config/config.go`, `internal/config/credentials.go`, `internal/cli/connect.go`, `internal/tracker/tracker.go`
**Skip:** do not retain the flat `tracker.type + tracker.token` shape as canonical or pass secrets through argv.

## Problem

Hero currently conflates four separate concerns in `TrackerConfig`: provider identity, workspace selection, provider settings, and credentials. `hero.json` and `hero.local.json` deserialize into the same Go type but do not behave as a general layered contract: merge logic is manually field-listed, empty values cannot intentionally override, deletion is impossible, unknown keys disappear, and persistence can write the resolved effective object back to the wrong layer. Confluence uses another top-level shape, while global credentials use a third identity model. The result is ambiguous for one provider and structurally incapable of representing multiple integrations with distinct roles.

The natural `tracker.jira.token` shape was invalid, but Hero neither rejected nor migrated it. Fixing only that path would preserve the deeper trap: a future Aha integration would again force a choice between more one-off top-level config and a breaking multi-provider redesign. The platform contract needs to separate stable integration identity from provider type now.

## Approach

### Canonical schema

Both files use the exact same JSON schema. `integrations.connections` is a map keyed by a user-stable integration ID; each entry declares its provider. Selection is separate from definition: `default` handles commands that need one integration and `roles` maps semantic jobs to integration IDs. Provider-specific settings live under `settings`; credentials live under `auth`.

```json
{
  "integrations": {
    "default": "jira-delivery",
    "roles": {
      "delivery": "jira-delivery",
      "roadmap": "aha-roadmap"
    },
    "connections": {
      "jira-delivery": {
        "provider": "jira",
        "settings": {
          "project": "MORPH",
          "base_url": "https://example.atlassian.net",
          "user_email": "developer@example.com"
        },
        "auth": {
          "token_env": "JIRA_TOKEN"
        }
      }
    }
  }
}
```

The committed file may contain the shared entry without `auth`, while local config fills only the credential:

```json
{
  "integrations": {
    "connections": {
      "jira-delivery": {
        "auth": { "token": "<local-only>" }
      }
    }
  }
}
```

Or local config may define the entire integration, including its selector, so no integration is enabled for other clones:

```json
{
  "integrations": {
    "default": "personal-jira",
    "roles": { "delivery": "personal-jira" },
    "connections": {
      "personal-jira": {
        "provider": "jira",
        "settings": {
          "project": "MORPH",
          "base_url": "https://example.atlassian.net",
          "user_email": "developer@example.com"
        },
        "auth": { "token": "<local-only>" }
      }
    }
  }
}
```

Integration IDs, not provider names, are references. This permits two Jira projects or two Jira identities later without schema changes. Provider adapters own validation of `settings`; core owns `provider`, `auth`, selection, layering, and redaction. Initial role vocabulary is `delivery`, `roadmap`, and `docs`, but role keys remain validated against a central registry so typos fail and new Hero-owned roles can be added deliberately. Existing tracker sync resolves `roles.delivery`, then `default`; ambiguous or missing selection is an actionable error, never “first map entry wins.” A command may accept `--integration <id>` to override selection explicitly.

### Layering contract

Parse each file into a presence-aware raw document before decoding the effective typed configuration. Apply committed config first and local config second using JSON Merge Patch semantics within the `integrations` subtree:

- objects merge recursively by key;
- scalar and array values in local replace committed values, including `false`, `0`, and empty string;
- explicit `null` removes an inherited optional field or a whole connection/role entry;
- absent means inherit;
- after merge, references, required fields, provider settings, and layer-specific secret rules are validated;
- map traversal and rendered output are sorted for deterministic behavior.

Deletion is a local effective-view tombstone only; it does not modify `hero.json`. A local connection whose partial overlay lacks `provider` is valid only when that provider is inherited from the same connection ID. A complete local-only entry must declare `provider`. If local deletes the selected connection without replacing `default`/affected roles, validation reports the dangling reference with exact JSON paths.

The same merge engine should become the canonical layer mechanism for configuration fields migrated into presence-aware nodes; do not add another hand-coded list of non-zero fields. For this delivery, strict semantics are required for `integrations`; unrelated legacy config may remain on its current merge path until migrated, but must not be regressed.

### Security and persistence boundary

Literal `auth.token` is valid only in `.hero/hero.local.json` and the existing user credential store. It is a validation error in `.hero/hero.json`, regardless of `.gitignore` state. `auth.token_env` is a non-secret reference and may appear in either layer. Credential precedence is: local literal token, selected global credential, local/committed `token_env`; conflicting literal sources are reported in redacted status, with local taking precedence. Provider settings from global credentials fill only absent fields and never override either workspace layer.

Load APIs return distinct raw committed, raw local, effective redacted metadata, and secret-bearing runtime views. Save APIs require a target layer and accept a patch for that layer; no generic `Save(effectiveConfig)` path may write resolved state. Committed writers reject secret-bearing values. Local writes merge into the existing local document atomically, preserve unrelated keys, use mode `0600`, and never print secret values. Status serialization uses a secret wrapper/redactor so `%v`, JSON, errors, and debug output cannot reveal a token accidentally.

### Legacy compatibility (migrator explicitly descoped)

User decision on 2026-07-15: **“we don't need a migrator - there isn't anyone that needs migration but me”.** The explicit migrator, backups, dry-run/check workflow, and migration rollout are therefore removed from delivery scope. This is DESCOPE-APPROVED, not an implementation skip.

Keep only a narrow read-only compatibility path:

1. Loaders accept canonical `integrations` and current `tracker`/`confluence` shapes with one deprecation warning and no rewrite. Existing literal local tokens remain readable. Canonical and legacy declarations together fail with both conflicting paths. The unsupported `tracker.jira.token` shape gets a path-specific error and a safe canonical `hero connect` command.
2. All writers emit only canonical shape. Legacy and canonical declarations together fail with both paths so users resolve ambiguity explicitly.

No automatic rewrite occurs during ordinary `hero` commands. This prevents surprising diffs and preserves a clean rollback path.

### Connect and introspection contract

Keep the human wizard, but implement it on a shared request/validation/persistence service. Add automation-safe non-interactive inputs for provider, integration ID, role/default selection, and non-secret settings. Secrets may come from an already populated local/global entry, `token_env`, an interactive no-echo terminal prompt, or an explicit `--token-stdin`/file-descriptor source. Raw token argv flags are forbidden. A single injected reader serves the whole interactive session; secure secret input must not silently fall back to echoed input.

`hero sync connect` and its `hero connect` alias expose the same flags and help. `hero connect --list` and a machine-readable `--json` status inspect the effective project, local layer, global store, and environment readiness. Output includes integration ID, provider, selected roles/default, config sources by field, credential source/readiness, and verification state; it never contains token bytes or token-derived masks. A validation failure identifies the exact config file and JSON path and provides a current runnable command.

## Changes

### Phase 1 — Contract, parser, and secret-safe layer APIs

1. Introduce canonical integration models and raw-layer parsing under `internal/config`.
   - Add presence-aware `IntegrationsConfig`, `IntegrationConfig`, selection, settings, and auth representations; keep provider settings schema-owned by adapters.
   - Implement recursive overlay, `null` tombstones, deterministic ordering, layer provenance, and strict unknown-field validation.
   - Validate connection IDs, provider names, registered roles, dangling selectors, provider-required settings, and committed literal secrets with file/path-specific errors.
2. Separate committed, local, global-credential, and effective runtime APIs.
   - Replace effective-object persistence in integration call sites with target-layer patch APIs and atomic writes.
   - Preserve unrelated local configuration, enforce `0600`, reject secrets on committed writes, and add redaction-safe credential types.
   - Audit every `Config.Load` → `Save` call for resolved-secret propagation; add a guard test around generic serialization even where call sites are unchanged.
3. Add schema documentation and examples adjacent to config types and generated reference material.
   - Document partial project + local auth, fully local integration, multiple providers/roles, explicit override/deletion, and credential precedence.

### Phase 2 — Compatibility and runtime selection

4. Keep narrow read-only legacy compatibility; explicit migration tooling is DESCOPE-APPROVED.
   - Map flat `tracker` to `legacy-tracker`/`delivery` and `confluence` to `legacy-confluence`/`docs` without crossing layer boundaries.
   - Do not auto-rewrite on load. Canonical and legacy declarations together fail with both paths.
5. Change tracker/wiki consumers to resolve an integration by explicit ID/role/default.
   - Update adapter factories and sync/import/push/pull/sprint/vocabulary consumers to accept resolved integration settings rather than the global singular tracker.
   - Preserve current delivery behavior through normalized legacy selection; add `--integration` where users need explicit selection.
   - Key global credential lookup by stable integration identity with a compatibility lookup for existing `{provider}:{project}` entries.

### Phase 3 — Connection workflow and observed bug repairs

6. Rebuild `internal/cli/connect.go` around a shared connect request/service.
   - Add provider-general non-interactive fields, integration ID, default/role assignment, target layer, protected token sources, and JSON result output.
   - Use one reader for interactive prompts, suppress terminal echo for secrets, and make unavailable secure input an explicit error.
   - Persist only the requested patch to the requested layer; local-only mode writes the complete connection and selectors locally.
7. Make list/status and failures truthful and redacted.
   - Include project-local, global, and env-backed connections; report source and readiness without token masks or values.
   - Replace `token_env is not configured` with provider/integration-aware missing-credential guidance naming every supported source.
   - Reject `tracker.jira.token` and other unknown fields at load with exact canonical configuration guidance.
8. Correct docs and generated commands everywhere.
   - Fix tracker import examples to `hero sync import`, document canonical connect/status flows, and remove claims that environment variables are the only credential source.
   - Propagate shared guidance across all six harness targets and add the tripwire's propagation checks.

## Acceptance Criteria

- WHEN `hero.json` defines a connection without auth and `hero.local.json` defines `auth.token` for the same ID THE SYSTEM SHALL produce one valid effective integration while retaining each field's source layer.
- WHEN `hero.local.json` defines a complete connection and its `default` or role selector absent from `hero.json` THE SYSTEM SHALL enable that integration only for that user.
- WHEN local config supplies `false`, `0`, an empty string, or `null` THE SYSTEM SHALL apply the documented replace/delete behavior rather than treating the value as absent.
- IF a selector references a deleted or nonexistent connection THEN THE SYSTEM SHALL fail with the source file and exact selector/connection paths.
- THE SYSTEM SHALL select by explicit `--integration`, then requested role, then `default`; it shall never select by map iteration order.
- THE SYSTEM SHALL support multiple connections using stable IDs, including multiple entries with the same provider.
- IF `.hero/hero.json` contains a literal credential THEN THE SYSTEM SHALL reject it before adapter construction and shall not echo the value.
- WHEN any command writes committed config THE SYSTEM SHALL serialize only the committed layer and SHALL NOT serialize local, global, environment-resolved, or effective credentials.
- WHEN `connect` updates local config THE SYSTEM SHALL preserve unrelated local keys and connections and keep the file mode at `0600`.
- WHEN a legacy flat tracker/confluence config is loaded THE SYSTEM SHALL preserve current runtime behavior and emit one actionable deprecation warning without rewriting either file.
- DESCOPE-APPROVED: no explicit migrator is required per the user's 2026-07-15 decision: “we don't need a migrator - there isn't anyone that needs migration but me”.
- IF legacy and canonical fields conflict THEN THE SYSTEM SHALL stop with both paths and require explicit resolution.
- IF an unknown field such as `tracker.jira` or a misspelled canonical field exists THEN THE SYSTEM SHALL fail validation instead of silently discarding it.
- WHEN non-interactive connect is used THE SYSTEM SHALL accept non-secret fields deterministically and accept secrets only from a protected source, never a raw token argv value.
- WHILE an interactive secret is entered THE SYSTEM SHALL suppress terminal echo and use one stable input session for all prompts.
- WHEN list/status is requested THE SYSTEM SHALL report effective integration identity, selection, provenance, and credential readiness in human and JSON forms without token values, fragments, hashes, or masks.
- WHEN any config/connect diagnostic is emitted THE SYSTEM SHALL name supported credential sources and a current runnable recovery command.
- THE SYSTEM SHALL keep `hero connect` and `hero sync connect` behavior/help equivalent and use canonical `hero sync import` examples.
- WHERE harness-facing guidance changes THE SYSTEM SHALL validate equivalent propagation to opencode, cursor, claude, copilot, codex, and generic targets.

## Boundaries

- Do not implement Aha or Productboard adapters in this work; make their future addition fit the contract without schema change.
- Do not make one command execute against multiple delivery trackers simultaneously. Multiple definitions and role-based selection are in scope; fan-out orchestration and cross-provider issue identity are not.
- Do not introduce OAuth, OS-keychain storage, or a general external secret broker. The schema may add credential methods later without moving provider settings.
- Do not accept secrets through chat, ordinary argv flags, committed config, logs, status output, or migration diffs.
- Do not migrate every unrelated top-level Hero config section to JSON Merge Patch in this delivery; the new integration subtree must establish the reusable mechanism without an opportunistic whole-config rewrite.
- Do not silently rewrite user files during load or remove legacy read support in the same release.

## Risks

- Presence-aware layering is a contract change: decoding directly into ordinary Go value fields will lose absent-versus-zero semantics. Keep raw layer documents distinct until merge completes.
- Strict unknown-field validation can expose pre-existing typos that were ignored. Legacy recognized shapes need explicit adapters, and errors must be corrective rather than generic.
- A global credential keyed only by old provider/project identity may be ambiguous when two integration IDs target the same project. Compatibility lookup must fail closed on ambiguity.
- Provider-specific settings hidden in an untyped map would defer errors too late. Each registered adapter must validate its settings schema at config-load time.
- Redaction is easiest to break through generic formatting or test failure output. Secret-bearing types and negative-output tests must cover every CLI/result path.
- Rollback to an old binary cannot understand canonical config. No automatic rewrite keeps legacy workspaces untouched; canonical adopters can revert their explicit configuration change with version control.
- This is a large feature spanning config and every tracker consumer. Deliver in the specified phases with passing compatibility tests at each boundary; do not ship a canonical writer before the dual reader is stable.

## Validation

- Add table-driven merge tests covering absent, zero, empty, override, object merge, `null` field deletion, whole-entry deletion, dangling references, deterministic order, and complete local-only config.
- Add strict-schema tests for unknown core fields, unknown provider settings, unknown roles/providers, exact JSON paths, and legacy `tracker.jira.token` guidance.
- Add secret-boundary tests that seed recognizable canary tokens and assert absence from `hero.json`, stdout/stderr, JSON status, errors, snapshots, and generic formatting.
- Test project/local/global/env precedence and provenance, including ambiguous old credential keys and unset environment variables.
- Test legacy flat tracker and confluence configurations without mutating their source files.
- Test that legacy reads remain non-mutating and canonical/legacy conflicts name both paths; explicit migration tests are DESCOPE-APPROVED.
- Add CLI tests with injected readers/writers for interactive Jira, GitHub, Linear, GitLab, and Confluence; cover multiline input, no-echo secret handling, protected stdin/FD, local-only definition, shared+local completion, failed verification, aliases, and machine JSON.
- Run mock-provider end-to-end flows for connect → status → `hero sync import --dry-run` → import using canonical, partial-overlay, fully-local, and legacy configs.
- Audit and test all consumers found by `rg 'TrackerConfig|\.Tracker|Confluence' internal cmd`; no consumer may bypass integration selection or persistence boundaries.
- Run `go test ./internal/config ./internal/tracker ./internal/wiki ./internal/cli`, then `go test ./...`, config/schema lint, documentation examples, and all-six-target harness propagation checks.
- Manually inspect representative shared/local files and verify secret separation and local permissions.

## Rollout and rollback

Ship dual-read capability with canonical writers and updated consumers. Watch config-load failures, conflict counts, adapter initialization failures, and secret-boundary test results. Abort canonical writes if load or initialization errors materially exceed baseline; the old read path remains available.

Rollback means reverting the binary and any explicit canonical config commit. Ordinary commands never rewrite legacy files, so rolling the new binary back does not mutate legacy data. Removal of legacy reads is a later major-version decision.

## Design decisions

- **Stable connection IDs over provider-keyed singleton objects:** provider names alone cannot represent two Jira projects/identities; IDs plus `provider` handle both today's case and future expansion.
- **Selection separate from connection definitions:** `default` and semantic `roles` state intent explicitly and avoid implicit “one tracker” behavior.
- **Same schema, different layer policy:** both files compose predictably; security validation, not a different JSON shape, prevents committed literals.
- **JSON Merge Patch semantics:** absent, zero, empty, and delete each have deterministic meanings familiar to tooling and suitable for partial overlays.
- **No load-time rewrite:** compatibility remains reversible and ordinary commands do not surprise users with config diffs; an explicit migrator is DESCOPE-APPROVED.
- **One coherent platform delivery:** schema, persistence, migration, connect, diagnostics, and the confirmed Jira defects share the same trust boundary; shipping only one slice would leave unsafe or contradictory behavior.

## Completion Ledger

Full test commands, exercise notes, and criterion-level evidence are recorded in
[`test-evidence.md`](test-evidence.md). The cold audit verified this ledger and
returned `SHIP` with 19/19 acceptance criteria.

### Acceptance Criteria

| Criterion | Status | Evidence |
|---|---|---|
| Shared project settings plus local auth | DONE | `TestIntegrationOverlayAndProvenance` |
| Complete local-only integration | DONE | `TestNonInteractiveConnectAllProvidersLocalOnly` |
| False, zero, empty, and null overlay semantics | DONE | merge-patch and null-deletion tests |
| Dangling selector errors include exact paths | DONE | `TestIntegrationNullDeletesAndDanglingSelector` |
| Explicit, role, then default selection | DONE | `ResolvedIntegrations.Select`, sync `--integration`, stable-ID tests |
| Multiple stable IDs, including one provider twice | DONE | two-Jira selection and credential ambiguity tests |
| Committed literal credentials are rejected and redacted | DONE | `TestCommittedSecretRejectedWithoutEcho` |
| Committed writes exclude all effective credentials | DONE | `TestLoadMutateSaveNeverCommitsEffectiveTrackerOrConfluenceSecrets` |
| Local writes preserve unrelated data and mode `0600` | DONE | `TestPatchLocalIntegrationsPreservesKeysAndMode` |
| Legacy reads remain compatible, warning-only, and non-mutating | DONE | legacy load and warning tests |
| No explicit migrator; retain narrow read-only compatibility | DONE | Revised requirement approved by user: “we don't need a migrator - there isn't anyone that needs migration but me” |
| Canonical and legacy conflicts fail with both paths | DONE | conflict regression tests |
| Unknown fields and invalid provider settings fail strictly | DONE | all-provider path/type/required-field validation table |
| Non-interactive connect uses protected secret input | DONE | all-provider `--token-stdin` tests and canary-output assertions |
| Interactive secret input is no-echo with one stable reader | DONE | shared-reader tests and `term.ReadPassword` boundary |
| List/status is truthful, source-aware, and fully redacted | DONE | JSON/human status tests and fresh-workspace exercise |
| Credential errors name all sources and recovery commands | DONE | `CredentialGuidance` and unknown-path tests |
| Connect aliases and canonical import examples agree | DONE | alias parity tests and documentation scan |
| Guidance propagates to all six harness targets | DONE | `TestHarnessNative_DoctorRoutingGuidanceAllTargets` |

### Changes

| Change | Status | Evidence |
|---|---|---|
| 1. Canonical models, parser, overlay, and validation | DONE | `internal/config/integrations.go` plus all-provider tests |
| 2. Layer/runtime APIs and persistence boundaries | DONE | target-layer patches and tracker/Confluence generic-save canary |
| 3. Schema documentation and examples | DONE | configuration and tracker setup documentation |
| 4. Narrow legacy normalization without an explicit migrator | DONE | Read-only compatibility retained; migrator removed by user decision |
| 5. Consumer integration selection | DONE | sync import/push/tracker operations honor `--integration` |
| 6. Shared connect workflow | DONE | interactive/non-interactive provider-general service and tests |
| 7. Truthful status and repaired diagnostics | DONE | redacted provenance/readiness output and error tests |
| 8. Docs and generated guidance parity | DONE | canonical docs plus six-target propagation test |

### Exercise the feature

- [x] DONE — A newly built CLI connected a fresh temporary Jira workspace using
  protected stdin, wrote split `0644`/`0600` committed/local files, returned
  redacted JSON provenance, and honored explicit `sync --integration`; the
  intentional `.invalid` endpoint failed only after reaching the Jira adapter.
