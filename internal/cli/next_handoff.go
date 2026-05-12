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
	handoffJSON  bool
	handoffCopy  bool
	handoffUser  string
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
	for _, cmd := range []*cobra.Command{nextSuggestCmd, nextAskCmd, nextReflectionCmd} {
		cmd.Flags().BoolVar(&handoffJSON, "json", false, "emit JSON instead of plain text")
		cmd.Flags().BoolVar(&handoffCopy, "copy", false, "also copy result to clipboard")
		cmd.Flags().StringVar(&handoffUser, "user", "", "fetch another user's value (defaults to you)")
	}
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
		fmt.Fprintln(cmd.OutOrStdout(), "(no per-user handoff files to ingest)")
		return nil
	}

	for _, p := range paths {
		if err := handoff.IngestUserFile(store, repoKey, p); err != nil {
			return fmt.Errorf("ingest %s: %w", p, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "ingested %s\n", filepath.Base(p))
	}
	return nil
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
func openHandoffStore() (*graph.Store, string, string, func(), error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	store, err := graph.Open(heroDir)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("opening graph: %w", err)
	}
	user := resolveHandoffUser(cfg)
	repoKey := gitutil.RepoKey(projectRoot)
	return store, user, repoKey, func() { store.Close() }, nil
}

func runNextSuggest(cmd *cobra.Command, args []string) error {
	store, user, repoKey, cleanup, err := openHandoffStore()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		text := strings.Join(args, " ")
		return handoff.RecordSuggestion(store, repoKey, handoff.NextSuggestion{
			User: user,
			Text: text,
		})
	}
	// Read path: ask the projection for the staleness-aware answer
	// so this command always agrees with .hero/next/<user>.md and
	// the user can never see a suggestion superseded by commits.
	text, rationale, source := projection.PickUserSuggestion(store, user, repoKey)
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
	store, user, repoKey, cleanup, err := openHandoffStore()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		text := strings.Join(args, " ")
		return handoff.RecordAsk(store, repoKey, handoff.UserAsk{
			User: user,
			Text: text,
		})
	}
	ask, err := handoff.LatestAsk(store, user)
	if err != nil {
		return err
	}
	return emitField(cmd.OutOrStdout(), ask, "no recorded user ask yet")
}

func runNextReflection(cmd *cobra.Command, args []string) error {
	store, user, repoKey, cleanup, err := openHandoffStore()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		text := strings.Join(args, " ")
		return handoff.RecordReflection(store, repoKey, handoff.SessionReflection{
			User: user,
			Text: text,
		})
	}
	refs, err := handoff.RecentReflections(store, user, 5)
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
	}
	return ""
}

func isNilPointer(val any) bool {
	switch v := val.(type) {
	case *handoff.UserAsk:
		return v == nil
	case *handoff.NextSuggestion:
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
