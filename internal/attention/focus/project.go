package focus

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve"
)

const (
	ProjectAvailable = "available"
	ProjectMissing   = "missing"
)

type ResolvedProject struct {
	Reference    *attention.ProjectReference `json:"project,omitempty"`
	Availability string                      `json:"availability"`
	Path         string                      `json:"path,omitempty"`
}

type ProjectResolver interface {
	ResolveReference(*attention.ProjectReference) ResolvedProject
	ResolveInput(string) (*attention.ProjectReference, error)
	ResolveCurrent() (*attention.ProjectReference, error)
}

type RegistryResolver struct {
	registry           *serve.Registry
	peers              map[string]string
	currentProjectRoot string
}

func NewRegistryResolver(registry *serve.Registry) *RegistryResolver {
	return &RegistryResolver{registry: registry, peers: make(map[string]string)}
}

func LoadRegistryResolver(projectRoot string) (*RegistryResolver, error) {
	registry, err := serve.LoadRegistry()
	if err != nil {
		return nil, err
	}
	resolver := NewRegistryResolver(registry)
	resolver.currentProjectRoot = projectRoot
	if projectRoot == "" {
		return resolver, nil
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load project peers: %w", err)
	}
	for alias, status := range cfg.ResolveAllRepos(projectRoot) {
		if status.Accessible {
			resolver.peers[alias] = status.Path
		}
	}
	return resolver, nil
}

// ResolveCurrent returns the canonical identity of the current workspace when
// it is registered globally. An unregistered workspace is intentionally not an
// error: callers may create a genuinely unbound personal item there.
func (r *RegistryResolver) ResolveCurrent() (*attention.ProjectReference, error) {
	root := r.currentProjectRoot
	if root == "" {
		var err error
		root, err = filepath.Abs(".")
		if err != nil {
			return nil, err
		}
	}
	slug := r.registry.FindByPath(root)
	if slug == "" {
		return nil, nil
	}
	entry := r.registry.Get(slug)
	if entry == nil {
		return nil, nil
	}
	return projectReference(slug, entry.Path)
}

func (r *RegistryResolver) ResolveReference(ref *attention.ProjectReference) ResolvedProject {
	if ref == nil {
		return ResolvedProject{Availability: ProjectAvailable}
	}
	for slug, entry := range r.registry.List() {
		cfg, err := config.Load(entry.Path)
		if err != nil || cfg.PeerID == "" || cfg.PeerID != ref.PeerID {
			continue
		}
		path, err := filepath.Abs(entry.Path)
		if err != nil {
			continue
		}
		copy := *ref
		copy.RegistrySlug = slug
		if copy.DisplayName == "" {
			copy.DisplayName = slug
		}
		return ResolvedProject{Reference: &copy, Availability: ProjectAvailable, Path: path}
	}
	for alias, path := range r.peers {
		cfg, err := config.Load(path)
		if err != nil || cfg.PeerID == "" || cfg.PeerID != ref.PeerID {
			continue
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		copy := *ref
		copy.RegistrySlug = alias
		if copy.DisplayName == "" {
			copy.DisplayName = alias
		}
		return ResolvedProject{Reference: &copy, Availability: ProjectAvailable, Path: abs}
	}
	return ResolvedProject{Reference: ref, Availability: ProjectMissing}
}

func (r *RegistryResolver) ResolveInput(value string) (*attention.ProjectReference, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var slug string
	var entry *serve.ProjectEntry
	if value == "." || filepath.IsAbs(value) {
		path := value
		if value == "." {
			path = "."
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		slug = r.registry.FindByPath(abs)
		if slug != "" {
			entry = r.registry.Get(slug)
		}
	} else {
		slug, entry = value, r.registry.Get(value)
		if entry == nil {
			if path, ok := r.peers[value]; ok {
				entry = &serve.ProjectEntry{Path: path}
			}
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("project %q is not in the user registry", value)
	}
	return projectReference(slug, entry.Path)
}

func projectReference(slug, path string) (*attention.ProjectReference, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load project %q: %w", slug, err)
	}
	if cfg.PeerID == "" {
		return nil, fmt.Errorf("project %q has no peer_id", slug)
	}
	displayName := slug
	if cfg.Peering != nil && strings.TrimSpace(cfg.Peering.Display) != "" {
		displayName = cfg.Peering.Display
	}
	return &attention.ProjectReference{PeerID: cfg.PeerID, RegistrySlug: slug, DisplayName: displayName}, nil
}
