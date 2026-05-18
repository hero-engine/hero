package data

// SummaryInputs is the per-request input bundle for the plain-English
// summary section.
type SummaryInputs struct {
	HeroDir string
	Target  string
}

// LoadSummary returns the narrator paragraph payload. Returns
// Available=false until the knowledge-narrator pipeline lands — the
// template surfaces the empty-state notice in that case.
func LoadSummary(in SummaryInputs) Summary {
	_ = in
	return Summary{Available: false}
}
