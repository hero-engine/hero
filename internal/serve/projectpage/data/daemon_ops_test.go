package data

import (
	"testing"
	"time"
)

func TestLoadDaemonOps_NoSnapshot(t *testing.T) {
	out := LoadDaemonOps(DaemonOpsInputs{})
	if out.HasSnapshot {
		t.Errorf("HasSnapshot = true, want false on nil snapshot")
	}
}

func TestLoadDaemonOps_FormatsUptime(t *testing.T) {
	now := time.Date(2026, 5, 19, 14, 30, 0, 0, time.UTC)
	out := LoadDaemonOps(DaemonOpsInputs{Snapshot: &DaemonOpsSnapshot{
		PID: 1234, Port: 7437, Version: "v1.2.3",
		StartedAt: now, UptimeSeconds: 3725, ProjectCount: 4,
	}})
	if !out.HasSnapshot {
		t.Fatalf("HasSnapshot = false")
	}
	if out.PID != 1234 || out.Port != 7437 {
		t.Errorf("PID/Port mismatch: %d %d", out.PID, out.Port)
	}
	// 3725s = 1h 2m
	if out.UptimePretty != "1h 2m" {
		t.Errorf("UptimePretty = %q, want \"1h 2m\"", out.UptimePretty)
	}
	if out.StartedAtPretty == "" {
		t.Errorf("StartedAtPretty empty, want formatted")
	}
}

func TestFormatUptime_Buckets(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0s"},
		{45, "45s"},
		{125, "2m 5s"},
		{3601, "1h 0m"},
		{86400*2 + 3600*3, "2d 3h"},
	}
	for _, c := range cases {
		got := formatUptime(c.in)
		if got != c.want {
			t.Errorf("formatUptime(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
