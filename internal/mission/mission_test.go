package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
)

// TestParseHeroCharter parses the actual repo charter so the parser
// stays honest as the file evolves.
func TestParseHeroCharter(t *testing.T) {
	data, err := os.ReadFile(repoMissionPath(t))
	if err != nil {
		t.Skipf("no mission file at expected path: %v", err)
	}
	m, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if m.Title == "" {
		t.Error("title not parsed")
	}
	if m.Version == "" {
		t.Error("version not parsed")
	}
	if m.Scope != "core" {
		t.Errorf("scope = %q, want core", m.Scope)
	}
	if m.MissionStatement == "" {
		t.Error("mission statement empty")
	}
	if len(m.Principles) < 3 {
		t.Errorf("principles parsed = %d, want ≥3", len(m.Principles))
	}
	if len(m.AntiPatterns) < 3 {
		t.Errorf("anti-patterns parsed = %d, want ≥3", len(m.AntiPatterns))
	}
	if len(m.VocabPreferred) == 0 {
		t.Error("preferred vocab empty")
	}
	if len(m.VocabBanned) == 0 {
		t.Error("banned vocab empty")
	}
	if m.MissionFitTest == "" {
		t.Error("mission-fit test empty")
	}
	for i, p := range m.Principles {
		if p.Name == "" {
			t.Errorf("principle[%d] name empty", i)
		}
		if p.Number == 0 && i > 0 {
			t.Errorf("principle[%d] number = 0", i)
		}
	}
}

func TestWriteGraph_RoundTrip(t *testing.T) {
	store := openStore(t)
	m := &Mission{
		Title:            "Test charter",
		Version:          "1",
		LockedAt:         "2026-04-28",
		LockedBy:         "tester",
		Scope:            "core",
		MissionStatement: "do the thing",
		Principles: []Principle{
			{Number: 1, Name: "It works", Body: "..."},
			{Number: 2, Name: "Less is more", Body: "..."},
		},
		AntiPatterns: []string{"complexity", "ceremony"},
	}
	if err := m.WriteGraph("repo-x", store); err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}
	node, err := store.GetNode("Mission", "core", "")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got, _ := node.Props["title"].(string); got != "Test charter" {
		t.Errorf("title in graph = %q", got)
	}
	if got, _ := node.Props["mission_statement"].(string); got != "do the thing" {
		t.Errorf("statement in graph = %q", got)
	}
}

func TestWriteGraph_Idempotent(t *testing.T) {
	store := openStore(t)
	m := &Mission{
		Title:    "X", Version: "1", Scope: "core",
		MissionStatement: "y",
	}
	if err := m.WriteGraph("repo-x", store); err != nil {
		t.Fatalf("first: %v", err)
	}
	first, _ := store.GetNodeID("Mission", "core", "")
	if err := m.WriteGraph("repo-x", store); err != nil {
		t.Fatalf("second: %v", err)
	}
	second, _ := store.GetNodeID("Mission", "core", "")
	if first != second {
		t.Errorf("re-ingest invalidated node: %d → %d", first, second)
	}
}

func TestParseVocabBullet(t *testing.T) {
	cases := map[string]VocabEntry{
		"**foo** — bar":  {Term: "foo", Gloss: "bar"},
		"foo - bar":      {Term: "foo", Gloss: "bar"},
		"**foo**":        {Term: "foo"},
		"*foo* — bar":    {Term: "foo", Gloss: "bar"},
	}
	for in, want := range cases {
		got := parseVocabBullet(in)
		if got != want {
			t.Errorf("parseVocabBullet(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestSplitNumberedHeading(t *testing.T) {
	n, name := splitNumberedHeading("1. It just works.")
	if n != 1 || name != "It just works" {
		t.Errorf("got (%d, %q), want (1, It just works)", n, name)
	}
	n, name = splitNumberedHeading("Random heading")
	if n != 0 || name != "Random heading" {
		t.Errorf("got (%d, %q), want (0, Random heading)", n, name)
	}
}

func openStore(t *testing.T) *graph.Store {
	t.Helper()
	store, err := graph.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graph.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func repoMissionPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Walk up until we find .hero/mission.md
	for {
		candidate := filepath.Join(dir, ".hero", "mission.md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return candidate // doesn't exist; caller will Skip
		}
		dir = parent
	}
}
