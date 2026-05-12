// Package refs implements the session-scoped ref-store backing
// two-tier MCP responses. A ref is a stable handle returned by a
// read-side tool that points to either cached content or the args
// needed to re-fetch it. Callers expand a ref via hero_expand to
// rehydrate the full content on demand, keeping context lean.
//
// Phase 1: additive. Read-side tools opt in by registering refs;
// callers opt in by passing `compact: true`. Behavior without opt-in
// is unchanged.
package refs

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Kind classifies a ref by its source. Stable kinds (spec, convention,
// decision, rule) resolve identically across sessions; query kinds
// (search, context, recap, why, blocked, feed) are session-scoped.
type Kind string

const (
	KindSpec       Kind = "spec"
	KindConvention Kind = "convention"
	KindDecision   Kind = "decision"
	KindRule       Kind = "rule"
	KindSearch     Kind = "search"
	KindContext    Kind = "context"
	KindRecap      Kind = "recap"
	KindWhy        Kind = "why"
	KindBlocked    Kind = "blocked"
	KindFeed       Kind = "feed"
)

// IsShareable reports whether refs of this kind resolve identically
// across sessions. Shareable kinds use deterministic IDs.
func (k Kind) IsShareable() bool {
	switch k {
	case KindSpec, KindConvention, KindDecision, KindRule:
		return true
	}
	return false
}

// defaultTTL returns the per-kind TTL for cached entries. Shareable
// kinds live longer because their content is more stable.
func (k Kind) defaultTTL() time.Duration {
	switch k {
	case KindSpec, KindConvention, KindDecision, KindRule:
		return 24 * time.Hour
	case KindSearch, KindContext:
		return 1 * time.Hour
	case KindRecap, KindWhy, KindBlocked, KindFeed:
		return 2 * time.Hour
	}
	return 1 * time.Hour
}

// Entry is a single ref-store row.
type Entry struct {
	RefID             string
	Kind              Kind
	Slug              string
	Scope             string
	SourceArgsJSON    string
	Content           string
	SourceFingerprint string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

// Store is a SQLite-backed session ref store.
type Store struct {
	db        *sql.DB
	path      string
	sessionID string

	mu       sync.Mutex
	hits     int
	misses   int
	refetch  int
	registers int
}

// SessionID derives a stable session identifier from cwd and the
// current PID. It is intentionally simple; the Curator (in
// active-context-management) will replace this with a richer scheme.
func SessionID(cwd string, pid int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%d", cwd, pid)))
	return hex.EncodeToString(h[:8])
}

// Open creates or opens the ref-store for the given hero workspace
// and session. Storage path: heroDir/sessions/<id>/refs.db
func Open(heroDir, sessionID string) (*Store, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	dir := filepath.Join(heroDir, "sessions", sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating session dir: %w", err)
	}
	dbPath := filepath.Join(dir, "refs.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening refs db: %w", err)
	}

	s := &Store{db: db, path: dbPath, sessionID: sessionID}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrating refs schema: %w", err)
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path returns the on-disk location of the database (for diagnostics).
func (s *Store) Path() string { return s.path }

// SessionID returns the session ID this store is scoped to.
func (s *Store) SessionID() string { return s.sessionID }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS refs (
			ref_id              TEXT PRIMARY KEY,
			kind                TEXT NOT NULL,
			slug                TEXT NOT NULL,
			scope               TEXT NOT NULL,
			source_args_json    TEXT NOT NULL,
			content             TEXT,
			source_fingerprint  TEXT,
			created_at          INTEGER NOT NULL,
			expires_at          INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS refs_kind_slug ON refs(kind, slug)`,
		`CREATE INDEX IF NOT EXISTS refs_expires ON refs(expires_at)`,
		`CREATE TABLE IF NOT EXISTS metrics (
			key   TEXT PRIMARY KEY,
			value INTEGER NOT NULL
		)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("exec %q: %w", q, err)
		}
	}
	return nil
}

// BuildRefID builds a stable ref ID from kind, slug, and scope.
// Shareable kinds use the deterministic form `<kind>:<slug>:<scope>`.
// Query kinds get a session-scoped suffix to prevent cross-session
// collisions.
func (s *Store) BuildRefID(kind Kind, slug, scope string) string {
	if kind.IsShareable() {
		return fmt.Sprintf("%s:%s:%s", kind, slug, scope)
	}
	// Hash the slug+scope+session to keep ids short and unique.
	h := sha256.Sum256([]byte(s.sessionID + "|" + slug + "|" + scope))
	return fmt.Sprintf("%s:%s:%s", kind, hex.EncodeToString(h[:6]), scope)
}

// Register inserts or updates an entry. Content may be empty; in that
// case the resolver will be invoked at expand time. Returns the ref
// ID (stable for shareable kinds, session-scoped otherwise).
func (s *Store) Register(kind Kind, slug, scope string, sourceArgs map[string]any, content, fingerprint string) (string, error) {
	if kind == "" {
		return "", fmt.Errorf("kind is required")
	}
	if slug == "" {
		return "", fmt.Errorf("slug is required")
	}
	if scope == "" {
		scope = "full"
	}
	argsJSON, err := json.Marshal(sourceArgs)
	if err != nil {
		return "", fmt.Errorf("marshalling source args: %w", err)
	}
	now := time.Now()
	expires := now.Add(kind.defaultTTL())

	refID := s.BuildRefID(kind, slug, scope)

	_, err = s.db.Exec(
		`INSERT INTO refs (ref_id, kind, slug, scope, source_args_json, content, source_fingerprint, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(ref_id) DO UPDATE SET
			source_args_json   = excluded.source_args_json,
			content            = excluded.content,
			source_fingerprint = excluded.source_fingerprint,
			created_at         = excluded.created_at,
			expires_at         = excluded.expires_at`,
		refID, string(kind), slug, scope, string(argsJSON), content, fingerprint, now.Unix(), expires.Unix(),
	)
	if err != nil {
		return "", fmt.Errorf("inserting ref: %w", err)
	}
	s.bump(&s.registers)
	return refID, nil
}

// Lookup returns the entry for refID or nil if not found.
func (s *Store) Lookup(refID string) (*Entry, error) {
	row := s.db.QueryRow(
		`SELECT ref_id, kind, slug, scope, source_args_json, content, source_fingerprint, created_at, expires_at
		 FROM refs WHERE ref_id = ?`, refID)
	var e Entry
	var kindStr string
	var content sql.NullString
	var fingerprint sql.NullString
	var created, expires int64
	if err := row.Scan(&e.RefID, &kindStr, &e.Slug, &e.Scope, &e.SourceArgsJSON, &content, &fingerprint, &created, &expires); err != nil {
		if err == sql.ErrNoRows {
			s.bump(&s.misses)
			return nil, nil
		}
		return nil, fmt.Errorf("looking up ref: %w", err)
	}
	e.Kind = Kind(kindStr)
	if content.Valid {
		e.Content = content.String
	}
	if fingerprint.Valid {
		e.SourceFingerprint = fingerprint.String
	}
	e.CreatedAt = time.Unix(created, 0)
	e.ExpiresAt = time.Unix(expires, 0)
	s.bump(&s.hits)
	return &e, nil
}

// UpdateContent replaces an entry's cached content and fingerprint.
// Used when a stale fingerprint forced a re-fetch.
func (s *Store) UpdateContent(refID, content, fingerprint string) error {
	now := time.Now()
	expires := now.Add(time.Hour) // refresh window; per-kind TTL re-applied on next Register
	res, err := s.db.Exec(
		`UPDATE refs SET content = ?, source_fingerprint = ?, expires_at = ? WHERE ref_id = ?`,
		content, fingerprint, expires.Unix(), refID,
	)
	if err != nil {
		return fmt.Errorf("updating ref content: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("ref %q not found", refID)
	}
	s.bump(&s.refetch)
	return nil
}

// Prune deletes entries whose expires_at is past. Returns the number
// of deleted rows.
func (s *Store) Prune() (int, error) {
	res, err := s.db.Exec(`DELETE FROM refs WHERE expires_at < ?`, time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("pruning refs: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Metrics returns a snapshot of in-memory counters since the store
// was opened. Persistent counters are also written so `hero pulse
// --refs` can read them across restarts.
type Metrics struct {
	Registers int `json:"registers"`
	Hits      int `json:"hits"`
	Misses    int `json:"misses"`
	Refetch   int `json:"refetch"`
}

// Metrics returns counter snapshot.
func (s *Store) Metrics() Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Metrics{
		Registers: s.registers,
		Hits:      s.hits,
		Misses:    s.misses,
		Refetch:   s.refetch,
	}
}

// PersistMetrics flushes in-memory counters into the metrics table so
// later processes (`hero pulse --refs`) can read them.
func (s *Store) PersistMetrics() error {
	m := s.Metrics()
	stmts := []struct {
		key string
		val int
	}{
		{"registers", m.Registers},
		{"hits", m.Hits},
		{"misses", m.Misses},
		{"refetch", m.Refetch},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, p := range stmts {
		_, err := tx.Exec(
			`INSERT INTO metrics(key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = metrics.value + excluded.value`,
			p.key, p.val,
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Reset in-memory counters so we don't double-persist.
	s.mu.Lock()
	s.registers, s.hits, s.misses, s.refetch = 0, 0, 0, 0
	s.mu.Unlock()
	return nil
}

// PersistedMetrics reads cumulative counters from the metrics table.
// Used by `hero pulse --refs` to summarise a session.
func (s *Store) PersistedMetrics() (Metrics, error) {
	rows, err := s.db.Query(`SELECT key, value FROM metrics`)
	if err != nil {
		return Metrics{}, err
	}
	defer rows.Close()
	out := Metrics{}
	for rows.Next() {
		var k string
		var v int
		if err := rows.Scan(&k, &v); err != nil {
			return Metrics{}, err
		}
		switch k {
		case "registers":
			out.Registers = v
		case "hits":
			out.Hits = v
		case "misses":
			out.Misses = v
		case "refetch":
			out.Refetch = v
		}
	}
	return out, rows.Err()
}

func (s *Store) bump(field *int) {
	s.mu.Lock()
	*field++
	s.mu.Unlock()
}

// ParseRefID splits a ref ID into its component parts. Returns an
// error if the format is unrecognised.
func ParseRefID(refID string) (kind Kind, slug, scope string, err error) {
	parts := strings.SplitN(refID, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("ref id %q must have form <kind>:<slug>:<scope>", refID)
	}
	return Kind(parts[0]), parts[1], parts[2], nil
}
