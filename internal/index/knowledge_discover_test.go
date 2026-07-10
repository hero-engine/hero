package index

import (
	"os"
	"path/filepath"
	"testing"
)

func writeKnowledge(t *testing.T, heroDir, rel, content string) string {
	t.Helper()
	path := filepath.Join(heroDir, "knowledge", rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// TestDiscoverKnowledge_FlatDecision is the repro: a flat decision file that
// hero ask/search could not reach before knowledge-surfacing.
func TestDiscoverKnowledge_FlatDecision(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeKnowledge(t, heroDir, "decisions/peer-manifest.md", `---
title: Peer Manifest Publish Boundary
type: decision
---
# Peer Manifest Publish Boundary
The manifest is the publish boundary between sibling repos.
`)

	entries, err := DiscoverKnowledge(heroDir)
	if err != nil {
		t.Fatalf("DiscoverKnowledge: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Slug != "decisions/peer-manifest" {
		t.Errorf("slug = %q, want decisions/peer-manifest", e.Slug)
	}
	if e.Kind != "decisions" {
		t.Errorf("kind = %q, want decisions", e.Kind)
	}
	if e.Type != "decision" {
		t.Errorf("type = %q, want decision", e.Type)
	}
	if e.Title != "Peer Manifest Publish Boundary" {
		t.Errorf("title = %q", e.Title)
	}
}

// TestDiscoverKnowledge_UntypedInvented covers the layout-agnostic invariant:
// an untyped file in a subdir Hero does not predefine is still captured, with
// kind from the subdir and title from the H1.
func TestDiscoverKnowledge_UntypedInvented(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeKnowledge(t, heroDir, "battlecards/acme-rival.md",
		"# Battlecard — Hero vs. RivalCorp\nRivalCorp competes on price.\n")

	entries, err := DiscoverKnowledge(heroDir)
	if err != nil {
		t.Fatalf("DiscoverKnowledge: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Kind != "battlecards" {
		t.Errorf("kind = %q, want battlecards", e.Kind)
	}
	if e.Title != "Battlecard — Hero vs. RivalCorp" {
		t.Errorf("title = %q", e.Title)
	}
}

// TestDiscoverKnowledge_SkipsRawAndSpecOwned confirms dedup: raw/ is skipped
// (own ingest) and a spec.md-owned directory's sidecar files are not captured.
func TestDiscoverKnowledge_SkipsRawAndSpecOwned(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeKnowledge(t, heroDir, "raw/source.md", "# Raw\nbytes\n")
	writeKnowledge(t, heroDir, "context/arch/spec.md", `---
title: Arch
type: context
---
# Arch
`)
	writeKnowledge(t, heroDir, "context/arch/delivery-audit.md", "# Audit\nsidecar\n")

	entries, err := DiscoverKnowledge(heroDir)
	if err != nil {
		t.Fatalf("DiscoverKnowledge: %v", err)
	}
	for _, e := range entries {
		if e.Kind == "raw" {
			t.Errorf("raw/ should be skipped, got %q", e.Slug)
		}
		if filepath.Base(e.Path) == "delivery-audit.md" {
			t.Errorf("spec-owned sidecar should be skipped, got %q", e.Slug)
		}
	}
}

// TestSearchKnowledge_ByContentAndKind exercises the FTS query and the
// kind/type filter.
func TestSearchKnowledge_ByContentAndKind(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeKnowledge(t, heroDir, "battlecards/acme.md",
		"# Battlecard\nRivalCorp competes on aggressive pricing.\n")
	writeKnowledge(t, heroDir, "decisions/local-first.md", `---
title: Local First
type: decision
---
# Local First
Peering is local-first.
`)

	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()
	entries, _ := DiscoverKnowledge(heroDir)
	for _, e := range entries {
		if err := idx.IndexKnowledge(e); err != nil {
			t.Fatalf("IndexKnowledge: %v", err)
		}
	}

	hits, err := idx.SearchKnowledge("RivalCorp pricing", nil, 10)
	if err != nil {
		t.Fatalf("SearchKnowledge: %v", err)
	}
	if len(hits) != 1 || hits[0].Kind != "battlecards" {
		t.Fatalf("content search: got %+v", hits)
	}

	// --type decision matches via the type column even though the subdir kind
	// is "decisions".
	byType, _ := idx.SearchKnowledge("local-first peering", []string{"decision"}, 10)
	if len(byType) != 1 || byType[0].Slug != "decisions/local-first" {
		t.Fatalf("kind/type filter: got %+v", byType)
	}
	// --type battlecards matches the subdir kind directly, and excludes the
	// decision.
	byKind, _ := idx.SearchKnowledge("pricing peering", []string{"battlecards"}, 10)
	if len(byKind) != 1 || byKind[0].Kind != "battlecards" {
		t.Fatalf("kind filter: got %+v", byKind)
	}
}

// TestRefreshKnowledge_SelfHeals verifies add/modify/remove parity with specs
// and that knowledge never lands in the specs table.
func TestRefreshKnowledge_SelfHeals(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	path := writeKnowledge(t, heroDir, "notes/idea.md", "# Idea\noriginal body\n")

	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh 1: %v", err)
	}
	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	// Present in knowledge, absent from specs (boundary).
	if hits, _ := idx.SearchKnowledge("original body", nil, 10); len(hits) != 1 {
		t.Fatalf("after refresh, want 1 knowledge hit, got %d", len(hits))
	}
	var specCount int
	idx.db.QueryRow("SELECT COUNT(*) FROM specs").Scan(&specCount)
	if specCount != 0 {
		t.Errorf("knowledge leaked into specs table: %d rows", specCount)
	}

	// Remove the file → orphan cleanup on next refresh.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh 2: %v", err)
	}
	idx2, _ := Open(heroDir)
	defer idx2.Close()
	if hits, _ := idx2.SearchKnowledge("original body", nil, 10); len(hits) != 0 {
		t.Fatalf("orphan not removed, got %d hits", len(hits))
	}
}

// TestKnowledgeInjection covers P2: a flat, scoped convention injects via
// BuildContext on a matching file; an unscoped battlecard never does.
func TestKnowledgeInjection(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeKnowledge(t, heroDir, "conventions/contracts-import-discipline.md", `---
title: Contracts Import Discipline
type: convention
scope:
  - internal/contracts/*.go
---
# Contracts Import Discipline
Never import internal packages across the contracts boundary.
`)
	writeKnowledge(t, heroDir, "battlecards/acme.md",
		"# Battlecard\nRivalCorp competes on price.\n")

	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	// Matching file → the scoped convention injects.
	ctx, err := idx.BuildContext([]string{"internal/contracts/manifest.go"})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	var found bool
	for _, c := range ctx.Conventions {
		if c.Slug == "conventions/contracts-import-discipline" {
			found = true
		}
	}
	if !found {
		t.Errorf("scoped flat convention did not inject for matching file; conventions=%+v", ctx.Conventions)
	}

	// Non-matching file → no injection.
	ctx2, _ := idx.BuildContext([]string{"internal/other/thing.go"})
	for _, c := range ctx2.Conventions {
		if c.Slug == "conventions/contracts-import-discipline" {
			t.Errorf("convention injected for non-matching file")
		}
	}

	// The nudge path (hero relevant) injects it too.
	nudge, err := idx.BuildNudge([]string{"internal/contracts/manifest.go"})
	if err != nil {
		t.Fatalf("BuildNudge: %v", err)
	}
	var nudged bool
	for _, c := range nudge.Conventions {
		if c.Slug == "conventions/contracts-import-discipline" {
			nudged = true
		}
	}
	if !nudged {
		t.Errorf("scoped flat convention did not nudge; nudge.Conventions=%+v", nudge.Conventions)
	}

	// The unscoped battlecard must never inject (pull-only).
	for _, block := range [][]ContextEntry{ctx.Conventions, ctx.Rules, ctx.Decisions} {
		for _, e := range block {
			if e.Slug == "battlecards/acme" {
				t.Errorf("unscoped battlecard leaked into injected context: %+v", e)
			}
		}
	}
}

// TestFlatTripwireSurfacesInAnchor covers the anchor follow-on: a flat tripwire
// in the isolated knowledge table (which never reaches the specs table) still
// appears in FindAllTripwires — the query behind `hero anchor` — with its
// parsed sections, at parity with spec.md-shaped tripwires.
func TestFlatTripwireSurfacesInAnchor(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeKnowledge(t, heroDir, "tripwires/no-raw-sql.md", `---
title: No Raw SQL in Handlers
type: tripwire
severity: critical
---
# No Raw SQL in Handlers

## Constraint
Never build SQL by string concatenation in HTTP handlers.

## Why
It is the primary injection vector.

## Instead
Use the query builder in internal/db.
`)

	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	// It must never leak into the specs table (isolation boundary).
	var specCount int
	idx.db.QueryRow("SELECT COUNT(*) FROM specs WHERE type = 'tripwire'").Scan(&specCount)
	if specCount != 0 {
		t.Fatalf("flat tripwire leaked into specs table: %d rows", specCount)
	}

	tripwires, err := idx.FindAllTripwires(heroDir)
	if err != nil {
		t.Fatalf("FindAllTripwires: %v", err)
	}
	var found *TripwireResult
	for i := range tripwires {
		if tripwires[i].Slug == "tripwires/no-raw-sql" {
			found = &tripwires[i]
		}
	}
	if found == nil {
		t.Fatalf("flat tripwire did not surface in FindAllTripwires; got %+v", tripwires)
	}
	if found.Severity != "critical" {
		t.Errorf("severity = %q, want critical", found.Severity)
	}
	if found.Constraint == "" || found.Why == "" || found.Instead == "" {
		t.Errorf("tripwire sections not parsed: %+v", found)
	}
}

// flatTripwireBody renders a flat tripwire file with the given title and an
// optional `triggers:` line (omitted when triggers is empty).
func flatTripwireBody(title, triggersLine string) string {
	return "---\ntitle: " + title + "\ntype: tripwire\nseverity: high\n" + triggersLine + "---\n# " + title + `

## Constraint
Do not do the forbidden thing.

## Why
It causes the bad outcome.

## Instead
Do the safe thing.
`
}

func hasTripwire(results []TripwireResult, slug string) bool {
	for i := range results {
		if results[i].Slug == slug {
			return true
		}
	}
	return false
}

// TestFlatTripwireHighlightsByTrigger is the flat-tripwire-trigger-parity repro:
// a flat tripwire with a `triggers:` list must now be trigger-highlighted by
// FindTripwiresByTrigger (before the fix it appeared only in FindAllTripwires),
// and it must highlight identically to a spec.md-shaped tripwire with the same
// triggers for the same context (layout-agnostic parity).
func TestFlatTripwireHighlightsByTrigger(t *testing.T) {
	heroDir := newRefreshHeroDir(t)

	// A flat tripwire in the isolated knowledge table.
	writeKnowledge(t, heroDir, "tripwires/no-globalstate.md",
		flatTripwireBody("No Global State", "triggers: [globalstate, \"mutable singleton\"]\n"))

	// A spec.md-shaped tripwire with identical triggers → reaches the specs
	// table via spec.Discover / IndexSpec (tripwire_triggers).
	writeRefreshSpec(t, heroDir, "no-globalstate-spec", `---
title: No Global State (spec-shaped)
type: tripwire
status: active
triggers: [globalstate, "mutable singleton"]
severity: high
---
# No Global State (spec-shaped)

## Constraint
Do not introduce a global.

## Why
It couples everything.

## Instead
Pass it explicitly.
`)

	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	// Single-token trigger match.
	matched, err := idx.FindTripwiresByTrigger("should we use globalstate here")
	if err != nil {
		t.Fatalf("FindTripwiresByTrigger: %v", err)
	}
	if !hasTripwire(matched, "tripwires/no-globalstate") {
		t.Errorf("flat tripwire did not trigger-highlight; got %+v", matched)
	}
	if !hasTripwire(matched, "no-globalstate-spec") {
		t.Errorf("spec-shaped tripwire did not trigger-highlight; got %+v", matched)
	}

	// Multi-word trigger (substring) match — parity on the second trigger too.
	matched, err = idx.FindTripwiresByTrigger("what about a mutable singleton")
	if err != nil {
		t.Fatalf("FindTripwiresByTrigger (multiword): %v", err)
	}
	if !hasTripwire(matched, "tripwires/no-globalstate") || !hasTripwire(matched, "no-globalstate-spec") {
		t.Errorf("multi-word trigger parity failed; got %+v", matched)
	}

	// The flat result must carry its parsed sections + severity, like the
	// FindAllTripwires flat branch.
	var flat *TripwireResult
	for i := range matched {
		if matched[i].Slug == "tripwires/no-globalstate" {
			flat = &matched[i]
		}
	}
	if flat == nil {
		t.Fatal("flat tripwire missing from multiword match")
	}
	if flat.Severity != "high" || flat.Constraint == "" || flat.Why == "" || flat.Instead == "" {
		t.Errorf("flat tripwire result not fully parsed: %+v", flat)
	}
}

// TestFlatTripwireNoTriggersNeverHighlights covers the negative AC: a flat
// tripwire with no `triggers:` still lists (FindAllTripwires) but is never
// trigger-highlighted.
func TestFlatTripwireNoTriggersNeverHighlights(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	writeKnowledge(t, heroDir, "tripwires/plain.md",
		flatTripwireBody("Plain Tripwire", ""))

	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	all, err := idx.FindAllTripwires(heroDir)
	if err != nil {
		t.Fatalf("FindAllTripwires: %v", err)
	}
	if !hasTripwire(all, "tripwires/plain") {
		t.Errorf("untriggered flat tripwire should still list; got %+v", all)
	}

	matched, err := idx.FindTripwiresByTrigger("plain tripwire mentioned in passing")
	if err != nil {
		t.Fatalf("FindTripwiresByTrigger: %v", err)
	}
	if hasTripwire(matched, "tripwires/plain") {
		t.Errorf("untriggered flat tripwire must never highlight; got %+v", matched)
	}
}

// TestFlatTripwireTriggerSelfHeals covers AC-5: editing away a flat tripwire's
// `triggers:` (delete-then-insert) leaves no stale rows, and removing the file
// orphan-cleans knowledge_triggers on the next refresh — parity with
// knowledge_scopes.
func TestFlatTripwireTriggerSelfHeals(t *testing.T) {
	heroDir := newRefreshHeroDir(t)
	path := writeKnowledge(t, heroDir, "tripwires/heals.md",
		flatTripwireBody("Heals", "triggers: [healtrigger]\n"))

	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh 1: %v", err)
	}
	idx, err := Open(heroDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	countRows := func(db *DB) int {
		var n int
		db.db.QueryRow("SELECT COUNT(*) FROM knowledge_triggers WHERE knowledge_slug = ?",
			"tripwires/heals").Scan(&n)
		return n
	}
	if got := countRows(idx); got != 1 {
		t.Errorf("after index, want 1 knowledge_triggers row, got %d", got)
	}
	idx.Close()

	// Edit-self-heal: rewrite the entry with no triggers via the same ingest
	// path — the delete-then-insert must drop the stale row.
	entry := parseKnowledgeFile(filepath.Join(heroDir, "knowledge"), path)
	entry.Triggers = nil
	idx2, _ := Open(heroDir)
	if err := idx2.IndexKnowledge(entry); err != nil {
		t.Fatalf("IndexKnowledge: %v", err)
	}
	if got := countRows(idx2); got != 0 {
		t.Errorf("after edit-away, want 0 knowledge_triggers rows, got %d", got)
	}
	idx2.Close()

	// Orphan cleanup: remove the file, refresh, assert no stale rows remain and
	// nothing highlights.
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := RefreshIfStale(heroDir); err != nil {
		t.Fatalf("refresh 2: %v", err)
	}
	idx3, _ := Open(heroDir)
	defer idx3.Close()
	if got := countRows(idx3); got != 0 {
		t.Errorf("after remove, want 0 knowledge_triggers rows, got %d", got)
	}
	matched, _ := idx3.FindTripwiresByTrigger("healtrigger now gone")
	if hasTripwire(matched, "tripwires/heals") {
		t.Errorf("removed flat tripwire still highlights; got %+v", matched)
	}
}
