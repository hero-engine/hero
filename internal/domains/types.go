// Package domains is the Go mirror of the domain-apps type system.
//
// Source of truth is the Rust crate at
// hero-code/crates/hero-core/src/domains/. Swift (desktop) and Go
// (this package) mirror those definitions; field names and semantics
// match across all three languages.
//
// See:
//   - hero-code/.hero/planning/features/domain-apps-core-types/spec.md
//   - hero-code/.hero/planning/initiatives/hero-app-domain-apps/spec.md
//
// This file holds pure type declarations. Process-start registry +
// duplicate-ContributorId detection live in registry.go.
package domains

// ───────────────────────────────────────────────────────────────────
// ID newtypes — match the Rust newtype shape.
// ───────────────────────────────────────────────────────────────────

// DomainID is a stable identifier for a domain ("chat", "code", "pm", …).
type DomainID string

const (
	DomainCore        DomainID = "core"
	DomainEngineering DomainID = "engineering"
	DomainSales       DomainID = "sales"
	DomainPM          DomainID = "pm"
	DomainQA          DomainID = "qa"
)

// ActivationRole describes how a bundled pack participates in a workspace.
// A workspace has exactly one primary pack and zero or more ordered extension
// packs. Core is implicit and therefore is not a configurable role.
type ActivationRole string

const (
	RolePrimary   ActivationRole = "primary"
	RoleExtension ActivationRole = "extension"
)

// Composition is the committed hero.json shape under the "domains" key.
// Extensions preserve declaration order; resolution removes duplicates while
// retaining the first occurrence.
type Composition struct {
	Primary    DomainID   `json:"primary"`
	Extensions []DomainID `json:"extensions,omitempty"`
}

// ResolvedComposition is the canonical, validated workspace domain stack.
// Core is implicit and always precedes Primary and Extensions.
type ResolvedComposition struct {
	Primary    DomainID
	Extensions []DomainID
}

// Stack returns the complete resolution order: Core, primary, then extensions.
func (c ResolvedComposition) Stack() []DomainID {
	stack := make([]DomainID, 0, 2+len(c.Extensions))
	stack = append(stack, DomainCore, c.Primary)
	stack = append(stack, c.Extensions...)
	return stack
}

// Contains reports whether id participates in the resolved workspace stack.
func (c ResolvedComposition) Contains(id DomainID) bool {
	if id == DomainCore || id == c.Primary {
		return true
	}
	for _, extension := range c.Extensions {
		if extension == id {
			return true
		}
	}
	return false
}

// Pack describes the activation roles supported by an embedded domain pack.
type Pack struct {
	ID      DomainID
	Roles   []ActivationRole
	Bundled bool
}

// ContributorID is a stable, namespaced identifier for any contributor —
// sidebar entry, right-panel zone, bottom-panel tab, ambient producer,
// status widget, or slash command. Must be namespaced under the
// declaring domain id (e.g. "pm.card.story"). The registry rejects
// duplicates at process start.
type ContributorID string

// TabKindRef points at a tab kind owned by the content-area system
// ("chat", "file_editor", "spec", …).
type TabKindRef string

// WorkspaceID is a canonicalised workspace root path.
type WorkspaceID string

// TabID is the persisted-tab identifier used in DomainState.TabOrder.
type TabID string

// SidebarViewID identifies a sidebar view ("sessions", "specs", …).
type SidebarViewID string

// RendererRef names a renderer registered against a ContributorID. The
// renderer-registry slice (D1) owns storage/resolution; this is just
// the manifest-side handle.
type RendererRef string

// ───────────────────────────────────────────────────────────────────
// Layer / display / refs
// ───────────────────────────────────────────────────────────────────

// DomainLayer is the three-layer composition tag — see initiative §1.
type DomainLayer string

const (
	LayerCore      DomainLayer = "core"
	LayerBaseline  DomainLayer = "baseline"
	LayerExtension DomainLayer = "extension"
)

// DomainDisplay carries the chrome-bar metadata for a domain.
// Icon is an SF Symbol name on macOS / a logical key elsewhere.
type DomainDisplay struct {
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// RendererOverride binds a contributor id to a non-default renderer
// while the declaring domain is active. Resolution order: active
// domain override → chat baseline override → default. See initiative §3.
type RendererOverride struct {
	ContributorID ContributorID `json:"contributor_id"`
	Renderer      RendererRef   `json:"renderer"`
}

// AgentSourceKind tags how an agent pack is loaded.
//
// Phase 2 v1 supports "builtin" (compiled into the binary) and
// "project" (scanned from the workspace's .claude/agents/ tree).
// "plugin" and "remote" are declared so the contract is stable but
// are not yet resolvable — the Swift-side resolver surfaces them as
// unsupported_source warnings.
type AgentSourceKind string

const (
	AgentSourceBuiltin AgentSourceKind = "builtin"
	AgentSourceProject AgentSourceKind = "project"
	AgentSourcePlugin  AgentSourceKind = "plugin"
	AgentSourceRemote  AgentSourceKind = "remote"
)

// AgentSource is the tagged source qualifier for an AgentRef. The JSON
// shape matches the Rust serde representation:
//
//	{"kind": "builtin"}
//	{"kind": "project"}
//	{"kind": "plugin", "value": "<plugin-name>"}
//	{"kind": "remote", "value": "<url>"}
type AgentSource struct {
	Kind  AgentSourceKind `json:"kind"`
	Value string          `json:"value,omitempty"`
}

// SkillSourceKind tags how a skill pack is loaded. Same shape as
// AgentSourceKind for now; kept separate so skills can diverge later.
type SkillSourceKind string

const (
	SkillSourceBuiltin SkillSourceKind = "builtin"
	SkillSourceProject SkillSourceKind = "project"
	SkillSourcePlugin  SkillSourceKind = "plugin"
	SkillSourceRemote  SkillSourceKind = "remote"
)

// SkillSource is the tagged source qualifier for a SkillRef.
type SkillSource struct {
	Kind  SkillSourceKind `json:"kind"`
	Value string          `json:"value,omitempty"`
}

// AgentRef points at an agent pack. Full semantics in the agent/skill
// pack contract spec (D2). Discovery and loading are owned by a
// follow-up spec — this is the contract a manifest uses to declare
// its refs.
type AgentRef struct {
	// ID is the stable agent id ("feature-delivery-lead").
	ID string `json:"id"`
	// Source records where the pack is loaded from. Defaults to
	// builtin when omitted from JSON.
	Source AgentSource `json:"source"`
	// VersionConstraint is an optional semver-ish hint; empty means
	// latest-wins.
	VersionConstraint string `json:"version_constraint,omitempty"`
}

// SkillRef points at a skill pack. Full semantics in D2.
type SkillRef struct {
	// ID is the stable skill id ("go-stack").
	ID string `json:"id"`
	// Source records where the pack is loaded from. Defaults to
	// builtin when omitted from JSON.
	Source SkillSource `json:"source"`
	// VersionConstraint is an optional semver-ish hint; empty means
	// latest-wins.
	VersionConstraint string `json:"version_constraint,omitempty"`
}

// ───────────────────────────────────────────────────────────────────
// Contributor structs — manifest-side metadata only.
//
// Go does not carry the SwiftUI body closures; renderer wiring is a
// pure UI concern (Swift / TUI). The Go mirror exists so the backend
// can enumerate manifest contents, serve the settings UX, and
// partition DSKG by domain.
// ───────────────────────────────────────────────────────────────────

// SidebarEntryContrib is a sidebar entry contributed by a domain.
type SidebarEntryContrib struct {
	ID    ContributorID `json:"id"`
	Label string        `json:"label"`
	Icon  string        `json:"icon"`
	Order int           `json:"order"`
}

// RightPanelZoneContrib is a right-panel zone tenant (Detail / Preview / Assist).
type RightPanelZoneContrib struct {
	ID    ContributorID `json:"id"`
	Label string        `json:"label"`
	Icon  string        `json:"icon"`
	Order int           `json:"order"`
}

// BottomPanelTypeContrib is a bottom-panel tab type (terminal, problems, output, …).
type BottomPanelTypeContrib struct {
	ID    ContributorID `json:"id"`
	Label string        `json:"label"`
	Icon  string        `json:"icon"`
	Order int           `json:"order"`
}

// AmbientProducerContrib is a producer of ambient-region cards.
type AmbientProducerContrib struct {
	ID    ContributorID `json:"id"`
	Label string        `json:"label"`
	Icon  string        `json:"icon"`
	Order int           `json:"order"`
}

// StatusWidgetContrib is a status-bar widget (running-agent count, sync state, …).
type StatusWidgetContrib struct {
	ID    ContributorID `json:"id"`
	Label string        `json:"label"`
	Icon  string        `json:"icon"`
	Order int           `json:"order"`
}

// CommandContrib is a slash-command / palette entry.
type CommandContrib struct {
	ID    ContributorID `json:"id"`
	Label string        `json:"label"`
	Icon  string        `json:"icon"`
	Order int           `json:"order"`
}

// ContributorType enumerates every surface a manifest can target.
type ContributorType string

const (
	ContribSidebarEntry    ContributorType = "sidebar_entry"
	ContribRightPanelZone  ContributorType = "right_panel_zone"
	ContribBottomPanelType ContributorType = "bottom_panel_type"
	ContribAmbientProducer ContributorType = "ambient_producer"
	ContribStatusWidget    ContributorType = "status_widget"
	ContribCommand         ContributorType = "command"
)

// ───────────────────────────────────────────────────────────────────
// DomainManifest
// ───────────────────────────────────────────────────────────────────

// DomainManifest is the contract a domain registers against
// DomainRegistry. See initiative spec §2.
type DomainManifest struct {
	ID      DomainID      `json:"id"`
	Display DomainDisplay `json:"display"`
	Layer   DomainLayer   `json:"layer"`
	// Extends is nil for core/chat; "chat" for every extension.
	Extends *DomainID `json:"extends,omitempty"`

	SidebarEntries   []SidebarEntryContrib    `json:"sidebar_entries,omitempty"`
	RightPanelZones  []RightPanelZoneContrib  `json:"right_panel_zones,omitempty"`
	BottomPanel      []BottomPanelTypeContrib `json:"bottom_panel,omitempty"`
	AmbientProducers []AmbientProducerContrib `json:"ambient_producers,omitempty"`
	StatusBar        []StatusWidgetContrib    `json:"status_bar,omitempty"`
	SlashCommands    []CommandContrib         `json:"slash_commands,omitempty"`

	Agents            []AgentRef         `json:"agents,omitempty"`
	Skills            []SkillRef         `json:"skills,omitempty"`
	RendererOverrides []RendererOverride `json:"renderer_overrides,omitempty"`

	// DefaultFirstView is what opens on first visit ("chat",
	// "file_editor", …).
	DefaultFirstView TabKindRef `json:"default_first_view"`

	// DSKGNamespace partitions the project knowledge graph (typically
	// equal to the domain id).
	DSKGNamespace string `json:"dskg_namespace"`
}

// ───────────────────────────────────────────────────────────────────
// Persistence shapes
// ───────────────────────────────────────────────────────────────────

// ToggleState is the per-contributor user override in a
// (workspace, window, domain) scope. "default" defers to the manifest;
// "forced_on" / "forced_off" are explicit user overrides.
type ToggleState string

const (
	ToggleDefault   ToggleState = "default"
	ToggleForcedOn  ToggleState = "forced_on"
	ToggleForcedOff ToggleState = "forced_off"
)

// DomainState is the persisted per-(workspace, window, domain)
// payload. D3 serialises this to window-state.json.
type DomainState struct {
	ContributorToggles map[ContributorID]ToggleState `json:"contributor_toggles,omitempty"`

	SidebarWidth       float64 `json:"sidebar_width,omitempty"`
	RightPanelWidth    float64 `json:"right_panel_width,omitempty"`
	RightPanelVisible  bool    `json:"right_panel_visible,omitempty"`
	BottomPanelVisible bool    `json:"bottom_panel_visible,omitempty"`
	BottomPanelHeight  float64 `json:"bottom_panel_height,omitempty"`

	BottomPanelActiveTab *string        `json:"bottom_panel_active_tab,omitempty"`
	SidebarActiveView    *SidebarViewID `json:"sidebar_active_view,omitempty"`
	LastActiveTabID      *TabID         `json:"last_active_tab_id,omitempty"`
	TabOrder             []TabID        `json:"tab_order,omitempty"`
}

// WindowState is the per-window persistence root.
type WindowState struct {
	WorkspaceID  WorkspaceID              `json:"workspace_id"`
	ActiveDomain DomainID                 `json:"active_domain"`
	DomainStates map[DomainID]DomainState `json:"domain_states,omitempty"`
}
