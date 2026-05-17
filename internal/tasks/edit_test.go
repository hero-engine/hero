package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender_RoundTrip(t *testing.T) {
	original := `- [ ] T-1 First task {kind: qa-blocker, assignee: chet}
- [x] T-2 Second task {done: 2026-05-15T14:22:00Z}
- [/] T-3 Third task {started: 2026-05-16T09:00:00Z}
`
	parsed := ParseTasks(original)
	rendered := Render(parsed, EditOptions{})
	reparsed := ParseTasks(rendered)

	if len(reparsed) != len(parsed) {
		t.Fatalf("count drifted: %d → %d", len(parsed), len(reparsed))
	}
	for i, p := range parsed {
		r := reparsed[i]
		if p.ID != r.ID || p.Text != r.Text || p.Status != r.Status {
			t.Errorf("round-trip drift at %d: %+v → %+v", i, p, r)
		}
		if p.Kind != r.Kind || p.Assignee != r.Assignee || p.DiscoveredAgainst != r.DiscoveredAgainst {
			t.Errorf("metadata drift at %d: %+v → %+v", i, p, r)
		}
		if p.Started != r.Started || p.Done != r.Done {
			t.Errorf("timestamp drift at %d: %+v → %+v", i, p, r)
		}
	}
}

func TestRender_StableOrder(t *testing.T) {
	tasks := []Task{
		{ID: "T-3", Text: "third", Status: StatusTodo},
		{ID: "T-1", Text: "first", Status: StatusTodo},
		{ID: "T-2", Text: "second", Status: StatusTodo},
	}
	body := Render(tasks, EditOptions{})
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if !strings.Contains(lines[0], "T-1") || !strings.Contains(lines[1], "T-2") || !strings.Contains(lines[2], "T-3") {
		t.Errorf("not numerically sorted:\n%s", body)
	}
}

func TestUpsertSection_CreatesNewSection(t *testing.T) {
	content := `## Goal

Some goal here.

## Boundaries

Some boundaries.`
	updated := UpsertSection(content, "Tasks", "- [ ] T-1 Hi\n")
	if !strings.Contains(updated, "## Tasks") {
		t.Errorf("Tasks section not created:\n%s", updated)
	}
	if !strings.Contains(updated, "- [ ] T-1 Hi") {
		t.Errorf("task body missing:\n%s", updated)
	}
	if !strings.Contains(updated, "## Goal") || !strings.Contains(updated, "## Boundaries") {
		t.Errorf("existing sections lost:\n%s", updated)
	}
}

func TestUpsertSection_ReplacesExisting(t *testing.T) {
	content := `## Goal

Some goal here.

## Tasks

- [ ] T-1 Old task

## Boundaries

Some boundaries.`
	updated := UpsertSection(content, "Tasks", "- [x] T-1 New task\n- [ ] T-2 Another\n")
	if strings.Contains(updated, "Old task") {
		t.Errorf("old content survived:\n%s", updated)
	}
	if !strings.Contains(updated, "New task") || !strings.Contains(updated, "Another") {
		t.Errorf("new content missing:\n%s", updated)
	}
	if !strings.Contains(updated, "## Boundaries") {
		t.Errorf("trailing section lost:\n%s", updated)
	}
	// Goal section content must remain intact.
	if !strings.Contains(updated, "Some goal here.") {
		t.Errorf("Goal body lost:\n%s", updated)
	}
}

func TestApplyToFile_WritesNewSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.md")
	original := "---\ntitle: x\n---\n\n## Goal\n\nSomething.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyToFile(path, "- [ ] T-1 Hello\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "## Tasks") || !strings.Contains(string(got), "T-1 Hello") {
		t.Errorf("file not updated:\n%s", got)
	}
}

func TestAddTask_AssignsNextID(t *testing.T) {
	existing := []Task{{ID: "T-1", Text: "x"}}
	newTask, updated := AddTask(existing, "fresh", AddOptions{Kind: "chore"})
	if newTask.ID != "T-2" {
		t.Errorf("new ID = %q, want T-2", newTask.ID)
	}
	if newTask.Status != StatusTodo {
		t.Errorf("default status = %q, want todo", newTask.Status)
	}
	if newTask.Kind != "chore" {
		t.Errorf("kind not applied: %+v", newTask)
	}
	if len(updated) != 2 {
		t.Errorf("updated count = %d, want 2", len(updated))
	}
}

func TestTransitionTo_StampsTimestamps(t *testing.T) {
	existing := []Task{
		{ID: "T-1", Text: "x", Status: StatusTodo},
	}
	doing, err := TransitionTo(existing, "T-1", StatusDoing)
	if err != nil {
		t.Fatal(err)
	}
	if doing[0].Status != StatusDoing {
		t.Errorf("status = %q, want doing", doing[0].Status)
	}
	if doing[0].Started == "" {
		t.Errorf("started timestamp not stamped")
	}

	done, err := TransitionTo(doing, "T-1", StatusDone)
	if err != nil {
		t.Fatal(err)
	}
	if done[0].Status != StatusDone || done[0].Done == "" {
		t.Errorf("done transition wrong: %+v", done[0])
	}
}

func TestTransitionTo_UnknownIDIsError(t *testing.T) {
	existing := []Task{{ID: "T-1", Text: "x"}}
	if _, err := TransitionTo(existing, "T-99", StatusDone); err == nil {
		t.Errorf("expected error for unknown ID")
	}
}

func TestTransitionTo_UnknownStatusIsError(t *testing.T) {
	existing := []Task{{ID: "T-1", Text: "x"}}
	if _, err := TransitionTo(existing, "T-1", "frobnicated"); err == nil {
		t.Errorf("expected error for bogus status")
	}
}
