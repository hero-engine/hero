package data

// WhyInputs is the per-request input bundle for the provenance-chain
// (Why) section. Target is the slug or path being traced.
type WhyInputs struct {
	HeroDir string
	Target  string
}

// LoadWhy returns the provenance trace payload for the given target.
// Today the home renders a structural placeholder (Available=false) so
// the empty-state notice points at the traversal-queries spec. When
// the traversal API exposes a JSON-shaped Why over the graph store the
// fetcher fills the Trace fields in place. Mirrors the now/data pattern
// of nil-safe degradation.
func LoadWhy(in WhyInputs) Why {
	// Placeholder. The hero-knowledge-home spec calls out that the
	// in-browser Why view ships with an empty-state notice when the
	// traversal-queries spec hasn't delivered the JSON-over-HTTP shape
	// yet. We surface Available=false so the template falls through.
	_ = in
	return Why{Available: false}
}
