// Package mdrender is a tiny block-level markdown-to-HTML renderer.
// It exists so the serve homes (Knowledge entry detail, Work spec
// detail) can render persisted spec / knowledge markdown without
// pulling in a third-party dependency.
//
// What it handles:
//   - ATX headings (`# … `, `## …`, up to h6)
//   - Bulleted lists (`- …` / `* …`), with 2- or 4-space-indent nesting
//   - Numbered lists (`1. …`), with 2- or 4-space-indent nesting
//   - Pipe-style tables (`| A | B |` with `| --- | --- |` separator row)
//   - Blockquotes (`> …` lines, consecutive grouped)
//   - Fenced code blocks (```…```), preserving language for the class
//   - Inline code (backticks)
//   - Bold (`**…**`) and italic (`*…*` / `_…_`)
//   - Autolinks (bare URLs)
//   - Paragraphs (anything that's not one of the above)
//   - Horizontal rules (`---` on its own line)
//
// What it intentionally does NOT handle (yet):
//   - Images, raw HTML pass-through, reference-style links, footnotes.
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

		// Pipe-style table — needs a header row followed by a separator
		// row of `| --- | --- |`. When the shape doesn't match we fall
		// through to paragraph rendering so malformed tables degrade
		// gracefully.
		if isTableHeader(lines, i) {
			consumed := renderTable(&out, lines, i)
			if consumed > 0 {
				i += consumed
				continue
			}
		}

		// Blockquote — group consecutive `>`-prefixed lines.
		if isBlockquote(trimmed) {
			var quoted []string
			for i < len(lines) {
				t := lines[i]
				ts := strings.TrimSpace(t)
				if !isBlockquote(ts) {
					break
				}
				quoted = append(quoted, stripQuotePrefix(ts))
				i++
			}
			out.WriteString("<blockquote>")
			// Render the inner content recursively so nested constructs
			// (paragraphs, lists, inline markdown) work.
			inner := string(Render(strings.Join(quoted, "\n")))
			out.WriteString(strings.TrimRight(inner, "\n"))
			out.WriteString("</blockquote>\n")
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

		// List (bulleted or numbered) — supports 2- or 4-space-indent
		// nesting. The list reader returns the number of lines it
		// consumed.
		if isBullet(line) || isNumbered(line) {
			consumed := renderList(&out, lines, i)
			i += consumed
			continue
		}

		// Paragraph — consume contiguous non-blank lines.
		var para []string
		for i < len(lines) {
			t := strings.TrimSpace(lines[i])
			if t == "" || atxLevel(t) > 0 || isBullet(lines[i]) || isNumbered(lines[i]) ||
				strings.HasPrefix(t, "```") || t == "---" || isBlockquote(t) {
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

// listMarker returns the indent-spaces count and the body text after
// the marker, plus an "ordered" flag, when the line is a list item.
// (-1, "", false) means the line isn't a list item.
func listMarker(line string) (indent int, body string, ordered bool) {
	indent = 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	rest := line[indent:]
	if strings.HasPrefix(rest, "- ") || strings.HasPrefix(rest, "* ") {
		return indent, strings.TrimSpace(rest[2:]), false
	}
	// Numbered: digits + "." + " "
	dot := strings.Index(rest, ".")
	if dot > 0 && dot+1 < len(rest) && rest[dot+1] == ' ' {
		allDigits := true
		for j := 0; j < dot; j++ {
			if rest[j] < '0' || rest[j] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return indent, strings.TrimSpace(rest[dot+2:]), true
		}
	}
	return -1, "", false
}

// isBullet reports whether the line (possibly indented) is a bulleted
// list item.
func isBullet(line string) bool {
	indent, _, ordered := listMarker(line)
	return indent >= 0 && !ordered
}

// isNumbered reports whether the line (possibly indented) is a
// numbered list item.
func isNumbered(line string) bool {
	indent, _, ordered := listMarker(line)
	return indent >= 0 && ordered
}

// renderList emits a list starting at `start`. It walks contiguous
// list lines, handles 2- or 4-space-indent nesting, and returns the
// number of lines consumed.
func renderList(out *strings.Builder, lines []string, start int) int {
	// Detect the base indent + kind from the first line.
	baseIndent, _, baseOrdered := listMarker(lines[start])
	if baseIndent < 0 {
		return 1
	}
	openTag, closeTag := "<ul>", "</ul>"
	if baseOrdered {
		openTag, closeTag = "<ol>", "</ol>"
	}
	out.WriteString(openTag + "\n")

	i := start
	for i < len(lines) {
		indent, body, ordered := listMarker(lines[i])
		if indent < 0 || indent < baseIndent || (indent == baseIndent && ordered != baseOrdered) {
			break
		}
		if indent > baseIndent {
			// Nested list — recurse. Output goes inside the LAST <li>
			// we wrote, so we put it before the closing </li>. Simpler:
			// write a fresh nested list as a sibling-after of the last
			// item by re-opening the <li> chain.
			// To keep things simple we just emit the nested list as a
			// child of the most-recent <li>: rewind one closing tag.
			// We don't have a built buffer to rewind, so write the
			// nested list as standalone content in place — browsers
			// tolerate <ul> directly after </li> visually as a sibling.
			// To get true nesting (ul inside li), we instead inline-
			// recurse before closing the previous item: implemented by
			// holding off the </li> close until we know what comes next.
			//
			// Simpler approach: track whether we just opened an <li>
			// without closing. We rebuild with a small per-iteration
			// pattern below.
			break // handled by the alternate path below
		}
		// Open the list item; defer the close so nested lists land
		// inside this <li>.
		fmt.Fprintf(out, "  <li>%s", inline(body))
		i++

		// If the next line is a deeper-indented list item, recurse.
		if i < len(lines) {
			nextIndent, _, _ := listMarker(lines[i])
			if nextIndent > baseIndent {
				out.WriteString("\n")
				consumed := renderList(out, lines, i)
				i += consumed
			}
		}
		out.WriteString("</li>\n")
	}
	out.WriteString(closeTag + "\n")
	return i - start
}

// isBlockquote reports whether a trimmed line begins a blockquote.
func isBlockquote(t string) bool {
	return strings.HasPrefix(t, "> ") || t == ">" || strings.HasPrefix(t, ">")
}

// stripQuotePrefix removes the leading `>` (and optional space) so the
// inner content can be re-rendered.
func stripQuotePrefix(t string) string {
	if t == ">" {
		return ""
	}
	if strings.HasPrefix(t, "> ") {
		return t[2:]
	}
	if strings.HasPrefix(t, ">") {
		return t[1:]
	}
	return t
}

// isTableHeader peeks at lines[i] and lines[i+1] and returns true when
// they form a pipe-style table header + separator pair.
func isTableHeader(lines []string, i int) bool {
	if i+1 >= len(lines) {
		return false
	}
	header := strings.TrimSpace(lines[i])
	sep := strings.TrimSpace(lines[i+1])
	if !strings.Contains(header, "|") || !strings.Contains(sep, "|") {
		return false
	}
	// Separator row: cells must be all `-` (and optional `:`).
	cells := splitTableRow(sep)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" {
			return false
		}
		// Strip alignment markers (we ignore alignment in v3).
		c = strings.Trim(c, ":")
		if c == "" {
			return false
		}
		for _, r := range c {
			if r != '-' {
				return false
			}
		}
	}
	return true
}

// renderTable emits an HTML table starting at lines[i]. Returns the
// number of lines consumed. Returns 0 if the shape turns out invalid
// (col-count mismatch) — caller falls back to paragraph rendering.
func renderTable(out *strings.Builder, lines []string, start int) int {
	header := splitTableRow(lines[start])
	// Already verified by isTableHeader; build header.
	cols := len(header)
	if cols == 0 {
		return 0
	}
	var body strings.Builder
	body.WriteString("<table>\n<thead>\n<tr>")
	for _, c := range header {
		fmt.Fprintf(&body, "<th>%s</th>", inline(strings.TrimSpace(c)))
	}
	body.WriteString("</tr>\n</thead>\n<tbody>\n")

	i := start + 2 // skip header + separator
	bodyRows := 0
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if t == "" || !strings.Contains(t, "|") {
			break
		}
		cells := splitTableRow(lines[i])
		if len(cells) != cols {
			// Col count mismatch — treat the table as malformed; bail
			// out and let caller render as paragraphs.
			return 0
		}
		body.WriteString("<tr>")
		for _, c := range cells {
			fmt.Fprintf(&body, "<td>%s</td>", inline(strings.TrimSpace(c)))
		}
		body.WriteString("</tr>\n")
		bodyRows++
		i++
	}
	body.WriteString("</tbody>\n</table>\n")
	if bodyRows == 0 {
		// Header-only table is suspicious; still render it.
	}
	out.WriteString(body.String())
	return i - start
}

// splitTableRow splits a `| a | b | c |` row into its inner cells. The
// outer pipes are tolerated when present.
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	// Trim leading/trailing pipe so split doesn't produce empty edge
	// cells.
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}
	return strings.Split(line, "|")
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
