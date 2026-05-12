package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var specSplitCmd = &cobra.Command{
	Use:   "split <slug>",
	Short: "Convert a single-file spec.md to three-file layout (requirements.md + design.md + tasks.md)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecSplit,
}

var specJoinCmd = &cobra.Command{
	Use:   "join <slug>",
	Short: "Convert a three-file layout back to a single spec.md",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpecJoin,
}

func init() {
	specCmd.AddCommand(specSplitCmd)
	specCmd.AddCommand(specJoinCmd)
}

// Section assignment for three-file split:
// requirements.md: frontmatter + goal, problem, background, acceptance criteria, boundaries
// design.md: design, architecture, approach
// tasks.md: changes, tasks, implementation

var requirementsSections = map[string]bool{
	"goal": true, "problem": true, "background": true,
	"acceptance criteria": true, "boundaries": true,
	"non-goals": true, "value assessment": true,
}

var designSections = map[string]bool{
	"design": true, "architecture": true, "approach": true,
	"proposed solution": true, "architecture overview": true,
}

// Everything else goes to tasks.md (changes, implementation details, etc.)

func runSpecSplit(cmd *cobra.Command, args []string) error {
	slug := args[0]
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	s := findSpec(slug, specs)
	if s == nil {
		return fmt.Errorf("spec %q not found", slug)
	}

	if s.ThreeFile {
		return fmt.Errorf("spec %q is already in three-file layout", slug)
	}

	dir := filepath.Dir(s.Path)

	// Split the raw content into frontmatter + sections
	content := s.RawContent
	frontmatter, body := splitFrontmatter(content)

	// Parse body into ordered sections
	sections := extractOrderedSections(body)

	// Build three files
	var reqParts, designParts, taskParts []string

	// Requirements always gets the frontmatter
	if frontmatter != "" {
		reqParts = append(reqParts, frontmatter)
	}

	for _, sec := range sections {
		lowerName := strings.ToLower(sec.name)
		if requirementsSections[lowerName] {
			reqParts = append(reqParts, sec.content)
		} else if designSections[lowerName] {
			designParts = append(designParts, sec.content)
		} else {
			taskParts = append(taskParts, sec.content)
		}
	}

	// Write the three files
	reqContent := strings.Join(reqParts, "\n\n")
	designContent := strings.Join(designParts, "\n\n")
	taskContent := strings.Join(taskParts, "\n\n")

	if err := os.WriteFile(filepath.Join(dir, "requirements.md"), []byte(strings.TrimSpace(reqContent)+"\n"), 0o644); err != nil {
		return err
	}
	if len(designParts) > 0 {
		if err := os.WriteFile(filepath.Join(dir, "design.md"), []byte(strings.TrimSpace(designContent)+"\n"), 0o644); err != nil {
			return err
		}
	}
	if len(taskParts) > 0 {
		if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(strings.TrimSpace(taskContent)+"\n"), 0o644); err != nil {
			return err
		}
	}

	// Remove original spec.md
	if err := os.Remove(s.Path); err != nil {
		return fmt.Errorf("removing spec.md: %w", err)
	}

	fmt.Printf("Split %s into three-file layout:\n", slug)
	fmt.Printf("  %s/requirements.md\n", dir)
	if len(designParts) > 0 {
		fmt.Printf("  %s/design.md\n", dir)
	}
	if len(taskParts) > 0 {
		fmt.Printf("  %s/tasks.md\n", dir)
	}
	return nil
}

func runSpecJoin(cmd *cobra.Command, args []string) error {
	slug := args[0]
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	s := findSpec(slug, specs)
	if s == nil {
		return fmt.Errorf("spec %q not found", slug)
	}

	if !s.ThreeFile {
		return fmt.Errorf("spec %q is already a single-file spec", slug)
	}

	dir := filepath.Dir(s.Path)

	// Read all three files and concatenate
	var parts []string
	for _, name := range []string{"requirements.md", "design.md", "tasks.md"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading %s: %w", name, err)
		}
		parts = append(parts, strings.TrimSpace(string(data)))
	}

	joined := strings.Join(parts, "\n\n") + "\n"

	// Write spec.md
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(joined), 0o644); err != nil {
		return err
	}

	// Remove the three files
	for _, name := range []string{"requirements.md", "design.md", "tasks.md"} {
		os.Remove(filepath.Join(dir, name))
	}

	fmt.Printf("Joined %s back to single-file spec.md\n", slug)
	return nil
}

// splitFrontmatter separates YAML frontmatter from body content.
func splitFrontmatter(content string) (frontmatter, body string) {
	if !strings.HasPrefix(content, "---") {
		return "", content
	}
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", content
	}
	fm := content[:idx+3+4] // include opening --- through closing ---
	after := rest[idx+4:]
	if strings.HasPrefix(after, "\n") {
		after = after[1:]
	}
	return strings.TrimSpace(fm), after
}

type orderedSection struct {
	name    string
	content string
}

// extractOrderedSections splits body into sections keyed by H1/H2 headings.
func extractOrderedSections(body string) []orderedSection {
	var sections []orderedSection
	var current strings.Builder
	var currentName string

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
			// Save previous section
			if currentName != "" || current.Len() > 0 {
				sections = append(sections, orderedSection{
					name:    currentName,
					content: current.String(),
				})
			}
			currentName = strings.TrimLeft(trimmed, "# ")
			current.Reset()
			current.WriteString(line)
			current.WriteString("\n")
		} else {
			current.WriteString(line)
			current.WriteString("\n")
		}
	}
	if currentName != "" || current.Len() > 0 {
		sections = append(sections, orderedSection{
			name:    currentName,
			content: current.String(),
		})
	}

	return sections
}
