package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TeamConnection holds the stored connection to a team server.
type TeamConnection struct {
	URL         string `json:"url"`
	Token       string `json:"token"`
	User        string `json:"user"`
	ConnectedAt string `json:"connected_at"`
	AutoSync    *bool  `json:"auto_sync,omitempty"`
}

// AutoSyncEnabled returns whether auto-sync is enabled. Defaults to
// true for existing connections (nil pointer = not explicitly set).
func (tc *TeamConnection) AutoSyncEnabled() bool {
	if tc.AutoSync == nil {
		return true
	}
	return *tc.AutoSync
}

// teamConfigPath returns ~/.hero/team.json
func teamConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hero", "team.json"), nil
}

// LoadTeamConnection loads the stored team server connection.
// Returns nil if not connected.
func LoadTeamConnection() *TeamConnection {
	path, err := teamConfigPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var tc TeamConnection
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil
	}
	if tc.URL == "" {
		return nil
	}
	return &tc
}

// SaveTeamConnection stores a team server connection.
func SaveTeamConnection(tc *TeamConnection) error {
	path, err := teamConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if tc.ConnectedAt == "" {
		tc.ConnectedAt = time.Now().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(tc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// RemoveTeamConnection deletes the stored connection.
func RemoveTeamConnection() error {
	path, err := teamConfigPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// TeamServerURL returns the team server URL if connected, empty string if not.
func TeamServerURL() string {
	tc := LoadTeamConnection()
	if tc == nil {
		return ""
	}
	return tc.URL
}

// TeamServerRequest makes an authenticated HTTP request to the team server.
func (tc *TeamConnection) AuthHeader() (string, string) {
	if tc.Token != "" {
		return "Authorization", fmt.Sprintf("Bearer %s", tc.Token)
	}
	return "", ""
}
