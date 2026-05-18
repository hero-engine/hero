package snapshot

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOverride_Empty(t *testing.T) {
	o, err := parseOverride("")
	if err != nil {
		t.Fatalf("parseOverride empty: %v", err)
	}
	if o.Version != 0 {
		t.Errorf("expected zero version, got %d", o.Version)
	}
}

func TestParseOverride_AllSections(t *testing.T) {
	src := `version: 1
renames:
  - from: domains-engineering
    to: domains/engineering
ignore:
  - id: scratch
additions:
  - id: serve-companion
    name: Web Companion
    intent: Companion shell embedded in hero serve.
    stage: shipping-v1
    owner: alice
overrides:
  - id: serve
    stage: shipping-v1
    owner: bob
`
	o, err := parseOverride(src)
	if err != nil {
		t.Fatalf("parseOverride: %v", err)
	}
	if o.Version != 1 {
		t.Errorf("version = %d, want 1", o.Version)
	}
	if len(o.Renames) != 1 || o.Renames[0].From != "domains-engineering" || o.Renames[0].To != "domains/engineering" {
		t.Errorf("renames = %+v", o.Renames)
	}
	if len(o.Ignore) != 1 || o.Ignore[0].ID != "scratch" {
		t.Errorf("ignore = %+v", o.Ignore)
	}
	if len(o.Additions) != 1 {
		t.Fatalf("additions count = %d, want 1; got %+v", len(o.Additions), o.Additions)
	}
	add := o.Additions[0]
	if add.ID != "serve-companion" || add.Stage != "shipping-v1" || add.Owner != "alice" {
		t.Errorf("addition = %+v", add)
	}
	if len(o.Overrides) != 1 || o.Overrides[0].ID != "serve" {
		t.Errorf("overrides = %+v", o.Overrides)
	}
}

func TestMerge_InferredOnly(t *testing.T) {
	detected := []CandidateSurface{
		{ID: "core", Name: "Core", Paths: []string{"cmd/"}, Confidence: 1.0, Signals: []string{"go.mod"}},
		{ID: "serve", Name: "Serve", Paths: []string{"internal/serve/"}, Confidence: 0.9, Signals: []string{"serve dir"}},
	}
	got := Merge(detected, SurfacesOverride{})
	if len(got) != 2 {
		t.Fatalf("merge len = %d, want 2", len(got))
	}
	if got[0].Source != "inferred" {
		t.Errorf("expected source=inferred, got %q", got[0].Source)
	}
}

func TestMerge_IgnoreRemoves(t *testing.T) {
	detected := []CandidateSurface{
		{ID: "core"},
		{ID: "scratch"},
	}
	override := SurfacesOverride{Ignore: []IgnoreOverride{{ID: "scratch"}}}
	got := Merge(detected, override)
	if len(got) != 1 || got[0].ID != "core" {
		t.Errorf("ignore failed: %+v", got)
	}
}

func TestMerge_Rename(t *testing.T) {
	detected := []CandidateSurface{
		{ID: "domains-engineering", Paths: []string{"domains/engineering/"}},
	}
	override := SurfacesOverride{
		Renames: []RenameOverride{{From: "domains-engineering", To: "domains/engineering"}},
	}
	got := Merge(detected, override)
	if len(got) != 1 || got[0].ID != "domains/engineering" {
		t.Errorf("rename failed: %+v", got)
	}
}

func TestMerge_FieldOverride(t *testing.T) {
	detected := []CandidateSurface{
		{ID: "serve", Confidence: 0.9},
	}
	override := SurfacesOverride{
		Overrides: []SurfaceFieldOverride{{
			ID:    "serve",
			Stage: "shipping-v1",
			Owner: "bob",
		}},
	}
	got := Merge(detected, override)
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	s := got[0]
	if s.Stage != StageShippingV1 {
		t.Errorf("stage = %q, want shipping-v1", s.Stage)
	}
	if !s.StagePinned {
		t.Errorf("expected stage pinned")
	}
	if s.Owner != "bob" {
		t.Errorf("owner = %q, want bob", s.Owner)
	}
	if s.Source != "override" {
		t.Errorf("source = %q, want override", s.Source)
	}
}

func TestMerge_Addition(t *testing.T) {
	override := SurfacesOverride{
		Additions: []SurfaceAddition{{
			ID:    "serve-companion",
			Name:  "Web Companion",
			Stage: "building",
		}},
	}
	got := Merge(nil, override)
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Source != "added" {
		t.Errorf("source = %q, want added", got[0].Source)
	}
	if got[0].Stage != StageBuilding {
		t.Errorf("stage = %q", got[0].Stage)
	}
}

func TestScanRepo_OnHeroRepo(t *testing.T) {
	// Walk up from cwd to find the hero repo root (it has hero.json).
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	rs, err := ScanRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs.Dirs) == 0 {
		t.Error("expected at least one top-level dir")
	}
	// "internal" must be present.
	found := false
	for _, d := range rs.Dirs {
		if d == "internal" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'internal' in dirs, got %v", rs.Dirs)
	}
}

func TestDetect_OnHeroRepo(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	rs, err := ScanRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	cands := Detect(rs)
	if len(cands) == 0 {
		t.Fatal("expected detected surfaces on hero repo")
	}
	want := []string{"core", "serve"}
	for _, w := range want {
		found := false
		for _, c := range cands {
			if c.ID == w {
				found = true
				break
			}
		}
		if !found {
			ids := []string{}
			for _, c := range cands {
				ids = append(ids, c.ID)
			}
			t.Errorf("missing surface %q; got %v", w, strings.Join(ids, ", "))
		}
	}
}
