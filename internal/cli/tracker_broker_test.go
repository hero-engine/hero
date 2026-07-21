package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	brokercontract "github.com/hero-engine/hero/contracts/trackerbroker"
)

func TestTrackerContractCommandReturnsConsumerFixture(t *testing.T) {
	var out bytes.Buffer
	trackerContractCmd.SetOut(&out)
	defer trackerContractCmd.SetOut(nil)
	if err := trackerContractCmd.RunE(trackerContractCmd, nil); err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out.Bytes(), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != brokercontract.Version {
		t.Fatalf("version = %q", fixture.Version)
	}
}

func TestTrackerBrokerCommandReturnsStructuredInvalidInput(t *testing.T) {
	var out bytes.Buffer
	trackerBrokerCmd.SetIn(strings.NewReader(`{"unknown":true}`))
	trackerBrokerCmd.SetOut(&out)
	defer trackerBrokerCmd.SetIn(nil)
	defer trackerBrokerCmd.SetOut(nil)
	if err := trackerBrokerCmd.RunE(trackerBrokerCmd, []string{"get_issue"}); err != nil {
		t.Fatal(err)
	}
	var response brokercontract.Response
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Version != brokercontract.Version || response.Operation != brokercontract.OperationGetIssue || response.Error == nil || response.Error.Code != "invalid_input" {
		t.Fatalf("response = %+v", response)
	}
}
