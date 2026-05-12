package store

// migration represents a single schema migration.
type migration struct {
	Version int
	Name    string
	SQL     string
}

// migrations is the ordered list of all schema migrations.
var migrations = []migration{
	{
		Version: 1,
		Name:    "initial schema",
		SQL: `
-- Organizations
CREATE TABLE IF NOT EXISTS orgs (
	id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name        TEXT NOT NULL,
	slug        TEXT NOT NULL UNIQUE,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Users
CREATE TABLE IF NOT EXISTS users (
	id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	email       TEXT NOT NULL UNIQUE,
	name        TEXT NOT NULL DEFAULT '',
	avatar_url  TEXT NOT NULL DEFAULT '',
	provider    TEXT NOT NULL DEFAULT 'github',
	provider_id TEXT NOT NULL DEFAULT '',
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider
	ON users (provider, provider_id) WHERE provider_id != '';

-- Org membership
CREATE TABLE IF NOT EXISTS org_members (
	org_id    UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	role      TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
	joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (org_id, user_id)
);

-- Teams (optional grouping within an org)
CREATE TABLE IF NOT EXISTS teams (
	id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	org_id  UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	name    TEXT NOT NULL,
	UNIQUE (org_id, name)
);

CREATE TABLE IF NOT EXISTS team_members (
	team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	PRIMARY KEY (team_id, user_id)
);

-- Repos linked to an org
CREATE TABLE IF NOT EXISTS repos (
	id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	org_id       UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	push_url     TEXT NOT NULL DEFAULT '',
	last_sync_at TIMESTAMPTZ,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (org_id, name)
);

-- API keys for CLI authentication
CREATE TABLE IF NOT EXISTS api_keys (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	org_id     UUID REFERENCES orgs(id) ON DELETE CASCADE,
	name       TEXT NOT NULL DEFAULT 'default',
	key_hash   TEXT NOT NULL UNIQUE,
	prefix     TEXT NOT NULL,
	scopes     TEXT[] NOT NULL DEFAULT '{}',
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	last_used  TIMESTAMPTZ
);

-- Refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL UNIQUE,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Synced specs (metadata from CLI push)
CREATE TABLE IF NOT EXISTS specs (
	id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	repo_id        UUID NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
	slug           TEXT NOT NULL,
	title          TEXT NOT NULL DEFAULT '',
	type           TEXT NOT NULL DEFAULT 'feature',
	status         TEXT NOT NULL DEFAULT 'planning',
	priority       TEXT NOT NULL DEFAULT '',
	claimed_by     TEXT NOT NULL DEFAULT '',
	tracker_id     TEXT NOT NULL DEFAULT '',
	tags           TEXT[] NOT NULL DEFAULT '{}',
	files_touched  TEXT[] NOT NULL DEFAULT '{}',
	sections       JSONB NOT NULL DEFAULT '{}',
	score          INTEGER,
	raw_content    TEXT NOT NULL DEFAULT '',
	checksum       TEXT NOT NULL DEFAULT '',
	synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (repo_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_specs_status ON specs (status);
CREATE INDEX IF NOT EXISTS idx_specs_type ON specs (type);
CREATE INDEX IF NOT EXISTS idx_specs_tracker ON specs (tracker_id) WHERE tracker_id != '';

-- Knowledge entries
CREATE TABLE IF NOT EXISTS knowledge (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	repo_id    UUID NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
	category   TEXT NOT NULL,
	slug       TEXT NOT NULL,
	title      TEXT NOT NULL DEFAULT '',
	content    TEXT NOT NULL DEFAULT '',
	checksum   TEXT NOT NULL DEFAULT '',
	synced_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (repo_id, category, slug)
);

-- Activity events (append-only)
CREATE TABLE IF NOT EXISTS activity_events (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	repo_id    UUID REFERENCES repos(id) ON DELETE SET NULL,
	user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
	event_type TEXT NOT NULL,
	payload    JSONB NOT NULL DEFAULT '{}',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activity_org_time
	ON activity_events (org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_type
	ON activity_events (event_type);
`,
	},
	{
		Version: 2,
		Name:    "github app governance",
		SQL: `
-- GitHub App installations linked to orgs
CREATE TABLE IF NOT EXISTS github_installations (
	id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	org_id           UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	installation_id  BIGINT NOT NULL UNIQUE,
	account_login    TEXT NOT NULL DEFAULT '',
	account_type     TEXT NOT NULL DEFAULT 'Organization',
	governance_mode  TEXT NOT NULL DEFAULT 'advisory' CHECK (governance_mode IN ('advisory', 'enforcement', 'disabled')),
	created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_installations_org ON github_installations (org_id);

-- PR spec-linkage check log (append-only)
CREATE TABLE IF NOT EXISTS pr_checks (
	id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	installation_id  UUID NOT NULL REFERENCES github_installations(id) ON DELETE CASCADE,
	repo_full_name   TEXT NOT NULL,
	pr_number        INTEGER NOT NULL,
	head_sha         TEXT NOT NULL,
	spec_slugs       TEXT[] NOT NULL DEFAULT '{}',
	has_spec         BOOLEAN NOT NULL DEFAULT false,
	conclusion       TEXT NOT NULL DEFAULT 'neutral',
	created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pr_checks_repo ON pr_checks (repo_full_name, pr_number);
CREATE INDEX IF NOT EXISTS idx_pr_checks_installation ON pr_checks (installation_id, created_at DESC);
`,
	},
	{
		Version: 3,
		Name:    "conventions",
		SQL: `
-- Conventions synced from repos
CREATE TABLE IF NOT EXISTS conventions (
	id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	repo_id    UUID NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
	slug       TEXT NOT NULL,
	title      TEXT NOT NULL DEFAULT '',
	status     TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active')),
	scope      TEXT[] NOT NULL DEFAULT '{}',
	content    TEXT NOT NULL DEFAULT '',
	checksum   TEXT NOT NULL DEFAULT '',
	synced_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (repo_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_conventions_repo_status ON conventions (repo_id, status);
`,
	},
	{
		Version: 4,
		Name:    "cross-org intelligence",
		SQL: `
-- Tech stack profiles for similarity matching (per-org, anonymized)
CREATE TABLE IF NOT EXISTS stack_profiles (
	org_id       UUID PRIMARY KEY REFERENCES orgs(id) ON DELETE CASCADE,
	languages    TEXT[] NOT NULL DEFAULT '{}',
	frameworks   TEXT[] NOT NULL DEFAULT '{}',
	spec_count   INTEGER NOT NULL DEFAULT 0,
	member_count INTEGER NOT NULL DEFAULT 0,
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Global anonymized pattern pool (no org-identifying data)
CREATE TABLE IF NOT EXISTS global_patterns (
	id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	pattern_type TEXT NOT NULL,
	category     TEXT NOT NULL DEFAULT '',
	title        TEXT NOT NULL,
	description  TEXT NOT NULL,
	frequency    INTEGER NOT NULL DEFAULT 1,
	languages    TEXT[] NOT NULL DEFAULT '{}',
	frameworks   TEXT[] NOT NULL DEFAULT '{}',
	metadata     JSONB NOT NULL DEFAULT '{}',
	created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (pattern_type, title)
);

CREATE INDEX IF NOT EXISTS idx_global_patterns_type ON global_patterns (pattern_type);
CREATE INDEX IF NOT EXISTS idx_global_patterns_langs ON global_patterns USING GIN (languages);

-- Global convention templates (anonymized, derived from popular conventions)
CREATE TABLE IF NOT EXISTS global_conventions (
	id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	slug        TEXT NOT NULL UNIQUE,
	title       TEXT NOT NULL,
	category    TEXT NOT NULL DEFAULT 'general',
	description TEXT NOT NULL DEFAULT '',
	template    TEXT NOT NULL DEFAULT '',
	scope       TEXT[] NOT NULL DEFAULT '{}',
	languages   TEXT[] NOT NULL DEFAULT '{}',
	frameworks  TEXT[] NOT NULL DEFAULT '{}',
	adoption    INTEGER NOT NULL DEFAULT 1,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_global_conventions_category ON global_conventions (category);
CREATE INDEX IF NOT EXISTS idx_global_conventions_langs ON global_conventions USING GIN (languages);

-- Tracks which orgs have opted into cross-org intelligence
CREATE TABLE IF NOT EXISTS intelligence_opt_in (
	org_id     UUID PRIMARY KEY REFERENCES orgs(id) ON DELETE CASCADE,
	opted_in   BOOLEAN NOT NULL DEFAULT false,
	opted_at   TIMESTAMPTZ,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`,
	},
	{
		Version: 5,
		Name:    "knowledge graph (federation phase 7)",
		SQL: `
-- Server-side mirror of internal/graph/. Holds the team-scope and
-- unit-scope deltas pushed from per-developer clients. Clients pull
-- since a server_time cursor (server-controlled monotonic clock —
-- avoids client-clock skew breaking incremental pulls).
--
-- Edges store endpoints by (type, key) rather than row id because
-- different clients have independent row id spaces.

CREATE TABLE IF NOT EXISTS graph_nodes (
	id           BIGSERIAL PRIMARY KEY,
	org_id       UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	repo         TEXT NOT NULL DEFAULT '',
	unit         TEXT NOT NULL DEFAULT '',
	type         TEXT NOT NULL,
	key          TEXT NOT NULL,
	props        JSONB NOT NULL,
	scope        TEXT NOT NULL,
	content_hash TEXT,
	source       JSONB NOT NULL,
	valid_from   TIMESTAMPTZ NOT NULL,
	valid_to     TIMESTAMPTZ,
	ingested_at  TIMESTAMPTZ NOT NULL,
	client_id    TEXT NOT NULL DEFAULT '',
	server_time  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_graph_nodes_current
	ON graph_nodes(org_id, type, key) WHERE valid_to IS NULL;
CREATE INDEX IF NOT EXISTS idx_graph_nodes_cursor
	ON graph_nodes(org_id, server_time);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_repo
	ON graph_nodes(org_id, repo);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_unit
	ON graph_nodes(org_id, unit) WHERE unit != '';

CREATE TABLE IF NOT EXISTS graph_edges (
	id           BIGSERIAL PRIMARY KEY,
	org_id       UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	repo         TEXT NOT NULL DEFAULT '',
	unit         TEXT NOT NULL DEFAULT '',
	from_type    TEXT NOT NULL,
	from_key     TEXT NOT NULL,
	to_type      TEXT NOT NULL,
	to_key       TEXT NOT NULL,
	type         TEXT NOT NULL,
	props        JSONB NOT NULL DEFAULT '{}',
	scope        TEXT NOT NULL,
	source       JSONB NOT NULL,
	valid_from   TIMESTAMPTZ NOT NULL,
	valid_to     TIMESTAMPTZ,
	ingested_at  TIMESTAMPTZ NOT NULL,
	client_id    TEXT NOT NULL DEFAULT '',
	server_time  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_graph_edges_current
	ON graph_edges(org_id, from_type, from_key, type, to_type, to_key)
	WHERE valid_to IS NULL;
CREATE INDEX IF NOT EXISTS idx_graph_edges_cursor
	ON graph_edges(org_id, server_time);
CREATE INDEX IF NOT EXISTS idx_graph_edges_repo
	ON graph_edges(org_id, repo);
`,
	},
	{
		Version: 6,
		Name:    "graph schema simplification — upsert + history",
		SQL: `
-- Replace the bitemporal valid_to pattern with simple primary-key tables.
-- Current state: one row per (org_id, type, key) / (org_id, from..to..type).
-- History: append-only audit log written when a row changes.

DROP TABLE IF EXISTS graph_edges;
DROP TABLE IF EXISTS graph_nodes;

CREATE TABLE graph_nodes (
	org_id      UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	repo        TEXT        NOT NULL DEFAULT '',
	unit        TEXT        NOT NULL DEFAULT '',
	type        TEXT        NOT NULL,
	key         TEXT        NOT NULL,
	props       JSONB       NOT NULL DEFAULT '{}',
	scope       TEXT        NOT NULL DEFAULT '',
	hash        TEXT        NOT NULL DEFAULT '',
	source      JSONB       NOT NULL DEFAULT '{}',
	client_id   TEXT        NOT NULL DEFAULT '',
	server_time TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (org_id, type, key)
);

CREATE INDEX idx_graph_nodes_cursor ON graph_nodes (org_id, server_time);
CREATE INDEX idx_graph_nodes_repo   ON graph_nodes (org_id, repo);
CREATE INDEX idx_graph_nodes_unit   ON graph_nodes (org_id, unit) WHERE unit != '';

CREATE TABLE graph_edges (
	org_id      UUID        NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
	repo        TEXT        NOT NULL DEFAULT '',
	unit        TEXT        NOT NULL DEFAULT '',
	from_type   TEXT        NOT NULL,
	from_key    TEXT        NOT NULL,
	to_type     TEXT        NOT NULL,
	to_key      TEXT        NOT NULL,
	type        TEXT        NOT NULL,
	props       JSONB       NOT NULL DEFAULT '{}',
	scope       TEXT        NOT NULL DEFAULT '',
	source      JSONB       NOT NULL DEFAULT '{}',
	client_id   TEXT        NOT NULL DEFAULT '',
	server_time TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (org_id, from_type, from_key, type, to_type, to_key)
);

CREATE INDEX idx_graph_edges_cursor ON graph_edges (org_id, server_time);
CREATE INDEX idx_graph_edges_repo   ON graph_edges (org_id, repo);
CREATE INDEX idx_graph_edges_to     ON graph_edges (org_id, to_type, to_key);

CREATE TABLE graph_node_history (
	id         BIGSERIAL   PRIMARY KEY,
	org_id     UUID        NOT NULL,
	type       TEXT        NOT NULL,
	key        TEXT        NOT NULL,
	props      JSONB       NOT NULL DEFAULT '{}',
	hash       TEXT        NOT NULL DEFAULT '',
	client_id  TEXT        NOT NULL DEFAULT '',
	changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_graph_node_history ON graph_node_history (org_id, type, key, changed_at DESC);

CREATE TABLE graph_edge_history (
	id        BIGSERIAL   PRIMARY KEY,
	org_id    UUID        NOT NULL,
	from_type TEXT        NOT NULL,
	from_key  TEXT        NOT NULL,
	to_type   TEXT        NOT NULL,
	to_key    TEXT        NOT NULL,
	type      TEXT        NOT NULL,
	props     JSONB       NOT NULL DEFAULT '{}',
	client_id TEXT        NOT NULL DEFAULT '',
	changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_graph_edge_history ON graph_edge_history (org_id, from_type, from_key, type, changed_at DESC);
`,
	},
	{
		Version: 7,
		Name:    "tenant isolation — row-level security",
		SQL: `
-- Enable RLS on every per-tenant table. The app sets
--   SET app.org_id = uuid
-- on each connection it hands to a request handler. Policies enforce
-- that reads and writes only see rows where org_id matches.
--
-- IMPORTANT: RLS only applies to non-superuser roles. The app runs
-- as 'hero'. Migrations and admin work that legitimately cross orgs
-- run as 'root', which bypasses RLS by design.
--
-- CockroachDB v26.1 RLS does NOT support subqueries in policy
-- expressions, so we denormalize org_id onto every per-tenant table
-- rather than reach through a foreign key. Backfill from the FK at
-- migration time.

-- Denormalize org_id onto tables that previously only had repo_id.
ALTER TABLE specs       ADD COLUMN IF NOT EXISTS org_id UUID;
ALTER TABLE knowledge   ADD COLUMN IF NOT EXISTS org_id UUID;
ALTER TABLE conventions ADD COLUMN IF NOT EXISTS org_id UUID;
ALTER TABLE pr_checks   ADD COLUMN IF NOT EXISTS org_id UUID;

UPDATE specs       SET org_id = (SELECT org_id FROM repos WHERE repos.id = specs.repo_id) WHERE org_id IS NULL;
UPDATE knowledge   SET org_id = (SELECT org_id FROM repos WHERE repos.id = knowledge.repo_id) WHERE org_id IS NULL;
UPDATE conventions SET org_id = (SELECT org_id FROM repos WHERE repos.id = conventions.repo_id) WHERE org_id IS NULL;
UPDATE pr_checks   SET org_id = (SELECT org_id FROM github_installations WHERE github_installations.id = pr_checks.installation_id) WHERE org_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_specs_org       ON specs       (org_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_org   ON knowledge   (org_id);
CREATE INDEX IF NOT EXISTS idx_conventions_org ON conventions (org_id);
CREATE INDEX IF NOT EXISTS idx_pr_checks_org   ON pr_checks   (org_id);

-- Enable + force RLS on every per-tenant table.
ALTER TABLE graph_nodes        ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_nodes        FORCE  ROW LEVEL SECURITY;
ALTER TABLE graph_edges        ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_edges        FORCE  ROW LEVEL SECURITY;
ALTER TABLE graph_node_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_node_history FORCE  ROW LEVEL SECURITY;
ALTER TABLE graph_edge_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_edge_history FORCE  ROW LEVEL SECURITY;
ALTER TABLE conventions        ENABLE ROW LEVEL SECURITY;
ALTER TABLE conventions        FORCE  ROW LEVEL SECURITY;
ALTER TABLE knowledge          ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge          FORCE  ROW LEVEL SECURITY;
ALTER TABLE specs              ENABLE ROW LEVEL SECURITY;
ALTER TABLE specs              FORCE  ROW LEVEL SECURITY;
ALTER TABLE activity_events    ENABLE ROW LEVEL SECURITY;
ALTER TABLE activity_events    FORCE  ROW LEVEL SECURITY;
ALTER TABLE pr_checks          ENABLE ROW LEVEL SECURITY;
ALTER TABLE pr_checks          FORCE  ROW LEVEL SECURITY;

-- One simple policy per table — direct org_id comparison, no subqueries.
CREATE POLICY org_isolation ON graph_nodes
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON graph_edges
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON graph_node_history
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON graph_edge_history
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON activity_events
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON specs
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON knowledge
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON conventions
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

CREATE POLICY org_isolation ON pr_checks
  USING      (org_id = current_setting('app.org_id', true)::uuid)
  WITH CHECK (org_id = current_setting('app.org_id', true)::uuid);

-- Note: orgs, users, repos, github_installations, refresh_tokens,
-- api_keys, stack_profiles, global_patterns, global_conventions,
-- intelligence_opt_in are not RLS-enabled. Cross-org by design
-- (intelligence) or auth-context-free (login flows). Access control
-- for these stays in application code.
`,
	},
	{
		Version: 8,
		Name:    "subproject column on specs",
		SQL: `
-- Monorepo subproject scope: declared in .hero/subprojects.json on the
-- client side, stamped into each spec's frontmatter as `+"`subproject:`"+`,
-- and synced through the upsert path. Empty string means "root scope".
-- Indexed only on non-empty values so filtering is cheap and the index
-- stays small for repos that don't use subprojects.

ALTER TABLE specs ADD COLUMN IF NOT EXISTS subproject TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_specs_subproject
  ON specs (org_id, subproject) WHERE subproject != '';
`,
	},
}
