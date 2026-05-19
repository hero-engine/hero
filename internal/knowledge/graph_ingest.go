// Package knowledge ingests files from .hero/knowledge/ into the
// unified knowledge graph. Today this covers raw/ — the immutable
// audit copy of `hero ingest` source bytes — which become Document
// nodes whose key is the content hash.
//
// notes/ and context/ are already covered by spec ingest, since they
// share the spec.md frontmatter shape.
package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/graph"
)

// WriteRawGraph walks heroDir/knowledge/raw/ and upserts a Document
// node for each file found. Idempotent: unchanged bytes produce the
// same content hash, which keys the node.
//
// Document props: title, source (original URL or path), ingested_at,
// raw_path (file path within the project), byte_count.
//
// repoKey stamps the partition column on each Document. raw bytes
// always live in a single repo's .hero/knowledge/raw/.
func WriteRawGraph(heroDir, repoKey string, store *graph.Store) (*RawGraphSummary, error) {
	rawDir := filepath.Join(heroDir, "knowledge", "raw")
	info, err := os.Stat(rawDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &RawGraphSummary{}, nil
		}
		return nil, fmt.Errorf("stat raw dir: %w", err)
	}
	if !info.IsDir() {
		return &RawGraphSummary{}, nil
	}

	source := map[string]any{"kind": "raw"}
	summary := &RawGraphSummary{}

	err = filepath.WalkDir(rawDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		hash := sha256Hex(bytes)
		fm := parseRawFrontmatter(bytes)
		props := map[string]any{
			"raw_path":   relPath(heroDir, path),
			"byte_count": len(bytes),
		}
		if fm.title != "" {
			props["title"] = fm.title
		}
		if fm.source != "" {
			props["source_url"] = fm.source
		}
		if fm.ingested != "" {
			props["ingested"] = fm.ingested
		}
		if fm.docType != "" {
			props["doc_type"] = fm.docType
		}

		if _, err := store.UpsertNode(&graph.Node{
			Type:        "Document",
			Domain:      "engineering",
			Key:         hash,
			Props:       props,
			Repo:        repoKey,
			ContentHash: hash,
			Source:      source,
		}); err != nil {
			return fmt.Errorf("upsert Document %s: %w", path, err)
		}
		summary.Documents++
		return nil
	})
	if err != nil {
		return nil, err
	}
	return summary, nil
}

// RawGraphSummary reports counts written by WriteRawGraph.
type RawGraphSummary struct {
	Documents int
}

type rawFrontmatter struct {
	source, ingested, title, docType string
}

// parseRawFrontmatter reads the leading YAML frontmatter block written
// by hero ingest and extracts the fields it cares about. It is
// deliberately tolerant — anything malformed yields a zero value for
// the affected field rather than an error, since raw bytes are
// authoritative regardless of metadata quality.
func parseRawFrontmatter(b []byte) rawFrontmatter {
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		return rawFrontmatter{}
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return rawFrontmatter{}
	}
	body := s[4 : 4+end]
	out := rawFrontmatter{}
	for _, line := range strings.Split(body, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "source":
			out.source = v
		case "ingested":
			out.ingested = v
		case "title":
			out.title = v
		case "type":
			out.docType = v
		}
	}
	return out
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func relPath(heroDir, path string) string {
	if rel, err := filepath.Rel(filepath.Dir(heroDir), path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}
