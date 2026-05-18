package recap

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// CommitSummary is one git commit.
type CommitSummary struct {
	Hash    string `json:"hash"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// SpecActivity groups commits under one spec.
type SpecActivity struct {
	Slug         string          `json:"slug"`
	Title        string          `json:"title"`
	Type         string          `json:"type"`
	Subproject   string          `json:"subproject,omitempty"`
	OldStatus    string          `json:"old_status,omitempty"`
	NewStatus    string          `json:"new_status"`
	Commits      []CommitSummary `json:"commits"`
	FilesTouched []string        `json:"files_touched"`
}

// KnowledgeEntry is a new or modified knowledge file.
type KnowledgeEntry struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

// Recap is the full activity digest.
type Recap struct {
	Since     time.Time        `json:"since"`
	Until     time.Time        `json:"until"`
	Specs     []SpecActivity   `json:"specs"`
	Knowledge []KnowledgeEntry `json:"knowledge,omitempty"`
	Unmatched []CommitSummary  `json:"unmatched,omitempty"`
}

// Build generates a recap from git history and the spec corpus.
func Build(heroDir, projectRoot string, since time.Time) (*Recap, error) {
	until := time.Now()

	commits, err := gitCommits(projectRoot, since)
	if err != nil {
		return nil, fmt.Errorf("reading git log: %w", err)
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	fileToSpec := buildFileIndex(specs)

	specMap := make(map[string]*SpecActivity)
	var unmatched []CommitSummary
	var knowledge []KnowledgeEntry
	knowledgeSeen := make(map[string]bool)

	for _, c := range commits {
		files := commitFiles(projectRoot, c.Hash)
		matched := false

		for _, f := range files {
			// Check for knowledge entries
			if isKnowledgePath(f) && !knowledgeSeen[f] {
				knowledgeSeen[f] = true
				action := "modified"
				if c.Subject != "" && (strings.HasPrefix(strings.ToLower(c.Subject), "add") || strings.Contains(strings.ToLower(c.Subject), "new")) {
					action = "new"
				}
				knowledge = append(knowledge, KnowledgeEntry{Path: f, Action: action})
			}

			if slug, ok := fileToSpec[f]; ok {
				matched = true
				sa, exists := specMap[slug]
				if !exists {
					s := findSpecBySlug(specs, slug)
					sa = &SpecActivity{
						Slug:       slug,
						NewStatus:  string(s.Status),
						Title:      s.Title,
						Type:       string(s.Type),
						Subproject: s.Subproject,
					}
					specMap[slug] = sa
				}
				if !containsFile(sa.FilesTouched, f) {
					sa.FilesTouched = append(sa.FilesTouched, f)
				}
				if !containsCommit(sa.Commits, c.Hash) {
					sa.Commits = append(sa.Commits, c)
				}
			}
		}

		if !matched && len(files) > 0 {
			unmatched = append(unmatched, c)
		}
	}

	var specActivities []SpecActivity
	for _, sa := range specMap {
		specActivities = append(specActivities, *sa)
	}

	return &Recap{
		Since:     since,
		Until:     until,
		Specs:     specActivities,
		Knowledge: knowledge,
		Unmatched: unmatched,
	}, nil
}

func buildFileIndex(specs []*spec.Spec) map[string]string {
	idx := make(map[string]string)
	for _, s := range specs {
		for _, f := range s.FilesTouched {
			idx[f] = s.Slug
		}
	}
	return idx
}

func findSpecBySlug(specs []*spec.Spec, slug string) *spec.Spec {
	for _, s := range specs {
		if s.Slug == slug {
			return s
		}
	}
	return &spec.Spec{Slug: slug}
}

func gitCommits(projectRoot string, since time.Time) ([]CommitSummary, error) {
	sinceStr := since.Format("2006-01-02T15:04:05")
	cmd := exec.Command("git", "-C", projectRoot, "log",
		"--since="+sinceStr,
		"--pretty=format:%H\t%s\t%an\t%aI",
		"--no-merges")
	// cmd.Stderr stays nil so *exec.ExitError captures stderr for inspection
	// below. If a future change wires cmd.Stderr to a writer, the empty-repo
	// substring check will need an alternate stderr source.
	out, err := cmd.Output()
	if err != nil {
		// A freshly-initialized repo with no commits yet is a legitimate
		// empty result, not an error. git log exits 128 with one of the
		// known stderr messages below; everything else is a real error.
		if ee, ok := err.(*exec.ExitError); ok {
			stderr := string(ee.Stderr)
			if strings.Contains(stderr, "does not have any commits yet") ||
				strings.Contains(stderr, "bad default revision 'HEAD'") {
				return nil, nil
			}
		}
		return nil, err
	}

	var commits []CommitSummary
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, CommitSummary{
			Hash:    parts[0][:8],
			Subject: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
	return commits, nil
}

func commitFiles(projectRoot, hash string) []string {
	cmd := exec.Command("git", "-C", projectRoot, "diff-tree", "--no-commit-id", "-r", "--name-only", hash)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

func isKnowledgePath(f string) bool {
	return strings.Contains(f, ".hero/") && (strings.Contains(f, "/decisions/") ||
		strings.Contains(f, "/conventions/") ||
		strings.Contains(f, "/context/") ||
		strings.Contains(f, "/knowledge/"))
}

func containsFile(files []string, f string) bool {
	for _, x := range files {
		if x == f {
			return true
		}
	}
	return false
}

func containsCommit(commits []CommitSummary, hash string) bool {
	for _, c := range commits {
		if c.Hash == hash {
			return true
		}
	}
	return false
}

// ParseSince parses a --since value: relative duration ("24h", "2d") or ISO date.
func ParseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Now().Add(-24 * time.Hour), nil
	}

	// Try relative: "2d", "48h", "1w"
	if len(s) >= 2 {
		numStr := s[:len(s)-1]
		unit := s[len(s)-1]
		var n int
		if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil {
			switch unit {
			case 'h':
				return time.Now().Add(-time.Duration(n) * time.Hour), nil
			case 'd':
				return time.Now().AddDate(0, 0, -n), nil
			case 'w':
				return time.Now().AddDate(0, 0, -n*7), nil
			}
		}
	}

	// Try ISO date
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse --since %q (use 24h, 2d, or YYYY-MM-DD)", s)
}

// RenderText renders a human-readable recap.
func RenderText(r *Recap) string {
	var sb strings.Builder

	sinceStr := r.Since.Format("2006-01-02 15:04")
	fmt.Fprintf(&sb, "Activity since %s\n\n", sinceStr)

	if len(r.Specs) == 0 && len(r.Knowledge) == 0 && len(r.Unmatched) == 0 {
		sb.WriteString("No activity in this window.\n")
		return sb.String()
	}

	for _, sa := range r.Specs {
		status := sa.NewStatus
		if sa.OldStatus != "" && sa.OldStatus != sa.NewStatus {
			status = sa.OldStatus + " → " + sa.NewStatus
		}
		fmt.Fprintf(&sb, "spec %s (%s):\n", sa.Slug, status)

		subjects := make([]string, 0, len(sa.Commits))
		for _, c := range sa.Commits {
			subjects = append(subjects, c.Subject)
		}
		fmt.Fprintf(&sb, "  %d commits — %s\n", len(sa.Commits), strings.Join(subjects, ", "))

		if len(sa.FilesTouched) > 0 {
			fmt.Fprintf(&sb, "  files: %s\n", strings.Join(sa.FilesTouched, ", "))
		}
		sb.WriteString("\n")
	}

	if len(r.Knowledge) > 0 {
		sb.WriteString("knowledge:\n")
		for _, k := range r.Knowledge {
			fmt.Fprintf(&sb, "  %s: %s\n", k.Action, k.Path)
		}
		sb.WriteString("\n")
	}

	if len(r.Unmatched) > 0 {
		fmt.Fprintf(&sb, "unmatched (%d commits):\n", len(r.Unmatched))
		for _, c := range r.Unmatched {
			fmt.Fprintf(&sb, "  %s\n", c.Subject)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderJSON renders the recap as JSON.
func RenderJSON(r *Recap) (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
