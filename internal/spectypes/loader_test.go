package spectypes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoad_CoreFile_NineCanonicalTypes verifies the loader picks up
// every core spec-type file shipped under core/spec-types/.
func TestLoad_CoreFile_NineCanonicalTypes(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	wantCore := []string{
		"initiative", "prd", "epic", "feature", "bug",
		"chore", "intake", "release", "sprint",
	}
	for _, name := range wantCore {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("missing core type %q", name)
			continue
		}
		if rec.Domain != "core" {
			t.Errorf("type %q: want Domain=core, got %q", name, rec.Domain)
		}
		if rec.Category != CategoryWork {
			t.Errorf("type %q: want Category=work, got %q", name, rec.Category)
		}
	}
}

// TestLoad_EngineeringOverlay_DecisionAndConvention verifies the
// engineering domain overlay adds decision + convention.
func TestLoad_EngineeringOverlay_DecisionAndConvention(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, name := range []string{"decision", "convention"} {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("missing engineering type %q", name)
			continue
		}
		if rec.Domain != "engineering" {
			t.Errorf("type %q: want Domain=engineering, got %q", name, rec.Domain)
		}
		if rec.Category != CategoryKnowledge {
			t.Errorf("type %q: want Category=knowledge, got %q", name, rec.Category)
		}
	}
}

// TestLoad_KindBlock_FeatureBugIntakePRDEpic verifies the kind enum
// is parsed for the types that declare one.
func TestLoad_KindBlock(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cases := map[string][]string{
		"feature": {"new", "refactor", "perf", "infra", "security", "ux"},
		"bug":     {"regression", "edge-case", "security", "data"},
		"intake":  {"customer", "support", "sales", "internal", "competitive"},
		"prd":     {"pitch", "ten-section", "lightweight"},
		"epic":    {"theme", "delivery", "bet", "milestone"},
	}
	for name, wantKinds := range cases {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Fatalf("missing type %q", name)
		}
		if len(rec.Kind.Values) != len(wantKinds) {
			t.Errorf("type %q: want %d kinds, got %d (%v)", name, len(wantKinds), len(rec.Kind.Values), rec.Kind.Values)
			continue
		}
		for i, want := range wantKinds {
			if rec.Kind.Values[i] != want {
				t.Errorf("type %q: kind[%d]=%q, want %q", name, i, rec.Kind.Values[i], want)
			}
		}
	}
}

// TestLoad_OwnerSchema_Engineering verifies the canonical owner
// values appear on work types.
func TestLoad_OwnerSchema(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rec, ok := reg.Lookup("feature")
	if !ok {
		t.Fatal("missing feature")
	}
	wantValues := []string{"pm", "engineering", "qa", "devops", "design", "docs"}
	if len(rec.Owner.Values) != len(wantValues) {
		t.Errorf("owner.values: got %v, want %v", rec.Owner.Values, wantValues)
	}
	if rec.Owner.Default != "engineering" {
		t.Errorf("owner.default: got %q, want %q", rec.Owner.Default, "engineering")
	}
	if rec.Owner.Classification != ClassificationOrgState {
		t.Errorf("owner.classification: got %q, want %q", rec.Owner.Classification, ClassificationOrgState)
	}
}

// TestLoad_TasksSchema_FeatureBug verifies the tasks_schema parses
// with the canonical item shape and bitemporal history mode.
func TestLoad_TasksSchema(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rec, ok := reg.Lookup("feature")
	if !ok {
		t.Fatal("missing feature")
	}
	if rec.TasksSchema.SectionHeading != "Tasks" {
		t.Errorf("tasks_schema.section_heading: got %q, want %q", rec.TasksSchema.SectionHeading, "Tasks")
	}
	if rec.TasksSchema.History != HistoryBitemporal {
		t.Errorf("tasks_schema.history: got %q, want %q", rec.TasksSchema.History, HistoryBitemporal)
	}
	for _, want := range []string{"id", "text", "status", "kind", "assignee", "discovered_against", "started", "done"} {
		if _, ok := rec.TasksSchema.ItemShape[want]; !ok {
			t.Errorf("tasks_schema.item_shape missing field %q", want)
		}
	}
}

// TestLoad_OwnerFlipAnnotation verifies the ready→delivering owner_flip
// is parsed correctly on feature and bug.
func TestLoad_OwnerFlipAnnotation(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, name := range []string{"feature", "bug"} {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Fatalf("missing %q", name)
		}
		var found bool
		for _, tr := range rec.Lifecycle.Transitions {
			if tr.From == "ready" && tr.To == "delivering" {
				if tr.OwnerFlip == nil {
					t.Errorf("%q: ready→delivering transition missing owner_flip", name)
				} else if tr.OwnerFlip.To != "engineering" {
					t.Errorf("%q: owner_flip.to = %q, want engineering", name, tr.OwnerFlip.To)
				}
				found = true
			}
		}
		if !found {
			t.Errorf("%q: no ready→delivering transition found", name)
		}
	}
}

// TestRegistry_Accessors covers the basic Registry surface.
func TestRegistry_Accessors(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	all := reg.All()
	if len(all) < 9 {
		t.Errorf("All(): got %d records, want >= 9", len(all))
	}

	if _, ok := reg.LookupFolder("features"); !ok {
		t.Error("LookupFolder(features): not found")
	}

	if rec, ok := reg.LookupByKind("feature", "perf"); !ok || rec.Name != "feature" {
		t.Error("LookupByKind(feature,perf): should resolve to feature")
	}
	if _, ok := reg.LookupByKind("feature", "nonsense"); ok {
		t.Error("LookupByKind(feature,nonsense): should not resolve")
	}

	if !contains(reg.Kinds("feature"), "perf") {
		t.Error("Kinds(feature): missing perf")
	}

	wt := reg.WorkTypes()
	if len(wt) < 9 {
		t.Errorf("WorkTypes(): got %d, want >= 9", len(wt))
	}
	kt := reg.KnowledgeTypes()
	if len(kt) < 2 {
		t.Errorf("KnowledgeTypes(): got %d, want >= 2", len(kt))
	}

	if reg.DefaultWorkType() == nil || reg.DefaultWorkType().Name != "feature" {
		t.Errorf("DefaultWorkType(): expected feature, got %+v", reg.DefaultWorkType())
	}

	if reg.ActiveDomain() != "engineering" {
		t.Errorf("ActiveDomain() = %q, want engineering", reg.ActiveDomain())
	}

	delivering := reg.AcceptingCommand("/deliver")
	if len(delivering) == 0 {
		t.Error("AcceptingCommand(/deliver): expected results")
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// TestJSONSchema_SchemaVersion verifies the JSON export carries the
// 1.1 schema version and contains all loaded types.
func TestJSONSchema_SchemaVersion(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	raw, err := reg.JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if parsed["schema_version"] != SchemaVersion {
		t.Errorf("schema_version: got %v, want %s", parsed["schema_version"], SchemaVersion)
	}
	if parsed["active_domain"] != "engineering" {
		t.Errorf("active_domain: got %v, want engineering", parsed["active_domain"])
	}
	types, ok := parsed["types"].([]any)
	if !ok || len(types) < 9 {
		t.Errorf("types: got %d, want >= 9", len(types))
	}
	// Ensure the JSON does NOT carry any "display" block (vocabulary
	// is separate per the spec's cross-language contract).
	if strings.Contains(string(raw), `"display"`) {
		t.Error("JSON export must not include a display block")
	}
}

// TestExportTo_WritesCacheFile verifies ExportTo writes the manifest
// to .hero/cache/spec-types.json under the given workspace root.
func TestExportTo_WritesCacheFile(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	tmp := t.TempDir()
	if err := ExportTo(reg, tmp); err != nil {
		t.Fatalf("ExportTo: %v", err)
	}
	out := filepath.Join(tmp, ".hero", "cache", "spec-types.json")
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected cache file at %s: %v", out, err)
	}

	// Acceptance: at least one record in the exported cache carries a
	// non-null frontmatter block. Regression pin for the loader gap
	// where the parser populated FrontmatterSchema but no source file
	// declared a `frontmatter:` block.
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var parsed struct {
		Types []struct {
			Name        string `json:"name"`
			Frontmatter *struct {
				Required []map[string]any `json:"required,omitempty"`
				Optional []map[string]any `json:"optional,omitempty"`
			} `json:"frontmatter,omitempty"`
		} `json:"types"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal cache: %v", err)
	}
	var populated int
	for _, ty := range parsed.Types {
		if ty.Frontmatter != nil && (len(ty.Frontmatter.Required) > 0 || len(ty.Frontmatter.Optional) > 0) {
			populated++
		}
	}
	if populated == 0 {
		t.Error("exported cache has zero records with a non-null frontmatter block; loader gap regression")
	}
}

// TestExportTo_SkipsWriteWhenOnlyTimestampChanged pins the idempotency
// guard: re-exporting an unchanged registry must not rewrite the cache
// file when the only difference is the generated_at timestamp. Without
// this, the PersistentPreRun export re-stamps the file on every `hero`
// invocation, leaving .hero/cache/spec-types.json perpetually dirty and
// racing the pre-commit hook that stages it.
func TestExportTo_SkipsWriteWhenOnlyTimestampChanged(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	tmp := t.TempDir()
	if err := ExportTo(reg, tmp); err != nil {
		t.Fatalf("ExportTo (first): %v", err)
	}
	out := filepath.Join(tmp, ".hero", "cache", "spec-types.json")
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat after first export: %v", err)
	}
	firstMod := fi.ModTime()

	// Force a detectable mtime gap, then re-export the same registry.
	// The only thing that would differ is generated_at, so the guard
	// must skip the write and leave the file (and its mtime) untouched.
	past := firstMod.Add(-2 * time.Second)
	if err := os.Chtimes(out, past, past); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	if err := ExportTo(reg, tmp); err != nil {
		t.Fatalf("ExportTo (second): %v", err)
	}
	fi2, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat after second export: %v", err)
	}
	if !fi2.ModTime().Equal(past) {
		t.Errorf("cache file was rewritten on timestamp-only re-export: mtime changed from %v to %v", past, fi2.ModTime())
	}
}

// TestLoad_FrontmatterSchema_PopulatedForCoreAndEngineering pins the
// canonical work types and engineering knowledge types to a non-empty
// frontmatter schema. Surfaces the spec-types-cache-frontmatter-empty
// bug if any of these types regress to a null/empty schema.
func TestLoad_FrontmatterSchema_PopulatedForCoreAndEngineering(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// All nine core work types ship a frontmatter schema. `initiative`
	// is owned by a parallel fix on the same file — verify it lands by
	// asserting non-empty here once that work merges; for now this fix
	// covers the other eight.
	requirePopulated := []string{
		"feature", "bug", "chore", "epic", "intake",
		"prd", "release", "sprint",
		// engineering domain overlay
		"convention", "decision",
	}
	for _, name := range requirePopulated {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("missing type %q", name)
			continue
		}
		if len(rec.Frontmatter.Required) == 0 && len(rec.Frontmatter.Optional) == 0 {
			t.Errorf("type %q: FrontmatterSchema is empty (required=%d, optional=%d)",
				name, len(rec.Frontmatter.Required), len(rec.Frontmatter.Optional))
			continue
		}
		// Every populated record must declare `title`, `type`, `status`
		// as required fields — the minimum contract per spec.
		gotRequired := map[string]bool{}
		for _, f := range rec.Frontmatter.Required {
			gotRequired[f.Name] = true
		}
		for _, want := range []string{"title", "type", "status"} {
			if !gotRequired[want] {
				t.Errorf("type %q: required field %q missing from frontmatter schema", name, want)
			}
		}
	}
}

func TestLoad_PMIncludesOwnedTypesWithoutShadowingCore(t *testing.T) {
	reg, err := Load("pm")
	if err != nil {
		t.Fatalf("Load(pm): %v", err)
	}
	for _, name := range []string{"roadmap-item"} {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("PM registry missing %q", name)
			continue
		}
		if rec.Domain != "pm" {
			t.Errorf("%s domain = %q, want pm", name, rec.Domain)
		}
	}
	for _, name := range []string{"feature", "epic", "intake", "prd"} {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("PM registry missing shared Core type %q", name)
			continue
		}
		if rec.Domain != "core" {
			t.Errorf("shared type %s domain = %q, want core", name, rec.Domain)
		}
	}
}

func TestLoad_QAIncludesOwnedTypesWithoutShadowingCore(t *testing.T) {
	reg, err := Load("qa")
	if err != nil {
		t.Fatalf("Load(qa): %v", err)
	}
	for _, name := range []string{"test-plan", "test-case", "regression-suite", "release-gate", "defect"} {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("QA registry missing %q", name)
			continue
		}
		if rec.Domain != "qa" {
			t.Errorf("%s domain = %q, want qa", name, rec.Domain)
		}
		if len(rec.Frontmatter.Required) == 0 {
			t.Errorf("%s has no required frontmatter schema", name)
		}
	}
	for _, name := range []string{"feature", "bug", "release", "intake", "prd"} {
		rec, ok := reg.Lookup(name)
		if !ok {
			t.Errorf("QA registry missing shared Core type %q", name)
			continue
		}
		if rec.Domain != "core" {
			t.Errorf("shared type %s domain = %q, want core", name, rec.Domain)
		}
	}
}

// TestLoad_FrontmatterFieldShape_FeatureStatus pins one representative
// field's full shape so changes to FieldDecl serialization are caught.
func TestLoad_FrontmatterFieldShape_FeatureStatus(t *testing.T) {
	reg, err := Load("engineering")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rec, ok := reg.Lookup("feature")
	if !ok {
		t.Fatal("missing feature")
	}
	var status FieldDecl
	for _, f := range rec.Frontmatter.Required {
		if f.Name == "status" {
			status = f
			break
		}
	}
	if status.Name == "" {
		t.Fatal("feature.frontmatter.required.status not declared")
	}
	if status.Type != "enum" {
		t.Errorf("status.type = %q, want enum", status.Type)
	}
	if !status.Required {
		t.Error("status.required should be true")
	}
	if len(status.Values) == 0 {
		t.Error("status.values should enumerate lifecycle states")
	}
	if status.Classification != ClassificationOrgState {
		t.Errorf("status.classification = %q, want org-state", status.Classification)
	}
}

// TestLoad_SalesOverlay_DealLifecycle verifies the sales domain overlay
// registers the `deal` work type with its full 7-state lifecycle and clean
// referential integrity. Guards the deal.yaml → deal.md conversion
// (sales-pack-reality-sync): a malformed deal.md would break every command
// in a sales workspace via exportSpecTypesCache.
func TestLoad_SalesOverlay_DealLifecycle(t *testing.T) {
	reg, err := Load("sales")
	if err != nil {
		t.Fatalf("Load(sales): %v", err)
	}
	rec, ok := reg.Lookup("deal")
	if !ok {
		t.Fatal(`sales overlay did not register type "deal"`)
	}
	if rec.Domain != "sales" {
		t.Errorf("deal.Domain = %q, want sales", rec.Domain)
	}
	if rec.Category != CategoryWork {
		t.Errorf("deal.Category = %q, want work", rec.Category)
	}
	gotStates := strings.Join(rec.Lifecycle.States, ",")
	wantStates := "prospect,qualifying,demo,proposal,negotiation,won,lost"
	if gotStates != wantStates {
		t.Errorf("deal lifecycle states = %q, want %q", gotStates, wantStates)
	}
	if rec.Lifecycle.Initial != "prospect" {
		t.Errorf("deal lifecycle initial = %q, want prospect", rec.Lifecycle.Initial)
	}
	if got := strings.Join(rec.Lifecycle.Terminal, ","); got != "won,lost" {
		t.Errorf("deal lifecycle terminal = %q, want %q", got, "won,lost")
	}
}
