// Package spectypes implements Hero's spec-type registry — the
// single in-process source of truth for what spec types exist, what
// frontmatter fields they accept, what lifecycle states they declare,
// and what kinds, owners, sections, and relations they carry.
//
// The registry loads canonical type declarations from
// core/spec-types/*.md at process start and overlays the active
// domain's extensions from domains/<active>/spec-types/*.md. Each
// declaration is a markdown file whose YAML frontmatter encodes the
// full schema; the prose section documents the type for humans.
//
// The registry is read-only after construction. One Registry per
// process. Callers needing display names compose registry data with
// the active vocabulary (internal/vocabulary) at render time.
//
// A language-neutral JSON export is written to
// .hero/cache/spec-types.json at schema version 1.1. See export.go.
package spectypes

// Category classifies a record as work-tracking or knowledge.
type Category string

const (
	CategoryWork      Category = "work"
	CategoryKnowledge Category = "knowledge"
)

// Classification tags a frontmatter field as content (Hero-owned) or
// org-state (tracker-owned under tracker-fronting policy). Surfaced
// in the JSON export so the integration layer can apply the right
// conflict policy.
type Classification string

const (
	ClassificationContent  Classification = "content"
	ClassificationOrgState Classification = "org-state"
)

// HistoryMode declares whether a checklist tracks bitemporal status
// flips (current AC behavior) or only the latest state.
type HistoryMode string

const (
	HistoryBitemporal HistoryMode = "bitemporal"
	HistoryNone       HistoryMode = "none"
)

// Record is a single spec-type declaration. Built by the loader from
// one core/spec-types/<name>.md or domains/<x>/spec-types/<name>.md
// file. Read-only after construction.
type Record struct {
	Name     string   // canonical type name ("feature", "bug", ...)
	Title    string   // display label
	Domain   string   // "core" or "<domain>"
	Category Category // work | knowledge
	Bucket   string   // folder bucket ("features", "bugs", ...)
	Location string   // location template, e.g. ".hero/planning/features/{slug}/spec.md"

	Lifecycle         Lifecycle
	Kind              KindSchema
	Owner             OwnerSchema
	Sections          SectionSpec
	TasksSchema       ChecklistSchema
	AcceptingCommands []string
	// ExtensionPoints declares the compatible amendment surfaces owned by
	// this canonical type. Packs may amend only a point named here.
	ExtensionPoints []string
	DefaultAgents   map[string]string
	Relations       []RelationDecl
	Frontmatter     FrontmatterSchema
}

// Lifecycle declares the state machine for a record.
type Lifecycle struct {
	States      []string
	Initial     string
	Terminal    []string
	Transitions []Transition
}

// Transition is one declared state edge.
type Transition struct {
	From      string
	To        string
	Gate      string
	OwnerFlip *OwnerFlip
}

// OwnerFlip is the registry's contract that a transition triggers an
// owner change. The handoff-coordinator agent reads this to know when
// to flip ownership.
type OwnerFlip struct {
	To string
}

// KindSchema declares a record's canonical kind enum.
type KindSchema struct {
	Values      []string
	Default     string
	Required    bool
	Description string
}

// HasKind reports whether this record declares a kind block at all.
func (k KindSchema) HasKind() bool {
	return len(k.Values) > 0
}

// OwnerSchema declares the per-record owner enum and classification.
type OwnerSchema struct {
	Values         []string
	Default        string
	Classification Classification
}

// SectionSpec declares required vs optional section headings.
type SectionSpec struct {
	Required []string
	Optional []string
}

// ChecklistSchema is the shape declaration for `## Tasks` (and could
// be reused for AC if/when AC is brought through the same lens; the
// AC infrastructure is intentionally untouched in this rework).
type ChecklistSchema struct {
	Required       bool
	SectionHeading string
	History        HistoryMode
	ItemShape      map[string]FieldDecl
}

// HasTasks reports whether this record declares any tasks_schema block.
func (c ChecklistSchema) HasTasks() bool {
	return c.SectionHeading != "" || len(c.ItemShape) > 0
}

// RelationDecl is one outgoing relation declaration.
type RelationDecl struct {
	Kind        string // "parent", "child", "blocks", ...
	TargetType  string
	Cardinality string // "zero-or-one", "many", ...
}

// FieldDecl is one frontmatter or item-shape field.
type FieldDecl struct {
	Name           string
	Type           string // "string", "int", "bool", "date", "enum", "list[T]", "ref(<type>)"
	Required       bool
	Default        string
	Values         []string // enum values, when Type=="enum"
	Format         string   // optional shape hint, e.g. "T-<int>"
	Classification Classification
	Description    string
}

// FrontmatterSchema enumerates the required and optional frontmatter
// fields a spec of this type should carry.
type FrontmatterSchema struct {
	Required []FieldDecl
	Optional []FieldDecl
}

// Amendment is a namespaced compatible extension to a canonical type.
type Amendment struct {
	ID             string   `json:"id"`
	Owner          string   `json:"owner"`
	TargetType     string   `json:"target_type"`
	ExtensionPoint string   `json:"extension_point"`
	Values         []string `json:"values,omitempty"`
}

// Registry is the read-only accessor surface. One instance per
// process, built once at startup by Load.
type Registry interface {
	All() []*Record
	Lookup(name string) (*Record, bool)
	LookupFolder(folder string) (*Record, bool)
	LookupByKind(name, kind string) (*Record, bool)
	Kinds(typeName string) []string
	WorkTypes() []*Record
	KnowledgeTypes() []*Record
	AcceptingCommand(cmd string) []*Record
	DefaultWorkType() *Record
	ActiveDomain() string
	Amendments() []Amendment
	JSONSchema() ([]byte, error)
}

// registry is the concrete Registry implementation. Returned by Load.
type registry struct {
	records      map[string]*Record
	order        []string // deterministic iteration order (load order)
	activeDomain string
	amendments   []Amendment
}

func (r *registry) ActiveDomain() string { return r.activeDomain }
func (r *registry) Amendments() []Amendment {
	out := make([]Amendment, len(r.amendments))
	copy(out, r.amendments)
	return out
}

func (r *registry) All() []*Record {
	out := make([]*Record, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.records[name])
	}
	return out
}

func (r *registry) Lookup(name string) (*Record, bool) {
	rec, ok := r.records[name]
	return rec, ok
}

func (r *registry) LookupFolder(folder string) (*Record, bool) {
	for _, rec := range r.records {
		if rec.Bucket == folder {
			return rec, true
		}
	}
	return nil, false
}

func (r *registry) LookupByKind(name, kind string) (*Record, bool) {
	rec, ok := r.records[name]
	if !ok {
		return nil, false
	}
	for _, v := range rec.Kind.Values {
		if v == kind {
			return rec, true
		}
	}
	return nil, false
}

func (r *registry) Kinds(typeName string) []string {
	rec, ok := r.records[typeName]
	if !ok {
		return nil
	}
	if len(rec.Kind.Values) == 0 {
		return nil
	}
	out := make([]string, len(rec.Kind.Values))
	copy(out, rec.Kind.Values)
	return out
}

func (r *registry) WorkTypes() []*Record {
	var out []*Record
	for _, name := range r.order {
		if r.records[name].Category == CategoryWork {
			out = append(out, r.records[name])
		}
	}
	return out
}

func (r *registry) KnowledgeTypes() []*Record {
	var out []*Record
	for _, name := range r.order {
		if r.records[name].Category == CategoryKnowledge {
			out = append(out, r.records[name])
		}
	}
	return out
}

func (r *registry) AcceptingCommand(cmd string) []*Record {
	var out []*Record
	for _, name := range r.order {
		rec := r.records[name]
		for _, c := range rec.AcceptingCommands {
			if c == cmd {
				out = append(out, rec)
				break
			}
		}
	}
	return out
}

// DefaultWorkType returns the canonical default for hero new with no
// --type flag. "feature" — the unit-of-work type — under the v1
// registry. Returns nil if "feature" is not registered (which would
// itself be a load-time error).
func (r *registry) DefaultWorkType() *Record {
	if rec, ok := r.records["feature"]; ok {
		return rec
	}
	return nil
}
