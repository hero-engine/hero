package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/gitutil"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/handoff"
	"github.com/hero-engine/hero/internal/projection"
	"github.com/hero-engine/hero/internal/sessions"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

// Field-grab CLI for the per-user durable handoff state. Each command
// runs in two modes:
//
//   - With no args: read the field and print to stdout.
//   - With text args: record a new value (joined with spaces).
//
// Reading goes directly to the graph — always current even if the
// projection hasn't fired on this machine yet. Writing goes through
// internal/handoff which handles supersession (singletons) or
// accumulation (reflections).

var (
	handoffJSON bool
	handoffCopy bool
	handoffUser string
	ingestQuiet bool
)

var nextSuggestCmd = &cobra.Command{
	Use:   "suggest [text]",
	Short: "Show or set the suggested next prompt",
	Long: `Without args, prints the current suggested-next-prompt for you
(or --user <slug>). With args, records a new suggestion that
supersedes the prior one for this user.

Examples:
  hero next suggest                       # print current
  hero next suggest --copy                # print + copy to clipboard
  hero next suggest --json                # JSON for scripting
  hero next suggest --user alice          # fetch alice's suggestion
  hero next suggest "let's tackle phase 4 of next-as-projection"`,
	RunE: runNextSuggest,
}

var nextAskCmd = &cobra.Command{
	Use:   "ask [text]",
	Short: "Show or set the last user ask",
	RunE:  runNextAsk,
}

var nextGoalCmd = &cobra.Command{
	Use:   "goal [text]",
	Short: "Show or set the session goal (durable intent)",
	Long: `Without args, prints the current session goal for you (or
--user <slug>) — the durable intent behind the work, captured
automatically from the session's opening messages. With args, records a
MANUAL goal that supersedes any auto-derived or marker goal.

The goal is captured automatically every checkpoint, so this override is
optional — fire it only to correct a wrong auto-derived opener. Use the
last-ask command (` + "`hero next ask`" + `) for the latest prompt; the
goal is the session's WHY, not the most recent refinement.

Examples:
  hero next goal                                  # print current goal
  hero next goal "add rate limiting to the login endpoint"`,
	RunE: runNextGoal,
}

var nextReflectionCmd = &cobra.Command{
	Use:   "reflection [text]",
	Short: "Show recent session reflections, or record a new one",
	RunE:  runNextReflection,
}

var nextIngestCmd = &cobra.Command{
	Use:   "ingest [path]",
	Short: "Read .hero/next/<user>.md back into the local graph (cross-machine continuity)",
	Long: `Round-trip ingest: parses a per-user handoff markdown file
and re-creates its UserAsk / NextSuggestion / SessionReflection
nodes in the local graph if not already present.

This is the load-bearing trick that makes solo-no-Cloud cross-
machine continuity work. Sequence: home laptop projects user-graph
→ .hero/next/<user>.md, commits, pushes; office desktop pulls,
runs 'hero next ingest', queries return the same suggestion.

With no path argument, ingests every .hero/next/*.md (excluding
*.local.md) so a session-start hook can rebuild user-graph state
for all known users in one call.`,
	RunE: runNextIngest,
}

func init() {
	for _, cmd := range []*cobra.Command{nextSuggestCmd, nextAskCmd, nextGoalCmd, nextReflectionCmd} {
		cmd.Flags().BoolVar(&handoffJSON, "json", false, "emit JSON instead of plain text")
		cmd.Flags().BoolVar(&handoffCopy, "copy", false, "also copy result to clipboard")
		cmd.Flags().StringVar(&handoffUser, "user", "", "fetch another user's value (defaults to you)")
	}
	nextIngestCmd.Flags().BoolVarP(&ingestQuiet, "quiet", "q", false, "suppress per-file ingest output (intended for hook invocations)")
}

func runNextIngest(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	store, err := graph.Open(heroDir)
	if err != nil {
		return fmt.Errorf("opening graph: %w", err)
	}
	defer store.Close()
	repoKey := gitutil.RepoKey(projectRoot)

	// Cold-clone rebuild (resume-brief-missing-project-context, Change
	// 2a): graph.db is a gitignored, rebuildable local cache, so a fresh
	// clone lands with an EMPTY project graph — the handoff ingest below
	// only rehydrates handoff singletons, never the project-context nodes
	// the resume brief reads (Commit / Feature / Bug). On a cold graph,
	// rebuild project context from the clone's LOCAL authoritative sources
	// (git log + committed specs) so the first resume isn't an empty map.
	// Runs BEFORE the handoff-file loop so it fires even when a cold clone
	// has no per-user handoff files yet (the loop early-returns on an empty
	// path set). Gated on an empty/near-empty graph so warm machines never
	// pay the reingest cost. Best-effort: a rebuild error warns and is
	// swallowed — it must never fail SessionStart ingest.
	rebuildProjectContextIfCold(cfg, projectRoot, heroDir, store, repoKey)

	var paths []string
	if len(args) > 0 {
		paths = args
	} else {
		// Walk .hero/next/, take *.md but skip *.local.md.
		entries, _ := os.ReadDir(filepath.Join(heroDir, "next"))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}
			if strings.HasSuffix(name, ".local.md") {
				continue
			}
			paths = append(paths, filepath.Join(heroDir, "next", name))
		}
	}

	if len(paths) == 0 {
		if !ingestQuiet {
			fmt.Fprintln(cmd.OutOrStdout(), "(no per-user handoff files to ingest)")
		}
		return nil
	}

	domain := graph.DomainFor(cfg, graph.IntrinsicActive)
	// localSlug is the identity the local reader (hero resume,
	// hero next ask/suggest) derives. Threaded into ingest so the
	// rehydrated singletons are mirrored under it when the file's
	// recorded user differs — closing the cross-machine slug-divergence
	// gap (see internal/handoff.IngestUserFile).
	localSlug := nextUserSlug(cfg)
	// singleTravelFile gates the cross-machine alias mirror: it may only
	// fire when exactly ONE distinct travel-eligible identity exists on
	// disk, so a brand-new teammate (empty graph, zero handoff nodes)
	// never has another user's handoff mirrored onto their slug. We count
	// distinct frontmatter `user:` values across .hero/next/*.md (the same
	// authoritative count resolveHandoffIdentity reads), not len(paths),
	// so an explicit-args invocation can't widen the gate.
	singleTravelFile := len(nextFileUserSlugs(heroDir)) == 1
	for _, p := range paths {
		if err := handoff.IngestUserFile(store, repoKey, domain, p, localSlug, singleTravelFile); err != nil {
			return fmt.Errorf("ingest %s: %w", p, err)
		}
		if !ingestQuiet {
			fmt.Fprintf(cmd.OutOrStdout(), "ingested %s\n", filepath.Base(p))
		}
	}
	return nil
}

// rebuildProjectContextIfCold rebuilds the project-context subgraph
// (specs + sessions + recent commits) from local authoritative sources
// when the graph has no project-context nodes for this repo — the
// cold-clone case. It is a no-op on a warm graph (the common per-session
// path) so it only pays the cost once, on a fresh clone.
//
// Repo partition uses gitutil.RepoKey(projectRoot) — the SAME key the
// digest's justChangedSection/in-flight readers filter on — so the
// rebuilt nodes are visible to `hero resume`. (reingestWork keys on
// filepath.Base, which can diverge from RepoKey when an origin remote is
// set; we deliberately do not reuse it here for that reason.)
//
// Best-effort by contract: every error warns to stderr and returns. It
// must never fail the SessionStart ingest path.
func rebuildProjectContextIfCold(cfg config.Config, projectRoot, heroDir string, store *graph.Store, repoKey string) {
	if !projectGraphCold(store, repoKey) {
		return
	}

	domain := graph.DomainFor(cfg, graph.IntrinsicActive)

	specs, err := spec.Discover(heroDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cold-clone rebuild: discovering specs: %v\n", err)
	} else if _, err := spec.WriteGraph(specs, repoKey, domain, store); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cold-clone rebuild: writing spec subgraph: %v\n", err)
	}

	sessList, err := sessions.List(heroDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cold-clone rebuild: listing sessions: %v\n", err)
	} else if _, err := sessions.WriteGraph(sessList, repoKey, store); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cold-clone rebuild: writing sessions subgraph: %v\n", err)
	}

	if _, err := gitutil.WriteGitLogGraph(projectRoot, repoKey, 0, store); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cold-clone rebuild: writing git-log subgraph: %v\n", err)
	}
}

// projectGraphCold reports whether the repo's project-context subgraph
// is empty/near-empty — no live Commit, Feature, or Bug nodes for this
// repo. These are the node types the resume brief's project-context
// sections read; if none exist, the graph is a fresh clone's cold cache
// (or a brand-new workspace) and warrants a rebuild. Errors are treated
// as "not cold" so a probe failure never triggers an unnecessary
// reingest.
func projectGraphCold(store *graph.Store, repoKey string) bool {
	var n int
	err := store.DB().QueryRow(
		`SELECT COUNT(*) FROM nodes
		  WHERE repo = ? AND valid_to IS NULL
		    AND type IN ('Commit','Feature','Bug')`,
		repoKey,
	).Scan(&n)
	if err != nil {
		return false
	}
	return n == 0
}

// resolveHandoffUser returns the requested user or the current user.
func resolveHandoffUser(cfg config.Config) string {
	if handoffUser != "" {
		return handoffUser
	}
	return nextUserSlug(cfg)
}

// openHandoffStore is the boilerplate every handoff command needs:
// load config, open the graph, derive repoKey + user.
func openHandoffStore() (*graph.Store, string, string, string, func(), error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, "", "", "", nil, fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil, "", "", "", nil, fmt.Errorf("opening graph: %w", err)
	}
	user := resolveHandoffUser(cfg)
	repoKey := gitutil.RepoKey(projectRoot)
	domain := graph.DomainFor(cfg, graph.IntrinsicActive)
	return store, user, repoKey, domain, func() { store.Close() }, nil
}

func runNextSuggest(cmd *cobra.Command, args []string) error {
	store, user, repoKey, domain, cleanup, err := openHandoffStore()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		text := strings.Join(args, " ")
		return handoff.RecordSuggestion(store, repoKey, handoff.NextSuggestion{
			User:   user,
			Domain: domain,
			Text:   text,
		})
	}
	// Read path: ask the projection for the staleness-aware answer
	// so this command always agrees with .hero/next/<user>.md and
	// the user can never see a suggestion superseded by commits.
	text, rationale, source := projection.PickUserSuggestion(store, user, repoKey, domain)
	if text == "" {
		return emitField(cmd.OutOrStdout(), (*handoff.NextSuggestion)(nil), "no suggested next prompt and no open feature to derive from")
	}
	derived := &handoff.NextSuggestion{
		User:      user,
		Text:      text,
		Rationale: rationale,
	}
	if err := emitField(cmd.OutOrStdout(), derived, "no suggested next prompt yet"); err != nil {
		return err
	}
	if !handoffJSON && source != projection.SuggestionFromAgent {
		fmt.Fprintf(cmd.OutOrStdout(), "(source: %s)\n", source)
	}
	return nil
}

func runNextAsk(cmd *cobra.Command, args []string) error {
	store, user, repoKey, domain, cleanup, err := openHandoffStore()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		text := strings.Join(args, " ")
		return handoff.RecordAsk(store, repoKey, handoff.UserAsk{
			User:   user,
			Domain: domain,
			Text:   text,
		})
	}
	ask, err := handoff.LatestAsk(store, user, repoKey, domain)
	if err != nil {
		return err
	}
	return emitField(cmd.OutOrStdout(), ask, "no recorded user ask yet")
}

func runNextGoal(cmd *cobra.Command, args []string) error {
	store, user, repoKey, domain, cleanup, err := openHandoffStore()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		text := strings.Join(args, " ")
		return handoff.RecordGoal(store, repoKey, handoff.SessionGoal{
			User:   user,
			Domain: domain,
			Text:   text,
			Source: handoff.GoalSourceManual,
		})
	}
	goal, err := handoff.LatestGoal(store, user, repoKey, domain)
	if err != nil {
		return err
	}
	return emitField(cmd.OutOrStdout(), goal, "no session goal recorded yet")
}

func runNextReflection(cmd *cobra.Command, args []string) error {
	store, user, repoKey, domain, cleanup, err := openHandoffStore()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		text := strings.Join(args, " ")
		return handoff.RecordReflection(store, repoKey, handoff.SessionReflection{
			User:   user,
			Domain: domain,
			Text:   text,
		})
	}
	refs, err := handoff.RecentReflections(store, user, repoKey, domain, 5)
	if err != nil {
		return err
	}
	if handoffJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(refs)
	}
	if len(refs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "(no recent reflections)")
		return nil
	}
	for _, r := range refs {
		fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", r.Text)
	}
	if handoffCopy {
		return copyToClipboard(joinReflections(refs))
	}
	return nil
}

// emitField prints a singleton handoff value (UserAsk or
// NextSuggestion) according to the active flags. Pass nil for a
// missing value.
func emitField(out io.Writer, val any, missing string) error {
	if handoffJSON {
		if val == nil || isNilPointer(val) {
			fmt.Fprintln(out, "null")
			return nil
		}
		return json.NewEncoder(out).Encode(val)
	}

	text := fieldText(val)
	if text == "" {
		fmt.Fprintln(out, "("+missing+")")
		return nil
	}
	fmt.Fprintln(out, text)
	if handoffCopy {
		return copyToClipboard(text)
	}
	return nil
}

func fieldText(val any) string {
	switch v := val.(type) {
	case *handoff.UserAsk:
		if v == nil {
			return ""
		}
		return v.Text
	case *handoff.NextSuggestion:
		if v == nil {
			return ""
		}
		return v.Text
	case *handoff.SessionGoal:
		if v == nil {
			return ""
		}
		return v.Text
	}
	return ""
}

func isNilPointer(val any) bool {
	switch v := val.(type) {
	case *handoff.UserAsk:
		return v == nil
	case *handoff.NextSuggestion:
		return v == nil
	case *handoff.SessionGoal:
		return v == nil
	}
	return val == nil
}

func joinReflections(refs []handoff.SessionReflection) string {
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		parts = append(parts, r.Text)
	}
	return strings.Join(parts, "\n")
}

// copyToClipboard tries the platform-native clipboard tool. Best-
// effort: returns nil even if no tool is available so --copy never
// fails the command — it just becomes a no-op when we can't.
func copyToClipboard(text string) error {
	tool, args := clipboardTool()
	if tool == "" {
		return nil
	}
	cmd := exec.Command(tool, args...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		// Soft-fail — don't error out a successful read.
		fmt.Fprintf(os.Stderr, "warning: clipboard copy failed (%v)\n", err)
	}
	return nil
}

// clipboardTool returns the command + args appropriate for this
// platform. Empty string means we don't know how to copy here.
func clipboardTool() (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", nil
	case "linux":
		// Prefer wl-copy on Wayland, fall back to xclip.
		if path, err := exec.LookPath("wl-copy"); err == nil {
			return path, nil
		}
		if path, err := exec.LookPath("xclip"); err == nil {
			return path, []string{"-selection", "clipboard"}
		}
	case "windows":
		return "clip.exe", nil
	}
	return "", nil
}
