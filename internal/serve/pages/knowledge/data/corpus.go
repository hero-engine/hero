package data

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// prettyAge returns a Linear-style relative time string ("14m ago",
// "2h ago", "3d ago"). Mirrors the Now-home helper of the same name.
func prettyAge(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// CorpusInputs is the per-request input bundle for the corpus / browse
// section.
type CorpusInputs struct {
	HeroDir string
	Limit   int // 0 → default (200)
}

// LoadCorpus walks `.hero/knowledge/<kind>/` collecting markdown entries
// and produces the metric-strip counters plus the entry index used by
// the browse view. Best-effort: returns zeros + nil rows when the
// directory is missing.
func LoadCorpus(in CorpusInputs) Corpus {
	if in.Limit <= 0 {
		in.Limit = 200
	}
	out := Corpus{}
	if in.HeroDir == "" {
		return out
	}
	root := filepath.Join(in.HeroDir, "knowledge")
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return out
	}

	entries := []CorpusEntry{}
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	newThisWeek := 0

	subdirs, _ := os.ReadDir(root)
	for _, sd := range subdirs {
		if !sd.IsDir() {
			continue
		}
		kind := singularizeKind(sd.Name())
		kindDir := filepath.Join(root, sd.Name())
		files, _ := os.ReadDir(kindDir)
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			full := filepath.Join(kindDir, f.Name())
			st, err := os.Stat(full)
			if err != nil {
				continue
			}
			slug := strings.TrimSuffix(f.Name(), ".md")
			entry := CorpusEntry{
				Kind:          kind,
				Slug:          slug,
				Title:         humanize(slug),
				Description:   "",
				Domain:        sd.Name(),
				UpdatedAt:     st.ModTime(),
				UpdatedPretty: prettyAge(st.ModTime()),
			}
			// Best-effort: pull the first heading or first non-empty line as title.
			if t, desc := readTitleAndDesc(full); t != "" {
				entry.Title = t
				if desc != "" {
					entry.Description = desc
				}
			}
			entries = append(entries, entry)
			if st.ModTime().After(weekAgo) {
				newThisWeek++
			}
		}
	}

	// Most-recent first.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt.After(entries[j].UpdatedAt)
	})

	out.TotalEntries = len(entries)
	out.NewThisWeek = newThisWeek

	if len(entries) > in.Limit {
		entries = entries[:in.Limit]
	}
	out.Entries = entries
	return out
}

// singularizeKind drops trailing 's' on the directory name so
// "conventions" → "convention". Falls through unchanged when nothing
// to strip — keeps "notes" mapped to "note" but leaves things like
// "context" alone (no trailing s).
func singularizeKind(dir string) string {
	dir = strings.ToLower(dir)
	switch dir {
	case "conventions":
		return "convention"
	case "decisions":
		return "decision"
	case "learnings":
		return "learning"
	case "notes":
		return "note"
	case "rules":
		return "rule"
	case "patterns":
		return "pattern"
	}
	return dir
}

// readTitleAndDesc returns (title, one-line-description) from a knowledge
// markdown file. Looks for the first "# Heading" or the `title:`
// frontmatter line, and falls back to a humanized slug.
func readTitleAndDesc(path string) (string, string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	lines := strings.Split(string(b), "\n")
	title := ""
	desc := ""
	inFrontmatter := false
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if i == 0 && line == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if line == "---" {
				inFrontmatter = false
				continue
			}
			if strings.HasPrefix(line, "title:") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
				title = strings.Trim(title, `"'`)
			}
			continue
		}
		if title == "" && strings.HasPrefix(line, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}
		if title != "" && desc == "" && line != "" && !strings.HasPrefix(line, "#") {
			// First non-heading paragraph line becomes the description.
			desc = line
			break
		}
	}
	if len(desc) > 140 {
		desc = desc[:137] + "…"
	}
	return title, desc
}

// humanize turns "agent-trust-tiers" into "Agent trust tiers".
func humanize(slug string) string {
	if slug == "" {
		return ""
	}
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if i == 0 && len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
