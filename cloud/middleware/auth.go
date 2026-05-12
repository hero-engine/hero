// Package middleware provides HTTP middleware for Hero Cloud.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/hero-engine/hero/cloud/auth"
)

type contextKey string

const claimsKey contextKey = "claims"

// Auth returns middleware that validates JWT bearer tokens.
// Requests without a valid token receive a 401 response.
func Auth(issuer *auth.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			claims, err := issuer.Validate(token)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves the JWT claims from the request context.
// Returns nil if no claims are present (unauthenticated request).
func GetClaims(ctx context.Context) *auth.Claims {
	claims, _ := ctx.Value(claimsKey).(*auth.Claims)
	return claims
}

// RequireRole returns middleware that checks the user has one of the specified roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			for _, role := range claims.Roles {
				if roleSet[role] {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
		})
	}
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}
