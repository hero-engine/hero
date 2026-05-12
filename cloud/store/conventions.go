package store

import (
	"context"
	"fmt"
	"time"
)

// Convention represents a synced convention from a repo.
type Convention struct {
	ID        string    `json:"id"`
	RepoID    string    `json:"repo_id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // draft, active
	Scope     []string  `json:"scope"`  // glob patterns
	Content   string    `json:"content"`
	Checksum  string    `json:"checksum"`
	SyncedAt  time.Time `json:"synced_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpsertConvention creates or updates a convention.
func (db *DB) UpsertConvention(ctx context.Context, conv *Convention) error {
	_, err := db.Conn(ctx).Exec(ctx, `
		INSERT INTO conventions (org_id, repo_id, slug, title, status, scope, content, checksum, synced_at)
		SELECT (SELECT org_id FROM repos WHERE id = $1),
		       $1, $2, $3, $4, $5, $6, $7, now()
		ON CONFLICT (repo_id, slug) DO UPDATE SET
			title = EXCLUDED.title,
			status = EXCLUDED.status,
			scope = EXCLUDED.scope,
			content = EXCLUDED.content,
			checksum = EXCLUDED.checksum,
			synced_at = now(),
			updated_at = now()
	`, conv.RepoID, conv.Slug, conv.Title, conv.Status, conv.Scope, conv.Content, conv.Checksum)
	return err
}

// GetActiveConventionsByOrg returns all active conventions for repos in an org.
func (db *DB) GetActiveConventionsByOrg(ctx context.Context, orgID string) ([]Convention, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT c.id, c.repo_id, c.slug, c.title, c.status, c.scope, c.content, c.checksum, c.synced_at, c.created_at, c.updated_at
		FROM conventions c
		JOIN repos r ON c.repo_id = r.id
		WHERE r.org_id = $1 AND c.status = 'active'
		ORDER BY c.slug
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conventions []Convention
	for rows.Next() {
		var c Convention
		if err := rows.Scan(
			&c.ID, &c.RepoID, &c.Slug, &c.Title, &c.Status, &c.Scope,
			&c.Content, &c.Checksum, &c.SyncedAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		conventions = append(conventions, c)
	}
	return conventions, nil
}

// GetActiveConventionsByRepo returns all active conventions for a specific repo.
func (db *DB) GetActiveConventionsByRepo(ctx context.Context, repoID string) ([]Convention, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT id, repo_id, slug, title, status, scope, content, checksum, synced_at, created_at, updated_at
		FROM conventions
		WHERE repo_id = $1 AND status = 'active'
		ORDER BY slug
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conventions []Convention
	for rows.Next() {
		var c Convention
		if err := rows.Scan(
			&c.ID, &c.RepoID, &c.Slug, &c.Title, &c.Status, &c.Scope,
			&c.Content, &c.Checksum, &c.SyncedAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		conventions = append(conventions, c)
	}
	return conventions, nil
}

// ListOrgConventions returns all conventions for repos in an org, optionally filtered.
func (db *DB) ListOrgConventions(ctx context.Context, orgID string, repoID, query string) ([]Convention, error) {
	q := `
		SELECT c.id, c.repo_id, c.slug, c.title, c.status, c.scope, c.content, c.checksum, c.synced_at, c.created_at, c.updated_at
		FROM conventions c
		JOIN repos r ON c.repo_id = r.id
		WHERE r.org_id = $1`
	args := []any{orgID}
	argN := 2

	if repoID != "" {
		q += fmt.Sprintf(" AND c.repo_id = $%d", argN)
		args = append(args, repoID)
		argN++
	}
	if query != "" {
		q += fmt.Sprintf(" AND (c.title ILIKE $%d OR c.slug ILIKE $%d OR c.content ILIKE $%d)", argN, argN, argN)
		args = append(args, "%"+query+"%")
		argN++
	}

	q += " ORDER BY c.slug"

	rows, err := db.Conn(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conventions []Convention
	for rows.Next() {
		var c Convention
		if err := rows.Scan(
			&c.ID, &c.RepoID, &c.Slug, &c.Title, &c.Status, &c.Scope,
			&c.Content, &c.Checksum, &c.SyncedAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		conventions = append(conventions, c)
	}
	return conventions, nil
}
