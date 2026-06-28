package mocktracker

import (
	"path"
	"strings"
	"sync"
)

// rateLimiter holds the admin-configured 429 injection rules. Each rule
// answers the next `count` matching requests with 429 + Retry-After,
// then expires. This drives the doWithRetry path in
// internal/tracker/fielderror.go.
type rateLimiter struct {
	mu    sync.Mutex
	rules []*injectRule
}

type injectRule struct {
	PathGlob          string
	RetryAfterSeconds int
	Remaining         int
}

func newRateLimiter() *rateLimiter { return &rateLimiter{} }

// inject registers a new 429 rule.
func (rl *rateLimiter) inject(pathGlob string, retryAfter, count int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if count <= 0 {
		count = 1
	}
	rl.rules = append(rl.rules, &injectRule{
		PathGlob:          pathGlob,
		RetryAfterSeconds: retryAfter,
		Remaining:         count,
	})
}

// throttle reports whether reqPath should be answered with a 429 and, if
// so, the Retry-After seconds to advertise. It consumes one unit of the
// first matching rule's budget.
func (rl *rateLimiter) throttle(reqPath string) (retryAfter int, limited bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for _, r := range rl.rules {
		if r.Remaining <= 0 {
			continue
		}
		if matchGlob(r.PathGlob, reqPath) {
			r.Remaining--
			return r.RetryAfterSeconds, true
		}
	}
	return 0, false
}

// reset clears all rules (used by /__admin/reset and tests).
func (rl *rateLimiter) reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rules = nil
}

// matchGlob does shell-style glob matching against the request path,
// honoring `*` across path separators (path.Match treats `*` as not
// matching `/`, so we also try a separator-collapsed comparison).
func matchGlob(glob, p string) bool {
	if ok, _ := path.Match(glob, p); ok {
		return true
	}
	// Fall back to a manual wildcard match where `*` spans `/`.
	return wildcardMatch(glob, p)
}

// wildcardMatch matches glob against s where `*` matches any run of
// characters (including `/`). No other metacharacters are supported.
func wildcardMatch(glob, s string) bool {
	parts := strings.Split(glob, "*")
	if len(parts) == 1 {
		return glob == s
	}
	// Anchor the first and last segments; greedily consume the middle.
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	last := parts[len(parts)-1]
	for _, mid := range parts[1 : len(parts)-1] {
		idx := strings.Index(s, mid)
		if idx < 0 {
			return false
		}
		s = s[idx+len(mid):]
	}
	return strings.HasSuffix(s, last)
}
