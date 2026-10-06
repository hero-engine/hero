package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

func TestNonInteractiveConnectFailedVerificationRetainsPriorCredential(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	connectIntegrationID = "jira-delivery"
	connectProject = "MORPH"
	connectBaseURL = "https://jira.example"
	connectUserEmail = "dev@example.com"
	connectRole = "delivery"
	connectTokenStdin = true
	connectLocalOnly = false
	connectGlobal = false
	connectJSON = true
	connectNoVerify = true

	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("prior-token\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := runConnectNonInteractive(cmd, root, config.Credentials{}, "jira"); err != nil {
		t.Fatal(err)
	}
	committedPath := filepath.Join(root, ".hero", config.ConfigFileName)
	localPath := filepath.Join(root, ".hero", config.LocalConfigFileName)
	committedBefore, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatal(err)
	}
	localBefore, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}

	originalProvider := connectProviders["jira"]
	defer func() { connectProviders["jira"] = originalProvider }()
	connectProviders["jira"] = connectProvider{
		intro: originalProvider.intro,
		name:  originalProvider.name,
		verify: func(map[string]string) error {
			return errors.New("HTTP 401 unauthorized")
		},
	}
	connectNoVerify = false
	cmd = &cobra.Command{}
	cmd.SetIn(strings.NewReader("replacement-token\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := runConnectNonInteractive(cmd, root, config.Credentials{}, "jira"); err == nil {
		t.Fatal("expected replacement verification to fail")
	}

	committedAfter, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatal(err)
	}
	localAfter, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(committedBefore, committedAfter) {
		t.Fatal("failed verification changed committed integration state")
	}
	if !bytes.Equal(localBefore, localAfter) {
		t.Fatal("failed verification replaced the prior credential")
	}
	if !bytes.Contains(localAfter, []byte("prior-token")) || bytes.Contains(localAfter, []byte("replacement-token")) {
		t.Fatal("prior credential was not retained exactly")
	}
}

func TestNonInteractiveConnectSplitsLayersAndRedactsStatus(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hero"), 0755)
	canary := "CLI-CONNECT-CANARY"
	connectIntegrationID = "jira-delivery"
	connectProject = "MORPH"
	connectBaseURL = "https://jira.example"
	connectUserEmail = "dev@example.com"
	connectRole = "delivery"
	connectTokenStdin = true
	connectLocalOnly = false
	connectGlobal = false
	connectJSON = true
	connectNoVerify = true
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader(canary + "\n"))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runConnectNonInteractive(cmd, root, config.Credentials{}, "jira"); err != nil {
		t.Fatal(err)
	}
	shared, _ := os.ReadFile(filepath.Join(root, ".hero", config.ConfigFileName))
	local, _ := os.ReadFile(filepath.Join(root, ".hero", config.LocalConfigFileName))
	if strings.Contains(string(shared), canary) {
		t.Fatal("secret leaked to committed config")
	}
	if !strings.Contains(string(local), canary) {
		t.Fatal("local token missing")
	}
	if strings.Contains(out.String(), canary) {
		t.Fatal("JSON result leaked token")
	}
	connectJSON = true
	out.Reset()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(root)
	if err := runConnectList(&out, root, config.Credentials{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), canary) {
		t.Fatal("status leaked token")
	}
}

func TestConnectAliasIntegrationFlagsEquivalent(t *testing.T) {
	for _, name := range []string{"integration-id", "role", "base-url", "user-email", "token-stdin", "local-only", "json", "no-verify"} {
		if connectCmd.Flags().Lookup(name) == nil || connectAliasCmd.Flags().Lookup(name) == nil {
			t.Fatalf("flag %s missing from alias", name)
		}
	}
}

func TestNonInteractiveConnectAllProvidersLocalOnly(t *testing.T) {
	for _, provider := range []string{"github", "jira", "linear", "gitlab", "aha", "confluence"} {
		t.Run(provider, func(t *testing.T) {
			root := t.TempDir()
			os.MkdirAll(filepath.Join(root, ".hero"), 0755)
			connectIntegrationID = provider + "-personal"
			connectProject = "PROJECT"
			connectBaseURL = "https://service.invalid"
			connectUserEmail = ""
			if provider == "jira" || provider == "confluence" {
				connectUserEmail = "dev@example.invalid"
			}
			connectRole = "delivery"
			connectTokenStdin = true
			connectLocalOnly = true
			connectGlobal = false
			connectJSON = true
			connectNoVerify = true
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader("provider-canary\n"))
			cmd.SetOut(&bytes.Buffer{})
			if err := runConnectNonInteractive(cmd, root, config.Credentials{}, provider); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Integrations == nil || cfg.Integrations.Connections[connectIntegrationID].Provider != provider {
				t.Fatalf("missing %s integration", provider)
			}
		})
	}
}

// AC-4 (sept-review-cleanup): connecting aha states the broker-only limit.
func TestNonInteractiveConnectAhaStatesBrokerOnlyLimit(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hero"), 0755)
	connectIntegrationID = "aha-roadmap"
	connectProject = "PRODUCT"
	connectBaseURL = "https://acme.aha.io"
	connectUserEmail = ""
	connectRole = "delivery"
	connectTokenStdin = true
	connectLocalOnly = true
	connectGlobal = false
	connectJSON = false
	connectNoVerify = true
	defer func() { connectJSON = false }()
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("provider-canary\n"))
	cmd.SetOut(&out)
	if err := runConnectNonInteractive(cmd, root, config.Credentials{}, "aha"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "raw tracker requests (hero_tracker_request) only") {
		t.Fatalf("missing broker-only notice:\n%s", out.String())
	}
}

// TestInteractivePromptsShareOneReader pins that consecutive prompts against
// the same stream each get their own line.
//
// It used to prove this by reassigning the package-level `connectInput`
// bufio.Reader, which existed precisely because a per-call bufio.Reader would
// buffer past the newline and swallow the second answer. The mutable global is
// gone; the guarantee now comes from prompt.Prompt reading unbuffered, and the
// stream arrives through cobra rather than through package state.
func TestInteractivePromptsShareOneReader(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("first\nsecond\n"))
	cmd.SetOut(&bytes.Buffer{})

	if got := connectPrompt(cmd, ""); got != "first" {
		t.Fatalf("first=%q", got)
	}
	if got := connectPrompt(cmd, ""); got != "second" {
		t.Fatalf("second=%q", got)
	}
}

func TestNonInteractiveConnectCreatesCodeHostCapabilityWithoutChangingDefault(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hero"), 0755)
	connectIntegrationID = "github-host"
	connectProject = "hero-engine/hero"
	connectBaseURL = ""
	connectUserEmail = ""
	connectRole = "code-host"
	connectTokenStdin = true
	connectLocalOnly = true
	connectGlobal = false
	connectJSON = true
	connectNoVerify = true
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("CODE-HOST-CONNECT-CANARY\n"))
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runConnectNonInteractive(cmd, root, config.Credentials{}, "github"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Integrations.Default != "" {
		t.Fatalf("code-host connect changed default to %q", cfg.Integrations.Default)
	}
	if cfg.Integrations.Roles["code-host"] != "github-host" {
		t.Fatalf("roles=%v", cfg.Integrations.Roles)
	}
	connection := cfg.Integrations.Connections["github-host"]
	if !connection.SupportsCapability(config.CapabilityCodeHost) || connection.SupportsCapability(config.CapabilityTracker) {
		t.Fatalf("capabilities=%v", connection.EffectiveCapabilities())
	}
	if strings.Contains(out.String(), "CANARY") {
		t.Fatal("connect result leaked token")
	}
}

func TestNonInteractiveConnectRejectsJiraCodeHostBeforePersistence(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".hero"), 0755)
	connectIntegrationID = "jira-host"
	connectProject = "HERO"
	connectBaseURL = "https://jira.invalid"
	connectUserEmail = ""
	connectRole = "code-host"
	connectTokenStdin = true
	connectLocalOnly = true
	connectGlobal = false
	connectJSON = true
	connectNoVerify = true
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("unused-canary\n"))
	cmd.SetOut(&bytes.Buffer{})
	err := runConnectNonInteractive(cmd, root, config.Credentials{}, "jira")
	if err == nil || !strings.Contains(err.Error(), `provider "jira" cannot serve role "code-host"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".hero", config.LocalConfigFileName)); !os.IsNotExist(statErr) {
		t.Fatalf("invalid role wrote local config: %v", statErr)
	}
}

func TestNonInteractiveConnectUpgradesExistingGitHubToDualCapability(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".hero")
	os.MkdirAll(dir, 0755)
	shared := `{"folder":".hero","integrations":{"default":"github-main","roles":{"delivery":"github-main"},"connections":{"github-main":{"provider":"github","settings":{"project":"hero-engine/hero"}}}}}`
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFileName), []byte(shared), 0644); err != nil {
		t.Fatal(err)
	}
	connectIntegrationID = "github-main"
	connectProject = "hero-engine/hero"
	connectBaseURL = ""
	connectUserEmail = ""
	connectRole = "code-host"
	connectTokenStdin = true
	connectLocalOnly = false
	connectGlobal = false
	connectJSON = true
	connectNoVerify = true
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("DUAL-CAPABILITY-CANARY\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := runConnectNonInteractive(cmd, root, config.Credentials{}, "github"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	connection := cfg.Integrations.Connections["github-main"]
	if !connection.SupportsCapability(config.CapabilityTracker) || !connection.SupportsCapability(config.CapabilityCodeHost) {
		t.Fatalf("capabilities=%v", connection.EffectiveCapabilities())
	}
	if cfg.Integrations.Roles["delivery"] != "github-main" || cfg.Integrations.Roles["code-host"] != "github-main" {
		t.Fatalf("roles=%v", cfg.Integrations.Roles)
	}
}
