package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
)

func TestExtractAskAndSuggestion_HappyPath(t *testing.T) {
	body := `---
session: claude-x
---

## Last user ask

> "lets just move into the logical next phase in the plan" — after
> finishing the previous run.

## Just finished

work happened.

## Proposed next ask

> *"Finish the traversal-queries phases — natural-language routing
> for hero why and hero blocked, plus the MCP registration so agents
> can call them mid-reasoning."*

## Blocked on

Nothing.
`
	ask, sug := extractAskAndSuggestion(body)
	if ask == "" || !contains(ask, "lets just move into the logical next phase") {
		t.Errorf("ask = %q", ask)
	}
	if sug == "" || !contains(sug, "Finish the traversal-queries phases") {
		t.Errorf("suggestion = %q", sug)
	}
}

func TestExtractAskAndSuggestion_PrefersSuggestedNextOverProposedAsk(t *testing.T) {
	body := `## Suggested next prompt

> let's tackle phase 4

## Proposed next ask

> something older and stale
`
	_, sug := extractAskAndSuggestion(body)
	if !contains(sug, "let's tackle phase 4") {
		t.Errorf("suggestion = %q, want phase 4 (suggested next prompt should win)", sug)
	}
}

func TestExtractAskAndSuggestion_EmptyOnNoSections(t *testing.T) {
	body := `## Some other section

just text, no quotes
`
	ask, sug := extractAskAndSuggestion(body)
	if ask != "" {
		t.Errorf("ask = %q, want empty", ask)
	}
	if sug != "" {
		t.Errorf("sug = %q, want empty", sug)
	}
}

func TestFirstQuoteOrText_PrefersBlockquote(t *testing.T) {
	got := firstQuoteOrText(`
> first quoted line
> second quoted line

next paragraph
`)
	if got != "first quoted line second quoted line" {
		t.Errorf("got %q", got)
	}
}

func TestFirstQuoteOrText_FallsBackToParagraph(t *testing.T) {
	got := firstQuoteOrText(`
just a plain sentence.

second paragraph.
`)
	if got != "just a plain sentence." {
		t.Errorf("got %q", got)
	}
}

func TestFirstQuoteOrText_SkipsItalicPlaceholder(t *testing.T) {
	got := firstQuoteOrText(`_(none recorded yet)_`)
	if got != "" {
		t.Errorf("italic placeholder treated as content: %q", got)
	}
}

// TestExtractAskAndSuggestion_RoundTripsProjectedShape verifies Change
// 2's round-trip assumption: a per-user file already in UserHandoffMD
// shape (## Last user ask / ## Suggested next prompt with blockquote
// bodies) extracts cleanly via the legacy extractAskAndSuggestion —
// the section names and blockquote convention match, so reusing the
// existing extractor is correct and ParseUserHandoff is not required.
func TestExtractAskAndSuggestion_RoundTripsProjectedShape(t *testing.T) {
	projected := `---
user: alice
updated: 2026-06-03T10:00:00Z
repo: hero
---

# alice's handoff

## Last user ask

> where did we leave off on team-mode handoff?

## Suggested next prompt

> let's land the per-user projection

## Recent reflections

- the merge driver is registered by hero install
`
	ask, sug := extractAskAndSuggestion(projected)
	if !contains(ask, "where did we leave off on team-mode handoff") {
		t.Errorf("ask did not round-trip from projected shape: %q", ask)
	}
	if !contains(sug, "let's land the per-user projection") {
		t.Errorf("suggestion did not round-trip from projected shape: %q", sug)
	}
}

// Test_runNextMigrateProjection_TeamModeCapturesPerUserFile pins Change
// 2 in team mode: the migration must capture and ingest the per-user
// .hero/next/<user>.md file (what resolveNextPath / the gate flags),
// NOT a hardcoded .hero/NEXT.md. The shared file is absent here, so a
// hardcoded read would capture nothing.
func Test_runNextMigrateProjection_TeamModeCapturesPerUserFile(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Next = &config.NextConfig{Mode: "team"}
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/alice"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	perUserDir := filepath.Join(env.heroDir, nextDirName)
	if err := os.MkdirAll(perUserDir, 0o755); err != nil {
		t.Fatal(err)
	}
	perUser := `## Last user ask

> capture THIS per-user content on migration

## Suggested next prompt

> finish the team-mode wiring
`
	if err := os.WriteFile(filepath.Join(perUserDir, "alice.md"), []byte(perUser), 0o644); err != nil {
		t.Fatal(err)
	}
	// No shared .hero/NEXT.md on disk — a hardcoded read would miss.

	if _, err := runCmd("next", "migrate-to-projection"); err != nil {
		t.Fatalf("migrate-to-projection (team): %v", err)
	}

	repoKey := gitutil.RepoKey(env.dir)
	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// The captured Note body must be the per-user file's content.
	noteBody := latestMigrationSnapshotBody(t, store)
	if !strings.Contains(noteBody, "capture THIS per-user content") {
		t.Errorf("migration snapshot did not capture the per-user file; got:\n%s", noteBody)
	}

	// The extracted UserAsk/NextSuggestion came from the per-user file.
	ask, _ := handoff.LatestAsk(store, "alice", repoKey, "engineering")
	if ask == nil || !strings.Contains(ask.Text, "capture THIS per-user content") {
		t.Errorf("UserAsk not ingested from per-user file: %+v", ask)
	}
	sug, _ := handoff.LatestSuggestion(store, "alice", repoKey, "engineering")
	if sug == nil || !strings.Contains(sug.Text, "finish the team-mode wiring") {
		t.Errorf("NextSuggestion not ingested from per-user file: %+v", sug)
	}
}

// Test_runNextMigrateProjection_SoloModeCapturesSharedFile is the
// regression guard for Change 2 against solo mode: solo migration must
// still capture the shared .hero/NEXT.md.
func Test_runNextMigrateProjection_SoloModeCapturesSharedFile(t *testing.T) {
	env := newTestEnv(t)
	cfg := config.DefaultConfig()
	cfg.Tracking = &config.TrackingConfig{DefaultAgent: "human/alice"}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save cfg: %v", err)
	}

	shared := `## Last user ask

> capture the SHARED file in solo mode

## Suggested next prompt

> keep solo behavior intact
`
	if err := os.WriteFile(filepath.Join(env.heroDir, nextFileName), []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runCmd("next", "migrate-to-projection"); err != nil {
		t.Fatalf("migrate-to-projection (solo): %v", err)
	}

	store, err := graph.Open(env.heroDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	noteBody := latestMigrationSnapshotBody(t, store)
	if !strings.Contains(noteBody, "capture the SHARED file in solo mode") {
		t.Errorf("solo migration did not capture the shared NEXT.md; got:\n%s", noteBody)
	}
}

// latestMigrationSnapshotBody returns the body prop of the most-recent
// next-md-migration-snapshot Note node.
func latestMigrationSnapshotBody(t *testing.T, store *graph.Store) string {
	t.Helper()
	var body string
	err := store.DB().QueryRow(
		`SELECT COALESCE(json_extract(props, '$.body'), '')
		   FROM nodes
		  WHERE type = 'Note' AND key LIKE 'next-md-migration-snapshot-%'
		    AND valid_to IS NULL
		  ORDER BY ingested_at DESC
		  LIMIT 1`,
	).Scan(&body)
	if err != nil {
		t.Fatalf("query migration snapshot note: %v", err)
	}
	return body
}
