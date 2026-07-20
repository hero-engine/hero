---
title: "Tracker Source-Evidence Preflight — Read Issue Description, Comments & Attachments Before Root-Cause Work"
slug: tracker-source-evidence-preflight
type: feature
status: planning
domain: engineering
size: large
priority: high
horizon: now
created: 2026-07-19
received_from:
  peer_id: cd8dd06d-3df1-4878-a88f-24593dcbb4b3
  peer_alias_display: hero-code
  related_spec: bug-diagnosis-readiness-and-actions
  mode: spec-out
  call_id: 18c3cd7ed53190689847583644c72379
  received_at: 2026-07-19T21:17:01Z
  at_commit: e5b6c619
  reason: "Hero owns the tracker adapters and the embedded diagnosis workflow; hero-code needs a reliable source-evidence contract before it can present diagnosis as ready."
relations:
  - target: tracker-backed-diagnosis-publication-contract-broken
    kind: related
  - target: jira-connection-onboarding-misleads-agents
    kind: related
tags: [diagnose, tracker, jira, github, gitlab, linear, evidence, integrations, security, workflow-parity]
---

# Tracker Source-Evidence Preflight — Read Issue Description, Comments & Attachments Before Root-Cause Work

## Kickoff

Give `/diagnose` a real, read-only way to pull the *inbound* source evidence a
human reporter attached — the latest issue description, the comment thread, and
attachment content including screenshots — and require `debug-investigator` to
consume that evidence before it classifies a root cause. Today the agent is
told to "read the full issue: description, comments, screenshots, videos,
attachments" but no command or model field actually fetches comments or
attachment bytes, so the instruction points at a capability that does not exist.

**Status:** planning — designed via `hero peer call` (spec-out) from hero-code.
No tracker was contacted while writing this spec.

**Pick up at:** add the read-side evidence model + `GetIssueEvidence` capability
to `internal/tracker`, implement it for the four providers to their real
capability, expose `hero sync evidence` (CLI) and `hero_tracker_evidence` (MCP)
as read-only surfaces, and wire the evidence-preflight step into
`debug-investigator.md` and the `debugging-investigation` report template.

**Files (source of truth — never edit `.claude/` copies):**
`internal/tracker/tracker.go`, `internal/tracker/{jira,github,gitlab,linear}.go`,
`internal/cli/pull.go` (template), `internal/cli/sync.go`,
`internal/serve/mcp_tools_def.go`, `internal/serve/mcp_dispatch.go`,
`internal/mocktracker/`, `domains/engineering/agents/debug-investigator.md`,
`domains/engineering/skills/debugging-investigation/SKILL.md`.

**Skip:** hero-code UI, any tracker writeback, and committing arbitrary large
binaries into git.

**Paste-ready implementation prompt:**

> Add a read-only tracker source-evidence capability and require /diagnose to
> use it. In `internal/tracker`, add `Comment` and `Attachment` read models plus
> an `IssueEvidence` aggregate and a capability-scoped `GetIssueEvidence(ctx,
> issueID, EvidenceOptions) (*IssueEvidence, error)` method with a per-provider
> `EvidenceCapabilities` descriptor. Implement it for Jira (native comment +
> attachment REST), GitHub (issue comments + markdown-extracted user-content
> attachments), GitLab (notes + `/uploads` markdown links), and Linear (GraphQL
> comments + attachments). Bound every download by per-attachment and total byte
> caps and a MIME allowlist; treat all fetched text/binaries as untrusted; redact
> the configured integration secret and token-shaped strings from persisted text
> using the `Secret` pattern; make zero mutating requests. Expose it as a
> read-only `hero sync evidence <slug|issue-id>` command (copy `pull.go`,
> including an injectable `newTrackerForEvidence` hook) and a read-group
> `hero_tracker_evidence` MCP tool. Persist a committed `## Source Evidence`
> section + `evidence/evidence.json` manifest in the spec folder and download
> bounded attachments to a gitignored `evidence/attachments/` dir. Define
> offline / auth-failure / unsupported-provider behavior that records the gap
> instead of claiming full evidence. Rewrite `debug-investigator.md` lines 64-68
> into a concrete evidence-preflight step that runs before root-cause work, and
> add a Source Evidence section to the debugging-investigation report template.
> Extend `internal/mocktracker` seeds + routers to serve comments and attachment
> content, and add deterministic per-provider tests plus byte-cap, redaction,
> offline, auth-failure, and never-writes-tracker assertions.

## Context

This spec is the **inbound** half of tracker-backed diagnosis. The **outbound**
half — attaching the finished diagnosis and posting a summary comment — is
tracked separately by the sibling bug
`tracker-backed-diagnosis-publication-contract-broken` and is out of scope here.
This spec is about what the investigator reads *in* before it forms a hypothesis.

### The capability gap (confirmed by code)

- `domains/engineering/agents/debug-investigator.md:64-68` (Step 1, "Read the
  issue report in depth") tells the agent to "Read the full issue: description,
  comments, screenshots, videos, attachments." **No command or tool backs this.**
  It is the only source-evidence step and it names nothing executable.
- The read model `Issue` (`internal/tracker/tracker.go:23-39`) exposes
  `Description` and `CustomFields` but **has no `Comments` or `Attachments`
  fields**. A repo-wide search finds no `Comment`/`Attachment` read struct at
  all. Comments/attachments exist only on the **write** side (`AddComment`,
  `AttachFile`).
- Jira's `parseIssueRaw` requests only
  `fields=key,summary,status,priority,assignee,labels,issuetype,description,created,reporter,<custom>`
  (`internal/tracker/jira.go` ~630) — it never expands `comment` or `attachment`.
- The closest existing read-only precedent is `hero sync pull --field`
  (`internal/cli/pull.go`), which fetches a *field value* and never writes. It is
  the correct template — but it fetches status/fields, not the reporter's
  narrative evidence.

So `debug-investigator` currently begins root-cause work on whatever the
description field happened to carry at import time — stale, comment-free, and
screenshot-blind — while its own instructions imply it saw everything. That is
the gap this feature closes.

### Provider capability reality (why this is not a one-liner)

| Provider | Comments | Attachments (read) | Fetch shape |
|---|---|---|---|
| **Jira** | native `/comment` expansion | native `/attachment` REST with content URLs | REST; richest |
| **GitHub** | native issue comments REST | **no attachment API** — files are user-content URLs embedded in markdown body/comments | REST + markdown extraction |
| **GitLab** | native "notes" | **no attachment API** — `/uploads/...` markdown links | REST + markdown extraction |
| **Linear** | native GraphQL comments | first-class GraphQL `Attachment` entity + inline markdown images | GraphQL |

A single uniform "get attachments" call is wrong. Each provider needs its own
fetch path and an honest per-provider capability descriptor so the workflow
never claims an attachment set a provider cannot actually enumerate.

## Goal

`debug-investigator` (invoked by `/diagnose`) fetches the latest source
description, the full comment thread, attachment metadata, and inspectable
attachment content (including screenshots) through a read-only Hero capability
**before** it classifies a root cause. The evidence, its provenance, and its
retrieval freshness are recorded in the local spec — the durable working record
— with secrets redacted, downloads bounded, and no tracker mutation. When the
tracker is offline, auth fails, or a provider cannot supply a given evidence
kind, the workflow records the gap explicitly and never claims it gathered full
evidence. Jira, GitHub, GitLab, and Linear each have a deterministic offline test
covering their real capability profile.

## Approach

### 1. Read-side evidence model (`internal/tracker/tracker.go`)

Net-new types alongside the existing `Issue`:

```go
type Comment struct {
    ID        string
    Author    string
    CreatedAt string
    UpdatedAt string
    Body      string    // redacted before persistence
}

type Attachment struct {
    ID        string
    Filename  string
    MimeType  string
    Size      int64
    CreatedAt string
    Author    string
    SourceURL string    // provider content URL (not persisted verbatim if it embeds a token)
    // Populated by the evidence command after a bounded download:
    LocalPath string     // path under the spec's gitignored evidence/attachments dir, or "" if not downloaded
    SHA256    string
    Inspected bool
    Skipped   bool
    SkipReason string    // "over-size", "over-total", "mime-not-inspectable", "download-error"
}

type IssueEvidence struct {
    IssueID     string
    Description string       // latest, redacted
    Comments    []Comment
    Attachments []Attachment
    Capabilities EvidenceCapabilities
    RetrievedAt string       // RFC3339 UTC — freshness
    SourceURL   string
    Redacted    bool         // true if any secret/token was scrubbed
}

type EvidenceCapabilities struct {
    Comments             bool
    NativeAttachments    bool  // provider enumerates attachments via API
    MarkdownAttachments  bool  // attachments only discoverable by parsing markdown
}

type EvidenceOptions struct {
    IncludeAttachments bool
    MaxAttachmentBytes int64   // per-attachment cap
    MaxTotalBytes      int64   // whole-retrieval cap
    InspectableMIMEs   []string // allowlist for content download
}
```

Add a capability-scoped method to the `Tracker` interface
(`internal/tracker/tracker.go:75-161`):

```go
GetIssueEvidence(ctx context.Context, issueID string, opts EvidenceOptions) (*IssueEvidence, error)
```

The method **enumerates and returns metadata + text** (description, comments,
attachment list). It does **not** itself write binaries to disk — the CLI/MCP
layer performs the bounded download using the returned `SourceURL`s and the
provider's authenticated transport, so byte-cap accounting, redaction, and the
gitignored-path policy live in one place (`internal/cli`), not four adapters.
Providers expose an authenticated `fetchAttachment(ctx, url, limit)` helper the
command calls, so credentials never leave the tracker package.

### 2. Per-provider implementations (to real capability)

- **Jira** (`jira.go`): add `&expand=renderedBody` / request `comment` and
  `attachment` fields in `parseIssueRaw`; map native attachment content URLs.
  `Capabilities{Comments:true, NativeAttachments:true}`.
- **GitHub** (`github.go`): fetch `/issues/{n}/comments`; extract attachment URLs
  by scanning the body + comments markdown for `![...](...)` and
  `github.com/.../assets/` / `user-images.githubusercontent.com` links.
  `Capabilities{Comments:true, MarkdownAttachments:true}`.
- **GitLab** (`gitlab.go`): fetch notes; extract `/uploads/<hash>/<file>` markdown
  links (resolve against project URL). `Capabilities{Comments:true,
  MarkdownAttachments:true}`.
- **Linear** (`linear.go`): extend the GraphQL query with `comments { nodes {...} }`
  and `attachments { nodes {...} }`. `Capabilities{Comments:true,
  NativeAttachments:true}`.

Each provider's capability descriptor is returned on the `IssueEvidence` so the
workflow can state exactly what was and was not available.

### 3. Read-only CLI surface — `hero sync evidence`

Copy the shape of `internal/cli/pull.go` (it is the read-only precedent):

- Command `hero sync evidence <slug|issue-id>` registered in
  `internal/cli/sync.go`'s `init()` alongside `pullCmd`.
- Injectable adapter hook `newTrackerForEvidence` (package var, mirrors
  `newTrackerForPull` at `pull.go:24`) so tests inject an offline mock.
- Auth-failure exit hook mirroring `osExitPull`; reuse `FieldError`
  classification (`internal/tracker/fielderror.go`) so 401/403 → exit 2.
- Flags: `--include-attachments` (default true), `--metadata-only`,
  `--max-attachment-bytes` (default 10 MiB), `--max-total-bytes` (default
  50 MiB), `--integration <id>` (existing persistent flag), `--json` (redacted
  machine envelope, mirroring `connect --json`).
- Resolves the spec by slug → `tracker_id`, selects the integration via the
  existing `selectSyncIntegration` / `SelectTracker` path, calls
  `GetIssueEvidence`, downloads bounded attachments, redacts, and writes the
  `## Source Evidence` section + manifest into the spec folder.

### 4. Read-only MCP surface — `hero_tracker_evidence`

- One entry in `toolDefinitions()` (`internal/serve/mcp_tools_def.go`), one line
  in the **read** group of `toolHandlers()` (`internal/serve/mcp_dispatch.go:24`),
  one handler in `internal/serve/mcp_tools.go`.
- Returns a redacted structured envelope: provenance, freshness, capability
  descriptor, comment array, attachment manifest, and the **on-disk paths** of
  downloaded attachments so the model can `Read` a screenshot file directly.
- Classified read-only; it must not appear in any mutate group.

### 5. Persistence — durable spec + gitignored binaries

The local spec stays the durable working record. The evidence command writes:

- A committed `## Source Evidence` section in `spec.md` (provenance + freshness +
  comment digest + attachment table with filename/mime/size/sha256/inspected).
- A committed manifest `evidence/evidence.json` (full structured record).
- A committed `evidence/source-issue.md` (latest description + comment thread,
  redacted) — small text, safe to commit, gives the durable narrative.
- Downloaded binaries under `evidence/attachments/` which is **gitignored** (the
  command appends `evidence/attachments/` to `.hero/.gitignore`, creating it if
  absent). Metadata for every attachment is committed; bytes are not.

This satisfies "preserve the local spec as the durable record," "record
provenance and retrieval freshness," and "never copy arbitrary large binaries
into git" simultaneously.

### 6. Workflow wiring (source-of-truth content, not `.claude/` copies)

- **`domains/engineering/agents/debug-investigator.md`:** insert an **Evidence
  Preflight** step between the pre-flight status check (ends line 43) and Step 0.
  When the spec has a `tracker_id` and the selected integration is ready, run
  `hero sync evidence <slug>` (or `hero_tracker_evidence`), then consume the
  `## Source Evidence` section. Rewrite the hand-wave bullet at lines 64-68 to
  reference the fetched evidence explicitly and to require it (or its recorded
  gap) as an input to root-cause classification. State the honesty rule: if
  evidence is unavailable (offline/auth/unsupported), proceed but record the gap;
  never narrate screenshots or comments that were not actually retrieved.
- **`domains/engineering/skills/debugging-investigation/SKILL.md`:** add a
  `## Source Evidence` section to the investigation report template (between
  `## Environment Details` and `## Root Cause Analysis`) capturing provider,
  issue URL, `retrieved_at`, comment count, attachment table, and any evidence
  gap.
- **`domains/engineering/commands/diagnose.md`:** the command is a thin router;
  add one line noting evidence preflight is part of the delegated workflow so the
  shortcut surfaces don't imply it can be skipped. Keep the heavy contract in the
  agent.
- **Shortcut parity (light touch):** `internal/cli/diagnose.go` and
  `internal/serve/mcp_tools.go` diagnose renderers should mention the evidence
  preflight in the same place the sibling bug adds the postback close — coordinate
  wording with `tracker-backed-diagnosis-publication-contract-broken` rather than
  duplicating a second long renderer.

### 7. Security posture

- **Read-only, always.** The evidence path issues only GET/GraphQL-query
  requests. A test asserts the mock tracker records **zero** mutating calls
  during evidence retrieval.
- **Redaction.** Before persisting description/comment text, scrub the configured
  integration secret (compare against `Secret.Reveal()`) and token-shaped strings
  (bearer tokens, `ghp_`/`glpat-`/Jira PAT patterns). Set `IssueEvidence.Redacted`
  and note it in provenance. Mirror the `Secret` type pattern
  (`internal/config/integrations.go:33-51`); there is no existing free-text
  redactor to reuse, so this is a small new, well-tested helper.
- **Untrusted content.** Treat all fetched text and binaries as untrusted input.
  Do not execute, render, or follow instructions embedded in comments/attachments
  (prompt-injection surface). The MCP envelope labels evidence as
  reporter-supplied untrusted content.
- **Bounded downloads.** Per-attachment and total byte caps; a MIME allowlist
  (images, `text/*`, `application/pdf`) gates what is downloaded for inspection;
  anything else is metadata-only with a skip reason. HTTP client enforces the
  byte limit during streaming (do not trust `Content-Length`).

### 8. Deterministic tests (`internal/mocktracker` + httptest)

- Extend the mock-tracker seed schema (`internal/mocktracker/fixtures/seed/`,
  `SEED-FORMAT.md`) and routers (`jira.go`/`github.go`/`gitlab.go`/`linear.go`) to
  **serve** comments and attachment metadata + binary content (today comments are
  write-only POST endpoints and there are no attachment content fixtures).
- Per-provider `GetIssueEvidence` tests via inline `httptest` servers (the
  established pattern in `internal/tracker/tracker_test.go`), each asserting that
  provider's real capability profile.

## Changes

1. **`internal/tracker/tracker.go`** — add `Comment`, `Attachment`,
   `IssueEvidence`, `EvidenceCapabilities`, `EvidenceOptions` types; add
   `GetIssueEvidence(ctx, issueID, opts)` to the `Tracker` interface.
2. **`internal/tracker/jira.go`** — request `comment`+`attachment` fields;
   implement `GetIssueEvidence` with native attachment URLs; declare
   `NativeAttachments`.
3. **`internal/tracker/github.go`** — fetch issue comments; extract markdown
   attachment URLs; implement `GetIssueEvidence`; declare `MarkdownAttachments`.
4. **`internal/tracker/gitlab.go`** — fetch notes; extract `/uploads` markdown
   links; implement `GetIssueEvidence`; declare `MarkdownAttachments`.
5. **`internal/tracker/linear.go`** — extend GraphQL with comments + attachments;
   implement `GetIssueEvidence`; declare `NativeAttachments`.
6. **`internal/tracker/evidence.go`** (new) — shared helpers: MIME allowlist,
   markdown attachment-URL extraction, bounded streaming `fetchAttachment`, and
   the free-text secret/token redactor mirroring the `Secret` pattern.
7. **`internal/cli/evidence.go`** (new, copy `pull.go` shape) — `hero sync
   evidence` command, `newTrackerForEvidence` hook, byte-cap enforcement,
   download to gitignored `evidence/attachments/`, redaction, and writing the
   `## Source Evidence` section + `evidence/evidence.json` +
   `evidence/source-issue.md`. Append `evidence/attachments/` to `.hero/.gitignore`.
8. **`internal/cli/sync.go`** — register the new subcommand in `init()`.
9. **`internal/serve/mcp_tools_def.go`** + **`internal/serve/mcp_dispatch.go`** +
   **`internal/serve/mcp_tools.go`** — add the read-only `hero_tracker_evidence`
   tool (definition, read-group handler mapping, handler).
10. **`domains/engineering/agents/debug-investigator.md`** — add the Evidence
    Preflight step; rewrite the lines 64-68 hand-wave into a concrete, honest
    consume-evidence step.
11. **`domains/engineering/skills/debugging-investigation/SKILL.md`** — add the
    `## Source Evidence` report section.
12. **`domains/engineering/commands/diagnose.md`** — one line noting evidence
    preflight is part of the delegated workflow.
13. **`internal/mocktracker/`** — extend seed schema + four provider routers to
    serve comments and attachment content; add fixtures.
14. **Tests** — see Validation.

## Boundaries

- **No tracker writeback of any kind.** This feature only reads. The outbound
  diagnosis attach/comment path is the sibling bug
  `tracker-backed-diagnosis-publication-contract-broken`.
- **No hero-code / Swift UI work.** hero-code consumes this contract; it is not
  modified here.
- **No committing arbitrary large binaries into git.** Downloaded attachment
  bytes live under a gitignored path; only bounded metadata + small redacted text
  are committed.
- Do not add caching/incremental-sync, an evidence diff viewer, OCR, or video
  transcoding. Screenshots are inspected as-is; videos are metadata-only.
- Do not reshape the existing `Issue` model or the import/pull field-sync paths.
- Do not build a general secret-scanning subsystem — a scoped token/secret
  redactor for evidence text is sufficient.

## Risks

- **Provider markdown extraction is heuristic.** GitHub/GitLab attachment
  discovery parses markdown; some links will be missed or misclassified. Mitigate
  with a conservative allowlist of known host patterns and record extraction as
  best-effort in the capability descriptor — never claim a complete attachment set
  for markdown-only providers.
- **Redaction is best-effort.** A novel token format could slip through. Keep the
  redactor pattern list current and always redact the exact configured secret;
  label persisted text as redacted-best-effort, not guaranteed-clean.
- **Untrusted content / prompt injection.** Comments and attachments are
  attacker-influenceable. The investigator must treat them as data, not
  instructions; the MCP envelope and agent step both state this.
- **Byte caps can hide the decisive screenshot.** If the one screenshot that
  matters exceeds the cap, it is skipped. Surface skipped attachments prominently
  in the Source Evidence section with their size so a human can raise the cap and
  re-run, rather than silently dropping them.
- **Auth scope.** A token valid for status sync may lack comment/attachment read
  scope; treat 403 on the evidence expansion as an evidence gap, not a hard
  failure of the whole diagnosis.

## Validation

### New tests

1. **Per-provider `GetIssueEvidence` (offline, httptest):** Jira (native
   comments+attachments), GitHub (comments + markdown-extracted attachments),
   GitLab (notes + `/uploads` links), Linear (GraphQL comments+attachments). Each
   asserts the returned `EvidenceCapabilities` matches the provider profile.
2. **Byte-cap / truncation:** an over-`MaxAttachmentBytes` attachment is
   metadata-only with `SkipReason:"over-size"`; a set exceeding `MaxTotalBytes`
   stops downloading and records the remainder as skipped.
3. **MIME allowlist:** a `.png` is downloaded+inspected; a `.zip`/executable is
   metadata-only with `SkipReason:"mime-not-inspectable"`.
4. **Redaction:** description/comment text containing the configured secret and a
   `ghp_`/`glpat-`-shaped token is scrubbed before persistence; `Redacted:true`.
5. **Never-writes-tracker:** driving evidence retrieval against `internal/mocktracker`
   asserts zero POST/PUT/PATCH/DELETE were received.
6. **Offline:** `newTrackerForEvidence` returns a transport error → CLI exits
   non-zero, evidence marked `unavailable: offline`; a workflow-level assertion
   that the gap is recorded, not papered over.
7. **Auth failure:** 401/403 → `FieldError` path → exit 2, evidence marked
   `unavailable: auth`.
8. **Unsupported capability:** a markdown-only provider reports
   `NativeAttachments:false` and the evidence explicitly flags the attachment set
   as best-effort rather than silently omitting.
9. **Provenance/freshness:** the written `## Source Evidence` section and
   `evidence.json` carry provider, issue URL, integration ID, and a valid
   RFC3339 `retrieved_at`.
10. **Gitignore policy:** after a run, `evidence/attachments/` is gitignored while
    `evidence/evidence.json` and `evidence/source-issue.md` are tracked.
11. **CLI read-only-ness:** `hero sync evidence --json` emits a redacted envelope
    and makes no mutating call (shared with test 5).
12. **Installed-content parity:** the evidence-preflight step is present in the
    generated `debug-investigator` across all install targets (reuse the existing
    six-target propagation check).

### Regression scope

- Existing `hero sync pull` / `sync import` field paths (must be untouched).
- Jira/GitHub/GitLab/Linear read paths and the mock-tracker routers.
- `/diagnose` batch mode and the non-tracker (no `tracker_id`) path — which must
  skip evidence retrieval cleanly and report `not_applicable`.

## Acceptance Criteria

- **AC-1:** WHEN `/diagnose` runs on a spec with a `tracker_id` and a ready integration THE SYSTEM SHALL fetch the latest issue description, comments, and attachment metadata before any root-cause classification.
- **AC-2:** WHEN source evidence is fetched THE SYSTEM SHALL record provenance (provider, issue ID, source URL, integration ID) and retrieval freshness (`retrieved_at` in RFC3339 UTC) in the local spec's `## Source Evidence` section.
- **AC-3:** WHEN an attachment is an inspectable MIME type within the per-attachment byte cap THE SYSTEM SHALL download its content to the spec's gitignored evidence directory and record its filename, MIME, size, and sha256 in the manifest.
- **AC-4:** IF an attachment exceeds the per-attachment or total byte cap or is not an inspectable MIME type THEN THE SYSTEM SHALL record it as metadata-only with a skip reason and SHALL NOT download its content.
- **AC-5:** THE SYSTEM SHALL treat evidence retrieval as read-only and SHALL NOT issue any create, update, or delete request to the tracker.
- **AC-6:** WHEN persisted evidence text contains the configured integration secret or a token-shaped string THE SYSTEM SHALL redact it before writing to disk or output and SHALL record that redaction occurred.
- **AC-7:** IF the tracker is unreachable THEN THE SYSTEM SHALL mark evidence `unavailable: offline`, exit the CLI non-zero, and the workflow SHALL proceed while explicitly recording the evidence gap rather than claiming full evidence.
- **AC-8:** IF tracker authentication fails THEN THE SYSTEM SHALL classify it via the existing `FieldError` path, mark evidence `unavailable: auth`, and SHALL NOT claim evidence was gathered.
- **AC-9:** WHERE a provider does not support a given evidence kind THE SYSTEM SHALL return the supported subset with a capability descriptor and SHALL flag the unsupported kind explicitly rather than silently omitting it.
- **AC-10:** WHEN `debug-investigator` begins root-cause classification THE SYSTEM SHALL have consumed the `## Source Evidence` section, or its recorded gap, as an input.
- **AC-11:** THE SYSTEM SHALL preserve the local spec as the durable working record — the `## Source Evidence` section, `evidence/evidence.json`, and redacted `evidence/source-issue.md` are committed while downloaded binary attachments remain gitignored.
- **AC-12:** WHEN evidence retrieval runs for Jira, GitHub, GitLab, or Linear THE SYSTEM SHALL be covered by a deterministic offline test exercising that provider's comment and attachment capability profile.
- **AC-13:** THE SYSTEM SHALL bound each evidence retrieval by configurable per-attachment and total byte limits with documented defaults.
- **AC-14:** THE SYSTEM SHALL expose evidence retrieval as a read-only `hero sync evidence` CLI command and a read-group `hero_tracker_evidence` MCP tool, neither of which mutates the tracker.
- **AC-15:** WHEN a bug spec has no `tracker_id` THE SYSTEM SHALL skip evidence retrieval and report it as `not_applicable` without any tracker request.

## Notes

- Designed via `hero peer call hero-code → hero` (spec-out, call
  `18c3cd7ed53190689847583644c72379`). No tracker was contacted during design.
- This is the inbound complement to
  `tracker-backed-diagnosis-publication-contract-broken` (outbound postback).
  The two share the `debug-investigator` workflow and the `internal/tracker`
  adapters; coordinate the diagnose-renderer wording so both the evidence
  preflight and the postback close land in the same shortcut-surface edits.
- `hero sync pull --field` (`internal/cli/pull.go`) is the read-only precedent to
  copy — including its injectable adapter hook and auth-exit hook — so the new
  command inherits the established offline-testable shape.
