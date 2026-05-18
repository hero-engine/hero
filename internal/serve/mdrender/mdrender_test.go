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
