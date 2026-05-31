package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/hero-engine/hero/internal/spec"
	_ "modernc.org/sqlite"
)

// ftsStopwords is a small list of question/glue words that are dropped
// from FTS5 queries so natural-language `hero ask` doesn't fail to
// match because of "what / is / the / are" pollution.
var ftsStopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true,
	"at": true, "be": true, "by": true, "do": true, "does": true,
	"for": true, "from": true, "has": true, "have": true, "how": true,
	"i": true, "in": true, "is": true, "it": true, "of": true,
	"on": true, "or": true, "that": true, "the": true, "this": true,
	"to": true, "was": true, "we": true, "were": true, "what": true,
	"when": true, "where": true, "which": true, "who": true, "why": true,
	"will": true, "with": true, "you": true,
}

// SanitizeFTSQuery converts a free-form natural-language query into an
// FTS5-safe expression. It drops every non-alphanumeric character (so
// `?`, `:`, `(`, `*` and friends can't trigger FTS5 syntax errors),
// splits on whitespace, drops stopwords, double-quotes each remaining
// token (so SQLite's FTS5 doesn't treat any leftover special token as
// an operator), and joins them with ` OR ` for permissive matching.
//
// Returns "" if the query has no usable content tokens; callers should
// treat that as "no match" without sending an empty MATCH to SQLite.
func SanitizeFTSQuery(query string) string {
	var b strings.Builder
	for _, r := range query {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(' ')
		}
	}
	raw := strings.Fields(b.String())
	terms := make([]string, 0, len(raw))
	for _, t := range raw {
		if ftsStopwords[t] {
			continue
		}
		terms = append(terms, `"`+t+`"`)
	}
	return strings.Join(terms, " OR ")
}

const (
	IndexFileName = "index.db"
)

// DB wraps the SQLite database for the spec corpus index.
type DB struct {
	db   *sql.DB
	path string
}

// Open opens or creates the index database at the given path.
func Open(heroDir string) (*DB, error) {
	dbPath := filepath.Join(heroDir, IndexFileName)

	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating hero directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening index database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	idx := &DB{db: db, path: dbPath}
	if err := idx.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating index schema: %w", err)
	}

	return idx, nil
}

// RawDB returns the underlying *sql.DB for direct queries (e.g. fts_nodes).
func (idx *DB) RawDB() *sql.DB { return idx.db }

// Close closes the database.
func (idx *DB) Close() error {
	return idx.db.Close()
}

func (idx *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS specs (
			slug        TEXT PRIMARY KEY,
			title       TEXT NOT NULL,
			type        TEXT NOT NULL,
			status      TEXT NOT NULL,
			path        TEXT NOT NULL,
			claimed_by  TEXT NOT NULL DEFAULT '',
			tags        TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL,
			modified_at TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS decisions (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			spec_slug TEXT NOT NULL,
			content   TEXT NOT NULL,
			FOREIGN KEY (spec_slug) REFERENCES specs(slug) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS root_causes (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			spec_slug TEXT NOT NULL,
			content   TEXT NOT NULL,
			FOREIGN KEY (spec_slug) REFERENCES specs(slug) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS files_touched (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			spec_slug TEXT NOT NULL,
			file_path TEXT NOT NULL,
			FOREIGN KEY (spec_slug) REFERENCES specs(slug) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS convention_scopes (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			spec_slug  TEXT NOT NULL,
			scope_glob TEXT NOT NULL,
			FOREIGN KEY (spec_slug) REFERENCES specs(slug) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS spec_relations (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			from_slug TEXT NOT NULL,
			to_slug   TEXT NOT NULL,
			relation  TEXT NOT NULL,
			FOREIGN KEY (from_slug) REFERENCES specs(slug) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS claims (
			spec_slug  TEXT PRIMARY KEY,
			claimed_by TEXT NOT NULL,
			claimed_at TEXT NOT NULL,
			FOREIGN KEY (spec_slug) REFERENCES specs(slug) ON DELETE CASCADE
		)`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS fts_specs USING fts5(
			slug,
			title,
			content,
			tokenize='porter'
		)`,

		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_specs_type ON specs(type)`,
		`CREATE INDEX IF NOT EXISTS idx_specs_status ON specs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_files_touched_path ON files_touched(file_path)`,
		`CREATE INDEX IF NOT EXISTS idx_convention_scopes_glob ON convention_scopes(scope_glob)`,
		`CREATE INDEX IF NOT EXISTS idx_spec_relations_from ON spec_relations(from_slug)`,
		`CREATE INDEX IF NOT EXISTS idx_spec_relations_to ON spec_relations(to_slug)`,

		// Phase B: unified cross-type search index projected from graph nodes.
		`CREATE VIRTUAL TABLE IF NOT EXISTS fts_nodes USING fts5(
			title,
			body,
			tokenize='porter'
		)`,

		`CREATE TABLE IF NOT EXISTS node_index (
			rowid      INTEGER PRIMARY KEY,
			node_type  TEXT NOT NULL,
			key        TEXT NOT NULL,
			repo       TEXT NOT NULL DEFAULT '',
			tags       TEXT NOT NULL DEFAULT '',
			valid_from TEXT NOT NULL DEFAULT '',
			path       TEXT NOT NULL DEFAULT '',
			UNIQUE(node_type, key)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_node_index_type ON node_index(node_type)`,
		`CREATE INDEX IF NOT EXISTS idx_node_index_key ON node_index(key)`,

		`CREATE TABLE IF NOT EXISTS tripwire_triggers (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			spec_slug  TEXT NOT NULL,
			trigger    TEXT NOT NULL,
			FOREIGN KEY (spec_slug) REFERENCES specs(slug) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tripwire_triggers_slug ON tripwire_triggers(spec_slug)`,
	}

	for _, m := range migrations {
		if _, err := idx.db.Exec(m); err != nil {
			return fmt.Errorf("executing migration: %w\nSQL: %s", err, m)
		}
	}

	// Schema evolution: add columns that may not exist in older databases.
	evolve := []string{
		`ALTER TABLE specs ADD COLUMN tracker_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE specs ADD COLUMN subproject TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_specs_subproject ON specs(subproject) WHERE subproject != ''`,
		`ALTER TABLE specs ADD COLUMN domain TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_specs_domain ON specs(domain) WHERE domain != ''`,
		// superseded-specs-soft-archive: frontmatter-driven genealogy.
		// Empty string means "not superseded" — non-empty carries the
		// replacement slug. The index queries de-weight (in retrieval)
		// and the context layer annotate based on this column alone.
		`ALTER TABLE specs ADD COLUMN superseded_by TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_specs_superseded ON specs(superseded_by) WHERE superseded_by != ''`,
	}
	for _, stmt := range evolve {
		_, _ = idx.db.Exec(stmt) // ignore "duplicate column" errors
	}

	return nil
}

// IndexSpec adds or updates a spec in the index.
func (idx *DB) IndexSpec(s *spec.Spec, fullContent string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tagsStr := strings.Join(s.Tags, ",")

	// Upsert spec
	_, err = tx.Exec(`
		INSERT INTO specs (slug, title, type, status, path, claimed_by, tags, tracker_id, subproject, domain, superseded_by, created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title=excluded.title,
			type=excluded.type,
			status=excluded.status,
			path=excluded.path,
			claimed_by=excluded.claimed_by,
			tags=excluded.tags,
			tracker_id=excluded.tracker_id,
			subproject=excluded.subproject,
			domain=excluded.domain,
			superseded_by=excluded.superseded_by,
			modified_at=excluded.modified_at
	`, s.Slug, s.Title, string(s.Type), string(s.Status), s.Path,
		s.ClaimedBy, tagsStr, s.TrackerID, s.Subproject, s.Domain, s.SupersededBy,
		s.CreatedAt.Format(time.RFC3339), s.ModifiedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upserting spec: %w", err)
	}

	// Clear old related data
	for _, table := range []string{"decisions", "root_causes", "files_touched", "convention_scopes", "tripwire_triggers", "spec_relations"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE spec_slug = ? OR from_slug = ?", table), s.Slug, s.Slug); err != nil {
			// spec_relations uses from_slug, others use spec_slug — try both patterns
			tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE spec_slug = ?", table), s.Slug)
		}
	}

	// Extract and store decisions from Approach section
	if approach, ok := s.Sections["approach"]; ok {
		if _, err := tx.Exec("INSERT INTO decisions (spec_slug, content) VALUES (?, ?)", s.Slug, approach); err != nil {
			return fmt.Errorf("inserting decision: %w", err)
		}
	}

	// Extract and store root causes from Investigation section
	if investigation, ok := s.Sections["investigation"]; ok {
		if _, err := tx.Exec("INSERT INTO root_causes (spec_slug, content) VALUES (?, ?)", s.Slug, investigation); err != nil {
			return fmt.Errorf("inserting root cause: %w", err)
		}
	}

	// Store files touched
	for _, fp := range s.FilesTouched {
		if _, err := tx.Exec("INSERT INTO files_touched (spec_slug, file_path) VALUES (?, ?)", s.Slug, fp); err != nil {
			return fmt.Errorf("inserting file touched: %w", err)
		}
	}

	// Store convention scopes
	for _, scope := range s.Scope {
		if _, err := tx.Exec("INSERT INTO convention_scopes (spec_slug, scope_glob) VALUES (?, ?)", s.Slug, scope); err != nil {
			return fmt.Errorf("inserting convention scope: %w", err)
		}
	}

	// Store tripwire triggers
	for _, trigger := range s.Triggers {
		if _, err := tx.Exec("INSERT INTO tripwire_triggers (spec_slug, trigger) VALUES (?, ?)", s.Slug, strings.ToLower(trigger)); err != nil {
			return fmt.Errorf("inserting tripwire trigger: %w", err)
		}
	}

	// Store relations
	for _, rel := range s.Relations {
		if _, err := tx.Exec("INSERT INTO spec_relations (from_slug, to_slug, relation) VALUES (?, ?, ?)",
			s.Slug, rel.Target, rel.Kind); err != nil {
			return fmt.Errorf("inserting relation: %w", err)
		}
	}

	// Update full-text search index
	_, _ = tx.Exec("DELETE FROM fts_specs WHERE slug = ?", s.Slug)
	if _, err := tx.Exec("INSERT INTO fts_specs (slug, title, content) VALUES (?, ?, ?)",
		s.Slug, s.Title, fullContent); err != nil {
		return fmt.Errorf("updating FTS index: %w", err)
	}

	return tx.Commit()
}

// RemoveSpec removes a spec from the index.
func (idx *DB) RemoveSpec(slug string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{"decisions", "root_causes", "files_touched", "convention_scopes", "tripwire_triggers"} {
		if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE spec_slug = ?", table), slug); err != nil {
			return err
		}
	}

	if _, err := tx.Exec("DELETE FROM spec_relations WHERE from_slug = ? OR to_slug = ?", slug, slug); err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM claims WHERE spec_slug = ?", slug); err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM fts_specs WHERE slug = ?", slug); err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM specs WHERE slug = ?", slug); err != nil {
		return err
	}

	return tx.Commit()
}

// SearchResult holds a single search hit.
type SearchResult struct {
	Slug      string
	Title     string
	Type      spec.Type
	Status    spec.Status
	Path      string
	Snippet   string
	ClaimedBy string
	Tags      string
	Domain    string
	// SupersededBy is non-empty when this spec carries a `superseded_by:`
	// frontmatter field. Retrieval de-weights such results and rendering
	// layers prefix the snippet with a redirect marker.
	SupersededBy string
}

// looksLikeTrackerID returns true if the query looks like a tracker issue ID
// (e.g. "PROJ-123", "#42", "MORPH-456").
func looksLikeTrackerID(query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return false
	}
	if regexp.MustCompile(`^[A-Za-z]+-\d+$`).MatchString(query) {
		return true
	}
	if strings.HasPrefix(query, "#") || regexp.MustCompile(`^\d+$`).MatchString(query) {
		return true
	}
	return false
}

// Search performs a full-text search over specs. If the query looks like a
// tracker issue ID (e.g. PROJ-123), it first tries an exact tracker_id match
// before falling back to FTS.
func (idx *DB) Search(query string) ([]SearchResult, error) {
	// Try exact tracker_id match first. A tracker-ID-shaped query that
	// finds nothing should return zero results — falling through to FTS
	// would surface unrelated specs whose names share a token prefix
	// (e.g. searching "MORPH-999" would match "morph-123-fix-login"),
	// which is more confusing than helpful for typo'd ticket IDs.
	if looksLikeTrackerID(query) {
		rows, err := idx.db.Query(`
			SELECT s.slug, s.title, s.type, s.status, s.path,
				'' as snippet, s.claimed_by, s.tags, s.domain, s.superseded_by
			FROM specs s
			WHERE UPPER(s.tracker_id) = UPPER(?)
			LIMIT 5
		`, query)
		if err != nil {
			return nil, fmt.Errorf("searching by tracker_id: %w", err)
		}
		results, _ := scanSearchResults(rows)
		rows.Close()
		return results, nil
	}

	// Fall back to full-text search. Sanitize the query: strip FTS5 special
	// characters, drop stopwords, OR the remaining terms together.
	ftsQuery := SanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}
	rows, err := idx.db.Query(`
		SELECT f.slug, s.title, s.type, s.status, s.path,
			snippet(fts_specs, 2, '>>>', '<<<', '...', 32) as snippet,
			s.claimed_by, s.tags, s.domain, s.superseded_by
		FROM fts_specs f
		JOIN specs s ON s.slug = f.slug
		WHERE fts_specs MATCH ?
		ORDER BY rank
		LIMIT 20
	`, ftsQuery)
	if err != nil {
		return nil, fmt.Errorf("searching index: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// SearchFiltered searches with optional type, status, and tag filters.
// SearchFiltered runs the FTS query with optional structured filters.
// Pass `subproject` empty to skip subproject filtering, "all" is also
// treated as no filter (callers may pass "all" to be explicit).
func (idx *DB) SearchFilteredScoped(query, specType, status, tag, since, subproject string) ([]SearchResult, error) {
	return idx.searchFilteredImpl(query, specType, status, tag, since, subproject)
}

func (idx *DB) SearchFiltered(query string, specType string, status string, tag string, since string) ([]SearchResult, error) {
	return idx.searchFilteredImpl(query, specType, status, tag, since, "")
}

func (idx *DB) searchFilteredImpl(query, specType, status, tag, since, subproject string) ([]SearchResult, error) {
	ftsQuery := SanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}
	var conditions []string
	var args []interface{}

	baseQuery := `
		SELECT f.slug, s.title, s.type, s.status, s.path,
			snippet(fts_specs, 2, '>>>', '<<<', '...', 32) as snippet,
			s.claimed_by, s.tags, s.domain, s.superseded_by
		FROM fts_specs f
		JOIN specs s ON s.slug = f.slug
		WHERE fts_specs MATCH ?`
	args = append(args, ftsQuery)

	if specType != "" {
		conditions = append(conditions, "s.type = ?")
		args = append(args, specType)
	}
	if status != "" {
		conditions = append(conditions, "s.status = ?")
		args = append(args, status)
	}
	if tag != "" {
		conditions = append(conditions, "s.tags LIKE ?")
		args = append(args, "%"+tag+"%")
	}
	if since != "" {
		conditions = append(conditions, "s.created_at >= ?")
		args = append(args, since)
	}
	if subproject != "" && subproject != "all" {
		conditions = append(conditions, "s.subproject = ?")
		args = append(args, subproject)
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	baseQuery += " ORDER BY rank LIMIT 20"

	rows, err := idx.db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("searching index: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// ListFilteredScoped is the subproject-aware variant of ListFiltered.
// Pass subproject empty or "all" to disable subproject filtering.
func (idx *DB) ListFilteredScoped(specType, status, tag, since, subproject string) ([]SearchResult, error) {
	return idx.listFilteredImpl(specType, status, tag, since, subproject)
}

// ListFiltered lists specs with optional type, status, and tag filters (no FTS query).
func (idx *DB) ListFiltered(specType string, status string, tag string, since string) ([]SearchResult, error) {
	return idx.listFilteredImpl(specType, status, tag, since, "")
}

func (idx *DB) listFilteredImpl(specType, status, tag, since, subproject string) ([]SearchResult, error) {
	var conditions []string
	var args []interface{}

	baseQuery := `SELECT slug, title, type, status, path, '' as snippet, claimed_by, tags, domain, superseded_by FROM specs WHERE 1=1`

	if specType != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, specType)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if tag != "" {
		conditions = append(conditions, "tags LIKE ?")
		args = append(args, "%"+tag+"%")
	}
	if since != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, since)
	}
	if subproject != "" && subproject != "all" {
		conditions = append(conditions, "subproject = ?")
		args = append(args, subproject)
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}
	baseQuery += " ORDER BY modified_at DESC LIMIT 50"

	rows, err := idx.db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("listing specs: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// SearchByFile finds specs that touch a given file path.
func (idx *DB) SearchByFile(filePath string) ([]SearchResult, error) {
	pattern := "%" + filePath + "%"

	rows, err := idx.db.Query(`
		SELECT DISTINCT s.slug, s.title, s.type, s.status, s.path,
			'' as snippet, s.claimed_by, s.tags, s.domain, s.superseded_by
		FROM files_touched ft
		JOIN specs s ON s.slug = ft.spec_slug
		WHERE ft.file_path LIKE ?
		ORDER BY s.modified_at DESC
		LIMIT 20
	`, pattern)
	if err != nil {
		return nil, fmt.Errorf("searching by file: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// FindConventionsForFiles finds active conventions whose scopes match any of the given file paths.
func (idx *DB) FindConventionsForFiles(filePaths []string) ([]SearchResult, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	// Get all active convention scopes
	rows, err := idx.db.Query(`
		SELECT cs.scope_glob, s.slug, s.title, s.type, s.status, s.path, s.claimed_by, s.tags, s.domain
		FROM convention_scopes cs
		JOIN specs s ON s.slug = cs.spec_slug
		WHERE s.type = 'convention' AND (s.status = 'active' OR s.status = 'draft')
		ORDER BY s.slug
	`)
	if err != nil {
		return nil, fmt.Errorf("querying convention scopes: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var results []SearchResult

	for rows.Next() {
		var glob string
		var r SearchResult
		var specType, status string
		if err := rows.Scan(&glob, &r.Slug, &r.Title, &specType, &status, &r.Path, &r.ClaimedBy, &r.Tags, &r.Domain); err != nil {
			return nil, err
		}

		if seen[r.Slug] {
			continue
		}

		// Check if any file path matches this glob
		for _, fp := range filePaths {
			matched, _ := filepath.Match(glob, fp)
			if matched {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}

			// Also try matching just the filename for extension-based globs like *.go
			base := filepath.Base(fp)
			matched, _ = filepath.Match(glob, base)
			if matched {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}

			// Try matching against each path segment for directory-based globs
			if strings.Contains(glob, "/") {
				// For globs like "src/api/**", check if the file is under that path
				globDir := strings.TrimSuffix(glob, "/**")
				globDir = strings.TrimSuffix(globDir, "/*")
				if strings.HasPrefix(fp, globDir+"/") || strings.HasPrefix(fp, globDir) {
					r.Type = spec.Type(specType)
					r.Status = spec.Status(status)
					results = append(results, r)
					seen[r.Slug] = true
					break
				}
			}

			// Catch-all glob
			if glob == "*" {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}
		}
	}

	return results, rows.Err()
}

// FindRulesForFiles finds active rules whose scopes match any of the given file paths.
// Rules use the same scope matching as conventions but are hard constraints.
func (idx *DB) FindRulesForFiles(filePaths []string) ([]SearchResult, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	rows, err := idx.db.Query(`
		SELECT cs.scope_glob, s.slug, s.title, s.type, s.status, s.path, s.claimed_by, s.tags, s.domain
		FROM convention_scopes cs
		JOIN specs s ON s.slug = cs.spec_slug
		WHERE s.type = 'rule' AND (s.status = 'active' OR s.status = 'draft')
		ORDER BY s.slug
	`)
	if err != nil {
		return nil, fmt.Errorf("querying rule scopes: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var results []SearchResult

	for rows.Next() {
		var glob string
		var r SearchResult
		var specType, status string
		if err := rows.Scan(&glob, &r.Slug, &r.Title, &specType, &status, &r.Path, &r.ClaimedBy, &r.Tags, &r.Domain); err != nil {
			return nil, err
		}

		if seen[r.Slug] {
			continue
		}

		for _, fp := range filePaths {
			matched, _ := filepath.Match(glob, fp)
			if matched {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}

			base := filepath.Base(fp)
			matched, _ = filepath.Match(glob, base)
			if matched {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}

			if strings.Contains(glob, "/") {
				globDir := strings.TrimSuffix(glob, "/**")
				globDir = strings.TrimSuffix(globDir, "/*")
				if strings.HasPrefix(fp, globDir+"/") || strings.HasPrefix(fp, globDir) {
					r.Type = spec.Type(specType)
					r.Status = spec.Status(status)
					results = append(results, r)
					seen[r.Slug] = true
					break
				}
			}

			if glob == "*" {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}
		}
	}

	return results, rows.Err()
}

// FindTripwiresForFiles returns active tripwires whose scope matches the given file paths.
func (idx *DB) FindTripwiresForFiles(filePaths []string) ([]SearchResult, error) {
	if len(filePaths) == 0 {
		return nil, nil
	}

	rows, err := idx.db.Query(`
		SELECT cs.scope_glob, s.slug, s.title, s.type, s.status, s.path, s.claimed_by, s.tags, s.domain
		FROM convention_scopes cs
		JOIN specs s ON s.slug = cs.spec_slug
		WHERE s.type = 'tripwire' AND s.status = 'active'
		ORDER BY s.slug
	`)
	if err != nil {
		return nil, fmt.Errorf("querying tripwire scopes: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var results []SearchResult

	for rows.Next() {
		var glob string
		var r SearchResult
		var specType, status string
		if err := rows.Scan(&glob, &r.Slug, &r.Title, &specType, &status, &r.Path, &r.ClaimedBy, &r.Tags, &r.Domain); err != nil {
			return nil, err
		}

		if seen[r.Slug] {
			continue
		}

		for _, fp := range filePaths {
			matched, _ := filepath.Match(glob, fp)
			if matched {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}

			base := filepath.Base(fp)
			matched, _ = filepath.Match(glob, base)
			if matched {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}

			if strings.Contains(glob, "/") {
				globDir := strings.TrimSuffix(glob, "/**")
				globDir = strings.TrimSuffix(globDir, "/*")
				if strings.HasPrefix(fp, globDir+"/") || strings.HasPrefix(fp, globDir) {
					r.Type = spec.Type(specType)
					r.Status = spec.Status(status)
					results = append(results, r)
					seen[r.Slug] = true
					break
				}
			}

			if glob == "*" {
				r.Type = spec.Type(specType)
				r.Status = spec.Status(status)
				results = append(results, r)
				seen[r.Slug] = true
				break
			}
		}
	}

	return results, rows.Err()
}

// TripwireResult holds a tripwire spec with its parsed sections for display.
type TripwireResult struct {
	SearchResult
	Severity   string
	Constraint string
	Why        string
	Instead    string
	Triggers   []string
}

// FindAllTripwires returns all active tripwires with their content sections.
func (idx *DB) FindAllTripwires(heroDir string) ([]TripwireResult, error) {
	rows, err := idx.db.Query(`
		SELECT s.slug, s.title, s.type, s.status, s.path, s.tags
		FROM specs s
		WHERE s.type = 'tripwire' AND s.status = 'active'
		ORDER BY s.slug
	`)
	if err != nil {
		return nil, fmt.Errorf("querying tripwires: %w", err)
	}
	defer rows.Close()

	var results []TripwireResult
	for rows.Next() {
		var r TripwireResult
		var specType, status string
		if err := rows.Scan(&r.Slug, &r.Title, &specType, &status, &r.Path, &r.Tags); err != nil {
			return nil, err
		}
		r.Type = spec.Type(specType)
		r.Status = spec.Status(status)

		// Load full spec to get sections and triggers
		s, err := spec.ParseFile(r.Path)
		if err != nil {
			continue
		}
		r.Severity = s.Severity
		if r.Severity == "" {
			r.Severity = "high"
		}
		r.Constraint = s.Sections["constraint"]
		r.Why = s.Sections["why"]
		r.Instead = s.Sections["instead"]
		r.Triggers = s.Triggers
		results = append(results, r)
	}
	return results, rows.Err()
}

// FindTripwiresByTrigger returns active tripwires whose trigger keywords
// match any token in the query text. Matching is case-insensitive.
func (idx *DB) FindTripwiresByTrigger(query string) ([]TripwireResult, error) {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return nil, nil
	}

	// Build the slug → trigger list for active tripwires
	rows, err := idx.db.Query(`
		SELECT tt.spec_slug, tt.trigger
		FROM tripwire_triggers tt
		JOIN specs s ON s.slug = tt.spec_slug
		WHERE s.type = 'tripwire' AND s.status = 'active'
	`)
	if err != nil {
		return nil, fmt.Errorf("querying tripwire triggers: %w", err)
	}
	defer rows.Close()

	matchedSlugs := make(map[string]bool)
	for rows.Next() {
		var slug, trigger string
		if err := rows.Scan(&slug, &trigger); err != nil {
			return nil, err
		}
		trigger = strings.ToLower(trigger)
		for _, tok := range tokens {
			if tok == trigger || strings.Contains(query, trigger) {
				matchedSlugs[slug] = true
				break
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(matchedSlugs) == 0 {
		return nil, nil
	}

	// Load the matched tripwires
	var placeholders []string
	var args []interface{}
	for slug := range matchedSlugs {
		placeholders = append(placeholders, "?")
		args = append(args, slug)
	}

	specRows, err := idx.db.Query(fmt.Sprintf(`
		SELECT s.slug, s.title, s.type, s.status, s.path, s.tags
		FROM specs s
		WHERE s.slug IN (%s)
		ORDER BY s.slug
	`, strings.Join(placeholders, ",")), args...)
	if err != nil {
		return nil, fmt.Errorf("loading matched tripwires: %w", err)
	}
	defer specRows.Close()

	var results []TripwireResult
	for specRows.Next() {
		var r TripwireResult
		var specType, status string
		if err := specRows.Scan(&r.Slug, &r.Title, &specType, &status, &r.Path, &r.Tags); err != nil {
			return nil, err
		}
		r.Type = spec.Type(specType)
		r.Status = spec.Status(status)

		s, err := spec.ParseFile(r.Path)
		if err != nil {
			continue
		}
		r.Severity = s.Severity
		if r.Severity == "" {
			r.Severity = "high"
		}
		r.Constraint = s.Sections["constraint"]
		r.Why = s.Sections["why"]
		r.Instead = s.Sections["instead"]
		r.Triggers = s.Triggers
		results = append(results, r)
	}
	return results, specRows.Err()
}

// FindConflicts finds in-flight specs that touch overlapping files with the given spec.
func (idx *DB) FindConflicts(slug string) ([]ConflictResult, error) {
	rows, err := idx.db.Query(`
		SELECT DISTINCT s.slug, s.title, s.type, s.status, s.path,
			ft2.file_path, s.claimed_by
		FROM files_touched ft1
		JOIN files_touched ft2 ON ft1.file_path = ft2.file_path AND ft1.spec_slug != ft2.spec_slug
		JOIN specs s ON s.slug = ft2.spec_slug
		WHERE ft1.spec_slug = ?
		AND s.status IN ('planning', 'in-review', 'delivering')
		ORDER BY s.slug, ft2.file_path
	`, slug)
	if err != nil {
		return nil, fmt.Errorf("finding conflicts: %w", err)
	}
	defer rows.Close()

	conflictMap := make(map[string]*ConflictResult)
	var order []string

	for rows.Next() {
		var resultSlug, title, specType, status, path, filePath, claimedBy string
		if err := rows.Scan(&resultSlug, &title, &specType, &status, &path, &filePath, &claimedBy); err != nil {
			return nil, err
		}

		if _, exists := conflictMap[resultSlug]; !exists {
			conflictMap[resultSlug] = &ConflictResult{
				Slug:      resultSlug,
				Title:     title,
				Type:      spec.Type(specType),
				Status:    spec.Status(status),
				Path:      path,
				ClaimedBy: claimedBy,
			}
			order = append(order, resultSlug)
		}
		conflictMap[resultSlug].OverlappingFiles = append(conflictMap[resultSlug].OverlappingFiles, filePath)
	}

	var results []ConflictResult
	for _, slug := range order {
		results = append(results, *conflictMap[slug])
	}

	return results, rows.Err()
}

// ConflictResult represents a spec that conflicts with another.
type ConflictResult struct {
	Slug             string
	Title            string
	Type             spec.Type
	Status           spec.Status
	Path             string
	ClaimedBy        string
	OverlappingFiles []string
}

// GetRelations returns all relations for a spec (both directions).
func (idx *DB) GetRelations(slug string) ([]RelationResult, error) {
	rows, err := idx.db.Query(`
		SELECT sr.from_slug, sr.to_slug, sr.relation, s.title, s.type, s.status
		FROM spec_relations sr
		JOIN specs s ON (
			CASE WHEN sr.from_slug = ? THEN s.slug = sr.to_slug
			     ELSE s.slug = sr.from_slug END
		)
		WHERE sr.from_slug = ? OR sr.to_slug = ?
		ORDER BY sr.relation, s.slug
	`, slug, slug, slug)
	if err != nil {
		return nil, fmt.Errorf("getting relations: %w", err)
	}
	defer rows.Close()

	var results []RelationResult
	for rows.Next() {
		var r RelationResult
		var fromSlug, toSlug, specType, status string
		if err := rows.Scan(&fromSlug, &toSlug, &r.Relation, &r.Title, &specType, &status); err != nil {
			return nil, err
		}
		if fromSlug == slug {
			r.Slug = toSlug
			r.Direction = "outgoing"
		} else {
			r.Slug = fromSlug
			r.Direction = "incoming"
		}
		r.Type = spec.Type(specType)
		r.Status = spec.Status(status)
		results = append(results, r)
	}

	return results, rows.Err()
}

// RelationResult represents a related spec.
type RelationResult struct {
	Slug      string
	Title     string
	Type      spec.Type
	Status    spec.Status
	Relation  string // "parent", "child", "depends-on", "supersedes", "related"
	Direction string // "incoming" or "outgoing"
}

// Claim marks a spec as claimed by a user.
func (idx *DB) Claim(slug, claimedBy string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check spec exists
	var exists int
	if err := tx.QueryRow("SELECT COUNT(*) FROM specs WHERE slug = ?", slug).Scan(&exists); err != nil || exists == 0 {
		return fmt.Errorf("spec %q not found in index", slug)
	}

	// Check if already claimed
	var existingClaim string
	err = tx.QueryRow("SELECT claimed_by FROM claims WHERE spec_slug = ?", slug).Scan(&existingClaim)
	if err == nil && existingClaim != "" && existingClaim != claimedBy {
		return fmt.Errorf("spec %q is already claimed by %s", slug, existingClaim)
	}

	_, err = tx.Exec(`
		INSERT INTO claims (spec_slug, claimed_by, claimed_at)
		VALUES (?, ?, ?)
		ON CONFLICT(spec_slug) DO UPDATE SET
			claimed_by=excluded.claimed_by,
			claimed_at=excluded.claimed_at
	`, slug, claimedBy, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("claiming spec: %w", err)
	}

	// Also update the specs table
	_, err = tx.Exec("UPDATE specs SET claimed_by = ? WHERE slug = ?", claimedBy, slug)
	if err != nil {
		return fmt.Errorf("updating spec claim: %w", err)
	}

	return tx.Commit()
}

// Unclaim removes a claim on a spec.
func (idx *DB) Unclaim(slug string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM claims WHERE spec_slug = ?", slug); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE specs SET claimed_by = '' WHERE slug = ?", slug); err != nil {
		return err
	}

	return tx.Commit()
}

// Stats returns summary statistics about the index.
type Stats struct {
	TotalSpecs   int
	Features     int
	Bugs         int
	Conventions  int
	Decisions    int
	Initiatives  int
	Rules        int
	External     int
	Context      int
	Notes        int
	Planning     int
	InReview     int
	Delivering   int
	Completed    int
	Active       int
	Accepted     int
	FilesTracked int
	DecisionDocs int
	RootCauses   int
	Claims       int
}

// GetStats returns index statistics.
func (idx *DB) GetStats() (Stats, error) {
	var s Stats

	queries := []struct {
		sql  string
		dest *int
	}{
		{"SELECT COUNT(*) FROM specs", &s.TotalSpecs},
		{"SELECT COUNT(*) FROM specs WHERE type = 'feature'", &s.Features},
		{"SELECT COUNT(*) FROM specs WHERE type = 'bug'", &s.Bugs},
		{"SELECT COUNT(*) FROM specs WHERE type = 'convention'", &s.Conventions},
		{"SELECT COUNT(*) FROM specs WHERE type = 'decision'", &s.Decisions},
		{"SELECT COUNT(*) FROM specs WHERE type = 'initiative'", &s.Initiatives},
		{"SELECT COUNT(*) FROM specs WHERE type = 'rule'", &s.Rules},
		{"SELECT COUNT(*) FROM specs WHERE type = 'external'", &s.External},
		{"SELECT COUNT(*) FROM specs WHERE type = 'context'", &s.Context},
		{"SELECT COUNT(*) FROM specs WHERE type = 'note'", &s.Notes},
		{"SELECT COUNT(*) FROM specs WHERE status = 'planning'", &s.Planning},
		{"SELECT COUNT(*) FROM specs WHERE status = 'in-review'", &s.InReview},
		{"SELECT COUNT(*) FROM specs WHERE status = 'delivering'", &s.Delivering},
		{"SELECT COUNT(*) FROM specs WHERE status = 'completed'", &s.Completed},
		{"SELECT COUNT(*) FROM specs WHERE status = 'active'", &s.Active},
		{"SELECT COUNT(*) FROM specs WHERE status = 'accepted'", &s.Accepted},
		{"SELECT COUNT(*) FROM files_touched", &s.FilesTracked},
		{"SELECT COUNT(*) FROM decisions", &s.DecisionDocs},
		{"SELECT COUNT(*) FROM root_causes", &s.RootCauses},
		{"SELECT COUNT(*) FROM claims", &s.Claims},
	}

	for _, q := range queries {
		if err := idx.db.QueryRow(q.sql).Scan(q.dest); err != nil {
			return s, fmt.Errorf("querying stats: %w", err)
		}
	}

	return s, nil
}

// AllSpecs returns all specs in the index.
func (idx *DB) AllSpecs() ([]SearchResult, error) {
	rows, err := idx.db.Query(`
		SELECT slug, title, type, status, path, '' as snippet, claimed_by, tags, domain, superseded_by
		FROM specs
		ORDER BY modified_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// SequenceItem represents a spec in a suggested delivery sequence.
type SequenceItem struct {
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Type          spec.Type   `json:"type"`
	Status        spec.Status `json:"status"`
	Order         int      `json:"order"`
	DependsOn     []string `json:"depends_on,omitempty"`
	ConflictsWith []string `json:"conflicts_with,omitempty"`
	Reason        string   `json:"reason"`
}

// SuggestSequence returns in-flight specs in a recommended delivery order.
// Order is determined by: (1) topological sort on depends-on relations,
// (2) specs with fewer file conflicts first among peers at the same topo level.
func (idx *DB) SuggestSequence() ([]SequenceItem, error) {
	// Get all in-flight specs
	allSpecs, err := idx.ListFiltered("", "planning,in-review,delivering", "", "")
	if err != nil {
		return nil, fmt.Errorf("listing specs: %w", err)
	}
	if len(allSpecs) == 0 {
		return nil, nil
	}

	slugSet := make(map[string]SearchResult)
	for _, s := range allSpecs {
		slugSet[s.Slug] = s
	}

	// Build dependency graph (depends-on edges)
	dependsOn := make(map[string][]string) // slug -> slugs it depends on
	for slug := range slugSet {
		rels, err := idx.GetRelations(slug)
		if err != nil {
			continue
		}
		for _, r := range rels {
			if r.Relation == "depends-on" && r.Direction == "outgoing" {
				if _, inFlight := slugSet[r.Slug]; inFlight {
					dependsOn[slug] = append(dependsOn[slug], r.Slug)
				}
			}
		}
	}

	// Build conflict counts
	conflictMap := make(map[string][]string) // slug -> conflicting slugs
	for slug := range slugSet {
		conflicts, err := idx.FindConflicts(slug)
		if err != nil {
			continue
		}
		for _, c := range conflicts {
			if _, inFlight := slugSet[c.Slug]; inFlight {
				conflictMap[slug] = append(conflictMap[slug], c.Slug)
			}
		}
	}

	// Kahn's algorithm for topological sort
	inDegree := make(map[string]int)
	for slug := range slugSet {
		inDegree[slug] = 0
	}
	for slug, deps := range dependsOn {
		_ = slug
		for _, dep := range deps {
			inDegree[dep] = inDegree[dep] // ensure entry exists
		}
		inDegree[slug] += len(deps)
	}

	// Wait — in depends-on: A depends-on B means B must come first.
	// So edges go from dependent → dependency. In-degree for topo should count
	// how many things depend on you (reverse). Let me recalculate:
	// Actually for Kahn's: if A depends-on B, then B must be delivered first.
	// Edge: B → A (B before A). In-degree of A increases.
	inDegree = make(map[string]int)
	for slug := range slugSet {
		inDegree[slug] = 0
	}
	for slug, deps := range dependsOn {
		inDegree[slug] += len(deps) // A has in-degree = number of deps
		for _, dep := range deps {
			_ = dep // dep → slug edge
		}
	}

	var queue []string
	for slug := range slugSet {
		if inDegree[slug] == 0 {
			queue = append(queue, slug)
		}
	}

	// Sort queue by conflict count (fewer conflicts first)
	sortByConflicts := func(slugs []string) {
		sort.Slice(slugs, func(i, j int) bool {
			ci, cj := len(conflictMap[slugs[i]]), len(conflictMap[slugs[j]])
			if ci != cj {
				return ci < cj
			}
			return slugs[i] < slugs[j]
		})
	}

	sortByConflicts(queue)

	var result []SequenceItem
	order := 1
	for len(queue) > 0 {
		slug := queue[0]
		queue = queue[1:]

		s := slugSet[slug]
		reason := "no dependencies"
		if deps := dependsOn[slug]; len(deps) > 0 {
			reason = fmt.Sprintf("depends on: %s", strings.Join(deps, ", "))
		} else if conflicts := conflictMap[slug]; len(conflicts) > 0 {
			reason = fmt.Sprintf("fewer file conflicts (%d)", len(conflicts))
		}

		result = append(result, SequenceItem{
			Slug:          slug,
			Title:         s.Title,
			Type:          s.Type,
			Status:        s.Status,
			Order:         order,
			DependsOn:     dependsOn[slug],
			ConflictsWith: conflictMap[slug],
			Reason:        reason,
		})
		order++

		// Remove edges and add newly free nodes
		var newlyFree []string
		for depSlug, deps := range dependsOn {
			for _, d := range deps {
				if d == slug {
					inDegree[depSlug]--
					if inDegree[depSlug] == 0 {
						newlyFree = append(newlyFree, depSlug)
					}
				}
			}
		}
		sortByConflicts(newlyFree)
		queue = append(queue, newlyFree...)
	}

	// Add any remaining (cyclic) specs
	for slug := range slugSet {
		found := false
		for _, item := range result {
			if item.Slug == slug {
				found = true
				break
			}
		}
		if !found {
			s := slugSet[slug]
			result = append(result, SequenceItem{
				Slug:          slug,
				Title:         s.Title,
				Type:          s.Type,
				Status:        s.Status,
				Order:         order,
				DependsOn:     dependsOn[slug],
				ConflictsWith: conflictMap[slug],
				Reason:        "circular dependency — manual ordering needed",
			})
			order++
		}
	}

	return result, nil
}

// CheckStale finds planning specs older than the given number of days.
func (idx *DB) CheckStale(staleDays int) ([]SearchResult, error) {
	cutoff := time.Now().AddDate(0, 0, -staleDays).Format(time.RFC3339)

	rows, err := idx.db.Query(`
		SELECT slug, title, type, status, path, '' as snippet, claimed_by, tags, domain, superseded_by
		FROM specs
		WHERE status IN ('planning', 'in-review')
		AND modified_at < ?
		ORDER BY modified_at ASC
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("checking stale specs: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// CheckUnclaimed finds planning/in-review specs that are not claimed.
func (idx *DB) CheckUnclaimed() ([]SearchResult, error) {
	rows, err := idx.db.Query(`
		SELECT s.slug, s.title, s.type, s.status, s.path, '' as snippet, s.claimed_by, s.tags, s.domain, s.superseded_by
		FROM specs s
		LEFT JOIN claims c ON s.slug = c.spec_slug
		WHERE s.status IN ('planning', 'in-review')
		AND c.spec_slug IS NULL
		ORDER BY s.modified_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("checking unclaimed specs: %w", err)
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

func scanSearchResults(rows *sql.Rows) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var specType, status string
		if err := rows.Scan(&r.Slug, &r.Title, &specType, &status, &r.Path, &r.Snippet, &r.ClaimedBy, &r.Tags, &r.Domain, &r.SupersededBy); err != nil {
			return nil, err
		}
		r.Type = spec.Type(specType)
		r.Status = spec.Status(status)
		results = append(results, r)
	}
	return results, rows.Err()
}

// Rebuild re-indexes all specs found under the hero directory.
func Rebuild(heroDir string) (*Stats, error) {
	idx, err := Open(heroDir)
	if err != nil {
		return nil, err
	}
	defer idx.Close()

	// Clear existing data
	for _, table := range []string{"fts_specs", "files_touched", "root_causes", "decisions",
		"convention_scopes", "tripwire_triggers", "spec_relations", "claims", "specs"} {
		if _, err := idx.db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return nil, fmt.Errorf("clearing %s: %w", table, err)
		}
	}

	// Discover and index all specs
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}

	for _, s := range specs {
		content, err := os.ReadFile(s.Path)
		if err != nil {
			continue
		}
		if err := idx.IndexSpec(s, string(content)); err != nil {
			return nil, fmt.Errorf("indexing %s: %w", s.Slug, err)
		}
	}

	// Project graph nodes into fts_nodes if graph.db exists.
	graphPath := filepath.Join(heroDir, "graph.db")
	if _, statErr := os.Stat(graphPath); statErr == nil {
		graphDB, openErr := sql.Open("sqlite", graphPath)
		if openErr == nil {
			_, _ = idx.ProjectGraphNodes(graphDB)
			graphDB.Close()
		}
	}

	stats, err := idx.GetStats()
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// BuildContext generates a context injection block for a set of file paths.
// Returns conventions that match, past specs that touched the same files,
// decisions that apply, and known risks.
func (idx *DB) BuildContext(filePaths []string) (*ContextBlock, error) {
	ctx := &ContextBlock{}

	// Find matching conventions
	conventions, err := idx.FindConventionsForFiles(filePaths)
	if err != nil {
		return nil, fmt.Errorf("finding conventions: %w", err)
	}
	for _, c := range conventions {
		ctx.Conventions = append(ctx.Conventions, ContextEntry{
			Slug:    c.Slug,
			Title:   c.Title,
			Type:    c.Type,
			Status:  c.Status,
			Path:    c.Path,
			Summary: c.Snippet,
		})
	}

	// Find matching rules (hard constraints, use same scope matching as conventions)
	rules, err := idx.FindRulesForFiles(filePaths)
	if err != nil {
		return nil, fmt.Errorf("finding rules: %w", err)
	}
	for _, r := range rules {
		ctx.Rules = append(ctx.Rules, ContextEntry{
			Slug:    r.Slug,
			Title:   r.Title,
			Type:    r.Type,
			Status:  r.Status,
			Path:    r.Path,
			Summary: r.Snippet,
		})
	}

	// Find matching tripwires (forbidden-option guardrails)
	tripwires, err := idx.FindTripwiresForFiles(filePaths)
	if err != nil {
		return nil, fmt.Errorf("finding tripwires: %w", err)
	}
	for _, tw := range tripwires {
		ctx.Tripwires = append(ctx.Tripwires, ContextEntry{
			Slug:    tw.Slug,
			Title:   tw.Title,
			Type:    tw.Type,
			Status:  tw.Status,
			Path:    tw.Path,
			Summary: tw.Snippet,
		})
	}

	// Find past specs that touched these files — split into in-flight vs completed
	seen := make(map[string]bool)
	for _, fp := range filePaths {
		results, err := idx.SearchByFile(fp)
		if err != nil {
			continue
		}
		for _, r := range results {
			if seen[r.Slug] || r.Type == spec.TypeConvention {
				continue
			}
			entry := ContextEntry{
				Slug:         r.Slug,
				Title:        r.Title,
				Type:         r.Type,
				Status:       r.Status,
				Path:         r.Path,
				Summary:      r.Snippet,
				SupersededBy: r.SupersededBy,
			}
			if r.Status == spec.StatusDelivering || r.Status == spec.StatusPlanning || r.Status == spec.StatusInReview {
				ctx.InFlight = append(ctx.InFlight, entry)
			} else {
				ctx.PastWork = append(ctx.PastWork, entry)
			}
			seen[r.Slug] = true
		}
	}

	// Find decisions
	decRows, err := idx.db.Query(`
		SELECT slug, title, type, status, path, '' as snippet, claimed_by, tags, domain, superseded_by
		FROM specs WHERE type = 'decision' AND status = 'accepted'
		ORDER BY modified_at DESC
	`)
	if err == nil {
		defer decRows.Close()
		decResults, _ := scanSearchResults(decRows)
		for _, r := range decResults {
			ctx.Decisions = append(ctx.Decisions, ContextEntry{
				Slug:   r.Slug,
				Title:  r.Title,
				Type:   r.Type,
				Status: r.Status,
				Path:   r.Path,
			})
		}
	}

	// Find bug specs that touched these files (known risks)
	for _, fp := range filePaths {
		pattern := "%" + fp + "%"
		bugRows, err := idx.db.Query(`
			SELECT DISTINCT s.slug, s.title, s.type, s.status, s.path
			FROM files_touched ft
			JOIN specs s ON s.slug = ft.spec_slug
			WHERE ft.file_path LIKE ? AND s.type = 'bug'
			ORDER BY s.modified_at DESC
		`, pattern)
		if err != nil {
			continue
		}
		for bugRows.Next() {
			var slug, title, specType, status, path string
			if err := bugRows.Scan(&slug, &title, &specType, &status, &path); err != nil {
				continue
			}
			if !seen[slug+"_risk"] {
				ctx.KnownRisks = append(ctx.KnownRisks, ContextEntry{
					Slug:   slug,
					Title:  title,
					Type:   spec.Type(specType),
					Status: spec.Status(status),
					Path:   path,
				})
				seen[slug+"_risk"] = true
			}
		}
		bugRows.Close()
	}

	// Find external knowledge entries (reference docs, runbooks)
	extRows, err := idx.db.Query(`
		SELECT slug, title, type, status, path, '' as snippet, claimed_by, tags, domain, superseded_by
		FROM specs WHERE type = 'external'
		ORDER BY modified_at DESC
	`)
	if err == nil {
		defer extRows.Close()
		extResults, _ := scanSearchResults(extRows)
		for _, r := range extResults {
			ctx.External = append(ctx.External, ContextEntry{
				Slug:   r.Slug,
				Title:  r.Title,
				Type:   r.Type,
				Status: r.Status,
				Path:   r.Path,
			})
		}
	}

	// Detect test coverage for queried files (only for files that exist on disk)
	for _, fp := range filePaths {
		if isTestFile(fp) {
			continue
		}
		// Only report coverage for files that actually exist
		if _, err := os.Stat(fp); err != nil {
			continue
		}
		entry := TestCoverageEntry{FilePath: fp}
		testFile := findTestFile(fp)
		if testFile != "" {
			entry.TestFile = testFile
			entry.HasTest = true
		}
		ctx.TestCoverage = append(ctx.TestCoverage, entry)
	}

	return ctx, nil
}

// isTestFile returns true if the file path looks like a test file.
func isTestFile(fp string) bool {
	base := filepath.Base(fp)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Go: _test.go
	if strings.HasSuffix(fp, "_test.go") {
		return true
	}
	// JS/TS: .test.ts, .spec.ts, .test.tsx, .spec.tsx, .test.js, .spec.js
	for _, suffix := range []string{".test", ".spec"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	// Python: test_*.py or *_test.py
	if ext == ".py" && (strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test")) {
		return true
	}
	// Java: *Test.java, *Tests.java, *Spec.java
	if ext == ".java" && (strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "Tests") || strings.HasSuffix(name, "Spec")) {
		return true
	}
	// Ruby: *_spec.rb, *_test.rb
	if ext == ".rb" && (strings.HasSuffix(name, "_spec") || strings.HasSuffix(name, "_test")) {
		return true
	}
	// Directory-based: file is inside a __tests__ or test/ or tests/ directory
	if strings.Contains(fp, "/__tests__/") || strings.Contains(fp, "/test/") || strings.Contains(fp, "/tests/") {
		return true
	}
	return false
}

// findTestFile attempts to locate a test file for a given source file.
// Returns the path if found, empty string otherwise.
func findTestFile(fp string) string {
	dir := filepath.Dir(fp)
	base := filepath.Base(fp)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var candidates []string

	switch ext {
	case ".go":
		candidates = []string{
			filepath.Join(dir, name+"_test.go"),
		}
	case ".ts", ".tsx", ".js", ".jsx":
		// Try .test and .spec variants
		for _, suffix := range []string{".test", ".spec"} {
			candidates = append(candidates, filepath.Join(dir, name+suffix+ext))
		}
		// Try __tests__ directory
		candidates = append(candidates, filepath.Join(dir, "__tests__", base))
		candidates = append(candidates, filepath.Join(dir, "__tests__", name+".test"+ext))
	case ".py":
		candidates = []string{
			filepath.Join(dir, "test_"+base),
			filepath.Join(dir, name+"_test.py"),
			filepath.Join(dir, "tests", "test_"+base),
		}
	case ".java":
		candidates = []string{
			filepath.Join(dir, name+"Test.java"),
			filepath.Join(dir, name+"Tests.java"),
		}
		// Also check src/test mirroring
		if idx := strings.Index(fp, "/src/main/"); idx >= 0 {
			testPath := fp[:idx] + "/src/test/" + fp[idx+len("/src/main/"):]
			testDir := filepath.Dir(testPath)
			candidates = append(candidates, filepath.Join(testDir, name+"Test.java"))
		}
	case ".rb":
		candidates = []string{
			filepath.Join(dir, name+"_spec.rb"),
			filepath.Join(dir, name+"_test.rb"),
		}
		// spec/ mirror
		if idx := strings.Index(fp, "/app/"); idx >= 0 {
			specPath := fp[:idx] + "/spec/" + fp[idx+len("/app/"):]
			specDir := filepath.Dir(specPath)
			candidates = append(candidates, filepath.Join(specDir, name+"_spec.rb"))
		}
	case ".rs":
		// Rust tests are usually inline, but check for separate test files
		candidates = []string{
			filepath.Join(dir, "tests", base),
			filepath.Join(dir, name+"_test.rs"),
		}
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// ContextBlock holds the context injection data for a set of files.
type ContextBlock struct {
	Tripwires     []ContextEntry     // forbidden-option guardrails matching file scope
	Conventions   []ContextEntry
	Rules         []ContextEntry
	InFlight      []ContextEntry     // specs currently being delivered on these files
	PastWork      []ContextEntry
	Decisions     []ContextEntry
	KnownRisks    []ContextEntry
	External      []ContextEntry
	CodeStructure []CodeContextEntry // code intelligence for relevant packages
	TestCoverage  []TestCoverageEntry // test coverage for queried files
}

// ContextEntry is a single item in a context block.
type ContextEntry struct {
	Slug    string
	Title   string
	Type    spec.Type
	Status  spec.Status
	Path    string
	Summary string
	// SupersededBy carries the replacement slug when this entry refers
	// to a spec that's been superseded. Renderers add a redirect marker
	// in the output so agents follow the replacement, not this entry.
	SupersededBy string
}

// CodeContextEntry holds code structure info for a package relevant to the queried files.
type CodeContextEntry struct {
	PackageName string
	PackagePath string
	Language    string
	Content     string // the full spec.md content for this package
}

// TestCoverageEntry reports whether a source file has an associated test file.
type TestCoverageEntry struct {
	FilePath string // the source file
	TestFile string // the test file path (empty if no test found)
	HasTest  bool
}

// IsEmpty returns true if the context block has no entries.
func (cb *ContextBlock) IsEmpty() bool {
	return len(cb.Tripwires) == 0 && len(cb.Conventions) == 0 && len(cb.Rules) == 0 &&
		len(cb.InFlight) == 0 && len(cb.PastWork) == 0 &&
		len(cb.Decisions) == 0 && len(cb.KnownRisks) == 0 &&
		len(cb.External) == 0 && len(cb.CodeStructure) == 0 &&
		len(cb.TestCoverage) == 0
}

// NudgeResult holds the output of a nudge check for a set of files.
type NudgeResult struct {
	Conventions    []ContextEntry // active conventions that apply to these files
	RelatedSpecs   []ContextEntry // past specs in this area
	PendingSpecs   []ContextEntry // in-flight specs touching the same files
	HasConventions bool
	HasPastWork    bool
	HasPending     bool
}

// IsEmpty returns true if the nudge has nothing to report.
func (n *NudgeResult) IsEmpty() bool {
	return !n.HasConventions && !n.HasPastWork && !n.HasPending
}

// BuildNudge checks what context exists for the given files and returns a
// lightweight nudge result. This is used when an agent is working without
// a spec to surface relevant conventions and past work.
func (idx *DB) BuildNudge(filePaths []string) (*NudgeResult, error) {
	result := &NudgeResult{}

	// Find matching conventions
	conventions, err := idx.FindConventionsForFiles(filePaths)
	if err != nil {
		return nil, fmt.Errorf("finding conventions for nudge: %w", err)
	}
	for _, c := range conventions {
		result.Conventions = append(result.Conventions, ContextEntry{
			Slug:   c.Slug,
			Title:  c.Title,
			Type:   c.Type,
			Status: c.Status,
			Path:   c.Path,
		})
	}
	result.HasConventions = len(result.Conventions) > 0

	// Find past work that touched these files
	seen := make(map[string]bool)
	for _, fp := range filePaths {
		results, err := idx.SearchByFile(fp)
		if err != nil {
			continue
		}
		for _, r := range results {
			// Surface completed specs and superseded specs. Superseded
			// entries are kept (not filtered) so the agent learns
			// "this is the old answer, follow <new-slug>" instead of
			// silently losing the lineage. Renderers add the redirect
			// marker from SupersededBy.
			if !seen[r.Slug] && (r.Status == spec.StatusCompleted || r.SupersededBy != "" || r.Status == spec.StatusSuperseded) {
				result.RelatedSpecs = append(result.RelatedSpecs, ContextEntry{
					Slug:         r.Slug,
					Title:        r.Title,
					Type:         r.Type,
					Status:       r.Status,
					SupersededBy: r.SupersededBy,
				})
				seen[r.Slug] = true
			}
		}
	}
	result.HasPastWork = len(result.RelatedSpecs) > 0

	// Find in-flight specs touching the same files
	for _, fp := range filePaths {
		results, err := idx.SearchByFile(fp)
		if err != nil {
			continue
		}
		for _, r := range results {
			if !seen[r.Slug] && (r.Status == spec.StatusPlanning || r.Status == spec.StatusInReview || r.Status == spec.StatusDelivering) {
				result.PendingSpecs = append(result.PendingSpecs, ContextEntry{
					Slug:   r.Slug,
					Title:  r.Title,
					Type:   r.Type,
					Status: r.Status,
				})
				seen[r.Slug] = true
			}
		}
	}
	result.HasPending = len(result.PendingSpecs) > 0

	return result, nil
}
