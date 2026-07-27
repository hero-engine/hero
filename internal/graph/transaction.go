package graph

import (
	"context"
	"database/sql"
	"fmt"
)

// Tx is a narrow transaction-scoped graph writer. It intentionally exposes
// only the mutations needed by aggregate writers that require all-or-nothing
// bitemporal updates.
type Tx struct {
	tx  *sql.Tx
	ctx context.Context
}

// WithTransaction runs fn in one graph transaction. Cancellation rolls back
// every node and edge mutation performed through Tx.
func (s *Store) WithTransaction(ctx context.Context, fn func(*Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin graph transaction: %w", err)
	}
	defer tx.Rollback()
	if err := fn(&Tx{tx: tx, ctx: ctx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit graph transaction: %w", err)
	}
	return nil
}

// UpsertNode applies the Store's canonical node identity and invalidation rules
// inside the surrounding transaction.
func (tx *Tx) UpsertNode(n *Node) (int64, error) {
	return upsertNode(tx.ctx, tx.tx, n)
}

// UpsertEdge applies the Store's canonical edge identity and invalidation rules
// inside the surrounding transaction.
func (tx *Tx) UpsertEdge(e *Edge) (int64, error) {
	return upsertEdge(tx.ctx, tx.tx, e)
}

// QueryContext supports narrowly-scoped reconciliation queries without
// exposing the transaction implementation.
func (tx *Tx) QueryContext(query string, args ...any) (*sql.Rows, error) {
	return tx.tx.QueryContext(tx.ctx, query, args...)
}

// InvalidateNodeByID retires one already-resolved current node and all current
// incident edges using the graph's bitemporal invalidation semantics.
func (tx *Tx) InvalidateNodeByID(id int64, at string) (int64, error) {
	edges, err := tx.tx.ExecContext(tx.ctx,
		`UPDATE edges SET valid_to = ?
		  WHERE valid_to IS NULL AND (from_id = ? OR to_id = ?)`,
		at, id, id,
	)
	if err != nil {
		return 0, fmt.Errorf("invalidating incident edges: %w", err)
	}
	node, err := tx.tx.ExecContext(tx.ctx,
		`UPDATE nodes SET valid_to = ? WHERE id = ? AND valid_to IS NULL`,
		at, id,
	)
	if err != nil {
		return 0, fmt.Errorf("invalidating node: %w", err)
	}
	if n, _ := node.RowsAffected(); n == 0 {
		return 0, ErrNotFound
	}
	n, err := edges.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("counting invalidated edges: %w", err)
	}
	return n, nil
}
