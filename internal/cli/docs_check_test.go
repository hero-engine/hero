package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestIsEngineSourceRepo verifies the source-repo signature: core/ +
// domains/ + .goreleaser.yaml present means engine repo; an installed
// workspace (none of the three) does not.
func TestIsEngineSourceRepo(t *testing.T) {
	root := repoRootForTest(t)
	if !isEngineSourceRepo(root) {
		t.Errorf("expected the hero repo root %q to be detected as the engine source repo", root)
	}

	// A bare installed-workspace layout (has .claude/ but no core/domains/
	// .goreleaser.yaml) must NOT be treated as the engine repo.
	fake := t.TempDir()
	if err := os.MkdirAll(filepath.Join(fake, ".claude", "agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if isEngineSourceRepo(fake) {
		t.Errorf("installed-workspace layout %q must not be detected as the engine source repo", fake)
	}
}

// TestCanonicalCountsNonZero guards against silent regression to actual:0
// in the engine repo — the original bug.
func TestCanonicalCountsNonZero(t *testing.T) {
	agents, commands, skills, err := canonicalInstallCounts("engineering")
	if err != nil {
		t.Fatalf("canonicalInstallCounts: %v", err)
	}
	if agents == 0 || commands == 0 || skills == 0 {
		t.Fatalf("canonical counts must be non-zero, got agents=%d commands=%d skills=%d",
			agents, commands, skills)
	}
}

// TestDocCountsMatchManifest ties the published counts in README.md and
// GETTING-STARTED.md to the canonical install manifest. If content is
// added/removed and the docs aren't refreshed, this fails — the drift
// guard the runtime checker's regex can't fully cover (the README table
// format isn't regex-matched at runtime).
func TestDocCountsMatchManifest(t *testing.T) {
	root := repoRootForTest(t)
	agents, commands, skills, err := canonicalInstallCounts(activeDomainForRoot(root))
	if err != nil {
		t.Fatalf("canonicalInstallCounts: %v", err)
	}

	// GETTING-STARTED.md prose (whitespace-collapsed to survive line wraps).
	getting := collapseWS(readDoc(t, root, "GETTING-STARTED.md"))
	wantProse := fmt.Sprintf("%d slash command definitions, %d agents, and %d skills", commands, agents, skills)
	if !strings.Contains(getting, wantProse) {
		t.Errorf("GETTING-STARTED.md missing canonical counts; want substring %q", wantProse)
	}

	// README.md count table (one row per surface).
	readme := readDoc(t, root, "README.md")
	for _, want := range []string{
		fmt.Sprintf("Slash command definitions | %d", commands),
		fmt.Sprintf("Agent definitions | %d", agents),
		fmt.Sprintf("Skill definitions | %d", skills),
	} {
		if !strings.Contains(readme, want) {
			t.Errorf("README.md count table missing canonical row %q", want)
		}
	}
}

func readDoc(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

var wsRun = regexp.MustCompile(`\s+`)

// collapseWS reduces every run of whitespace to a single space so
// substring checks survive line wrapping in the source markdown.
func collapseWS(s string) string {
	return wsRun.ReplaceAllString(s, " ")
}
