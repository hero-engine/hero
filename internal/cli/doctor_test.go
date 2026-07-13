package cli

import (
	"strings"
	"testing"
)

func TestBuildDoctorReport(t *testing.T) {
	base := doctorInfo{
		exe:           "/Users/dev/go/bin/hero",
		exeResolved:   "/Users/dev/go/bin/hero",
		pathHero:      "/Users/dev/go/bin/hero",
		binaryVersion: "0.14.0",
		binarySchema:  "4",
		graphSchema:   "4",
		heroDir:       "/repo/.hero",
	}

	t.Run("reports exe, version, both schemas", func(t *testing.T) {
		report := buildDoctorReport(base)
		for _, want := range []string{
			"/Users/dev/go/bin/hero", // exe path
			"v0.14.0",                // binary version (displayVersion)
			"binary schema:   4",     // binary schema
			"graph schema:    4",     // graph schema
		} {
			if !strings.Contains(report, want) {
				t.Errorf("report missing %q:\n%s", want, report)
			}
		}
	})

	t.Run("agree verdict", func(t *testing.T) {
		report := buildDoctorReport(base)
		if !strings.Contains(report, "Verdict: OK") {
			t.Errorf("expected agree verdict:\n%s", report)
		}
	})

	t.Run("binary older than graph verdict", func(t *testing.T) {
		info := base
		info.binarySchema = "2"
		info.graphSchema = "4"
		report := buildDoctorReport(info)
		if !strings.Contains(report, "OLDER than the workspace graph") {
			t.Errorf("expected binary-older verdict:\n%s", report)
		}
		if !strings.Contains(report, "will NOT help") {
			t.Errorf("binary-older verdict must warn `hero upgrade` won't help:\n%s", report)
		}
	})

	t.Run("binary newer than graph verdict", func(t *testing.T) {
		info := base
		info.binarySchema = "4"
		info.graphSchema = "2"
		report := buildDoctorReport(info)
		if !strings.Contains(report, "NEWER than the workspace graph") {
			t.Errorf("expected binary-newer verdict:\n%s", report)
		}
	})

	t.Run("PATH divergence fires the flag", func(t *testing.T) {
		info := base
		// Simulate a harness resolving `hero` to a different binary than
		// the one running (the Defect-2 signal).
		info.pathHero = "/usr/local/bin/hero"
		info.pathHeroResolved = "/usr/local/bin/hero"
		report := buildDoctorReport(info)
		if !strings.Contains(report, "DIFFERENT binary than the one running") {
			t.Errorf("expected PATH-divergence warning:\n%s", report)
		}
	})

	t.Run("no PATH divergence when same binary", func(t *testing.T) {
		report := buildDoctorReport(base) // pathHero == exe
		if strings.Contains(report, "DIFFERENT binary than the one running") {
			t.Errorf("should not warn when PATH hero matches running binary:\n%s", report)
		}
	})

	t.Run("degrades gracefully outside a workspace", func(t *testing.T) {
		info := base
		info.heroDir = ""
		info.graphSchema = ""
		report := buildDoctorReport(info)
		if !strings.Contains(report, "no hero workspace found") {
			t.Errorf("expected graceful no-workspace message:\n%s", report)
		}
		if !strings.Contains(report, "cannot compare") {
			t.Errorf("expected cannot-compare verdict outside workspace:\n%s", report)
		}
	})
}
