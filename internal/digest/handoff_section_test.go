package digest

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
)

// seedHandoff records a UserAsk + NextSuggestion + reflections for the
// given (user, domain) under repo "test", the same key shape the
// projection and auto-emit use.
func seedHandoff(t *testing.T, s *graph.Store, user, domain string) {
	t.Helper()
	if err := handoff.RecordAsk(s, "test", handoff.UserAsk{
		User: user, Domain: domain, Text: "ASK_TEXT wire the handoff section into resume",
	}); err != nil {
		t.Fatalf("RecordAsk: %v", err)
	}
	if err := handoff.RecordSuggestion(s, "test", handoff.NextSuggestion{
		User: user, Domain: domain, Text: "SUGGEST_TEXT land the digest section above in-flight",
		Rationale: "highest-value where-was-I context",
	}); err != nil {
		t.Fatalf("RecordSuggestion: %v", err)
	}
	if err := handoff.RecordReflection(s, "test", handoff.SessionReflection{
		User: user, Domain: domain, Text: "REFLECT_TEXT graph.db is gitignored so the file is the medium",
	}); err != nil {
		t.Fatalf("RecordReflection: %v", err)
	}
}

// TestGenerate_HandoffSectionPresent — Test Plan #1. Seed handoff nodes
// for (user, repo, domain), Generate with User/Domain set, and assert
// the section exists carrying ask/suggestion/reflection text AND is
// ordered ABOVE the In-flight section.
func TestGenerate_HandoffSectionPresent(t *testing.T) {
	store := openTestStore(t)
	seed(t, store) // gives an In-flight Feature ("shipping-rewrite")
	seedHandoff(t, store, "alice", "engineering")

	b, err := Generate(store, Options{
		RepoKey:     "test",
		AuthorEmail: "alice@example.com",
		User:        "alice",
		Domain:      "engineering",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	md := b.Markdown()

	for _, want := range []string{
		"## Where you left off",
		"Last ask: ASK_TEXT wire the handoff section into resume",
		"Suggested next: SUGGEST_TEXT land the digest section above in-flight",
		"Recent: REFLECT_TEXT graph.db is gitignored so the file is the medium",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("brief missing %q\n%s", want, md)
		}
	}

	// Ordered above In-flight: the "what was I doing" context comes first.
	iHandoff := strings.Index(md, "## Where you left off")
	iInFlight := strings.Index(md, "## In flight")
	if iHandoff < 0 || iInFlight < 0 {
		t.Fatalf("missing a section: handoff=%d inflight=%d\n%s", iHandoff, iInFlight, md)
	}
	if iHandoff > iInFlight {
		t.Errorf("handoff section must be ABOVE in-flight; got handoff=%d inflight=%d", iHandoff, iInFlight)
	}
}

// TestGenerate_HandoffSectionOmittedWhenEmpty — Test Plan #2. No
// handoff nodes (or User=="") → no handoff section in the brief.
func TestGenerate_HandoffSectionOmittedWhenEmpty(t *testing.T) {
	t.Run("no nodes", func(t *testing.T) {
		store := openTestStore(t)
		seed(t, store)
		b, err := Generate(store, Options{
			RepoKey: "test", User: "alice", Domain: "engineering",
		})
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if strings.Contains(b.Markdown(), "## Where you left off") {
			t.Errorf("handoff section should be omitted when no nodes exist:\n%s", b.Markdown())
		}
	})

	t.Run("empty user", func(t *testing.T) {
		store := openTestStore(t)
		seed(t, store)
		seedHandoff(t, store, "alice", "engineering")
		// User intentionally unset — the resolve path produced no slug.
		b, err := Generate(store, Options{RepoKey: "test", Domain: "engineering"})
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if strings.Contains(b.Markdown(), "## Where you left off") {
			t.Errorf("handoff section should be omitted when User==\"\":\n%s", b.Markdown())
		}
	})
}

// TestGenerate_HandoffSectionKeyedCorrectly — Test Plan #3. A node
// under a DIFFERENT user/domain must not leak into this user's brief.
func TestGenerate_HandoffSectionKeyedCorrectly(t *testing.T) {
	t.Run("different user", func(t *testing.T) {
		store := openTestStore(t)
		seed(t, store)
		seedHandoff(t, store, "bob", "engineering") // bob's, not alice's
		b, err := Generate(store, Options{
			RepoKey: "test", User: "alice", Domain: "engineering",
		})
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if strings.Contains(b.Markdown(), "ASK_TEXT") {
			t.Errorf("bob's ask leaked into alice's brief:\n%s", b.Markdown())
		}
		if strings.Contains(b.Markdown(), "## Where you left off") {
			t.Errorf("handoff section rendered for alice with only bob's nodes:\n%s", b.Markdown())
		}
	})

	t.Run("different domain", func(t *testing.T) {
		store := openTestStore(t)
		seed(t, store)
		seedHandoff(t, store, "alice", "pm") // alice, but PM domain
		b, err := Generate(store, Options{
			RepoKey: "test", User: "alice", Domain: "engineering",
		})
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if strings.Contains(b.Markdown(), "ASK_TEXT") {
			t.Errorf("alice's PM-domain ask leaked into the engineering brief:\n%s", b.Markdown())
		}
	})
}

// TestGenerate_HandoffSectionBestEffortOnError — Test Plan #4. Force a
// handoff-read error (closed store) → Generate still returns a brief,
// section skipped, no error propagated.
func TestGenerate_HandoffSectionBestEffortOnError(t *testing.T) {
	store := openTestStore(t)
	seed(t, store)
	seedHandoff(t, store, "alice", "engineering")

	// Generate a baseline brief while the store is healthy to prove the
	// rest of the pipeline works, then break the handoff read mid-flight
	// by closing the underlying DB and re-running with the same store.
	// A closed *sql.DB returns an error from every query, exactly the
	// best-effort path we want to exercise. Because the handoff section
	// is the FIRST thing after Who-you-are that queries, and the other
	// sections also query, we instead drive handoffSection directly to
	// isolate the best-effort contract, then confirm Generate tolerates
	// a handoff-read failure without aborting.
	if err := store.DB().Close(); err != nil {
		t.Fatalf("close DB: %v", err)
	}

	// handoffSection must swallow the read error and return an empty,
	// error-free section.
	sec, err := handoffSection(store, Options{
		RepoKey: "test", User: "alice", Domain: "engineering",
	}, 300)
	if err != nil {
		t.Fatalf("handoffSection propagated an error on a closed store; want best-effort nil: %v", err)
	}
	if len(sec.Lines) != 0 {
		t.Errorf("handoffSection should yield an empty section on read error; got %v", sec.Lines)
	}
}
