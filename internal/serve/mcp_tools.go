package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/acceptance"
	"github.com/hero-engine/hero/internal/active"
	"github.com/hero-engine/hero/internal/codescan"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/contract"
	"github.com/hero-engine/hero/internal/coverage"
	"github.com/hero-engine/hero/internal/demos"
	"github.com/hero-engine/hero/internal/drift"
	"github.com/hero-engine/hero/internal/environment"
	"github.com/hero-engine/hero/internal/errpattern"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/herotest"
	"github.com/hero-engine/hero/internal/impact"
	"github.com/hero-engine/hero/internal/mission"
	"github.com/hero-engine/hero/internal/index"
	"github.com/hero-engine/hero/internal/pulse"
	"github.com/hero-engine/hero/internal/recap"
	"github.com/hero-engine/hero/internal/refs"
	"github.com/hero-engine/hero/internal/retrieval"
	"github.com/hero-engine/hero/internal/score"
	"github.com/hero-engine/hero/internal/search"
	"github.com/hero-engine/hero/internal/sizing"
	"github.com/hero-engine/hero/internal/skills"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/hero-engine/hero/internal/synthesize"
	"github.com/hero-engine/hero/internal/tracking"
	"github.com/hero-engine/hero/internal/traversal"
	"github.com/hero-engine/hero/internal/vocabulary"
)

// Tool implementations and helpers extracted from mcp.go.
//
// Categories follow the dispatch table in mcp_dispatch.go:
//   - read   (no state change; returns data)
//   - mutate (writes state)
//   - analyze (computes a derived view)
// Helpers (formatters, arg parsers, enrichers) sit between groups.
//
// Adding a new tool: implement here in the appropriate group, then
// register it in mcp_dispatch.go::toolHandlers and document it in
// mcp_tools_def.go::toolDefinitions.

func (s *MCPServer) toolContext(args map[string]interface{}) (string, error) {
	filesStr, _ := args["files"].(string)
	if filesStr == "" {
		return "", fmt.Errorf("files parameter is required")
	}

	filePaths := strings.Split(filesStr, ",")
	for i := range filePaths {
		filePaths[i] = strings.TrimSpace(filePaths[i])
	}

	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	ctx, err := idx.BuildContext(filePaths)
	if err != nil {
		return "", fmt.Errorf("building context: %w", err)
	}

	// Enrich with active spec context — full content of any spec currently
	// being worked on, so post-compaction sessions pick up the right spec.
	activeBlock := s.activeSpecContext()

	// Enrich with code structure for relevant packages
	s.enrichCodeStructure(ctx, filePaths)

	// Enrich with error patterns
	errorPatternsBlock := enrichErrorPatterns(s.projectRoot, filePaths)

	if ctx.IsEmpty() && errorPatternsBlock == "" && activeBlock == "" {
		return "No relevant context found in the spec corpus for these files.", nil
	}

	result := ""
	if activeBlock != "" {
		result += activeBlock + "\n\n"
	}
	cfg, _ := config.Load(s.projectRoot)
	vocab := activeVocab(&cfg)
	result += formatContextBlockWithVocab(ctx, vocab)
	if errorPatternsBlock != "" {
		result += "\n\n" + errorPatternsBlock
	}

	if argCompact(args) {
		summary := fmt.Sprintf("Context for %d file(s): %s. %s",
			len(filePaths), summariseFileList(filePaths), summariseContext(ctx, errorPatternsBlock != "", activeBlock != ""))
		sourceArgs := map[string]any{"files": filesStr}
		envText, err := s.registerRef(refs.KindContext, argHash(sourceArgs), "bundle",
			sourceArgs, result, fingerprintArgs(sourceArgs), summary)
		if err == nil {
			return envText, nil
		}
	}

	return result, nil
}

func summariseFileList(paths []string) string {
	if len(paths) == 0 {
		return "(none)"
	}
	if len(paths) <= 3 {
		return strings.Join(paths, ", ")
	}
	return fmt.Sprintf("%s, %s, +%d more", paths[0], paths[1], len(paths)-2)
}

func summariseContext(ctx *index.ContextBlock, hasErrPatterns, hasActive bool) string {
	parts := []string{}
	if ctx != nil {
		if n := len(ctx.Conventions); n > 0 {
			parts = append(parts, fmt.Sprintf("%d conventions", n))
		}
		if n := len(ctx.Decisions); n > 0 {
			parts = append(parts, fmt.Sprintf("%d decisions", n))
		}
		if n := len(ctx.Rules); n > 0 {
			parts = append(parts, fmt.Sprintf("%d rules", n))
		}
		if n := len(ctx.PastWork); n > 0 {
			parts = append(parts, fmt.Sprintf("%d past specs", n))
		}
		if n := len(ctx.InFlight); n > 0 {
			parts = append(parts, fmt.Sprintf("%d in-flight", n))
		}
	}
	if hasErrPatterns {
		parts = append(parts, "error patterns")
	}
	if hasActive {
		parts = append(parts, "active spec(s)")
	}
	if len(parts) == 0 {
		return "no entries"
	}
	return strings.Join(parts, ", ")
}

// activeSpecContext returns a formatted block with the full content of any
// specs currently being actively worked on.
func (s *MCPServer) activeSpecContext() string {
	slugs := active.ActiveSpecs(s.heroDir)
	if len(slugs) == 0 {
		return ""
	}

	allSpecs, err := spec.Discover(s.heroDir)
	if err != nil {
		return ""
	}

	specBySlug := make(map[string]*spec.Spec, len(allSpecs))
	for _, sp := range allSpecs {
		specBySlug[sp.Slug] = sp
	}

	var parts []string
	for _, slug := range slugs {
		sp, ok := specBySlug[slug]
		if !ok {
			continue
		}
		content, err := os.ReadFile(sp.Path)
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("## [ACTIVE SPEC] %s\n\nPath: %s\n\n%s", sp.Title, sp.Path, string(content)))
	}

	if len(parts) == 0 {
		return ""
	}

	return "# Active Specs (currently being worked on)\n\n" + strings.Join(parts, "\n\n---\n\n")
}

// ensureFreshIndex performs a cheap mtime-vs-stamp diff against disk
// and re-indexes any drifted specs before the caller queries the
// index. Steady-state cost when nothing changed: stat each spec.md
// (microseconds total). Errors are logged but never fatal — stale
// query results beat aborting the tool call.
//
// Spec: index-staleness-auto-refresh.
func (s *MCPServer) ensureFreshIndex() {
	if _, err := index.RefreshIfStale(s.heroDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: index refresh failed: %v\n", err)
	}
}

func (s *MCPServer) toolSearch(args map[string]interface{}) (string, error) {
	s.ensureFreshIndex()
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query parameter is required")
	}

	specType, _ := args["type"].(string)
	status, _ := args["status"].(string)
	subproject, _ := args["subproject"].(string)

	q := retrieval.Query{Text: query}
	if specType != "" || status != "" || (subproject != "" && subproject != "all") {
		q.Filters = make(map[string]string)
		if specType != "" {
			q.Types = []string{specType}
			q.Filters["type"] = specType
		}
		if status != "" {
			q.Filters["status"] = status
		}
		if subproject != "" && subproject != "all" {
			q.Filters["subproject"] = subproject
		}
	}

	ret, err := retrieval.New(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening retrieval layer: %w", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(q)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No results found.", nil
	}

	output := formatRetrievalResults(results)

	// Prepend any tripwire matches for the query
	if tw := s.tripwireWarning(query); tw != "" {
		output = tw + "\n" + output
	}

	if argCompact(args) {
		summary := fmt.Sprintf("%d hit(s) for query %q", len(results), query)
		if specType != "" {
			summary += fmt.Sprintf(", type=%s", specType)
		}
		if status != "" {
			summary += fmt.Sprintf(", status=%s", status)
		}
		sourceArgs := map[string]any{"query": query, "type": specType, "status": status}
		envText, err := s.registerRef(refs.KindSearch, argHash(sourceArgs), "results",
			sourceArgs, output, fingerprintArgs(sourceArgs), summary)
		if err == nil {
			return envText, nil
		}
	}

	return output, nil
}

func (s *MCPServer) toolStatus(args map[string]interface{}) (string, error) {
	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	return formatStatusOutput(specs), nil
}

func (s *MCPServer) toolCheck(args map[string]interface{}) (string, error) {
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	// Rebuild to get fresh stats
	stats, err := idx.GetStats()
	if err != nil {
		return "", err
	}

	staleDays := 14
	if cfg.Team != nil && cfg.Team.StaleDays > 0 {
		staleDays = cfg.Team.StaleDays
	}

	stale, _ := idx.CheckStale(staleDays)
	unclaimed, _ := idx.CheckUnclaimed()

	return formatCheckOutput(stats, stale, unclaimed, staleDays), nil
}

func (s *MCPServer) toolNudge(args map[string]interface{}) (string, error) {
	filesStr, _ := args["files"].(string)
	if filesStr == "" {
		return "", fmt.Errorf("files parameter is required")
	}

	filePaths := strings.Split(filesStr, ",")
	for i := range filePaths {
		filePaths[i] = strings.TrimSpace(filePaths[i])
	}

	ret, err := retrieval.New(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening retrieval layer: %w", err)
	}
	defer ret.Close()

	result, err := ret.NudgeFiles(filePaths)
	if err != nil {
		return "", err
	}

	if result.IsEmpty() {
		return "No relevant context for these files.", nil
	}

	return formatNudgeOutput(result), nil
}

func (s *MCPServer) toolList(args map[string]interface{}) (string, error) {
	s.ensureFreshIndex()
	sel, err := selectorFromMCPArgs(args)
	if err != nil {
		return "", err
	}

	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	out := sel.Apply(specs)
	if len(out) == 0 {
		return "No specs found matching the given filters.", nil
	}

	format, _ := args["format"].(string)
	return renderMCPSpecs(out, format)
}

func (s *MCPServer) toolQueue(args map[string]interface{}) (string, error) {
	s.ensureFreshIndex()
	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	horizons := splitMCPArg(args["horizon"])
	hzs := make([]spec.Horizon, 0, len(horizons))
	for _, h := range horizons {
		hzs = append(hzs, spec.Horizon(h))
	}

	limit, err := mcpIntArg(args, "limit")
	if err != nil {
		return "", err
	}

	sel := spec.Selector{
		Filter: spec.Filter{
			Horizons:             hzs,
			Ready:                true,
			ExcludeClosedDefault: true,
			Subproject:           stringOr(args["subproject"], ""),
		},
		Sort:  spec.SortPriority,
		Limit: limit,
	}

	out := sel.Apply(specs)
	if len(out) == 0 {
		return "Queue is empty — every open spec is either blocked or has no `## Kickoff` section to surface. Run hero_list with blocked=true to see what's waiting on dependencies.", nil
	}

	format, _ := args["format"].(string)
	if format == "" {
		format = "kickoff"
	}
	return renderMCPSpecs(out, format)
}

func (s *MCPServer) toolKickoff(args map[string]interface{}) (string, error) {
	s.ensureFreshIndex()
	slug, _ := args["slug"].(string)
	if strings.TrimSpace(slug) == "" {
		return "", fmt.Errorf("slug parameter is required")
	}

	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	// Ambient size-drift hint (prepended above the kickoff body when
	// non-quiet/non-zero). Passing the slug as ActiveSpec naturally
	// lights up rule 1 of the noise filter when the user is kicking
	// off a spec that has drift on it. See spec
	// roadmap-review-ambient-surfacing.
	cfg, _ := config.Load(s.projectRoot)
	ambientOpts := sizing.AmbientDriftOpts{ActiveSpec: slug}
	if cfg.Roadmap != nil {
		ambientOpts.RecencyDays = cfg.Roadmap.AmbientRecencyDaysOrDefault()
		ambientOpts.StopNaggingHours = cfg.Roadmap.StopNaggingHoursOrDefault()
	}
	ambient := sizing.AmbientDrift(s.heroDir, s.projectRoot, ambientOpts)
	driftPrefix := ""
	if !ambient.Quiet && ambient.Count > 0 {
		driftPrefix = ambient.Hint + "\n\n"
	}

	for _, sp := range specs {
		if sp.Slug != slug {
			continue
		}
		body := strings.TrimSpace(sp.Kickoff())
		if body == "" {
			return driftPrefix + fmt.Sprintf("Spec %q exists but has no `## Kickoff` section. Run /design or /deliver to author one, or hand-edit %s.", slug, sp.Path), nil
		}
		return driftPrefix + fmt.Sprintf("## %s — %s\n_%s · %s · horizon: %s_\n\n%s\n",
			sp.Slug, sp.Title, sp.Type, sp.Status, sp.EffectiveHorizon(), body), nil
	}

	return "", fmt.Errorf("spec %q not found in workspace", slug)
}

// selectorFromMCPArgs builds a Selector from the loose string-typed
// MCP argument map. All filters are optional; defaults match the CLI
// (ExcludeClosedDefault=true, Sort=recency).
func selectorFromMCPArgs(args map[string]interface{}) (spec.Selector, error) {
	types := splitMCPArg(args["type"])
	statuses := splitMCPArg(args["status"])
	horizons := splitMCPArg(args["horizon"])
	tags := splitMCPArg(args["tag"])

	ts := make([]spec.Type, 0, len(types))
	for _, t := range types {
		ts = append(ts, spec.Type(t))
	}
	ss := make([]spec.Status, 0, len(statuses))
	for _, st := range statuses {
		ss = append(ss, spec.Status(st))
	}
	hs := make([]spec.Horizon, 0, len(horizons))
	for _, h := range horizons {
		hs = append(hs, spec.Horizon(h))
	}

	ready := mcpBoolArg(args, "ready")
	blocked := mcpBoolArg(args, "blocked")
	if ready && blocked {
		return spec.Selector{}, fmt.Errorf("ready and blocked are mutually exclusive")
	}

	mine, _ := args["mine"].(string)
	stale, err := mcpIntArg(args, "stale")
	if err != nil {
		return spec.Selector{}, err
	}
	limit, err := mcpIntArg(args, "limit")
	if err != nil {
		return spec.Selector{}, err
	}

	sortKey := spec.Sort(strings.TrimSpace(stringOr(args["sort"], string(spec.SortRecency))))
	switch sortKey {
	case spec.SortRecency, spec.SortStatus, spec.SortAlpha, spec.SortPriority, "":
	default:
		return spec.Selector{}, fmt.Errorf("unknown sort %q", sortKey)
	}

	subproject := stringOr(args["subproject"], "")

	return spec.Selector{
		Filter: spec.Filter{
			Types:                ts,
			Statuses:             ss,
			Horizons:             hs,
			Tags:                 tags,
			Ready:                ready,
			Blocked:              blocked,
			Pinned:               mcpBoolArg(args, "pinned"),
			MineUser:             mine,
			StaleDays:            stale,
			ExcludeClosedDefault: true,
			Subproject:           subproject,
		},
		Sort:  sortKey,
		Limit: limit,
	}, nil
}

func renderMCPSpecs(specs []*spec.Spec, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		var buf strings.Builder
		if err := renderSpecsJSONForMCP(&buf, specs); err != nil {
			return "", err
		}
		return buf.String(), nil
	case "kickoff":
		var buf strings.Builder
		for i, sp := range specs {
			if i > 0 {
				buf.WriteString("\n---\n\n")
			}
			pin := ""
			if sp.Pinned {
				pin = " ★"
			}
			fmt.Fprintf(&buf, "## %s — %s%s\n_%s · %s · horizon: %s_\n\n",
				sp.Slug, sp.Title, pin, sp.Type, sp.Status, sp.EffectiveHorizon())
			body := strings.TrimSpace(sp.Kickoff())
			if body == "" {
				fmt.Fprintf(&buf, "_(no `## Kickoff` section — run /design or /deliver, or hand-edit %s)_\n", sp.Path)
			} else {
				buf.WriteString(body)
				buf.WriteString("\n")
			}
		}
		return buf.String(), nil
	case "text", "":
		var buf strings.Builder
		for _, sp := range specs {
			pin := ""
			if sp.Pinned {
				pin = " ★"
			}
			fmt.Fprintf(&buf, "- %s — %s (%s/%s)%s\n", sp.Slug, sp.Title, sp.Type, sp.Status, pin)
		}
		return buf.String(), nil
	}
	return "", fmt.Errorf("unknown format %q (text, kickoff, json)", format)
}

func renderSpecsJSONForMCP(w *strings.Builder, specs []*spec.Spec) error {
	type row struct {
		Slug    string   `json:"slug"`
		Title   string   `json:"title"`
		Type    string   `json:"type"`
		Status  string   `json:"status"`
		Horizon string   `json:"horizon"`
		Tags    []string `json:"tags,omitempty"`
		Pinned  bool     `json:"pinned,omitempty"`
		Kickoff string   `json:"kickoff,omitempty"`
		Path    string   `json:"path,omitempty"`
	}
	rows := make([]row, len(specs))
	for i, sp := range specs {
		rows[i] = row{
			Slug:    sp.Slug,
			Title:   sp.Title,
			Type:    string(sp.Type),
			Status:  string(sp.Status),
			Horizon: string(sp.EffectiveHorizon()),
			Tags:    sp.Tags,
			Pinned:  sp.Pinned,
			Kickoff: sp.Kickoff(),
			Path:    sp.Path,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func splitMCPArg(v interface{}) []string {
	s, _ := v.(string)
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func mcpBoolArg(args map[string]interface{}, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true", "yes", "on", "1":
			return true
		}
	}
	return false
}

func mcpIntArg(args map[string]interface{}, key string) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, nil
	}
	switch x := v.(type) {
	case float64:
		return int(x), nil
	case int:
		return x, nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("%s must be numeric: %w", key, err)
		}
		return n, nil
	}
	return 0, nil
}

func stringOr(v interface{}, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func (s *MCPServer) toolKnowledge(args map[string]interface{}) (string, error) {
	s.ensureFreshIndex()
	typeFilter, _ := args["type"].(string)

	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	var entries []*spec.Spec
	for _, sp := range specs {
		if !sp.IsKnowledge() {
			continue
		}
		if typeFilter != "" && string(sp.Type) != typeFilter {
			continue
		}
		entries = append(entries, sp)
	}

	if len(entries) == 0 {
		if typeFilter != "" {
			return fmt.Sprintf("No %s entries found in the knowledge base.", typeFilter), nil
		}
		return "No entries in the knowledge base.", nil
	}

	return formatKnowledgeOutput(entries), nil
}

func (s *MCPServer) toolReadSpec(args map[string]interface{}) (string, error) {
	s.ensureFreshIndex()
	slug, _ := args["slug"].(string)
	if slug == "" {
		return "", fmt.Errorf("slug parameter is required")
	}

	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	// Look up by exact slug match (avoid FTS which chokes on hyphens)
	all, err := idx.AllSpecs()
	if err != nil {
		return "", err
	}
	var specPath string
	var specTitle, specStatus, specType string
	for _, r := range all {
		if r.Slug == slug {
			specPath = r.Path
			specTitle = r.Title
			specStatus = string(r.Status)
			specType = string(r.Type)
			break
		}
	}

	// Fallback to filesystem discovery when the index doesn't have the
	// slug — covers freshly-created specs whose index entry hasn't
	// landed yet (ensureFreshIndex can silently fail or race the write).
	if specPath == "" {
		if specs, discErr := spec.Discover(s.heroDir); discErr == nil {
			for _, sp := range specs {
				if sp.Slug == slug {
					specPath = sp.Path
					specTitle = sp.Title
					specStatus = string(sp.Status)
					specType = string(sp.Type)
					break
				}
			}
		}
	}

	if specPath == "" {
		return fmt.Sprintf("Spec %q not found.", slug), nil
	}

	content, err := os.ReadFile(specPath)
	if err != nil {
		return "", fmt.Errorf("reading spec file: %w", err)
	}

	// Prepend the SUPERSEDED banner when applicable — the on-disk file
	// stays clean; the banner is a render-time concern so readers see
	// the redirect without git churn on the source. See spec
	// superseded-specs-soft-archive.
	rendered := string(content)
	if parsed, perr := spec.Parse(rendered, specPath, time.Now()); perr == nil {
		rendered = spec.RenderSpecBody(parsed, rendered)
	}

	if argCompact(args) {
		summary := buildSpecSummary(specTitle, specType, specStatus, rendered)
		fingerprint := fingerprintFile(specPath)
		envText, err := s.registerRef(refs.KindSpec, slug, "full",
			map[string]any{"slug": slug},
			rendered, fingerprint, summary)
		if err != nil {
			// Ref store unavailable — fall back to legacy shape.
			return rendered, nil
		}
		return envText, nil
	}

	return rendered, nil
}

// ---------------------------------------------------------------------------
// New tool implementations (v0.4)
// ---------------------------------------------------------------------------

func (s *MCPServer) toolAsk(args map[string]interface{}) (string, error) {
	question, _ := args["question"].(string)
	if question == "" {
		return "", fmt.Errorf("question parameter is required")
	}
	typeFilter, _ := args["type"].(string)

	limitStr, _ := args["limit"].(string)
	limit := 20
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	q := retrieval.Query{Text: question, Limit: limit}
	if typeFilter != "" {
		q.Types = []string{typeFilter}
		q.Filters = map[string]string{"type": typeFilter}
	}

	ret, err := retrieval.New(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening retrieval layer: %w", err)
	}
	defer ret.Close()

	results, err := ret.Retrieve(q)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No knowledge found for this question.", nil
	}

	queryTokens := search.Tokenize(question)

	var answerSentences []string
	var citations []string
	var topScore float64

	for _, r := range results {
		if r.Path == "" {
			continue
		}
		data, err := os.ReadFile(r.Path)
		if err != nil {
			continue
		}
		passages := search.ExtractPassages(string(data), queryTokens, 3)
		if len(passages) == 0 {
			continue
		}
		sc := search.ScorePassage(passages[0], queryTokens)
		if sc > topScore {
			topScore = sc
		}
		for _, p := range passages {
			if len(answerSentences) < 3 {
				answerSentences = append(answerSentences, p)
			}
		}
		citations = append(citations, fmt.Sprintf("  %s  %s", r.Key, r.Path))
	}

	if len(answerSentences) == 0 {
		return "No knowledge found for this question.", nil
	}

	answer := strings.Join(answerSentences, ". ")
	if !strings.HasSuffix(answer, ".") && !strings.HasSuffix(answer, "!") && !strings.HasSuffix(answer, "?") {
		answer += "."
	}

	confidence := search.Confidence(topScore)
	var sb strings.Builder

	// Prepend any tripwire matches for the question
	if tw := s.tripwireWarning(question); tw != "" {
		sb.WriteString(tw)
		sb.WriteString("\n")
	}

	sb.WriteString(answer)
	sb.WriteString("\n\nConfidence: " + confidence)
	if len(citations) > 0 {
		sb.WriteString("\n\nSources:\n")
		sb.WriteString(strings.Join(citations, "\n"))
	}
	return sb.String(), nil
}

func (s *MCPServer) toolAnchor(args map[string]interface{}) (string, error) {
	ctx, _ := args["context"].(string)

	var sb strings.Builder

	// 1. Load and render project mission
	m, _ := mission.LoadFile(s.heroDir)
	if m != nil && m.MissionStatement != "" {
		sb.WriteString("## Mission\n\n")
		sb.WriteString(m.MissionStatement)
		sb.WriteString("\n\n")
	}

	// 2. Load all active tripwires
	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	tripwires, err := idx.FindAllTripwires(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("loading tripwires: %w", err)
	}

	// 3. If context provided, find trigger-matched tripwires to highlight
	var highlighted map[string]bool
	if ctx != "" {
		matched, _ := idx.FindTripwiresByTrigger(ctx)
		if len(matched) > 0 {
			highlighted = make(map[string]bool)
			for _, tw := range matched {
				highlighted[tw.Slug] = true
			}
		}
	}

	if len(tripwires) > 0 {
		sb.WriteString("## Tripwires (Do Not Violate)\n\n")

		// Render highlighted tripwires first if any
		if len(highlighted) > 0 {
			sb.WriteString("### ⚠ Relevant to your current context\n\n")
			for _, tw := range tripwires {
				if !highlighted[tw.Slug] {
					continue
				}
				formatTripwireBlock(&sb, tw)
			}
			sb.WriteString("### All active tripwires\n\n")
		}

		for _, tw := range tripwires {
			if highlighted[tw.Slug] {
				continue // already rendered above
			}
			formatTripwireBlock(&sb, tw)
		}
	} else {
		sb.WriteString("No active tripwires defined.\n")
	}

	if m != nil && m.MissionFitTest != "" {
		sb.WriteString("\n## Mission-Fit Test\n\n")
		sb.WriteString(m.MissionFitTest)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// tripwireWarning checks if any active tripwire triggers match the query
// and returns a warning block to prepend, or "" if no matches.
func (s *MCPServer) tripwireWarning(query string) string {
	idx, err := index.Open(s.heroDir)
	if err != nil {
		return ""
	}
	defer idx.Close()

	matched, err := idx.FindTripwiresByTrigger(query)
	if err != nil || len(matched) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## TRIPWIRE WARNING\n\n")
	sb.WriteString("Your query matches one or more forbidden-option tripwires. Review before proceeding:\n\n")
	for _, tw := range matched {
		formatTripwireBlock(&sb, tw)
	}
	return sb.String()
}

func formatTripwireBlock(sb *strings.Builder, tw index.TripwireResult) {
	fmt.Fprintf(sb, "**%s** [%s]: %s\n", tw.Slug, tw.Severity, tw.Title)
	if tw.Constraint != "" {
		fmt.Fprintf(sb, "- Constraint: %s\n", tw.Constraint)
	}
	if tw.Why != "" {
		fmt.Fprintf(sb, "- Why: %s\n", tw.Why)
	}
	if tw.Instead != "" {
		fmt.Fprintf(sb, "- Instead: %s\n", tw.Instead)
	}
	sb.WriteString("\n")
}

func (s *MCPServer) toolPulse(args map[string]interface{}) (string, error) {
	since, _ := args["since"].(string)
	format, _ := args["format"].(string)

	cfg, _ := config.Load(s.projectRoot)

	sprintDays := 14
	if cfg.Pulse != nil && cfg.Pulse.SprintDays > 0 {
		sprintDays = cfg.Pulse.SprintDays
	}

	period := pulse.CalcPeriod(since, false, sprintDays)
	p, err := pulse.BuildPulse(s.heroDir, period, 3, 7)
	if err != nil {
		return "", fmt.Errorf("building pulse: %w", err)
	}

	// Populate drift
	driftSummaries := drift.DriftSummaries(s.heroDir, s.projectRoot)
	var driftEntries []pulse.DriftEntry
	for _, ds := range driftSummaries {
		driftEntries = append(driftEntries, pulse.DriftEntry{
			Slug:         ds.Slug,
			Title:        ds.Title,
			Warnings:     ds.Warnings,
			HasViolation: ds.HasViolation,
		})
	}
	pulse.PopulateDrift(p, driftEntries)

	// Ambient size-drift surface (workspace-wide; count + hint only).
	// Quiet/zero → leave SizeDrift nil so the renderers and JSON output
	// omit the field entirely. See spec roadmap-review-ambient-surfacing.
	ambientOpts := sizing.AmbientDriftOpts{}
	if cfg.Roadmap != nil {
		ambientOpts.RecencyDays = cfg.Roadmap.AmbientRecencyDaysOrDefault()
		ambientOpts.StopNaggingHours = cfg.Roadmap.StopNaggingHoursOrDefault()
	}
	ambient := sizing.AmbientDrift(s.heroDir, s.projectRoot, ambientOpts)
	if !ambient.Quiet && ambient.Count > 0 {
		p.SizeDrift = &pulse.AmbientSizeDrift{
			Count: ambient.Count,
			Hint:  ambient.Hint,
		}
	}

	switch format {
	case "json":
		return pulse.RenderJSON(p)
	case "markdown", "md":
		return pulse.RenderMarkdown(p), nil
	default:
		return pulse.RenderText(p), nil
	}
}

func (s *MCPServer) toolPlan(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	content, _ := args["content"].(string)
	if slug == "" || content == "" {
		return "", fmt.Errorf("slug and content are required")
	}

	// Find the spec to determine the plan path
	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	for _, sp := range specs {
		if sp.Slug == slug {
			planPath := filepath.Join(filepath.Dir(sp.Path), "plan.md")
			if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
				return "", fmt.Errorf("writing plan: %w", err)
			}
			return fmt.Sprintf("Plan saved to %s (%d bytes)", planPath, len(content)), nil
		}
	}

	// Spec not found — try common paths
	for _, dir := range []string{"planning/features", "planning/bugs"} {
		planDir := filepath.Join(s.heroDir, dir, slug)
		if info, statErr := os.Stat(planDir); statErr == nil && info.IsDir() {
			planPath := filepath.Join(planDir, "plan.md")
			if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
				return "", fmt.Errorf("writing plan: %w", err)
			}
			return fmt.Sprintf("Plan saved to %s (%d bytes)", planPath, len(content)), nil
		}
	}

	return "", fmt.Errorf("spec %q not found — create it first with /design", slug)
}

func (s *MCPServer) toolContract(args map[string]interface{}) (string, error) {
	action, _ := args["action"].(string)
	slug, _ := args["slug"].(string)

	switch action {
	case "status":
		if slug == "" {
			return "", fmt.Errorf("slug is required for status action")
		}
		specs, err := spec.Discover(s.heroDir)
		if err != nil {
			return "", fmt.Errorf("discovering specs: %w", err)
		}
		for _, sp := range specs {
			if sp.Slug == slug {
				report := contract.Status(sp)
				data, _ := json.MarshalIndent(report, "", "  ")
				return string(data), nil
			}
		}
		return "", fmt.Errorf("spec %q not found", slug)

	case "link":
		if slug == "" {
			return "", fmt.Errorf("slug is required for link action")
		}
		idxFloat, _ := args["criterion_index"].(float64)
		testRef, _ := args["test_ref"].(string)
		if idxFloat == 0 || testRef == "" {
			return "", fmt.Errorf("criterion_index and test_ref are required for link action")
		}
		specs, err := spec.Discover(s.heroDir)
		if err != nil {
			return "", fmt.Errorf("discovering specs: %w", err)
		}
		for _, sp := range specs {
			if sp.Slug == slug {
				if err := contract.Link(sp.Path, s.projectRoot, int(idxFloat), testRef); err != nil {
					return "", err
				}
				return fmt.Sprintf("Linked criterion %d to %s", int(idxFloat), testRef), nil
			}
		}
		return "", fmt.Errorf("spec %q not found", slug)

	case "check":
		specs, err := spec.Discover(s.heroDir)
		if err != nil {
			return "", fmt.Errorf("discovering specs: %w", err)
		}
		var results []contract.RegressionResult
		for _, sp := range specs {
			if slug != "" && sp.Slug != slug {
				continue
			}
			if sp.Status != spec.StatusCompleted && slug == "" {
				continue
			}
			results = append(results, contract.Check(sp, s.projectRoot)...)
		}
		data, _ := json.MarshalIndent(results, "", "  ")
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown action %q (use status, link, or check)", action)
	}
}

func (s *MCPServer) toolImpact(args map[string]interface{}) (string, error) {
	filePathsRaw, _ := args["file_paths"].([]interface{})
	if len(filePathsRaw) == 0 {
		return "", fmt.Errorf("file_paths parameter is required")
	}

	var filePaths []string
	for _, fp := range filePathsRaw {
		if s, ok := fp.(string); ok {
			filePaths = append(filePaths, s)
		}
	}

	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	reports, err := impact.Analyze(idx, filePaths)
	if err != nil {
		return "", fmt.Errorf("analyzing impact: %w", err)
	}

	return impact.RenderJSON(reports)
}

func (s *MCPServer) toolRecap(args map[string]interface{}) (string, error) {
	sinceStr, _ := args["since"].(string)
	subproject, _ := args["subproject"].(string)

	since, err := recap.ParseSince(sinceStr)
	if err != nil {
		return "", err
	}

	r, err := recap.Build(s.heroDir, s.projectRoot, since)
	if err != nil {
		return "", fmt.Errorf("building recap: %w", err)
	}

	if subproject != "" && subproject != "all" {
		filtered := r.Specs[:0]
		for _, sa := range r.Specs {
			if sa.Subproject == subproject {
				filtered = append(filtered, sa)
			}
		}
		r.Specs = filtered
	}

	full, err := recap.RenderJSON(r)
	if err != nil {
		return "", err
	}

	if argCompact(args) {
		summary := fmt.Sprintf("%d spec(s) active, %d knowledge update(s), %d unmatched commit(s) since %s",
			len(r.Specs), len(r.Knowledge), len(r.Unmatched), r.Since.Format("2006-01-02 15:04"))
		sourceArgs := map[string]any{"since": sinceStr, "subproject": subproject}
		envText, regErr := s.registerRef(refs.KindRecap, argHash(sourceArgs), "digest",
			sourceArgs, full, fingerprintArgs(sourceArgs), summary)
		if regErr == nil {
			return envText, nil
		}
	}

	return full, nil
}

func (s *MCPServer) toolDrift(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	inFlight, _ := args["in_flight"].(bool)
	initiative, _ := args["initiative"].(string)
	since, _ := args["since"].(string)

	var reports []*drift.Report
	var err error

	switch {
	case inFlight:
		reports, err = drift.AnalyzeAll(s.heroDir, s.projectRoot, since)
	case initiative != "":
		reports, err = drift.AnalyzeInitiative(s.heroDir, s.projectRoot, initiative, since)
	case slug != "":
		specs, discoverErr := spec.Discover(s.heroDir)
		if discoverErr != nil {
			return "", fmt.Errorf("discovering specs: %w", discoverErr)
		}
		for _, sp := range specs {
			if sp.Slug == slug {
				reports = []*drift.Report{drift.Analyze(sp, s.projectRoot, since)}
				break
			}
		}
		if reports == nil {
			return "", fmt.Errorf("spec %q not found", slug)
		}
	default:
		return "", fmt.Errorf("provide slug, in_flight, or initiative parameter")
	}

	if err != nil {
		return "", fmt.Errorf("analyzing drift: %w", err)
	}

	return drift.RenderJSON(reports)
}

func (s *MCPServer) toolSkillRun(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	if slug == "" {
		return "", fmt.Errorf("slug parameter is required")
	}
	paramsStr, _ := args["params"].(string)

	skillsDir := filepath.Join(s.heroDir, "skills")
	allSkills, err := skills.Discover(skillsDir)
	if err != nil {
		return "", fmt.Errorf("discovering skills: %w", err)
	}

	var skill *skills.Skill
	for _, sk := range allSkills {
		if sk.Slug == slug {
			skill = sk
			break
		}
	}
	if skill == nil {
		return fmt.Sprintf("Skill %q not found in %s", slug, skillsDir), nil
	}

	// Parse params string: key=value,key=value
	params := make(map[string]string)
	if paramsStr != "" {
		for _, pair := range strings.Split(paramsStr, ",") {
			pair = strings.TrimSpace(pair)
			if eqIdx := strings.Index(pair, "="); eqIdx >= 0 {
				k := strings.TrimSpace(pair[:eqIdx])
				v := strings.TrimSpace(pair[eqIdx+1:])
				params[k] = v
			}
		}
	}

	// Render skill steps for the agent to execute
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skill: %s\n", skill.Title))
	if skill.Notes != "" {
		sb.WriteString(fmt.Sprintf("Notes: %s\n", skill.Notes))
	}
	sb.WriteString(fmt.Sprintf("\nSteps (%d):\n", len(skill.Steps)))
	for _, step := range skill.Steps {
		text := step.Raw
		if interpolated, err := skills.InterpolateParams(step.Raw, params); err == nil {
			text = interpolated
		}
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", step.Index, step.Kind, text))
	}
	return sb.String(), nil
}

func (s *MCPServer) toolClaim(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	if slug == "" {
		return "", fmt.Errorf("slug parameter is required")
	}
	action, _ := args["action"].(string)
	if action == "" {
		action = "claim"
	}
	agentArg, _ := args["agent"].(string)

	// Resolve agent
	agent := agentArg
	if agent == "" {
		agent = os.Getenv("HERO_AGENT")
	}
	if agent == "" {
		cfg, _ := config.Load(s.projectRoot)
		if cfg.Tracking != nil && cfg.Tracking.DefaultAgent != "" {
			agent = cfg.Tracking.DefaultAgent
		}
	}
	if agent == "" {
		agent = "mcp-agent"
	}

	logPath := filepath.Join(s.heroDir, "events.log")
	now := time.Now()

	// Find spec file
	specs, _ := spec.Discover(s.heroDir)
	specPath := ""
	for _, sp := range specs {
		if sp.Slug == slug {
			specPath = sp.Path
			break
		}
	}

	switch action {
	case "claim":
		if specPath != "" {
			_ = tracking.UpdateSpecFrontmatter(specPath, "claim", agent, now)
		}
		if idx, err := index.Open(s.heroDir); err == nil {
			_ = idx.Claim(slug, agent)
			idx.Close()
		}
		_ = tracking.AppendEvent(logPath, tracking.ClaimEvent{
			Event: "claimed",
			Slug:  slug,
			Agent: agent,
			At:    now,
		})
		return fmt.Sprintf("Claimed %s for %s", slug, agent), nil

	case "release":
		if specPath != "" {
			_ = tracking.UpdateSpecFrontmatter(specPath, "release", agent, now)
		}
		if idx, err := index.Open(s.heroDir); err == nil {
			_ = idx.Unclaim(slug)
			idx.Close()
		}
		_ = tracking.AppendEvent(logPath, tracking.ClaimEvent{
			Event: "released",
			Slug:  slug,
			Agent: agent,
			At:    now,
		})
		return fmt.Sprintf("Released claim on %s", slug), nil

	case "complete":
		durMins := 0
		events, _ := tracking.ReadEvents(logPath)
		for i := len(events) - 1; i >= 0; i-- {
			evt := events[i]
			if evt.Slug == slug && evt.Agent == agent && evt.Event == "claimed" {
				durMins = int(now.Sub(evt.At).Minutes())
				break
			}
		}
		if specPath != "" {
			_ = tracking.UpdateSpecFrontmatter(specPath, "complete", agent, now)
		}
		if idx, err := index.Open(s.heroDir); err == nil {
			_ = idx.Unclaim(slug)
			idx.Close()
		}
		_ = tracking.AppendEvent(logPath, tracking.ClaimEvent{
			Event:           "completed",
			Slug:            slug,
			Agent:           agent,
			At:              now,
			DurationMinutes: durMins,
		})
		msg := fmt.Sprintf("Completed %s (agent: %s", slug, agent)
		if durMins > 0 {
			msg += fmt.Sprintf(", %d min", durMins)
		}
		msg += ")"
		return msg, nil

	default:
		return "", fmt.Errorf("unknown action %q; use claim, release, or complete", action)
	}
}

func (s *MCPServer) toolVelocity(args map[string]interface{}) (string, error) {
	sinceStr, _ := args["since"].(string)
	agentFilter, _ := args["agent"].(string)

	logPath := filepath.Join(s.heroDir, "events.log")
	events, err := tracking.ReadEvents(logPath)
	if err != nil {
		return "", fmt.Errorf("reading events log: %w", err)
	}

	var sinceTime time.Time
	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			sinceTime = t
		}
	}

	results := tracking.CalcVelocity(events, sinceTime)

	if len(results) == 0 {
		return "No completed specs found in the events log.", nil
	}

	var sb strings.Builder
	sb.WriteString("Agent velocity:\n\n")
	sb.WriteString(fmt.Sprintf("  %-30s  %8s  %10s  %-20s  %-20s\n",
		"Agent", "Done", "Avg days", "Fastest", "Slowest"))
	sb.WriteString("  " + strings.Repeat("─", 90) + "\n")

	for _, v := range results {
		if agentFilter != "" && v.Agent != agentFilter {
			continue
		}
		sb.WriteString(fmt.Sprintf("  %-30s  %8d  %10.1f  %-20s  %-20s\n",
			v.Agent, v.SpecsDone, v.AvgDays, v.FastestSlug, v.SlowestSlug))
	}
	return sb.String(), nil
}

// ---------------------------------------------------------------------------
// Output formatting helpers
// ---------------------------------------------------------------------------

// enrichCodeStructure adds code intelligence for packages containing the queried files.
func (s *MCPServer) enrichCodeStructure(ctx *index.ContextBlock, filePaths []string) {
	cfg, err := config.Load(s.projectRoot)
	if err != nil || cfg.CodeScan.IsDisabled() {
		return
	}
	codeDir := cfg.CodeDir(s.projectRoot)
	if _, err := os.Stat(codeDir); os.IsNotExist(err) {
		return
	}

	// Determine which package directories the queried files belong to
	seen := make(map[string]bool)
	for _, fp := range filePaths {
		dir := filepath.Dir(fp)
		if dir == "" {
			dir = "."
		}
		// slugify to match code knowledge directory names
		slug := strings.ReplaceAll(dir, "/", "-")
		slug = strings.ReplaceAll(slug, "\\", "-")
		slug = strings.ReplaceAll(slug, ".", "-")
		slug = strings.ToLower(slug)
		if slug == "" || slug == "-" {
			slug = "root"
		}
		if seen[slug] {
			continue
		}
		seen[slug] = true

		specPath := filepath.Join(codeDir, slug, "spec.md")
		content, err := os.ReadFile(specPath)
		if err != nil {
			continue
		}
		ctx.CodeStructure = append(ctx.CodeStructure, index.CodeContextEntry{
			PackageName: filepath.Base(dir),
			PackagePath: dir,
			Content:     string(content),
		})
	}

	// Check if any config vars reference the queried files
	configVars, err := codescan.GetConfigVars(codeDir)
	if err == nil && !strings.HasPrefix(configVars, "No environment") {
		// Check if any queried files are mentioned in the config vars output
		for _, fp := range filePaths {
			if strings.Contains(configVars, fp) {
				ctx.CodeStructure = append(ctx.CodeStructure, index.CodeContextEntry{
					PackageName: "Environment Variables",
					PackagePath: "(config)",
					Content:     configVars,
				})
				break
			}
		}
	}
}

// formatContextBlockWithVocab is the same as formatContextBlock but
// renders spec type names through the supplied vocabulary. A nil vocab
// preserves the canonical literal — keeping engineering / legacy
// workspaces' MCP output identical to today.
func formatContextBlockWithVocab(ctx *index.ContextBlock, vocab *vocabulary.Vocabulary) string {
	var sb strings.Builder
	sb.WriteString("## Relevant context from spec corpus\n\n")

	if len(ctx.Tripwires) > 0 {
		sb.WriteString("### Tripwires (do not violate)\n")
		for _, tw := range ctx.Tripwires {
			summary := tw.Summary
			if summary == "" {
				summary = tw.Title
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s (path: %s)\n", tw.Slug, summary, tw.Path))
		}
		sb.WriteString("\n")
	}

	if len(ctx.Conventions) > 0 {
		sb.WriteString("### Conventions to follow\n")
		for _, c := range ctx.Conventions {
			summary := c.Summary
			if summary == "" {
				summary = c.Title
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s (path: %s)\n", c.Slug, summary, c.Path))
		}
		sb.WriteString("\n")
	}

	if len(ctx.Rules) > 0 {
		sb.WriteString("### Rules (hard constraints)\n")
		for _, r := range ctx.Rules {
			summary := r.Summary
			if summary == "" {
				summary = r.Title
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s (path: %s)\n", r.Slug, summary, r.Path))
		}
		sb.WriteString("\n")
	}

	if len(ctx.InFlight) > 0 {
		sb.WriteString("### In-flight specs touching these files\n")
		for _, s := range ctx.InFlight {
			sb.WriteString(fmt.Sprintf("- **%s** (%s, %s): %s\n", s.Slug, displayType(vocab, string(s.Type)), s.Status, s.Title))
		}
		sb.WriteString("\n")
	}

	if len(ctx.PastWork) > 0 {
		sb.WriteString("### Past work in this area\n")
		for _, p := range ctx.PastWork {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", p.Slug, p.Status, p.Title))
		}
		sb.WriteString("\n")
	}

	if len(ctx.Decisions) > 0 {
		sb.WriteString("### Decisions that apply\n")
		for _, d := range ctx.Decisions {
			sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", d.Slug, d.Status, d.Title))
		}
		sb.WriteString("\n")
	}

	if len(ctx.KnownRisks) > 0 {
		sb.WriteString("### Known risks\n")
		for _, r := range ctx.KnownRisks {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", r.Slug, r.Title))
		}
		sb.WriteString("\n")
	}

	if len(ctx.External) > 0 {
		sb.WriteString("### External references\n")
		for _, e := range ctx.External {
			sb.WriteString(fmt.Sprintf("- **%s**: %s (path: %s)\n", e.Slug, e.Title, e.Path))
		}
		sb.WriteString("\n")
	}

	if len(ctx.CodeStructure) > 0 {
		sb.WriteString("### Code structure (relevant packages)\n\n")
		for _, cs := range ctx.CodeStructure {
			sb.WriteString(fmt.Sprintf("#### Package: %s (`%s`)\n\n", cs.PackageName, cs.PackagePath))
			// Include the content but skip the YAML frontmatter
			content := cs.Content
			if idx := strings.Index(content, "---\n"); idx >= 0 {
				if end := strings.Index(content[idx+4:], "---\n"); end >= 0 {
					content = strings.TrimSpace(content[idx+4+end+4:])
				}
			}
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
	}

	if len(ctx.TestCoverage) > 0 {
		sb.WriteString("### Test coverage\n")
		for _, tc := range ctx.TestCoverage {
			if tc.HasTest {
				sb.WriteString(fmt.Sprintf("- `%s` → has test: `%s`\n", tc.FilePath, tc.TestFile))
			} else {
				sb.WriteString(fmt.Sprintf("- `%s` → **no test file found**\n", tc.FilePath))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatSearchResults(results []index.SearchResult) string {
	var sb strings.Builder
	for _, r := range results {
		claimStr := ""
		if r.ClaimedBy != "" {
			claimStr = fmt.Sprintf(" [%s]", r.ClaimedBy)
		}
		sb.WriteString(fmt.Sprintf("%-30s  %-10s  %-10s  %s%s\n", r.Slug, r.Type, r.Status, r.Title, claimStr))
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", r.Snippet))
		}
	}
	sb.WriteString(fmt.Sprintf("\n%d result(s)\n", len(results)))
	return sb.String()
}

// formatRetrievalResults formats results from retrieval.Retrieve. FTS5 results
// use the standard tabular layout; graph results use a compact markdown form.
func formatRetrievalResults(results []retrieval.Result) string {
	var sb strings.Builder
	for _, r := range results {
		if r.Source == "graph" {
			title := r.Title
			if title == "" {
				title = r.Key
			}
			line := fmt.Sprintf("- **%s** _(%s, `%s`)_", title, r.Type, r.Key)
			if r.Snippet != "" {
				line += " — " + r.Snippet
			}
			sb.WriteString(line + "\n")
		} else {
			claimStr := ""
			if r.ClaimedBy != "" {
				claimStr = fmt.Sprintf(" [%s]", r.ClaimedBy)
			}
			sb.WriteString(fmt.Sprintf("%-30s  %-10s  %-10s  %s%s\n", r.Key, r.Type, r.Status, r.Title, claimStr))
			if r.Snippet != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", r.Snippet))
			}
		}
	}
	sb.WriteString(fmt.Sprintf("\n%d result(s)\n", len(results)))
	return sb.String()
}

func formatStatusOutput(specs []*spec.Spec) string {
	var sb strings.Builder

	groups := map[string][]*spec.Spec{
		"delivering": {},
		"in-review":  {},
		"planning":   {},
		"completed":  {},
	}
	var knowledge []*spec.Spec

	for _, sp := range specs {
		if sp.IsKnowledge() {
			knowledge = append(knowledge, sp)
			continue
		}
		switch sp.Status {
		case spec.StatusDelivering:
			groups["delivering"] = append(groups["delivering"], sp)
		case spec.StatusInReview:
			groups["in-review"] = append(groups["in-review"], sp)
		case spec.StatusPlanning:
			groups["planning"] = append(groups["planning"], sp)
		case spec.StatusCompleted:
			groups["completed"] = append(groups["completed"], sp)
		}
	}

	for _, label := range []string{"delivering", "in-review", "planning"} {
		items := groups[label]
		if len(items) > 0 {
			sb.WriteString(fmt.Sprintf("%s (%d):\n", strings.ToUpper(label[:1])+label[1:], len(items)))
			for _, sp := range items {
				sb.WriteString(fmt.Sprintf("  %-30s  %-10s  %s\n", sp.Slug, sp.Type, sp.Title))
			}
			sb.WriteString("\n")
		}
	}

	completed := groups["completed"]
	if len(completed) > 0 {
		sb.WriteString(fmt.Sprintf("Completed (%d):\n", len(completed)))
		for _, sp := range completed {
			sb.WriteString(fmt.Sprintf("  %-30s  %-10s  %s\n", sp.Slug, sp.Type, sp.Title))
		}
		sb.WriteString("\n")
	}

	if len(knowledge) > 0 {
		sb.WriteString(fmt.Sprintf("Knowledge (%d):\n", len(knowledge)))
		for _, sp := range knowledge {
			sb.WriteString(fmt.Sprintf("  %-30s  %-10s  %-8s  %s\n", sp.Slug, sp.Type, sp.Status, sp.Title))
		}
		sb.WriteString("\n")
	}

	inFlight := len(groups["delivering"]) + len(groups["in-review"]) + len(groups["planning"])
	sb.WriteString(fmt.Sprintf("Total: %d in-flight, %d completed, %d knowledge\n", inFlight, len(completed), len(knowledge)))

	return sb.String()
}

func formatCheckOutput(stats index.Stats, stale, unclaimed []index.SearchResult, staleDays int) string {
	var sb strings.Builder
	sb.WriteString("Hero workspace health check\n")
	sb.WriteString("===========================\n\n")
	sb.WriteString(fmt.Sprintf("Corpus: %d specs total\n", stats.TotalSpecs))
	sb.WriteString(fmt.Sprintf("  %d features, %d bugs, %d conventions, %d decisions\n",
		stats.Features, stats.Bugs, stats.Conventions, stats.Decisions))
	sb.WriteString(fmt.Sprintf("  %d files tracked, %d approach docs, %d root causes\n\n",
		stats.FilesTracked, stats.DecisionDocs, stats.RootCauses))

	issues := 0

	if len(stale) > 0 {
		issues += len(stale)
		sb.WriteString(fmt.Sprintf("Stale specs (>%d days in planning/in-review):\n", staleDays))
		for _, sp := range stale {
			sb.WriteString(fmt.Sprintf("  %-30s  %-10s  %s\n", sp.Slug, sp.Status, sp.Title))
		}
		sb.WriteString("\n")
	}

	if len(unclaimed) > 0 {
		issues += len(unclaimed)
		sb.WriteString("Unclaimed specs:\n")
		for _, sp := range unclaimed {
			sb.WriteString(fmt.Sprintf("  %-30s  %-10s  %s\n", sp.Slug, sp.Status, sp.Title))
		}
		sb.WriteString("\n")
	}

	if issues == 0 {
		sb.WriteString("No issues found.\n")
	} else {
		sb.WriteString(fmt.Sprintf("%d issue(s) found.\n", issues))
	}

	return sb.String()
}

func formatNudgeOutput(result *index.NudgeResult) string {
	var sb strings.Builder
	sb.WriteString("Hero — relevant context for these files:\n\n")

	if result.HasConventions {
		sb.WriteString("Conventions:\n")
		for _, c := range result.Conventions {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", c.Slug, c.Title))
		}
		sb.WriteString("\n")
	}

	if result.HasPastWork {
		sb.WriteString(fmt.Sprintf("Past work: %d spec(s) touched these files\n", len(result.RelatedSpecs)))
		for _, r := range result.RelatedSpecs {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", r.Slug, r.Title))
		}
		sb.WriteString("\n")
	}

	if result.HasPending {
		sb.WriteString("In-flight specs in this area:\n")
		for _, p := range result.PendingSpecs {
			sb.WriteString(fmt.Sprintf("  - %s (%s): %s\n", p.Slug, p.Status, p.Title))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatKnowledgeOutput(entries []*spec.Spec) string {
	var sb strings.Builder

	typeOrder := []spec.Type{
		spec.TypeConvention, spec.TypeDecision, spec.TypeRule,
		spec.TypeExternal, spec.TypeContext, spec.TypeNote,
	}
	typeLabels := map[spec.Type]string{
		spec.TypeConvention: "Conventions",
		spec.TypeDecision:   "Decisions",
		spec.TypeRule:       "Rules",
		spec.TypeExternal:   "External",
		spec.TypeContext:    "Context",
		spec.TypeNote:       "Notes",
	}

	grouped := map[spec.Type][]*spec.Spec{}
	for _, sp := range entries {
		grouped[sp.Type] = append(grouped[sp.Type], sp)
	}

	for _, t := range typeOrder {
		items := grouped[t]
		if len(items) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s (%d):\n", typeLabels[t], len(items)))
		for _, sp := range items {
			sb.WriteString(fmt.Sprintf("  %-30s  %-8s  %s\n", sp.Slug, sp.Status, sp.Title))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Total: %d knowledge entries\n", len(entries)))
	return sb.String()
}

func (s *MCPServer) toolTestGenerate(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	if slug == "" {
		return "", fmt.Errorf("slug parameter is required")
	}
	modeOverride, _ := args["mode"].(string)

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	specs, err := spec.Discover(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	var target *spec.Spec
	for _, sp := range specs {
		if sp.Slug == slug {
			target = sp
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("spec %q not found", slug)
	}

	testFile, err := herotest.Generate(s.projectRoot, target, cfg.Testing, modeOverride)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Generated test file: %s", testFile), nil
}

func (s *MCPServer) toolDemoRecord(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	if slug == "" {
		return "", fmt.Errorf("slug parameter is required")
	}

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}

	frameworkName := "playwright"
	if cfg.Demos != nil {
		frameworkName = cfg.Demos.FrameworkOrDefault()
	}

	fw, err := demos.Get(frameworkName)
	if err != nil {
		return "", err
	}

	testFile := herotest.TestFilePath(slug, cfg.Testing)
	if !herotest.TestFileExists(s.projectRoot, slug, cfg.Testing) {
		return "", fmt.Errorf("no test file found for %q at %s (run hero_test_generate first)", slug, testFile)
	}

	result, err := fw.Record(slug, testFile, cfg.Demos, cfg.Testing, s.projectRoot)
	if err != nil {
		if result != nil {
			data, _ := json.MarshalIndent(result, "", "  ")
			return string(data), nil
		}
		return "", err
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data), nil
}

func (s *MCPServer) toolCode(args map[string]interface{}) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		action = "search"
	}
	query, _ := args["query"].(string)
	kind, _ := args["kind"].(string)
	pkg, _ := args["package"].(string)

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	codeDir := cfg.CodeDir(s.projectRoot)

	switch action {
	case "search":
		results, err := codescan.Search(codeDir, codescan.SearchOptions{
			Query:   query,
			Kind:    kind,
			Package: pkg,
		})
		if err != nil {
			return "", err
		}
		if len(results) == 0 {
			return "No results found.", nil
		}
		var sb strings.Builder
		for _, r := range results {
			if r.SymbolName != "" {
				sb.WriteString(fmt.Sprintf("- **%s** (%s) in `%s` (`%s`)", r.SymbolName, r.SymbolKind, r.PackageName, r.PackagePath))
				if r.Signature != "" {
					sb.WriteString(fmt.Sprintf("\n  `%s`", r.Signature))
				}
				if r.Doc != "" {
					sb.WriteString(fmt.Sprintf("\n  %s", r.Doc))
				}
				sb.WriteString("\n")
			} else {
				sb.WriteString(fmt.Sprintf("- **%s** (`%s`)\n", r.PackageName, r.PackagePath))
			}
		}
		return sb.String(), nil

	case "package":
		if pkg == "" {
			return "", fmt.Errorf("package parameter is required for 'package' action")
		}
		return codescan.GetPackageInfo(codeDir, pkg)

	case "deps":
		return codescan.GetDependencyGraph(codeDir)

	case "hot":
		return codescan.GetHotFiles(codeDir)

	case "config":
		return codescan.GetConfigVars(codeDir)

	case "endpoints":
		return codescan.GetEndpoints(codeDir)

	case "overview":
		return codescan.GetOverview(codeDir)

	case "unenriched":
		limit := 20
		if l, _ := args["limit"].(string); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		symbols, err := codescan.GetUnenrichedSymbols(codeDir, limit)
		if err != nil {
			return "", err
		}
		if len(symbols) == 0 {
			return "All symbols are enriched.", nil
		}
		data, err := json.MarshalIndent(symbols, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "errors":
		patterns, err := errpattern.LoadPatterns(s.heroDir)
		if err != nil {
			return "", fmt.Errorf("loading error patterns: %w", err)
		}
		if len(patterns) == 0 {
			return "No error patterns recorded yet.", nil
		}
		if query != "" {
			patterns = errpattern.MatchByError(patterns, query)
			if len(patterns) == 0 {
				return "No error patterns match the given query.", nil
			}
		}
		return errpattern.FormatPatterns(patterns), nil

	default:
		return "", fmt.Errorf("unknown action: %s (use search, package, deps, hot, config, endpoints, errors, unenriched, or overview)", action)
	}
}

// enrichErrorPatterns loads error patterns and matches against the given file paths.
func enrichErrorPatterns(projectRoot string, filePaths []string) string {
	heroDir := filepath.Join(projectRoot, ".hero")
	patterns, err := errpattern.LoadPatterns(heroDir)
	if err != nil || len(patterns) == 0 {
		return ""
	}
	matched := errpattern.MatchByFile(patterns, filePaths)
	return errpattern.FormatPatterns(matched)
}

func (s *MCPServer) toolErrorPattern(args map[string]interface{}) (string, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return "", fmt.Errorf("id parameter is required")
	}

	p := errpattern.Pattern{ID: id}
	p.PatternRe, _ = args["pattern"].(string)
	p.Severity, _ = args["severity"].(string)
	p.Symptom, _ = args["symptom"].(string)
	p.RootCause, _ = args["root_cause"].(string)
	p.Fix, _ = args["fix"].(string)

	if stackStr, ok := args["stack"].(string); ok && stackStr != "" {
		for _, s := range strings.Split(stackStr, ",") {
			p.Stack = append(p.Stack, strings.TrimSpace(s))
		}
	}
	if filesStr, ok := args["files"].(string); ok && filesStr != "" {
		for _, f := range strings.Split(filesStr, ",") {
			p.Files = append(p.Files, strings.TrimSpace(f))
		}
	}

	if err := errpattern.SavePattern(s.heroDir, p); err != nil {
		return "", fmt.Errorf("saving error pattern: %w", err)
	}

	return fmt.Sprintf("Error pattern '%s' saved to .hero/knowledge/error-patterns/%s.md", id, id), nil
}

func (s *MCPServer) toolSynthesize(args map[string]interface{}) (string, error) {
	slugsStr, _ := args["slugs"].(string)
	if slugsStr == "" {
		return "", fmt.Errorf("slugs parameter is required (comma-separated spec slugs)")
	}
	var slugs []string
	for _, part := range strings.Split(slugsStr, ",") {
		if p := strings.TrimSpace(part); p != "" {
			slugs = append(slugs, p)
		}
	}

	pkt, err := synthesize.Assemble(s.heroDir, s.projectRoot, slugs)
	if err != nil {
		return "", err
	}

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	outPath := filepath.Join(cfg.ExplainersDir(s.projectRoot), pkt.OutSlug, "spec.md")
	today := time.Now().Format("2006-01-02")
	return pkt.AgentPacket(outPath, today), nil
}

func (s *MCPServer) toolEnrich(args map[string]interface{}) (string, error) {
	enrichmentsStr, _ := args["enrichments"].(string)
	if enrichmentsStr == "" {
		return "", fmt.Errorf("enrichments parameter is required")
	}

	var enrichments []struct {
		Package     string `json:"package"`
		Symbol      string `json:"symbol"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(enrichmentsStr), &enrichments); err != nil {
		return "", fmt.Errorf("parsing enrichments JSON: %w", err)
	}
	if len(enrichments) == 0 {
		return "No enrichments provided.", nil
	}

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	codeDir := cfg.CodeDir(s.projectRoot)

	cache, err := codescan.LoadEnrichmentCache(codeDir)
	if err != nil {
		return "", fmt.Errorf("loading enrichment cache: %w", err)
	}

	count := 0
	for _, e := range enrichments {
		if e.Package == "" || e.Symbol == "" || e.Description == "" {
			continue
		}
		key := codescan.EnrichmentKey(e.Package, e.Symbol, "")
		cache.Set(key, e.Description)
		count++
	}

	if err := cache.Save(codeDir); err != nil {
		return "", fmt.Errorf("saving enrichment cache: %w", err)
	}

	// Regenerate knowledge files with enrichments applied
	checksums, _ := codescan.LoadChecksums(codeDir)
	scanner := codescan.NewScanner(cfg.CodeScan, s.projectRoot)
	result, err := scanner.Scan(checksums)
	if err != nil {
		return fmt.Sprintf("Saved %d enrichment(s) but failed to regenerate knowledge: %v", count, err), nil
	}

	if err := codescan.GenerateKnowledgeWithEnrichments(result, codeDir, cache); err != nil {
		return fmt.Sprintf("Saved %d enrichment(s) but failed to regenerate knowledge: %v", count, err), nil
	}

	return fmt.Sprintf("Saved %d enrichment(s) and regenerated knowledge files.", count), nil
}

func (s *MCPServer) toolScore(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	if slug == "" {
		return "", fmt.Errorf("slug parameter is required")
	}

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(s.projectRoot)

	// Find spec by slug
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	var target *spec.Spec
	for _, sp := range specs {
		if sp.Slug == slug {
			target = sp
			break
		}
	}
	if target == nil {
		// Try as a file path
		target, err = spec.ParseFile(slug)
		if err != nil {
			return "", fmt.Errorf("spec %q not found", slug)
		}
	}

	scoreCfg := score.DefaultConfig()
	if cfg.Score != nil && cfg.Score.MinScore > 0 {
		scoreCfg.MinScore = cfg.Score.MinScore
	}

	result := score.Score(target, scoreCfg)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling result: %w", err)
	}

	// Build a readable summary + JSON
	var sb strings.Builder
	fmt.Fprintf(&sb, "Score: %d/100  Grade: %s", result.Score, result.Grade)
	if result.Deliverable {
		sb.WriteString("  (deliverable)\n")
	} else {
		fmt.Fprintf(&sb, "  (below minimum %d)\n", scoreCfg.MinScore)
	}
	sb.WriteString("\nDimensions:\n")
	for _, d := range result.Dimensions {
		fmt.Fprintf(&sb, "  %-25s %3.0f  %s\n", d.Name, d.Score, d.Details)
	}
	if len(result.Suggestions) > 0 {
		sb.WriteString("\nSuggestions:\n")
		for _, sug := range result.Suggestions {
			fmt.Fprintf(&sb, "  → %s\n", sug)
		}
	}
	sb.WriteString("\n---\n")
	sb.Write(data)

	return sb.String(), nil
}

func (s *MCPServer) toolDiagnose(args map[string]interface{}) (string, error) {
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(s.projectRoot)

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	batch, _ := args["batch"].(bool)
	slug, _ := args["slug"].(string)

	if batch {
		return s.toolDiagnoseBatch(specs)
	}

	if slug == "" {
		return "", fmt.Errorf("slug parameter is required (or set batch=true)")
	}

	var target *spec.Spec
	for _, sp := range specs {
		if sp.Slug == slug {
			target = sp
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("spec %q not found", slug)
	}

	if target.Type != spec.TypeBug {
		return "", fmt.Errorf("spec %q is type %q, not bug", slug, target.Type)
	}
	if target.Status == spec.StatusCompleted {
		return "", fmt.Errorf("spec %q is already completed", slug)
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return "", fmt.Errorf("reading spec: %w", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Bug: %s\n", target.Title)
	if target.TrackerID != "" {
		fmt.Fprintf(&sb, "Tracker: %s\n", target.TrackerID)
	}
	fmt.Fprintf(&sb, "Spec: %s\n\n", target.Path)
	fmt.Fprintf(&sb, "--- Spec Content ---\n%s\n--- End Spec ---\n\n", string(data))
	sb.WriteString("Investigation Instructions:\n")
	sb.WriteString("  1. Load the debugging-investigation skill\n")
	fmt.Fprintf(&sb, "  2. Read the spec at: %s\n", target.Path)
	sb.WriteString("  3. Investigate the root cause in the codebase\n")
	sb.WriteString("  4. Write findings into the spec file on disk:\n")
	sb.WriteString("     - ## Investigation (what you found)\n")
	sb.WriteString("     - ## Root Cause (classified root cause)\n")
	sb.WriteString("     - ## Suggested Fix Approach (how to fix)\n")
	sb.WriteString("  5. Do NOT move, delete, or rename the spec file\n")

	return sb.String(), nil
}

func (s *MCPServer) toolDiagnoseBatch(specs []*spec.Spec) (string, error) {
	var bugs []map[string]string
	for _, sp := range specs {
		if sp.Type != spec.TypeBug || sp.Status == spec.StatusCompleted {
			continue
		}
		// Only imported (undiagnosed) bugs
		_, hasInvestigation := sp.Sections["investigation"]
		_, hasRootCause := sp.Sections["root cause"]
		if hasInvestigation || hasRootCause {
			continue
		}
		bugs = append(bugs, map[string]string{
			"slug":       sp.Slug,
			"title":      sp.Title,
			"tracker_id": sp.TrackerID,
			"path":       sp.Path,
		})
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Undiagnosed Bugs: %d\n\n", len(bugs))
	for i, b := range bugs {
		tracker := ""
		if b["tracker_id"] != "" {
			tracker = " [" + b["tracker_id"] + "]"
		}
		title := b["title"]
		if title == "" {
			title = b["slug"]
		}
		fmt.Fprintf(&sb, "%d. %s%s\n   Path: %s\n", i+1, title, tracker, b["path"])
	}

	if len(bugs) == 0 {
		sb.WriteString("No undiagnosed bugs found. Import bugs with 'hero import --type bug' first.\n")
	}

	return sb.String(), nil
}

func (s *MCPServer) toolVerify(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	if slug == "" {
		return "", fmt.Errorf("slug parameter is required")
	}

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(s.projectRoot)

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return "", fmt.Errorf("discovering specs: %w", err)
	}

	var target *spec.Spec
	for _, sp := range specs {
		if sp.Slug == slug {
			target = sp
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("spec %q not found", slug)
	}

	// Score the spec
	scoreCfg := score.DefaultConfig()
	if cfg.Score != nil && cfg.Score.MinScore > 0 {
		scoreCfg.MinScore = cfg.Score.MinScore
	}
	result := score.Score(target, scoreCfg)

	// Build verification report
	var sb strings.Builder
	fmt.Fprintf(&sb, "Verification Report: %s\n", target.Slug)
	fmt.Fprintf(&sb, "Spec Quality: %d/100 (Grade %s)\n\n", result.Score, result.Grade)

	// Extract acceptance criteria
	acSection := ""
	for name, content := range target.Sections {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "acceptance") || strings.Contains(lower, "success criteria") ||
			strings.Contains(lower, "done when") || strings.Contains(lower, "definition of done") {
			acSection = content
			break
		}
	}

	if acSection != "" {
		sb.WriteString("Acceptance Criteria:\n")
		criteria := extractCriteriaItemsMCP(acSection)
		for i, c := range criteria {
			fmt.Fprintf(&sb, "  %d. [ ] %s\n", i+1, c)
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("⚠ No acceptance criteria section found.\n\n")
	}

	if len(target.FilesTouched) > 0 {
		sb.WriteString("Expected File Changes:\n")
		for _, f := range target.FilesTouched {
			fmt.Fprintf(&sb, "  - %s\n", f)
		}
		sb.WriteString("\n")
	}

	// Test strategy
	for name, content := range target.Sections {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "test") || strings.Contains(lower, "verification") {
			sb.WriteString("Test Strategy:\n")
			items := extractCriteriaItemsMCP(content)
			for _, item := range items {
				fmt.Fprintf(&sb, "  • %s\n", item)
			}
			sb.WriteString("\n")
			break
		}
	}

	sb.WriteString("Instructions: For each acceptance criterion, examine the implementation code, run tests if possible, and report PASS/FAIL with evidence. Give an overall verdict at the end.\n")

	return sb.String(), nil
}

// extractCriteriaItemsMCP pulls list items from a markdown section.
func extractCriteriaItemsMCP(section string) []string {
	var items []string
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) < 3 {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			items = append(items, strings.TrimSpace(trimmed[2:]))
		} else if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			for i := 1; i < len(trimmed); i++ {
				if trimmed[i] == '.' || trimmed[i] == ')' {
					if i+1 < len(trimmed) && trimmed[i+1] == ' ' {
						items = append(items, strings.TrimSpace(trimmed[i+2:]))
					}
					break
				}
				if trimmed[i] < '0' || trimmed[i] > '9' {
					break
				}
			}
		}
	}
	return items
}

func (s *MCPServer) toolConflicts(args map[string]interface{}) (string, error) {
	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	slug, _ := args["slug"].(string)

	if slug != "" {
		// Find conflicts for a specific spec
		conflicts, err := idx.FindConflicts(slug)
		if err != nil {
			return "", fmt.Errorf("finding conflicts: %w", err)
		}
		if len(conflicts) == 0 {
			return fmt.Sprintf("No conflicts found for spec '%s'. No other in-flight specs touch the same files.", slug), nil
		}

		var buf strings.Builder
		fmt.Fprintf(&buf, "## Conflicts for `%s`\n\n", slug)
		fmt.Fprintf(&buf, "Found %d spec(s) with overlapping files:\n\n", len(conflicts))
		for _, c := range conflicts {
			fmt.Fprintf(&buf, "### %s\n", c.Slug)
			fmt.Fprintf(&buf, "- **Title:** %s\n", c.Title)
			fmt.Fprintf(&buf, "- **Type:** %s\n", c.Type)
			fmt.Fprintf(&buf, "- **Status:** %s\n", c.Status)
			if c.ClaimedBy != "" {
				fmt.Fprintf(&buf, "- **Claimed by:** %s\n", c.ClaimedBy)
			}
			fmt.Fprintf(&buf, "- **Overlapping files:**\n")
			for _, f := range c.OverlappingFiles {
				fmt.Fprintf(&buf, "  - `%s`\n", f)
			}
			buf.WriteString("\n")
		}
		return buf.String(), nil
	}

	// Find all conflicting pairs — use FindAllConflicts via index
	// For local mode, iterate all in-flight specs and find conflicts
	allSpecs, err := idx.AllSpecs()
	if err != nil {
		return "", fmt.Errorf("listing specs: %w", err)
	}

	seen := make(map[string]bool)
	var allConflicts []index.ConflictResult
	for _, sp := range allSpecs {
		if sp.Status != "planning" && sp.Status != "in-review" && sp.Status != "delivering" {
			continue
		}
		conflicts, err := idx.FindConflicts(sp.Slug)
		if err != nil {
			continue
		}
		for _, c := range conflicts {
			pairKey := sp.Slug + ":" + c.Slug
			if sp.Slug > c.Slug {
				pairKey = c.Slug + ":" + sp.Slug
			}
			if !seen[pairKey] {
				seen[pairKey] = true
				allConflicts = append(allConflicts, c)
			}
		}
	}

	if len(allConflicts) == 0 {
		return "No file conflicts found among in-flight specs.", nil
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "## All Spec Conflicts\n\n")
	fmt.Fprintf(&buf, "Found %d conflicting pair(s):\n\n", len(allConflicts))
	for _, c := range allConflicts {
		fmt.Fprintf(&buf, "- **%s** (%s): %d overlapping file(s)\n", c.Slug, c.Status, len(c.OverlappingFiles))
		for _, f := range c.OverlappingFiles {
			fmt.Fprintf(&buf, "  - `%s`\n", f)
		}
	}
	return buf.String(), nil
}

func (s *MCPServer) toolSequence(args map[string]interface{}) (string, error) {
	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	items, err := idx.SuggestSequence()
	if err != nil {
		return "", fmt.Errorf("suggesting sequence: %w", err)
	}

	if len(items) == 0 {
		return "No in-flight specs to sequence.", nil
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "## Suggested Delivery Sequence\n\n")
	fmt.Fprintf(&buf, "%d in-flight spec(s) ordered by dependencies and file conflict risk:\n\n", len(items))

	for _, item := range items {
		fmt.Fprintf(&buf, "### %d. %s\n", item.Order, item.Slug)
		fmt.Fprintf(&buf, "- **Title:** %s\n", item.Title)
		fmt.Fprintf(&buf, "- **Type:** %s | **Status:** %s\n", item.Type, item.Status)
		fmt.Fprintf(&buf, "- **Reason:** %s\n", item.Reason)
		if len(item.DependsOn) > 0 {
			fmt.Fprintf(&buf, "- **Depends on:** %s\n", strings.Join(item.DependsOn, ", "))
		}
		if len(item.ConflictsWith) > 0 {
			fmt.Fprintf(&buf, "- **Conflicts with:** %s\n", strings.Join(item.ConflictsWith, ", "))
		}
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

func (s *MCPServer) toolWarnings(args map[string]interface{}) (string, error) {
	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	slug, _ := args["slug"].(string)
	var buf strings.Builder

	if slug != "" {
		// Spec-specific warnings
		fmt.Fprintf(&buf, "## Warnings for `%s`\n\n", slug)

		// Check if spec exists
		specResult, err := idx.Search(slug)
		if err != nil || len(specResult) == 0 {
			return fmt.Sprintf("Spec '%s' not found.", slug), nil
		}
		sp := specResult[0]

		var warnings []string

		// Stale check
		if sp.Status == "planning" || sp.Status == "in-review" {
			stale, _ := idx.CheckStale(14)
			for _, st := range stale {
				if st.Slug == slug {
					warnings = append(warnings, "**Stale**: This spec has been in "+string(sp.Status)+" for over 14 days.")
					break
				}
			}
		}

		// Conflict check
		conflicts, _ := idx.FindConflicts(slug)
		if len(conflicts) > 0 {
			conflictSlugs := make([]string, len(conflicts))
			for i, c := range conflicts {
				conflictSlugs[i] = c.Slug
			}
			warnings = append(warnings, fmt.Sprintf("**File conflicts**: Overlaps with %d spec(s): %s. Coordinate delivery order.",
				len(conflicts), strings.Join(conflictSlugs, ", ")))
		}

		// Unclaimed check
		if sp.ClaimedBy == "" && (sp.Status == "planning" || sp.Status == "in-review") {
			warnings = append(warnings, "**Unclaimed**: No one has claimed this spec yet.")
		}

		if len(warnings) == 0 {
			return fmt.Sprintf("No warnings for spec '%s'. Looks good.", slug), nil
		}

		for i, w := range warnings {
			fmt.Fprintf(&buf, "%d. %s\n", i+1, w)
		}
		return buf.String(), nil
	}

	// Workspace-wide warnings
	fmt.Fprintf(&buf, "## Workspace Warnings\n\n")

	var warnings []string

	// Stale specs
	stale, _ := idx.CheckStale(14)
	if len(stale) > 0 {
		slugs := make([]string, len(stale))
		for i, s := range stale {
			slugs[i] = s.Slug
		}
		warnings = append(warnings, fmt.Sprintf("**%d stale spec(s)** (>14 days in planning/in-review): %s",
			len(stale), strings.Join(slugs, ", ")))
	}

	// Unclaimed in-flight specs
	allSpecs, _ := idx.AllSpecs()
	var unclaimed []string
	for _, sp := range allSpecs {
		if sp.ClaimedBy == "" && (sp.Status == "planning" || sp.Status == "in-review" || sp.Status == "delivering") {
			unclaimed = append(unclaimed, sp.Slug)
		}
	}
	if len(unclaimed) > 0 {
		warnings = append(warnings, fmt.Sprintf("**%d unclaimed spec(s)**: %s",
			len(unclaimed), strings.Join(unclaimed, ", ")))
	}

	// File conflicts
	seen := make(map[string]bool)
	var conflictPairs int
	for _, sp := range allSpecs {
		if sp.Status != "planning" && sp.Status != "in-review" && sp.Status != "delivering" {
			continue
		}
		conflicts, _ := idx.FindConflicts(sp.Slug)
		for _, c := range conflicts {
			pairKey := sp.Slug + ":" + c.Slug
			if sp.Slug > c.Slug {
				pairKey = c.Slug + ":" + sp.Slug
			}
			if !seen[pairKey] {
				seen[pairKey] = true
				conflictPairs++
			}
		}
	}
	if conflictPairs > 0 {
		warnings = append(warnings, fmt.Sprintf("**%d file conflict pair(s)** detected among in-flight specs. Run `hero_conflicts` for details.", conflictPairs))
	}

	// Size drift — emit one warning per drifted spec, individually
	// actionable. Unlike `hero check` (which rate-limits to two
	// summary lines for human readability), MCP consumers benefit
	// from per-spec entries: a model can route each one to the right
	// next action. Spec: spec-size-and-promotion-nudge.
	specs, _ := spec.Discover(s.heroDir)
	leafDrift, containerDrift := sizing.CollectDrift(specs)
	for _, d := range leafDrift {
		kind := sizing.ClassifyLeafDriftKind(d.Declared, d.Bucket)
		primary, alternative := sizing.SuggestedAction(d.Slug, d.Declared, d.Bucket, kind)
		warnings = append(warnings, fmt.Sprintf(
			"**Size drift (leaf)** `%s`: declared `%s`, computed `%s`. Run %s, or %s.",
			d.Slug, d.Declared, d.Bucket, primary, alternative))
	}
	for _, d := range containerDrift {
		declared := d.Declared
		if declared == "" {
			declared = "(unset)"
		}
		if d.Indeterminate {
			// Indeterminate rollups carry no actionable alternative —
			// keep the existing single-clause form.
			warnings = append(warnings, fmt.Sprintf(
				"**Size drift (container)** `%s`: rollup indeterminate (%d child(ren) missing both declared and computable size). Declared: `%s`.",
				d.Slug, d.ChildCount, declared))
			continue
		}
		kind := sizing.DriftKindContainerLow
		if d.Declared == "" {
			kind = sizing.DriftKindContainerUnset
		}
		primary, alternative := sizing.SuggestedAction(d.Slug, d.Declared, d.Rollup, kind)
		warnings = append(warnings, fmt.Sprintf(
			"**Size drift (container)** `%s`: declared `%s`, rollup `%s` (%d child(ren)). Run %s, or %s.",
			d.Slug, declared, d.Rollup, d.ChildCount, primary, alternative))
	}

	if len(warnings) == 0 {
		return "No warnings. Workspace looks healthy.", nil
	}

	for i, w := range warnings {
		fmt.Fprintf(&buf, "%d. %s\n", i+1, w)
	}
	return buf.String(), nil
}

func (s *MCPServer) toolInsights(args map[string]interface{}) (string, error) {
	idx, err := index.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening index: %w", err)
	}
	defer idx.Close()

	var buf strings.Builder
	fmt.Fprintf(&buf, "## Workspace Insights\n\n")

	allSpecs, _ := idx.AllSpecs()
	if len(allSpecs) == 0 {
		return "No specs found. Create specs to start building institutional knowledge.", nil
	}

	// Analyze spec type distribution
	typeCounts := make(map[string]int)
	statusCounts := make(map[string]int)
	for _, sp := range allSpecs {
		typeCounts[string(sp.Type)]++
		statusCounts[string(sp.Status)]++
	}

	fmt.Fprintf(&buf, "### Project Profile\n")
	fmt.Fprintf(&buf, "- **Total specs:** %d\n", len(allSpecs))
	fmt.Fprintf(&buf, "- **Types:** ")
	first := true
	for t, c := range typeCounts {
		if !first {
			buf.WriteString(", ")
		}
		fmt.Fprintf(&buf, "%s (%d)", t, c)
		first = false
	}
	buf.WriteString("\n")

	// Insights based on patterns
	fmt.Fprintf(&buf, "\n### Recommendations\n\n")
	insightCount := 0

	// Check if there are many bugs vs features
	if bugs, ok := typeCounts["bug"]; ok {
		features := typeCounts["feature"]
		if features > 0 && bugs > features {
			insightCount++
			fmt.Fprintf(&buf, "%d. **High bug-to-feature ratio** (%d bugs vs %d features). ", insightCount, bugs, features)
			fmt.Fprintf(&buf, "Projects with similar ratios often benefit from adding test coverage conventions and pre-delivery verification gates.\n")
		}
	}

	// Check for stale specs
	stale, _ := idx.CheckStale(14)
	if len(stale) > 3 {
		insightCount++
		fmt.Fprintf(&buf, "%d. **Spec backlog building up** (%d specs stale >14 days). ", insightCount, len(stale))
		fmt.Fprintf(&buf, "Similar projects reduce backlog by running weekly triage sessions and using `hero_sequence` to prioritize delivery order.\n")
	}

	// Check for unclaimed specs
	var unclaimed int
	for _, sp := range allSpecs {
		if sp.ClaimedBy == "" && (sp.Status == "planning" || sp.Status == "delivering") {
			unclaimed++
		}
	}
	if unclaimed > 2 {
		insightCount++
		fmt.Fprintf(&buf, "%d. **%d unclaimed specs** in active states. ", insightCount, unclaimed)
		fmt.Fprintf(&buf, "Teams that assign ownership early see 40%% faster delivery on average.\n")
	}

	// Check for conflict density
	seen := make(map[string]bool)
	conflictPairs := 0
	for _, sp := range allSpecs {
		if sp.Status != "planning" && sp.Status != "in-review" && sp.Status != "delivering" {
			continue
		}
		conflicts, _ := idx.FindConflicts(sp.Slug)
		for _, c := range conflicts {
			key := sp.Slug + ":" + c.Slug
			if sp.Slug > c.Slug {
				key = c.Slug + ":" + sp.Slug
			}
			if !seen[key] {
				seen[key] = true
				conflictPairs++
			}
		}
	}
	if conflictPairs > 2 {
		insightCount++
		fmt.Fprintf(&buf, "%d. **High file overlap** (%d conflicting spec pairs). ", insightCount, conflictPairs)
		fmt.Fprintf(&buf, "Use `hero_sequence` to find optimal delivery order and reduce merge conflict risk.\n")
	}

	// Check for missing conventions
	convSpecs, _ := idx.ListFiltered("convention", "active", "", "")
	if len(convSpecs) == 0 && len(allSpecs) > 5 {
		insightCount++
		fmt.Fprintf(&buf, "%d. **No active conventions defined.** ", insightCount)
		fmt.Fprintf(&buf, "Projects of this size typically benefit from conventions around code review, testing, and documentation standards.\n")
	}

	if insightCount == 0 {
		fmt.Fprintf(&buf, "No specific recommendations at this time. Your workspace patterns look healthy.\n")
	}

	return buf.String(), nil
}

func (s *MCPServer) toolCI(args map[string]interface{}) (string, error) {
	branch, _ := args["branch"].(string)
	if branch == "" {
		// Try to get current branch
		out, err := exec.Command("git", "-C", s.projectRoot, "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err == nil {
			branch = strings.TrimSpace(string(out))
		}
		if branch == "" {
			branch = "main"
		}
	}

	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", err
	}

	if cfg.Environment == nil || cfg.Environment.CI == nil {
		return "No CI provider configured. Add environment.ci to hero.json.", nil
	}

	project := ""
	token := ""
	if cfg.Tracker != nil {
		project = cfg.Tracker.Project
		if cfg.Tracker.TokenEnv != "" {
			token = os.Getenv(cfg.Tracker.TokenEnv)
		}
	}

	provider, err := environment.NewCIProvider(cfg.Environment.CI.Provider, project, token)
	if err != nil {
		return "", err
	}

	status, err := provider.PipelineStatus(branch)
	if err != nil {
		return "", err
	}

	data, _ := json.MarshalIndent(status, "", "  ")
	return string(data), nil
}

func (s *MCPServer) toolFeed(args map[string]interface{}) (string, error) {
	logPath := filepath.Join(s.heroDir, "events.log")

	filter := feed.Filter{Limit: 20}

	if since, ok := args["since"].(string); ok && since != "" {
		t, err := feed.ParseSince(since)
		if err != nil {
			return "", err
		}
		filter.Since = t
	}
	if t, ok := args["type"].(string); ok {
		filter.Type = t
	}
	if slug, ok := args["slug"].(string); ok {
		filter.Slug = slug
	}
	if sp, ok := args["subproject"].(string); ok {
		filter.Subproject = sp
	}
	if limit, ok := args["limit"].(float64); ok {
		filter.Limit = int(limit)
	}

	events, err := feed.ReadEvents(logPath, filter)
	if err != nil {
		return "", err
	}

	full, err := feed.FormatJSON(events)
	if err != nil {
		return "", err
	}

	if argCompact(args) {
		summary := fmt.Sprintf("%d feed event(s)", len(events))
		if filter.Type != "" {
			summary += fmt.Sprintf(", type=%s", filter.Type)
		}
		if filter.Slug != "" {
			summary += fmt.Sprintf(", slug=%s", filter.Slug)
		}
		sourceArgs := map[string]any{
			"since": fmt.Sprintf("%v", args["since"]),
			"type":  filter.Type,
			"slug":  filter.Slug,
			"limit": filter.Limit,
		}
		envText, regErr := s.registerRef(refs.KindFeed, argHash(sourceArgs), "items",
			sourceArgs, full, fingerprintArgs(sourceArgs), summary)
		if regErr == nil {
			return envText, nil
		}
	}

	return full, nil
}

func (s *MCPServer) toolEvent(args map[string]interface{}) (string, error) {
	eventType, _ := args["type"].(string)
	message, _ := args["message"].(string)
	slug, _ := args["slug"].(string)

	if eventType == "" || message == "" {
		return "", fmt.Errorf("type and message are required")
	}
	if !feed.IsValidType(eventType) {
		return "", fmt.Errorf("unknown event type %q — valid: %s", eventType, strings.Join(feed.ValidTypes, ", "))
	}

	logPath := filepath.Join(s.heroDir, "events.log")
	evt := feed.FeedEvent{
		Type:    eventType,
		Agent:   "mcp/hero",
		Slug:    slug,
		Message: message,
	}

	if err := feed.AppendEvent(logPath, evt); err != nil {
		return "", err
	}
	return fmt.Sprintf("Logged %s event", eventType), nil
}

func (s *MCPServer) toolActive(args map[string]interface{}) (string, error) {
	action, _ := args["action"].(string)
	if action == "" {
		action = "list"
	}

	switch action {
	case "list":
		r := active.Load(s.heroDir)
		if len(r.Sessions) == 0 {
			return "No active sessions.", nil
		}
		var lines []string
		for id, sess := range r.Sessions {
			age := time.Since(sess.Started).Round(time.Minute)
			lines = append(lines, fmt.Sprintf("- %s: spec=%s command=%s (%s ago)", id, sess.Spec, sess.Command, age))
		}
		return strings.Join(lines, "\n"), nil

	case "register":
		sessionID, _ := args["session_id"].(string)
		specSlug, _ := args["spec"].(string)
		cmd, _ := args["command"].(string)
		if sessionID == "" || specSlug == "" {
			return "", fmt.Errorf("session_id and spec are required for register")
		}
		if cmd == "" {
			cmd = "/deliver"
		}
		if err := active.Register(s.heroDir, sessionID, specSlug, cmd); err != nil {
			return "", err
		}
		return fmt.Sprintf("Registered session %s → spec %s (%s)", sessionID, specSlug, cmd), nil

	case "unregister":
		sessionID, _ := args["session_id"].(string)
		if sessionID == "" {
			return "", fmt.Errorf("session_id is required for unregister")
		}
		if err := active.Unregister(s.heroDir, sessionID); err != nil {
			return "", err
		}
		return fmt.Sprintf("Unregistered session %s", sessionID), nil

	case "prune":
		pruned, err := active.Prune(s.heroDir, 24*time.Hour)
		if err != nil {
			return "", err
		}
		if pruned == 0 {
			return "No stale sessions to prune.", nil
		}
		return fmt.Sprintf("Pruned %d stale session(s).", pruned), nil

	default:
		return "", fmt.Errorf("unknown action %q — use list, register, unregister, or prune", action)
	}
}

func (s *MCPServer) toolCoverage(args map[string]interface{}) (string, error) {
	slug, _ := args["slug"].(string)
	all, _ := args["all"].(bool)
	testDir, _ := args["test_dir"].(string)

	if !all && slug == "" {
		return "", fmt.Errorf("slug or all parameter is required")
	}

	if all {
		reports, err := coverage.AnalyzeAll(s.projectRoot, s.heroDir, testDir)
		if err != nil {
			return "", err
		}
		if len(reports) == 0 {
			return "No specs with acceptance criteria found.", nil
		}
		var parts []string
		for _, r := range reports {
			out, err := coverage.FormatJSON(r)
			if err != nil {
				continue
			}
			parts = append(parts, out)
		}
		return "[" + strings.Join(parts, ",\n") + "]", nil
	}

	r, err := coverage.Analyze(s.projectRoot, s.heroDir, slug, testDir)
	if err != nil {
		return "", err
	}
	return coverage.FormatJSON(r)
}


// toolWhy is the MCP wrapper around `hero why <target>` — multi-hop
// origin traversal. Returns the trace as JSON so an agent can both
// render it and walk it programmatically.
func (s *MCPServer) toolWhy(args map[string]interface{}) (string, error) {
	target, _ := args["target"].(string)
	if target == "" {
		return "", fmt.Errorf("target parameter is required")
	}
	depth := traversal.DefaultDepth
	if d, ok := args["depth"].(float64); ok && d > 0 {
		depth = int(d)
	}

	store, err := graph.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(s.projectRoot)
	trace, err := traversal.Why(store, repoKey, target, depth)
	if err != nil {
		return "", err
	}

	out, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return "", err
	}
	full := string(out)

	if argCompact(args) {
		hopCount := 0
		hopCount = len(trace.Chains)
		summary := fmt.Sprintf("origin trace for %q at depth %d: %d chain(s)", target, depth, hopCount)
		sourceArgs := map[string]any{"target": target, "depth": depth}
		envText, regErr := s.registerRef(refs.KindWhy, argHash(sourceArgs), "trace",
			sourceArgs, full, fingerprintArgs(sourceArgs), summary)
		if regErr == nil {
			return envText, nil
		}
	}

	return full, nil
}

// toolBlocked is the MCP wrapper around `hero blocked` — surfaces
// open Features whose dependencies are unmet OR whose ACs are
// failing/regressed. The agent receives a structured tree so it
// can prioritize without re-querying the graph.
func (s *MCPServer) toolBlocked(args map[string]interface{}) (string, error) {
	store, err := graph.Open(s.heroDir)
	if err != nil {
		return "", fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()

	repoKey := gitutil.RepoKey(s.projectRoot)

	// Dep-blocked features (single-hop today; recursive comes when
	// the depends_on graph grows).
	type depBlocker struct {
		Type   string `json:"type"`
		Key    string `json:"key"`
		Status string `json:"status"`
	}
	type blockedFeature struct {
		Slug       string       `json:"slug"`
		Title      string       `json:"title"`
		DepBlocked []depBlocker `json:"dep_blocked,omitempty"`
		FailingACs []string     `json:"failing_acs,omitempty"`
	}
	rows, err := store.DB().Query(
		`SELECT f.key,
		        COALESCE(json_extract(f.props, '$.title'), f.key) AS title,
		        b.type, b.key,
		        COALESCE(json_extract(b.props, '$.status'), '')
		   FROM nodes f
		   JOIN edges e ON e.from_id = f.id AND e.type IN ('depends_on','blocks') AND e.valid_to IS NULL
		   JOIN nodes b ON e.to_id = b.id AND b.valid_to IS NULL
		  WHERE f.type = 'Feature' AND f.repo = ? AND f.valid_to IS NULL
		    AND COALESCE(json_extract(f.props, '$.status'), '') NOT IN ('completed','superseded')
		    AND COALESCE(json_extract(b.props, '$.status'), '') NOT IN ('completed','accepted')
		  ORDER BY f.key`, repoKey)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	byFeature := map[string]*blockedFeature{}
	for rows.Next() {
		var fkey, ftitle, btype, bkey, bstatus string
		if err := rows.Scan(&fkey, &ftitle, &btype, &bkey, &bstatus); err != nil {
			return "", err
		}
		bf, ok := byFeature[fkey]
		if !ok {
			bf = &blockedFeature{Slug: fkey, Title: ftitle}
			byFeature[fkey] = bf
		}
		bf.DepBlocked = append(bf.DepBlocked, depBlocker{Type: btype, Key: bkey, Status: bstatus})
	}

	// AC failures — same `acceptance.FailingAcrossCorpus` the CLI uses.
	if failing, err := acceptance.FailingAcrossCorpus(store); err == nil {
		for _, c := range failing {
			if c.Parent == "" {
				continue
			}
			bf, ok := byFeature[c.Parent]
			if !ok {
				bf = &blockedFeature{Slug: c.Parent, Title: c.Parent}
				byFeature[c.Parent] = bf
			}
			bf.FailingACs = append(bf.FailingACs, c.Key)
		}
	}

	if len(byFeature) == 0 {
		empty := `{"blocked":[]}`
		if argCompact(args) {
			summary := "no blocked features"
			envText, regErr := s.registerRef(refs.KindBlocked, "all", "list",
				map[string]any{}, empty, "", summary)
			if regErr == nil {
				return envText, nil
			}
		}
		return empty, nil
	}
	out := make([]*blockedFeature, 0, len(byFeature))
	for _, bf := range byFeature {
		out = append(out, bf)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	wrapper := map[string]any{"blocked": out}
	b, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return "", err
	}
	full := string(b)

	if argCompact(args) {
		depCount, acCount := 0, 0
		for _, bf := range out {
			depCount += len(bf.DepBlocked)
			acCount += len(bf.FailingACs)
		}
		summary := fmt.Sprintf("%d blocked feature(s); %d dep block(s), %d failing AC(s)", len(out), depCount, acCount)
		envText, regErr := s.registerRef(refs.KindBlocked, "all", "list",
			map[string]any{}, full, "", summary)
		if regErr == nil {
			return envText, nil
		}
	}

	return full, nil
}