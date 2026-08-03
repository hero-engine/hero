package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/knowledge"
	"github.com/spf13/cobra"
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
		// Two predicates, not one: this gate needs BOTH an input terminal to
		// read the answer from and an output terminal to render the conflict
		// on. The local helper this replaced answered both questions with a
		// single function, which is what made it look interchangeable with
		// brief.go's output-only check.
		if !prompt.IsInputTTY(in) || !prompt.IsOutputTTY(out) {
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
		// Two predicates, not one: this gate needs BOTH an input terminal to
		// read the answer from and an output terminal to render the conflict
		// on. The local helper this replaced answered both questions with a
		// single function, which is what made it look interchangeable with
		// brief.go's output-only check.
		if !prompt.IsInputTTY(in) || !prompt.IsOutputTTY(out) {
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
	// Not prompt.Choice: this site re-asks on an invalid answer instead of
	// erroring, and its option list renders slash-separated. Choice would
	// rewrite both, and this child is not allowed a single visible change.
	// prompt.Prompt removes the bufio fork without touching either.
	//
	// The reader used to be hoisted out of the closure so a buffered
	// read-ahead could not swallow the next conflict's answer. prompt.Prompt
	// reads a byte at a time and never reads past the newline, so each call is
	// self-contained and no shared state is needed.
	return func(c knowledge.Conflict) (knowledge.ConflictStrategy, error) {
		for {
			fmt.Fprintf(out, "Conflict: %s (%s)\n", c.RelPath, c.Reason)
			fmt.Fprintf(out, "Source: %s\nDestination: %s\n", c.SourcePath, c.DestPath)
			// promptLine, not prompt.Prompt: this loop exits on end-of-stream,
			// and `unterminated` is the same condition bufio's ReadString
			// reported as io.EOF — including the no-data case. Without it a
			// trailing invalid answer with no newline would re-ask once more
			// before stopping. See prompt_line.go.
			line, atEOF, err := promptLine(in, out, "Choose [fail/skip/overwrite/merge]: ")
			if err != nil && !errors.Is(err, io.EOF) {
				return "", err
			}
			// A partial answer before end-of-stream is still an answer, so the
			// EOF check comes after the switch, exactly as it did before.
			// Answers are case-folded: `OVERWRITE` and `Skip` have always been
			// accepted.
			choice := knowledge.ConflictStrategy(strings.ToLower(line))
			switch choice {
			case knowledge.ConflictFail, knowledge.ConflictSkip, knowledge.ConflictOverwrite, knowledge.ConflictMerge:
				return choice, nil
			default:
				fmt.Fprintf(out, "Invalid choice %q.\n", line)
			}
			if atEOF {
				return "", io.EOF
			}
		}
	}
}
