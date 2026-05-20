package data

import "time"

// RegistryInputs is the per-request input bundle for the Registry
// Membership section. Entry nil renders the "not registered" state.
type RegistryInputs struct {
	Slug             string
	Entry            *RegistryEntryView
	IsDefaultProject bool
}

// RegistryEntryView is the minimal shape this loader needs. Defined
// here (rather than re-exporting projectpage.RegistryEntry) so the
// data package stays standalone and easy to test.
type RegistryEntryView struct {
	Path         string
	RegisteredAt time.Time
}

// Registry is what the partial renders. Registered=false means the
// project isn't in ~/.hero/projects.json. CanRemove gates the Phase 4
// "Remove from registry" button — true only when the project IS in
// the registry (no point offering Remove on an unregistered project).
type Registry struct {
	Registered         bool
	Slug               string
	Path               string
	RegisteredAt       time.Time
	RegisteredAtPretty string
	IsDefaultProject   bool
	CanRemove          bool
}

// LoadRegistry shapes the registry-membership data for the partial.
// Nil Entry yields Registered=false.
func LoadRegistry(in RegistryInputs) Registry {
	out := Registry{
		Slug:             in.Slug,
		IsDefaultProject: in.IsDefaultProject,
	}
	if in.Entry == nil {
		return out
	}
	out.Registered = true
	out.CanRemove = in.Slug != ""
	out.Path = in.Entry.Path
	out.RegisteredAt = in.Entry.RegisteredAt
	if !in.Entry.RegisteredAt.IsZero() {
		out.RegisteredAtPretty = in.Entry.RegisteredAt.Format("2006-01-02 15:04")
	}
	return out
}
