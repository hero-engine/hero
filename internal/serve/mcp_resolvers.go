package serve

import (
	"fmt"
	"os"

	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/refs"
)

// setupResolvers registers a resolver for each ref kind the server
// produces. Resolvers run when hero_expand is called against a ref
// whose cached content is empty — most often after a process restart
// dropped in-memory state but the refs.db row survives.
//
// Phase 1: stable kinds (spec/convention/decision/rule) re-read the
// underlying file. Query kinds (search/context/recap/why/blocked/feed)
// don't have a stable replay path here — Phase 1 keeps content cached
// so resolvers won't normally be invoked. If they are, we return an
// error and hero_expand surfaces a structured re-fetch hint to the
// caller.
func (s *MCPServer) setupResolvers() {
	if s.refsRegistry == nil {
		s.refsRegistry = refs.NewRegistry()
	}

	specResolver := func(slug, scope string, args map[string]any) (string, string, error) {
		idx, err := index.Open(s.heroDir)
		if err != nil {
			return "", "", fmt.Errorf("opening index: %w", err)
		}
		defer idx.Close()
		all, err := idx.AllSpecs()
		if err != nil {
			return "", "", err
		}
		for _, r := range all {
			if r.Slug == slug {
				content, err := os.ReadFile(r.Path)
				if err != nil {
					return "", "", fmt.Errorf("reading spec %q: %w", slug, err)
				}
				return string(content), fingerprintFile(r.Path), nil
			}
		}
		return "", "", fmt.Errorf("spec %q not found", slug)
	}

	s.refsRegistry.Register(refs.KindSpec, specResolver)
	s.refsRegistry.Register(refs.KindConvention, specResolver)
	s.refsRegistry.Register(refs.KindDecision, specResolver)
	s.refsRegistry.Register(refs.KindRule, specResolver)

	queryNoReplay := func(kind refs.Kind) refs.Resolver {
		return func(slug, scope string, args map[string]any) (string, string, error) {
			return "", "", fmt.Errorf("ref of kind %q is session-scoped and cannot be replayed; re-run the producing tool", kind)
		}
	}
	s.refsRegistry.Register(refs.KindSearch, queryNoReplay(refs.KindSearch))
	s.refsRegistry.Register(refs.KindContext, queryNoReplay(refs.KindContext))
	s.refsRegistry.Register(refs.KindRecap, queryNoReplay(refs.KindRecap))
	s.refsRegistry.Register(refs.KindWhy, queryNoReplay(refs.KindWhy))
	s.refsRegistry.Register(refs.KindBlocked, queryNoReplay(refs.KindBlocked))
	s.refsRegistry.Register(refs.KindFeed, queryNoReplay(refs.KindFeed))
}
