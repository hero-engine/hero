package codehost

import (
	"context"
	"strings"
	"testing"

	"github.com/hero-engine/hero/contracts/codehostbroker"
)

func TestPrepareRejectsReadsWithoutEchoingRequestMaterial(t *testing.T) {
	request := codehostbroker.Request{
		Version:   codehostbroker.Version,
		Operation: codehostbroker.OperationGetPullRequest,
		Payload:   []byte(`{"body":"prepare-canary"}`),
	}
	response := NewBroker(t.TempDir()).Prepare(context.Background(), request)
	if response.Error == nil || response.Error.Code != codehostbroker.ErrorUnsupportedOperation {
		t.Fatalf("response=%+v", response)
	}
	if err := codehostbroker.ValidatePreparationResponse(response); err != nil {
		t.Fatalf("read preparation envelope validation=%v", err)
	}
	if strings.Contains(response.Error.Message, "prepare-canary") {
		t.Fatal("preparation error echoed request material")
	}
}
