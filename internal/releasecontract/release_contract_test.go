package releasecontract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type requirement struct {
	name       string
	source     string
	value      string
	exactCount int
}

var requirements = []requirement{
	{"v-prefixed binary version", "goreleaser", "main.version=v{{.Version}}", 0},
	{"current checkout runtime", "release", "actions/checkout@v6", 2},
	{"artifact uploader", "release", "actions/upload-artifact@v4", 1},
	{"manual candidate isolation", "release", "if: github.event_name == 'workflow_dispatch'", 1},
	{"tag-only publisher", "release", "if: startsWith(github.ref, 'refs/tags/v')", 1},
	{"candidate builder", "release", "python3 scripts/release_candidate.py", 1},
	{"final Hero license gate", "release", "test -f LICENSE", 1},
	{"final notice gate", "release", "test -f THIRD_PARTY_NOTICES.txt", 1},
	{"stable artifact name", "release", "name: hero-darwin-arm64", 1},
	{"Darwin ARM64 binary selection", "release", "*/hero_darwin_arm64_*/hero", 0},
	{"candidate count", "release", `candidate_count="$(printf '%s\n' "$candidate_list" | sed '/^$/d' | wc -l | tr -d ' ')"`, 0},
	{"exactly-one guard", "release", `if [ "$candidate_count" -ne 1 ]; then`, 0},
	{"raw executable staging", "release", `install -m 0755 "$candidate" release-artifacts/hero-darwin-arm64/hero`, 0},
	{"checksum record", "release", `release-artifacts/hero-darwin-arm64/hero.sha256`, 0},
	{"missing-file failure", "release", "if-no-files-found: error", 1},
	{"bounded retention", "release", "retention-days: 90", 0},
}

func TestHeroCodeReleaseArtifactContract(t *testing.T) {
	release, goreleaser := contractFiles(t)
	if failures := validateContract(release, goreleaser); len(failures) != 0 {
		t.Fatalf("release contract violations:\n%s", strings.Join(failures, "\n"))
	}
}

func TestEveryReleaseContractGuardIsFalsifiable(t *testing.T) {
	release, goreleaser := contractFiles(t)
	for _, req := range requirements {
		t.Run(req.name, func(t *testing.T) {
			mutatedRelease, mutatedGoReleaser := release, goreleaser
			if req.source == "release" {
				mutatedRelease = strings.Replace(release, req.value, "contract-value-removed", 1)
			} else {
				mutatedGoReleaser = strings.Replace(goreleaser, req.value, "contract-value-removed", 1)
			}
			if failures := validateContract(mutatedRelease, mutatedGoReleaser); len(failures) == 0 {
				t.Fatalf("removing %q did not trip the contract guard", req.value)
			}
		})
	}

	t.Run("legacy checkout action", func(t *testing.T) {
		mutated := strings.Replace(release, "actions/checkout@v6", "actions/checkout@v4", 1)
		if failures := validateContract(mutated, goreleaser); len(failures) == 0 {
			t.Fatal("legacy checkout action did not trip the contract guard")
		}
	})
	t.Run("unprefixed binary version", func(t *testing.T) {
		mutated := strings.Replace(goreleaser, "main.version=v{{.Version}}", "main.version={{.Version}}", 1)
		if failures := validateContract(release, mutated); len(failures) == 0 {
			t.Fatal("unprefixed binary version did not trip the contract guard")
		}
	})
}

func contractFiles(t *testing.T) (string, string) {
	t.Helper()
	repoRoot := filepath.Join("..", "..")
	release := readContractFile(t, filepath.Join(repoRoot, ".github", "workflows", "release.yml"))
	goreleaser := readContractFile(t, filepath.Join(repoRoot, ".goreleaser.yaml"))
	return release, goreleaser
}

func validateContract(release, goreleaser string) []string {
	var failures []string
	for _, req := range requirements {
		content := release
		if req.source == "goreleaser" {
			content = goreleaser
		}
		count := strings.Count(content, req.value)
		if count == 0 {
			failures = append(failures, fmt.Sprintf("%s: missing %q", req.name, req.value))
		} else if req.exactCount != 0 && count != req.exactCount {
			failures = append(failures, fmt.Sprintf("%s: occurs %d times, want %d", req.name, count, req.exactCount))
		}
	}
	if strings.Contains(release, "actions/checkout@v4") {
		failures = append(failures, "release workflow regressed to the Node 20 checkout action")
	}
	if strings.Contains(goreleaser, "main.version={{.Version}}") {
		failures = append(failures, "release binary version no longer matches the v-prefixed source tag")
	}
	return failures
}

func readContractFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
