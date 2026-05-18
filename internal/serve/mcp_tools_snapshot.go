package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/refs"
	"github.com/hero-engine/hero/internal/snapshot"
	"github.com/hero-engine/hero/internal/spec"
)

// toolSnapshot implements the hero_snapshot MCP tool. Supports
// rendering the live snapshot, an archived snapshot at a given
// date, the archive history list, and triggering a manual archive
// — all behind a single tool name with flags per the design.
func (s *MCPServer) toolSnapshot(args map[string]interface{}) (string, error) {
	cfg, err := config.Load(s.projectRoot)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	heroDir := s.heroDir
	if heroDir == "" {
		heroDir = cfg.HeroDir(s.projectRoot)
	}

	// history: true returns the enumerated archive list.
	if v, _ := args["history"].(bool); v {
		archives, err := snapshot.List(heroDir)
		if err != nil {
			return "", err
		}
		return formatSnapshotHistory(archives)
	}

	// archive: true writes a manual archive and returns the record.
	if v, _ := args["archive"].(bool); v {
		label, _ := args["label"].(string)
		archive := cfg.SnapshotArchive()
		result, err := snapshot.Project(snapshot.ProjectOptions{
			ProjectRoot:        s.projectRoot,
			HeroDir:            heroDir,
			ProjectName:        filepath.Base(s.projectRoot),
			NextMDPath:         "",
			AgentsMDPath:       "",
			ManualArchive:      true,
			ManualArchiveLabel: label,
			ArchiveConfig: snapshot.ArchiveConfig{
				StalenessCutoff:   archive.StalenessCutoff,
				MilestonesEnabled: cfg.SnapshotMilestonesEnabled(),
				ReleaseTagPattern: archive.ReleaseTagPattern,
				Retention:         archive.Retention,
				RetentionCount:    archive.RetentionCount,
			},
		})
		if err != nil {
			return "", err
		}
		return formatSnapshotArchiveResult(result)
	}

	// at: <date> returns the matching archive body.
	if at, _ := args["at"].(string); at != "" && strings.ToLower(at) != "latest" {
		rec, err := snapshot.FindArchive(heroDir, at)
		if err != nil {
			return "", err
		}
		if rec == nil {
			return "", fmt.Errorf("no archive found for %q", at)
		}
		data, err := os.ReadFile(rec.Path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Default path: render the live snapshot. Build from the same
	// graph the CLI uses; respect optional section / surface filters.
	allSpecs, _ := spec.Discover(heroDir)
	override, _ := snapshot.LoadOverride(heroDir)
	snap, err := snapshot.Build(snapshot.BuildOptions{
		ProjectRoot: s.projectRoot,
		HeroDir:     heroDir,
		ProjectName: filepath.Base(s.projectRoot),
	}, allSpecs, override, nil)
	if err != nil {
		return "", err
	}

	if compact, _ := args["compact"].(bool); compact {
		// Two-tier: short summary + a ref the agent can pass to
		// hero_expand. We reuse the existing context ref-kind since
		// snapshot bodies are first-class context, not specs.
		full, err := snapshot.Render(snap, snapshot.FormatMarkdown)
		if err != nil {
			return "", err
		}
		summary := fmt.Sprintf("Project snapshot: %d surfaces, %d shipping, %d in flight, %d blockers.",
			len(snap.Surfaces),
			countShipping(snap),
			countInFlight(snap),
			len(snap.Blockers))
		envText, regErr := s.registerRef(refs.KindContext, "snapshot", "full",
			map[string]any{"kind": "snapshot"},
			string(full), fmt.Sprintf("snapshot-%d", snap.GeneratedAt.Unix()),
			summary)
		if regErr != nil {
			// Fall back to inline if ref-store fails.
			return summary + "\n\n" + string(full), nil
		}
		return envText, nil
	}

	if section, _ := args["section"].(string); section != "" && section != "all" {
		out, err := renderSnapshotSectionForMCP(snap, section)
		if err != nil {
			return "", err
		}
		return out, nil
	}

	data, err := snapshot.Render(snap, snapshot.FormatMarkdown)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func countShipping(s *snapshot.Snapshot) int {
	if s == nil {
		return 0
	}
	n := 0
	for _, sf := range s.Surfaces {
		switch sf.Stage {
		case snapshot.StageShippingV1, snapshot.StageShipped:
			n++
		}
	}
	return n
}

func countInFlight(s *snapshot.Snapshot) int {
	if s == nil {
		return 0
	}
	n := 0
	for _, a := range s.Assignments {
		if a.Spec != nil && a.Spec.Status == spec.StatusDelivering {
			n++
		}
	}
	return n
}

func formatSnapshotHistory(archives []snapshot.ArchiveRecord) (string, error) {
	type row struct {
		Date      string `json:"date"`
		Trigger   string `json:"trigger"`
		Label     string `json:"label,omitempty"`
		GitCommit string `json:"git_commit,omitempty"`
		Path      string `json:"path"`
	}
	rows := make([]row, 0, len(archives))
	for _, a := range archives {
		rows = append(rows, row{
			Date:      a.Date,
			Trigger:   string(a.Trigger),
			Label:     a.Label,
			GitCommit: a.GitCommit,
			Path:      a.Path,
		})
	}
	data, err := json.MarshalIndent(map[string]interface{}{
		"archives": rows,
		"count":    len(rows),
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatSnapshotArchiveResult(result *snapshot.ProjectResult) (string, error) {
	if result == nil {
		return "{}", nil
	}
	type row struct {
		Date      string `json:"date"`
		Trigger   string `json:"trigger"`
		Label     string `json:"label,omitempty"`
		GitCommit string `json:"git_commit,omitempty"`
		Path      string `json:"path"`
	}
	rows := make([]row, 0, len(result.Archives))
	for _, a := range result.Archives {
		rows = append(rows, row{
			Date:      a.Date,
			Trigger:   string(a.Trigger),
			Label:     a.Label,
			GitCommit: a.GitCommit,
			Path:      a.Path,
		})
	}
	data, err := json.MarshalIndent(map[string]interface{}{
		"written":   rows,
		"count":     len(rows),
		"durationMS": result.DurationMS,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func renderSnapshotSectionForMCP(s *snapshot.Snapshot, section string) (string, error) {
	full, err := snapshot.Render(s, snapshot.FormatMarkdown)
	if err != nil {
		return "", err
	}
	body := string(full)
	headers := map[string]string{
		"surfaces":    "## Surfaces\n",
		"initiatives": "## Active initiatives\n",
		"recent":      "## Recently completed",
		"next":        "## Next up across surfaces\n",
		"risks":       "## Open risks & blockers\n",
		"health":      "## Snapshot health\n",
	}
	start, ok := headers[strings.ToLower(section)]
	if !ok {
		return "", fmt.Errorf("unknown section %q", section)
	}
	idx := strings.Index(body, start)
	if idx < 0 {
		return "", fmt.Errorf("section %q not present", section)
	}
	rest := body[idx:]
	endIdx := strings.Index(rest[len(start):], "\n## ")
	if endIdx < 0 {
		return rest, nil
	}
	return rest[:len(start)+endIdx+1], nil
}
