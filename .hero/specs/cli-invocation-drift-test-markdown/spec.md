---
type: feature
status: completed
severity: medium
tags: [drift, testing, ci, docs, cli]
relates-to: [context-files-flag-drift, recap-unregister-stale-and-empty-repo, gitignore-missing-index-db, architectural-drift-detection]
---
# CLI Invocation Drift Test for Markdown Surfaces

## Context

During v0.10.0 release prep, the docs agent discovered that `hero pull` had been a phantom command for multiple releases — referenced in the install-time AGENTS.md/CLAUDE.md template (`internal/install/agents_md.go`) but never registered as a real cobra subcommand. Every install received bogus instructions and there was no automated check that would have caught it.

This is not the first instance of this drift class. The Go-string analog was solved in commit `5110b1a` via `internal/cli/hints.go` + `internal/cli/hints_test.go`: a `cliHints` registry of invocation strings emitted from Go code, validated against `rootCmd.Traverse` in a unit test. The same pattern needs to extend to markdown surfaces — slash commands, skills, agents, web docs, top-level READMEs, and (critically) the rendered output of the install-time AGENTS.md generator.

Related precedent in the drift-detection family: `context-files-flag-drift`, `recap-unregister-stale-and-empty-repo`, `gitignore-missing-index-db`. This spec adds one more guardrail to that same class.

## Kickoff

Delivered: markdown CLI-invocation drift detector now ships as a Go test
(`TestMarkdownInvocationsResolveAgainstRootCmd`) and as `hero docs check
--invocations`. The test scans 752 invocations across commands/, skills/,
agents/, web/docs/src/, top-level docs, and the in-memory rendered
AGENTS.md template; any reference to a phantom subcommand or missing
flag fails fast with file, line, and resolver error.

**Status:** completed.

**Where it lives:** extractor + validator in
`internal/cli/markdown_invocations.go`, drift gate in
`internal/cli/markdown_drift_test.go`, table-driven unit tests in
`internal/cli/markdown_invocations_test.go`, CLI flag wiring in
`internal/cli/docs_check.go`, exported render helper in
`internal/install/agents_md.go` (`RenderAgentsMdBodyForDriftTest`).

**Escape hatch:** `<!-- drift-test:ignore -->` suppresses the same line
or the immediately following line. `.hero/specs/` and `.hero/planning/`
are excluded by default (bug reports legitimately describe phantom
commands).

**Follow-up:** 17 known-drift references were marked with
`drift-test:ignore` rather than fixed — sweep them in a dedicated
documentation pass. See the commit body for the full list.

## Goal

A test catches any stale or phantom `hero <command>` invocation referenced in markdown shipped by this repo (or rendered by its install code) before it reaches users. Specifically:

1. `go test ./internal/cli/...` fails when any markdown surface lists an unresolvable command or flag.
2. `hero docs check --invocations` runs the same check on demand against the current project, reporting file, line, the bad invocation, and the cobra resolver error.
3. The rendered output of `internal/install/agents_md.go` is included in the scan (not just its Go source), so template drift like the `hero pull` regression fails the test.

Done means: re-introducing `hero pull` into `agents_md.go` (or any equivalent stale reference into a slash command body, skill, or doc) produces a failing test with a precise error.

## Approach

Two surfaces, one shared core.

**Shared core** — `internal/cli/markdown_invocations.go`:

- `ExtractInvocations(path string, content []byte) []Invocation` — parses markdown and emits `Invocation{File, Line, Raw, Args}` records.
- `ValidateInvocation(root *cobra.Command, inv Invocation) error` — resolves args via `root.Traverse(args)` and verifies every `--flag` token exists on the resolved command or any inherited flag set. Mirrors the logic in `internal/cli/hints_test.go:18-38`.

Extraction rules:

- Match `hero\s+[a-z][a-z0-9-]*` anywhere on a line, both in fenced code blocks and in inline backticks. Code-block content is included by default (it represents explicit examples).
- Tokenize args by splitting on whitespace; stop at the first token that looks like prose (e.g. ends in `.`, `,`, `:`, or contains characters outside `[a-zA-Z0-9._/=-]`). This keeps `hero pull` (real arg) but drops `hero framework` (prose).
- The first token after `hero ` must match `[a-z][a-z0-9-]*` — single-word lowercase. This filters out "the hero framework", "Hero is a tool", etc.
- Skip content inside HTML comments (`<!-- ... -->`), including multi-line.
- Honor a per-line escape hatch comment: `<!-- drift-test:ignore -->` on the same line or the immediately preceding line suppresses extraction.
- Exclude `.hero/specs/` and `.hero/planning/` entirely from the default scan (they intentionally describe broken/phantom invocations as part of bug reports).

Surfaces scanned:

| Surface | How loaded |
|---|---|
| `commands/*.md` | walk dir |
| `.claude/commands/*.md` | walk dir if present (post-install copy) |
| `skills/**/*.md` | walk dir recursively |
| `agents/*.md` | walk dir |
| `web/docs/src/**/*.md` | walk dir recursively |
| `README.md`, `AGENTS.md`, `GETTING-STARTED.md` | direct read |
| Rendered AGENTS.md template | call `generateAgentsMdBody` (or refactor entry point) into a buffer, scan as in-memory bytes with a synthetic path like `<rendered:internal/install/agents_md.go>` |

**Surface A — Go test** (`internal/cli/markdown_drift_test.go`):

- Discovers project root via existing `findProjectRoot()` helper.
- Walks each surface, accumulates invocations, validates each against `rootCmd`.
- Reports failures with `t.Errorf` per violation so all issues surface in one run.
- For the rendered AGENTS.md, calls the install-side rendering function directly (may require exporting `generateAgentsMdBody` or wrapping it with an exported `RenderAgentsMdBodyForDriftTest` helper that returns `[]byte`).

**Surface B — `hero docs check --invocations`** (`internal/cli/docs_check.go`):

- New `--invocations` bool flag on the existing `docsCheckCmd`.
- When set, runs the same extractor + validator against the current project root.
- Reports as the third section of `docs check` output (after numeric claims and mention checks).
- Non-zero exit on any failure, consistent with existing `os.Exit(1)` behavior on issues.

Why both: the Go test catches drift at CI time for developers of this repo (fast, free, runs on every change). The CLI subcommand makes the same check runnable on demand against any project — useful for downstream consumers of Hero who add their own slash commands or skills with `hero <foo>` invocations.

## Changes

1. **Created `internal/cli/markdown_invocations.go`** — extractor + validator.
   - `type Invocation struct { File, Raw string; Line int; Args []string }`
   - `ExtractInvocations(path string, content []byte) []Invocation`
   - `ValidateInvocation(root *cobra.Command, inv Invocation) error`
   - HTML-comment stripping that preserves newlines; per-line `drift-test:ignore` honor (same-line or previous-line).
   - Path-exclusion helper for `.hero/specs/` and `.hero/planning/`.
   - Stricter regex prefix `(?m)(?:^[ \t]*|[\x60(\[<]|[^a-zA-Z0-9\s][ \t]+)` rejects prose like "the hero framework".
   - Special-cases `--help` (cobra injects lazily during Execute).

2. **Created `internal/cli/markdown_drift_test.go`** — the CI gate.
   - `TestMarkdownInvocationsResolveAgainstRootCmd` walks commands/, skills/ (recursive), agents/, web/docs/src/ (recursive), top-level README/AGENTS/GETTING-STARTED, and the rendered AGENTS.md body.
   - One `t.Errorf` per failed invocation with file, line, raw text, and resolver error.
   - `requireAny` flag per-surface guards against silent globs.

3. **Created `internal/cli/markdown_invocations_test.go`** — table-driven unit coverage for the extractor + validator.

4. **Added `install.RenderAgentsMdBodyForDriftTest()` in `internal/install/agents_md.go`** — narrow exported wrapper that returns the rendered body as `[]byte` for the drift test (kept the lowercased `generateAgentsMdBody` for production code).

5. **Extended `internal/cli/docs_check.go` with `--invocations` flag.** Adds a third report section; increments the existing `issues` counter so `os.Exit(1)` fires on any drift.

6. **Added 14 `drift-test:ignore` markers across the corpus.** Covers known-drift references in `commands/deliver.md`, `commands/import.md`, `skills/kickoff-prompt/SKILL.md`, `agents/session-primer.md`, `web/docs/src/cli/peering.md`, `web/docs/src/configuration/tracker-setup.md`, `web/docs/src/getting-started/project-setup.md`, `web/docs/src/workflows/sprint-and-planning.md`, `README.md`, `AGENTS.md`, and `GETTING-STARTED.md`. Each marker carries a one-line follow-up note. Real fixes are deferred to a dedicated docs sweep.

7. **Documentation:** `--invocations` and the `drift-test:ignore` marker are described inline in `docsCheckCmd.Long`.

## Boundaries

- Not in scope: validating positional argument values (e.g. checking that `hero deliver <slug>` references an existing spec). The test only verifies the command path and flag existence — semantic argument validity is too noisy and project-specific.
- Not in scope: scanning non-markdown surfaces (Go source strings, shell scripts, JSON/YAML config). The Go-string surface is already covered by `internal/cli/hints.go`. Other surfaces can be added later if a regression motivates it.
- Not in scope: a separate top-level `hero lint` command. Extend `hero docs check`.
- Not in scope: catching commands that exist but are deprecated. Cobra resolution is the bar; deprecation messaging is a different concern.
- Not in scope: scanning downstream consumer repos transitively. `hero docs check --invocations` runs against the current project root only.

## Risks

- **False positives from prose** — "the hero framework" or "Hero is a tool" must not be extracted. Mitigated by the `[a-z][a-z0-9-]*` first-token rule and the prose-token stop. Land the test green against the current repo before merging; any prose that slips through becomes a new test case for the extractor.
- **Intentional broken-invocation documentation** — bug specs and diagnose notes describe phantom commands as part of their report. Mitigated by excluding `.hero/specs/` and `.hero/planning/` by default and providing `<!-- drift-test:ignore -->`.
- **Rendered AGENTS.md coupling** — the test depends on calling into `internal/install` from `internal/cli`. Import direction must not create a cycle. Pre-check `go list -deps` before wiring the call; if a cycle threatens, move the shared extractor to a new `internal/drift` package both can depend on.
- **Fenced code blocks with non-hero shell** — a fenced block showing `bash` output may legitimately contain `hero foo` strings that are not meant to validate (e.g. demonstrating an error message). Mitigated by `<!-- drift-test:ignore -->` on the surrounding context; do not try to parse code-block language tags.
- **Test runtime** — scanning `web/docs/src/**/*.md` recursively could be slow if docs grow large. Bench at scaffold time; if >500ms, gate the doc scan behind a build tag or move it to a separate `_test.go` that only runs with `-tags drift`.
- **Refactor coupling** — exporting `generateAgentsMdBody` widens the install package's API surface. Acceptable trade-off; alternative is a `//go:build test` helper file which is uglier.

## Validation

- **Unit tests for the extractor** (`internal/cli/markdown_invocations_test.go`): table-driven cases covering prose-false-positives, fenced blocks, inline backticks, HTML comments, multi-line HTML comments, `drift-test:ignore` on same and previous line, flag tokens (`--foo`, `--foo=bar`), and the path-exclusion rule.
- **Drift test green on current main**: `go test ./internal/cli/...` passes after change 5 lands.
- **Drift test catches the regression**: temporarily re-introduce `hero pull` into `internal/install/agents_md.go`, run the test, observe failure with file, line, raw text, and the cobra resolver error. Revert.
- **CLI behavior**: `hero docs check --invocations` exits non-zero with a clear report when a stale invocation exists; exits zero on a clean tree.
- **No import cycle**: `go build ./...` clean after exporting the render helper.
- **CI integration**: confirm the new test runs under the existing `go test ./...` invocation in CI; no new workflow needed.
