package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var (
	newSpecType    string
	newInteractive bool
)

// newStdin is the reader used for interactive prompts. Tests can replace it.
var newStdin io.Reader = os.Stdin

var newCmd = &cobra.Command{
	Use:   "new <slug>",
	Short: "Scaffold a new spec from template",
	Long: `Creates a new spec with frontmatter and standard sections.

The slug becomes the directory name under the appropriate planning subdirectory.
For example: hero new csv-export --type feature creates
  .hero/planning/features/csv-export/spec.md

Use --interactive (-i) to be prompted for title, tags, and other fields.

Supported types: feature (default), bug, initiative, convention, decision,
rule, external, context, note.`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

func init() {
	newCmd.Flags().StringVar(&newSpecType, "type", "feature", "spec type: feature, bug, initiative, convention, decision, rule, external, context, note")
	newCmd.Flags().BoolVarP(&newInteractive, "interactive", "i", false, "prompt for title, tags, and other fields")
}

// interactiveInputs holds values collected during interactive mode.
type interactiveInputs struct {
	title     string
	tags      []string
	claimedBy string
}

func runNew(cmd *cobra.Command, args []string) error {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	heroDir := cfg.HeroDir(projectRoot)
	if _, err := os.Stat(heroDir); os.IsNotExist(err) {
		return fmt.Errorf("no hero workspace found (run 'hero init' first)")
	}

	slug := args[0]

	// Validate slug
	if slug == "" || strings.Contains(slug, "/") || strings.Contains(slug, " ") {
		return fmt.Errorf("invalid slug %q — use lowercase-kebab-case without spaces or slashes", slug)
	}

	// Determine target directory based on type
	specType := strings.ToLower(newSpecType)
	targetDir, err := specTargetDir(heroDir, specType, slug)
	if err != nil {
		return err
	}

	specPath := filepath.Join(targetDir, "spec.md")

	// Check for collision
	if _, err := os.Stat(specPath); err == nil {
		return fmt.Errorf("spec already exists: %s", specPath)
	}

	// Collect interactive inputs if requested
	var inputs *interactiveInputs
	if newInteractive {
		inputs, err = collectInteractiveInputs(slug, specType)
		if err != nil {
			return fmt.Errorf("interactive input: %w", err)
		}
	}

	// Create directory
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Generate spec content
	var content string

	// Check for custom template
	customBody, hasCustom := loadCustomTemplate(heroDir, specType)

	if inputs != nil {
		if hasCustom {
			content = generateSpecFromCustomInteractive(slug, specType, inputs, customBody)
		} else {
			content = generateSpecTemplateInteractive(slug, specType, inputs)
		}
	} else {
		if hasCustom {
			content = generateSpecFromCustom(slug, specType, customBody)
		} else {
			content = generateSpecTemplate(slug, specType)
		}
	}

	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	// Render the type name through the active vocabulary so the "Created"
	// line speaks the workspace dialect (e.g. "Story" under agile-scrum)
	// while the on-disk frontmatter stays canonical.
	displayed := displayType(activeVocab(&cfg), specType)
	if displayed == "" || displayed == specType {
		fmt.Printf("Created %s spec: %s\n", specType, specPath)
	} else {
		fmt.Printf("Created %s spec (type=%s): %s\n", displayed, specType, specPath)
	}
	return nil
}

// specTargetDir returns the directory where a new spec should be created.
func specTargetDir(heroDir, specType, slug string) (string, error) {
	switch specType {
	case "feature":
		return filepath.Join(heroDir, "planning", "features", slug), nil
	case "bug":
		return filepath.Join(heroDir, "planning", "bugs", slug), nil
	case "initiative":
		return filepath.Join(heroDir, "planning", "initiatives", slug), nil
	case "convention":
		return filepath.Join(heroDir, "knowledge", "conventions", slug), nil
	case "decision":
		return filepath.Join(heroDir, "knowledge", "decisions", slug), nil
	case "rule":
		return filepath.Join(heroDir, "knowledge", "rules", slug), nil
	case "external":
		return filepath.Join(heroDir, "knowledge", "external", slug), nil
	case "context":
		return filepath.Join(heroDir, "knowledge", "context", slug), nil
	case "note":
		return filepath.Join(heroDir, "knowledge", "notes", slug), nil
	default:
		return "", fmt.Errorf("unknown spec type %q — use feature, bug, initiative, convention, decision, rule, external, context, or note", specType)
	}
}

// generateSpecTemplate creates the initial spec content with appropriate sections.
func generateSpecTemplate(slug, specType string) string {
	title := slugToTitle(slug)
	date := time.Now().Format("2006-01-02")

	switch specType {
	case "feature":
		return fmt.Sprintf(`---
title: %s
type: feature
status: planning
created: %s
tags: []
---
# %s

## Goal

<!-- What does this feature accomplish? -->

## Background

<!-- Why is this needed? What problem does it solve? -->

## Design

<!-- How will this be implemented? Key decisions and tradeoffs. -->

## Changes

<!-- Files and areas of the codebase that will be modified. -->

## Acceptance Criteria

<!-- How do we know this is done? -->
`, title, date, title)

	case "bug":
		return fmt.Sprintf(`---
title: %s
type: bug
status: planning
created: %s
tags: []
---
# %s

## Problem

<!-- What is broken? How does it manifest? -->

## Steps to Reproduce

<!-- Minimal steps to trigger the bug. -->

## Expected Behavior

<!-- What should happen instead? -->

## Root Cause

<!-- Analysis of why this happens. -->

## Fix

<!-- How will this be fixed? -->

## Changes

<!-- Files and areas of the codebase that will be modified. -->
`, title, date, title)

	case "initiative":
		return fmt.Sprintf(`---
title: %s
type: initiative
status: planning
created: %s
tags: []
---
# %s

## Vision

<!-- What is the overarching goal? -->

## Scope

<!-- What is included and excluded? -->

## Child Specs

<!-- List feature/bug specs that belong to this initiative. -->

## Success Criteria

<!-- How do we measure success? -->
`, title, date, title)

	case "convention":
		return fmt.Sprintf(`---
title: %s
type: convention
status: draft
created: %s
scope: ["*"]
tags: []
---
# %s

## Rule

<!-- What is the convention? State it clearly. -->

## Rationale

<!-- Why does this convention exist? -->

## Examples

<!-- Good and bad examples. -->

## Exceptions

<!-- When is it acceptable to deviate? -->
`, title, date, title)

	case "decision":
		return fmt.Sprintf(`---
title: %s
type: decision
status: proposed
created: %s
tags: []
---
# %s

## Context

<!-- What situation prompted this decision? -->

## Decision

<!-- What was decided? -->

## Alternatives Considered

<!-- What other options were evaluated? -->

## Consequences

<!-- What are the implications of this decision? -->
`, title, date, title)

	case "rule":
		return fmt.Sprintf(`---
title: %s
type: rule
status: active
created: %s
scope: ["*"]
tags: []
---
# %s

## Constraint

<!-- What is the hard requirement? State it precisely. -->

## Rationale

<!-- Why does this rule exist? (compliance, security, performance, etc.) -->

## Enforcement

<!-- How is this enforced? (CI check, review gate, automated test, etc.) -->

## Exceptions

<!-- Under what circumstances can this rule be waived, and who approves? -->
`, title, date, title)

	case "external":
		return fmt.Sprintf(`---
title: %s
type: external
status: active
created: %s
url: ""
local_path: ""
tags: []
---
# %s

## Description

<!-- What is this external resource? -->

## When to Reference

<!-- When should an agent or engineer consult this resource? -->

## Key Sections

<!-- Important areas within this resource. -->
`, title, date, title)

	case "context":
		return fmt.Sprintf(`---
title: %s
type: context
status: active
created: %s
tags: []
---
# %s

## Overview

<!-- What aspect of the project does this describe? -->

## Details

<!-- Key information (team structure, deployment topology, environment setup, etc.) -->

## References

<!-- Links or paths to related resources. -->
`, title, date, title)

	case "note":
		return fmt.Sprintf(`---
title: %s
type: note
created: %s
tags: []
---
# %s

<!-- Brainstorm, conversation capture, stream-of-consciousness — no structure required. -->
`, title, date, title)

	default:
		// Should not reach here due to validation, but be safe
		return fmt.Sprintf("---\ntitle: %s\ntype: %s\nstatus: planning\ncreated: %s\n---\n# %s\n", title, specType, date, title)
	}
}

// slugToTitle converts a kebab-case slug to a Title Case string.
func slugToTitle(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// collectInteractiveInputs prompts the user for spec metadata.
func collectInteractiveInputs(slug, specType string) (*interactiveInputs, error) {
	scanner := bufio.NewScanner(newStdin)
	inputs := &interactiveInputs{}

	defaultTitle := slugToTitle(slug)

	// Title
	fmt.Printf("Title [%s]: ", defaultTitle)
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			inputs.title = text
		} else {
			inputs.title = defaultTitle
		}
	} else {
		inputs.title = defaultTitle
	}

	// Tags
	fmt.Print("Tags (comma-separated) []: ")
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			for _, t := range strings.Split(text, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					inputs.tags = append(inputs.tags, t)
				}
			}
		}
	}

	// Claimed by (only for work specs)
	if specType == "feature" || specType == "bug" || specType == "initiative" {
		fmt.Print("Claimed by []: ")
		if scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text != "" {
				inputs.claimedBy = text
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	return inputs, nil
}

// generateSpecTemplateInteractive creates spec content using interactive inputs.
func generateSpecTemplateInteractive(slug, specType string, inputs *interactiveInputs) string {
	title := inputs.title
	if title == "" {
		title = slugToTitle(slug)
	}
	date := time.Now().Format("2006-01-02")

	tagsStr := "[]"
	if len(inputs.tags) > 0 {
		tagsStr = "[" + strings.Join(inputs.tags, ", ") + "]"
	}

	// Build frontmatter with interactive values
	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString(fmt.Sprintf("title: %s\n", title))
	fm.WriteString(fmt.Sprintf("type: %s\n", specType))

	// Set initial status based on type
	switch specType {
	case "convention":
		fm.WriteString("status: draft\n")
	case "decision":
		fm.WriteString("status: proposed\n")
	case "rule", "external", "context", "note":
		fm.WriteString("status: active\n")
	default:
		fm.WriteString("status: planning\n")
	}

	fm.WriteString(fmt.Sprintf("created: %s\n", date))

	if inputs.claimedBy != "" {
		fm.WriteString(fmt.Sprintf("claimed_by: %s\n", inputs.claimedBy))
	}

	if specType == "convention" || specType == "rule" {
		fm.WriteString("scope: [\"*\"]\n")
	}

	fm.WriteString(fmt.Sprintf("tags: %s\n", tagsStr))
	fm.WriteString("---\n")

	// Use the same body sections as the non-interactive version
	body := generateSpecBody(title, specType)
	return fm.String() + body
}

// generateSpecBody returns the section content (without frontmatter) for a spec type.
func generateSpecBody(title, specType string) string {
	switch specType {
	case "feature":
		return fmt.Sprintf(`# %s

## Goal

<!-- What does this feature accomplish? -->

## Background

<!-- Why is this needed? What problem does it solve? -->

## Design

<!-- How will this be implemented? Key decisions and tradeoffs. -->

## Changes

<!-- Files and areas of the codebase that will be modified. -->

## Acceptance Criteria

<!-- How do we know this is done? -->
`, title)

	case "bug":
		return fmt.Sprintf(`# %s

## Problem

<!-- What is broken? How does it manifest? -->

## Steps to Reproduce

<!-- Minimal steps to trigger the bug. -->

## Expected Behavior

<!-- What should happen instead? -->

## Root Cause

<!-- Analysis of why this happens. -->

## Fix

<!-- How will this be fixed? -->

## Changes

<!-- Files and areas of the codebase that will be modified. -->
`, title)

	case "initiative":
		return fmt.Sprintf(`# %s

## Vision

<!-- What is the overarching goal? -->

## Scope

<!-- What is included and excluded? -->

## Child Specs

<!-- List feature/bug specs that belong to this initiative. -->

## Success Criteria

<!-- How do we measure success? -->
`, title)

	case "convention":
		return fmt.Sprintf(`# %s

## Rule

<!-- What is the convention? State it clearly. -->

## Rationale

<!-- Why does this convention exist? -->

## Examples

<!-- Good and bad examples. -->

## Exceptions

<!-- When is it acceptable to deviate? -->
`, title)

	case "decision":
		return fmt.Sprintf(`# %s

## Context

<!-- What situation prompted this decision? -->

## Decision

<!-- What was decided? -->

## Alternatives Considered

<!-- What other options were evaluated? -->

## Consequences

<!-- What are the implications of this decision? -->
`, title)

	case "rule":
		return fmt.Sprintf(`# %s

## Constraint

<!-- What is the hard requirement? State it precisely. -->

## Rationale

<!-- Why does this rule exist? (compliance, security, performance, etc.) -->

## Enforcement

<!-- How is this enforced? (CI check, review gate, automated test, etc.) -->

## Exceptions

<!-- Under what circumstances can this rule be waived, and who approves? -->
`, title)

	case "external":
		return fmt.Sprintf(`# %s

## Description

<!-- What is this external resource? -->

## When to Reference

<!-- When should an agent or engineer consult this resource? -->

## Key Sections

<!-- Important areas within this resource. -->
`, title)

	case "context":
		return fmt.Sprintf(`# %s

## Overview

<!-- What aspect of the project does this describe? -->

## Details

<!-- Key information (team structure, deployment topology, environment setup, etc.) -->

## References

<!-- Links or paths to related resources. -->
`, title)

	case "note":
		return fmt.Sprintf(`# %s

<!-- Brainstorm, conversation capture, stream-of-consciousness — no structure required. -->
`, title)

	default:
		return fmt.Sprintf("# %s\n", title)
	}
}

// loadCustomTemplate checks for a custom template file at
// .hero/knowledge/templates/<type>.md and returns the body content
// with {{.Title}} and {{.Date}} as placeholders.
// Returns (body, true) if found, ("", false) if not.
func loadCustomTemplate(heroDir, specType string) (string, bool) {
	templatePath := filepath.Join(heroDir, "knowledge", "templates", specType+".md")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// applyTemplatePlaceholders replaces {{.Title}} and {{.Date}} in a template body.
func applyTemplatePlaceholders(body, title, date string) string {
	result := strings.ReplaceAll(body, "{{.Title}}", title)
	result = strings.ReplaceAll(result, "{{.Date}}", date)
	return result
}

// generateSpecFromCustom creates spec content using a custom template body.
func generateSpecFromCustom(slug, specType, customBody string) string {
	title := slugToTitle(slug)
	date := time.Now().Format("2006-01-02")

	fm := buildFrontmatter(title, specType, date, "", "[]")
	body := applyTemplatePlaceholders(customBody, title, date)

	return fm + body
}

// generateSpecFromCustomInteractive creates spec content using a custom template
// body with interactive inputs.
func generateSpecFromCustomInteractive(slug, specType string, inputs *interactiveInputs, customBody string) string {
	title := inputs.title
	if title == "" {
		title = slugToTitle(slug)
	}
	date := time.Now().Format("2006-01-02")

	tagsStr := "[]"
	if len(inputs.tags) > 0 {
		tagsStr = "[" + strings.Join(inputs.tags, ", ") + "]"
	}

	fm := buildFrontmatter(title, specType, date, inputs.claimedBy, tagsStr)
	body := applyTemplatePlaceholders(customBody, title, date)

	return fm + body
}

// buildFrontmatter generates the YAML frontmatter for a spec.
func buildFrontmatter(title, specType, date, claimedBy, tagsStr string) string {
	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString(fmt.Sprintf("title: %s\n", title))
	fm.WriteString(fmt.Sprintf("type: %s\n", specType))

	switch specType {
	case "convention":
		fm.WriteString("status: draft\n")
	case "decision":
		fm.WriteString("status: proposed\n")
	case "rule", "external", "context":
		fm.WriteString("status: active\n")
	case "note":
		// Notes don't have status in their frontmatter template
	default:
		fm.WriteString("status: planning\n")
	}

	fm.WriteString(fmt.Sprintf("created: %s\n", date))

	if claimedBy != "" {
		fm.WriteString(fmt.Sprintf("claimed_by: %s\n", claimedBy))
	}

	if specType == "convention" || specType == "rule" {
		fm.WriteString("scope: [\"*\"]\n")
	}

	fm.WriteString(fmt.Sprintf("tags: %s\n", tagsStr))
	fm.WriteString("---\n")

	return fm.String()
}
