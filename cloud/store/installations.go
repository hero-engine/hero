package store

import (
	"context"
	"time"
)

// Installation represents a GitHub App installation on an org/user account.
type Installation struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"org_id"`
	InstallationID int64     `json:"installation_id"` // GitHub's installation ID
	AccountLogin   string    `json:"account_login"`
	AccountType    string    `json:"account_type"` // "Organization" or "User"
	GovernanceMode string    `json:"governance_mode"` // advisory, enforcement, disabled
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PRCheck represents a spec-linkage check performed on a pull request.
type PRCheck struct {
	ID             string    `json:"id"`
	InstallationID string    `json:"installation_id"`
	RepoFullName   string    `json:"repo_full_name"`
	PRNumber       int       `json:"pr_number"`
	HeadSHA        string    `json:"head_sha"`
	SpecSlugs      []string  `json:"spec_slugs"`
	HasSpec        bool      `json:"has_spec"`
	Conclusion     string    `json:"conclusion"` // success, failure, neutral
	CreatedAt      time.Time `json:"created_at"`
}

// UpsertInstallation creates or updates a GitHub App installation record.
func (db *DB) UpsertInstallation(ctx context.Context, inst *Installation) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO github_installations (org_id, installation_id, account_login, account_type, governance_mode)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (installation_id) DO UPDATE SET
			org_id = EXCLUDED.org_id,
			account_login = EXCLUDED.account_login,
			account_type = EXCLUDED.account_type,
			updated_at = now()
	`, inst.OrgID, inst.InstallationID, inst.AccountLogin, inst.AccountType, inst.GovernanceMode)
	return err
}

// DeleteInstallation removes an installation record.
func (db *DB) DeleteInstallation(ctx context.Context, installationID int64) error {
	_, err := db.pool.Exec(ctx, `
		DELETE FROM github_installations WHERE installation_id = $1
	`, installationID)
	return err
}

// GetInstallationByGitHubID retrieves an installation by GitHub's installation ID.
func (db *DB) GetInstallationByGitHubID(ctx context.Context, installationID int64) (*Installation, error) {
	var inst Installation
	err := db.pool.QueryRow(ctx, `
		SELECT id, org_id, installation_id, account_login, account_type, governance_mode, created_at, updated_at
		FROM github_installations WHERE installation_id = $1
	`, installationID).Scan(
		&inst.ID, &inst.OrgID, &inst.InstallationID, &inst.AccountLogin,
		&inst.AccountType, &inst.GovernanceMode, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// GetInstallationByOrg retrieves installations for an org.
func (db *DB) GetInstallationByOrg(ctx context.Context, orgID string) ([]Installation, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, org_id, installation_id, account_login, account_type, governance_mode, created_at, updated_at
		FROM github_installations WHERE org_id = $1
		ORDER BY created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []Installation
	for rows.Next() {
		var inst Installation
		if err := rows.Scan(
			&inst.ID, &inst.OrgID, &inst.InstallationID, &inst.AccountLogin,
			&inst.AccountType, &inst.GovernanceMode, &inst.CreatedAt, &inst.UpdatedAt,
		); err != nil {
			return nil, err
		}
		installations = append(installations, inst)
	}
	return installations, nil
}

// UpdateGovernanceMode changes the governance mode for an installation.
func (db *DB) UpdateGovernanceMode(ctx context.Context, installationID int64, mode string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE github_installations SET governance_mode = $1, updated_at = now()
		WHERE installation_id = $2
	`, mode, installationID)
	return err
}

// RecordPRCheck logs a spec-linkage check result. The org_id is derived
// from the installation so the row passes RLS WITH CHECK against the
// caller's session-bound app.org_id.
func (db *DB) RecordPRCheck(ctx context.Context, check *PRCheck) error {
	_, err := db.Conn(ctx).Exec(ctx, `
		INSERT INTO pr_checks (org_id, installation_id, repo_full_name, pr_number, head_sha, spec_slugs, has_spec, conclusion)
		SELECT (SELECT org_id FROM github_installations WHERE id = $1),
		       $1, $2, $3, $4, $5, $6, $7
	`, check.InstallationID, check.RepoFullName, check.PRNumber, check.HeadSHA,
		check.SpecSlugs, check.HasSpec, check.Conclusion)
	return err
}

// PRCheckStats returns spec linkage statistics for an org.
func (db *DB) PRCheckStats(ctx context.Context, orgID string) (total, linked, unlinked int, err error) {
	err = db.Conn(ctx).QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE has_spec = true),
			COUNT(*) FILTER (WHERE has_spec = false)
		FROM pr_checks
		WHERE org_id = $1
	`, orgID).Scan(&total, &linked, &unlinked)
	return
}

// PRCheckStatsForRepo returns spec linkage stats for a specific repo.
func (db *DB) PRCheckStatsForRepo(ctx context.Context, orgID, repoFullName string) (total, linked, unlinked int, err error) {
	err = db.Conn(ctx).QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE has_spec = true),
			COUNT(*) FILTER (WHERE has_spec = false)
		FROM pr_checks
		WHERE org_id = $1 AND repo_full_name = $2
	`, orgID, repoFullName).Scan(&total, &linked, &unlinked)
	return
}
