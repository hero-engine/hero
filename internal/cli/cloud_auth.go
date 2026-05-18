package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Hero Cloud",
	Long: `Opens a browser to authenticate with Hero Cloud via GitHub OAuth.

After authenticating, a token is stored locally at ~/.hero/credentials.json
for use with cloud commands (sync, status --remote, etc.).

Set HERO_CLOUD_URL to point to a self-hosted instance.`,
	RunE: runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke Hero Cloud credentials",
	Long:  `Removes locally stored credentials and revokes the refresh token on the server.`,
	RunE:  runLogout,
}

// Credentials stored at ~/.hero/credentials.json
type cloudCredentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	CloudURL     string `json:"cloud_url"`
	ExpiresAt    string `json:"expires_at"`
}

func runLogin(cmd *cobra.Command, args []string) error {
	cloudURL := cloudBaseURL()

	// Request the GitHub OAuth URL from the server
	req, err := http.NewRequest("GET", cloudURL+"/api/v1/auth/github", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to Hero Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Hero Cloud returned %d — is the server running at %s?", resp.StatusCode, cloudURL)
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	// Open browser
	fmt.Println("Opening browser for GitHub authentication...")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", result.URL)

	_ = openBrowser(result.URL)

	// Start a local callback server to receive the token
	tokenCh := make(chan *cloudCredentials, 1)
	errCh := make(chan error, 1)

	server := &http.Server{Addr: ":19876"}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The cloud server redirects here after OAuth with tokens as query params
		accessToken := r.URL.Query().Get("access_token")
		refreshToken := r.URL.Query().Get("refresh_token")

		if accessToken == "" {
			// Might be the initial redirect — show waiting page
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h2>Authenticating with Hero Cloud...</h2><p>You can close this window.</p></body></html>`)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h2>Authenticated!</h2><p>You can close this window and return to the terminal.</p></body></html>`)

		tokenCh <- &cloudCredentials{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			CloudURL:     cloudURL,
			ExpiresAt:    time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	fmt.Println("Waiting for authentication...")

	select {
	case creds := <-tokenCh:
		server.Close()
		if err := saveCredentials(creds); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}
		fmt.Println("Authenticated successfully!")
		fmt.Printf("Credentials saved to %s\n", credentialsPath())
		return nil

	case err := <-errCh:
		return fmt.Errorf("callback server failed: %w", err)

	case <-time.After(5 * time.Minute):
		server.Close()
		return fmt.Errorf("authentication timed out after 5 minutes")
	}
}

func runLogout(cmd *cobra.Command, args []string) error {
	path := credentialsPath()

	creds, err := loadCredentials()
	if err != nil || creds == nil {
		fmt.Println("Not currently logged in.")
		return nil
	}

	// Try to revoke on server
	if creds.RefreshToken != "" && creds.CloudURL != "" {
		body := strings.NewReader(fmt.Sprintf(`{"refresh_token":"%s"}`, creds.RefreshToken))
		req, _ := http.NewRequest("POST", creds.CloudURL+"/api/v1/auth/logout", body)
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
			if creds.AccessToken != "" {
				req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
			}
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}
	}

	// Remove local credentials
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing credentials: %w", err)
	}

	fmt.Println("Logged out. Credentials removed.")
	return nil
}

func cloudBaseURL() string {
	if url := os.Getenv("HERO_CLOUD_URL"); url != "" {
		return strings.TrimRight(url, "/")
	}
	return "https://cloud.herospec.dev"
}

func credentialsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hero", "credentials.json")
}

func saveCredentials(creds *cloudCredentials) error {
	path := credentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func loadCredentials() (*cloudCredentials, error) {
	data, err := os.ReadFile(credentialsPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var creds cloudCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// LoadCloudToken returns the current access token, refreshing if needed.
// Returns empty string if not logged in.
func LoadCloudToken() string {
	creds, err := loadCredentials()
	if err != nil || creds == nil {
		return ""
	}

	if creds.ExpiresAt != "" {
		exp, err := time.Parse(time.RFC3339, creds.ExpiresAt)
		if err == nil && time.Now().After(exp) {
			if refreshed := tryRefresh(creds); refreshed != nil {
				return refreshed.AccessToken
			}
			return ""
		}
	}

	return creds.AccessToken
}

func tryRefresh(creds *cloudCredentials) *cloudCredentials {
	if creds.RefreshToken == "" || creds.CloudURL == "" {
		return nil
	}

	body := strings.NewReader(fmt.Sprintf(`{"refresh_token":"%s"}`, creds.RefreshToken))
	resp, err := http.Post(creds.CloudURL+"/api/v1/auth/refresh", "application/json", body)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokens); err != nil {
		return nil
	}

	newCreds := &cloudCredentials{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		CloudURL:     creds.CloudURL,
		ExpiresAt:    time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).Format(time.RFC3339),
	}
	_ = saveCredentials(newCreds)
	return newCreds
}


