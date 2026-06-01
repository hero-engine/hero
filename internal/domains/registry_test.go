package domains

import (
	"errors"
	"strings"
	"testing"
)

// Smoke tests for the Go mirror of the domain-apps core-types delivery.
// See hero-code/.hero/planning/features/domain-apps-core-types/spec.md.

func newSidebarManifest(id string, sidebarIDs ...string) DomainManifest {
	entries := make([]SidebarEntryContrib, 0, len(sidebarIDs))
	for _, sid := range sidebarIDs {
		entries = append(entries, SidebarEntryContrib{
			ID: ContributorID(sid), Label: sid, Icon: "circle", Order: 0,
		})
	}
	chat := DomainID("chat")
	return DomainManifest{
		ID:               DomainID(id),
		Display:          DomainDisplay{Name: id, Icon: "circle", Color: "#888"},
		Layer:            LayerExtension,
		Extends:          &chat,
		SidebarEntries:   entries,
		DefaultFirstView: TabKindRef("chat"),
		DSKGNamespace:    id,
	}
}

func TestRegister_TwoUniqueManifestsClean(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(newSidebarManifest("a", "a.sidebar.things")); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := r.Register(newSidebarManifest("b", "b.sidebar.things")); err != nil {
		t.Fatalf("register b: %v", err)
	}
	if got, want := len(r.Manifests()), 2; got != want {
		t.Errorf("manifests = %d, want %d", got, want)
	}
	if got, want := len(r.SidebarEntries(nil)), 2; got != want {
		t.Errorf("sidebar entries = %d, want %d", got, want)
	}
	if _, ok := r.Manifest(DomainID("a")); !ok {
		t.Errorf("manifest a missing")
	}
}

func TestRegister_RejectsDuplicateAcrossManifests(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(newSidebarManifest("a", "shared.sidebar.dup")); err != nil {
		t.Fatalf("register a: %v", err)
	}
	err := r.Register(newSidebarManifest("b", "shared.sidebar.dup"))
	if err == nil {
		t.Fatal("expected duplicate ContributorID error, got nil")
	}
	var dup *DuplicateContributorError
	if !errors.As(err, &dup) {
		t.Fatalf("expected *DuplicateContributorError, got %T: %v", err, err)
	}
	if dup.ContributorID != "shared.sidebar.dup" {
		t.Errorf("dup.ContributorID = %q, want shared.sidebar.dup", dup.ContributorID)
	}
	if dup.FirstManifest != "a" || dup.SecondManifest != "b" {
		t.Errorf("dup manifests = (%q,%q), want (a,b)", dup.FirstManifest, dup.SecondManifest)
	}
	msg := err.Error()
	for _, sub := range []string{"shared.sidebar.dup", `"a"`, `"b"`} {
		if !strings.Contains(msg, sub) {
			t.Errorf("error message missing %q: %s", sub, msg)
		}
	}
	if len(r.Manifests()) != 1 {
		t.Errorf("registry state changed on error: manifests = %d", len(r.Manifests()))
	}
}

func TestRegister_RejectsDuplicateWithinSameManifest(t *testing.T) {
	r := NewRegistry()
	chat := DomainID("chat")
	m := DomainManifest{
		ID:      "solo",
		Display: DomainDisplay{Name: "solo", Icon: "circle", Color: "#888"},
		Layer:   LayerExtension,
		Extends: &chat,
		SidebarEntries: []SidebarEntryContrib{
			{ID: "solo.dup", Label: "first", Icon: "circle"},
		},
		SlashCommands: []CommandContrib{
			{ID: "solo.dup", Label: "second", Icon: "circle"},
		},
		DefaultFirstView: TabKindRef("chat"),
		DSKGNamespace:    "solo",
	}
	if err := r.Register(m); err == nil {
		t.Fatal("expected intra-manifest duplicate error, got nil")
	}
	if len(r.Manifests()) != 0 {
		t.Errorf("registry committed despite error: manifests = %d", len(r.Manifests()))
	}
}

func TestManifestForContributorReturnsOwner(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(newSidebarManifest("pm", "pm.sidebar.backlog")); err != nil {
		t.Fatal(err)
	}
	owner, ok := r.ManifestForContributor("pm.sidebar.backlog")
	if !ok || owner != "pm" {
		t.Errorf("owner = (%q,%v), want (pm,true)", owner, ok)
	}
	if _, ok := r.ManifestForContributor("never.declared"); ok {
		t.Errorf("expected unknown contributor lookup to return false")
	}
}
