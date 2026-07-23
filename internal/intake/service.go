package intake

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/spec"
)

type Service struct {
	repository Repository
	now        func() time.Time
}

type CaptureRequest struct {
	Slug       string
	Title      string
	Body       string
	Provenance *attention.ProvenanceReference
	Source     SourceMetadata
}

type SourceMetadata struct {
	SenderPeerID    string
	RecipientPeerID string
	ThreadID        string
}

type PromoteRequest struct {
	Slug             string
	Type             string
	AllowReplay      bool
	GenerateTemplate func(slug, artifactType string) string
}

type PromoteResult struct {
	Slug       string `json:"slug"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Path       string `json:"-"`
	IntakeSlug string `json:"intake_slug"`
}

func NewService(heroDir string) *Service {
	return &Service{repository: Repository{HeroDir: heroDir}, now: time.Now}
}

func DefaultRoadmapTemplate(slug, artifactType string) string {
	title := strings.ReplaceAll(strings.Title(strings.ReplaceAll(slug, "-", " ")), "Csv", "CSV")
	date := time.Now().Format("2006-01-02")
	if artifactType == "bug" {
		return fmt.Sprintf("---\ntitle: %s\nslug: %s\ntype: bug\nstatus: planning\ncreated: %s\ntags: []\n---\n# %s\n\n## Problem\n\n## Steps to Reproduce\n\n## Expected Behavior\n\n## Root Cause\n\n## Fix\n\n## Changes\n", title, slug, date, title)
	}
	return fmt.Sprintf("---\ntitle: %s\nslug: %s\ntype: feature\nstatus: planning\ncreated: %s\ntags: []\n---\n# %s\n\n## Goal\n\n## Background\n\n## Design\n\n## Changes\n\n## Acceptance Criteria\n", title, slug, date, title)
}

func (s *Service) Resolve(slug string) (*spec.Spec, error) {
	return s.repository.Resolve(slug)
}

func (s *Service) Capture(req CaptureRequest) (string, error) {
	if req.Slug == "" || strings.Contains(req.Slug, "/") {
		return "", fmt.Errorf("invalid slug %q — use lowercase-kebab-case without slashes", req.Slug)
	}
	if req.Provenance != nil {
		specs, err := spec.Discover(s.repository.HeroDir)
		if err != nil {
			return "", fmt.Errorf("discovering specs: %w", err)
		}
		for _, existing := range specs {
			if existing.Type == spec.TypeIntake && sameSource(existing, req.Provenance) {
				return existing.Path, nil
			}
		}
	}
	targetDir := filepath.Join(s.repository.HeroDir, "planning", "intake", req.Slug)
	specPath := filepath.Join(targetDir, "spec.md")
	if _, err := os.Stat(specPath); err == nil {
		existing, resolveErr := s.Resolve(req.Slug)
		if resolveErr == nil && sameSource(existing, req.Provenance) {
			return specPath, nil
		}
		return "", fmt.Errorf("intake already exists: %s", specPath)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	content := GenerateContent(req.Title, req.Slug, s.now().Format("2006-01-02"), req.Body, req.Provenance, req.Source)
	if err := spec.AtomicWriteFile(specPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing intake: %w", err)
	}
	return specPath, nil
}

func (s *Service) Promote(req PromoteRequest) (PromoteResult, error) {
	in, err := s.Resolve(req.Slug)
	if err != nil {
		return PromoteResult{}, err
	}
	artifactType := strings.ToLower(req.Type)
	if artifactType != "feature" && artifactType != "bug" {
		return PromoteResult{}, fmt.Errorf("--type must be feature or bug, got %q", artifactType)
	}
	if in.Status == spec.StatusPromoted {
		if !req.AllowReplay {
			return PromoteResult{}, fmt.Errorf("intake %q is already promoted", req.Slug)
		}
		for _, candidateType := range []string{"feature", "bug"} {
			path := filepath.Join(s.repository.HeroDir, "planning", candidateType+"s", req.Slug, "spec.md")
			if _, statErr := os.Stat(path); statErr == nil {
				if candidateType != artifactType {
					return PromoteResult{}, fmt.Errorf("intake %q was promoted as %s, not %s", req.Slug, candidateType, artifactType)
				}
				return PromoteResult{Slug: req.Slug, Type: candidateType, Status: string(spec.StatusPlanning), Path: path, IntakeSlug: req.Slug}, nil
			}
		}
		return PromoteResult{}, fmt.Errorf("promoted intake %q has no roadmap artifact", req.Slug)
	}
	if req.GenerateTemplate == nil {
		return PromoteResult{}, fmt.Errorf("roadmap template generator is required")
	}
	newDir := filepath.Join(s.repository.HeroDir, "planning", artifactType+"s", req.Slug)
	newPath := filepath.Join(newDir, "spec.md")
	if _, statErr := os.Stat(newPath); statErr == nil {
		return PromoteResult{}, fmt.Errorf("%s spec already exists: %s", artifactType, newPath)
	}
	content := InjectRelation(req.GenerateTemplate(req.Slug, artifactType), "derived_from", req.Slug)
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		return PromoteResult{}, fmt.Errorf("creating directory: %w", err)
	}
	if err := spec.AtomicWriteFile(newPath, []byte(content), 0o644); err != nil {
		return PromoteResult{}, fmt.Errorf("writing %s spec: %w", artifactType, err)
	}
	data, err := os.ReadFile(in.Path)
	if err != nil {
		return PromoteResult{}, fmt.Errorf("reading intake: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "status", string(spec.StatusPromoted))
	updated = spec.SetFrontmatterField(updated, "promoted_to", req.Slug)
	updated = InjectRelation(updated, "promotes_to", req.Slug)
	if err := spec.AtomicWriteFile(in.Path, []byte(updated), 0o644); err != nil {
		return PromoteResult{}, fmt.Errorf("updating intake: %w", err)
	}
	return PromoteResult{Slug: req.Slug, Type: artifactType, Status: string(spec.StatusPlanning), Path: newPath, IntakeSlug: req.Slug}, nil
}

func (s *Service) Reject(slug string) (bool, error) {
	in, err := s.Resolve(slug)
	if err != nil {
		return false, err
	}
	if in.Status == spec.StatusRejected {
		return false, nil
	}
	data, err := os.ReadFile(in.Path)
	if err != nil {
		return false, fmt.Errorf("reading intake: %w", err)
	}
	updated := spec.SetFrontmatterField(string(data), "status", string(spec.StatusRejected))
	if err := spec.AtomicWriteFile(in.Path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("updating intake: %w", err)
	}
	return true, nil
}

func GenerateContent(title, slug, date, body string, provenance *attention.ProvenanceReference, source SourceMetadata) string {
	var b strings.Builder
	safeHeading := strings.NewReplacer("\r", " ", "\n", " ").Replace(title)
	fmt.Fprintf(&b, "---\ntitle: %s\nslug: %s\ntype: intake\nstatus: planning\ncreated: %s\ntags: []\n", safeYAMLTitle(title), slug, date)
	if provenance != nil {
		fmt.Fprintf(&b, "source:\n  kind: %s\n  id: %s\n", strconv.Quote(provenance.Kind), strconv.Quote(provenance.SourceID))
		if source.SenderPeerID != "" {
			fmt.Fprintf(&b, "  sender_peer_id: %s\n", strconv.Quote(source.SenderPeerID))
		}
		if source.RecipientPeerID != "" {
			fmt.Fprintf(&b, "  recipient_peer_id: %s\n", strconv.Quote(source.RecipientPeerID))
		}
		if source.ThreadID != "" {
			fmt.Fprintf(&b, "  thread_id: %s\n", strconv.Quote(source.ThreadID))
		}
	}
	fmt.Fprintf(&b, "---\n# %s\n\n## Signal\n\n", safeHeading)
	if body == "" {
		b.WriteString("<!-- What is the idea or inbound signal? Capture it in its own words. -->\n")
	} else {
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n## Notes\n\n<!-- Triage thoughts, links, who asked. Promote with `hero intake promote`. -->\n")
	return b.String()
}

func safeYAMLTitle(title string) string {
	if title == strings.TrimSpace(title) && title != "" && !strings.ContainsAny(title, ":\r\n#[]{}&*!|>'\"%@`") {
		return title
	}
	return strconv.Quote(title)
}

func InjectRelation(content, kind, target string) string {
	return injectRelation(content, kind, target)
}

func injectRelation(content, kind, target string) string {
	end := strings.Index(content[4:], "\n---")
	if !strings.HasPrefix(content, "---\n") || end < 0 {
		return content
	}
	end += 4
	block := fmt.Sprintf("relations:\n  - target: %s\n    kind: %s\n", target, kind)
	return content[:end] + "\n" + block + content[end:]
}

func sameSource(existing *spec.Spec, source *attention.ProvenanceReference) bool {
	if source == nil {
		return false
	}
	data, err := os.ReadFile(existing.Path)
	if err != nil {
		return false
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return false
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return false
	}
	frontmatter := content[4 : end+4]
	var inSource bool
	var kind, id string
	for _, raw := range strings.Split(frontmatter, "\n") {
		if raw == "source:" {
			inSource = true
			continue
		}
		if inSource && len(raw) > 0 && raw[0] != ' ' {
			break
		}
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "kind:") {
			kind = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "kind:")), `"`)
		}
		if strings.HasPrefix(line, "id:") {
			id = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "id:")), `"`)
		}
	}
	return kind == source.Kind && id == source.SourceID
}
