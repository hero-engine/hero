package metrics

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hero-engine/hero/internal/feed"
)

func TestHoursSaved_PublishedFormula(t *testing.T) {
	c := Counts{
		ProposalsMergedNoEdit:   100,
		ProposalsMergedWithEdit: 40,
		AutoImportedSpecs:       200,
		AutoDiagnosedBugs:       10,
		AutoReviewedSpecs:       20,
		KnowledgeInjections:     500,
		ScheduledJobsExecuted:   12,
	}
	k := DefaultCoefficients()
	got := HoursSaved(c, k)
	// 100*1.5 + 40*0.5 + 200*0.1 + 10*1.0 + 20*0.5 + 500*0.05 + 12*0.25
	// = 150 + 20 + 20 + 10 + 10 + 25 + 3 = 238
	want := 238.0
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("HoursSaved = %v, want %v", got, want)
	}
}

func TestHoursSaved_EmptyIsZero(t *testing.T) {
	if got := HoursSaved(Counts{}, DefaultCoefficients()); got != 0 {
		t.Errorf("empty HoursSaved = %v, want 0", got)
	}
}

func TestMoneyChain(t *testing.T) {
	k := DefaultCoefficients()
	hours := 340.0
	dollars := DollarsSaved(hours, k)
	if math.Abs(dollars-51000) > 1e-6 {
		t.Errorf("DollarsSaved(340) = %v, want 51000", dollars)
	}
	net := NetValue(hours, 1140.0, k)
	if math.Abs(net-49860) > 1e-6 {
		t.Errorf("NetValue = %v, want 49860", net)
	}
	roi := ROIMultiple(net, 1140.0)
	// 49860 / 1140 = 43.736…
	if math.Abs(roi-43.7368) > 0.01 {
		t.Errorf("ROIMultiple = %v, want ~43.7", roi)
	}
}

func TestROIMultiple_ZeroSpendSentinel(t *testing.T) {
	if got := ROIMultiple(100, 0); got != 0 {
		t.Errorf("ROIMultiple(_, 0) = %v, want 0 sentinel", got)
	}
	if got := FormatROIMultiple(0); got != "—" {
		t.Errorf("FormatROIMultiple(0) = %q, want em-dash", got)
	}
}

func TestFormatROIMultiple_Buckets(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "—"},
		{3.214, "3.21×"},
		{9.99, "9.99×"},
		{10, "10.0×"},
		{44.0, "44.0×"},
		{999.4, "999.4×"},
		{1000, "999×"},
		{50000, "999×"},
	}
	for _, c := range cases {
		got := FormatROIMultiple(c.in)
		if got != c.want {
			t.Errorf("FormatROIMultiple(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDollars(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "$0"},
		{500, "$500"},
		{1000, "$1.0K"},
		{49900, "$49.9K"},
		{100000, "$100K"},
		{1140, "$1.1K"},
	}
	for _, c := range cases {
		got := FormatDollars(c.in)
		if got != c.want {
			t.Errorf("FormatDollars(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLoadCounts_EmptyHeroDir(t *testing.T) {
	c := LoadCounts("", Last(24*time.Hour))
	if c != (Counts{}) {
		t.Errorf("LoadCounts(\"\") = %+v, want zero value", c)
	}
}

func TestLoadCounts_MissingLog(t *testing.T) {
	dir := t.TempDir()
	c := LoadCounts(dir, Last(24*time.Hour))
	if c != (Counts{}) {
		t.Errorf("LoadCounts(no log) = %+v, want zero value", c)
	}
}

func TestLoadCounts_ReadsLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "events.log")
	if err := os.WriteFile(logPath, nil, 0o644); err != nil {
		t.Fatalf("seed log: %v", err)
	}
	now := time.Now().UTC()
	for _, e := range []feed.FeedEvent{
		{Timestamp: now, Type: "delivery_complete", Message: "1"},
		{Timestamp: now, Type: "delivery_complete", Message: "2"},
		{Timestamp: now, Type: "spec_created", Message: "3"},
		{Timestamp: now, Type: "decision_made", Message: "4"},
	} {
		if err := feed.AppendEvent(logPath, e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	c := LoadCounts(dir, Last(24*time.Hour))
	if c.SpecsDelivered != 2 {
		t.Errorf("SpecsDelivered = %d, want 2", c.SpecsDelivered)
	}
	if c.AutoImportedSpecs != 1 {
		t.Errorf("AutoImportedSpecs = %d, want 1", c.AutoImportedSpecs)
	}
	if c.AutoDiagnosedBugs != 1 {
		t.Errorf("AutoDiagnosedBugs = %d, want 1", c.AutoDiagnosedBugs)
	}
}
