package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CrossRepoRef represents a parsed cross-repo relation target.
type CrossRepoRef struct {
	Repo string // repo alias (empty for local)
	Slug string // spec slug
}

// ParseRelationTarget splits a relation target into repo alias and slug.
// "auth-service/session-tokens" → {Repo: "auth-service", Slug: "session-tokens"}
// "session-tokens" → {Repo: "", Slug: "session-tokens"}
func ParseRelationTarget(target string) CrossRepoRef {
	parts := strings.SplitN(target, "/", 2)
	if len(parts) == 2 {
		return CrossRepoRef{Repo: parts[0], Slug: parts[1]}
	}
	return CrossRepoRef{Slug: target}
}

// IsRemote returns true if the ref points to another repo.
func (r CrossRepoRef) IsRemote() bool {
	return r.Repo != ""
}

// CrossRepoResolver resolves spec slugs across configured repositories.
// It dual-keys by alias (display) and peer_id (canonical). Existing
// alias-keyed callers continue to work; new code paths can look up
// repos by their stable peer_id even when the local alias differs
// from the peer's alias.
//
// Caches resolved specs per invocation to avoid repeated disk reads.
type CrossRepoResolver struct {
	// repoPaths maps alias → absolute path to project root.
	repoPaths map[string]string
	// aliasToPeerID maps alias → peer_id (when discoverable from the
	// peer's hero.json). Empty when the peer lacks a peer_id.
	aliasToPeerID map[string]string
	// peerIDToAlias maps peer_id → alias. Reverse of aliasToPeerID;
	// used for display when the join key is the UUID.
	peerIDToAlias map[string]string
	// heroFolder is the name of the hero directory (usually ".hero")
	heroFolder string

	mu    sync.Mutex
	cache map[string]map[string]*Spec // repo alias → slug → spec
}

// NewCrossRepoResolver creates a resolver from a map of alias →
// absolute paths. Peer IDs are read lazily from each peer's hero.json
// on the first lookup that needs them.
func NewCrossRepoResolver(repoPaths map[string]string, heroFolder string) *CrossRepoResolver {
	return &CrossRepoResolver{
		repoPaths:     repoPaths,
		aliasToPeerID: make(map[string]string),
		peerIDToAlias: make(map[string]string),
		heroFolder:    heroFolder,
		cache:         make(map[string]map[string]*Spec),
	}
}

// WithPeerIDs pre-populates the alias↔peer_id mapping. Useful when
// the caller has already loaded hero.json's repo_meta block and
// wants to skip the on-demand reads.
func (r *CrossRepoResolver) WithPeerIDs(aliasToPeerID map[string]string) *CrossRepoResolver {
	r.mu.Lock()
	defer r.mu.Unlock()
	for alias, peerID := range aliasToPeerID {
		if peerID == "" {
			continue
		}
		r.aliasToPeerID[alias] = peerID
		r.peerIDToAlias[peerID] = alias
	}
	return r
}

// PeerIDForAlias returns the peer_id for the given alias, reading
// from cache or from the peer's hero.json on demand. Returns the
// empty string when the alias is unknown or the peer lacks a peer_id.
func (r *CrossRepoResolver) PeerIDForAlias(alias string) string {
	r.mu.Lock()
	if id, ok := r.aliasToPeerID[alias]; ok {
		r.mu.Unlock()
		return id
	}
	repoPath, ok := r.repoPaths[alias]
	r.mu.Unlock()
	if !ok {
		return ""
	}

	id := readPeerIDFromHeroJSON(filepath.Join(repoPath, r.heroFolder, "hero.json"))
	r.mu.Lock()
	r.aliasToPeerID[alias] = id // cache even empty result
	if id != "" {
		r.peerIDToAlias[id] = alias
	}
	r.mu.Unlock()
	return id
}

// AliasForPeerID returns the local alias for the given peer_id,
// reading peer hero.json files lazily as needed. Returns the empty
// string when no configured peer matches.
func (r *CrossRepoResolver) AliasForPeerID(peerID string) string {
	if peerID == "" {
		return ""
	}
	r.mu.Lock()
	if alias, ok := r.peerIDToAlias[peerID]; ok {
		r.mu.Unlock()
		return alias
	}
	// Snapshot the aliases we haven't probed yet.
	var toProbe []string
	for alias := range r.repoPaths {
		if _, seen := r.aliasToPeerID[alias]; !seen {
			toProbe = append(toProbe, alias)
		}
	}
	r.mu.Unlock()

	for _, alias := range toProbe {
		if id := r.PeerIDForAlias(alias); id == peerID {
			return alias
		}
	}
	r.mu.Lock()
	alias := r.peerIDToAlias[peerID]
	r.mu.Unlock()
	return alias
}

// ResolveByPeerID looks up a spec by canonical peer_id + slug. Returns
// nil and a clear error if no configured peer has that peer_id or the
// slug is missing.
func (r *CrossRepoResolver) ResolveByPeerID(peerID, slug string) (*Spec, error) {
	alias := r.AliasForPeerID(peerID)
	if alias == "" {
		return nil, fmt.Errorf("no configured repo has peer_id %q", peerID)
	}
	return r.Resolve(CrossRepoRef{Repo: alias, Slug: slug})
}

// Resolve looks up a spec in a remote repo by alias and slug.
// Returns nil and an error if the repo or spec is not found.
func (r *CrossRepoResolver) Resolve(ref CrossRepoRef) (*Spec, error) {
	if !ref.IsRemote() {
		return nil, fmt.Errorf("not a remote ref: %s", ref.Slug)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check cache
	if repoCache, ok := r.cache[ref.Repo]; ok {
		if s, ok := repoCache[ref.Slug]; ok {
			return s, nil
		}
		// We loaded the repo but the slug wasn't found
		return nil, fmt.Errorf("spec %q not found in repo %q", ref.Slug, ref.Repo)
	}

	// Load all specs from the remote repo
	repoPath, ok := r.repoPaths[ref.Repo]
	if !ok {
		return nil, fmt.Errorf("repo alias %q not configured", ref.Repo)
	}

	heroDir := repoPath + "/" + r.heroFolder
	specs, err := Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs in %q: %w", ref.Repo, err)
	}

	// Cache all specs from this repo
	repoCache := make(map[string]*Spec, len(specs))
	for _, s := range specs {
		repoCache[s.Slug] = s
	}
	r.cache[ref.Repo] = repoCache

	if s, ok := repoCache[ref.Slug]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("spec %q not found in repo %q", ref.Slug, ref.Repo)
}

// CrossRepoRelations returns all cross-repo relations from a spec.
func CrossRepoRelations(s *Spec) []CrossRepoRef {
	var refs []CrossRepoRef
	for _, rel := range s.Relations {
		ref := ParseRelationTarget(rel.Target)
		if ref.IsRemote() {
			refs = append(refs, ref)
		}
	}
	return refs
}

// readPeerIDFromHeroJSON reads the peer_id from a peer's hero.json
// without depending on the internal/config package (which would
// create an import cycle: config → spec via discovery helpers). A
// minimal JSON shape is enough.
func readPeerIDFromHeroJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var shape struct {
		PeerID string `json:"peer_id"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return ""
	}
	return shape.PeerID
}
