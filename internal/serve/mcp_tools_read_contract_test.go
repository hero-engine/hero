package serve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/graph"
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

// hero-spec-handoff-tools AC-4: hero_handoff returns the briefing hero next
// shows. handoff-shape: markdown is the briefing body only; updated and repo
// come back as fields.
func TestToolHandoff(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	empty := callTool(t, srv, "hero_handoff", map[string]interface{}{})
	if empty.IsError || empty.Content[0].Text != `{"markdown":"","updated_at":null,"repo":null}` {
		t.Fatalf("no briefing = %s", empty.Content[0].Text)
	}
	call := func(content string) HeroHandoff {
		t.Helper()
		if err := os.WriteFile(filepath.Join(heroDir, "NEXT.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		result := callTool(t, srv, "hero_handoff", map[string]interface{}{})
		if result.IsError {
			t.Fatal(result.Content[0].Text)
		}
		var got HeroHandoff
		if err := json.Unmarshal([]byte(result.Content[0].Text), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	got := call("---\nupdated: \"2026-10-06T22:00:00Z\"\n---\n# Next\n\nDo the thing.\n")
	if got.Markdown != "# Next\n\nDo the thing.\n" || got.UpdatedAt == nil || *got.UpdatedAt != "2026-10-06T22:00:00Z" || got.Repo != nil {
		t.Errorf("handoff = %+v", got)
	}
	// Real NEXT.md files open with a managed snapshot block before their
	// frontmatter; neither belongs in the briefing.
	got = call("<!-- hero:managed-start v=dev -->\n## Project snapshot\n\nProject shape: see [SNAPSHOT.md](.hero/SNAPSHOT.md).\n<!-- hero:managed-end -->\n\n---\nupdated: 2026-10-06T23:00:00Z\nrepo: hero-engine/hero-harness\n---\n\n## Just finished\n\nX\n")
	if got.Markdown != "## Just finished\n\nX\n" {
		t.Errorf("markdown = %q, want the briefing body only", got.Markdown)
	}
	if got.UpdatedAt == nil || *got.UpdatedAt != "2026-10-06T23:00:00Z" || got.Repo == nil || *got.Repo != "hero-engine/hero-harness" {
		t.Errorf("fields = updated %v repo %v", got.UpdatedAt, got.Repo)
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
		data, _ := os.ReadFile(path)
		out[rel] = info.ModTime().String() + ":" + string(data)
		return nil
	})
	return out
}

func callWork(t *testing.T, srv *MCPServer, args map[string]interface{}) HeroWork {
	t.Helper()
	result := callTool(t, srv, "hero_work", args)
	if result.IsError {
		t.Fatalf("hero_work error: %s", result.Content[0].Text)
	}
	var w HeroWork
	if err := json.Unmarshal([]byte(result.Content[0].Text), &w); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	return w
}

// hero-work-tool AC-1: the HeroWork shape.
func TestToolWorkShape(t *testing.T) {
	srv, _ := readContractWorkspace(t)
	result := callTool(t, srv, "hero_work", map[string]interface{}{})
	raw := result.Content[0].Text
	w := callWork(t, srv, map[string]interface{}{})
	if w.SchemaVersion != 1 || len(w.Revision) != 16 || w.GeneratedAt == "" || w.HeroVersion != "1.0.0" {
		t.Errorf("header = %+v", w)
	}
	if strings.Join(w.WatchGlobs, ",") != strings.Join(WatchGlobs, ",") {
		t.Errorf("watch_globs = %v", w.WatchGlobs)
	}
	slugs := map[string]string{}
	for _, it := range w.Items {
		slugs[it.Slug] = it.Lane
		if it.Slug == "feat" && (it.Next == nil || it.Next.Command != "/deliver feat") {
			t.Errorf("feat next = %+v", it.Next)
		}
	}
	if _, ok := slugs["naming"]; ok {
		t.Error("knowledge spec listed as work")
	}
	if slugs["feat"] != "designed" || slugs["base"] != "in_progress" || slugs["init"] != "ready" {
		t.Errorf("lanes = %v", slugs)
	}
	if w.Polish == nil || w.Suggested == nil || !strings.Contains(raw, `"polish":[`) || !strings.Contains(raw, `"suggested":[`) {
		t.Errorf("polish/suggested must be lists: %s", raw)
	}
	if len(w.Suggested) == 0 || w.Suggested[len(w.Suggested)-1].Next == nil || w.Suggested[len(w.Suggested)-1].Next.Command != "/discover" {
		t.Errorf("thin backlog should end suggested with Explore: %+v", w.Suggested)
	}
}

// hero-work-tool AC-2: revision ignores generated_at and tracks content.
func TestToolWorkRevision(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	orig := readContractNow
	defer func() { readContractNow = orig }()
	first := callWork(t, srv, map[string]interface{}{})
	readContractNow = func() time.Time { return time.Now().Add(time.Hour) }
	second := callWork(t, srv, map[string]interface{}{})
	if first.Revision != second.Revision || first.GeneratedAt == second.GeneratedAt {
		t.Fatalf("revision moved with time alone: %s vs %s", first.Revision, second.Revision)
	}
	writeContractSpec(t, heroDir, "planning/features/base", "---\ntitle: Base\nslug: base\ntype: feature\nstatus: completed\n---\n# Base\n")
	third := callWork(t, srv, map[string]interface{}{})
	if third.Revision == second.Revision {
		t.Error("revision unchanged after a spec changed")
	}
}

// hero-work-tool AC-3: recent_days is validated.
func TestToolWorkRecentDays(t *testing.T) {
	srv, _ := readContractWorkspace(t)
	for _, bad := range []interface{}{0, -1, 1.5, "7"} {
		result := callTool(t, srv, "hero_work", map[string]interface{}{"recent_days": bad})
		if !result.IsError || !strings.Contains(result.Content[0].Text, "recent_days") {
			t.Errorf("recent_days=%v accepted", bad)
		}
	}
	callWork(t, srv, map[string]interface{}{"recent_days": 30})
}

// hero-work-tool AC-4: read-only, writes nothing.
func TestToolWorkWritesNothing(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	// Seed every watched glob so a modification (not only a creation) shows.
	for rel, content := range map[string]string{
		"NEXT.md": "# Next\n", "hero.json": "{}\n", "next/chet.md": "# Chet\n",
		"specs/done/spec.md": "---\ntitle: Done\nslug: done\ntype: feature\nstatus: completed\n---\n# Done\n",
	} {
		path := filepath.Join(heroDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	watched := func(path string) bool {
		if path == "NEXT.md" || path == "hero.json" {
			return true
		}
		for _, dir := range []string{"planning", "specs", "knowledge", "next"} {
			if strings.HasPrefix(path, dir+string(filepath.Separator)) {
				return true
			}
		}
		return false
	}
	before := snapshotTree(t, heroDir)
	callWork(t, srv, map[string]interface{}{})
	after := snapshotTree(t, heroDir)
	for path, mod := range after {
		if watched(path) && before[path] != mod {
			t.Errorf("hero_work wrote %s (under watch_globs)", path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok && watched(path) {
			t.Errorf("hero_work deleted %s (under watch_globs)", path)
		}
	}
	ann := annotationsForSafety(toolSafetyClasses()["hero_work"])
	if ann.ReadOnlyHint == nil || !*ann.ReadOnlyHint {
		t.Error("hero_work must be readOnlyHint=true")
	}
}

// Pre-release sweep (audit test gaps): CRLF frontmatter, a knowledge entry
// pointing at an initiative (not a child), recorded pass/fail AC states,
// and an initiative's verify null at the tool level.
func TestReadContractSweepGaps(t *testing.T) {
	srv, heroDir := readContractWorkspace(t)
	crlf := strings.ReplaceAll("---\ntitle: Crlf\nslug: crlf\ntype: feature\nstatus: planning\nparent: init\n---\n\n# Crlf\n\nBody here.\n\n## Changes\n\n1. x\n\n## Acceptance Criteria\n\n- **AC-1:** THE SYSTEM SHALL one.\n- **AC-2:** THE SYSTEM SHALL two.\n", "\n", "\r\n")
	writeContractSpec(t, heroDir, "planning/features/crlf", crlf)
	writeContractSpec(t, heroDir, "knowledge/notes/aside", "---\ntitle: Aside\nslug: aside\ntype: note\nstatus: active\nparent: init\n---\n# Aside\n")

	store, err := graph.Open(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	for ac, status := range map[string]string{"AC-1": "passing", "AC-2": "failing"} {
		if _, err := store.UpsertNode(&graph.Node{Type: "Criterion", Domain: "engineering", Key: "crlf:" + ac,
			Props: map[string]any{"ac_id": ac, "status": status, "parent": "crlf"}, ContentHash: ac + status}); err != nil {
			t.Fatal(err)
		}
	}
	store.Close()

	var got HeroSpec
	result := callTool(t, srv, "hero_spec", map[string]interface{}{"slug": "crlf"})
	if result.IsError {
		t.Fatal(result.Content[0].Text)
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &got); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got.Body, "---") || strings.Contains(got.Body, "slug: crlf") || !strings.Contains(got.Body, "Body here.") {
		t.Errorf("CRLF frontmatter not stripped: %q", got.Body)
	}
	states := map[string]string{}
	for _, ac := range got.ACs {
		states[ac.ID] = ac.State
	}
	if states["AC-1"] != "pass" || states["AC-2"] != "fail" {
		t.Errorf("recorded AC states = %v, want AC-1 pass, AC-2 fail", states)
	}

	var init HeroSpec
	result = callTool(t, srv, "hero_spec", map[string]interface{}{"slug": "init"})
	if result.IsError {
		t.Fatal(result.Content[0].Text)
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &init); err != nil {
		t.Fatal(err)
	}
	foundChild := false
	for _, ch := range init.Relations.Children {
		if ch.Slug == "aside" {
			t.Error("a knowledge note pointing at an initiative must not be a child")
		}
		foundChild = foundChild || ch.Slug == "crlf"
	}
	if !foundChild {
		t.Errorf("crlf (parent: init) missing from init's children: %+v", init.Relations.Children)
	}
	if init.Item.Verify != nil {
		t.Errorf("initiative verify must be null at the tool level, got %+v", init.Item.Verify)
	}
}
