package serve

import (
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/methodology"
	"github.com/hero-engine/hero/internal/vocabulary"
)

// activeVocab returns the active vocabulary for the workspace, or nil
// when neither `vocabulary:` nor `methodology:` is set in hero.json.
// Returning nil is load-bearing: callers pass the result to
// (*vocabulary.Vocabulary).Display, which is nil-safe and falls through
// to the canonical type literal — preserving engineering's MCP tool
// output bit-for-bit when no vocab/methodology layer is opted in.
//
// Mirrors internal/cli.activeVocab — kept package-local rather than
// exposed from cli to avoid a serve → cli import.
func activeVocab(cfg *config.Config) *vocabulary.Vocabulary {
	if cfg == nil {
		return nil
	}
	if cfg.Vocabulary == "" && cfg.Methodology == "" {
		return nil
	}

	vocabs, err := vocabulary.Load(vocabulary.CoreFS(), nil)
	if err != nil || len(vocabs) == 0 {
		return nil
	}

	merged := *cfg
	if merged.Vocabulary == "" && merged.Methodology != "" {
		methodologies, mErr := methodology.Load(methodology.CoreFS(), nil)
		if mErr == nil && len(methodologies) > 0 {
			if m, ok := methodologies[merged.Methodology]; ok && m != nil {
				if derived := methodology.DeriveVocabularyName(&merged, m); derived != "" {
					merged.Vocabulary = derived
				}
			}
		}
	}

	v, err := vocabulary.Resolve(&merged, vocabs)
	if err != nil {
		return nil
	}
	return v
}

// displayType renders a canonical type literal through the active
// vocabulary. When vocab is nil, returns the literal unchanged.
//
// Mirrors internal/cli.displayType. Translation maps the flat type
// literal stored in spec frontmatter ("feature", "bug", "initiative")
// to the (type, kind) pair vocabulary YAML expects.
func displayType(v *vocabulary.Vocabulary, t string) string {
	if v == nil || t == "" {
		return t
	}
	typeName, kind := canonicalize(t)
	return v.Display(typeName, kind)
}

func canonicalize(t string) (typeName, kind string) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "feature", "bug", "chore", "refactor", "perf", "infra", "security", "ux":
		return "spec", strings.ToLower(strings.TrimSpace(t))
	case "initiative":
		return "roadmap-item", ""
	default:
		return strings.ToLower(strings.TrimSpace(t)), ""
	}
}
