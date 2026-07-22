package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	brokercontract "github.com/hero-engine/hero/contracts/trackerbroker"
	evidencecontract "github.com/hero-engine/hero/contracts/trackerevidence"
	"github.com/hero-engine/hero/internal/tracker"
	"github.com/spf13/cobra"
)

var trackerCmd = &cobra.Command{
	Use:   "tracker",
	Short: "Credential-safe structured tracker access",
}

var trackerBrokerCmd = &cobra.Command{
	Use:   "broker <get_issue|search|request|cli>",
	Short: "Execute a tracker-broker/v1 request read from stdin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot := findProjectRoot()
		b := tracker.NewBroker(projectRoot)
		var response brokercontract.Response
		switch args[0] {
		case string(brokercontract.OperationGetIssue):
			var request brokercontract.GetIssueRequest
			if err := decodeBrokerInput(cmd.InOrStdin(), &request); err != nil {
				response = invalidBrokerInput(brokercontract.OperationGetIssue, err)
			} else {
				response = b.GetIssue(cmd.Context(), request)
			}
		case string(brokercontract.OperationSearch):
			var request brokercontract.SearchRequest
			if err := decodeBrokerInput(cmd.InOrStdin(), &request); err != nil {
				response = invalidBrokerInput(brokercontract.OperationSearch, err)
			} else {
				response = b.Search(cmd.Context(), request)
			}
		case string(brokercontract.OperationRequest):
			var request brokercontract.RequestRequest
			if err := decodeBrokerInput(cmd.InOrStdin(), &request); err != nil {
				response = invalidBrokerInput(brokercontract.OperationRequest, err)
			} else {
				response = b.Request(cmd.Context(), request)
			}
		case string(brokercontract.OperationCLI):
			var request brokercontract.CLIRequest
			if err := decodeBrokerInput(cmd.InOrStdin(), &request); err != nil {
				response = invalidBrokerInput(brokercontract.OperationCLI, err)
			} else {
				response = b.CLI(cmd.Context(), request)
			}
		default:
			return fmt.Errorf("unknown broker operation %q", args[0])
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		return enc.Encode(response)
	},
}

var trackerContractCmd = &cobra.Command{
	Use:   "contract [tracker-broker|tracker-evidence]",
	Short: "Print an embedded tracker consumer fixture",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contract := "tracker-broker"
		if len(args) == 1 {
			contract = args[0]
		}
		var fixture []byte
		var err error
		switch contract {
		case "tracker-broker":
			fixture, err = brokercontract.ConsumerFixture()
		case "tracker-evidence":
			fixture, err = evidencecontract.ConsumerFixture()
		default:
			return fmt.Errorf("unknown tracker contract %q", contract)
		}
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(fixture)
		return err
	},
}

func init() {
	trackerCmd.AddCommand(trackerBrokerCmd, trackerContractCmd)
	trackerBrokerCmd.SetIn(os.Stdin)
}

func decodeBrokerInput(r io.Reader, target any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("expected exactly one JSON object")
		}
		return err
	}
	return nil
}

func invalidBrokerInput(op brokercontract.Operation, err error) brokercontract.Response {
	return brokercontract.Response{
		Version: brokercontract.Version, Operation: op, Effect: brokercontract.EffectRead,
		Error: &brokercontract.Error{Code: "invalid_input", Message: err.Error(), Retryable: false},
	}
}
