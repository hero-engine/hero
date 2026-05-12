// Package store provides the data access layer for Hero Cloud,
// backed by YugabyteDB (Postgres-compatible).
package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a YugabyteDB connection pool and provides data access methods.
type DB struct {
	pool *pgxpool.Pool
}

// Querier is the minimal SQL surface satisfied by both *pgxpool.Pool and
// *pgxpool.Conn. Store methods accept it so the same code path works for
// pool-level (cross-tenant, e.g. auth) and request-scoped (RLS-enforced)
// connections.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

type connKeyType struct{}

var connKey = connKeyType{}

// Conn returns the request-scoped connection stashed in ctx by WithOrg,
// falling back to the pool if none is set. Use this in store methods
// instead of db.pool directly so that RLS-bound sessions are honored
// when present.
func (db *DB) Conn(ctx context.Context) Querier {
	if c, ok := ctx.Value(connKey).(*pgxpool.Conn); ok {
		return c
	}
	return db.pool
}

// WithOrg acquires a dedicated connection from the pool, sets
//
//	app.org_id = <orgID>
//
// on it, and returns a context carrying that connection plus a release
// function the caller MUST defer. Subsequent store calls that use
// db.Conn(ctx) will run on the bound session and so be subject to the
// org_isolation RLS policies.
func (db *DB) WithOrg(ctx context.Context, orgID string) (context.Context, func(), error) {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return ctx, func() {}, fmt.Errorf("acquire conn: %w", err)
	}
	if _, err := conn.Exec(ctx, "SET app.org_id = $1", orgID); err != nil {
		conn.Release()
		return ctx, func() {}, fmt.Errorf("set app.org_id: %w", err)
	}
	release := func() {
		// Best-effort reset so a recycled pool conn doesn't leak the binding.
		_, _ = conn.Exec(context.Background(), "RESET app.org_id")
		conn.Release()
	}
	return context.WithValue(ctx, connKey, conn), release, nil
}

// Pool returns the raw connection pool. Reserved for the following
// store areas, which intentionally do NOT use db.Conn(ctx) because they
// either run before an org context exists, span orgs by design, or
// touch tables that are not subject to RLS:
//
//   - users.go          — auth flows; users live outside the tenant boundary
//   - orgs.go           — orgs / members / teams; org table itself is the boundary
//   - repos.go          — repos table is RLS-exempt and joins to orgs
//   - installations.go  — github_installations webhook setup; no org context yet
//   - intelligence.go   — cross-org anonymized aggregates by design
//   - migrations / Close — schema management
//
// Anything else that touches a per-tenant table (graph_*, specs,
// knowledge, conventions, activity_events, pr_checks) MUST use
// db.Conn(ctx) so the RLS-bound session is honored.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Connect establishes a connection pool to YugabyteDB.
func Connect(ctx context.Context, connString string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	// Reasonable defaults for a cloud service
	config.MaxConns = 20
	config.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.pool.Close()
}

// Migrate runs all pending database migrations in order.
func (db *DB) Migrate(ctx context.Context) error {
	// Create migrations tracking table
	if _, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name    TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	for _, m := range migrations {
		var exists bool
		err := db.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			m.Version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking migration %d: %w", m.Version, err)
		}
		if exists {
			continue
		}

		// Run migration — split on semicolons for multi-statement
		for _, stmt := range splitStatements(m.SQL) {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("migration %d (%s): %w", m.Version, m.Name, err)
			}
		}

		// Record it
		if _, err := db.pool.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			m.Version, m.Name,
		); err != nil {
			return fmt.Errorf("recording migration %d: %w", m.Version, err)
		}
	}

	return nil
}

// splitStatements splits SQL on semicolons, respecting basic quoting and
// skipping `--` line comments (so a semicolon in a comment doesn't terminate
// the statement). Block comments /* ... */ are not handled — avoid them in
// migrations.
func splitStatements(sql string) []string {
	var stmts []string
	var current strings.Builder
	inQuote := false
	inComment := false
	quoteChar := byte(0)

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if inComment {
			current.WriteByte(ch)
			if ch == '\n' {
				inComment = false
			}
			continue
		}
		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar {
				inQuote = false
			}
			continue
		}
		if ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			inComment = true
			current.WriteByte(ch)
			continue
		}
		if ch == '\'' || ch == '"' {
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
			continue
		}
		if ch == ';' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				stmts = append(stmts, s)
			}
			current.Reset()
			continue
		}
		current.WriteByte(ch)
	}

	s := strings.TrimSpace(current.String())
	if s != "" && !commentOnly(s) {
		stmts = append(stmts, s)
	}
	return stmts
}

// commentOnly reports whether s is entirely whitespace and -- comments.
// We use this to drop trailing comment blocks after the last semicolon
// in a migration so they aren't sent to the server as SQL.
func commentOnly(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "--") {
			continue
		}
		return false
	}
	return true
}
