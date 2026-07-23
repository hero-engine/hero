package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/focus"
	"github.com/hero-engine/hero/internal/attention/projection"
	"github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/attention/suggestion"
	"github.com/hero-engine/hero/internal/projectregistry"
	"github.com/spf13/cobra"
)

var attentionCmd = newAttentionCommand()
var attentionProjectionServiceLoader = attentionProjectionService

func newAttentionCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "attention", Short: "Read and act on user-global attention"}
	cmd.AddCommand(newAttentionTodayCommand(), newAttentionActCommand())
	return cmd
}

func newAttentionTodayCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use: "today", Short: "Show unread Mail, Today Focus, and pending suggestions",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			service, err := attentionProjectionServiceLoader()
			if err != nil {
				return err
			}
			snapshot, err := service.Snapshot()
			if err != nil {
				return err
			}
			if jsonOut {
				return writeFocusJSON(cmd.OutOrStdout(), snapshot)
			}
			for _, row := range snapshot.Rows {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", row.ID, row.Group, row.Availability, row.Title)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the v1 Attention snapshot as JSON")
	return cmd
}

func newAttentionActCommand() *cobra.Command {
	var revision int64
	var key, input string
	var jsonOut bool
	cmd := &cobra.Command{
		Use: "act <row-id> <action-id>", Short: "Dispatch an action advertised by an Attention row",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := attentionProjectionServiceLoader()
			if err != nil {
				return err
			}
			raw := json.RawMessage(strings.TrimSpace(input))
			if len(raw) == 0 {
				raw = json.RawMessage(`{}`)
			}
			if !json.Valid(raw) {
				return fmt.Errorf("--input must be valid JSON")
			}
			result := service.Dispatch(attention.ActionRequest{
				SchemaVersion: attention.SchemaVersion, RowID: args[0], ActionID: args[1],
				RowRevision: revision, IdempotencyKey: key, Input: raw,
			})
			if jsonOut {
				return writeFocusJSON(cmd.OutOrStdout(), result)
			}
			if result.Error != nil {
				return result.Error
			}
			if result.RemovedRowID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "%s removed from Today\n", result.RemovedRowID)
			} else if result.Row != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "%s updated\n", result.Row.ID)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Action completed")
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&revision, "revision", 0, "required row revision")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "stable retry key")
	cmd.Flags().StringVar(&input, "input", "{}", "action input as a JSON object")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the v1 Attention action result as JSON")
	return cmd
}

func attentionProjectionService() (*projection.Service, error) {
	rootOverride := focusStateRootOverride
	if rootOverride == "" {
		rootOverride = mailStateRootOverride
	}
	var (
		root string
		err  error
	)
	if rootOverride == "" {
		root, err = state.Ensure(state.Options{ProjectRoot: findProjectRoot()})
	} else {
		root, err = state.Ensure(state.Options{Root: rootOverride})
	}
	if err != nil {
		return nil, err
	}
	registry, err := projectregistry.Load()
	if err != nil {
		return nil, err
	}
	mailSource, err := projection.NewRegistryMailSource(root, registry)
	if err != nil {
		return nil, err
	}
	focusStore, err := focus.NewStore(root)
	if err != nil {
		return nil, err
	}
	resolver, err := focusResolverLoader()
	if err != nil {
		return nil, err
	}
	focusSource := focus.NewService(focusStore, resolver)
	suggestionStore, err := suggestion.NewStore(root)
	if err != nil {
		return nil, err
	}
	suggestionSource := suggestion.NewService(suggestionStore, focusSource, resolver)
	return projection.NewService(mailSource, focusSource, suggestionSource), nil
}
