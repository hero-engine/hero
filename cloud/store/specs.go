package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Spec represents a synced spec in the cloud.
type Spec struct {
	ID           string          `json:"id"`
	RepoID       string          `json:"repo_id"`
	Slug         string          `json:"slug"`
	Title        string          `json:"title"`
	Type         string          `json:"type"`
	Status       string          `json:"status"`
	Priority     string          `json:"priority,omitempty"`
	ClaimedBy    string          `json:"claimed_by,omitempty"`
	TrackerID    string          `json:"tracker_id,omitempty"`
	Subproject   string          `json:"subproject,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	FilesTouched []string        `json:"files_touched,omitempty"`
	Sections     json.RawMessage `json:"sections,omitempty"`
	Score        *int            `json:"score,omitempty"`
	RawContent   string          `json:"raw_content,omitempty"`
	Checksum     string          `json:"checksum"`
	SyncedAt     time.Time       `json:"synced_at"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// UpsertSpec creates or updates a spec by repo_id+slug.
func (db *DB) UpsertSpec(ctx context.Context, s *Spec) (*Spec, error) {
	sectionsJSON := s.Sections
	if sectionsJSON == nil {
		sectionsJSON = json.RawMessage("{}")
	}

	var result Spec
	err := db.Conn(ctx).QueryRow(ctx, `
		INSERT INTO specs (org_id, repo_id, slug, title, type, status, priority, claimed_by,
			tracker_id, subproject, tags, files_touched, sections, score, raw_content, checksum, synced_at)
		SELECT (SELECT org_id FROM repos WHERE id = $1),
		       $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, now()
		ON CONFLICT (repo_id, slug)
		DO UPDATE SET
			title = EXCLUDED.title,
			type = EXCLUDED.type,
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			claimed_by = EXCLUDED.claimed_by,
			tracker_id = EXCLUDED.tracker_id,
			subproject = EXCLUDED.subproject,
			tags = EXCLUDED.tags,
			files_touched = EXCLUDED.files_touched,
			sections = EXCLUDED.sections,
			score = EXCLUDED.score,
			raw_content = EXCLUDED.raw_content,
			checksum = EXCLUDED.checksum,
			synced_at = now(),
			updated_at = now()
		RETURNING id, repo_id, slug, title, type, status, priority, claimed_by,
			tracker_id, subproject, tags, files_touched, sections, score, raw_content, checksum,
			synced_at, created_at, updated_at
	`, s.RepoID, s.Slug, s.Title, s.Type, s.Status, s.Priority, s.ClaimedBy,
		s.TrackerID, s.Subproject, s.Tags, s.FilesTouched, sectionsJSON, s.Score, s.RawContent, s.Checksum,
	).Scan(
		&result.ID, &result.RepoID, &result.Slug, &result.Title, &result.Type,
		&result.Status, &result.Priority, &result.ClaimedBy, &result.TrackerID,
		&result.Subproject, &result.Tags, &result.FilesTouched, &result.Sections, &result.Score,
		&result.RawContent, &result.Checksum, &result.SyncedAt, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upserting spec: %w", err)
	}
	return &result, nil
}

// GetSpec retrieves a spec by repo_id and slug.
func (db *DB) GetSpec(ctx context.Context, repoID, slug string) (*Spec, error) {
	var s Spec
	err := db.Conn(ctx).QueryRow(ctx, `
		SELECT id, repo_id, slug, title, type, status, priority, claimed_by,
			tracker_id, subproject, tags, files_touched, sections, score, raw_content, checksum,
			synced_at, created_at, updated_at
		FROM specs WHERE repo_id = $1 AND slug = $2
	`, repoID, slug).Scan(
		&s.ID, &s.RepoID, &s.Slug, &s.Title, &s.Type, &s.Status,
		&s.Priority, &s.ClaimedBy, &s.TrackerID, &s.Subproject, &s.Tags, &s.FilesTouched,
		&s.Sections, &s.Score, &s.RawContent, &s.Checksum,
		&s.SyncedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting spec: %w", err)
	}
	return &s, nil
}

// GetSpecBySlugForOrg finds a spec by slug across all repos in an org.
func (db *DB) GetSpecBySlugForOrg(ctx context.Context, orgID, slug string) (*Spec, error) {
	var s Spec
	err := db.Conn(ctx).QueryRow(ctx, `
		SELECT s.id, s.repo_id, s.slug, s.title, s.type, s.status, s.priority, s.claimed_by,
			s.tracker_id, s.subproject, s.tags, s.files_touched, s.sections, s.score, s.raw_content, s.checksum,
			s.synced_at, s.created_at, s.updated_at
		FROM specs s
		JOIN repos r ON s.repo_id = r.id
		WHERE r.org_id = $1 AND s.slug = $2
		LIMIT 1
	`, orgID, slug).Scan(
		&s.ID, &s.RepoID, &s.Slug, &s.Title, &s.Type, &s.Status,
		&s.Priority, &s.ClaimedBy, &s.TrackerID, &s.Subproject, &s.Tags, &s.FilesTouched,
		&s.Sections, &s.Score, &s.RawContent, &s.Checksum,
		&s.SyncedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting spec by slug for org: %w", err)
	}
	return &s, nil
}

// ListRepoSpecs returns all specs for a repo, with optional filters.
// Pass subproject="" to disable the subproject filter; "all" is also
// treated as no filter for symmetry with the CLI.
func (db *DB) ListRepoSpecs(ctx context.Context, repoID string, specType, status, subproject string) ([]Spec, error) {
	query := `
		SELECT id, repo_id, slug, title, type, status, priority, claimed_by,
			tracker_id, subproject, tags, files_touched, sections, score, checksum,
			synced_at, created_at, updated_at
		FROM specs WHERE repo_id = $1
	`
	args := []any{repoID}
	argN := 2

	if specType != "" {
		query += fmt.Sprintf(" AND type = $%d", argN)
		args = append(args, specType)
		argN++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, status)
		argN++
	}
	if subproject != "" && subproject != "all" {
		query += fmt.Sprintf(" AND subproject = $%d", argN)
		args = append(args, subproject)
		argN++
	}
	query += " ORDER BY updated_at DESC"

	rows, err := db.Conn(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing specs: %w", err)
	}
	defer rows.Close()

	var specs []Spec
	for rows.Next() {
		var s Spec
		if err := rows.Scan(
			&s.ID, &s.RepoID, &s.Slug, &s.Title, &s.Type, &s.Status,
			&s.Priority, &s.ClaimedBy, &s.TrackerID, &s.Subproject, &s.Tags, &s.FilesTouched,
			&s.Sections, &s.Score, &s.Checksum,
			&s.SyncedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning spec: %w", err)
		}
		specs = append(specs, s)
	}
	return specs, rows.Err()
}

// ListRepoSubprojects returns the distinct non-empty subproject values
// for a repo, ordered alphabetically. Cheap lookup powered by the
// partial index on (org_id, subproject).
func (db *DB) ListRepoSubprojects(ctx context.Context, repoID string) ([]string, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT DISTINCT subproject
		FROM specs
		WHERE repo_id = $1 AND subproject != ''
		ORDER BY subproject
	`, repoID)
	if err != nil {
		return nil, fmt.Errorf("listing subprojects: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListOrgSpecs returns specs across all repos in an org, with optional filters.
func (db *DB) ListOrgSpecs(ctx context.Context, orgID string, specType, status, repoID, query, subproject, sort string, limit, offset int) ([]Spec, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
		FROM specs s
		JOIN repos r ON s.repo_id = r.id
		WHERE r.org_id = $1`
	args := []any{orgID}
	argN := 2

	if specType != "" {
		baseQuery += fmt.Sprintf(" AND s.type = $%d", argN)
		args = append(args, specType)
		argN++
	}
	if status != "" {
		baseQuery += fmt.Sprintf(" AND s.status = $%d", argN)
		args = append(args, status)
		argN++
	}
	if repoID != "" {
		baseQuery += fmt.Sprintf(" AND s.repo_id = $%d", argN)
		args = append(args, repoID)
		argN++
	}
	if query != "" {
		baseQuery += fmt.Sprintf(" AND (s.title ILIKE $%d OR s.slug ILIKE $%d)", argN, argN)
		args = append(args, "%"+query+"%")
		argN++
	}
	if subproject != "" && subproject != "all" {
		baseQuery += fmt.Sprintf(" AND s.subproject = $%d", argN)
		args = append(args, subproject)
		argN++
	}

	// Count total
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := db.Conn(ctx).QueryRow(ctx, "SELECT COUNT(*) "+baseQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting org specs: %w", err)
	}

	// Sort
	orderBy := " ORDER BY s.updated_at DESC"
	switch sort {
	case "title":
		orderBy = " ORDER BY s.title ASC"
	case "status":
		orderBy = " ORDER BY s.status ASC, s.updated_at DESC"
	case "type":
		orderBy = " ORDER BY s.type ASC, s.updated_at DESC"
	case "created":
		orderBy = " ORDER BY s.created_at DESC"
	}

	selectQuery := `
		SELECT s.id, s.repo_id, s.slug, s.title, s.type, s.status, s.priority,
			s.claimed_by, s.tracker_id, s.subproject, s.tags, s.files_touched, s.sections,
			s.score, s.checksum, s.synced_at, s.created_at, s.updated_at
		` + baseQuery + orderBy + fmt.Sprintf(" LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, offset)

	rows, err := db.Conn(ctx).Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing org specs: %w", err)
	}
	defer rows.Close()

	var specs []Spec
	for rows.Next() {
		var s Spec
		if err := rows.Scan(
			&s.ID, &s.RepoID, &s.Slug, &s.Title, &s.Type, &s.Status,
			&s.Priority, &s.ClaimedBy, &s.TrackerID, &s.Subproject, &s.Tags, &s.FilesTouched,
			&s.Sections, &s.Score, &s.Checksum,
			&s.SyncedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning org spec: %w", err)
		}
		specs = append(specs, s)
	}
	return specs, total, rows.Err()
}

// SpecStatusCounts returns the count of specs per status for an org.
func (db *DB) SpecStatusCounts(ctx context.Context, orgID string) (map[string]int, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT s.status, COUNT(*)
		FROM specs s
		JOIN repos r ON s.repo_id = r.id
		WHERE r.org_id = $1
		GROUP BY s.status
		ORDER BY s.status
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("spec status counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// SearchSpecs searches specs across all repos in an org.
func (db *DB) SearchSpecs(ctx context.Context, orgID, query string, limit int) ([]Spec, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	pattern := "%" + query + "%"

	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT s.id, s.repo_id, s.slug, s.title, s.type, s.status, s.priority,
			s.claimed_by, s.tracker_id, s.tags, s.files_touched, s.sections,
			s.score, s.checksum, s.synced_at, s.created_at, s.updated_at
		FROM specs s
		JOIN repos r ON s.repo_id = r.id
		WHERE r.org_id = $1
			AND (s.title ILIKE $2 OR s.slug ILIKE $2 OR s.raw_content ILIKE $2)
		ORDER BY s.updated_at DESC
		LIMIT $3
	`, orgID, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("searching specs: %w", err)
	}
	defer rows.Close()

	var specs []Spec
	for rows.Next() {
		var s Spec
		if err := rows.Scan(
			&s.ID, &s.RepoID, &s.Slug, &s.Title, &s.Type, &s.Status,
			&s.Priority, &s.ClaimedBy, &s.TrackerID, &s.Tags, &s.FilesTouched,
			&s.Sections, &s.Score, &s.Checksum,
			&s.SyncedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning spec: %w", err)
		}
		specs = append(specs, s)
	}
	return specs, rows.Err()
}

// DeleteSpec removes a spec by repo_id and slug.
func (db *DB) DeleteSpec(ctx context.Context, repoID, slug string) error {
	_, err := db.Conn(ctx).Exec(ctx, `
		DELETE FROM specs WHERE repo_id = $1 AND slug = $2
	`, repoID, slug)
	if err != nil {
		return fmt.Errorf("deleting spec: %w", err)
	}
	return nil
}

// SpecConflict represents a spec that shares files with another in-flight spec.
type SpecConflict struct {
	Slug             string   `json:"slug"`
	Title            string   `json:"title"`
	Type             string   `json:"type"`
	Status           string   `json:"status"`
	RepoID           string   `json:"repo_id"`
	ClaimedBy        string   `json:"claimed_by,omitempty"`
	OverlappingFiles []string `json:"overlapping_files"`
}

// FindConflicts finds in-flight specs that touch overlapping files with the given spec.
func (db *DB) FindConflicts(ctx context.Context, orgID, slug string) ([]SpecConflict, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT s2.slug, s2.title, s2.type, s2.status, s2.repo_id, COALESCE(s2.claimed_by, ''),
			array(SELECT unnest(s1.files_touched) INTERSECT SELECT unnest(s2.files_touched)) AS overlapping
		FROM specs s1
		JOIN specs s2 ON s1.repo_id = s2.repo_id AND s1.slug != s2.slug
			AND s1.files_touched && s2.files_touched
		JOIN repos r ON r.id = s1.repo_id
		WHERE r.org_id = $1
		AND s1.slug = $2
		AND s2.status IN ('planning', 'in-review', 'delivering')
		ORDER BY s2.slug
	`, orgID, slug)
	if err != nil {
		return nil, fmt.Errorf("finding conflicts: %w", err)
	}
	defer rows.Close()

	var results []SpecConflict
	for rows.Next() {
		var c SpecConflict
		if err := rows.Scan(&c.Slug, &c.Title, &c.Type, &c.Status, &c.RepoID, &c.ClaimedBy, &c.OverlappingFiles); err != nil {
			return nil, fmt.Errorf("scanning conflict: %w", err)
		}
		results = append(results, c)
	}
	return results, rows.Err()
}

// FindAllConflicts finds all conflicting spec pairs within an org.
func (db *DB) FindAllConflicts(ctx context.Context, orgID string) ([]SpecConflict, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		SELECT DISTINCT ON (LEAST(s1.slug, s2.slug), GREATEST(s1.slug, s2.slug))
			s1.slug, s1.title, s1.type, s1.status, s1.repo_id, COALESCE(s1.claimed_by, ''),
			array(SELECT unnest(s1.files_touched) INTERSECT SELECT unnest(s2.files_touched)) AS overlapping
		FROM specs s1
		JOIN specs s2 ON s1.repo_id = s2.repo_id AND s1.slug < s2.slug
			AND s1.files_touched && s2.files_touched
		JOIN repos r ON r.id = s1.repo_id
		WHERE r.org_id = $1
		AND s1.status IN ('planning', 'in-review', 'delivering')
		AND s2.status IN ('planning', 'in-review', 'delivering')
		ORDER BY LEAST(s1.slug, s2.slug), GREATEST(s1.slug, s2.slug)
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("finding all conflicts: %w", err)
	}
	defer rows.Close()

	var results []SpecConflict
	for rows.Next() {
		var c SpecConflict
		if err := rows.Scan(&c.Slug, &c.Title, &c.Type, &c.Status, &c.RepoID, &c.ClaimedBy, &c.OverlappingFiles); err != nil {
			return nil, fmt.Errorf("scanning conflict: %w", err)
		}
		results = append(results, c)
	}
	return results, rows.Err()
}

// SequenceItem represents a spec in a suggested delivery sequence.
type SequenceItem struct {
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Type          string   `json:"type"`
	Status        string   `json:"status"`
	Order         int      `json:"order"`
	ConflictCount int      `json:"conflict_count"`
	ConflictsWith []string `json:"conflicts_with,omitempty"`
	Reason        string   `json:"reason"`
}

// SuggestSequence returns in-flight specs in recommended delivery order.
func (db *DB) SuggestSequence(ctx context.Context, orgID string) ([]SequenceItem, error) {
	rows, err := db.Conn(ctx).Query(ctx, `
		WITH in_flight AS (
			SELECT s.slug, s.title, s.type, s.status, s.files_touched
			FROM specs s
			JOIN repos r ON r.id = s.repo_id
			WHERE r.org_id = $1
			AND s.status IN ('planning', 'in-review', 'delivering')
		),
		conflict_counts AS (
			SELECT a.slug,
				COUNT(DISTINCT b.slug) AS conflicts,
				array_agg(DISTINCT b.slug) AS conflict_slugs
			FROM in_flight a
			LEFT JOIN in_flight b ON a.slug != b.slug AND a.files_touched && b.files_touched
			GROUP BY a.slug
		)
		SELECT f.slug, f.title, f.type, f.status,
			COALESCE(c.conflicts, 0), COALESCE(c.conflict_slugs, ARRAY[]::TEXT[])
		FROM in_flight f
		LEFT JOIN conflict_counts c ON c.slug = f.slug
		ORDER BY COALESCE(c.conflicts, 0) ASC, f.slug
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("suggesting sequence: %w", err)
	}
	defer rows.Close()

	var result []SequenceItem
	order := 1
	for rows.Next() {
		var item SequenceItem
		if err := rows.Scan(&item.Slug, &item.Title, &item.Type, &item.Status, &item.ConflictCount, &item.ConflictsWith); err != nil {
			return nil, fmt.Errorf("scanning sequence item: %w", err)
		}
		item.Order = order
		if item.ConflictCount == 0 {
			item.Reason = "no file conflicts — safe to deliver independently"
		} else {
			item.Reason = fmt.Sprintf("%d file conflict(s) — coordinate with conflicting specs", item.ConflictCount)
		}
		var filtered []string
		for _, s := range item.ConflictsWith {
			if s != "" {
				filtered = append(filtered, s)
			}
		}
		item.ConflictsWith = filtered
		result = append(result, item)
		order++
	}
	return result, rows.Err()
}
