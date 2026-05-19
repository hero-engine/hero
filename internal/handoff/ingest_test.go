package handoff

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleHandoffMarkdown = `---
user: alice
updated: 2026-04-29T03:23:29Z
repo: chet-bellows/hero
session: sess-42
---

# alice's handoff

## Last user ask

> where did we leave off on the auth bug?

## Suggested next prompt

> let's tackle phase 5 of next-as-projection

_Rationale: rounds out the cross-machine continuity story_

## Recent reflections

- merge=ours plus stop-hook regen is the right policy
- author-agnostic NEXT.md is cleaner than per-author roll-ups

## Tried and failed (this session)

Nothing this session.

## Your recent activity

- ` + "`abc1234`" + ` — feat: phase 4 projection wiring
`

func TestParseUserHandoff_HappyPath(t *testing.T) {
	parsed, err := ParseUserHandoff([]byte(sampleHandoffMarkdown))
	if err != nil {
		t.Fatalf("ParseUserHandoff: %v", err)
	}
	if parsed.User != "alice" {
		t.Errorf("User = %q, want alice", parsed.User)
	}
	if parsed.Ask == nil || parsed.Ask.Text != "where did we leave off on the auth bug?" {
		t.Errorf("Ask = %+v", parsed.Ask)
	}
	if parsed.Suggestion == nil {
		t.Fatal("Suggestion = nil")
	}
	if parsed.Suggestion.Text != "let's tackle phase 5 of next-as-projection" {
		t.Errorf("Suggestion.Text = %q", parsed.Suggestion.Text)
	}
	if parsed.Suggestion.Rationale != "rounds out the cross-machine continuity story" {
		t.Errorf("Suggestion.Rationale = %q", parsed.Suggestion.Rationale)
	}
	if len(parsed.Reflections) != 2 {
		t.Errorf("Reflections len = %d, want 2", len(parsed.Reflections))
	}
}

func TestParseUserHandoff_PlaceholdersTreatedAsEmpty(t *testing.T) {
	md := `---
user: bob
---

# bob's handoff

## Last user ask

_(none recorded — ` + "`hero next ask \"...\"`" + ` to set)_

## Suggested next prompt

_(none recorded — ` + "`hero next suggest \"...\"`" + ` to set)_

## Recent reflections

_(none yet)_
`
	parsed, err := ParseUserHandoff([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Ask != nil {
		t.Errorf("Ask = %+v, want nil for placeholder", parsed.Ask)
	}
	if parsed.Suggestion != nil {
		t.Errorf("Suggestion = %+v, want nil for placeholder", parsed.Suggestion)
	}
	if len(parsed.Reflections) != 0 {
		t.Errorf("Reflections = %v, want none for placeholder", parsed.Reflections)
	}
}

func TestIngestUserFile_RoundTripsAcrossMachines(t *testing.T) {
	// Simulate the cross-machine scenario:
	//   1. Machine A: graph state → file projection (write).
	//   2. File travels via git push/pull (just write to disk here).
	//   3. Machine B: clean graph + file → ingest → graph state.
	//   4. Machine B's queries return the same suggestion text.
	dir := t.TempDir()
	path := filepath.Join(dir, "alice.md")
	if err := os.WriteFile(path, []byte(sampleHandoffMarkdown), 0o644); err != nil {
		t.Fatal(err)
	}

	storeB := openTestStore(t)
	if err := IngestUserFile(storeB, "repo-x", "engineering", path); err != nil {
		t.Fatalf("IngestUserFile: %v", err)
	}

	got, err := LatestSuggestion(storeB, "alice", "repo-x", "engineering")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("LatestSuggestion = nil after ingest")
	}
	if got.Text != "let's tackle phase 5 of next-as-projection" {
		t.Errorf("Suggestion.Text = %q after round-trip", got.Text)
	}

	ask, _ := LatestAsk(storeB, "alice", "repo-x", "engineering")
	if ask == nil || ask.Text != "where did we leave off on the auth bug?" {
		t.Errorf("Ask round-trip lost: %+v", ask)
	}

	refs, _ := RecentReflections(storeB, "alice", "repo-x", "engineering", 10)
	if len(refs) != 2 {
		t.Errorf("Reflections len after ingest = %d, want 2", len(refs))
	}
}

func TestIngestUserFile_IdempotentOnReingest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alice.md")
	if err := os.WriteFile(path, []byte(sampleHandoffMarkdown), 0o644); err != nil {
		t.Fatal(err)
	}
	store := openTestStore(t)

	if err := IngestUserFile(store, "repo-x", "engineering", path); err != nil {
		t.Fatal(err)
	}
	first, _ := RecentReflections(store, "alice", "repo-x", "engineering", 10)

	// Re-ingest the same file. Reflections shouldn't double up.
	if err := IngestUserFile(store, "repo-x", "engineering", path); err != nil {
		t.Fatal(err)
	}
	second, _ := RecentReflections(store, "alice", "repo-x", "engineering", 10)

	if len(first) != len(second) {
		t.Errorf("reflections grew on re-ingest: %d → %d", len(first), len(second))
	}
}

func TestIngestUserFile_MissingFileIsNoOp(t *testing.T) {
	store := openTestStore(t)
	err := IngestUserFile(store, "repo-x", "engineering", "/no/such/path.md")
	if err != nil {
		t.Errorf("missing file should be no-op, got %v", err)
	}
}

func TestIngestUserFile_SkipsAutoDerivedSuggestion(t *testing.T) {
	// The projection renders "_Source: auto-derived_" on fallback
	// suggestions. Re-ingesting them would corrupt the graph because
	// the source footer would itself become recorded text.
	autoDerivedFile := `---
user: alice
---

# alice's handoff

## Last user ask

> something

## Suggested next prompt

> let's tackle Some Feature

_Rationale: highest-priority open feature: Some Feature_

_Source: auto-derived from open feature — ` + "`hero next suggest \"...\"`" + ` to override._

## Recent reflections

_(none yet)_
`
	dir := t.TempDir()
	path := dir + "/alice.md"
	if err := os.WriteFile(path, []byte(autoDerivedFile), 0o644); err != nil {
		t.Fatal(err)
	}
	store := openTestStore(t)
	if err := IngestUserFile(store, "repo-x", "engineering", path); err != nil {
		t.Fatal(err)
	}
	got, _ := LatestSuggestion(store, "alice", "repo-x", "engineering")
	if got != nil {
		t.Errorf("LatestSuggestion = %+v, want nil (auto-derived should not round-trip)", got)
	}
}
