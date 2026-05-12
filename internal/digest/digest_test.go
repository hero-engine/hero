package digest

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
)

func openTestStore(t *testing.T) *graph.Store {
	t.Helper()
	s, err := graph.Open(filepath.Join(t.TempDir(), "hero"))
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func seed(t *testing.T, s *graph.Store) {
	t.Helper()
	now := time.Now().UTC()

	// Person + Feature claimed by them
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Person", Key: "alice@example.com",
		Props:       map[string]any{"email": "alice@example.com", "name": "Alice"},
		ContentHash: "h-p",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Feature", Key: "shipping-rewrite", Repo: "test",
		Props: map[string]any{
			"title": "Shipping rewrite", "status": "delivering",
			"priority": "P0", "claimed_by": "alice@example.com",
		},
		ContentHash: "h-f1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Feature", Key: "billing-revamp", Repo: "test",
		Props: map[string]any{
			"title": "Billing revamp", "status": "planning", "priority": "P1",
		},
		ContentHash: "h-f2",
	}); err != nil {
		t.Fatal(err)
	}
	// A commit
	if _, err := s.UpsertNode(&graph.Node{
		Type: "Commit", Key: "abc123def456", Repo: "test",
		Props: map[string]any{
			"sha": "abc123def456", "subject": "feat: ship something",
			"author_name": "Alice", "date": now.Format(time.RFC3339),
		},
		ContentHash: "h-c",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_BriefContainsExpectedSections(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)

	b, err := Generate(store, Options{
		RepoKey:     "test",
		AuthorEmail: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	md := b.Markdown()
	for _, want := range []string{
		"## Who you are",
		"Alice",
		"## In flight",
		"shipping-rewrite",
		"## Just changed",
		"abc123d",
		"## Tried and failed",
		"## Blocked on",
		"## Nearby",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("brief missing %q\n%s", want, md)
		}
	}
}

func TestGenerate_ClaimedByBoostsScore(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)

	b, err := Generate(store, Options{
		RepoKey: "test", AuthorEmail: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	md := b.Markdown()
	// shipping-rewrite is claimed by alice — should appear before billing-revamp
	i1 := strings.Index(md, "shipping-rewrite")
	i2 := strings.Index(md, "billing-revamp")
	if !(i1 > 0 && (i2 < 0 || i1 < i2)) {
		t.Errorf("claimed-by feature should rank first; got shipping=%d billing=%d", i1, i2)
	}
}

func TestGenerate_BudgetTracking(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)

	b, err := Generate(store, Options{
		RepoKey:     "test",
		TokenBudget: 1500,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if b.BudgetTotal != 1500 {
		t.Errorf("BudgetTotal = %d, want 1500", b.BudgetTotal)
	}
	if b.BudgetUsed <= 0 {
		t.Errorf("BudgetUsed = %d, want > 0", b.BudgetUsed)
	}
}

func TestGenerate_SoftBudgetAllowsExcessForHighSignal(t *testing.T) {
	store := openTestStore(t)
	// Stuff a ton of attempts in
	for i := 0; i < 50; i++ {
		key := "a" + string(rune('A'+i%26))
		if _, err := store.UpsertNode(&graph.Node{
			Type: "Attempt", Key: key, Repo: "test",
			Props: map[string]any{
				"body":    "tried something silly that didn't work, attempt " + key,
				"outcome": "failed",
			},
			ContentHash: "h-" + key,
		}); err != nil {
			t.Fatal(err)
		}
	}
	b, err := Generate(store, Options{
		RepoKey:     "test",
		TokenBudget: 3000,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// We expect: section did NOT trim down to a hard limit. With soft
	// budgets, sections only trim past 2× target.
	var triedSec *BriefSection
	for i := range b.Sections {
		if strings.Contains(b.Sections[i].Title, "Tried") {
			triedSec = &b.Sections[i]
			break
		}
	}
	if triedSec == nil {
		t.Fatal("no Tried section")
	}
	if len(triedSec.Lines) < 5 {
		t.Errorf("expected several attempts kept, got %d", len(triedSec.Lines))
	}
}

func TestMarkdown_RendersDigDeeperHintWhenTruncated(t *testing.T) {
	// Force a section to truncate by exceeding hard cap.
	sec := BriefSection{Title: "Tried and failed (skip these dead ends)"}
	for i := 0; i < 2000; i++ {
		sec.Lines = append(sec.Lines, "- some long-ish line of attempt body that will eventually exceed budget")
	}
	dropped := trimToBudget(&sec, 100)
	if dropped == 0 {
		t.Fatal("expected trimming")
	}
	sec.Truncated = dropped
	b := &Brief{Sections: []BriefSection{sec}}
	md := b.Markdown()
	if !strings.Contains(md, "hero recall") {
		t.Errorf("expected 'hero recall' hint in truncated output:\n%s", md)
	}
}
