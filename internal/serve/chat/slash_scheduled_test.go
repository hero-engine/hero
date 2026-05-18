package chat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScheduledHandlerWritesYAML(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := DispatchRequest{
		Prompt:  "/scheduled every weekday at 9am -> /pulse",
		Slash:   &SlashInvoc{Name: "scheduled", Args: "every weekday at 9am -> /pulse"},
		Context: DispatchContext{Workspace: root},
	}
	out := make(chan Event, 8)
	if err := scheduledHandler(context.Background(), req, out); err != nil {
		t.Fatalf("scheduledHandler: %v", err)
	}
	close(out)

	var path string
	for ev := range out {
		if ev.Type == EvDone {
			if outcome, ok := ev.Payload["outcome"].(map[string]interface{}); ok {
				if f, ok := outcome["file"].(string); ok {
					path = f
				}
			}
		}
	}
	if path == "" {
		t.Fatal("expected outcome.file in done payload")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{"trigger:", "action:", "miss_policy: queue", "status: draft"} {
		if !strings.Contains(content, want) {
			t.Errorf("yaml missing %q\n--- got ---\n%s", want, content)
		}
	}
}

func TestScheduledHandlerBadInput(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := DispatchRequest{
		Prompt:  "/scheduled just some words",
		Slash:   &SlashInvoc{Name: "scheduled", Args: "just some words"},
		Context: DispatchContext{Workspace: root},
	}
	out := make(chan Event, 4)
	if err := scheduledHandler(context.Background(), req, out); err != nil {
		t.Fatalf("scheduledHandler: %v", err)
	}
	close(out)
	var sawErr bool
	var msg string
	for ev := range out {
		if ev.Type == EvError {
			sawErr = true
			msg, _ = ev.Payload["message"].(string)
		}
	}
	if !sawErr {
		t.Fatal("expected error event")
	}
	if !strings.Contains(msg, "Example:") {
		t.Errorf("error message should include example, got %q", msg)
	}
}
