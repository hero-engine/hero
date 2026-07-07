---
title: "Knowledge Content Retrieval — layout-agnostic ingest + pull (ask/search)"
slug: knowledge-content-retrieval
type: feature
status: planning
priority: P2
size: medium
domain: engineering
created: 2026-07-06
tags: [knowledge, retrieval, ingest, hero-ask, hero-search, all-domains]
relations:
  - { target: knowledge-surfacing, kind: parent }
  - { target: knowledge-context-injection, kind: enables }
  - { target: hero-ask, kind: related }
  - { target: unified-retrieval-layer, kind: related }
  - { target: sales-pack-reality-sync, kind: informed-by }
  - { target: knowledge-retrieved-through-unified-corpus, kind: decided-in }
delivery_method: manual
---

# Knowledge Content Retrieval — layout-agnostic ingest + pull (ask/search)

Phase 1 of the [[knowledge-surfacing]] initiative. Builds the foundation — a
layout-agnostic knowledge ingest into the existing retrieval corpus — and lights
up the **pull** surfaces (`hero ask`, `hero search`). Phase 2
([[knowledge-context-injection]]) rides this foundation for **push**.

## Goal

Make every hand-authored `.hero/knowledge/**/*.md` file retrievable by content
through `hero ask` and `hero search`, **regardless of the file's layout or
whether it carries frontmatter** — without regressing the boundary that keeps
knowledge out of work-spec discovery (`hero list`, `hero queue`, `verify`).

Capture without retrieval is dead weight: `/note`, `knowledge.auto_capture`, and
domain agents all *write* into `.hero/knowledge/`, but nothing reads flat
knowledge back. This phase makes the capture side pay off for pull.

## Kickoff

Whether a captured knowledge entry surfaces is decided by accidental layout, not
intent: `<slug>/spec.md` entries are indexed and found; flat `<name>.md` entries
are not. Verified in this repo: all 23 `decisions/*.md` and 4 `conventions/*.md`
are flat → `hero ask`/`search` return nothing; the tripwire (a spec.md dir) is
fine. Fix: a **layout-agnostic** ingest that walks `.hero/knowledge/**` (except
`raw/`) and indexes every `*.md` — flat, slug/spec, or three-file; typed or
untyped; known or domain-invented subdir — into the existing corpus under
`category=knowledge`, `kind` from the subdir, deduped against what Discover
already indexes for spec.md-shaped knowledge. Expose via `hero ask` + `hero
search --knowledge`. Do NOT loosen `nonWorkFlatTypes`; the `category` marker
keeps knowledge out of work discovery. Start: `internal/index/refresh.go`,
`internal/spec/spec.go:1085` (Discover), `internal/cli/ask.go`,
`internal/cli/search.go`, `internal/knowledge/graph_ingest.go` (sibling ingest).

## Principle — capture is layout-agnostic (inherited from the initiative)

Work specs are nudged toward `<slug>/spec.md` because a spec grows children
(audits, retros, three-file splits). **Knowledge is not.** Flat is fine,
slug/spec is fine, three-file is fine. Following the slug/spec convention is a
*nudge*, never a *precondition for being found*. So this ingest:

- **Keys on location + content, not shape.** Any `*.md` under
  `.hero/knowledge/**` (except `raw/`) is captured — `foo.md`, `foo/spec.md`,
  `foo/requirements.md`, in `decisions/`, `battlecards/`, or a subdir a pack
  invents tomorrow.
- **Is best-effort and lossy-tolerant.** No frontmatter → title from first `# H1`
  or filename. Unknown subdir → `kind` = subdir name. Malformed file → index what
  reads, don't drop it. Bar = "captured and findable," not "well-formed."
- **Never gates retrievability on convention adherence.** Any advisory "consider
  slug/spec so this can hold children" nudge belongs in `hero check --knowledge`,
  not in whether the entry is searchable.

## Problem

Hero stores knowledge in two shapes; only the spec.md one is retrievable.

- **spec.md-shaped** (`<slug>/spec.md`, `type: convention|decision|context|rule`)
  — `spec.Discover` loads `spec.md` type-agnostically, so it lands in `fts_specs`
  and is reachable via `hero ask --type` and `hero search`. This is what the
  completed `hero-ask` was designed and tested against
  (`.hero/conventions/badger-storage/spec.md`).
- **flat-file** (`.hero/knowledge/<subdir>/*.md`) — the *canonical* home that
  `auto_capture`, `/note`, and domain agents write to (sales
  `battlecards/<competitor>.md`, `playbooks/<slug>.md`). Reachable by **no**
  content surface: `spec.Discover` skips flat knowledge-typed and untyped files
  (`nonWorkFlatTypes`, `internal/spec/spec.go:1161-1196`), and the knowledge
  graph ingest covers only `raw/` (`internal/knowledge/graph_ingest.go:31`).

Verified (live binary, this repo's `.hero/`): `hero ask "what is the peer
manifest publish boundary"` → *"No knowledge found"* though
`decisions/peer-manifest-publish-boundary.md` says exactly that; 60 flat files,
0 reachable. The sales pack worked around this with `ls`-based path lookup —
honest but brittle (no ranking, caller must know the exact directory, can't
answer "which battlecard mentions RivalCorp's pricing?").

## Scope

**In scope**
- A **layout-agnostic** knowledge ingest walking `.hero/knowledge/` (excluding
  `raw/`) that indexes each `*.md` — flat, slug/spec, or three-file — into the
  existing corpus with `category=knowledge`, `kind` from the first subdir
  segment, and `title` from frontmatter or first `# H1` (or filename).
- Dedup against the spec corpus: a knowledge entry already indexed as a
  `<slug>/spec.md` by Discover must not double-appear; key knowledge rows by file
  path or content hash.
- Self-healing refresh parity with specs (mirror `RefreshIfStale`): new/changed/
  removed knowledge files reflect on the next read without a manual reindex.
- `hero ask` reaching the flat-knowledge corpus; `--type` accepting knowledge
  `kind`s (known subdirs map to the `type` vocabulary; free-form subdirs match by
  name).
- `hero search --knowledge` (opt-in) ranking knowledge entries, kept out of
  default work-search results.
- Repoint the sales pack's battlecard/playbook lookup from `ls` path lookup to
  the content surface (closes the `sales-pack-reality-sync` AC#6 deviation).

**Out of scope (this phase)**
- Push / file-scoped injection — that is [[knowledge-context-injection]] (P2).
- Semantic/vector retrieval (rides embeddings once knowledge is in the corpus).
- Cold-start knowledge digest in NEXT/QUEUE/SNAPSHOT, `/prime` (follow-on).
- Changing where or in what shape knowledge is authored.
- Loosening `nonWorkFlatTypes` or merging knowledge into work discovery.

## Acceptance Criteria

- WHERE a `.hero/knowledge/**/*.md` file exists in **any** layout (flat,
  `<slug>/spec.md`, or three-file) with content, THE SYSTEM SHALL make it
  retrievable by `hero ask` with a citation whose path is that file — verified
  against a flat decision that returns "No knowledge found" today
  (`peer-manifest-publish-boundary.md`).
- WHERE a knowledge file carries no `type:` frontmatter (e.g. the sales
  battlecard template), THE SYSTEM SHALL still index it, deriving `title` from
  H1/filename and `kind` from the subdir (`battlecards`).
- WHERE a knowledge file sits in a subdir Hero does not predefine, THE SYSTEM
  SHALL still capture it with `kind` = that subdir name (no allow-list of
  subdirs).
- WHEN the same knowledge entry exists once on disk, THE SYSTEM SHALL surface it
  once (no double-index between the knowledge ingest and Discover's spec corpus).
- WHEN `hero search` runs **without** `--knowledge`, THE SYSTEM SHALL NOT include
  `.hero/knowledge/**` knowledge files in results (work-discovery parity).
- WHEN `hero search --knowledge <query>` runs, THE SYSTEM SHALL return matching
  knowledge entries ranked by relevance, with `kind` and path.
- THE SYSTEM SHALL NOT surface any knowledge file in `hero list`, `hero queue`,
  or `verify` child-resolution (regression-guard the sales `battlecard NOT in
  hero list` invariant).
- WHEN a knowledge file is added/edited/removed, THE SYSTEM SHALL reflect it on
  the next read-side query without a manual full reindex.
- WHEN `hero ask --type <kind>` names a knowledge kind present under
  `.hero/knowledge/`, THE SYSTEM SHALL restrict results to that kind.
- THE SYSTEM SHALL apply capture uniformly across domains — a pm playbook, a
  sales battlecard, an engineering decision — with no domain-specific code path.

## Design notes / open questions

- **Where the ingest lives.** (a) extend `spec.Discover` + `internal/index` to
  emit knowledge rows under a category column, or (b) a parallel
  `internal/knowledge` ingest (sibling to the `raw/` graph ingest) writing into
  the same index. Prefer (b) — Discover stays a *work-spec* walker and the
  knowledge boundary stays explicit — but (a) reuses `RefreshIfStale` directly.
  Both satisfy the ACs; delivery lead picks during design-out.
- **`kind` vs `type`.** Recommendation: `kind` = raw subdir name; map known ones
  (`decisions`→decision, `conventions`→convention, `rules`→rule, `context`) onto
  the `type` vocabulary so `--type` keeps working; free-string the rest.
- **`category` marker.** Needs a persisted `category` (work|knowledge)
  discriminator on index rows so default work surfaces filter cleanly. Confirm
  schema can carry it or add a migration.
- **Dedup key.** Key knowledge rows by file path or content hash, not slug alone,
  so a `<slug>/spec.md` convention and a `notes/<slug>.md` never collide.
- **Layout coexistence.** A single knowledge subdir may hold both flat files and
  spec.md dirs (e.g. `notes/` today: 5 flat + 12 dirs). The walk must handle both
  in the same directory without skipping either.

## Validation

- Repro-first: assert the current miss (`hero ask` on a flat decision → no
  result), then the fix (same query → cited result).
- Layout-matrix fixture: flat, `<slug>/spec.md`, and three-file knowledge files —
  plus one untyped and one in an invented subdir — all retrievable via one
  surface.
- Cross-domain fixture: one pm, one sales (untyped battlecard), one engineering.
- Boundary regression: `hero list` / `hero queue` still exclude every knowledge
  file; re-run `sales-pack-reality-sync` AC#7.
- `go test ./...` green; `hero check --knowledge` still passes.
