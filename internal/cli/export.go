package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/knowledge"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var exportConflict = string(knowledge.ConflictFail)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export Hero workspace artifacts",
}

var exportKnowledgeCmd = &cobra.Command{
	Use:   "knowledge <destination>",
	Short: "Export the knowledge base to a destination directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runExportKnowledge,
}

var exportMocksCmd = &cobra.Command{
	Use:   "mocks <destination>",
	Short: "Export mock artifacts to a destination directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runExportMocks,
}

func init() {
	exportKnowledgeCmd.Flags().StringVar(&exportConflict, "conflict", string(knowledge.ConflictFail), "conflict strategy: fail, skip, overwrite, merge, interactive")
	exportMocksCmd.Flags().StringVar(&exportConflict, "conflict", string(knowledge.ConflictFail), "conflict strategy: fail, skip, overwrite, merge, interactive")
	exportCmd.AddCommand(exportKnowledgeCmd)
	exportCmd.AddCommand(exportMocksCmd)
}

func runExportKnowledge(cmd *cobra.Command, args []string) error {
	strategy := knowledge.ConflictStrategy(exportConflict)
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	source := cfg.KnowledgeDir(projectRoot)
	destination := args[0]

	opts := knowledge.Options{Strategy: strategy}
	if strategy == knowledge.ConflictInteractive {
		in := cmd.InOrStdin()
		out := cmd.OutOrStdout()
		if !exportIsTerminal(in) || !exportIsTerminal(out) {
			return fmt.Errorf("--conflict interactive requires an attached terminal")
		}
		opts.Prompt = promptConflictStrategy(in, out)
	}

	summary, err := knowledge.Export(source, destination, opts)
	if err != nil {
		if summary != nil {
			printExportKnowledgeSummary(cmd, destination, summary)
		}
		return err
	}
	printExportKnowledgeSummary(cmd, destination, summary)
	return nil
}

func runExportMocks(cmd *cobra.Command, args []string) error {
	strategy := knowledge.ConflictStrategy(exportConflict)
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	source := cfg.MocksDir(projectRoot)
	destination := args[0]

	opts := knowledge.Options{Strategy: strategy}
	if strategy == knowledge.ConflictInteractive {
		in := cmd.InOrStdin()
		out := cmd.OutOrStdout()
		if !exportIsTerminal(in) || !exportIsTerminal(out) {
			return fmt.Errorf("--conflict interactive requires an attached terminal")
		}
		opts.Prompt = promptConflictStrategy(in, out)
	}

	summary, err := knowledge.ExportMocks(source, destination, opts)
	if err != nil {
		if summary != nil {
			printExportSummary(cmd, "Mocks export destination", destination, summary)
		}
		return err
	}
	printExportSummary(cmd, "Mocks export destination", destination, summary)
	return nil
}

func printExportKnowledgeSummary(cmd *cobra.Command, destination string, summary *knowledge.Summary) {
	printExportSummary(cmd, "Knowledge export destination", destination, summary)
}

func printExportSummary(cmd *cobra.Command, label, destination string, summary *knowledge.Summary) {
	destAbs, err := filepath.Abs(destination)
	if err != nil {
		destAbs = destination
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", label, destAbs)
	fmt.Fprintf(cmd.OutOrStdout(), "copied=%d skipped=%d overwritten=%d merged=%d identical=%d conflicts=%d\n",
		summary.Copied, summary.Skipped, summary.Overwritten, summary.Merged, summary.Identical, summary.Conflicts)
}

func promptConflictStrategy(in io.Reader, out io.Writer) func(knowledge.Conflict) (knowledge.ConflictStrategy, error) {
	reader := bufio.NewReader(in)
	return func(c knowledge.Conflict) (knowledge.ConflictStrategy, error) {
		for {
			fmt.Fprintf(out, "Conflict: %s (%s)\n", c.RelPath, c.Reason)
			fmt.Fprintf(out, "Source: %s\nDestination: %s\n", c.SourcePath, c.DestPath)
			fmt.Fprint(out, "Choose [fail/skip/overwrite/merge]: ")
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return "", err
			}
			choice := knowledge.ConflictStrategy(strings.ToLower(strings.TrimSpace(line)))
			switch choice {
			case knowledge.ConflictFail, knowledge.ConflictSkip, knowledge.ConflictOverwrite, knowledge.ConflictMerge:
				return choice, nil
			default:
				fmt.Fprintf(out, "Invalid choice %q.\n", strings.TrimSpace(line))
			}
			if err == io.EOF {
				return "", io.EOF
			}
		}
	}
}

func exportIsTerminal(r any) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
