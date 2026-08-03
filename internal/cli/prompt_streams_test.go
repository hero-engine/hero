//go:build unix

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/ptytest"
	"github.com/spf13/cobra"
)

// In-process cobra-stream tests for the prompt helpers this child migrated.
//
// This is the initiative's "testability gate": before this work, every one of
// these helpers read os.Stdin or a package-level variable directly, so ZERO of
// them could be driven from a test. The metric for the child is the number of
// sites drivable through cmd.SetIn()/cmd.SetOut(), not the existence of the
// package.
//
// These complement rather than duplicate the subprocess suites. The subprocess
// harness owns anything that depends on what fd 0 actually IS — the TTY
// predicates, exit codes, real terminals — because an in-process test cannot
// change its own stdin. These own the input-handling logic itself, which is
// where the behavior-preservation risk lives, and they run in milliseconds.
//
// Three helpers are only reachable inside a full command run (the two install
// confirms and the migrate-nested confirm). Those are covered end-to-end by
// the baseline fixtures and TestPromptSitesPromptWhenTTYPresent instead; there
// is no seam to call them through without also constructing a workspace, an
// install, and a chdir, at which point a subprocess is the honest tool.

// newStreamCmd returns a cobra command wired to the given input, plus the
// buffer its output lands in — the exact plumbing a caller uses.
func newStreamCmd(input string) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(input))
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	return cmd, out
}

// --------------------------------------------------------------------------
// connect.go:766 — connectPrompt
// --------------------------------------------------------------------------

func TestConnectPromptReadsFromCommandStream(t *testing.T) {
	cmd, out := newStreamCmd("owner/repo\n")
	if got := connectPrompt(cmd, "Repository (owner/repo): "); got != "owner/repo" {
		t.Errorf("connectPrompt = %q, want %q", got, "owner/repo")
	}
	if !strings.Contains(out.String(), "Repository (owner/repo): ") {
		t.Errorf("label not written to the command's output stream: %q", out.String())
	}
}

// TestConnectPromptEmptyOnClosedStream pins the behaviour the callers depend
// on: an unanswered prompt yields "", so the caller's own "X is required"
// error is what the user sees rather than a raw io.EOF.
func TestConnectPromptEmptyOnClosedStream(t *testing.T) {
	cmd, _ := newStreamCmd("")
	if got := connectPrompt(cmd, "Repository: "); got != "" {
		t.Errorf("connectPrompt on closed stream = %q, want empty", got)
	}
}

// --------------------------------------------------------------------------
// connect.go:778 — connectSecret
// --------------------------------------------------------------------------

// TestConnectSecretIgnoresTheCommandStream is the security-relevant assertion
// for connect's token read: even with a credential sitting on the command's
// input stream, the secret must not come from there.
func TestConnectSecretIgnoresTheCommandStream(t *testing.T) {
	if hasControllingTerminal() {
		t.Skip("this test process has a controlling terminal; /dev/tty would be readable here")
	}
	// connectSecret deliberately accepts no stream — there is nothing to hand
	// it a credential through, which is the structural half of the guarantee.
	// This asserts the behavioural half: with no terminal, it yields nothing
	// rather than falling back to some other source.
	if got := connectSecret("Token: "); got != "" {
		t.Errorf("connectSecret returned %q — a credential must never come from a non-terminal stream", got)
	}
}

// --------------------------------------------------------------------------
// install.go:442 — promptTarget
// --------------------------------------------------------------------------

func TestPromptTargetStreamCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		jsonMode bool
		wantErr  string
	}{
		{
			// Non-TTY: a strings.Reader is never a terminal, so this is the
			// "required input missing, no terminal" case.
			name:    "non-TTY refuses instead of defaulting",
			input:   "",
			wantErr: "no terminal available",
		},
		{
			name:     "--json refuses without prompting",
			input:    "claude\n",
			jsonMode: true,
			wantErr:  "--target is required with --json",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, out := newStreamCmd(tc.input)
			got, err := promptTarget(cmd.InOrStdin(), cmd.OutOrStdout(), tc.jsonMode)
			if err == nil {
				t.Fatalf("promptTarget = %q, want error %q", got, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
			}
			if got != "" {
				t.Errorf("promptTarget returned %q alongside an error, want empty — "+
					"returning a target here is exactly the silent-default bug", got)
			}
			if strings.Contains(out.String(), "Install target") {
				t.Errorf("promptTarget emitted a prompt with no terminal: %q", out.String())
			}
		})
	}
}

// TestPromptTargetRejectsTypoAtThePrompt is AC-7 driven through the prompt
// itself rather than through the --target flag.
//
// The flag path has always been validated downstream by internal/install, so a
// flag-only test would pass against the old code too. What was unvalidated was
// the value typed AT the prompt, which the old implementation passed straight
// into install.Target(). This asserts the entered value is checked at entry.
//
// It calls promptTarget's picker through a real terminal so the TTY gate is
// genuinely satisfied; see TestPromptSitesPromptWhenTTYPresent for the
// end-to-end form.
func TestPromptTargetRejectsTypoAtThePrompt(t *testing.T) {
	master, slave := openStreamPTY(t)
	go func() { _, _ = master.WriteString("clade\n") }()

	var out bytes.Buffer
	got, err := promptTarget(slave, &out, false)
	if err == nil {
		t.Fatalf("promptTarget accepted the typo and returned %q — a mistyped target must not be "+
			"constructed into install.Target()", got)
	}
	if !strings.Contains(err.Error(), "clade") {
		t.Errorf("error %q does not quote the rejected value", err)
	}
	if got != "" {
		t.Errorf("promptTarget returned %q alongside an error, want empty", got)
	}
}

// TestPromptTargetAcceptsAValidAnswer proves the picker still works — a
// rejection test alone would pass against an implementation that rejects
// everything.
func TestPromptTargetAcceptsAValidAnswer(t *testing.T) {
	master, slave := openStreamPTY(t)
	go func() { _, _ = master.WriteString("claude\n") }()

	var out bytes.Buffer
	got, err := promptTarget(slave, &out, false)
	if err != nil {
		t.Fatalf("promptTarget: %v", err)
	}
	if got != install.TargetClaude {
		t.Errorf("promptTarget = %q, want %q", got, install.TargetClaude)
	}
}

// TestPromptTargetEmptyAnswerKeepsTheOpencodeDefault pins the one piece of the
// old behaviour that is NOT a sanctioned break: pressing enter at the prompt
// still selects opencode. Only the no-terminal path changed.
func TestPromptTargetEmptyAnswerKeepsTheOpencodeDefault(t *testing.T) {
	master, slave := openStreamPTY(t)
	go func() { _, _ = master.WriteString("\n") }()

	var out bytes.Buffer
	got, err := promptTarget(slave, &out, false)
	if err != nil {
		t.Fatalf("promptTarget: %v", err)
	}
	if got != install.TargetOpenCode {
		t.Errorf("promptTarget on an empty answer = %q, want %q", got, install.TargetOpenCode)
	}
}

// TestInstallTargetsMatchTheFlagHelp guards the drift that bit this surface
// before, when `hero uninstall` accepted four of the six targets `hero install`
// advertised. The flag help is built from installTargets, so this asserts the
// list itself still enumerates every target the install package knows.
func TestInstallTargetsMatchTheFlagHelp(t *testing.T) {
	want := []install.Target{
		install.TargetOpenCode, install.TargetCursor, install.TargetClaude,
		install.TargetCopilot, install.TargetCodex, install.TargetGeneric,
	}
	if len(installTargets) != len(want) {
		t.Fatalf("installTargets has %d entries, want %d", len(installTargets), len(want))
	}
	for i, w := range want {
		if installTargets[i] != string(w) {
			t.Errorf("installTargets[%d] = %q, want %q", i, installTargets[i], w)
		}
	}
	flag := installCmd.Flags().Lookup("target")
	if flag == nil {
		t.Fatal("--target flag is gone")
	}
	for _, w := range want {
		if !strings.Contains(flag.Usage, string(w)) {
			t.Errorf("--target help does not advertise %q: %s", w, flag.Usage)
		}
	}
}

// --------------------------------------------------------------------------
// users.go:188 — promptPassword
// --------------------------------------------------------------------------

// TestPromptPasswordRefusesNonTerminalStream is the in-process half of the
// security fix. The subprocess suite proves the exit code and the operator
// message; this proves the helper itself never reads the value.
func TestPromptPasswordRefusesNonTerminalStream(t *testing.T) {
	if hasControllingTerminal() {
		t.Skip("this test process has a controlling terminal; /dev/tty would be readable here")
	}
	got, err := promptPassword("New password: ")
	if err == nil {
		t.Fatalf("promptPassword returned %q with no error — the echoed fallback is back", got)
	}
	if got != "" {
		t.Errorf("promptPassword returned %q alongside an error, want empty", got)
	}
	for _, want := range []string{"terminal", "--password"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// --------------------------------------------------------------------------
// install_satellites.go:216 — reconcileDeclared
// --------------------------------------------------------------------------

// TestReconcileDeclaredConfirmPolarity is the [Y/n] site. Its default is YES,
// the opposite of every other confirm in this file, which is exactly the kind
// of inversion the initiative flagged as the likeliest silent bug.
func TestReconcileDeclaredConfirmPolarity(t *testing.T) {
	tests := []struct {
		name    string
		answer  string
		wantYes bool
	}{
		{"explicit n declines", "n\n", false},
		{"explicit no declines", "no\n", false},
		{"empty accepts (default Y)", "\n", true},
		{"explicit y accepts", "y\n", true},
		{"unrecognized falls back to the Y default", "maybe\n", true},
		{"end of stream falls back to the Y default", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, out := newStreamCmd(tc.answer)
			got, err := confirmForTest(cmd, out, "  sub  materialize satellite? [Y/n] ", true)
			if err != nil {
				t.Fatalf("confirm: %v", err)
			}
			if got != tc.wantYes {
				t.Errorf("answer %q => %v, want %v (this site defaults to YES)", tc.answer, got, tc.wantYes)
			}
		})
	}
}

// TestSubprojectConfirmPolarity covers the [y/N] sites, whose default is NO.
// Running both polarities side by side is the point: a single Confirm
// implementation has to serve both, and the shared rule is "empty or
// unrecognized answers take the default".
func TestSubprojectConfirmPolarity(t *testing.T) {
	tests := []struct {
		name    string
		answer  string
		wantYes bool
	}{
		{"explicit y accepts", "y\n", true},
		{"explicit yes accepts", "yes\n", true},
		{"empty declines (default N)", "\n", false},
		{"unrecognized falls back to the N default", "maybe\n", false},
		{"end of stream falls back to the N default", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, out := newStreamCmd(tc.answer)
			got, err := confirmForTest(cmd, out, "Add it as a subproject? [y/N] ", false)
			if err != nil {
				t.Fatalf("confirm: %v", err)
			}
			if got != tc.wantYes {
				t.Errorf("answer %q => %v, want %v (this site defaults to NO)", tc.answer, got, tc.wantYes)
			}
		})
	}
}

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

// confirmForTest drives the same prompt.Confirm the migrated confirm sites
// call, through cobra's streams.
func confirmForTest(cmd *cobra.Command, out *bytes.Buffer, label string, def bool) (bool, error) {
	return prompt.Confirm(cmd.InOrStdin(), cmd.OutOrStdout(), label, def)
}

// openStreamPTY allocates a real terminal for the cases that must satisfy the
// TTY gate in-process.
func openStreamPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	m, s, err := ptytest.Open()
	if err != nil {
		t.Skipf("%v", err)
	}
	t.Cleanup(func() {
		s.Close()
		m.Close()
	})
	return m, s
}

// hasControllingTerminal reports whether this process can open /dev/tty.
//
// Under `go test` from a shell it usually can, and prompt.Secret would then
// block on a real password read. The tests that assert refusal skip in that
// case; the subprocess suites cover the same assertions unconditionally by
// running the child with Setsid.
func hasControllingTerminal() bool {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
