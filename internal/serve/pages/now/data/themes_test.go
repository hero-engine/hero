package data

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTaggedSpec writes a spec with explicit tags so the cluster
// detector has tag co-occurrence to pick up. tags is a flat YAML
// inline list, e.g. `[serve, dashboard]`.
func writeTaggedSpec(t *testing.T, heroDir, slug, typ, status, tags string) string {
	t.Helper()
	dir := filepath.Join(heroDir, "planning", typ+"s", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "---\ntitle: " + slug + "\ntype: " + typ + "\nstatus: " + status + "\n"
	if tags != "" {
		body += "tags: " + tags + "\n"
	}
	body += "---\n\nbody.\n"
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadThemes_OmittedBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	// Two specs share a tag — below the 3-item threshold → no
	// cluster surfaces. The section must render nothing, not an
	// empty header.
	writeTaggedSpec(t, dir, "a", "feature", "delivering", "[serve]")
	writeTaggedSpec(t, dir, "b", "feature", "delivering", "[serve]")
	got := LoadThemes(ThemesInputs{HeroDir: dir, Window: WindowAll})
	if len(got.Clusters) != 0 {
		t.Errorf("expected no clusters below threshold, got %d: %+v", len(got.Clusters), got.Clusters)
	}
}

func TestLoadThemes_SurfacesAtThreshold(t *testing.T) {
	dir := t.TempDir()
	writeTaggedSpec(t, dir, "a", "feature", "delivering", "[serve]")
	writeTaggedSpec(t, dir, "b", "feature", "delivering", "[serve]")
	writeTaggedSpec(t, dir, "c", "feature", "delivering", "[serve]")
	got := LoadThemes(ThemesInputs{HeroDir: dir, Window: WindowAll})
	if len(got.Clusters) != 1 {
		t.Fatalf("expected 1 cluster at threshold, got %d", len(got.Clusters))
	}
	if got.Clusters[0].Label != "serve" {
		t.Errorf("expected label=serve, got %q", got.Clusters[0].Label)
	}
	if got.Clusters[0].ItemCount != 3 {
		t.Errorf("expected ItemCount=3, got %d", got.Clusters[0].ItemCount)
	}
}

func TestLoadThemes_SkipsNotesAndContext(t *testing.T) {
	dir := t.TempDir()
	writeTaggedSpec(t, dir, "a", "note", "planning", "[serve]")
	writeTaggedSpec(t, dir, "b", "note", "planning", "[serve]")
	writeTaggedSpec(t, dir, "c", "note", "planning", "[serve]")
	got := LoadThemes(ThemesInputs{HeroDir: dir, Window: WindowAll})
	if len(got.Clusters) != 0 {
		t.Errorf("notes should be filtered out of work-cluster signal, got %+v", got.Clusters)
	}
}

func TestLoadThemes_AggregateAcrossProjects(t *testing.T) {
	d1 := t.TempDir()
	d2 := t.TempDir()
	writeTaggedSpec(t, d1, "alpha-a", "feature", "delivering", "[ui]")
	writeTaggedSpec(t, d1, "alpha-b", "feature", "delivering", "[ui]")
	writeTaggedSpec(t, d2, "beta-c", "feature", "delivering", "[ui]")
	got := LoadThemes(ThemesInputs{
		Aggregate: []ActivityProject{
			{Slug: "alpha", HeroDir: d1},
			{Slug: "beta", HeroDir: d2},
		},
		Window: WindowAll,
	})
	if !got.Aggregate {
		t.Errorf("expected Aggregate=true")
	}
	if len(got.Clusters) != 1 {
		t.Fatalf("expected 1 cross-project cluster, got %d", len(got.Clusters))
	}
	if got.Clusters[0].ItemCount != 3 {
		t.Errorf("expected ItemCount=3, got %d", got.Clusters[0].ItemCount)
	}
	// At least one item must be project-tagged.
	hasProj := false
	for _, it := range got.Clusters[0].Items {
		if it.Project != "" {
			hasProj = true
			break
		}
	}
	if !hasProj {
		t.Errorf("expected items to carry project slugs in aggregate mode")
	}
}

func TestLoadThemes_EmptyHeroDir(t *testing.T) {
	got := LoadThemes(ThemesInputs{HeroDir: ""})
	if len(got.Clusters) != 0 {
		t.Errorf("expected empty themes for empty HeroDir, got %+v", got.Clusters)
	}
}
