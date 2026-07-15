package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// CredentialEntry holds a saved credential for one tracker/service instance.
type CredentialEntry struct {
	Token     string `json:"token"`
	BaseURL   string `json:"base_url,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
}

// Credentials is the full credentials store, keyed by "{type}:{project}".
// Example key: "github:owner/repo", "jira:PROJ", "linear:ENG", "confluence:ENG".
type Credentials map[string]CredentialEntry

// CredentialsPath returns the path to the XDG credentials file.
// Respects XDG_CONFIG_HOME; defaults to ~/.config/hero/credentials.json.
func CredentialsPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fall back to relative path if home dir is not determinable
			return filepath.Join(".config", "hero", "credentials.json")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "hero", "credentials.json")
}

// LoadCredentials reads the credentials file.
// Returns an empty Credentials map (no error) if the file doesn't exist.
func LoadCredentials() (Credentials, error) {
	path := CredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(Credentials), nil
		}
		return nil, fmt.Errorf("reading credentials: %w", err)
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	return creds, nil
}

// SaveCredentials writes the credentials file with mode 0600.
func SaveCredentials(creds Credentials) error {
	path := CredentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating credentials dir: %w", err)
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

// CredentialKey returns the canonical map key for a tracker type + project.
func CredentialKey(trackerType, project string) string {
	return trackerType + ":" + project
}

// IntegrationCredentialKey keys new credentials by stable workspace identity.
func IntegrationCredentialKey(id string) string { return "integration:" + id }
func SetIntegrationCredential(creds Credentials, id string, entry CredentialEntry) {
	creds[IntegrationCredentialKey(id)] = entry
}

// SetCredential upserts a credential entry for the given type+project.
func SetCredential(creds Credentials, trackerType, project string, entry CredentialEntry) {
	creds[CredentialKey(trackerType, project)] = entry
}

// GetCredential looks up a credential entry for the given type+project.
// Returns the entry and true if found.
func GetCredential(creds Credentials, trackerType, project string) (CredentialEntry, bool) {
	entry, ok := creds[CredentialKey(trackerType, project)]
	return entry, ok
}

// RemoveCredential deletes the credential entry for the given type+project.
func RemoveCredential(creds Credentials, trackerType, project string) {
	delete(creds, CredentialKey(trackerType, project))
}

// ApplyCredentials fills token (and optionally base_url, user_email) into cfg
// from the credentials store, if the tracker has a project and no token is
// already set. Returns the (possibly modified) config.
func ApplyCredentials(cfg Config, creds Credentials) Config {
	cfg, _ = ApplyCredentialsStrict(cfg, creds)
	return cfg
}

// ApplyCredentialsStrict prefers stable-ID credentials and fails closed when
// a legacy provider:project credential would ambiguously serve multiple IDs.
func ApplyCredentialsStrict(cfg Config, creds Credentials) (Config, error) {
	if cfg.Integrations != nil {
		legacyUsers := map[string][]string{}
		for id, x := range cfg.Integrations.Connections {
			project := rawString(x.Settings, "project")
			if x.Provider == "confluence" {
				project = rawString(x.Settings, "space_key")
			}
			legacyUsers[CredentialKey(x.Provider, project)] = append(legacyUsers[CredentialKey(x.Provider, project)], id)
		}
		for id, x := range cfg.Integrations.Connections {
			if x.Auth == nil {
				x.Auth = &IntegrationAuth{}
			}
			if x.Auth.Token == "" {
				entry, ok := creds[IntegrationCredentialKey(id)]
				if !ok {
					project := rawString(x.Settings, "project")
					if x.Provider == "confluence" {
						project = rawString(x.Settings, "space_key")
					}
					key := CredentialKey(x.Provider, project)
					if len(legacyUsers[key]) > 1 {
						if _, exists := creds[key]; exists {
							return cfg, fmt.Errorf("legacy global credential %q is ambiguous for integration IDs %v; save credentials by stable integration ID", key, legacyUsers[key])
						}
					}
					entry, ok = creds[key]
				}
				if ok {
					x.Auth.Token = Secret(entry.Token)
					if _, exists := x.Settings["base_url"]; !exists && entry.BaseURL != "" {
						x.Settings["base_url"], _ = json.Marshal(entry.BaseURL)
					}
					if _, exists := x.Settings["user_email"]; !exists && entry.UserEmail != "" {
						x.Settings["user_email"], _ = json.Marshal(entry.UserEmail)
					}
				}
			}
			cfg.Integrations.Connections[id] = x
		}
		r := ResolvedIntegrations{Config: cfg.Integrations}
		if t, ok := r.DeliveryTracker(); ok {
			cfg.Tracker = t
		}
		if c, ok := r.DocsConfluence(); ok {
			cfg.Confluence = c
		}
	}
	if cfg.Tracker != nil && cfg.Tracker.Project != "" && cfg.Tracker.Token == "" {
		if entry, ok := GetCredential(creds, cfg.Tracker.Type, cfg.Tracker.Project); ok {
			if entry.Token != "" {
				cfg.Tracker.Token = entry.Token
			}
			if entry.BaseURL != "" && cfg.Tracker.BaseURL == "" {
				cfg.Tracker.BaseURL = entry.BaseURL
			}
			if entry.UserEmail != "" && cfg.Tracker.UserEmail == "" {
				cfg.Tracker.UserEmail = entry.UserEmail
			}
		}
	}
	if cfg.Confluence != nil && cfg.Confluence.SpaceKey != "" && cfg.Confluence.Token == "" {
		if entry, ok := GetCredential(creds, "confluence", cfg.Confluence.SpaceKey); ok {
			if entry.Token != "" {
				cfg.Confluence.Token = entry.Token
			}
			if entry.UserEmail != "" && cfg.Confluence.UserEmail == "" {
				cfg.Confluence.UserEmail = entry.UserEmail
			}
		}
	}
	return cfg, nil
}
