// Package wiki syncs completed specs to documentation targets such as
// GitHub Wiki. It reads spec files and pushes them as wiki pages using
// the target's API. Currently only GitHub Wiki (via the git-based API)
// is supported.
package wiki

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
)

// Syncer pushes specs to a wiki target.
type Syncer interface {
	// SyncSpec pushes a single spec to the wiki. Returns the page name that was written.
	SyncSpec(s *spec.Spec, content string) (string, error)

	// SyncAll pushes all provided specs to the wiki. Returns a list of page names written.
	SyncAll(specs []*spec.Spec) ([]string, error)

	// Name returns the wiki target name (e.g. "github-wiki").
	Name() string
}

// New creates a Syncer from config. Returns an error if the sync target
// is unrecognized or required configuration is missing.
func New(cfg *config.SyncConfig, trackerCfg *config.TrackerConfig) (Syncer, error) {
	if cfg == nil || cfg.Target == "" || cfg.Target == "none" {
		return nil, fmt.Errorf("no wiki sync target configured (target is %q)", safeTarget(cfg))
	}

	switch cfg.Target {
	case "github-wiki":
		if trackerCfg == nil || trackerCfg.Project == "" {
			return nil, fmt.Errorf("github-wiki sync requires tracker.project to be set (owner/repo)")
		}
		token, err := resolveToken(trackerCfg.TokenEnv)
		if err != nil {
			return nil, fmt.Errorf("github-wiki sync requires a token: %w", err)
		}
		return newGitHubWiki(trackerCfg.Project, token)
	case "confluence":
		return nil, fmt.Errorf("confluence sync requires a ConfluenceConfig — use wiki.NewConfluence(cfg.Confluence)")
	default:
		return nil, fmt.Errorf("unknown wiki sync target: %q", cfg.Target)
	}
}

// NewConfluence creates a Confluence Syncer from ConfluenceConfig.
func NewConfluence(cfg *config.ConfluenceConfig) (Syncer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("confluence config is required")
	}
	return newConfluenceWiki(cfg)
}

func safeTarget(cfg *config.SyncConfig) string {
	if cfg == nil {
		return "nil"
	}
	return cfg.Target
}

func resolveToken(envVar string) (string, error) {
	if envVar == "" {
		return "", fmt.Errorf("token_env is not configured")
	}
	token := os.Getenv(envVar)
	if token == "" {
		return "", fmt.Errorf("environment variable %q is not set", envVar)
	}
	return token, nil
}

// gitHubWiki syncs specs to a GitHub Wiki by cloning the wiki repo,
// writing markdown files, and pushing. GitHub wikis are backed by a
// git repo at <repo>.wiki.git.
type gitHubWiki struct {
	owner string
	repo  string
	token string
}

func newGitHubWiki(project, token string) (*gitHubWiki, error) {
	parts := strings.SplitN(project, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("github project must be in owner/repo format, got %q", project)
	}
	return &gitHubWiki{owner: parts[0], repo: parts[1], token: token}, nil
}

func (g *gitHubWiki) Name() string { return "github-wiki" }

// SyncSpec pushes a single spec to the wiki.
func (g *gitHubWiki) SyncSpec(s *spec.Spec, content string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "hero-wiki-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := g.cloneWiki(tmpDir); err != nil {
		return "", err
	}

	pageName := specToPageName(s)
	pageContent := specToWikiPage(s, content)

	pagePath := filepath.Join(tmpDir, pageName+".md")
	if err := os.WriteFile(pagePath, []byte(pageContent), 0o644); err != nil {
		return "", fmt.Errorf("writing wiki page: %w", err)
	}

	if err := g.commitAndPush(tmpDir, fmt.Sprintf("hero: sync %s", s.Slug)); err != nil {
		return "", err
	}

	return pageName, nil
}

// SyncAll pushes all specs to the wiki in a single commit.
func (g *gitHubWiki) SyncAll(specs []*spec.Spec) ([]string, error) {
	tmpDir, err := os.MkdirTemp("", "hero-wiki-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := g.cloneWiki(tmpDir); err != nil {
		return nil, err
	}

	var pages []string
	for _, s := range specs {
		content, err := os.ReadFile(s.Path)
		if err != nil {
			continue // skip unreadable specs
		}

		pageName := specToPageName(s)
		pageContent := specToWikiPage(s, string(content))

		pagePath := filepath.Join(tmpDir, pageName+".md")
		if err := os.WriteFile(pagePath, []byte(pageContent), 0o644); err != nil {
			return nil, fmt.Errorf("writing wiki page %s: %w", pageName, err)
		}
		pages = append(pages, pageName)
	}

	if len(pages) == 0 {
		return nil, nil
	}

	if err := g.commitAndPush(tmpDir, fmt.Sprintf("hero: sync %d specs", len(pages))); err != nil {
		return nil, err
	}

	return pages, nil
}

func (g *gitHubWiki) cloneWiki(dir string) error {
	wikiURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.wiki.git", g.token, g.owner, g.repo)

	cmd := exec.Command("git", "clone", "--depth", "1", wikiURL, ".")
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloning wiki repo: %w (ensure the wiki is initialized on GitHub)", err)
	}
	return nil
}

func (g *gitHubWiki) commitAndPush(dir, message string) error {
	// Check for changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = dir
	out, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return nil // nothing to commit
	}

	// Add all changes
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = dir
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// Commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = dir
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Push
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = dir
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

// specToPageName converts a spec to a wiki page name.
// Uses the pattern: Type-Slug (e.g. "Feature-csv-export", "Bug-login-timeout").
func specToPageName(s *spec.Spec) string {
	typeName := titleCase(string(s.Type))
	return typeName + "-" + s.Slug
}

// titleCase uppercases the first letter of a string.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// specToWikiPage wraps the spec content with wiki metadata.
func specToWikiPage(s *spec.Spec, rawContent string) string {
	var sb strings.Builder

	// Strip frontmatter from content — wiki pages don't need it
	body := stripFrontmatter(rawContent)

	// Add wiki header
	sb.WriteString(fmt.Sprintf("<!-- hero:managed spec=%s -->\n", s.Slug))
	sb.WriteString(fmt.Sprintf("**Type:** %s | **Status:** %s", s.Type, s.Status))
	if s.ClaimedBy != "" {
		sb.WriteString(fmt.Sprintf(" | **Assigned:** %s", s.ClaimedBy))
	}
	if len(s.Tags) > 0 {
		sb.WriteString(fmt.Sprintf(" | **Tags:** %s", strings.Join(s.Tags, ", ")))
	}
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(body)
	sb.WriteString("\n\n---\n*Synced by [Hero](https://github.com/hero-engine/hero)*\n")

	return sb.String()
}

// stripFrontmatter removes YAML frontmatter from markdown content.
func stripFrontmatter(content string) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 {
		return content
	}

	first := strings.TrimSpace(lines[0])
	if first != "---" {
		return content
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "")
		}
	}

	return content
}
