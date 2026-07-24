package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/attention/conformance"
	"github.com/hero-engine/hero/internal/serve"
)

var releasePaths = []string{
	"contracts/attention/schema/v1",
	"contracts/attention/testdata/v1",
	"contracts/attention/conformance/v1",
	"cmd/attention-conformance",
	"internal/attention/conformance",
	"internal/serve/mcp_tools_def.go",
	"internal/serve/mcp_dispatch.go",
	"internal/serve/mcp_tools_attention_contract.go",
	"internal/serve/api_attention.go",
}

func main() {
	check := flag.Bool("check", false, "fail if the checked-in bundle differs from canonical sources")
	releaseReady := flag.Bool("release-ready", false, "also require clean committed contract inputs before claiming publication readiness")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		fatal(fmt.Errorf("run from the Hero repository root: %w", err))
	}
	toolJSON, err := json.Marshal(serve.AttentionToolDefinitions())
	if err != nil {
		fatal(err)
	}
	bundle, err := conformance.Build(root, toolJSON)
	if err != nil {
		fatal(err)
	}
	output := filepath.Join(root, filepath.FromSlash(conformance.BundlePath))
	if *check || *releaseReady {
		if err := conformance.Check(output, bundle); err != nil {
			fatal(err)
		}
		if *releaseReady {
			args := append([]string{"status", "--porcelain", "--"}, releasePaths...)
			command := exec.Command("git", args...)
			command.Dir = root
			status, err := command.Output()
			if err != nil {
				fatal(err)
			}
			if err := validateReleaseStatus(string(status)); err != nil {
				fatal(err)
			}
			fmt.Printf("Attention conformance bundle is release-ready: %s\n", bundle.ManifestSHA256)
			return
		}
		fmt.Printf("Attention conformance bundle is current: %s\n", bundle.ManifestSHA256)
		return
	}
	if err := conformance.Write(output, bundle); err != nil {
		fatal(err)
	}
	fmt.Printf("Wrote %s (manifest SHA-256 %s)\n", conformance.BundlePath, bundle.ManifestSHA256)
}

func validateReleaseStatus(status string) error {
	if strings.TrimSpace(status) != "" {
		return fmt.Errorf("Attention contract output is uncommitted; commit or release the generated bundle before claiming it is ready")
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
