package wiki

import (
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

func TestNew_NoConfig(t *testing.T) {
	_, err := New(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestNew_NoneTarget(t *testing.T) {
	cfg := &config.SyncConfig{Target: "none"}
	_, err := New(cfg, nil)
	if err == nil {
		t.Fatal("expected error for 'none' target")
	}
}

func TestNew_UnknownTarget(t *testing.T) {
	cfg := &config.SyncConfig{Target: "totally-unknown"}
	_, err := New(cfg, nil)
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
	if !strings.Contains(err.Error(), "unknown wiki sync target") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNew_ConfluenceRequiresConfig(t *testing.T) {
	cfg := &config.SyncConfig{Target: "confluence"}
	_, err := New(cfg, nil)
	if err == nil {
		t.Fatal("expected error for confluence without config")
	}
	if !strings.Contains(err.Error(), "ConfluenceConfig") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNew_GitHubWikiNoProject(t *testing.T) {
	cfg := &config.SyncConfig{Target: "github-wiki"}
	_, err := New(cfg, nil)
	if err == nil {
		t.Fatal("expected error when tracker config is nil")
	}
	if !strings.Contains(err.Error(), "tracker.project") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNew_GitHubWikiNoToken(t *testing.T) {
	cfg := &config.SyncConfig{Target: "github-wiki"}
	tcfg := &config.TrackerConfig{Project: "owner/repo", TokenEnv: "HERO_TEST_NONEXISTENT_TOKEN_XYZ"}
	_, err := New(cfg, tcfg)
	if err == nil {
		t.Fatal("expected error when token env is not set")
	}
}

func TestNew_GitHubWikiSuccess(t *testing.T) {
	t.Setenv("HERO_TEST_WIKI_TOKEN", "test-token-123")

	cfg := &config.SyncConfig{Target: "github-wiki"}
	tcfg := &config.TrackerConfig{Project: "owner/repo", TokenEnv: "HERO_TEST_WIKI_TOKEN"}
	syncer, err := New(cfg, tcfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if syncer.Name() != "github-wiki" {
		t.Errorf("expected name github-wiki, got %s", syncer.Name())
	}
}

func TestNew_GitHubWikiBadProject(t *testing.T) {
	t.Setenv("HERO_TEST_WIKI_TOKEN", "test-token-123")

	cfg := &config.SyncConfig{Target: "github-wiki"}
	tcfg := &config.TrackerConfig{Project: "noslash", TokenEnv: "HERO_TEST_WIKI_TOKEN"}
	_, err := New(cfg, tcfg)
	if err == nil {
		t.Fatal("expected error for bad project format")
	}
	if !strings.Contains(err.Error(), "owner/repo") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSpecToPageName(t *testing.T) {
	tests := []struct {
		specType spec.Type
		slug     string
		want     string
	}{
		{spec.TypeFeature, "csv-export", "Feature-csv-export"},
		{spec.TypeBug, "login-timeout", "Bug-login-timeout"},
		{spec.TypeConvention, "error-format", "Convention-error-format"},
		{spec.TypeDecision, "use-postgres", "Decision-use-postgres"},
		{spec.TypeInitiative, "q3-auth", "Initiative-q3-auth"},
	}

	for _, tt := range tests {
		s := &spec.Spec{Type: tt.specType, Slug: tt.slug}
		got := specToPageName(s)
		if got != tt.want {
			t.Errorf("specToPageName(%s, %s) = %q, want %q", tt.specType, tt.slug, got, tt.want)
		}
	}
}

func TestSpecToWikiPage(t *testing.T) {
	s := &spec.Spec{
		Slug:      "csv-export",
		Title:     "CSV Export",
		Type:      spec.TypeFeature,
		Status:    spec.StatusCompleted,
		ClaimedBy: "alice",
		Tags:      []string{"export", "data"},
	}
	rawContent := `---
title: CSV Export
type: feature
status: completed
---
# CSV Export

## Goal

Export user data to CSV.
`

	page := specToWikiPage(s, rawContent)

	// Should have hero:managed marker
	if !strings.Contains(page, "<!-- hero:managed") {
		t.Error("missing hero:managed marker")
	}

	// Should have metadata line
	if !strings.Contains(page, "**Type:** feature") {
		t.Error("missing type in metadata")
	}
	if !strings.Contains(page, "**Status:** completed") {
		t.Error("missing status in metadata")
	}
	if !strings.Contains(page, "**Assigned:** alice") {
		t.Error("missing assigned in metadata")
	}
	if !strings.Contains(page, "**Tags:** export, data") {
		t.Error("missing tags in metadata")
	}

	// Should have body content without frontmatter
	if !strings.Contains(page, "# CSV Export") {
		t.Error("missing body content")
	}
	if !strings.Contains(page, "Export user data to CSV") {
		t.Error("missing goal content")
	}

	// Should NOT have frontmatter delimiters in body
	bodyStart := strings.Index(page, "---\n\n")
	bodyAfterHeader := page[bodyStart+5:]
	if strings.HasPrefix(strings.TrimSpace(bodyAfterHeader), "---\ntitle:") {
		t.Error("frontmatter was not stripped")
	}

	// Should have footer
	if !strings.Contains(page, "Synced by") {
		t.Error("missing footer")
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantHas string
		wantNot string
	}{
		{
			name:    "with frontmatter",
			input:   "---\ntitle: Foo\n---\n# Hello\n",
			wantHas: "# Hello",
			wantNot: "title: Foo",
		},
		{
			name:    "no frontmatter",
			input:   "# Hello\nWorld\n",
			wantHas: "# Hello",
		},
		{
			name:    "empty content",
			input:   "",
			wantHas: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFrontmatter(tt.input)
			if tt.wantHas != "" && !strings.Contains(got, tt.wantHas) {
				t.Errorf("result should contain %q, got %q", tt.wantHas, got)
			}
			if tt.wantNot != "" && strings.Contains(got, tt.wantNot) {
				t.Errorf("result should not contain %q, got %q", tt.wantNot, got)
			}
		})
	}
}

func TestSpecToWikiPage_NoOptionalFields(t *testing.T) {
	s := &spec.Spec{
		Slug:   "simple-spec",
		Title:  "Simple Spec",
		Type:   spec.TypeFeature,
		Status: spec.StatusPlanning,
		// No ClaimedBy, no Tags
	}
	rawContent := "# Simple Spec\n\nSome content.\n"

	page := specToWikiPage(s, rawContent)

	if strings.Contains(page, "**Assigned:**") {
		t.Error("should not include Assigned when empty")
	}
	if strings.Contains(page, "**Tags:**") {
		t.Error("should not include Tags when empty")
	}
}

// TestSyncerInterface verifies gitHubWiki satisfies the Syncer interface.
func TestSyncerInterface(t *testing.T) {
	_ = time.Now() // use time import
	var _ Syncer = &gitHubWiki{}
}
