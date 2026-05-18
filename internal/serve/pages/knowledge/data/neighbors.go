package data

// NeighborsInputs is the per-request input bundle for the suggested-
// neighbors section.
type NeighborsInputs struct {
	HeroDir string
	Target  string
}

// LoadNeighbors returns the up-to-four neighbor rows displayed under
// "You might also explore". When the unified-retrieval-layer spec has
// not shipped the related-node query we return an empty payload and
// the template renders nothing — the section header points the user
// at /knowledge for the full browse view.
func LoadNeighbors(in NeighborsInputs) Neighbors {
	_ = in
	return Neighbors{}
}
