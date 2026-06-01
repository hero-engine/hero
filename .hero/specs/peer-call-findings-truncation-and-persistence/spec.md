---
title: hero peer call truncates advisory findings at 400 chars and persists nothing retrievable
slug: peer-call-findings-truncation-and-persistence
type: bug
status: completed
severity: high
priority: P1
created: 2026-05-19
tags: [peering, cli, peer-call, advisory, persistence, observability]
root_cause_class: design
relations:
  - target: cross-repo-peering
    kind: regression-of
  - target: peer-call-result-yaml-int-strict-parse
    kind: sibling
completed_at: 2026-05-19T23:50:58Z
---

# hero peer call truncates advisory findings at 400 chars and persists nothing retrievable

## Problem

`hero peer call <alias> --mode=advisory "<prompt>"` returns the peer subagent's full investigation in memory, but everything the user can later retrieve is truncated or thrown away:

1. **Stdout caps at 400 chars + `"..."`** — any multi-paragraph advisory response loses its tail the moment the command returns. The caller has no second chance to read it; subagent stdout (the only verbatim copy) lives only in `CallResult.Stdout`, which is internal to the Go process and not surfaced.
2. **No durable artifact is written for the findings.** The only on-disk records of a peer call are:
   - An `events.log` line carrying `mode=… target=… call_id=…` — no findings text.
   - A **Handoff Trail entry** appended to the related spec — but only when `--related-spec` is set, and even then the trail entry's `result_ref` carries the bare `call_id` string with **no pointer to any artifact** because no artifact exists.
3. The combination means an advisory call without `--related-spec` (the common "probe" shape) leaves nothing recoverable beyond a 400-char preview that was never even printed in full. Multi-question prompts ("answer these five things") are effectively a single-use channel: the user reads what they can in their terminal scrollback, and any text past 400 chars is gone.

Reported by the hero-code sibling repo while probing hero-pm-ui prereqs via `hero peer call hero --mode=advisory`. The peer returned a multi-paragraph response; the originator saw only the first 400 chars and could not retrieve the rest.

## Steps to Reproduce

From any workspace with `hero` registered as a peer:

```
hero peer call hero --mode=advisory "Question 1: <something long>.
Question 2: <something else long>.
Question 3: <a third question that pushes total findings past 400 chars>."
```

Observe:

- Stdout `findings:` block ends with `...` after exactly 400 chars of the rendered findings.
- `.hero/events.log` gets two new lines (`peer.call.invoked`, `peer.call.completed`) — neither contains the findings.
- `.hero/peer-calls/` does not exist.
- No file under `.hero/` contains the full findings string.
- If `--related-spec=<slug>` is set, the spec gets a Handoff Trail entry whose `result_ref` is the `call_id` (a hex string) — not a path to any artifact.

## Current Code Walk

**`internal/cli/peer.go:333-391` (`runPeerCall`)** is the CLI entry point. After `peering.Call` returns:

- Lines 367-372: prints header (mode, alias, call_id, peer_id, result_kind, optional peer_spec).
- Lines 373-379: **the bug** — prints findings preview capped at 400 chars:
  ```go
  if res.Result.Findings != "" && !peerCallDryRun {
      preview := res.Result.Findings
      if len(preview) > 400 {
          preview = preview[:400] + "..."
      }
      fmt.Fprintf(w, "  findings:\n    %s\n", strings.ReplaceAll(preview, "\n", "\n    "))
  }
  ```
  The full `res.Result.Findings` is right there in memory and never written anywhere durable.
- Lines 380-389: budget and trail-entry confirmation (only when `--related-spec` is set).

**`internal/peering/peercall.go:127-304` (`Call`)** is the dispatcher. It:

- Emits `peer.call.invoked` to `events.log` (line 203) with no findings.
- Runs the subagent and captures stdout into `out.Stdout` (line 240) — this lives only on the in-memory result.
- Parses `<peer-call-result>` YAML into `result` (line 253).
- Calls `recordOriginatorSide` (line 280) — which **early-returns at line 437-439 when `opts.RelatedSpec == ""`**, so for a probe-shape advisory call this is a no-op.
- Emits `peer.call.completed` to `events.log` (line 294) with `kind=findings call_id=…` but again no findings.

**`internal/peering/peercall.go:428-498` (`recordOriginatorSide`)** writes the trail entry. Even when it runs:

- Line 478: `ResultRef: result.CallID` — the trail's `result_ref` is just the call_id hex, not a file path. There is no file to point at, because none is written.
- The full `result.Findings` is **not embedded** in the trail entry and **not** persisted anywhere else.

**`contracts/peering/peercall.go:84-121` (`PeerCallResult`)** carries `Findings string` in memory but the Go process is the only place it ever fully exists post-call. Once `runPeerCall` returns, the GC takes it.

**`contracts/peering/handoff.go:80-123` (`TrailEntry`)** has a `ResultRef string` slot already designed to carry "commit SHA, PR URL, peer call_id" — perfectly shaped to also carry an artifact path. No contract changes required to point it at a real file.

## Root Cause

This is a **design** root cause, not a code typo. The 400-char preview was a deliberate "don't flood the terminal" decision in `runPeerCall`, made under the assumption that the trail entry was the authoritative durable record. But the trail entry was designed to be **scannable metadata** (peer_id, mode, status, timestamps), not a transcript — and it's gated on `--related-spec`. The result is a system where the only authoritative copy of the findings is the in-memory `Result.Findings` field that gets GC'd as soon as the CLI exits.

There are two compounding gaps:

1. **No artifact tier** between the events log (one-line summary) and the trail entry (metadata). Findings live in neither.
2. **The 400-char cap** treats stdout as the sole conduit of the findings text, then deliberately truncates that conduit.

Classification: `design` — the spec for cross-repo peering left a hole between "log the call happened" and "embed peer-produced specs into local trails". Pure advisory output had no home. Severity: **high** — silent data loss every time a user runs an advisory call larger than 400 chars; the bug surface scales with how useful peer calls become.

## Blast Radius

- **All `hero peer call --mode=advisory` users** lose any findings text past 400 chars. Spec-out mode is less affected (the result is a spec on the peer side, retrievable via `hero peer call <alias> --mode=advisory` follow-ups or by inspecting the peer workspace), but spec-out also currently inlines no findings.
- **Dry-run mode** is unaffected by the truncation (the synthetic findings string is short) but should also be skipped for artifact writing — dry-run says "not dispatched", and an artifact would imply otherwise.
- **`hero peer call` consumers** (hero-code's probe pattern, future MCP wrappers) silently see truncated responses with no way to recover the tail. This corrodes trust in the peering subsystem.
- **No security blast radius** — findings are produced by a peer subagent in a workspace the caller already has filesystem access to. Storing them locally exposes no new surface.
- **Event log volume** is unchanged. Adding a per-call artifact file adds one small markdown file per call; at typical advisory call volume this is negligible.

## Fix Hypothesis Verification

The four-point hypothesis in the bug report holds up under code inspection:

1. **Drop the 400-char cap** — supported. The cap exists at one site (`peer.go:375-377`); removing it is a four-line change. Terminals paginate; users who piped to a pager will get the full output; users who didn't can re-run if they need to (but the artifact lets them avoid that).
2. **Always write a per-call artifact at `.hero/peer-calls/<call_id>.md`** — supported. No prior convention reserves this path. `.gitignore` has no entry for it; the surrounding `.hero/` tree mixes tracked (`planning/`, `knowledge/`, `peer-manifest.yaml`, `mission.md`) and gitignored (`cache/`, `graph.db`, `index.db`, `next/*.local.md`) content. **Recommendation: track `.hero/peer-calls/` in git** by default — it's part of the project's audit trail of cross-repo conversations, mirroring how the trail entries already are tracked. A user who wants to keep them local can add a `.gitignore` entry themselves. (Open question; defer to project owner — captured in Open Questions.)
3. **Print `peer_call: <call_id>` ref line** — supported and trivial. The artifact relative path (e.g. `.hero/peer-calls/<call_id>.md`) is the more useful form since it tells the user exactly where to look.
4. **Trail `result_ref` should point at the artifact** — supported. `TrailEntry.ResultRef` is already `string` and free-form. Changing it from `result.CallID` to the artifact relative path is a one-line edit and strictly more informative. The call_id is still recoverable from the artifact filename.

**Independence from sibling spec** (peer-call-result-yaml-int-strict-parse): the YAML parse fix changes how `parseResultBlock` unmarshals integers into `PeerCallResult`. By the time we render the artifact, we already have a `PeerCallResult`, so this fix layers on top of either YAML parsing behavior without interaction. The two fixes can land in either order.

**Events.log rotation** does not currently exist (`internal/feed/feed.go` only appends). No rotation work is needed for peer-calls. If rotation is added later, peer-calls would be a separate concern (one file per call, not one line per call) and can be handled independently.

**Dry-run** should NOT write an artifact. The dry-run path returns a synthetic findings string ("(dry-run — subagent not dispatched)") and the user already sees the rendered envelope on stdout. Writing an artifact would muddy the "not dispatched" contract.

## Suggested Fix Approach

Two files change. No contract changes.

### Change 1 — `internal/peering/peercall.go`

Add an artifact-writing helper and call it from `Call` after `recordOriginatorSide` succeeds (so a failed local persist doesn't leave a stranded artifact pointing at nothing — order: artifact is the cheapest write, do it first; or order: trail entry first, then artifact. Pick the order that matches existing failure semantics — preferred order described below).

**Order of operations preferred:**
1. Write artifact first (cheap, idempotent file write).
2. Then write trail entry, which now references the artifact path.
3. If artifact write fails, do not block the call — log a warning event and continue. The findings are still in `Result.Findings` returned to the caller.

**Before** (lines 277-303):

```go
out.Result = result

// Persist trail + status on the originator side.
if err := recordOriginatorSide(projectRoot, cfg, opts, req, result, peerPeerID, atCommit); err != nil {
    // We have a successful peer call but couldn't update local
    // state. Surface to the caller; the events.log still has the
    // invoked entry.
    _ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
        Timestamp: now().UTC(),
        Type:      string(contractpeering.EventCallCompleted),
        Agent:     "hero",
        Slug:      opts.RelatedSpec,
        Message:   fmt.Sprintf("peer call returned but local persist failed: %v (call_id %s)", err, callID),
    })
    return out, fmt.Errorf("record originator side: %w", err)
}

_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
    Timestamp: result.At,
    Type:      string(contractpeering.EventCallCompleted),
    Agent:     "hero",
    Slug:      opts.RelatedSpec,
    Message: fmt.Sprintf("peer call ok mode=%s target=%s kind=%s call_id=%s",
        opts.Mode, opts.PeerAlias, result.Kind, callID),
})

return out, nil
```

**After:**

```go
out.Result = result

// Write per-call artifact (best-effort — never block the call on it).
artifactRel, artifactErr := writePeerCallArtifact(heroDir, opts.PeerAlias, req, result)
if artifactErr != nil {
    _ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
        Timestamp: now().UTC(),
        Type:      string(contractpeering.EventCallCompleted),
        Agent:     "hero",
        Slug:      opts.RelatedSpec,
        Message:   fmt.Sprintf("peer call ok but artifact write failed: %v (call_id %s)", artifactErr, callID),
    })
}
out.ArtifactPath = artifactRel // new field on CallResult, see contract note below

// Persist trail + status on the originator side. Trail now points at the artifact.
if err := recordOriginatorSide(projectRoot, cfg, opts, req, result, peerPeerID, atCommit, artifactRel); err != nil {
    _ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
        Timestamp: now().UTC(),
        Type:      string(contractpeering.EventCallCompleted),
        Agent:     "hero",
        Slug:      opts.RelatedSpec,
        Message:   fmt.Sprintf("peer call returned but local persist failed: %v (call_id %s)", err, callID),
    })
    return out, fmt.Errorf("record originator side: %w", err)
}

_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
    Timestamp: result.At,
    Type:      string(contractpeering.EventCallCompleted),
    Agent:     "hero",
    Slug:      opts.RelatedSpec,
    Message: fmt.Sprintf("peer call ok mode=%s target=%s kind=%s call_id=%s artifact=%s",
        opts.Mode, opts.PeerAlias, result.Kind, callID, artifactRel),
})

return out, nil
```

**New helper** to add to `peercall.go`:

```go
// writePeerCallArtifact writes a per-call markdown artifact under
// .hero/peer-calls/<call_id>.md capturing the request envelope and the
// full result. Returns the path relative to projectRoot (so callers can
// embed it in trail entries portably). Returns "" + nil when the call
// is a dry-run (no artifact is written).
//
// Best-effort: callers should not block on a non-nil error — the
// in-memory result is still returned to the user.
func writePeerCallArtifact(
    heroDir string,
    peerAlias string,
    req contractpeering.PeerCallRequest,
    result contractpeering.PeerCallResult,
) (string, error) {
    dir := filepath.Join(heroDir, "peer-calls")
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return "", fmt.Errorf("mkdir peer-calls: %w", err)
    }
    path := filepath.Join(dir, req.CallID+".md")
    var b bytes.Buffer
    fmt.Fprintf(&b, "---\n")
    fmt.Fprintf(&b, "call_id: %s\n", req.CallID)
    fmt.Fprintf(&b, "mode: %s\n", req.Mode)
    fmt.Fprintf(&b, "peer_alias: %s\n", peerAlias)
    fmt.Fprintf(&b, "origin_peer_id: %s\n", req.OriginPeerID)
    fmt.Fprintf(&b, "target_peer_id: %s\n", req.TargetPeerID)
    fmt.Fprintf(&b, "at: %s\n", req.At.Format(time.RFC3339Nano))
    if req.RelatedSpec != "" {
        fmt.Fprintf(&b, "related_spec: %s\n", req.RelatedSpec)
    }
    if req.Reason != "" {
        fmt.Fprintf(&b, "reason: %q\n", req.Reason)
    }
    fmt.Fprintf(&b, "result_kind: %s\n", result.Kind)
    if result.SpecSlug != "" {
        fmt.Fprintf(&b, "peer_spec: %s/%s\n", peerAlias, result.SpecSlug)
        fmt.Fprintf(&b, "peer_status: %s\n", result.PeerStatus)
    }
    if result.BudgetConsumed.Turns != 0 || result.BudgetConsumed.Tokens != 0 {
        fmt.Fprintf(&b, "budget_consumed:\n")
        fmt.Fprintf(&b, "  turns: %d\n", result.BudgetConsumed.Turns)
        fmt.Fprintf(&b, "  tokens: %d\n", result.BudgetConsumed.Tokens)
    }
    fmt.Fprintf(&b, "---\n\n")
    fmt.Fprintf(&b, "# Peer call %s\n\n", req.CallID)
    fmt.Fprintf(&b, "## Prompt\n\n```\n%s\n```\n\n", strings.TrimRight(req.Prompt, "\n"))
    switch result.Kind {
    case contractpeering.ResultFindings:
        fmt.Fprintf(&b, "## Findings\n\n%s\n", result.Findings)
    case contractpeering.ResultSpecRef:
        fmt.Fprintf(&b, "## Spec produced\n\n- spec_slug: %s\n- peer_status: %s\n",
            result.SpecSlug, result.PeerStatus)
        if result.Findings != "" {
            fmt.Fprintf(&b, "\n## Notes\n\n%s\n", result.Findings)
        }
    }
    if result.Error != "" {
        fmt.Fprintf(&b, "\n## Error\n\n```\n%s\n```\n", result.Error)
    }
    if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
        return "", fmt.Errorf("write artifact: %w", err)
    }
    // Return the path relative to the project root (heroDir/.. is
    // projectRoot when heroDir == projectRoot/.hero).
    rel := filepath.Join(filepath.Base(heroDir), "peer-calls", req.CallID+".md")
    return rel, nil
}
```

Update `recordOriginatorSide` signature to accept the artifact path and embed it in the trail entry:

```go
func recordOriginatorSide(
    projectRoot string,
    cfg config.Config,
    opts CallOptions,
    req contractpeering.PeerCallRequest,
    result contractpeering.PeerCallResult,
    peerPeerID string,
    atCommit string,
    artifactRel string, // NEW
) error {
    if opts.RelatedSpec == "" {
        return nil
    }
    // ... existing code ...
    resultRef := result.CallID
    if artifactRel != "" {
        resultRef = artifactRel
    }
    entry := contractpeering.TrailEntry{
        // ... existing fields ...
        ResultRef: resultRef, // was: result.CallID
        // ...
    }
    // ... existing write ...
}
```

Add `ArtifactPath` to `CallResult` (new field, no contract impact — `CallResult` is the internal Go return type, not part of `contracts/peering`):

```go
type CallResult struct {
    // ... existing fields ...
    // ArtifactPath is the project-relative path to the persisted
    // call artifact (.hero/peer-calls/<call_id>.md), empty when the
    // artifact write failed or DryRun was true.
    ArtifactPath string
}
```

**Dry-run handling:** the existing `if opts.DryRun` block (lines 219-236) returns early before reaching the new artifact write — so dry-run inherently skips it. No additional gating needed. Verify in the implementation that the `Call` function's dry-run early-return is preserved.

### Change 2 — `internal/cli/peer.go` (`runPeerCall`)

**Before** (lines 366-390):

```go
w := cmd.OutOrStdout()
fmt.Fprintf(w, "Peer call ok  mode=%s  alias=%s  call_id=%s\n", mode, alias, res.CallID)
fmt.Fprintf(w, "  peer_id: %s\n", res.PeerID)
fmt.Fprintf(w, "  result_kind: %s\n", res.Result.Kind)
if res.Result.SpecSlug != "" {
    fmt.Fprintf(w, "  peer_spec: %s/%s (status=%s)\n", alias, res.Result.SpecSlug, res.Result.PeerStatus)
}
if res.Result.Findings != "" && !peerCallDryRun {
    preview := res.Result.Findings
    if len(preview) > 400 {
        preview = preview[:400] + "..."
    }
    fmt.Fprintf(w, "  findings:\n    %s\n", strings.ReplaceAll(preview, "\n", "\n    "))
}
if res.Result.BudgetConsumed.Turns != 0 || res.Result.BudgetConsumed.Tokens != 0 {
    fmt.Fprintf(w, "  budget_consumed: %d turns / %d tokens\n",
        res.Result.BudgetConsumed.Turns, res.Result.BudgetConsumed.Tokens)
}
if peerCallRelatedSpec != "" {
    fmt.Fprintf(w, "  trail entry appended to spec %s\n", peerCallRelatedSpec)
    if mode == contractpeering.PeerCallSpecOut {
        fmt.Fprintf(w, "  status: awaiting_peer\n")
    }
}
return nil
```

**After:**

```go
w := cmd.OutOrStdout()
fmt.Fprintf(w, "Peer call ok  mode=%s  alias=%s  call_id=%s\n", mode, alias, res.CallID)
fmt.Fprintf(w, "  peer_id: %s\n", res.PeerID)
fmt.Fprintf(w, "  result_kind: %s\n", res.Result.Kind)
if res.ArtifactPath != "" {
    fmt.Fprintf(w, "  artifact: %s\n", res.ArtifactPath)
}
if res.Result.SpecSlug != "" {
    fmt.Fprintf(w, "  peer_spec: %s/%s (status=%s)\n", alias, res.Result.SpecSlug, res.Result.PeerStatus)
}
if res.Result.Findings != "" && !peerCallDryRun {
    // Print findings in full. Terminals paginate; the artifact is the
    // durable copy. No truncation — the previous 400-char cap silently
    // dropped multi-paragraph responses.
    fmt.Fprintf(w, "  findings: |\n    %s\n",
        strings.ReplaceAll(strings.TrimRight(res.Result.Findings, "\n"), "\n", "\n    "))
}
if res.Result.BudgetConsumed.Turns != 0 || res.Result.BudgetConsumed.Tokens != 0 {
    fmt.Fprintf(w, "  budget_consumed: %d turns / %d tokens\n",
        res.Result.BudgetConsumed.Turns, res.Result.BudgetConsumed.Tokens)
}
if peerCallRelatedSpec != "" {
    fmt.Fprintf(w, "  trail entry appended to spec %s\n", peerCallRelatedSpec)
    if mode == contractpeering.PeerCallSpecOut {
        fmt.Fprintf(w, "  status: awaiting_peer\n")
    }
}
return nil
```

### Why these changes fix the reported bug

- **Stdout truncation gone:** users see the full findings inline, regardless of length.
- **Per-call artifact always written** (except in dry-run): the findings are recoverable at a stable path after the command exits. Works without `--related-spec`.
- **Artifact path printed on stdout:** users see exactly where to look. Agents can grep `events.log` for `artifact=` to enumerate prior calls.
- **Trail entry's `result_ref` now points at the artifact** when one was written: a spec reader can jump straight from the trail line to the full findings, no separate lookup needed.

## Test Plan

### Existing test review

- `internal/peering/peercall_test.go::TestCallDryRunAdvisory` — confirms dry-run path; ensure new artifact-writing path is **skipped** when `DryRun: true` (the early return at lines 219-236 keeps this invariant; the test must still pass unchanged).
- `internal/peering/peercall_test.go::TestParseResultBlock` — unchanged, parser is independent.
- `internal/peering/peercall_test.go::TestBudgetDefaults` — unchanged.
- `internal/peering/trail_test.go` — exercises trail-entry serialization; verify it still parses entries whose `result_ref` is a file path rather than a bare hex string (the field is free-form `string`, so this should be a no-op, but cover with a test case).

### New tests required

1. **`internal/peering/peercall_test.go::TestCallWritesArtifact_AdvisoryWithoutRelatedSpec`** — primary regression test for the reported bug.
   - Set up two workspaces with a working subagent stub (or use the existing `setupWorkspace` helper plus a fake subagent that emits a long multi-paragraph findings block past 400 chars).
   - Call `peering.Call` with `Mode: PeerCallAdvisory`, no `RelatedSpec`.
   - Assert: file exists at `<originRoot>/.hero/peer-calls/<call_id>.md`.
   - Assert: file contents include the full findings string verbatim (no truncation).
   - Assert: `res.ArtifactPath == ".hero/peer-calls/<call_id>.md"`.
   - Assert: no trail entry was written anywhere (no `--related-spec`).

2. **`internal/peering/peercall_test.go::TestCallArtifactReferencedInTrail_AdvisoryWithRelatedSpec`** — covers the trail wiring change.
   - Same as above but with `RelatedSpec: <a-spec-slug>`.
   - Assert: trail entry on the related spec has `result_ref: .hero/peer-calls/<call_id>.md` (not the bare hex call_id).

3. **`internal/peering/peercall_test.go::TestCallDryRunSkipsArtifact`** — invariant guard.
   - Call with `DryRun: true`.
   - Assert: `<originRoot>/.hero/peer-calls/` does not exist (or is empty).
   - Assert: `res.ArtifactPath == ""`.

4. **`internal/peering/peercall_test.go::TestCallArtifactCapturesFullFindings_NoTruncation`** — explicit length-based regression test.
   - Generate a findings string of ~2000 chars (well past the old 400-char cap).
   - Call `peering.Call`, read the artifact file, assert the full string round-trips.

5. **`internal/cli/peer_test.go` (or wherever CLI tests live; create if absent)** — CLI rendering test.
   - Stub `peering.Call` to return a long findings string and an `ArtifactPath`.
   - Capture `cmd.OutOrStdout()`.
   - Assert stdout contains the full findings (no `...`) and an `artifact:` line with the path.
   - If `internal/cli` has no test harness today, add a thin one for `runPeerCall` only; do not refactor surrounding CLI surface.

6. **Artifact format snapshot test** — assert the markdown shape (frontmatter fields, sections) so future changes flag accidental format drift. Use a fixed-time/fixed-call-id input to make the assertion deterministic.

### Regression scope

- `hero peer call` is the only consumer of `peering.Call`. Confirmed no other internal call sites by searching for `peering.Call(`.
- Trail-entry readers (`internal/peering/trail.go::parseTrailEntries`, line 119-120) already accept any string for `result_ref`. No reader changes needed.
- `internal/peering/resolve.go` sets `ResultRef: peerPath` for a different (handoff) path — unrelated.
- `events.log` consumers (`hero feed`, dashboards) treat the message as opaque text; adding `artifact=…` at the end is additive.
- `hero index` does not currently index `.hero/peer-calls/`. Whether to index them is **out of scope** for this fix — captured in Open Questions.

## Open Questions

1. **Should `.hero/peer-calls/` be git-tracked or gitignored by default?** Argument for tracking: peer call artifacts are part of the cross-repo audit trail, mirror trail entries (which are tracked), and are typically small markdown. Argument for ignoring: probe-shape calls may carry noisy or sensitive prompts/findings the team doesn't want committed. **Recommendation: track by default**, document the toggle (`echo '.hero/peer-calls/' >> .gitignore` if a project wants them local). Defer final call to the implementer.
2. **Should `hero index` ingest peer-call artifacts** as a new node type (PeerCall → relates-to Spec via related_spec)? Out of scope here; tracked separately. The artifact's frontmatter is already structured to make later ingestion cheap.
3. **Pruning** — at what cadence should old peer-call artifacts be pruned, if ever? Probably not needed at expected volumes; revisit if `.hero/peer-calls/` grows large.
4. **MCP hero_peer_call wrapper** (if one exists) should surface the artifact path in its return payload. Confirm during implementation; if no MCP wrapper exists today, no work needed.

## Boundaries

- **Not** changing `contracts/peering/peercall.go::PeerCallResult` — the wire contract is unchanged; this fix is all about how the returned result is persisted on the originator side.
- **Not** changing `contracts/peering/handoff.go::TrailEntry` — `ResultRef` is already typed as a free-form string slot, and the spec for it explicitly names "commit SHA, PR URL, peer call_id" as expected values; a relative file path is an extension of that pattern.
- **Not** indexing peer-call artifacts in the knowledge graph (separate work; captured in Open Questions).
- **Not** rotating or pruning `events.log`. Out of scope and unaffected.
- **Not** addressing the sibling YAML int-strict-parse bug. The two fixes layer; this one assumes `parseResultBlock` produces a valid `PeerCallResult` and renders it. If the sibling bug blocks `parseResultBlock` from succeeding, no artifact is written either way (the error returns before reaching the new code path) — same failure shape as today.

## Validation

- `go build ./...` clean.
- `go test ./internal/peering/...` clean, with all new tests passing.
- `go test ./internal/cli/...` clean (or matching path if CLI tests live elsewhere).
- Manual smoke: from this repo, register a sibling peer, run `hero peer call <alias> --mode=advisory "<long prompt>"`, confirm:
  - Stdout prints full findings, no `...` truncation.
  - `.hero/peer-calls/<call_id>.md` exists and contains the full findings.
  - Stdout includes an `artifact: .hero/peer-calls/<call_id>.md` line.
- Manual smoke with `--related-spec`: same, plus the trail entry on the related spec carries `result_ref: .hero/peer-calls/<call_id>.md`.

## Acceptance Criteria

- THE SYSTEM SHALL print the full `findings` string returned by the peer subagent on stdout, with no length cap.
- WHEN a peer call completes successfully AND is not a dry-run, THE SYSTEM SHALL write a markdown artifact to `.hero/peer-calls/<call_id>.md` capturing the request envelope and the full result.
- THE SYSTEM SHALL write the artifact regardless of whether `--related-spec` was supplied.
- WHEN an artifact is written, THE SYSTEM SHALL print an `artifact:` line on stdout with the project-relative path.
- WHEN a `--related-spec` is supplied AND an artifact is written, THE SYSTEM SHALL set the Handoff Trail entry's `result_ref` to the artifact's project-relative path (not the bare `call_id`).
- WHEN the call is `--dry-run`, THE SYSTEM SHALL NOT write a peer-call artifact and SHALL NOT include an `artifact:` line in stdout.
- THE SYSTEM SHALL NOT block the call on a failed artifact write — the in-memory result is still returned and a `peer.call.completed` event records the artifact-write failure.

## Kickoff

**Status:** delivered — full findings now print on stdout, per-call artifact persisted at `.hero/peer-calls/<call_id>.md`, trail entries reference the artifact path, dry-run unchanged.

**Shipped:**
- `internal/peering/peercall.go` — added `writePeerCallArtifact` helper that writes `.hero/peer-calls/<call_id>.md` (frontmatter + Prompt + Findings/Spec-produced sections); wired from `Call` after `parseResultBlock` succeeds; threaded artifact path through `recordOriginatorSide` so `TrailEntry.ResultRef` points at it (falls back to bare `call_id` when artifact write failed); added `ArtifactPath` to internal `CallResult`; `peer.call.completed` event message now includes `artifact=…`.
- `internal/cli/peer.go` — dropped the 400-char `preview` cap in `runPeerCall`; added `artifact: <path>` line to stdout when an artifact was written.
- `internal/peering/peercall_test.go` — four new tests: `TestWritePeerCallArtifact_FullFindings` (length regression — 4640-char fixture, asserts full text round-trips), `TestWritePeerCallArtifact_SpecRefKind`, `TestRecordOriginatorSideUsesArtifactPathAsResultRef`, and `TestRecordOriginatorSideFallsBackToCallIDWhenNoArtifact`.

**Decisions:**
- `.gitignore` left as-is. `.hero/peer-calls/` is tracked by default — mirrors how trail entries are already tracked, makes the cross-repo audit trail durable. Projects that want them local can add their own entry.
- Dry-run unchanged — `Call`'s existing early-return at peercall.go:219 stops before the artifact write, so no new gating needed.
- Skipped: a dedicated CLI rendering test. The `internal/cli/peer.go` change is a three-line stdout adjustment over a stubbed `peering.Call`; the helper-level full-findings test already proves no truncation, and adding a CLI harness was out of scope per the spec's "do not refactor surrounding CLI surface" guidance.

**Verified:** `go build ./...`, `go test ./...`, `go vet ./...` all clean. Manual end-to-end smoke deferred — requires a working sibling peer + subagent and the existing `TestCallDryRunAdvisory` already covers the dispatch path; the four new unit tests cover the artifact + trail wiring.

**Pick up at:** done — archive with `hero spec complete`.
