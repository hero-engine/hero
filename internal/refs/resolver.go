package refs

import (
	"fmt"
	"sync"
)

// Resolver rehydrates the full content for a ref of a given kind.
// It receives the source-args originally registered with the entry
// and returns the current full content plus a fresh fingerprint.
type Resolver func(slug, scope string, sourceArgs map[string]any) (content, fingerprint string, err error)

// Registry is a kind-keyed resolver registry. Each producing tool
// registers its kinds at server startup; hero_expand looks up the
// resolver for the ref's kind to refetch when cache misses or
// fingerprints diverge.
type Registry struct {
	mu        sync.RWMutex
	resolvers map[Kind]Resolver
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{resolvers: map[Kind]Resolver{}}
}

// Register installs a resolver for the given kind, replacing any
// prior entry. Safe for concurrent use.
func (r *Registry) Register(kind Kind, fn Resolver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolvers[kind] = fn
}

// Get returns the resolver for kind, or nil if unregistered.
func (r *Registry) Get(kind Kind) Resolver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resolvers[kind]
}

// Resolve looks up the resolver for entry.Kind and invokes it. Used
// by hero_expand to (re)fetch when content is empty or fingerprint
// is stale. Returns ErrNoResolver if no resolver is registered.
func (r *Registry) Resolve(e *Entry) (content, fingerprint string, err error) {
	if e == nil {
		return "", "", fmt.Errorf("nil entry")
	}
	resolver := r.Get(e.Kind)
	if resolver == nil {
		return "", "", fmt.Errorf("no resolver registered for kind %q", e.Kind)
	}
	var args map[string]any
	if e.SourceArgsJSON != "" {
		_ = unmarshalArgs(e.SourceArgsJSON, &args)
	}
	return resolver(e.Slug, e.Scope, args)
}
