package tracker

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// FieldErrorKind classifies a field-push failure so the CLI can choose
// the right exit code and message (see the Acceptance Criteria for
// auth/rate-limit handling).
type FieldErrorKind int

const (
	// FieldErrorOther is any failure not specially classified.
	FieldErrorOther FieldErrorKind = iota
	// FieldErrorAuth is a 401/403 — credential / permission problem.
	FieldErrorAuth
	// FieldErrorRateLimited is a 429 that survived the single retry.
	FieldErrorRateLimited
)

// FieldError wraps a tracker field-push failure with a classification.
// The CLI maps Kind to an exit code: FieldErrorAuth → 2, others → 1.
type FieldError struct {
	Kind    FieldErrorKind
	Status  int
	Message string
}

func (e *FieldError) Error() string { return e.Message }

// classifyHTTPError builds a FieldError from a non-2xx response. The
// caller passes the provider name and already-read body for the
// message. Auth failures (401/403) get a credential-config hint.
func classifyHTTPError(provider string, status int, body string) *FieldError {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &FieldError{
			Kind:    FieldErrorAuth,
			Status:  status,
			Message: fmt.Sprintf("%s API returned %d — check tracker credentials (token / API key) in hero.json or the environment: %s", provider, status, body),
		}
	case http.StatusTooManyRequests:
		return &FieldError{
			Kind:    FieldErrorRateLimited,
			Status:  status,
			Message: fmt.Sprintf("%s API rate-limited (429) after one retry: %s", provider, body),
		}
	default:
		return &FieldError{
			Kind:    FieldErrorOther,
			Status:  status,
			Message: fmt.Sprintf("%s API returned %d: %s", provider, status, body),
		}
	}
}

// maxRetryAfter caps the honored Retry-After delay so a hostile or
// misconfigured tracker can't park the CLI for minutes.
const maxRetryAfter = 30 * time.Second

// doWithRetry performs req via client. On a 429 it retries exactly once
// after honoring the Retry-After header (capped at maxRetryAfter),
// using rebuild to produce a fresh *http.Request (request bodies are
// single-use, so the caller supplies a factory). The returned response
// body is left open for the caller to read and close. sleep is
// injectable so tests don't actually wait.
func doWithRetry(client *http.Client, rebuild func() (*http.Request, error), sleep func(time.Duration)) (*http.Response, error) {
	if sleep == nil {
		sleep = time.Sleep
	}
	req, err := rebuild()
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		return resp, nil
	}

	// 429 — retry once after the advised delay.
	delay := parseRetryAfter(resp.Header.Get("Retry-After"))
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if delay > maxRetryAfter {
		delay = maxRetryAfter
	}
	if delay > 0 {
		sleep(delay)
	}

	retryReq, err := rebuild()
	if err != nil {
		return nil, err
	}
	return client.Do(retryReq)
}

// parseRetryAfter parses a Retry-After header value (delta-seconds
// form only; HTTP-date form is ignored and treated as zero). Returns 0
// on empty / unparseable input.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}
