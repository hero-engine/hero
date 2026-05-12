package cli

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/spec"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a self-contained HTML report of the workspace",
	Long: `Produces a standalone HTML file summarizing the hero workspace state:
  - Spec counts by status and type
  - In-flight work summary with age
  - Coverage: specs with tracker links, claims, changes sections
  - Velocity: completed specs over time
  - Stale spec warnings
  - Knowledge base summary

The report is saved to .hero/reports/report.html and can be opened in any browser.`,
	RunE: runReport,
}

var (
	reportOutput string
	reportOpen   bool
)

func init() {
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "output path (default: .hero/reports/report.html)")
	reportCmd.Flags().BoolVar(&reportOpen, "open", false, "open the report in the default browser after generating")
}

// reportData holds all data needed to render the HTML report.
type reportData struct {
	ProjectName   string
	GeneratedAt   string
	TotalSpecs    int
	InFlight      int
	Completed     int
	StatusCounts  []statusCount
	TypeCounts    []statusCount
	StaleSpecs    []reportSpec
	InFlightSpecs []reportSpec
	ClaimedSpecs  []reportSpec
	RecentSpecs   []reportSpec
	StaleDays     int
	TrackerType   string
	TrackerLinked int
	TrackerTotal  int
	CoverageData  coverageData
	VelocityData  []velocityMonth
}

type statusCount struct {
	Label string
	Count int
	Color string
}

type reportSpec struct {
	Slug       string
	Title      string
	Type       string
	Status     string
	Age        string
	ClaimedBy  string
	TrackerID  string
	HasChanges bool
}

type coverageData struct {
	WithTracker int
	WithClaim   int
	WithChanges int
	Total       int
}

type velocityMonth struct {
	Month     string
	Completed int
}

func runReport(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	specs, err := spec.Discover(heroDir)
	if err != nil {
		return fmt.Errorf("discovering specs: %w", err)
	}

	data := buildReportData(specs, cfg, projectRoot)

	// Determine output path
	outPath := reportOutput
	if outPath == "" {
		reportsDir := filepath.Join(heroDir, "reports")
		if err := os.MkdirAll(reportsDir, 0o755); err != nil {
			return fmt.Errorf("creating reports directory: %w", err)
		}
		outPath = filepath.Join(reportsDir, "report.html")
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating report file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"coveragePct": func(part, total int) int {
			if total == 0 {
				return 0
			}
			return (part * 100) / total
		},
	}).Parse(reportTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("rendering report: %w", err)
	}

	fmt.Printf("Report generated: %s\n", outPath)

	if reportOpen {
		if err := openBrowser(outPath); err != nil {
			fmt.Printf("Could not open browser: %v\n", err)
			fmt.Printf("Open the file manually: %s\n", outPath)
		}
	}

	return nil
}

func buildReportData(specs []*spec.Spec, cfg config.Config, projectRoot string) reportData {
	staleDays := 14
	if cfg.Team != nil && cfg.Team.StaleDays > 0 {
		staleDays = cfg.Team.StaleDays
	}

	data := reportData{
		ProjectName: filepath.Base(projectRoot),
		GeneratedAt: time.Now().Format("2006-01-02 15:04"),
		TotalSpecs:  len(specs),
		StaleDays:   staleDays,
	}

	// Status counts for work specs
	statusMap := map[string]int{}
	// Type counts for knowledge specs
	typeMap := map[string]int{}

	for _, s := range specs {
		if s.IsKnowledge() {
			typeMap[string(s.Type)]++
		} else {
			statusMap[string(s.Status)]++
		}

		// In-flight
		if s.IsWorkSpec() && s.IsInFlight() {
			data.InFlight++
			age := time.Since(s.ModifiedAt)
			rs := reportSpec{
				Slug:       s.Slug,
				Title:      s.Title,
				Type:       string(s.Type),
				Status:     string(s.Status),
				Age:        formatAge(age),
				ClaimedBy:  s.ClaimedBy,
				TrackerID:  s.TrackerID,
				HasChanges: len(s.FilesTouched) > 0,
			}
			data.InFlightSpecs = append(data.InFlightSpecs, rs)

			if age > time.Duration(staleDays)*24*time.Hour {
				data.StaleSpecs = append(data.StaleSpecs, rs)
			}
		}

		// Completed
		if s.Status == spec.StatusCompleted {
			data.Completed++
		}

		// Claimed
		if s.ClaimedBy != "" {
			data.ClaimedSpecs = append(data.ClaimedSpecs, reportSpec{
				Slug:      s.Slug,
				Title:     s.Title,
				Type:      string(s.Type),
				Status:    string(s.Status),
				ClaimedBy: s.ClaimedBy,
			})
		}

		// Coverage
		if s.IsWorkSpec() {
			data.CoverageData.Total++
			if s.TrackerID != "" {
				data.CoverageData.WithTracker++
			}
			if s.ClaimedBy != "" {
				data.CoverageData.WithClaim++
			}
			if len(s.FilesTouched) > 0 {
				data.CoverageData.WithChanges++
			}
		}

		// Tracker
		if s.TrackerID != "" {
			data.TrackerLinked++
		}
	}

	data.TrackerTotal = len(specs)
	if cfg.Tracker != nil && cfg.Tracker.Type != "none" && cfg.Tracker.Type != "" {
		data.TrackerType = cfg.Tracker.Type
	}

	// Status counts
	statusColors := map[string]string{
		"planning":   "#ffc107",
		"in-review":  "#17a2b8",
		"delivering": "#4a9eff",
		"completed":  "#28a745",
	}
	for _, key := range []string{"planning", "in-review", "delivering", "completed"} {
		if c := statusMap[key]; c > 0 {
			data.StatusCounts = append(data.StatusCounts, statusCount{
				Label: key,
				Count: c,
				Color: statusColors[key],
			})
		}
	}

	// Type counts
	typeColors := map[string]string{
		"convention": "#6f42c1",
		"decision":   "#e83e8c",
		"rule":       "#fd7e14",
		"external":   "#20c997",
		"context":    "#6c757d",
		"note":       "#adb5bd",
	}
	for _, key := range []string{"convention", "decision", "rule", "external", "context", "note"} {
		if c := typeMap[key]; c > 0 {
			data.TypeCounts = append(data.TypeCounts, statusCount{
				Label: key,
				Count: c,
				Color: typeColors[key],
			})
		}
	}

	// Velocity — completed specs by month
	monthMap := map[string]int{}
	for _, s := range specs {
		if s.Status == spec.StatusCompleted && !s.ModifiedAt.IsZero() {
			key := s.ModifiedAt.Format("2006-01")
			monthMap[key]++
		}
	}
	var months []string
	for m := range monthMap {
		months = append(months, m)
	}
	sort.Strings(months)
	// Keep last 6 months
	if len(months) > 6 {
		months = months[len(months)-6:]
	}
	for _, m := range months {
		data.VelocityData = append(data.VelocityData, velocityMonth{
			Month:     m,
			Completed: monthMap[m],
		})
	}

	// Recent specs (last 10 by modified date)
	sorted := make([]*spec.Spec, len(specs))
	copy(sorted, specs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ModifiedAt.After(sorted[j].ModifiedAt)
	})
	limit := 10
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for _, s := range sorted[:limit] {
		data.RecentSpecs = append(data.RecentSpecs, reportSpec{
			Slug:   s.Slug,
			Title:  s.Title,
			Type:   string(s.Type),
			Status: string(s.Status),
			Age:    formatAge(time.Since(s.ModifiedAt)),
		})
	}

	return data
}

var reportTemplate = `<!DOCTYPE html>
<!-- Hero Report | Generated: {{.GeneratedAt}} -->
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hero Report — {{.ProjectName}}</title>
    <style>
        :root {
            --bg: #ffffff;
            --bg2: #f8f9fa;
            --bg3: #e9ecef;
            --text: #212529;
            --text2: #6c757d;
            --border: #dee2e6;
            --primary: #4a9eff;
            --success: #28a745;
            --warning: #ffc107;
            --danger: #dc3545;
            --shadow: 0 1px 3px rgba(0,0,0,0.1);
            --radius: 8px;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            font-size: 14px; line-height: 1.6; color: var(--text);
            background: var(--bg2); padding: 32px;
        }
        .container { max-width: 1000px; margin: 0 auto; }
        .header {
            background: var(--bg); border-radius: var(--radius);
            padding: 32px; margin-bottom: 24px; box-shadow: var(--shadow);
        }
        .header h1 { font-size: 24px; font-weight: 700; }
        .header .meta { color: var(--text2); margin-top: 4px; font-size: 13px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
        .card {
            background: var(--bg); border-radius: var(--radius);
            padding: 24px; box-shadow: var(--shadow);
        }
        .card h2 { font-size: 16px; font-weight: 600; margin-bottom: 16px; color: var(--text); }
        .metric { text-align: center; }
        .metric .number { font-size: 36px; font-weight: 700; color: var(--primary); }
        .metric .label { font-size: 12px; color: var(--text2); text-transform: uppercase; letter-spacing: 0.5px; margin-top: 4px; }
        table { width: 100%; border-collapse: collapse; }
        th { font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; color: var(--text2); font-weight: 600; text-align: left; padding: 8px 12px; border-bottom: 2px solid var(--border); }
        td { padding: 8px 12px; border-bottom: 1px solid var(--border); font-size: 13px; }
        tr:hover td { background: var(--bg2); }
        .badge {
            display: inline-block; padding: 2px 8px; border-radius: 12px;
            font-size: 11px; font-weight: 500;
        }
        .badge-planning { background: #fff3cd; color: #856404; }
        .badge-in-review { background: #d1ecf1; color: #0c5460; }
        .badge-delivering { background: #cce5ff; color: #004085; }
        .badge-completed { background: #d4edda; color: #155724; }
        .badge-convention { background: #e8daef; color: #4a235a; }
        .badge-decision { background: #fadbd8; color: #78281f; }
        .badge-rule { background: #fdebd0; color: #784212; }
        .badge-external { background: #d1f2eb; color: #0e6655; }
        .badge-context { background: #eaecee; color: #2c3e50; }
        .badge-note { background: #eaecee; color: #6c757d; }
        .badge-feature { background: #cce5ff; color: #004085; }
        .badge-bug { background: #f8d7da; color: #721c24; }
        .bar-chart { display: flex; align-items: flex-end; gap: 8px; height: 120px; margin-top: 16px; }
        .bar-col { display: flex; flex-direction: column; align-items: center; flex: 1; }
        .bar {
            width: 100%; border-radius: 4px 4px 0 0;
            background: var(--primary); min-height: 4px; transition: height 0.3s;
        }
        .bar-label { font-size: 11px; color: var(--text2); margin-top: 6px; }
        .bar-value { font-size: 12px; font-weight: 600; margin-bottom: 4px; }
        .legend { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 12px; }
        .legend-item { display: flex; align-items: center; gap: 6px; font-size: 12px; }
        .legend-dot { width: 10px; height: 10px; border-radius: 50%; }
        .coverage-bar { height: 8px; border-radius: 4px; background: var(--bg3); overflow: hidden; margin: 8px 0; }
        .coverage-fill { height: 100%; border-radius: 4px; }
        .coverage-row { margin-bottom: 12px; }
        .coverage-label { display: flex; justify-content: space-between; font-size: 12px; }
        .stale-warning { color: var(--danger); font-weight: 500; }
        .section { margin-bottom: 24px; }
        .empty { color: var(--text2); font-style: italic; padding: 16px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>` + "{{.ProjectName}}" + ` — Hero Report</h1>
            <div class="meta">Generated {{.GeneratedAt}} ` + `{{- if .TrackerType}} | Tracker: {{.TrackerType}}{{end}}</div>
        </div>

        <div class="grid">
            <div class="card metric">
                <div class="number">{{.TotalSpecs}}</div>
                <div class="label">Total Specs</div>
            </div>
            <div class="card metric">
                <div class="number" style="color: var(--primary);">{{.InFlight}}</div>
                <div class="label">In Flight</div>
            </div>
            <div class="card metric">
                <div class="number" style="color: var(--success);">{{.Completed}}</div>
                <div class="label">Completed</div>
            </div>
            <div class="card metric">
                <div class="number {{- if .StaleSpecs}} stale-warning{{end}}">{{len .StaleSpecs}}</div>
                <div class="label">Stale ({{.StaleDays}}+ days)</div>
            </div>
        </div>

        <div class="grid" style="grid-template-columns: 1fr 1fr;">
            <div class="card">
                <h2>Work Specs by Status</h2>
                {{if .StatusCounts}}
                <div class="legend">
                    {{range .StatusCounts}}
                    <div class="legend-item">
                        <div class="legend-dot" style="background: {{.Color}};"></div>
                        {{.Label}}: {{.Count}}
                    </div>
                    {{end}}
                </div>
                {{else}}<div class="empty">No work specs found.</div>{{end}}
            </div>
            <div class="card">
                <h2>Knowledge Base</h2>
                {{if .TypeCounts}}
                <div class="legend">
                    {{range .TypeCounts}}
                    <div class="legend-item">
                        <div class="legend-dot" style="background: {{.Color}};"></div>
                        {{.Label}}: {{.Count}}
                    </div>
                    {{end}}
                </div>
                {{else}}<div class="empty">No knowledge entries found.</div>{{end}}
            </div>
        </div>

        {{if .VelocityData}}
        <div class="card section">
            <h2>Velocity — Completed per Month</h2>
            <div class="bar-chart">
                {{range .VelocityData}}
                <div class="bar-col">
                    <div class="bar-value">{{.Completed}}</div>
                    <div class="bar" style="height: {{.Completed}}0%;"></div>
                    <div class="bar-label">{{.Month}}</div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        {{if .CoverageData.Total}}
        <div class="card section">
            <h2>Coverage</h2>
            <div class="coverage-row">
                <div class="coverage-label">
                    <span>Tracker linked</span>
                    <span>{{.CoverageData.WithTracker}}/{{.CoverageData.Total}}</span>
                </div>
                <div class="coverage-bar">
                    <div class="coverage-fill" style="width: {{coveragePct .CoverageData.WithTracker .CoverageData.Total}}%; background: var(--primary);"></div>
                </div>
            </div>
            <div class="coverage-row">
                <div class="coverage-label">
                    <span>Claimed / assigned</span>
                    <span>{{.CoverageData.WithClaim}}/{{.CoverageData.Total}}</span>
                </div>
                <div class="coverage-bar">
                    <div class="coverage-fill" style="width: {{coveragePct .CoverageData.WithClaim .CoverageData.Total}}%; background: var(--success);"></div>
                </div>
            </div>
            <div class="coverage-row">
                <div class="coverage-label">
                    <span>Has Changes section</span>
                    <span>{{.CoverageData.WithChanges}}/{{.CoverageData.Total}}</span>
                </div>
                <div class="coverage-bar">
                    <div class="coverage-fill" style="width: {{coveragePct .CoverageData.WithChanges .CoverageData.Total}}%; background: var(--warning);"></div>
                </div>
            </div>
        </div>
        {{end}}

        {{if .InFlightSpecs}}
        <div class="card section">
            <h2>In-Flight Work</h2>
            <table>
                <thead><tr><th>Spec</th><th>Type</th><th>Status</th><th>Age</th><th>Owner</th></tr></thead>
                <tbody>
                {{range .InFlightSpecs}}
                <tr>
                    <td><strong>{{.Slug}}</strong>{{if .Title}}<br><span style="color: var(--text2);">{{.Title}}</span>{{end}}</td>
                    <td><span class="badge badge-{{.Type}}">{{.Type}}</span></td>
                    <td><span class="badge badge-{{.Status}}">{{.Status}}</span></td>
                    <td>{{.Age}}</td>
                    <td>{{if .ClaimedBy}}{{.ClaimedBy}}{{else}}—{{end}}</td>
                </tr>
                {{end}}
                </tbody>
            </table>
        </div>
        {{end}}

        {{if .StaleSpecs}}
        <div class="card section">
            <h2 class="stale-warning">Stale Specs ({{.StaleDays}}+ days)</h2>
            <table>
                <thead><tr><th>Spec</th><th>Status</th><th>Age</th><th>Owner</th></tr></thead>
                <tbody>
                {{range .StaleSpecs}}
                <tr>
                    <td><strong>{{.Slug}}</strong></td>
                    <td><span class="badge badge-{{.Status}}">{{.Status}}</span></td>
                    <td class="stale-warning">{{.Age}}</td>
                    <td>{{if .ClaimedBy}}{{.ClaimedBy}}{{else}}—{{end}}</td>
                </tr>
                {{end}}
                </tbody>
            </table>
        </div>
        {{end}}

        {{if .RecentSpecs}}
        <div class="card section">
            <h2>Recently Modified</h2>
            <table>
                <thead><tr><th>Spec</th><th>Type</th><th>Status</th><th>Last Updated</th></tr></thead>
                <tbody>
                {{range .RecentSpecs}}
                <tr>
                    <td><strong>{{.Slug}}</strong>{{if .Title}}<br><span style="color: var(--text2);">{{.Title}}</span>{{end}}</td>
                    <td><span class="badge badge-{{.Type}}">{{.Type}}</span></td>
                    <td><span class="badge badge-{{.Status}}">{{.Status}}</span></td>
                    <td>{{.Age}}</td>
                </tr>
                {{end}}
                </tbody>
            </table>
        </div>
        {{end}}
    </div>
</body>
</html>
` // end of reportTemplate
