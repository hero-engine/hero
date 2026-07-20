package cli

import (
	"errors"
	"os"
	"testing"

	"github.com/hero-engine/hero/internal/tracker"
)

type evidenceMockTracker struct {
	activityMockTracker
}

func (m *evidenceMockTracker) GetIssueEvidence(string) (*tracker.IssueEvidence, error) {
	return nil, errors.New("not used")
}

func (m *evidenceMockTracker) DownloadEvidenceAttachment(contentURL string) ([]byte, error) {
	if contentURL == "bad" {
		return nil, errors.New("download denied")
	}
	return []byte("image bytes"), nil
}

func TestDownloadEvidenceAttachments_WritesInspectableFilesAndReportsOmissions(t *testing.T) {
	heroDir := t.TempDir()
	evidence := &tracker.IssueEvidence{Attachments: []tracker.EvidenceAttachment{
		{ID: "9", Filename: "screen.png", Content: "good"},
		{ID: "10", Filename: "private.png", Content: "bad"},
	}}
	if err := downloadEvidenceAttachments(heroDir, "morph-14171", &evidenceMockTracker{}, evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Attachments[0].LocalPath == "" {
		t.Fatal("successful attachment did not receive a local path")
	}
	data, err := os.ReadFile(evidence.Attachments[0].LocalPath)
	if err != nil || string(data) != "image bytes" {
		t.Fatalf("attachment file data=%q err=%v", data, err)
	}
	if len(evidence.Omissions) != 1 {
		t.Fatalf("omissions=%v, want failed attachment reported explicitly", evidence.Omissions)
	}
}
