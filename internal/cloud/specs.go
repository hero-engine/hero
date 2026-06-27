package cloud

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hero-engine/hero/internal/spec"
)

type cloudSpec struct {
	Slug         string            `json:"slug"`
	Title        string            `json:"title"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	Priority     string            `json:"priority,omitempty"`
	ClaimedBy    string            `json:"claimed_by,omitempty"`
	TrackerID    string            `json:"tracker_id,omitempty"`
	Subproject   string            `json:"subproject,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	FilesTouched []string          `json:"files_touched,omitempty"`
	Sections     map[string]string `json:"sections,omitempty"`
	RawContent   string            `json:"raw_content,omitempty"`
	Checksum     string            `json:"checksum"`
}

type SyncResult struct {
	Synced    int `json:"synced"`
	Total     int `json:"total"`
	Knowledge int `json:"knowledge"`
}

// SyncSpecs discovers all specs in heroDir and pushes them to the cloud
// sync endpoint. It is the library equivalent of `hero sync cloud`.
func SyncSpecs(ctx context.Context, client *http.Client, syncURL, heroDir string) (*SyncResult, error) {
	specs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, fmt.Errorf("discovering specs: %w", err)
	}
	if len(specs) == 0 {
		return &SyncResult{}, nil
	}

	payload := make([]cloudSpec, 0, len(specs))
	for _, s := range specs {
		cs := cloudSpec{
			Slug:         s.Slug,
			Title:        s.Title,
			Type:         string(s.Type),
			Status:       string(s.Status),
			Priority:     s.Priority,
			ClaimedBy:    s.ClaimedBy,
			TrackerID:    s.TrackerID,
			Subproject:   s.Subproject,
			Tags:         s.Tags,
			FilesTouched: s.FilesTouched,
			Checksum:     contentChecksum(s.RawContent),
		}
		if len(s.Sections) > 0 {
			headings := make(map[string]string, len(s.Sections))
			for k := range s.Sections {
				headings[k] = ""
			}
			cs.Sections = headings
		}
		payload = append(payload, cs)
	}

	knowledge := discoverKnowledge(heroDir)

	reqPayload := map[string]interface{}{"specs": payload}
	if len(knowledge) > 0 {
		reqPayload["knowledge"] = knowledge
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", syncURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("syncing to cloud: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud sync failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result SyncResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

func contentChecksum(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:8])
}
