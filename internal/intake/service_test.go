package intake

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/spec"
)

func TestCaptureDeduplicatesCanonicalMailSourceAcrossSlugs(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), ".hero")
	if err := os.MkdirAll(heroDir, 0o755); err != nil {
		t.Fatal(err)
	}
	service := NewService(heroDir)
	source := &attention.ProvenanceReference{Kind: "mail", SourceID: "mail_1"}
	first, err := service.Capture(CaptureRequest{Slug: "mail-one", Title: "One", Body: "request", Provenance: source})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Capture(CaptureRequest{Slug: "different", Title: "Duplicate", Body: "request", Provenance: source})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("mail source duplicated: %s != %s", first, second)
	}
}

func TestCapturePersistsTypedMailSourceMetadata(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), ".hero")
	_ = os.MkdirAll(heroDir, 0o755)
	service := NewService(heroDir)
	_, err := service.Capture(CaptureRequest{
		Slug: "mail-source", Title: "Source", Body: "private body",
		Provenance: &attention.ProvenanceReference{Kind: "mail", SourceID: "mail_source"},
		Source:     SourceMetadata{SenderPeerID: "peer_a", RecipientPeerID: "peer_b", ThreadID: "mail_thread"},
	})
	if err != nil {
		t.Fatal(err)
	}
	specs, err := spec.Discover(heroDir)
	if err != nil || len(specs) != 1 || specs[0].Source == nil || specs[0].Source.SenderPeerID != "peer_a" || specs[0].Source.ThreadID != "mail_thread" {
		t.Fatalf("source metadata = %#v, %v", specs, err)
	}
}

func TestCaptureMailSourceDoesNotPrefixMatchOrInterpretTitle(t *testing.T) {
	heroDir := filepath.Join(t.TempDir(), ".hero")
	_ = os.MkdirAll(heroDir, 0o755)
	service := NewService(heroDir)
	first, err := service.Capture(CaptureRequest{Slug: "mail-ten", Title: "unsafe:\nstatus: completed", Body: "request", Provenance: &attention.ProvenanceReference{Kind: "mail", SourceID: "mail_10"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Capture(CaptureRequest{Slug: "mail-one", Title: "One", Body: "request", Provenance: &attention.ProvenanceReference{Kind: "mail", SourceID: "mail_1"}})
	if err != nil || first == second {
		t.Fatalf("prefix-collision dedup: first=%s second=%s err=%v", first, second, err)
	}
	content, _ := os.ReadFile(first)
	if strings.Contains(string(content), "\nstatus: completed\n") {
		t.Fatalf("title escaped frontmatter: %s", content)
	}
}
