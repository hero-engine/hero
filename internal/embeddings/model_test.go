package embeddings

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// makeTestModel creates a small deterministic model in a temp directory.
// 10 tokens x 4 dimensions. Returns the loaded model and cleanup func.
func makeTestModel(t *testing.T) *Model {
	t.Helper()

	dir := t.TempDir()

	// Vocabulary: 10 tokens.
	tokens := []string{
		"hello", "world", "foo", "bar", "test",
		"validate", "session", "token", "error", "code",
	}
	vocabPath := filepath.Join(dir, "vocab.txt")
	var vocabContent string
	for _, tok := range tokens {
		vocabContent += tok + "\n"
	}
	if err := os.WriteFile(vocabPath, []byte(vocabContent), 0o644); err != nil {
		t.Fatalf("writing vocab: %v", err)
	}

	// Weights: 10 x 4 = 40 float32 values.
	// Each token gets a distinctive vector for deterministic testing.
	weights := []float32{
		// hello:    [1, 0, 0, 0]
		1, 0, 0, 0,
		// world:    [0, 1, 0, 0]
		0, 1, 0, 0,
		// foo:      [0, 0, 1, 0]
		0, 0, 1, 0,
		// bar:      [0, 0, 0, 1]
		0, 0, 0, 1,
		// test:     [1, 1, 0, 0]
		1, 1, 0, 0,
		// validate: [0.5, 0.5, 0.5, 0]
		0.5, 0.5, 0.5, 0,
		// session:  [0, 0.5, 0.5, 0.5]
		0, 0.5, 0.5, 0.5,
		// token:    [0.5, 0, 0.5, 0.5]
		0.5, 0, 0.5, 0.5,
		// error:    [-1, 0, 0, 0]
		-1, 0, 0, 0,
		// code:     [0, -1, 0, 0]
		0, -1, 0, 0,
	}

	weightsPath := filepath.Join(dir, "weights.bin")
	buf := make([]byte, len(weights)*4)
	for i, v := range weights {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	if err := os.WriteFile(weightsPath, buf, 0o644); err != nil {
		t.Fatalf("writing weights: %v", err)
	}

	m, err := LoadModel(dir, 4)
	if err != nil {
		t.Fatalf("LoadModel: %v", err)
	}
	return m
}

func TestLoadModel(t *testing.T) {
	m := makeTestModel(t)

	if m.Dim() != 4 {
		t.Errorf("Dim() = %d, want 4", m.Dim())
	}
	if len(m.vocab) != 10 {
		t.Errorf("vocab size = %d, want 10", len(m.vocab))
	}
	if len(m.weights) != 40 {
		t.Errorf("weights len = %d, want 40", len(m.weights))
	}
}

func TestLoadModel_MissingFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadModel(dir, 4)
	if err == nil {
		t.Fatal("expected error for missing files")
	}
}

func TestLoadModel_InvalidDim(t *testing.T) {
	_, err := LoadModel(t.TempDir(), 0)
	if err == nil {
		t.Fatal("expected error for dim=0")
	}

	_, err = LoadModel(t.TempDir(), -1)
	if err == nil {
		t.Fatal("expected error for negative dim")
	}
}

func TestLoadModel_WeightsSizeMismatch(t *testing.T) {
	dir := t.TempDir()

	// Write a small vocab.
	if err := os.WriteFile(filepath.Join(dir, "vocab.txt"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Write wrong-sized weights (should be 2*4*4=32 bytes, write 16).
	if err := os.WriteFile(filepath.Join(dir, "weights.bin"), make([]byte, 16), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadModel(dir, 4)
	if err == nil {
		t.Fatal("expected error for weights size mismatch")
	}
}

// --- Tokenizer tests ---

func TestTokenize_Whitespace(t *testing.T) {
	tokens := Tokenize("hello world foo")
	want := []string{"hello", "world", "foo"}
	assertTokens(t, tokens, want)
}

func TestTokenize_Punctuation(t *testing.T) {
	tokens := Tokenize("foo.bar baz-qux")
	want := []string{"foo", "bar", "baz", "qux"}
	assertTokens(t, tokens, want)
}

func TestTokenize_CamelCase(t *testing.T) {
	tokens := Tokenize("validateSessionToken")
	want := []string{"validate", "session", "token"}
	assertTokens(t, tokens, want)
}

func TestTokenize_CamelCase_Acronym(t *testing.T) {
	tokens := Tokenize("HTMLParser")
	want := []string{"html", "parser"}
	assertTokens(t, tokens, want)
}

func TestTokenize_Empty(t *testing.T) {
	tokens := Tokenize("")
	if len(tokens) != 0 {
		t.Errorf("expected empty tokens, got %v", tokens)
	}
}

func TestTokenize_Mixed(t *testing.T) {
	tokens := Tokenize("validateSession.Token foo_bar")
	want := []string{"validate", "session", "token", "foo", "bar"}
	assertTokens(t, tokens, want)
}

func TestTokenize_Lowercase(t *testing.T) {
	tokens := Tokenize("HELLO WORLD")
	want := []string{"hello", "world"}
	assertTokens(t, tokens, want)
}

// --- Embed tests ---

func TestEmbed_Dimension(t *testing.T) {
	m := makeTestModel(t)
	vec := m.Embed("hello world")
	if len(vec) != 4 {
		t.Fatalf("Embed dimension = %d, want 4", len(vec))
	}
}

func TestEmbed_Normalized(t *testing.T) {
	m := makeTestModel(t)
	vec := m.Embed("hello world")
	norm := l2norm(vec)
	if math.Abs(float64(norm)-1.0) > 1e-5 {
		t.Errorf("L2 norm = %f, want ~1.0", norm)
	}
}

func TestEmbed_SingleToken(t *testing.T) {
	m := makeTestModel(t)
	// "hello" has weight [1,0,0,0]. After normalize: [1,0,0,0].
	vec := m.Embed("hello")
	if vec[0] < 0.99 || vec[1] > 0.01 || vec[2] > 0.01 || vec[3] > 0.01 {
		t.Errorf("unexpected vector for 'hello': %v", vec)
	}
}

func TestEmbed_AllUnknown(t *testing.T) {
	m := makeTestModel(t)
	vec := m.Embed("xyzzy plugh")
	// All unknown -> zero vector.
	for i, v := range vec {
		if v != 0 {
			t.Errorf("vec[%d] = %f, want 0 for all-unknown input", i, v)
		}
	}
}

func TestEmbed_EmptyInput(t *testing.T) {
	m := makeTestModel(t)
	vec := m.Embed("")
	for i, v := range vec {
		if v != 0 {
			t.Errorf("vec[%d] = %f, want 0 for empty input", i, v)
		}
	}
}

func TestEmbed_MeanPool(t *testing.T) {
	m := makeTestModel(t)
	// "hello world" -> mean([1,0,0,0], [0,1,0,0]) = [0.5,0.5,0,0]
	// After normalize: [1/sqrt(2), 1/sqrt(2), 0, 0]
	vec := m.Embed("hello world")
	expected := float32(1.0 / math.Sqrt(2.0))
	if math.Abs(float64(vec[0]-expected)) > 1e-5 {
		t.Errorf("vec[0] = %f, want %f", vec[0], expected)
	}
	if math.Abs(float64(vec[1]-expected)) > 1e-5 {
		t.Errorf("vec[1] = %f, want %f", vec[1], expected)
	}
}

func TestEmbed_CamelCaseSplit(t *testing.T) {
	m := makeTestModel(t)
	// "validateSessionToken" should split into "validate", "session", "token"
	// which are all in the vocabulary.
	vec := m.Embed("validateSessionToken")
	norm := l2norm(vec)
	if math.Abs(float64(norm)-1.0) > 1e-5 {
		t.Errorf("L2 norm = %f, want ~1.0", norm)
	}
	// verify it's non-zero (all tokens matched)
	allZero := true
	for _, v := range vec {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("expected non-zero vector for camelCase input with known tokens")
	}
}

// --- EmbedBatch tests ---

func TestEmbedBatch(t *testing.T) {
	m := makeTestModel(t)
	texts := []string{"hello", "world", "hello world"}
	batch := m.EmbedBatch(texts)

	if len(batch) != 3 {
		t.Fatalf("batch length = %d, want 3", len(batch))
	}

	// Verify each batch result matches individual Embed.
	for i, text := range texts {
		single := m.Embed(text)
		for j := range single {
			if math.Abs(float64(batch[i][j]-single[j])) > 1e-7 {
				t.Errorf("batch[%d][%d] = %f, single = %f", i, j, batch[i][j], single[j])
			}
		}
	}
}

// --- CosineSimilarity tests ---

func TestCosineSimilarity_Identical(t *testing.T) {
	a := []float32{1, 0, 0, 0}
	sim := CosineSimilarity(a, a)
	if math.Abs(float64(sim)-1.0) > 1e-5 {
		t.Errorf("identical vectors: similarity = %f, want 1.0", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1, 0, 0, 0}
	b := []float32{0, 1, 0, 0}
	sim := CosineSimilarity(a, b)
	if math.Abs(float64(sim)) > 1e-5 {
		t.Errorf("orthogonal vectors: similarity = %f, want 0.0", sim)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1, 0, 0, 0}
	b := []float32{-1, 0, 0, 0}
	sim := CosineSimilarity(a, b)
	if math.Abs(float64(sim)+1.0) > 1e-5 {
		t.Errorf("opposite vectors: similarity = %f, want -1.0", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	sim := CosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("different lengths: similarity = %f, want 0", sim)
	}
}

// --- LoadModelFromBytes tests ---

func TestLoadModelFromBytes(t *testing.T) {
	// Build in-memory vocab + weights matching the test model format.
	vocab := "hello\nworld\nfoo\n"
	dim := 2
	// 3 tokens x 2 dims = 6 float32 values.
	weights := make([]byte, 3*dim*4)
	vals := []float32{1, 0, 0, 1, 0.5, 0.5}
	for i, v := range vals {
		binary.LittleEndian.PutUint32(weights[i*4:], math.Float32bits(v))
	}
	config := []byte(`{"dim": 2}`)

	m, err := LoadModelFromBytes([]byte(vocab), weights, config)
	if err != nil {
		t.Fatalf("LoadModelFromBytes: %v", err)
	}
	if m.Dim() != 2 {
		t.Errorf("Dim() = %d, want 2", m.Dim())
	}
	if len(m.vocab) != 3 {
		t.Errorf("vocab size = %d, want 3", len(m.vocab))
	}

	// Verify embedding works.
	vec := m.Embed("hello")
	if len(vec) != 2 {
		t.Fatalf("Embed dim = %d, want 2", len(vec))
	}
	if vec[0] < 0.99 {
		t.Errorf("vec[0] = %f, want ~1.0", vec[0])
	}
}

func TestLoadModelFromBytes_NoConfig(t *testing.T) {
	// Without config, defaults to DefaultDim=256.
	vocab := "hello\nworld\n"
	weights := make([]byte, 2*DefaultDim*4) // 2 tokens x 256 dims

	m, err := LoadModelFromBytes([]byte(vocab), weights, nil)
	if err != nil {
		t.Fatalf("LoadModelFromBytes: %v", err)
	}
	if m.Dim() != DefaultDim {
		t.Errorf("Dim() = %d, want %d", m.Dim(), DefaultDim)
	}
}

func TestLoadModelFromBytes_EmptyVocab(t *testing.T) {
	_, err := LoadModelFromBytes([]byte(""), []byte{}, nil)
	if err == nil {
		t.Fatal("expected error for empty vocab")
	}
}

func TestLoadModelFromBytes_WeightsMismatch(t *testing.T) {
	vocab := "hello\nworld\n"
	config := []byte(`{"dim": 4}`)
	weights := make([]byte, 16) // 2 tokens x 4 dims = 32 bytes, but we give 16

	_, err := LoadModelFromBytes([]byte(vocab), weights, config)
	if err == nil {
		t.Fatal("expected error for weights size mismatch")
	}
}

func TestLoadModelFromBytes_InvalidConfig(t *testing.T) {
	// Invalid JSON config falls back to DefaultDim.
	vocab := "hello\n"
	weights := make([]byte, 1*DefaultDim*4)

	m, err := LoadModelFromBytes([]byte(vocab), weights, []byte("not json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Dim() != DefaultDim {
		t.Errorf("Dim() = %d, want %d (fallback from invalid config)", m.Dim(), DefaultDim)
	}
}

func TestLoadModelFromBytes_EmbeddedDefault(t *testing.T) {
	// Verify the actual embedded model loads via LoadModelFromBytes.
	// This is the path used by LoadModelFromConfig when no filesystem
	// override is present.
	m, err := LoadModelFromConfig("hero-embed-v1")
	if err != nil {
		t.Fatalf("LoadModelFromConfig: %v", err)
	}
	if m == nil {
		t.Fatal("expected embedded model to load")
	}
	if m.Dim() != 256 {
		t.Errorf("embedded model dim = %d, want 256", m.Dim())
	}

	// Sanity: embedding should produce a non-zero vector.
	vec := m.Embed("test query")
	nonZero := false
	for _, v := range vec {
		if v != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Error("embedded model produced a zero vector for 'test query'")
	}
}

// --- Helpers ---

func assertTokens(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("tokens length = %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("tokens[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func l2norm(vec []float32) float32 {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}
	return float32(math.Sqrt(float64(sum)))
}
