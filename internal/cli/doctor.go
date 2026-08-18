package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/graph"
	"github.com/hero-engine/hero/internal/install"
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

	// inventory is the per-target install introspection rendered as the
	// "Installed harness targets" section. inventoryErr records a non-fatal
	// introspection failure — doctor still reports binary/PATH/schema.
	inventory    []install.TargetInventory
	inventoryErr string
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

	// Introspect installed harness targets. A failure here must not fail
	// doctor — it is triage and must still report binary/PATH/schema.
	if projectRoot != "" {
		if inv, err := install.Inventory(projectRoot, activeDomainForRoot(projectRoot)); err != nil {
			info.inventoryErr = err.Error()
		} else {
			info.inventory = inv
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

	b.WriteString(buildInventorySection(info))

	b.WriteString(doctorVerdict(info.binarySchema, info.graphSchema))
	return b.String()
}

// inventoryTargetNames is the full seven-target set, used to render the
// "not installed:" line for targets absent from the row set.
var inventoryTargetNames = []install.Target{
	install.TargetClaude, install.TargetOpenCode, install.TargetCursor,
	install.TargetCopilot, install.TargetCodex, install.TargetGeneric,
	install.TargetGrok,
}

// buildInventorySection renders the "Installed harness targets" section: one
// row per installed target with expected-vs-actual agent/command/skill counts,
// a single "not installed:" line for absent targets, an in-section WARNING when
// an installed target is short on content (fixed by `hero upgrade`), and a
// codex footnote when codex is present. Pure so tests can drive it directly.
func buildInventorySection(info doctorInfo) string {
	var b strings.Builder
	b.WriteString("Installed harness targets\n")

	if info.inventoryErr != "" {
		fmt.Fprintf(&b, "  install introspection unavailable: %s\n\n", info.inventoryErr)
		return b.String()
	}

	if len(info.inventory) == 0 {
		b.WriteString("  no harness targets installed — run `hero install --target <claude|codex|copilot|cursor|opencode|generic|grok>`\n\n")
		return b.String()
	}

	rows := [][]string{{"TARGET", "AGENTS", "COMMANDS", "SKILLS", "ROOT FILE"}}
	installed := map[install.Target]bool{}
	incomplete := 0
	hasCodex := false
	var codex install.TargetInventory
	hasGrok := false
	var grok install.TargetInventory
	for _, inv := range info.inventory {
		installed[inv.Target] = true
		if inv.Target == install.TargetCodex {
			hasCodex = true
			codex = inv
		}
		if inv.Target == install.TargetGrok {
			hasGrok = true
			grok = inv
		}
		if kindShort(inv.Agents) || kindShort(inv.Commands) || kindShort(inv.Skills) {
			incomplete++
		}
		rows = append(rows, []string{
			string(inv.Target),
			kindCell(inv.Agents),
			kindCell(inv.Commands),
			kindCell(inv.Skills),
			inv.RootFile,
		})
	}
	b.WriteString(renderInventoryTable(rows))

	var notInstalled []string
	for _, t := range inventoryTargetNames {
		if !installed[t] {
			notInstalled = append(notInstalled, string(t))
		}
	}
	sort.Strings(notInstalled)
	if len(notInstalled) > 0 {
		fmt.Fprintf(&b, "  not installed: %s\n", strings.Join(notInstalled, ", "))
	}

	if incomplete > 0 {
		noun, verb := "target", "is"
		if incomplete > 1 {
			noun, verb = "targets", "are"
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "  WARNING: %d installed %s %s incomplete (marked !) — content is\n", incomplete, noun, verb)
		b.WriteString("           missing. Run `hero upgrade` to re-materialize the missing\n")
		b.WriteString("           agents, commands, and skills.\n")
	}

	if hasCodex {
		cmds := codex.Commands.Expected
		canonical := codex.Skills.Expected - cmds
		b.WriteString("\n")
		fmt.Fprintf(&b, "  codex has no command loader — its %d commands install as skills under\n", cmds)
		fmt.Fprintf(&b, "  .agents/skills/command-<name>/ (%d canonical + %d commands = %d).\n", canonical, cmds, codex.Skills.Expected)
	}
	if hasGrok {
		cmds := grok.Commands.Expected
		canonical := grok.Skills.Expected - cmds
		b.WriteString("\n")
		fmt.Fprintf(&b, "  grok has no standalone command loader — its %d commands install as skills under\n", cmds)
		fmt.Fprintf(&b, "  .grok/skills/command-<name>/ (%d canonical + %d commands = %d).\n", canonical, cmds, grok.Skills.Expected)
	}

	b.WriteString("\n")
	return b.String()
}

// kindShort reports whether an installed, applicable content kind is below its
// expected count — the per-cell shortfall predicate. A NotApplicable cell
// (codex commands) is never short.
func kindShort(k install.KindCount) bool {
	return !k.NotApplicable && k.Actual < k.Expected
}

// kindCell renders one count cell: "actual/expected" (with a trailing " !"
// when short), or "—" for a NotApplicable kind.
func kindCell(k install.KindCount) string {
	if k.NotApplicable {
		return "—"
	}
	cell := fmt.Sprintf("%d/%d", k.Actual, k.Expected)
	if k.Actual < k.Expected {
		cell += " !"
	}
	return cell
}

// renderInventoryTable formats rows (header first) into an aligned, two-space-
// indented table. Target and root-file columns are left-aligned; the three
// count columns are right-aligned so the numbers line up.
func renderInventoryTable(rows [][]string) string {
	const ncol = 5
	widths := make([]int, ncol)
	for _, r := range rows {
		for i, c := range r {
			if w := len([]rune(c)); w > widths[i] {
				widths[i] = w
			}
		}
	}
	rightAlign := map[int]bool{1: true, 2: true, 3: true}
	var b strings.Builder
	for _, r := range rows {
		var line strings.Builder
		line.WriteString("  ")
		for i, c := range r {
			if i > 0 {
				line.WriteString("   ")
			}
			line.WriteString(padCell(c, widths[i], rightAlign[i]))
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteString("\n")
	}
	return b.String()
}

// padCell pads s to width w with spaces, right- or left-aligned. Width is
// measured in runes so the multibyte em dash aligns correctly.
func padCell(s string, w int, right bool) string {
	n := w - len([]rune(s))
	if n <= 0 {
		return s
	}
	if right {
		return strings.Repeat(" ", n) + s
	}
	return s + strings.Repeat(" ", n)
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
