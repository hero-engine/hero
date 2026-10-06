package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestPeerListShowsRepoRegisteredViaLocal is a regression test for a bug
// where `hero admin repos add <alias> <path> --local` (which writes to
// hero.local.json, gitignored) registered successfully but was then
// silently invisible to `hero peer list` — which reported "No peers
// configured" even though the alias was present in hero.local.json.
//
// Root cause: config.Load merges hero.local.json onto hero.json via
// MergeLocal, but MergeLocal never copied the Repos/RepoMeta maps, so any
// command reading cfg.Repos (peer list/show, mail-reply resolution) only
// ever saw hero.json's repos, never hero.local.json's.
func TestPeerListShowsRepoRegisteredViaLocal(t *testing.T) {
	env := newTestEnv(t)

	reposLocal = true
	t.Cleanup(func() { reposLocal = false })

	addCmd := &cobra.Command{}
	addCmd.SetOut(&bytes.Buffer{})
	if err := runReposAdd(addCmd, []string{"bookwyrm-max", "../bookwyrm-max"}); err != nil {
		t.Fatalf("runReposAdd --local: %v", err)
	}

	// The alias must land in hero.local.json, not the shared hero.json.
	localRepos := readReposFile(t, env.dir, "hero.local.json")
	if got := localRepos["bookwyrm-max"]; got != "../bookwyrm-max" {
		t.Fatalf("hero.local.json repos[bookwyrm-max] = %v, want ../bookwyrm-max", got)
	}
	sharedRepos := readReposFile(t, env.dir, "hero.json")
	if _, ok := sharedRepos["bookwyrm-max"]; ok {
		t.Fatalf("hero.json unexpectedly contains the --local alias: %v", sharedRepos)
	}

	var out bytes.Buffer
	listCmd := &cobra.Command{}
	listCmd.SetOut(&out)
	if err := runPeerList(listCmd, nil); err != nil {
		t.Fatalf("runPeerList: %v", err)
	}

	got := out.String()
	if strings.Contains(got, "No peers configured") {
		t.Fatalf("hero peer list did not see the --local repo:\n%s", got)
	}
	if !strings.Contains(got, "bookwyrm-max") {
		t.Fatalf("hero peer list output missing alias %q:\n%s", "bookwyrm-max", got)
	}

	// hero peer show must resolve it too, for the same reason.
	var showOut bytes.Buffer
	showCmd := &cobra.Command{}
	showCmd.SetOut(&showOut)
	if err := runPeerShow(showCmd, []string{"bookwyrm-max"}); err != nil {
		t.Fatalf("runPeerShow: %v", err)
	}
	if !strings.Contains(showOut.String(), "Peer: bookwyrm-max") {
		t.Fatalf("hero peer show output missing peer header:\n%s", showOut.String())
	}
}

// readReposFile parses the "repos" map out of the named hero config file
// (hero.json or hero.local.json) in the given project root's .hero dir.
// Returns an empty map if the file doesn't exist.
func readReposFile(t *testing.T, projectRoot, filename string) map[string]any {
	t.Helper()
	path := filepath.Join(projectRoot, ".hero", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}
		}
		t.Fatalf("read %s: %v", path, err)
	}
	var shape struct {
		Repos map[string]any `json:"repos"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if shape.Repos == nil {
		return map[string]any{}
	}
	return shape.Repos
}
