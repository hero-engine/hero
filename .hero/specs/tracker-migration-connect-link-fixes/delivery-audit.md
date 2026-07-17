# Delivery audit — tracker-migration-connect-link-fixes

**Audited:** `git diff main...HEAD` (commit e42e272, branch `fix/tracker-migration-connect-link`)
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] AC-1 connect fails before persisting on unknown-schema key, writes neither file — `internal/cli/connect.go:260` (validate before `connection` built at :267 and before any Patch* write); asserted by `TestNonInteractiveConnect_GitlabUserEmail_RejectedAndWritesNothing` (connect_link_fixes_test.go:19) which byte-compares `hero.json` unchanged AND asserts `hero.local.json` absent AND re-runs `config.Load` to prove the workspace is not bricked.
- [✓] AC-2 gitlab + `--user-email` rejected at connect time with flag-named message — `settingErrorToFlag` (connect.go:318-327) emits `--user-email is not valid for provider gitlab`; test asserts the exact string.
- [✓] AC-3 no connect path persists an invalid provider-bearing connection — surface check runs before the local/committed/global branch (connect.go:260); backstop `validateMergedConnectionSettings` wired into BOTH `PatchLocalIntegrations` (integrations.go:517) and `PatchCommittedIntegrations` (integrations.go:578). `TestPatchBackstopRejectsInvalidProviderSettings` covers committed+local; the `--global` branch still routes settings through `PatchCommittedIntegrations` (connect.go:276), so it is structurally covered (no dedicated --global test, but the invalid state is unrepresentable on the write it uses).
- [✓] AC-4 already-linked without `--force` refuses and mentions `--force` — link.go:60-62; `TestLink_AlreadyLinked_MentionsForce`.
- [✓] AC-5 `--force` verifies new issue, overwrites, prints old→new — link.go:74-88 (GetIssue kept at :71-73; prints `Re-pointed spec …`); `TestLink_Force_RepointsAndPrintsTransition` asserts transition line printed, `tracker_id: 15` on disk, old `ECHO-176` gone.
- [✓] AC-6 spec directory resolves to `<dir>/spec.md` — `resolveSpec` directory branch (score.go:72-81); `TestLink_AcceptsDirAndSlug/directory`.
- [✓] AC-7 bare slug resolves via discovery — `resolveSpec` slug branch (unchanged, reached after dir/path); `TestLink_AcceptsDirAndSlug/slug` + `TestResolveSpec_DirAndSlug/slug`.
- [✓] AC-8 `--user-email` help text names Jira/Confluence-only + docs — connect.go:71 (`"user email (Jira/Confluence Cloud only)"`); hero-json.md +7 lines making the constraint explicit.
- [✓] AC-9 `sync link` docs document `--force` + dir/slug — tracker-integration.md (+22, three-form example + `--force` migration block), tracker-setup.md GitLab migration note.

## Changes
- [✓] `ValidateConnectionSettings` adapter added next to `validateProviderSettings` — integrations.go:365-377, matches the spec's 1a snippet.
- [✓] Connect-time call + `settingErrorToFlag` helper — connect.go:260, :309-327 (1b).
- [✓] Latent twin `updateHeroJSON` guarded — connect.go:605-607, `id` in scope from :590 (1c).
- [✓] Patch* backstop skips no-provider overlays, no DisallowUnknownFields — `validateMergedConnectionSettings` (integrations.go:381-417) skips `provider == ""`, validates only the settings subtree (1d).
- [✓] `--force` flag + relaxed guard + transition print — link.go (2).
- [✓] `resolveSpec` directory branch, `link` switched off raw `ParseFile`, writes resolved path with three-file guard — score.go:72-81, link.go:52, :77-81 (3a/3b/3c).
- [✓] Docs: tracker-integration.md, hero-json.md, tracker-setup.md (4).

## Flagged judgment call — fixture edit (VERIFIED LEGITIMATE)
`TestPatchLocalIntegrationsPreservesKeysAndMode` (integrations_test.go:101) had `base_url` added to the jira connection in the patch. This is a correct fixture repair, not a weakening:
- The jira provider schema requires `["project", "base_url"]` (integrations.go:283). The old fixture (`{"provider":"jira","settings":{"project":"P"}}`) was schema-invalid — missing required `base_url`.
- The new 1d backstop now runs `validateProviderSettings` on the merged doc inside `PatchLocalIntegrations`, so the old fixture would now (correctly) fail `base_url is required`. Adding `base_url` restores a schema-valid shape that a real jira connect actually produces.
- Every preservation assertion is UNCHANGED and still meaningful: `personal:"keep"`, `old` connection, `LOCAL-CANARY` token (line 109), and the `0600` mode check (line 113). The edit touched only the input patch, not the assertions.

## Audit notes
- Build clean (`go build ./cmd/hero`). `go test ./internal/config/... ./internal/cli/... ./internal/spec/...` all green (cli 15.9s). New tests confirmed executing (not skipped), with real behavioral assertions on files/stdout/errors.
- Boundaries respected: no live GitLab client touched; no fourth resolver (extended the single `resolveSpec`); no harness-install surface (`.claude`/`.codex`/`.agents`/instruction files) touched — doc edits are `web/docs/` only. Diff scoped to the spec's named files plus the projected handoff files (NEXT/QUEUE/SNAPSHOT/next), which correctly travel with the commit.
- The three-file (`requirements.md`) case is guarded per 3c (link.go:78-80), matching the spec's "guard, don't build out" decision.
- **Process note (not a delivery defect):** the committed `spec.md` has no `## Completion Ledger` section and status is still `delivering`. `hero spec verify` Gate 1 parses that ledger; add it and flip status before closing. The delivery work itself is complete and evidenced.
