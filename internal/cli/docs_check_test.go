package cli

import (
	"os"
	"path/filepath"
	"regexp"
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

// TestNarrativeDocsAvoidMutableInventoryCounts keeps changing install/runtime
// inventories out of the onboarding narrative. Exact installed values belong
// to `hero doctor`; the filtered MCP registry belongs to `tools/list`.
func TestNarrativeDocsAvoidMutableInventoryCounts(t *testing.T) {
	root := repoRootForTest(t)
	mutableCount := regexp.MustCompile(`(?i)\b\d+\s+(?:slash\s+command\s+definitions|commands|specialist\s+agents|agents|skills|mcp\s+tools)\b`)
	for _, name := range []string{"README.md", "GETTING-STARTED.md", "MCP-SETUP.md"} {
		if match := mutableCount.FindString(readDoc(t, root, name)); match != "" {
			t.Errorf("%s contains mutable narrative inventory count %q", name, match)
		}
	}
}

func TestRootDocsUseSpecPathForSyncPull(t *testing.T) {
	root := repoRootForTest(t)
	for _, name := range []string{"README.md", "GETTING-STARTED.md", "MCP-SETUP.md", "CROSS-REPO-PEERING.md", "TEAM-SERVER.md"} {
		for _, invocation := range ExtractInvocations(name, []byte(readDoc(t, root, name))) {
			if len(invocation.Args) < 2 || invocation.Args[0] != "sync" || invocation.Args[1] != "pull" {
				continue
			}
			if len(invocation.Args) < 3 || !regexp.MustCompile(`^\.hero/.+/spec\.md$`).MatchString(invocation.Args[2]) {
				t.Errorf("%s:%d uses slug-only or missing sync-pull path: %q", name, invocation.Line, invocation.Raw)
			}
		}
	}
}

func TestRootDocsRejectUnshippedApprovalBridgeAndSecretArgv(t *testing.T) {
	root := repoRootForTest(t)
	for _, name := range []string{"README.md", "GETTING-STARTED.md", "MCP-SETUP.md", "CROSS-REPO-PEERING.md", "TEAM-SERVER.md"} {
		content := readDoc(t, root, name)
		for _, forbidden := range []string{"hero agent approve", "--auth-token", "HERO_TEAM_TOKEN"} {
			if regexp.MustCompile(regexp.QuoteMeta(forbidden)).MatchString(content) {
				t.Errorf("%s contains forbidden root-doc guidance %q", name, forbidden)
			}
		}
	}
	if content := readDoc(t, root, "TEAM-SERVER.md"); !regexp.MustCompile(`HERO_AUTH_TOKEN`).MatchString(content) {
		t.Error("TEAM-SERVER.md must direct the process manager to inject HERO_AUTH_TOKEN")
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
