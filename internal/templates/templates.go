// Package templates extracts spec authoring patterns from the completed
// spec corpus and generates learned templates for hero new.
package templates

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/spec"
)

const (
	// MinCorpusSize is the minimum number of completed specs of a type
	// before patterns are extracted.
	MinCorpusSize = 5

	// SectionThreshold is the minimum frequency (0-1) a section must
	// appear at to be included in the learned template.
	SectionThreshold = 0.6
)

// TypePattern holds discovered patterns for a single spec type.
type TypePattern struct {
	Type       spec.Type
	CorpusSize int

	// Sections sorted by median position, with frequency.
	Sections []SectionPattern

	// Frontmatter fields and their frequencies.
	FrontmatterFields []FieldFreq

	// Acceptance criteria statistics.
	CriteriaCount CriteriaStats

	// EARS ratio (0-1).
	EARSRatio float64

	// Common tags ranked by frequency.
	TopTags []string
}

// SectionPattern tracks a section heading and its corpus statistics.
type SectionPattern struct {
	Name       string
	Frequency  float64 // 0-1
	MedianPos  int     // median position among specs that have it
}

// FieldFreq tracks a frontmatter field and its frequency.
type FieldFreq struct {
	Name      string
	Frequency float64
}

// CriteriaStats holds acceptance criteria count statistics.
type CriteriaStats struct {
	Mean   float64
	Median float64
	Min    int
	Max    int
}

// AnalyzeCorpus reads all completed specs and extracts patterns per type.
func AnalyzeCorpus(heroDir string) (map[spec.Type]*TypePattern, error) {
	allSpecs, err := spec.Discover(heroDir)
	if err != nil {
		return nil, err
	}

	// Group completed specs by type
	byType := make(map[spec.Type][]*spec.Spec)
	for _, s := range allSpecs {
		if s.Status == spec.StatusCompleted {
			byType[s.Type] = append(byType[s.Type], s)
		}
	}

	patterns := make(map[spec.Type]*TypePattern)
	for t, specs := range byType {
		if len(specs) < MinCorpusSize {
			// Below threshold — include with corpus size for reporting
			patterns[t] = &TypePattern{Type: t, CorpusSize: len(specs)}
			continue
		}
		patterns[t] = analyzeType(t, specs)
	}

	return patterns, nil
}

func analyzeType(t spec.Type, specs []*spec.Spec) *TypePattern {
	p := &TypePattern{
		Type:       t,
		CorpusSize: len(specs),
	}

	// Section analysis
	sectionCounts := make(map[string]int)
	sectionPositions := make(map[string][]int)
	fmFieldCounts := make(map[string]int)
	var criteriaCounts []int
	var totalEARS, totalCriteria int
	tagCounts := make(map[string]int)

	for _, s := range specs {
		// Sections
		i := 0
		for name := range s.Sections {
			sectionCounts[name]++
			sectionPositions[name] = append(sectionPositions[name], i)
			i++
		}

		// Frontmatter fields — check for common optional fields
		if s.Priority != "" {
			fmFieldCounts["priority"]++
		}
		if len(s.Relations) > 0 {
			for _, r := range s.Relations {
				if r.Kind == "depends-on" {
					fmFieldCounts["depends-on"]++
					break
				}
			}
			for _, r := range s.Relations {
				if r.Kind == "parent" {
					fmFieldCounts["parent"]++
					break
				}
			}
		}
		if len(s.Tags) > 0 {
			fmFieldCounts["tags"]++
			for _, tag := range s.Tags {
				tagCounts[tag]++
			}
		}

		// Acceptance criteria
		criteria := s.AcceptanceCriteria()
		if len(criteria) > 0 {
			criteriaCounts = append(criteriaCounts, len(criteria))
			for _, c := range criteria {
				totalCriteria++
				if c.Kind.IsEARS() {
					totalEARS++
				}
			}
		}
	}

	n := float64(len(specs))

	// Build section patterns
	for name, count := range sectionCounts {
		freq := float64(count) / n
		positions := sectionPositions[name]
		sort.Ints(positions)
		medianPos := positions[len(positions)/2]

		p.Sections = append(p.Sections, SectionPattern{
			Name:      name,
			Frequency: freq,
			MedianPos: medianPos,
		})
	}
	sort.Slice(p.Sections, func(i, j int) bool {
		return p.Sections[i].MedianPos < p.Sections[j].MedianPos
	})

	// Frontmatter fields
	for name, count := range fmFieldCounts {
		p.FrontmatterFields = append(p.FrontmatterFields, FieldFreq{
			Name:      name,
			Frequency: float64(count) / n,
		})
	}
	sort.Slice(p.FrontmatterFields, func(i, j int) bool {
		return p.FrontmatterFields[i].Frequency > p.FrontmatterFields[j].Frequency
	})

	// Criteria stats
	if len(criteriaCounts) > 0 {
		sort.Ints(criteriaCounts)
		sum := 0
		for _, c := range criteriaCounts {
			sum += c
		}
		p.CriteriaCount = CriteriaStats{
			Mean:   float64(sum) / float64(len(criteriaCounts)),
			Median: float64(criteriaCounts[len(criteriaCounts)/2]),
			Min:    criteriaCounts[0],
			Max:    criteriaCounts[len(criteriaCounts)-1],
		}
	}
	if totalCriteria > 0 {
		p.EARSRatio = float64(totalEARS) / float64(totalCriteria)
	}

	// Top tags
	type tagCount struct {
		tag   string
		count int
	}
	var tags []tagCount
	for tag, count := range tagCounts {
		tags = append(tags, tagCount{tag, count})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].count > tags[j].count })
	for i, tc := range tags {
		if i >= 5 {
			break
		}
		p.TopTags = append(p.TopTags, tc.tag)
	}

	return p
}

// GenerateLearnedTemplate renders a .learned.md file from a pattern.
func GenerateLearnedTemplate(p *TypePattern) string {
	var b strings.Builder

	fmt.Fprintf(&b, "---\ntype: learned-template\nspec_type: %s\ncorpus_size: %d\ngenerated: %s\n---\n\n",
		p.Type, p.CorpusSize, time.Now().UTC().Format(time.RFC3339))

	fmt.Fprintf(&b, "# Learned %s Template\n\n", titleCase(string(p.Type)))

	// Section frequencies
	fmt.Fprintf(&b, "## Discovered sections (by frequency)\n\n")
	for i, s := range p.Sections {
		fmt.Fprintf(&b, "%d. %s (%.0f%%)\n", i+1, titleCase(s.Name), s.Frequency*100)
	}

	// Criteria stats
	if p.CriteriaCount.Max > 0 {
		fmt.Fprintf(&b, "\n## Acceptance criteria profile\n\n")
		fmt.Fprintf(&b, "- Mean count: %.1f\n", p.CriteriaCount.Mean)
		fmt.Fprintf(&b, "- Median count: %.0f\n", p.CriteriaCount.Median)
		fmt.Fprintf(&b, "- Range: %d-%d\n", p.CriteriaCount.Min, p.CriteriaCount.Max)
		fmt.Fprintf(&b, "- EARS ratio: %.0f%%\n", p.EARSRatio*100)
	}

	// Frontmatter fields
	if len(p.FrontmatterFields) > 0 {
		fmt.Fprintf(&b, "\n## Common frontmatter fields\n\n")
		for _, f := range p.FrontmatterFields {
			fmt.Fprintf(&b, "- %s (%.0f%%)\n", f.Name, f.Frequency*100)
		}
	}

	// Scaffold body
	fmt.Fprintf(&b, "\n## Scaffold body\n\n")
	for _, s := range p.Sections {
		if s.Frequency >= SectionThreshold {
			fmt.Fprintf(&b, "## %s\n\n", titleCase(s.Name))
		}
	}

	return b.String()
}

// LoadLearnedTemplate reads the scaffold body from a learned template file.
// Returns the scaffold body and true if found, empty string and false otherwise.
func LoadLearnedTemplate(knowledgeDir, specType string) (string, bool) {
	path := filepath.Join(knowledgeDir, "templates", specType+".learned.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	content := string(data)
	// Extract the scaffold body section
	marker := "## Scaffold body"
	idx := strings.Index(content, marker)
	if idx < 0 {
		return "", false
	}
	body := strings.TrimSpace(content[idx+len(marker):])
	if body == "" {
		return "", false
	}
	return body, true
}

// WritePatternFiles writes learned template files for all patterns
// that meet the minimum corpus threshold.
func WritePatternFiles(knowledgeDir string, patterns map[spec.Type]*TypePattern) (int, error) {
	dir := filepath.Join(knowledgeDir, "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}

	written := 0
	for _, p := range patterns {
		if p.CorpusSize < MinCorpusSize {
			continue
		}
		content := GenerateLearnedTemplate(p)
		path := filepath.Join(dir, string(p.Type)+".learned.md")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// round1 rounds to 1 decimal place
func round1(f float64) float64 {
	return math.Round(f*10) / 10
}
