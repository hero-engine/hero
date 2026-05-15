package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/peering"
	"github.com/hero-engine/hero/internal/spec"
)

// `hero handoff` — cross-repo async drop. The companion to the spec
// designed in .hero/planning/features/cross-repo-peering/spec.md
// (Phase 1). The default verb is the drop itself; `status` and
// `accept` are subcommands.

var (
	handoffTitle  string
	handoffType   string
	handoffReason string
)

var handoffCmd = &cobra.Command{
	Use:   "handoff <slug> <peer-alias>",
	Short: "Hand off a local spec to a peer workspace (async drop)",
	Long: `Async-drop a local spec onto a peer workspace. The receiving workspace
gets a scaffolded spec stamped with provenance (` + "`received_from`" + `), and
this side's spec status moves to handed_off with a Handoff Trail entry.

Examples:
  hero handoff order-failure-error-display app --reason "Backend job"
  hero handoff csv-export web --title "CSV export client wiring"
  hero handoff foo app --type bug

Subcommands:
  hero handoff status [<slug>]    Show handoff state (one spec or all)
  hero handoff accept <slug>      Pick up a handed_back spec`,
	Args: cobra.ExactArgs(2),
	RunE: runHandoff,
}

var handoffStatusCmd = &cobra.Command{
	Use:   "status [<slug>]",
	Short: "Show handoff state for one spec or all in-flight handoffs",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHandoffStatus,
}

var handoffAcceptCmd = &cobra.Command{
	Use:   "accept <slug>",
	Short: "Pick up a handed_back spec — move its status to delivering or in-review",
	Args:  cobra.ExactArgs(1),
	RunE:  runHandoffAccept,
}

func init() {
	handoffCmd.Flags().StringVar(&handoffTitle, "title", "", "override the receiver-side spec title")
	handoffCmd.Flags().StringVar(&handoffType, "type", "", "override the receiver-side spec type (feature, bug, …)")
	handoffCmd.Flags().StringVar(&handoffReason, "reason", "", "rationale for the handoff (recorded in trail + receiver provenance)")

	handoffCmd.AddCommand(handoffStatusCmd)
	handoffCmd.AddCommand(handoffAcceptCmd)
}

func runHandoff(cmd *cobra.Command, args []string) error {
	slug, peerAlias := args[0], args[1]
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if _, ok := cfg.Repos[peerAlias]; !ok {
		var aliases []string
		for a := range cfg.Repos {
			aliases = append(aliases, a)
		}
		sort.Strings(aliases)
		if len(aliases) == 0 {
			return fmt.Errorf("peer %q is not configured — register one with `hero repos add <alias> <path>`", peerAlias)
		}
		return fmt.Errorf("peer %q is not configured — configured peers: %s", peerAlias, strings.Join(aliases, ", "))
	}

	res, err := peering.Handoff(projectRoot, slug, peering.HandoffOptions{
		PeerAlias: peerAlias,
		Title:     handoffTitle,
		Type:      handoffType,
		Reason:    handoffReason,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(),
		"Handed off %s to %s (peer_id %s)\n  peer slug: %s\n  peer path: %s\n",
		slug, peerAlias, res.PeerID, res.PeerSlug, res.PeerPath)
	if res.PeerSlug != slug {
		fmt.Fprintf(cmd.OutOrStdout(),
			"  note: slug collided on peer side — used %q instead\n", res.PeerSlug)
	}
	return nil
}

func runHandoffStatus(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	if len(args) == 1 {
		slug := args[0]
		for _, s := range specs {
			if s.Slug == slug {
				return renderHandoffStatusOne(cmd, s)
			}
		}
		return fmt.Errorf("no spec with slug %q", slug)
	}

	// All in-flight handoffs.
	any := false
	w := cmd.OutOrStdout()
	for _, s := range specs {
		if s.Status == spec.StatusHandedOff || s.Status == spec.StatusAwaitingPeer ||
			s.Status == spec.StatusHandedBack || s.ReceivedFrom != nil {
			renderHandoffStatusOne(cmd, s)
			fmt.Fprintln(w)
			any = true
		}
	}
	if !any {
		fmt.Fprintln(w, "No in-flight handoffs.")
	}
	return nil
}

func renderHandoffStatusOne(cmd *cobra.Command, s *spec.Spec) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s  [%s]\n", s.Slug, string(s.Status))
	if s.Title != "" {
		fmt.Fprintf(w, "  title: %s\n", s.Title)
	}
	if s.ReceivedFrom != nil {
		fmt.Fprintf(w, "  received_from: peer_id=%s originator_slug=%s alias=%s\n",
			s.ReceivedFrom.PeerID, s.ReceivedFrom.OriginatorSlug, s.ReceivedFrom.PeerAliasDisplay)
		if s.ReceivedFrom.Reason != "" {
			fmt.Fprintf(w, "  reason: %s\n", s.ReceivedFrom.Reason)
		}
	}
	entries, err := peering.ReadTrail(s.Path)
	if err != nil {
		return fmt.Errorf("read trail: %w", err)
	}
	if len(entries) == 0 {
		fmt.Fprintln(w, "  trail: (empty)")
		return nil
	}
	fmt.Fprintln(w, "  trail:")
	for _, e := range entries {
		ts := e.At.UTC().Format(time.RFC3339)
		fmt.Fprintf(w, "    %s  %s %s  %s/%s  mode=%s\n",
			ts, e.Direction, e.PeerAliasDisplay, e.PeerAliasDisplay, peerSpecSlug(e), e.Mode)
		if e.Reason != "" {
			fmt.Fprintf(w, "      reason: %s\n", e.Reason)
		}
		if e.ResultRef != "" {
			fmt.Fprintf(w, "      result_ref: %s\n", e.ResultRef)
		}
	}
	return nil
}

func peerSpecSlug(e contractpeering.TrailEntry) string {
	if e.PeerSpec == "" {
		return "?"
	}
	parts := strings.SplitN(e.PeerSpec, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return e.PeerSpec
}

func runHandoffAccept(cmd *cobra.Command, args []string) error {
	slug := args[0]
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}
	var target *spec.Spec
	for _, s := range specs {
		if s.Slug == slug {
			target = s
			break
		}
	}
	if target == nil {
		return fmt.Errorf("no spec with slug %q", slug)
	}
	if target.Status != spec.StatusHandedBack {
		return fmt.Errorf("spec %q is in status %q — only handed_back specs can be accepted", slug, target.Status)
	}

	next, err := promptNextStatus(cmd)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(target.Path)
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "status", string(next))

	// Append a trail entry recording the accept transition. We don't
	// have a fresh peer event here — the originator is acting alone.
	entry := contractpeering.TrailEntry{
		At:        time.Now().UTC(),
		Direction: contractpeering.DirectionOut,
		Mode:      contractpeering.ModeAccepted,
		Reason:    fmt.Sprintf("accepted from handed_back → %s", next),
	}
	updated = peering.AppendTrailToContent(updated, entry)

	if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write spec: %w", err)
	}

	// Emit a peer.handoff.accepted event for parity with the other
	// handoff transitions.
	_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
		Timestamp: time.Now().UTC(),
		Type:      string(contractpeering.EventHandoffAccepted),
		Agent:     "hero",
		Slug:      slug,
		Message:   fmt.Sprintf("handed_back → %s", next),
	})

	fmt.Fprintf(cmd.OutOrStdout(), "Accepted %s: status %s → %s\n", slug, spec.StatusHandedBack, next)
	return nil
}

// promptNextStatus asks the user whether the accepted spec should
// become `delivering` (default) or `in-review`. Reads from stdin —
// when non-interactive (e.g. piped) defaults to delivering.
func promptNextStatus(cmd *cobra.Command) (spec.Status, error) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "Pick the next status for this spec:")
	fmt.Fprintln(w, "  1) delivering (default)")
	fmt.Fprintln(w, "  2) in-review")
	fmt.Fprint(w, "> ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return spec.StatusDelivering, nil
	}
	choice := strings.TrimSpace(scanner.Text())
	switch choice {
	case "", "1", "delivering", "d":
		return spec.StatusDelivering, nil
	case "2", "in-review", "review", "r":
		return spec.StatusInReview, nil
	default:
		return "", fmt.Errorf("unrecognized choice %q (1 or 2)", choice)
	}
}
