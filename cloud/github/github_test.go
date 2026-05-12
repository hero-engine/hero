package github

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"
	"testing"
)

func makePR(title, body, headRef string) *PullRequestPayload {
	pr := &PullRequestPayload{
		Title: title,
		Body:  body,
	}
	pr.Head.SHA = "abc123"
	pr.Head.Ref = headRef
	pr.Base.SHA = "def456"
	pr.Base.Ref = "main"
	return pr
}

func TestCheckPR_BranchSpec(t *testing.T) {
	pr := makePR("feat: deliver my-feature", "Some description", "hero/deliver/my-feature")

	result := CheckPR(pr, ModeAdvisory)
	if !result.HasSpec {
		t.Fatal("expected HasSpec=true for hero/deliver/* branch")
	}
	if result.BranchSpec != "my-feature" {
		t.Fatalf("expected BranchSpec=my-feature, got %q", result.BranchSpec)
	}
	if result.Conclusion != "success" {
		t.Fatalf("expected conclusion=success, got %q", result.Conclusion)
	}
}

func TestCheckPR_BodySpec(t *testing.T) {
	pr := makePR("fix: something", "This implements spec:auth-flow changes", "fix/something")

	result := CheckPR(pr, ModeAdvisory)
	if !result.HasSpec {
		t.Fatal("expected HasSpec=true for spec:auth-flow in body")
	}
	if len(result.BodySpecs) != 1 || result.BodySpecs[0] != "auth-flow" {
		t.Fatalf("expected BodySpecs=[auth-flow], got %v", result.BodySpecs)
	}
}

func TestCheckPR_TitleSpec(t *testing.T) {
	pr := makePR("feat: hero:export-csv implementation", "", "feat/export")

	result := CheckPR(pr, ModeAdvisory)
	if !result.HasSpec {
		t.Fatal("expected HasSpec=true for hero:export-csv in title")
	}
	if len(result.TitleSpecs) != 1 || result.TitleSpecs[0] != "export-csv" {
		t.Fatalf("expected TitleSpecs=[export-csv], got %v", result.TitleSpecs)
	}
}

func TestCheckPR_NoSpec_Advisory(t *testing.T) {
	pr := makePR("fix: random bug", "Just fixing stuff", "fix/random")

	result := CheckPR(pr, ModeAdvisory)
	if result.HasSpec {
		t.Fatal("expected HasSpec=false")
	}
	if result.Conclusion != "neutral" {
		t.Fatalf("expected conclusion=neutral in advisory mode, got %q", result.Conclusion)
	}
}

func TestCheckPR_NoSpec_Enforcement(t *testing.T) {
	pr := makePR("fix: random bug", "Just fixing stuff", "fix/random")

	result := CheckPR(pr, ModeEnforcement)
	if result.HasSpec {
		t.Fatal("expected HasSpec=false")
	}
	if result.Conclusion != "failure" {
		t.Fatalf("expected conclusion=failure in enforcement mode, got %q", result.Conclusion)
	}
}

func TestCheckPR_MultipleSpecs(t *testing.T) {
	pr := makePR("feat: spec:auth-flow and more", "Also implements spec:user-profile\nAnd spec:auth-flow again", "hero/deliver/auth-flow")

	result := CheckPR(pr, ModeAdvisory)
	if !result.HasSpec {
		t.Fatal("expected HasSpec=true")
	}
	expected := map[string]bool{"auth-flow": true, "user-profile": true}
	for _, slug := range result.SpecSlugs {
		if !expected[slug] {
			t.Fatalf("unexpected slug %q in SpecSlugs", slug)
		}
		delete(expected, slug)
	}
	if len(expected) > 0 {
		t.Fatalf("missing slugs: %v", expected)
	}
}

func TestGenerateAppJWT(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	jwt, err := GenerateAppJWT(12345, key)
	if err != nil {
		t.Fatalf("generating JWT: %v", err)
	}

	parts := 0
	for _, c := range jwt {
		if c == '.' {
			parts++
		}
	}
	if parts != 2 {
		t.Fatalf("expected JWT with 2 dots, got %d", parts)
	}
}

func TestParsePrivateKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	derBytes := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: derBytes})

	parsed, err := ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("parsing key: %v", err)
	}
	if parsed.N.Cmp(key.N) != 0 {
		t.Fatal("parsed key doesn't match original")
	}
}

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	secret := "test-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !verifySignature(body, sig, secret) {
		t.Fatal("expected valid signature")
	}

	if verifySignature(body, "sha256=invalid", secret) {
		t.Fatal("expected invalid signature to fail")
	}
}

// --- Compliance tests ---

func TestMatchGlob_Simple(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "src/main.go", true}, // no / in pattern matches basename
		{"src/*.go", "src/main.go", true},
		{"src/*.go", "src/sub/main.go", false},
		{"*.js", "main.go", false},
	}
	for _, tt := range tests {
		if got := matchGlob(tt.pattern, tt.path); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestMatchGlob_Doublestar(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"src/**/*.go", "src/main.go", true},
		{"src/**/*.go", "src/pkg/main.go", true},
		{"src/**/*.go", "src/pkg/sub/main.go", true},
		{"src/**", "src/anything", true},
		{"src/**", "src/a/b/c", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "a/b/c.go", true},
		{"src/**/*.go", "lib/main.go", false},
		{"cloud/**", "cloud/api/router.go", true},
		{"cloud/**", "internal/cli/root.go", false},
	}
	for _, tt := range tests {
		if got := matchGlob(tt.pattern, tt.path); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestMatchConventions(t *testing.T) {
	conventions := []Convention{
		{Slug: "api-style", Title: "API Style", Scope: []string{"cloud/api/**"}, Status: "active"},
		{Slug: "cli-style", Title: "CLI Style", Scope: []string{"internal/cli/**"}, Status: "active"},
		{Slug: "draft-conv", Title: "Draft", Scope: []string{"**"}, Status: "draft"},
	}
	files := []string{"cloud/api/router.go", "cloud/api/helpers.go", "README.md"}

	matches := MatchConventions(conventions, files)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Convention.Slug != "api-style" {
		t.Fatalf("expected api-style, got %q", matches[0].Convention.Slug)
	}
	if len(matches[0].MatchedFiles) != 2 {
		t.Fatalf("expected 2 matched files, got %d", len(matches[0].MatchedFiles))
	}
}

func TestMatchConventions_Multiple(t *testing.T) {
	conventions := []Convention{
		{Slug: "api-style", Title: "API Style", Scope: []string{"cloud/**"}, Status: "active"},
		{Slug: "go-style", Title: "Go Style", Scope: []string{"**/*.go"}, Status: "active"},
	}
	files := []string{"cloud/api/router.go", "internal/cli/root.go"}

	matches := MatchConventions(conventions, files)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
}

func TestDetectScopeDrift_NoDrift(t *testing.T) {
	scope := []string{"cloud/api/**", "cloud/store/**"}
	files := []string{"cloud/api/router.go", "cloud/store/orgs.go"}

	result := DetectScopeDrift(scope, files)
	if result.HasDrift {
		t.Fatalf("expected no drift, got drift files: %v", result.DriftFiles)
	}
	if len(result.InScopeFiles) != 2 {
		t.Fatalf("expected 2 in-scope files, got %d", len(result.InScopeFiles))
	}
}

func TestDetectScopeDrift_WithDrift(t *testing.T) {
	scope := []string{"cloud/api/**"}
	files := []string{"cloud/api/router.go", "internal/cli/root.go", "README.md"}

	result := DetectScopeDrift(scope, files)
	if !result.HasDrift {
		t.Fatal("expected drift")
	}
	if len(result.DriftFiles) != 2 {
		t.Fatalf("expected 2 drift files, got %d: %v", len(result.DriftFiles), result.DriftFiles)
	}
	if len(result.InScopeFiles) != 1 {
		t.Fatalf("expected 1 in-scope file, got %d", len(result.InScopeFiles))
	}
}

func TestDetectScopeDrift_EmptyScope(t *testing.T) {
	result := DetectScopeDrift(nil, []string{"a.go", "b.go"})
	if result.HasDrift {
		t.Fatal("expected no drift when scope is empty")
	}
	if len(result.InScopeFiles) != 2 {
		t.Fatalf("expected all files in-scope, got %d", len(result.InScopeFiles))
	}
}

func TestFormatComplianceSummary(t *testing.T) {
	result := &CheckResult{
		Summary: "PR linked to spec(s): my-feature",
		ComplianceMatches: []ConventionMatch{
			{
				Convention:   Convention{Slug: "api-style", Title: "API Style", Scope: []string{"cloud/api/**"}, Status: "active"},
				MatchedFiles: []string{"cloud/api/router.go"},
			},
		},
		ScopeDrift: &ScopeDriftResult{
			SpecSlug:     "my-feature",
			SpecScope:    []string{"cloud/api/**"},
			InScopeFiles: []string{"cloud/api/router.go"},
			DriftFiles:   []string{"README.md"},
			HasDrift:     true,
		},
	}

	FormatComplianceSummary(result)

	if !strings.Contains(result.Summary, "Applicable Conventions") {
		t.Fatal("expected compliance section in summary")
	}
	if !strings.Contains(result.Summary, "api-style") {
		t.Fatal("expected convention slug in summary")
	}
	if !strings.Contains(result.Summary, "Scope Drift Warning") {
		t.Fatal("expected scope drift section in summary")
	}
	if !strings.Contains(result.Summary, "README.md") {
		t.Fatal("expected drift file in summary")
	}
}

func TestFormatComplianceSummary_NoDrift(t *testing.T) {
	result := &CheckResult{
		Summary: "base summary",
	}
	FormatComplianceSummary(result)
	if result.Summary != "base summary" {
		t.Fatal("expected summary unchanged when no compliance data")
	}
}
