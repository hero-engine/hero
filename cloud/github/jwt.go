package github

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// GenerateAppJWT creates a RS256-signed JWT for GitHub App authentication.
// GitHub Apps use this to obtain installation access tokens.
func GenerateAppJWT(appID int64, key *rsa.PrivateKey) (string, error) {
	now := time.Now()

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"iss": appID,
		"iat": now.Add(-60 * time.Second).Unix(), // 60s clock drift allowance
		"exp": now.Add(10 * time.Minute).Unix(),  // GitHub max: 10 min
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := headerB64 + "." + payloadB64

	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}
