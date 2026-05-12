package impact

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/internal/index"
)

// SpecRef is a spec affected by a file change.
type SpecRef struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// ConvRef is a convention governing a file.
type ConvRef struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// DecisionRef is a decision mentioning a file.
type DecisionRef struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// Report is the impact analysis for one file.
type Report struct {
	FilePath    string        `json:"file_path"`
	Specs       []SpecRef     `json:"specs"`
	Conventions []ConvRef     `json:"conventions"`
	Decisions   []DecisionRef `json:"decisions"`
}

// Analyze runs impact analysis for the given file paths against the index.
func Analyze(idx *index.DB, filePaths []string) ([]Report, error) {
	var reports []Report

	for _, fp := range filePaths {
		r := Report{FilePath: fp}

		// Specs referencing this file via FilesTouched
		specResults, err := idx.SearchByFile(fp)
		if err == nil {
			for _, sr := range specResults {
				r.Specs = append(r.Specs, SpecRef{
					Slug:   sr.Slug,
					Title:  sr.Title,
					Status: string(sr.Status),
				})
			}
		}

		// Conventions with matching scope globs
		convResults, err := idx.FindConventionsForFiles([]string{fp})
		if err == nil {
			for _, cr := range convResults {
				r.Conventions = append(r.Conventions, ConvRef{
					Slug:  cr.Slug,
					Title: cr.Title,
				})
			}
		}

		// Decisions mentioning the file path
		decResults, err := idx.Search(fp)
		if err == nil {
			for _, dr := range decResults {
				if dr.Type == "decision" {
					r.Decisions = append(r.Decisions, DecisionRef{
						Slug:  dr.Slug,
						Title: dr.Title,
					})
				}
			}
		}

		reports = append(reports, r)
	}

	return reports, nil
}

// RenderText renders a human-readable impact report.
func RenderText(reports []Report) string {
	if len(reports) == 0 {
		return "No files to analyze.\n"
	}

	var sb strings.Builder
	for i, r := range reports {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(r.FilePath)
		sb.WriteString("\n")

		empty := true

		if len(r.Specs) > 0 {
			empty = false
			fmt.Fprintf(&sb, "\n  Specs (%d):\n", len(r.Specs))
			for _, s := range r.Specs {
				fmt.Fprintf(&sb, "    - %s (%s) — %s\n", s.Slug, s.Status, s.Title)
			}
		}

		if len(r.Conventions) > 0 {
			empty = false
			fmt.Fprintf(&sb, "\n  Conventions (%d):\n", len(r.Conventions))
			for _, c := range r.Conventions {
				fmt.Fprintf(&sb, "    - %s — %s\n", c.Slug, c.Title)
			}
		}

		if len(r.Decisions) > 0 {
			empty = false
			fmt.Fprintf(&sb, "\n  Decisions (%d):\n", len(r.Decisions))
			for _, d := range r.Decisions {
				fmt.Fprintf(&sb, "    - %s — %s\n", d.Slug, d.Title)
			}
		}

		if empty {
			sb.WriteString("\n  No impact detected — no specs, conventions, or decisions reference this file.\n")
		}

		total := len(r.Specs) + len(r.Conventions) + len(r.Decisions)
		if total > 0 {
			fmt.Fprintf(&sb, "\n  Summary: changing this file may affect %d spec(s), %d convention(s), %d decision(s).\n",
				len(r.Specs), len(r.Conventions), len(r.Decisions))
		}
	}
	return sb.String()
}

// RenderJSON renders reports as JSON.
func RenderJSON(reports []Report) (string, error) {
	data, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
