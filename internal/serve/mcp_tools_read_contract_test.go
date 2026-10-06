package serve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeContractSpec(t *testing.T, heroDir, rel, content string) {
	t.Helper()
	path := filepath.Join(heroDir, rel, "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readContractWorkspace(t *testing.T) (*MCPServer, string) {
	t.Helper()
	root := t.TempDir()
	heroDir := filepath.Join(root, ".hero")
	writeContractSpec(t, heroDir, "planning/initiatives/init", "---\ntitle: Init\nslug: init\ntype: initiative\nstatus: planning\nchild:\n  - feat\n---\n# Init\n")
	writeContractSpec(t, heroDir, "planning/initiatives/init/feat", "---\ntitle: \"Feat: one\"\nslug: feat\ntype: feature\nstatus: planning\npriority: P1\nparent: init\ndepends-on: [base]\nrelations:\n  - target: other\n    kind: related\n---\n\n# Feat\n\nBody text.\n\n## Changes\n\n1. Do it.\n\n## Acceptance Criteria\n\n- **AC-1:** THE SYSTEM SHALL one.\n- **AC-2:** THE SYSTEM SHALL two.\n")
	writeContractSpec(t, heroDir, "planning/features/base", "---\ntitle: Base\nslug: base\ntype: feature\nstatus: delivering\n---\n# Base\n")
	writeContractSpec(t, heroDir, "knowledge/conventions/naming", "---\ntitle: Naming\nslug: naming\ntype: convention\nstatus: active\n---\n# Naming\n")
	return NewMCPServer(heroDir, root, "1.0.0"), heroDir
}

// hero-spec-handoff-tools AC-1/AC-2: hero_spec returns the contract shape.
func TestToolSpecReturnsContractShape(t *testing.T) {
	srv, _ := readContractWorkspace(t)
	result := callTool(t, srv, "hero_spec", map[string]interface{}{"slug": "feat"})
	if result.IsError {
		t.Fatalf("hero_spec error: %s", result.Content[0].Text)
	}
	var got HeroSpec
	if err := json.Unmarshal([]byte(result.Content[0].Text), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, result.Content[0].Text)
	}
	if got.Item.Slug != "feat" || got.Item.Title != "Feat: one" || got.Item.Lane != "designed" || got.Item.Revision == "" {
		t.Errorf("item = %+v", got.Item)
	}
	if got.Item.Next == nil || got.Item.Next.Enabled || got.Item.Next.Reason == nil || *got.Item.Next.Reason != "waits on base" {
		t.Errorf("next = %+v", got.Item.Next)
	}
	if strings.HasPrefix(got.Body, "---") || !strings.Contains(got.Body, "Body text.") {
		t.Errorf("body must be Markdown without frontmatter: %q", got.Body)
	}
	if got.Relations.Parent == nil || got.Relations.Parent.Slug != "init" || got.Relations.Parent.Type != "initiative" {
		t.Errorf("parent = %+v", got.Relations.Parent)
	}
	if len(got.Relations.DependsOn) != 1 || got.Relations.DependsOn[0].Status != "delivering" {
		t.Errorf("depends_on = %+v", got.Relations.DependsOn)
	}
	if len(got.Relations.Related) != 1 || got.Relations.Related[0].Status != "missing" {
		t.Errorf("related (missing target) = %+v", got.Relations.Related)
	}
	if len(got.ACs) != 2 || got.ACs[0].ID != "AC-1" || got.ACs[0].State != "unknown" {
		t.Errorf("acs = %+v", got.ACs)
	}
	raw := result.Content[0].Text
	for _, key := range []string{`"children":[]`, `"blocks":[]`} {
		if !strings.Contains(raw, key) {
			t.Errorf("empty relation lists must be [], missing %s", key)
		}
	}

	parent := callTool(t, srv, "hero_spec", map[string]interface{}{"slug": "init"})
	var init HeroSpec
	_ = json.Unmarshal([]byte(parent.Content[0].Text), &init)
	if len(init.Relations.Children) != 1 || init.Relations.Children[0].Slug != "feat" {
		t.Errorf("initiative children = %+v", init.Relations.Children)
	}
}

// hero-spec-handoff-tools AC-3: non-work and unknown slugs are clear errors.
func TestToolSpecRejectsNonWorkAndUnknown(t *testing.T) {
	srv, _ := readContractWorkspace(t)
	for slug, want := range map[string]string{"naming": "not a work spec", "nope": "no spec with slug"} {
		result := callTool(t, srv, "hero_spec", map[string]interface{}{"slug": slug})
		if !result.IsError || !strings.Contains(result.Content[0].Text, want) {
			t.Errorf("%s: %+v", slug, result)
		}
	}
}

// hero-spec-handoff-tools AC-4: hero_handoff returns the briefing hero next shows.
func TestToolHandoff(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	empty := callTool(t, srv, "hero_handoff", map[string]interface{}{})
	if empty.IsError || empty.Content[0].Text != `{"markdown":"","updated_at":null}` {
		t.Fatalf("no briefing = %s", empty.Content[0].Text)
	}
	brief := "---\nupdated: \"2026-10-06T22:00:00Z\"\n---\n# Next\n\nDo the thing.\n"
	if err := os.WriteFile(filepath.Join(heroDir, "NEXT.md"), []byte(brief), 0o644); err != nil {
		t.Fatal(err)
	}
	result := callTool(t, srv, "hero_handoff", map[string]interface{}{})
	var got HeroHandoff
	if err := json.Unmarshal([]byte(result.Content[0].Text), &got); err != nil {
		t.Fatal(err)
	}
	if got.Markdown != brief || got.UpdatedAt == nil || *got.UpdatedAt != "2026-10-06T22:00:00Z" {
		t.Errorf("handoff = %+v", got)
	}
	// Real NEXT.md files open with a managed snapshot block before their
	// frontmatter.
	managed := "<!-- hero:managed-start v=dev -->\n## Project snapshot\n<!-- hero:managed-end -->\n\n---\nupdated: 2026-10-06T23:00:00Z\n---\n# Next\n"
	os.WriteFile(filepath.Join(heroDir, "NEXT.md"), []byte(managed), 0o644)
	result = callTool(t, srv, "hero_handoff", map[string]interface{}{})
	got = HeroHandoff{}
	_ = json.Unmarshal([]byte(result.Content[0].Text), &got)
	if got.UpdatedAt == nil || *got.UpdatedAt != "2026-10-06T23:00:00Z" {
		t.Errorf("managed-block NEXT.md updated_at = %v", got.UpdatedAt)
	}
}

// hero-spec-handoff-tools AC-5: both tools are read-only and write nothing
// under the contract's watch globs.
func TestReadContractToolsAreReadOnlyAndWriteNothing(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	before := snapshotTree(t, heroDir)
	callTool(t, srv, "hero_spec", map[string]interface{}{"slug": "feat"})
	callTool(t, srv, "hero_handoff", map[string]interface{}{})
	after := snapshotTree(t, heroDir)
	for _, dir := range []string{"planning", "specs", "knowledge", "next"} {
		for path, mod := range after {
			if strings.HasPrefix(path, dir+string(filepath.Separator)) && before[path] != mod {
				t.Errorf("read tool wrote %s", path)
			}
		}
	}
	if _, ok := after["NEXT.md"]; ok {
		t.Error("read tool created NEXT.md")
	}
	for _, name := range []string{"hero_spec", "hero_handoff"} {
		ann := annotationsForSafety(toolSafetyClasses()[name])
		if _, ok := toolSafetyClasses()[name]; !ok || ann.ReadOnlyHint == nil || !*ann.ReadOnlyHint {
			t.Errorf("%s must advertise readOnlyHint=true, got %+v", name, ann)
		}
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		out[rel] = info.ModTime().String() + ":" + string(rune(info.Size()))
		return nil
	})
	return out
}
