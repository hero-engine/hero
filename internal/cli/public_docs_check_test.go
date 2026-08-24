package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve"
)

func TestCanonicalMCPInventoryUsesRuntimeRegistryAndConfiguredProfiles(t *testing.T) {
	root := t.TempDir()
	heroDir := filepath.Join(root, config.DefaultFolder)
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	document := `{"serve":{"tool_filter":{"deny":["hero_status"],"profiles":{"minimal":["hero_search","hero_status"]}}}}`
	if err := os.WriteFile(filepath.Join(heroDir, config.ConfigFileName), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}

	inventory, err := canonicalMCPInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Total != len(serve.MCPToolDefinitions()) {
		t.Fatalf("total = %d, runtime registry = %d", inventory.Total, len(serve.MCPToolDefinitions()))
	}
	if inventory.Default != inventory.Total-1 {
		t.Fatalf("default = %d, want %d", inventory.Default, inventory.Total-1)
	}
	if len(inventory.Profiles) != 1 || inventory.Profiles[0].Name != "minimal" || inventory.Profiles[0].Count != 1 {
		t.Fatalf("profiles = %#v, want minimal=1 after deny precedence", inventory.Profiles)
	}
}

func TestPublicDocsContractPassesForRepository(t *testing.T) {
	if issues := publicDocsIssues(repoRootForTest(t)); len(issues) != 0 {
		t.Fatalf("public docs contract failed:\n%s", strings.Join(issues, "\n"))
	}
}

func TestPublicNarrativeMutationsAreRejected(t *testing.T) {
	cases := []struct {
		content string
		want    string
	}{
		{"Hero v0.9 is ready.", "stale v0.9"},
		{"Hero is open source.", "visibility gate"},
		{"Hero is licensed under MIT.", "Apache-2.0 licensed"},
		{"Hero Cloud is open source.", "proprietary"},
		{"Pass --auth-token abc.", "secret-bearing"},
		{"Hero installs all workflows as slash commands.", "harness-native"},
		{"Hero uses one global graph across all repositories.", "own graph"},
		{"hero spec complete is the normal delivery close.", "hero spec verify"},
		{"hero upgrade replaces the binary.", "workspace files"},
		{"hero-code is included in this repository.", "separate proprietary"},
		{"Sprout is included in Hero's Apache-2.0 grant.", "separate MIT-licensed"},
		{"Sprout is proprietary.", "separate public MIT-licensed"},
		{"Sprout is included in this repository.", "separate dependency"},
		{"Preview outcome: continuity proof is still being proven.", "continuity-proof qualifier"},
		{"Hero does not promise that every tool or session applies it perfectly.", "perfection disclaimer"},
		{`Source: https://github.com/hero-engine/hero/`, "source link"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			issues := publicNarrativeIssues("mutation.md", tc.content)
			if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), tc.want) {
				t.Fatalf("issues = %v, want %q", issues, tc.want)
			}
		})
	}
}

func TestPublicNarrativeSurfaceDiscoveryIncludesNavigationReleasesAndAssets(t *testing.T) {
	surfaces := publicNarrativeSurfaces(repoRootForTest(t))
	for _, path := range []string{
		"web/docs/mkdocs.yml",
		"web/docs/src/releases/index.md",
		"web/docs/src/about/third-party-notices.md",
		"web/docs/src/revision.json",
		"web/landing/site/index.html",
		"web/landing/site/revision.json",
		"<rendered:internal/install/agents_md.go>",
	} {
		if _, ok := surfaces[path]; !ok {
			t.Errorf("public surface %s was not discovered", path)
		}
	}
}

func TestPublicTruthAuthoritiesRejectInventedContinuityQualifiers(t *testing.T) {
	root := t.TempDir()
	for _, path := range publicTruthAuthorityPaths {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("The continuity proof is still being proven."), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	issues := publicTruthAuthorityIssues(root)
	if len(issues) != len(publicTruthAuthorityPaths)*2 {
		t.Fatalf("issues = %v, want two invented qualifiers for each authority", issues)
	}
}

func TestPublicRepositoryBoundaryRequiresSproutAndProprietarySeparation(t *testing.T) {
	surfaces := publicNarrativeSurfaces(repoRootForTest(t))
	if issues := repositoryBoundaryIssues(surfaces); len(issues) != 0 {
		t.Fatalf("repository boundary issues = %v", issues)
	}
	delete(surfaces, "README.md")
	if issues := repositoryBoundaryIssues(surfaces); len(issues) < 4 {
		t.Fatalf("missing README boundary should fail four checks, got %v", issues)
	}
}

func TestRepositoryLicenseRequiresCanonicalApacheTextAndNotices(t *testing.T) {
	repositoryRoot := repoRootForTest(t)
	root := t.TempDir()
	for _, name := range []string{"LICENSE", "THIRD_PARTY_NOTICES.txt"} {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if issues := repositoryLicenseIssues(root); len(issues) != 0 {
		t.Fatalf("canonical repository license failed: %v", issues)
	}
	if err := os.WriteFile(filepath.Join(root, "LICENSE"), []byte("not canonical\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if issues := repositoryLicenseIssues(root); len(issues) == 0 || !strings.Contains(issues[0], "canonical") {
		t.Fatalf("mutated license issues = %v", issues)
	}
}

func TestMarkedPublicQuickstartsRunInCleanWorkspaces(t *testing.T) {
	issues := publicQuickstartIssues(baselineBinary(t), repoRootForTest(t))
	if len(issues) != 0 {
		t.Fatalf("quickstart issues:\n%s", strings.Join(issues, "\n"))
	}
}

func TestInvocationValidationRequiresArgumentsAndFlagValues(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"valid verify", []string{"spec", "verify", "example"}, false},
		{"missing verify slug", []string{"spec", "verify"}, true},
		{"missing target value", []string{"install", "project", ".", "--target"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateExecutableInvocation(rootCmd, tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateInvocation error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestExecutablePublicCommandMutationRejectsMissingArguments(t *testing.T) {
	surfaces := map[string]string{"broken.md": "```bash\nhero spec verify\n```"}
	issues := publicExecutableInvocationIssues(surfaces)
	if len(issues) != 1 || !strings.Contains(issues[0], "accepts 1 arg") {
		t.Fatalf("issues = %v", issues)
	}
}

func TestPublicConfigExamplesUseProductionDecoder(t *testing.T) {
	content := "<!-- hero-config -->\n```json\n{\"serve\":{\"port\":\"not-a-number\"}}\n```"
	issues := publicConfigExampleIssues("broken.md", content)
	if len(issues) != 1 || !strings.Contains(issues[0], "config.Load") {
		t.Fatalf("issues = %v", issues)
	}
}

func TestDocsDependencyBoundsRejectUnpinnedOrBreakingMajors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "requirements-docs.txt")
	if err := os.WriteFile(path, []byte("mkdocs>=1.6\nmkdocs-material==10.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	issues := docsDependencyIssues(path)
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "mkdocs must be exactly pinned") || !strings.Contains(joined, "crosses supported major 9") {
		t.Fatalf("issues = %v", issues)
	}
}

func TestRevisionTemplateRejectsMissingFieldsAndLandingMetadata(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"web/docs/src", "web/landing/site"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "web/docs/src/revision.json"), []byte(`{"source_revision":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web/landing/site/revision.json"), []byte(`{"source_revision":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web/landing/site/index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	issues := revisionTemplateIssues(root)
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "missing generated_at") || !strings.Contains(joined, "missing build-time source revision metadata") {
		t.Fatalf("issues = %v", issues)
	}
}

func TestRevisionTemplateRejectsVisibleLandingProvenance(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"web/docs/src", "web/landing/site"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "web/docs/src/revision.json"), []byte(`{"source_revision":"x","current_release":"x","generated_at":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "web/landing/site/revision.json"), []byte(`{"source_revision":"x","source_commit":"x","source_digest":"x","source_dirty":false,"generated_at":"x","canonical_url":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	landing := `<meta name="hero-source-revision" content="BUILD_TIME_SOURCE_REVISION"><p>Artifact revision BUILD_TIME_SOURCE_REVISION</p>`
	if err := os.WriteFile(filepath.Join(root, "web/landing/site/index.html"), []byte(landing), 0o644); err != nil {
		t.Fatal(err)
	}
	issues := revisionTemplateIssues(root)
	if !strings.Contains(strings.Join(issues, "\n"), "build provenance is rendered as marketing copy") {
		t.Fatalf("issues = %v", issues)
	}
}

func TestProductionCrawlChecksPagesAndExactRevision(t *testing.T) {
	expected := strings.Repeat("a", 40)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/revision.json" {
			writer.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(writer, `{"source_revision":%q,"generated_at":"2026-08-23T12:00:00Z"}`, expected)
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		fmt.Fprint(writer, "<!doctype html><title>Hero</title>")
	}))
	defer server.Close()
	bases := map[string]string{"docs": server.URL, "landing": server.URL}

	if issues := productionPublicIssues(server.Client(), "all", expected, bases); len(issues) != 0 {
		t.Fatalf("production crawl failed: %v", issues)
	}
	issues := productionPublicIssues(server.Client(), "docs", strings.Repeat("b", 40), bases)
	if len(issues) != 1 || !strings.Contains(issues[0], "does not match") {
		t.Fatalf("revision mismatch issues = %v", issues)
	}
}

func TestProductionCrawlFailsClosedWhenSurfaceIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close()
	expected := strings.Repeat("a", 40)
	issues := productionPublicIssues(server.Client(), "landing", expected, map[string]string{"landing": url})
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), url) {
		t.Fatalf("issues = %v", issues)
	}
}
