package chat

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Registry tracks all currently connected Hero adapters keyed by
// connection id. Safe for concurrent use.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*registryEntry
}

type registryEntry struct {
	adapter     HeroAdapter
	connectedAt time.Time
	lastSeen    time.Time
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]*registryEntry)}
}

// Register adds an adapter to the registry. Returns an error if the
// id is empty or already registered.
func (r *Registry) Register(id string, a HeroAdapter) error {
	if id == "" {
		return fmt.Errorf("registry: id required")
	}
	if a == nil {
		return fmt.Errorf("registry: adapter required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[id]; ok {
		return fmt.Errorf("registry: id %q already registered", id)
	}
	now := time.Now()
	r.entries[id] = &registryEntry{
		adapter:     a,
		connectedAt: now,
		lastSeen:    now,
	}
	return nil
}

// Deregister removes an adapter and calls Close on it. Unknown ids
// are silently ignored.
func (r *Registry) Deregister(id string) {
	r.mu.Lock()
	entry, ok := r.entries[id]
	if ok {
		delete(r.entries, id)
	}
	r.mu.Unlock()
	if ok && entry.adapter != nil {
		_ = entry.adapter.Close()
	}
}

// Touch updates the last-seen timestamp for an adapter id. No-op for
// unknown ids.
func (r *Registry) Touch(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[id]; ok {
		e.lastSeen = time.Now()
	}
}

// All returns a snapshot of every connected adapter as AdapterInfo,
// sorted by connection time ascending (oldest first) for deterministic
// JSON output.
func (r *Registry) All() []AdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AdapterInfo, 0, len(r.entries))
	for id, e := range r.entries {
		out = append(out, AdapterInfo{
			ID:          id,
			Adapter:     e.adapter.Name(),
			Version:     e.adapter.Version(),
			Kinds:       append([]Kind(nil), e.adapter.Kinds()...),
			ConnectedAt: e.connectedAt,
			LastSeen:    e.lastSeen,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ConnectedAt.Before(out[j].ConnectedAt)
	})
	return out
}

// Get returns the adapter registered under id, or nil if absent.
func (r *Registry) Get(id string) HeroAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if e, ok := r.entries[id]; ok {
		return e.adapter
	}
	return nil
}

// ByKind returns the first connected adapter supporting kind, in
// registration order. Returns nil when no adapter matches.
func (r *Registry) ByKind(k Kind) HeroAdapter {
	for _, info := range r.All() {
		if supports(info.Kinds, k) {
			if a := r.Get(info.ID); a != nil {
				return a
			}
		}
	}
	return nil
}

// PreferHeroCode returns the first connected adapter supporting kind
// where adapter.Name() == "hero-code"; falls back to ByKind if none
// match. This is the default tiebreak when multiple adapters can
// serve the same kind.
func (r *Registry) PreferHeroCode(k Kind) HeroAdapter {
	for _, info := range r.All() {
		if info.Adapter == "hero-code" && supports(info.Kinds, k) {
			if a := r.Get(info.ID); a != nil {
				return a
			}
		}
	}
	return r.ByKind(k)
}

func supports(kinds []Kind, want Kind) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}
