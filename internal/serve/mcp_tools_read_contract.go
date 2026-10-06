package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/nextdoc"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/workmodel"
)

// The read contract (read-contract-v1): read-only JSON views of the work
// for every Hero client. They re-index when stale and never write.

// SpecRef is a related spec, as hero_spec reports it.
type SpecRef struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// SpecRelations groups a spec's relations by kind.
type SpecRelations struct {
	Parent    *SpecRef  `json:"parent"`
	Children  []SpecRef `json:"children"`
	DependsOn []SpecRef `json:"depends_on"`
	Blocks    []SpecRef `json:"blocks"`
	Related   []SpecRef `json:"related"`
}

// SpecAC is one acceptance criterion with its latest recorded result.
type SpecAC struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	State string `json:"state"`
}

// HeroSpec is hero_spec's reply.
type HeroSpec struct {
	Item      workmodel.Item `json:"item"`
	Body      string         `json:"body"`
	Relations SpecRelations  `json:"relations"`
	ACs       []SpecAC       `json:"acs"`
}

// HeroHandoff is hero_handoff's reply.
type HeroHandoff struct {
	Markdown  string  `json:"markdown"`
	UpdatedAt *string `json:"updated_at"`
}

func (s *MCPServer) toolSpec(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	if strings.TrimSpace(slug) == "" {
		return "", fmt.Errorf("slug parameter is required")
	}
	s.ensureFreshIndex()
	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}
	corpus := workmodel.NewCorpus(specs)
	target := corpus.Lookup(slug)
	if target == nil {
		return "", fmt.Errorf("no spec with slug %q", slug)
	}
	item, ok := workmodel.BuildOne(target, specs, workmodel.Options{Root: s.projectRoot, Now: readContractNow()})
	if !ok {
		return "", fmt.Errorf("%q is a %s, not a work spec (hero_spec serves features, bugs, enhancements, initiatives, epics and initiative decisions)", slug, target.Type)
	}
	item.Next = workmodel.NextFor(item, target, corpus)

	reply := HeroSpec{
		Item:      item,
		Body:      specBody(target.RawContent),
		Relations: specRelations(target, specs, corpus),
		ACs:       s.specACs(target),
	}
	data, err := json.Marshal(reply)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *MCPServer) toolHandoff(map[string]interface{}) (string, error) {
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	reply := HeroHandoff{}
	data, err := os.ReadFile(nextdoc.HandoffPath(s.heroDir, cfg))
	if err == nil {
		reply.Markdown = string(data)
		if updated := frontmatterValue(reply.Markdown, "updated"); updated != "" {
			reply.UpdatedAt = &updated
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("reading handoff: %w", err)
	}
	out, err := json.Marshal(reply)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// specBody returns the Markdown with its YAML frontmatter removed.
func specBody(raw string) string {
	if !strings.HasPrefix(raw, "---\n") {
		return raw
	}
	end := strings.Index(raw[4:], "\n---")
	if end < 0 {
		return raw
	}
	rest := raw[4+end+4:]
	return strings.TrimLeft(rest, "\r\n")
}

// frontmatterValue returns a scalar from the first `---`-delimited block,
// unquoted. NEXT.md carries a managed snapshot block before its
// frontmatter, so the block need not open the file.
func frontmatterValue(raw, key string) string {
	lines := strings.Split(raw, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimRight(line, "\r") != "---" {
			continue
		}
		if start < 0 {
			start = i
			continue
		}
		for _, l := range lines[start+1 : i] {
			if v, ok := strings.CutPrefix(l, key+":"); ok {
				return strings.Trim(strings.TrimSpace(v), `"'`)
			}
		}
		return ""
	}
	return ""
}

func specRelations(target *spec.Spec, specs []*spec.Spec, c *workmodel.Corpus) SpecRelations {
	ref := func(slug string) SpecRef {
		if s := c.Lookup(slug); s != nil {
			return SpecRef{Slug: s.Slug, Title: s.Title, Type: string(s.Type), Status: string(s.Status)}
		}
		return SpecRef{Slug: slug, Status: "missing"}
	}
	rel := SpecRelations{Children: []SpecRef{}, DependsOn: []SpecRef{}, Blocks: []SpecRef{}, Related: []SpecRef{}}
	seen := map[string]bool{}
	add := func(list *[]SpecRef, slug string) {
		key := fmt.Sprintf("%p:%s", list, slug)
		if slug == "" || slug == target.Slug || seen[key] {
			return
		}
		seen[key] = true
		*list = append(*list, ref(slug))
	}
	if p := workmodel.ParentSlug(target); p != "" {
		r := ref(p)
		rel.Parent = &r
	}
	for _, r := range target.Relations {
		switch r.Kind {
		case "depends-on", "depends_on":
			add(&rel.DependsOn, r.Target)
		case "blocks":
			add(&rel.Blocks, r.Target)
		case "child", "children":
			add(&rel.Children, r.Target)
		case "related", "relates-to", "relates_to", "supersedes", "conflicts-with":
			add(&rel.Related, r.Target)
		}
	}
	if target.Type == spec.TypeInitiative {
		for _, slug := range spec.DeclaredChildren(target) {
			add(&rel.Children, slug)
		}
	}
	for _, s := range specs {
		if workmodel.ParentSlug(s) == target.Slug {
			add(&rel.Children, s.Slug)
		}
	}
	return rel
}

// specACs pairs the spec's acceptance criteria with their latest recorded
// result (`hero spec verify` writes these to the graph); unknown otherwise.
func (s *MCPServer) specACs(target *spec.Spec) []SpecAC {
	states := map[string]string{}
	if store, err := graph.Open(s.heroDir); err == nil {
		if criteria, err := acceptance.ListBySpec(store, target.Slug); err == nil {
			for _, c := range criteria {
				switch c.Status {
				case "passing":
					states[c.ACID] = "pass"
				case "failing", "regressed":
					states[c.ACID] = "fail"
				}
			}
		}
		store.Close()
	}
	acs := []SpecAC{}
	for _, ac := range target.ParseAcceptanceCriteria() {
		state := states[ac.ID]
		if state == "" {
			state = "unknown"
		}
		acs = append(acs, SpecAC{ID: ac.ID, Text: ac.Statement, State: state})
	}
	return acs
}

// readContractNow is the clock the read tools use; tests pin it.
var readContractNow = time.Now
