---
title: Document `hero import` — URL, file, and directory ingest
type: feature
status: planning
severity: low
tags: [docs, cli, import, knowledge]
created: 2026-05-18
relates-to: [hero-import-directory-unsupported]
---

# Document `hero import` — URL, file, and directory ingest

## Kickoff

Adds a dedicated CLI Reference page for `hero import` so users can discover
the URL / single-file / directory input shapes plus the `--max-bytes`,
`--tag`, and `--type` flags without grepping the binary.

**Status:** planning — directory ingest shipped in v0.10 via
`hero-import-directory-unsupported`; docs never caught up.

**Pick up at:** scaffold `web/docs/src/cli/import.md` from the
`internal/cli/import.go` source. Cover the three input shapes (URL, file,
directory), each flag, and one worked directory example. Then wire it into
`mkdocs.yml` and add a row to `cli/overview.md`.

→ `.hero/planning/features/hero-import-docs-page/spec.md`

**Files:** `internal/cli/import.go`, `internal/cli/import_test.go`,
`.hero/specs/hero-import-directory-unsupported/spec.md`,
`web/docs/mkdocs.yml`, `web/docs/src/cli/overview.md`.
**Skip:** documenting the underlying knowledge-entry stub format — that's
covered in the existing Knowledge Base concepts page.

## Context

`hero import` now accepts three input shapes — URL, single file, and
directory — after the v0.10 fix landed via
`.hero/specs/hero-import-directory-unsupported/spec.md`. The directory
branch walks the tree, ingests each text-ish file as its own knowledge
entry with slug `<title-slug>-<filename-slug>`, applies a shared group
tag, honors a `--max-bytes` cap, and skips files whose stub entry already
exists.

None of this is documented under `web/docs/src/`. The only public
description lives in the binary's `--help` output. A new user who wants
to ingest a vendor docs tree or a standards repo has no doc page to land
on — they have to read source or `hero import --help`. The CLI overview
table lists no `import` row at all (it's owned by no group on
`web/docs/src/cli/overview.md`).

This is the smallest possible docs change that closes that gap.

## Goal

A new `web/docs/src/cli/import.md` page exists, is reachable from the
CLI Reference nav and the CLI overview table, and covers — accurately
against `internal/cli/import.go` — the three input shapes, every flag,
and the directory-mode specifics (slug scheme, group tag, skip-if-exists,
extension allowlist, hidden-file/dir filter, size cap behavior). A
reader can paste any example and have it work against a real project.

## Approach

Treat this as a reference-style page: short prose, lots of concrete
examples, every flag named. Mirror the structure of
`web/docs/src/cli/search-and-context.md` (intro paragraph, per-command
sections, runnable code blocks).

Verify every fact against `internal/cli/import.go` before writing it
down. Don't restate behavior that already lives in the knowledge-base
concept docs (raw file layout, enrichment workflow); link out instead.

The page is structured around the three input shapes because that is
the user's mental model when they type the command — they have either a
URL, a file, or a directory in hand.

## Changes

1. **New file `web/docs/src/cli/import.md`.** Sections:
   - Intro paragraph — what `hero import` does (writes a raw copy under
     `.hero/knowledge/raw/` plus a stub knowledge entry for agent
     enrichment) and when to use it.
   - **URL input** — single example
     (`hero import https://docs.example.com/api-reference "API Reference"`)
     and a note that the HTML `<title>` is auto-extracted when no title
     argument is given.
   - **Single-file input** — single example
     (`hero import ./docs/architecture.md "Architecture Overview"`) and a
     note that the filename basename is used when no title is given.
   - **Directory input** — single worked example
     (`hero import ./vendor-docs/api-standards/ "API Standards"`) followed
     by the rules the walker applies:
     - Per-file slug: `<title-slug>-<filename-slug>` (filename slug derives
       from the relative path under the directory, extension stripped).
     - Group tag: every entry from one directory import is tagged with the
       `<title-slug>` so `hero search --tag <title-slug>` retrieves the
       whole import group.
     - Extension allowlist (verbatim from `isTextExt` in `import.go`):
       `.md`, `.markdown`, `.txt`, `.rst`, `.adoc`, `.mdx`, `.org`,
       `.json`, `.yaml`, `.yml`. Files with other extensions are silently
       skipped.
     - Hidden-file and hidden-directory filter: anything whose name starts
       with `.` is skipped (so `.git`, `.obsidian`, dotfiles are excluded).
     - Skip-if-exists is **per file**, so re-running the same import after
       an agent has enriched one stub does not clobber that enrichment.
     - Files exceeding `--max-bytes` are skipped with a per-file note.
     - Output: a summary line of the form
       `Ingested N files (M skipped as already-present) from <dir> under tag <group-tag>`.
   - **Flags** — table with three rows:
     - `--max-bytes <uint>` — per-file size cap, default `1048576` (1 MiB).
       Applies to directory walk only.
     - `--tag <string>` — user-supplied tag added to every entry created
       by the invocation (deduped against the auto `ingested` and group
       tags).
     - `--type <string>` — knowledge entry type, default `context`.
       Accepts `context`, `convention`, `decision` (call out the values
       the help text advertises).
   - **What gets created** — short bullet list pointing at
     `.hero/knowledge/raw/<slug>.md` (the raw copy with frontmatter) and
     `.hero/knowledge/<type>/<slug>/spec.md` (the stub the agent enriches).
   - **Next steps** — one-line pointer to `hero index` (to make new entries
     searchable) and to the Knowledge Base concept page for enrichment.

2. **Update `web/docs/mkdocs.yml`.** Insert `- Import: cli/import.md`
   under the `CLI Reference` nav block, placed after
   `Search & Context` (it's the most natural neighbor — both touch the
   knowledge corpus).

3. **Update `web/docs/src/cli/overview.md`.** Add a new row to the
   Command Groups table pointing at `import.md`, e.g.
   `| [Import](import.md) | Ingest external content (URL, file, directory) into the knowledge base | \`import\` |`.

## Boundaries

- Not documenting the raw-file frontmatter shape or the stub knowledge
  entry template — those are concerns of the Knowledge Base concept doc,
  not the CLI reference.
- Not documenting `hero enrich`, `hero ask`, or the enrichment workflow.
  Link out to existing pages.
- Not adding new flags or behavior — pure documentation of what shipped.
- Not changing `internal/cli/import.go` or its `--help` text.
- Not building a broader "every CLI command has a dedicated page" pass.

## Risks

- **Source drift.** Future flag changes in `internal/cli/import.go` will
  silently desync the page. The follow-up `cli-examples-smoke-test` work
  mentioned in `.hero/specs/hero-import-directory-unsupported/spec.md`
  would catch the example-block case but not the flag table. Lowest-cost
  mitigation here: keep flag descriptions terse and copy them verbatim
  from the cobra `Flags()` registrations so a future grep finds both.
- **Extension allowlist staleness.** If `isTextExt` gains or loses an
  extension, the page is wrong. Mitigation: write the list as a single
  inline `code span` of comma-separated extensions so it's easy to spot
  and update in one place.

## Validation

- `mkdocs serve` from `web/docs/` renders the new page without errors and
  the nav shows `CLI Reference → Import`.
- Every code block on the page either is a literal command or has been
  pasted and run against a temp project successfully.
- The flag table matches the output of `hero import --help` byte-for-byte
  on flag names, defaults, and one-line descriptions.
- The extension allowlist on the page matches `isTextExt` in
  `internal/cli/import.go` exactly.
- The directory worked example produces the documented summary line shape
  when actually run.

## Related concerns

Documentation gaps spotted while scoping this spec — flagged but
**explicitly out of scope** for this work:

- `hero context imports` is listed in the CLI overview's Cross-Repo Peering
  row but has no dedicated section on `web/docs/src/cli/peering.md` — it
  appears in the row's command list only.
- `hero snapshot` shipped in v0.10 (commit `5110b1a`) and is named in
  the Search & Context group on the overview page, but its coverage in
  `web/docs/src/cli/search-and-context.md` should be audited to confirm
  it explains the project snapshot projector, the `/project` home, and
  the archive containment behavior.

Each is its own follow-up spec if confirmed as a real gap.
