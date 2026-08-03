package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/install"
	"github.com/hero-engine/hero/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	noteFrom string
)

var noteCmd = &cobra.Command{
	Use:   "note [slug] [inline text...]",
	Short: "Capture a quick note in the knowledge base",
	Long: `Creates a note in the knowledge base with minimal friction.

Notes are for brainstorms, conversation captures, stream-of-consciousness
thinking, and anything that isn't ready to be a spec yet.

Usage:
  hero note auth-brainstorm                    # create empty note with slug
  hero note "thinking about auth flow"         # inline text becomes title and content
  hero note auth-ideas --from conversation.md  # import content from a file
  cat convo.txt | hero note piped-thoughts     # pipe content via stdin

If no slug is given, one is auto-generated from the current date and time.`,
	RunE: runNote,
}

func init() {
	noteCmd.Flags().StringVar(&noteFrom, "from", "", "import note content from a file")
}

func runNote(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	// Determine slug and inline content
	var slug string
	var inlineText string

	if len(args) == 0 {
		// Auto-generate slug from timestamp
		slug = now.Format("2006-01-02-1504")
	} else if len(args) == 1 {
		// Could be a slug or a quoted sentence
		arg := args[0]
		if strings.Contains(arg, " ") {
			// It's inline text — derive slug from it
			slug = textToSlug(arg)
			inlineText = arg
		} else {
			slug = arg
		}
	} else {
		// Multiple args — join as inline text
		inlineText = strings.Join(args, " ")
		slug = textToSlug(inlineText)
	}

	// Validate slug
	if slug == "" || strings.Contains(slug, "/") {
		return fmt.Errorf("invalid slug %q — use lowercase-kebab-case without slashes", slug)
	}

	notesDir := cfg.NotesDir(projectRoot)
	targetDir := filepath.Join(notesDir, slug)
	specPath := filepath.Join(targetDir, "spec.md")

	// Check for collision
	if _, err := os.Stat(specPath); err == nil {
		return fmt.Errorf("note already exists: %s", specPath)
	}

	// Determine content body
	var body string

	if noteFrom != "" {
		// Read from file
		data, err := os.ReadFile(noteFrom)
		if err != nil {
			return fmt.Errorf("reading --from file: %w", err)
		}
		body = string(data)
	} else {
		// Read from stdin when it is not a terminal.
		//
		// This replaces note.go's own hasPipedInput() helper, which asked
		// "is stdin NOT a character device". Polarity is the obvious hazard
		// (hasPipedInput is the negation of install.go's isTerminal, built on
		// the same syscall), but the subtler one is /dev/null: it IS a
		// character device, so the old helper answered "no piped input" for
		// it and fell through to the inline text. term.IsTerminal answers
		// "not a terminal" for /dev/null, so a bare inversion would read it
		// and produce an empty note, silently discarding text the user
		// passed on the command line.
		//
		// The empty-body fallback below preserves the old outcome for that
		// case without reintroducing a second TTY predicate.
		in := cmd.InOrStdin()
		if !prompt.IsInputTTY(in) {
			data, err := io.ReadAll(in)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			body = string(data)
		}
		if body == "" {
			body = inlineText
		}
	}

	// Build title
	title := slugToTitle(slug)
	if inlineText != "" {
		title = inlineText
		// Truncate long titles
		if len(title) > 80 {
			title = title[:77] + "..."
		}
	}

	// Create directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Resolve active scope from cwd (satellite or subfolder of root).
	scope := resolveActiveScope(projectRoot, heroDir)

	// Generate note content
	content := generateNoteContent(title, slug, date, body, scope)

	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing note: %w", err)
	}

	if scope != "" {
		fmt.Printf("Created note: %s (scope: %s)\n", specPath, scope)
	} else {
		fmt.Printf("Created note: %s\n", specPath)
	}
	return nil
}

// resolveActiveScope reads the active subproject scope for the current
// cwd, given the workspace root. Returns "" (root scope) when cwd is
// at root or under no declared subproject.
func resolveActiveScope(projectRoot, heroDir string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	ws, err := workspace.Locate(cwd)
	if err != nil {
		return ""
	}
	subs, _ := install.LoadSubprojects(heroDir)
	var declared []string
	if subs != nil {
		declared = subs.DeclaredPaths()
	}
	return ws.Scope(declared)
}

// generateNoteContent creates the note markdown with minimal frontmatter.
// If scope is non-empty, it is stamped into a `subproject:` field so the
// note carries its monorepo scope through to the graph and surfaces.
func generateNoteContent(title, slug, date, body, scope string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %s\n", title))
	sb.WriteString(fmt.Sprintf("slug: %s\n", slug))
	sb.WriteString("type: note\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", date))
	if scope != "" {
		sb.WriteString(fmt.Sprintf("subproject: %s\n", scope))
	}
	sb.WriteString("tags: []\n")
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("# %s\n", title))

	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("\n<!-- Brainstorm, conversation capture, stream-of-consciousness — no structure required. -->\n")
	}

	return sb.String()
}

// textToSlug converts free-form text to a kebab-case slug.
func textToSlug(text string) string {
	// Lowercase, replace spaces with hyphens, strip non-alphanumeric
	text = strings.ToLower(text)
	var result []byte
	prevHyphen := false
	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result = append(result, byte(c))
			prevHyphen = false
		} else if c == ' ' || c == '-' || c == '_' {
			if !prevHyphen && len(result) > 0 {
				result = append(result, '-')
				prevHyphen = true
			}
		}
		// skip other characters
	}
	// Trim trailing hyphen
	s := string(result)
	s = strings.TrimRight(s, "-")

	// Truncate long slugs
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}

	return s
}
