# Delivery audit — spec-title-keeps-yaml-quotes

**Audited:** `git diff origin/main...021bc39c` (excluding `.hero/hero.json`)
**Verdict:** SHIP
**Surface:** clean

## Acceptance criteria
- [✓] AC-1: quoted YAML titles decoded for every consumer — `internal/spec/spec.go:514` (`s.Title = unquoteYAMLScalar(val)` in `parseFrontmatter`), helper at `internal/spec/spec.go:1948-1960` decodes via yaml.v3. `TestParseTitleUnquotesYAMLScalars` (`internal/spec/title_unquote_test.go`) asserts double quotes with `: `, single quotes with `''`, escaped `\"`, and `''` inside double quotes. Re-run by the auditor: PASS. Real CLI exercise: `real-exercise.log` shows `graph-unpartitioned-writers-duplicate-nodes`, whose frontmatter title is double-quoted, listed without quotes.
- [✓] AC-2: unquoted or undecodable titles unchanged — same test, cases `Plain title with "inner" quotes` and `"unterminated` (yaml error falls through to the raw value).

## Changes
- [✓] `internal/spec/spec.go`: `unquoteYAMLScalar` applied to `title` only — present in diff; no other frontmatter key changed.
- [✓] `internal/spec/title_unquote_test.go`: table test — new file, 6 cases.

## Open items
- None.

## Audit notes
- Ledger claim "no writer re-serializes a parsed title" spot-checked: `title:` writers in `internal/cli` (`new.go`, `note.go`, `import.go`, `sync_import.go`, `sprint.go`) write user or tracker input, not a parsed `spec.Spec.Title`; `handoff.go:240` is display output only.
- Evidence: `go-tests.log` (full `go test ./...`, 0 FAIL lines), `vet.log` empty. The auditor re-ran `go test ./internal/spec ./internal/cli -count=1` (targeted tests) and `go vet` on both packages: pass.
