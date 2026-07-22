package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	projectcontract "github.com/hero-engine/hero/contracts/trackerproject"
)

func TestJoinProjectSnapshotLocalSlugs(t *testing.T) {
	env := newTestEnv(t)
	dir := filepath.Join(env.heroDir, "planning", "bugs", "morph-297")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte("---\ntitle: Bug\nslug: morph-297\ntype: bug\nstatus: planning\ntracker_id: MORPH-297\n---\n\n# Bug\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := &projectcontract.Snapshot{Items: []projectcontract.Item{{TrackerID: "MORPH-297"}, {TrackerID: "MORPH-999"}}}
	joinProjectSnapshotLocalSlugs(snapshot, env.heroDir)
	if snapshot.Items[0].LocalSlug != "morph-297" || snapshot.Items[1].LocalSlug != "" {
		t.Fatalf("items = %+v", snapshot.Items)
	}
}

func TestTrackerProjectSnapshotContractJSONIsStable(t *testing.T) {
	snapshot := projectcontract.Snapshot{Version: projectcontract.Version, Provider: "jira", Project: projectcontract.Project{ID: "MORPH"}, Iterations: []projectcontract.Iteration{}, Items: []projectcontract.Item{}, GeneratedAt: time.Date(2026, 7, 22, 1, 2, 3, 0, time.UTC), Complete: true}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var decoded projectcontract.Snapshot
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != "tracker-project-snapshot/v1" || !decoded.Complete || decoded.Project.ID != "MORPH" {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestEffectiveProjectSnapshotBoardExplicitFlagOverridesConfig(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		configured, explicit string
		explicitSet          bool
		want                 string
	}{
		{name: "config default", configured: "MORPH board", want: "MORPH board"},
		{name: "explicit override", configured: "MORPH board", explicit: "42", explicitSet: true, want: "42"},
		{name: "explicit empty override", configured: "MORPH board", explicitSet: true, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectiveProjectSnapshotBoard(tc.configured, tc.explicit, tc.explicitSet); got != tc.want {
				t.Fatalf("board = %q, want %q", got, tc.want)
			}
		})
	}
}
