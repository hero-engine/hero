package mocktracker

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo)
)

// Store is the in-memory SQLite state the four tracker handlers project
// from. A single shared connection (SetMaxOpenConns(1)) keeps :memory:
// state coherent across requests; an external RWMutex (held by Server)
// serializes admin writes against reads.
type Store struct {
	db *sql.DB
}

// NewStore opens a fresh in-memory SQLite DB and creates the
// tracker-neutral schema. The caller seeds it (see seed.go) and then
// calls RebuildAliases. The DB is reset by reopening, not by mutation.
func NewStore(ctx context.Context) (*Store, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// :memory: is per-connection; pin to one so every request sees the
	// same database.
	db.SetMaxOpenConns(1)
	for _, ddl := range schemaDDL {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			db.Close()
			return nil, fmt.Errorf("schema: %w", err)
		}
	}
	return &Store{db: db}, nil
}

// DB exposes the underlying *sql.DB for the seeder (sprout.Apply needs a
// *sql.DB to run one transaction per seed file).
func (s *Store) DB() *sql.DB { return s.db }

// Close releases the connection.
func (s *Store) Close() error { return s.db.Close() }

var trailingInt = regexp.MustCompile(`(\d+)$`)

// iidOf derives a stable per-tracker IID from a global_id: the trailing
// integer when present (ACME-100 → 100), else a deterministic hash so
// every node still gets a number. Issues persist their IID in id_alias
// so rotate-ids can rewrite it.
func iidOf(globalID string) string {
	if m := trailingInt.FindStringSubmatch(globalID); m != nil {
		return m[1]
	}
	var h uint32 = 2166136261
	for i := 0; i < len(globalID); i++ {
		h ^= uint32(globalID[i])
		h *= 16777619
	}
	return fmt.Sprintf("%d", h%100000)
}

// RebuildAliases seeds id_alias with one row per issue, IID derived from
// the global_id. Run once after seeding.
func (s *Store) RebuildAliases(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT global_id FROM issue ORDER BY global_id`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO id_alias (global_id, iid) VALUES (?, ?)`, id, iidOf(id)); err != nil {
			return err
		}
	}
	return nil
}

// IssueRow is one issue projected with its labels and current IID.
type IssueRow struct {
	GlobalID    string
	Type        string
	Title       string
	Body        string
	EpicID      string
	MilestoneID string
	IterationID string
	Status      string
	Assignee    string
	Weight      sql.NullInt64
	Severity    string
	Labels      []string
	IID         string
}

// IssueFilter narrows a list query. Empty fields are ignored.
type IssueFilter struct {
	State       string   // "open" | "closed" | "all" | ""
	Labels      []string // issue must carry ALL of these
	Assignee    string   // username, or "none"/"unassigned" for unassigned
	IterationID string
	Search      string // case-insensitive substring on title
}

// ListIssues returns issues matching filter, ordered by global_id for
// deterministic pagination.
func (s *Store) ListIssues(ctx context.Context, f IssueFilter) ([]IssueRow, error) {
	var where []string
	var args []any

	switch strings.ToLower(f.State) {
	case "open", "opened":
		where = append(where, "status != 'closed'")
	case "closed":
		where = append(where, "status = 'closed'")
	}
	switch strings.ToLower(f.Assignee) {
	case "":
		// no filter
	case "none", "unassigned", "empty":
		where = append(where, "(assignee IS NULL OR assignee = '')")
	default:
		where = append(where, "assignee = ?")
		args = append(args, f.Assignee)
	}
	if f.IterationID != "" {
		where = append(where, "iteration_id = ?")
		args = append(args, f.IterationID)
	}
	if f.Search != "" {
		where = append(where, "LOWER(title) LIKE ?")
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
	}
	for _, lbl := range f.Labels {
		where = append(where, "global_id IN (SELECT issue_id FROM label WHERE name = ?)")
		args = append(args, lbl)
	}

	q := `SELECT global_id, type, title, body, epic_id, milestone_id, iteration_id,
	             status, assignee, weight, severity FROM issue`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY global_id"

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IssueRow
	for rows.Next() {
		r, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.hydrate(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func scanIssue(rows *sql.Rows) (IssueRow, error) {
	var r IssueRow
	var epic, milestone, iteration, assignee, severity, body sql.NullString
	if err := rows.Scan(&r.GlobalID, &r.Type, &r.Title, &body, &epic, &milestone,
		&iteration, &r.Status, &assignee, &r.Weight, &severity); err != nil {
		return r, err
	}
	r.Body = body.String
	r.EpicID = epic.String
	r.MilestoneID = milestone.String
	r.IterationID = iteration.String
	r.Assignee = assignee.String
	r.Severity = severity.String
	return r, nil
}

// hydrate fills Labels and IID for an issue row.
func (s *Store) hydrate(ctx context.Context, r *IssueRow) error {
	labels, err := s.labels(ctx, r.GlobalID)
	if err != nil {
		return err
	}
	r.Labels = labels
	r.IID = s.iid(ctx, r.GlobalID)
	return nil
}

func (s *Store) labels(ctx context.Context, issueID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM label WHERE issue_id = ? ORDER BY name`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// iid returns the current IID for an issue, falling back to a derived
// value if no alias row exists.
func (s *Store) iid(ctx context.Context, globalID string) string {
	var iid string
	err := s.db.QueryRowContext(ctx, `SELECT iid FROM id_alias WHERE global_id = ?`, globalID).Scan(&iid)
	if err != nil || iid == "" {
		return iidOf(globalID)
	}
	return iid
}

// GetIssueByGlobalID returns one issue by its stable global id (key-based
// trackers: jira/linear).
func (s *Store) GetIssueByGlobalID(ctx context.Context, globalID string) (*IssueRow, error) {
	row := s.db.QueryRowContext(ctx, `SELECT global_id, type, title, body, epic_id, milestone_id,
		iteration_id, status, assignee, weight, severity FROM issue WHERE global_id = ?`, globalID)
	r, err := scanIssueRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.hydrate(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

// GetIssueByIID resolves an IID (github number / gitlab iid) to an issue
// via id_alias.
func (s *Store) GetIssueByIID(ctx context.Context, iid string) (*IssueRow, error) {
	var globalID string
	err := s.db.QueryRowContext(ctx, `SELECT global_id FROM id_alias WHERE iid = ?`, iid).Scan(&globalID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetIssueByGlobalID(ctx, globalID)
}

func scanIssueRow(row *sql.Row) (*IssueRow, error) {
	var r IssueRow
	var epic, milestone, iteration, assignee, severity, body sql.NullString
	if err := row.Scan(&r.GlobalID, &r.Type, &r.Title, &body, &epic, &milestone,
		&iteration, &r.Status, &assignee, &r.Weight, &severity); err != nil {
		return nil, err
	}
	r.Body = body.String
	r.EpicID = epic.String
	r.MilestoneID = milestone.String
	r.IterationID = iteration.String
	r.Assignee = assignee.String
	r.Severity = severity.String
	return &r, nil
}

// UpdateIssue applies a column→value patch to one issue. Allowed columns
// are the writable content fields; labels are handled separately via
// SetLabels. Unknown columns are ignored.
func (s *Store) UpdateIssue(ctx context.Context, globalID string, fields map[string]any) error {
	allowed := map[string]bool{
		"title": true, "body": true, "status": true, "assignee": true,
		"weight": true, "severity": true, "type": true,
	}
	var sets []string
	var args []any
	for col, val := range fields {
		if !allowed[col] {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, globalID)
	q := fmt.Sprintf("UPDATE issue SET %s WHERE global_id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

// SetLabels replaces an issue's full label set.
func (s *Store) SetLabels(ctx context.Context, issueID string, labels []string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM label WHERE issue_id = ?`, issueID); err != nil {
		return err
	}
	for _, l := range labels {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO label (issue_id, name) VALUES (?, ?)`, issueID, l); err != nil {
			return err
		}
	}
	return nil
}

// CreateIssue inserts a new issue with a freshly minted global_id and
// IID. Used by the POST endpoints. Returns the created row.
func (s *Store) CreateIssue(ctx context.Context, prefix, title, body string, labels []string) (*IssueRow, error) {
	var n int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM issue`).Scan(&n)
	globalID := fmt.Sprintf("%s-%d", prefix, 9000+n+1) // distinct from seed range
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO issue (global_id, type, title, body, status) VALUES (?, 'story', ?, ?, 'open')`,
		globalID, title, body); err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO id_alias (global_id, iid) VALUES (?, ?)`, globalID, iidOf(globalID)); err != nil {
		return nil, err
	}
	if err := s.SetLabels(ctx, globalID, labels); err != nil {
		return nil, err
	}
	return s.GetIssueByGlobalID(ctx, globalID)
}

// Mutate writes one field out-of-band (the /__admin/mutate drift plane).
// field "title"|"body"|"status"|"assignee"|"weight"|"severity" map to
// columns; "labels" replaces the label set (comma-separated value).
func (s *Store) Mutate(ctx context.Context, globalID, field, value string) error {
	if field == "labels" {
		return s.SetLabels(ctx, globalID, strings.Split(value, ","))
	}
	return s.UpdateIssue(ctx, globalID, map[string]any{field: value})
}

// RotateIID rewrites an issue's IID while keeping its global_id. A
// new IID equal to the old one is a no-op (key-based trackers).
func (s *Store) RotateIID(ctx context.Context, globalID, newIID string) error {
	if newIID == "" {
		newIID = iidOf(globalID) + "0" // perturb deterministically
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO id_alias (global_id, iid) VALUES (?, ?)`, globalID, newIID)
	return err
}

// --- container reads (epics / milestones / iterations) ---

type EpicRow struct {
	GlobalID string
	Title    string
	ParentID string
}

func (s *Store) ListEpics(ctx context.Context) ([]EpicRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT global_id, title, parent_id FROM epic ORDER BY global_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EpicRow
	for rows.Next() {
		var e EpicRow
		var parent sql.NullString
		if err := rows.Scan(&e.GlobalID, &e.Title, &parent); err != nil {
			return nil, err
		}
		e.ParentID = parent.String
		out = append(out, e)
	}
	return out, rows.Err()
}

type MilestoneRow struct {
	GlobalID string
	Title    string
	Due      string
}

func (s *Store) ListMilestones(ctx context.Context) ([]MilestoneRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT global_id, title, due FROM milestone ORDER BY global_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MilestoneRow
	for rows.Next() {
		var m MilestoneRow
		var due sql.NullString
		if err := rows.Scan(&m.GlobalID, &m.Title, &due); err != nil {
			return nil, err
		}
		m.Due = due.String
		out = append(out, m)
	}
	return out, rows.Err()
}

type IterationRow struct {
	GlobalID string
	Name     string
	Start    string
	End      string
}

func (s *Store) ListIterations(ctx context.Context) ([]IterationRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT global_id, name, "start", "end" FROM iteration ORDER BY global_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IterationRow
	for rows.Next() {
		var it IterationRow
		var start, end sql.NullString
		if err := rows.Scan(&it.GlobalID, &it.Name, &start, &end); err != nil {
			return nil, err
		}
		it.Start = start.String
		it.End = end.String
		out = append(out, it)
	}
	return out, rows.Err()
}
