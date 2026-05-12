package spec

import (
	"fmt"
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
// It caches resolved specs per invocation to avoid repeated disk reads.
type CrossRepoResolver struct {
	// repoPaths maps alias → absolute path to project root
	repoPaths map[string]string
	// heroFolder is the name of the hero directory (usually ".hero")
	heroFolder string

	mu    sync.Mutex
	cache map[string]map[string]*Spec // repo alias → slug → spec
}

// NewCrossRepoResolver creates a resolver from a map of alias → absolute paths.
func NewCrossRepoResolver(repoPaths map[string]string, heroFolder string) *CrossRepoResolver {
	return &CrossRepoResolver{
		repoPaths:  repoPaths,
		heroFolder: heroFolder,
		cache:      make(map[string]map[string]*Spec),
	}
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
