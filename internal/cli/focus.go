package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/spf13/cobra"
)

var (
	focusStateRootOverride string
	focusResolverLoader    = func() (focus.ProjectResolver, error) { return focus.LoadRegistryResolver(findProjectRoot()) }
)

var focusCmd = newFocusCommand()

func newFocusCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "focus", Short: "Manage private prompt-backed intentions across projects"}
	cmd.AddCommand(newFocusAddCommand(), newFocusListCommand(), newFocusShowCommand(), newFocusMoveCommand(), newFocusDoneCommand(), newFocusLaunchCommand())
	return cmd
}

func newFocusAddCommand() *cobra.Command {
	var title, promptFile, project, lifecycle string
	cmd := &cobra.Command{
		Use: "add", Short: "Add a durable Focus item", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if promptFile == "" {
				return errors.New("--prompt-file is required (use - for stdin)")
			}
			prompt, err := readFocusPrompt(cmd.InOrStdin(), promptFile)
			if err != nil {
				return err
			}
			service, resolver, err := focusService()
			if err != nil {
				return err
			}
			var ref *attention.ProjectReference
			if project == "" {
				ref, err = resolver.ResolveCurrent()
			} else {
				ref, err = resolver.ResolveInput(project)
			}
			if err != nil {
				return err
			}
			item, err := service.Create(focus.CreateRequest{Title: title, Prompt: prompt, Lifecycle: lifecycle, Project: ref})
			if err != nil {
				return focusCLIError(err)
			}
			return writeFocusJSON(cmd.OutOrStdout(), item)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "short title (required)")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "prompt file path, or - for stdin (required)")
	cmd.Flags().StringVar(&project, "project", "", "registered project slug, absolute path, or .")
	cmd.Flags().StringVar(&lifecycle, "state", attention.FocusInbox, "initial state: inbox, today, later, or done")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newFocusListCommand() *cobra.Command {
	var state string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "list", Short: "List Focus items (done excluded by default)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, _, err := focusService()
			if err != nil {
				return err
			}
			items, err := service.List(state)
			if err != nil {
				return focusCLIError(err)
			}
			if jsonOutput {
				return writeFocusJSON(cmd.OutOrStdout(), items)
			}
			for _, item := range items {
				project := "unbound"
				if item.Project != nil {
					project = item.Project.DisplayName + " (" + item.Availability + ")"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\trev:%d\n", item.ID, item.Lifecycle, item.Title, project, item.Revision)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&state, "state", "active", "inbox, today, later, done, or all (default: active)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	return cmd
}

func newFocusShowCommand() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "show <id>", Short: "Show a Focus item", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := focusService()
			if err != nil {
				return err
			}
			item, err := service.Get(args[0])
			if err != nil {
				return focusCLIError(err)
			}
			if jsonOutput {
				return writeFocusJSON(cmd.OutOrStdout(), item)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] rev:%d\nProject: %s\nTitle: %s\n", item.ID, item.Lifecycle, item.Revision, focusProjectLabel(item), item.Title)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	return cmd
}

func newFocusMoveCommand() *cobra.Command {
	var revision int64
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "move <id> <state>", Short: "Move a Focus item with optimistic revision checking", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if revision < 1 {
				return errors.New("--revision must be a positive revision")
			}
			service, _, err := focusService()
			if err != nil {
				return err
			}
			item, err := service.Move(args[0], args[1], revision)
			if err != nil {
				return focusCLIError(err)
			}
			if jsonOutput {
				return writeFocusJSON(cmd.OutOrStdout(), item)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s moved to %s (revision %d)\n", item.ID, item.Lifecycle, item.Revision)
			return nil
		},
	}
	cmd.Flags().Int64Var(&revision, "revision", 0, "expected item revision (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	_ = cmd.MarkFlagRequired("revision")
	return cmd
}

func newFocusDoneCommand() *cobra.Command {
	var revision int64
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "done <id>", Short: "Mark a Focus item done", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if revision < 1 {
				return errors.New("--revision must be a positive revision")
			}
			service, _, err := focusService()
			if err != nil {
				return err
			}
			item, err := service.Move(args[0], attention.FocusDone, revision)
			if err != nil {
				return focusCLIError(err)
			}
			if jsonOutput {
				return writeFocusJSON(cmd.OutOrStdout(), item)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s marked done (revision %d)\n", item.ID, item.Revision)
			return nil
		},
	}
	cmd.Flags().Int64Var(&revision, "revision", 0, "expected item revision (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit JSON")
	_ = cmd.MarkFlagRequired("revision")
	return cmd
}

func newFocusLaunchCommand() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "launch <id>", Short: "Return a safe launch intent without starting a session", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, _, err := focusService()
			if err != nil {
				return err
			}
			intent, err := service.LaunchIntent(args[0])
			if err != nil {
				return focusCLIError(err)
			}
			if jsonOutput {
				return writeFocusJSON(cmd.OutOrStdout(), intent)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\nPath: %s\nPrompt:\n%s", intent.Project.DisplayName, intent.Path, intent.Prompt)
			if !strings.HasSuffix(intent.Prompt, "\n") {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit typed launch intent as JSON")
	return cmd
}

func focusService() (*focus.Service, focus.ProjectResolver, error) {
	root := focusStateRootOverride
	var err error
	if root == "" {
		root, err = attentionstate.Ensure(attentionstate.Options{ProjectRoot: findProjectRoot()})
	} else {
		root, err = attentionstate.Ensure(attentionstate.Options{Root: root})
	}
	if err != nil {
		return nil, nil, err
	}
	store, err := focus.NewStore(root)
	if err != nil {
		return nil, nil, err
	}
	resolver, err := focusResolverLoader()
	if err != nil {
		return nil, nil, err
	}
	return focus.NewService(store, resolver), resolver, nil
}

func readFocusPrompt(stdin io.Reader, path string) (string, error) {
	var b []byte
	var err error
	if path == "-" {
		b, err = io.ReadAll(io.LimitReader(stdin, attention.MaxFocusPromptBytes+1))
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read focus prompt: %w", err)
	}
	if len(b) > attention.MaxFocusPromptBytes {
		return "", errors.New("focus prompt exceeds 65536 bytes")
	}
	return string(b), nil
}

func writeFocusJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func focusProjectLabel(item focus.ListedItem) string {
	if item.Project == nil {
		return "unbound"
	}
	return item.Project.DisplayName + " (" + item.Availability + ")"
}

func focusCLIError(err error) error {
	switch {
	case errors.Is(err, focus.ErrStale):
		return fmt.Errorf("stale: %w", err)
	case errors.Is(err, focus.ErrIdempotencyConflict):
		return fmt.Errorf("idempotency_conflict: %w", err)
	case errors.Is(err, focus.ErrNotFound):
		return fmt.Errorf("missing: %w", err)
	default:
		return err
	}
}
