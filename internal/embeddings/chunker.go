package embeddings

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/spec"
)

// ChunkResult holds extracted chunks for a single corpus.
type ChunkResult struct {
	Chunks []TextChunk
	Corpus string
}

// TextChunk is a pre-embedding chunk with its text and metadata.
type TextChunk struct {
	ID       string // corpus:source_id[:section]
	Corpus   string
	SourceID string
	Section  string
	Text     string
}

// ChunkSpecs extracts chunks from all specs in the hero directory.
// Each spec section (## Goal, ## Problem, ## Design, etc.) becomes one chunk.
// Specs with no parseable sections get one chunk for the full body.
func ChunkSpecs(heroDir string) ([]TextChunk, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	var chunks []TextChunk
	for _, s := range specs {
		// Skip conventions — they have their own corpus.
		if s.Type == spec.TypeConvention {
			continue
		}

		prefix := specMetadataPrefix(s)

		if len(s.Sections) == 0 {
			// No sections: single chunk with the full body.
			body := bodyFromRawContent(s.RawContent)
			if body == "" {
				continue
			}
			chunks = append(chunks, TextChunk{
				ID:       fmt.Sprintf("spec:%s:", s.Slug),
				Corpus:   "spec",
				SourceID: s.Slug,
				Section:  "",
				Text:     prefix + body,
			})
			continue
		}

		for section, content := range s.Sections {
			if strings.TrimSpace(content) == "" {
				continue
			}
			chunks = append(chunks, TextChunk{
				ID:       fmt.Sprintf("spec:%s:%s", s.Slug, section),
				Corpus:   "spec",
				SourceID: s.Slug,
				Section:  section,
				Text:     prefix + content,
			})
		}
	}

	return chunks, nil
}

// ChunkKnowledge extracts chunks from .hero/knowledge/**/*.md files.
// One chunk per file (knowledge files are already written tight).
func ChunkKnowledge(heroDir string) ([]TextChunk, error) {
	knowledgeDir := filepath.Join(heroDir, "knowledge")
	if _, err := os.Stat(knowledgeDir); os.IsNotExist(err) {
		return nil, nil
	}

	var chunks []TextChunk
	err := filepath.Walk(knowledgeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		// Skip spec.md files — those are convention specs handled by ChunkConventions.
		if info.Name() == "spec.md" {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // skip unreadable files
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return nil
		}

		relPath, _ := filepath.Rel(knowledgeDir, path)
		chunkID := fmt.Sprintf("knowledge:%s", relPath)

		// Include filename in text prefix for better embedding quality.
		baseName := strings.TrimSuffix(info.Name(), ".md")
		prefixedText := fmt.Sprintf("Knowledge: %s\n\n%s", baseName, text)

		chunks = append(chunks, TextChunk{
			ID:       chunkID,
			Corpus:   "knowledge",
			SourceID: relPath,
			Section:  "",
			Text:     prefixedText,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking knowledge dir: %w", err)
	}

	return chunks, nil
}

// ChunkConventions extracts chunks from convention specs.
// One chunk per convention file.
func ChunkConventions(heroDir string) ([]TextChunk, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	var chunks []TextChunk
	for _, s := range specs {
		if s.Type != spec.TypeConvention {
			continue
		}

		body := bodyFromRawContent(s.RawContent)
		if body == "" {
			continue
		}

		text := fmt.Sprintf("Convention: %s\n\n%s", s.Title, body)

		chunks = append(chunks, TextChunk{
			ID:       fmt.Sprintf("convention:%s", s.Slug),
			Corpus:   "convention",
			SourceID: s.Slug,
			Section:  "",
			Text:     text,
		})
	}

	return chunks, nil
}

// ChunkEvents extracts chunks from graph events (UserAsk, SessionReflection, NextSuggestion).
// One chunk per event. Requires the graph.db to be open.
func ChunkEvents(graphDB *sql.DB) ([]TextChunk, error) {
	if graphDB == nil {
		return nil, nil
	}

	rows, err := graphDB.Query(`
		SELECT id, type, key,
		       COALESCE(json_extract(props, '$.body'), '') AS body,
		       COALESCE(json_extract(props, '$.subject'), '') AS subject
		FROM nodes
		WHERE valid_to IS NULL
		  AND type IN ('UserAsk', 'SessionReflection', 'NextSuggestion')
		ORDER BY ingested_at DESC
		LIMIT 1000
	`)
	if err != nil {
		return nil, fmt.Errorf("querying events: %w", err)
	}
	defer rows.Close()

	var chunks []TextChunk
	for rows.Next() {
		var id int64
		var typ, key, body, subject string
		if err := rows.Scan(&id, &typ, &key, &body, &subject); err != nil {
			return nil, fmt.Errorf("scanning event row: %w", err)
		}

		text := strings.TrimSpace(subject + "\n" + body)
		if text == "" {
			continue
		}

		chunkID := fmt.Sprintf("event:%d", id)
		chunks = append(chunks, TextChunk{
			ID:       chunkID,
			Corpus:   "event",
			SourceID: fmt.Sprintf("%d", id),
			Section:  "",
			Text:     text,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating events: %w", err)
	}

	return chunks, nil
}

// ChunkCodeSymbols extracts chunks from codescan output in the graph.
// One chunk per function/type symbol with signature + docstring.
// Returns nil if no code symbols exist (gated on master-ingest-restore).
func ChunkCodeSymbols(graphDB *sql.DB) ([]TextChunk, error) {
	if graphDB == nil {
		return nil, nil
	}

	rows, err := graphDB.Query(`
		SELECT id, key,
		       COALESCE(json_extract(props, '$.signature'), '') AS signature,
		       COALESCE(json_extract(props, '$.doc_comment'), '') AS doc,
		       COALESCE(json_extract(props, '$.body'), '') AS body,
		       COALESCE(json_extract(props, '$.kind'), '') AS kind,
		       COALESCE(json_extract(props, '$.path'), '') AS path
		FROM nodes
		WHERE valid_to IS NULL
		  AND type = 'Symbol'
	`)
	if err != nil {
		return nil, fmt.Errorf("querying code symbols: %w", err)
	}
	defer rows.Close()

	var chunks []TextChunk
	for rows.Next() {
		var id int64
		var key, signature, doc, body, kind, path string
		if err := rows.Scan(&id, &key, &signature, &doc, &body, &kind, &path); err != nil {
			return nil, fmt.Errorf("scanning symbol row: %w", err)
		}

		text := formatCodeSymbol(key, kind, signature, doc, body, path)
		if strings.TrimSpace(text) == "" {
			continue
		}

		chunkID := fmt.Sprintf("code:%s", key)
		chunks = append(chunks, TextChunk{
			ID:       chunkID,
			Corpus:   "code",
			SourceID: key,
			Section:  "",
			Text:     text,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating symbols: %w", err)
	}

	return chunks, nil
}

// specMetadataPrefix builds a metadata prefix for spec chunks to improve
// embedding quality. Includes title, type, status, and tags.
func specMetadataPrefix(s *spec.Spec) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Title: %s\n", s.Title))
	b.WriteString(fmt.Sprintf("Type: %s | Status: %s\n", s.Type, s.Status))
	if len(s.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(s.Tags, ", ")))
	}
	b.WriteString("\n")
	return b.String()
}

// bodyFromRawContent strips YAML frontmatter from raw content and returns
// the body. If there is no frontmatter, returns the content as-is.
func bodyFromRawContent(content string) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 {
		return ""
	}

	if strings.TrimSpace(lines[0]) != "---" {
		return strings.TrimSpace(content)
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			body := strings.Join(lines[i+1:], "")
			return strings.TrimSpace(body)
		}
	}

	return strings.TrimSpace(content)
}

// formatCodeSymbol builds the text representation of a code symbol for
// embedding. Includes package, file, kind, signature, doc comment, and
// a truncated body (first 500 chars).
func formatCodeSymbol(key, kind, signature, doc, body, path string) string {
	var b strings.Builder

	pkg := packageFromPath(path)
	if pkg != "" {
		b.WriteString(fmt.Sprintf("// Package: %s\n", pkg))
	}
	if path != "" {
		b.WriteString(fmt.Sprintf("// File: %s\n", path))
	}
	if kind != "" {
		b.WriteString(fmt.Sprintf("// %s: %s\n", kind, key))
	}
	if signature != "" {
		b.WriteString(fmt.Sprintf("// Signature: %s\n", signature))
	}
	if doc != "" {
		b.WriteString(fmt.Sprintf("// %s\n", doc))
	}

	if body != "" {
		truncated := body
		if len(truncated) > 500 {
			truncated = truncated[:500]
		}
		b.WriteString(truncated)
	}

	return b.String()
}

// packageFromPath extracts the Go package name from a file path.
// For "internal/embeddings/chunker.go", returns "embeddings".
func packageFromPath(path string) string {
	if path == "" {
		return ""
	}
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}
