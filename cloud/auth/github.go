// Package auth provides JWT token generation, validation, and OAuth flows.
//
// github.go implements the GitHub OAuth2 authorization code flow.
package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GitHubConfig holds GitHub OAuth application credentials.
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// GitHubUser represents the user info returned from GitHub's API.
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubAuthorizeURL returns the URL to redirect the user to for GitHub login.
func (c *GitHubConfig) AuthorizeURL(state string) string {
	v := url.Values{
		"client_id":    {c.ClientID},
		"redirect_uri": {c.RedirectURL},
		"scope":        {"user:email read:org"},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + v.Encode()
}

// ExchangeCode exchanges an authorization code for an access token.
func (c *GitHubConfig) ExchangeCode(code string) (string, error) {
	v := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"code":          {code},
		"redirect_uri":  {c.RedirectURL},
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(v.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchanging code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("github oauth error: %s — %s", result.Error, result.ErrorDesc)
	}

	return result.AccessToken, nil
}

// FetchUser retrieves the authenticated user's profile from GitHub.
func FetchGitHubUser(accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decoding user: %w", err)
	}

	// If email is private, fetch from emails endpoint
	if user.Email == "" {
		email, err := fetchPrimaryEmail(accessToken)
		if err == nil {
			user.Email = email
		}
	}

	return &user, nil
}

func fetchPrimaryEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no verified email found")
}
