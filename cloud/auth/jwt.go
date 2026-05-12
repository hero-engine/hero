// Package auth provides JWT token generation, validation, and OAuth flows.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTokenExpired  = errors.New("token expired")
	ErrTokenInvalid  = errors.New("token invalid")
	ErrTokenMalformed = errors.New("token malformed")
)

// Claims represents the payload of a Hero Cloud JWT.
type Claims struct {
	UserID string   `json:"sub"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	OrgID  string   `json:"org_id,omitempty"`
	Roles  []string `json:"roles,omitempty"`
	Exp    int64    `json:"exp"`
	Iat    int64    `json:"iat"`
}

// TokenPair holds an access token and refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Issuer creates and validates JWT tokens.
type Issuer struct {
	secret         []byte
	accessTTL      time.Duration
	refreshTTL     time.Duration
}

// NewIssuer creates a token issuer with the given HMAC secret.
func NewIssuer(secret string, accessTTL, refreshTTL time.Duration) *Issuer {
	return &Issuer{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Issue creates a new access token for the given claims.
func (iss *Issuer) Issue(claims Claims) (string, error) {
	now := time.Now()
	claims.Iat = now.Unix()
	if claims.Exp == 0 {
		claims.Exp = now.Add(iss.accessTTL).Unix()
	}
	return iss.encode(claims)
}

// Validate parses and validates a token, returning the claims.
func (iss *Issuer) Validate(token string) (*Claims, error) {
	claims, err := iss.decode(token)
	if err != nil {
		return nil, err
	}

	if time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpired
	}

	return claims, nil
}

// GenerateRefreshToken creates a cryptographically random refresh token.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

// HashToken returns a SHA-256 hash of a token for safe storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// encode creates a simple JWT (header.payload.signature) using HMAC-SHA256.
func (iss *Issuer) encode(claims Claims) (string, error) {
	header := base64url([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshaling claims: %w", err)
	}
	payloadEnc := base64url(payload)

	sigInput := header + "." + payloadEnc
	sig := iss.sign([]byte(sigInput))

	return sigInput + "." + base64url(sig), nil
}

// decode parses a JWT and verifies the signature.
func (iss *Issuer) decode(token string) (*Claims, error) {
	parts := split(token, '.', 3)
	if len(parts) != 3 {
		return nil, ErrTokenMalformed
	}

	// Verify signature
	sigInput := parts[0] + "." + parts[1]
	sig, err := base64urldecode(parts[2])
	if err != nil {
		return nil, ErrTokenMalformed
	}

	expected := iss.sign([]byte(sigInput))
	if !hmac.Equal(sig, expected) {
		return nil, ErrTokenInvalid
	}

	// Decode payload
	payload, err := base64urldecode(parts[1])
	if err != nil {
		return nil, ErrTokenMalformed
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrTokenMalformed
	}

	return &claims, nil
}

func (iss *Issuer) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, iss.secret)
	mac.Write(data)
	return mac.Sum(nil)
}

func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64urldecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// split splits s into at most n parts on sep.
func split(s string, sep byte, n int) []string {
	var parts []string
	for i := 0; i < n-1; i++ {
		idx := -1
		for j := 0; j < len(s); j++ {
			if s[j] == sep {
				idx = j
				break
			}
		}
		if idx < 0 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	parts = append(parts, s)
	return parts
}
