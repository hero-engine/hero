package mdrender

import (
	"strings"
	"testing"
)

func TestRender_HeadingsAndParagraphs(t *testing.T) {
	out := string(Render("# Title\n\nA paragraph with **bold** and *em*."))
	if !strings.Contains(out, "<h1>Title</h1>") {
		t.Errorf("missing <h1>; got %q", out)
	}
	if !strings.Contains(out, "<strong>bold</strong>") {
		t.Errorf("missing <strong>; got %q", out)
	}
	if !strings.Contains(out, "<em>em</em>") {
		t.Errorf("missing <em>; got %q", out)
	}
}

func TestRender_BulletList(t *testing.T) {
	out := string(Render("- one\n- two\n- three\n"))
	if !strings.Contains(out, "<ul>") || !strings.Contains(out, "<li>one</li>") {
		t.Errorf("bullet list not rendered: %q", out)
	}
}

func TestRender_NumberedList(t *testing.T) {
	out := string(Render("1. first\n2. second\n"))
	if !strings.Contains(out, "<ol>") || !strings.Contains(out, "<li>first</li>") {
		t.Errorf("numbered list not rendered: %q", out)
	}
}

func TestRender_FencedCode(t *testing.T) {
	out := string(Render("```go\nfmt.Println(\"hi\")\n```"))
	if !strings.Contains(out, `<pre><code class="language-go">`) {
		t.Errorf("missing fenced code with language; got %q", out)
	}
	if !strings.Contains(out, "fmt.Println(&#34;hi&#34;)") {
		t.Errorf("code body not escaped: %q", out)
	}
}

func TestRender_InlineCode(t *testing.T) {
	out := string(Render("Use `hero pull` to sync."))
	if !strings.Contains(out, "<code>hero pull</code>") {
		t.Errorf("inline code missing: %q", out)
	}
}

func TestRender_LinkSafety(t *testing.T) {
	out := string(Render("[click](javascript:alert(1))"))
	if strings.Contains(out, "javascript:") {
		t.Errorf("javascript: scheme leaked through: %q", out)
	}
	if !strings.Contains(out, `href="#"`) {
		t.Errorf("unsafe link not sanitized to #: %q", out)
	}
}

func TestRender_NoRawHTML(t *testing.T) {
	out := string(Render("<script>alert(1)</script>"))
	if strings.Contains(out, "<script>") {
		t.Errorf("raw HTML leaked through: %q", out)
	}
}

func TestRender_Autolink(t *testing.T) {
	out := string(Render("See https://example.com for more."))
	if !strings.Contains(out, `href="https://example.com"`) {
		t.Errorf("autolink missing: %q", out)
	}
}

func TestRender_Table(t *testing.T) {
	md := "| Col A | Col B |\n| ----- | ----- |\n| v1    | v2    |\n| v3    | v4    |\n"
	out := string(Render(md))
	for _, want := range []string{
		"<table>",
		"<thead>",
		"<th>Col A</th>",
		"<th>Col B</th>",
		"<tbody>",
		"<td>v1</td>",
		"<td>v4</td>",
		"</table>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table render missing %q in %q", want, out)
		}
	}
}

func TestRender_TableMalformedFallsBack(t *testing.T) {
	// Separator row missing — should NOT render as a table; should
	// instead render the two lines as paragraph text without panic.
	md := "| Col A | Col B |\n| v1    | v2    |\n"
	out := string(Render(md))
	if strings.Contains(out, "<table>") {
		t.Errorf("malformed table should not render <table>: %q", out)
	}
	if strings.Contains(out, "<th>") {
		t.Errorf("malformed table should not render <th>: %q", out)
	}
}

func TestRender_TableColCountMismatchFallsBack(t *testing.T) {
	// Header has 2 cols, body has 3 cols — should bail out.
	md := "| A | B |\n| --- | --- |\n| 1 | 2 | 3 |\n"
	out := string(Render(md))
	if strings.Contains(out, "<table>") {
		t.Errorf("col-count mismatch should not render <table>: %q", out)
	}
}

func TestRender_Blockquote(t *testing.T) {
	md := "> Status: delivered\n> Second line of the quote.\n> Third line here.\n"
	out := string(Render(md))
	if !strings.Contains(out, "<blockquote>") {
		t.Errorf("blockquote missing: %q", out)
	}
	if !strings.Contains(out, "Status: delivered") {
		t.Errorf("blockquote body missing: %q", out)
	}
	if !strings.Contains(out, "</blockquote>") {
		t.Errorf("blockquote close missing: %q", out)
	}
}

func TestRender_NestedBulletList(t *testing.T) {
	md := "- top one\n  - nested A\n  - nested B\n- top two\n"
	out := string(Render(md))
	// Expect outer <ul>, an inner <ul> within an <li>, both nested
	// items present, and the second top item present.
	if !strings.Contains(out, "<ul>") {
		t.Errorf("outer <ul> missing: %q", out)
	}
	// There should be at least two <ul> opens (outer + nested).
	if strings.Count(out, "<ul>") < 2 {
		t.Errorf("nested <ul> missing (got %d <ul> opens) in %q", strings.Count(out, "<ul>"), out)
	}
	for _, want := range []string{"top one", "nested A", "nested B", "top two"} {
		if !strings.Contains(out, want) {
			t.Errorf("item %q missing: %q", want, out)
		}
	}
}
