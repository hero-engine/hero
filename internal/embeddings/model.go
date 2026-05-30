// Package embeddings provides a static Model2Vec embedding engine and vector
// storage layer for semantic search over project content. The engine runs
// entirely in-process with no CGo or external dependencies: load a vocabulary
// and weight matrix, tokenize text, look up vectors, mean-pool, L2-normalize.
//
// Storage is backed by the existing index.db SQLite database. Vectors are
// serialized as raw little-endian float32 blobs. Retrieval is brute-force
// cosine similarity over the requested corpora, which completes in <10ms at
// project scale (10K-100K chunks x 256-dim).
package embeddings

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/hero-engine/hero/internal/embeddings/defaultmodel"
)

// Model holds the static embedding weights and vocabulary.
type Model struct {
	vocab   map[string]int // token -> row index in weight matrix
	dim     int            // embedding dimension
	weights []float32      // flat matrix: vocab_size x dim (row-major)
}

// DefaultDim is the embedding dimension used by the default hero-embed-v1 model.
const DefaultDim = 256

// LoadModelFromConfig loads the embedding model. It checks for a filesystem
// override at ~/.hero/models/embeddings/<modelName>/ first (useful for
// testing custom models). If none is found, it falls back to the model
// weights embedded in the binary.
//
// Returns a non-nil error only for genuine failures (e.g. corrupt weights).
func LoadModelFromConfig(modelName string) (*Model, error) {
	if modelName == "" {
		modelName = "hero-embed-v1"
	}

	// Try filesystem override first.
	if home, err := os.UserHomeDir(); err == nil {
		modelDir := filepath.Join(home, ".hero", "models", "embeddings", modelName)
		vocabPath := filepath.Join(modelDir, "vocab.txt")
		weightsPath := filepath.Join(modelDir, "weights.bin")
		if _, err := os.Stat(vocabPath); err == nil {
			if _, err := os.Stat(weightsPath); err == nil {
				dim := DefaultDim
				configPath := filepath.Join(modelDir, "config.json")
				if data, err := os.ReadFile(configPath); err == nil {
					var cfg struct {
						Dim int `json:"dim"`
					}
					if jsonErr := json.Unmarshal(data, &cfg); jsonErr == nil && cfg.Dim > 0 {
						dim = cfg.Dim
					}
				}
				return LoadModel(modelDir, dim)
			}
		}
	}

	// Fall back to embedded model.
	if modelName == "hero-embed-v1" && len(defaultmodel.Weights) > 0 {
		return LoadModelFromBytes(defaultmodel.Vocab, defaultmodel.Weights, defaultmodel.Config)
	}

	return nil, nil
}

// LoadModelFromBytes loads a model from in-memory byte slices (used for
// the binary-embedded default model).
func LoadModelFromBytes(vocabData, weightsData, configData []byte) (*Model, error) {
	dim := DefaultDim
	if len(configData) > 0 {
		var cfg struct {
			Dim int `json:"dim"`
		}
		if err := json.Unmarshal(configData, &cfg); err == nil && cfg.Dim > 0 {
			dim = cfg.Dim
		}
	}

	// Parse vocabulary.
	vocab := make(map[string]int)
	scanner := bufio.NewScanner(bytes.NewReader(vocabData))
	idx := 0
	for scanner.Scan() {
		token := strings.TrimSpace(scanner.Text())
		if token != "" {
			vocab[token] = idx
			idx++
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("embeddings: reading embedded vocab: %w", err)
	}
	if len(vocab) == 0 {
		return nil, fmt.Errorf("embeddings: embedded vocab is empty")
	}

	// Parse weight matrix.
	expectedBytes := len(vocab) * dim * 4
	if len(weightsData) != expectedBytes {
		return nil, fmt.Errorf("embeddings: embedded weights size mismatch: got %d bytes, expected %d (vocab=%d, dim=%d)",
			len(weightsData), expectedBytes, len(vocab), dim)
	}

	weights := make([]float32, len(vocab)*dim)
	for i := range weights {
		bits := binary.LittleEndian.Uint32(weightsData[i*4 : (i+1)*4])
		weights[i] = math.Float32frombits(bits)
	}

	return &Model{
		vocab:   vocab,
		dim:     dim,
		weights: weights,
	}, nil
}

// LoadModel loads a Model2Vec model from a directory containing vocab.txt and
// weights.bin.
//
//   - vocab.txt: one token per line, line number (0-indexed) is the row index
//   - weights.bin: raw little-endian float32 values, vocab_size x dim
func LoadModel(dir string, dim int) (*Model, error) {
	if dim <= 0 {
		return nil, fmt.Errorf("embeddings: dim must be positive, got %d", dim)
	}

	// Load vocabulary.
	vocabPath := filepath.Join(dir, "vocab.txt")
	f, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("embeddings: opening vocab: %w", err)
	}
	defer f.Close()

	vocab := make(map[string]int)
	scanner := bufio.NewScanner(f)
	idx := 0
	for scanner.Scan() {
		token := strings.TrimSpace(scanner.Text())
		if token != "" {
			vocab[token] = idx
			idx++
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("embeddings: reading vocab: %w", err)
	}
	if len(vocab) == 0 {
		return nil, fmt.Errorf("embeddings: vocab is empty")
	}

	// Load weight matrix.
	weightsPath := filepath.Join(dir, "weights.bin")
	data, err := os.ReadFile(weightsPath)
	if err != nil {
		return nil, fmt.Errorf("embeddings: reading weights: %w", err)
	}

	expectedBytes := len(vocab) * dim * 4
	if len(data) != expectedBytes {
		return nil, fmt.Errorf("embeddings: weights size mismatch: got %d bytes, expected %d (vocab=%d, dim=%d)",
			len(data), expectedBytes, len(vocab), dim)
	}

	weights := make([]float32, len(vocab)*dim)
	for i := range weights {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		weights[i] = math.Float32frombits(bits)
	}

	return &Model{
		vocab:   vocab,
		dim:     dim,
		weights: weights,
	}, nil
}

// Dim returns the embedding dimension.
func (m *Model) Dim() int { return m.dim }

// Embed produces a normalized float32 vector for the input text. If no tokens
// match the vocabulary, a zero vector of length dim is returned.
func (m *Model) Embed(text string) []float32 {
	tokens := Tokenize(text)

	vec := make([]float32, m.dim)
	matched := 0

	for _, tok := range tokens {
		row, ok := m.vocab[tok]
		if !ok {
			continue
		}
		offset := row * m.dim
		for j := 0; j < m.dim; j++ {
			vec[j] += m.weights[offset+j]
		}
		matched++
	}

	if matched == 0 {
		return vec // zero vector
	}

	// Mean pool.
	scale := 1.0 / float32(matched)
	for j := range vec {
		vec[j] *= scale
	}

	// L2 normalize.
	normalize(vec)
	return vec
}

// EmbedBatch embeds multiple texts. Each result is independent and identical
// to calling Embed on each text individually.
func (m *Model) EmbedBatch(texts []string) [][]float32 {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		results[i] = m.Embed(text)
	}
	return results
}

// CosineSimilarity computes the cosine similarity between two normalized
// vectors. For normalized vectors this is simply the dot product. Returns 0
// if the vectors have different lengths.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return dot
}

// normalize L2-normalizes vec in place. If the norm is zero the vector is
// left unchanged (already a zero vector).
func normalize(vec []float32) {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}
	if sum == 0 {
		return
	}
	inv := float32(1.0 / math.Sqrt(float64(sum)))
	for i := range vec {
		vec[i] *= inv
	}
}

// Tokenize splits text into lowercase tokens suitable for vocabulary lookup.
// Steps:
//  1. Split on whitespace (preserving original case for camelCase detection)
//  2. Split camelCase boundaries within each word
//  3. Split on punctuation boundaries
//  4. Lowercase all resulting tokens
func Tokenize(text string) []string {
	words := strings.Fields(text)

	var tokens []string
	for _, w := range words {
		// Split camelCase BEFORE lowercasing (need original case).
		camelParts := splitCamelCase(w)
		for _, cp := range camelParts {
			// Split on punctuation boundaries.
			punctParts := splitPunctuation(cp)
			for _, pp := range punctParts {
				lower := strings.ToLower(pp)
				if lower != "" {
					tokens = append(tokens, lower)
				}
			}
		}
	}
	return tokens
}

// splitPunctuation splits a string at punctuation boundaries, discarding the
// punctuation characters. "foo.bar" -> ["foo", "bar"].
func splitPunctuation(s string) []string {
	var parts []string
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if buf.Len() > 0 {
				parts = append(parts, buf.String())
				buf.Reset()
			}
		} else {
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	return parts
}

// splitCamelCase splits on camelCase boundaries.
//
//	"validateSessionToken" -> ["validate", "Session", "Token"]
//	"HTMLParser"           -> ["HTML", "Parser"]
//	"simpleTest"           -> ["simple", "Test"]
func splitCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	runes := []rune(s)
	var parts []string
	start := 0

	for i := 1; i < len(runes); i++ {
		// Split when transitioning from lowercase to uppercase.
		if unicode.IsLower(runes[i-1]) && unicode.IsUpper(runes[i]) {
			parts = append(parts, string(runes[start:i]))
			start = i
		}
		// Split when a run of uppercase is followed by a lowercase
		// (e.g., HTMLParser -> HTML, Parser).
		if i >= 2 && unicode.IsUpper(runes[i-2]) && unicode.IsUpper(runes[i-1]) && unicode.IsLower(runes[i]) {
			parts = append(parts, string(runes[start:i-1]))
			start = i - 1
		}
	}

	parts = append(parts, string(runes[start:]))
	return parts
}
