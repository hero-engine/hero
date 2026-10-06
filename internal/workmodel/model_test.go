package workmodel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

var testNow = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type corpus struct {
	t    *testing.T
	root string
}

func newCorpus(t *testing.T) *corpus {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hero"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &corpus{t: t, root: root}
}

// write puts a spec at .hero/<rel>/spec.md and returns its path.
func (c *corpus) write(rel, content string) string {
	c.t.Helper()
	path := filepath.Join(c.root, ".hero", rel, "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		c.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		c.t.Fatal(err)
	}
	return path
}

func (c *corpus) build() ([]*spec.Spec, map[string]Item) {
	c.t.Helper()
	specs, err := spec.Discover(filepath.Join(c.root, ".hero"))
	if err != nil {
		c.t.Fatal(err)
	}
	items := Build(specs, Options{Now: testNow, RecentDays: 14, Root: c.root})
	bySlug := map[string]Item{}
	for _, it := range items {
		bySlug[it.Slug] = it
	}
	return specs, bySlug
}

const designedBody = "\n## Changes\n\n1. Do it.\n\n## Acceptance Criteria\n\n- **AC-1:** THE SYSTEM SHALL work.\n"

func fm(slug, typ, status string, extra string) string {
	return "---\ntitle: \"" + slug + ": title\"\nslug: " + slug + "\ntype: " + typ + "\nstatus: " + status + "\ncreated: 2026-09-01\n" + extra + "---\n\n# " + slug + "\n"
}

func seed(c *corpus) {
	c.write("planning/features/stub", fm("stub", "feature", "planning", "priority: P1\n"))
	c.write("planning/features/ready", fm("ready", "feature", "planning", "priority: high\nseverity: moderate\nsize: small\n")+designedBody)
	c.write("planning/features/blocked", fm("blocked", "feature", "planning", "depends-on: [ready]\n")+designedBody)
	c.write("planning/features/active", fm("active", "feature", "delivering", "")+designedBody)
	c.write("planning/bugs/undiagnosed", fm("undiagnosed", "bug", "planning", "severity: critical\ntracker_id: PROJ-9\n"))
	c.write("planning/bugs/diagnosed", fm("diagnosed", "bug", "planning", "")+"\n## Root Cause\n\nX.\n\n## Fix\n\nY.\n")
	c.write("specs/recent", fm("recent", "feature", "completed", "completed_at: 2026-10-01T00:00:00Z\n")+designedBody)
	c.write("specs/old", fm("old", "feature", "completed", "completed_at: 2026-06-01T00:00:00Z\n")+designedBody)
	c.write("specs/gone", fm("gone", "feature", "superseded", "")+designedBody)
	c.write("planning/initiatives/init", fm("init", "initiative", "planning", "child:\n  - ready\n  - recent\n  - choice\n"))
	c.write("planning/initiatives/init/choice", fm("choice", "decision", "proposed", "parent: init\n")+"\n## Decision\n\nYes.\n")
	c.write("knowledge/conventions/naming", fm("naming", "convention", "active", ""))
	c.write("planning/decisions/loose", fm("loose", "decision", "proposed", "")+"\n## Decision\n\nNo parent.\n")
}

// AC-1
func TestBuildIncludesOnlyWorkSpecsInPathOrder(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	specs, items := c.build()
	for _, slug := range []string{"stub", "ready", "blocked", "active", "undiagnosed", "diagnosed", "recent", "old", "gone", "init", "choice"} {
		if _, ok := items[slug]; !ok {
			t.Errorf("missing work item %s", slug)
		}
	}
	for _, slug := range []string{"naming", "loose"} {
		if _, ok := items[slug]; ok {
			t.Errorf("non-work spec %s included", slug)
		}
	}
	ordered := Build(specs, Options{Now: testNow, Root: c.root})
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1].Path > ordered[i].Path {
			t.Fatalf("not path-ordered at %d: %s > %s", i, ordered[i-1].Path, ordered[i].Path)
		}
	}
	if items["ready"].Title != "ready: title" {
		t.Errorf("title = %q (quotes must be decoded)", items["ready"].Title)
	}
	if items["ready"].Path != ".hero/planning/features/ready/spec.md" {
		t.Errorf("path not repo-relative: %s", items["ready"].Path)
	}
}

// AC-2
func TestLanes(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	_, items := c.build()
	want := map[string]string{
		"stub":        LaneNone,
		"ready":       LaneReady,
		"blocked":     LaneDesigned,
		"active":      LaneInProgress,
		"undiagnosed": LaneNone,
		"diagnosed":   LaneReady,
		"recent":      LaneRecentlyDone,
		"old":         LaneNone,
		"gone":        LaneNone,
		"init":        LaneInProgress, // a child (recent) is finished
		"choice":      LaneReady,
	}
	for slug, lane := range want {
		if got := items[slug].Lane; got != lane {
			t.Errorf("%s lane = %s, want %s", slug, got, lane)
		}
	}
}

// AC-2: RecentDays bounds recently_done.
func TestRecentDaysWindow(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	specs, _ := c.build()
	items := Build(specs, Options{Now: testNow, RecentDays: 3, Root: c.root})
	for _, it := range items {
		if it.Slug == "recent" && it.Lane != LaneNone {
			t.Fatalf("recent outside a 3-day window should be none, got %s", it.Lane)
		}
	}
}

// AC-3
func TestVerifyState(t *testing.T) {
	c := newCorpus(t)
	ledger := "\n## Completion Ledger\n\n### Acceptance Criteria\n\n| # | Criterion | Status | Note |\n|---|---|---|---|\n| 1 | AC-1 | DONE | ok |\n\n### Changes\n\n| # | Item | Status | Note |\n|---|---|---|---|\n| 1 | Do it | DONE | ok |\n"
	passed := c.write("specs/passed", fm("passed", "feature", "completed", "completed_at: 2026-10-01T00:00:00Z\n")+designedBody+ledger)
	c.write("specs/noaudit", fm("noaudit", "feature", "completed", "completed_at: 2026-10-01T00:00:00Z\n")+designedBody+ledger)
	held := c.write("planning/features/held", fm("held", "feature", "delivering", "")+designedBody)
	c.write("planning/features/fresh", fm("fresh", "feature", "planning", "")+designedBody)
	c.write("planning/features/reg", fm("reg", "feature", "regressed", "")+designedBody)
	writeAudit := func(specPath, slug, verdict string) {
		time.Sleep(10 * time.Millisecond) // audit must postdate the spec
		body := "# Delivery audit — " + slug + "\n\n**Verdict:** " + verdict + "\n**Surface:** clean\n"
		if err := os.WriteFile(filepath.Join(filepath.Dir(specPath), "delivery-audit.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeAudit(passed, "passed", "SHIP")
	writeAudit(held, "held", "HOLD")
	c.write("planning/initiatives/i", fm("i", "initiative", "planning", "child:\n  - d\n"))
	c.write("planning/initiatives/i/d", fm("d", "decision", "accepted", "parent: i\n")+"\n## Decision\n\nYes.\n")
	_, items := c.build()
	for slug, want := range map[string]string{"passed": VerifyPassed, "noaudit": VerifyPartial, "held": VerifyFailed, "fresh": VerifyNotRun, "reg": VerifyFailed} {
		if items[slug].Verify == nil || items[slug].Verify.State != want {
			t.Errorf("%s verify = %+v, want %s", slug, items[slug].Verify, want)
		}
	}
	if a := items["passed"].Verify.Audit; a == nil || *a != "ship" {
		t.Errorf("passed audit = %v", a)
	}
	if items["d"].Verify != nil {
		t.Errorf("decision verify should be null, got %+v", items["d"].Verify)
	}
}

// AC-4
func TestNormalizationAndProgress(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	_, items := c.build()
	check := func(name string, got *string, want string) {
		t.Helper()
		if want == "" {
			if got != nil {
				t.Errorf("%s = %q, want null", name, *got)
			}
			return
		}
		if got == nil || *got != want {
			t.Errorf("%s = %v, want %s", name, got, want)
		}
	}
	check("stub priority", items["stub"].Priority, "high")
	check("ready priority", items["ready"].Priority, "high")
	check("ready severity", items["ready"].Severity, "medium")
	check("undiagnosed severity", items["undiagnosed"].Severity, "critical")
	check("blocked priority", items["blocked"].Priority, "")
	check("ready size", items["ready"].Size, "small")
	if tr := items["undiagnosed"].Tracker; tr == nil || tr.ID != "PROJ-9" || tr.URL != nil {
		t.Errorf("tracker = %+v", tr)
	}
	if p := items["init"].Progress; p == nil || p.Done != 1 || p.Total != 3 {
		t.Errorf("init progress = %+v, want 1/3", p)
	}
	if items["ready"].Progress != nil {
		t.Error("non-initiative progress must be null")
	}
	if items["choice"].Parent == nil || *items["choice"].Parent != "init" {
		t.Errorf("choice parent = %v", items["choice"].Parent)
	}
	// JSON shape: null fields serialize as null, not omitted.
	data, _ := json.Marshal(items["blocked"])
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	for _, key := range []string{"priority", "severity", "progress", "completed_at", "tracker", "next"} {
		if v, ok := m[key]; !ok || v != nil {
			t.Errorf("JSON %s = %v (present=%v), want null", key, v, ok)
		}
	}
}

// AC-5
func TestRevisionChangesOnlyWithInputs(t *testing.T) {
	c := newCorpus(t)
	seed(c)
	_, first := c.build()
	_, again := c.build()
	for slug, it := range first {
		if again[slug].Revision != it.Revision || len(it.Revision) != 16 {
			t.Fatalf("%s revision unstable or wrong length: %s vs %s", slug, it.Revision, again[slug].Revision)
		}
	}
	// A dependency's status change moves the dependent's revision only.
	c.write("planning/features/ready", fm("ready", "feature", "delivering", "priority: high\nseverity: moderate\nsize: small\n")+designedBody)
	_, after := c.build()
	if after["blocked"].Revision == first["blocked"].Revision {
		t.Error("blocked revision unchanged after its dependency's status changed")
	}
	if after["ready"].Revision == first["ready"].Revision {
		t.Error("ready revision unchanged after its own file changed")
	}
	if after["stub"].Revision != first["stub"].Revision {
		t.Error("unrelated stub revision changed")
	}
	// An audit report change moves the revision.
	auditPath := filepath.Join(c.root, ".hero", "planning", "features", "active", "delivery-audit.md")
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(auditPath, []byte("# Delivery audit — active\n\n**Verdict:** HOLD\n"), 0o644)
	_, audited := c.build()
	if audited["active"].Revision == first["active"].Revision {
		t.Error("active revision unchanged after an audit report appeared")
	}
}
