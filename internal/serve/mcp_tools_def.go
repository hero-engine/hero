package serve

// toolDefinitions returns the canonical list of Hero MCP tools advertised
// to clients via tools/list. Adding or modifying a tool: update the entry
// here, register a handler in mcp_dispatch.go, and implement the handler
// in the appropriate mcp_tools_*.go file.
func (s *MCPServer) toolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "hero_context",
			Description: "Get conventions, rules, past work, decisions, and known risks for the given file paths. Returns a structured context block that helps AI agents understand project constraints. Pass `compact: true` to receive a [hero envelope] summary plus ref_id only — call hero_expand to retrieve the full bundle.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"files":   {Type: "string", Description: "Comma-separated file paths to get context for (e.g. 'src/auth.go,src/middleware.go')"},
					"compact": {Type: "boolean", Description: "If true, return a compact summary + ref_id instead of the full bundle. Default: false (full bundle)."},
				},
				Required: []string{"files"},
			},
		},
		{
			Name:        "hero_search",
			Description: "Full-text search across all indexed specs and knowledge entries. Returns matching specs with type, status, and snippet. Pass `compact: true` for a hit-count summary plus ref_id; call hero_expand to retrieve the full result list.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"query":      {Type: "string", Description: "Search query text"},
					"type":       {Type: "string", Description: "Filter by spec type: feature, bug, convention, decision, initiative, rule, external, context, note"},
					"status":     {Type: "string", Description: "Filter by status: planning, in-review, delivering, completed, active, accepted, draft"},
					"subproject": {Type: "string", Description: "Filter to a specific subproject scope (e.g. \"engines/mlx\"). When the user is working in a satellite or you have been told the active subproject, pass it here so results are scope-relevant. Pass \"all\" to disable when the user is asking a workspace-wide question."},
					"compact":    {Type: "boolean", Description: "If true, return summary + ref_id only. Default: false."},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "hero_snapshot",
			Description: "Project-shape rollup — surfaces (with stages), active initiatives, recently-completed work, what's next, and open risks across the whole project. Reads .hero/SNAPSHOT.md when fresh; otherwise builds from the live graph. Pass `at: <YYYY-MM-DD>` to read an archived snapshot, `history: true` to list archived snapshots, or `archive: true` to write a manual archive. Snapshot archives are isolated from default discovery — they only surface here.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"compact": {Type: "boolean", Description: "Return summary + ref_id (call hero_expand for the full body). Default: false."},
					"section": {Type: "string", Description: "Limit to one section: surfaces, initiatives, recent, next, risks, health. Default: all."},
					"surface": {Type: "string", Description: "Restrict the rollup to one surface id."},
					"at":      {Type: "string", Description: "Read a specific archived snapshot. Accepts YYYY-MM-DD or YYYY-MM-DD--<label>. Default: latest (live snapshot)."},
					"history": {Type: "boolean", Description: "Return the enumerated archive list ({date, trigger, label, git_commit, path}). Mutually exclusive with at."},
					"archive": {Type: "boolean", Description: "Write a manual archive of the current snapshot and return the new archive record."},
					"label":   {Type: "string", Description: "Optional label slug attached to a manual archive (only used when archive: true)."},
				},
			},
		},
		{
			Name:        "hero_status",
			Description: "Get the current workspace status showing all specs grouped by status (delivering, in-review, planning, completed) and knowledge base entries.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "hero_check",
			Description: "Run a workspace health check. Reports stale specs, unclaimed work, and corpus statistics.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"stale_days": {Type: "string", Description: "Number of days before a spec is considered stale (default: 14)"},
				},
			},
		},
		{
			Name:        "hero_nudge",
			Description: "Get nudge information for files being worked on. Returns relevant conventions, past work, and in-flight specs for the given file paths.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"files": {Type: "string", Description: "Comma-separated file paths to check for nudges"},
				},
				Required: []string{"files"},
			},
		},
		{
			Name:        "hero_list",
			Description: "List specs with rich filters. Composable across type, status, horizon, tag, ready-vs-blocked, pinned, mine, stale. Use --format kickoff to receive paste-ready session-opener prompts. For the curated ready-now ranked queue, prefer hero_queue.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"type":       {Type: "string", Description: "Filter by spec type (comma-separated for multiple): feature, bug, convention, decision, initiative, rule, external, context, note"},
					"status":     {Type: "string", Description: "Filter by status (comma-separated): planning, in-review, delivering, completed, active, accepted, draft"},
					"horizon":    {Type: "string", Description: "Filter by horizon (comma-separated): now, next, someday, parking"},
					"tag":        {Type: "string", Description: "Require tag (comma-separated for multiple — all must be present)"},
					"ready":      {Type: "string", Description: "Set to 'true' to filter to ready-to-pick-up specs (open + deps satisfied). Mutually exclusive with blocked."},
					"blocked":    {Type: "string", Description: "Set to 'true' to filter to specs with at least one unmet hard dependency."},
					"pinned":     {Type: "string", Description: "Set to 'true' to filter to specs with `pinned: true` in frontmatter."},
					"mine":       {Type: "string", Description: "Filter to specs claimed by this user (matches claimed_by frontmatter)."},
					"stale":      {Type: "string", Description: "Only specs untouched for at least N days (numeric)."},
					"subproject": {Type: "string", Description: "Filter to a specific subproject scope (e.g. \"engines/mlx\"). When the user is working in a satellite or you have been told the active subproject, pass it here so results are scope-relevant. Pass \"all\" to disable when the user is asking a workspace-wide question."},
					"sort":       {Type: "string", Description: "Sort key: recency (default), status, alpha, priority"},
					"limit":      {Type: "string", Description: "Cap result count (numeric, 0 = unlimited)"},
					"format":     {Type: "string", Description: "Output format: text (default), kickoff, json"},
				},
			},
		},
		{
			Name:        "hero_queue",
			Description: "Ranked list of ready-to-work specs with paste-ready kickoff prompts. Curated front door over the spec corpus — equivalent to hero_list with ready=true, sort=priority, format=kickoff. Call this when the user asks 'what should I work on?' or 'give me prompts for new sessions'.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"limit":      {Type: "string", Description: "Cap result count (numeric, 0 = unlimited)"},
					"horizon":    {Type: "string", Description: "Filter by horizon (comma-separated): now, next, someday, parking"},
					"subproject": {Type: "string", Description: "Filter to a specific subproject scope (e.g. \"engines/mlx\"). When the user is working in a satellite or you have been told the active subproject, pass it here so results are scope-relevant. Pass \"all\" to disable when the user is asking a workspace-wide question."},
					"format":     {Type: "string", Description: "Output format: kickoff (default), text, json"},
				},
			},
		},
		{
			Name:        "hero_kickoff",
			Description: "Return the `## Kickoff` section for a spec — the paste-ready cold-start prompt the user can drop into a fresh session to pick this work back up. Use when the user asks for a session-opener for a specific spec.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug": {Type: "string", Description: "The spec slug (e.g. 'kickoff-prompts-queue')."},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_knowledge",
			Description: "List all knowledge base entries (conventions, decisions, rules, external docs, context, notes) with optional type filtering.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"type": {Type: "string", Description: "Filter by knowledge type: convention, decision, rule, external, context, note"},
				},
			},
		},
		{
			Name:        "hero_read_spec",
			Description: "Read the full content of a spec by its slug. Returns the complete markdown content of the spec file. Pass `compact: true` for a title + status + 1-line essence summary plus a stable ref_id of the form `spec:<slug>:full` — call hero_expand later to retrieve the full body.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug":    {Type: "string", Description: "The slug of the spec to read (e.g. 'auth-login', 'naming-convention')"},
					"compact": {Type: "boolean", Description: "If true, return summary + ref_id only. Default: false (full body)."},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_ask",
			Description: "Answer a question extractively from the knowledge base using BM25/TF-IDF scoring. No LLM — returns best-matching passages from specs and knowledge entries with source citations.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"question": {Type: "string", Description: "The question to answer"},
					"type":     {Type: "string", Description: "Restrict to a knowledge type: convention, decision, context, rule"},
					"limit":    {Type: "string", Description: "Max entries to search (default: 20)"},
				},
				Required: []string{"question"},
			},
		},
		{
			Name:        "hero_anchor",
			Description: "Re-anchor on project first principles. Call this BEFORE proposing architectural alternatives, when hitting a dead end, or when brainstorming solutions. Returns project mission and all active tripwires (forbidden options). Prevents drift from first principles.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"context": {Type: "string", Description: "Optional: what you're deciding or considering. Used to highlight relevant tripwires."},
				},
			},
		},
		{
			Name:        "hero_pulse",
			Description: "Get a sprint pulse report — done this sprint, in-flight, at-risk, and knowledge updates. Derived from spec status and git history, no LLM.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"since":  {Type: "string", Description: "Start date YYYY-MM-DD (default: 14 days ago)"},
					"format": {Type: "string", Description: "Output format: text (default), json, markdown"},
				},
			},
		},
		{
			Name:        "hero_skill_run",
			Description: "Preview or execute a saved skill workflow. Returns the skill steps for the agent to follow. Use hero_search to find available skill slugs.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug":   {Type: "string", Description: "The skill slug to run"},
					"params": {Type: "string", Description: "Comma-separated key=value pairs for skill parameters (e.g. 'name=foo,env=prod')"},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_claim",
			Description: "Claim, release, or complete a spec. Records the action in events.log and updates spec frontmatter.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug":   {Type: "string", Description: "The spec slug to act on"},
					"action": {Type: "string", Description: "Action to perform: claim (default), release, complete"},
					"agent":  {Type: "string", Description: "Agent identity (default: from HERO_AGENT env or config)"},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_velocity",
			Description: "Get per-agent delivery velocity metrics — specs completed, average days per spec, fastest and slowest slugs.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"since": {Type: "string", Description: "Start date YYYY-MM-DD to limit the window (default: all time)"},
					"agent": {Type: "string", Description: "Filter to a specific agent"},
				},
			},
		},
		{
			Name:        "hero_test_generate",
			Description: "Generate test files from spec acceptance criteria. Uses the configured framework (default: Playwright) and mode (agent/assisted/autonomous).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug": {Type: "string", Description: "Spec slug to generate tests for"},
					"mode": {Type: "string", Description: "Override generation mode: agent, assisted, autonomous"},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_demo_record",
			Description: "Record a video demo for a delivered spec by running its tests with video capture enabled.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug": {Type: "string", Description: "Spec slug to record demo for"},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_code",
			Description: "Search code intelligence: find symbols (functions, types, classes, interfaces), browse packages, view dependency graph, identify hot files, list detected environment variables, look up known error patterns, and get unenriched symbols for deep scanning. Requires 'hero scan' to have been run first.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"action":  {Type: "string", Description: "Action to perform: 'search' (find symbols/packages), 'package' (get full package info), 'deps' (dependency graph), 'hot' (hot files), 'config' (environment variables), 'endpoints' (API endpoints), 'errors' (known error patterns), 'unenriched' (get symbols needing AI descriptions), 'overview' (all packages summary). Default: 'search'"},
					"query":   {Type: "string", Description: "Search query for symbols/packages. Supports glob patterns like 'Handle*', '*Service'. Required for 'search' action."},
					"kind":    {Type: "string", Description: "Filter by symbol kind: func, method, interface, struct, class, type, const, enum, trait"},
					"package": {Type: "string", Description: "Filter to a specific package path (e.g. 'internal/serve'). For 'package' action, this is required."},
					"limit":   {Type: "string", Description: "Max symbols to return for 'unenriched' action (default: 20)"},
				},
			},
		},
		{
			Name:        "hero_error_pattern",
			Description: "Save a new error pattern to the knowledge base. Use after diagnosing a bug to capture the pattern for future reference.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"id":         {Type: "string", Description: "Slug-like identifier for the pattern (e.g. 'nil-pointer-config-load')"},
					"pattern":    {Type: "string", Description: "Regex to match error text"},
					"stack":      {Type: "string", Description: "Comma-separated technology stack (e.g. 'go,postgres')"},
					"severity":   {Type: "string", Description: "Severity level: common, rare, critical"},
					"files":      {Type: "string", Description: "Comma-separated relevant file paths"},
					"symptom":    {Type: "string", Description: "Description of the error symptom"},
					"root_cause": {Type: "string", Description: "Description of the root cause"},
					"fix":        {Type: "string", Description: "Description of the fix"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "hero_enrich",
			Description: "Write LLM-generated descriptions for code symbols. Used during deep code scanning. Call hero_code with action 'unenriched' first to get symbols needing descriptions, then call this tool with the descriptions.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"enrichments": {Type: "string", Description: "JSON array of objects with fields: package (string), symbol (string), description (string). Example: [{\"package\":\"internal/serve\",\"symbol\":\"MCPServer\",\"description\":\"Main MCP protocol server handling JSON-RPC requests\"}]"},
				},
				Required: []string{"enrichments"},
			},
		},
		{
			Name:        "hero_synthesize",
			Description: "Assemble the material for a feature 'explainer' knowledge entry (how a feature works, as it exists now) from a cluster of spec slugs. Returns the synthesis instructions, the target path, the provenance frontmatter, and the assembled context (source specs, git activity across the delivery window, and referenced decisions). YOU then write the explainer markdown to the target path following the instructions, and run hero index. Part of feature-knowledge-synthesis.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slugs": {Type: "string", Description: "Comma-separated spec slugs that make up the feature (e.g. 'cold-start-trust-hardening' or 'feat-a,feat-b,feat-c'). An initiative slug among them sets the explainer's title and out-path."},
				},
				Required: []string{"slugs"},
			},
		},
		{
			Name:        "hero_score",
			Description: "Score a spec's quality and readiness for delivery. Returns a score (0-100), grade (A-F), per-dimension breakdown, warnings, and actionable suggestions. Use before delivering a spec to check readiness.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug": {Type: "string", Description: "Spec slug or relative path to spec.md file"},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_diagnose",
			Description: "Prepare a bug spec for diagnosis. In single mode (with slug), returns the spec content and investigation instructions for the debug-investigator agent. In batch mode (batch=true), returns all undiagnosed bugs in the pipeline ready for diagnosis.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug":  {Type: "string", Description: "Spec slug of the bug to diagnose (omit for batch mode)"},
					"batch": {Type: "boolean", Description: "List all undiagnosed bugs for batch diagnosis"},
				},
			},
		},
		{
			Name:        "hero_verify",
			Description: "Verify a spec's implementation against its acceptance criteria. Returns the acceptance criteria as a checklist, expected file changes, test strategy, and a structured verification prompt. Use after manual delivery or to review agent-delivered work.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug": {Type: "string", Description: "Spec slug to verify"},
				},
				Required: []string{"slug"},
			},
		},
		{
			Name:        "hero_conflicts",
			Description: "Find in-flight specs that touch overlapping files with the given spec. Detects potential merge conflicts and coordination needs before they become problems. Omit slug to find all conflicting pairs.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug": {Type: "string", Description: "Spec slug to check for conflicts (omit to find all conflicting pairs)"},
				},
			},
		},
		{
			Name:        "hero_sequence",
			Description: "Suggest an optimal delivery order for in-flight specs based on dependency relationships (depends-on) and file overlap. Specs with no dependencies and fewer conflicts are recommended first to minimize merge risk.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]PropSchema{},
			},
		},
		{
			Name:        "hero_warnings",
			Description: "Surface proactive warnings about the current spec or workspace. Detects stale specs, file conflict risks, unclaimed work, missing conventions, and specs with no tests defined. Use before starting work to catch issues early.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug": {Type: "string", Description: "Spec slug to check (omit for workspace-wide warnings)"},
				},
			},
		},
		{
			Name:        "hero_insights",
			Description: "Get cross-project intelligence and recommendations. Analyzes workspace patterns and suggests conventions, practices, and optimizations commonly adopted by similar projects. Powered by anonymized aggregate data from the Hero network.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]PropSchema{},
			},
		},
		{
			Name:        "hero_contract",
			Description: "Show contract status or link criteria to tests for completed specs. Actions: 'status' shows which criteria have test links, 'link' adds a verified_by annotation, 'check' runs linked tests and reports regressions.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"action":          {Type: "string", Description: "Action: status, link, or check"},
					"slug":            {Type: "string", Description: "Spec slug"},
					"criterion_index": {Type: "integer", Description: "1-based criterion index (for link action)"},
					"test_ref":        {Type: "string", Description: "file::testname (for link action)"},
				},
				Required: []string{"action"},
			},
		},
		{
			Name:        "hero_plan",
			Description: "Persist an execution plan alongside a spec. Captures agent-generated plans so they survive the session, are visible to the team, and can be checked against the implementation. Overwrites any existing plan for the spec.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug":    {Type: "string", Description: "Spec slug to attach the plan to"},
					"content": {Type: "string", Description: "Plan content (markdown)"},
				},
				Required: []string{"slug", "content"},
			},
		},
		{
			Name:        "hero_impact",
			Description: "Analyze the blast radius of changing a file: which specs, conventions, and decisions are affected. Returns specs that list the file in their Changes section, conventions whose scope covers it, and decisions that mention it.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"file_paths": {Type: "array", Description: "File paths to analyze (relative to project root)", Items: &PropSchema{Type: "string"}},
				},
				Required: []string{"file_paths"},
			},
		},
		{
			Name:        "hero_recap",
			Description: "Generate a spec-grouped activity summary for a time window. Groups recent git commits by the spec they relate to, surfaces status transitions, and lists new/modified knowledge entries. Pass `compact: true` for a counts-only summary plus ref_id; call hero_expand for the full digest.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"since":      {Type: "string", Description: "Duration (24h, 2d, 1w) or ISO date (YYYY-MM-DD). Default: 24h"},
					"subproject": {Type: "string", Description: "Filter to a specific subproject scope (e.g. \"engines/mlx\"). When the user is working in a satellite or you have been told the active subproject, pass it here so results are scope-relevant. Pass \"all\" to disable when the user is asking a workspace-wide question."},
					"compact":    {Type: "boolean", Description: "If true, return summary + ref_id only. Default: false."},
				},
			},
		},
		{
			Name:        "hero_drift",
			Description: "Report drift between spec and code for one or more specs. Detects missing files, renamed files, unaddressed acceptance criteria, and boundary violations. All signals are heuristic and local — no LLM calls.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug":       {Type: "string", Description: "Spec slug to analyze"},
					"in_flight":  {Type: "boolean", Description: "Analyze all delivering specs"},
					"initiative": {Type: "string", Description: "Analyze all child specs of an initiative"},
					"since":      {Type: "string", Description: "Only count drift since this git ref"},
				},
			},
		},
		{
			Name:        "hero_ci",
			Description: "Query CI pipeline status for the current or specified branch. Shows pass/fail, failed step, and run URL.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"branch": {Type: "string", Description: "Branch to check (defaults to current git branch)"},
				},
			},
		},
		{
			Name:        "hero_feed",
			Description: "Query the cross-session activity feed. Returns recent significant events from all agents working in this repo. Pass `compact: true` for an event-count summary plus ref_id; call hero_expand for the full event list.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"since":      {Type: "string", Description: "Duration (1h, 30m) or RFC 3339 timestamp"},
					"type":       {Type: "string", Description: "Filter by event type"},
					"slug":       {Type: "string", Description: "Filter by spec slug"},
					"subproject": {Type: "string", Description: "Filter to a specific subproject scope (e.g. \"engines/mlx\"). When the user is working in a satellite or you have been told the active subproject, pass it here so results are scope-relevant. Pass \"all\" to disable when the user is asking a workspace-wide question."},
					"limit":      {Type: "integer", Description: "Max events to return (default 20)"},
					"compact":    {Type: "boolean", Description: "If true, return summary + ref_id only. Default: false."},
				},
			},
		},
		{
			Name:        "hero_event",
			Description: "Log a significant event to the cross-session activity feed. Other agents will see this.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"type":    {Type: "string", Description: "Event type: spec_created, spec_updated, files_modified, decision_made, blocker_hit, delivery_complete"},
					"message": {Type: "string", Description: "Human-readable description of what happened"},
					"slug":    {Type: "string", Description: "Spec slug (optional)"},
				},
				Required: []string{"type", "message"},
			},
		},
		{
			Name:        "hero_active",
			Description: "Show or manage active spec sessions. Active specs get priority in context injection so post-compaction sessions pick up the right spec.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"action":     {Type: "string", Description: "Action: list (default), register, unregister, prune"},
					"session_id": {Type: "string", Description: "Session ID (required for register/unregister)"},
					"spec":       {Type: "string", Description: "Spec slug (required for register)"},
					"command":    {Type: "string", Description: "Command that started the session (for register, e.g. /deliver)"},
				},
			},
		},
		{
			Name:        "hero_coverage",
			Description: "Report which acceptance criteria have test coverage and which are untested. All analysis is local and heuristic — no LLM calls, no test execution.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"slug":     {Type: "string", Description: "Spec slug to analyze"},
					"all":      {Type: "boolean", Description: "Analyze all specs with acceptance criteria"},
					"test_dir": {Type: "string", Description: "Override test file discovery root"},
				},
			},
		},
		{
			Name:        "hero_why",
			Description: "Trace where something came from. Resolves the target (spec slug, AC id, file path, or commit SHA) to a graph node and walks origin edges in reverse — multi-hop, depth-bounded — returning the chain of decisions, specs, and commits that led to its existence. The v2 traversal showcase: cross-subgraph queries that no grep can answer. Pass `compact: true` for a path-summary plus ref_id; call hero_expand for the full trace.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"target":  {Type: "string", Description: "Node key to trace (e.g. spec slug, '<slug>:AC-N', file path, commit SHA)"},
					"depth":   {Type: "integer", Description: "Max recursion depth (default 4)"},
					"compact": {Type: "boolean", Description: "If true, return summary + ref_id only. Default: false."},
				},
				Required: []string{"target"},
			},
		},
		{
			Name:        "hero_blocked",
			Description: "List open features that are blocked. Walks Feature→depends_on→Feature edges plus failing/regressed Criterion nodes joined by parent spec. Combines the v2 dependency-tree query with AC-graph status so the model sees both kinds of blockers in one place. Pass `compact: true` for counts plus ref_id; call hero_expand for the full list.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"compact": {Type: "boolean", Description: "If true, return summary + ref_id only. Default: false."},
				},
			},
		},
		{
			Name: "hero_expand",
			Description: "Rehydrate a previously-returned compact ref into its full content. Read-side tools called with `compact: true` return a [hero envelope] block containing a `ref_id`. Pass that ref_id (or an array of ref_ids in `ref_ids`) to expand back to verbatim content. Stable kinds (spec, convention, decision, rule) resolve identically across sessions; query kinds (search, context, recap, why, blocked, feed) are session-scoped. Unknown or expired refs return a structured error with a rehydrate hint naming the producing tool.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]PropSchema{
					"ref_id":  {Type: "string", Description: "A single ref ID of the form <kind>:<slug>:<scope> (e.g. 'spec:hero-ask:full')."},
					"ref_ids": {Type: "string", Description: "Multiple ref IDs as a JSON array (e.g. '[\"spec:hero-ask:full\",\"convention:api-handlers:full\"]') or comma-separated."},
				},
			},
		},
	}
}