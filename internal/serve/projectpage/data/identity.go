// Package data hosts the read-only section loaders for the per-project
// Project page. Each loader is a pure function over a typed input
// bundle so it unit-tests cleanly. Missing inputs degrade to an
// empty-state result rather than erroring — the page is read-only and
// must never 500 on absent files.
package data

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

// IdentityInputs is the per-request input bundle for the Identity
// section loader. ProjectRoot empty disables every git / fs probe and
// yields a fully-empty Identity (the section renders "no project").
type IdentityInputs struct {
	ProjectRoot string
	HeroDir     string
	Slug        string
}

// Identity is what the template renders. Empty fields are rendered as
// "—" by the partial; the loader never returns errors.
type Identity struct {
	Name           string
	Slug           string
	ProjectRoot    string // absolute path
	ProjectRootURL string // file:// URL form
	GitRemote      string
	DefaultBranch  string
	LastTouchedAt  time.Time

	SpecCount       int
	NoteCount       int
	DecisionCount   int
	ConventionCount int
	PeerCount       int

	// HasProject is false when ProjectRoot is empty — the partial
	// renders an empty-state row in that case rather than the metadata
	// grid.
	HasProject bool
}

// LoadIdentity builds the Identity section payload. Best-effort across
// every probe — a failure on one field never affects the others.
func LoadIdentity(in IdentityInputs) Identity {
	if in.ProjectRoot == "" {
		return Identity{Slug: in.Slug}
	}
	id := Identity{
		Name:           filepath.Base(in.ProjectRoot),
		Slug:           in.Slug,
		ProjectRoot:    in.ProjectRoot,
		ProjectRootURL: "file://" + in.ProjectRoot,
		HasProject:     true,
	}
	if id.Slug == "" {
		id.Slug = id.Name
	}
	id.GitRemote = gitRemote(in.ProjectRoot)
	id.DefaultBranch = gitDefaultBranch(in.ProjectRoot)
	id.LastTouchedAt = lastTouchedAt(in.HeroDir)

	if in.HeroDir != "" {
		specs, err := spec.Discover(in.HeroDir)
		if err == nil {
			for _, s := range specs {
				if s == nil {
					continue
				}
				switch s.Type {
				case spec.TypeNote:
					id.NoteCount++
				case spec.TypeDecision:
					id.DecisionCount++
				case spec.TypeConvention:
					id.ConventionCount++
				}
				id.SpecCount++
			}
		}
		id.PeerCount = len(LoadPeers(PeersInputs{ProjectRoot: in.ProjectRoot, HeroDir: in.HeroDir}).Rows)
	}
	return id
}

func gitRemote(projectRoot string) string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitDefaultBranch(projectRoot string) string {
	// Try the symbolic-ref of origin/HEAD first — the canonical
	// "default branch" answer when the repo has a remote.
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		// "refs/remotes/origin/main" → "main"
		s = strings.TrimPrefix(s, "refs/remotes/origin/")
		if s != "" {
			return s
		}
	}
	// Fall back to current branch (best effort; "main" / "master" /
	// whatever the user is on right now).
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectRoot
	out, err = cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// lastTouchedAt returns the most-recent modification time across the
// .hero directory's spec files. Returns zero time when the directory
// is missing or empty.
func lastTouchedAt(heroDir string) time.Time {
	if heroDir == "" {
		return time.Time{}
	}
	var latest time.Time
	_ = filepath.WalkDir(heroDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest
}

