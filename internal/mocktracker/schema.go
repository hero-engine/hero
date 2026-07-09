package mocktracker

// schemaDDL is the tracker-neutral relational schema the Sprout seed YAML
// targets. It is created BEFORE seeding (sprout.Apply only writes rows; it
// never creates the target tables). One canonical schema is projected into
// each tracker's wire shape by the per-mode handlers, which is what lets a
// single Acme seed back all four tracker modes at once.
//
// Column names mirror the mock-tracker-server spec verbatim. `start` / `end`
// are quoted everywhere they appear in queries (end is a SQLite keyword).
var schemaDDL = []string{
	`CREATE TABLE issue (
		global_id    TEXT PRIMARY KEY,
		type         TEXT,
		title        TEXT,
		body         TEXT,
		epic_id      TEXT,
		milestone_id TEXT,
		iteration_id TEXT,
		status       TEXT,
		assignee     TEXT,
		weight       INTEGER,
		severity     TEXT
	)`,
	`CREATE TABLE epic (
		global_id TEXT PRIMARY KEY,
		title     TEXT,
		parent_id TEXT
	)`,
	`CREATE TABLE milestone (
		global_id TEXT PRIMARY KEY,
		title     TEXT,
		due       TEXT
	)`,
	`CREATE TABLE iteration (
		global_id TEXT PRIMARY KEY,
		name      TEXT,
		"start"   TEXT,
		"end"     TEXT
	)`,
	`CREATE TABLE label (
		issue_id TEXT,
		name     TEXT
	)`,
	`CREATE TABLE app_user (
		username TEXT PRIMARY KEY,
		email    TEXT,
		display  TEXT
	)`,
	// id_alias maps a stable global_id to the per-tracker IID (GitHub
	// number / GitLab iid). /__admin/rotate-ids rewrites iid while the
	// global_id stays put, validating the stable-external-id contract.
	`CREATE TABLE id_alias (
		global_id TEXT PRIMARY KEY,
		iid       TEXT
	)`,
}
