// Package github provides GitHub App integration for Hero Cloud.
//
// A GitHub App authenticates using a private key to generate JWTs,
// then exchanges them for installation access tokens scoped to
// specific repositories. This enables Hero Cloud to create check
// runs, post PR comments, and read repository contents.
package github

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// AppConfig holds the GitHub App credentials.
type AppConfig struct {
	AppID      int64
	PrivateKey *rsa.PrivateKey
	WebhookSecret string
}

// ParsePrivateKey parses a PEM-encoded RSA private key.
func ParsePrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private key")
	}

	// Try PKCS1 first, then PKCS8
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err2 != nil {
		return nil, fmt.Errorf("parsing private key: PKCS1: %v, PKCS8: %v", err, err2)
	}

	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return rsaKey, nil
}

// App provides GitHub App API operations.
type App struct {
	config *AppConfig
	client *http.Client

	// Installation token cache
	mu     sync.Mutex
	tokens map[int64]*cachedToken
}

type cachedToken struct {
	Token     string
	ExpiresAt time.Time
}

// NewApp creates a GitHub App client.
func NewApp(config *AppConfig) *App {
	return &App{
		config: config,
		client: &http.Client{Timeout: 15 * time.Second},
		tokens: make(map[int64]*cachedToken),
	}
}

// Config returns the app configuration.
func (a *App) Config() *AppConfig {
	return a.config
}

// InstallationToken returns an access token for a GitHub App installation,
// caching tokens until they expire.
func (a *App) InstallationToken(installationID int64) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check cache
	if cached, ok := a.tokens[installationID]; ok {
		if time.Now().Before(cached.ExpiresAt.Add(-time.Minute)) {
			return cached.Token, nil
		}
	}

	// Generate app JWT
	jwt, err := a.generateJWT()
	if err != nil {
		return "", fmt.Errorf("generating app JWT: %w", err)
	}

	// Exchange for installation token
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	a.tokens[installationID] = &cachedToken{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
	}

	return result.Token, nil
}

// generateJWT creates a short-lived JWT signed with the app's private key.
// This JWT is used to authenticate as the GitHub App itself.
func (a *App) generateJWT() (string, error) {
	return GenerateAppJWT(a.config.AppID, a.config.PrivateKey)
}
