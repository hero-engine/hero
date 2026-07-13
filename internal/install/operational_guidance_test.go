package install

import (
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/managed"
)

// TestOperationalGuidanceSection_Render asserts the shared, domain-agnostic
// operational-guidance section renders the verbatim v0.25.0 routing
// paragraph: prefer the in-process MCP surface, run `hero doctor` on
// schema/version confusion, and explicitly do NOT treat `hero upgrade` as a
// schema fix. It also pins the stable SectionID and the domain-neutral H2
// title the orchestrator wraps the body with.
func TestOperationalGuidanceSection_Render(t *testing.T) {
	sec := newHeroOperationalGuidanceSection()

	if got := sec.SectionID(); got != "install:hero-operational-guidance" {
		t.Errorf("SectionID() = %q, want install:hero-operational-guidance", got)
	}
	if got := sec.SectionTitle(); got != "Hero Binary & MCP Surface" {
		t.Errorf("SectionTitle() = %q, want Hero Binary & MCP Surface", got)
	}

	body, err := sec.Render(managed.Context{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	for _, want := range []string{
		"Prefer Hero's MCP tools over shelling out to a bare `hero`",
		"run `hero doctor`",
		"do NOT run `hero upgrade`",
		"cannot fix a wrong-binary-on-PATH situation.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Render output missing %q\n--- body ---\n%s", want, body)
		}
	}

	// `hero upgrade` must appear only as the explicitly-rejected remedy,
	// never as the recommended fix for schema confusion.
	if !strings.Contains(body, "do NOT run `hero upgrade` to \"fix schema\"") {
		t.Errorf("Render output must frame `hero upgrade` as a non-fix\n--- body ---\n%s", body)
	}

	// The section owns only the body; the orchestrator adds the heading, so
	// Render itself must not embed a markdown heading line.
	if strings.Contains(body, "\n## ") || strings.HasPrefix(body, "## ") ||
		strings.Contains(body, "\n### ") || strings.HasPrefix(body, "### ") {
		t.Errorf("Render output must not embed a heading (orchestrator emits it from SectionTitle)\n--- body ---\n%s", body)
	}
}
