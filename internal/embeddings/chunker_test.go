package embeddings

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// writeSpecFile creates a spec.md at the given path relative to heroDir.
// relDir should be like "planning/features/my-feature".
func writeSpecFile(t *testing.T, heroDir, relDir, content string) string {
	t.Helper()
	dir := filepath.Join(heroDir, relDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating spec dir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing spec file: %v", err)
	}
	return path
}

func TestChunkSpecs_WithSections(t *testing.T) {
	heroDir := t.TempDir()

	writeSpecFile(t, heroDir, "planning/features/auth-flow", `---
title: Auth Flow
slug: auth-flow
type: feature
status: planning
tags: auth, security
---
# Auth Flow

## Goal
Implement OAuth2 login.

## Problem
Users cannot log in with SSO.

## Design
Use the standard OAuth2 code flow.
`)

	chunks, err := ChunkSpecs(heroDir)
	if err != nil {
		t.Fatalf("ChunkSpecs: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks (goal, problem, design), got %d", len(chunks))
	}

	// Build a map for easier assertion.
	bySection := make(map[string]TextChunk)
	for _, c := range chunks {
		bySection[c.Section] = c
	}

	// Verify goal chunk.
	goal, ok := bySection["goal"]
	if !ok {
		t.Fatal("missing 'goal' section chunk")
	}
	if goal.ID != "spec:auth-flow:goal" {
		t.Errorf("goal ID = %q, want %q", goal.ID, "spec:auth-flow:goal")
	}
	if goal.Corpus != "spec" {
		t.Errorf("goal Corpus = %q, want %q", goal.Corpus, "spec")
	}
	if goal.SourceID != "auth-flow" {
		t.Errorf("goal SourceID = %q, want %q", goal.SourceID, "auth-flow")
	}
	if !strings.Contains(goal.Text, "Title: Auth Flow") {
		t.Error("goal text should contain metadata prefix with title")
	}
	if !strings.Contains(goal.Text, "Type: feature | Status: planning") {
		t.Error("goal text should contain type and status")
	}
	if !strings.Contains(goal.Text, "Tags: auth, security") {
		t.Error("goal text should contain tags")
	}
	if !strings.Contains(goal.Text, "Implement OAuth2 login") {
		t.Error("goal text should contain section content")
	}

	// Verify problem chunk.
	problem, ok := bySection["problem"]
	if !ok {
		t.Fatal("missing 'problem' section chunk")
	}
	if problem.ID != "spec:auth-flow:problem" {
		t.Errorf("problem ID = %q, want %q", problem.ID, "spec:auth-flow:problem")
	}

	// Verify design chunk.
	design, ok := bySection["design"]
	if !ok {
		t.Fatal("missing 'design' section chunk")
	}
	if design.ID != "spec:auth-flow:design" {
		t.Errorf("design ID = %q, want %q", design.ID, "spec:auth-flow:design")
	}
}

func TestChunkSpecs_NoSections(t *testing.T) {
	heroDir := t.TempDir()

	writeSpecFile(t, heroDir, "planning/features/simple", `---
title: Simple Feature
slug: simple
type: feature
status: planning
---
Just some body text with no sections.
`)

	chunks, err := ChunkSpecs(heroDir)
	if err != nil {
		t.Fatalf("ChunkSpecs: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for spec with no sections, got %d", len(chunks))
	}

	c := chunks[0]
	if c.ID != "spec:simple:" {
		t.Errorf("ID = %q, want %q", c.ID, "spec:simple:")
	}
	if c.Section != "" {
		t.Errorf("Section = %q, want empty", c.Section)
	}
	if !strings.Contains(c.Text, "Just some body text") {
		t.Error("chunk text should contain body")
	}
}

func TestChunkSpecs_SkipsConventions(t *testing.T) {
	heroDir := t.TempDir()

	// Create a convention spec — should be skipped by ChunkSpecs.
	writeSpecFile(t, heroDir, "knowledge/conventions/error-handling", `---
title: Error Handling Convention
slug: error-handling
type: convention
status: active
---
Always wrap errors with context.
`)

	// Create a feature spec — should be included.
	writeSpecFile(t, heroDir, "planning/features/test-feature", `---
title: Test Feature
slug: test-feature
type: feature
status: planning
---
## Goal
Build something.
`)

	chunks, err := ChunkSpecs(heroDir)
	if err != nil {
		t.Fatalf("ChunkSpecs: %v", err)
	}

	for _, c := range chunks {
		if c.Corpus != "spec" {
			t.Errorf("unexpected corpus %q in ChunkSpecs output", c.Corpus)
		}
		if strings.Contains(c.ID, "error-handling") {
			t.Error("ChunkSpecs should skip convention specs")
		}
	}
}

func TestChunkSpecs_EmptyDir(t *testing.T) {
	heroDir := t.TempDir()

	chunks, err := ChunkSpecs(heroDir)
	if err != nil {
		t.Fatalf("ChunkSpecs: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty dir, got %d", len(chunks))
	}
}

func TestChunkKnowledge(t *testing.T) {
	heroDir := t.TempDir()
	knowledgeDir := filepath.Join(heroDir, "knowledge")

	// Create a knowledge file at root.
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(knowledgeDir, "api-patterns.md"),
		[]byte("# API Patterns\nUse REST for CRUD, gRPC for internal."),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create a nested knowledge file.
	decisionsDir := filepath.Join(knowledgeDir, "decisions")
	if err := os.MkdirAll(decisionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(decisionsDir, "use-sqlite.md"),
		[]byte("# Use SQLite\nSQLite is sufficient for single-machine workloads."),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create a spec.md that should be skipped.
	if err := os.WriteFile(
		filepath.Join(decisionsDir, "spec.md"),
		[]byte("---\ntitle: Some Convention\n---\nShould be skipped."),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	chunks, err := ChunkKnowledge(heroDir)
	if err != nil {
		t.Fatalf("ChunkKnowledge: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	// Build a map by ID for easier assertion.
	byID := make(map[string]TextChunk)
	for _, c := range chunks {
		byID[c.ID] = c
	}

	// Verify root-level knowledge file.
	apiChunk, ok := byID["knowledge:api-patterns.md"]
	if !ok {
		t.Fatal("missing knowledge:api-patterns.md chunk")
	}
	if apiChunk.Corpus != "knowledge" {
		t.Errorf("Corpus = %q, want %q", apiChunk.Corpus, "knowledge")
	}
	if !strings.Contains(apiChunk.Text, "Knowledge: api-patterns") {
		t.Error("chunk text should contain filename prefix")
	}
	if !strings.Contains(apiChunk.Text, "Use REST for CRUD") {
		t.Error("chunk text should contain file content")
	}

	// Verify nested knowledge file.
	sqliteChunk, ok := byID["knowledge:decisions/use-sqlite.md"]
	if !ok {
		t.Fatal("missing knowledge:decisions/use-sqlite.md chunk")
	}
	if sqliteChunk.SourceID != "decisions/use-sqlite.md" {
		t.Errorf("SourceID = %q, want %q", sqliteChunk.SourceID, "decisions/use-sqlite.md")
	}
}

func TestChunkKnowledge_EmptyDir(t *testing.T) {
	heroDir := t.TempDir()
	// No knowledge directory at all.

	chunks, err := ChunkKnowledge(heroDir)
	if err != nil {
		t.Fatalf("ChunkKnowledge: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for missing knowledge dir, got %d", len(chunks))
	}
}

func TestChunkConventions(t *testing.T) {
	heroDir := t.TempDir()

	// Create a convention spec.
	writeSpecFile(t, heroDir, "knowledge/conventions/error-handling", `---
title: Error Handling
slug: error-handling
type: convention
status: active
---
Always wrap errors with fmt.Errorf and %w.
`)

	// Create a non-convention spec — should be excluded.
	writeSpecFile(t, heroDir, "planning/features/login", `---
title: Login
slug: login
type: feature
status: planning
---
## Goal
Build login.
`)

	chunks, err := ChunkConventions(heroDir)
	if err != nil {
		t.Fatalf("ChunkConventions: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 convention chunk, got %d", len(chunks))
	}

	c := chunks[0]
	if c.ID != "convention:error-handling" {
		t.Errorf("ID = %q, want %q", c.ID, "convention:error-handling")
	}
	if c.Corpus != "convention" {
		t.Errorf("Corpus = %q, want %q", c.Corpus, "convention")
	}
	if !strings.Contains(c.Text, "Convention: Error Handling") {
		t.Error("chunk text should contain convention title prefix")
	}
	if !strings.Contains(c.Text, "Always wrap errors") {
		t.Error("chunk text should contain convention body")
	}
}

func TestChunkConventions_EmptyDir(t *testing.T) {
	heroDir := t.TempDir()

	chunks, err := ChunkConventions(heroDir)
	if err != nil {
		t.Fatalf("ChunkConventions: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks, got %d", len(chunks))
	}
}

func TestChunkEvents(t *testing.T) {
	db := setupGraphTestDB(t)

	// Insert some event nodes.
	insertTestEvent(t, db, "UserAsk", "ask-1", "What is Hero?", "A spec-driven tool.")
	insertTestEvent(t, db, "SessionReflection", "refl-1", "Session summary", "We built the search feature.")
	insertTestEvent(t, db, "NextSuggestion", "next-1", "Suggested next", "Improve test coverage.")
	// Insert a non-event node — should be excluded.
	insertTestEvent(t, db, "Spec", "spec-1", "Some spec", "Not an event.")

	chunks, err := ChunkEvents(db)
	if err != nil {
		t.Fatalf("ChunkEvents: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("expected 3 event chunks, got %d", len(chunks))
	}

	for _, c := range chunks {
		if c.Corpus != "event" {
			t.Errorf("Corpus = %q, want %q", c.Corpus, "event")
		}
		if !strings.HasPrefix(c.ID, "event:") {
			t.Errorf("ID = %q, should start with 'event:'", c.ID)
		}
	}
}

func TestChunkEvents_NilDB(t *testing.T) {
	chunks, err := ChunkEvents(nil)
	if err != nil {
		t.Fatalf("ChunkEvents with nil DB: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for nil DB, got %d", len(chunks))
	}
}

func TestChunkCodeSymbols(t *testing.T) {
	db := setupGraphTestDB(t)

	// Insert a code symbol node.
	insertTestSymbol(t, db, "main.Run", "func", "func Run(ctx context.Context)", "Run starts the app.", "func Run(ctx context.Context) { ... }", "cmd/app/main.go")

	chunks, err := ChunkCodeSymbols(db)
	if err != nil {
		t.Fatalf("ChunkCodeSymbols: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("expected 1 code chunk, got %d", len(chunks))
	}

	c := chunks[0]
	if c.ID != "code:main.Run" {
		t.Errorf("ID = %q, want %q", c.ID, "code:main.Run")
	}
	if c.Corpus != "code" {
		t.Errorf("Corpus = %q, want %q", c.Corpus, "code")
	}
	if !strings.Contains(c.Text, "// Package: app") {
		t.Error("chunk text should contain package name")
	}
	if !strings.Contains(c.Text, "// File: cmd/app/main.go") {
		t.Error("chunk text should contain file path")
	}
	if !strings.Contains(c.Text, "// Signature: func Run(ctx context.Context)") {
		t.Error("chunk text should contain signature")
	}
	if !strings.Contains(c.Text, "Run starts the app") {
		t.Error("chunk text should contain doc comment")
	}
}

func TestChunkCodeSymbols_NilDB(t *testing.T) {
	chunks, err := ChunkCodeSymbols(nil)
	if err != nil {
		t.Fatalf("ChunkCodeSymbols with nil DB: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for nil DB, got %d", len(chunks))
	}
}

func TestChunkIDFormats(t *testing.T) {
	heroDir := t.TempDir()

	writeSpecFile(t, heroDir, "planning/features/my-feat", `---
title: My Feature
slug: my-feat
type: feature
status: planning
---
## Goal
Build it.
`)

	// Spec chunks.
	specChunks, err := ChunkSpecs(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range specChunks {
		if !strings.HasPrefix(c.ID, "spec:") {
			t.Errorf("spec chunk ID %q should start with 'spec:'", c.ID)
		}
	}

	// Knowledge chunks.
	knowledgeDir := filepath.Join(heroDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(knowledgeDir, "test.md"), []byte("test content"), 0o644); err != nil {
		t.Fatal(err)
	}
	knowledgeChunks, err := ChunkKnowledge(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range knowledgeChunks {
		if !strings.HasPrefix(c.ID, "knowledge:") {
			t.Errorf("knowledge chunk ID %q should start with 'knowledge:'", c.ID)
		}
	}

	// Convention chunks.
	writeSpecFile(t, heroDir, "knowledge/conventions/my-conv", `---
title: My Convention
slug: my-conv
type: convention
status: active
---
Do the right thing.
`)
	convChunks, err := ChunkConventions(heroDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range convChunks {
		if !strings.HasPrefix(c.ID, "convention:") {
			t.Errorf("convention chunk ID %q should start with 'convention:'", c.ID)
		}
	}
}

func TestBodyFromRawContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "with frontmatter",
			content: "---\ntitle: Test\n---\nBody text here.",
			want:    "Body text here.",
		},
		{
			name:    "without frontmatter",
			content: "Just plain text.",
			want:    "Just plain text.",
		},
		{
			name:    "empty",
			content: "",
			want:    "",
		},
		{
			name:    "frontmatter only",
			content: "---\ntitle: Test\n---\n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bodyFromRawContent(tt.content)
			if got != tt.want {
				t.Errorf("bodyFromRawContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPackageFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"internal/embeddings/chunker.go", "embeddings"},
		{"cmd/app/main.go", "app"},
		{"main.go", "."},
		{"", ""},
	}

	for _, tt := range tests {
		got := packageFromPath(tt.path)
		if got != tt.want {
			t.Errorf("packageFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestFormatCodeSymbol_BodyTruncation(t *testing.T) {
	// Body longer than 500 chars should be truncated.
	longBody := strings.Repeat("x", 1000)
	text := formatCodeSymbol("foo.Bar", "func", "func Bar()", "", longBody, "pkg/foo.go")

	// The body portion should be at most 500 chars.
	if strings.Count(text, "x") > 500 {
		t.Error("body should be truncated to 500 chars")
	}
}

// --- Test helpers for graph database ---

func setupGraphTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "graph.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening test graph db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create a minimal nodes table matching the graph schema.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS nodes (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			type        TEXT NOT NULL,
			key         TEXT NOT NULL,
			props       TEXT NOT NULL DEFAULT '{}',
			scope       TEXT NOT NULL DEFAULT 'team',
			repo        TEXT NOT NULL DEFAULT '',
			unit        TEXT NOT NULL DEFAULT '',
			domain      TEXT NOT NULL DEFAULT '',
			content_hash TEXT,
			source      TEXT NOT NULL DEFAULT '{}',
			valid_from  TEXT NOT NULL,
			valid_to    TEXT,
			ingested_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("creating nodes table: %v", err)
	}

	return db
}

func insertTestEvent(t *testing.T, db *sql.DB, typ, key, subject, body string) {
	t.Helper()
	props := `{"subject":"` + subject + `","body":"` + body + `"}`
	_, err := db.Exec(`
		INSERT INTO nodes (type, key, props, valid_from, ingested_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, typ, key, props)
	if err != nil {
		t.Fatalf("inserting test event: %v", err)
	}
}

func insertTestSymbol(t *testing.T, db *sql.DB, key, kind, signature, doc, body, path string) {
	t.Helper()
	props := `{"kind":"` + kind + `","signature":"` + signature + `","doc_comment":"` + doc + `","body":"` + body + `","path":"` + path + `"}`
	_, err := db.Exec(`
		INSERT INTO nodes (type, key, props, valid_from, ingested_at)
		VALUES ('Symbol', ?, ?, datetime('now'), datetime('now'))
	`, key, props)
	if err != nil {
		t.Fatalf("inserting test symbol: %v", err)
	}
}
