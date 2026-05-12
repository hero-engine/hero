package cli

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <url-or-file> [title]",
	Short: "Ingest external content into the knowledge base",
	Long: `Read a URL, file, or directory and add it to the knowledge base.
The raw content is stored in .hero/knowledge/raw/ and a stub knowledge
entry is created for the agent to enrich.

Examples:
  hero import https://docs.example.com/api-reference "API Reference"
  hero import ./docs/architecture.md "Architecture Overview"
  hero import ./vendor-docs/ "Vendor Documentation"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runImport,
}

var (
	importTag  string
	importType string
)

func init() {
	importCmd.Flags().StringVar(&importTag, "tag", "", "tag for the knowledge entry")
	importCmd.Flags().StringVar(&importType, "type", "context", "knowledge type (context, convention, decision)")
}

func runImport(cmd *cobra.Command, args []string) error {
	source := args[0]
	title := ""
	if len(args) > 1 {
		title = strings.Join(args[1:], " ")
	}

	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	rawDir := filepath.Join(heroDir, "knowledge", "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return fmt.Errorf("creating raw dir: %w", err)
	}

	// Determine if source is URL or file
	var content []byte
	var sourceName string

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		content, sourceName, err = fetchURL(source)
		if err != nil {
			return fmt.Errorf("fetching URL: %w", err)
		}
		if title == "" {
			title = sourceName
		}
	} else {
		content, err = os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		if title == "" {
			title = filepath.Base(source)
		}
		sourceName = source
	}

	// Generate slug from title
	slug := slugify(title)

	// Store raw content
	rawPath := filepath.Join(rawDir, slug+".md")
	rawHeader := fmt.Sprintf("---\nsource: %s\ningested: %s\ntitle: %s\n---\n\n",
		sourceName, time.Now().UTC().Format(time.RFC3339), title)
	if err := os.WriteFile(rawPath, []byte(rawHeader+string(content)), 0o644); err != nil {
		return fmt.Errorf("writing raw: %w", err)
	}
	fmt.Printf("Raw stored: %s (%d bytes)\n", rawPath, len(content))

	// Create knowledge entry stub
	entryDir := filepath.Join(heroDir, "knowledge", importType, slug)
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		return fmt.Errorf("creating entry dir: %w", err)
	}

	entryPath := filepath.Join(entryDir, "spec.md")
	if _, err := os.Stat(entryPath); err == nil {
		fmt.Printf("Knowledge entry already exists: %s (skipping)\n", entryPath)
		return nil
	}

	tags := ""
	if importTag != "" {
		tags = fmt.Sprintf("tags: [%s, ingested]\n", importTag)
	} else {
		tags = "tags: [ingested]\n"
	}

	entry := fmt.Sprintf(`---
title: %s
type: %s
status: active
%screated: %s
source: %s
raw_path: %s
---

# %s

> Ingested from: %s
> Raw content: %s

## Summary

<!-- This entry was auto-created by hero ingest. The agent should read
the raw content at %s and write a proper summary here. -->

_Pending enrichment — run hero ask or have the agent summarize the raw content._

## Key Points

- _To be filled by agent_

## Relevance

- _How this relates to the project — to be filled by agent_
`,
		title, importType, tags,
		time.Now().Format("2006-01-02"),
		sourceName, rawPath,
		title, sourceName, rawPath, rawPath)

	if err := os.WriteFile(entryPath, []byte(entry), 0o644); err != nil {
		return fmt.Errorf("writing entry: %w", err)
	}

	fmt.Printf("Knowledge entry created: %s\n", entryPath)
	fmt.Printf("Run 'hero index' to make it searchable.\n")
	fmt.Printf("The agent will enrich this entry on next session.\n")

	return nil
}

func fetchURL(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Try to extract title from HTML
	title := url
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		if m := regexp.MustCompile(`<title>([^<]+)</title>`).FindSubmatch(body); len(m) > 1 {
			title = strings.TrimSpace(string(m[1]))
		}
	}

	return body, title, nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	if s == "" {
		s = fmt.Sprintf("ingest-%x", sha256.Sum256([]byte(time.Now().String())))[:16]
	}
	return s
}
