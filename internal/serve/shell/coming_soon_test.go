package shell

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/serve/edition"
)

// TestComingSoonStub_NoInlineStyles guards polish-v2 Fix 3 — the
// shared coming-soon shell fragment must render with zero inline
// `style="…"` attributes. All styling lives in shell.css under the
// `.cs-*` rules; future theme changes should not be defeated by
// inline style attributes baked into the template.
func TestComingSoonStub_NoInlineStyles(t *testing.T) {
	r := New(edition.Local, nil, "hero", "main", "test-user", "test")

	stub := struct {
		Home string
		Slug string
		View string
		Note string
	}{
		Home: "knowledge",
		Slug: "recent",
		View: "Recent",
		Note: "An optional note.",
	}

	var buf bytes.Buffer
	if err := r.RenderFragment(&buf, "coming-soon", stub); err != nil {
		t.Fatalf("RenderFragment: %v", err)
	}
	out := buf.String()

	if strings.Contains(out, `style="`) {
		t.Errorf("coming-soon fragment still contains inline style attribute:\n%s", out)
	}
	// Sanity-check that the class hooks landed.
	for _, want := range []string{"cs-card", "cs-header", "cs-title", "cs-body", "cs-link"} {
		if !strings.Contains(out, want) {
			t.Errorf("coming-soon fragment missing class %q:\n%s", want, out)
		}
	}
}
