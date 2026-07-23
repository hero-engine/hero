package focus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/serve"
)

func TestRegistryResolverUsesPeerIDAsCanonicalIdentity(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "renamed-project")
	if err := os.MkdirAll(filepath.Join(project, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".hero", "hero.json"), []byte(`{"peer_id":"peer-canonical","peering":{"display":"Renamed Project"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := serve.LoadRegistryFrom(filepath.Join(base, "projects.json"))
	if err != nil {
		t.Fatal(err)
	}
	slug, err := registry.Add(project)
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewRegistryResolver(registry)
	ref := &attention.ProjectReference{PeerID: "peer-canonical", RegistrySlug: "old-slug", DisplayName: "Old Name"}
	resolved := resolver.ResolveReference(ref)
	if resolved.Availability != ProjectAvailable || resolved.Path != project {
		t.Fatalf("resolved = %#v", resolved)
	}
	if resolved.Reference.RegistrySlug != slug || resolved.Reference.PeerID != ref.PeerID {
		t.Fatalf("reference = %#v", resolved.Reference)
	}
	input, err := resolver.ResolveInput(slug)
	if err != nil || input.PeerID != ref.PeerID {
		t.Fatalf("input = %#v, %v", input, err)
	}
	resolver.currentProjectRoot = project
	current, err := resolver.ResolveCurrent()
	if err != nil || current == nil || current.PeerID != ref.PeerID || current.RegistrySlug != slug || current.DisplayName != "Renamed Project" {
		t.Fatalf("current = %#v, %v", current, err)
	}
	resolver.currentProjectRoot = filepath.Join(base, "unregistered")
	current, err = resolver.ResolveCurrent()
	if err != nil || current != nil {
		t.Fatalf("unregistered current = %#v, %v", current, err)
	}

	missing := resolver.ResolveReference(&attention.ProjectReference{PeerID: "peer-missing", DisplayName: "Missing"})
	if missing.Availability != ProjectMissing || missing.Path != "" {
		t.Fatalf("missing = %#v", missing)
	}

	peerProject := filepath.Join(base, "configured-peer")
	if err := os.MkdirAll(filepath.Join(peerProject, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peerProject, ".hero", "hero.json"), []byte(`{"peer_id":"peer-sibling"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver.peers["sibling"] = peerProject
	peerRef := &attention.ProjectReference{PeerID: "peer-sibling", DisplayName: "Sibling"}
	peerResolved := resolver.ResolveReference(peerRef)
	if peerResolved.Availability != ProjectAvailable || peerResolved.Path != peerProject {
		t.Fatalf("peer resolved = %#v", peerResolved)
	}
	peerInput, err := resolver.ResolveInput("sibling")
	if err != nil || peerInput.PeerID != "peer-sibling" {
		t.Fatalf("peer input = %#v, %v", peerInput, err)
	}
}
