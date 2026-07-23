package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/suggestion"
)

type cliFocusResolver struct {
	ref         *attention.ProjectReference
	path        string
	autoCurrent bool
}

func (r cliFocusResolver) ResolveInput(value string) (*attention.ProjectReference, error) {
	if value == "" {
		return nil, nil
	}
	if value != "demo" {
		return nil, errors.New("unknown project")
	}
	return r.ref, nil
}
func (r cliFocusResolver) ResolveReference(ref *attention.ProjectReference) focus.ResolvedProject {
	if ref != nil && r.ref != nil && ref.PeerID == r.ref.PeerID {
		return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectAvailable, Path: r.path}
	}
	return focus.ResolvedProject{Reference: ref, Availability: focus.ProjectMissing}
}
func (r cliFocusResolver) ResolveCurrent() (*attention.ProjectReference, error) {
	if r.autoCurrent {
		return r.ref, nil
	}
	return nil, nil
}

func withFocusCLI(t *testing.T, resolver focus.ProjectResolver) {
	t.Helper()
	oldRoot, oldLoader := focusStateRootOverride, focusResolverLoader
	focusStateRootOverride = t.TempDir()
	focusResolverLoader = func() (focus.ProjectResolver, error) { return resolver, nil }
	t.Cleanup(func() { focusStateRootOverride, focusResolverLoader = oldRoot, oldLoader })
}

func runFocusCommand(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := newFocusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestFocusCLIEndToEnd(t *testing.T) {
	ref := &attention.ProjectReference{PeerID: "peer-demo", RegistrySlug: "demo", DisplayName: "Demo"}
	projectPath := filepath.Join(t.TempDir(), "demo")
	withFocusCLI(t, cliFocusResolver{ref: ref, path: projectPath})
	prompt := "Continue from the exact checkpoint.\nSecond line.\n"
	out, err := runFocusCommand(t, prompt, "add", "--title", "Checkpoint", "--prompt-file", "-", "--project", "demo", "--state", "today")
	if err != nil {
		t.Fatal(err)
	}
	var created focus.Item
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("add output %q: %v", out, err)
	}
	if created.Prompt != prompt || created.Lifecycle != attention.FocusToday || created.Project == nil {
		t.Fatalf("created = %#v", created)
	}

	out, err = runFocusCommand(t, "", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var listed []focus.ListedItem
	if err := json.Unmarshal([]byte(out), &listed); err != nil || len(listed) != 1 || listed[0].Availability != focus.ProjectAvailable {
		t.Fatalf("list = %q, %v", out, err)
	}

	out, err = runFocusCommand(t, "", "move", created.ID, "later", "--revision", fmtRevision(created.Revision), "--json")
	if err != nil {
		t.Fatal(err)
	}
	var moved focus.Item
	if err := json.Unmarshal([]byte(out), &moved); err != nil || moved.Lifecycle != attention.FocusLater {
		t.Fatalf("move = %q, %v", out, err)
	}

	out, err = runFocusCommand(t, "", "launch", created.ID, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var intent focus.LaunchIntent
	if err := json.Unmarshal([]byte(out), &intent); err != nil || intent.Prompt != prompt || intent.Path != projectPath {
		t.Fatalf("launch = %q, %v", out, err)
	}

	_, err = runFocusCommand(t, "", "done", created.ID, "--revision", fmtRevision(created.Revision))
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale done err = %v", err)
	}
	out, err = runFocusCommand(t, "", "done", created.ID, "--revision", fmtRevision(moved.Revision), "--json")
	if err != nil {
		t.Fatal(err)
	}
	var done focus.Item
	if err := json.Unmarshal([]byte(out), &done); err != nil || done.Lifecycle != attention.FocusDone {
		t.Fatalf("done = %q, %v", out, err)
	}
	out, err = runFocusCommand(t, "", "list", "--json")
	if err != nil || string(out) != "[]\n" {
		t.Fatalf("default list after done = %q, %v", out, err)
	}
	if _, err := runFocusCommand(t, "secret", "add", "--title", "No file"); err == nil || !strings.Contains(err.Error(), "--prompt-file") {
		t.Fatalf("missing prompt-file err = %v", err)
	}
}

func TestFocusCLIUnboundItemStaysVisibleButCannotLaunch(t *testing.T) {
	withFocusCLI(t, cliFocusResolver{})
	out, err := runFocusCommand(t, "saved\n", "add", "--title", "Unbound", "--prompt-file", "-")
	if err != nil {
		t.Fatal(err)
	}
	var item focus.Item
	_ = json.Unmarshal([]byte(out), &item)
	shown, err := runFocusCommand(t, "", "show", item.ID, "--json")
	if err != nil || !strings.Contains(shown, `"availability":"available"`) {
		t.Fatalf("show = %q, %v", shown, err)
	}
	if _, err := runFocusCommand(t, "", "launch", item.ID); err == nil || !strings.Contains(err.Error(), "no project target") {
		t.Fatalf("launch err = %v", err)
	}
}

func TestFocusCLIAutoBindsRegisteredCurrentWorkspace(t *testing.T) {
	ref := &attention.ProjectReference{PeerID: "peer-current", RegistrySlug: "current", DisplayName: "current"}
	withFocusCLI(t, cliFocusResolver{ref: ref, path: t.TempDir(), autoCurrent: true})
	out, err := runFocusCommand(t, "resume here\n", "add", "--title", "Current work", "--prompt-file", "-")
	if err != nil {
		t.Fatal(err)
	}
	var item focus.Item
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		t.Fatal(err)
	}
	if item.Project == nil || item.Project.PeerID != ref.PeerID || item.Project.RegistrySlug != ref.RegistrySlug {
		t.Fatalf("auto-bound project = %#v", item.Project)
	}
}

func TestFocusSuggestionCLIEndToEndAndStructuredErrors(t *testing.T) {
	ref := &attention.ProjectReference{PeerID: "peer-demo", RegistrySlug: "demo", DisplayName: "Demo"}
	projectPath := filepath.Join(t.TempDir(), "demo")
	withFocusCLI(t, cliFocusResolver{ref: ref, path: projectPath})
	prompt := "Investigate the separate cache issue.\n"
	reasonFile := filepath.Join(t.TempDir(), "reason.txt")
	if err := os.WriteFile(reasonFile, []byte("It is outside the accepted delivery"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := runFocusCommand(t, prompt, "suggest", "--title", "Cache issue", "--reason-file", reasonFile, "--prompt-file", "-", "--project", "demo", "--source-kind", "run", "--source-id", "run-1", "--idempotency-key", "run-1:cache")
	if err != nil {
		t.Fatal(err)
	}
	var created suggestion.Presented
	if err := json.Unmarshal([]byte(out), &created); err != nil || created.State != suggestion.StatePending || len(created.Actions) != 4 {
		t.Fatalf("suggest = %q, %v", out, err)
	}

	out, err = runFocusCommand(t, "", "suggestions", "--pending", "--json")
	if err != nil || !strings.Contains(out, created.ID) || !strings.Contains(out, `"reason":"It is outside the accepted delivery"`) {
		t.Fatalf("suggestions = %q, %v", out, err)
	}

	out, err = runFocusCommand(t, "", "suggestion", created.ID, "do-next", "--revision", fmtRevision(created.Revision), "--idempotency-key", "accept-1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result suggestion.ActionResult
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.Focus == nil || result.Focus.Lifecycle != attention.FocusToday || result.Launch == nil || result.Launch.Path != projectPath {
		t.Fatalf("action = %q, %v", out, err)
	}

	replayed, err := runFocusCommand(t, "", "suggestion", created.ID, "do-next", "--revision", fmtRevision(created.Revision), "--idempotency-key", "accept-1", "--json")
	if err != nil || replayed != out {
		t.Fatalf("replay = %q, %v; want %q", replayed, err, out)
	}

	if _, err := runFocusCommand(t, "", "suggestion", created.ID, "today", "--revision", fmtRevision(created.Revision), "--idempotency-key", "different"); err == nil || !strings.Contains(err.Error(), "stale:") {
		t.Fatalf("stale error = %v", err)
	}
	if _, err := runFocusCommand(t, "", "suggestion", "suggestion_missing", "today", "--revision", "1", "--idempotency-key", "missing"); err == nil || !strings.Contains(err.Error(), "missing:") {
		t.Fatalf("missing error = %v", err)
	}
}

func fmtRevision(v int64) string { return fmt.Sprintf("%d", v) }
