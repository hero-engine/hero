package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/spf13/cobra"
)

var nextMigrateProjectionCmd = &cobra.Command{
	Use:   "migrate-to-projection",
	Short: "Switch NEXT.md from agent-authored to graph-projected",
	Long: `One-time migration that:

1. Captures the current NEXT.md content as a Note node so nothing
   is lost when projection takes over.
2. Extracts structured fields from the existing markdown:
     - "Last user ask" / first frontmatter quote → UserAsk node
     - "Proposed next ask" / "Suggested next prompt" → NextSuggestion
3. Updates .gitattributes so .hero/NEXT.md uses the hero-next merge
   driver (regen-on-conflict).
4. Sets next.projected = true in hero.json. From this point forward,
   hero next checkpoint regenerates NEXT.md from the graph each turn.

Idempotent — re-running detects the projected flag and is a no-op
(prints a notice and exits 0).

To opt out, set next.projected = false in hero.json. The captured
historical Note remains in the graph either way.`,
	RunE: runNextMigrateProjection,
}

func runNextMigrateProjection(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	out := cmd.OutOrStdout()

	if cfg.NextProjected() {
		fmt.Fprintln(out, "Already migrated. next.projected is true in hero.json.")
		return nil
	}

	heroDir := cfg.HeroDir(projectRoot)
	nextPath := filepath.Join(heroDir, "NEXT.md")
	body, err := os.ReadFile(nextPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read NEXT.md: %w", err)
	}

	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("open graph: %w", err)
	}
	defer store.Close()
	repoKey := gitutil.RepoKey(projectRoot)
	user := nextUserSlug(cfg)

	// 1. Capture existing content as a Note node — nothing is lost.
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := captureNextSnapshot(store, repoKey, body); err != nil {
			return fmt.Errorf("capturing snapshot: %w", err)
		}
		fmt.Fprintln(out, "captured existing NEXT.md as Note:next-md-migration-snapshot")
	}

	// 2. Extract structured fields → graph nodes.
	if user != "" {
		if ask, sug := extractAskAndSuggestion(string(body)); ask != "" || sug != "" {
			if ask != "" {
				if err := handoff.RecordAsk(store, repoKey, handoff.UserAsk{
					User: user, Text: ask,
				}); err != nil {
					return fmt.Errorf("ingest ask: %w", err)
				}
				fmt.Fprintln(out, "ingested last user ask → UserAsk:"+user)
			}
			if sug != "" {
				if err := handoff.RecordSuggestion(store, repoKey, handoff.NextSuggestion{
					User: user, Text: sug,
				}); err != nil {
					return fmt.Errorf("ingest suggestion: %w", err)
				}
				fmt.Fprintln(out, "ingested suggested next → NextSuggestion:"+user)
			}
		}
	}

	// 3. Update .gitattributes so NEXT.md uses the hero-next merge
	// driver. The Phase-6 marker block already covers .hero/next/*.md;
	// this appends a NEXT.md line to it idempotently.
	if err := ensureNextMDMergeDirective(projectRoot); err != nil {
		return fmt.Errorf("update .gitattributes: %w", err)
	}
	fmt.Fprintln(out, "updated .gitattributes for .hero/NEXT.md merge=hero-next")

	// 4. Flip next.projected = true in .hero/hero.json (or create
	// the file if it doesn't exist with just that field).
	if err := setNextProjected(heroDir, true); err != nil {
		return fmt.Errorf("update hero.json: %w", err)
	}
	fmt.Fprintln(out, "set next.projected = true in .hero/hero.json")

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Migration complete. Run 'hero next checkpoint' to regenerate NEXT.md from the graph.")
	return nil
}

// captureNextSnapshot stores the full NEXT.md body as a Note node
// keyed by a timestamped slug. The snapshot is searchable via hero
// search and recoverable in case the projection ever regrets a wipe.
func captureNextSnapshot(store *graph.Store, repoKey string, body []byte) error {
	stamp := time.Now().UTC().Format("20060102T150405")
	key := "next-md-migration-snapshot-" + stamp
	props := map[string]any{
		"title":     "NEXT.md migration snapshot " + stamp,
		"body":      string(body),
		"captured":  time.Now().UTC().Format(time.RFC3339),
		"reason":    "next-as-projection migration; preserving pre-projection content",
	}
	_, err := store.UpsertNode(&graph.Node{
		Type:   "Note",
		Key:    key,
		Props:  props,
		Repo:   repoKey,
		Source: map[string]any{"kind": "next-md-migration"},
	})
	return err
}

// extractAskAndSuggestion tries to pull the user-ask and next-
// suggestion text out of an existing hand-written NEXT.md. Tolerant
// of formatting drift — returns whatever it finds.
func extractAskAndSuggestion(body string) (ask, suggestion string) {
	sections := splitNextSections(body)
	ask = firstQuoteOrText(sections["last user ask"])
	// "Suggested next prompt" is the new spec'd section name; older
	// NEXT.md drafts used "Proposed next ask" — prefer the new name
	// when both are present so a manually-updated file wins over a
	// stale drafting carried over from a prior session.
	if s := firstQuoteOrText(sections["suggested next prompt"]); s != "" {
		suggestion = s
	} else if s := firstQuoteOrText(sections["proposed next ask"]); s != "" {
		suggestion = s
	}
	return ask, suggestion
}

// splitNextSections is a small section splitter for the existing
// hand-written NEXT.md format. Lowercased headings as keys.
func splitNextSections(body string) map[string]string {
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

// firstQuoteOrText returns the first blockquote body in the section,
// or the first non-empty paragraph if no blockquote is present.
func firstQuoteOrText(section string) string {
	if section == "" {
		return ""
	}
	var quote []string
	inQuote := false
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, ">") {
			inQuote = true
			quote = append(quote, strings.TrimSpace(strings.TrimPrefix(t, ">")))
			continue
		}
		if inQuote && t == "" {
			break // end of blockquote
		}
		if inQuote {
			break // blockquote ended
		}
	}
	if len(quote) > 0 {
		return strings.TrimSpace(strings.Join(quote, " "))
	}
	// No blockquote — first non-italic, non-empty paragraph.
	var para []string
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(t, "_") || strings.HasPrefix(t, "*_") {
			continue // italic placeholder
		}
		para = append(para, t)
	}
	return strings.TrimSpace(strings.Join(para, " "))
}

// ensureNextMDMergeDirective appends a NEXT.md merge-directive line
// inside the existing Phase-6 marker block in .gitattributes. If the
// marker block doesn't exist yet, falls back to creating a new one.
// Idempotent.
func ensureNextMDMergeDirective(projectRoot string) error {
	path := filepath.Join(projectRoot, ".gitattributes")
	existing, _ := os.ReadFile(path)
	src := string(existing)

	directives := []string{
		".hero/NEXT.md merge=" + mergeDriverName,
		".hero/SNAPSHOT.md merge=" + mergeDriverName,
	}

	startIdx := strings.Index(src, gaMarkerStart)
	if startIdx < 0 {
		// No existing block — create one with all directives.
		block := fmt.Sprintf(`%s
.hero/NEXT.md merge=%s
.hero/next/*.md merge=%s
.hero/SNAPSHOT.md merge=%s
%s`, gaMarkerStart, mergeDriverName, mergeDriverName, mergeDriverName, gaMarkerEnd)
		body := mergeMarkerBlock(src, gaMarkerStart, gaMarkerEnd, block)
		return os.WriteFile(path, []byte(body), 0o644)
	}
	// Existing block — splice in any missing directive.
	body := src
	for _, directive := range directives {
		if strings.Contains(body, directive) {
			continue
		}
		endIdx := strings.Index(body, gaMarkerEnd)
		if endIdx < 0 {
			return fmt.Errorf("malformed .gitattributes: missing %q", gaMarkerEnd)
		}
		insertAt := strings.LastIndex(body[:endIdx], "\n") + 1
		body = body[:insertAt] + directive + "\n" + body[insertAt:]
	}
	if body == src {
		return nil
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// setNextProjected updates .hero/hero.json (or creates it) with
// next.projected = value. Preserves any other fields in the file.
//
// Important: the config lives at .hero/hero.json, not at the project
// root — config.Load reads from that path. Writing to the wrong path
// produces silent no-ops at runtime.
func setNextProjected(heroDir string, value bool) error {
	path := filepath.Join(heroDir, "hero.json")
	var doc map[string]any
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse hero.json: %w", err)
		}
	}
	if doc == nil {
		doc = map[string]any{}
	}

	next, _ := doc["next"].(map[string]any)
	if next == nil {
		next = map[string]any{}
	}
	next["projected"] = value
	doc["next"] = next

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode hero.json: %w", err)
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
