# Delivery Audit — tripwire-system
**Spec:** `.hero/planning/features/tripwire-system/spec.md`
**Audit date:** 2026-06-09
**Auditor:** cold delivery audit
**Verdict:** SHIP

---

## Audit Method

Cold read of each referenced source file against the 9 ACs in the spec's Completion Ledger. Build and test suite run fresh. No assumptions taken from the ledger — each claim was independently verified in source.

---

## AC Verdicts

### AC-1 — TypeTripwire parsed with triggers/scope/severity from frontmatter
**PASS**

- `TypeTripwire Type = "tripwire"` at `internal/spec/spec.go:25`.
- `Triggers []string` field on `Spec` struct at line 136.
- `case "triggers": s.Triggers = parseList(val)` in `parseFrontmatter` at line 509.
- `typeFromPath` detects `/tripwires/` path at line 818–820.
- `statusFromPath` defaults tripwires to `StatusActive` at line 847–849.
- `IsKnowledge()` includes `TypeTripwire` at line 1133.
- `Scope` and `Severity` fields already existed on `Spec`; both are parsed from frontmatter (lines 499, 464).

### AC-2 — Session start includes all active tripwires in prime context (before conventions/decisions, not gated by includeKnowledge)
**PASS**

`internal/serve/prime.go` lines 76–98: `## Tripwires (Do Not Violate)` section written unconditionally (no `if includeKnowledge` guard) after Active Work but before the `if includeKnowledge && len(conventions) > 0` block that starts at line 101. Only `status = active` entries populate the `tripwires` slice (line 38). Full `constraint` and `instead` section text is rendered inline.

### AC-3 — hero_context with file paths includes scope-matched tripwires in a dedicated section
**PASS**

`internal/index/index.go:745` — `FindTripwiresForFiles()` queries `WHERE s.type = 'tripwire' AND s.status = 'active'` and applies scope-glob matching (including `*` wildcard for global tripwires) against the provided file paths.

Called at `index.go:1527` inside the `ContextBlock` builder, populating `ctx.Tripwires`. Rendered in `internal/serve/mcp_tools.go:1511–1521` as `### Tripwires (do not violate)` before conventions and rules in `formatContextBlockWithVocab`.

### AC-4 — hero_anchor returns mission + all active tripwires with full Constraint/Why/Instead text
**PASS**

`toolAnchor` at `mcp_tools.go:859`: loads mission via `mission.LoadFile`, calls `FindAllTripwires` (which reads full spec file for each result, extracting `constraint`, `why`, and `instead` sections), and renders each tripwire via `formatTripwireBlock` which emits all three fields. CLI equivalent at `internal/cli/anchor.go` via `printTripwire`.

### AC-5 — hero_anchor with context highlights trigger-matched tripwires first
**PASS**

`mcp_tools.go:884–916`: when `context` arg is non-empty, calls `FindTripwiresByTrigger(ctx)`, builds `highlighted` set, renders matched tripwires under `### ⚠ Relevant to your current context` before rendering remaining tripwires under `### All active tripwires`.

### AC-6 — hero_ask/hero_search query matching a trigger prepends TRIPWIRE WARNING block
**PASS**

`tripwireWarning()` at `mcp_tools.go:930–951`: opens index, calls `FindTripwiresByTrigger(query)`, renders `## TRIPWIRE WARNING` block with full tripwire details for each match.

Called in `toolSearch` at line 254 and in `toolAsk` at line 845. Both prepend the warning block before search results / answer text.

### AC-7 — Superseded tripwires excluded from prime context and retrieval
**PASS**

All three index query functions use `WHERE s.type = 'tripwire' AND s.status = 'active'`:
- `FindAllTripwires` (index.go:838)
- `FindTripwiresByTrigger` (index.go:886–887)
- `FindTripwiresForFiles` (index.go:754–755)

Prime context collection at `prime.go:38` also filters `s.Status == spec.StatusActive`. Superseded entries remain in the database but are never returned.

### AC-8 — hero tripwire list and hero tripwire check <text> CLI commands exist
**PASS**

`internal/cli/tripwire.go`:
- `tripwireListCmd` registered with `Use: "list"`, runs `runTripwireList` which calls `FindAllTripwires` and prints slug, severity, title, triggers, and first line of constraint.
- `tripwireCheckCmd` registered with `Use: "check <text>"`, runs `runTripwireCheck` which calls `FindTripwiresByTrigger` and exits with code 1 on match (CI-friendly).

Both registered under `tripwireCmd` parent.

### AC-9 — hero_anchor MCP tool registered with "when to call it" description
**PASS**

`internal/serve/mcp_tools_def.go:162–169`:
```
Name: "hero_anchor"
Description: "Re-anchor on project first principles. Call this BEFORE proposing architectural
  alternatives, when hitting a dead end, or when brainstorming solutions. Returns project mission
  and all active tripwires (forbidden options). Prevents drift from first principles."
```

Matches spec design intent exactly. Tool dispatch wired in `mcp_dispatch.go`.

---

## Build and Test Evidence

```
go build ./...          — clean, no errors
internal/cli            — PASS
internal/index          — PASS
internal/spec           — PASS
internal/serve          — PASS (+ all sub-packages)
```

All 24 test packages in the four required areas pass.

---

## Surface Notes

One minor discrepancy between ledger claim and actual line numbers: the ledger says `toolAsk` call is at line 845 and `toolSearch` at line 254. Independently verified: `toolSearch` tripwireWarning call is at line 254, `toolAsk` at line 845. Matches.

The spec describes a `## Tripwires` section in `hero_context` output but the implementation renders it as `### Tripwires (do not violate)` (H3 inside the larger context block). This is a rendering choice consistent with the surrounding section hierarchy — not a gap.

The spec mentions `scope: ["*"]` as a global tripwire signal. `FindTripwiresForFiles` handles this via an explicit `if glob == "*"` branch (index.go:810–815). Correct.

---

## Verdict

**SHIP.** All 9 ACs pass. Build is clean. All four test packages pass. The implementation is complete and matches the spec's intent. Archive when ready.
