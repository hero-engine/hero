package embeddings

import (
	"context"
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
	Corpora  map[string]CorpusStorageStats
	Total    int
}

// CorpusStorageStats reports persisted coverage metadata for one corpus.
type CorpusStorageStats struct {
	Count                  int
	NewestEmbeddedAt       time.Time
	SuccessfulExtractionAt time.Time
	ExtractionOutcome      string
}

// ReconcileStats reports the writes committed by one corpus transaction.
type ReconcileStats struct {
	Written int
	Pruned  int
}

// RawDB returns the underlying *sql.DB.
func (s *Storage) RawDB() *sql.DB { return s.db }

// OpenStorage opens the vector storage, creating tables if needed. The db
// argument must be an already-open *sql.DB pointing at the shared index.db.
func OpenStorage(db *sql.DB) (*Storage, error) {
	return OpenStorageContext(context.Background(), db)
}

// OpenStorageContext opens vector storage using context-aware migrations.
func OpenStorageContext(ctx context.Context, db *sql.DB) (*Storage, error) {
	s := &Storage{db: db}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("embeddings: acquiring migration connection: %w", err)
	}
	defer conn.Close()
	if err := setDeadlineBusyTimeout(ctx, conn); err != nil {
		return nil, fmt.Errorf("embeddings: configuring migration deadline: %w", err)
	}
	if err := s.migrate(ctx, conn); err != nil {
		return nil, fmt.Errorf("embeddings: migrating storage schema: %w", err)
	}
	return s, nil
}

func (s *Storage) migrate(ctx context.Context, execer contextExecer) error {
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
		`CREATE TABLE IF NOT EXISTS vec_corpus_generations (
			corpus       TEXT PRIMARY KEY,
			outcome      TEXT NOT NULL,
			completed_at TEXT NOT NULL
		)`,
	}
	for _, m := range migrations {
		if _, err := execer.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("executing migration: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

// Upsert inserts or updates a chunk. If an existing chunk has the same
// text_hash the write is skipped and changed=false is returned.
func (s *Storage) Upsert(chunk Chunk) (changed bool, err error) {
	hashes, err := s.StoredHashes(context.Background(), []string{chunk.ID})
	if err != nil {
		return false, err
	}
	if hashes[chunk.ID] == chunk.TextHash {
		return false, nil // no change
	}
	_, err = s.ReconcileCorpus(context.Background(), chunk.Corpus, []Chunk{chunk}, nil, false)
	if err != nil {
		return false, fmt.Errorf("upserting chunk %q: %w", chunk.ID, err)
	}
	return true, nil
}

// StoredHashes returns existing text hashes for the requested chunk IDs.
// Reads are bounded to stay below SQLite's variable limit.
func (s *Storage) StoredHashes(ctx context.Context, chunkIDs []string) (map[string]string, error) {
	const batchSize = 500
	hashes := make(map[string]string, len(chunkIDs))
	for start := 0; start < len(chunkIDs); start += batchSize {
		end := start + batchSize
		if end > len(chunkIDs) {
			end = len(chunkIDs)
		}
		placeholders := make([]string, end-start)
		args := make([]any, end-start)
		for i, id := range chunkIDs[start:end] {
			placeholders[i] = "?"
			args[i] = id
		}
		conn, err := s.db.Conn(ctx)
		if err != nil {
			return nil, fmt.Errorf("acquiring stored-hash connection: %w", err)
		}
		if err := setDeadlineBusyTimeout(ctx, conn); err != nil {
			conn.Close()
			return nil, err
		}
		rows, err := conn.QueryContext(ctx,
			`SELECT chunk_id, text_hash FROM vec_chunks WHERE chunk_id IN (`+strings.Join(placeholders, ",")+`)`,
			args...,
		)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("reading stored chunk hashes: %w", err)
		}
		for rows.Next() {
			var id, hash string
			if err := rows.Scan(&id, &hash); err != nil {
				rows.Close()
				conn.Close()
				return nil, fmt.Errorf("scanning stored chunk hash: %w", err)
			}
			hashes[id] = hash
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			conn.Close()
			return nil, fmt.Errorf("iterating stored chunk hashes: %w", err)
		}
		rows.Close()
		conn.Close()
	}
	return hashes, nil
}

// ReconcileCorpus atomically writes changed chunks and, when authoritative,
// prunes chunks absent from keepIDs and records a successful generation.
func (s *Storage) ReconcileCorpus(
	ctx context.Context,
	corpus string,
	changed []Chunk,
	keepIDs []string,
	authoritative bool,
) (stats ReconcileStats, err error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return stats, fmt.Errorf("acquiring %q corpus connection: %w", corpus, err)
	}
	defer conn.Close()
	if err := setDeadlineBusyTimeout(ctx, conn); err != nil {
		return stats, fmt.Errorf("configuring %q corpus deadline: %w", corpus, err)
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return stats, fmt.Errorf("beginning %q corpus transaction: %w", corpus, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO vec_chunks (chunk_id, corpus, source_id, section, text_hash, vector, embedded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chunk_id) DO UPDATE SET
			corpus=excluded.corpus,
			source_id=excluded.source_id,
			section=excluded.section,
			text_hash=excluded.text_hash,
			vector=excluded.vector,
			embedded_at=excluded.embedded_at
	`)
	if err != nil {
		return stats, fmt.Errorf("preparing %q corpus upsert: %w", corpus, err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, chunk := range changed {
		if chunk.Corpus != corpus {
			return stats, fmt.Errorf("chunk %q belongs to corpus %q, not %q", chunk.ID, chunk.Corpus, corpus)
		}
		if _, err = stmt.ExecContext(ctx,
			chunk.ID, chunk.Corpus, chunk.SourceID, chunk.Section,
			chunk.TextHash, encodeVector(chunk.Vector), now,
		); err != nil {
			return stats, fmt.Errorf("upserting chunk %q: %w", chunk.ID, err)
		}
		stats.Written++
	}

	if authoritative {
		stats.Pruned, err = pruneCorpus(ctx, tx, corpus, keepIDs)
		if err != nil {
			return stats, err
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO vec_corpus_generations (corpus, outcome, completed_at)
			VALUES (?, 'complete', ?)
			ON CONFLICT(corpus) DO UPDATE SET
				outcome=excluded.outcome,
				completed_at=excluded.completed_at
		`, corpus, now); err != nil {
			return stats, fmt.Errorf("recording %q corpus generation: %w", corpus, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return stats, fmt.Errorf("committing %q corpus transaction: %w", corpus, err)
	}
	return stats, nil
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
	return pruneCorpus(context.Background(), s.db, corpus, keepIDs)
}

type contextExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func setDeadlineBusyTimeout(ctx context.Context, execer contextExecer) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return context.DeadlineExceeded
	}
	milliseconds := remaining.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	_, err := execer.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", milliseconds))
	return err
}

func pruneCorpus(ctx context.Context, execer contextExecer, corpus string, keepIDs []string) (pruned int, err error) {
	if len(keepIDs) == 0 {
		// Remove all chunks for this corpus.
		res, err := execer.ExecContext(ctx, `DELETE FROM vec_chunks WHERE corpus = ?`, corpus)
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
	res, err := execer.ExecContext(ctx, query, args...)
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
	rows, err := s.db.Query(`
		SELECT corpus, COUNT(*), COALESCE(MAX(embedded_at), '')
		FROM vec_chunks
		GROUP BY corpus
	`)
	if err != nil {
		return nil, fmt.Errorf("querying vec_chunks stats: %w", err)
	}
	defer rows.Close()

	stats := &StorageStats{
		ByCorpus: make(map[string]int),
		Corpora:  make(map[string]CorpusStorageStats),
	}
	for _, corpus := range []string{"spec", "knowledge", "convention", "event", "code"} {
		stats.Corpora[corpus] = CorpusStorageStats{}
	}

	for rows.Next() {
		var corpus string
		var count int
		var newest string
		if err := rows.Scan(&corpus, &count, &newest); err != nil {
			return nil, err
		}
		stats.ByCorpus[corpus] = count
		corpusStats := stats.Corpora[corpus]
		corpusStats.Count = count
		corpusStats.NewestEmbeddedAt, _ = time.Parse(time.RFC3339Nano, newest)
		stats.Corpora[corpus] = corpusStats
		stats.Total += count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	generations, err := s.db.Query(`SELECT corpus, outcome, completed_at FROM vec_corpus_generations`)
	if err != nil {
		return nil, fmt.Errorf("querying corpus generation stats: %w", err)
	}
	defer generations.Close()
	for generations.Next() {
		var corpus, outcome, completedAt string
		if err := generations.Scan(&corpus, &outcome, &completedAt); err != nil {
			return nil, err
		}
		corpusStats := stats.Corpora[corpus]
		corpusStats.ExtractionOutcome = outcome
		corpusStats.SuccessfulExtractionAt, _ = time.Parse(time.RFC3339Nano, completedAt)
		stats.Corpora[corpus] = corpusStats
	}
	return stats, generations.Err()
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
