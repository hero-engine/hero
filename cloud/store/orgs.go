package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Org represents a Hero Cloud organization.
type Org struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrgMember represents a user's membership in an org.
type OrgMember struct {
	OrgID    string    `json:"org_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
	// Joined from users table
	Email     string `json:"email,omitempty"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// CreateOrg creates a new organization and adds the creator as owner.
func (db *DB) CreateOrg(ctx context.Context, name, slug, ownerID string) (*Org, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var org Org
	err = tx.QueryRow(ctx, `
		INSERT INTO orgs (name, slug) VALUES ($1, $2)
		RETURNING id, name, slug, created_at, updated_at
	`, name, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating org: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO org_members (org_id, user_id, role) VALUES ($1, $2, 'owner')
	`, org.ID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("adding owner: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing: %w", err)
	}

	return &org, nil
}

// GetOrgByID retrieves an org by ID.
func (db *DB) GetOrgByID(ctx context.Context, id string) (*Org, error) {
	var org Org
	err := db.pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at FROM orgs WHERE id = $1
	`, id).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting org: %w", err)
	}
	return &org, nil
}

// GetOrgBySlug retrieves an org by slug.
func (db *DB) GetOrgBySlug(ctx context.Context, slug string) (*Org, error) {
	var org Org
	err := db.pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at FROM orgs WHERE slug = $1
	`, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting org: %w", err)
	}
	return &org, nil
}

// ListUserOrgs returns all orgs a user belongs to.
func (db *DB) ListUserOrgs(ctx context.Context, userID string) ([]Org, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT o.id, o.name, o.slug, o.created_at, o.updated_at
		FROM orgs o
		JOIN org_members om ON o.id = om.org_id
		WHERE om.user_id = $1
		ORDER BY o.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing orgs: %w", err)
	}
	defer rows.Close()

	var orgs []Org
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning org: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// GetMemberRole returns the user's role in an org, or empty string if not a member.
func (db *DB) GetMemberRole(ctx context.Context, orgID, userID string) (string, error) {
	var role string
	err := db.pool.QueryRow(ctx, `
		SELECT role FROM org_members WHERE org_id = $1 AND user_id = $2
	`, orgID, userID).Scan(&role)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting member role: %w", err)
	}
	return role, nil
}

// ListOrgMembers returns all members of an org.
func (db *DB) ListOrgMembers(ctx context.Context, orgID string) ([]OrgMember, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT om.org_id, om.user_id, om.role, om.joined_at,
		       u.email, u.name, u.avatar_url
		FROM org_members om
		JOIN users u ON om.user_id = u.id
		WHERE om.org_id = $1
		ORDER BY om.joined_at
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}
	defer rows.Close()

	var members []OrgMember
	for rows.Next() {
		var m OrgMember
		if err := rows.Scan(&m.OrgID, &m.UserID, &m.Role, &m.JoinedAt,
			&m.Email, &m.Name, &m.AvatarURL); err != nil {
			return nil, fmt.Errorf("scanning member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// AddOrgMember adds a user to an org with the specified role.
func (db *DB) AddOrgMember(ctx context.Context, orgID, userID, role string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO org_members (org_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (org_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, orgID, userID, role)
	if err != nil {
		return fmt.Errorf("adding member: %w", err)
	}
	return nil
}

// RemoveOrgMember removes a user from an org.
func (db *DB) RemoveOrgMember(ctx context.Context, orgID, userID string) error {
	_, err := db.pool.Exec(ctx, `
		DELETE FROM org_members WHERE org_id = $1 AND user_id = $2
	`, orgID, userID)
	if err != nil {
		return fmt.Errorf("removing member: %w", err)
	}
	return nil
}
