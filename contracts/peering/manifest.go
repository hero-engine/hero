package peering

import "time"

// PeerManifest is the published surface a Hero workspace exposes to
// its peers. Written to <repo>/.hero/peer-manifest.yaml by
// `hero index` and read across the boundary by a sibling Hero.
//
// Default publish set is empty: only conventions explicitly marked
// peer-surface (via convention frontmatter `peer: true` or via
// hero.json: peering.publish_conventions globs) appear in the
// Conventions list. Same applies to contracts.
type PeerManifest struct {
	// Schema is the manifest schema version. Currently 1.
	Schema int `yaml:"schema" json:"schema"`

	// ContractsVersion is the PeeringContractsVersion at write time —
	// recorded so a reader can detect a shape mismatch.
	ContractsVersion int `yaml:"contracts_version" json:"contracts_version"`

	// Repo identifies the workspace this manifest belongs to.
	Repo RepoIdentity `yaml:"repo" json:"repo"`

	// GeneratedAt is the time `hero index` produced this manifest.
	GeneratedAt time.Time `yaml:"generated_at" json:"generated_at"`

	// Conventions lists every peer-surface convention this workspace
	// publishes. Empty by default — opt-in only.
	Conventions []ConventionEntry `yaml:"conventions,omitempty" json:"conventions,omitempty"`

	// Contracts lists the Go symbols this workspace owns, with their
	// governing convention slug for cross-reference.
	Contracts *ContractsSection `yaml:"contracts,omitempty" json:"contracts,omitempty"`
}

// RepoIdentity is the workspace-identifying block at the top of a
// peer manifest. PeerID is canonical; Name and Display are display.
type RepoIdentity struct {
	// PeerID is the workspace's stable UUID, minted at hero init.
	// This is the canonical identifier across all peering operations.
	PeerID string `yaml:"peer_id" json:"peer_id"`

	// Name is the workspace's local short name (from hero.json:name
	// or directory name). Display form, not a join key.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// Display is an optional human-readable label.
	Display string `yaml:"display,omitempty" json:"display,omitempty"`

	// ScopeHint is a free-form tag describing the workspace's role
	// (e.g. "backend", "web", "desktop").
	ScopeHint string `yaml:"scope_hint,omitempty" json:"scope_hint,omitempty"`
}

// ConventionEntry is one peer-surface convention published in the
// manifest. The path is relative to the workspace's .hero/ directory.
type ConventionEntry struct {
	Slug    string   `yaml:"slug" json:"slug"`
	Title   string   `yaml:"title" json:"title"`
	Surface []string `yaml:"surface,omitempty" json:"surface,omitempty"`
	Path    string   `yaml:"path" json:"path"`
	Digest  string   `yaml:"digest,omitempty" json:"digest,omitempty"`
}

// ContractsSection lists Go symbols this workspace owns. Consumed by
// the contract-import boundary detector (Phase 3 of cross-repo-peering)
// as the primary signal that a session is touching a peer's surface.
type ContractsSection struct {
	Package string          `yaml:"package,omitempty" json:"package,omitempty"`
	Version int             `yaml:"version,omitempty" json:"version,omitempty"`
	Shapes  []ContractEntry `yaml:"shapes,omitempty" json:"shapes,omitempty"`
}

// ContractEntry names a single owned contract shape and its governing
// convention slug.
type ContractEntry struct {
	Kind       string `yaml:"kind" json:"kind"`
	GoSymbol   string `yaml:"go_symbol" json:"go_symbol"`
	Convention string `yaml:"convention,omitempty" json:"convention,omitempty"`
}
