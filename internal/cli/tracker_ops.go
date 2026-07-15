package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment <issue-id> <message>",
	Short: "Post a comment to a tracker issue",
	Long: `Posts a plain-text comment to the specified tracker issue.

The message can contain Markdown formatting (support varies by tracker).
Use - as the message to read from stdin.`,
	Args: cobra.ExactArgs(2),
	RunE: runComment,
}

var attachCmd = &cobra.Command{
	Use:   "attach <issue-id> <file-path> [display-name]",
	Short: "Attach a file to a tracker issue",
	Long: `Uploads a file as an attachment to the specified tracker issue.

For Jira, the file is uploaded as a native attachment.
For GitHub and Linear (which don't support attachments via API),
the file contents are posted as a comment instead.

The optional display-name overrides the filename shown in the tracker.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runAttach,
}

func runComment(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	message := args[1]

	if message == "-" {
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		message = strings.TrimSpace(string(data))
	}

	t, err := initTracker()
	if err != nil {
		return err
	}

	if err := t.AddComment(issueID, message); err != nil {
		return fmt.Errorf("posting comment to %s: %w", issueID, err)
	}

	fmt.Printf("Comment posted to %s\n", issueID)
	return nil
}

func runAttach(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	filePath := args[1]

	displayName := filepath.Base(filePath)
	if len(args) > 2 {
		displayName = args[2]
	} else if displayName == "spec.md" {
		// Derive a better name from the parent directory (slug) and spec title
		displayName = deriveSpecAttachmentName(filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	t, err := initTracker()
	if err != nil {
		return err
	}

	if err := t.AttachFile(issueID, filePath, displayName); err != nil {
		return fmt.Errorf("attaching file to %s: %w", issueID, err)
	}

	fmt.Printf("File %q attached to %s\n", displayName, issueID)
	return nil
}

func initTracker() (tracker.Tracker, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if err := selectSyncIntegration(&cfg); err != nil {
		return nil, err
	}

	if cfg.Tracker == nil || cfg.Tracker.Type == "" || cfg.Tracker.Type == "none" {
		return nil, fmt.Errorf("no tracker configured — set tracker.type in hero.json")
	}

	t, err := tracker.NewWithJiraConfig(cfg.Tracker, cfg.Jira, cfg.TrackerKnowledgeDir(projectRoot))
	if err != nil {
		return nil, fmt.Errorf("initializing tracker: %w", err)
	}
	return t, nil
}

func selectSyncIntegration(cfg *config.Config) error {
	if syncIntegration == "" {
		return nil
	}
	t, err := cfg.SelectTracker(syncIntegration)
	if err != nil {
		return fmt.Errorf("selecting integration: %w", err)
	}
	cfg.Tracker = t
	return nil
}

// deriveSpecAttachmentName creates a descriptive filename from a spec.md path.
// e.g. ".hero/planning/features/cloud-auth/spec.md" → "cloud-auth-spec.md"
// If the spec has a title, uses "slug-title.md" format.
func deriveSpecAttachmentName(filePath string) string {
	dir := filepath.Dir(filePath)
	slug := filepath.Base(dir)
	if slug == "" || slug == "." || slug == "/" {
		return "spec.md"
	}

	// Try to parse the spec for a title
	s, err := spec.ParseFile(filePath)
	if err == nil && s.Title != "" {
		// Sanitize title for filename
		title := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			if r == ' ' {
				return '-'
			}
			return -1
		}, s.Title)
		title = strings.ToLower(title)
		// Trim consecutive hyphens
		for strings.Contains(title, "--") {
			title = strings.ReplaceAll(title, "--", "-")
		}
		title = strings.Trim(title, "-")
		if title != "" && title != slug {
			return slug + "-" + title + ".md"
		}
	}

	return slug + "-spec.md"
}
