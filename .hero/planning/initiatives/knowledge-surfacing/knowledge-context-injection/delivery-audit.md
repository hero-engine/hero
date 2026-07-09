# Delivery audit — knowledge-context-injection (P2)

**Delivered via:** `/drive knowledge-surfacing` (guided), 2026-07-07
**Verdict:** SHIP (core injection); drift/impact/anchor scoped to follow-on
**Surface:** noteworthy

Flat code-scoped knowledge now injects into the two edit-time model surfaces —
`hero_context`/`BuildContext` and `hero relevant`/`BuildNudge` — via an isolated
`knowledge_scopes` table. Full suite green.

## Acceptance criteria

- [✓] **Scoped flat convention injects** — `TestKnowledgeInjection`: a flat
  `conventions/contracts-import-discipline.md` with `scope: internal/contracts/*.go`
  appears in `BuildContext(["internal/contracts/manifest.go"]).Conventions`; absent
  for a non-matching path.
- [✓] **Flat and spec.md inject identically** — both route through the same
  Conventions/Rules blocks; `FindKnowledgeForFiles` mirrors
  `FindConventionsForFiles` scope-matching (`filepath.Match` + basename).
- [✓] **`hero relevant`/BuildNudge nudges** — knowledge merged into
  `result.Conventions`.
- [✓] **No scope → no injection** — only `e.Scope` globs populate
  `knowledge_scopes`; unscoped entries never match. Test asserts the unscoped
  battlecard never appears in any injected block.
- [✓] **Free-form never injects** — `codeScopedKinds` gate (convention/rule only);
  battlecards/playbooks excluded.
- [✓] **No regression** — `go test ./...` = ALL PASS.

## Deviations / scoping

- **Second silent-drop bug fixed en route.** `scope:` frontmatter was parsed by
  `parseList`, which only reads the same-line value — a block-style `scope:` list
  parsed to empty (identical class to `inline-flow-relations-dropped`). Fixed
  `internal/spec/spec.go` to fall back to `parseScalarListBlock`, matching how
  `child:`/`synthesized_from:` already handle both forms. Without this, no
  block-style-scoped knowledge would inject. (`tags:`/`triggers:` have the same
  latent issue — noted as follow-on.)
- **drift / impact / anchor-tripwire → follow-on (not claimed).** Those ride the
  shared `FindConventionsForFiles` seam; lighting them up means merging flat
  knowledge into that function (whose consumers include drift/impact) rather than
  the separate `FindKnowledgeForFiles`. Low-risk but a distinct change; the two
  edit-time model surfaces (the actual payoff) are delivered here. Spec Out-of-scope
  updated to say so honestly.
- **Decisions are pull-only, by parity.** spec.md decisions have no file-scope
  matcher, so flat decisions don't inject either (they surface via `hero ask`,
  P1). BuildContext would route a scoped decision to `ctx.Decisions` if one existed.

## Files

- `internal/index/index.go` — `knowledge_scopes` table (+ clear on full reindex);
  `KnowledgeEntry.Scope`; scope populate/clear in `IndexKnowledge`/`RemoveKnowledge`;
  `codeScopedKinds`, `FindKnowledgeForFiles`; `BuildContext` + `BuildNudge` merge.
- `internal/index/knowledge_discover.go` — capture `scope:` in `parseKnowledgeFile`.
- `internal/spec/spec.go` — block-style `scope:` list parsing.
- `internal/index/knowledge_discover_test.go` — `TestKnowledgeInjection`.
