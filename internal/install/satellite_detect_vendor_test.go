package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsVendorShaped(t *testing.T) {
	cases := []struct {
		base string
		want bool
	}{
		{"myproj-vendor", true},
		{"my-vendor", true},
		{"vendor-libs", true},
		{"vendor-third-party", true},
		{"vendored", true},

		{"vendor", false}, // already in noise map (not vendor-shaped)
		{"vendors", false},
		{"my-vendor-stuff", false}, // not a tail/head match
		{"app", false},
		{"engines", false},
	}
	for _, tc := range cases {
		if got := isVendorShaped(tc.base); got != tc.want {
			t.Errorf("isVendorShaped(%q) = %v, want %v", tc.base, got, tc.want)
		}
	}
}

func TestDetectCandidatesSkipsVendorShaped(t *testing.T) {
	root := t.TempDir()

	// myproj-vendor — should be skipped (suffix match), so children
	// shouldn't surface as candidates.
	for _, sub := range []string{
		"myproj-vendor/lib-a",
		"myproj-vendor/lib-b/sub",
		"vendor-libs/inner",
		"vendored/pkg",
	} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		// Plant a Cargo.toml so the file would be flagged as a candidate
		// if the detector descended.
		if err := os.WriteFile(filepath.Join(root, sub, "Cargo.toml"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// app — first-party, should be a candidate.
	app := filepath.Join(root, "app")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cs, err := DetectCandidates(root, &SubprojectsManifest{}, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.Path == "myproj-vendor" || c.Path == "vendor-libs" || c.Path == "vendored" {
			t.Errorf("vendor-shaped folder %q should be skipped, got it as a candidate", c.Path)
		}
		// Children of vendor-shaped folders shouldn't appear either.
		for _, prefix := range []string{"myproj-vendor/", "vendor-libs/", "vendored/"} {
			if len(c.Path) > len(prefix) && c.Path[:len(prefix)] == prefix {
				t.Errorf("descendant %q of vendor-shaped folder shouldn't be a candidate", c.Path)
			}
		}
	}
	// app should be there.
	hit := false
	for _, c := range cs {
		if c.Path == "app" {
			hit = true
		}
	}
	if !hit {
		t.Errorf("expected first-party 'app' as a candidate; got %v", cs)
	}
}
