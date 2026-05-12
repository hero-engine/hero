// Package graph provides hero's unified knowledge graph store.
//
// The graph holds three subgraphs in one substrate: code (packages,
// symbols, imports), work (features, sessions, commits, decisions),
// and knowledge (notes, documents, memories). Markdown surfaces in
// .hero/ are projections; the graph is the source of truth for
// everything except hand-authored plans and notes.
//
// Bitemporal: every node and edge carries valid_from / valid_to
// (world time) and ingested_at (when hero learned the fact). Updates
// invalidate prior rows rather than overwriting them, so history is
// preserved and concurrent edits are resolvable as conflicts rather
// than silent overwrites.
package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// FileName is the default database filename inside .hero/.
const FileName = "graph.db"

// Scope describes the visibility (and sync routing) of a node or edge.
//
// Federation model (see graph-memory-federation/spec.md):
//   - local:  per-developer, never leaves the machine.
//   - team:   per-repo team graph; default for ingested data.
//   - unit:   business-unit / product-line join graph spanning a
//             handful of related repos. Cross-repo entities and edges
//             are promoted here.
//   - public: cross-org / open-source patterns and shared learnings.
//
// Org-wide scope was considered and rejected: at 1000+ repos with
// varied business units, "the whole org" is structurally noisy.
type Scope string

const (
	ScopeLocal  Scope = "local"
	ScopeTeam   Scope = "team"
	ScopeUnit   Scope = "unit"
	ScopePublic Scope = "public"
)

// Store wraps the SQLite database backing the knowledge graph.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (or creates) the graph database under heroDir/graph.db.
func Open(heroDir string) (*Store, error) {
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating hero directory: %w", err)
	}
	dbPath := filepath.Join(heroDir, FileName)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening graph database: %w", err)
	}

	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", pragma, err)
		}
	}

	s := &Store{db: db, path: dbPath}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating graph schema: %w", err)
	}
	return s, nil
}

// Close closes the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

// Path returns the on-disk location of the database file.
func (s *Store) Path() string { return s.path }

// DB exposes the underlying *sql.DB for advanced queries. Most callers
// should use the typed helpers in this package instead.
func (s *Store) DB() *sql.DB { return s.db }

// schemaVersion is bumped when the migration list changes.
//
// v1: initial schema (nodes, edges, sync_state, meta).
// v2: federation contracts — adds `repo` and `unit` partition columns
//     to nodes and edges so per-repo / per-unit sync filtering is
//     possible without scanning the props blob.
const schemaVersion = "2"

// migration is one ordered, idempotent step in the schema timeline.
// version is the resulting schema_version after the step applies.
type migration struct {
	version    string
	statements []string
}

// migrations are applied in order. Each step's `statements` must be
// safe to run on a fresh empty database AND to apply against a db at
// the immediately-prior version. Use IF NOT EXISTS / OR IGNORE
// liberally so re-running on an up-to-date db is a no-op.
var migrations = []migration{
	{
		version: "1",
		statements: []string{
			`CREATE TABLE IF NOT EXISTS nodes (
				id           INTEGER PRIMARY KEY AUTOINCREMENT,
				type         TEXT    NOT NULL,
				key          TEXT    NOT NULL,
				props        TEXT    NOT NULL,
				scope        TEXT    NOT NULL,
				content_hash TEXT,
				source       TEXT    NOT NULL,
				valid_from   TEXT    NOT NULL,
				valid_to     TEXT,
				ingested_at  TEXT    NOT NULL
			)`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_current
				ON nodes(type, key) WHERE valid_to IS NULL`,
			`CREATE INDEX IF NOT EXISTS idx_nodes_type     ON nodes(type)`,
			`CREATE INDEX IF NOT EXISTS idx_nodes_scope    ON nodes(scope)`,
			`CREATE INDEX IF NOT EXISTS idx_nodes_ingested ON nodes(ingested_at)`,
			`CREATE TABLE IF NOT EXISTS edges (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				from_id     INTEGER NOT NULL REFERENCES nodes(id),
				to_id       INTEGER NOT NULL REFERENCES nodes(id),
				type        TEXT    NOT NULL,
				props       TEXT    NOT NULL DEFAULT '{}',
				scope       TEXT    NOT NULL,
				source      TEXT    NOT NULL,
				valid_from  TEXT    NOT NULL,
				valid_to    TEXT,
				ingested_at TEXT    NOT NULL
			)`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_edges_current
				ON edges(from_id, type, to_id) WHERE valid_to IS NULL`,
			`CREATE INDEX IF NOT EXISTS idx_edges_from     ON edges(from_id, type)`,
			`CREATE INDEX IF NOT EXISTS idx_edges_to       ON edges(to_id, type)`,
			`CREATE INDEX IF NOT EXISTS idx_edges_ingested ON edges(ingested_at)`,
			`CREATE TABLE IF NOT EXISTS sync_state (
				server_url       TEXT PRIMARY KEY,
				last_push_at     TEXT,
				last_pull_at     TEXT,
				last_pull_cursor TEXT,
				org              TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS meta (
				key   TEXT PRIMARY KEY,
				value TEXT NOT NULL
			)`,
		},
	},
	{
		// Federation contracts. Adds repo/unit partition columns so the
		// sync layer can filter by partition without parsing props.
		version: "2",
		statements: []string{
			`ALTER TABLE nodes ADD COLUMN repo TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE nodes ADD COLUMN unit TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE edges ADD COLUMN repo TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE edges ADD COLUMN unit TEXT NOT NULL DEFAULT ''`,
			`CREATE INDEX IF NOT EXISTS idx_nodes_repo ON nodes(repo)`,
			`CREATE INDEX IF NOT EXISTS idx_nodes_unit ON nodes(unit)`,
			`CREATE INDEX IF NOT EXISTS idx_edges_repo ON edges(repo)`,
			`CREATE INDEX IF NOT EXISTS idx_edges_unit ON edges(unit)`,
		},
	},
}

func (s *Store) migrate() error {
	// Ensure meta exists so we can read schema_version. The meta table
	// is created by the v1 migration if not already present, but on a
	// fresh db we need it before we can read.
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("ensuring meta table: %w", err)
	}

	currentVersion := ""
	_ = s.db.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&currentVersion)

	applied := false
	for _, m := range migrations {
		if currentVersion >= m.version {
			continue
		}
		for _, stmt := range m.statements {
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration v%s: %w", m.version, err)
			}
		}
		if _, err := s.db.Exec(
			`INSERT INTO meta(key, value) VALUES ('schema_version', ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			m.version,
		); err != nil {
			return fmt.Errorf("recording schema_version=%s: %w", m.version, err)
		}
		currentVersion = m.version
		applied = true
	}
	_ = applied // reserved for future log/telemetry

	// Seed install_id if missing.
	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO meta(key, value)
		 VALUES ('install_id', lower(hex(randomblob(8))))`,
	); err != nil {
		return fmt.Errorf("seeding install_id: %w", err)
	}

	if currentVersion != schemaVersion {
		return fmt.Errorf("graph schema version mismatch: db=%s binary=%s", currentVersion, schemaVersion)
	}
	return nil
}

// nowRFC3339 returns the current UTC time formatted for graph timestamps.
func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

// jsonOrEmpty marshals v to JSON. nil maps/slices/structs serialize to
// "null", which the schema disallows for required JSON columns; this
// helper returns "{}" or "[]"-style empty defaults instead.
func jsonOrEmpty(v any, empty string) (string, error) {
	if v == nil {
		return empty, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if string(b) == "null" {
		return empty, nil
	}
	return string(b), nil
}
