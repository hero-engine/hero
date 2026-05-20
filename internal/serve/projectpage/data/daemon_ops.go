package data

import (
	"fmt"
	"time"
)

// DaemonOpsInputs is the per-request bundle for the Daemon Ops section.
// Snapshot is the daemon-status view (PID/port/uptime/version/served-
// project count) the aggregate handler pulls from the same internal
// path /api/status uses. Empty snapshot is allowed — the partial
// renders "daemon status unavailable" in that case.
type DaemonOpsInputs struct {
	Snapshot *DaemonOpsSnapshot
}

// DaemonOpsSnapshot is the projectpage-package-local view of the
// daemon-status response. Defined here (rather than re-importing
// internal/serve.DaemonStatusResponse) so the data loader stays
// import-clean of the parent serve package and the unit tests can
// construct fixtures without dragging the daemon in.
type DaemonOpsSnapshot struct {
	PID           int
	Port          int
	Version       string
	StartedAt     time.Time
	UptimeSeconds int64
	ProjectCount  int
}

// DaemonOps is what the partial renders.
type DaemonOps struct {
	HasSnapshot   bool
	PID           int
	Port          int
	Version       string
	StartedAt     time.Time
	StartedAtPretty string
	UptimePretty  string
	ProjectCount  int
}

// LoadDaemonOps shapes the snapshot into the view struct. Always
// succeeds; missing snapshot yields HasSnapshot=false.
func LoadDaemonOps(in DaemonOpsInputs) DaemonOps {
	if in.Snapshot == nil {
		return DaemonOps{}
	}
	s := in.Snapshot
	out := DaemonOps{
		HasSnapshot:  true,
		PID:          s.PID,
		Port:         s.Port,
		Version:      s.Version,
		StartedAt:    s.StartedAt,
		ProjectCount: s.ProjectCount,
		UptimePretty: formatUptime(s.UptimeSeconds),
	}
	if !s.StartedAt.IsZero() {
		out.StartedAtPretty = s.StartedAt.Format("2006-01-02 15:04:05")
	}
	return out
}

// formatUptime renders a seconds count as a compact human string —
// "3h 12m" / "4d 2h" / "47s". Zero seconds renders as "0s".
func formatUptime(secs int64) string {
	if secs <= 0 {
		return "0s"
	}
	d := secs / 86400
	secs -= d * 86400
	h := secs / 3600
	secs -= h * 3600
	m := secs / 60
	s := secs - m*60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh", d, h)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
