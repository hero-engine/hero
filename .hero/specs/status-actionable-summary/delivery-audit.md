# Delivery audit — status-actionable-summary

**Audited:** `git diff HEAD -- .hero/planning/features/status-actionable-summary/spec.md README.md internal/cli/mail_test.go internal/cli/status.go internal/cli/status_test.go internal/spec/select.go internal/spec/select_test.go web/docs/src/cli/search-and-context.md`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria

- [✓] AC-1: print work, corpus, and horizon-hidden counts before lists — `internal/cli/status.go:269-280`; exact mixed-view and horizon assertions in `TestStatusMixedViewCountsOrderingAndSuppression` and `TestStatusHorizonOptionsOnlyFilterOpenWork`.
- [✓] AC-2: render all selected-horizon handed-back, delivering, and in-review planning work in precedence order — `internal/cli/status.go:186-236`; lifecycle precedence asserted by `TestStatusMixedViewCountsOrderingAndSuppression`.
- [✓] AC-3: bound Upcoming to ten with ready work before blocked work and canonical priority ordering — `internal/cli/status.go:217-233,290-296`; dependency partitioning is asserted by `TestStatusMixedViewCountsOrderingAndSuppression`, and the cap plus pinned priority are asserted by `TestStatusBoundsUpcomingAndWaiting`.
- [✓] AC-4: print omitted counts and runnable full-list commands — `internal/cli/status.go:295-299,316-339`; exact Upcoming and Waiting hints in `TestStatusBoundsUpcomingAndWaiting`.
- [✓] AC-5: show at most five authoritative completions ordered by timestamp and slug tie-break — `internal/cli/status.go:237-243,342-358`; `TestStatusRecentlyCompletedUsesAuthoritativeTimestamp`.
- [✓] AC-6: count undated completions without fabricating recency — `internal/cli/status.go:188-196,346-357`; missing-timestamp assertions in `TestStatusRecentlyCompletedUsesAuthoritativeTimestamp` and `TestStatusMixedViewCountsOrderingAndSuppression`.
- [✓] AC-7: summarize intake and knowledge without exhaustive rows — `internal/cli/status.go:172-181,302-313`; exact counts and hidden-title assertions in `TestStatusMixedViewCountsOrderingAndSuppression`.
- [✓] AC-8: preserve operational signals — workspace, dialect, Mail, peer reconciliation, smoke, async, connection, and version paths remain at `internal/cli/status.go:67-138`; human behavior is asserted by `TestMailCLIJSONCommandsAndErrors`, `TestStatusPreservesPeerReconciliationSignal`, `TestStatus_SurfacesSmokeFailures`, `TestStatusPreservesAsyncConnectionAndVersionSignals`, and the existing `TestStatusDialect_*` tests. The supplied focused retained-signal run and full `go test ./...` both passed.
- [✓] AC-9: exclude archive-path mismatches from active work and warn once — `internal/cli/status.go:186-203,282-284`; both mismatch directions in `TestStatusArchiveInconsistencies`.
- [✓] AC-10: apply horizon options only to open work — workspace-wide corpus and completion counts are collected before open-work horizon filtering at `internal/cli/status.go:172-205`; default, `--all`, and explicit-horizon behavior in `TestStatusHorizonOptionsOnlyFilterOpenWork`.
- [✓] AC-11: preserve unbounded JSON fields and horizon behavior — the JSON path remains separate at `internal/cli/status.go:63-65,369-395`; `TestStatusJSONContractRemainsUnbounded` asserts allowed top-level keys, exact per-spec keys, unbounded rows, and explicit-horizon behavior.
- [✓] AC-12: render a concise zero-count empty state — `internal/cli/status.go:286-288`; `TestStatusEmpty`.
- [✓] AC-13: validate every emitted list command against the real command tree — all five forms are executed by `TestStatusEmittedListHintsResolve`; supplied real-workspace evidence also reports every printed hint succeeded.

## Changes

- [✓] Refactor human status collection — `internal/cli/status.go:143-267` adds one collected view for lifecycle partitions, corpus counts, horizon semantics, completion history, canonical dependency checks, and resolved-path integrity detection.
- [✓] Replace exhaustive corpus rendering — `internal/cli/status.go:269-359` renders bounded operational sections and browse hints while retaining the operational preamble and footer signals at `internal/cli/status.go:67-138`.
- [✓] Reuse canonical selection semantics — `internal/spec/select.go:226-242` exposes and reuses `SortByPriority`; `TestSortByPriorityMatchesSelector` checks parity with `Selector`.
- [✓] Expand human status tests — `internal/cli/status_test.go:17-547` covers counts, ordering, bounds, completion timestamps, suppression, archive integrity, horizons, JSON, hint resolution, help, peer reconciliation, async, connection, version, smoke, and empty state; `internal/cli/mail_test.go:76-95` covers retained JSON and human Mail output.
- [✓] Protect the JSON contract — `internal/cli/status_test.go:254-285,503-547` checks unbounded collection and exact existing field names; supplied real-command evidence reports 583 items with the expected schema.
- [✓] Update help and user documentation — `internal/cli/status.go:28-38`, `README.md:239-244`, and `web/docs/src/cli/search-and-context.md:289-342` describe the compact view, limits, full-list commands, JSON surface, and representative output.
