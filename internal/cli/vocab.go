package cli

import (
	"strings"
	"sync"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/methodology"
	"github.com/hero-engine/hero/internal/vocabulary"
)

// activeVocab returns the active vocabulary for the given config, or nil
// when neither `vocabulary:` nor `methodology:` is set in hero.json.
// Returning nil is load-bearing: callers pass the result to
// (*vocabulary.Vocabulary).Display, which is nil-safe and falls through
// to the canonical type literal — preserving engineering's rendering
// bit-for-bit when no vocab/methodology layer is opted in.
//
// Precedence is owned by vocabulary.Resolve (see its doc comment). When
// the workspace has neither vocab nor methodology, returns nil to
// signal "no rendering layer active" — distinct from the resolver's
// internal "default" fallback.
func activeVocab(cfg *config.Config) *vocabulary.Vocabulary {
	if cfg == nil {
		return nil
	}
	if cfg.Vocabulary == "" && cfg.Methodology == "" {
		// Engineering / legacy workspace — no vocab layer.
		return nil
	}

	vocabs, err := loadVocabsCached()
	if err != nil || len(vocabs) == 0 {
		return nil
	}

	methodologies, _ := loadMethodologiesCached()
	v, err := vocabulary.Resolve(cfg, vocabs, methodologies)
	if err != nil {
		return nil
	}
	return v
}

// activeMethodology returns the active methodology for the given config,
// or nil when no methodology layer is opted in. Mirrors activeVocab's
// nil-return-when-not-configured semantics so callers can branch on
// "is a methodology active?" without false positives from default
// fallbacks.
func activeMethodology(cfg *config.Config) *methodology.Methodology {
	if cfg == nil || cfg.Methodology == "" {
		return nil
	}
	methodologies, err := loadMethodologiesCached()
	if err != nil || len(methodologies) == 0 {
		return nil
	}
	m, err := methodology.Resolve(cfg, methodologies)
	if err != nil {
		return nil
	}
	return m
}

// displayType renders a canonical type literal under the active
// vocabulary. When no vocabulary is active (engineering/legacy), the
// canonical literal is returned unchanged.
//
// Translation note: existing engineering specs store types as "feature",
// "bug", "initiative", etc. (top-level type literals). Vocabulary YAML
// files use the canonical (type, kind) pair where the work artifact is
// (type=spec, kind=feature). displayType bridges the two by mapping the
// flat type literal to the (type, kind) pair the vocabulary expects.
func displayType(v *vocabulary.Vocabulary, t string) string {
	if v == nil || t == "" {
		return t
	}
	typeName, kind := canonicalize(t)
	return v.Display(typeName, kind)
}

// canonicalize maps Hero's flat type literal (as stored in spec
// frontmatter) to the (type, kind) pair vocabulary YAML files use.
// The mapping is intentionally narrow — only the work-type literals
// engineering specs carry today. Knowledge types (decision, convention,
// rule, …) map to themselves at the type level with empty kind.
func canonicalize(t string) (typeName, kind string) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "feature", "bug", "chore", "refactor", "perf", "infra", "security", "ux":
		return "spec", strings.ToLower(strings.TrimSpace(t))
	case "initiative":
		// Engineering's "initiative" is the canonical "roadmap-item" in
		// vocabulary terms.
		return "roadmap-item", ""
	default:
		return strings.ToLower(strings.TrimSpace(t)), ""
	}
}

// Cached loaders. The embedded vocabulary and methodology corpora are
// immutable for the lifetime of the process; loading them once per run
// avoids reparsing five+ YAML files on every CLI invocation. Errors are
// returned as-is so callers can branch — the cache only protects the
// happy path.
var (
	vocabOnce   sync.Once
	vocabCache  map[string]*vocabulary.Vocabulary
	vocabErr    error
	methodOnce  sync.Once
	methodCache map[string]*methodology.Methodology
	methodErr   error
)

func loadVocabsCached() (map[string]*vocabulary.Vocabulary, error) {
	vocabOnce.Do(func() {
		vocabCache, vocabErr = vocabulary.Load(vocabulary.CoreFS(), nil)
	})
	return vocabCache, vocabErr
}

func loadMethodologiesCached() (map[string]*methodology.Methodology, error) {
	methodOnce.Do(func() {
		methodCache, methodErr = methodology.Load(methodology.CoreFS(), nil)
	})
	return methodCache, methodErr
}

// resetVocabCacheForTesting clears the package-level caches so tests can
// re-load the corpus with different inputs. Test-only.
func resetVocabCacheForTesting() {
	vocabOnce = sync.Once{}
	vocabCache = nil
	vocabErr = nil
	methodOnce = sync.Once{}
	methodCache = nil
	methodErr = nil
}

// dialectLine returns a single-line summary of the active vocabulary
// and methodology, or an empty string when neither is configured.
// Quiet for engineering / legacy workspaces (both unset) so existing
// output is unchanged.
func dialectLine(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	if cfg.Vocabulary == "" && cfg.Methodology == "" {
		return ""
	}
	vocab := activeVocab(cfg)
	m := activeMethodology(cfg)
	var vocabName, methName string
	if vocab != nil {
		vocabName = vocab.Name
	}
	if m != nil {
		methName = m.Name
	}
	switch {
	case vocabName != "" && methName != "":
		return "Vocabulary: " + vocabName + " · Methodology: " + methName
	case vocabName != "":
		return "Vocabulary: " + vocabName
	case methName != "":
		return "Methodology: " + methName
	}
	return ""
}

// loadConfigSilent reads hero.json from the project root and returns a
// pointer to it, or nil on any error. Used by render helpers that need
// the active vocabulary/methodology but must not abort rendering when
// the workspace is misconfigured or absent.
func loadConfigSilent() *config.Config {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil
	}
	return &cfg
}
