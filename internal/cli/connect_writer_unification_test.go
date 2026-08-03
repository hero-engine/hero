package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/ptytest"
	"github.com/spf13/cobra"
)

// Regression tests for `connect-writer-unification`.
//
// The bug these cover is silent wrong behaviour, not a crash: `hero connect`
// had two write paths that disagreed about what a connection is, and the
// interactive one wrote no `capabilities` at all. With none written,
// EffectiveCapabilities falls back to the provider's legacy `tracker`, and
// ResolveCodeHostConnection rejects the connection.
//
// So these assert on the RESOLVER'S VERDICT and on persisted state, never on
// what got printed. A test that asserted the JSON looked right would have
// passed against the broken code: the JSON was never malformed, the capability
// inference was wrong. Each test names the mutation it dies to.

// connectTestState installs a clean, known set of the package-level connect
// flags and restores the defaults afterwards.
//
// These are process globals shared with every other test in the package, and
// several of those leave them set. Anything that reads them must pin them.
func connectTestState(t *testing.T) {
	t.Helper()
	reset := func() {
		connectList = false
		connectRemove = ""
		connectGlobal = false
		connectProject = ""
		connectIntegrationID = ""
		connectRole = "delivery"
		connectBaseURL = ""
		connectUserEmail = ""
		connectTokenStdin = false
		connectLocalOnly = false
		connectJSON = false
		connectNoVerify = false
	}
	reset()
	t.Cleanup(reset)
}

// newConnectWorkspace returns an isolated workspace root with HOME pointed
// somewhere harmless, so credential loading never touches the developer's.
func newConnectWorkspace(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "work")
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	t.Setenv("HOME", filepath.Join(base, "home"))
	return root
}

// newRoutingCmd builds a command wired exactly like the real `hero connect`,
// so cobra — not the test — decides which flags count as Changed.
func newRoutingCmd(stdin string) (*cobra.Command, *strings.Reader, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "connect [type]", RunE: runConnect, Args: cobra.MaximumNArgs(1), SilenceUsage: true, SilenceErrors: true}
	cmd.Flags().BoolVar(&connectList, "list", false, "list saved connections")
	cmd.Flags().StringVar(&connectRemove, "remove", "", "remove connection for the given tracker type")
	cmd.Flags().BoolVar(&connectGlobal, "global", false, "save token globally")
	cmd.Flags().StringVar(&connectProject, "project", "", "project identifier")
	addIntegrationConnectFlags(cmd)

	in := strings.NewReader(stdin)
	out := &bytes.Buffer{}
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(out)
	return cmd, in, out
}

// TestRoleFlagRoutesToTheFlagDrivenPath is the regression test for the routing
// predicate at connect.go's runConnect.
//
// `--role` was missing from the trigger set, so `hero connect github --role
// code-host` landed on the interactive path — the one that hardcoded
// `delivery` and ignored the flag entirely.
//
// The assertion is which path RAN, observed as state rather than as printed
// text: the interactive path reads the repository off the command's input
// stream, and the flag-driven path rejects the invocation before reading
// anything. So an unconsumed stream is proof the flag-driven path handled it.
//
// Dies to: deleting `|| cmd.Flags().Changed("role")` from the predicate.
func TestRoleFlagRoutesToTheFlagDrivenPath(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)
	t.Chdir(root)

	cmd, in, out := newRoutingCmd("owner/repo\n")
	cmd.SetArgs([]string{"github", "--role", "code-host"})
	err := cmd.Execute()

	if err == nil {
		t.Fatal("expected the flag-driven path to reject a --role connect with no --project")
	}
	if !strings.Contains(err.Error(), "--project is required for non-interactive connect") {
		t.Errorf("error = %v, want the flag-driven path's --project requirement", err)
	}
	remaining, _ := io.ReadAll(in)
	if string(remaining) != "owner/repo\n" {
		t.Errorf("input stream was consumed (%q left) — the interactive path handled a --role invocation", remaining)
	}
	if strings.Contains(out.String(), "Repository (owner/repo): ") {
		t.Errorf("interactive prompt reached on a --role invocation:\n%s", out.String())
	}
}

// TestRoleFlagConnectIsAcceptedByTheCodeHostResolver is AC-2 and AC-4 end to
// end through runConnect: the flag route must produce a connection that
// ResolveCodeHostConnection (internal/config/integrations.go) accepts.
//
// Dies to: removing the CapabilityCodeHost inference in writeConnection, which
// is exactly the state the interactive writer used to leave behind.
func TestRoleFlagConnectIsAcceptedByTheCodeHostResolver(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)
	t.Chdir(root)

	cmd, _, _ := newRoutingCmd("code-host-token\n")
	cmd.SetArgs([]string{"github", "--role", "code-host", "--project", "hero-engine/hero",
		"--token-stdin", "--local-only", "--no-verify"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("connect: %v", err)
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	host, err := cfg.ResolveCodeHostConnection("")
	if err != nil {
		t.Fatalf("code-host resolver rejected the connection --role code-host just created: %v", err)
	}
	if host.ID != "github-hero-engine-hero" {
		t.Errorf("resolved connection id = %q", host.ID)
	}
	if cfg.Integrations.Default != "" {
		t.Errorf("a code-host connection claimed the default: %q", cfg.Integrations.Default)
	}
}

// TestInteractiveConnectIsAcceptedByTheCodeHostResolver is AC-3 and AC-4 for
// the interactive path — the one that used to write no `capabilities` at all.
//
// The role arrives the way connectPromptRole delivers it on a non-terminal
// stream (the flag value); the credential is seeded, because prompt.Secret
// reads /dev/tty by construction and no test may weaken that.
//
// Dies to: reverting runConnectInteractive to the old writer's behaviour —
// hardcoding role `delivery`, or dropping the capability write.
func TestInteractiveConnectIsAcceptedByTheCodeHostResolver(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)
	connectRole = "code-host"
	connectNoVerify = true
	connectLocalOnly = true

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	known := map[string]string{"provider": "github", "project": "hero-engine/hero", "token": "interactive-token"}
	if err := runConnectInteractive(cmd, root, config.Credentials{}, "github", known); err != nil {
		t.Fatalf("interactive connect: %v", err)
	}

	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := cfg.ResolveCodeHostConnection(""); err != nil {
		t.Fatalf("code-host resolver rejected an interactively created code-host connection: %v", err)
	}
	if cfg.Integrations.Default != "" {
		t.Errorf("interactive code-host connect claimed the default: %q", cfg.Integrations.Default)
	}
}

// TestBothPathsPersistIdenticalState is AC-7 and AC-5: equivalent inputs
// through the two paths must leave the workspace in the same state, including
// how the `default` flag resolved.
//
// It compares the persisted files themselves rather than a projection of them,
// so a divergence in any field — role, capabilities, default, settings —
// fails it.
//
// Dies to: any per-path special case in writeConnection, and to the old
// interactive writer's unconditional `"default": id`.
func TestBothPathsPersistIdenticalState(t *testing.T) {
	cases := []struct {
		provider  string
		role      string
		project   string
		baseURL   string
		userEmail string
	}{
		{provider: "github", role: "delivery", project: "owner/repo"},
		{provider: "github", role: "code-host", project: "owner/repo"},
		{provider: "jira", role: "delivery",
			project: "PROJ", baseURL: "https://jira.example", userEmail: "dev@example.com"},
		{provider: "linear", role: "delivery", project: "ENG"},
		{provider: "gitlab", role: "code-host",
			project: "group/project", baseURL: "https://gitlab.example"},
		{provider: "confluence", role: "delivery",
			project: "ENG", baseURL: "https://wiki.example", userEmail: "dev@example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.provider+"/"+tc.role, func(t *testing.T) {
			const token = "PAIRED-PATH-TOKEN"

			connectTestState(t)
			interactiveRoot := newConnectWorkspace(t)
			connectRole = tc.role
			connectNoVerify = true
			connectLocalOnly = true
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(""))
			cmd.SetOut(&bytes.Buffer{})
			known := map[string]string{"provider": tc.provider, "project": tc.project, "base_url": tc.baseURL, "user_email": tc.userEmail, "token": token}
			if err := runConnectInteractive(cmd, interactiveRoot, config.Credentials{}, tc.provider, known); err != nil {
				t.Fatalf("interactive connect: %v", err)
			}

			connectTestState(t)
			flagRoot := newConnectWorkspace(t)
			connectRole = tc.role
			connectNoVerify = true
			connectLocalOnly = true
			connectTokenStdin = true
			connectProject = tc.project
			connectBaseURL = tc.baseURL
			connectUserEmail = tc.userEmail
			flagCmd := &cobra.Command{}
			flagCmd.SetIn(strings.NewReader(token + "\n"))
			flagCmd.SetOut(&bytes.Buffer{})
			if err := runConnectNonInteractive(flagCmd, flagRoot, config.Credentials{}, tc.provider); err != nil {
				t.Fatalf("flag-driven connect: %v", err)
			}

			for _, name := range []string{config.ConfigFileName, config.LocalConfigFileName} {
				interactive := readIfExists(t, filepath.Join(interactiveRoot, ".hero", name))
				flagged := readIfExists(t, filepath.Join(flagRoot, ".hero", name))
				if interactive != flagged {
					t.Errorf("%s diverges between the two paths\n--- interactive ---\n%s\n--- flag-driven ---\n%s",
						name, interactive, flagged)
				}
			}
		})
	}
}

func readIfExists(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestWriteConnectionRejectsRoleWithNoCapability is AC-6. An unmapped role
// must fail rather than persist an empty capability set — an empty set is
// precisely what re-opens the legacy-`tracker` fallback this work closed.
//
// Dies to: dropping the ValidateProviderRole check from writeConnection.
func TestWriteConnectionRejectsRoleWithNoCapability(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)
	connectLocalOnly = true

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := writeConnection(cmd, root, config.Credentials{}, connectionInput{
		provider: "github", id: "github-x", project: "owner/repo",
		token: "t", role: "archivist", verification: "not-checked",
	})
	if err == nil {
		t.Fatal("an unmapped role was persisted")
	}
	if !strings.Contains(err.Error(), `unknown integration role "archivist"`) {
		t.Errorf("error = %v", err)
	}
	for _, name := range []string{config.ConfigFileName, config.LocalConfigFileName} {
		if got := readIfExists(t, filepath.Join(root, ".hero", name)); got != "" {
			t.Errorf("%s was written for a rejected role:\n%s", name, got)
		}
	}
}

// TestInteractiveConnectNeverPromptsUnderJSON is AC-9. Under --json, stdout is
// a machine-readable contract, so the path whose every value comes from a
// prompt must refuse instead of asking.
//
// The assertions are that nothing was written to the output stream and nothing
// was read from the input stream — not that some particular sentence appeared.
func TestInteractiveConnectNeverPromptsUnderJSON(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)
	connectJSON = true

	in := strings.NewReader("owner/repo\n")
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetIn(in)
	cmd.SetOut(out)

	err := runConnectInteractive(cmd, root, config.Credentials{}, "github", map[string]string{"provider": "github"})
	if err == nil {
		t.Fatal("interactive connect proceeded under --json")
	}
	if out.Len() != 0 {
		t.Errorf("--json run wrote %q to stdout; it must stay a machine-readable stream", out.String())
	}
	if remaining, _ := io.ReadAll(in); string(remaining) != "owner/repo\n" {
		t.Errorf("--json run consumed input (%q left) — it prompted", remaining)
	}
}

// TestInteractiveConnectFailsFastOnAnUnansweredStream is AC-8: a non-terminal
// stream with the required value missing exits with the pre-existing message
// and never blocks. The `go test` timeout is what proves "never blocks"; the
// error text is the compatibility half, and the golden fixtures at
// testdata/prompt_baseline/connect_github_repo_prompt.{pipe,closed}.txt hold
// the byte-level version of the same guarantee.
func TestInteractiveConnectFailsFastOnAnUnansweredStream(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	err := runConnectInteractive(cmd, root, config.Credentials{}, "github", map[string]string{"provider": "github"})
	if err == nil || err.Error() != "repository is required" {
		t.Fatalf("error = %v, want %q", err, "repository is required")
	}
}

func TestInteractiveConnectFailsBeforeALiveNonTTYPipeCanBeRead(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })

	cmd := &cobra.Command{}
	cmd.SetIn(reader)
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	done := make(chan error, 1)
	go func() {
		done <- runConnectInteractive(cmd, root, config.Credentials{}, "github", map[string]string{"provider": "github"})
	}()

	select {
	case err := <-done:
		if err == nil || err.Error() != "repository is required" {
			t.Fatalf("error = %v, want repository missing error", err)
		}
	case <-time.After(time.Second):
		t.Fatal("interactive collector read the live non-TTY pipe")
	}
	if out.Len() != 0 {
		t.Errorf("non-TTY collector printed a prompt/banner: %q", out.String())
	}
}

// TestConnectHasExactlyOneCommittedWriter is AC-1, guarded structurally.
//
// The defect was two writers drifting apart, and nothing about a behavioural
// test stops a third from being added next to them. Committed-integration
// patching is the act of persisting a connection, so it must appear once.
func TestConnectHasExactlyOneCommittedWriter(t *testing.T) {
	src, err := os.ReadFile("connect.go")
	if err != nil {
		t.Fatalf("read connect.go: %v", err)
	}
	if n := strings.Count(string(src), "config.PatchCommittedIntegrations("); n != 1 {
		t.Errorf("connect.go persists committed integrations from %d places, want 1 (writeConnection)", n)
	}
}

// TestConnectRoleOptionsOfferOnlyServiceableRoles pins that the interactive
// role picker cannot offer a role the provider would then be rejected for.
func TestConnectRoleOptionsOfferOnlyServiceableRoles(t *testing.T) {
	want := map[string][]string{
		"github":     {"delivery", "roadmap", "code-host"},
		"gitlab":     {"delivery", "roadmap", "code-host"},
		"jira":       {"delivery", "roadmap"},
		"linear":     {"delivery", "roadmap"},
		"confluence": {"docs"},
	}
	for provider, expected := range want {
		got := connectRoleOptions(provider)
		if strings.Join(got, ",") != strings.Join(expected, ",") {
			t.Errorf("connectRoleOptions(%q) = %v, want %v", provider, got, expected)
		}
		for _, role := range got {
			if _, err := config.ValidateProviderRole(provider, role); err != nil {
				t.Errorf("offered role %q for %q that the writer rejects: %v", role, provider, err)
			}
		}
	}
}

// TestConnectPromptRoleReadsFromARealTerminal drives the role picker through a
// genuine pty, because prompt.IsInputTTY is the real predicate and there is
// deliberately no way to fake it.
func TestConnectPromptRoleReadsFromARealTerminal(t *testing.T) {
	connectTestState(t)
	master, slave, err := ptytest.Open()
	if err != nil {
		t.Skipf("%v", err)
	}
	defer master.Close()
	defer slave.Close()

	go func() { _, _ = io.WriteString(master, "code-host\n") }()

	cmd := &cobra.Command{}
	cmd.SetIn(slave)
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	role, err := connectPromptRole(cmd, "github")
	if err != nil {
		t.Fatalf("connectPromptRole: %v", err)
	}
	if role != "code-host" {
		t.Errorf("role = %q, want %q", role, "code-host")
	}
	if !strings.Contains(out.String(), "delivery|roadmap|code-host") {
		t.Errorf("picker did not offer github's serviceable roles: %q", out.String())
	}
}

// TestConnectPromptRoleStaysSilentOnANonTerminal is the other half: a piped or
// closed stdin must not grow a prompt it never had, which is what keeps every
// recorded fixture in testdata/prompt_baseline byte-identical.
func TestConnectPromptRoleStaysSilentOnANonTerminal(t *testing.T) {
	connectTestState(t)
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("code-host\n"))
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	role, err := connectPromptRole(cmd, "github")
	if err != nil {
		t.Fatalf("connectPromptRole: %v", err)
	}
	if role != "delivery" {
		t.Errorf("role = %q, want the flag default %q", role, "delivery")
	}
	if out.Len() != 0 {
		t.Errorf("role picker wrote %q to a non-terminal run", out.String())
	}
}

// TestConnectFailsWhenHeroJSONCannotBeWritten covers the behaviour change the
// unification made to a failed `hero.json` write: the old interactive path
// printed `warning: could not update hero.json: …` to stderr, reported
// `Connected.`, and exited 0. It now returns the error.
//
// The `--global` shape is what makes this test sharp. The credential goes to
// `$HOME`, which stays writable, so only the committed write fails — and the
// two assertions can then be independent:
//
//   - a non-zero exit, which is the behaviour change itself, and
//   - an absent credential, which is the third-order effect of the write
//     ordering. `writeConnection` patches `hero.json` before it stores the
//     token; the old path stored the token first and patched second, so the
//     same failure used to leave a credential on disk for a connection that
//     was never recorded.
//
// Dies to: restoring warn-and-continue around PatchCommittedIntegrations, and
// to swapping the order of the committed patch and the credential write.
func TestConnectFailsWhenHeroJSONCannotBeWritten(t *testing.T) {
	connectTestState(t)
	root := newConnectWorkspace(t)
	t.Chdir(root)

	// Read-only .hero makes the committed patch's temp file fail. Perms are
	// restored on cleanup so t.TempDir can remove the tree.
	heroDir := filepath.Join(root, ".hero")
	if err := os.Chmod(heroDir, 0o555); err != nil {
		t.Fatalf("chmod .hero: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(heroDir, 0o755) })

	cmd, _, _ := newRoutingCmd("global-token\n")
	cmd.SetArgs([]string{"github", "--role", "code-host", "--project", "owner/repo",
		"--token-stdin", "--global", "--no-verify"})
	err := cmd.Execute()

	if err == nil {
		t.Fatal("a failed hero.json write reported success; it must be an error, not a warning")
	}
	if _, statErr := os.Stat(config.CredentialsPath()); !os.IsNotExist(statErr) {
		t.Errorf("a credential was stored for a connection that was never recorded (%s): %v",
			config.CredentialsPath(), statErr)
	}
	if err := os.Chmod(heroDir, 0o755); err != nil {
		t.Fatalf("restore .hero perms: %v", err)
	}
	if got := readIfExists(t, filepath.Join(heroDir, config.ConfigFileName)); got != "" {
		t.Errorf("%s was written after its own write failed:\n%s", config.ConfigFileName, got)
	}
	if got := readIfExists(t, filepath.Join(heroDir, config.LocalConfigFileName)); got != "" {
		t.Errorf("%s was written after the committed write failed:\n%s", config.LocalConfigFileName, got)
	}
}
