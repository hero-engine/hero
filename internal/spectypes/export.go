package spectypes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// SchemaVersion is the JSON-export schema version. Bumped to 1.1 to
// carry kind, tasks_schema, and owner blocks alongside the v1 shape.
const SchemaVersion = "1.1"

// jsonExport is the wire-shape consumed by hero-code (Rust dashboard).
// Top-level shape is governed by §Cross-language contract of the
// spec-type-registry feature spec.
type jsonExport struct {
	SchemaVersion string             `json:"schema_version"`
	ActiveDomain  string             `json:"active_domain"`
	GeneratedAt   string             `json:"generated_at"`
	Types         []jsonExportRecord `json:"types"`
}

type jsonExportRecord struct {
	Name              string                  `json:"name"`
	Title             string                  `json:"title"`
	Domain            string                  `json:"domain"`
	Category          string                  `json:"category"`
	Location          string                  `json:"location"`
	Bucket            string                  `json:"bucket"`
	Lifecycle         *jsonLifecycle          `json:"lifecycle,omitempty"`
	Kind              *jsonKind               `json:"kind,omitempty"`
	Owner             *jsonOwner              `json:"owner,omitempty"`
	Frontmatter       *jsonFrontmatter        `json:"frontmatter,omitempty"`
	Sections          *jsonSections           `json:"sections,omitempty"`
	TasksSchema       *jsonTasksSchema        `json:"tasks_schema,omitempty"`
	AcceptingCommands []string                `json:"accepting_commands,omitempty"`
	DefaultAgents     map[string]string       `json:"default_agents,omitempty"`
	Relations         []jsonRelation          `json:"relations,omitempty"`
}

type jsonLifecycle struct {
	States      []string         `json:"states"`
	Initial     string           `json:"initial,omitempty"`
	Terminal    []string         `json:"terminal,omitempty"`
	Transitions []jsonTransition `json:"transitions,omitempty"`
}

type jsonTransition struct {
	From      string         `json:"from"`
	To        string         `json:"to"`
	Gate      string         `json:"gate,omitempty"`
	OwnerFlip *jsonOwnerFlip `json:"owner_flip,omitempty"`
}

type jsonOwnerFlip struct {
	To string `json:"to"`
}

type jsonKind struct {
	Values      []string `json:"values"`
	Default     string   `json:"default,omitempty"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
}

type jsonOwner struct {
	Values         []string `json:"values"`
	Default        string   `json:"default,omitempty"`
	Classification string   `json:"classification,omitempty"`
}

type jsonFrontmatter struct {
	Required []jsonField `json:"required,omitempty"`
	Optional []jsonField `json:"optional,omitempty"`
}

type jsonField struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	Required       bool     `json:"required,omitempty"`
	Default        string   `json:"default,omitempty"`
	Values         []string `json:"values,omitempty"`
	Format         string   `json:"format,omitempty"`
	Classification string   `json:"classification,omitempty"`
	Description    string   `json:"description,omitempty"`
}

type jsonSections struct {
	Required []string `json:"required,omitempty"`
	Optional []string `json:"optional,omitempty"`
}

type jsonTasksSchema struct {
	Required       bool                 `json:"required"`
	SectionHeading string               `json:"section_heading,omitempty"`
	History        string               `json:"history,omitempty"`
	ItemShape      map[string]jsonField `json:"item_shape,omitempty"`
}

type jsonRelation struct {
	Kind        string `json:"kind"`
	TargetType  string `json:"target_type"`
	Cardinality string `json:"cardinality,omitempty"`
}

// JSONSchema returns the registry's data marshaled as the
// cross-language schema-1.1 JSON document. Stable byte output for the
// same registry (sorted record order via Load).
func (r *registry) JSONSchema() ([]byte, error) {
	doc := buildExport(r)
	return json.MarshalIndent(doc, "", "  ")
}

func buildExport(r *registry) jsonExport {
	doc := jsonExport{
		SchemaVersion: SchemaVersion,
		ActiveDomain:  r.activeDomain,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	for _, name := range r.order {
		doc.Types = append(doc.Types, exportRecord(r.records[name]))
	}
	return doc
}

func exportRecord(rec *Record) jsonExportRecord {
	out := jsonExportRecord{
		Name:              rec.Name,
		Title:             rec.Title,
		Domain:            rec.Domain,
		Category:          string(rec.Category),
		Location:          rec.Location,
		Bucket:            rec.Bucket,
		AcceptingCommands: append([]string(nil), rec.AcceptingCommands...),
	}
	if len(rec.Lifecycle.States) > 0 || len(rec.Lifecycle.Transitions) > 0 {
		lc := &jsonLifecycle{
			States:   append([]string(nil), rec.Lifecycle.States...),
			Initial:  rec.Lifecycle.Initial,
			Terminal: append([]string(nil), rec.Lifecycle.Terminal...),
		}
		for _, t := range rec.Lifecycle.Transitions {
			jt := jsonTransition{From: t.From, To: t.To, Gate: t.Gate}
			if t.OwnerFlip != nil {
				jt.OwnerFlip = &jsonOwnerFlip{To: t.OwnerFlip.To}
			}
			lc.Transitions = append(lc.Transitions, jt)
		}
		out.Lifecycle = lc
	}
	if rec.Kind.HasKind() {
		out.Kind = &jsonKind{
			Values:      append([]string(nil), rec.Kind.Values...),
			Default:     rec.Kind.Default,
			Required:    rec.Kind.Required,
			Description: rec.Kind.Description,
		}
	}
	if len(rec.Owner.Values) > 0 {
		out.Owner = &jsonOwner{
			Values:         append([]string(nil), rec.Owner.Values...),
			Default:        rec.Owner.Default,
			Classification: string(rec.Owner.Classification),
		}
	}
	if len(rec.Sections.Required) > 0 || len(rec.Sections.Optional) > 0 {
		out.Sections = &jsonSections{
			Required: append([]string(nil), rec.Sections.Required...),
			Optional: append([]string(nil), rec.Sections.Optional...),
		}
	}
	if rec.TasksSchema.HasTasks() {
		ts := &jsonTasksSchema{
			Required:       rec.TasksSchema.Required,
			SectionHeading: rec.TasksSchema.SectionHeading,
			History:        string(rec.TasksSchema.History),
		}
		if len(rec.TasksSchema.ItemShape) > 0 {
			ts.ItemShape = map[string]jsonField{}
			for fname, f := range rec.TasksSchema.ItemShape {
				ts.ItemShape[fname] = jsonField{
					Name:     fname,
					Type:     f.Type,
					Required: f.Required,
					Default:  f.Default,
					Values:   append([]string(nil), f.Values...),
					Format:   f.Format,
				}
			}
		}
		out.TasksSchema = ts
	}
	if len(rec.DefaultAgents) > 0 {
		out.DefaultAgents = map[string]string{}
		for k, v := range rec.DefaultAgents {
			out.DefaultAgents[k] = v
		}
	}
	for _, rel := range rec.Relations {
		out.Relations = append(out.Relations, jsonRelation{
			Kind:        rel.Kind,
			TargetType:  rel.TargetType,
			Cardinality: rel.Cardinality,
		})
	}
	if len(rec.Frontmatter.Required) > 0 || len(rec.Frontmatter.Optional) > 0 {
		fm := &jsonFrontmatter{}
		for _, f := range rec.Frontmatter.Required {
			fm.Required = append(fm.Required, jsonField{
				Name: f.Name, Type: f.Type, Required: f.Required,
				Default: f.Default, Values: f.Values, Format: f.Format,
				Classification: string(f.Classification), Description: f.Description,
			})
		}
		for _, f := range rec.Frontmatter.Optional {
			fm.Optional = append(fm.Optional, jsonField{
				Name: f.Name, Type: f.Type, Required: f.Required,
				Default: f.Default, Values: f.Values, Format: f.Format,
				Classification: string(f.Classification), Description: f.Description,
			})
		}
		out.Frontmatter = fm
	}
	return out
}

// ExportTo writes the registry's JSON manifest to the given workspace
// root, creating the .hero/cache/ directory if needed. Path is
// <workspaceRoot>/.hero/cache/spec-types.json per the cross-language
// contract.
//
// Skips the write when only the generated_at timestamp changed. This
// export runs in the CLI's PersistentPreRun on every `hero` invocation,
// so an unconditional write would re-stamp the timestamp constantly,
// leaving .hero/cache/spec-types.json perpetually dirty and racing the
// pre-commit hook that stages it. Mirrors WriteManifest's guard.
func ExportTo(reg Registry, workspaceRoot string) error {
	data, err := reg.JSONSchema()
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}
	cacheDir := filepath.Join(workspaceRoot, ".hero", "cache")
	outPath := filepath.Join(cacheDir, "spec-types.json")

	if existing, readErr := os.ReadFile(outPath); readErr == nil {
		if stripGeneratedAt(existing) == stripGeneratedAt(data) {
			return nil
		}
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	return nil
}

var reGeneratedAt = regexp.MustCompile(`"generated_at": ".*?"`)

// stripGeneratedAt blanks the generated_at value so two exports that
// differ only by timestamp compare equal.
func stripGeneratedAt(b []byte) string {
	return reGeneratedAt.ReplaceAllString(string(b), `"generated_at": ""`)
}
