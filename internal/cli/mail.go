package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hero-engine/hero/contracts/attention"
	"github.com/hero-engine/hero/internal/attention/mail"
	attentionstate "github.com/hero-engine/hero/internal/attention/state"
	"github.com/hero-engine/hero/internal/config"
	"github.com/spf13/cobra"
)

var mailStateRootOverride string
var mailCmd = newMailCommand()

func newMailCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "mail", Short: "Exchange private asynchronous mail with local Hero projects", SilenceErrors: true, SilenceUsage: true}
	cmd.AddCommand(newMailSendCommand(), newMailInboxCommand(), newMailShowCommand(), newMailReplyCommand(), newMailAckCommand())
	return cmd
}

func newMailSendCommand() *cobra.Command {
	var subject, bodyFile, kind, key, messageID string
	var jsonOut bool
	cmd := &cobra.Command{Use: "send <peer>", Short: "Send mail to a configured local peer", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readMailBody(cmd.InOrStdin(), bodyFile)
		if err != nil {
			return writeMailError(cmd, jsonOut, err)
		}
		svc, err := mailService()
		if err != nil {
			return writeMailError(cmd, jsonOut, err)
		}
		result, err := svc.Send(mail.SendRequest{RecipientAlias: args[0], Subject: subject, Body: body, Kind: kind, MessageID: messageID, IdempotencyKey: key})
		if err != nil {
			return writeMailError(cmd, jsonOut, err)
		}
		if jsonOut {
			return writeMailJSON(cmd.OutOrStdout(), result)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Sent %s to %s: %s\n", result.MessageID, result.Recipient.DisplayName, subject)
		return nil
	}}
	cmd.Flags().StringVar(&subject, "subject", "", "subject (required)")
	cmd.Flags().StringVar(&bodyFile, "body-file", "-", "body file path, or - for stdin")
	cmd.Flags().StringVar(&kind, "kind", attention.MailKindNotice, "message kind")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "retry key (defaults to message ID)")
	cmd.Flags().StringVar(&messageID, "message-id", "", "caller-supplied message ID")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	_ = cmd.MarkFlagRequired("subject")
	return cmd
}
func newMailInboxCommand() *cobra.Command {
	var project string
	var unread, jsonOut bool
	cmd := &cobra.Command{Use: "inbox", Short: "List received mail", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		svc, e := mailService()
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		items, e := svc.Inbox(project, unread)
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		if jsonOut {
			return writeMailJSON(cmd.OutOrStdout(), items)
		}
		for _, v := range items {
			status := "unread"
			if v.Receipt != nil && v.Receipt.ReadAt != "" {
				status = "read"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", v.ID, status, v.Sender.DisplayName, v.Subject)
		}
		return nil
	}}
	cmd.Flags().StringVar(&project, "project", "", "peer alias or peer ID mailbox")
	cmd.Flags().BoolVar(&unread, "unread", false, "show unread mail only")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}
func newMailShowCommand() *cobra.Command {
	var noMark, jsonOut bool
	cmd := &cobra.Command{Use: "show <message-id>", Short: "Show a message", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		svc, e := mailService()
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		v, e := svc.Show(args[0], !noMark)
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		if jsonOut {
			return writeMailJSON(cmd.OutOrStdout(), v)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s [%s]\nFrom: %s\nSubject: %s\n\n%s\n", v.ID, v.Kind, v.Sender.DisplayName, v.Subject, v.Body)
		return nil
	}}
	cmd.Flags().BoolVar(&noMark, "no-mark-read", false, "do not mark the message read")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}
func newMailReplyCommand() *cobra.Command {
	var bodyFile, subject, kind, key string
	var jsonOut bool
	cmd := &cobra.Command{Use: "reply <message-id>", Short: "Reply to a received message", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		body, e := readMailBody(cmd.InOrStdin(), bodyFile)
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		svc, e := mailService()
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		v, e := svc.Reply(mail.ReplyRequest{MessageID: args[0], Subject: subject, Body: body, Kind: kind, IdempotencyKey: key})
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		if jsonOut {
			return writeMailJSON(cmd.OutOrStdout(), v)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Sent reply %s in thread %s\n", v.MessageID, v.ThreadID)
		return nil
	}}
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "body file path, or - for stdin (required)")
	cmd.Flags().StringVar(&subject, "subject", "", "reply subject (defaults to Re: original)")
	cmd.Flags().StringVar(&kind, "kind", attention.MailKindResponse, "message kind")
	cmd.Flags().StringVar(&key, "idempotency-key", "", "retry key")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}
func newMailAckCommand() *cobra.Command {
	var note string
	var jsonOut bool
	cmd := &cobra.Command{Use: "ack <message-id>", Short: "Acknowledge a message", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		svc, e := mailService()
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		v, e := svc.Ack(args[0], note)
		if e != nil {
			return writeMailError(cmd, jsonOut, e)
		}
		if jsonOut {
			return writeMailJSON(cmd.OutOrStdout(), v)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Acknowledged %s\n", args[0])
		return nil
	}}
	cmd.Flags().StringVar(&note, "note", "", "short acknowledgement note")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func mailService() (*mail.Service, error) {
	project := findProjectRoot()
	root := mailStateRootOverride
	var err error
	if root == "" {
		root, err = attentionstate.Ensure(attentionstate.Options{ProjectRoot: project})
	} else {
		root, err = attentionstate.Ensure(attentionstate.Options{Root: root})
	}
	if err != nil {
		return nil, err
	}
	store, err := mail.NewStore(root)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(project)
	if err != nil {
		return nil, err
	}
	return mail.NewService(store, project, cfg), nil
}
func readMailBody(stdin io.Reader, path string) (string, error) {
	if path == "" {
		return "", errors.New("--body-file is required (use - for stdin)")
	}
	var b []byte
	var err error
	if path == "-" {
		b, err = io.ReadAll(io.LimitReader(stdin, attention.MaxBodyBytes+1))
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("read mail body: %w", err)
	}
	if len(b) > attention.MaxBodyBytes {
		return "", errors.New("mail body exceeds 65536 bytes")
	}
	return string(b), nil
}

type mailError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeMailError(cmd *cobra.Command, jsonOut bool, err error) error {
	if !jsonOut {
		return mailCLIError(err)
	}
	code := "validation"
	if errors.Is(err, mail.ErrNotFound) {
		code = "missing"
	}
	if errors.Is(err, mail.ErrIdempotencyConflict) {
		code = "idempotency_conflict"
	}
	_ = writeMailJSON(cmd.OutOrStdout(), mailError{Code: code, Message: err.Error()})
	return mailCLIError(err)
}
func mailCLIError(err error) error {
	if errors.Is(err, mail.ErrNotFound) {
		return fmt.Errorf("missing: %w", err)
	}
	if errors.Is(err, mail.ErrIdempotencyConflict) {
		return fmt.Errorf("idempotency_conflict: %w", err)
	}
	return err
}
func writeMailJSON(w io.Writer, v any) error {
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	return e.Encode(v)
}
