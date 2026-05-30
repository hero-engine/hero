package embeddings

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestRealModel_LoadAndEmbed loads the real hero-embed-v1 model from
// ~/.hero/models/embeddings/hero-embed-v1/ and validates embedding
// quality. This test is skipped if the model is not installed.
func TestRealModel_LoadAndEmbed(t *testing.T) {
	model, err := LoadModelFromConfig("hero-embed-v1")
	if err != nil {
		t.Fatalf("LoadModelFromConfig: %v", err)
	}
	if model == nil {
		t.Skip("hero-embed-v1 model not installed; skipping real model test")
	}

	t.Logf("Model loaded: dim=%d, vocab=%d", model.dim, len(model.vocab))

	queries := []string{
		"how does the scan pipeline work",
		"authentication and session management",
		"graph node edge traversal",
		"spec design convention knowledge",
		"vector search retrieval",
		"database migration",
		"validateSessionToken",
		"error handling",
	}

	for _, q := range queries {
		start := time.Now()
		vec := model.Embed(q)
		dur := time.Since(start)

		nonZero := 0
		for _, v := range vec {
			if v != 0 {
				nonZero++
			}
		}
		t.Logf("  %-45s  %5s  nonzero=%d/%d", q, dur.Round(time.Microsecond), nonZero, model.Dim())

		if nonZero == 0 {
			t.Errorf("query %q produced a zero vector (no vocab hits)", q)
		}
	}
}

// TestRealModel_SimilarityQuality validates that semantically related
// queries score higher than unrelated ones.
func TestRealModel_SimilarityQuality(t *testing.T) {
	model, err := LoadModelFromConfig("hero-embed-v1")
	if err != nil {
		t.Fatalf("LoadModelFromConfig: %v", err)
	}
	if model == nil {
		t.Skip("hero-embed-v1 model not installed; skipping real model test")
	}

	type pair struct {
		a, b    string
		related bool
	}

	pairs := []pair{
		{"scan pipeline ingestion", "how does hero scan work", true},
		{"scan pipeline ingestion", "authentication login", false},
		{"database query retrieval", "search index lookup", true},
		{"database query retrieval", "deploy server landing page", false},
		{"graph node traversal", "edge relationship path", true},
		{"graph node traversal", "color blue red green", false},
		{"error handling retry", "failure recovery fallback", true},
		{"error handling retry", "pretty font styling", false},
	}

	for _, p := range pairs {
		a := model.Embed(p.a)
		b := model.Embed(p.b)
		sim := CosineSimilarity(a, b)
		label := "UNRELATED"
		if p.related {
			label = "RELATED  "
		}
		t.Logf("  %s  %.4f  %-30s  <->  %s", label, sim, p.a, p.b)
	}

	// Verify related pairs score higher than unrelated pairs.
	for i := 0; i < len(pairs)-1; i += 2 {
		relatedPair := pairs[i]
		unrelatedPair := pairs[i+1]

		relatedSim := CosineSimilarity(model.Embed(relatedPair.a), model.Embed(relatedPair.b))
		unrelatedSim := CosineSimilarity(model.Embed(unrelatedPair.a), model.Embed(unrelatedPair.b))

		if relatedSim <= unrelatedSim {
			t.Errorf("related pair (%q, %q) scored %.4f <= unrelated pair (%q, %q) scored %.4f",
				relatedPair.a, relatedPair.b, relatedSim,
				unrelatedPair.a, unrelatedPair.b, unrelatedSim)
		}
	}
}

// TestRealModel_VocabCoverage checks what fraction of typical code/tech
// tokens hit the vocabulary.
func TestRealModel_VocabCoverage(t *testing.T) {
	model, err := LoadModelFromConfig("hero-embed-v1")
	if err != nil {
		t.Fatalf("LoadModelFromConfig: %v", err)
	}
	if model == nil {
		t.Skip("hero-embed-v1 model not installed; skipping real model test")
	}

	codeTerms := []string{
		"function", "error", "database", "query", "session", "token",
		"authentication", "api", "server", "client", "migration", "deploy",
		"test", "build", "index", "search", "graph", "node", "edge", "spec",
		"convention", "knowledge", "vector", "scan", "pipeline", "retrieval",
		"hybrid", "import", "export", "create", "update", "delete", "list",
		"read", "write", "open", "close", "start", "stop", "run", "status",
		"config", "schema", "model", "table", "column", "row", "key", "value",
		"request", "response", "handler", "middleware", "route", "path",
		"file", "directory", "package", "module", "interface", "struct",
		"type", "string", "int", "bool", "map", "slice", "array", "channel",
	}

	hits := 0
	for _, term := range codeTerms {
		if _, ok := model.vocab[term]; ok {
			hits++
		} else {
			t.Logf("  MISS: %s", term)
		}
	}

	coverage := float64(hits) / float64(len(codeTerms)) * 100
	t.Logf("Vocabulary coverage: %d/%d (%.1f%%)", hits, len(codeTerms), coverage)

	if coverage < 70 {
		t.Errorf("vocabulary coverage %.1f%% is below 70%% threshold", coverage)
	}
}

// TestRealModel_LoadPerformance benchmarks model loading time.
func TestRealModel_LoadPerformance(t *testing.T) {
	home, _ := os.UserHomeDir()
	modelDir := fmt.Sprintf("%s/.hero/models/embeddings/hero-embed-v1", home)
	if _, err := os.Stat(modelDir); os.IsNotExist(err) {
		t.Skip("hero-embed-v1 model not installed; skipping")
	}

	start := time.Now()
	model, err := LoadModelFromConfig("hero-embed-v1")
	loadTime := time.Since(start)

	if err != nil {
		t.Fatalf("LoadModelFromConfig: %v", err)
	}
	if model == nil {
		t.Fatal("model is nil")
	}

	t.Logf("Model load time: %s", loadTime)

	if loadTime > 2*time.Second {
		t.Errorf("model load took %s, expected under 2s", loadTime)
	}
}

// TestRealModel_EmbedPerformance benchmarks embedding throughput.
func TestRealModel_EmbedPerformance(t *testing.T) {
	model, err := LoadModelFromConfig("hero-embed-v1")
	if err != nil {
		t.Fatalf("LoadModelFromConfig: %v", err)
	}
	if model == nil {
		t.Skip("hero-embed-v1 model not installed; skipping")
	}

	texts := []string{
		"The retrieval pipeline merges BM25 lexical results with vector similarity",
		"Authentication middleware validates JWT tokens and checks session expiry",
		"Graph traversal finds all nodes connected by convention edges",
		"The scan command walks the file tree and projects structural nodes",
		"Database migration adds a new column for embedding vectors",
	}

	iterations := 1000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		model.Embed(texts[i%len(texts)])
	}
	elapsed := time.Since(start)

	perEmbed := elapsed / time.Duration(iterations)
	t.Logf("Embed performance: %d iterations in %s (%.1f µs/embed, %.0f embeds/sec)",
		iterations, elapsed, float64(perEmbed.Microseconds()), float64(time.Second)/float64(perEmbed))

	if perEmbed > 1*time.Millisecond {
		t.Errorf("embedding took %s per call, expected under 1ms", perEmbed)
	}
}
