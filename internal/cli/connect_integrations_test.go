package cli

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

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
	for _, provider := range []string{"github", "jira", "linear", "gitlab", "confluence"} {
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

func TestInteractivePromptsShareOneReader(t *testing.T) {
	connectInput = bufio.NewReader(strings.NewReader("first\nsecond\n"))
	if got := prompt(""); got != "first" {
		t.Fatalf("first=%q", got)
	}
	if got := prompt(""); got != "second" {
		t.Fatalf("second=%q", got)
	}
}
