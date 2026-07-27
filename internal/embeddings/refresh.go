package embeddings

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"
)

// CorpusRefreshStats reports the outcome of one configured corpus.
type CorpusRefreshStats struct {
	Corpus  string
	Outcome string
	Reason  string
	Added   int
	Updated int
	Pruned  int
	Skipped int
}

// RefreshStats reports what the refresh operation did.
type RefreshStats struct {
	Added       int
	Updated     int
	Pruned      int
	Skipped     int // unchanged (hash match)
	Corpora     map[string]CorpusRefreshStats
	Unavailable bool
	Elapsed     time.Duration
}

func (s RefreshStats) String() string {
	return fmt.Sprintf("added=%d updated=%d pruned=%d skipped=%d elapsed=%s",
		s.Added, s.Updated, s.Pruned, s.Skipped, s.Elapsed.Round(time.Millisecond))
}

// Refresh walks the enabled corpora using a background context.
func Refresh(heroDir string, model *Model, indexDB *sql.DB, graphDB *sql.DB, scope []string) (*RefreshStats, error) {
	return RefreshContext(context.Background(), heroDir, model, indexDB, graphDB, scope)
}

// RefreshContext walks the enabled corpora, reads stored hashes before
// embedding, and reconciles each complete authoritative corpus atomically.
func RefreshContext(
	ctx context.Context,
	heroDir string,
	model *Model,
	indexDB *sql.DB,
	graphDB *sql.DB,
	scope []string,
) (*RefreshStats, error) {
	if model == nil {
		return nil, fmt.Errorf("embedding model is unavailable")
	}
	return refreshWithEmbedder(ctx, heroDir, model, indexDB, graphDB, scope)
}

type embedder interface {
	Embed(string) []float32
}

func refreshWithEmbedder(
	ctx context.Context,
	heroDir string,
	model embedder,
	indexDB *sql.DB,
	graphDB *sql.DB,
	scope []string,
) (stats *RefreshStats, retErr error) {
	start := time.Now()
	stats = &RefreshStats{Corpora: make(map[string]CorpusRefreshStats)}
	defer func() { stats.Elapsed = time.Since(start) }()

	if model == nil {
		return stats, fmt.Errorf("embedding model is unavailable")
	}
	if indexDB == nil {
		return stats, fmt.Errorf("embedding index database is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}

	storage, err := OpenStorageContext(ctx, indexDB)
	if err != nil {
		return stats, fmt.Errorf("opening storage: %w", err)
	}

	seen := make(map[string]bool, len(scope))
	for _, corpus := range scope {
		if seen[corpus] {
			continue
		}
		seen[corpus] = true
		corpusStats := CorpusRefreshStats{Corpus: corpus}
		extraction, err := ExtractCorpus(ctx, heroDir, graphDB, corpus)
		if err != nil {
			corpusStats.Outcome = "partial"
			corpusStats.Reason = extraction.Reason
			stats.Corpora[corpus] = corpusStats
			return stats, fmt.Errorf("extracting %s chunks: %w", corpus, err)
		}
		if extraction.Unavailable {
			corpusStats.Outcome = "unavailable"
			corpusStats.Reason = extraction.Reason
			stats.Corpora[corpus] = corpusStats
			stats.Unavailable = true
			continue
		}
		if !extraction.Complete || !extraction.Authoritative {
			corpusStats.Outcome = "partial"
			corpusStats.Reason = extraction.Reason
			stats.Corpora[corpus] = corpusStats
			continue
		}
		if err := ctx.Err(); err != nil {
			corpusStats.Outcome = "partial"
			corpusStats.Reason = err.Error()
			stats.Corpora[corpus] = corpusStats
			return stats, err
		}

		keepIDs := make([]string, 0, len(extraction.Chunks))
		hashByID := make(map[string]string, len(extraction.Chunks))
		for _, tc := range extraction.Chunks {
			keepIDs = append(keepIDs, tc.ID)
			hashByID[tc.ID] = textHash(tc.Text)
		}
		storedHashes, err := storage.StoredHashes(ctx, keepIDs)
		if err != nil {
			corpusStats.Outcome = "partial"
			corpusStats.Reason = err.Error()
			stats.Corpora[corpus] = corpusStats
			return stats, fmt.Errorf("reading %s hashes: %w", corpus, err)
		}

		changed := make([]Chunk, 0, len(extraction.Chunks))
		for _, tc := range extraction.Chunks {
			hash := hashByID[tc.ID]
			if storedHash, exists := storedHashes[tc.ID]; exists && storedHash == hash {
				corpusStats.Skipped++
				continue
			}
			if err := ctx.Err(); err != nil {
				corpusStats.Outcome = "partial"
				corpusStats.Reason = err.Error()
				stats.Corpora[corpus] = corpusStats
				return stats, err
			}
			vector := model.Embed(tc.Text)
			if err := ctx.Err(); err != nil {
				corpusStats.Outcome = "partial"
				corpusStats.Reason = err.Error()
				stats.Corpora[corpus] = corpusStats
				return stats, err
			}
			changed = append(changed, Chunk{
				ID:       tc.ID,
				Corpus:   tc.Corpus,
				SourceID: tc.SourceID,
				Section:  tc.Section,
				TextHash: hash,
				Vector:   vector,
			})
			if _, exists := storedHashes[tc.ID]; exists {
				corpusStats.Updated++
			} else {
				corpusStats.Added++
			}
		}

		reconciled, err := storage.ReconcileCorpus(ctx, corpus, changed, keepIDs, true)
		if err != nil {
			corpusStats.Outcome = "partial"
			corpusStats.Reason = err.Error()
			stats.Corpora[corpus] = corpusStats
			return stats, fmt.Errorf("reconciling %s corpus: %w", corpus, err)
		}
		corpusStats.Pruned = reconciled.Pruned
		corpusStats.Outcome = "complete"
		stats.Corpora[corpus] = corpusStats
		stats.Added += corpusStats.Added
		stats.Updated += corpusStats.Updated
		stats.Pruned += corpusStats.Pruned
		stats.Skipped += corpusStats.Skipped
	}

	return stats, nil
}

// textHash returns the sha256 hex digest of the given text.
func textHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)
}
