package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/version"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Reconcile which hero binary is running against which graph",
	Long: `Reports which hero binary is actually running, what your shell/harness
resolves 'hero' to on PATH, and whether the running binary's schema agrees
with the workspace graph.

Use this when a tool reports a schema/version mismatch. The common cause is
a stale hero binary on PATH (e.g. a GUI app inheriting a different PATH than
your login shell) reading a current graph — not a workspace problem. When the
PATH-resolved 'hero' differs from the running binary, doctor flags it: that is
the signal a harness may be shelling out to a different, older binary than the
one serving MCP. 'hero upgrade' does NOT fix this — it updates workspace files,
not the binary.`,
	RunE: runDoctor,
}

func init() {
	RegisterSmoke(doctorCmd, func(cmd *cobra.Command) error {
		return runDoctor(cmd, nil)
	})
}

// doctorInfo is the raw material for the doctor report. Gathered by
// runDoctor from the live environment; constructed directly in tests so
// the report logic can be exercised without touching PATH or the graph.
type doctorInfo struct {
	exe              string // os.Executable()
	exeResolved      string // filepath.EvalSymlinks(exe)
	pathHero         string // exec.LookPath("hero"), "" if not found
	pathHeroResolved string // filepath.EvalSymlinks(pathHero), "" if unresolved
	pathHeroErr      string // LookPath error, "" if found
	binaryVersion    string
	binarySchema     string
	graphSchema      string // "" when not in a workspace / no graph
	heroDir          string // "" when not in a workspace
}

func runDoctor(cmd *cobra.Command, args []string) error {
	info := doctorInfo{
		binaryVersion: rootCmd.Version,
		binarySchema:  graph.CompiledSchemaVersion(),
	}
	if info.binaryVersion == "" {
		info.binaryVersion = "dev"
	}

	if exe, err := os.Executable(); err == nil {
		info.exe = exe
		if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
			info.exeResolved = resolved
		}
	}

	if p, err := exec.LookPath("hero"); err == nil {
		info.pathHero = p
		if resolved, rerr := filepath.EvalSymlinks(p); rerr == nil {
			info.pathHeroResolved = resolved
		}
	} else {
		info.pathHeroErr = err.Error()
	}

	// Locate the workspace graph, if any, without migrating it — a stale
	// binary must still be able to read and report the graph's schema.
	projectRoot := findProjectRoot()
	if cfg, err := config.Load(projectRoot); err == nil {
		heroDir := cfg.HeroDir(projectRoot)
		if _, err := os.Stat(heroDir); err == nil {
			info.heroDir = heroDir
			if gs, err := graph.ReadSchemaVersion(heroDir); err == nil {
				info.graphSchema = gs
			}
		}
	}

	fmt.Fprint(cmd.OutOrStdout(), buildDoctorReport(info))
	return nil
}

// buildDoctorReport renders the doctor report. Pure so tests can drive
// every verdict and the PATH-divergence flag without a live environment.
func buildDoctorReport(info doctorInfo) string {
	var b strings.Builder
	b.WriteString("hero doctor\n\n")

	b.WriteString("Running binary\n")
	fmt.Fprintf(&b, "  os.Executable(): %s\n", orUnknown(info.exe))
	if info.exeResolved != "" && info.exeResolved != info.exe {
		fmt.Fprintf(&b, "  resolved path:   %s\n", info.exeResolved)
	}
	fmt.Fprintf(&b, "  version:         %s\n", displayVersion(info.binaryVersion))
	fmt.Fprintf(&b, "  binary schema:   %s\n", info.binarySchema)
	b.WriteString("\n")

	b.WriteString("PATH resolution\n")
	if info.pathHero == "" {
		fmt.Fprintf(&b, "  `hero` on PATH:  not found (%s)\n", orUnknown(info.pathHeroErr))
	} else {
		fmt.Fprintf(&b, "  `hero` on PATH:  %s\n", info.pathHero)
		if info.pathHeroResolved != "" && info.pathHeroResolved != info.pathHero {
			fmt.Fprintf(&b, "  resolved path:   %s\n", info.pathHeroResolved)
		}
		if pathDiffersFromRunning(info) {
			fmt.Fprintf(&b, "  WARNING: PATH `hero` is a DIFFERENT binary than the one running.\n"+
				"           A harness that shells out to a bare `hero` may bind this other,\n"+
				"           possibly older, binary instead of the one serving you now.\n")
		}
	}
	b.WriteString("\n")

	b.WriteString("Workspace graph\n")
	if info.heroDir == "" {
		b.WriteString("  no hero workspace found (run inside a workspace to compare schemas)\n\n")
		b.WriteString("Verdict: cannot compare — no workspace graph in scope.\n")
		return b.String()
	}
	if info.graphSchema == "" {
		fmt.Fprintf(&b, "  workspace:       %s\n", info.heroDir)
		b.WriteString("  graph schema:    none (graph not yet created)\n\n")
		b.WriteString("Verdict: cannot compare — no graph database in this workspace yet.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "  workspace:       %s\n", info.heroDir)
	fmt.Fprintf(&b, "  graph schema:    %s\n", info.graphSchema)
	b.WriteString("\n")

	b.WriteString(doctorVerdict(info.binarySchema, info.graphSchema))
	return b.String()
}

// doctorVerdict states whether the binary and graph schemas agree and, if
// not, the TRUE remediation for each direction.
func doctorVerdict(binarySchema, graphSchema string) string {
	switch version.CompareVersions(binarySchema, graphSchema) {
	case 0:
		return fmt.Sprintf("Verdict: OK — binary and graph agree on schema %s.\n", binarySchema)
	case -1:
		return fmt.Sprintf(
			"Verdict: this hero binary is OLDER than the workspace graph "+
				"(binary schema %s, graph schema %s).\n"+
				"  This is almost always the WRONG hero binary on PATH, not a workspace problem.\n"+
				"  Fix: point your harness at the newer hero binary (check the PATH warning above).\n"+
				"  `hero upgrade` will NOT help — it updates workspace files, not this binary.\n",
			binarySchema, graphSchema)
	default:
		return fmt.Sprintf(
			"Verdict: this hero binary is NEWER than the workspace graph "+
				"(binary schema %s, graph schema %s).\n"+
				"  Opening the graph normally migrates it up to the binary's schema; if this\n"+
				"  persists, a stale binary elsewhere may be holding the graph open.\n"+
				"  Fix: re-run your command with this binary, or check the PATH warning above.\n",
			binarySchema, graphSchema)
	}
}

// pathDiffersFromRunning reports whether the PATH-resolved `hero` is a
// different file than the running binary. Compares resolved (symlink-free)
// paths when available so a symlink to the same real file isn't flagged.
func pathDiffersFromRunning(info doctorInfo) bool {
	pathReal := info.pathHeroResolved
	if pathReal == "" {
		pathReal = info.pathHero
	}
	exeReal := info.exeResolved
	if exeReal == "" {
		exeReal = info.exe
	}
	if pathReal == "" || exeReal == "" {
		return false
	}
	return pathReal != exeReal
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
