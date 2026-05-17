package spectypes

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	hero "github.com/hero-engine/hero"
	"gopkg.in/yaml.v3"
)

// Load builds a Registry by reading core/spec-types/*.md and the
// active domain's domains/<domain>/spec-types/*.md. Domain types
// overlay core: a domain may *extend* core (add types) but may not
// *redefine* a core type — collision is a load error.
//
// activeDomain is consulted to pick the domain overlay. Empty or
// "engineering" both select engineering.
func Load(activeDomain string) (Registry, error) {
	if activeDomain == "" {
		activeDomain = "engineering"
	}

	reg := &registry{
		records:      map[string]*Record{},
		activeDomain: activeDomain,
	}

	// Core first.
	coreFS := hero.CoreSpecTypesFS()
	if err := loadFromFS(reg, coreFS, "core"); err != nil {
		return nil, fmt.Errorf("loading core spec-types: %w", err)
	}

	// Domain overlay. Missing directory is fine — not every domain has
	// extension types.
	if domainFS := hero.DomainSpecTypesFS(activeDomain); domainFS != nil {
		if err := loadFromFS(reg, domainFS, activeDomain); err != nil {
			return nil, fmt.Errorf("loading %s spec-types: %w", activeDomain, err)
		}
	}

	// Deterministic order: load order is preserved within each
	// scope, but sort within each scope alphabetically by name for
	// stable JSON export.
	sort.Strings(reg.order)

	if len(reg.records) == 0 {
		return nil, fmt.Errorf("registry is empty after loading; core/spec-types/ produced no records")
	}

	return reg, nil
}

// LoadFromFS is a test seam: build a registry from arbitrary
// filesystems rather than the embedded one. coreFS must be non-nil;
// domainFS may be nil.
func LoadFromFS(coreFS, domainFS fs.FS, activeDomain string) (Registry, error) {
	if activeDomain == "" {
		activeDomain = "engineering"
	}
	reg := &registry{
		records:      map[string]*Record{},
		activeDomain: activeDomain,
	}
	if err := loadFromFS(reg, coreFS, "core"); err != nil {
		return nil, fmt.Errorf("loading core spec-types: %w", err)
	}
	if domainFS != nil {
		if err := loadFromFS(reg, domainFS, activeDomain); err != nil {
			return nil, fmt.Errorf("loading %s spec-types: %w", activeDomain, err)
		}
	}
	sort.Strings(reg.order)
	if len(reg.records) == 0 {
		return nil, fmt.Errorf("registry is empty")
	}
	return reg, nil
}

func loadFromFS(reg *registry, fsys fs.FS, scope string) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("reading spec-types directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if strings.EqualFold(name, "README.md") {
			continue
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		rec, err := parseRecord(data, name, scope)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", name, err)
		}
		if rec == nil {
			continue
		}
		if existing, dup := reg.records[rec.Name]; dup {
			// Domain re-declaring a core type is a hard error.
			if scope != "core" && existing.Domain == "core" {
				return fmt.Errorf(
					"domain %q attempts to redefine core type %q (declared at core/spec-types/%s.md)",
					scope, rec.Name, rec.Name)
			}
			return fmt.Errorf("duplicate spec-type %q in scope %q", rec.Name, scope)
		}
		reg.records[rec.Name] = rec
		reg.order = append(reg.order, rec.Name)
	}
	return nil
}

// rawFrontmatter mirrors the YAML shape we expect in each spec-type
// markdown file's frontmatter. Anything optional is zero-value-safe.
type rawFrontmatter struct {
	Title             string           `yaml:"title"`
	Type              string           `yaml:"type"`
	Domain            string           `yaml:"domain"`
	Category          string           `yaml:"category"`
	Bucket            string           `yaml:"bucket"`
	Location          string           `yaml:"location"`
	Lifecycle         *rawLifecycle    `yaml:"lifecycle,omitempty"`
	Kind              *rawKind         `yaml:"kind,omitempty"`
	Owner             *rawOwner        `yaml:"owner,omitempty"`
	TasksSchema       *rawTasksSchema  `yaml:"tasks_schema,omitempty"`
	Sections          *rawSections     `yaml:"sections,omitempty"`
	AcceptingCommands []string         `yaml:"accepting_commands,omitempty"`
	DefaultAgents     map[string]string `yaml:"default_agents,omitempty"`
	Relations         []rawRelation    `yaml:"relations,omitempty"`
	Frontmatter       *rawFrontSchema  `yaml:"frontmatter,omitempty"`
}

type rawLifecycle struct {
	States      []string         `yaml:"states"`
	Initial     string           `yaml:"initial"`
	Terminal    []string         `yaml:"terminal"`
	Transitions []rawTransition  `yaml:"transitions"`
}

type rawTransition struct {
	From      string        `yaml:"from"`
	To        string        `yaml:"to"`
	Gate      string        `yaml:"gate"`
	OwnerFlip *rawOwnerFlip `yaml:"owner_flip,omitempty"`
}

type rawOwnerFlip struct {
	To string `yaml:"to"`
}

type rawKind struct {
	Values      []string `yaml:"values"`
	Default     string   `yaml:"default"`
	Required    bool     `yaml:"required"`
	Description string   `yaml:"description"`
}

type rawOwner struct {
	Values         []string `yaml:"values"`
	Default        string   `yaml:"default"`
	Classification string   `yaml:"classification"`
}

type rawTasksSchema struct {
	Required       bool                       `yaml:"required"`
	SectionHeading string                     `yaml:"section_heading"`
	History        string                     `yaml:"history"`
	ItemShape      map[string]rawItemField    `yaml:"item_shape"`
}

type rawItemField struct {
	Type     string   `yaml:"type"`
	Required bool     `yaml:"required"`
	Default  string   `yaml:"default"`
	Values   []string `yaml:"values"`
	Format   string   `yaml:"format"`
}

type rawSections struct {
	Required []string `yaml:"required"`
	Optional []string `yaml:"optional"`
}

type rawRelation struct {
	Kind        string `yaml:"kind"`
	TargetType  string `yaml:"target_type"`
	Cardinality string `yaml:"cardinality"`
}

type rawFrontSchema struct {
	Required []rawFieldDecl `yaml:"required"`
	Optional []rawFieldDecl `yaml:"optional"`
}

type rawFieldDecl struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"`
	Required       bool     `yaml:"required"`
	Default        string   `yaml:"default"`
	Values         []string `yaml:"values"`
	Format         string   `yaml:"format"`
	Classification string   `yaml:"classification"`
	Description    string   `yaml:"description"`
}

// parseRecord extracts the YAML frontmatter block from raw markdown
// and converts it into a *Record. Returns (nil, nil) if the file has
// no frontmatter and should be skipped (e.g. README.md slipped past
// the name filter).
func parseRecord(data []byte, filename, scope string) (*Record, error) {
	fm, err := extractFrontmatter(data)
	if err != nil {
		return nil, err
	}
	if len(fm) == 0 {
		return nil, nil
	}

	var raw rawFrontmatter
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	if raw.Type == "" {
		return nil, fmt.Errorf("frontmatter missing required field 'type'")
	}
	if raw.Category == "" {
		return nil, fmt.Errorf("frontmatter missing required field 'category'")
	}

	cat := Category(raw.Category)
	if cat != CategoryWork && cat != CategoryKnowledge {
		return nil, fmt.Errorf("invalid category %q (must be work or knowledge)", raw.Category)
	}

	rec := &Record{
		Name:     raw.Type,
		Title:    raw.Title,
		Domain:   defaultStr(raw.Domain, scope),
		Category: cat,
		Bucket:   raw.Bucket,
		Location: raw.Location,
	}

	if raw.Lifecycle != nil {
		lc := Lifecycle{
			States:   append([]string(nil), raw.Lifecycle.States...),
			Initial:  raw.Lifecycle.Initial,
			Terminal: append([]string(nil), raw.Lifecycle.Terminal...),
		}
		for _, t := range raw.Lifecycle.Transitions {
			tr := Transition{From: t.From, To: t.To, Gate: t.Gate}
			if t.OwnerFlip != nil {
				tr.OwnerFlip = &OwnerFlip{To: t.OwnerFlip.To}
			}
			lc.Transitions = append(lc.Transitions, tr)
		}
		rec.Lifecycle = lc

		// Referential integrity: every transition.from/to must be a
		// declared state; initial / terminal must be declared too.
		stateSet := map[string]bool{}
		for _, s := range lc.States {
			stateSet[s] = true
		}
		if lc.Initial != "" && !stateSet[lc.Initial] {
			return nil, fmt.Errorf("lifecycle.initial %q is not in lifecycle.states", lc.Initial)
		}
		for _, term := range lc.Terminal {
			if !stateSet[term] {
				return nil, fmt.Errorf("lifecycle.terminal %q is not in lifecycle.states", term)
			}
		}
		for _, t := range lc.Transitions {
			if !stateSet[t.From] {
				return nil, fmt.Errorf("transition.from %q is not in lifecycle.states", t.From)
			}
			if !stateSet[t.To] {
				return nil, fmt.Errorf("transition.to %q is not in lifecycle.states", t.To)
			}
		}
	}

	if raw.Kind != nil {
		rec.Kind = KindSchema{
			Values:      append([]string(nil), raw.Kind.Values...),
			Default:     raw.Kind.Default,
			Required:    raw.Kind.Required,
			Description: raw.Kind.Description,
		}
		// distinct values
		seen := map[string]bool{}
		for _, v := range rec.Kind.Values {
			if seen[v] {
				return nil, fmt.Errorf("kind.values contains duplicate %q", v)
			}
			seen[v] = true
		}
		if rec.Kind.Default != "" && !seen[rec.Kind.Default] {
			return nil, fmt.Errorf("kind.default %q is not in kind.values", rec.Kind.Default)
		}
	}

	if raw.Owner != nil {
		rec.Owner = OwnerSchema{
			Values:         append([]string(nil), raw.Owner.Values...),
			Default:        raw.Owner.Default,
			Classification: Classification(raw.Owner.Classification),
		}
	}

	if raw.TasksSchema != nil {
		ts := ChecklistSchema{
			Required:       raw.TasksSchema.Required,
			SectionHeading: raw.TasksSchema.SectionHeading,
			History:        HistoryMode(raw.TasksSchema.History),
			ItemShape:      map[string]FieldDecl{},
		}
		for fname, f := range raw.TasksSchema.ItemShape {
			ts.ItemShape[fname] = FieldDecl{
				Name:     fname,
				Type:     f.Type,
				Required: f.Required,
				Default:  f.Default,
				Values:   append([]string(nil), f.Values...),
				Format:   f.Format,
			}
		}
		rec.TasksSchema = ts
	}

	if raw.Sections != nil {
		rec.Sections = SectionSpec{
			Required: append([]string(nil), raw.Sections.Required...),
			Optional: append([]string(nil), raw.Sections.Optional...),
		}
	}

	rec.AcceptingCommands = append([]string(nil), raw.AcceptingCommands...)
	if len(raw.DefaultAgents) > 0 {
		rec.DefaultAgents = map[string]string{}
		for k, v := range raw.DefaultAgents {
			rec.DefaultAgents[k] = v
		}
	}
	for _, rel := range raw.Relations {
		rec.Relations = append(rec.Relations, RelationDecl{
			Kind:        rel.Kind,
			TargetType:  rel.TargetType,
			Cardinality: rel.Cardinality,
		})
	}

	if raw.Frontmatter != nil {
		for _, f := range raw.Frontmatter.Required {
			rec.Frontmatter.Required = append(rec.Frontmatter.Required, convertField(f))
		}
		for _, f := range raw.Frontmatter.Optional {
			rec.Frontmatter.Optional = append(rec.Frontmatter.Optional, convertField(f))
		}
	}

	_ = path.Base // keep import shape ready for future expansion
	_ = filename
	return rec, nil
}

func convertField(f rawFieldDecl) FieldDecl {
	return FieldDecl{
		Name:           f.Name,
		Type:           f.Type,
		Required:       f.Required,
		Default:        f.Default,
		Values:         append([]string(nil), f.Values...),
		Format:         f.Format,
		Classification: Classification(f.Classification),
		Description:    f.Description,
	}
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// extractFrontmatter pulls the YAML block bounded by "---\n" markers
// at the very start of a markdown file. Returns nil if no frontmatter
// is present (caller treats that as "skip this file").
func extractFrontmatter(data []byte) ([]byte, error) {
	const delim = "---\n"
	if !bytes.HasPrefix(data, []byte(delim)) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		return nil, nil
	}
	// Find the closing delimiter.
	rest := data[len(delim):]
	if bytes.HasPrefix(data, []byte("---\r\n")) {
		rest = data[len("---\r\n"):]
	}
	endIdx := bytes.Index(rest, []byte("\n---\n"))
	if endIdx < 0 {
		endIdx = bytes.Index(rest, []byte("\n---\r\n"))
	}
	if endIdx < 0 {
		return nil, fmt.Errorf("frontmatter has no closing '---' delimiter")
	}
	return rest[:endIdx], nil
}
