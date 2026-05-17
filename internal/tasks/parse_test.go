package tasks

import (
	"strings"
	"testing"
)

func TestParseTasks_CanonicalForm(t *testing.T) {
	body := `
- [ ] T-1 Fix login redirect loop {kind: qa-blocker, assignee: chet, discovered_against: checkout-flow}
- [x] T-2 Migrate token storage to keychain {kind: chore, done: 2026-05-15T14:22:00Z}
- [/] T-3 Wire up retry-with-backoff {assignee: bwheeler, started: 2026-05-16T09:00:00Z}
`
	parsed := ParseTasks(body)
	if len(parsed) != 3 {
		t.Fatalf("parsed %d tasks, want 3 — %+v", len(parsed), parsed)
	}

	if parsed[0].ID != "T-1" || parsed[0].Status != StatusTodo {
		t.Errorf("T-1: %+v", parsed[0])
	}
	if parsed[0].Kind != "qa-blocker" || parsed[0].Assignee != "chet" || parsed[0].DiscoveredAgainst != "checkout-flow" {
		t.Errorf("T-1 metadata: %+v", parsed[0])
	}
	if !strings.Contains(parsed[0].Text, "Fix login redirect loop") {
		t.Errorf("T-1 text wrong: %q", parsed[0].Text)
	}

	if parsed[1].Status != StatusDone {
		t.Errorf("T-2 status = %q, want done", parsed[1].Status)
	}
	if parsed[1].Done != "2026-05-15T14:22:00Z" {
		t.Errorf("T-2 done = %q", parsed[1].Done)
	}

	if parsed[2].Status != StatusDoing {
		t.Errorf("T-3 status = %q, want doing", parsed[2].Status)
	}
	if parsed[2].Started != "2026-05-16T09:00:00Z" {
		t.Errorf("T-3 started = %q", parsed[2].Started)
	}
}

func TestParseTasks_EmptyBody(t *testing.T) {
	if parsed := ParseTasks(""); parsed != nil {
		t.Errorf("expected nil, got %+v", parsed)
	}
	if parsed := ParseTasks("\n\n   \n"); parsed != nil {
		t.Errorf("whitespace-only body should yield nil, got %+v", parsed)
	}
}

func TestParseTasks_NoMetadata(t *testing.T) {
	body := "- [ ] T-1 Simple task with no metadata"
	parsed := ParseTasks(body)
	if len(parsed) != 1 {
		t.Fatalf("parsed %d, want 1", len(parsed))
	}
	if parsed[0].Text != "Simple task with no metadata" {
		t.Errorf("text = %q", parsed[0].Text)
	}
	if parsed[0].Kind != "" || parsed[0].Assignee != "" {
		t.Errorf("metadata should be empty: %+v", parsed[0])
	}
}

func TestParseTasks_TolerantOfPartialMetadata(t *testing.T) {
	body := "- [/] T-7 Halfway-done thing {kind: chore}"
	parsed := ParseTasks(body)
	if len(parsed) != 1 {
		t.Fatalf("parsed %d, want 1", len(parsed))
	}
	if parsed[0].Status != StatusDoing || parsed[0].Kind != "chore" {
		t.Errorf("got %+v", parsed[0])
	}
}

func TestParseTasks_IgnoresNonTaskLines(t *testing.T) {
	body := `
Some prose explaining the section.

- [ ] T-1 Real task
- Not a task at all
- [ ] Also not a task (missing T-N)
- [ ] T-2 Another real task

More prose.`
	parsed := ParseTasks(body)
	if len(parsed) != 2 {
		t.Fatalf("parsed %d, want 2 — %+v", len(parsed), parsed)
	}
	if parsed[0].ID != "T-1" || parsed[1].ID != "T-2" {
		t.Errorf("IDs wrong: %+v", parsed)
	}
}

func TestParseTasks_DuplicateIDKeepsFirst(t *testing.T) {
	body := `
- [ ] T-1 First entry
- [x] T-1 Duplicate — should be ignored
`
	parsed := ParseTasks(body)
	if len(parsed) != 1 {
		t.Fatalf("parsed %d, want 1", len(parsed))
	}
	if parsed[0].Status != StatusTodo {
		t.Errorf("first occurrence should win: %+v", parsed[0])
	}
}

func TestNextID(t *testing.T) {
	if got := NextID(nil); got != "T-1" {
		t.Errorf("empty: NextID = %q, want T-1", got)
	}
	existing := []Task{{ID: "T-1"}, {ID: "T-3"}, {ID: "T-2"}}
	if got := NextID(existing); got != "T-4" {
		t.Errorf("NextID(1,2,3) = %q, want T-4", got)
	}
	// Malformed IDs are tolerated and don't disturb the max.
	mixed := []Task{{ID: "T-5"}, {ID: "broken"}}
	if got := NextID(mixed); got != "T-6" {
		t.Errorf("mixed: NextID = %q, want T-6", got)
	}
}

func TestSortByID_NumericNotLexical(t *testing.T) {
	in := []Task{{ID: "T-10"}, {ID: "T-2"}, {ID: "T-1"}}
	out := SortByID(in)
	got := []string{out[0].ID, out[1].ID, out[2].ID}
	want := []string{"T-1", "T-2", "T-10"}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("SortByID order: got %v, want %v", got, want)
		}
	}
}

func TestParseTasks_MarkerCaseTolerant(t *testing.T) {
	body := `
- [X] T-1 Capital X means done
- [x] T-2 Lower x also means done
- [ ] T-3 Space means todo
`
	parsed := ParseTasks(body)
	if len(parsed) != 3 {
		t.Fatalf("parsed %d, want 3", len(parsed))
	}
	if parsed[0].Status != StatusDone || parsed[1].Status != StatusDone {
		t.Errorf("status wrong: %+v %+v", parsed[0], parsed[1])
	}
	if parsed[2].Status != StatusTodo {
		t.Errorf("status wrong: %+v", parsed[2])
	}
}
