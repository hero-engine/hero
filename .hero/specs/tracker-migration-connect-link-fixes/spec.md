---
title: "Tracker-migration rough edges: connect persists unreadable config, and sync link can't re-point or take a spec dir"
slug: tracker-migration-connect-link-fixes
type: bug
status: completed
domain: engineering
root_cause_class: design
severity: high
size: small
priority: high
horizon: now
created: 2026-07-16
tags: [gitlab, tracker, migration, connect, sync-link, config-validation, cli, ergonomics]
relates-to: [layered-integration-configuration, jira-connection-onboarding-misleads-agents]
delivery_method: manual
completed_at: 2026-07-17T05:32:22Z
---

# Tracker-migration rough edges: connect persists unreadable config, and sync link can't re-point or take a spec dir

## Issue

Found while wiring GitLab tracker support on hero **v0.26.0**, migrating a project
(noeta-studios/chronecho) off a dead Linear tracker onto its GitLab equivalent.
No external tracker ticket — discovered during development. Three small, contained
defects on the Linear→GitLab migration path, all shippable in one PR.

The GitLab connector itself **works**: verification hit the live GitLab API,
linking succeeded once the config was hand-repaired, and a sync dry-run went from
25 spurious creates to 15 correct ones. These are rough edges on a working feature —
frame them that way. Background: on the old singular `tracker.type` schema with no
GitLab connector, `tracker.type` had to sit at `"none"`, so nothing synced for a
month and the board drifted to 139 specs vs. 25 issues. The connector closes that
gap; these three defects are what a real migrant hits on the way through.

## Investigation

Three defects, each independently verified against the code below.

### Defect 1 — `hero connect` persists a config the read path rejects (write/read schema asymmetry)

**Symptom.** This succeeds:

```
hero connect gitlab --integration-id gitlab-chronecho \
  --project noeta-studios/chronecho --base-url https://gitlab.com \
  --user-email developer@example.com --token-stdin
# → "Connected integration gitlab-chronecho (gitlab). Inspect with 'hero connect --list'."
```

The very next command fails:

```
hero connect --list
# → Error: $.integrations.connections.gitlab-chronecho.settings.user_email: unknown gitlab setting
```

**Confirmed root cause.** `runConnectNonInteractive` assembles the settings map and
writes `user_email` whenever the `--user-email` flag is non-empty, with **no check
that the provider's schema allows it**:

- `internal/cli/connect.go:257-259` — `if connectUserEmail != "" { settings["user_email"] = connectUserEmail }`. Conditional on the *flag*, unconditional across *providers*.
- The assembled `connection` (connect.go:264) is marshaled and persisted via `config.PatchCommittedIntegrations` (connect.go:273) — or `PatchLocalIntegrations` under `--local-only` (connect.go:268). **Neither Patch function validates provider settings** — they merge-patch raw JSON and write (`internal/config/integrations.go:439-546`).
- GitLab's provider schema has **no** `user_email` key — only `project`, `base_url`, `post_on_design`, `post_on_deliver`, `size_mapping` (`internal/config/integrations.go:284-288`). Only `jira` (line 282) and `confluence` (line 290) declare `user_email`.
- The **read** path validator `validateProviderSettings` rejects unknown keys at `internal/config/integrations.go:295-298` (`unknown %s setting`). It runs from `ResolvedIntegrations.validate` (integrations.go:261) inside `ResolveIntegrationDocuments` (integrations.go:124).

The read behavior is **correct and intended** — `internal/config/integrations_test.go:175`
(`gitlab-inapplicable-email` case) already asserts gitlab+`user_email` must be
rejected on read. The bug is the **write** path, which has no matching schema gate.

**Amplification beyond the triage.** The failure is worse than "`--list` breaks."
`ResolveIntegrationDocuments` returns the validation error (integrations.go:124), and
`config.Load` propagates it directly:

- `internal/config/config.go:1452-1454` — `resolved, ierr := ResolveIntegrationDocuments(...); if ierr != nil { return cfg, ierr }`.

So the invalid `user_email` bricks **every** command that calls `config.Load`, not
just `--list`. `hero status`, `hero sync`, `hero list` — all fail with the same
opaque JSON-path error until the file is hand-repaired. The one-line "Connected"
success message is the last thing that works.

**UX trap / bug class.** The error surfaces in a *later, unrelated* command and
names a raw JSON path, so it reads like `--list` (or the whole workspace) is broken,
not like `connect` chose a bad flag. On a fresh setup a user reasonably concludes
the GitLab connector is broken and gives up. This is a **general class**: *any*
flag/provider mismatch silently persists a config the read path can't load. The
only workaround today is hand-editing `.hero/hero.json` to delete
`settings.user_email` (confirmed — there is no CLI escape hatch).

Note a second, currently-latent write site with the same shape:
`updateHeroJSON` (`internal/cli/connect.go:572-574`) also writes `user_email`
unconditionally across providers. It's only reached by the *interactive* flows,
and interactive GitLab calls `saveConnection(..., "")` with an empty email
(connect.go:482), so it doesn't bite today — but it's the same latent bug and
should be fixed in the same pass.

### Defect 2 — `hero sync link` has no `--force`, so a spec can't be re-pointed

**Symptom.** During tracker migration, every spec already carries a Linear
`tracker_id` (e.g. `ECHO-176`). Re-pointing it at the GitLab equivalent fails:

```
hero sync link .hero/planning/features/... 15
# → Error: spec <slug> is already linked to tracker issue ECHO-176
```

**Confirmed root cause.** `runLink` hard-errors on any non-empty `tracker_id` with
no escape hatch, and `linkCmd` has no `--force` flag:

- `internal/cli/link.go:48-50` — `if s.TrackerID != "" { return fmt.Errorf("spec %s is already linked to tracker issue %s", ...) }`.
- `internal/cli/link.go:13-21` — `linkCmd` defines no flags (registered at `internal/cli/sync.go:75`).

For anyone migrating trackers — exactly the workflow the GitLab connector enables —
Hero cannot re-point a spec off a dead Linear issue onto its GitLab equivalent. The
reporter hand-edited three specs' frontmatter as the workaround.

### Defect 3 — `hero sync link` rejects a spec directory argument

**Symptom.**

```
hero sync link .hero/planning/features/ai-sdk-v5-upgrade 15
# → Error: ... is a directory
```

**Confirmed root cause.** `runLink` passes the raw arg straight to `spec.ParseFile`,
which `os.ReadFile`s it and fails on a directory:

- `internal/cli/link.go:43` — `s, err := spec.ParseFile(specPath)` with `specPath = args[0]`.
- `internal/spec/spec.go:345-357` — `ParseFile` does `os.ReadFile(path)`; a directory yields "is a directory".
- The same raw path is reused for the write at `internal/cli/link.go:64` — `writeTrackerID(specPath, issueID)` (`internal/cli/sync.go:230`).

Specs live on disk as `<dir>/spec.md` (verified: `.hero/planning/features/agent-outposts/spec.md`,
and `spec.Discover` walks for `spec.md` / three-file `requirements.md` at
`internal/spec/spec.go:1166+`). Hero identifies specs by **slug** everywhere else,
so passing the directory — or the bare slug — is the obvious thing a user tries.

The obvious sibling helper does **not** solve this: `resolveSpec`
(`internal/cli/score.go:71-90`) routes any arg containing a path separator straight
to `spec.ParseFile`, so a directory path (separators, no `.md` suffix) still hits
`ParseFile` and still fails. There are effectively **three** resolvers today:

| Resolver | Location | Handles | Directory arg? |
|---|---|---|---|
| `resolveSpec(arg, heroDir)` | `score.go:71` | path **or** slug | ✗ path→`ParseFile`, breaks on dir |
| `resolveSpecBySlug(heroDir, slug)` | `sync_push.go:406` | slug only (via `Discover`) | n/a (slug-only) |
| inline `spec.ParseFile(args[0])` | `link.go:43` | path only | ✗ breaks on dir, no slug support |

**Related-command note (called out, not fixed beyond link):** `resolveSpec` is also
used by `hero spec score` (`score.go`), so it shares the directory-rejection gap —
fixing `resolveSpec` fixes both `score` and `link` in one change. `resolveSpecBySlug`
(used by `sync push`, `pull`, `spec set-owner`) is slug-only and does not hit this
gap. No fourth ad-hoc path handler should be added.

### Migration dependency (why order matters)

`config.Load` bridges a valid integration into the legacy `cfg.Tracker` via
`DeliveryTracker()` (`internal/config/config.go:1459-1460`), and `runLink` reads
`cfg.Tracker` (link.go:35, 53). So while Defect 1's invalid config is on disk,
`config.Load` errors and `hero sync link` never even reaches its own logic. **Defect 1
must be fixed first** — it gates the whole migration path that Defects 2 and 3 sit on.

### Root cause

A **design gap**: the integration config has a strict read-time schema validator
(`validateProviderSettings`) but **no matching write-time gate**, so `connect` can
persist a settings map the loader will refuse — bricking every subsequent
config-loading command with a misleading, displaced error. Secondarily, the
tracker-migration ergonomics were never built out: `sync link` has no re-point
escape hatch and no directory/slug resolution, both of which a real migrant needs.

### Severity

**High.** Defect 1 silently bricks a fresh GitLab setup: `connect` reports success,
then every config-loading command fails with an opaque, displaced error, and the
only recovery is hand-editing JSON. It is a general class (any flag/provider
mismatch), and it blocks the migration path outright. Defects 2 and 3 are **medium**
ergonomics blockers that make tracker migration require manual frontmatter edits.
Caused entirely by our code; no external factor. Workaround exists (hand-edit JSON /
frontmatter) but is exactly the friction the feature is supposed to remove.

## Goal

`hero connect` never persists an integration the loader can't read: an invalid
flag/provider combination fails **at connect time** with a clear, flag-oriented
message and writes nothing. `hero sync link` supports tracker migration end to end:
`--force` re-points an already-linked spec (printing old→new and still verifying the
new issue exists), and the spec argument accepts a spec directory or a bare slug in
addition to an explicit `spec.md` path.

## Acceptance Criteria

- **AC-1:** IF an assembled connection's settings contain a key not in the provider's schema THEN `hero connect` SHALL fail before persisting and write neither `hero.json` nor `hero.local.json`.
- **AC-2:** WHEN `hero connect gitlab ... --user-email <x>` is run THE SYSTEM SHALL reject it at connect time with a message naming the offending flag (e.g. `--user-email is not valid for provider gitlab`).
- **AC-3:** THE SYSTEM SHALL guarantee that no `hero connect` code path (committed, `--local-only`, or `--global`) persists a provider-bearing connection whose settings fail the provider schema.
- **AC-4:** WHEN `hero sync link <spec> <issue>` targets a spec that already has a `tracker_id` and `--force` is NOT passed THE SYSTEM SHALL refuse and mention `--force`.
- **AC-5:** WHEN `hero sync link <spec> <issue> --force` is run on an already-linked spec THE SYSTEM SHALL verify the new issue exists, overwrite `tracker_id`, and print the old→new transition.
- **AC-6:** WHEN `hero sync link <dir> <issue>` is given a spec directory THE SYSTEM SHALL resolve it to the spec inside (`<dir>/spec.md`) and link that spec.
- **AC-7:** WHEN `hero sync link <slug> <issue>` is given a bare slug THE SYSTEM SHALL resolve it via workspace discovery and link the matching spec.
- **AC-8:** THE `--user-email` flag help text and the tracker docs SHALL state that `user_email` applies only to Jira and Confluence Cloud (rejected for github/linear/gitlab), so the constraint is discoverable before the user hits the connect-time error.
- **AC-9:** THE `hero sync link` docs SHALL document `--force` (re-pointing during tracker migration) and show that a spec directory or bare slug is accepted, not only an explicit `spec.md` path.

## Fix Plan

Ordered. Defect 1 first — it gates the migration path.

### 1. Defect 1 — validate assembled settings before persisting

**1a. Add an exported schema-check wrapper** in `internal/config/integrations.go`
(next to `validateProviderSettings`, ~line 363). `validateProviderSettings` takes
`(id string, IntegrationConfig)` where `Settings` is `map[string]json.RawMessage`;
the connect path holds `map[string]any`, so add a thin adapter:

```go
// ValidateConnectionSettings checks an assembled settings map against the
// provider's schema before it is persisted, so connect never writes a config
// the read path will reject.
func ValidateConnectionSettings(id, provider string, settings map[string]any) error {
	raw := make(map[string]json.RawMessage, len(settings))
	for k, v := range settings {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		raw[k] = b
	}
	return validateProviderSettings(id, IntegrationConfig{Provider: provider, Settings: raw})
}
```

**1b. Call it at connect time, with a flag-oriented error.** In
`runConnectNonInteractive` (`internal/cli/connect.go`), after the settings map is
fully assembled (**after line 259**, before `connection` is built at line 264):

```go
if err := config.ValidateConnectionSettings(id, provider, settings); err != nil {
	return fmt.Errorf("cannot connect %s: %s", provider, settingErrorToFlag(err, provider))
}
```

Add a small helper (in connect.go) that rewrites the raw JSON-path message into a
flag the user actually chose, since `connect` is where the flag was selected:

```go
// settingErrorToFlag translates a settings-schema error into flag vocabulary.
var settingFlagNames = map[string]string{
	"user_email": "--user-email", "base_url": "--base-url", "project": "--project",
}
func settingErrorToFlag(err error, provider string) string {
	for key, flag := range settingFlagNames {
		if strings.Contains(err.Error(), ".settings."+key+":") {
			return fmt.Sprintf("%s is not valid for provider %s", flag, provider)
		}
	}
	return err.Error()
}
```

**1c. Fix the latent twin.** In `updateHeroJSON` (`internal/cli/connect.go`), after
the settings map is assembled (**after line 581**, before the `PatchCommittedIntegrations`
call at line 582), add the same `config.ValidateConnectionSettings(id, trackerType, settings)`
guard so the interactive path can't regress into the same bug.

**1d. Backstop (recommended — kills the whole class).** Make the persistence
functions refuse an invalid provider-bearing connection, so *no* code path
(committed, `--local-only`, `--global`) can bypass the check. In both
`PatchCommittedIntegrations` (after the merge at `integrations.go:518`) and
`PatchLocalIntegrations` (after the merge at `integrations.go:460`), iterate the
merged `integrations.connections`; for each connection that has a non-empty
`provider` string, marshal its `settings` to `map[string]json.RawMessage` and run
`validateProviderSettings`. **Skip connections with no `provider`** — token-only
local/credential overlays (connect.go:283, 539) legitimately carry only
`auth.token` and must not be rejected. Do **not** full-decode with
`DisallowUnknownFields` here (that would trip on unrelated pre-existing keys) — call
only `validateProviderSettings` on the settings subtree.

> Recommendation: implement 1b/1c (friendly, flag-named error — best UX) **and** 1d
> (the single chokepoint that makes the invalid state unrepresentable). They are
> complementary, not duplicative: 1b/1c give the good message at the surface, 1d is
> the backstop no path bypasses. If only one is chosen, 1d is the true class-killer
> but emits the raw JSON-path error; 1b/1c alone leave `--local-only`/`--global`
> and any future writer unguarded.

**1e. Fix the misleading flag help text.** `--user-email` is registered at
`connect.go:71` as `"provider user email"` — advertised as a general flag with no
hint it's provider-specific, which is what led the reporter to pass it to gitlab.
Change it to name the constraint, e.g.
`"user email (Jira/Confluence Cloud only)"`. This is the surface counterpart to the
1b/1c connect-time rejection — help text and behavior must agree.

### 2. Defect 2 — `--force` on `sync link`

In `internal/cli/link.go`:

- Add a package var `var linkForce bool` and an `init()` (or extend the existing
  registration flow) to register the flag on `linkCmd`:
  ```go
  func init() {
  	linkCmd.Flags().BoolVar(&linkForce, "force", false, "overwrite an existing tracker_id (re-point a migrated spec)")
  }
  ```
- Relax the guard at **link.go:48-50**:
  ```go
  if s.TrackerID != "" && !linkForce {
  	return fmt.Errorf("spec %s is already linked to tracker issue %s (use --force to re-point)", s.Slug, s.TrackerID)
  }
  ```
- **Keep** the `t.GetIssue(issueID)` verification (link.go:58) — `--force` still
  proves the *new* issue exists before overwriting.
- After the successful write (link.go:64-68), print the transition when re-pointing:
  ```go
  if s.TrackerID != "" && linkForce {
  	fmt.Printf("Re-pointed spec %s: %s → %s\n", s.Slug, s.TrackerID, issueID)
  }
  ```

### 3. Defect 3 — accept a spec directory (and slug) in `sync link`

**3a. Extend the single shared resolver** `resolveSpec` (`internal/cli/score.go:71`)
to handle a directory before the existing path/slug branches:

```go
func resolveSpec(arg, heroDir string) (*spec.Spec, error) {
	// Directory: resolve to the spec inside it.
	if info, err := os.Stat(arg); err == nil && info.IsDir() {
		if p := filepath.Join(arg, "spec.md"); fileExists(p) {
			return spec.ParseFile(p)
		}
		if fileExists(filepath.Join(arg, "requirements.md")) {
			return spec.ParseThreeFile(arg)
		}
		return nil, fmt.Errorf("no spec.md or requirements.md in %s", arg)
	}
	// existing: direct file path, then slug via Discover …
}
```

(`score.go` already imports `os`/`filepath`/`strings`; add a tiny `fileExists`
helper or inline the `os.Stat` checks.) This single change also fixes
`hero spec score <dir>` for free.

**3b. Switch `link` off raw `ParseFile`.** In `runLink` (`internal/cli/link.go`),
replace lines 42-46 with `resolveSpec` (it already has `heroDir` at link.go:30):

```go
s, err := resolveSpec(specPath, heroDir)
if err != nil {
	return fmt.Errorf("resolving spec %s: %w", specPath, err)
}
```

This gives `link` directory **and** bare-slug support in one move.

**3c. Write to the resolved on-disk path, not the raw arg.** Change the write at
link.go:64 to use the resolved spec's path: `writeTrackerID(s.Path, issueID)`.
Guard the three-file case: when `s.ThreeFile` is true, `s.Path` is a *virtual*
`<dir>/spec.md` (spec.go:397) that doesn't exist on disk, so frontmatter must be
written to `requirements.md` instead. Either special-case it
(`filepath.Join(dir, "requirements.md")` when `s.ThreeFile`) or explicitly reject
three-file specs in `link` with a clear message. The repo currently uses
single-file `spec.md` throughout, so the common path is correct; this is a
correctness guard, not a blocker.

### 4. Documentation — bring docs in line with shipped behavior

Docs update **in the same PR** as the code — they describe behavior that only
exists once Defects 1–3 land, so none of this is written ahead of the fix. Audited
state: the docs are not currently *wrong* (they already omit `user_email` from the
gitlab schema and show `sync link` with an explicit `spec.md` path — the only forms
that work today); they're **incomplete** for the new behavior.

- **`web/docs/src/cli/tracker-integration.md`** (line ~42) — the `hero sync link`
  example currently shows only `... /spec.md PROJ-1234`. Add: (a) that the spec
  argument also accepts a **spec directory** or a **bare slug**, and (b) a short
  note on `--force` for re-pointing a spec during a **tracker migration** (the
  Linear→GitLab case), including that `--force` still verifies the new issue exists.
- **`web/docs/src/configuration/hero-json.md`** (per-provider settings table, lines
  ~166–174) — add an explicit line that `user_email` is valid **only** for `jira`
  and `confluence`, and is rejected for `github`/`linear`/`gitlab` at connect time.
  The table implies this by omission today; make it explicit so it's discoverable
  before the connect-time error (AC-8).
- **`web/docs/src/configuration/tracker-setup.md`** — GitLab tab (lines ~67–85):
  the `--user-email` guidance is already jira-scoped (line 6), so no correction
  needed; add a brief **migration** note (re-point existing links onto GitLab with
  `hero sync link <slug> <new-id> --force`) near the GitLab setup or the sync
  section, since GitLab support is the feature that enables tracker migration.
- **CLI flag reference** — if the docs include a generated/enumerated flag list for
  `sync link`, ensure `--force` appears (grep the docs tree for the `sync link`
  synopsis; if flags are hand-maintained there, add `--force`). If flag docs are
  generated from cobra, the new flag surfaces automatically — verify, don't
  duplicate.
- **Release notes** — a one-line entry under the next version in
  `web/docs/src/releases/index.md` is appropriate at release time (handled by
  `/release`); noted here so it isn't missed, not authored now.

Scope note: `README.md`, `AGENTS.md`, `CLAUDE.md`, and the peer-call/next files
also matched `sync link` in the grep, but none document its flags or path forms —
they reference the command in passing and need no change. This is **not** a
harness-facing change (see Boundaries): the doc edits are user-facing website docs
under `web/docs/`, not `hero install` instruction surfaces, so the
all-install-targets tripwire does not apply.

## Test Plan

### Existing coverage to build on
- `internal/config/integrations_test.go:167+` (`TestProviderSettingsValidationIsSpecificAndTypeStrict`) — table of read-time schema rejections, including `gitlab-inapplicable-email` (line 175). The new write-time checks should mirror these expectations so read and write agree.
- `internal/cli/sync_push_merge_test.go:415` — pattern for a `resolveSpecBySlug`-style CLI test harness (`env.heroDir`) to copy for link tests.

### New / changed tests

**Defect 1**
- `internal/config` unit test for `ValidateConnectionSettings`: gitlab+`user_email` → error naming the setting; gitlab with only `project`+`base_url` → ok; jira+`user_email` → ok. Reuse the existing table shape.
- `internal/config` test for the 1d backstop: `PatchCommittedIntegrations` and `PatchLocalIntegrations` reject a merged connection whose provider-bearing settings are invalid, but **accept** a token-only patch with no `provider` (regression guard against false positives on credential overlays).
- `internal/cli` connect test (mirroring the repro): non-interactive gitlab connect with `--user-email` returns a flag-named error, and **no** `hero.json`/`hero.local.json` is written (assert files absent/unchanged). Follow with a `config.Load` to prove the workspace is not bricked.

**Defect 2**
- `internal/cli` link test: linking an already-linked spec without `--force` errors and mentions `--force`; with `--force` (against a stub tracker whose `GetIssue` succeeds) it overwrites `tracker_id` and prints old→new. Add a case where `--force` is set but the new issue does **not** exist → still errors (verification preserved).

**Defect 3**
- `internal/cli` (or `score`) resolver test: `resolveSpec` given a directory returns the spec at `<dir>/spec.md`; given a bare slug resolves via `Discover`; given `<dir>` with neither file errors cleanly.
- `internal/cli` link test: `sync link <dir> <issue>` and `sync link <slug> <issue>` both write `tracker_id` to the correct on-disk `spec.md`.

## Validation

1. Reproduce Defect 1 end to end in a scratch workspace: run the failing
   `hero connect gitlab ... --user-email ...` and confirm it now fails at connect
   time with `--user-email is not valid for provider gitlab`, `.hero/hero.json` is
   unchanged, and a subsequent `hero connect --list` / `hero status` still works.
2. Positive path: `hero connect gitlab --integration-id gitlab-x --project g/p
   --base-url https://gitlab.com --token-stdin` succeeds, and `hero connect --list`
   shows it.
3. Defect 2: on a spec already linked to a Linear id, `hero sync link <slug> <new>`
   errors mentioning `--force`; `--force` re-points and prints old→new.
4. Defect 3: `hero sync link <spec-dir> <issue>` and `hero sync link <slug> <issue>`
   both succeed and write `tracker_id` to `<dir>/spec.md`.
5. `go test ./internal/config/... ./internal/cli/... ./internal/spec/...` green.
6. `go build ./cmd/hero` clean.
7. Docs: `hero sync link --help` shows `--force`; `hero connect --help` shows the
   corrected `--user-email` text; the updated `tracker-integration.md` /
   `hero-json.md` / `tracker-setup.md` render and the docs site builds
   (`hero docs check` clean, and the mkdocs build if run for the site).

## Boundaries

- Not redesigning the integration schema or the `layered-integration-configuration`
  model — only adding a write-time gate that mirrors the existing read-time schema.
- Not adding `--force`/directory handling to commands beyond `sync link` and the
  shared `resolveSpec` (which also covers `spec score`). The `resolveSpecBySlug`
  users (`sync push`, `pull`, `spec set-owner`) are slug-only and out of scope.
- Not touching the live GitLab tracker client or verification logic — those work.
- No harness-facing surface (`hero install`, instruction files, slash commands,
  agents, skills) is touched, so the all-install-targets tripwire does not apply.
- Full three-file (`requirements.md`) write support in `link` is only guarded, not
  built out — the repo uses single-file `spec.md`.

## Risks

- **1d backstop false positives.** If the provider-bearing filter is wrong, a
  legitimate token-only local/credential patch could be rejected. The explicit
  "skip connections with no `provider`" rule and the dedicated regression test
  cover this — verify it.
- **Re-decoding on write.** Do not use `DisallowUnknownFields` in the Patch
  backstop; it would reject unrelated pre-existing top-level keys. Validate only
  the settings subtree via `validateProviderSettings`.
- **Three-file `link` write.** `writeTrackerID(s.Path, ...)` on a `ThreeFile` spec
  would create a stray `spec.md` instead of editing `requirements.md`. Guard it
  (see 3c).
- **`--force` blast radius.** `--force` intentionally overwrites `tracker_id`;
  keeping the `GetIssue` verification prevents pointing a spec at a non-existent
  issue.

## Kickoff

Fixes three GitLab-migration rough edges: `hero connect` writing a config it can't
read back, and `hero sync link` refusing to re-point a migrated spec or accept a
spec directory.

**Status:** delivering → verifying — all three defects implemented, tests green, cold audit returned SHIP. Committed on branch `fix/tracker-migration-connect-link` (commit `e42e272`).

**Pick up at:** delivery is complete pending `hero spec verify`. If reopening: the write-time gate lives in `config.ValidateConnectionSettings` + `validateMergedConnectionSettings` (backstop in both Patch* functions), `--force` and dir/slug live in `internal/cli/link.go`/`score.go`, docs updated under `web/docs/src/`. New tests in `internal/cli/connect_link_fixes_test.go` and `internal/config/integrations_test.go`.

→ `.hero/planning/bugs/tracker-migration-connect-link-fixes/spec.md`

**Files:** `internal/cli/connect.go:71,257`, `internal/config/integrations.go:267,439,494`, `internal/cli/link.go:43,48,64`, `internal/cli/score.go:71`; docs: `web/docs/src/cli/tracker-integration.md`, `web/docs/src/configuration/hero-json.md`, `web/docs/src/configuration/tracker-setup.md`
**Skip:** validating each config file in isolation inside Patch* (the read path validates the *merged* result — per-file validation false-positives on token-only overlays); adding a fourth ad-hoc path resolver (extend `resolveSpec`).

## Completion Ledger

**Understanding:** three contained tracker-migration defects on a working GitLab connector — connect persisting a config the loader rejects (write/read schema asymmetry), `sync link` unable to re-point a linked spec, and `sync link` rejecting a spec dir/slug. Defect 1 delivered first (it gates the migration path). **Validation:** full suite green (`go test ./internal/config/... ./internal/cli/... ./internal/spec/...`), `go build ./cmd/hero` clean, CLI exercised end-to-end in a scratch workspace, and an independent cold audit returned SHIP (report: `delivery-audit.md`).

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Evidence |
|---|---|---|---|
| AC-1 | Out-of-schema key → connect fails before persisting, writes neither file | DONE | `internal/cli/connect.go` validate-before-build; `TestNonInteractiveConnect_GitlabUserEmail_RejectedAndWritesNothing` (byte-compares hero.json, asserts hero.local.json absent, re-Loads to prove not bricked) |
| AC-2 | `connect gitlab … --user-email` rejected at connect time naming the flag | DONE | `settingErrorToFlag` → `--user-email is not valid for provider gitlab`; `TestValidateConnectionSettings` |
| AC-3 | No connect path persists a provider-bearing invalid connection | DONE | 1b/1c surface guards + 1d `validateMergedConnectionSettings` backstop in both `PatchCommitted/LocalIntegrations`; `TestPatchBackstopRejectsInvalidProviderSettings` |
| AC-4 | Already-linked spec without `--force` refuses and mentions `--force` | DONE | `internal/cli/link.go` guard `(use --force to re-point)`; `TestLink_AlreadyLinked_MentionsForce` |
| AC-5 | `--force` verifies new issue, overwrites `tracker_id`, prints old→new | DONE | keeps `GetIssue`; prints `Re-pointed spec …: old → new`; `TestLink_Force_RepointsAndPrintsTransition`, `TestLink_Force_NonexistentIssueStillErrors` |
| AC-6 | `sync link <dir> <issue>` resolves `<dir>/spec.md` and links it | DONE | `resolveSpec` dir branch (`score.go`); `TestLink_AcceptsDirAndSlug/directory` |
| AC-7 | `sync link <slug> <issue>` resolves via discovery and links | DONE | `resolveSpec` slug branch via `spec.Discover`; `TestLink_AcceptsDirAndSlug/slug` |
| AC-8 | `--user-email` help text + docs state Jira/Confluence-only | DONE | `connect.go:71`; `web/docs/src/configuration/hero-json.md` explicit rejection line |
| AC-9 | `sync link` docs document `--force` and dir/slug acceptance | DONE | `web/docs/src/cli/tracker-integration.md`, `web/docs/src/configuration/tracker-setup.md` |

### Changes (Fix Plan)

| # | Item | Status | Evidence |
|---|---|---|---|
| 1a | Exported `ValidateConnectionSettings` adapter | DONE | `internal/config/integrations.go` (marshals `map[string]any`→`RawMessage`, delegates to `validateProviderSettings`) |
| 1b | Validate in `runConnectNonInteractive` + `settingErrorToFlag` | DONE | `internal/cli/connect.go` |
| 1c | Same guard in `updateHeroJSON` (latent twin) | DONE | `internal/cli/connect.go` |
| 1d | Backstop in both Patch functions; skip no-provider overlays; no `DisallowUnknownFields` | DONE | `validateMergedConnectionSettings`; `TestPatchBackstopAcceptsTokenOnlyOverlay` guards false positives |
| 1e | Fix `--user-email` help text | DONE | `internal/cli/connect.go:71` |
| 2 | `--force` on `sync link` | DONE | `internal/cli/link.go` |
| 3a | Extend shared `resolveSpec` for directories (no 4th resolver) | DONE | `internal/cli/score.go`; `TestResolveSpec_DirAndSlug` |
| 3b | Switch `link` off raw `ParseFile` to `resolveSpec` | DONE | `internal/cli/link.go` |
| 3c | Write to resolved path; guard three-file → `requirements.md` | DONE | `internal/cli/link.go` |
| 4 | Docs: tracker-integration / hero-json / tracker-setup | DONE | three website docs updated |

**Non-DONE rows:** none.
