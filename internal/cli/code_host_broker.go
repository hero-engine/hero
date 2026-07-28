package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
	"github.com/hero-engine/hero/internal/codehost"
	"github.com/spf13/cobra"
)

const maxCodeHostBrokerInputBytes = 1 << 20

type codeHostBrokerService interface {
	Execute(context.Context, codehostbroker.Request) codehostbroker.Response
	Prepare(context.Context, codehostbroker.Request) codehostbroker.PreparationResponse
}

type codeHostContractEmission struct {
	Version  string                           `json:"version"`
	SHA256   string                           `json:"sha256"`
	Bounds   codehostbroker.Bounds            `json:"bounds"`
	Policies []codehostbroker.OperationPolicy `json:"policies"`
	Fixture  string                           `json:"fixture"`
}

var newCodeHostBrokerService = func(projectRoot string) codeHostBrokerService {
	return codehost.NewBroker(projectRoot)
}

var codeHostCmd = newCodeHostCommand()

func newCodeHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code-host",
		Short: "Credential-safe structured code-host access",
	}
	cmd.AddCommand(newCodeHostContractCommand(), newCodeHostBrokerCommand())
	return cmd
}

func newCodeHostContractCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "contract",
		Short: "Print the code-host-broker/v1 consumer contract",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fixture, err := codehostbroker.ConsumerFixture()
			if err != nil {
				return err
			}
			digest, err := codehostbroker.ConsumerFixtureDigest()
			if err != nil {
				return err
			}
			sum := sha256.Sum256(fixture)
			if hex.EncodeToString(sum[:]) != digest {
				return errors.New("embedded code-host consumer fixture digest mismatch")
			}
			policies := codehostbroker.Policies()
			var bounds codehostbroker.Bounds
			if len(policies) > 0 {
				bounds = policies[0].Bounds
			}
			return encodeCodeHostJSON(cmd.OutOrStdout(), codeHostContractEmission{
				Version:  codehostbroker.Version,
				SHA256:   digest,
				Bounds:   bounds,
				Policies: policies,
				Fixture:  string(fixture),
			})
		},
	}
}

func newCodeHostBrokerCommand() *cobra.Command {
	var prepare bool
	cmd := &cobra.Command{
		Use:   "broker <operation>",
		Short: "Execute one code-host-broker/v1 request read from stdin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			operation := codehostbroker.Operation(args[0])
			if !codehostbroker.IsOperation(operation) {
				return fmt.Errorf("unknown code-host broker operation %q", args[0])
			}
			if prepare && codehostbroker.IsRead(operation) {
				return encodeCodeHostJSON(cmd.OutOrStdout(), codeHostCLIPreparationError(
					operation,
					codehostbroker.ErrorUnsupportedOperation,
					"read operations do not support preparation",
					"operation",
				))
			}
			request, decodeErr := decodeCodeHostBrokerInput(cmd.InOrStdin())
			if decodeErr != nil {
				code := codehostbroker.ErrorInvalidInput
				if errors.Is(decodeErr, errCodeHostInputTooLarge) {
					code = codehostbroker.ErrorInputTooLarge
				}
				if prepare {
					return encodeCodeHostJSON(cmd.OutOrStdout(), codeHostCLIPreparationError(operation, code, "stdin must contain exactly one bounded code-host-broker/v1 request", "stdin"))
				}
				return encodeCodeHostJSON(cmd.OutOrStdout(), codeHostCLIErrorResponse(operation, code, "stdin must contain exactly one bounded code-host-broker/v1 request", "stdin"))
			}
			if request.Operation != operation {
				if prepare {
					return encodeCodeHostJSON(cmd.OutOrStdout(), codeHostCLIPreparationError(operation, codehostbroker.ErrorInvalidInput, "request operation does not match the command operation", "operation"))
				}
				return encodeCodeHostJSON(cmd.OutOrStdout(), codeHostCLIErrorResponse(operation, codehostbroker.ErrorInvalidInput, "request operation does not match the command operation", "operation"))
			}
			broker := newCodeHostBrokerService(findProjectRoot())
			if prepare {
				return encodeCodeHostJSON(cmd.OutOrStdout(), broker.Prepare(cmd.Context(), request))
			}
			return encodeCodeHostJSON(cmd.OutOrStdout(), broker.Execute(cmd.Context(), request))
		},
	}
	cmd.Flags().BoolVar(&prepare, "prepare", false, "read current mutation revisions without executing the mutation")
	return cmd
}

var errCodeHostInputTooLarge = errors.New("code-host broker input exceeds bound")

func decodeCodeHostBrokerInput(reader io.Reader) (codehostbroker.Request, error) {
	limited := &io.LimitedReader{R: reader, N: maxCodeHostBrokerInputBytes + 1}
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	var request codehostbroker.Request
	if err := decoder.Decode(&request); err != nil {
		if limited.N <= 0 {
			return codehostbroker.Request{}, errCodeHostInputTooLarge
		}
		return codehostbroker.Request{}, err
	}
	var extra any
	err := decoder.Decode(&extra)
	if limited.N <= 0 {
		return codehostbroker.Request{}, errCodeHostInputTooLarge
	}
	if err != io.EOF {
		return codehostbroker.Request{}, errors.New("expected exactly one JSON object")
	}
	return request, nil
}

func codeHostCLIPreparationError(operation codehostbroker.Operation, code, message, field string) codehostbroker.PreparationResponse {
	policy, _ := codehostbroker.Policy(operation)
	return codehostbroker.PreparationResponse{
		Version: codehostbroker.Version, Operation: operation,
		Error: &codehostbroker.ContractError{
			Code: code, Message: boundedCodeHostCLIText(message, policy.Bounds.ErrorDetailBytes),
			Field: field, Retry: codehostbroker.RetryForError(code),
		},
	}
}

func codeHostCLIErrorResponse(operation codehostbroker.Operation, code, message, field string) codehostbroker.Response {
	policy, _ := codehostbroker.Policy(operation)
	observedAt := time.Now().UTC().Format(time.RFC3339Nano)
	repository := codehostbroker.RepositoryIdentity{
		Host: "unresolved.invalid", Owner: "unresolved", Name: "unresolved", FullName: "unresolved/unresolved",
	}
	return codehostbroker.Response{
		Version: codehostbroker.Version, Operation: operation,
		Provider: "unresolved", ConnectionID: "unresolved", Repository: repository,
		Policy: policy, Bounds: policy.Bounds,
		CapabilityRevision:  "capability:cli-input-rejected",
		ObservationRevision: "observation:cli-input-rejected",
		ObservedAt:          observedAt, Freshness: codehostbroker.FreshnessUnavailable,
		RateLimit:       codehostbroker.RateLimit{ObservedAt: observedAt},
		Completeness:    codehostbroker.CompletenessUnavailable,
		PartialFailures: []codehostbroker.PartialFailure{},
		Error: &codehostbroker.ContractError{
			Code: code, Message: boundedCodeHostCLIText(message, policy.Bounds.ErrorDetailBytes),
			Field: field, Retry: codehostbroker.RetryForError(code),
		},
	}
}

func boundedCodeHostCLIText(value string, limit int) string {
	value = strings.ToValidUTF8(value, "")
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func encodeCodeHostJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
