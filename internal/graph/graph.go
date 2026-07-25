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
	"strconv"
	"strings"
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
//     handful of related repos. Cross-repo entities and edges
//     are promoted here.
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

// CompiledSchemaVersion returns the schema version this binary was
// compiled against. Diagnostics (hero doctor, the MCP initialize stamp)
// use it to report which schema the running binary understands.
func CompiledSchemaVersion() string { return schemaVersion }

// ReadSchemaVersion returns the schema_version recorded in the graph at
// heroDir WITHOUT running migrations. Diagnostics need the graph's
// schema even when this binary is too old to migrate it (running Open
// would fail in that case). Returns "" with a nil error when no graph
// database exists yet.
func ReadSchemaVersion(heroDir string) (string, error) {
	dbPath := filepath.Join(heroDir, FileName)
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()
	var v string
	_ = db.QueryRow(`SELECT value FROM meta WHERE key = 'schema_version'`).Scan(&v)
	return v, nil
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
//
//	to nodes and edges so per-repo / per-unit sync filtering is
//	possible without scanning the props blob.
//
// v3: domain-scoped knowledge graph — adds the `domain` namespace
//
//	column on nodes and edges so PM / engineering / future packs
//	can coexist without silently mixing in shared queries. The
//	DEFAULT 'engineering' clause backfills every existing row in
//	place at ALTER time (SQLite renders the literal default at
//	read time), so the migration is invisible to engineering-only
//	workspaces.
//
// v4: graph conflict detection — adds `client_id` to nodes so
//
//	concurrent pushes from different install IDs can be detected
//	via FindGraphConflicts. DEFAULT '' is safe for existing rows
//	(they pre-date federation push and have no client provenance).
//
// v5: repo-scoped node identity — the live-row unique index moves from
//
//	(type, key) to (type, key, repo). Identity had ignored the
//	partition column added in v2, so a sibling repo's ingest of a
//	slug the local repo also owned matched the local live row,
//	saw a differing partition, and invalidated-and-reinserted it
//	under the sibling's repoKey. Widening a unique index cannot
//	fail on existing data.
const schemaVersion = "5"

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
			// Superseded by v5, which repo-scopes this. Kept as-is so the
			// v1 step still describes v1: migrations replay in order and
			// each must be valid at its own point in the timeline.
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
	{
		// Domain-scoped knowledge graph. Adds the `domain` namespace
		// column so PM / engineering / future packs can coexist without
		// silently mixing in shared queries. Default 'engineering'
		// backfills existing rows in place; engineering-only workspaces
		// see no behavior change.
		version: "3",
		statements: []string{
			`ALTER TABLE nodes ADD COLUMN domain TEXT NOT NULL DEFAULT 'engineering'`,
			`ALTER TABLE edges ADD COLUMN domain TEXT NOT NULL DEFAULT 'engineering'`,
			`CREATE INDEX IF NOT EXISTS idx_nodes_domain ON nodes(domain)`,
			`CREATE INDEX IF NOT EXISTS idx_edges_domain ON edges(domain)`,
		},
	},
	{
		// v4: client_id for graph conflict detection. Stores the install ID
		// of the client that pushed each node, enabling FindGraphConflicts
		// to detect concurrent pushes from different clients. DEFAULT ''
		// is safe for existing rows (they pre-date federation push).
		version: "4",
		statements: []string{
			`ALTER TABLE nodes ADD COLUMN client_id TEXT NOT NULL DEFAULT ''`,
			`CREATE INDEX IF NOT EXISTS idx_nodes_client ON nodes(client_id)`,
		},
	},
	{
		// v5: repo-scoped node identity. v1 made (type, key) unique among
		// live rows across EVERY repo partition, which predates the repo
		// column entirely (v2). The result was that federation could not
		// hold two repos' copies of one slug: ingesting a sibling's copy
		// tombstoned the local node and re-keyed it to the sibling, so
		// every reader filtering on the local repoKey found nothing.
		//
		// Dropping and recreating is safe on existing data — the new index
		// is strictly weaker, so any row set that satisfied (type, key)
		// also satisfies (type, key, repo).
		version: "5",
		statements: []string{
			`DROP INDEX IF EXISTS idx_nodes_current`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_current
				ON nodes(type, key, repo) WHERE valid_to IS NULL`,
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
				if isColumnAlreadyExists(err) {
					continue
				}
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

	warning, err := checkSchemaMismatch(schemaVersion, currentVersion)
	if err != nil {
		return err
	}
	if warning != "" {
		fmt.Fprint(os.Stderr, warning)
	}
	return nil
}

// schemaLess reports whether schema version a is older than b. Schema
// versions are numeric strings ("2".."4"..); a lexical compare inverts
// once the schema reaches double digits ("10" < "9" lexically), so parse
// to ints. A schema that doesn't parse falls back to a lexical compare —
// a safe last resort, since unknown values are rare and never expected in
// a Hero-managed graph.
func schemaLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	if aerr != nil || berr != nil {
		return a < b
	}
	return ai < bi
}

// checkSchemaMismatch reconciles the graph's stored schema (graphSchema)
// with this binary's compiled schema (binarySchema). It returns a
// warning to print when the graph is NEWER than the binary (tolerated —
// migrations were additive through v4, so an older binary can still read the extra
// columns) and an error when the binary is NEWER than the graph.
//
// Both messages name the running binary via os.Executable() so the
// caller can see WHICH hero is complaining, and point at `hero doctor`
// rather than the misleading `hero upgrade` — the reported bug is a
// stray binary on PATH reading a current graph, which `hero upgrade`
// (a workspace-file operation) cannot fix.
func checkSchemaMismatch(binarySchema, graphSchema string) (warning string, err error) {
	if graphSchema == binarySchema {
		return "", nil
	}
	exe, _ := os.Executable()
	if exe == "" {
		exe = "unknown"
	}
	if schemaLess(binarySchema, graphSchema) {
		// Stays a warning rather than a hard error: an older binary can still
		// READ a newer graph, and graph.db is a regenerable cache.
		//
		// NOTE: v5 is the first non-additive step — it changed node identity
		// from (type, key) to (type, key, repo) — so a pre-v5 binary WRITING
		// here reintroduces cross-partition tombstoning. This function cannot
		// warn about that: binarySchema is always the compile-time constant,
		// and the binary that would do the damage is running its own older
		// copy of this code. A real guard has to live on the write path of a
		// v5+ binary, or in provenance on the rows themselves. Tracked in
		// graph-node-identity-repo-scoped's follow-ups.
		return fmt.Sprintf(
			"Warning: graph schema is newer than this hero binary.\n"+
				"  running binary: %s (schema %s)\n"+
				"  graph schema:   %s\n"+
				"This is almost always the WRONG hero binary on PATH, not a workspace problem.\n"+
				"Run `hero doctor` to see which binary your shell/harness resolves and how to fix it.\n"+
				"(`hero upgrade` will NOT help — it updates workspace files, not this binary.)\n",
			exe, binarySchema, graphSchema), nil
	}
	return "", fmt.Errorf(
		"graph schema version mismatch — this hero binary is newer than the workspace graph.\n"+
			"  running binary: %s (schema %s)\n"+
			"  graph schema:   %s\n"+
			"This is almost always the WRONG hero binary on PATH, not a workspace problem.\n"+
			"Run `hero doctor` to see which binary your shell/harness resolves and how to fix it.\n"+
			"(`hero upgrade` will NOT help — it updates workspace files, not this binary.)",
		exe, binarySchema, graphSchema)
}

// isColumnAlreadyExists returns true if err is SQLite's "duplicate
// column name" error, which happens when ALTER TABLE ADD COLUMN retries
// after a partially-applied migration.
func isColumnAlreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
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
