---
title: "Jira connection onboarding misleads agents and can leak the token"
slug: jira-connection-onboarding-misleads-agents
type: bug
status: superseded
domain: engineering
root_cause_class: design
severity: high
size: medium
priority: high
horizon: now
created: 2026-07-14
tags: [jira, tracker, credentials, cli, agent-ux, security]
relates-to: [hero-self-consistency, generated-command-refs-validated]
superseded-by: layered-integration-configuration
---

# Jira connection onboarding misleads agents and can leak the token

> **Superseded 2026-07-15:** The investigation and reproduction remain the evidence record, but delivery is folded into [Layered Integration Configuration](../../features/layered-integration-configuration/spec.md). That feature replaces the singular tracker schema and includes every confirmed onboarding, automation, persistence, redaction, listing, and documentation defect below.

## Issue

A fresh Hero project with Jira tracker fields configured could not be connected by an agent running the Homebrew `hero` CLI. The agent saw `tracker token_env is not configured`, concluded `hero sync import` required an environment variable, failed to automate `hero sync connect jira` through piped input/fake TTY, and told the user to either run the command manually or export `JIRA_TOKEN`. The user explicitly did not want an environment-variable credential. The later chat transcript established that the token was present at the intuitive but unsupported path `.hero/hero.local.json` → `tracker.jira.token`; the implemented single-active-tracker schema expects `tracker.token` and silently ignores the extra `jira` object.

Found from a Codex desktop session screenshot on 2026-07-14. No tracker ticket is attached. The behavior is provider-general in the shared connect/config code, but Jira exposes it most sharply because setup needs four values and basic-auth identity.

## Goal

Make tracker connection truthful, automation-safe, and secret-safe. A fresh-session agent must be able to determine every supported credential source, configure Jira without attempting fragile TTY scripting or placing a token in chat/argv/environment, verify the effective connection, and import issues. Project-local connection must preserve existing local settings and must never serialize a literal token into committed `hero.json`.

## Kickoff

Fixes fresh Jira setup so agents can use the supported local credential path without inventing a `token_env` requirement, fragile TTY piping, or leaking the token into committed config.

**Status:** planning — diagnosis is complete; no code has changed.

**Pick up at:** add regression tests for secret-safe config persistence and missing-token guidance, then design the non-interactive connect input contract around a protected token source.

→ `/deliver jira-connection-onboarding-misleads-agents`

**Files:** `internal/cli/connect.go:16`, `internal/config/config.go:1088`, `internal/cli/sync_import.go:107`, `internal/config/credentials.go:76`
**Skip:** do not make `token_env` mandatory, pass tokens in argv, or redesign sync for simultaneous trackers in this bug; literal `tracker.token` in `hero.local.json` is already supported for the tracker selected by `tracker.type`.

## Summary

### Categorization

| Attribute | Assessment |
|-----------|------------|
| **Criticality** | high — onboarding is blocked for agents and the successful project-local path can copy a valid token into committed `hero.json` |
| **Ease of Fix** | moderate — errors and tests are small, but safe non-interactive input and lossless split-config persistence need an explicit contract |
| **Caused by our codebase?** | Yes — shared CLI/config behavior creates both the false guidance and unsafe write path |
| **Needs more research?** | No — the reported failure, credential resolution, write path, and secondary defects are confirmed in source or locally reproduced |

### Background

Hero's workspace schema intentionally selects one active tracker through the discriminator `tracker.type`; the remaining fields in the same `tracker` object configure that selected provider. Separately, the global credential store can retain many `{type}:{project}` entries. Hero supports three credential sources for the active tracker: a literal local token, a matching global credential entry, or an environment variable. The product does not explain this active-config-versus-saved-connections distinction, does not reject the plausible provider-nested shape, exposes only an interactive connection wizard, and emits an error naming just one credential source. That combination predictably sends a cold agent toward both the wrong JSON shape and an unwanted environment-variable/TTY workflow.

### Analysis

`sync import` does not require `token_env`. It loads committed config, deep-merges `.hero/hero.local.json`, applies the global credential store, and then resolves `tracker.token` before consulting `tracker.token_env`. In the reported project, standard JSON decoding ignored `tracker.jira` because `TrackerConfig` has no such field, so the literal token never entered the effective config. No schema validation identified the wrong path; the resulting error was nevertheless `tracker token_env is not configured`, which falsely diagnosed the credential source instead of the malformed/unsupported local shape. `connect jira` has no setup flags or stdin/documented machine mode; each ordinary prompt creates a fresh buffered scanner, and the secret prompt deliberately switches to `/dev/tty`. Multi-line piped input therefore loses buffered lines at the second prompt and cannot supply the secret when a controlling terminal exists.

The successful default connect path has a more serious defect: it writes the token to `hero.local.json`, reloads the merged config (now containing the token), and saves that merged object to `hero.json` while adding non-secret fields. Because `TrackerConfig.Token` is serialized when non-empty, a valid Jira token can land in the committed file despite help text promising otherwise.

### Root Cause

The fundamental root cause is **design**: Hero has two different multiplicity models—one active workspace tracker selected by `tracker.type`, but many saved credentials keyed by `{type}:{project}`—without a validated or clearly documented configuration contract connecting them. The permissive decoder silently accepts the natural-looking `tracker.jira.token` shape, then credential resolution reports as though `token_env` were the only contract. Connection remains a human-only wizard, and the implementation lacks a safe split-config write boundary, allowing resolved secret state to cross back into committed configuration. The reported token was therefore not at a supported path, but Hero converted that actionable configuration error into misleading guidance; Jira and the user's expectation of provider-keyed credentials are not the underlying failure.

### Source

The defect spans `internal/cli/connect.go` (interactive acquisition, listing, persistence), `internal/config/config.go` (layer merge, token resolution, generic save), `internal/config/credentials.go` (global credential fallback), and `internal/cli/sync_import.go` (consumer and stale command examples).

### Fix Direction

Define and validate the single-active-tracker contract: `tracker.type` selects the provider and sibling `tracker.*` fields configure it, while the global credential store may contain many provider/project entries but only the matching entry is applied. Reject unknown provider-nested fields such as `tracker.jira` with a migration-safe, path-specific error (or explicitly normalize them if compatibility policy requires), and use the same credential-source contract in errors, status/list output, docs, and tests. Add a non-interactive Jira setup path whose non-secret fields can be flags/config and whose token arrives through a protected source such as an existing local/global credential entry or a dedicated stdin/FD mechanism—not argv or chat. Separate committed-config writes from effective merged config, merge project-local updates losslessly, suppress terminal echo for secrets, and provide a connection/status check that reports source and readiness without revealing token material.

---

## Problem Statement

### Minimal reproduction

1. In a Hero workspace, configure `.hero/hero.json` with `tracker.type: jira`, `project`, `base_url`, and `user_email`, but omit `token_env`.
2. Put the API token at the provider-nested path used in the reported project:

   ```json
   {"tracker":{"jira":{"token":"<redacted>"}}}
   ```

3. `hero sync import` silently ignores `tracker.jira`, because `TrackerConfig` is a flat discriminated object and Go's default JSON decoder permits unknown fields.
4. Adapter construction then fails with `initializing tracker: tracker token_env is not configured`, falsely presenting an environment variable as mandatory instead of identifying `tracker.jira.token` as unsupported. Moving the same redacted value to `tracker.token` is the implemented literal-token contract and unit tests confirm it wins over `token_env`.
5. Attempt automation with `printf 'https://example.atlassian.net\nPROJ\nuser@example.com\nfake-token\n' | hero connect jira`.
6. Observed locally: the first prompt consumes the URL and the next prompt receives no line, ending with `project key is required`. A fake TTY does not solve the secret boundary because `promptSecret` explicitly opens `/dev/tty`.

### Expected

- Missing-credential diagnostics name all supported safe sources and a runnable next action.
- The config contract makes clear that a workspace has one active tracker selected by `tracker.type`, while saved global credentials may contain multiple provider/project entries.
- A provider-nested credential such as `tracker.jira.token` is rejected or migrated explicitly, never silently ignored.
- A non-interactive agent can configure non-secret Jira fields and point Hero at a protected token source without handling the token value in chat or argv.
- `hero connect --list`/status recognizes project-local and global connections without exposing secrets.
- Connection writes preserve existing local config and never put a literal token into committed config.

### Actual

- Error text names only `token_env`.
- Unknown `tracker` members are silently discarded, so the plausible `tracker.jira.token` shape becomes indistinguishable from no credential.
- Docs conflict: generated/current CLI code supports local and global literal credentials, while the tracker setup page says tokens are always environment variables and never documents the flat local shape or single-active-tracker model.
- Setup is interactive-only and piped stdin fails on the second ordinary prompt.
- The secret prompt reads `/dev/tty` and leaves terminal echo enabled.
- Listing inspects only global credentials, so it can say no connections exist when `hero.local.json` is valid.
- Default local connection can overwrite unrelated local settings and copy the newly stored token into `hero.json`.

## Environment Details

- Repository: Hero engine, Go/Cobra CLI.
- Installed command: Homebrew `/opt/homebrew/bin/hero` as shown in the report; the installed binary has no `hero version` command, so exact build identity cannot be printed.
- Local source and installed help both expose `hero connect jira` and canonical `hero sync connect jira` as interactive commands.
- Harness: Codex desktop, where tool execution is non-interactive unless a user enters data in a terminal.
- Jira network/auth success was not exercised because no real credential was needed to prove the local control-flow defects.

---

## Root Cause Analysis

### Load-bearing claims

| Claim | Grounding | Evidence |
|---|---|---|
| `sync import` accepts a literal local token and does not require `token_env` | read + tested | `Config.Load` merges local config, `ResolveToken` returns `Token` first; focused config/tracker tests pass |
| a workspace intentionally has one active tracker, not a map of simultaneous tracker adapters | read + history | `Config` has one `*TrackerConfig`; `tracker.New*` switches once on `cfg.Type`; every sync consumer constructs one adapter; this shape dates to the original tracker integration |
| multiple saved connections are supported independently of active tracker selection | read + tested | global `Credentials` is a map keyed by `{type}:{project}`, but `ApplyCredentials` selects only the entry matching the active `cfg.Tracker.Type` and `.Project` |
| `tracker.jira.token` is unsupported and silently ignored | read + reproduced from decode semantics | `TrackerConfig` has no `jira` field and config decoding uses `json.Unmarshal` without `DisallowUnknownFields` |
| the missing-token message drives the agent toward env vars | read + observed in report | both nil/no-source branches say only `tracker token_env is not configured` |
| piped multi-line setup fails before the token prompt | read + reproduced | each `prompt` creates a new buffered scanner; reproduction fails at project key |
| fake TTY/piped stdin cannot reliably provide the secret | read | `promptSecret` preferentially opens `/dev/tty`, bypassing stdin |
| project-local connect can leak the token into `hero.json` | read | `SaveLocal(token)` → `config.Load` merges token → `cfg.Save` serializes merged `TrackerConfig.Token` |
| project-local connect overwrites unrelated local settings | read | `saveConnection` constructs a zero-value local config and `SaveLocal` replaces the entire file |
| `connect --list` ignores valid project-local credentials | read | `runConnect` loads only global `Credentials`; `runConnectList` receives no project config |

No load-bearing claim remains assumed.

### Confirmed findings

1. **Ambiguous, unvalidated schema contract:** the runtime is consistently single-active-tracker: `Config.Tracker` is singular, `TrackerConfig.Type` is the provider discriminator, adapter factories switch on it once, and sync consumers receive one adapter. The multi-entry global credential map is storage for reuse across workspaces/projects, not simultaneous workspace integration. However, unknown members are silently accepted, public docs do not explain this distinction, and the tracker setup page contradicts literal local/global credential support. Thus `tracker.jira.token` looks structurally reasonable but disappears without a diagnostic.
2. **False credential diagnosis:** `TrackerConfig.ResolveToken` explicitly prioritizes `Token`, yet its no-token error names `token_env` exclusively (`internal/config/config.go:1088-1104`). After the invalid nested field is discarded, this is the immediate reason the agent recommends an unwanted env var.
3. **No agent-safe setup surface:** Jira setup accepts no base URL, project, email, token source, or non-interactive flags (`internal/cli/connect.go:16-51,191-226`).
4. **Broken stdin composition:** `prompt` constructs and discards a new buffered scanner for each question (`internal/cli/connect.go:567-574`). The first scanner may read ahead, so later scanners see EOF. The local reproduction consistently failed at the second prompt.
5. **TTY override and visible secret:** `promptSecret` opens `/dev/tty` when possible and intentionally does not suppress echo (`internal/cli/connect.go:576-597`). This defeats stdin automation and displays the token in terminal scrollback/screenshares.
6. **Secret crosses the persistence boundary:** local connect saves the secret, then `updateHeroJSON` reloads the effective merged config and calls the generic committed save (`internal/cli/connect.go:340-398`; `internal/config/config.go:1381-1399,1488-1492`). The merged tracker has `Token` populated, so JSON marshaling includes it.

### Classification

`root_cause_class: design`. The user's file is invalid under the current schema, but silent unknown-field acceptance, contradictory guidance, the misleading resolver error, missing non-interactive setup, and unsafe persistence are Hero defects. The evidence does not support simultaneous trackers as the intended runtime model; adding them would be a separate feature rather than the repair for this incident.

---

## Code Flow (End to End)

1. `internal/cli/sync_import.go:107-139` — `hero sync import` finds the project, loads config, validates tracker type, then constructs the adapter.
2. `internal/config/config.go:20-37,1020-1035` — `Config` contains one `Tracker`; `TrackerConfig.Type` discriminates the active provider and all provider connection fields are siblings in that object.
3. `internal/config/config.go:1381-1399,1495-1515` — `Config.Load` reads committed `hero.json` and `.hero/hero.local.json` with permissive `json.Unmarshal`; the unknown reported member `tracker.jira` is discarded without error.
4. `internal/config/config.go:1537-1565` — supported local `tracker.token`, `token_env`, base URL, and user email fields merge onto the singular committed tracker.
5. `internal/config/credentials.go:18-20,69-99` — the global store can retain many `{type}:{project}` entries, but only the entry matching the active tracker is applied.
6. `internal/tracker/tracker.go:165-230` — the adapter factory switches once on `TrackerConfig.Type`; Jira adapter construction calls `ResolveToken` before creating the sole adapter.
7. `internal/config/config.go:1088-1104` — literal `tracker.token` succeeds first; after the provider-nested token was ignored, the no-source branch returns the misleading `token_env` error consumed by the agent.
8. `internal/cli/connect.go:54-88,191-226` — the suggested recovery routes to the Jira interactive wizard with no non-interactive field contract.
9. `internal/cli/connect.go:567-597` — fresh scanners lose piped read-ahead and the token prompt switches to `/dev/tty`, causing the agent's scripted/fake-TTY attempts to fail.
10. `internal/cli/connect.go:340-384` — successful local setup creates a new partial config and replaces `hero.local.json`, potentially deleting unrelated local state.
11. `internal/cli/connect.go:387-398` — setup then reloads the effective config, including the secret just written locally.
12. `internal/config/config.go:1476-1492` — generic `Config.Save` marshals that effective config to committed `.hero/hero.json`, including non-empty `tracker.token`.
13. `internal/cli/connect.go:62-64,95-114` — later `connect --list` sees only the global store and cannot confirm the local connection the command created.

---

## Key Files

### Connection CLI

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/connect.go` | 16-114 | command contract and global-only listing |
| `internal/cli/connect.go` | 191-226 | Jira-only interactive acquisition and verification |
| `internal/cli/connect.go` | 340-450 | destructive local save and merged committed save |
| `internal/cli/connect.go` | 567-597 | piped input loss, `/dev/tty` override, echoed secret |
| `internal/cli/connect_alias.go` | 5-18 | top-level/canonical alias relationship |

### Configuration and tracker construction

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/config/config.go` | 1020-1026 | token and `token_env` are both supported fields |
| `internal/config/config.go` | 1088-1104 | token precedence and misleading error |
| `internal/config/config.go` | 1381-1492 | effective config load followed by generic committed save |
| `internal/config/config.go` | 1495-1565 | local file replacement and nested tracker merge |
| `internal/config/credentials.go` | 76-99 | global credential fallback |
| `internal/tracker/tracker.go` | 165-230 | all adapter constructors resolve the same credential contract |

### Consumer and guidance

| File | Lines | Relevance |
|------|-------|-----------|
| `internal/cli/sync_import.go` | 18-66 | help examples incorrectly say `hero import`, which is a different top-level command |
| `internal/cli/sync_import.go` | 107-139 | import initialization surfaces the misleading token error |
| `README.md` | 253-255 | grouped tracker commands are accurate but omit credential-source/non-interactive guidance |
| `GETTING-STARTED.md` | 280-299 | connect/import happy path omits credential setup and verification details |
| `web/docs/src/configuration/tracker-setup.md` | 1-76 | documents only `token_env`, incorrectly says tokens are always environment-backed, and omits the single-active-tracker/local credential shape |

## Changes

1. Make the active-tracker schema explicit and diagnosable.
   - Document `tracker` as one flat discriminated object: `type` selects the workspace's active provider; `project`, `token_env`, `base_url`, `user_email`, and local-only `token` configure that provider.
   - Document the global credential map separately as multiple saved `{type}:{project}` connections, only one of which is selected by the workspace config.
   - Detect unknown/provider-nested tracker members such as `tracker.jira`; fail with an exact path and corrected redacted example, or normalize them through an explicit compatibility migration. Do not silently discard them.
   - Do not expand this bug into simultaneous multi-tracker sync; that requires a separate contract for issue-ID namespaces, per-command selection, spec routing, and adapter lifecycle.
2. Make committed and local config persistence secret-safe and lossless.
   - Add a config API that reads/writes the committed layer without local/global resolution, or explicitly strips secret-only fields before saving committed config.
   - Change local connection writes to load the existing local layer, merge only the intended tracker/confluence credential fields, and preserve unrelated local settings.
   - Add regression tests proving valid tokens never appear in `hero.json`, existing local sections survive, and file modes remain `0600`.
3. Define an automation-safe `connect` contract shared by `hero connect` and `hero sync connect`.
   - Accept non-secret provider fields non-interactively and allow the secret to be sourced from a pre-existing local/global entry or a protected stdin/file-descriptor mechanism.
   - Do not accept raw tokens as ordinary argv flags, where they appear in shell history/process listings.
   - Keep the guided wizard for humans, but share validation/persistence logic with the non-interactive path.
4. Repair prompt handling and secret input.
   - Use one stable reader for an interactive session instead of constructing a scanner per prompt.
   - Suppress echo when reading a secret and make the behavior explicit when no secure terminal is available.
   - Test interactive and non-interactive modes with injected readers/writers rather than real TTY dependence.
5. Make diagnostics and status reflect the real credential contract.
   - Replace `tracker token_env is not configured` with provider-aware guidance naming local config, global credentials, and environment fallback without printing values.
   - Make `connect --list` or a dedicated status/check command report effective project-local/global/env readiness and source, redacted.
   - Ensure `sync import` wraps failures with a runnable recovery command suitable for both humans and agents.
6. Align all generated and user-facing guidance.
   - Correct `sync import` help examples from dead/ambiguous `hero import` forms to `hero sync import`.
   - Document Jira Cloud fields, the local/global credential choices, secure automation, and a redacted verification step in README, getting-started, web docs, and installed shared command guidance.
   - Because this touches harness-facing guidance, verify propagation across all six install targets per the active `harness-changes-cover-all-targets` tripwire.

## Secondary Defects

1. **High — valid secret may be committed.** The default `connect` path's effective-config reload/save can serialize `tracker.token` into `.hero/hero.json`.
2. **Medium — local config is destructively replaced.** Connecting a tracker can erase local Confluence, model, team, or other personal settings because `SaveLocal` receives a new sparse object and overwrites the file.
3. **Medium — secret input is echoed.** The code explicitly avoids echo suppression, exposing API tokens in terminal scrollback and screenshares.
4. **Medium — connection listing lies by omission.** `hero connect --list` says it shows saved connections but only examines `~/.config/hero/credentials.json`, not the default project-local store.
5. **Medium — import help names the wrong command.** `syncImportCmd` examples repeatedly say `hero import`, but top-level `hero import` imports knowledge from a URL/file/directory; the canonical tracker command is `hero sync import`.
6. **Low — release identity is unavailable.** `hero version` does not exist, making Homebrew/source drift harder to diagnose when tracker behavior differs between installations.
7. **Medium — tracker setup documentation contradicts runtime.** It says tokens are always environment-backed even though literal project-local and global credential sources have existed since the original integration.

## Boundaries

- Do not redesign Jira issue search/import semantics or Jira custom-field mapping.
- Do not add OAuth, OS-keychain integration, or a general credential broker in this bug unless separately designed.
- Do not accept credentials through chat, command-line argv, committed config, or logs.
- Do not remove environment-variable support; it remains a valid fallback, just not the only documented path.
- Do not add simultaneous multi-tracker execution to this bug. Current code and history consistently implement one active tracker; a multi-active model needs its own feature design.
- Provider-general shared fixes are in scope; provider-specific behavior unrelated to connection onboarding is not.

## Risks

- Changing config save semantics can affect every command that loads effective config and later saves it; audit call sites for secret propagation rather than patching only Jira.
- Non-interactive secret transport can create a new leak if implemented as `--token VALUE`; tests must inspect argv/help/output and committed artifacts for token absence.
- Listing env-backed readiness must not reveal token contents and should distinguish configured-but-unset from resolved.
- Existing users may rely on interactive prompts; preserve that path while making its reader and echo behavior safe.
- Harness-facing docs/guidance changes must cover `opencode`, `cursor`, `claude`, `copilot`, `codex`, and `generic` rather than one generated instruction file.

## Validation

- Unit-test `ResolveToken` success and errors for literal local token, global credential, set/unset environment variable, and no source.
- Add config-contract tests for the supported flat local shape and for `tracker.jira.token`/other unknown provider-nested members; assert the latter produces a path-specific diagnostic or explicit migration result.
- Add factory/consumer tests proving `tracker.type` selects one adapter and credential lookup chooses the matching `{type}:{project}` entry even when other providers are stored.
- Add CLI tests for Jira setup using injected input: interactive wizard, non-interactive protected-token source, missing fields, failed verification, and both top-level aliases.
- Assert piped/machine mode consumes all intended fields deterministically and never switches input sources unexpectedly.
- Assert secret values are absent from stdout/stderr, `hero.json`, process-facing examples, and snapshots; assert local/global files retain `0600`.
- Seed unrelated sections in `hero.local.json`, connect a tracker, and verify byte-semantic preservation after reload.
- Connect locally and globally, then verify list/status reports both sources with redaction.
- Run a mock Jira end-to-end: connect, `hero sync import --dry-run`, normal import, list/status, and reconnect/update.
- Run focused packages (`go test ./internal/config ./internal/tracker ./internal/cli`) and the full suite before delivery.
- Run install-target propagation checks for any shared harness guidance changed.

## Notes

### Tripwire check

`hero anchor "Jira tracker credential onboarding, non-interactive CLI configuration, secret storage, hero.local.json merging, and accurate missing-token diagnostics"` returned one active high tripwire: harness-facing guidance must cover every install target. The fix direction therefore places canonical behavior in shared CLI/config code and requires all-target propagation testing for generated guidance; it does not propose a Codex- or Claude-only workaround.

### Evidence and tests run during diagnosis

- `hero connect --help` and `hero sync connect --help`: confirmed interactive-only flags and local/global storage claims.
- Piped four-line `hero connect jira` reproduction: confirmed failure at the second prompt with `project key is required`.
- `go test ./internal/config -run 'TestTrackerResolveToken|TestMergeLocal_TrackerToken|TestLoad_AppliesLocalOverride|TestLoad_Credential' -count=1`: passed.
- `go test ./internal/tracker -run 'TestNew' -count=1`: passed.
- `go test ./internal/cli -run 'Test.*Connect|TestSyncImport' -count=1`: passed, demonstrating the current suite lacks behavioral coverage for the reproduced connect path and unsafe persistence.
- Source/history review: all relevant connect and token-resolution lines date to the original tracker integration commit; this is not a recent Jira API regression.
- Challenge source pass: confirmed `Config.Tracker` is singular, every adapter factory switches on `tracker.type`, every sync consumer constructs one adapter, and the global credential map's multiplicity is storage-only because `ApplyCredentials` selects the active `{type}:{project}` key.
- Challenge docs/history pass: confirmed the original integration introduced the single-discriminator config and multi-entry credential store together; no spec, test, or later change describes simultaneous workspace trackers. Public tracker setup docs instead contradict runtime by claiming environment variables are the only credential source.

## Investigation History

### Round 1 — Initial diagnosis
- **Date**: 2026-07-14T23:00:00Z
- **Agent**: debug-investigator
- **Root cause**: Hero's multi-source credential resolution is exposed through misleading `token_env`-only diagnostics, an interactive-only wizard, and an unsafe effective-config persistence boundary.
- **Confidence**: High
- **Key evidence**: `ResolveToken` accepts `tracker.token`; prompt automation fails in the shared scanner/TTY flow; local connect can merge a token into the object later saved as committed config.

### Round 2 — Challenged (layer)
- **Date**: 2026-07-15T15:01:29Z
- **Agent**: debug-investigator
- **Challenged by**: engineer
- **Feedback**: "oh - well that seems a bug becuase how would you have different trackers integrated if the type wasn't in there no?"
- **Revised root cause**: Hero intentionally separates one active tracker (`tracker.type` plus flat sibling fields) from many saved `{type}:{project}` credentials, but never validates or explains that distinction; it silently discards the natural `tracker.jira.token` shape and misreports the result as a `token_env` requirement, alongside the previously confirmed onboarding and persistence defects.
- **What changed**: The reported file did not contain the previously assumed supported `tracker.token`; it contained unsupported `tracker.jira.token`. The engineer's multi-tracker concern was tested across config, factories, consumers, credentials, tests, docs, specs, and history. Evidence supports a missing schema-validation/guidance contract, not an existing or intended simultaneous-tracker runtime. The original misleading-error, automation, and secret-persistence findings remain valid and are now layered under the corrected immediate trigger.
- **Confidence**: High

## Recap

The reported `tracker.jira.token` is not supported by Hero's current single-active-tracker schema; the valid local path is flat `tracker.token`, with the provider selected by `tracker.type`. That invalid shape should not have failed silently: contradictory docs, permissive decoding, and a `token_env`-only error converted it into an onboarding dead end, while the previously confirmed automation and secret-persistence defects keep severity high. Simultaneous workspace trackers are not evidenced as current intent and remain outside this bug.
