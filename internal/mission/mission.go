// Package mission parses .hero/mission.md (the project charter) and
// upserts it as a Mission graph node so every context-emitting
// command can inject the mission as the highest-priority block in
// every agent session.
//
// This is Phase 1 of project-charter — the parser + ingest path.
// Phase 2 wires the injection sites; Phase 3 is the hero init wizard
// that synthesizes a starter draft.
package mission

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// Mission is the parsed projection of .hero/mission.md.
type Mission struct {
	Title    string
	Version  string
	LockedAt string
	LockedBy string
	Scope    string // "core" / "<vertical-name>" — defaults to "core"

	// Sections, all stored as raw markdown (modulo trimming) so
	// callers can render them verbatim into context bundles.
	MissionStatement string
	Principles       []Principle
	VocabPreferred   []VocabEntry
	VocabBanned      []VocabEntry
	AntiPatterns     []string
	MissionFitTest   string

	// Body is the full file body after frontmatter — useful when a
	// command wants to inject the entire charter rather than picking
	// specific sections.
	Body string
}

// Principle is a numbered experience principle. Hero's mission file
// uses "### N. <Name>." headings; the body is the explanation.
type Principle struct {
	Number int
	Name   string
	Body   string
}

// VocabEntry is one term in the Vocabulary section.
type VocabEntry struct {
	Term string
	Gloss string
}

// Path returns the canonical mission-file location for a hero
// workspace.
func Path(heroDir string) string {
	return filepath.Join(heroDir, "mission.md")
}

// Preamble returns a one-line mission anchor suitable for prepending
// to terse command outputs (`hero relevant`, `hero deliver`,
// `hero blocked`). Returns empty string when no charter is loaded —
// callers can use `if p := mission.Preamble(m); p != "" { print(p) }`.
//
// Format: `Mission — <statement> (Mission-fit test on every commit)`.
// We keep it to ≤200 chars so the anchor never drowns the actual
// command output.
func Preamble(m *Mission) string {
	if m == nil || m.MissionStatement == "" {
		return ""
	}
	stmt := strings.SplitN(m.MissionStatement, "\n", 2)[0]
	stmt = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(stmt, "**"), "**"))
	const max = 180
	if len(stmt) > max {
		stmt = stmt[:max] + "…"
	}
	return "Mission — " + stmt
}

// LoadFile reads heroDir/mission.md and parses it. Returns nil
// (no error) when the file doesn't exist — fresh workspaces don't
// have a charter yet.
func LoadFile(heroDir string) (*Mission, error) {
	path := Path(heroDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read mission: %w", err)
	}
	return Parse(data)
}

// Parse decodes a mission.md byte slice. Mission files are tolerant
// of formatting variation — missing sections produce empty fields,
// not parse errors. The contract is "any valid markdown with the
// canonical frontmatter is a valid mission."
func Parse(data []byte) (*Mission, error) {
	body, fm := splitFrontmatter(string(data))
	m := &Mission{
		Title:    fm["title"],
		Version:  fm["version"],
		LockedAt: fm["locked_at"],
		LockedBy: fm["locked_by"],
		Scope:    fm["scope"],
		Body:     strings.TrimSpace(body),
	}
	if m.Scope == "" {
		m.Scope = "core"
	}

	sections := splitSections(body)
	m.MissionStatement = strings.TrimSpace(sections["mission"])
	m.MissionFitTest = strings.TrimSpace(sections["mission-fit test"])
	m.Principles = parsePrinciples(sections["principles"])
	m.VocabPreferred, m.VocabBanned = parseVocab(sections["vocabulary"])
	m.AntiPatterns = parseAntiPatterns(sections["anti-patterns"])
	return m, nil
}

// WriteGraph upserts the Mission as a graph node keyed by scope so a
// single workspace can host multiple charters (core + per-vertical).
// Bitemporal: edits land as new rows; old rows get valid_to. Mission
// is global by design — never scoped local — and lives in the team
// scope alongside specs.
func (m *Mission) WriteGraph(repoKey string, store *graph.Store) error {
	if m == nil {
		return nil
	}
	props := map[string]any{
		"title":             m.Title,
		"version":           m.Version,
		"locked_at":         m.LockedAt,
		"locked_by":         m.LockedBy,
		"scope":             m.Scope,
		"mission_statement": m.MissionStatement,
		"mission_fit_test":  m.MissionFitTest,
		"body":              m.Body,
	}
	if names := principleNames(m.Principles); len(names) > 0 {
		props["principles"] = names
	}
	if len(m.AntiPatterns) > 0 {
		props["anti_patterns"] = m.AntiPatterns
	}
	if pref := termList(m.VocabPreferred); len(pref) > 0 {
		props["vocab_preferred"] = pref
	}
	if banned := termList(m.VocabBanned); len(banned) > 0 {
		props["vocab_banned"] = banned
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:        "Mission",
		Key:         m.Scope,
		Props:       props,
		Repo:        repoKey,
		ContentHash: hashMission(m),
		Source:      map[string]any{"kind": "mission", "path": Path("")},
	})
	return err
}

func principleNames(ps []Principle) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}

func termList(vs []VocabEntry) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Term)
	}
	return out
}

func hashMission(m *Mission) string {
	keys := []string{
		m.Title, m.Version, m.LockedAt, m.LockedBy, m.Scope,
		m.MissionStatement, m.MissionFitTest,
	}
	keys = append(keys, principleNames(m.Principles)...)
	keys = append(keys, m.AntiPatterns...)
	keys = append(keys, termList(m.VocabPreferred)...)
	keys = append(keys, termList(m.VocabBanned)...)
	sum := sha256.Sum256([]byte(strings.Join(keys, "|")))
	return hex.EncodeToString(sum[:])
}

// splitFrontmatter peels the leading YAML frontmatter and returns
// (body, kv-map). Tolerant of the simple key:value format the charter
// uses; nested YAML (like multi-line `source:`) is collapsed into the
// first line for our purposes.
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
	body = s[4+end+4:]
	for _, line := range strings.Split(header, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Skip multi-line YAML continuations (lines with no `:` at
		// the right indent) — already handled by Cut returning ok=false.
		if k == "" || strings.HasPrefix(k, "#") {
			continue
		}
		fm[k] = v
	}
	return body, fm
}

// splitSections walks `## <heading>` markers and returns a map keyed
// by the lowercased heading. Subheadings (`### …`) stay inline in the
// section body so the principles / vocab parsers can re-split them.
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

// parsePrinciples reads "### N. <Name>." subheadings within the
// principles section and returns one Principle per match. The body
// runs from after the heading to the next "### " or end of section.
func parsePrinciples(section string) []Principle {
	if strings.TrimSpace(section) == "" {
		return nil
	}
	var out []Principle
	var cur *Principle
	flush := func() {
		if cur == nil {
			return
		}
		cur.Body = strings.TrimSpace(cur.Body)
		out = append(out, *cur)
		cur = nil
	}
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "### ") {
			flush()
			head := strings.TrimPrefix(line, "### ")
			head = strings.TrimSpace(head)
			num, name := splitNumberedHeading(head)
			cur = &Principle{Number: num, Name: name}
			continue
		}
		if cur != nil {
			cur.Body += line + "\n"
		}
	}
	flush()
	sort.SliceStable(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// splitNumberedHeading takes "1. It just works." → (1, "It just works")
// and tolerates missing numbers (returns 0 + the whole heading).
func splitNumberedHeading(h string) (int, string) {
	dot := strings.Index(h, ". ")
	if dot < 0 {
		return 0, strings.TrimSuffix(strings.TrimSpace(h), ".")
	}
	num := 0
	for _, r := range h[:dot] {
		if r < '0' || r > '9' {
			num = 0
			break
		}
		num = num*10 + int(r-'0')
	}
	if num == 0 {
		return 0, strings.TrimSuffix(strings.TrimSpace(h), ".")
	}
	name := strings.TrimSpace(h[dot+2:])
	name = strings.TrimSuffix(name, ".")
	return num, name
}

// parseVocab pulls "Preferred:" and "Banned:" subsections, then bullet
// lines from each. Bullets look like "- **term** — gloss" or "- term — gloss".
func parseVocab(section string) (preferred, banned []VocabEntry) {
	if strings.TrimSpace(section) == "" {
		return nil, nil
	}
	mode := ""
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "**Preferred:**") || strings.EqualFold(trimmed, "Preferred:"):
			mode = "pref"
			continue
		case strings.HasPrefix(trimmed, "**Banned:**") || strings.EqualFold(trimmed, "Banned:"):
			mode = "banned"
			continue
		}
		if mode == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, "* ") {
			continue
		}
		bullet := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
		entry := parseVocabBullet(bullet)
		if entry.Term == "" {
			continue
		}
		if mode == "pref" {
			preferred = append(preferred, entry)
		} else {
			banned = append(banned, entry)
		}
	}
	return preferred, banned
}

// parseVocabBullet splits "**term** — gloss" or "term — gloss" into
// (term, gloss). Em-dash and ASCII-double-hyphen both supported.
func parseVocabBullet(bullet string) VocabEntry {
	for _, sep := range []string{" — ", " - ", " -- "} {
		if i := strings.Index(bullet, sep); i > 0 {
			return VocabEntry{
				Term:  cleanTerm(strings.TrimSpace(bullet[:i])),
				Gloss: strings.TrimSpace(bullet[i+len(sep):]),
			}
		}
	}
	return VocabEntry{Term: cleanTerm(bullet)}
}

func cleanTerm(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "**")
	s = strings.TrimSuffix(s, "**")
	s = strings.TrimPrefix(s, "*")
	s = strings.TrimSuffix(s, "*")
	return strings.TrimSpace(s)
}

// parseAntiPatterns walks "### <name>" subheadings and emits one
// entry per heading (the heading text). The bodies aren't preserved
// in the graph — they live in the file for human reading.
func parseAntiPatterns(section string) []string {
	var out []string
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "### ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "### "))
			if name != "" {
				out = append(out, name)
			}
		}
	}
	return out
}
