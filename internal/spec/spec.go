package spec

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Type represents the kind of spec.
type Type string

const (
	TypeFeature    Type = "feature"
	TypeBug        Type = "bug"
	TypeConvention Type = "convention"
	TypeDecision   Type = "decision"
	TypeInitiative Type = "initiative"
	TypeRule       Type = "rule"
	TypeExternal   Type = "external"
	TypeContext    Type = "context"
	TypeNote       Type = "note"
	TypeTripwire   Type = "tripwire"
)

// Status represents the lifecycle state of a spec.
type Status string

const (
	// Work spec states (feature, bug)
	StatusPlanning   Status = "planning"
	StatusInReview   Status = "in-review"
	StatusDelivering Status = "delivering"
	StatusCompleted  Status = "completed"
	// StatusRegressed is set automatically by hero ac record when an
	// AC regresses on a spec that was previously claiming completed.
	// Reset to completed once the regressed AC is passing again.
	StatusRegressed Status = "regressed"

	// Cross-repo handoff states (from the originator's view; the
	// receiver's spec uses the standard lifecycle). See
	// cross-repo-peering spec for the full state machine.
	//
	// StatusHandedOff — the originator just gave the spec to a peer
	// (initial transition; appears briefly until the peer-side spec
	// is observed in `delivering` and the status flips to
	// StatusAwaitingPeer).
	StatusHandedOff Status = "handed_off"
	// StatusAwaitingPeer — steady state while the peer works.
	StatusAwaitingPeer Status = "awaiting_peer"
	// StatusHandedBack — the peer finished (or explicitly bounced),
	// the ball is back on the originator's side for verification or
	// follow-up.
	StatusHandedBack Status = "handed_back"

	// Convention states
	StatusDraft  Status = "draft"
	StatusActive Status = "active"

	// Decision states
	StatusProposed Status = "proposed"
	StatusAccepted Status = "accepted"

	// Shared terminal state
	StatusSuperseded Status = "superseded"
)

// Horizon is the temporal segmentation a spec carries to distinguish
// actionable-now work from captured-for-later thinking. Per
// spec-prioritization. Different from status (lifecycle position) and
// priority (urgency within the now).
type Horizon string

const (
	// HorizonNow — actively being worked or imminent. The standup set.
	HorizonNow Horizon = "now"
	// HorizonNext — queued for the next phase/sprint. Concrete enough
	// to commit to soon.
	HorizonNext Horizon = "next"
	// HorizonSomeday — captured because the thinking is worth keeping.
	// Not committing to time. Hidden from default views.
	HorizonSomeday Horizon = "someday"
	// HorizonParking — explicitly deferred (e.g., dependent on a
	// future capability or business condition). Reasoning preserved.
	HorizonParking Horizon = "parking"
)

// IsValidHorizon reports whether h is one of the four canonical
// values. Empty string is treated as "unset" — callers default to
// HorizonNow when convenient.
func IsValidHorizon(h Horizon) bool {
	switch h {
	case HorizonNow, HorizonNext, HorizonSomeday, HorizonParking:
		return true
	}
	return false
}

// Spec represents a parsed spec document with extracted metadata.
type Spec struct {
	Slug         string
	Title        string
	Type         Type
	Status       Status
	Path         string    // absolute path to spec.md
	CreatedAt    time.Time // from frontmatter or file mtime
	ModifiedAt   time.Time // file modification time
	Tags         []string  // from frontmatter
	Scope        []string  // glob patterns (conventions/rules/tripwires only)
	Subproject   string    // monorepo subproject scope identifier (forward-slash path relative to root, e.g. "engines/mlx"); empty = workspace root
	Triggers     []string  // keywords that activate this tripwire in retrieval
	Priority     string    // hero-level priority (e.g. "critical", "high", "medium", "low")
	Severity     string    // hero-level severity (e.g. "critical", "high", "medium", "low")
	Horizon      Horizon   // when: now / next / someday / parking. Empty = unset (treated as now in default views).
	Pinned       bool      // floats this spec to the top of `hero queue` regardless of other ranking signals
	ClaimedBy      string    // who is working on this
	DeliveryMethod string    // "agent", "manual", or "" (unset)
	TrackerID    string    // external tracker issue ID (e.g. "#42", "PROJ-123", "LIN-abc")
	URL          string    // external knowledge URL
	LocalPath    string    // local path to external docs
	Relations    []Relation
	Smoke        *SmokeConfig      // nil if no smoke: frontmatter field
	Sections     map[string]string // section name (lowercase) -> content
	FilesTouched []string          // extracted from Changes section

	// ReceivedFrom is populated when this spec was scaffolded by a
	// cross-repo handoff or spec-out peer call. peer_id is the
	// canonical join key back to the originating workspace.
	ReceivedFrom *ReceivedFromBlock

	// Tracker metadata — populated from tracker-prefixed frontmatter fields
	// (e.g. jira_status, github_assignee, linear_priority).
	TrackerName     string // which tracker: "jira", "github", "linear"
	TrackerStatus   string // tracker-native status (e.g. "In Progress", "Open")
	TrackerPriority string // tracker-native priority (e.g. "High", "Critical")
	TrackerSeverity string // tracker-native severity
	TrackerAssignee string // tracker-native assignee display name or email
	TrackerType     string // tracker issue type (e.g. "Story", "Bug", "Task")
	Description     string // short description (2-3 sentences) from tracker
	RawContent      string // full file content including frontmatter
	ThreeFile       bool   // true if loaded from three-file layout
}

// ReceivedFromBlock mirrors contracts/peering.ReceivedFrom for use in
// the parsed spec model. Kept in the spec package (rather than
// importing peering directly) so the spec package stays a leaf with
// no contracts dependency, and so a reader can inspect the block
// without dragging in the wire-shape package.
type ReceivedFromBlock struct {
	PeerID           string
	PeerAliasDisplay string
	OriginatorSlug   string
	HandedOffAt      time.Time
	AtCommit         string
	Reason           string
}

// SmokeConfig holds the smoke: frontmatter block for a spec.
// Set Deferred: true when a spec uses the "smoke: deferred" escape hatch.
type SmokeConfig struct {
	Deferred bool     // true when smoke: deferred (escape hatch, no script yet)
	Script   string   // e.g. scripts/smoke/feature-slug.sh
	Expects  []string // AC IDs the smoke exercises, e.g. ["feature-slug:AC-1"]
	RunsOn   []string // trigger conditions, e.g. ["commit-touches:internal/cli/*.go", "nightly"]
}

// Relation represents a link between two specs.
type Relation struct {
	Target string // slug of the related spec
	Kind   string // "parent", "child", "depends-on", "supersedes", "related"
}

// CriterionKind classifies an acceptance criterion by EARS pattern.
type CriterionKind int

const (
	// CriterionFreeform is a bullet that does not match any EARS pattern.
	CriterionFreeform CriterionKind = iota
	// CriterionUbiquitous — "THE SYSTEM SHALL <behavior>"
	CriterionUbiquitous
	// CriterionEvent — "WHEN <trigger> THE SYSTEM SHALL <behavior>"
	CriterionEvent
	// CriterionState — "WHILE <state> THE SYSTEM SHALL <behavior>"
	CriterionState
	// CriterionOptional — "WHERE <feature> IS ENABLED THE SYSTEM SHALL <behavior>"
	CriterionOptional
	// CriterionUnwanted — "IF <trigger> THEN THE SYSTEM SHALL <behavior>"
	CriterionUnwanted
)

// String returns the lowercase name of the criterion kind.
func (k CriterionKind) String() string {
	switch k {
	case CriterionUbiquitous:
		return "ubiquitous"
	case CriterionEvent:
		return "event"
	case CriterionState:
		return "state"
	case CriterionOptional:
		return "optional"
	case CriterionUnwanted:
		return "unwanted"
	default:
		return "freeform"
	}
}

// IsEARS reports whether this criterion was classified as any EARS pattern.
func (k CriterionKind) IsEARS() bool {
	return k != CriterionFreeform
}

// TestLink is a reference from a criterion to a test that verifies it.
type TestLink struct {
	File string // e.g. "e2e/csv-export.spec.ts"
	Name string // e.g. "streams large exports"
}

// Criterion is a single parsed acceptance-criteria bullet.
type Criterion struct {
	Raw        string        // original bullet text (without the "- " prefix)
	Kind       CriterionKind // classification
	Trigger    string        // WHEN/WHILE/IF/WHERE clause, empty for ubiquitous/freeform
	Behavior   string        // SHALL clause, empty for freeform
	VerifiedBy []TestLink    // verified_by: annotations linking to tests
}

// ParseFile reads and parses a spec.md file.
func ParseFile(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	return Parse(string(data), path, info.ModTime())
}

// ParseThreeFile loads a spec from a three-file layout directory
// (requirements.md + design.md + tasks.md). Frontmatter comes from
// requirements.md; the three files are concatenated in order.
func ParseThreeFile(dir string) (*Spec, error) {
	reqPath := filepath.Join(dir, "requirements.md")
	reqData, err := os.ReadFile(reqPath)
	if err != nil {
		return nil, fmt.Errorf("reading requirements.md: %w", err)
	}

	designPath := filepath.Join(dir, "design.md")
	designData, _ := os.ReadFile(designPath) // optional

	tasksPath := filepath.Join(dir, "tasks.md")
	tasksData, _ := os.ReadFile(tasksPath) // optional

	// Concatenate: requirements (with frontmatter) + design + tasks
	var combined strings.Builder
	combined.Write(reqData)
	if len(designData) > 0 {
		combined.WriteString("\n\n")
		combined.Write(designData)
	}
	if len(tasksData) > 0 {
		combined.WriteString("\n\n")
		combined.Write(tasksData)
	}

	info, err := os.Stat(reqPath)
	if err != nil {
		return nil, err
	}

	s, err := Parse(combined.String(), reqPath, info.ModTime())
	if err != nil {
		return nil, err
	}
	// Override path to point to the directory, not requirements.md
	s.Path = filepath.Join(dir, "spec.md") // virtual path for compatibility
	s.ThreeFile = true
	return s, nil
}

// Parse parses spec content and extracts metadata.
func Parse(content, path string, modTime time.Time) (*Spec, error) {
	s := &Spec{
		Path:       path,
		ModifiedAt: modTime,
		CreatedAt:  modTime,
		Sections:   make(map[string]string),
		RawContent: content,
	}

	// Parse frontmatter first (overrides path-based defaults)
	body := s.parseFrontmatter(content)

	// Fall back to path-based type and status if not set by frontmatter
	if s.Type == "" {
		s.Type = typeFromPath(path)
	}
	if s.Status == "" {
		s.Status = statusFromPath(path)
	}
	if s.Slug == "" {
		s.Slug = slugFromPath(path)
	}

	// Parse sections from body (content after frontmatter)
	s.parseSections(body)

	// Extract title from first H1 in body
	if s.Title == "" {
		s.Title = extractTitle(body)
	}

	// Extract files touched from Changes section
	if changes, ok := s.Sections["changes"]; ok {
		s.FilesTouched = extractFilePaths(changes)
	}

	return s, nil
}

// parseFrontmatter extracts YAML-like frontmatter delimited by ---.
// Returns the body content after the frontmatter.
// Supports: title, type, status, tags, scope, claimed_by, created, relations.
func (s *Spec) parseFrontmatter(content string) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Check for opening ---
	first := strings.TrimSpace(lines[0])
	if first != "---" {
		return content
	}

	// Find closing ---
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 {
		return content
	}

	// Parse frontmatter lines (simple key: value, no nested YAML)
	for i := 1; i < closeIdx; i++ {
		raw := lines[i]
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Top-level keys live flush-left; indented lines are part of
		// a multi-line value. Skip them — the structured value-handlers
		// for keys that need them (relations, source) consume their
		// own continuation lines.
		if leadingSpaceCount(raw) > 0 {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])

		switch key {
		case "title":
			s.Title = val
		case "type":
			s.Type = Type(val)
		case "status":
			s.Status = Status(val)
		case "slug":
			s.Slug = val
		case "claimed_by":
			s.ClaimedBy = val
		case "delivery_method":
			s.DeliveryMethod = val
		case "tracker_id":
			s.TrackerID = val
		case "priority":
			s.Priority = val
		case "severity":
			s.Severity = val
		case "horizon":
			s.Horizon = Horizon(val)
		case "pinned":
			s.Pinned = parseBool(val)
		case "url":
			s.URL = val
		case "local_path":
			s.LocalPath = val
		case "description":
			s.Description = val
		case "created":
			if t, err := time.Parse("2006-01-02", val); err == nil {
				s.CreatedAt = t
			} else if t, err := time.Parse(time.RFC3339, val); err == nil {
				s.CreatedAt = t
			}
		case "tags":
			s.Tags = parseList(val)
		case "scope":
			s.Scope = parseList(val)
		case "subproject":
			s.Subproject = val
		case "triggers":
			s.Triggers = parseList(val)
		case "relates-to", "depends-on", "supersedes", "parent", "child":
			for _, target := range parseList(val) {
				s.Relations = append(s.Relations, Relation{Target: target, Kind: key})
			}
		case "relations":
			// YAML-list-of-objects format:
			//   relations:
			//     - target: foo
			//       kind: parent
			//     - target: bar
			//       kind: depends-on
			rels, consumed := parseRelationsBlock(lines, i+1, closeIdx)
			s.Relations = append(s.Relations, rels...)
			i = consumed - 1 // outer loop will i++ next iteration
		case "smoke":
			// Inline scalar: smoke: deferred
			// Block form:
			//   smoke:
			//     script: scripts/smoke/feature-slug.sh
			//     expects: [feature-slug:AC-1]
			//     runs_on: [commit-touches:..., nightly]
			if val == "deferred" || val == "none" {
				s.Smoke = &SmokeConfig{Deferred: true}
			} else {
				smokeCfg, consumed := parseSmokeBlock(lines, i+1, closeIdx)
				s.Smoke = smokeCfg
				i = consumed - 1
			}
		case "received_from":
			// Block form recording cross-repo provenance:
			//   received_from:
			//     peer_id: 9c1c2f3e-...
			//     peer_alias_display: client
			//     originator_slug: order-failure-error-display
			//     handed_off_at: 2026-05-15T14:00:00Z
			//     at_commit: 3176736
			//     reason: "Symptom is in the client, root cause is the API response shape."
			rf, consumed := parseReceivedFromBlock(lines, i+1, closeIdx)
			s.ReceivedFrom = rf
			i = consumed - 1
		default:
			// Parse tracker-prefixed fields: jira_*, github_*, linear_*
			for _, prefix := range []string{"jira_", "github_", "linear_"} {
				if strings.HasPrefix(key, prefix) {
					trackerName := strings.TrimSuffix(prefix, "_")
					field := strings.TrimPrefix(key, prefix)
					if s.TrackerName == "" {
						s.TrackerName = trackerName
					}
					switch field {
					case "id":
						if s.TrackerID == "" {
							s.TrackerID = val
						}
					case "status":
						s.TrackerStatus = val
					case "priority":
						s.TrackerPriority = val
					case "severity":
						s.TrackerSeverity = val
					case "assignee":
						s.TrackerAssignee = val
					case "type":
						s.TrackerType = val
					case "url":
						if s.URL == "" {
							s.URL = val
						}
					}
					break
				}
			}
		}
	}

	// Return body after frontmatter
	body := strings.Join(lines[closeIdx+1:], "")
	return body
}

// leadingSpaceCount counts leading spaces in a frontmatter line. Used
// to distinguish top-level YAML keys (no indent) from continuation
// lines (any indent) that belong to a structured value above them.
func leadingSpaceCount(line string) int {
	n := 0
	for _, c := range line {
		if c == ' ' {
			n++
			continue
		}
		break
	}
	return n
}

// parseRelationsBlock walks the indented YAML-list-of-objects under
// `relations:` and returns the parsed slice plus the index of the
// next line the outer parser should resume from.
//
// Recognised shape (any indent that's deeper than the parent key):
//
//	relations:
//	  - target: <slug>
//	    kind: parent | depends-on | supersedes | …
//	  - target: <slug>
//	    kind: …
//
// Tolerant: missing `kind` defaults to "related"; unknown kinds are
// kept as-is so consumers can decide what to do.
func parseRelationsBlock(lines []string, start, end int) ([]Relation, int) {
	var out []Relation
	var current Relation
	idx := start
	for ; idx < end; idx++ {
		raw := lines[idx]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		// A top-level key (no indent) ends the relations block.
		if leadingSpaceCount(raw) == 0 {
			break
		}
		switch {
		case strings.HasPrefix(trimmed, "- "):
			// Flush prior entry, start a new one.
			if current.Target != "" {
				out = append(out, normalizeRelation(current))
			}
			current = Relation{}
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			applyRelField(&current, rest)
		default:
			applyRelField(&current, trimmed)
		}
	}
	if current.Target != "" {
		out = append(out, normalizeRelation(current))
	}
	return out, idx
}

// parseSmokeBlock parses the indented YAML under a `smoke:` key.
// Returns a populated SmokeConfig and the index of the next line the
// outer parser should resume from.
//
// Recognised shape:
//
//	smoke:
//	  script: scripts/smoke/feature-slug.sh
//	  expects: [feature-slug:AC-1, feature-slug:AC-2]
//	  runs_on: [commit-touches:internal/cli/*.go, nightly]
func parseSmokeBlock(lines []string, start, end int) (*SmokeConfig, int) {
	cfg := &SmokeConfig{}
	idx := start
	for ; idx < end; idx++ {
		raw := lines[idx]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		// Any un-indented line ends the block.
		if leadingSpaceCount(raw) == 0 {
			break
		}
		k, v, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "script":
			cfg.Script = v
		case "expects":
			cfg.Expects = parseList(v)
		case "runs_on":
			cfg.RunsOn = parseList(v)
		}
	}
	return cfg, idx
}

// parseReceivedFromBlock parses the indented YAML block under a
// `received_from:` key on a receiver-side spec. Returns a populated
// block and the index of the next line the outer parser should
// resume from. Unknown fields are tolerated.
func parseReceivedFromBlock(lines []string, start, end int) (*ReceivedFromBlock, int) {
	out := &ReceivedFromBlock{}
	idx := start
	for ; idx < end; idx++ {
		raw := lines[idx]
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if leadingSpaceCount(raw) == 0 {
			break
		}
		k, v, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(strings.Trim(v, "\""))
		switch k {
		case "peer_id":
			out.PeerID = v
		case "peer_alias_display":
			out.PeerAliasDisplay = v
		case "originator_slug":
			out.OriginatorSlug = v
		case "handed_off_at":
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				out.HandedOffAt = t
			}
		case "at_commit":
			out.AtCommit = v
		case "reason":
			out.Reason = v
		}
	}
	return out, idx
}

// applyRelField parses "key: value" inside a YAML object and assigns
// to the open Relation. Quietly ignores unrecognised fields so future
// metadata (notes, tags) doesn't break parsing.
func applyRelField(r *Relation, line string) {
	k, v, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	k = strings.TrimSpace(k)
	v = strings.TrimSpace(v)
	switch k {
	case "target":
		r.Target = v
	case "kind", "type":
		r.Kind = v
	}
}

// normalizeRelation defaults missing kind to "related" so downstream
// edge mapping has something to work with.
func normalizeRelation(r Relation) Relation {
	if r.Kind == "" {
		r.Kind = "related"
	}
	return r
}

// parseBool parses a YAML-style scalar boolean. Accepts true/false,
// yes/no, on/off, 1/0 (case-insensitive). Anything else returns false.
func parseBool(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "true", "yes", "on", "1":
		return true
	}
	return false
}

// parseList parses a comma-separated or bracket-enclosed list.
// Supports: "a, b, c" or "[a, b, c]" or "a" (single value).
func parseList(val string) []string {
	val = strings.TrimSpace(val)
	val = strings.Trim(val, "[]")
	if val == "" {
		return nil
	}

	parts := strings.Split(val, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func typeFromPath(path string) Type {
	if strings.Contains(path, "/bugs/") {
		return TypeBug
	}
	if strings.Contains(path, "/conventions/") {
		return TypeConvention
	}
	if strings.Contains(path, "/decisions/") {
		return TypeDecision
	}
	if strings.Contains(path, "/initiatives/") {
		return TypeInitiative
	}
	if strings.Contains(path, "/rules/") {
		return TypeRule
	}
	if strings.Contains(path, "/external/") {
		return TypeExternal
	}
	if strings.Contains(path, "/context/") {
		return TypeContext
	}
	if strings.Contains(path, "/notes/") {
		return TypeNote
	}
	if strings.Contains(path, "/tripwires/") {
		return TypeTripwire
	}
	return TypeFeature
}

func statusFromPath(path string) Status {
	if strings.Contains(path, "/planning/") {
		return StatusPlanning
	}
	if strings.Contains(path, "/conventions/") {
		return StatusActive
	}
	if strings.Contains(path, "/decisions/") {
		return StatusAccepted
	}
	if strings.Contains(path, "/rules/") {
		return StatusActive
	}
	if strings.Contains(path, "/external/") {
		return StatusActive
	}
	if strings.Contains(path, "/context/") {
		return StatusActive
	}
	if strings.Contains(path, "/notes/") {
		return StatusActive
	}
	if strings.Contains(path, "/tripwires/") {
		return StatusActive
	}
	return StatusCompleted
}

func slugFromPath(path string) string {
	// Path is like .hero/planning/features/my-feature/spec.md
	// or .hero/specs/my-feature/spec.md
	// or .hero/conventions/error-handling/spec.md
	// We want the directory name containing spec.md
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}

func extractTitle(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func (s *Spec) parseSections(content string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentSection string
	var currentContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			// Save previous section
			if currentSection != "" {
				s.Sections[currentSection] = strings.TrimSpace(currentContent.String())
			}
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			currentContent.Reset()
		} else if currentSection != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	// Save last section
	if currentSection != "" {
		s.Sections[currentSection] = strings.TrimSpace(currentContent.String())
	}
}

// extractFilePaths pulls file-path-like strings from the Changes section.
func extractFilePaths(changesContent string) []string {
	var paths []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(changesContent))
	for scanner.Scan() {
		line := scanner.Text()
		for _, word := range strings.Fields(line) {
			word = strings.Trim(word, "`\"'(),;:")
			if looksLikeFilePath(word) && !seen[word] {
				paths = append(paths, word)
				seen[word] = true
			}
		}
	}

	return paths
}

func looksLikeFilePath(s string) bool {
	if !strings.Contains(s, "/") {
		return false
	}
	ext := filepath.Ext(s)
	if ext == "" {
		return false
	}
	validExts := map[string]bool{
		".go": true, ".java": true, ".groovy": true, ".gradle": true,
		".rs": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".mjs": true, ".cjs": true, ".mts": true,
		".py": true, ".rb": true, ".sql": true, ".md": true,
		".json": true, ".yaml": true, ".yml": true, ".toml": true,
		".xml": true, ".html": true, ".css": true, ".scss": true,
		".sh": true, ".bash": true,
		".kt": true, ".kts": true, ".swift": true, ".c": true, ".h": true,
		".cpp": true, ".hpp": true, ".cs": true, ".php": true,
	}
	return validExts[ext]
}

// Discover finds all spec.md files under the hero directory,
// including specs/, planning/, conventions/, and decisions/.
func Discover(heroDir string) ([]*Spec, error) {
	var specs []*Spec

	// Track directories we've already loaded as three-file to avoid
	// loading the same spec directory twice.
	loadedDirs := make(map[string]bool)

	err := filepath.Walk(heroDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := info.Name()
			if base == ".git" {
				return filepath.SkipDir
			}
			// Check for three-file layout: requirements.md in the directory
			reqPath := filepath.Join(path, "requirements.md")
			if _, reqErr := os.Stat(reqPath); reqErr == nil {
				// Only load if there's no spec.md (three-file takes lower priority)
				specPath := filepath.Join(path, "spec.md")
				if _, specErr := os.Stat(specPath); os.IsNotExist(specErr) {
					spec, parseErr := ParseThreeFile(path)
					if parseErr == nil {
						specs = append(specs, spec)
						loadedDirs[path] = true
					}
				}
			}
			return nil
		}
		if info.Name() != "spec.md" {
			return nil
		}

		// Skip if we already loaded this directory as three-file
		dir := filepath.Dir(path)
		if loadedDirs[dir] {
			return nil
		}

		spec, parseErr := ParseFile(path)
		if parseErr != nil {
			return nil // skip unparseable specs
		}
		specs = append(specs, spec)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return specs, nil
}

// Kickoff returns the spec's `## Kickoff` section body — the paste-ready
// cold-start prompt authored by spec-writing skills (`/design`, `/deliver`)
// and surfaced by `hero queue`. Returns the empty string if the section
// is missing.
func (s *Spec) Kickoff() string {
	return s.Sections["kickoff"]
}

// AcceptanceCriteria parses the "acceptance criteria" section into a list of
// Criterion values, classifying each bullet against the EARS patterns that
// Hero recognizes. Bullets that do not match any pattern are returned as
// CriterionFreeform so existing specs continue to work unchanged.
func (s *Spec) AcceptanceCriteria() []Criterion {
	section, ok := s.Sections["acceptance criteria"]
	if !ok {
		return nil
	}

	var out []Criterion
	scanner := bufio.NewScanner(strings.NewReader(section))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for verified_by: annotation on the previous criterion
		if strings.HasPrefix(trimmed, "verified_by:") && len(out) > 0 {
			ref := strings.TrimSpace(strings.TrimPrefix(trimmed, "verified_by:"))
			if link, ok := parseTestLink(ref); ok {
				out[len(out)-1].VerifiedBy = append(out[len(out)-1].VerifiedBy, link)
			}
			continue
		}

		var raw string
		switch {
		case strings.HasPrefix(trimmed, "- "):
			raw = strings.TrimPrefix(trimmed, "- ")
		case strings.HasPrefix(trimmed, "* "):
			raw = strings.TrimPrefix(trimmed, "* ")
		default:
			continue
		}
		out = append(out, ClassifyCriterion(raw))
	}
	return out
}

// ClassifyCriterion parses a single bullet string and classifies it against
// the EARS patterns. The lookup is case-insensitive on the EARS keywords but
// preserves the original casing in Trigger/Behavior for downstream rendering.
func ClassifyCriterion(raw string) Criterion {
	text := strings.TrimSpace(raw)
	// Drop a single trailing period so Behavior values stay clean.
	trimmed := strings.TrimRight(text, ".")

	// Helpers for case-insensitive keyword lookup.
	upper := strings.ToUpper(trimmed)

	const shall = " THE SYSTEM SHALL "

	// Event: WHEN <trigger> THE SYSTEM SHALL <behavior>
	if strings.HasPrefix(upper, "WHEN ") {
		if idx := strings.Index(upper, shall); idx > 0 {
			trigger := strings.TrimSpace(trimmed[len("WHEN "):idx])
			behavior := strings.TrimSpace(trimmed[idx+len(shall):])
			return Criterion{Raw: raw, Kind: CriterionEvent, Trigger: trigger, Behavior: behavior}
		}
	}

	// State: WHILE <state> THE SYSTEM SHALL <behavior>
	if strings.HasPrefix(upper, "WHILE ") {
		if idx := strings.Index(upper, shall); idx > 0 {
			trigger := strings.TrimSpace(trimmed[len("WHILE "):idx])
			behavior := strings.TrimSpace(trimmed[idx+len(shall):])
			return Criterion{Raw: raw, Kind: CriterionState, Trigger: trigger, Behavior: behavior}
		}
	}

	// Optional: WHERE <feature> IS ENABLED THE SYSTEM SHALL <behavior>
	if strings.HasPrefix(upper, "WHERE ") {
		if idx := strings.Index(upper, shall); idx > 0 {
			trigger := strings.TrimSpace(trimmed[len("WHERE "):idx])
			behavior := strings.TrimSpace(trimmed[idx+len(shall):])
			return Criterion{Raw: raw, Kind: CriterionOptional, Trigger: trigger, Behavior: behavior}
		}
	}

	// Unwanted: IF <trigger> THEN THE SYSTEM SHALL <behavior>
	if strings.HasPrefix(upper, "IF ") {
		const thenShall = " THEN THE SYSTEM SHALL "
		if idx := strings.Index(upper, thenShall); idx > 0 {
			trigger := strings.TrimSpace(trimmed[len("IF "):idx])
			behavior := strings.TrimSpace(trimmed[idx+len(thenShall):])
			return Criterion{Raw: raw, Kind: CriterionUnwanted, Trigger: trigger, Behavior: behavior}
		}
	}

	// Ubiquitous: THE SYSTEM SHALL <behavior>
	if strings.HasPrefix(upper, "THE SYSTEM SHALL ") {
		behavior := strings.TrimSpace(trimmed[len("THE SYSTEM SHALL "):])
		return Criterion{Raw: raw, Kind: CriterionUbiquitous, Behavior: behavior}
	}

	return Criterion{Raw: raw, Kind: CriterionFreeform}
}

// parseTestLink parses a "file::testname" reference into a TestLink.
func parseTestLink(ref string) (TestLink, bool) {
	parts := strings.SplitN(ref, "::", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return TestLink{}, false
	}
	return TestLink{File: strings.TrimSpace(parts[0]), Name: strings.TrimSpace(parts[1])}, true
}

// IsWorkSpec returns true if the spec type is a feature or bug (has a delivery lifecycle).
func (s *Spec) IsWorkSpec() bool {
	return s.Type == TypeFeature || s.Type == TypeBug
}

// IsKnowledge returns true if the spec type is a knowledge base entry
// (convention, decision, rule, external, or context).
func (s *Spec) IsKnowledge() bool {
	return s.Type == TypeConvention || s.Type == TypeDecision ||
		s.Type == TypeRule || s.Type == TypeExternal || s.Type == TypeContext ||
		s.Type == TypeNote || s.Type == TypeTripwire
}

// IsInFlight returns true if the spec is currently being worked on,
// either locally (planning/in-review/delivering) or by a peer
// (handed_off/awaiting_peer). handed_back also counts: the ball is
// back on this side awaiting verification.
func (s *Spec) IsInFlight() bool {
	switch s.Status {
	case StatusPlanning, StatusInReview, StatusDelivering,
		StatusHandedOff, StatusAwaitingPeer, StatusHandedBack:
		return true
	}
	return false
}

// IsHandoffPending reports whether this spec is in the originator's
// "elsewhere" states: handed_off or awaiting_peer. These are excluded
// from the active-delivering invariant (per spec-status-integrity)
// because the actual work is happening in a peer workspace.
func (s *Spec) IsHandoffPending() bool {
	return s.Status == StatusHandedOff || s.Status == StatusAwaitingPeer
}

// IsHandedBack reports whether the peer has completed (or bounced)
// and the originator must verify or pick the spec back up.
func (s *Spec) IsHandedBack() bool {
	return s.Status == StatusHandedBack
}

// IsLocallyDelivering reports whether the spec is being actively
// worked on by this workspace (i.e. counts toward the
// active-delivering invariant). Excludes handoff-pending states
// where the work is happening on a peer.
//
// Coordination point with spec-status-integrity: when that spec adds
// an explicit "max concurrent active-delivering" invariant, it must
// consult IsLocallyDelivering (not IsInFlight) so the handoff states
// are excluded. Until that integrity check lands, this helper
// documents the contract at the source.
//
// TODO(cross-repo-peering, spec-status-integrity): wire the active-
// delivering invariant check to use IsLocallyDelivering.
func (s *Spec) IsLocallyDelivering() bool {
	return s.Status == StatusDelivering
}

// EffectiveHorizon returns the spec's horizon, defaulting to
// HorizonNow when the field is unset (legacy specs predating
// spec-prioritization). Callers should use this rather than reading
// s.Horizon directly so the default is consistent across surfaces.
func (s *Spec) EffectiveHorizon() Horizon {
	if IsValidHorizon(s.Horizon) {
		return s.Horizon
	}
	return HorizonNow
}

// IsActiveHorizon reports whether this spec is in the actionable-now
// set per spec-prioritization: horizon ∈ {now, next}. Hidden-by-
// default views (someday, parking) return false.
func (s *Spec) IsActiveHorizon() bool {
	h := s.EffectiveHorizon()
	return h == HorizonNow || h == HorizonNext
}

// SetFrontmatterField sets or updates a key-value pair in YAML frontmatter.
// If no frontmatter exists, one is created. Returns the updated content.
func SetFrontmatterField(content, key, value string) string {
	lines := strings.Split(content, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "---\n" + key + ": " + value + "\n---\n" + content
	}

	// Find closing ---
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx < 0 {
		return "---\n" + key + ": " + value + "\n---\n" + content
	}

	// Look for existing key
	found := false
	for j := 1; j < closeIdx; j++ {
		trimmed := strings.TrimSpace(lines[j])
		if strings.HasPrefix(trimmed, key+":") {
			lines[j] = key + ": " + value
			found = true
			break
		}
	}
	if !found {
		newLine := key + ": " + value
		// For hero-level fields (no tracker prefix), insert before any
		// tracker comment section (# Jira, # Github, # Linear) so the
		// field lands in the hero section of the frontmatter.
		insertIdx := closeIdx
		if !strings.Contains(key, "_") {
			for j := 1; j < closeIdx; j++ {
				trimmed := strings.TrimSpace(lines[j])
				if strings.HasPrefix(trimmed, "# ") {
					insertIdx = j
					break
				}
			}
		}
		lines = append(lines[:insertIdx], append([]string{newLine}, lines[insertIdx:]...)...)
	}

	return strings.Join(lines, "\n")
}
