package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultFolder       = ".hero"
	ConfigFileName      = "hero.json"
	LocalConfigFileName = "hero.local.json"
)

var legacyIntegrationWarning sync.Once

func warnLegacyIntegrations() {
	legacyIntegrationWarning.Do(func() {
		fmt.Fprintln(os.Stderr, "hero: legacy tracker/confluence config is deprecated; define stable IDs under integrations.connections (no files were rewritten)")
	})
}

// Config represents the hero.json configuration file.
type Config struct {
	Folder string `json:"folder"`
	// PeerID is the stable UUID identifying this workspace across all
	// peering operations (handoffs, peer calls, manifest, trail
	// entries). Minted at `hero init`; migrated in on first invocation
	// when missing on an older workspace. See contracts/peering for
	// the wire-shape side. Display alias for human reading lives
	// outside Config (registered via `hero repos add` on peers).
	PeerID  string         `json:"peer_id,omitempty"`
	Peering *PeeringConfig `json:"peering,omitempty"`
	Domain  string         `json:"domain,omitempty"`
	Team    *TeamConfig    `json:"team,omitempty"`
	Tracker *TrackerConfig `json:"tracker,omitempty"`
	// Integrations is the canonical provider-neutral integration contract.
	// hero.local.json uses this exact shape and overlays hero.json by stable ID.
	Integrations          *IntegrationsConfig          `json:"integrations,omitempty"`
	IntegrationProvenance map[string]IntegrationSource `json:"-"`
	Import                *ImportConfig                `json:"import,omitempty"`
	Sync                  *SyncConfig                  `json:"sync,omitempty"`
	Conventions           *ConventionConfig            `json:"conventions,omitempty"`
	Serve                 *ServeConfig                 `json:"serve,omitempty"`
	Knowledge             *KnowledgeConfig             `json:"knowledge,omitempty"`
	Jira                  *JiraConfig                  `json:"jira,omitempty"`
	Confluence            *ConfluenceConfig            `json:"confluence,omitempty"`
	Models                *ModelConfig                 `json:"models,omitempty"`
	Mockups               *MockupsConfig               `json:"mockups,omitempty"`
	Prime                 *PrimeConfig                 `json:"prime,omitempty"`
	Hooks                 *HooksConfig                 `json:"hooks,omitempty"`
	Tracking              *TrackingConfig              `json:"tracking,omitempty"`
	Sessions              *SessionsConfig              `json:"sessions,omitempty"`
	Pulse                 *PulseConfig                 `json:"pulse,omitempty"`
	Testing               *TestingConfig               `json:"testing,omitempty"`
	Demos                 *DemosConfig                 `json:"demos,omitempty"`
	CodeScan              *CodeScanConfig              `json:"code_scan,omitempty"`
	Score                 *ScoreConfig                 `json:"score,omitempty"`
	Cloud                 *CloudConfig                 `json:"cloud,omitempty"`
	Next                  *NextConfig                  `json:"next,omitempty"`
	Snapshot              *SnapshotConfig              `json:"snapshot,omitempty"`
	Specs                 *SpecsConfig                 `json:"specs,omitempty"`
	Delivery              *DeliveryConfig              `json:"delivery,omitempty"`
	Verify                *VerifyConfig                `json:"verify,omitempty"`
	Environment           *EnvironmentConfig           `json:"environment,omitempty"`
	// Vocabulary names the active vocabulary preset (e.g. "default",
	// "agile-scrum", "shape-up", "kanban", "jira", "linear"). When set,
	// it wins the precedence chain in internal/vocabulary.Resolve over
	// tracker- and methodology-inferred defaults. Empty falls through
	// to inference.
	Vocabulary string `json:"vocabulary,omitempty"`
	// VocabularyOverrides applies per-key tweaks on top of the resolved
	// vocabulary. Supported key shapes:
	//   types.<type>                  e.g. "types.spec" -> "Story"
	//   kinds.<type>.<kind>            e.g. "kinds.spec.feature" -> "Story"
	//   sections.<canonical>           e.g. "sections.acceptance_criteria" -> "Done When"
	//   lifecycle.<type>.<status>      e.g. "lifecycle.spec.in-flight" -> "Cooking"
	VocabularyOverrides map[string]string `json:"vocabulary_overrides,omitempty"`
	// Methodology names the active methodology profile (e.g. "scrum",
	// "kanban", "shape-up", "waterfall", "scrumban"). When set, it wins
	// the precedence chain in internal/methodology.Resolve over
	// tracker- and delivery-preset-inferred defaults. Empty falls through
	// to inference.
	Methodology string `json:"methodology,omitempty"`
	// MethodologyOverrides applies per-key tweaks on top of the resolved
	// methodology profile. Supported key shapes:
	//   time_boxes.<level>.duration_default   e.g. "time_boxes.iteration.duration_default" -> "3w"
	//   time_boxes.<level>.required           e.g. "time_boxes.iteration.required" -> "false"
	//   estimation.<type>.required_field       e.g. "estimation.feature.required_field" -> "appetite"
	//   in_flight_tracking                     e.g. "in_flight_tracking" -> "wip_aging"
	MethodologyOverrides map[string]string `json:"methodology_overrides,omitempty"`
	// PM holds product-management-specific workspace settings, including
	// the active methodology presets that influence vocabulary
	// auto-selection. The shape mirrors hero-pm's design; only the
	// subset needed for vocabulary resolution is declared here.
	PM    *PMConfig         `json:"pm,omitempty"`
	Repos map[string]string `json:"repos,omitempty"`
	// RepoMeta carries peer-discovery metadata for each entry in Repos,
	// keyed by the same alias. peer_id is the canonical join key for
	// cross-repo peering; the Repos map keeps its alias→path shape
	// untouched for backward compatibility.
	RepoMeta map[string]RepoMetaEntry `json:"repo_meta,omitempty"`
	Content  *ContentConfig           `json:"content,omitempty"`
	// Chat holds chat-dispatcher settings consumed by
	// internal/serve/chat (capability resolver, hero-code probe).
	Chat *ChatConfig `json:"chat,omitempty"`

	// Embeddings controls the semantic embedding layer used for vector
	// search over project content (specs, knowledge, conventions, events,
	// code). Nil means defaults apply (enabled, all corpora, hero-embed-v1).
	Embeddings *EmbeddingsConfig `json:"embeddings,omitempty"`

	// Sprint, when present, opts the workspace into the planned-sprint
	// UI surfaces (the Sprint tab on the Work page, sprint-shaped
	// metric tiles). Absent/empty workspaces never see sprint UI —
	// they run on rolling activity windows instead. Per the
	// hero-serve-dashboard-redesign spec.
	Sprint *SprintConfig `json:"sprint,omitempty"`

	// Roadmap holds settings for ambient roadmap-shape surfacing
	// (NEXT.md, hero_pulse/hero_kickoff, delivery-lead pre-flight).
	// Nil/empty → documented defaults apply.
	// See spec roadmap-review-ambient-surfacing.
	Roadmap *RoadmapConfig `json:"roadmap,omitempty"`
}

// RoadmapConfig tunes the ambient roadmap-shape surfacing helper.
// Both fields are optional; zero falls back to the documented default.
// Negative values are rejected at load time so a typo doesn't silently
// disable the surfacing.
type RoadmapConfig struct {
	// AmbientRecencyDays is the recency window (in days) used by the
	// noise threshold in sizing.AmbientDrift. Specs whose `spec.md`
	// file was committed within this window contribute to the surfaced
	// drift count regardless of horizon. Default: 7.
	AmbientRecencyDays int `json:"ambient_recency_days,omitempty"`
	// StopNaggingHours is the suppression window (in hours) after a
	// `/roadmap-review` session record lands under
	// `.hero/knowledge/roadmap-review-sessions/`. Within this window
	// the ambient surfaces stay quiet unless the filtered drift count
	// has grown above the recorded `drift_count_at_exit`. Default: 24.
	StopNaggingHours int `json:"stop_nagging_hours,omitempty"`
}

// AmbientRecencyDaysOrDefault returns AmbientRecencyDays or 7 if unset.
func (r *RoadmapConfig) AmbientRecencyDaysOrDefault() int {
	if r == nil || r.AmbientRecencyDays <= 0 {
		return 7
	}
	return r.AmbientRecencyDays
}

// StopNaggingHoursOrDefault returns StopNaggingHours or 24 if unset.
func (r *RoadmapConfig) StopNaggingHoursOrDefault() int {
	if r == nil || r.StopNaggingHours <= 0 {
		return 24
	}
	return r.StopNaggingHours
}

// SprintConfig opts a workspace into planned-sprint UI. Presence is
// the gate; the fields below are display sugar for the Sprint tab.
type SprintConfig struct {
	// Name is the human label shown on the Sprint tab.
	Name string `json:"name,omitempty"`
	// Goal is a one-line statement of what this sprint is trying to
	// achieve.
	Goal string `json:"goal,omitempty"`
	// StartedAt is an ISO-8601 date string (e.g. "2026-05-19"). Used to
	// compute the day-counter on the Sprint tab.
	StartedAt string `json:"started_at,omitempty"`
	// Specs is the list of spec slugs scoped to this sprint. When
	// non-empty, the Sprint tab filters its tile counts to this set.
	Specs []string `json:"specs,omitempty"`
}

// HasSprintConfig reports whether a workspace has opted into the
// planned-sprint UI by setting `sprint:` in hero.json. Empty Sprint
// blocks (e.g. `"sprint": {}`) are treated as no sprint — the user has
// to set at least a Name for the tab to surface, mirroring how a real
// agile tracker would model "no current iteration."
func (c Config) HasSprintConfig() bool {
	if c.Sprint == nil {
		return false
	}
	if c.Sprint.Name == "" && c.Sprint.Goal == "" && len(c.Sprint.Specs) == 0 {
		return false
	}
	return true
}

// EmbeddingsConfig controls the semantic embedding layer.
type EmbeddingsConfig struct {
	// Enabled turns semantic embeddings on/off. Default: true when model is available.
	Enabled *bool `json:"enabled,omitempty"`
	// Scope lists which corpora to embed. Default: ["spec", "knowledge", "convention", "event", "code"]
	Scope []string `json:"scope,omitempty"`
	// Model names the Model2Vec model to use. Default: "hero-embed-v1"
	Model string `json:"model,omitempty"`
	// RetrievalSupersedeRespect, when false, skips the vector-path
	// supersede overlay in retrieveHybrid. The lexical path still
	// de-weights superseded specs (parent spec behavior); only the
	// hybrid post-fusion overlay is bypassed. Default: true. Use
	// when score calibration in production looks wrong and a quick
	// rollback is needed; see spec embeddings-superseded-respect.
	RetrievalSupersedeRespect *bool `json:"retrieval_supersede_respect,omitempty"`
}

// IsEmbeddingsEnabled returns true if embeddings are enabled.
// Default: true (if not explicitly disabled).
func (c Config) IsEmbeddingsEnabled() bool {
	if c.Embeddings == nil || c.Embeddings.Enabled == nil {
		return true
	}
	return *c.Embeddings.Enabled
}

// EmbeddingsScope returns the list of corpora to embed.
func (c Config) EmbeddingsScope() []string {
	if c.Embeddings == nil || len(c.Embeddings.Scope) == 0 {
		return []string{"spec", "knowledge", "convention", "event", "code"}
	}
	return c.Embeddings.Scope
}

// EmbeddingsModel returns the configured model name or the default.
func (c Config) EmbeddingsModel() string {
	if c.Embeddings == nil || c.Embeddings.Model == "" {
		return "hero-embed-v1"
	}
	return c.Embeddings.Model
}

// RetrievalSupersedeRespect reports whether the hybrid retrieval path
// should apply the supersede overlay (de-weight + annotation on the post-
// fusion RRF score). Default: true. See spec embeddings-superseded-respect.
func (c Config) RetrievalSupersedeRespect() bool {
	if c.Embeddings == nil || c.Embeddings.RetrievalSupersedeRespect == nil {
		return true
	}
	return *c.Embeddings.RetrievalSupersedeRespect
}

// ChatConfig holds chat-dispatcher settings.
//
// Hero serve never runs inference. The only chat-side configuration
// is which adapter the user prefers (display tiebreak) and where to
// reach hero-code if it's not running on the well-known default.
type ChatConfig struct {
	// PreferredInteractive names the adapter type (e.g. "hero-code",
	// "claude-code-bridge") preferred for interactive turns. Empty
	// falls through to "hero-code first, then first connected".
	PreferredInteractive string `json:"preferred_interactive,omitempty"`

	// Headless holds endpoint settings for the canonical headless
	// adapter (hero-code).
	Headless *ChatHeadlessConfig `json:"headless,omitempty"`
}

// ChatHeadlessConfig holds endpoint settings for the canonical
// headless adapter (hero-code).
type ChatHeadlessConfig struct {
	// Endpoint is where hero-code listens. Recognized shapes:
	//   unix:///path/to/socket
	//   http://host:port
	//   https://host:port
	// Empty disables the proactive probe; hero-code can still
	// register itself by initializing over MCP with a hero_dispatch
	// capability.
	Endpoint string `json:"endpoint,omitempty"`

	// FallbackEndpoint is tried when Endpoint is unreachable.
	FallbackEndpoint string `json:"fallback_endpoint,omitempty"`
}

// RepoMetaEntry holds peer-discovery metadata about a configured
// sibling repo. Written by `hero repos scan` and `hero repos add`
// when the sibling has a peer_id in its hero.json.
type RepoMetaEntry struct {
	PeerID    string `json:"peer_id,omitempty"`
	ScannedAt string `json:"scanned_at,omitempty"`
}

// PeeringConfig holds peer-related workspace settings.
type PeeringConfig struct {
	// Display is an optional human-readable label for this workspace
	// in peer manifests.
	Display string `json:"display,omitempty"`

	// ScopeHint is an optional role tag ("backend", "web", …) for
	// this workspace in peer manifests.
	ScopeHint string `json:"scope_hint,omitempty"`

	// PublishConventions is a list of convention slugs (or glob
	// patterns) that should be marked peer-surface even without the
	// per-convention `peer: true` frontmatter flag. Default empty —
	// publish set is empty by default.
	PublishConventions []string `json:"publish_conventions,omitempty"`

	// Subagent configures the LLM CLI used by `hero peer call` to
	// spawn a subagent in a peer workspace. Nil → use built-in
	// defaults (claude CLI). See SubagentConfig for the contract.
	Subagent *SubagentConfig `json:"subagent,omitempty"`
}

// SubagentConfig configures the local LLM CLI invocation used by
// `hero peer call`. Hero shells out to Command + Args with the peer
// workspace as cwd, pipes the prompt envelope on stdin, and parses a
// structured `<peer-call-result>` block out of stdout.
type SubagentConfig struct {
	// Command is the executable to invoke. Default: "claude".
	Command string `json:"command,omitempty"`

	// Args are passed before any Hero-added flags (budget, prompt
	// piping mode, etc.). Default: ["-p"] (non-interactive print
	// mode).
	Args []string `json:"args,omitempty"`

	// EnvPassthrough is the list of environment variables to forward
	// to the subagent. Default: ["ANTHROPIC_API_KEY", "PATH", "HOME"].
	EnvPassthrough []string `json:"env_passthrough,omitempty"`
}

// ContentConfig declares where canonical agent/command/skill content
// lives in the project. Default (when this block or any field is
// absent): `.hero/agents/`, `.hero/commands/`, `.hero/skills/`. Override
// by setting a project-relative path; install symlinks point at the
// configured location instead of materializing copies under `.hero/`.
//
// Primary use case: tool developers dogfooding their tool on itself,
// where the embedded source dirs ARE the canonical content and the
// rendered-copy step would be a no-op duplication.
type ContentConfig struct {
	AgentsPath   string `json:"agents_path,omitempty"`
	CommandsPath string `json:"commands_path,omitempty"`
	SkillsPath   string `json:"skills_path,omitempty"`
}

// PMConfig holds product-management workspace settings. Only the
// subset consumed by vocabulary resolution is declared today; the
// fuller shape lives in hero-pm and may grow this struct over time.
type PMConfig struct {
	// Presets selects the active methodology presets (delivery cadence,
	// roadmap shape, etc.). The "delivery" value (one of "sprint",
	// "cycle", "continuous", "flow") influences which vocabulary
	// preset is auto-selected when no explicit Vocabulary is set.
	Presets *PMPresets `json:"presets,omitempty"`
}

// PMPresets selects active methodology presets.
type PMPresets struct {
	// Delivery is the active delivery cadence preset. Recognized values
	// today: "sprint" (auto-selects agile-scrum vocabulary), "cycle"
	// (auto-selects shape-up), "continuous"/"flow" (auto-selects
	// kanban). Other values are ignored by vocabulary resolution.
	Delivery string `json:"delivery,omitempty"`
}

// EnvironmentConfig holds environment awareness settings.
type EnvironmentConfig struct {
	CI      *CIConfig      `json:"ci,omitempty"`
	Deploy  *DeployConfig  `json:"deploy,omitempty"`
	Runtime *RuntimeConfig `json:"runtime,omitempty"`
}

// CIConfig holds CI provider settings.
type CIConfig struct {
	Provider string `json:"provider"` // "github-actions", "gitlab-ci", etc.
}

// DeployConfig holds deployment provider settings (future).
type DeployConfig struct {
	Provider     string   `json:"provider"`
	Environments []string `json:"environments,omitempty"`
}

// RuntimeConfig holds runtime observability settings (future).
type RuntimeConfig struct {
	Provider string `json:"provider"`
	Service  string `json:"service,omitempty"`
}

// SpecsConfig holds spec authoring settings.
type SpecsConfig struct {
	// Layout controls the default spec file layout: "single" (default) or "three-file".
	Layout string `json:"layout,omitempty"`
}

// DeliveryConfig holds delivery mode settings.
type DeliveryConfig struct {
	// DefaultMode is the default delivery mode: "supervised" (default), "autopilot", or "dry-run".
	DefaultMode string `json:"default_mode,omitempty"`
	// AutopilotHaltOn is the list of conditions that halt autopilot delivery.
	// Valid: "drift", "test", "boundary", "lint". Default: all.
	AutopilotHaltOn []string `json:"autopilot_halt_on,omitempty"`
	// DryRunWritesPlan controls whether dry-run writes plan.md. Default: true.
	DryRunWritesPlan *bool `json:"dry_run_writes_plan,omitempty"`
	// WIPWarningThreshold is the number of in-flight (status: delivering)
	// specs at which `hero deliver` prints a soft advisory recommending
	// the operator finish one before starting another. 0 or unset → 5.
	// Never a hard block — it warns to stderr and continues.
	WIPWarningThreshold int `json:"wip_warning_threshold,omitempty"`
}

// VerifyConfig holds delivery verification gate settings.
type VerifyConfig struct {
	// RunTests controls whether hero verify runs the test command.
	// Default: true.
	RunTests *bool `json:"run_tests,omitempty"`
	// TestCommand overrides the auto-detected test command.
	// Empty uses stack detection (Go → "go test ./...", etc.).
	TestCommand string `json:"test_command,omitempty"`
}

// RunTestsOrDefault returns RunTests or true if unset.
func (c *VerifyConfig) RunTestsOrDefault() bool {
	if c == nil || c.RunTests == nil {
		return true
	}
	return *c.RunTests
}

// TestCommandOrDefault returns TestCommand or empty (auto-detect) if unset.
func (c *VerifyConfig) TestCommandOrDefault() string {
	if c == nil {
		return ""
	}
	return c.TestCommand
}

// NextConfig holds handoff briefing settings.
type NextConfig struct {
	// Mode controls how NEXT.md briefings are organized.
	// "solo" (default): single shared .hero/NEXT.md file.
	// "team": per-user files in .hero/next/<user>.md.
	Mode string `json:"mode,omitempty"`

	// Projected, when true, switches NEXT.md from agent-authored
	// content to a fresh graph projection each turn. Set automatically
	// by `hero next migrate-to-projection` after that command captures
	// the existing NEXT.md as a graph Note and ingests structured
	// fields. Defaults false so legacy repos keep working unchanged
	// until the user explicitly opts in.
	Projected bool `json:"projected,omitempty"`

	// GoalCapture selects which rungs of the SessionGoal priority ladder
	// run at checkpoint time. "floor" (default) runs the always-on
	// opening-window goal plus the best-effort marker grep and the
	// manual override. "embed" additionally inserts the confidence-gated
	// hero-embed-v1 selector between window and marker. The field is
	// introduced ahead of the embed implementation so enabling it later
	// is a config flip, not a schema change. Empty → "floor".
	GoalCapture string `json:"goal_capture,omitempty"`
}

// SnapshotConfig holds settings for the project-snapshot projector
// and its archive subsystem. See the project-snapshot spec for the
// full design — archives are written by the projector after the live
// SNAPSHOT.md is rendered, governed by the trigger model defined here.
type SnapshotConfig struct {
	// Archive carries the archive-related sub-settings. Nil-safe:
	// readers should call accessor methods that supply defaults.
	Archive *SnapshotArchiveConfig `json:"archive,omitempty"`
}

// SnapshotArchiveConfig controls when archives are written and how
// many are retained on disk. Defaults follow the spec: monthly
// staleness cutoff, milestones on, retention=all.
type SnapshotArchiveConfig struct {
	// StalenessCutoff: weekly | biweekly | monthly | quarterly | off.
	// Empty → "monthly".
	StalenessCutoff string `json:"staleness_cutoff,omitempty"`
	// Milestones enables release-tag / initiative-completion auto
	// archiving. Default true.
	Milestones *bool `json:"milestones,omitempty"`
	// ReleaseTagPattern is a regex tested against new git tag names.
	// Empty → "v[0-9].*".
	ReleaseTagPattern string `json:"release_tag_pattern,omitempty"`
	// Retention: all | last-N | none. Empty → "all".
	Retention string `json:"retention,omitempty"`
	// RetentionCount is the N for "last-N".
	RetentionCount int `json:"retention_count,omitempty"`
}

// CloudConfig holds Hero Cloud sync settings.
type CloudConfig struct {
	OrgID  string `json:"org_id"`
	RepoID string `json:"repo_id"`
}

// PulseConfig holds sprint pulse report settings.
type PulseConfig struct {
	// StaleDeliveringDays is the number of days without a commit before a delivering spec is considered at risk. Default: 3.
	StaleDeliveringDays int `json:"stale_delivering_days"`
	// StalePlanningDays is the number of days before a planning spec is considered at risk. Default: 7.
	StalePlanningDays int `json:"stale_planning_days"`
	// SprintDays is the default sprint window length in days. Default: 14.
	SprintDays int `json:"sprint_days"`
	// SprintStartDay is the day of the week the sprint starts. Default: "monday".
	SprintStartDay string `json:"sprint_start_day"`
}

// TestingConfig holds pluggable test generation settings.
type TestingConfig struct {
	// Framework is the test framework adapter to use. Default: "playwright".
	Framework string `json:"framework"`

	// Mode controls test generation behavior: "agent", "assisted", "autonomous". Default: "autonomous".
	Mode string `json:"mode"`

	// TestDir is the directory for generated test files (relative to project root). Default: "e2e".
	TestDir string `json:"test_dir"`

	// RunnerCommand is the command to run tests. Default: "npx playwright test".
	RunnerCommand string `json:"runner_command"`

	// BaseURL is the application base URL for navigation assertions. Default: "".
	BaseURL string `json:"base_url"`

	// ConfigPath is the path to the framework config file (e.g. playwright.config.ts). Default: "".
	ConfigPath string `json:"config_path"`
}

// FrameworkOrDefault returns Framework or "playwright" if unset.
func (c *TestingConfig) FrameworkOrDefault() string {
	if c == nil || c.Framework == "" {
		return "playwright"
	}
	return c.Framework
}

// ModeOrDefault returns Mode or "autonomous" if unset.
func (c *TestingConfig) ModeOrDefault() string {
	if c == nil || c.Mode == "" {
		return "autonomous"
	}
	return c.Mode
}

// TestDirOrDefault returns TestDir or "e2e" if unset.
func (c *TestingConfig) TestDirOrDefault() string {
	if c == nil || c.TestDir == "" {
		return "e2e"
	}
	return c.TestDir
}

// RunnerCommandOrDefault returns RunnerCommand or "npx playwright test" if unset.
func (c *TestingConfig) RunnerCommandOrDefault() string {
	if c == nil || c.RunnerCommand == "" {
		return "npx playwright test"
	}
	return c.RunnerCommand
}

// VideoSize holds video resolution dimensions.
type VideoSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DemosConfig holds demo recording settings.
type DemosConfig struct {
	// Mode controls when demos are recorded: "auto" or "manual". Default: "manual".
	Mode string `json:"mode"`

	// Framework is the recording framework adapter. Default: "playwright".
	Framework string `json:"framework"`

	// OutputDir is the base directory for demo recordings. Default: ".hero/demos".
	OutputDir string `json:"output_dir"`

	// VideoSize is the video resolution. Default: 1280x720.
	VideoSize *VideoSize `json:"video_size,omitempty"`

	// OnDeliver auto-records when a spec transitions to completed. Default: false.
	OnDeliver bool `json:"on_deliver"`
}

// ModeOrDefault returns Mode or "manual" if unset.
func (c *DemosConfig) ModeOrDefault() string {
	if c == nil || c.Mode == "" {
		return "manual"
	}
	return c.Mode
}

// FrameworkOrDefault returns Framework or "playwright" if unset.
func (c *DemosConfig) FrameworkOrDefault() string {
	if c == nil || c.Framework == "" {
		return "playwright"
	}
	return c.Framework
}

// OutputDirOrDefault returns OutputDir or ".hero/demos" if unset.
func (c *DemosConfig) OutputDirOrDefault() string {
	if c == nil || c.OutputDir == "" {
		return ".hero/demos"
	}
	return c.OutputDir
}

// VideoWidth returns the video width or 1280 if unset.
func (c *DemosConfig) VideoWidth() int {
	if c == nil || c.VideoSize == nil || c.VideoSize.Width == 0 {
		return 1280
	}
	return c.VideoSize.Width
}

// VideoHeight returns the video height or 720 if unset.
func (c *DemosConfig) VideoHeight() int {
	if c == nil || c.VideoSize == nil || c.VideoSize.Height == 0 {
		return 720
	}
	return c.VideoSize.Height
}

// DemosDir returns the absolute path to the demos output directory for a given project root.
func (c Config) DemosDir(projectRoot string) string {
	dir := ".hero/demos"
	if c.Demos != nil {
		dir = c.Demos.OutputDirOrDefault()
	}
	return filepath.Join(projectRoot, dir)
}

// TrackingConfig holds claim tracking settings.
type TrackingConfig struct {
	// StaleClaimDays is the number of days before a claim is considered stale. Default: 2.
	StaleClaimDays int `json:"stale_claim_days"`
	// DefaultAgent is the default agent identity for claims/sessions.
	DefaultAgent string `json:"default_agent"`
}

// StaleClaimDaysOrDefault returns StaleClaimDays or 2 if unset.
func (t *TrackingConfig) StaleClaimDaysOrDefault() int {
	if t == nil || t.StaleClaimDays == 0 {
		return 2
	}
	return t.StaleClaimDays
}

// SessionsConfig holds session log settings.
type SessionsConfig struct {
	// RetentionDays is the number of days to retain session logs. Default: 30.
	RetentionDays int `json:"retention_days"`
	// GitIgnore controls whether session logs are gitignored by default. Default: true.
	GitIgnore bool `json:"gitignore"`
}

// RetentionDaysOrDefault returns RetentionDays or 30 if unset.
func (s *SessionsConfig) RetentionDaysOrDefault() int {
	if s == nil || s.RetentionDays == 0 {
		return 30
	}
	return s.RetentionDays
}

// HooksConfig holds git hook integration settings.
type HooksConfig struct {
	// BranchPatterns are the patterns used to match branch names to spec slugs.
	// Use {{slug}} as the placeholder. Default: ["feat/{{slug}}", "feature/{{slug}}", "fix/{{slug}}", "{{slug}}"].
	BranchPatterns []string `json:"branch_patterns"`

	// SlugTransform controls how branch names are transformed into slugs.
	// Options: "kebab" (default).
	SlugTransform string `json:"slug_transform"`

	// RevertOnDelete controls whether specs are reverted to planning when their
	// branch is deleted. Default: false.
	RevertOnDelete bool `json:"revert_on_delete"`

	// InjectCommitPrefix controls whether the spec slug is prepended to commit
	// messages via the prepare-commit-msg hook. Default: false.
	InjectCommitPrefix bool `json:"inject_commit_prefix"`

	// StatusTruth, when true, makes the pre-commit hook run hero check
	// status and BLOCK the commit if any spec is lying or partial.
	// Opt-in: set to true to enable. Off by default — silent on most
	// repos, vocal on those that have signed up for verification.
	// (spec-status-integrity AC-5)
	StatusTruth bool `json:"status_truth"`
}

// ImportConfig holds default settings for `hero import` and inventory generation.
type ImportConfig struct {
	// DefaultType is the spec type for imported issues: "feature" or "bug" (default: "feature").
	DefaultType string `json:"default_type"`

	// Limit is the default maximum number of issues to fetch (default: 30, max: 100).
	Limit int `json:"limit"`

	// BaseFilter defines the base query that always applies when importing issues.
	// This sets the "what kind of issues does this project work on" defaults.
	// When not configured, defaults to: issue_type=Bug, assignee=unassigned, status=New.
	// CLI flags and Filter/Preset fields override individual base filter fields.
	BaseFilter *ImportFilter `json:"base_filter,omitempty"`

	// Filter holds additional query filters applied on top of the base filter.
	// CLI flags override these. Use this for narrowing beyond the base (e.g. labels, priority).
	Filter *ImportFilter `json:"filter,omitempty"`

	// Presets holds named filter presets that can be referenced via `hero import --preset <name>`
	// or `/import <name>`. Each preset is a complete ImportFilter. When a preset is active,
	// it replaces the default filter — CLI flags still override individual fields.
	Presets map[string]*ImportFilter `json:"presets,omitempty"`

	// ByType holds per-spec-type filters, keyed by hero spec type
	// ("bug", "feature", "story", "epic", "initiative"). The effective
	// filter for a type is base_filter merged with by_type[type], where
	// by_type overrides base per non-empty field. A plain `hero sync
	// import` (no positional/--type arg) runs each type's effective
	// filter and unions the results (dedup by external id); a scoped
	// import (`hero import bugs`) runs only that type's effective filter.
	// Precedence sits between --preset and base_filter: --jql > --filter
	// > CLI field flags > --preset > by_type > base_filter defaults.
	ByType map[string]*ImportFilter `json:"by_type,omitempty"`

	// Inventory controls whether an inventory report is generated alongside imports.
	// Default: true.
	Inventory *bool `json:"inventory,omitempty"`

	// InventoryPath is the path (relative to .hero/planning/) where the inventory
	// report is written. Default: "bugs/inventory.md" when DefaultType is "bug",
	// "features/inventory.md" otherwise.
	InventoryPath string `json:"inventory_path"`

	// AutoRefresh enables periodic background re-import and status sync when
	// hero serve is running. Default: false.
	AutoRefresh bool `json:"auto_refresh"`

	// RefreshInterval is the interval between automatic refreshes (e.g. "30m", "1h").
	// Default: "30m". Only used when AutoRefresh is true.
	RefreshInterval string `json:"refresh_interval"`

	// OnRefresh controls what happens when a refresh detects tracker-side changes
	// to previously imported issues. See RefreshBehavior.
	OnRefresh *RefreshBehavior `json:"on_refresh,omitempty"`
}

// RefreshBehavior controls how import refresh handles tracker-side changes.
type RefreshBehavior struct {
	// UpdateStatus syncs the tracker issue status back into the local spec's
	// frontmatter (equivalent to bulk `hero pull`). Default: true.
	UpdateStatus *bool `json:"update_status,omitempty"`

	// MarkReassigned adds a "reassigned" tag and clears claimed_by when the
	// tracker issue is assigned to someone else. Default: true.
	MarkReassigned *bool `json:"mark_reassigned,omitempty"`

	// RemoveResolved controls what happens to local specs whose tracker issues
	// are resolved/closed. Options: "mark" (set status to completed),
	// "archive" (move to specs/), "keep" (leave as-is). Default: "mark".
	RemoveResolved string `json:"remove_resolved"`
}

// ShouldUpdateStatus returns whether status should be synced from tracker on refresh.
func (r *RefreshBehavior) ShouldUpdateStatus() bool {
	if r == nil || r.UpdateStatus == nil {
		return true
	}
	return *r.UpdateStatus
}

// ShouldMarkReassigned returns whether reassigned issues should be tagged.
func (r *RefreshBehavior) ShouldMarkReassigned() bool {
	if r == nil || r.MarkReassigned == nil {
		return true
	}
	return *r.MarkReassigned
}

// ResolvedAction returns the action to take for resolved tracker issues.
func (r *RefreshBehavior) ResolvedAction() string {
	if r == nil || r.RemoveResolved == "" {
		return "mark"
	}
	return r.RemoveResolved
}

// ImportFilter holds default query filters for issue import.
// Precedence: JQL > FilterID > individual field filters.
type ImportFilter struct {
	// JQL is a raw Jira Query Language string. When set, overrides all other filters.
	// Only applicable to Jira trackers.
	JQL string `json:"jql"`

	// FilterID is a saved Jira filter ID. When set, overrides field-level filters.
	// Only applicable to Jira trackers.
	FilterID string `json:"filter_id"`

	// IssueType filters by tracker issue type (e.g. "Bug", "Story", "Task").
	IssueType string `json:"issue_type"`

	// Assignee filters by assignee. Use "unassigned" or "none" to get only
	// unassigned issues. Use a display name or email for a specific person.
	Assignee string `json:"assignee"`

	// Labels filters by one or more labels (AND logic on Jira, comma-separated on GitHub).
	Labels []string `json:"labels"`

	// Status filters by tracker-native status (e.g. "New", "Open", "To Do").
	// By default, completed/done issues are excluded.
	Status string `json:"status"`

	// Priority filters by priority name (e.g. "Critical", "High", "Medium").
	Priority string `json:"priority"`

	// OrderBy controls sort order. Default: "created DESC".
	// Jira: JQL ORDER BY clause value. GitHub: "created", "updated", "comments".
	OrderBy string `json:"order_by"`
}

// InventoryEnabled returns whether inventory report generation is enabled.
// Defaults to true if not explicitly set.
func (c *ImportConfig) InventoryEnabled() bool {
	if c == nil || c.Inventory == nil {
		return true
	}
	return *c.Inventory
}

// EffectiveBaseFilter returns the base filter with defaults applied.
// If no base_filter is configured, returns {IssueType: "Bug", Assignee: "unassigned", Status: "New"}.
// Configured fields override the defaults; empty string means "use default".
func (c *ImportConfig) EffectiveBaseFilter() *ImportFilter {
	defaults := &ImportFilter{
		IssueType: "Bug",
		Assignee:  "unassigned",
		Status:    "New",
	}
	if c == nil || c.BaseFilter == nil {
		return defaults
	}
	bf := c.BaseFilter
	if bf.IssueType != "" {
		defaults.IssueType = bf.IssueType
	}
	if bf.Assignee != "" {
		defaults.Assignee = bf.Assignee
	}
	if bf.Status != "" {
		defaults.Status = bf.Status
	}
	// Pass through non-defaulted fields as-is
	if bf.JQL != "" {
		defaults.JQL = bf.JQL
	}
	if bf.FilterID != "" {
		defaults.FilterID = bf.FilterID
	}
	if len(bf.Labels) > 0 {
		defaults.Labels = bf.Labels
	}
	if bf.Priority != "" {
		defaults.Priority = bf.Priority
	}
	if bf.OrderBy != "" {
		defaults.OrderBy = bf.OrderBy
	}
	return defaults
}

// mergeFilter overlays non-empty fields from over onto a copy of base
// and returns the merged filter. over wins per-field; base fields
// survive where over leaves them empty. Neither input is mutated. A nil
// base is treated as an empty filter; a nil over returns a copy of base.
func mergeFilter(base, over *ImportFilter) *ImportFilter {
	out := &ImportFilter{}
	if base != nil {
		*out = *base
	}
	if over == nil {
		return out
	}
	if over.JQL != "" {
		out.JQL = over.JQL
	}
	if over.FilterID != "" {
		out.FilterID = over.FilterID
	}
	if over.IssueType != "" {
		out.IssueType = over.IssueType
	}
	if over.Assignee != "" {
		out.Assignee = over.Assignee
	}
	if len(over.Labels) > 0 {
		out.Labels = over.Labels
	}
	if over.Status != "" {
		out.Status = over.Status
	}
	if over.Priority != "" {
		out.Priority = over.Priority
	}
	if over.OrderBy != "" {
		out.OrderBy = over.OrderBy
	}
	return out
}

// HasByType reports whether any per-type filters are configured.
func (c *ImportConfig) HasByType() bool {
	return c != nil && len(c.ByType) > 0
}

// ByTypeKeys returns the spec-type keys that have a by_type entry, in no
// particular order. Empty when none are configured.
func (c *ImportConfig) ByTypeKeys() []string {
	if c == nil {
		return nil
	}
	keys := make([]string, 0, len(c.ByType))
	for k := range c.ByType {
		keys = append(keys, k)
	}
	return keys
}

// EffectiveFilterForType returns the effective filter for a spec type:
// the effective base filter merged with by_type[specType] (by_type
// overrides base per non-empty field). When no by_type entry exists for
// the type, this equals EffectiveBaseFilter — so callers get sane
// defaults for every type regardless of whether it's enumerated in
// by_type. specType is matched case-insensitively.
func (c *ImportConfig) EffectiveFilterForType(specType string) *ImportFilter {
	base := c.EffectiveBaseFilter()
	if c == nil || len(c.ByType) == 0 {
		return base
	}
	var over *ImportFilter
	for k, v := range c.ByType {
		if strings.EqualFold(k, specType) {
			over = v
			break
		}
	}
	return mergeFilter(base, over)
}

// EffectiveInventoryPath returns the inventory report path, applying defaults.
func (c *ImportConfig) EffectiveInventoryPath(specType string) string {
	if c != nil && c.InventoryPath != "" {
		return c.InventoryPath
	}
	switch specType {
	case "bug":
		return "bugs/inventory.md"
	case "feature":
		return "features/inventory.md"
	default:
		return "inventory.md"
	}
}

// ServeConfig holds hero serve daemon settings.
type ServeConfig struct {
	Port      int   `json:"port"`         // HTTP port (default: 7437)
	AutoWatch bool  `json:"auto_watch"`   // auto-watch for file changes (default: true)
	UI        *bool `json:"ui,omitempty"` // serve the embedded dashboard UI (default: true)

	// HealthTTL controls how long the /p/<slug>/project Health section
	// trusts a cached `hero check` result before rendering it as
	// "stale". Phase 5 of hero-serve-project-section. Format accepts
	// any time.Duration string ("5m", "30s", "1h"). Empty or invalid
	// values fall back to defaultHealthTTL (5 minutes).
	HealthTTL string `json:"health_ttl,omitempty"`

	// ToolFilter controls which MCP tools are exposed to clients.
	// If empty, all tools are exposed.
	ToolFilter *MCPToolFilter `json:"tool_filter,omitempty"`
}

// defaultHealthTTL is the fallback TTL when serve.health_ttl is absent
// or malformed. Picked to match the parent spec's "5-minute" default.
const defaultHealthTTL = 5 * time.Minute

// HealthTTLDuration returns the parsed health TTL, falling back to the
// 5-minute default for empty or invalid values. Never returns an error —
// a bad config string should not crash the daemon (the dashboard just
// uses the default and the operator sees the cached "stale" chip
// according to that schedule).
func (c *ServeConfig) HealthTTLDuration() time.Duration {
	if c == nil || c.HealthTTL == "" {
		return defaultHealthTTL
	}
	d, err := time.ParseDuration(c.HealthTTL)
	if err != nil || d <= 0 {
		return defaultHealthTTL
	}
	return d
}

// UIEnabled returns whether the dashboard UI should be served.
// Defaults to true if not explicitly set.
func (c *ServeConfig) UIEnabled() bool {
	if c == nil || c.UI == nil {
		return true
	}
	return *c.UI
}

// MCPToolFilter defines rules for filtering the MCP tool registry.
type MCPToolFilter struct {
	// Allow is an explicit allowlist of tool names. If non-empty, only these tools are exposed.
	Allow []string `json:"allow,omitempty"`

	// Deny is an explicit denylist of tool names. Denied tools are hidden even if in Allow.
	Deny []string `json:"deny,omitempty"`

	// Profiles maps named profiles (e.g. "readonly", "full") to lists of allowed tools.
	// Clients can request a profile via the X-Hero-Profile header.
	Profiles map[string][]string `json:"profiles,omitempty"`
}

// TeamConfig holds team coordination settings.
type TeamConfig struct {
	RequireReview bool   `json:"require_review"`
	StaleDays     int    `json:"stale_days"`
	AutoContext   bool   `json:"auto_context"`
	NudgeLevel    string `json:"nudge_level"` // off, gentle, assertive
}

// TrackerConfig holds work tracker integration settings.
type TrackerConfig struct {
	Type          string `json:"type"`            // github, jira, linear, gitlab, none
	Project       string `json:"project"`         // project identifier (e.g. "owner/repo" for GitHub, project key for Jira, team key for Linear, "namespace/project" or numeric ID for GitLab)
	Token         string `json:"token,omitempty"` // literal API token (set in hero.local.json or credentials store; never in hero.json)
	TokenEnv      string `json:"token_env"`       // env var name holding the API token (e.g. "GITHUB_TOKEN")
	BaseURL       string `json:"base_url"`        // API base URL (required for Jira and GitLab, optional override for GitHub/Linear)
	UserEmail     string `json:"user_email"`      // user email for services requiring basic auth (e.g. Jira Cloud, Confluence Cloud)
	PostOnDesign  bool   `json:"post_on_design"`  // create issue when spec enters design
	PostOnDeliver bool   `json:"post_on_deliver"`
	// SizeMapping configures bidirectional sync between Hero's declared
	// `size:` frontmatter (6-tier ladder) and a tracker-side field
	// (e.g. Jira story_points, GitHub size/* label). Absent → no
	// mapping; sync push/pull leave size alone. See the
	// spec-size-and-promotion-nudge spec for the contract.
	SizeMapping *SizeMappingConfig `json:"size_mapping,omitempty"`
}

// SizeMappingConfig declares how Hero's local size tier maps to a
// tracker-side field. Non-destructive by design — the configured field
// is read and written, but conflicts are surfaced rather than silently
// overwritten.
type SizeMappingConfig struct {
	// Field is the tracker-side field name that carries leaf-tier size
	// (e.g. "story_points" for Jira, "estimate" for Linear, or a label
	// prefix like "size/" for GitHub). Required when SizeMapping is set.
	Field string `json:"field"`
	// Thresholds maps Hero tier names to inclusive numeric bands
	// [min, max] for numeric fields. A nil upper bound means
	// "unbounded". For label-style fields (GitHub size/*) the band is
	// not consulted on the way out — the tier name is used directly —
	// but inverse-mapping (label → tier) still reads it for sanity.
	// Tier keys must be drawn from the 6-tier ladder:
	//   trivial / small / medium / large / x-large / giant
	Thresholds map[string][]*float64 `json:"thresholds,omitempty"`
	// ContainerField is the tracker-side field carrying epic/initiative
	// container size (e.g. Jira epic label, Linear project field).
	// Optional — empty means container size is local-only.
	ContainerField string `json:"container_field,omitempty"`
}

// Validate checks the SizeMappingConfig for the obvious structural
// errors at load time. Empty SizeMapping is fine (see TrackerConfig
// docs); callers should nil-check first.
func (s *SizeMappingConfig) Validate() error {
	if s == nil {
		return nil
	}
	if s.Field == "" {
		return fmt.Errorf("tracker.size_mapping.field is required")
	}
	if len(s.Thresholds) == 0 {
		return fmt.Errorf("tracker.size_mapping.thresholds must cover at least one tier")
	}
	validTiers := map[string]struct{}{
		"trivial": {}, "small": {}, "medium": {},
		"large": {}, "x-large": {}, "giant": {},
	}
	for tier, band := range s.Thresholds {
		if _, ok := validTiers[tier]; !ok {
			return fmt.Errorf("tracker.size_mapping.thresholds: unknown tier %q (valid: trivial small medium large x-large giant)", tier)
		}
		if len(band) != 2 {
			return fmt.Errorf("tracker.size_mapping.thresholds[%q]: band must be [min, max] (max may be null for unbounded)", tier)
		}
	}
	return nil
}

// ResolveToken returns the API token for this tracker.
// Priority: Token field (from local config / credentials) > TokenEnv env var.
func (t *TrackerConfig) ResolveToken() (string, error) {
	if t == nil {
		return "", fmt.Errorf("no usable tracker credential; add integrations.connections.<id>.auth.token to .hero/hero.local.json, configure a stable-ID global credential, or set auth.token_env; inspect with 'hero connect --list'")
	}
	if t.Token != "" {
		return t.Token, nil
	}
	if t.TokenEnv == "" {
		return "", fmt.Errorf("no usable tracker credential; add integrations.connections.<id>.auth.token to .hero/hero.local.json, configure a stable-ID global credential, or set auth.token_env; inspect with 'hero connect --list'")
	}
	token := os.Getenv(t.TokenEnv)
	if token == "" {
		return "", fmt.Errorf("environment variable %q is not set", t.TokenEnv)
	}
	return token, nil
}

// SyncConfig holds wiki/docs sync settings.
type SyncConfig struct {
	Target string `json:"target"` // github-wiki, confluence, none
	Auto   bool   `json:"auto"`
}

// ConventionConfig holds convention enforcement settings.
type ConventionConfig struct {
	Enforce      bool   `json:"enforce"`
	ScopeDefault string `json:"scope_default"`
}

// KnowledgeConfig holds knowledge base settings.
type KnowledgeConfig struct {
	AutoCapture bool `json:"auto_capture"` // auto-capture learnings at end of workflows (default: true)
	// ExplainerSynthesis controls synthesis autonomy for the
	// feature-knowledge-synthesis trust handshake: "auto" | "review" | "off".
	// Default "review" (propose, don't auto-write). Empty is treated as the
	// default by ExplainerSynthesisMode.
	ExplainerSynthesis string `json:"explainer_synthesis,omitempty"`
}

// ScoreConfig holds spec quality scoring settings.
type ScoreConfig struct {
	// MinScore is the minimum score (0-100) required for delivery. Default: 40.
	MinScore int `json:"min_score"`
}

// CodeScanConfig holds code intelligence scanning settings.
type CodeScanConfig struct {
	// Depth controls analysis depth: "normal" (structure only), "deep" (structure + LLM descriptions), "disabled".
	// Default: "normal".
	Depth string `json:"depth"`

	// Parser selects the parsing backend: "auto" (tree-sitter if available, else heuristic),
	// "heuristic" (pure Go regex), "treesitter" (requires tree-sitter CLI on PATH).
	// Default: "auto".
	Parser string `json:"parser"`

	// Exclude is a list of directory names to skip during code scanning.
	// Default: ["vendor", "node_modules", "dist", "build", ".git", "generated", "third_party"].
	Exclude []string `json:"exclude,omitempty"`
}

// DefaultCodeScanConfig returns a CodeScanConfig with sensible defaults.
func DefaultCodeScanConfig() *CodeScanConfig {
	return &CodeScanConfig{
		Depth:  "normal",
		Parser: "auto",
		Exclude: []string{
			// Package managers / dependencies
			"node_modules", "vendor", "bower_components",
			"jspm_packages", ".pnpm", ".yarn",
			// Build output
			"dist", "build", "out", "output", ".output",
			"target", "bin", "obj",
			// Generated / cached
			"generated", "third_party", "__generated__",
			"__pycache__", ".cache", ".parcel-cache",
			// Virtual envs
			".venv", "venv", "env", ".env",
			// Framework dirs
			".next", ".nuxt", ".svelte-kit", ".angular",
			// VCS / IDE
			".git", ".hg", ".svn",
			".idea", ".vscode",
			// Test coverage / reports
			"coverage", ".nyc_output", "htmlcov",
			// Temp
			"tmp", "temp", "logs",
		},
	}
}

// IsDisabled returns true if code scanning is disabled.
func (c *CodeScanConfig) IsDisabled() bool {
	return c != nil && c.Depth == "disabled"
}

// IsDeep returns true if deep (LLM-enhanced) scanning is enabled.
func (c *CodeScanConfig) IsDeep() bool {
	return c != nil && c.Depth == "deep"
}

// ShouldExclude returns true if the given directory name should be skipped.
func (c *CodeScanConfig) ShouldExclude(dirName string) bool {
	if c == nil {
		return false
	}
	for _, ex := range c.Exclude {
		if ex == dirName {
			return true
		}
	}
	return false
}

// JiraConfig holds advanced Jira integration settings beyond basic tracker config.
type JiraConfig struct {
	// EpicLinkField is the custom field ID used for epic links (e.g. "customfield_10014").
	// Defaults to "customfield_10014" which is standard for Jira Cloud.
	EpicLinkField string `json:"epic_link_field"`

	// SprintField is the custom field ID for sprint membership (e.g. "customfield_10020").
	SprintField string `json:"sprint_field"`

	// StoryPointsField is the custom field ID for story points (e.g. "customfield_10016").
	// Defaults to "customfield_10016" which is standard for Jira Cloud.
	StoryPointsField string `json:"story_points_field"`

	// AcceptanceCriteriaField is the custom field ID for acceptance criteria.
	// No default — this varies widely between Jira instances. When empty,
	// acceptance criteria will not be imported from sprint items.
	AcceptanceCriteriaField string `json:"acceptance_criteria_field"`

	// SeverityField is the custom field ID for severity (e.g. "customfield_10100").
	// Deprecated: use CustomFields instead. Kept for backward compat; if set, it's
	// treated as CustomFields: {"severity": "customfield_10100"}.
	SeverityField string `json:"severity_field,omitempty"`

	// CustomFields maps human-readable field names to Jira custom field IDs.
	// Names are case-insensitive. Values must be Jira custom field IDs like
	// "customfield_10100". If a name is specified without an ID (empty string),
	// Hero auto-discovers the ID via the Jira field API on first use and writes
	// the resolved mapping back to hero.json for future runs.
	//
	// Hero also auto-discovers common severity-like fields (Severity, Criticality,
	// Impact, etc.) without explicit configuration. Any discovered fields are
	// merged into this map and persisted.
	//
	// Example:
	//   "custom_fields": {
	//     "Severity": "customfield_10100",
	//     "Customer Impact": ""
	//   }
	CustomFields map[string]string `json:"custom_fields,omitempty"`

	// PushStatusTransitions maps Hero status names to Jira transition IDs.
	// Example: {"delivering": "31", "completed": "41"}
	PushStatusTransitions map[string]string `json:"push_status_transitions"`

	// ImportEpics controls whether epic hierarchy is imported on sprint load.
	ImportEpics bool `json:"import_epics"`
}

// ConfluenceConfig holds Confluence wiki sync settings.
type ConfluenceConfig struct {
	// BaseURL is the Confluence instance URL (e.g. "https://mycompany.atlassian.net/wiki").
	BaseURL string `json:"base_url"`

	// SpaceKey is the Confluence space key to sync into (e.g. "ENG").
	SpaceKey string `json:"space_key"`

	// Token is the literal Confluence API token or PAT.
	// Set in hero.local.json or the credentials store; never in hero.json.
	Token string `json:"token,omitempty"`

	// TokenEnv is the env var holding the Confluence API token or PAT.
	TokenEnv string `json:"token_env"`

	// UserEmail is the email address for Confluence Cloud basic-auth (token auth requires email).
	UserEmail string `json:"user_email"`

	// ParentPageTitle is the optional parent page under which Hero pages are created.
	// Defaults to "Hero Specs".
	ParentPageTitle string `json:"parent_page_title"`

	// LabelPrefix is prepended to all labels applied to synced pages (e.g. "hero-").
	LabelPrefix string `json:"label_prefix"`
}

// ResolveToken returns the API token for Confluence.
// Priority: Token field (from local config / credentials) > TokenEnv env var.
func (c *ConfluenceConfig) ResolveToken() (string, error) {
	if c == nil {
		return "", fmt.Errorf("confluence token_env is not configured")
	}
	if c.Token != "" {
		return c.Token, nil
	}
	if c.TokenEnv == "" {
		return "", fmt.Errorf("confluence token_env is not configured")
	}
	token := os.Getenv(c.TokenEnv)
	if token == "" {
		return "", fmt.Errorf("environment variable %q is not set", c.TokenEnv)
	}
	return token, nil
}

// MockupsConfig holds renderer-selection settings for `/mock`. Read by
// `hero mock detect`; overrides auto-detect but not an explicit
// `--renderer` flag. Empty Renderer means "no override".
type MockupsConfig struct {
	// Renderer pins the renderer used by `/mock` regardless of stack
	// auto-detection. Valid values: "html", "swiftui". Empty/unset is
	// treated as "no override" — auto-detect runs.
	Renderer string `json:"renderer,omitempty"`
}

// ModelConfig holds model role configuration for the Hero workflow.
type ModelConfig struct {
	// Roles maps role names (e.g. "design", "execution", "review") to model identifiers.
	// The model identifier is passed through to the agent layer as a hint.
	// Example: {"design": "claude-opus-4", "execution": "claude-sonnet-4-5", "review": "o3"}
	Roles map[string]string `json:"roles"`

	// DefaultModel is used when no role-specific model is configured.
	DefaultModel string `json:"default_model"`
}

// PrimeConfig controls auto-prime behavior for MCP sessions.
type PrimeConfig struct {
	// Auto enables automatic context injection into MCP Instructions at session start.
	// When true, handleInitialize returns dynamic project context.
	// Default: true.
	Auto *bool `json:"auto,omitempty"`

	// IncludeKnowledge includes conventions and decisions in the auto-prime context.
	// Default: true.
	IncludeKnowledge *bool `json:"include_knowledge,omitempty"`
}

// AutoEnabled returns whether auto-prime is enabled. Defaults to true.
func (c *PrimeConfig) AutoEnabled() bool {
	if c == nil || c.Auto == nil {
		return true
	}
	return *c.Auto
}

// KnowledgeEnabled returns whether knowledge entries should be included. Defaults to true.
func (c *PrimeConfig) KnowledgeEnabled() bool {
	if c == nil || c.IncludeKnowledge == nil {
		return true
	}
	return *c.IncludeKnowledge
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Folder: DefaultFolder,
		Team: &TeamConfig{
			RequireReview: false,
			StaleDays:     14,
			AutoContext:   true,
			NudgeLevel:    "gentle",
		},
		Tracker: &TrackerConfig{
			Type:          "none",
			Project:       "",
			TokenEnv:      "",
			BaseURL:       "",
			PostOnDesign:  false,
			PostOnDeliver: false,
		},
		Sync: &SyncConfig{
			Target: "none",
			Auto:   false,
		},
		Conventions: &ConventionConfig{
			Enforce:      true,
			ScopeDefault: "*",
		},
		Knowledge: &KnowledgeConfig{
			AutoCapture: true,
		},
		CodeScan: DefaultCodeScanConfig(),
	}
}

// Load reads hero.json from the given project root directory.
// Returns default config if the file doesn't exist.
// After loading hero.json, it deep-merges hero.local.json on top (if present).
func Load(projectRoot string) (Config, error) {
	cfg := DefaultConfig()

	heroDir := filepath.Join(projectRoot, cfg.Folder)
	configPath := filepath.Join(heroDir, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Still try to apply local overrides and credentials
			local, lerr := LoadLocal(projectRoot, cfg.Folder)
			if lerr == nil {
				cfg = MergeLocal(cfg, local)
			}
			localPath := filepath.Join(heroDir, LocalConfigFileName)
			localData, _ := os.ReadFile(localPath)
			if hasLegacyIntegration(localData) {
				warnLegacyIntegrations()
			}
			if resolved, rerr := ResolveIntegrationDocuments(configPath, nil, localPath, localData); rerr != nil {
				return cfg, rerr
			} else if resolved != nil {
				cfg.Integrations = resolved.Config
				cfg.IntegrationProvenance = resolved.Provenance
				if tc, ok := resolved.DeliveryTracker(); ok {
					cfg.Tracker = tc
				}
				if cc, ok := resolved.DocsConfluence(); ok {
					cfg.Confluence = cc
				}
			}
			if creds, cerr := LoadCredentials(); cerr == nil {
				var aerr error
				cfg, aerr = ApplyCredentialsStrict(cfg, creds)
				if aerr != nil {
					return cfg, aerr
				}
			}
			return cfg, nil
		}
		return cfg, err
	}
	if hasLegacyIntegration(data) {
		warnLegacyIntegrations()
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", configPath, err)
	}
	localPath := filepath.Join(heroDir, LocalConfigFileName)
	localData, localReadErr := os.ReadFile(localPath)
	if localReadErr != nil && !errors.Is(localReadErr, os.ErrNotExist) {
		return cfg, localReadErr
	}
	if hasLegacyIntegration(localData) {
		warnLegacyIntegrations()
	}
	resolved, ierr := ResolveIntegrationDocuments(configPath, data, localPath, localData)
	if ierr != nil {
		return cfg, ierr
	}
	if resolved != nil {
		cfg.Integrations = resolved.Config
		cfg.IntegrationProvenance = resolved.Provenance
		if tc, ok := resolved.DeliveryTracker(); ok {
			cfg.Tracker = tc
		}
		if cc, ok := resolved.DocsConfluence(); ok {
			cfg.Confluence = cc
		}
	}

	if cfg.Folder == "" {
		cfg.Folder = DefaultFolder
	}

	// Ensure Team is populated
	if cfg.Team == nil {
		cfg.Team = &TeamConfig{StaleDays: 14, AutoContext: true, NudgeLevel: "gentle"}
	}
	if cfg.Team.NudgeLevel == "" {
		cfg.Team.NudgeLevel = "gentle"
	}
	if cfg.Conventions == nil {
		cfg.Conventions = &ConventionConfig{Enforce: true, ScopeDefault: "*"}
	}
	if cfg.Knowledge == nil {
		cfg.Knowledge = &KnowledgeConfig{AutoCapture: true}
	}
	if cfg.CodeScan == nil {
		cfg.CodeScan = DefaultCodeScanConfig()
	} else {
		// Fill in defaults for fields not specified
		defaults := DefaultCodeScanConfig()
		if cfg.CodeScan.Depth == "" {
			cfg.CodeScan.Depth = defaults.Depth
		}
		if cfg.CodeScan.Parser == "" {
			cfg.CodeScan.Parser = defaults.Parser
		}
		if len(cfg.CodeScan.Exclude) == 0 {
			cfg.CodeScan.Exclude = defaults.Exclude
		}
	}

	// Validate roadmap ambient-surfacing config: reject negative values
	// so a typo doesn't silently disable the surfacing. Zero/unset is
	// fine — defaults apply.
	if cfg.Roadmap != nil {
		if cfg.Roadmap.AmbientRecencyDays < 0 {
			return cfg, fmt.Errorf("parsing %s: roadmap.ambient_recency_days must be >= 0", configPath)
		}
		if cfg.Roadmap.StopNaggingHours < 0 {
			return cfg, fmt.Errorf("parsing %s: roadmap.stop_nagging_hours must be >= 0", configPath)
		}
	}

	// Validate the size_mapping block when a tracker is configured.
	// Absent block is fine (most workspaces); when present, the shape
	// must be sane or we surface the error at load time so a typo
	// doesn't silently disable size sync.
	if cfg.Tracker != nil && cfg.Tracker.Type != "" && cfg.Tracker.Type != "none" && cfg.Tracker.SizeMapping != nil {
		if err := cfg.Tracker.SizeMapping.Validate(); err != nil {
			return cfg, fmt.Errorf("parsing %s: %w", configPath, err)
		}
	}

	// Apply local config overrides
	local, err := LoadLocal(projectRoot, cfg.Folder)
	if err == nil {
		cfg = MergeLocal(cfg, local)
	}

	// Apply credentials store (lowest priority — local config overrides it above)
	if creds, cerr := LoadCredentials(); cerr == nil {
		var aerr error
		cfg, aerr = ApplyCredentialsStrict(cfg, creds)
		if aerr != nil {
			return cfg, aerr
		}
	}

	return cfg, nil
}

// Save writes hero.json to the hero folder inside the given project root.
func (c Config) Save(projectRoot string) error {
	committed := c.forCommittedSave()
	if err := ValidateCommittedIntegrations(committed.Integrations, filepath.Join(projectRoot, c.Folder, ConfigFileName)); err != nil {
		return err
	}
	heroDir := filepath.Join(projectRoot, c.Folder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		return err
	}

	configPath := filepath.Join(heroDir, ConfigFileName)

	data, err := json.MarshalIndent(committed, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return os.WriteFile(configPath, data, 0o644)
}

// forCommittedSave creates a secret-free copy for the legacy generic Save
// surface. Load returns a runtime-effective Config for compatibility, so its
// Tracker/Confluence and canonical auth may contain local/global credentials.
// Those values are runtime-only and must never cross the committed boundary.
func (c Config) forCommittedSave() Config {
	out := c
	if c.Integrations != nil {
		out.Tracker = nil
		out.Confluence = nil
	} else if c.Tracker != nil {
		x := *c.Tracker
		x.Token = ""
		out.Tracker = &x
	}
	if c.Integrations == nil && c.Confluence != nil {
		x := *c.Confluence
		x.Token = ""
		out.Confluence = &x
	}
	if c.Integrations != nil {
		x := &IntegrationsConfig{Default: c.Integrations.Default, Roles: map[string]string{}, Connections: map[string]IntegrationConfig{}}
		for k, v := range c.Integrations.Roles {
			x.Roles[k] = v
		}
		for id, v := range c.Integrations.Connections {
			n := IntegrationConfig{Provider: v.Provider, Settings: map[string]json.RawMessage{}}
			for k, raw := range v.Settings {
				n.Settings[k] = append(json.RawMessage(nil), raw...)
			}
			if v.Auth != nil {
				n.Auth = &IntegrationAuth{TokenEnv: v.Auth.TokenEnv}
			}
			x.Connections[id] = n
		}
		out.Integrations = x
	}
	return out
}

// LoadLocal reads hero.local.json from the hero folder inside the given project root.
// Returns a zero-value Config (no error) if the file doesn't exist.
// folder should be the hero folder name (e.g. ".hero").
func LoadLocal(projectRoot, folder string) (Config, error) {
	if folder == "" {
		folder = DefaultFolder
	}
	localPath := filepath.Join(projectRoot, folder, LocalConfigFileName)
	data, err := os.ReadFile(localPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var local Config
	if err := json.Unmarshal(data, &local); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", localPath, err)
	}
	return local, nil
}

// SaveLocal writes a partial Config as hero.local.json into the hero folder.
// Only fields that should be stored locally (tokens, personal preferences) should
// be set in the passed Config; zero/nil values are written as-is.
func SaveLocal(projectRoot, folder string, c Config) error {
	if folder == "" {
		folder = DefaultFolder
	}
	heroDir := filepath.Join(projectRoot, folder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		return err
	}
	localPath := filepath.Join(heroDir, LocalConfigFileName)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(localPath, data, 0o600)
}

// MergeLocal deep-merges local config fields on top of base.
// Non-zero/non-nil fields from local overwrite the corresponding base fields.
func MergeLocal(base, local Config) Config {
	if local.Folder != "" {
		base.Folder = local.Folder
	}

	if local.Tracker != nil {
		if base.Tracker == nil {
			base.Tracker = &TrackerConfig{}
		}
		if local.Tracker.Type != "" {
			base.Tracker.Type = local.Tracker.Type
		}
		if local.Tracker.Project != "" {
			base.Tracker.Project = local.Tracker.Project
		}
		if local.Tracker.Token != "" {
			base.Tracker.Token = local.Tracker.Token
		}
		if local.Tracker.TokenEnv != "" {
			base.Tracker.TokenEnv = local.Tracker.TokenEnv
		}
		if local.Tracker.BaseURL != "" {
			base.Tracker.BaseURL = local.Tracker.BaseURL
		}
		if local.Tracker.UserEmail != "" {
			base.Tracker.UserEmail = local.Tracker.UserEmail
		}
	}

	if local.Confluence != nil {
		if base.Confluence == nil {
			base.Confluence = &ConfluenceConfig{}
		}
		if local.Confluence.BaseURL != "" {
			base.Confluence.BaseURL = local.Confluence.BaseURL
		}
		if local.Confluence.SpaceKey != "" {
			base.Confluence.SpaceKey = local.Confluence.SpaceKey
		}
		if local.Confluence.Token != "" {
			base.Confluence.Token = local.Confluence.Token
		}
		if local.Confluence.TokenEnv != "" {
			base.Confluence.TokenEnv = local.Confluence.TokenEnv
		}
		if local.Confluence.UserEmail != "" {
			base.Confluence.UserEmail = local.Confluence.UserEmail
		}
	}

	if local.Team != nil {
		if base.Team == nil {
			base.Team = &TeamConfig{}
		}
		if local.Team.NudgeLevel != "" {
			base.Team.NudgeLevel = local.Team.NudgeLevel
		}
		if local.Team.StaleDays != 0 {
			base.Team.StaleDays = local.Team.StaleDays
		}
		// bool fields: only override if team block is explicitly provided
		if local.Team.RequireReview {
			base.Team.RequireReview = true
		}
		if local.Team.AutoContext {
			base.Team.AutoContext = true
		}
	}

	if local.Serve != nil {
		if base.Serve == nil {
			base.Serve = &ServeConfig{}
		}
		if local.Serve.Port != 0 {
			base.Serve.Port = local.Serve.Port
		}
		if local.Serve.HealthTTL != "" {
			base.Serve.HealthTTL = local.Serve.HealthTTL
		}
	}

	if local.Models != nil {
		if base.Models == nil {
			base.Models = &ModelConfig{}
		}
		if local.Models.DefaultModel != "" {
			base.Models.DefaultModel = local.Models.DefaultModel
		}
		if len(local.Models.Roles) > 0 {
			if base.Models.Roles == nil {
				base.Models.Roles = make(map[string]string)
			}
			for k, v := range local.Models.Roles {
				base.Models.Roles[k] = v
			}
		}
	}

	// Dialect fields (vocabulary + methodology). Scalars: local non-empty
	// wins. Override maps: entry-by-entry merge, local entries replace
	// base entries on key collision; non-colliding base keys preserved.
	if local.Vocabulary != "" {
		base.Vocabulary = local.Vocabulary
	}
	if len(local.VocabularyOverrides) > 0 {
		if base.VocabularyOverrides == nil {
			base.VocabularyOverrides = make(map[string]string)
		}
		for k, v := range local.VocabularyOverrides {
			base.VocabularyOverrides[k] = v
		}
	}
	if local.Methodology != "" {
		base.Methodology = local.Methodology
	}
	if len(local.MethodologyOverrides) > 0 {
		if base.MethodologyOverrides == nil {
			base.MethodologyOverrides = make(map[string]string)
		}
		for k, v := range local.MethodologyOverrides {
			base.MethodologyOverrides[k] = v
		}
	}

	return base
}

// HeroDir returns the absolute path to the hero folder for a given project root.
func (c Config) HeroDir(projectRoot string) string {
	return filepath.Join(projectRoot, c.Folder)
}

// PlanningDir returns the path to the planning directory.
func (c Config) PlanningDir(projectRoot string) string {
	return filepath.Join(c.HeroDir(projectRoot), "planning")
}

// SpecsDir returns the path to the specs directory.
func (c Config) SpecsDir(projectRoot string) string {
	return filepath.Join(c.HeroDir(projectRoot), "specs")
}

// KnowledgeDir returns the path to the knowledge directory.
func (c Config) KnowledgeDir(projectRoot string) string {
	return filepath.Join(c.HeroDir(projectRoot), "knowledge")
}

// ConventionsDir returns the path to the conventions subdirectory under knowledge.
func (c Config) ConventionsDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "conventions")
}

// DecisionsDir returns the path to the decisions subdirectory under knowledge.
func (c Config) DecisionsDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "decisions")
}

// RulesDir returns the path to the rules subdirectory under knowledge.
func (c Config) RulesDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "rules")
}

// ExternalDir returns the path to the external knowledge subdirectory.
func (c Config) ExternalDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "external")
}

// ContextDir returns the path to the project context subdirectory under knowledge.
func (c Config) ContextDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "context")
}

// TemplatesDir returns the path to the templates subdirectory under knowledge.
func (c Config) TemplatesDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "templates")
}

// NotesDir returns the path to the notes subdirectory under knowledge.
func (c Config) NotesDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "notes")
}

// ExplainersDir returns the path to the explainers subdirectory under
// knowledge — synthesized "how a feature works" entries.
func (c Config) ExplainersDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "explainers")
}

// TrackerKnowledgeDir returns the path to the tracker knowledge subdirectory.
// Used for caching discovered field mappings and other tracker integration state.
func (c Config) TrackerKnowledgeDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "tracker")
}

// CodeDir returns the path to the code intelligence subdirectory under knowledge.
func (c Config) CodeDir(projectRoot string) string {
	return filepath.Join(c.KnowledgeDir(projectRoot), "code")
}

// ResolveRepoPath resolves a repo alias to an absolute path.
// Paths can be relative (resolved against projectRoot) or absolute.
func (c Config) ResolveRepoPath(projectRoot, alias string) (string, error) {
	path, ok := c.Repos[alias]
	if !ok {
		return "", fmt.Errorf("repo alias %q not configured — add it to hero.json repos or run 'hero repos add %s <path>'", alias, alias)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving path for repo %q: %w", alias, err)
	}
	return abs, nil
}

// ResolveAllRepos returns resolved paths for all configured repos,
// with a status indicator for each (accessible or not).
func (c Config) ResolveAllRepos(projectRoot string) map[string]RepoStatus {
	result := make(map[string]RepoStatus, len(c.Repos))
	for alias := range c.Repos {
		rs := RepoStatus{Alias: alias}
		abs, err := c.ResolveRepoPath(projectRoot, alias)
		if err != nil {
			rs.Error = err.Error()
			result[alias] = rs
			continue
		}
		rs.Path = abs
		heroDir := filepath.Join(abs, c.Folder)
		if info, err := os.Stat(heroDir); err == nil && info.IsDir() {
			rs.Accessible = true
		} else {
			rs.Error = "no .hero/ directory found"
		}
		result[alias] = rs
	}
	return result
}

// RepoStatus describes the resolved state of a configured repo alias.
type RepoStatus struct {
	Alias      string `json:"alias"`
	Path       string `json:"path"`
	Accessible bool   `json:"accessible"`
	Error      string `json:"error,omitempty"`
}

// NextDir returns the path to the per-user next directory.
func (c Config) NextDir(projectRoot string) string {
	return filepath.Join(c.HeroDir(projectRoot), "next")
}

// NextMode returns "team" or "solo" based on configuration.
func (c Config) NextMode() string {
	if c.Next != nil && c.Next.Mode == "team" {
		return "team"
	}
	return "solo"
}

// NextProjected reports whether NEXT.md should be regenerated from
// the graph each Stop hook (true) or left as agent-authored content
// (false, default). Flipped by `hero next migrate-to-projection`.
func (c Config) NextProjected() bool {
	return c.Next != nil && c.Next.Projected
}

// NextGoalCapture returns the configured SessionGoal capture mode —
// "floor" (default: window + marker + manual) or "embed" (also runs the
// confidence-gated embeddings selector). Any unrecognized value falls
// back to "floor" so a typo can never silently disable goal capture.
func (c Config) NextGoalCapture() string {
	if c.Next != nil && c.Next.GoalCapture == "embed" {
		return "embed"
	}
	return "floor"
}

// MocksDir returns the path to the mocks directory.
func (c Config) MocksDir(projectRoot string) string {
	return filepath.Join(c.HeroDir(projectRoot), "mocks")
}

// SnapshotArchive returns the resolved archive config with documented
// defaults filled in. Always returns a non-nil pointer so callers can
// read fields without nil-checking.
func (c Config) SnapshotArchive() SnapshotArchiveConfig {
	out := SnapshotArchiveConfig{}
	if c.Snapshot != nil && c.Snapshot.Archive != nil {
		out = *c.Snapshot.Archive
	}
	if out.StalenessCutoff == "" {
		out.StalenessCutoff = "monthly"
	}
	if out.ReleaseTagPattern == "" {
		out.ReleaseTagPattern = "v[0-9].*"
	}
	if out.Retention == "" {
		out.Retention = "all"
	}
	return out
}

// SnapshotMilestonesEnabled reports whether milestone-triggered
// archives should fire. Defaults to true when unset.
func (c Config) SnapshotMilestonesEnabled() bool {
	if c.Snapshot == nil || c.Snapshot.Archive == nil || c.Snapshot.Archive.Milestones == nil {
		return true
	}
	return *c.Snapshot.Archive.Milestones
}
