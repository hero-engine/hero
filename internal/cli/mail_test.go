package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/mail"
)

func TestMailCLIJSONCommandsAndErrors(t *testing.T) {
	original, _ := os.Getwd()
	defer os.Chdir(original)
	state := t.TempDir()
	a := t.TempDir()
	b := t.TempDir()
	writeCLIProject := func(root, id, alias, target string) {
		t.Helper()
		_ = os.MkdirAll(filepath.Join(root, ".hero"), 0o700)
		repos := "{}"
		if alias != "" {
			encoded, _ := json.Marshal(map[string]string{alias: target})
			repos = string(encoded)
		}
		cfg := `{"folder":".hero","peer_id":"` + id + `","repos":` + repos + `}`
		if err := os.WriteFile(filepath.Join(root, ".hero", "hero.json"), []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		manifest := "schema: 1\ncontracts_version: 1\nrepo:\n  peer_id: " + id + "\n  name: " + id + "\ngenerated_at: 2026-07-22T18:00:00Z\n"
		if err := os.WriteFile(filepath.Join(root, ".hero", "peer-manifest.yaml"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeCLIProject(a, "peer_a", "b", b)
	writeCLIProject(b, "peer_b", "a", a)
	mailStateRootOverride = state
	defer func() { mailStateRootOverride = "" }()
	run := func(root string, args []string, stdin string) (string, error) {
		t.Helper()
		if err := os.Chdir(root); err != nil {
			t.Fatal(err)
		}
		cmd := newMailCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetIn(bytes.NewBufferString(stdin))
		cmd.SetArgs(args)
		err := cmd.Execute()
		return out.String(), err
	}
	out, err := run(a, []string{"send", "b", "--subject", "Hello", "--body-file", "-", "--idempotency-key", "cli-key", "--json"}, "secret body")
	if err != nil {
		t.Fatal(err)
	}
	var delivery attention.MailDelivery
	if err := json.Unmarshal([]byte(out), &delivery); err != nil {
		t.Fatalf("send JSON: %v: %s", err, out)
	}
	out, err = run(b, []string{"inbox", "--unread", "--json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	var inbox []json.RawMessage
	if json.Unmarshal([]byte(out), &inbox) != nil || len(inbox) != 1 {
		t.Fatalf("inbox JSON: %s", out)
	}
	if err := os.Chdir(b); err != nil {
		t.Fatal(err)
	}
	statusOut, err := runCmd("status", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var statusPayload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(statusOut), &statusPayload); err != nil || statusPayload["specs"] == nil || statusPayload["mail"] == nil {
		t.Fatalf("status JSON missing existing/mail fields: %s, %v", statusOut, err)
	}
	var summary mail.UnreadSummary
	if err := json.Unmarshal(statusPayload["mail"], &summary); err != nil || summary.Count != 1 || len(summary.Items) != 1 {
		t.Fatalf("status mail summary = %#v, %v", summary, err)
	}
	statusOut, err = runCmd("status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statusOut, "Project Mail — unread (1):") ||
		!strings.Contains(statusOut, "Hello") {
		t.Fatalf("human status missing unread Project Mail summary: %s", statusOut)
	}
	resumeOut, err := runCmd("resume", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var resumePayload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(resumeOut), &resumePayload); err != nil || resumePayload["mail"] == nil {
		t.Fatalf("resume JSON missing mail: %s, %v", resumeOut, err)
	}
	out, err = run(b, []string{"show", delivery.MessageID, "--no-mark-read", "--json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("show JSON: %s", out)
	}
	out, err = run(b, []string{"ack", delivery.MessageID, "--note", "seen", "--json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("ack JSON: %s", out)
	}
	var acknowledged mail.ActionResult
	if err := json.Unmarshal([]byte(out), &acknowledged); err != nil {
		t.Fatal(err)
	}
	out, err = run(b, []string{"dismiss", delivery.MessageID, "--revision", fmt.Sprint(acknowledged.Receipt.Revision), "--idempotency-key", "dismiss-1", "--json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	var dismissed mail.ActionResult
	if err := json.Unmarshal([]byte(out), &dismissed); err != nil || dismissed.Receipt.DismissedAt == "" {
		t.Fatalf("dismiss JSON: %s, %v", out, err)
	}
	out, err = run(b, []string{"promote", delivery.MessageID, "--revision", fmt.Sprint(dismissed.Receipt.Revision), "--idempotency-key", "promote-1", "--type", "intake", "--json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	var promoted mail.ActionResult
	if err := json.Unmarshal([]byte(out), &promoted); err != nil || promoted.Artifact == nil {
		t.Fatalf("promote JSON: %s, %v", out, err)
	}
	out, err = run(b, []string{"add-to-today", delivery.MessageID, "--revision", fmt.Sprint(promoted.Receipt.Revision), "--idempotency-key", "today-1", "--json"}, "")
	if err != nil {
		t.Fatal(err)
	}
	var today mail.ActionResult
	if err := json.Unmarshal([]byte(out), &today); err != nil || today.FocusItemID == "" {
		t.Fatalf("today JSON: %s, %v", out, err)
	}
	out, err = run(b, []string{"reply", delivery.MessageID, "--body-file", "-", "--json"}, "answer")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("reply JSON: %s", out)
	}
	out, err = run(b, []string{"show", "mail_missing", "--json"}, "")
	if err == nil {
		t.Fatal("JSON error must retain nonzero command result")
	}
	var coded mailError
	firstLine, _, _ := strings.Cut(out, "\n")
	if json.Unmarshal([]byte(firstLine), &coded) != nil || coded.Code != "missing" {
		t.Fatalf("structured error: %s", out)
	}
}

func TestMailCommandTreeFeedsDynamicShellCompletion(t *testing.T) {
	cmd := newMailCommand()
	got := map[string]bool{}
	for _, child := range cmd.Commands() {
		got[child.Name()] = true
	}
	for _, name := range []string{"send", "inbox", "show", "reply", "ack", "read", "dismiss", "promote", "add-to-today"} {
		if !got[name] {
			t.Errorf("dynamic completion tree missing mail %s", name)
		}
	}
}
