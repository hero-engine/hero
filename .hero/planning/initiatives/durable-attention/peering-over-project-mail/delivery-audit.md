# Delivery audit — peering-over-project-mail

**Audited:** `git diff HEAD`
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1 typed peer Mail without model launch — `internal/peering/peercall.go:57-138`; `TestCallDeliversTypedMailWithoutReceiverTreeWrite`
- [✓] AC-2 same-thread wait response or structured pending timeout — `internal/peering/peercall.go:140-162`; `TestCallIdempotentAndWaitsForSameThreadReply`; `TestCallTimeoutIsStructuredPending`
- [✓] AC-3 handoff sends Mail without changing either spec tree — `internal/peering/handoff.go:39-112`; `TestHandoffSendsMailWithoutChangingEitherSpecTree`; `TestHandoffDismissCreatesNoWork`
- [✓] AC-4 explicit receiver promotion through Mail/Intake with reply — `internal/peering/receive.go:28-87`; `TestReceivePromotesOnceAndReplies`
- [✓] AC-5 peer-call and receive retries are idempotent — `internal/peering/peercall_test.go:47-79`; `internal/peering/handoff_test.go:97-124`
- [✓] AC-6 legacy status, provenance, trail, and artifact interpretation remains — `internal/cli/handoff.go:146-242`; `internal/peering/resolve.go`; `TestLegacyTrailAndReceivedFromRemainReadable`; existing trail/resolve tests
- [✓] AC-7 legacy `hero handoff accept` remains — `internal/cli/handoff.go:256-318`; full CLI suite evidence
- [✓] AC-8 deprecated subagent config loads, warns once, and is ignored — `internal/config/config.go`; `TestLoadWarnsOnceForIgnoredPeeringSubagent`; `TestDeprecatedSubagentConfigIsIgnored`
- [✓] AC-9 local Mail works without a model CLI or Serve — filesystem-only two-workspace fixtures in `internal/peering/peercall_test.go` and `internal/peering/handoff_test.go`
- [✓] AC-10 all six harness targets receive async/explicit-promotion guidance — `TestAllTargetsInstallAsyncPeeringGuidance`
- [✓] AC-11 old multi-CLI spec is superseded with provenance retained — `.hero/planning/features/peer-call-multi-cli/spec.md`; initiative/spec projections; supersession event evidence

## Changes
- [✓] Replace active subprocess peer call with Mail request/reply — `internal/peering/peercall.go`; active-path search found no subprocess, result-fence, auth-detection, or runner symbols
- [✓] Refactor handoff and add receiver-owned promotion/reply — `internal/peering/handoff.go`; `internal/peering/receive.go`
- [✓] Add backward-compatible Mail metadata to peering contracts — `contracts/peering/handoff.go`; `contracts/peering/peercall.go`
- [✓] Add async peer CLI, wait, prompt-file/stdin, JSON, and budget hints — `internal/cli/peer.go`
- [✓] Add Mail handoff request/receive/status while retaining legacy accept — `internal/cli/handoff.go`
- [✓] Preserve legacy reads/reconciliation without new peering-only status writes — `internal/peering/resolve.go`; new call/handoff implementations do not write specs
- [✓] Replace subprocess coverage with Mail lifecycle tests — `internal/peering/peercall_test.go`; `internal/peering/handoff_test.go`; `internal/peering/integration_test.go`
- [✓] Update canonical skill, command, generated AGENTS, help, and docs — `domains/engineering/`; `internal/install/agents_md.go`; `CROSS-REPO-PEERING.md`; `GETTING-STARTED.md`; `README.md`
- [✓] Add ignored-subagent deprecation coverage — `internal/config/peering_deprecation_test.go`
- [✓] Supersede the multi-CLI spec and retain projections — `.hero/planning/features/peer-call-multi-cli/spec.md`; `.hero/planning/initiatives/durable-attention/`

## Open items (if any)

None.

## Audit notes

None.
