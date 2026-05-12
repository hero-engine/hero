---
title: NEXT No-Op Writes — Avoid Timestamp-Only Handoff Churn
type: feature
status: completed
priority: P1
tags: [next-md, handoff, projection, git-noise, ux]
created: 2026-05-04
relations:
  - target: next-as-projection
    kind: hardens
  - target: pre-commit-auto-stage-next
    kind: reduces-noise-for
mission_alignment: |
  NEXT files are supposed to make sessions start omniscient, not create
  meaningless git activity. When Hero rewrites tracked handoff files only
  because `updated:` changed, the corpus appears active even though nothing
  useful changed. Suppressing no-op writes keeps the handoff surface trustworthy
  and reduces merge/diff noise around one of Hero's most important artifacts.
principles_check: |
  Serves #1 (it just works) by making the Stop/checkpoint hook quiet when it
  has nothing new to say, and #3 (sessions start omniscient) by preserving the
  value of NEXT without making users manage timestamp churn. The implementation
  must keep timestamps meaningful: update them when semantic content changes,
  preserve them when it does not.
horizon: now
smoke: deferred
delivery_method: manual
---

# NEXT No-Op Writes

## Goal

Stop Hero from touching tracked NEXT handoff files when their content has not
semantically changed. In particular, repeated checkpoints with the same graph,
same ask/suggestion/reflections, and same projected sections must not dirty
`.hero/NEXT.md` or `.hero/next/<user>.md` just because the frontmatter
`updated:` timestamp would be regenerated.

## Background

The current checkpoint path writes aggressively:

- legacy `.hero/NEXT.md` path strips old machine blocks and writes the result
  every time, even when the bytes are identical.
- projected `.hero/NEXT.md` is rendered by `projection.NextMD`, which stamps
  `updated: time.Now()` in frontmatter.
- `.hero/next/<user>.md` is rendered by `projection.UserHandoffMD`, which also
  stamps `updated: time.Now()` in frontmatter.
- `.hero/next/<user>.local.md` is gitignored, but is also rewritten every
  checkpoint.

The tracked files are the painful part. They show up in `git status`, get
auto-staged by hooks, and create meaningless commits/diffs when the only
observable change is a timestamp.

This violates the intent of `next-as-projection`: projections should reduce
merge churn and handoff drift, not manufacture activity.

## Design

### Introduce content-aware writes

Add a small helper in the NEXT/checkpoint code path:

```go
writeFileIfChanged(path string, content []byte, mode fs.FileMode) (changed bool, err error)
```

Behavior:

- create parent directories as needed.
- if the file does not exist, write it and return `changed=true`.
- if existing bytes equal proposed bytes, do not write and return
  `changed=false`.
- if bytes differ, write and return `changed=true`.

Use this for legacy NEXT cleanup and the local machine-state file. This avoids
plain byte-identical rewrites.

### Treat `updated:` as metadata, not semantic content

Projection renderers intentionally include `updated: <now>`, so byte comparison
alone is not enough for tracked projected files.

For `.hero/NEXT.md` and `.hero/next/<user>.md`, compare a normalized version of
the existing and proposed content where the frontmatter `updated:` value is
replaced with a stable placeholder. If normalized content is equal:

- do not write the file.
- keep the existing `updated:` timestamp.
- report `changed=false`.

If normalized content differs:

- write the proposed content with the newly generated `updated:` timestamp.
- report `changed=true`.

This keeps `updated:` meaningful: it means "the projected handoff content last
changed," not "the hook last fired."

### Keep local state pragmatic

`.hero/next/<user>.local.md` is gitignored, so timestamp churn there does not
create git noise. Still, use the byte-level no-op helper so repeated identical
machine blocks do not churn mtime unnecessarily.

Do not normalize local machine-state timestamps unless there is a real local
timestamp field causing noise. Local state is allowed to change when branch,
dirty files, hot files, or activity-since-last-checkpoint changes.

### Checkpoint output

`hero next checkpoint` can keep its current success message. In quiet mode,
there is no output either way.

Non-quiet mode may optionally print a clearer message later, but this feature
does not require a new UX surface. The important behavior is filesystem
silence when nothing changed.

## Changes

- `internal/cli/checkpoint.go`
  - add byte-level no-op writer helper.
  - add projected NEXT writer that preserves existing `updated:` when
    normalized content is unchanged.
  - use no-op writes in legacy NEXT, local state, projected project NEXT, and
    user handoff writes.
- `internal/projection/projection.go`
  - no required public API change if normalization happens at write time.
  - optional: accept an injected timestamp only if tests become cleaner.
- `internal/projection/user_handoff.go`
  - same as above; prefer write-time normalization over renderer API churn.
- `internal/cli/checkpoint_test.go` or a new focused test file
  - cover repeated checkpoint/projection calls and mtime/content stability.

## Acceptance Criteria

- **AC-1:** In legacy mode, running `hero next checkpoint` twice with no content
change leaves `.hero/NEXT.md` byte-identical and does not advance its mtime on
the second run.
✅ **passing** — byte-level no-op writes are covered by
`TestWriteFileIfChangedSkipsIdenticalContent`; full suite passed with
`go test ./...`.

- **AC-2:** In projected mode, rendering `.hero/NEXT.md` twice with the same
graph state does not rewrite the file and preserves the prior `updated:`
timestamp.
✅ **passing** — covered by
`TestWriteProjectedNextMDSkipsUpdatedOnlyChange`; full suite passed with
`go test ./...`.

- **AC-3:** Rendering `.hero/next/<user>.md` twice with the same user handoff
graph state does not rewrite the file and preserves the prior `updated:`
timestamp.
✅ **passing** — covered by
`TestWriteUserHandoffFileSkipsUpdatedOnlyChange`; full suite passed with
`go test ./...`.

- **AC-4:** When project/user handoff content changes semantically, the relevant
file is written and its `updated:` timestamp advances.
✅ **passing** — covered by `TestWriteProjectedFileUpdatesOnSemanticChange`;
full suite passed with `go test ./...`.

- **AC-5:** `hero next checkpoint -q` remains silent and exits 0 in both no-op
and changed cases.
✅ **passing** — covered by `TestNextCheckpointQuietIsSilent`; full suite
passed with `go test ./...`.

- **AC-6:** Existing checkpoint, projection, and hook tests continue to pass:
`go test ./internal/cli ./internal/projection`.
✅ **passing** — verified by `go test ./internal/cli ./internal/projection`
and full `go test ./...`.

## Kickoff

Deliver `next-noop-writes`. Focus on the checkpoint writer path, not a broader
NEXT redesign. Add a helper that skips byte-identical writes, then add
frontmatter-aware normalization so projected tracked files do not rewrite when
only `updated:` changed. Verify repeated checkpoints preserve mtime and
timestamps when semantic content is unchanged, while real content changes still
advance `updated:`.
