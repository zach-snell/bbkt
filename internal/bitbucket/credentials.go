package bitbucket

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AuthType distinguishes between credential storage methods.
type AuthType string

const (
	AuthTypeAPIToken AuthType = "api_token"
	AuthTypeOAuth    AuthType = "oauth"
)

// Credentials holds persisted authentication data.
// Supports both API Token (Basic Auth) and OAuth 2.0 (Bearer Auth).
type Credentials struct {
	AuthType  AuthType  `json:"auth_type"`
	CreatedAt time.Time `json:"created_at"`

	// API Token fields (auth_type=api_token)
	Email    string `json:"email,omitempty"`
	APIToken string `json:"api_token,omitempty"`

	// OAuth fields (auth_type=oauth)
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Scopes       string `json:"scopes,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

// IsOAuth returns true if these credentials use OAuth.
func (c *Credentials) IsOAuth() bool {
	return c.AuthType == AuthTypeOAuth
}

// IsAPIToken returns true if these credentials use an API token.
func (c *Credentials) IsAPIToken() bool {
	return c.AuthType == AuthTypeAPIToken
}

// IsExpired returns true if OAuth access token is expired (with 5 min buffer).
func (c *Credentials) IsExpired() bool {
	if !c.IsOAuth() {
		return false
	}
	expiry := c.CreatedAt.Add(time.Duration(c.ExpiresIn) * time.Second)
	return time.Now().After(expiry.Add(-5 * time.Minute))
}

// CredentialsPath returns the path to the credentials file.
func CredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".config", "bitbucket-mcp", "credentials.json"), nil
}

// SaveCredentials persists credentials to disk with secure permissions.
func SaveCredentials(creds *Credentials) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing credentials file: %w", err)
	}

	return nil
}

// LoadCredentials reads persisted credentials from disk.
func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading credentials file: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials file: %w", err)
	}

	return &creds, nil
}

// RemoveCredentials deletes the stored credentials file.
func RemoveCredentials() error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// APITokenLogin prompts the user for email + API token and stores them.
func APITokenLogin() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Bitbucket API Token Authentication")
	fmt.Println("===================================")
	fmt.Println()
	fmt.Println("Create an API token at:")
	fmt.Println("  https://bitbucket.org/account/settings/api-tokens/")
	fmt.Println()
	fmt.Println("Required scopes: Repositories (Read/Write), Pull Requests (Read/Write),")
	fmt.Println("                 Pipelines (Read/Write), Account (Read)")
	fmt.Println()

	fmt.Print("Atlassian email: ")
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading email: %w", err)
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	fmt.Print("API token: ")
	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("API token is required")
	}

	// Verify credentials by hitting the user API
	fmt.Println("\nVerifying credentials...")
	client := NewClient(email, token, "")
	userData, err := client.Get("/user")
	if err != nil {
		return fmt.Errorf("credential verification failed: %w\n\nCheck that your email and API token are correct", err)
	}

	var user struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
	}
	if jsonErr := json.Unmarshal(userData, &user); jsonErr == nil {
		name := user.DisplayName
		if name == "" {
			name = user.Nickname
		}
		fmt.Printf("Authenticated as: %s\n", name)
	}

	creds := &Credentials{
		AuthType:  AuthTypeAPIToken,
		CreatedAt: time.Now(),
		Email:     email,
		APIToken:  token,
	}

	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	path, _ := CredentialsPath()
	fmt.Printf("\nCredentials saved to: %s\n", path)
	fmt.Println("You can now use the Bitbucket MCP server.")
	return nil
}
