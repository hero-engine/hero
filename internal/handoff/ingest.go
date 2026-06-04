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
	Goal        *SessionGoal
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
//
// localSlug is the slug the LOCAL reader (`hero resume`,
// `hero next ask/suggest/reflection`) will derive via nextUserSlug.
// When it differs from the file's recorded `user:` — which happens
// whenever git user.name / $USER diverges between the authoring and
// reading machines — the singletons are ALSO mirrored under the local
// slug so the read finds them. Mirroring is gated on safety on two
// fronts: it fires only when the local slug currently has zero handoff
// nodes of its own, AND only when singleTravelFile is true.
//
// singleTravelFile reports whether exactly ONE travel-eligible
// `.hero/next/*.md` file (excluding `*.local.md`) is present on disk.
// The caller computes this — the handoff package never enumerates the
// directory, to keep it free of a `cli` import. The flag closes the
// empty-graph-teammate leak: a brand-new teammate has zero handoff
// nodes too, so the zero-node gate alone would mirror whichever other
// user's file sorts first onto their identity. Aliasing is only
// unambiguous when a single user's handoff file exists (the solo
// cross-machine case — one person, two machines). When two or more
// travel-eligible files exist you cannot know which (if any) is "you,"
// so mirroring is suppressed and read-side resolution / fail-loud
// diagnostics handle the load instead.
//
// Pass "" for localSlug to disable mirroring (records only under the
// file identity).
func IngestUserFile(store *graph.Store, repoKey, domain, path, localSlug string, singleTravelFile bool) error {
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

	// Record under the file's recorded identity (the machine-independent
	// truth), then — when the local reader looks under a different slug —
	// mirror under that slug too, but only when it's safe to do so.
	users := []string{parsed.User}
	if alias := safeAliasSlug(store, repoKey, domain, parsed.User, localSlug, singleTravelFile); alias != "" {
		users = append(users, alias)
	}

	for _, user := range users {
		if parsed.Ask != nil && parsed.Ask.Text != "" {
			ask := *parsed.Ask
			ask.User = user
			ask.Domain = domain
			if err := RecordAsk(store, repoKey, ask); err != nil {
				return fmt.Errorf("ingest ask: %w", err)
			}
		}
		if parsed.Goal != nil && parsed.Goal.Text != "" {
			goal := *parsed.Goal
			goal.User = user
			goal.Domain = domain
			if err := RecordGoal(store, repoKey, goal); err != nil {
				return fmt.Errorf("ingest goal: %w", err)
			}
		}
		if parsed.Suggestion != nil && parsed.Suggestion.Text != "" {
			sug := *parsed.Suggestion
			sug.User = user
			sug.Domain = domain
			if err := RecordSuggestion(store, repoKey, sug); err != nil {
				return fmt.Errorf("ingest suggestion: %w", err)
			}
		}
		if len(parsed.Reflections) > 0 {
			existing, _ := RecentReflections(store, user, repoKey, domain, 100)
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
				r := ref
				r.User = user
				r.Domain = domain
				r.Text = text
				if err := RecordReflection(store, repoKey, r); err != nil {
					return fmt.Errorf("ingest reflection: %w", err)
				}
				seen[text] = struct{}{}
			}
		}
	}
	return nil
}

// safeAliasSlug decides whether the rehydrated singletons should be
// mirrored under localSlug in addition to fileUser. It returns the
// alias slug to mirror under, or "" to skip mirroring.
//
// The gate is deliberately conservative: alias only when localSlug is
// non-empty, differs from the file's user, exactly ONE travel-eligible
// handoff file exists on disk (singleTravelFile), and the local slug
// currently has ZERO handoff nodes of its own in the graph.
//
// The zero-node check is the anti-corruption invariant for an EXISTING
// teammate — a real second member already has their own handoff nodes,
// so the alias short-circuits and never attributes one person's handoff
// to another. But a brand-new teammate with an empty graph ALSO has zero
// nodes, so the zero-node check alone would mirror whichever other
// user's file sorts first onto their identity (asks/suggestions
// self-heal next checkpoint, but reflections leak and persist). The
// singleTravelFile condition closes that gap: when two or more
// travel-eligible files exist the alias is suppressed entirely, because
// no single identity is unambiguously "you." With exactly one file, the
// mirror is the solo cross-machine case — one person, two machines —
// and the alias fires so the cross-machine load succeeds.
func safeAliasSlug(store *graph.Store, repoKey, domain, fileUser, localSlug string, singleTravelFile bool) string {
	if localSlug == "" || localSlug == fileUser || !singleTravelFile {
		return ""
	}
	if ask, _ := LatestAsk(store, localSlug, repoKey, domain); ask != nil && ask.Text != "" {
		return ""
	}
	if sug, _ := LatestSuggestion(store, localSlug, repoKey, domain); sug != nil && sug.Text != "" {
		return ""
	}
	if refs, _ := RecentReflections(store, localSlug, repoKey, domain, 1); len(refs) > 0 {
		return ""
	}
	return localSlug
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
	if raw := stripBlockquote(sections["session goal"]); raw != "" {
		// The render glues a soft prefix ("Session opened with —" /
		// "Likely goal —") onto auto-derived goals and leaves marker/
		// manual goals bare. Recover both the text and the framing from
		// that prefix so the goal round-trips faithfully cross-machine
		// without masquerading as a manual override.
		text, source := parseGoalSection(raw)
		if text != "" {
			out.Goal = &SessionGoal{Text: text, Source: source, SessionID: fm["session"]}
		}
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

// parseGoalSection recovers the goal text and its source framing from
// the rendered "## Session goal" body. The projection writes a soft
// prefix for auto-derived goals and leaves marker/manual goals bare;
// this inverts that so the source survives the markdown round trip.
// A bare goal is treated as a marker (asserted, priority 2) — high
// enough to outrank a fresh window pass on the receiving machine but
// below a deliberate manual override.
func parseGoalSection(body string) (text, source string) {
	body = strings.TrimSpace(body)
	if rest, ok := strings.CutPrefix(body, "Session opened with —"); ok {
		return strings.TrimSpace(rest), GoalSourceAutoWindow
	}
	if rest, ok := strings.CutPrefix(body, "Likely goal —"); ok {
		return strings.TrimSpace(rest), GoalSourceAutoEmbed
	}
	return body, GoalSourceMarker
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
