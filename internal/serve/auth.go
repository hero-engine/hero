package serve

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// AuthConfig defines the auth mode for the team server.
type AuthConfig struct {
	Mode      string `json:"mode"` // none, token, password, oauth
	JWTSecret string `json:"-"`    // loaded from env or config
}

// JWTClaims is the payload of a Hero JWT.
type JWTClaims struct {
	Sub   string `json:"sub"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

// IssueJWT creates a signed JWT for a user.
func IssueJWT(user *User, secret string, expiry time.Duration) (string, error) {
	if secret == "" {
		secret = "hero-default-secret"
	}
	if expiry == 0 {
		expiry = 30 * 24 * time.Hour // 30 days
	}

	header := base64url([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claims := JWTClaims{
		Sub:   user.ID,
		Name:  user.DisplayName,
		Email: user.Email,
		Role:  user.Role,
		Iat:   time.Now().Unix(),
		Exp:   time.Now().Add(expiry).Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64url(claimsJSON)

	signature := signJWT(header+"."+payload, secret)
	return header + "." + payload + "." + signature, nil
}

// ValidateJWT verifies and decodes a JWT.
func ValidateJWT(token, secret string) (*JWTClaims, error) {
	if secret == "" {
		secret = "hero-default-secret"
	}

	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	expected := signJWT(parts[0]+"."+parts[1], secret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid signature")
	}

	claimsJSON, err := base64urlDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// TeamAuthMiddleware creates middleware that supports multiple auth methods:
// Bearer <jwt>, Bearer <token>, or Basic username:password.
func TeamAuthMiddleware(jq *JobQueue, staticToken, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				jsonError(w, "authorization required", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 {
				jsonError(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			scheme := strings.ToLower(parts[0])
			credential := parts[1]

			switch scheme {
			case "bearer":
				// Try JWT first
				if claims, err := ValidateJWT(credential, jwtSecret); err == nil {
					r.Header.Set("X-Hero-User", claims.Sub)
					r.Header.Set("X-Hero-User-Name", claims.Name)
					r.Header.Set("X-Hero-User-Role", claims.Role)
					next.ServeHTTP(w, r)
					return
				}
				// Fall back to static token
				if staticToken != "" && credential == staticToken {
					next.ServeHTTP(w, r)
					return
				}
				jsonError(w, "invalid token", http.StatusForbidden)

			case "basic":
				decoded, err := base64.StdEncoding.DecodeString(credential)
				if err != nil {
					jsonError(w, "invalid basic auth encoding", http.StatusUnauthorized)
					return
				}
				colonIdx := strings.IndexByte(string(decoded), ':')
				if colonIdx < 0 {
					jsonError(w, "invalid basic auth format", http.StatusUnauthorized)
					return
				}
				username := string(decoded[:colonIdx])
				password := string(decoded[colonIdx+1:])

				user, err := jq.AuthenticateUser(username, password)
				if err != nil {
					jsonError(w, "invalid username or password", http.StatusForbidden)
					return
				}

				r.Header.Set("X-Hero-User", user.ID)
				r.Header.Set("X-Hero-User-Name", user.DisplayName)
				r.Header.Set("X-Hero-User-Role", user.Role)
				next.ServeHTTP(w, r)

			default:
				jsonError(w, "unsupported auth scheme", http.StatusUnauthorized)
			}
		})
	}
}

// TokenAuthMiddleware creates middleware that validates Bearer tokens.
// If token is empty, all requests are allowed (no-auth mode).
func TokenAuthMiddleware(token string) func(http.Handler) http.Handler {
	if token == "" {
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				jsonError(w, "authorization required", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				jsonError(w, "invalid authorization format (use Bearer <token>)", http.StatusUnauthorized)
				return
			}

			if parts[1] != token {
				jsonError(w, "invalid token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ResolveAuthToken gets the team auth token from config or environment.
func ResolveAuthToken(configToken string) string {
	if configToken != "" {
		return configToken
	}
	return os.Getenv("HERO_AUTH_TOKEN")
}

func signJWT(input, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return base64url(mac.Sum(nil))
}

func base64url(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64urlDecode(s string) ([]byte, error) {
	// Add padding back
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
