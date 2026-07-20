# Delivery audit — serve-auto-refresh-bypasses-bulk-import

**Audited:** `git diff --cached` at `c3823e0`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] Auto-refresh invokes the canonical bulk workflow exactly once per cycle — `internal/serve/refresh.go:75`, `internal/serve/refresh.go:95`, and `internal/serve/refresh.go:131`; `TestNewBulkImportCommandUsesCanonicalBulkRefresh` and `TestImportRefresherRefreshRunsOneBulkImportAndPublishes` verify the exact command, working directory, and one runner call.
- [✓] Per-type imports use independent bulk queries and can discover new work — `internal/serve/refresh.go:115` delegates to the canonical `sync import` implementation; `TestFetchByTypeUnion_UnionsAndDedups` and `TestFetchByTypeUnion_AppliesLimitPerType` verify independent query execution, union/deduplication, and per-query limits.
- [✓] Background and bulk refresh avoid `GetIssue` and per-ticket APIs — the staged server refresher removes tracker/spec traversal; `TestServerAutoRefreshHasNoPerTicketTrackerPath`, `TestRefreshImportedSpecs_UpdatesPreservesAndIsIdempotent`, and `TestRefreshImportedSpecs_DoesNotFetchMissingIssuesIndividually` enforce both sides of the boundary.
- [✓] Server stop cancels an in-flight import and future cycles — `internal/serve/refresh.go:54`, `internal/serve/refresh.go:68`, and `internal/serve/refresh.go:87`; `TestImportRefresherRunStopsAndCancelsInFlightImport` verifies context cancellation and loop exit.
- [✓] Success publishes an index event; failure reports without publishing success — `internal/serve/refresh.go:95`; `TestImportRefresherRefreshRunsOneBulkImportAndPublishes`, `TestImportRefresherRefreshFailureDoesNotPublish`, and `TestFormatImportOutputBoundsDaemonLogs` exercise the success, failure, and bounded-output paths.
- [✓] Documentation isolates full-ticket evidence from background sync — `web/docs/src/configuration/hero-json.md:198` and `web/docs/src/configuration/tracker-setup.md:147` document canonical bulk refresh, configured filters, independent `by_type` queries, per-query limits, and explicit `hero sync evidence <spec-slug>` retrieval.

## Changes
- [✓] Replace the private server loop with a cancellable canonical bulk import — `internal/serve/refresh.go:32`, `internal/serve/refresh.go:120`, and `internal/serve/refresh.go:131`; `internal/serve/server.go:373` uses the simplified constructor.
- [✓] Rewrite refresher tests around an injected bulk runner — `internal/serve/refresh_test.go:14` through `internal/serve/refresh_test.go:153` assert command construction, one-call delegation, events, failure, cancellation, bounded output, and absence of the private per-ticket path.
- [✓] Clarify auto-refresh and evidence behavior in both configuration guides — `web/docs/src/configuration/hero-json.md:198` through `web/docs/src/configuration/hero-json.md:205` and `web/docs/src/configuration/tracker-setup.md:147` through `web/docs/src/configuration/tracker-setup.md:155` explicitly cover filters, independent `by_type` queries, per-query limits, discovery, and deep-evidence separation.

## Open items (if any)
- None.

## Audit notes
- The Completion Ledger contains one `DONE` row for every acceptance criterion and every Changes item, with corresponding implementation and test evidence.
- Provided test evidence reports the focused packages, race-enabled server package, full repository suite, and strict MkDocs build all passed. `git diff --cached --check` also passed during this audit.
