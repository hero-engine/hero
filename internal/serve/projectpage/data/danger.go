package data

// DangerInputs is the per-request bundle for the Danger Zone section.
// Phase 4 of hero-serve-project-section.
type DangerInputs struct {
	// Slug is the project slug. Surfaced into the section so the typed-
	// confirmation gate has the exact target string to validate against.
	Slug string

	// Registered reports whether the project is currently in the global
	// registry. The Danger Zone section is hidden entirely when this is
	// false — no point offering destructive actions on a project the
	// daemon doesn't track.
	Registered bool
}

// DangerVerb is one destructive operation surfaced inside the Danger
// Zone. Endpoint is the absolute URL the form posts to when the typed
// confirmation matches the slug.
type DangerVerb struct {
	Verb        string
	Label       string
	Description string
	Endpoint    string
}

// Danger is what the partial renders. Visible=false hides the entire
// section (used when the project is not registered).
type Danger struct {
	Visible bool
	Slug    string
	Verbs   []DangerVerb
}

// LoadDanger shapes the Danger Zone view. Today only `deregister` is
// surfaced — the parent spec calls out a future `archive` button which
// is omitted here because no top-level `hero archive` verb exists yet.
// The seam is in place; adding archive is one new entry in this slice.
func LoadDanger(in DangerInputs) Danger {
	if !in.Registered || in.Slug == "" {
		return Danger{}
	}
	out := Danger{
		Visible: true,
		Slug:    in.Slug,
	}
	out.Verbs = append(out.Verbs, DangerVerb{
		Verb:        "deregister",
		Label:       "Deregister project",
		Description: "Remove this project from ~/.hero/projects.json. Reversible within 5 seconds via the undo toast.",
		Endpoint:    "/api/" + in.Slug + "/registry/remove",
	})
	return out
}
