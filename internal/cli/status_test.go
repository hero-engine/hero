package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/async"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/version"
)

func TestStatusEmpty(t *testing.T) {
	_ = newTestEnv(t)

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	if !strings.Contains(output, "Work: 0 in progress · 0 upcoming (0 ready, 0 blocked) · 0 waiting · 0 completed") {
		t.Errorf("status should show zero work counts: %q", output)
	}
	if !strings.Contains(output, "Other: 0 intake · 0 knowledge · 0 hidden by horizon") {
		t.Errorf("status should show zero corpus counts: %q", output)
	}
	if !strings.Contains(output, "No operational work.") {
		t.Errorf("status should show the concise empty state: %q", output)
	}
	for _, heading := range []string{"In progress (", "Upcoming (", "Waiting (", "Recently completed ("} {
		if strings.Contains(output, heading) {
			t.Errorf("empty status should not render %q: %q", heading, output)
		}
	}
}

func TestStatusMixedViewCountsOrderingAndSuppression(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/returned/spec.md", statusFixture("Returned Work", "feature", "handed_back", ""))
	env.addSpec("planning/bugs/delivering/spec.md", statusFixture("Delivering Work", "bug", "delivering", "claimed_by: alice"))
	env.addSpec("planning/features/reviewing/spec.md", statusFixture("Reviewing Work", "feature", "in-review", ""))
	env.addSpec("planning/features/ready/spec.md", statusFixture("Ready Work", "feature", "planning", ""))
	env.addSpec("planning/features/blocked/spec.md", statusFixture("Blocked Work", "feature", "planning", `relations:
  - target: open-prereq
    kind: depends-on`))
	env.addSpec("specs/done-timestamped/spec.md", statusFixture("Just Finished", "feature", "completed", "completed_at: 2026-07-25T12:00:00Z"))
	env.addSpec("specs/done-undated/spec.md", statusFixture("Old Finished", "bug", "completed", ""))
	env.addSpec("planning/intake/new-idea/spec.md", statusFixture("Intake Detail Must Stay Hidden", "intake", "planning", ""))
	env.addSpec("knowledge/conventions/open-prereq/spec.md", statusFixture("Knowledge Detail Must Stay Hidden", "convention", "active", ""))

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	if !strings.Contains(output, "Work: 3 in progress · 2 upcoming (1 ready, 1 blocked) · 0 waiting · 2 completed") {
		t.Errorf("mixed counts are wrong:\n%s", output)
	}
	if !strings.Contains(output, "Other: 1 intake · 1 knowledge · 0 hidden by horizon") {
		t.Errorf("corpus counts are wrong:\n%s", output)
	}
	returned := strings.Index(output, "returned")
	delivering := strings.Index(output, "delivering")
	reviewing := strings.Index(output, "reviewing")
	if returned < 0 || delivering < 0 || reviewing < 0 || !(returned < delivering && delivering < reviewing) {
		t.Errorf("in-progress precedence should be handed_back, delivering, in-review:\n%s", output)
	}
	ready := strings.Index(output, "Ready Work")
	blocked := strings.Index(output, "Blocked Work")
	if ready < 0 || blocked < 0 || ready > blocked {
		t.Errorf("ready upcoming work should precede blocked work:\n%s", output)
	}
	if !strings.Contains(output, "Blocked Work  [blocked]") {
		t.Errorf("blocked upcoming work should be explicit:\n%s", output)
	}
	for _, hidden := range []string{"Intake Detail Must Stay Hidden", "Knowledge Detail Must Stay Hidden", "Old Finished"} {
		if strings.Contains(output, hidden) {
			t.Errorf("default status should suppress exhaustive corpus row %q:\n%s", hidden, output)
		}
	}
	if !strings.Contains(output, "Just Finished") || !strings.Contains(output, "… 1 more — `hero list --status completed --sort recency`") {
		t.Errorf("recent completion and undated completion hint should both be represented:\n%s", output)
	}
	for _, hint := range []string{
		"`hero list --type intake`",
		"`hero list --type convention,decision,rule,external,context,note`",
	} {
		if !strings.Contains(output, hint) {
			t.Errorf("status missing corpus hint %s:\n%s", hint, output)
		}
	}
}

func TestStatusBoundsUpcomingAndWaiting(t *testing.T) {
	env := newTestEnv(t)
	for i := 0; i < 12; i++ {
		extra := ""
		if i == 11 {
			extra = "pinned: true"
		}
		env.addSpec(fmt.Sprintf("planning/features/upcoming-%02d/spec.md", i),
			statusFixture(fmt.Sprintf("Upcoming %02d", i), "feature", "planning", extra))
	}
	for i := 0; i < 11; i++ {
		status := "handed_off"
		if i%2 == 1 {
			status = "awaiting_peer"
		}
		env.addSpec(fmt.Sprintf("planning/features/waiting-%02d/spec.md", i),
			statusFixture(fmt.Sprintf("Waiting %02d", i), "feature", status, ""))
	}

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if got := strings.Count(outputSection(output, "Upcoming (", "\nWaiting ("), "upcoming-"); got != 10 {
		t.Errorf("Upcoming rendered %d rows, want 10:\n%s", got, output)
	}
	if !strings.Contains(output, "upcoming-11") {
		t.Errorf("canonical priority ordering should keep the pinned item in the bounded view:\n%s", output)
	}
	if !strings.Contains(output, "… 2 more — `hero list --status planning --sort priority`") {
		t.Errorf("missing upcoming omitted-count hint:\n%s", output)
	}
	if got := strings.Count(outputSection(output, "Waiting (", "\nConnections:"), "waiting-"); got != 10 {
		t.Errorf("Waiting rendered %d rows, want 10:\n%s", got, output)
	}
	if !strings.Contains(output, "… 1 more — `hero list --status handed_off,awaiting_peer --sort priority`") {
		t.Errorf("missing waiting omitted-count hint:\n%s", output)
	}
}

func TestStatusRecentlyCompletedUsesAuthoritativeTimestamp(t *testing.T) {
	env := newTestEnv(t)
	completed := []struct {
		slug      string
		timestamp string
	}{
		{"alpha-tie", "2026-07-25T12:00:00Z"},
		{"beta-tie", "2026-07-25T12:00:00Z"},
		{"third", "2026-07-24T12:00:00Z"},
		{"fourth", "2026-07-23T12:00:00Z"},
		{"fifth", "2026-07-22T12:00:00Z"},
		{"sixth", "2026-07-21T12:00:00Z"},
	}
	for _, item := range completed {
		env.addSpec("specs/"+item.slug+"/spec.md",
			statusFixture("Completed "+item.slug, "feature", "completed", "completed_at: "+item.timestamp))
	}
	env.addSpec("specs/missing-timestamp/spec.md",
		statusFixture("Missing Timestamp", "bug", "completed", ""))

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	recent := outputSection(output, "Recently completed (", "\nConnections:")
	if got := strings.Count(recent, "Completed "); got != 5 {
		t.Errorf("recent section rendered %d rows, want 5:\n%s", got, recent)
	}
	if strings.Index(recent, "alpha-tie") > strings.Index(recent, "beta-tie") {
		t.Errorf("timestamp ties should sort by slug:\n%s", recent)
	}
	for _, omitted := range []string{"sixth", "missing-timestamp", "Missing Timestamp"} {
		if strings.Contains(recent, omitted) {
			t.Errorf("recent section should omit %q:\n%s", omitted, recent)
		}
	}
	if !strings.Contains(recent, "… 2 more — `hero list --status completed --sort recency`") {
		t.Errorf("recent section should account for timestamped and undated omissions:\n%s", recent)
	}
}

func TestStatusArchiveInconsistencies(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("specs/stale-open/spec.md", statusFixture("Stale Open Archive Work", "feature", "planning", ""))
	env.addSpec("planning/features/stale-completed/spec.md",
		statusFixture("Completed In Planning", "feature", "completed", "completed_at: 2026-07-25T12:00:00Z"))

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if !strings.Contains(output, "Work: 0 in progress · 0 upcoming (0 ready, 0 blocked) · 0 waiting · 1 completed") {
		t.Errorf("archive mismatches should not enter active counts:\n%s", output)
	}
	if strings.Contains(output, "Stale Open Archive Work") {
		t.Errorf("non-completed archive work must not appear active:\n%s", output)
	}
	if !strings.Contains(output, "Completed In Planning") {
		t.Errorf("completed work left in planning should still contribute to completion history:\n%s", output)
	}
	if !strings.Contains(output, "Archive inconsistencies: 2 — run `hero check` for details") {
		t.Errorf("archive mismatch warning is missing:\n%s", output)
	}
}

func TestStatusHorizonOptionsOnlyFilterOpenWork(t *testing.T) {
	env := newTestEnv(t)
	for _, horizon := range []string{"now", "next", "someday", "parking"} {
		env.addSpec("planning/features/"+horizon+"/spec.md",
			statusFixture("Work "+horizon, "feature", "planning", "horizon: "+horizon))
	}
	env.addSpec("specs/completed-parking/spec.md",
		statusFixture("Completed Parking", "feature", "completed", "horizon: parking\ncompleted_at: 2026-07-25T12:00:00Z"))
	env.addSpec("planning/intake/later-idea/spec.md",
		statusFixture("Later Idea", "intake", "planning", "horizon: parking"))
	env.addSpec("knowledge/notes/later-note/spec.md",
		statusFixture("Later Note", "note", "active", "horizon: parking"))

	defaultOutput, err := runCmd("status")
	if err != nil {
		t.Fatalf("default status: %v", err)
	}
	if !strings.Contains(defaultOutput, "2 upcoming (2 ready, 0 blocked)") ||
		!strings.Contains(defaultOutput, "1 intake · 1 knowledge · 2 hidden by horizon") {
		t.Errorf("default horizon counts are wrong:\n%s", defaultOutput)
	}

	allOutput, err := runCmd("status", "--all")
	if err != nil {
		t.Fatalf("status --all: %v", err)
	}
	if !strings.Contains(allOutput, "4 upcoming (4 ready, 0 blocked)") ||
		!strings.Contains(allOutput, "0 waiting · 1 completed") {
		t.Errorf("--all open/completed counts are wrong:\n%s", allOutput)
	}
	if strings.Contains(allOutput, "hidden by horizon") {
		t.Errorf("--all should omit the hidden-by-horizon count:\n%s", allOutput)
	}
	if !strings.Contains(allOutput, "Other: 1 intake · 1 knowledge") {
		t.Errorf("--all should preserve workspace-wide corpus counts:\n%s", allOutput)
	}

	somedayOutput, err := runCmd("status", "--horizon", "someday")
	if err != nil {
		t.Fatalf("status --horizon someday: %v", err)
	}
	if !strings.Contains(somedayOutput, "1 upcoming (1 ready, 0 blocked)") ||
		!strings.Contains(somedayOutput, "0 waiting · 1 completed") ||
		!strings.Contains(somedayOutput, "Other: 1 intake · 1 knowledge") {
		t.Errorf("explicit horizon should filter only open work:\n%s", somedayOutput)
	}
	if strings.Contains(somedayOutput, "hidden by horizon") {
		t.Errorf("explicit horizon should omit hidden-by-horizon count:\n%s", somedayOutput)
	}
}

func TestStatusJSONContractRemainsUnbounded(t *testing.T) {
	env := newTestEnv(t)
	for i := 0; i < 12; i++ {
		env.addSpec(fmt.Sprintf("planning/features/open-%02d/spec.md", i),
			statusFixture(fmt.Sprintf("Open %02d", i), "feature", "planning", "horizon: now"))
	}
	env.addSpec("planning/features/later/spec.md", statusFixture("Later", "feature", "planning", "horizon: someday"))
	env.addSpec("knowledge/notes/reference/spec.md", statusFixture("Reference", "note", "active", ""))

	output, err := runCmd("status", "--json")
	if err != nil {
		t.Fatalf("status --json: %v", err)
	}
	payload := decodeStatusJSON(t, output)
	if got := len(payload.Specs); got != 13 {
		t.Fatalf("default JSON returned %d specs, want 13 unbounded active-horizon + knowledge entries", got)
	}
	for _, item := range payload.Specs {
		if item.Slug == "" || item.Title == "" || item.Type == "" || item.Status == "" || item.Horizon == "" {
			t.Errorf("JSON item lost an existing field: %+v", item)
		}
	}

	output, err = runCmd("status", "--json", "--horizon", "someday")
	if err != nil {
		t.Fatalf("status --json --horizon someday: %v", err)
	}
	payload = decodeStatusJSON(t, output)
	if got := len(payload.Specs); got != 2 {
		t.Fatalf("explicit-horizon JSON returned %d specs, want someday work + unfiltered knowledge", got)
	}
}

func TestStatusEmittedListHintsResolve(t *testing.T) {
	env := newTestEnv(t)
	env.addSpec("planning/features/open/spec.md", statusFixture("Open", "feature", "planning", ""))
	env.addSpec("specs/done/spec.md", statusFixture("Done", "feature", "completed", "completed_at: 2026-07-25T12:00:00Z"))
	env.addSpec("planning/intake/idea/spec.md", statusFixture("Idea", "intake", "planning", ""))
	env.addSpec("knowledge/notes/note/spec.md", statusFixture("Note", "note", "active", ""))

	commands := [][]string{
		{"list", "--status", "planning", "--sort", "priority"},
		{"list", "--status", "handed_off,awaiting_peer", "--sort", "priority"},
		{"list", "--status", "completed", "--sort", "recency"},
		{"list", "--type", "intake"},
		{"list", "--type", "convention,decision,rule,external,context,note"},
	}
	for _, command := range commands {
		if _, err := runCmd(command...); err != nil {
			t.Errorf("emitted hint `hero %s` does not resolve: %v", strings.Join(command, " "), err)
		}
	}
}

func TestStatusHelpDescribesCompactView(t *testing.T) {
	output := statusCmd.Long
	for _, want := range []string{"compact operational briefing", "five", "hero list"} {
		if !strings.Contains(output, want) {
			t.Errorf("status help missing %q:\n%s", want, output)
		}
	}
}

func TestStatusPreservesAsyncConnectionAndVersionSignals(t *testing.T) {
	env := newTestEnv(t)

	store := async.DefaultStore()
	if err := store.Add(async.Job{
		ID:        "status-running-job",
		Type:      async.JobDeliver,
		Slug:      "async-status-work",
		Branch:    "feature/async-status",
		Status:    async.StatusRunning,
		StartedAt: time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("add async job: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join(os.Getenv("HOME"), ".hero", "async-jobs.json"))
	})

	originalVersion := rootCmd.Version
	rootCmd.Version = "2.0.0"
	t.Cleanup(func() { rootCmd.Version = originalVersion })
	version.StampInit(env.heroDir, "1.0.0")

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	for _, want := range []string{
		"Async Delivery:",
		"async-status-work",
		"running",
		"branch:feature/async-status",
		"Connections:",
		"tracker    (none)",
		"Hero 2.0.0 (workspace 1.0.0 — upgrade available)",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("status missing retained operational signal %q:\n%s", want, output)
		}
	}
}

func TestStatusPreservesPeerReconciliationSignal(t *testing.T) {
	env := newTestEnv(t)
	peerRoot := t.TempDir()
	peerSpecDir := filepath.Join(peerRoot, ".hero", "specs", "features", "peer-completed")
	if err := os.MkdirAll(peerSpecDir, 0o755); err != nil {
		t.Fatalf("create peer spec directory: %v", err)
	}
	peerContent := statusFixture("Peer Completed", "feature", "completed",
		"completed_at: 2026-07-25T12:00:00Z")
	if err := os.WriteFile(filepath.Join(peerSpecDir, "spec.md"), []byte(peerContent), 0o644); err != nil {
		t.Fatalf("write peer spec: %v", err)
	}

	cfg, err := config.Load(env.dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Repos = map[string]string{"hero-code": peerRoot}
	if err := cfg.Save(env.dir); err != nil {
		t.Fatalf("save peer config: %v", err)
	}

	local := statusFixture("Awaiting Peer Completion", "feature", "awaiting_peer", "") + `
## Handoff Trail

- 2026-07-25T11:00:00Z — out → hero-code (peer_id: peer-code)
  mode: spec-out
  originating_spec: awaiting-peer-completion
  peer_spec: hero-code/peer-completed
  peer_status: delivering
`
	env.addSpec("planning/features/awaiting-peer-completion/spec.md", local)

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if !strings.Contains(output, "Peer-side completion detected for 1 spec(s): [awaiting-peer-completion] — moved to handed_back.") {
		t.Errorf("status missing peer reconciliation signal:\n%s", output)
	}
	if !strings.Contains(output, "awaiting-peer-completion") || !strings.Contains(output, "handed_back") {
		t.Errorf("reconciled work should appear in progress as handed_back:\n%s", output)
	}
}

// TestStatus_SurfacesSmokeFailures verifies per-feature-smoke-coverage AC-6:
// hero status surfaces failed smokes in its default output.
func TestStatus_SurfacesSmokeFailures(t *testing.T) {
	env := newTestEnv(t)

	// Seed a last-run.json with one failed and one passed smoke.
	smokeDir := filepath.Join(env.heroDir, "smoke")
	if err := os.MkdirAll(smokeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll smoke dir: %v", err)
	}
	records := []SmokeRunRecord{
		{Slug: "failing-feature", Status: "fail", Timestamp: time.Now(), DurationMS: 250, Error: "smoke script exited 1"},
		{Slug: "passing-feature", Status: "pass", Timestamp: time.Now(), DurationMS: 120},
	}
	data, _ := json.Marshal(records)
	if err := os.WriteFile(filepath.Join(smokeDir, "last-run.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	if !strings.Contains(output, "Smoke failures") {
		t.Errorf("expected 'Smoke failures' section in output; got:\n%s", output)
	}
	if !strings.Contains(output, "failing-feature") {
		t.Errorf("expected failing-feature in smoke failures; got:\n%s", output)
	}
	if strings.Contains(output, "passing-feature") {
		t.Errorf("passing-feature should not appear in smoke failures; got:\n%s", output)
	}
}

// TestStatus_NoSmokeFailuresSilent verifies that when all smokes pass,
// no smoke failure section is rendered.
func TestStatus_NoSmokeFailuresSilent(t *testing.T) {
	env := newTestEnv(t)

	smokeDir := filepath.Join(env.heroDir, "smoke")
	if err := os.MkdirAll(smokeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll smoke dir: %v", err)
	}
	records := []SmokeRunRecord{
		{Slug: "all-good", Status: "pass", Timestamp: time.Now(), DurationMS: 100},
	}
	data, _ := json.Marshal(records)
	if err := os.WriteFile(filepath.Join(smokeDir, "last-run.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	output, err := runCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if strings.Contains(output, "Smoke failures") {
		t.Errorf("no smoke failures expected; got:\n%s", output)
	}
}

func TestStatusNoWorkspace(t *testing.T) {
	_ = newTestEnvEmpty(t)

	_, err := runCmd("status")
	if err == nil {
		t.Fatal("status should error without workspace")
	}

	if !strings.Contains(err.Error(), "no hero workspace found") {
		t.Errorf("error should mention no workspace: %v", err)
	}
}

func statusFixture(title, specType, status, extra string) string {
	if extra != "" {
		extra += "\n"
	}
	return fmt.Sprintf(`---
title: %s
type: %s
status: %s
%s---
# %s
`, title, specType, status, extra, title)
}

func outputSection(output, start, end string) string {
	startIndex := strings.Index(output, start)
	if startIndex < 0 {
		return ""
	}
	section := output[startIndex:]
	if endIndex := strings.Index(section, end); endIndex >= 0 {
		section = section[:endIndex]
	}
	return section
}

type decodedStatusPayload struct {
	Workspace string           `json:"workspace"`
	HeroDir   string           `json:"hero_dir"`
	Specs     []statusJSONSpec `json:"specs"`
}

func decodeStatusJSON(t *testing.T, output string) decodedStatusPayload {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		t.Fatalf("decode raw status JSON: %v\n%s", err, output)
	}
	for key := range raw {
		if key != "workspace" && key != "hero_dir" && key != "specs" && key != "mail" {
			t.Errorf("status JSON gained unexpected top-level field %q", key)
		}
	}
	for _, required := range []string{"workspace", "hero_dir", "specs"} {
		if _, ok := raw[required]; !ok {
			t.Errorf("status JSON lost top-level field %q", required)
		}
	}
	var rawSpecs []map[string]json.RawMessage
	if err := json.Unmarshal(raw["specs"], &rawSpecs); err != nil {
		t.Fatalf("decode raw status specs: %v", err)
	}
	for _, item := range rawSpecs {
		if len(item) != 5 {
			t.Errorf("status JSON per-spec shape changed: keys=%v", item)
		}
		for _, required := range []string{"slug", "title", "type", "status", "horizon"} {
			if _, ok := item[required]; !ok {
				t.Errorf("status JSON spec lost field %q: %v", required, item)
			}
		}
	}

	var payload decodedStatusPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode status JSON: %v\n%s", err, output)
	}
	if payload.Workspace == "" || payload.HeroDir == "" {
		t.Errorf("status JSON lost top-level workspace fields: %+v", payload)
	}
	return payload
}
