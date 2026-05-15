package peering

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	contractpeering "github.com/hero-engine/hero/contracts/peering"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/feed"
	"github.com/hero-engine/hero/internal/spec"
)

// HandoffOptions configures an async handoff drop.
type HandoffOptions struct {
	// PeerAlias is the local alias of the receiving workspace.
	// Required.
	PeerAlias string

	// Title overrides the receiver-side spec title. Optional —
	// defaults to the originator's title.
	Title string

	// Type overrides the receiver-side spec type. Optional —
	// defaults to the originator's type.
	Type string

	// Reason captures the rationale for the handoff. Optional but
	// strongly recommended.
	Reason string

	// At fixes the wall-clock time of the operation. Zero means
	// time.Now(). Exposed for tests.
	At time.Time
}

// HandoffResult is what Handoff returns to its caller.
type HandoffResult struct {
	// PeerSlug is the slug the spec landed under on the receiver
	// (may differ from the originator's slug after collision suffix).
	PeerSlug string
	// PeerPath is the absolute filesystem path of the receiver spec.
	PeerPath string
	// PeerID is the canonical id of the receiving workspace.
	PeerID string
	// OriginPeerID is the originator's peer_id.
	OriginPeerID string
	// PeerType is the spec type chosen for the receiver scaffold.
	PeerType string
	// PeerAliasDisplay echoes the alias used (for output).
	PeerAliasDisplay string
}

// Handoff performs an async drop of a local spec to a peer workspace.
// Order of writes is critical and documented in cross-repo-peering's
// Safety section: RECEIVER scaffold first, then ORIGINATOR status +
// trail. A failure between the two writes leaves the peer's spec
// orphan-pointing-back; `hero check` surfaces this as a recoverable
// inconsistency (re-running `hero handoff` is idempotent on the same
// originator slug).
//
// projectRoot is the originator's project root. originSlug is the
// originator's spec slug (must currently exist with an in-flight
// status).
func Handoff(projectRoot, originSlug string, opts HandoffOptions) (*HandoffResult, error) {
	if originSlug == "" {
		return nil, errors.New("origin slug required")
	}
	if opts.PeerAlias == "" {
		return nil, errors.New("peer alias required")
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.PeerID == "" {
		return nil, fmt.Errorf("workspace has no peer_id yet — run `hero` once to mint, or `hero init` in a fresh workspace")
	}

	// Resolve peer path + peer_id.
	peerPath, err := cfg.ResolveRepoPath(projectRoot, opts.PeerAlias)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(peerPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("peer %q path %q not a directory", opts.PeerAlias, peerPath)
	}
	peerHeroDir := filepath.Join(peerPath, cfg.Folder)
	peerCfgPath := filepath.Join(peerHeroDir, "hero.json")
	peerPeerID := readPeerIDFromJSON(peerCfgPath)
	if peerPeerID == "" {
		return nil, fmt.Errorf("peer %q has no peer_id in its hero.json — run `hero` once in %s to mint one", opts.PeerAlias, peerPath)
	}

	// Load the originator spec.
	heroDir := cfg.HeroDir(projectRoot)
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discover local specs: %w", err)
	}
	var origin *spec.Spec
	for _, s := range specs {
		if s.Slug == originSlug {
			origin = s
			break
		}
	}
	if origin == nil {
		return nil, fmt.Errorf("no spec with slug %q in this workspace", originSlug)
	}

	at := opts.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	atCommit := readGitHEAD(projectRoot)

	// Choose receiver type. Default: originator's type. Override
	// allowed but must be a known work-spec type.
	targetType := string(origin.Type)
	if opts.Type != "" {
		targetType = opts.Type
	}
	if targetType == "" {
		targetType = "feature"
	}

	// Pick a non-colliding slug on the peer side.
	peerBucket := bucketForType(targetType)
	peerPlanning := filepath.Join(peerHeroDir, "planning", peerBucket)
	chosenSlug := pickFreeSlug(peerPlanning, originSlug)
	peerSpecDir := filepath.Join(peerPlanning, chosenSlug)
	peerSpecPath := filepath.Join(peerSpecDir, "spec.md")

	// --- Write 1: receiver scaffold. ----------------------------
	if err := os.MkdirAll(peerSpecDir, 0o755); err != nil {
		return nil, fmt.Errorf("create peer spec dir: %w", err)
	}
	title := opts.Title
	if title == "" {
		title = origin.Title
	}
	if title == "" {
		title = chosenSlug
	}
	receiverContent := renderReceiverScaffold(receiverScaffold{
		Title:            title,
		Type:             targetType,
		OriginPeerID:     cfg.PeerID,
		OriginatorSlug:   originSlug,
		PeerAliasDisplay: opts.PeerAlias,
		HandedOffAt:      at,
		AtCommit:         atCommit,
		Reason:           opts.Reason,
		OriginTitle:      origin.Title,
		OriginPath:       origin.Path,
	})
	if err := os.WriteFile(peerSpecPath, []byte(receiverContent), 0o644); err != nil {
		return nil, fmt.Errorf("write peer spec: %w", err)
	}

	// Append a reciprocal trail entry on the receiver. We don't know
	// the receiver-side display name for US (the originator); use
	// the originator's local name as a best-effort label.
	receiverTrailEntry := contractpeering.TrailEntry{
		At:               at,
		Direction:        contractpeering.DirectionIn,
		PeerAliasDisplay: workspaceDisplayName(cfg, projectRoot),
		PeerID:           cfg.PeerID,
		Mode:             contractpeering.ModeAsyncDrop,
		OriginatingSpec:  originSlug,
		PeerSpec:         opts.PeerAlias + "/" + chosenSlug,
		AtCommit:         atCommit,
		Reason:           opts.Reason,
	}
	if err := AppendTrail(peerSpecPath, receiverTrailEntry); err != nil {
		return nil, fmt.Errorf("append peer trail: %w", err)
	}

	// Receiver-side event.
	_ = feed.AppendEvent(filepath.Join(peerHeroDir, "events.log"), feed.FeedEvent{
		Timestamp: at,
		Type:      string(contractpeering.EventHandoffReceived),
		Agent:     "hero",
		Slug:      chosenSlug,
		Message: fmt.Sprintf("received from %s/%s (peer_id %s)",
			workspaceDisplayName(cfg, projectRoot), originSlug, cfg.PeerID),
	})

	// --- Write 2: originator status + trail. --------------------
	originatorTrail := contractpeering.TrailEntry{
		At:               at,
		Direction:        contractpeering.DirectionOut,
		PeerAliasDisplay: opts.PeerAlias,
		PeerID:           peerPeerID,
		Mode:             contractpeering.ModeAsyncDrop,
		OriginatingSpec:  originSlug,
		PeerSpec:         opts.PeerAlias + "/" + chosenSlug,
		AtCommit:         atCommit,
		Reason:           opts.Reason,
	}

	originData, err := os.ReadFile(origin.Path)
	if err != nil {
		return nil, fmt.Errorf("read origin spec: %w", err)
	}
	withStatus := spec.SetFrontmatterField(string(originData), "status", string(spec.StatusHandedOff))
	withTrail := AppendTrailToContent(withStatus, originatorTrail)
	if err := os.WriteFile(origin.Path, []byte(withTrail), 0o644); err != nil {
		return nil, fmt.Errorf("write origin spec: %w", err)
	}

	// Originator event.
	_ = feed.AppendEvent(filepath.Join(heroDir, "events.log"), feed.FeedEvent{
		Timestamp: at,
		Type:      string(contractpeering.EventHandoffSent),
		Agent:     "hero",
		Slug:      originSlug,
		Message: fmt.Sprintf("handed off to %s/%s (peer_id %s)",
			opts.PeerAlias, chosenSlug, peerPeerID),
	})

	return &HandoffResult{
		PeerSlug:         chosenSlug,
		PeerPath:         peerSpecPath,
		PeerID:           peerPeerID,
		OriginPeerID:     cfg.PeerID,
		PeerType:         targetType,
		PeerAliasDisplay: opts.PeerAlias,
	}, nil
}

// receiverScaffold is the input to renderReceiverScaffold.
type receiverScaffold struct {
	Title            string
	Type             string
	OriginPeerID     string
	OriginatorSlug   string
	PeerAliasDisplay string
	HandedOffAt      time.Time
	AtCommit         string
	Reason           string
	OriginTitle      string
	OriginPath       string
}

func renderReceiverScaffold(s receiverScaffold) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %q\n", s.Title)
	fmt.Fprintf(&b, "type: %s\n", s.Type)
	b.WriteString("status: planning\n")
	b.WriteString("received_from:\n")
	fmt.Fprintf(&b, "  peer_id: %s\n", s.OriginPeerID)
	if s.PeerAliasDisplay != "" {
		fmt.Fprintf(&b, "  peer_alias_display: %s\n", s.PeerAliasDisplay)
	}
	fmt.Fprintf(&b, "  originator_slug: %s\n", s.OriginatorSlug)
	fmt.Fprintf(&b, "  handed_off_at: %s\n", s.HandedOffAt.UTC().Format(time.RFC3339))
	if s.AtCommit != "" {
		fmt.Fprintf(&b, "  at_commit: %s\n", s.AtCommit)
	}
	if s.Reason != "" {
		fmt.Fprintf(&b, "  reason: %q\n", s.Reason)
	}
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# %s\n\n", s.Title)
	b.WriteString("## Provenance\n\n")
	fmt.Fprintf(&b, "Handed off from peer `%s` (peer_id `%s`).\n", s.PeerAliasDisplay, s.OriginPeerID)
	fmt.Fprintf(&b, "Originator spec: `%s`", s.OriginatorSlug)
	if s.OriginTitle != "" && s.OriginTitle != s.Title {
		fmt.Fprintf(&b, " (\"%s\")", s.OriginTitle)
	}
	b.WriteString(".\n\n")
	if s.Reason != "" {
		b.WriteString("**Reason:** ")
		b.WriteString(s.Reason)
		b.WriteString("\n\n")
	}
	b.WriteString("## Context\n\n")
	b.WriteString("_Scaffolded by `hero handoff`. Flesh out goal, design, and acceptance criteria before delivering._\n")
	return b.String()
}

// pickFreeSlug returns originSlug if no `peerPlanning/<slug>/spec.md`
// exists, otherwise tries originSlug-2, -3, ... until it finds a free
// slot. Caps the attempts to avoid silly loops.
func pickFreeSlug(peerPlanning, originSlug string) string {
	if _, err := os.Stat(filepath.Join(peerPlanning, originSlug)); os.IsNotExist(err) {
		return originSlug
	}
	for n := 2; n < 100; n++ {
		candidate := fmt.Sprintf("%s-%d", originSlug, n)
		if _, err := os.Stat(filepath.Join(peerPlanning, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
	// Last-resort timestamp suffix.
	return fmt.Sprintf("%s-%d", originSlug, time.Now().Unix())
}

// bucketForType maps a spec type to its planning subdirectory. Mirrors
// the convention used elsewhere in the codebase (features/, bugs/,
// initiatives/, …).
func bucketForType(t string) string {
	switch t {
	case "bug":
		return "bugs"
	case "initiative":
		return "initiatives"
	case "feature":
		return "features"
	default:
		// Fall back to plural form of the type. Best-effort.
		if strings.HasSuffix(t, "s") {
			return t
		}
		return t + "s"
	}
}

// workspaceDisplayName returns a human label for this workspace. It
// prefers the configured Peering.Display, then the project root's
// base directory name.
func workspaceDisplayName(cfg config.Config, projectRoot string) string {
	if cfg.Peering != nil && cfg.Peering.Display != "" {
		return cfg.Peering.Display
	}
	return filepath.Base(projectRoot)
}

// readPeerIDFromJSON reads peer_id from a hero.json file. Returns
// empty on any error — callers must validate.
func readPeerIDFromJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var shape struct {
		PeerID string `json:"peer_id"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return ""
	}
	return shape.PeerID
}

// readGitHEAD returns the short SHA of HEAD in the given dir, or ""
// when not in a git checkout.
func readGitHEAD(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
