package serve

import (
	"encoding/json"
	"testing"
)

// TestToolMetadataDriftGuard is the load-bearing anti-drift guard (AC-4, AC-5,
// AC-8): every advertised tool must declare an in-enum category, a valid tier,
// and end with MCP annotations, and the emitted _meta must carry both facets
// under their namespaced keys. A tool added without a category/tier fails here,
// naming that tool.
func TestToolMetadataDriftGuard(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	definitions := server.toolDefinitions()
	if len(definitions) == 0 {
		t.Fatal("no tool definitions advertised")
	}

	usedCategories := make(map[ToolCategory]bool)
	for _, definition := range definitions {
		if !ToolCategoryValid(definition.Category) {
			t.Errorf("tool %q has invalid or missing category %q (not in the closed taxonomy)", definition.Name, definition.Category)
		}
		usedCategories[definition.Category] = true

		if !ToolTierValid(definition.Tier) {
			t.Errorf("tool %q has invalid or missing tier %q (want %q or %q)", definition.Name, definition.Tier, TierEager, TierDeferrable)
		}

		if definition.Annotations == nil {
			t.Errorf("tool %q has no MCP annotations (safety class not backfilled)", definition.Name)
		}

		if definition.Meta == nil {
			t.Errorf("tool %q has no _meta", definition.Name)
			continue
		}
		if got := definition.Meta[MetaKeyCategory]; got != string(definition.Category) {
			t.Errorf("tool %q _meta[%s] = %v, want %q", definition.Name, MetaKeyCategory, got, definition.Category)
		}
		if got := definition.Meta[MetaKeyTier]; got != string(definition.Tier) {
			t.Errorf("tool %q _meta[%s] = %v, want %q", definition.Name, MetaKeyTier, got, definition.Tier)
		}
	}

	// No documented-but-unused drift: every category const must be carried by
	// at least one live tool, and no emitted category is outside the taxonomy
	// (already checked per-tool above).
	for category := range toolCategorySet {
		if !usedCategories[category] {
			t.Errorf("documented category %q is unused by any tool (dead taxonomy entry)", category)
		}
	}
}

// TestToolEagerSetIsSmallAndDefensible pins the advisory eager set. Marking the
// wrong tool eager wastes a harness's budget, so the set is deliberately the
// tight session-warmup loop. Changing it should be a loud, reviewed edit.
func TestToolEagerSetIsSmallAndDefensible(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	want := map[string]bool{
		"hero_context": true,
		"hero_anchor":  true,
		"hero_search":  true,
		"hero_status":  true,
		"hero_list":    true,
		"hero_queue":   true,
	}
	got := make(map[string]bool)
	for _, definition := range server.toolDefinitions() {
		if definition.Tier == TierEager {
			got[definition.Name] = true
		}
	}
	if len(got) != len(want) {
		t.Fatalf("eager set = %v, want %v", keys(got), keys(want))
	}
	for name := range want {
		if !got[name] {
			t.Errorf("expected %q to be eager", name)
		}
	}
}

// TestToolMetadataIsAdditiveAndBackwardCompatible verifies AC-6: category and
// tier live only in _meta, never as top-level wire fields, so a client that
// ignores _meta still sees a valid, callable tool.
func TestToolMetadataIsAdditiveAndBackwardCompatible(t *testing.T) {
	server := NewMCPServer(t.TempDir(), t.TempDir(), "test")
	var sample *ToolDefinition
	for index, definition := range server.toolDefinitions() {
		if definition.Name == "hero_search" {
			sample = &server.toolDefinitions()[index]
			break
		}
	}
	if sample == nil {
		t.Fatal("hero_search not advertised")
	}

	encoded, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// A client that ignores _meta still sees a valid, callable tool.
	if _, ok := wire["name"]; !ok {
		t.Error("wire tool missing name")
	}
	if _, ok := wire["inputSchema"]; !ok {
		t.Error("wire tool missing inputSchema")
	}

	// The facets must NOT appear as top-level keys.
	if _, ok := wire["category"]; ok {
		t.Error("category leaked as a top-level wire field; must live only in _meta")
	}
	if _, ok := wire["tier"]; ok {
		t.Error("tier leaked as a top-level wire field; must live only in _meta")
	}

	// They must appear inside _meta under the namespaced keys.
	metaRaw, ok := wire["_meta"]
	if !ok {
		t.Fatal("wire tool missing _meta")
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		t.Fatalf("unmarshal _meta: %v", err)
	}
	if meta[MetaKeyCategory] != string(CategorySearchAndKnowledge) {
		t.Errorf("_meta[%s] = %v, want %q", MetaKeyCategory, meta[MetaKeyCategory], CategorySearchAndKnowledge)
	}
	if meta[MetaKeyTier] != string(TierEager) {
		t.Errorf("_meta[%s] = %v, want %q", MetaKeyTier, meta[MetaKeyTier], TierEager)
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestConditionalWritersAreNotReadOnly pins the tools whose primary action
// reads but which write to disk on some inputs. Their MCP annotations must
// advertise readOnlyHint=false, or a conforming harness could auto-invoke a
// state-writer believing it safe. A cold audit found three of these still
// misclassified as read-only; this test makes a regression loud rather than
// silent. Each name is verified against its handler's write path in the
// toolSafetyClasses comment.
func TestConditionalWritersAreNotReadOnly(t *testing.T) {
	writers := []string{
		"hero_plan",
		"hero_error_pattern",
		"hero_contract",
		"hero_active",
		"hero_snapshot",
	}
	s := &MCPServer{}
	defs := map[string]ToolDefinition{}
	for _, d := range s.toolDefinitions() {
		defs[d.Name] = d
	}
	for _, name := range writers {
		d, ok := defs[name]
		if !ok {
			t.Errorf("%s: not advertised in tools/list (renamed or removed?)", name)
			continue
		}
		if d.Annotations == nil {
			t.Errorf("%s: no annotations — a writer must carry readOnlyHint=false", name)
			continue
		}
		if d.Annotations.ReadOnlyHint == nil || *d.Annotations.ReadOnlyHint {
			t.Errorf("%s: readOnlyHint is not false (it writes on some inputs); "+
				"a harness could auto-call it believing it safe", name)
		}
	}
}
