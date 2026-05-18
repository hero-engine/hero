# Import

`hero import` ingests external content — a URL, a single file, or a whole
directory — into the knowledge base. Each ingest writes a raw copy under
`.hero/knowledge/raw/<slug>.md` plus a stub knowledge entry under
`.hero/knowledge/<type>/<slug>/spec.md` for an agent to enrich later.

Use it to seed the corpus with vendor docs, standards repos, API
references, architecture writeups, or any prose the project depends on
but doesn't author itself.

## URL input

```bash
hero import https://docs.example.com/api-reference "API Reference"
```

The HTTP body is fetched and stored verbatim. If you omit the title
argument and the response is `text/html`, the page's `<title>` tag is
extracted and used as the title; otherwise the URL itself becomes the
title.

## Single-file input

```bash
hero import ./docs/architecture.md "Architecture Overview"
```

If you omit the title argument, the filename basename is used as the
title.

## Directory input

```bash
hero import ./vendor-docs/api-standards/ "API Standards"
```

The walker recurses through the directory and ingests every text-ish
file as its own knowledge entry. Rules the walk applies:

- **Per-file slug**: `<title-slug>-<filename-slug>`, where the filename
  slug derives from the file's path relative to the import root with
  the extension stripped. Re-running the same import is idempotent.
- **Group tag**: every entry from one directory import is tagged with
  the `<title-slug>`. `hero search --tag api-standards` then retrieves
  the whole import group.
- **Extension allowlist**: `.md`, `.markdown`, `.txt`, `.rst`, `.adoc`,
  `.mdx`, `.org`, `.json`, `.yaml`, `.yml`. Files with any other
  extension are silently skipped.
- **Hidden filter**: any file or directory whose name starts with `.`
  is skipped, so `.git`, `.obsidian`, and dotfiles are excluded
  automatically.
- **Skip-if-exists is per file**: re-running the same import after an
  agent has enriched one stub does not clobber that enrichment. New
  files in the tree are still picked up.
- **Size cap**: files larger than `--max-bytes` are skipped with a
  per-file note (`Skipping <path> (N bytes > --max-bytes M)`).
- **Summary line**: on completion the command prints
  `Ingested N files (M skipped as already-present) from <dir> under tag "<group-tag>"`.

If the walk finds zero ingestable files (everything filtered out by
extension or size), the command exits with an error pointing at the
filters.

## Flags

| Flag | Default | Description |
|---|---|---|
| `--tag <string>` | `""` | tag for the knowledge entry (added on top of the auto `ingested` tag and, for directory imports, the group tag) |
| `--type <string>` | `context` | knowledge type — `context`, `convention`, or `decision` |
| `--max-bytes <uint>` | `1048576` | skip files larger than this many bytes during directory ingest (1 MiB default) |

`--max-bytes` only affects the directory walk. Single-file and URL
ingests are not size-capped.

## What gets created

- `.hero/knowledge/raw/<slug>.md` — the raw copy, with a small
  frontmatter block recording `source`, `ingested` timestamp, and
  `title`.
- `.hero/knowledge/<type>/<slug>/spec.md` — the stub knowledge entry,
  pre-tagged `ingested` plus any `--tag` and group tag. The body is a
  placeholder for the agent to fill in.

## Next steps

After ingesting, run [`hero index`](search-and-context.md#index-and-search)
so the new entries become searchable. See the
[Knowledge Base concepts page](../concepts/knowledge-base.md) for how
the enrichment workflow turns stubs into useful entries.
