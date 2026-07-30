package codehost

import (
	"context"
	"testing"
	"time"

	"github.com/hero-engine/hero/contracts/codehostbroker"
)

const cancelAfterApplyResponseDelay = 30 * time.Second

func executeThenCancelAfterAttempt(
	t *testing.T,
	broker *Broker,
	request codehostbroker.Request,
	attempts func() int,
) codehostbroker.Response {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	responses := make(chan codehostbroker.Response, 1)
	go func() {
		responses <- broker.Execute(ctx, request)
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if attempts() == 1 {
			cancel()
			break
		}
		select {
		case response := <-responses:
			t.Fatalf("broker returned before provider attempt: %+v", response)
		case <-timer.C:
			t.Fatal("provider attempt was not observed")
		case <-ticker.C:
		}
	}

	select {
	case response := <-responses:
		return response
	case <-time.After(5 * time.Second):
		t.Fatal("broker did not reconcile after cancellation")
		return codehostbroker.Response{}
	}
}
