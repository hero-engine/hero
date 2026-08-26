package install

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// AgentPurpose is the portable model-routing category declared by a canonical
// agent descriptor. It is intentionally narrower than Hero Code's runtime
// ModelCategory because chat, goal evaluation, and embeddings are not agent
// roles.
type AgentPurpose string

const (
	AgentPurposeDesign   AgentPurpose = "design"
	AgentPurposeDiagnose AgentPurpose = "diagnose"
	AgentPurposeAgent    AgentPurpose = "agent"
	AgentPurposeDraft    AgentPurpose = "draft"
	AgentPurposeReview   AgentPurpose = "review"
	AgentPurposeAssist   AgentPurpose = "assist"
)

func (p AgentPurpose) valid() bool {
	switch p {
	case AgentPurposeDesign, AgentPurposeDiagnose, AgentPurposeAgent,
		AgentPurposeDraft, AgentPurposeReview, AgentPurposeAssist:
		return true
	default:
		return false
	}
}

// validateCanonicalAgentPurposes checks every installable descriptor under
// agents/. Callers pass a raw Core, Engineering, PM, or Sales pack filesystem;
// custom project agents do not travel through this source-validation boundary.
func validateCanonicalAgentPurposes(content fs.FS) error {
	return validateAgentPurposes(content, func([]byte) bool { return true })
}

// validateInstallAgentPurposes validates the canonical packs covered by the
// portable routing contract while leaving QA-only descriptors untouched. A
// composed extension rewrites its domains field to ["*"], so raw-pack tests
// remain the exhaustive guard for PM and the runtime check treats an omitted
// wildcard purpose as outside this contract. ContentFS is always canonical;
// custom and legacy sources use the separate SourceDir boundary.
func validateInstallAgentPurposes(content fs.FS) error {
	return validateAgentPurposes(content, func(raw []byte) bool {
		domains, ok := readAgentDomainsFrontmatter(raw)
		if !ok || len(domains) == 0 {
			return true // universal Core agents
		}
		for _, domain := range domains {
			switch domain {
			case "engineering", "pm", "sales":
				return true
			}
		}
		return false
	})
}

func validateAgentPurposes(content fs.FS, required func([]byte) bool) error {
	entries, err := fs.ReadDir(content, "agents")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || isContentReadme(entry.Name()) {
			continue
		}
		name := "agents/" + entry.Name()
		raw, err := fs.ReadFile(content, name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		value, count := readAgentPurposeFrontmatter(raw)
		if count == 0 && !required(raw) {
			continue
		}
		if _, err := parseAgentPurpose(value, count); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func parseRequiredAgentPurpose(raw []byte) (AgentPurpose, error) {
	value, count := readAgentPurposeFrontmatter(raw)
	return parseAgentPurpose(value, count)
}

func parseAgentPurpose(value string, count int) (AgentPurpose, error) {
	if count == 0 || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("missing required purpose")
	}
	if count != 1 {
		return "", fmt.Errorf("purpose must be declared exactly once")
	}
	purpose := AgentPurpose(value)
	if !purpose.valid() {
		return "", fmt.Errorf("unknown purpose %q", value)
	}
	return purpose, nil
}

func readAgentPurposeFrontmatter(raw []byte) (value string, count int) {
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return "", 0
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		if line != strings.TrimLeft(line, " \t") {
			continue
		}
		key, rawValue, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "purpose" {
			continue
		}
		count++
		value = strings.Trim(strings.TrimSpace(rawValue), `"'`)
	}
	return value, count
}
