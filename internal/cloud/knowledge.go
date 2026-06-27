package cloud

import (
	"os"
	"path/filepath"
	"strings"
)

type cloudKnowledge struct {
	Category string `json:"category"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Checksum string `json:"checksum"`
}

// discoverKnowledge walks heroDir/knowledge/ and returns entries for all
// .md files found. Handles both standalone .md files and spec.md inside
// subdirectories, with or without YAML frontmatter.
func discoverKnowledge(heroDir string) []cloudKnowledge {
	knowledgeDir := filepath.Join(heroDir, "knowledge")
	if _, err := os.Stat(knowledgeDir); os.IsNotExist(err) {
		return nil
	}

	var entries []cloudKnowledge

	_ = filepath.Walk(knowledgeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		rel, _ := filepath.Rel(knowledgeDir, path)
		category, slug := knowledgeCategoryAndSlug(rel)

		title := extractKnowledgeTitle(string(content))
		if title == "" {
			title = slug
		}

		entries = append(entries, cloudKnowledge{
			Category: category,
			Slug:     slug,
			Title:    title,
			Content:  string(content),
			Checksum: contentChecksum(string(content)),
		})
		return nil
	})

	return entries
}

// knowledgeCategoryAndSlug derives a category and slug from a relative path
// like "conventions/naming.md" or "decisions/auth/spec.md".
func knowledgeCategoryAndSlug(rel string) (string, string) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 1 {
		return "general", strings.TrimSuffix(parts[0], ".md")
	}
	category := parts[0]
	rest := parts[1:]
	if rest[len(rest)-1] == "spec.md" && len(rest) >= 2 {
		return category, rest[len(rest)-2]
	}
	return category, strings.TrimSuffix(strings.Join(rest, "/"), ".md")
}

// extractKnowledgeTitle pulls the title from YAML frontmatter or the first
// markdown heading.
func extractKnowledgeTitle(content string) string {
	lines := strings.SplitN(content, "\n", 30)

	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				break
			}
			if strings.HasPrefix(lines[i], "title:") {
				t := strings.TrimPrefix(lines[i], "title:")
				t = strings.TrimSpace(t)
				t = strings.Trim(t, "\"'")
				return t
			}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}

	return ""
}
