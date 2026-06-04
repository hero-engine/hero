// Package digest is hero's per-turn context digester.
//
// The principle: hero captures everything (the graph is unbounded);
// the model sees a bounded, ranked, pruned brief tailored to the
// current turn. As the corpus grows 10x, the brief stays the same
// size — just denser, more selective. If the brief grows linearly
// with corpus size we built a logger, not a digester.
//
// This package owns three concerns:
//  1. Querying the graph for candidate facts (more than will fit)
//  2. Scoring candidates (recency × focus_match × signal_weight)
//  3. Enforcing a token budget — top-K by score, byte-aware rendering
//
// All three live together because they tune against each other.
package digest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
)

// Options tunes a brief render.
type Options struct {
	RepoKey     string   // partition filter; required
	Branch      string   // current branch (frontmatter only)
	SessionID   string   // anchors "tried" attempts to a session
	AuthorEmail string   // who's asking (powers Person lookup, focus_match)
	User        string   // user slug — handoff singleton key (last ask / suggested-next / reflections)
	Domain      string   // domain partition for the handoff key; "" → handoff section omitted
	FocusFiles  []string // files in working tree → biases scoring
	TokenBudget int      // default 3000 if 0
	Now         time.Time
}

// Brief is the structured output of a digest. Markdown renders cleanly
// from it; JSON callers can post-process.
type Brief struct {
	Generated   string         `json:"generated"`
	RepoKey     string         `json:"repo"`
	Branch      string         `json:"branch,omitempty"`
	SessionID   string         `json:"session,omitempty"`
	BudgetUsed  int            `json:"budget_used"`
	BudgetTotal int            `json:"budget_total"`
	Sections    []BriefSection `json:"sections"`
}

// BriefSection is a named, budgeted block of facts.
type BriefSection struct {
	Title     string   `json:"title"`
	Lines     []string `json:"lines"`
	Truncated int      `json:"truncated_count,omitempty"` // how many candidates didn't fit
	Tokens    int      `json:"tokens_used"`               // approx
}

// Soft-budget knobs. The intent is "scale brief size to project
// complexity, but don't let it drift to log-territory." A small repo
// gets a small brief; a complex one with rich history grows — but
// only when high-scoring items justify it.
const (
	defaultFloor       = 1500  // never go smaller — minimum useful brief
	defaultSoftTarget  = 3000  // typical brief size; first place we trim past
	defaultHardCap     = 12000 // beyond this we're a logger, not a digester
	relevanceThreshold = 0.05  // below this score, items are dropped past target
)

// Section budget allocation as fractions of soft target. Sections grow
// proportionally if the soft target rises with corpus complexity.
type budgetPlan struct {
	Mission, WhoYouAre, Handoff, Sprint, InFlight, JustChanged, TriedAndFailed, BlockedOn, Nearby, Acceptance int
}

func planFor(softTarget int) budgetPlan {
	if softTarget <= 0 {
		softTarget = defaultSoftTarget
	}
	return budgetPlan{
		Mission:        softTarget * 250 / 3000,
		WhoYouAre:      softTarget * 100 / 3000,
		Handoff:        softTarget * 300 / 3000,
		Sprint:         softTarget * 400 / 3000,
		InFlight:       softTarget * 400 / 3000,
		JustChanged:    softTarget * 350 / 3000,
		TriedAndFailed: softTarget * 600 / 3000,
		BlockedOn:      softTarget * 300 / 3000,
		Nearby:         softTarget * 600 / 3000,
		Acceptance:     softTarget * 350 / 3000,
	}
}

// Generate produces a Brief by running graph queries, scoring
// candidates, and packing them into the budget.
func Generate(store *graph.Store, opts Options) (*Brief, error) {
	if store == nil {
		return nil, fmt.Errorf("digest: nil Store")
	}
	if opts.RepoKey == "" {
		return nil, fmt.Errorf("digest: RepoKey required")
	}
	if opts.TokenBudget == 0 {
		opts.TokenBudget = 3000
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	plan := planFor(opts.TokenBudget)

	b := &Brief{
		Generated:   opts.Now.Format(time.RFC3339),
		RepoKey:     opts.RepoKey,
		Branch:      opts.Branch,
		SessionID:   opts.SessionID,
		BudgetTotal: opts.TokenBudget,
	}

	// 0. Mission charter — highest-priority block per project-charter
	// AC-3. Renders nothing for repos without a Mission node so
	// pre-charter workspaces stay clean.
	if sec, err := missionSection(store, opts, plan.Mission); err != nil {
		return nil, err
	} else if len(sec.Lines) > 0 {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 1. Who you are
	if sec, err := whoYouAreSection(store, opts, plan.WhoYouAre); err != nil {
		return nil, err
	} else {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 1b. Where you left off — the handoff singletons (last user ask,
	// suggested-next, recent reflections) for the current user. Placed
	// just after identity and BEFORE in-flight/nearby: "what was I
	// doing" is the highest-value context for a fresh or cross-machine
	// session. Renders nothing when User=="" or no handoff nodes exist
	// (fresh repos stay clean), and is best-effort — a read error is
	// logged and skipped, never failing Generate.
	if sec, err := handoffSection(store, opts, plan.Handoff); err != nil {
		return nil, err
	} else if len(sec.Lines) > 0 {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 2. Active sprint (only renders if a Sprint node with state=active
	// exists — fresh repos and non-tracker workflows skip this section).
	if sec, err := sprintSection(store, opts, plan.Sprint); err != nil {
		return nil, err
	} else if len(sec.Lines) > 0 {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 3. What's in flight
	if sec, err := inFlightSection(store, opts, plan.InFlight); err != nil {
		return nil, err
	} else {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 3. What just changed
	if sec, err := justChangedSection(store, opts, plan.JustChanged); err != nil {
		return nil, err
	} else {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 4. What's been tried (high signal — agents repeat dead ends)
	if sec, err := triedSection(store, opts, plan.TriedAndFailed); err != nil {
		return nil, err
	} else {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 5. Blocked on
	if sec, err := blockedSection(store, opts, plan.BlockedOn); err != nil {
		return nil, err
	} else {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 5b. Acceptance criteria — open ACs that flipped recently +
	// currently failing/regressed across the corpus. Renders nothing
	// when no Criterion nodes exist (fresh repos, pre-Phase-1
	// corpora) — the section only shows up when it has a signal.
	if sec, err := acceptanceSection(store, opts, plan.Acceptance); err != nil {
		return nil, err
	} else if len(sec.Lines) > 0 {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	// 6. Nearby — Decisions/Notes related to focus files
	if sec, err := nearbySection(store, opts, plan.Nearby); err != nil {
		return nil, err
	} else {
		b.Sections = append(b.Sections, sec)
		b.BudgetUsed += sec.Tokens
	}

	return b, nil
}

// --- scoring ---------------------------------------------------------------

// score combines recency decay (half-life 30 days), focus-file match
// boost, and a per-section signal weight that captures how
// expensive-to-recover the fact is. Attempts score high because
// repeating them costs the agent real time.
func score(itemTime time.Time, focusBoost, signalWeight float64, now time.Time) float64 {
	if itemTime.IsZero() {
		itemTime = now.Add(-365 * 24 * time.Hour) // unknown date sinks
	}
	ageDays := now.Sub(itemTime).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	const halfLifeDays = 30.0
	recency := pow2(-ageDays / halfLifeDays)
	return recency * (1 + focusBoost) * signalWeight
}

func pow2(x float64) float64 {
	if x < -50 {
		return 0
	}
	if x > 50 {
		return 1e15
	}
	return math.Pow(2, x)
}

// --- query + render --------------------------------------------------------

func whoYouAreSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Who you are"}
	if opts.AuthorEmail == "" {
		sec.Lines = []string{"_unknown — pass --email or set git user.email_"}
		sec.Tokens = approxTokens(sec.Lines)
		return sec, nil
	}
	row := store.DB().QueryRow(
		`SELECT json_extract(props, '$.name') FROM nodes
		  WHERE type = 'Person' AND key = ? AND valid_to IS NULL`,
		strings.ToLower(opts.AuthorEmail),
	)
	var name sql.NullString
	if err := row.Scan(&name); err != nil && err != sql.ErrNoRows {
		return sec, err
	}
	display := name.String
	if display == "" {
		display = opts.AuthorEmail
	}
	sec.Lines = append(sec.Lines, fmt.Sprintf("%s <%s>", display, opts.AuthorEmail))

	// Features they've claimed — high signal for "what am I working on?"
	rows, err := store.DB().Query(
		`SELECT json_extract(props, '$.title'), key
		   FROM nodes
		  WHERE type = 'Feature' AND repo = ? AND valid_to IS NULL
		    AND lower(COALESCE(json_extract(props, '$.claimed_by'), '')) = lower(?)
		    AND COALESCE(json_extract(props, '$.status'), '') NOT IN ('completed','superseded')
		  LIMIT 5`,
		opts.RepoKey, opts.AuthorEmail,
	)
	if err == nil {
		defer rows.Close()
		var claims []string
		for rows.Next() {
			var title sql.NullString
			var key string
			if err := rows.Scan(&title, &key); err == nil {
				claims = append(claims, fmt.Sprintf("%s (`%s`)", title.String, key))
			}
		}
		if len(claims) > 0 {
			sec.Lines = append(sec.Lines, "Claimed: "+strings.Join(claims, "; "))
		}
	}

	enforceBudget(&sec, budget)
	return sec, nil
}

// handoffSection surfaces the per-user handoff singletons — the last
// user ask, the agent's suggested next step, and the most recent
// reflections — keyed by the same (user, repo, domain) triple the
// projection and auto-emit use. This closes the "load" half of the
// handoff loop: capture lands in the graph, travels via git, and now
// surfaces at session start.
//
// Best-effort by contract: an empty User (non-CLI callers that don't
// know the slug) or a read error yields an empty section the caller
// skips — it never fails Generate. When all three singletons are
// absent the section is empty too, so fresh repos stay clean.
func handoffSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Where you left off"}
	if opts.User == "" {
		return sec, nil
	}

	logSkip := func(what string, err error) {
		fmt.Fprintf(os.Stderr, "warning: digest handoff section: %s: %v\n", what, err)
	}

	if ask, err := handoff.LatestAsk(store, opts.User, opts.RepoKey, opts.Domain); err != nil {
		logSkip("last ask", err)
		return BriefSection{Title: sec.Title}, nil
	} else if ask != nil && ask.Text != "" {
		sec.Lines = append(sec.Lines, "Last ask: "+oneLine(ask.Text))
	}

	if sug, err := handoff.LatestSuggestion(store, opts.User, opts.RepoKey, opts.Domain); err != nil {
		logSkip("suggested next", err)
		return BriefSection{Title: sec.Title}, nil
	} else if sug != nil && sug.Text != "" {
		sec.Lines = append(sec.Lines, "Suggested next: "+oneLine(sug.Text))
	}

	if refs, err := handoff.RecentReflections(store, opts.User, opts.RepoKey, opts.Domain, 3); err != nil {
		logSkip("recent reflections", err)
		return BriefSection{Title: sec.Title}, nil
	} else {
		for _, r := range refs {
			if r.Text == "" {
				continue
			}
			sec.Lines = append(sec.Lines, "Recent: "+oneLine(r.Text))
		}
	}

	// All three absent → empty section, caller omits it.
	if len(sec.Lines) == 0 {
		return sec, nil
	}
	sec.Truncated = trimToBudget(&sec, budget)
	return sec, nil
}

// sprintSection surfaces the currently-active Sprint and its issues
// when one exists in the graph. Renders nothing for fresh repos or
// repos without tracker integration — the section only shows up when
// it has something useful to say.
func sprintSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Active sprint"}
	row := store.DB().QueryRow(
		`SELECT id, key,
		        COALESCE(json_extract(props, '$.name'), '') AS name,
		        COALESCE(json_extract(props, '$.goal'), '') AS goal,
		        COALESCE(json_extract(props, '$.start'), '') AS start,
		        COALESCE(json_extract(props, '$.end'), '') AS end_date
		   FROM nodes
		  WHERE type = 'Sprint' AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.state'), '') = 'active'
		  ORDER BY ingested_at DESC
		  LIMIT 1`,
	)
	var (
		sprintID                        int64
		key, name, goal, start, endDate string
	)
	if err := row.Scan(&sprintID, &key, &name, &goal, &start, &endDate); err != nil {
		// No active sprint — empty section, caller skips.
		return sec, nil
	}

	header := name
	if header == "" {
		header = key
	}
	if start != "" || endDate != "" {
		header = fmt.Sprintf("%s (%s → %s)", header, start, endDate)
	}
	sec.Lines = append(sec.Lines, "**"+header+"**")
	if goal != "" {
		sec.Lines = append(sec.Lines, "_Goal: "+oneLine(goal)+"_")
	}

	// Issues belonging to this sprint, grouped by status.
	rows, err := store.DB().Query(
		`SELECT i.key,
		        COALESCE(json_extract(i.props, '$.title'), i.key) AS title,
		        COALESCE(json_extract(i.props, '$.status'), '') AS status,
		        COALESCE(json_extract(i.props, '$.assignee'), '') AS assignee
		   FROM nodes i
		   JOIN edges e ON e.from_id = i.id AND e.type = 'belongs_to' AND e.valid_to IS NULL
		  WHERE i.type = 'Issue' AND i.valid_to IS NULL
		    AND e.to_id = ?
		  ORDER BY status, i.key
		  LIMIT 30`,
		sprintID,
	)
	if err != nil {
		return sec, fmt.Errorf("sprint issues: %w", err)
	}
	defer rows.Close()
	type issueRow struct{ key, title, status, assignee string }
	var rowsBuf []issueRow
	for rows.Next() {
		var r issueRow
		if err := rows.Scan(&r.key, &r.title, &r.status, &r.assignee); err != nil {
			return sec, err
		}
		rowsBuf = append(rowsBuf, r)
	}
	for _, r := range rowsBuf {
		line := fmt.Sprintf("- `%s` %s — _%s_", r.key, oneLine(r.title), r.status)
		if opts.AuthorEmail != "" && strings.EqualFold(r.assignee, opts.AuthorEmail) {
			line += " ← yours"
		}
		sec.Lines = append(sec.Lines, line)
	}
	sec.Truncated = trimToBudget(&sec, budget)
	return sec, nil
}

func inFlightSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "In flight"}
	rows, err := store.DB().Query(
		`SELECT key,
		        COALESCE(json_extract(props, '$.title'), key)        AS title,
		        COALESCE(json_extract(props, '$.status'), '')        AS status,
		        COALESCE(json_extract(props, '$.priority'), 'P9')    AS priority,
		        COALESCE(json_extract(props, '$.claimed_by'), '')    AS claimed_by,
		        ingested_at
		   FROM nodes
		  WHERE type = 'Feature' AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') NOT IN ('completed','superseded')
		  ORDER BY priority, ingested_at DESC
		  LIMIT 30`,
		opts.RepoKey,
	)
	if err != nil {
		return sec, err
	}
	defer rows.Close()

	type cand struct {
		key, title, status, priority, claimedBy string
		ingested                                time.Time
		score                                   float64
	}
	var cands []cand
	for rows.Next() {
		var c cand
		var ingested string
		if err := rows.Scan(&c.key, &c.title, &c.status, &c.priority, &c.claimedBy, &ingested); err != nil {
			return sec, err
		}
		c.ingested, _ = time.Parse(time.RFC3339, ingested)
		focus := 0.0
		if opts.AuthorEmail != "" && strings.EqualFold(c.claimedBy, opts.AuthorEmail) {
			focus = 2.0 // claimed by you: huge boost
		}
		c.score = score(c.ingested, focus, 1.0, opts.Now)
		cands = append(cands, c)
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })

	for _, c := range cands {
		line := fmt.Sprintf("- **%s** (`%s`, %s, %s)", c.title, c.key, c.priority, c.status)
		if c.claimedBy != "" {
			line += " — " + c.claimedBy
		}
		sec.Lines = append(sec.Lines, line)
	}
	sec.Truncated = trimToBudget(&sec, budget)
	return sec, nil
}

func justChangedSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Just changed"}
	rows, err := store.DB().Query(
		`SELECT json_extract(props, '$.sha'),
		        json_extract(props, '$.subject'),
		        json_extract(props, '$.author_name'),
		        json_extract(props, '$.date')
		   FROM nodes
		  WHERE type = 'Commit' AND repo = ? AND valid_to IS NULL
		  ORDER BY json_extract(props, '$.date') DESC
		  LIMIT 30`,
		opts.RepoKey,
	)
	if err != nil {
		return sec, err
	}
	defer rows.Close()

	for rows.Next() {
		var sha, subj, author, date sql.NullString
		if err := rows.Scan(&sha, &subj, &author, &date); err != nil {
			return sec, err
		}
		shortSha := sha.String
		if len(shortSha) > 7 {
			shortSha = shortSha[:7]
		}
		line := fmt.Sprintf("- `%s` %s", shortSha, oneLine(subj.String))
		if author.String != "" {
			line += "  _(" + author.String + ")_"
		}
		sec.Lines = append(sec.Lines, line)
	}
	sec.Truncated = trimToBudget(&sec, budget)
	return sec, nil
}

func triedSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Tried and failed (skip these dead ends)"}

	// All recent Attempts in this repo, regardless of session — agents
	// from earlier sessions/devs leaving dead-end signal is the whole
	// value here.
	rows, err := store.DB().Query(
		`SELECT json_extract(props, '$.body'), ingested_at
		   FROM nodes
		  WHERE type = 'Attempt' AND repo = ? AND valid_to IS NULL
		  ORDER BY ingested_at DESC
		  LIMIT 50`,
		opts.RepoKey,
	)
	if err != nil {
		return sec, err
	}
	defer rows.Close()

	type cand struct {
		body  string
		score float64
	}
	var cands []cand
	for rows.Next() {
		var body sql.NullString
		var ingested string
		if err := rows.Scan(&body, &ingested); err != nil {
			return sec, err
		}
		if !body.Valid || body.String == "" {
			continue
		}
		t, _ := time.Parse(time.RFC3339, ingested)
		cands = append(cands, cand{
			body:  body.String,
			score: score(t, 0, 1.5, opts.Now), // signal weight 1.5 — high
		})
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	for _, c := range cands {
		sec.Lines = append(sec.Lines, "- "+oneLine(c.body))
	}
	sec.Truncated = trimToBudget(&sec, budget)
	if len(sec.Lines) == 0 {
		sec.Lines = []string{"_no recorded dead ends_"}
		sec.Tokens = approxTokens(sec.Lines)
	}
	return sec, nil
}

func blockedSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Blocked on"}
	rows, err := store.DB().Query(
		`SELECT f.key, b.key, COALESCE(json_extract(b.props, '$.status'), '')
		   FROM nodes f
		   JOIN edges e ON e.from_id = f.id AND e.type IN ('depends_on','blocks') AND e.valid_to IS NULL
		   JOIN nodes b ON e.to_id = b.id AND b.valid_to IS NULL
		  WHERE f.type = 'Feature' AND f.repo = ? AND f.valid_to IS NULL
		    AND COALESCE(json_extract(f.props, '$.status'), '') NOT IN ('completed','superseded')
		    AND COALESCE(json_extract(b.props, '$.status'), '') NOT IN ('completed','accepted')
		  ORDER BY f.key`,
		opts.RepoKey,
	)
	if err != nil {
		return sec, err
	}
	defer rows.Close()

	for rows.Next() {
		var fkey, bkey, bstatus string
		if err := rows.Scan(&fkey, &bkey, &bstatus); err != nil {
			return sec, err
		}
		sec.Lines = append(sec.Lines, fmt.Sprintf("- `%s` ← waiting on `%s` (%s)", fkey, bkey, bstatus))
	}
	sec.Truncated = trimToBudget(&sec, budget)
	if len(sec.Lines) == 0 {
		sec.Lines = []string{"_nothing blocked_"}
		sec.Tokens = approxTokens(sec.Lines)
	}
	return sec, nil
}

// missionSection renders the project charter as the highest-priority
// block in every brief. The mission statement is always shown; a
// compact list of principles follows when budget allows.
//
// This is the project-charter Phase 2 injection point — every command
// that calls digest.Generate gains the mission block automatically.
func missionSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Mission"}
	row := store.DB().QueryRow(
		`SELECT key,
		        COALESCE(json_extract(props, '$.title'), '') AS title,
		        COALESCE(json_extract(props, '$.version'), '') AS version,
		        COALESCE(json_extract(props, '$.mission_statement'), '') AS statement,
		        COALESCE(json_extract(props, '$.principles'), '') AS principles,
		        COALESCE(json_extract(props, '$.mission_fit_test'), '') AS mft
		   FROM nodes
		  WHERE type = 'Mission' AND repo = ? AND valid_to IS NULL
		  ORDER BY key
		  LIMIT 1`,
		opts.RepoKey,
	)
	var key, title, version, statement, principlesJSON, mft string
	if err := row.Scan(&key, &title, &version, &statement, &principlesJSON, &mft); err != nil {
		// Mission rows just don't exist yet for fresh repos. Render
		// nothing rather than erroring — caller filters empty sections.
		return sec, nil
	}
	if statement == "" && title == "" {
		return sec, nil
	}
	if title != "" {
		header := title
		if version != "" {
			header += " (v" + version + ")"
		}
		sec.Lines = append(sec.Lines, "_"+header+"_")
	}
	if statement != "" {
		sec.Lines = append(sec.Lines, summarizeStatement(statement, 280))
	}
	if principlesJSON != "" {
		// principles is stored as a JSON array of strings; render
		// inline as "Principles: A · B · C" to stay terse.
		var ps []string
		if err := json.Unmarshal([]byte(principlesJSON), &ps); err == nil && len(ps) > 0 {
			sec.Lines = append(sec.Lines, "Principles: "+strings.Join(ps, " · "))
		}
	}
	if mft != "" {
		sec.Lines = append(sec.Lines, "Mission-fit test: "+summarizeStatement(mft, 200))
	}
	enforceBudget(&sec, budget)
	return sec, nil
}

// acceptanceSection surfaces failing/regressed Criterion nodes
// (always relevant) and recently flipped Criterion nodes (signal of
// "what changed since last session"). Renders empty when no
// Criterion nodes exist for the repo.
func acceptanceSection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Acceptance criteria"}

	// Recently flipped: valid_from within the last 7 days. Picks up
	// all status flips a fresh `hero ac record` produced.
	since := opts.Now.Add(-7 * 24 * time.Hour).Format(time.RFC3339)

	failingRows, err := store.DB().Query(
		`SELECT key,
		        COALESCE(json_extract(props, '$.status'), '') AS status,
		        COALESCE(json_extract(props, '$.statement'), '') AS statement
		   FROM nodes
		  WHERE type = 'Criterion' AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') IN ('failing','regressed')
		  ORDER BY key
		  LIMIT 12`,
		opts.RepoKey,
	)
	if err != nil {
		return sec, err
	}
	defer failingRows.Close()
	for failingRows.Next() {
		var key, status, statement string
		if err := failingRows.Scan(&key, &status, &statement); err != nil {
			return sec, err
		}
		sec.Lines = append(sec.Lines, fmt.Sprintf("- ❌ `%s` (%s) — %s", key, status, summarizeStatement(statement, 90)))
	}

	flipsRows, err := store.DB().Query(
		`SELECT key,
		        COALESCE(json_extract(props, '$.status'), '') AS status
		   FROM nodes
		  WHERE type = 'Criterion' AND repo = ? AND valid_to IS NULL
		    AND valid_from >= ?
		    AND COALESCE(json_extract(props, '$.status'), '') = 'passing'
		  ORDER BY valid_from DESC
		  LIMIT 8`,
		opts.RepoKey, since,
	)
	if err != nil {
		return sec, err
	}
	defer flipsRows.Close()
	var flips []string
	for flipsRows.Next() {
		var key, status string
		if err := flipsRows.Scan(&key, &status); err != nil {
			return sec, err
		}
		flips = append(flips, fmt.Sprintf("- ✅ `%s` flipped to passing", key))
	}
	sec.Lines = append(sec.Lines, flips...)

	sec.Truncated = trimToBudget(&sec, budget)
	if len(sec.Lines) == 0 {
		// Don't render the section at all if nothing to say.
		return sec, nil
	}
	return sec, nil
}

// summarizeStatement clips long AC statements to a single-line preview
// suitable for a brief block.
func summarizeStatement(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// nearbySection surfaces Decisions, Notes, and recent commits whose
// touched files overlap with opts.FocusFiles. Empty FocusFiles means
// we fall back to top recent Initiatives (phase 4 behavior).
func nearbySection(store *graph.Store, opts Options, budget int) (BriefSection, error) {
	sec := BriefSection{Title: "Nearby"}

	if len(opts.FocusFiles) > 0 {
		// For each focus file, find Symbol nodes defined in it and
		// recent Commits that touched it.
		seen := map[string]bool{}
		var lines []string
		for _, f := range opts.FocusFiles {
			fileKey := opts.RepoKey + ":" + filepath.ToSlash(f)
			fileID, err := store.GetNodeID("File", fileKey)
			if err != nil {
				continue
			}

			// Recent commits that touched this file (strong "what just changed nearby" signal).
			rows, err := store.DB().Query(
				`SELECT c.key,
				        json_extract(c.props, '$.subject'),
				        json_extract(c.props, '$.date')
				   FROM nodes c
				   JOIN edges e ON e.from_id = c.id AND e.type = 'touches' AND e.valid_to IS NULL
				  WHERE c.type = 'Commit' AND e.to_id = ? AND c.valid_to IS NULL
				  ORDER BY json_extract(c.props, '$.date') DESC
				  LIMIT 5`,
				fileID,
			)
			if err == nil {
				for rows.Next() {
					var sha string
					var subject, date sql.NullString
					if err := rows.Scan(&sha, &subject, &date); err != nil {
						continue
					}
					line := fmt.Sprintf("- _%s_: `%s` %s", filepath.Base(f), shortSha(sha), oneLine(subject.String))
					if !seen[line] {
						lines = append(lines, line)
						seen[line] = true
					}
				}
				rows.Close()
			}
		}
		sec.Lines = lines
	}

	// Always append top recent open Initiatives as background.
	rows, err := store.DB().Query(
		`SELECT COALESCE(json_extract(props, '$.title'), key), key
		   FROM nodes
		  WHERE type = 'Initiative' AND repo = ? AND valid_to IS NULL
		    AND COALESCE(json_extract(props, '$.status'), '') NOT IN ('completed','superseded')
		  ORDER BY ingested_at DESC
		  LIMIT 5`,
		opts.RepoKey,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var title, key string
			if err := rows.Scan(&title, &key); err == nil {
				sec.Lines = append(sec.Lines, fmt.Sprintf("- _initiative_: %s (`%s`)", title, key))
			}
		}
	}

	sec.Truncated = trimToBudget(&sec, budget)
	if len(sec.Lines) == 0 {
		sec.Lines = []string{"_no nearby context_"}
		sec.Tokens = approxTokens(sec.Lines)
	}
	return sec, nil
}

// --- budget enforcement ----------------------------------------------------

// approxTokens treats 4 chars ≈ 1 token. Cheap, good enough for ranking.
func approxTokens(lines []string) int {
	total := 0
	for _, l := range lines {
		total += (len(l) + 3) / 4
	}
	return total
}

// trimToBudget drops lowest-priority items (last in list) past the
// section's soft target. Once the section is at-or-under target it
// stops trimming — items past the target are kept if they're already
// included (the assumption: every line that made it into the section
// was scored above the relevance floor by the upstream query).
//
// A hard cap of 4× the soft target prevents pathological growth on a
// single noisy section. Returns the count dropped.
func trimToBudget(sec *BriefSection, softTarget int) int {
	if softTarget <= 0 {
		sec.Tokens = approxTokens(sec.Lines)
		return 0
	}
	hardCap := softTarget * 4
	dropped := 0
	// Hard cap: always enforce.
	for approxTokens(sec.Lines) > hardCap && len(sec.Lines) > 0 {
		sec.Lines = sec.Lines[:len(sec.Lines)-1]
		dropped++
	}
	// Soft target: trim only if there's clearly excess (>2× target).
	// Below 2× we trust the candidate ranking and let it stand.
	for approxTokens(sec.Lines) > 2*softTarget && len(sec.Lines) > 0 {
		sec.Lines = sec.Lines[:len(sec.Lines)-1]
		dropped++
	}
	sec.Tokens = approxTokens(sec.Lines)
	return dropped
}

// enforceBudget for tiny sections that must always fit (whoYouAre).
// Truncates the last line if needed.
func enforceBudget(sec *BriefSection, budget int) {
	if budget <= 0 {
		sec.Tokens = approxTokens(sec.Lines)
		return
	}
	for approxTokens(sec.Lines) > budget && len(sec.Lines) > 0 {
		last := sec.Lines[len(sec.Lines)-1]
		if len(last) > 32 {
			sec.Lines[len(sec.Lines)-1] = last[:len(last)-16] + "…"
		} else {
			sec.Lines = sec.Lines[:len(sec.Lines)-1]
		}
	}
	sec.Tokens = approxTokens(sec.Lines)
}

// --- markdown render -------------------------------------------------------

// Markdown renders a Brief as the primary surface delivered to LLMs.
// Compact, header-stable, predictable byte-for-byte for the same graph
// state — which makes prompt caching effective.
func (b *Brief) Markdown() string {
	var s strings.Builder
	fmt.Fprintf(&s, "<!-- hero brief: repo=%s budget=%d/%d generated=%s -->\n",
		b.RepoKey, b.BudgetUsed, b.BudgetTotal, b.Generated)
	for _, sec := range b.Sections {
		fmt.Fprintf(&s, "## %s\n\n", sec.Title)
		if len(sec.Lines) == 0 {
			s.WriteString("_(none)_\n\n")
			continue
		}
		for _, line := range sec.Lines {
			s.WriteString(line)
			if !strings.HasSuffix(line, "\n") {
				s.WriteString("\n")
			}
		}
		if sec.Truncated > 0 {
			// Tell the model where to dig deeper. Natural-language
			// routing — no need to know specific tool names.
			topic := sectionRecallTopic(sec.Title)
			fmt.Fprintf(&s, "_…+%d more — `hero recall %s` to dig deeper_\n", sec.Truncated, topic)
		}
		s.WriteString("\n")
	}
	return s.String()
}

// JSON returns the Brief serialized for tool consumers.
func (b *Brief) JSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

// --- helpers ---------------------------------------------------------------

func oneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return s
}

func shortSha(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

// sectionRecallTopic returns a short keyword to suggest in the
// "dig deeper" hint for a given section.
func sectionRecallTopic(title string) string {
	t := strings.ToLower(title)
	switch {
	case strings.Contains(t, "tried"):
		return "<topic>"
	case strings.Contains(t, "blocked"):
		return "<feature>"
	case strings.Contains(t, "flight"):
		return "<feature-or-area>"
	case strings.Contains(t, "changed"):
		return "<file-or-topic>"
	case strings.Contains(t, "nearby"):
		return "<file-or-topic>"
	default:
		return "<topic>"
	}
}
