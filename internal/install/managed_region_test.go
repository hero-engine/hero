package install

import (
	"strings"
	"testing"
)

func TestFindManagedRegion(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		want     ManagedRegion
		wantBody string
	}{
		{
			name:  "no markers",
			input: "# Hello\n\nJust user content.\n",
			want:  ManagedRegion{Present: false},
		},
		{
			name: "versioned pair",
			input: "# Project\n\n" +
				"<!-- hero:managed-start v=0.7.1 -->\nHero says hi.\n<!-- hero:managed-end -->\n\n" +
				"User text.\n",
			want:     ManagedRegion{Present: true, Version: "0.7.1"},
			wantBody: "Hero says hi.",
		},
		{
			name: "versioned without end marker — treated as to-EOF",
			input: "<!-- hero:managed-start v=0.7.1 -->\nbody only, no end.\n",
			want:     ManagedRegion{Present: true, Version: "0.7.1"},
			wantBody: "body only, no end.",
		},
		{
			name: "legacy pair",
			input: "# Project\n\n" +
				legacyMarker + "\n# Hero — Spec-Driven\n...\n" + legacyMarker + "\n\nUser tail.\n",
			want:     ManagedRegion{Present: true, Legacy: true},
			wantBody: "# Hero — Spec-Driven\n...",
		},
		{
			name:  "legacy single marker — treats rest of file as region",
			input: legacyMarker + "\nblob without closing marker\n",
			want:     ManagedRegion{Present: true, Legacy: true},
			wantBody: "blob without closing marker",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FindManagedRegion(c.input)
			if got.Present != c.want.Present {
				t.Errorf("Present: got %v want %v", got.Present, c.want.Present)
			}
			if got.Legacy != c.want.Legacy {
				t.Errorf("Legacy: got %v want %v", got.Legacy, c.want.Legacy)
			}
			if got.Version != c.want.Version {
				t.Errorf("Version: got %q want %q", got.Version, c.want.Version)
			}
			if got.Body != c.wantBody {
				t.Errorf("Body: got %q want %q", got.Body, c.wantBody)
			}
		})
	}
}

func TestRenderManagedRegion(t *testing.T) {
	got := RenderManagedRegion("v0.7.2", "Hello world.")
	want := "<!-- hero:managed-start v=v0.7.2 -->\nHello world.\n<!-- hero:managed-end -->\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	// Roundtrip: a rendered region must be findable.
	mr := FindManagedRegion(got)
	if !mr.Present || mr.Version != "v0.7.2" || mr.Body != "Hello world." {
		t.Errorf("roundtrip failed: %+v", mr)
	}
}

func TestRenderManagedRegion_NoVersionDefaultsToDev(t *testing.T) {
	got := RenderManagedRegion("", "x")
	if !strings.Contains(got, "v=dev") {
		t.Errorf("expected default dev version, got: %q", got)
	}
}

func TestInsertManagedRegion_EmptyFile(t *testing.T) {
	region := RenderManagedRegion("v1", "hi")
	out := InsertManagedRegion("", region)
	if !strings.HasPrefix(out, "<!-- hero:managed-start v=v1 -->") {
		t.Errorf("expected region at start, got %q", out)
	}
}

func TestInsertManagedRegion_NoExistingRegion_PreservesH1(t *testing.T) {
	existing := "# My Project\n\nProject description.\nMore stuff.\n"
	region := RenderManagedRegion("v1", "managed body")
	out := InsertManagedRegion(existing, region)

	if !strings.HasPrefix(out, "# My Project\n") {
		t.Error("expected H1 at the top")
	}
	if !strings.Contains(out, "managed body") {
		t.Error("expected managed body present")
	}
	if !strings.Contains(out, "Project description.") {
		t.Error("expected user content preserved")
	}
	// User content must appear AFTER the managed region.
	regionIdx := strings.Index(out, "<!-- hero:managed-start")
	userIdx := strings.Index(out, "Project description.")
	if userIdx < regionIdx {
		t.Errorf("user content should come after region, got region at %d, user at %d", regionIdx, userIdx)
	}
}

func TestInsertManagedRegion_NoH1(t *testing.T) {
	existing := "Just some prose without a title.\nMore prose.\n"
	region := RenderManagedRegion("v1", "managed body")
	out := InsertManagedRegion(existing, region)

	if !strings.HasPrefix(out, "<!-- hero:managed-start") {
		t.Error("region should be at the very top when no H1")
	}
	if !strings.Contains(out, "Just some prose") {
		t.Error("user content lost")
	}
}

func TestInsertManagedRegion_ReplaceExistingVersioned(t *testing.T) {
	existing := "# Project\n\n" +
		"<!-- hero:managed-start v=v0.6.0 -->\nold managed body\n<!-- hero:managed-end -->\n\n" +
		"User content below.\n"
	region := RenderManagedRegion("v0.8.0", "new managed body")
	out := InsertManagedRegion(existing, region)

	if !strings.Contains(out, "v0.8.0") {
		t.Error("new version stamp missing")
	}
	if strings.Contains(out, "v0.6.0") {
		t.Error("old version stamp leaked")
	}
	if strings.Contains(out, "old managed body") {
		t.Error("old managed body should have been replaced")
	}
	if !strings.Contains(out, "new managed body") {
		t.Error("new managed body missing")
	}
	if !strings.Contains(out, "User content below.") {
		t.Error("user content below the region was lost")
	}
	if !strings.HasPrefix(out, "# Project\n") {
		t.Error("H1 above the region was lost")
	}
}

func TestInsertManagedRegion_UpgradesLegacyMarker(t *testing.T) {
	existing := "# Project\n\n" +
		legacyMarker + "\nlegacy hero blob\n" + legacyMarker + "\n\n" +
		"User content tail.\n"
	region := RenderManagedRegion("v0.7.2", "fresh managed body")
	out := InsertManagedRegion(existing, region)

	if strings.Contains(out, legacyMarker) {
		t.Error("legacy markers should have been replaced")
	}
	if !strings.Contains(out, "<!-- hero:managed-start v=v0.7.2 -->") {
		t.Error("versioned start marker missing")
	}
	if !strings.Contains(out, "User content tail.") {
		t.Error("user content lost")
	}
}

func TestInsertManagedRegion_Idempotent(t *testing.T) {
	region := RenderManagedRegion("v1", "stable body")
	first := InsertManagedRegion("# Title\n\nuser line\n", region)
	second := InsertManagedRegion(first, region)
	if first != second {
		t.Errorf("expected idempotent insert\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestIsLegacyHeroStub(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    bool
	}{
		{
			name:  "pure legacy stub — nothing outside markers",
			input: legacyMarker + "\nHero body\n" + legacyMarker + "\n",
			want:  true,
		},
		{
			name:  "pure versioned stub — nothing outside markers",
			input: "<!-- hero:managed-start v=v1 -->\nbody\n<!-- hero:managed-end -->\n",
			want:  true,
		},
		{
			name:  "stub with only whitespace outside",
			input: "\n\n" + legacyMarker + "\nbody\n" + legacyMarker + "\n\n  \n",
			want:  true,
		},
		{
			name: "user content above",
			input: "# My CLAUDE.md\n\nUser content first.\n\n" +
				legacyMarker + "\nbody\n" + legacyMarker + "\n",
			want: false,
		},
		{
			name: "user content below",
			input: legacyMarker + "\nbody\n" + legacyMarker + "\n\nUser content after.\n",
			want:  false,
		},
		{
			name:  "no markers at all",
			input: "# CLAUDE.md\n\nAll user content.\n",
			want:  false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := IsLegacyHeroStub(c.input)
			if got != c.want {
				t.Errorf("IsLegacyHeroStub: got %v want %v\ninput: %q", got, c.want, c.input)
			}
		})
	}
}
