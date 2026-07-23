// Package projectregistry owns the machine-global registry of Hero projects.
// It is intentionally independent of the serve package so non-server domain
// services can resolve project identities without creating import cycles.
package projectregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type ProjectEntry struct {
	Path          string    `json:"path"`
	Registered    time.Time `json:"registered"`
	registeredRaw string
}

func (e *ProjectEntry) UnmarshalJSON(data []byte) error {
	var raw struct {
		Path       string          `json:"path"`
		Registered json.RawMessage `json:"registered"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.Path = raw.Path
	if len(raw.Registered) == 0 || string(raw.Registered) == "null" {
		return nil
	}
	var value string
	if err := json.Unmarshal(raw.Registered, &value); err != nil {
		return fmt.Errorf("decode registered value: %w", err)
	}
	if value == "auto" {
		e.registeredRaw = value
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return err
	}
	e.Registered = parsed
	return nil
}

func (e ProjectEntry) MarshalJSON() ([]byte, error) {
	registered := any(e.Registered)
	if e.registeredRaw == "auto" && e.Registered.IsZero() {
		registered = "auto"
	}
	return json.Marshal(struct {
		Path       string `json:"path"`
		Registered any    `json:"registered"`
	}{Path: e.Path, Registered: registered})
}

type Registry struct {
	Projects map[string]*ProjectEntry `json:"projects"`
	mu       sync.RWMutex
	path     string
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".hero"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects.json"), nil
}

func Load() (*Registry, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

func LoadFrom(path string) (*Registry, error) {
	r := &Registry{Projects: make(map[string]*ProjectEntry), path: path}
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

func (r *Registry) FilePath() string { return r.path }

func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}
	return os.WriteFile(r.path, data, 0o644)
}

func (r *Registry) Add(projectRoot string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	if _, err := os.Stat(filepath.Join(absPath, ".hero")); os.IsNotExist(err) {
		return "", fmt.Errorf("no .hero directory found at %s (run 'hero init' first)", absPath)
	}
	slug := filepath.Base(absPath)
	if existing, ok := r.Projects[slug]; ok {
		existingAbs, _ := filepath.Abs(existing.Path)
		if existingAbs != absPath {
			return "", fmt.Errorf("slug %q already registered for %s (this project is at %s)", slug, existing.Path, absPath)
		}
		return slug, nil
	}
	r.Projects[slug] = &ProjectEntry{Path: absPath, Registered: time.Now().UTC()}
	return slug, nil
}

func (r *Registry) Remove(slug string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.Projects[slug]; !ok {
		return fmt.Errorf("project %q is not registered", slug)
	}
	delete(r.Projects, slug)
	return nil
}

func (r *Registry) List() map[string]*ProjectEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*ProjectEntry, len(r.Projects))
	for k, v := range r.Projects {
		entry := *v
		result[k] = &entry
	}
	return result
}

func (r *Registry) Get(slug string) *ProjectEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if e, ok := r.Projects[slug]; ok {
		entry := *e
		return &entry
	}
	return nil
}

func (r *Registry) Slugs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	slugs := make([]string, 0, len(r.Projects))
	for k := range r.Projects {
		slugs = append(slugs, k)
	}
	sort.Strings(slugs)
	return slugs
}

func (r *Registry) HasProject(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.Projects[slug]
	return ok
}

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

func (r *Registry) Count() int { r.mu.RLock(); defer r.mu.RUnlock(); return len(r.Projects) }
