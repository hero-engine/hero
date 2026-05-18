// Package mdrender is a tiny block-level markdown-to-HTML renderer.
// It exists so the serve homes (Knowledge entry detail, Work spec
// detail) can render persisted spec / knowledge markdown without
// pulling in a third-party dependency.
//
// What it handles:
//   - ATX headings (`# … `, `## …`, up to h6)
//   - Bulleted lists (`- …` / `* …`)
//   - Numbered lists (`1. …`)
//   - Fenced code blocks (```…```), preserving language for the class
//   - Inline code (backticks)
//   - Bold (`**…**`) and italic (`*…*` / `_…_`)
//   - Autolinks (bare URLs)
//   - Paragraphs (anything that's not one of the above)
//   - Horizontal rules (`---` on its own line)
//
// What it intentionally does NOT handle (yet):
//   - Tables, blockquotes, nested lists, images, raw HTML pass-through,
//     reference-style links, footnotes.
//
// Output is HTML-escaped before any inline transforms run, so the
// renderer is safe to feed untrusted markdown — there is no path that
// emits an unescaped substring from the input.
package mdrender

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
)

// Render converts markdown to safe HTML. Returns template.HTML so
// callers can drop the result into html/template without re-escaping.
func Render(md string) template.HTML {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	lines := strings.Split(md, "\n")
	var out strings.Builder

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Fenced code block.
		if strings.HasPrefix(trimmed, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			var buf strings.Builder
			i++
			for i < len(lines) {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
					i++
					break
				}
				buf.WriteString(lines[i])
				buf.WriteByte('\n')
				i++
			}
			if lang != "" {
				fmt.Fprintf(&out, `<pre><code class="language-%s">`, template.HTMLEscapeString(lang))
			} else {
				out.WriteString(`<pre><code>`)
			}
			out.WriteString(template.HTMLEscapeString(strings.TrimRight(buf.String(), "\n")))
			out.WriteString("</code></pre>\n")
			continue
		}

		// Blank line.
		if trimmed == "" {
			i++
			continue
		}

		// Horizontal rule.
		if trimmed == "---" || trimmed == "***" {
			out.WriteString("<hr/>\n")
			i++
			continue
		}

		// Heading (ATX). Up to h6.
		if level := atxLevel(trimmed); level > 0 {
			text := strings.TrimSpace(trimmed[level:])
			fmt.Fprintf(&out, "<h%d>%s</h%d>\n", level, inline(text), level)
			i++
			continue
		}

		// Bulleted list.
		if isBullet(trimmed) {
			out.WriteString("<ul>\n")
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !isBullet(t) {
					break
				}
				item := strings.TrimSpace(t[2:])
				fmt.Fprintf(&out, "  <li>%s</li>\n", inline(item))
				i++
			}
			out.WriteString("</ul>\n")
			continue
		}

		// Numbered list.
		if isNumbered(trimmed) {
			out.WriteString("<ol>\n")
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !isNumbered(t) {
					break
				}
				dot := strings.Index(t, ".")
				item := strings.TrimSpace(t[dot+1:])
				fmt.Fprintf(&out, "  <li>%s</li>\n", inline(item))
				i++
			}
			out.WriteString("</ol>\n")
			continue
		}

		// Paragraph — consume contiguous non-blank lines.
		var para []string
		for i < len(lines) {
			t := strings.TrimSpace(lines[i])
			if t == "" || atxLevel(t) > 0 || isBullet(t) || isNumbered(t) ||
				strings.HasPrefix(t, "```") || t == "---" {
				break
			}
			para = append(para, t)
			i++
		}
		fmt.Fprintf(&out, "<p>%s</p>\n", inline(strings.Join(para, " ")))
	}

	return template.HTML(out.String())
}

// atxLevel returns the ATX heading level (1-6) for a line, or 0 when
// the line isn't a heading.
func atxLevel(trimmed string) int {
	level := 0
	for level < len(trimmed) && level < 6 && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0
	}
	return level
}

func isBullet(t string) bool {
	return strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ")
}

func isNumbered(t string) bool {
	dot := strings.Index(t, ".")
	if dot <= 0 || dot >= len(t)-1 || t[dot+1] != ' ' {
		return false
	}
	for j := 0; j < dot; j++ {
		if t[j] < '0' || t[j] > '9' {
			return false
		}
	}
	return true
}

// inline transforms inline markdown into HTML on already-escaped text.
// Order matters: we escape FIRST, then convert markers — so user
// input cannot smuggle raw HTML through.
func inline(text string) string {
	// Pull code spans out first; the escape pass on the rest must not
	// disturb literal backticks inside code spans.
	escaped := template.HTMLEscapeString(text)

	// Inline code: `code`. Operate on the escaped string so any HTML
	// inside the span is literal.
	escaped = backtickRun(escaped)

	// Bold then italic. Bold first so `**foo**` is recognized whole
	// before italic catches the leading `*`.
	escaped = wrap(escaped, "**", "<strong>", "</strong>")
	escaped = wrap(escaped, "__", "<strong>", "</strong>")
	escaped = wrap(escaped, "*", "<em>", "</em>")
	escaped = wrap(escaped, "_", "<em>", "</em>")

	// Markdown links [text](url) — text is already escaped; sanitize
	// the URL so javascript: never gets through.
	escaped = renderLinks(escaped)

	// Bare URL autolink.
	escaped = autolink(escaped)

	return escaped
}

// backtickRun replaces `code` spans with <code>code</code>. Greedy
// match on a single line is fine — multi-line spans are rare in spec
// markdown and would be a fenced block anyway.
func backtickRun(s string) string {
	var out strings.Builder
	for {
		i := strings.IndexByte(s, '`')
		if i < 0 {
			out.WriteString(s)
			break
		}
		j := strings.IndexByte(s[i+1:], '`')
		if j < 0 {
			out.WriteString(s)
			break
		}
		out.WriteString(s[:i])
		out.WriteString("<code>")
		out.WriteString(s[i+1 : i+1+j])
		out.WriteString("</code>")
		s = s[i+1+j+1:]
	}
	return out.String()
}

// wrap pairs occurrences of marker around `open`/`close`. Skips
// markers that border whitespace on the inside (so a literal `*` in
// prose doesn't get rewritten).
func wrap(s, marker, open, close string) string {
	out := s
	for {
		i := strings.Index(out, marker)
		if i < 0 {
			break
		}
		j := strings.Index(out[i+len(marker):], marker)
		if j < 0 {
			break
		}
		inner := out[i+len(marker) : i+len(marker)+j]
		if inner == "" || inner[0] == ' ' || inner[len(inner)-1] == ' ' {
			// Skip this opener — advance past it so we don't loop.
			out = out[:i] + "\x00" + out[i+len(marker):]
			continue
		}
		out = out[:i] + open + inner + close + out[i+len(marker)+j+len(marker):]
	}
	// Restore any temporarily-skipped markers.
	return strings.ReplaceAll(out, "\x00", marker[:1])
}

// renderLinks rewrites [text](href) into <a href="href">text</a> with
// the href safety-checked.
func renderLinks(s string) string {
	var out strings.Builder
	for {
		open := strings.IndexByte(s, '[')
		if open < 0 {
			out.WriteString(s)
			break
		}
		close := strings.IndexByte(s[open:], ']')
		if close < 0 {
			out.WriteString(s)
			break
		}
		close += open
		if close+1 >= len(s) || s[close+1] != '(' {
			out.WriteString(s[:close+1])
			s = s[close+1:]
			continue
		}
		hrefEnd := strings.IndexByte(s[close+2:], ')')
		if hrefEnd < 0 {
			out.WriteString(s[:close+1])
			s = s[close+1:]
			continue
		}
		hrefEnd += close + 2
		text := s[open+1 : close]
		href := s[close+2 : hrefEnd]
		out.WriteString(s[:open])
		fmt.Fprintf(&out, `<a href="%s">%s</a>`, template.HTMLEscapeString(safeHref(href)), text)
		s = s[hrefEnd+1:]
	}
	return out.String()
}

// autolink wraps bare http(s) URLs in an <a> tag. Operates on already-
// escaped text so we look for the escaped scheme.
func autolink(s string) string {
	var out strings.Builder
	for {
		i := indexAny(s, []string{"http://", "https://"})
		if i < 0 {
			out.WriteString(s)
			break
		}
		// Find URL end — first whitespace or HTML-significant char.
		end := i
		for end < len(s) {
			c := s[end]
			if c == ' ' || c == '\t' || c == '\n' || c == '<' || c == ')' {
				break
			}
			end++
		}
		out.WriteString(s[:i])
		// Don't autolink if we're already inside an href= attribute.
		if i >= 6 && strings.HasSuffix(out.String(), `href="`) {
			out.WriteString(s[i:end])
		} else {
			href := safeHref(s[i:end])
			fmt.Fprintf(&out, `<a href="%s">%s</a>`, template.HTMLEscapeString(href), s[i:end])
		}
		s = s[end:]
	}
	return out.String()
}

func indexAny(s string, needles []string) int {
	best := -1
	for _, n := range needles {
		i := strings.Index(s, n)
		if i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	return best
}

// safeHref returns href if it parses to an http(s), mailto, fragment,
// or workspace-relative URL. Otherwise returns "#". Defends against
// `javascript:` injection.
func safeHref(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return "#"
	}
	if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "/") {
		return href
	}
	u, err := url.Parse(href)
	if err != nil {
		return "#"
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto":
		return href
	case "":
		return href
	}
	return "#"
}
