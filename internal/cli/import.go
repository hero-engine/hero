package cli

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
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
	Use:   "import <url-or-file-or-directory> [title]",
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

const defaultMaxIngestFileBytes uint64 = 1 << 20 // 1 MiB

var (
	importTag      string
	importType     string
	importMaxBytes uint64
)

func init() {
	importCmd.Flags().StringVar(&importTag, "tag", "", "tag for the knowledge entry")
	importCmd.Flags().StringVar(&importType, "type", "context", "knowledge type (context, convention, decision)")
	importCmd.Flags().Uint64Var(&importMaxBytes, "max-bytes", defaultMaxIngestFileBytes, "skip files larger than this many bytes during directory ingest")
}

// writeIngestArgs bundles everything writeSingleIngest needs to emit
// one raw file + one stub knowledge entry. Used by single-file, URL,
// and directory-walk branches so the three paths stay identical.
type writeIngestArgs struct {
	slug       string
	title      string
	content    []byte
	sourceName string
	kType      string
	extraTags  []string
	rawDir     string
	heroDir    string
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

	// URL branch
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		content, sourceName, err := fetchURL(source)
		if err != nil {
			return fmt.Errorf("fetching URL: %w", err)
		}
		if title == "" {
			title = sourceName
		}
		_, err = writeSingleIngest(writeIngestArgs{
			slug:       slugify(title),
			title:      title,
			content:    content,
			sourceName: sourceName,
			kType:      importType,
			extraTags:  []string{importTag},
			rawDir:     rawDir,
			heroDir:    heroDir,
		})
		return err
	}

	// File or directory branch
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return ingestDirectory(source, title, importType, importTag, heroDir, rawDir)
	}

	content, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	if title == "" {
		title = filepath.Base(source)
	}
	_, err = writeSingleIngest(writeIngestArgs{
		slug:       slugify(title),
		title:      title,
		content:    content,
		sourceName: source,
		kType:      importType,
		extraTags:  []string{importTag},
		rawDir:     rawDir,
		heroDir:    heroDir,
	})
	return err
}

// writeSingleIngest writes one raw file under rawDir and one stub
// knowledge entry under heroDir/knowledge/<kType>/<slug>/spec.md.
// Returns (false, nil) when the stub already exists — that is the
// per-entry skip-if-exists check that preserves any enrichment the
// agent has already done. Note slug collisions: two source paths that
// slugify to the same string will silently skip the second; acceptable
// for v1, the skip line tells the user.
func writeSingleIngest(a writeIngestArgs) (bool, error) {
	rawPath := filepath.Join(a.rawDir, a.slug+".md")
	rawHeader := fmt.Sprintf("---\nsource: %s\ningested: %s\ntitle: %s\n---\n\n",
		a.sourceName, time.Now().UTC().Format(time.RFC3339), a.title)
	if err := os.WriteFile(rawPath, []byte(rawHeader+string(a.content)), 0o644); err != nil {
		return false, fmt.Errorf("writing raw: %w", err)
	}
	fmt.Printf("Raw stored: %s (%d bytes)\n", rawPath, len(a.content))

	entryDir := filepath.Join(a.heroDir, "knowledge", a.kType, a.slug)
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		return false, fmt.Errorf("creating entry dir: %w", err)
	}

	entryPath := filepath.Join(entryDir, "spec.md")
	if _, err := os.Stat(entryPath); err == nil {
		fmt.Printf("Knowledge entry already exists: %s (skipping)\n", entryPath)
		return false, nil
	}

	tagList := dedupeTags(append([]string{"ingested"}, a.extraTags...))
	tags := fmt.Sprintf("tags: [%s]\n", strings.Join(tagList, ", "))

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
		a.title, a.kType, tags,
		time.Now().Format("2006-01-02"),
		a.sourceName, rawPath,
		a.title, a.sourceName, rawPath, rawPath)

	if err := os.WriteFile(entryPath, []byte(entry), 0o644); err != nil {
		return false, fmt.Errorf("writing entry: %w", err)
	}

	fmt.Printf("Knowledge entry created: %s\n", entryPath)
	return true, nil
}

// dedupeTags returns a stable-order slice with empties and duplicates removed.
func dedupeTags(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// ingestDirectory walks dir and ingests every text-ish file under it as
// its own knowledge entry. All entries share the groupSlug tag so the
// import group is queryable via hero search --tag.
func ingestDirectory(dir, groupTitle, kType, userTag, heroDir, rawDir string) error {
	if groupTitle == "" {
		groupTitle = filepath.Base(filepath.Clean(dir))
	}
	groupSlug := slugify(groupTitle)

	fmt.Printf("Scanning %s ...\n", dir)

	var ingested, skipped int
	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != dir && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if !isTextExt(d.Name()) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if uint64(info.Size()) > importMaxBytes {
			fmt.Printf("Skipping %s (%d bytes > --max-bytes %d)\n", path, info.Size(), importMaxBytes)
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		fileSlug := slugify(strings.TrimSuffix(rel, filepath.Ext(rel)))
		slug := groupSlug + "-" + fileSlug
		entryTitle := groupTitle + " — " + rel

		wrote, err := writeSingleIngest(writeIngestArgs{
			slug:       slug,
			title:      entryTitle,
			content:    content,
			sourceName: path,
			kType:      kType,
			extraTags:  []string{groupSlug, userTag},
			rawDir:     rawDir,
			heroDir:    heroDir,
		})
		if err != nil {
			return err
		}
		if wrote {
			ingested++
		} else {
			skipped++
		}
		return nil
	})
	if walkErr != nil {
		return walkErr
	}
	if ingested == 0 && skipped == 0 {
		return fmt.Errorf("no ingestable files found under %s — check extension filter and size cap", dir)
	}
	fmt.Printf("Ingested %d files (%d skipped as already-present) from %s under tag %q\n",
		ingested, skipped, dir, groupSlug)
	fmt.Printf("Run 'hero index' to make entries searchable.\n")
	return nil
}

// isTextExt returns true for extensions the directory ingest treats as
// text/markup/structured-data. Binary and lockfile extensions are out.
func isTextExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown", ".txt", ".rst", ".adoc", ".mdx", ".org",
		".json", ".yaml", ".yml":
		return true
	}
	return false
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
