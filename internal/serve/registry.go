package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Project registry — tracks all known Hero projects on this machine
// ---------------------------------------------------------------------------

// ProjectEntry represents a registered project in the registry.
type ProjectEntry struct {
	Path       string    `json:"path"`
	Registered time.Time `json:"registered"`
}

// Registry stores all known Hero projects.
type Registry struct {
	Projects map[string]*ProjectEntry `json:"projects"`
	mu       sync.RWMutex
	path     string // file path to the registry JSON
}

// registryDir returns the global hero config directory (~/.hero).
func registryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".hero"), nil
}

// registryPath returns the default path to the registry file.
func registryPath() (string, error) {
	dir, err := registryDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects.json"), nil
}

// LoadRegistry reads the project registry from the default location.
// If the file doesn't exist, returns an empty registry.
func LoadRegistry() (*Registry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	return LoadRegistryFrom(path)
}

// LoadRegistryFrom reads the project registry from a specific path.
func LoadRegistryFrom(path string) (*Registry, error) {
	r := &Registry{
		Projects: make(map[string]*ProjectEntry),
		path:     path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("reading registry %s: %w", path, err)
	}

	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("parsing registry %s: %w", path, err)
	}

	if r.Projects == nil {
		r.Projects = make(map[string]*ProjectEntry)
	}
	return r, nil
}

// Save writes the registry to disk.
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}

	return os.WriteFile(r.path, data, 0o644)
}

// Add registers a project. The slug is derived from the directory name unless
// the project has a hero.json with a custom name. Returns the slug used.
func (r *Registry) Add(projectRoot string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Verify the directory has a .hero folder (or the configured folder)
	heroDir := filepath.Join(absPath, ".hero")
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return "", fmt.Errorf("no .hero directory found at %s (run 'hero init' first)", absPath)
	}

	slug := filepath.Base(absPath)

	// Check for duplicate slug with a different path
	if existing, ok := r.Projects[slug]; ok {
		existingAbs, _ := filepath.Abs(existing.Path)
		if existingAbs != absPath {
			return "", fmt.Errorf("slug %q already registered for %s (this project is at %s)", slug, existing.Path, absPath)
		}
		// Same path — idempotent, just return
		return slug, nil
	}

	r.Projects[slug] = &ProjectEntry{
		Path:       absPath,
		Registered: time.Now().UTC(),
	}

	return slug, nil
}

// Remove unregisters a project by slug.
func (r *Registry) Remove(slug string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Projects[slug]; !ok {
		return fmt.Errorf("project %q is not registered", slug)
	}

	delete(r.Projects, slug)
	return nil
}

// List returns all registered project slugs and their entries.
func (r *Registry) List() map[string]*ProjectEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy
	result := make(map[string]*ProjectEntry, len(r.Projects))
	for k, v := range r.Projects {
		entry := *v
		result[k] = &entry
	}
	return result
}

// Get returns a single project entry by slug, or nil if not found.
func (r *Registry) Get(slug string) *ProjectEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if e, ok := r.Projects[slug]; ok {
		entry := *e
		return &entry
	}
	return nil
}

// Slugs returns all registered project slugs in sorted order.
func (r *Registry) Slugs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	slugs := make([]string, 0, len(r.Projects))
	for k := range r.Projects {
		slugs = append(slugs, k)
	}
	return slugs
}

// HasProject returns true if the slug is registered.
func (r *Registry) HasProject(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.Projects[slug]
	return ok
}

// FindByPath returns the slug of a registered project with the given path.
// Returns empty string if not found.
func (r *Registry) FindByPath(projectRoot string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return ""
	}

	for slug, entry := range r.Projects {
		entryAbs, _ := filepath.Abs(entry.Path)
		if entryAbs == absPath {
			return slug
		}
	}
	return ""
}

// Count returns the number of registered projects.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Projects)
}
