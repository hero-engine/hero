package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	evidencecontract "github.com/hero-engine/hero/contracts/trackerevidence"
)

func TestTrackerContractCommandReturnsEvidenceConsumerFixture(t *testing.T) {
	var out bytes.Buffer
	trackerContractCmd.SetOut(&out)
	defer trackerContractCmd.SetOut(nil)
	if err := trackerContractCmd.RunE(trackerContractCmd, []string{"tracker-evidence"}); err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out.Bytes(), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != evidencecontract.Version {
		t.Fatalf("version = %q", fixture.Version)
	}
}

func TestTrackerContractCommandDefaultsToBrokerFixture(t *testing.T) {
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
	if fixture.Version == evidencecontract.Version {
		t.Fatalf("default contract unexpectedly changed to %q", fixture.Version)
	}
}
