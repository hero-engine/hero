package scan

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hero-engine/hero/internal/config"
)

type fakeScanner struct {
	id     string
	called string
}

func (f *fakeScanner) ID() string { return f.id }
func (f *fakeScanner) Scan(sub string, opts ScanOpts) (*Report, error) {
	f.called = sub
	return &Report{Summary: "fake ran " + sub}, nil
}

func TestDispatch_RoutesByActiveDomain(t *testing.T) {
	t.Cleanup(resetDispatcherForTest)
	resetDispatcherForTest()

	eng := &fakeScanner{id: "engineering-code-scan"}
	Register(eng)
	pm := &fakeScanner{id: "pm-roadmap-scan"}
	Register(pm)

	RegisterManifest("engineering", &Manifest{
		ScannerID:   "engineering-code-scan",
		Subcommands: []ManifestSubcmd{{ID: "scan"}},
	})
	RegisterManifest("pm", &Manifest{
		ScannerID:   "pm-roadmap-scan",
		Subcommands: []ManifestSubcmd{{ID: "scan"}},
	})

	for _, tc := range []struct {
		name        string
		domain      string
		wantCalled  string
		wantSummary string
	}{
		{"engineering-explicit", "engineering", "scan", "fake ran scan"},
		{"engineering-default", "", "scan", "fake ran scan"},
		{"pm", "pm", "scan", "fake ran scan"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			eng.called = ""
			pm.called = ""
			report, err := Dispatch("scan", ScanOpts{Config: config.Config{Domain: tc.domain}})
			if err != nil {
				t.Fatalf("Dispatch: %v", err)
			}
			if report.Summary != tc.wantSummary {
				t.Errorf("summary = %q, want %q", report.Summary, tc.wantSummary)
			}
		})
	}
}

func TestDispatch_NoManifestReturnsNotFound(t *testing.T) {
	t.Cleanup(resetDispatcherForTest)
	resetDispatcherForTest()

	_, err := Dispatch("scan", ScanOpts{Config: config.Config{Domain: "qa"}})
	if !errors.Is(err, ErrScannerNotFound) {
		t.Fatalf("Dispatch with no manifest: err = %v, want ErrScannerNotFound", err)
	}
}

func TestDispatch_UnknownSubcommand(t *testing.T) {
	t.Cleanup(resetDispatcherForTest)
	resetDispatcherForTest()

	Register(&fakeScanner{id: "engineering-code-scan"})
	RegisterManifest("engineering", &Manifest{
		ScannerID:   "engineering-code-scan",
		Subcommands: []ManifestSubcmd{{ID: "scan"}},
	})

	_, err := Dispatch("import", ScanOpts{Config: config.Config{Domain: "engineering"}})
	if !errors.Is(err, ErrSubcommandUnsupported) {
		t.Fatalf("Dispatch with unknown sub: err = %v, want ErrSubcommandUnsupported", err)
	}
}

func TestDispatch_ManifestWithoutMatchingScanner(t *testing.T) {
	t.Cleanup(resetDispatcherForTest)
	resetDispatcherForTest()

	RegisterManifest("engineering", &Manifest{
		ScannerID:   "engineering-code-scan",
		Subcommands: []ManifestSubcmd{{ID: "scan"}},
	})

	_, err := Dispatch("scan", ScanOpts{Config: config.Config{Domain: "engineering"}})
	if err == nil {
		t.Fatal("Dispatch should fail when manifest names an unregistered scanner_id")
	}
	if !strings.Contains(err.Error(), "engineering-code-scan") {
		t.Errorf("error should name the missing scanner_id, got %v", err)
	}
}

func TestParseManifest_Happy(t *testing.T) {
	data := []byte(`manifest_version: "1"
scanner_id: engineering-code-scan
display_name: Engineering
subcommands:
  - id: scan
    description: scan project
    flags:
      - { name: code, type: bool, default: false }
emits:
  node_types: [Repo, Package]
  edge_kinds: [belongs_to]
config_keys: [code_scan.depth]
`)
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if m.ScannerID != "engineering-code-scan" {
		t.Errorf("ScannerID = %q", m.ScannerID)
	}
	if !m.HasSubcommand("scan") {
		t.Error("HasSubcommand(scan) should be true")
	}
	if m.HasSubcommand("import") {
		t.Error("HasSubcommand(import) should be false")
	}
}

func TestParseManifest_Errors(t *testing.T) {
	for _, tc := range []struct {
		name string
		yaml string
		want string
	}{
		{"missing-scanner-id", "subcommands: []", "scanner_id is required"},
		{"future-version", `scanner_id: x
manifest_version: "2"`, "unsupported manifest_version"},
		{"bad-flag-type", `scanner_id: x
subcommands:
  - id: scan
    flags:
      - { name: weird, type: rune }`, "unsupported type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLoadAndRegisterManifest_MissingFileIsClean(t *testing.T) {
	t.Cleanup(resetDispatcherForTest)
	resetDispatcherForTest()

	emptyFS := fstest.MapFS{} // no scan-manifest.yaml
	if err := LoadAndRegisterManifest("qa", emptyFS); err != nil {
		t.Fatalf("expected nil error for missing manifest, got %v", err)
	}
	// And Dispatch should now report ErrScannerNotFound for the domain.
	_, err := Dispatch("scan", ScanOpts{Config: config.Config{Domain: "qa"}})
	if !errors.Is(err, ErrScannerNotFound) {
		t.Errorf("Dispatch after no-manifest load: err = %v, want ErrScannerNotFound", err)
	}
}
