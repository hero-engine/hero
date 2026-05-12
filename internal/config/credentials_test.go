package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCredentials_Missing(t *testing.T) {
	// Point XDG_CONFIG_HOME at a temp dir with no credentials file
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected empty credentials, got %d entries", len(creds))
	}
}

func TestSaveAndLoadCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := make(Credentials)
	SetCredential(creds, "github", "owner/repo", CredentialEntry{Token: "ghp_test"})
	SetCredential(creds, "jira", "PROJ", CredentialEntry{
		Token:   "jira-token",
		BaseURL: "https://example.atlassian.net",
	})

	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 entries, got %d", len(loaded))
	}

	entry, ok := GetCredential(loaded, "github", "owner/repo")
	if !ok {
		t.Fatal("github:owner/repo not found in loaded credentials")
	}
	if entry.Token != "ghp_test" {
		t.Errorf("Token = %q, want %q", entry.Token, "ghp_test")
	}

	jiraEntry, ok := GetCredential(loaded, "jira", "PROJ")
	if !ok {
		t.Fatal("jira:PROJ not found in loaded credentials")
	}
	if jiraEntry.BaseURL != "https://example.atlassian.net" {
		t.Errorf("BaseURL = %q, want %q", jiraEntry.BaseURL, "https://example.atlassian.net")
	}
}

func TestCredentials_FileMode(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := make(Credentials)
	SetCredential(creds, "linear", "ENG", CredentialEntry{Token: "lin_secret"})

	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	path := CredentialsPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestCredentialsPath_XDG(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	got := CredentialsPath()
	want := filepath.Join(tmpDir, "hero", "credentials.json")
	if got != want {
		t.Errorf("CredentialsPath = %q, want %q", got, want)
	}
}

func TestRemoveCredential(t *testing.T) {
	creds := make(Credentials)
	SetCredential(creds, "github", "acme/app", CredentialEntry{Token: "tok"})
	SetCredential(creds, "jira", "PROJ", CredentialEntry{Token: "tok2"})

	RemoveCredential(creds, "github", "acme/app")

	if _, ok := GetCredential(creds, "github", "acme/app"); ok {
		t.Error("expected github:acme/app to be removed")
	}
	if _, ok := GetCredential(creds, "jira", "PROJ"); !ok {
		t.Error("jira:PROJ should still exist after removing github entry")
	}
}

func TestApplyCredentials_TrackerToken(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tracker = &TrackerConfig{Type: "github", Project: "acme/app"}

	creds := make(Credentials)
	SetCredential(creds, "github", "acme/app", CredentialEntry{Token: "from-creds"})

	result := ApplyCredentials(cfg, creds)
	if result.Tracker.Token != "from-creds" {
		t.Errorf("Token = %q, want %q", result.Tracker.Token, "from-creds")
	}
}

func TestApplyCredentials_DoesNotOverwriteExistingToken(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tracker = &TrackerConfig{Type: "github", Project: "acme/app", Token: "already-set"}

	creds := make(Credentials)
	SetCredential(creds, "github", "acme/app", CredentialEntry{Token: "from-creds"})

	result := ApplyCredentials(cfg, creds)
	if result.Tracker.Token != "already-set" {
		t.Errorf("existing Token should not be overwritten, got %q", result.Tracker.Token)
	}
}

func TestApplyCredentials_NoMatch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Tracker = &TrackerConfig{Type: "github", Project: "acme/app"}

	creds := make(Credentials) // empty

	result := ApplyCredentials(cfg, creds)
	if result.Tracker.Token != "" {
		t.Errorf("expected empty token when no creds match, got %q", result.Tracker.Token)
	}
}
