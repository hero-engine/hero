package data

import "github.com/hero-engine/hero/internal/serve/opsrunner"

// OpsLookup is the minimum surface the operations loader needs from
// the runner. Defined here (rather than imported as a concrete type)
// so tests can inject a stub without depending on the opsrunner
// package itself.
type OpsLookup interface {
	Lookup(slug, verb string) (jobID string, ok bool)
}

// OperationsInputs is the per-request bundle for the Operations section.
type OperationsInputs struct {
	// Slug is the project slug used for lookup and POST URLs.
	Slug string

	// Lookup is the runner probe. Nil-tolerant: when nil, the section
	// renders with every verb's InFlight=false, ActiveJobID="" — but
	// Available stays true so the buttons still POST.
	Lookup OpsLookup

	// Available reports whether the runner is wired at all. When false
	// the template renders the "Operations require a project-aware
	// build of hero serve" empty state. Defensive only — should never
	// be false in practice once Phase 3 lands.
	Available bool
}

// Operations is what the partial renders.
type Operations struct {
	Slug      string
	Available bool
	Verbs     []OpVerb
}

// OpVerb is one row in the Operations section. Verb is the URL-safe
// allowlist key; Label and Description come from opsrunner. InFlight
// + ActiveJobID let the template mark a button as already-running so
// the page-load JS can immediately reattach an EventSource.
type OpVerb struct {
	Verb        string
	Label       string
	Description string
	InFlight    bool
	ActiveJobID string
}

// LoadOperations builds the Operations section data. Always returns a
// fully-populated slice of verbs (one per allowlist entry) so the
// template renders a stable button row even when the runner is nil.
func LoadOperations(in OperationsInputs) Operations {
	out := Operations{Slug: in.Slug, Available: in.Available}
	for _, v := range opsrunner.AllVerbs() {
		row := OpVerb{
			Verb:        v,
			Label:       opsrunner.VerbLabel(v),
			Description: opsrunner.VerbDescription(v),
		}
		if in.Lookup != nil {
			if id, ok := in.Lookup.Lookup(in.Slug, v); ok {
				row.InFlight = true
				row.ActiveJobID = id
			}
		}
		out.Verbs = append(out.Verbs, row)
	}
	return out
}
