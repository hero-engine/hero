package domains

import (
	"fmt"
	"sync"
)

// DuplicateContributorError is raised when two manifests declare the
// same ContributorID. The diagnostic names both conflicting manifests
// and the offending id (acceptance criterion §"Authoring").
type DuplicateContributorError struct {
	ContributorID   ContributorID
	FirstManifest   DomainID
	SecondManifest  DomainID
}

func (e *DuplicateContributorError) Error() string {
	return fmt.Sprintf(
		"duplicate ContributorID %q declared by domain %q and domain %q — "+
			"contributor ids must be unique across all manifests; namespace "+
			"under the declaring domain id",
		string(e.ContributorID), string(e.FirstManifest), string(e.SecondManifest),
	)
}

// Registry is the process-start domain-manifest registry. Concurrent-
// safe; the desktop / CLI / TUI consume the same instance.
type Registry struct {
	mu        sync.Mutex
	manifests []DomainManifest
	owners    map[ContributorID]DomainID
}

// NewRegistry creates an empty registry. Production wireup builds one
// at process start and feeds it the built-in manifests (D6).
func NewRegistry() *Registry {
	return &Registry{owners: make(map[ContributorID]DomainID)}
}

// Register adds a manifest. Returns a *DuplicateContributorError if
// any ContributorID is already registered, or appears more than once
// within the manifest itself. The registry is left unchanged on error
// so callers can decide whether to surface the failure or proceed.
func (r *Registry) Register(m DomainManifest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	declared := collectContributorIDs(m)

	// Cross-manifest collisions first.
	for _, id := range declared {
		if owner, ok := r.owners[id]; ok {
			return &DuplicateContributorError{
				ContributorID:  id,
				FirstManifest:  owner,
				SecondManifest: m.ID,
			}
		}
	}

	// Intra-manifest duplicates.
	seen := make(map[ContributorID]struct{}, len(declared))
	for _, id := range declared {
		if _, dup := seen[id]; dup {
			return &DuplicateContributorError{
				ContributorID:  id,
				FirstManifest:  m.ID,
				SecondManifest: m.ID,
			}
		}
		seen[id] = struct{}{}
	}

	// Commit.
	for _, id := range declared {
		r.owners[id] = m.ID
	}
	r.manifests = append(r.manifests, m)
	return nil
}

// Manifests returns a snapshot of every registered manifest, in
// registration order.
func (r *Registry) Manifests() []DomainManifest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]DomainManifest, len(r.manifests))
	copy(out, r.manifests)
	return out
}

// Manifest looks up a single manifest by domain id. Returns (zero,
// false) if not found.
func (r *Registry) Manifest(id DomainID) (DomainManifest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.manifests {
		if m.ID == id {
			return m, true
		}
	}
	return DomainManifest{}, false
}

// ManifestForContributor returns the domain id that declared a given
// contributor id, if any.
func (r *Registry) ManifestForContributor(id ContributorID) (DomainID, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	owner, ok := r.owners[id]
	return owner, ok
}

// SidebarEntries flattens sidebar entries across the effective set.
// The `forDomain` parameter is reserved for active-domain filtering
// once switching-wireup (D4) lands; for now it's accepted but ignored,
// matching the Rust/Swift sides.
func (r *Registry) SidebarEntries(_ *DomainID) []SidebarEntryContrib {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []SidebarEntryContrib
	for _, m := range r.manifests {
		out = append(out, m.SidebarEntries...)
	}
	return out
}

// RightPanelZones flattens right-panel zones across all manifests.
func (r *Registry) RightPanelZones(_ *DomainID) []RightPanelZoneContrib {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []RightPanelZoneContrib
	for _, m := range r.manifests {
		out = append(out, m.RightPanelZones...)
	}
	return out
}

// BottomPanelTypes flattens bottom-panel tab types across all manifests.
func (r *Registry) BottomPanelTypes(_ *DomainID) []BottomPanelTypeContrib {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []BottomPanelTypeContrib
	for _, m := range r.manifests {
		out = append(out, m.BottomPanel...)
	}
	return out
}

// AmbientProducers flattens ambient producers across all manifests.
func (r *Registry) AmbientProducers(_ *DomainID) []AmbientProducerContrib {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []AmbientProducerContrib
	for _, m := range r.manifests {
		out = append(out, m.AmbientProducers...)
	}
	return out
}

// StatusWidgets flattens status-bar widgets across all manifests.
func (r *Registry) StatusWidgets(_ *DomainID) []StatusWidgetContrib {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []StatusWidgetContrib
	for _, m := range r.manifests {
		out = append(out, m.StatusBar...)
	}
	return out
}

// Commands flattens slash commands across all manifests.
func (r *Registry) Commands(_ *DomainID) []CommandContrib {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []CommandContrib
	for _, m := range r.manifests {
		out = append(out, m.SlashCommands...)
	}
	return out
}

// ContributorCount returns the total number of contributors of the
// given type across every registered manifest.
func (r *Registry) ContributorCount(kind ContributorType) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	total := 0
	for _, m := range r.manifests {
		switch kind {
		case ContribSidebarEntry:
			total += len(m.SidebarEntries)
		case ContribRightPanelZone:
			total += len(m.RightPanelZones)
		case ContribBottomPanelType:
			total += len(m.BottomPanel)
		case ContribAmbientProducer:
			total += len(m.AmbientProducers)
		case ContribStatusWidget:
			total += len(m.StatusBar)
		case ContribCommand:
			total += len(m.SlashCommands)
		}
	}
	return total
}

// collectContributorIDs returns every ContributorID declared by a
// manifest across all six surfaces, in a stable order (sidebar →
// right-panel → bottom-panel → ambient → status → command). Used by
// Register for duplicate detection and commit.
func collectContributorIDs(m DomainManifest) []ContributorID {
	total := len(m.SidebarEntries) + len(m.RightPanelZones) + len(m.BottomPanel) +
		len(m.AmbientProducers) + len(m.StatusBar) + len(m.SlashCommands)
	out := make([]ContributorID, 0, total)
	for _, c := range m.SidebarEntries {
		out = append(out, c.ID)
	}
	for _, c := range m.RightPanelZones {
		out = append(out, c.ID)
	}
	for _, c := range m.BottomPanel {
		out = append(out, c.ID)
	}
	for _, c := range m.AmbientProducers {
		out = append(out, c.ID)
	}
	for _, c := range m.StatusBar {
		out = append(out, c.ID)
	}
	for _, c := range m.SlashCommands {
		out = append(out, c.ID)
	}
	return out
}
