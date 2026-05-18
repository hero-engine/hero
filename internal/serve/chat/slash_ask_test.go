package chat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAskHandlerNoCorpus runs /ask against an empty workspace and
// asserts the handler streams a "no knowledge" token and a chat.done
// without erroring.
func TestAskHandlerNoCorpus(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := DispatchRequest{
		Prompt:  "/ask what specs do we have",
		Slash:   &SlashInvoc{Name: "ask", Args: "what specs do we have"},
		Context: DispatchContext{Workspace: root},
	}
	out := make(chan Event, 8)
	if err := askHandler(context.Background(), req, out); err != nil {
		t.Fatalf("askHandler: %v", err)
	}
	close(out)

	var sawToken, sawDone bool
	for ev := range out {
		switch ev.Type {
		case EvToken:
			sawToken = true
			if text, _ := ev.Payload["text"].(string); !strings.Contains(strings.ToLower(text), "no knowledge") {
				t.Errorf("expected 'no knowledge' text, got %q", text)
			}
		case EvDone:
			sawDone = true
		}
	}
	if !sawToken {
		t.Error("expected a token event")
	}
	if !sawDone {
		t.Error("expected a done event")
	}
}

func TestAskHandlerMissingWorkspace(t *testing.T) {
	req := DispatchRequest{
		Prompt: "/ask hello",
		Slash:  &SlashInvoc{Name: "ask", Args: "hello"},
	}
	out := make(chan Event, 4)
	if err := askHandler(context.Background(), req, out); err != nil {
		t.Fatalf("askHandler: %v", err)
	}
	close(out)
	var sawErr, sawDone bool
	for ev := range out {
		if ev.Type == EvError {
			sawErr = true
		}
		if ev.Type == EvDone {
			sawDone = true
		}
	}
	if !sawErr || !sawDone {
		t.Fatalf("expected error + done events, got sawErr=%v sawDone=%v", sawErr, sawDone)
	}
}

func TestAskHandlerEmptyQuestion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	req := DispatchRequest{
		Prompt:  "/ask",
		Slash:   &SlashInvoc{Name: "ask", Args: ""},
		Context: DispatchContext{Workspace: root},
	}
	out := make(chan Event, 4)
	if err := askHandler(context.Background(), req, out); err != nil {
		t.Fatalf("askHandler: %v", err)
	}
	close(out)
	var sawErr bool
	for ev := range out {
		if ev.Type == EvError {
			sawErr = true
		}
	}
	if !sawErr {
		t.Error("expected error for empty question")
	}
}
