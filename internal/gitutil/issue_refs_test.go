package gitutil

import "testing"

func TestParseIssueRefs_GitHub(t *testing.T) {
	got := parseIssueRefs("fix: stop the bleeding (fixes #42)")
	if len(got) != 1 {
		t.Fatalf("got %d refs, want 1: %+v", len(got), got)
	}
	if got[0].key != "GH#42" || got[0].edgeType != "fixes" || got[0].tracker != "github" {
		t.Errorf("ref wrong: %+v", got[0])
	}
}

func TestParseIssueRefs_Jira(t *testing.T) {
	got := parseIssueRefs("closes PROJ-123: drop the legacy adapter")
	if len(got) != 1 {
		t.Fatalf("got %d refs, want 1: %+v", len(got), got)
	}
	if got[0].key != "PROJ-123" || got[0].edgeType != "closes" || got[0].tracker != "jira" {
		t.Errorf("ref wrong: %+v", got[0])
	}
}

func TestParseIssueRefs_BareReferenceIsMention(t *testing.T) {
	got := parseIssueRefs("refactor: split the parser (see PROJ-9)")
	if len(got) != 1 {
		t.Fatalf("got %d refs, want 1: %+v", len(got), got)
	}
	if got[0].edgeType != "mentions" {
		t.Errorf("expected mentions edge, got %q", got[0].edgeType)
	}
}

func TestParseIssueRefs_MultipleAndDedup(t *testing.T) {
	got := parseIssueRefs("fix: scoped cleanup, fixes #1, fixes #2, see #1 again")
	if len(got) != 2 {
		t.Fatalf("got %d refs, want 2: %+v", len(got), got)
	}
}

func TestParseIssueRefs_BreaksVerb(t *testing.T) {
	got := parseIssueRefs("breaking: drop legacy api (breaks PROJ-99)")
	if len(got) != 1 || got[0].edgeType != "breaks" {
		t.Errorf("expected breaks edge, got %+v", got)
	}
}

func TestParseIssueRefs_None(t *testing.T) {
	got := parseIssueRefs("chore: tidy up imports")
	if len(got) != 0 {
		t.Errorf("expected 0 refs, got %+v", got)
	}
}

func TestParseIssueRefs_LowercaseTrackerNotMatched(t *testing.T) {
	// proj-1 is lowercase — must not match the Jira pattern, which
	// requires uppercase project keys.
	got := parseIssueRefs("misc: see proj-1")
	if len(got) != 0 {
		t.Errorf("lowercase project key should not match: %+v", got)
	}
}
