package serve

import (
	"bytes"
	"context"
	"encoding/json"

	brokercontract "github.com/hero-engine/hero/contracts/trackerbroker"
	evidencecontract "github.com/hero-engine/hero/contracts/trackerevidence"
	"github.com/hero-engine/hero/internal/tracker"
)

func (s *MCPServer) toolTrackerLoadEvidence(args map[string]interface{}) (string, error) {
	var request evidencecontract.Request
	data, err := json.Marshal(args)
	if err == nil {
		err = json.Unmarshal(data, &request)
	}
	if err != nil {
		status := evidencecontract.Status{
			Version: evidencecontract.Version,
			Status:  evidencecontract.StateUnavailable,
			Error:   &evidencecontract.Error{Code: evidencecontract.ErrorProviderUnavailable, Message: "invalid evidence load request"},
		}
		out, _ := json.Marshal(status)
		return string(out), nil
	}
	status := tracker.NewEvidenceLoader(s.projectRoot).Load(s.ctx, request)
	out, err := json.Marshal(status)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *MCPServer) toolTrackerGetIssue(args map[string]interface{}) (string, error) {
	var request brokercontract.GetIssueRequest
	if err := decodeTrackerBrokerArgs(args, &request); err != nil {
		return encodeTrackerBrokerResponse(invalidTrackerBrokerResponse(brokercontract.OperationGetIssue, err)), nil
	}
	return encodeTrackerBrokerResponse(tracker.NewBroker(s.projectRoot).GetIssue(context.Background(), request)), nil
}

func (s *MCPServer) toolTrackerSearch(args map[string]interface{}) (string, error) {
	var request brokercontract.SearchRequest
	if err := decodeTrackerBrokerArgs(args, &request); err != nil {
		return encodeTrackerBrokerResponse(invalidTrackerBrokerResponse(brokercontract.OperationSearch, err)), nil
	}
	return encodeTrackerBrokerResponse(tracker.NewBroker(s.projectRoot).Search(context.Background(), request)), nil
}

func (s *MCPServer) toolTrackerRequest(args map[string]interface{}) (string, error) {
	var request brokercontract.RequestRequest
	if err := decodeTrackerBrokerArgs(args, &request); err != nil {
		return encodeTrackerBrokerResponse(invalidTrackerBrokerResponse(brokercontract.OperationRequest, err)), nil
	}
	return encodeTrackerBrokerResponse(tracker.NewBroker(s.projectRoot).Request(context.Background(), request)), nil
}

func (s *MCPServer) toolTrackerCLI(args map[string]interface{}) (string, error) {
	var request brokercontract.CLIRequest
	if err := decodeTrackerBrokerArgs(args, &request); err != nil {
		return encodeTrackerBrokerResponse(invalidTrackerBrokerResponse(brokercontract.OperationCLI, err)), nil
	}
	return encodeTrackerBrokerResponse(tracker.NewBroker(s.projectRoot).CLI(context.Background(), request)), nil
}

func decodeTrackerBrokerArgs(args map[string]interface{}, target any) error {
	b, err := json.Marshal(args)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}

func invalidTrackerBrokerResponse(op brokercontract.Operation, err error) brokercontract.Response {
	return brokercontract.Response{
		Version: brokercontract.Version, Operation: op, Effect: brokercontract.EffectRead,
		Error: &brokercontract.Error{Code: "invalid_input", Message: err.Error()},
	}
}

func encodeTrackerBrokerResponse(response brokercontract.Response) string {
	b, _ := json.Marshal(response)
	return string(b)
}
