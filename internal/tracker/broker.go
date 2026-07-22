package tracker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	brokercontract "github.com/hero-engine/hero/contracts/trackerbroker"
	"github.com/hero-engine/hero/internal/config"
)

const (
	defaultBrokerOutputLimit = 64 * 1024
	maxBrokerOutputLimit     = 1024 * 1024
	maxBrokerSearchLimit     = 100
	maxBrokerRedirects       = 5
	brokerTimeout            = 30 * time.Second
)

// Broker is Hero's in-process credential boundary. The CLI and MCP layers are
// intentionally thin adapters over these methods.
type Broker struct {
	projectRoot string
	loadConfig  func(string) (config.Config, error)
	httpClient  *http.Client
	lookPath    func(string) (string, error)
	now         func() time.Time
}

func NewBroker(projectRoot string) *Broker {
	return &Broker{
		projectRoot: projectRoot,
		loadConfig:  config.Load,
		httpClient:  &http.Client{Timeout: brokerTimeout},
		lookPath:    exec.LookPath,
		now:         time.Now,
	}
}

type brokerResolved struct {
	connection config.TrackerConnection
	token      config.Secret
	cfg        config.Config
	tracker    Tracker
}

func (b *Broker) resolve(connectionID string, needTracker bool) (brokerResolved, error) {
	cfg, err := b.loadConfig(b.projectRoot)
	if err != nil {
		return brokerResolved{}, err
	}
	cn, err := cfg.ResolveTrackerConnection(connectionID)
	if err != nil {
		return brokerResolved{}, err
	}
	token, err := cn.ResolveToken()
	if err != nil {
		return brokerResolved{}, err
	}
	if len(token.Reveal()) > defaultBrokerOutputLimit {
		return brokerResolved{}, fmt.Errorf("integration %q credential exceeds the 65536-byte safety bound", cn.ID)
	}
	r := brokerResolved{connection: cn, token: token, cfg: cfg}
	if needTracker {
		tc := cn.TrackerConfig()
		tc.Token = token.Reveal()
		r.tracker, err = NewWithJiraConfig(tc, cfg.Jira, filepath.Join(b.projectRoot, ".hero", "knowledge", "tracker"))
		if err != nil {
			return brokerResolved{}, err
		}
	}
	return r, nil
}

func baseResponse(op brokercontract.Operation) brokercontract.Response {
	return brokercontract.Response{Version: brokercontract.Version, Operation: op, Effect: brokercontract.EffectRead}
}

func finishResponse(resp brokercontract.Response, start time.Time, now func() time.Time) brokercontract.Response {
	resp.DurationMS = now().Sub(start).Milliseconds()
	if resp.DurationMS < 0 {
		resp.DurationMS = 0
	}
	return resp
}

func failResponse(resp brokercontract.Response, code, message string, retryable bool) brokercontract.Response {
	message, _ = truncateString(message, 8*1024, false)
	resp.Error = &brokercontract.Error{Code: code, Message: message, Retryable: retryable}
	return resp
}

func selectionErrorCode(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "connection_id is required"):
		return "ambiguous_connection"
	case strings.Contains(s, "not found"):
		return "connection_not_found"
	case strings.Contains(s, "no tracker"):
		return "connection_unavailable"
	case strings.Contains(s, "no usable credential"):
		return "credential_unavailable"
	default:
		return "configuration_error"
	}
}

func (b *Broker) GetIssue(ctx context.Context, in brokercontract.GetIssueRequest) brokercontract.Response {
	start := b.now()
	resp := baseResponse(brokercontract.OperationGetIssue)
	if strings.TrimSpace(in.IssueID) == "" {
		return finishResponse(failResponse(resp, "invalid_issue_id", "issue_id is required", false), start, b.now)
	}
	if in.Detail == "" {
		in.Detail = brokercontract.DetailNormalized
	}
	if in.Detail != brokercontract.DetailNormalized && in.Detail != brokercontract.DetailEvidence {
		return finishResponse(failResponse(resp, "invalid_detail", "detail must be normalized or evidence", false), start, b.now)
	}
	r, err := b.resolve(in.ConnectionID, true)
	if err != nil {
		return finishResponse(failResponse(resp, selectionErrorCode(err), safeError(err, ""), false), start, b.now)
	}
	resp.Provider, resp.ConnectionID = r.connection.Provider, r.connection.ID
	redact := newRedactor(r.token.Reveal(), r.connection.UserEmail)

	select {
	case <-ctx.Done():
		return finishResponse(failResponse(resp, "cancelled", ctx.Err().Error(), false), start, b.now)
	default:
	}
	issueID := strings.TrimSpace(in.IssueID)
	t := r.tracker
	if r.connection.Provider == "github" {
		if project, id, ok := splitProjectIssue(issueID); ok {
			if strings.Count(project, "/") != 1 || strings.ContainsAny(project, `:@\\`) {
				return finishResponse(failResponse(resp, "invalid_issue_id", "GitHub issue_id must use owner/repo#number", false), start, b.now)
			}
			t, err = newGitHub(project, r.token.Reveal(), r.connection.BaseURL)
			issueID = id
		}
	} else if r.connection.Provider == "gitlab" {
		if project, id, ok := splitProjectIssue(issueID); ok {
			if strings.HasPrefix(project, "/") || strings.ContainsAny(project, `:@\\`) {
				return finishResponse(failResponse(resp, "invalid_issue_id", "GitLab issue_id must use namespace/project#iid", false), start, b.now)
			}
			t, err = newGitLab(project, r.token.Reveal(), r.connection.BaseURL)
			issueID = id
		}
	}
	if err != nil {
		return finishResponse(failResponse(resp, "invalid_issue_id", redact.apply(err.Error()), false), start, b.now)
	}
	if err := validateBrokerIssueID(r.connection.Provider, issueID); err != nil {
		return finishResponse(failResponse(resp, "invalid_issue_id", err.Error(), false), start, b.now)
	}
	if in.Detail == brokercontract.DetailEvidence {
		evidenceTracker, ok := t.(EvidenceTracker)
		if !ok {
			return finishResponse(failResponse(resp, "unsupported_operation", "evidence detail is not supported by this provider", false), start, b.now)
		}
		evidence, err := evidenceTracker.GetIssueEvidence(issueID)
		if err != nil {
			return finishResponse(failResponse(resp, "provider_error", redact.apply(err.Error()), false), start, b.now)
		}
		setBoundedResult(&resp, evidence, redact)
		return finishResponse(resp, start, b.now)
	}
	issue, err := t.GetIssue(issueID)
	if err != nil {
		return finishResponse(failResponse(resp, "provider_error", redact.apply(err.Error()), false), start, b.now)
	}
	setBoundedResult(&resp, contractIssue(*issue), redact)
	return finishResponse(resp, start, b.now)
}

func splitProjectIssue(value string) (project, issue string, ok bool) {
	i := strings.LastIndex(value, "#")
	if i <= 0 || i == len(value)-1 || !strings.Contains(value[:i], "/") {
		return "", "", false
	}
	return value[:i], value[i+1:], true
}

func validateBrokerIssueID(provider, issueID string) error {
	if issueID == "" || strings.ContainsAny(issueID, "/\\?#\r\n\x00") || strings.Contains(issueID, "..") {
		return errors.New("issue_id contains path or control characters")
	}
	if provider == "github" || provider == "gitlab" {
		if _, err := strconv.Atoi(issueID); err != nil {
			return fmt.Errorf("%s issue_id must end in a numeric issue number", provider)
		}
	}
	return nil
}

type brokerCursor struct {
	Version      string `json:"v"`
	Provider     string `json:"provider"`
	ConnectionID string `json:"connection_id"`
	QueryHash    string `json:"query_hash"`
	Native       string `json:"native"`
}

func queryHash(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:])
}

func encodeCursor(c brokerCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(raw string) (brokerCursor, error) {
	var c brokerCursor
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || json.Unmarshal(b, &c) != nil || c.Version != brokercontract.Version {
		return c, errors.New("cursor is invalid")
	}
	return c, nil
}

func (b *Broker) Search(ctx context.Context, in brokercontract.SearchRequest) brokercontract.Response {
	start := b.now()
	ctx, cancel := context.WithTimeout(ctx, brokerTimeout)
	defer cancel()
	resp := baseResponse(brokercontract.OperationSearch)
	if strings.TrimSpace(in.Query) == "" {
		return finishResponse(failResponse(resp, "invalid_query", "query is required", false), start, b.now)
	}
	if in.Limit == 0 {
		in.Limit = 30
	}
	if in.Limit < 1 || in.Limit > maxBrokerSearchLimit {
		return finishResponse(failResponse(resp, "invalid_limit", "limit must be between 1 and 100", false), start, b.now)
	}
	r, err := b.resolve(in.ConnectionID, true)
	if err != nil {
		return finishResponse(failResponse(resp, selectionErrorCode(err), safeError(err, ""), false), start, b.now)
	}
	resp.Provider, resp.ConnectionID = r.connection.Provider, r.connection.ID
	redact := newRedactor(r.token.Reveal(), r.connection.UserEmail)
	var nativeCursor string
	if in.Cursor != "" {
		cursor, err := decodeCursor(in.Cursor)
		if err != nil || cursor.Provider != resp.Provider || cursor.ConnectionID != resp.ConnectionID || cursor.QueryHash != queryHash(in.Query) {
			return finishResponse(failResponse(resp, "cursor_mismatch", "cursor does not match this provider, connection, and query", false), start, b.now)
		}
		nativeCursor = cursor.Native
	}

	var issues []Issue
	var next string
	switch t := r.tracker.(type) {
	case *jira:
		fields := "key,summary,status,priority,assignee,labels,issuetype,created,updated,reporter,description"
		issues, next, _, err = t.searchIssuesPage(ctx, in.Query, in.Limit, fields, nativeCursor)
	case *gitHub:
		issues, next, err = t.brokerSearchPage(ctx, in.Query, in.Limit, nativeCursor)
	case *gitLab:
		issues, next, err = t.brokerSearchPage(ctx, in.Query, in.Limit, nativeCursor)
	case *linear:
		issues, next, err = t.brokerSearchPage(ctx, in.Query, in.Limit, nativeCursor)
	default:
		err = fmt.Errorf("broad search is not supported by provider %q", resp.Provider)
	}
	if err != nil {
		code := "provider_error"
		if strings.Contains(err.Error(), "not supported") {
			code = "unsupported_operation"
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			code = "cancelled"
		}
		return finishResponse(failResponse(resp, code, redact.apply(err.Error()), false), start, b.now)
	}
	items := make([]brokercontract.Issue, 0, len(issues))
	for _, issue := range issues {
		items = append(items, contractIssue(issue))
	}
	setBoundedResult(&resp, brokercontract.SearchResult{Items: items}, redact)
	if next != "" {
		resp.NextCursor = encodeCursor(brokerCursor{Version: brokercontract.Version, Provider: resp.Provider, ConnectionID: resp.ConnectionID, QueryHash: queryHash(in.Query), Native: next})
	}
	return finishResponse(resp, start, b.now)
}

func contractIssue(in Issue) brokercontract.Issue {
	return brokercontract.Issue{
		ID: in.ID, Title: in.Title, Status: in.Status, URL: in.URL,
		Priority: in.Priority, Severity: in.Severity, Assignee: in.Assignee,
		Labels: in.Labels, IssueType: in.IssueType, Reporter: in.Reporter,
		CreatedAt: in.CreatedAt, UpdatedAt: in.UpdatedAt, Description: in.Description,
		CustomFields: in.CustomFields,
	}
}

func normalizeOutputLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultBrokerOutputLimit, nil
	}
	if limit < 1 || limit > maxBrokerOutputLimit {
		return 0, fmt.Errorf("output_limit must be between 1 and %d", maxBrokerOutputLimit)
	}
	return limit, nil
}

func (b *Broker) Request(ctx context.Context, in brokercontract.RequestRequest) brokercontract.Response {
	start := b.now()
	ctx, cancel := context.WithTimeout(ctx, brokerTimeout)
	defer cancel()
	resp := baseResponse(brokercontract.OperationRequest)
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	resp.Effect = httpEffect(method)
	if resp.Effect == "" {
		resp.Effect = brokercontract.EffectRead
		return finishResponse(failResponse(resp, "invalid_method", "method must be GET, HEAD, OPTIONS, POST, PUT, PATCH, or DELETE", false), start, b.now)
	}
	limit, err := normalizeOutputLimit(in.OutputLimit)
	if err != nil {
		return finishResponse(failResponse(resp, "invalid_limit", err.Error(), false), start, b.now)
	}
	if len(in.Body) > maxBrokerOutputLimit {
		return finishResponse(failResponse(resp, "input_too_large", "request body exceeds 1048576 bytes", false), start, b.now)
	}
	r, err := b.resolve(in.ConnectionID, false)
	if err != nil {
		return finishResponse(failResponse(resp, selectionErrorCode(err), safeError(err, ""), false), start, b.now)
	}
	resp.Provider, resp.ConnectionID = r.connection.Provider, r.connection.ID
	redact := newRedactor(r.token.Reveal(), r.connection.UserEmail)
	target, origin, err := brokerRequestURL(r.connection, in.RelativePath, in.Query)
	if err != nil {
		return finishResponse(failResponse(resp, "unsafe_request", redact.apply(err.Error()), false), start, b.now)
	}
	if err := validateCallerHeaders(in.Headers); err != nil {
		return finishResponse(failResponse(resp, "unsafe_headers", err.Error(), false), start, b.now)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), strings.NewReader(in.Body))
	if err != nil {
		return finishResponse(failResponse(resp, "invalid_request", "could not create request", false), start, b.now)
	}
	for k, v := range in.Headers {
		req.Header.Set(k, v)
	}
	injectAuth(req, r.connection, r.token.Reveal())
	client := *b.httpClient
	client.Timeout = brokerTimeout
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if resp.Effect == brokercontract.EffectWriteNonIdempotent {
			return errors.New("redirect rejected for non-idempotent request")
		}
		if len(via) >= maxBrokerRedirects {
			return errors.New("redirect limit exceeded")
		}
		if !sameOrigin(next.URL, origin) {
			return errors.New("cross-origin redirect rejected")
		}
		injectAuth(next, r.connection, r.token.Reveal())
		return nil
	}
	providerResp, err := client.Do(req)
	if err != nil {
		if providerResp != nil && providerResp.Body != nil {
			providerResp.Body.Close()
		}
		code := "provider_error"
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			code = "cancelled"
		} else if strings.Contains(err.Error(), "redirect") {
			code = "unsafe_redirect"
		}
		return finishResponse(failResponse(resp, code, redact.apply(err.Error()), false), start, b.now)
	}
	defer providerResp.Body.Close()
	resp.StatusCode = intPtr(providerResp.StatusCode)
	body, truncated, err := readBounded(providerResp.Body, limit+redact.maxLen())
	if err != nil {
		return finishResponse(failResponse(resp, "provider_error", redact.apply(err.Error()), false), start, b.now)
	}
	truncated = truncated || len(body) > limit
	resp.Body, truncated = truncateString(redact.apply(string(body)), limit, truncated)
	resp.Truncated = truncated
	if providerResp.StatusCode < 200 || providerResp.StatusCode >= 300 {
		resp = failResponse(resp, "provider_http_error", fmt.Sprintf("provider returned HTTP %d", providerResp.StatusCode), providerResp.StatusCode == 429 || providerResp.StatusCode >= 500)
	}
	return finishResponse(resp, start, b.now)
}

func httpEffect(method string) brokercontract.Effect {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return brokercontract.EffectRead
	case http.MethodPut, http.MethodDelete:
		return brokercontract.EffectWriteIdempotent
	case http.MethodPost, http.MethodPatch:
		return brokercontract.EffectWriteNonIdempotent
	default:
		return ""
	}
}

func brokerRequestURL(c config.TrackerConnection, relativePath string, query map[string][]string) (*url.URL, *url.URL, error) {
	base := strings.TrimSpace(c.BaseURL)
	switch c.Provider {
	case "github":
		if base == "" {
			base = defaultGitHubAPI
		}
	case "linear":
		if base == "" {
			base = defaultLinearAPI
		}
	}
	origin, err := url.Parse(base)
	if err != nil || origin.Scheme == "" || origin.Host == "" || origin.User != nil || origin.Fragment != "" {
		return nil, nil, errors.New("configured tracker origin is invalid")
	}
	if origin.Scheme != "https" && origin.Scheme != "http" {
		return nil, nil, errors.New("configured tracker origin must use http or https")
	}
	if relativePath == "" || strings.HasPrefix(relativePath, "//") || strings.Contains(relativePath, "\\") || strings.ContainsAny(relativePath, "\r\n\x00") {
		return nil, nil, errors.New("relative_path is invalid")
	}
	rel, err := url.Parse(relativePath)
	if err != nil || rel.IsAbs() || rel.Host != "" || rel.User != nil || rel.Fragment != "" || rel.RawQuery != "" || rel.Opaque != "" {
		return nil, nil, errors.New("relative_path must not contain an origin, userinfo, query, or fragment")
	}
	root := &url.URL{Scheme: origin.Scheme, Host: origin.Host, Path: "/"}
	target := root.ResolveReference(rel)
	if !sameOrigin(target, origin) {
		return nil, nil, errors.New("relative_path resolved outside the configured origin")
	}
	values := target.Query()
	if len(query) > 100 {
		return nil, nil, errors.New("query exceeds 100 keys")
	}
	queryBytes := 0
	for key, all := range query {
		if strings.ContainsAny(key, "\r\n\x00") {
			return nil, nil, errors.New("query key contains control characters")
		}
		for _, value := range all {
			queryBytes += len(key) + len(value)
			if queryBytes > defaultBrokerOutputLimit {
				return nil, nil, errors.New("query exceeds 65536 bytes")
			}
			values.Add(key, value)
		}
	}
	target.RawQuery = values.Encode()
	return target, origin, nil
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func validateCallerHeaders(headers map[string]string) error {
	blocked := map[string]bool{
		"authorization": true, "proxy-authorization": true, "cookie": true,
		"set-cookie": true, "private-token": true, "x-api-key": true,
		"x-auth-token": true, "x-gitlab-token": true, "host": true,
		"forwarded": true, "x-forwarded-host": true, "x-forwarded-proto": true,
		"x-http-method-override": true, "x-method-override": true,
	}
	if len(headers) > 50 {
		return errors.New("headers exceed 50 entries")
	}
	bytes := 0
	for key, value := range headers {
		bytes += len(key) + len(value)
		if bytes > defaultBrokerOutputLimit {
			return errors.New("headers exceed 65536 bytes")
		}
		if blocked[strings.ToLower(strings.TrimSpace(key))] {
			return fmt.Errorf("caller-supplied credential header %q is forbidden", key)
		}
		if strings.ContainsAny(key+value, "\r\n\x00") {
			return fmt.Errorf("header %q contains control characters", key)
		}
	}
	return nil
}

func injectAuth(req *http.Request, c config.TrackerConnection, token string) {
	switch c.Provider {
	case "jira":
		if c.UserEmail != "" {
			req.SetBasicAuth(c.UserEmail, token)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	case "github":
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
	case "gitlab":
		req.Header.Set("PRIVATE-TOKEN", token)
	case "linear":
		req.Header.Set("Authorization", token)
	}
}

func (b *Broker) CLI(ctx context.Context, in brokercontract.CLIRequest) brokercontract.Response {
	start := b.now()
	ctx, cancel := context.WithTimeout(ctx, brokerTimeout)
	defer cancel()
	resp := baseResponse(brokercontract.OperationCLI)
	limit, err := normalizeOutputLimit(in.OutputLimit)
	if err != nil {
		return finishResponse(failResponse(resp, "invalid_limit", err.Error(), false), start, b.now)
	}
	if len(in.Stdin) > maxBrokerOutputLimit || len(in.Arguments) > 256 {
		return finishResponse(failResponse(resp, "input_too_large", "CLI input exceeds the 1048576-byte stdin or 256-argument bound", false), start, b.now)
	}
	argumentBytes := 0
	for _, arg := range in.Arguments {
		argumentBytes += len(arg)
	}
	if argumentBytes > defaultBrokerOutputLimit {
		return finishResponse(failResponse(resp, "input_too_large", "CLI arguments exceed 65536 bytes", false), start, b.now)
	}
	r, err := b.resolve(in.ConnectionID, false)
	if err != nil {
		return finishResponse(failResponse(resp, selectionErrorCode(err), safeError(err, ""), false), start, b.now)
	}
	resp.Provider, resp.ConnectionID = r.connection.Provider, r.connection.ID
	redact := newRedactor(r.token.Reveal(), r.connection.UserEmail)
	if redact.apply(in.Stdin) != in.Stdin {
		return finishResponse(failResponse(resp, "unsafe_cli", "credential literals are forbidden in stdin", false), start, b.now)
	}
	decl, err := cliDeclaration(r.connection, in.Executable, in.Arguments, r.token.Reveal())
	if err != nil {
		return finishResponse(failResponse(resp, "unsafe_cli", redact.apply(err.Error()), false), start, b.now)
	}
	resp.Effect = decl.effect
	path, err := b.lookPath(decl.executable)
	if err != nil {
		return finishResponse(failResponse(resp, "executable_unavailable", fmt.Sprintf("allowed executable %q was not found", decl.executable), false), start, b.now)
	}
	sandboxDir, err := os.MkdirTemp("", "hero-tracker-cli-*")
	if err != nil {
		return finishResponse(failResponse(resp, "execution_error", "could not create isolated CLI working directory", false), start, b.now)
	}
	defer os.RemoveAll(sandboxDir)
	decl.addEnv["GH_CONFIG_DIR"] = sandboxDir
	decl.addEnv["GLAB_CONFIG_DIR"] = sandboxDir
	decl.addEnv["GH_PAGER"] = "cat"
	decl.addEnv["GLAB_PAGER"] = "cat"
	decl.addEnv["PAGER"] = "cat"
	decl.addEnv["GH_PROMPT_DISABLED"] = "1"
	decl.addEnv["GLAB_NO_PROMPT"] = "1"
	cmd := exec.CommandContext(ctx, path, in.Arguments...)
	cmd.Dir = sandboxDir
	cmd.Stdin = strings.NewReader(in.Stdin)
	cmd.Env = childEnvironment(os.Environ(), decl.removeEnv, decl.addEnv)
	lookaheadLimit := limit + redact.maxLen()
	stdout := &limitedBuffer{limit: lookaheadLimit}
	stderr := &limitedBuffer{limit: lookaheadLimit}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if ctx.Err() != nil {
			exitCode = -1
			resp = failResponse(resp, "cancelled", ctx.Err().Error(), false)
		} else if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			resp = failResponse(resp, "execution_error", redact.apply(err.Error()), false)
		}
	}
	resp.ExitCode = intPtr(exitCode)
	resp.Stdout, resp.Truncated = truncateString(redact.apply(stdout.String()), limit, stdout.truncated || stdout.buf.Len() > limit)
	var stderrTruncated bool
	resp.Stderr, stderrTruncated = truncateString(redact.apply(stderr.String()), limit, stderr.truncated || stderr.buf.Len() > limit)
	resp.Truncated = resp.Truncated || stderrTruncated
	if err != nil && resp.Error == nil {
		resp = failResponse(resp, "command_failed", fmt.Sprintf("provider command exited with status %d", exitCode), false)
	}
	return finishResponse(resp, start, b.now)
}

type cliPolicy struct {
	executable string
	effect     brokercontract.Effect
	removeEnv  map[string]bool
	addEnv     map[string]string
}

func cliDeclaration(c config.TrackerConnection, executable string, args []string, token string) (cliPolicy, error) {
	if executable == "" || filepath.Base(executable) != executable || strings.ContainsAny(executable, `/\\`) {
		return cliPolicy{}, errors.New("executable must be a declared bare identity")
	}
	var p cliPolicy
	switch c.Provider {
	case "github":
		if executable != "gh" {
			return p, errors.New("provider allows only the gh executable")
		}
		host := providerHost(c)
		p = cliPolicy{executable: "gh", effect: cliEffect(args), removeEnv: trackerCredentialEnvSet(), addEnv: map[string]string{"GH_HOST": host}}
		if host == "github.com" {
			p.addEnv["GH_TOKEN"] = token
		} else {
			p.addEnv["GH_ENTERPRISE_TOKEN"] = token
		}
	case "gitlab":
		if executable != "glab" {
			return p, errors.New("provider allows only the glab executable")
		}
		p = cliPolicy{executable: "glab", effect: cliEffect(args), removeEnv: trackerCredentialEnvSet(), addEnv: map[string]string{"GITLAB_TOKEN": token, "GITLAB_HOST": strings.TrimRight(c.BaseURL, "/")}}
	default:
		return p, fmt.Errorf("provider %q does not declare a credential-safe CLI", c.Provider)
	}
	if err := validateCLIArgs(args, token); err != nil {
		return cliPolicy{}, err
	}
	if c.TokenEnv != "" {
		p.removeEnv[c.TokenEnv] = true
	}
	return p, nil
}

func providerHost(c config.TrackerConnection) string {
	if c.BaseURL == "" && c.Provider == "github" {
		return "github.com"
	}
	u, _ := url.Parse(c.BaseURL)
	return u.Hostname()
}

func envSet(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, key := range keys {
		m[key] = true
	}
	return m
}

func trackerCredentialEnvSet() map[string]bool {
	return envSet(
		"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN", "GH_HOST", "GH_REPO", "GH_CONFIG_DIR",
		"GITLAB_TOKEN", "GLAB_TOKEN", "GITLAB_HOST", "GL_HOST", "GLAB_REPO", "GLAB_CONFIG_DIR",
		"JIRA_TOKEN", "JIRA_API_TOKEN", "ATLASSIAN_API_TOKEN",
		"LINEAR_TOKEN", "LINEAR_API_KEY",
		"CI_JOB_TOKEN", "CI_SERVER_HOST", "CI_SERVER_FQDN", "CI_API_V4_URL", "CI_PROJECT_PATH",
	)
}

func validateCLIArgs(args []string, token string) error {
	if len(args) == 0 {
		return errors.New("arguments must name a provider command")
	}
	blockedCommands := map[string]bool{"auth": true, "config": true, "credential": true, "credentials": true, "extension": true, "alias": true, "secret": true}
	if blockedCommands[strings.ToLower(args[0])] {
		return fmt.Errorf("command %q is not brokerable", args[0])
	}
	for i, arg := range args {
		lower := strings.ToLower(arg)
		if token != "" && newRedactor(token, "").apply(arg) != arg {
			return errors.New("credential literals are forbidden in arguments")
		}
		if strings.Contains(lower, "://") || strings.HasPrefix(lower, "//") {
			return errors.New("URL and host overrides are forbidden in arguments")
		}
		if strings.ContainsAny(arg, ";|&`$<>\r\n\x00") {
			return errors.New("shell escape characters are forbidden in arguments")
		}
		if i <= 1 && (lower == "checkout" || lower == "clone" || lower == "ssh" || lower == "shell" || lower == "exec" || lower == "codespace") {
			return errors.New("shell and repository execution commands are forbidden")
		}
		if lower == "--hostname" || strings.HasPrefix(lower, "--hostname=") || lower == "--host" || strings.HasPrefix(lower, "--host=") || lower == "-h" || lower == "-h=" || lower == "--header" || strings.HasPrefix(lower, "--header=") {
			return errors.New("host and credential header overrides are forbidden")
		}
		if strings.Contains(lower, "show-token") || strings.Contains(lower, "show_token") || strings.Contains(lower, "print-token") {
			return errors.New("credential display commands are forbidden")
		}
		if lower == "--token" || strings.HasPrefix(lower, "--token=") || lower == "--password" || strings.HasPrefix(lower, "--password=") {
			return errors.New("credential arguments are forbidden")
		}
		if (lower == "--repo" || lower == "-r") && i+1 < len(args) {
			repo := args[i+1]
			if strings.Count(repo, "/") > 1 || strings.Contains(repo, "://") || strings.Contains(repo, "@") {
				return errors.New("repository host overrides are forbidden")
			}
		}
		if strings.HasPrefix(lower, "--repo=") || strings.HasPrefix(lower, "-r=") {
			_, repo, _ := strings.Cut(arg, "=")
			if strings.Count(repo, "/") > 1 || strings.Contains(repo, "://") || strings.Contains(repo, "@") {
				return errors.New("repository host overrides are forbidden")
			}
		}
		if strings.Contains(arg, "=@") || strings.HasPrefix(arg, "@") || lower == "--output" || strings.HasPrefix(lower, "--output=") {
			return errors.New("filesystem input and output arguments are forbidden")
		}
		if (lower == "--input" || strings.HasPrefix(lower, "--input=")) && lower != "--input=-" && !(lower == "--input" && i+1 < len(args) && args[i+1] == "-") {
			return errors.New("CLI input files are forbidden; use broker stdin with --input -")
		}
	}
	return nil
}

func cliEffect(args []string) brokercontract.Effect {
	joined := strings.ToLower(strings.Join(args, " "))
	if strings.Contains(joined, "--method post") || strings.Contains(joined, "--method=post") || strings.Contains(joined, "--method patch") || strings.Contains(joined, "--method=patch") || strings.Contains(joined, " create") || strings.Contains(joined, " comment") || strings.Contains(joined, " close") || strings.Contains(joined, " reopen") || strings.Contains(joined, " merge") {
		return brokercontract.EffectWriteNonIdempotent
	}
	if strings.Contains(joined, "--method put") || strings.Contains(joined, "--method=put") || strings.Contains(joined, "--method delete") || strings.Contains(joined, "--method=delete") || strings.Contains(joined, " edit") || strings.Contains(joined, " update") {
		return brokercontract.EffectWriteIdempotent
	}
	if args[0] == "api" {
		for _, arg := range args {
			if arg == "-f" || strings.HasPrefix(arg, "-f=") || arg == "--field" || strings.HasPrefix(arg, "--field=") || arg == "--raw-field" || strings.HasPrefix(arg, "--raw-field=") || arg == "--input" || strings.HasPrefix(arg, "--input=") {
				return brokercontract.EffectWriteNonIdempotent
			}
		}
		return brokercontract.EffectRead
	}
	readCommands := map[string]bool{"list": true, "view": true, "status": true, "search": true, "browse": true, "diff": true, "checks": true}
	if readCommands[args[0]] || (len(args) > 1 && readCommands[args[1]]) {
		return brokercontract.EffectRead
	}
	// An unknown provider subcommand is conservatively a write. This avoids
	// presenting a newly-added provider command as harmless before Hero has a
	// declaration for it.
	return brokercontract.EffectWriteNonIdempotent
}

func childEnvironment(parent []string, remove map[string]bool, add map[string]string) []string {
	out := make([]string, 0, len(parent)+len(add))
	for _, entry := range parent {
		key, _, _ := strings.Cut(entry, "=")
		if !remove[key] {
			out = append(out, entry)
		}
	}
	for key, value := range add {
		out = append(out, key+"="+value)
	}
	return out
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.buf.Write(p)
	}
	if n > remaining {
		b.truncated = true
	}
	return n, nil
}

func (b *limitedBuffer) String() string { return strings.ToValidUTF8(b.buf.String(), "�") }

func readBounded(r io.Reader, limit int) ([]byte, bool, error) {
	b, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, false, err
	}
	truncated := len(b) > limit
	if truncated {
		b = b[:limit]
	}
	if !utf8.Valid(b) {
		b = []byte(strings.ToValidUTF8(string(b), "�"))
	}
	return b, truncated, nil
}

type redactor struct{ replacements []string }

func newRedactor(token, userEmail string) redactor {
	if token == "" {
		return redactor{}
	}
	jsonToken, _ := json.Marshal(token)
	escapedJSON := strings.Trim(string(jsonToken), `"`)
	values := []string{token, escapedJSON, url.QueryEscape(token), url.PathEscape(token), base64.StdEncoding.EncodeToString([]byte(token)), "Bearer " + token}
	if userEmail != "" {
		values = append(values, base64.StdEncoding.EncodeToString([]byte(userEmail+":"+token)))
	}
	return redactor{replacements: values}
}

func (r redactor) apply(value string) string {
	for _, secret := range r.replacements {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	return value
}

func (r redactor) maxLen() int {
	max := 0
	for _, value := range r.replacements {
		if len(value) > max {
			max = len(value)
		}
	}
	return max
}

func safeError(err error, token string) string {
	if err == nil {
		return ""
	}
	return newRedactor(token, "").apply(err.Error())
}

func intPtr(v int) *int { return &v }

func setBoundedResult(resp *brokercontract.Response, value any, redact redactor) {
	b, err := json.Marshal(value)
	if err != nil {
		resp.Error = &brokercontract.Error{Code: "encoding_error", Message: "could not encode tracker result"}
		return
	}
	redacted := redact.apply(string(b))
	if len(redacted) <= maxBrokerOutputLimit {
		resp.Result = json.RawMessage(redacted)
		return
	}
	preview, _ := truncateString(redacted, maxBrokerOutputLimit/2, true)
	resp.Result, _ = json.Marshal(map[string]string{"truncated_json_preview": preview})
	resp.Truncated = true
}

func truncateString(value string, limit int, already bool) (string, bool) {
	if len(value) <= limit {
		return value, already
	}
	value = value[:limit]
	for !utf8.ValidString(value) && len(value) > 0 {
		value = value[:len(value)-1]
	}
	return value, true
}
