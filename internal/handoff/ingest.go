package handoff

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// ParsedHandoff is the structured form of .hero/next/<user>.md after
// parsing — the inverse of projection.UserHandoffMD.
type ParsedHandoff struct {
	User        string
	Ask         *UserAsk
	Suggestion  *NextSuggestion
	Reflections []SessionReflection
}

// IngestUserFile reads a per-user handoff markdown file, parses out
// the structured handoff sections, and upserts them into the graph.
// This is the load-bearing trick for solo-no-Cloud cross-machine
// continuity: the file IS the federation medium between machines.
//
// Idempotent: re-running on the same file produces no new graph
// edits when the content hasn't changed (UpsertNode skips no-op
// writes). Reflections are deduped by text match against existing
// entries since their timestamps are not preserved through markdown.
func IngestUserFile(store *graph.Store, repoKey, path string) error {
	if store == nil {
		return fmt.Errorf("handoff: nil store")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no file → nothing to ingest, success
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	parsed, err := ParseUserHandoff(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if parsed.User == "" {
		// No user in frontmatter → can't attribute. Skip silently.
		return nil
	}

	if parsed.Ask != nil && parsed.Ask.Text != "" {
		parsed.Ask.User = parsed.User
		if err := RecordAsk(store, repoKey, *parsed.Ask); err != nil {
			return fmt.Errorf("ingest ask: %w", err)
		}
	}
	if parsed.Suggestion != nil && parsed.Suggestion.Text != "" {
		parsed.Suggestion.User = parsed.User
		if err := RecordSuggestion(store, repoKey, *parsed.Suggestion); err != nil {
			return fmt.Errorf("ingest suggestion: %w", err)
		}
	}
	if len(parsed.Reflections) > 0 {
		existing, _ := RecentReflections(store, parsed.User, 100)
		seen := make(map[string]struct{}, len(existing))
		for _, e := range existing {
			seen[strings.TrimSpace(e.Text)] = struct{}{}
		}
		for _, ref := range parsed.Reflections {
			text := strings.TrimSpace(ref.Text)
			if text == "" {
				continue
			}
			if _, ok := seen[text]; ok {
				continue
			}
			ref.User = parsed.User
			ref.Text = text
			if err := RecordReflection(store, repoKey, ref); err != nil {
				return fmt.Errorf("ingest reflection: %w", err)
			}
			seen[text] = struct{}{}
		}
	}
	return nil
}

// ParseUserHandoff inverts projection.UserHandoffMD. Tolerant of
// formatting drift — missing sections produce nil/empty fields, not
// errors.
func ParseUserHandoff(data []byte) (*ParsedHandoff, error) {
	body, fm := splitFrontmatter(string(data))
	out := &ParsedHandoff{User: fm["user"]}

	sections := splitSections(body)

	if txt := stripBlockquote(sections["last user ask"]); txt != "" {
		out.Ask = &UserAsk{Text: txt, SessionID: fm["session"]}
	}
	if sec := sections["suggested next prompt"]; strings.TrimSpace(sec) != "" {
		// Skip auto-derived suggestions entirely. The projection
		// renders a "_Source: auto-derived from open feature_" footer
		// when it had to fall back from a stale agent suggestion. Re-
		// ingesting that text would corrupt the graph: the next
		// projection would treat the derived text as fresh agent
		// emission and the source footer would itself become part of
		// the recorded suggestion. Only round-trip what the agent
		// actually wrote.
		if !isAutoDerivedSection(sec) {
			text, rationale := parseSuggestionSection(sec)
			if text != "" {
				out.Suggestion = &NextSuggestion{
					Text:      text,
					Rationale: rationale,
					SessionID: fm["session"],
				}
			}
		}
	}
	if sec := sections["recent reflections"]; strings.TrimSpace(sec) != "" {
		for _, bullet := range parseBullets(sec) {
			out.Reflections = append(out.Reflections, SessionReflection{
				Text:      bullet,
				SessionID: fm["session"],
			})
		}
	}
	return out, nil
}

// splitFrontmatter peels the leading `---\n...\n---\n` block and
// returns (body, kv-map). Mirrors the parser used in the mission
// package — same simple key:value contract.
func splitFrontmatter(s string) (body string, fm map[string]string) {
	fm = map[string]string{}
	if !strings.HasPrefix(s, "---\n") {
		return s, fm
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return s, fm
	}
	header := s[4 : 4+end]
	body = strings.TrimLeft(s[4+end+4:], "\n")
	for _, line := range strings.Split(header, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(strings.ToLower(k))
		if k == "" {
			continue
		}
		fm[k] = strings.TrimSpace(v)
	}
	return body, fm
}

// splitSections walks `## <heading>` markers and returns a map keyed
// by the lowercased heading text. `### N. ...` subheadings stay
// inline in the section body. The pattern matches mission.go's
// implementation deliberately so both parsers can be reasoned about
// the same way.
func splitSections(body string) map[string]string {
	out := map[string]string{}
	var current string
	var buf strings.Builder
	flush := func() {
		if current == "" {
			return
		}
		out[strings.ToLower(strings.TrimSpace(current))] = strings.TrimSpace(buf.String())
		buf.Reset()
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			flush()
			current = strings.TrimPrefix(line, "## ")
			// Strip parenthetical suffix like "(this session)" so
			// the section key is stable.
			if i := strings.Index(current, "("); i > 0 {
				current = strings.TrimSpace(current[:i])
			}
			continue
		}
		if current == "" {
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()
	return out
}

// stripBlockquote removes leading `> ` from each line and joins. If
// the section is the placeholder italic ("_(none recorded — ...)_"),
// returns empty string so the parser treats it as "no value."
func stripBlockquote(section string) string {
	section = strings.TrimSpace(section)
	if section == "" || strings.HasPrefix(section, "_(") {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "> ")
		line = strings.TrimPrefix(line, ">")
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// isAutoDerivedSection returns true when the suggestion section
// contains the "_Source: auto-derived..._" footer the projection
// renders for fallback suggestions. Agent-emitted suggestions never
// carry this footer.
func isAutoDerivedSection(section string) bool {
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "_Source:") || strings.HasPrefix(t, "*Source:") {
			return true
		}
	}
	return false
}

// parseSuggestionSection extracts the suggestion body (blockquote)
// and the optional rationale (italicized "_Rationale: ..._" line).
func parseSuggestionSection(section string) (text, rationale string) {
	rationale = ""
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "_Rationale:") || strings.HasPrefix(t, "*Rationale:") {
			t = strings.TrimPrefix(t, "_Rationale:")
			t = strings.TrimPrefix(t, "*Rationale:")
			t = strings.TrimSuffix(t, "_")
			t = strings.TrimSuffix(t, "*")
			rationale = strings.TrimSpace(t)
			break
		}
	}
	// Strip the rationale line from the section before extracting
	// the blockquote body.
	var bodyLines []string
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "_Rationale:") || strings.HasPrefix(t, "*Rationale:") {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	text = stripBlockquote(strings.Join(bodyLines, "\n"))
	return text, rationale
}

// parseBullets returns "- " bullet bodies, in document order. Skips
// placeholder italic lines like "_(none yet)_".
func parseBullets(section string) []string {
	var out []string
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "- ") {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(t, "- "))
		if body == "" {
			continue
		}
		out = append(out, body)
	}
	return out
}
