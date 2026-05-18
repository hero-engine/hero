package chat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoteHandlerWritesFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := DispatchRequest{
		Prompt:  "/note check the registry\nthe registry is the source of truth",
		Slash:   &SlashInvoc{Name: "note", Args: "check the registry\nthe registry is the source of truth"},
		Context: DispatchContext{Workspace: root},
	}
	out := make(chan Event, 8)
	if err := noteHandler(context.Background(), req, out); err != nil {
		t.Fatalf("noteHandler: %v", err)
	}
	close(out)

	var notePath string
	var sawDone bool
	for ev := range out {
		if ev.Type == EvDone {
			sawDone = true
			if outcome, ok := ev.Payload["outcome"].(map[string]interface{}); ok {
				if f, ok := outcome["file"].(string); ok {
					notePath = f
				}
			}
		}
	}
	if !sawDone {
		t.Fatal("expected done event")
	}
	if notePath == "" {
		t.Fatal("expected outcome.file in done payload")
	}

	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	content := string(data)
	for _, want := range []string{"---", "type: note", "status: active", "created:", "the registry is the source of truth"} {
		if !strings.Contains(content, want) {
			t.Errorf("note missing %q\n--- got ---\n%s", want, content)
		}
	}
}

func TestNoteHandlerEmptyArgs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := DispatchRequest{
		Prompt:  "/note",
		Slash:   &SlashInvoc{Name: "note"},
		Context: DispatchContext{Workspace: root},
	}
	out := make(chan Event, 4)
	if err := noteHandler(context.Background(), req, out); err != nil {
		t.Fatalf("noteHandler: %v", err)
	}
	close(out)
	var sawErr bool
	for ev := range out {
		if ev.Type == EvError {
			sawErr = true
		}
	}
	if !sawErr {
		t.Fatal("expected error event for empty /note")
	}
}
