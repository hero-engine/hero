// Package active manages a registry of active spec sessions so that
// context injection and compaction recovery can prioritize the right spec.
package active

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const registryFileName = ".active-sessions.json"

// Session represents an active spec being worked on.
type Session struct {
	Spec    string    `json:"spec"`
	Command string    `json:"command"`
	Started time.Time `json:"started"`
}

// Registry holds the set of active sessions, keyed by session ID.
type Registry struct {
	Sessions map[string]Session `json:"sessions"`
}

var mu sync.Mutex

// registryPath returns the path to the registry file inside the hero directory.
func registryPath(heroDir string) string {
	return filepath.Join(heroDir, registryFileName)
}

// Load reads the registry from disk. Returns an empty registry if the file
// does not exist or cannot be parsed.
func Load(heroDir string) *Registry {
	r := &Registry{Sessions: make(map[string]Session)}
	data, err := os.ReadFile(registryPath(heroDir))
	if err != nil {
		return r
	}
	_ = json.Unmarshal(data, r)
	if r.Sessions == nil {
		r.Sessions = make(map[string]Session)
	}
	return r
}

// Save writes the registry to disk atomically.
func (r *Registry) Save(heroDir string) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(registryPath(heroDir), data, 0o644)
}

// Register adds or updates an active session.
func Register(heroDir, sessionID, specSlug, command string) error {
	mu.Lock()
	r := Load(heroDir)
	r.Sessions[sessionID] = Session{
		Spec:    specSlug,
		Command: command,
		Started: time.Now().UTC(),
	}
	mu.Unlock()
	return r.Save(heroDir)
}

// Unregister removes a session from the registry.
func Unregister(heroDir, sessionID string) error {
	mu.Lock()
	r := Load(heroDir)
	delete(r.Sessions, sessionID)
	mu.Unlock()
	return r.Save(heroDir)
}

// ActiveSpecs returns the unique set of spec slugs currently being worked on.
func ActiveSpecs(heroDir string) []string {
	r := Load(heroDir)
	seen := make(map[string]bool)
	var slugs []string
	for _, s := range r.Sessions {
		if !seen[s.Spec] {
			seen[s.Spec] = true
			slugs = append(slugs, s.Spec)
		}
	}
	return slugs
}

// Prune removes sessions older than the given duration.
func Prune(heroDir string, maxAge time.Duration) (int, error) {
	mu.Lock()
	r := Load(heroDir)
	cutoff := time.Now().UTC().Add(-maxAge)
	pruned := 0
	for id, s := range r.Sessions {
		if s.Started.Before(cutoff) {
			delete(r.Sessions, id)
			pruned++
		}
	}
	mu.Unlock()
	if pruned > 0 {
		return pruned, r.Save(heroDir)
	}
	return 0, nil
}
