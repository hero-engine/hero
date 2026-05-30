package embeddings

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Storage handles vector chunk persistence in the index database. It adds
// its own tables to the shared index.db schema via an idempotent migration
// that runs on open.
type Storage struct {
	db *sql.DB
}

// Chunk represents a text chunk with its embedding.
type Chunk struct {
	ID       string    // unique chunk ID (corpus:source_id:section)
	Corpus   string    // "spec", "knowledge", "convention", "event", "code"
	SourceID string    // spec slug, file path, event ID, symbol ID
	Section  string    // "## Problem" for spec chunks; "" otherwise
	TextHash string    // sha256 of chunk text for invalidation
	Vector   []float32 // embedding vector
}

// ScoredChunk pairs a chunk with its similarity score.
type ScoredChunk struct {
	Chunk
	Score float64
}

// StorageStats reports chunk counts by corpus and total.
type StorageStats struct {
	ByCorpus map[string]int
	Total    int
}

// RawDB returns the underlying *sql.DB.
func (s *Storage) RawDB() *sql.DB { return s.db }

// OpenStorage opens the vector storage, creating tables if needed. The db
// argument must be an already-open *sql.DB pointing at the shared index.db.
func OpenStorage(db *sql.DB) (*Storage, error) {
	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("embeddings: migrating storage schema: %w", err)
	}
	return s, nil
}

func (s *Storage) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS vec_chunks (
			chunk_id    TEXT PRIMARY KEY,
			corpus      TEXT NOT NULL,
			source_id   TEXT NOT NULL,
			section     TEXT NOT NULL DEFAULT '',
			text_hash   TEXT NOT NULL,
			vector      BLOB NOT NULL,
			embedded_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_vec_chunks_corpus ON vec_chunks(corpus)`,
		`CREATE INDEX IF NOT EXISTS idx_vec_chunks_source ON vec_chunks(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vec_chunks_hash ON vec_chunks(text_hash)`,
	}
	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("executing migration: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

// Upsert inserts or updates a chunk. If an existing chunk has the same
// text_hash the write is skipped and changed=false is returned.
func (s *Storage) Upsert(chunk Chunk) (changed bool, err error) {
	// Check existing hash to avoid redundant writes.
	var existingHash string
	err = s.db.QueryRow(
		`SELECT text_hash FROM vec_chunks WHERE chunk_id = ?`,
		chunk.ID,
	).Scan(&existingHash)

	if err == nil && existingHash == chunk.TextHash {
		return false, nil // no change
	}

	blob := encodeVector(chunk.Vector)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = s.db.Exec(`
		INSERT INTO vec_chunks (chunk_id, corpus, source_id, section, text_hash, vector, embedded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chunk_id) DO UPDATE SET
			corpus=excluded.corpus,
			source_id=excluded.source_id,
			section=excluded.section,
			text_hash=excluded.text_hash,
			vector=excluded.vector,
			embedded_at=excluded.embedded_at
	`, chunk.ID, chunk.Corpus, chunk.SourceID, chunk.Section, chunk.TextHash, blob, now)
	if err != nil {
		return false, fmt.Errorf("upserting chunk %q: %w", chunk.ID, err)
	}
	return true, nil
}

// Delete removes a chunk by ID.
func (s *Storage) Delete(chunkID string) error {
	_, err := s.db.Exec(`DELETE FROM vec_chunks WHERE chunk_id = ?`, chunkID)
	if err != nil {
		return fmt.Errorf("deleting chunk %q: %w", chunkID, err)
	}
	return nil
}

// PruneCorpus removes all chunks for a corpus not in the given set of IDs.
func (s *Storage) PruneCorpus(corpus string, keepIDs []string) (pruned int, err error) {
	if len(keepIDs) == 0 {
		// Remove all chunks for this corpus.
		res, err := s.db.Exec(`DELETE FROM vec_chunks WHERE corpus = ?`, corpus)
		if err != nil {
			return 0, fmt.Errorf("pruning corpus %q: %w", corpus, err)
		}
		n, _ := res.RowsAffected()
		return int(n), nil
	}

	// Build placeholder list for the keep set.
	placeholders := make([]string, len(keepIDs))
	args := make([]interface{}, 0, len(keepIDs)+1)
	args = append(args, corpus)
	for i, id := range keepIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`DELETE FROM vec_chunks WHERE corpus = ? AND chunk_id NOT IN (%s)`,
		strings.Join(placeholders, ","),
	)
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("pruning corpus %q: %w", corpus, err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// QuerySimilar returns the top-K most similar chunks to the query vector.
// It computes cosine similarity via brute-force scan over the requested
// corpora. For normalized vectors cosine similarity is the dot product.
func (s *Storage) QuerySimilar(queryVec []float32, topK int, corpora []string) ([]ScoredChunk, error) {
	if topK <= 0 {
		topK = 10
	}

	// Build query with optional corpus filter.
	query := `SELECT chunk_id, corpus, source_id, section, text_hash, vector FROM vec_chunks`
	var args []interface{}

	if len(corpora) > 0 {
		placeholders := make([]string, len(corpora))
		for i, c := range corpora {
			placeholders[i] = "?"
			args = append(args, c)
		}
		query += fmt.Sprintf(` WHERE corpus IN (%s)`, strings.Join(placeholders, ","))
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying vec_chunks: %w", err)
	}
	defer rows.Close()

	var results []ScoredChunk

	for rows.Next() {
		var c Chunk
		var blob []byte
		if err := rows.Scan(&c.ID, &c.Corpus, &c.SourceID, &c.Section, &c.TextHash, &blob); err != nil {
			return nil, fmt.Errorf("scanning vec_chunk row: %w", err)
		}

		vec := decodeVector(blob)
		score := float64(CosineSimilarity(queryVec, vec))

		results = append(results, ScoredChunk{
			Chunk: c,
			Score: score,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort descending by score.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// Stats returns chunk counts per corpus and total index size.
func (s *Storage) Stats() (*StorageStats, error) {
	rows, err := s.db.Query(`SELECT corpus, COUNT(*) FROM vec_chunks GROUP BY corpus`)
	if err != nil {
		return nil, fmt.Errorf("querying vec_chunks stats: %w", err)
	}
	defer rows.Close()

	stats := &StorageStats{
		ByCorpus: make(map[string]int),
	}

	for rows.Next() {
		var corpus string
		var count int
		if err := rows.Scan(&corpus, &count); err != nil {
			return nil, err
		}
		stats.ByCorpus[corpus] = count
		stats.Total += count
	}

	return stats, rows.Err()
}

// encodeVector serializes a float32 slice as raw little-endian bytes.
func encodeVector(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// decodeVector deserializes raw little-endian float32 bytes into a slice.
func decodeVector(data []byte) []float32 {
	n := len(data) / 4
	vec := make([]float32, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		vec[i] = math.Float32frombits(bits)
	}
	return vec
}
