package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Repo represents a repository linked to an org.
type Repo struct {
	ID         string     `json:"id"`
	OrgID      string     `json:"org_id"`
	Name       string     `json:"name"`
	PushURL    string     `json:"push_url"`
	LastSyncAt *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateRepo creates a new repo in an org.
func (db *DB) CreateRepo(ctx context.Context, orgID, name, pushURL string) (*Repo, error) {
	var r Repo
	err := db.pool.QueryRow(ctx, `
		INSERT INTO repos (org_id, name, push_url)
		VALUES ($1, $2, $3)
		RETURNING id, org_id, name, push_url, last_sync_at, created_at
	`, orgID, name, pushURL).Scan(
		&r.ID, &r.OrgID, &r.Name, &r.PushURL, &r.LastSyncAt, &r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating repo: %w", err)
	}
	return &r, nil
}

// GetRepoByID retrieves a repo by ID, scoped to an org.
func (db *DB) GetRepoByID(ctx context.Context, orgID, repoID string) (*Repo, error) {
	var r Repo
	err := db.pool.QueryRow(ctx, `
		SELECT id, org_id, name, push_url, last_sync_at, created_at
		FROM repos WHERE id = $1 AND org_id = $2
	`, repoID, orgID).Scan(
		&r.ID, &r.OrgID, &r.Name, &r.PushURL, &r.LastSyncAt, &r.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting repo: %w", err)
	}
	return &r, nil
}

// ListOrgRepos returns all repos in an org.
func (db *DB) ListOrgRepos(ctx context.Context, orgID string) ([]Repo, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, org_id, name, push_url, last_sync_at, created_at
		FROM repos WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("listing repos: %w", err)
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.OrgID, &r.Name, &r.PushURL, &r.LastSyncAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning repo: %w", err)
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// TouchRepoSyncTime updates the last_sync_at timestamp for a repo.
func (db *DB) TouchRepoSyncTime(ctx context.Context, repoID string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE repos SET last_sync_at = now() WHERE id = $1
	`, repoID)
	if err != nil {
		return fmt.Errorf("updating sync time: %w", err)
	}
	return nil
}
