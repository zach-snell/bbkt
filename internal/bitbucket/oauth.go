package bitbucket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	authURL       = "https://bitbucket.org/site/oauth2/authorize"
	tokenEndpoint = "https://bitbucket.org/site/oauth2/access_token" //nolint:gosec // Not a hardcoded credential, just an endpoint url
)

// RefreshOAuth uses the refresh token to get a new access token.
// Updates the Credentials in place and persists to disk.
func RefreshOAuth(creds *Credentials) error {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {creds.RefreshToken},
	}

	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating refresh request: %w", err)
	}
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scopes       string `json:"scopes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing refresh response: %w", err)
	}

	creds.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		creds.RefreshToken = result.RefreshToken
	}
	creds.ExpiresIn = result.ExpiresIn
	creds.Scopes = result.Scopes
	creds.CreatedAt = time.Now()

	return SaveProfile(creds)
}

// OAuthLogin performs the Authorization Code Grant flow with a localhost callback.
// Opens the user's browser, waits for the callback, exchanges the code, and stores credentials.
func OAuthLogin(clientID, clientSecret, profileName string) error {
	// Generate state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("generating state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("finding free port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	// Build authorize URL
	params := url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"state":         {state},
	}
	authorizeURL := authURL + "?" + params.Encode()

	// Channel for the auth code
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Setup callback server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch - possible CSRF attack")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("OAuth error: %s - %s", errParam, errDesc)
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>%s: %s</p><p>You can close this window.</p></body></html>", errParam, errDesc)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, "<html><body><h2>Authenticated!</h2><p>You can close this window and return to your terminal.</p></body></html>")
		codeCh <- code
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}
	go func() {
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Open browser
	fmt.Printf("\nOpening browser for Bitbucket authentication...\n")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", authorizeURL)
	fmt.Printf("Callback URL: %s\n", callbackURL)
	fmt.Printf("Waiting for authentication...\n\n")
	openBrowser(authorizeURL)

	// Wait for code or error (timeout after 5 minutes)
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		_ = srv.Shutdown(context.Background())
		return err
	case <-time.After(5 * time.Minute):
		_ = srv.Shutdown(context.Background())
		return fmt.Errorf("authentication timed out after 5 minutes")
	}

	_ = srv.Shutdown(context.Background())

	// Exchange code for tokens
	fmt.Println("Exchanging code for tokens...")

	formData := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}

	tokenReq, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("creating token request: %w", err)
	}
	tokenReq.SetBasicAuth(clientID, clientSecret)
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tokenResp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("exchanging code: %w", err)
	}
	defer tokenResp.Body.Close()

	body, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return fmt.Errorf("reading token response: %w", err)
	}

	if tokenResp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed (%d): %s", tokenResp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scopes       string `json:"scopes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}

	creds := &Credentials{
		ProfileName:          profileName,
		AuthType:             AuthTypeOAuth,
		CreatedAt:            time.Now(),
		AccessToken:          result.AccessToken,
		RefreshToken:         result.RefreshToken,
		TokenType:            result.TokenType,
		ExpiresIn:            result.ExpiresIn,
		Scopes:               result.Scopes,
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		AccessibleWorkspaces: FetchAccessibleWorkspaces(NewClient("", "", result.AccessToken)),
	}

	if err := SaveProfile(creds); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	path, _ := CredentialsPath()
	fmt.Printf("\nAuthentication successful!\n")
	fmt.Printf("Scopes: %s\n", result.Scopes)
	fmt.Printf("Credentials saved to: %s\n", path)
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}
