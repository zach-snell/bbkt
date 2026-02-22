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

// ProfileStore holds multiple authentication profiles
type ProfileStore struct {
	ActiveProfile string                  `json:"active_profile"`
	Profiles      map[string]*Credentials `json:"profiles"`
}

// Credentials holds persisted authentication data.
// Supports both API Token (Basic Auth) and OAuth 2.0 (Bearer Auth).
type Credentials struct {
	ProfileName string    `json:"-"`
	AuthType    AuthType  `json:"auth_type"`
	CreatedAt   time.Time `json:"created_at"`

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

	// Derived cache data
	AccessibleWorkspaces []string `json:"accessible_workspaces,omitempty"`
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
	return filepath.Join(home, ".config", "bbkt", "credentials.json"), nil
}

// SaveProfileStore persists the entire ProfileStore to disk.
func SaveProfileStore(store *ProfileStore) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling profile store: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing credentials file: %w", err)
	}

	return nil
}

// SaveProfile saves a single credential under its ProfileName.
// If it is the first profile being saved, it is automatically marked as active.
func SaveProfile(creds *Credentials) error {
	if creds.ProfileName == "" {
		creds.ProfileName = "default"
	}

	store, err := LoadProfileStore()
	if err != nil {
		// If the file doesn't exist, we start a fresh store
		store = &ProfileStore{
			Profiles: make(map[string]*Credentials),
		}
	}

	if store.Profiles == nil {
		store.Profiles = make(map[string]*Credentials)
	}

	store.Profiles[creds.ProfileName] = creds
	if store.ActiveProfile == "" {
		store.ActiveProfile = creds.ProfileName
	}

	return SaveProfileStore(store)
}

// LoadProfileStore reads the persisted profile store from disk.
// It automatically migrates older single-credential files to the new ProfileStore format.
func LoadProfileStore() (*ProfileStore, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading credentials file: %w", err)
	}

	// Check if this is the old scalar Credentials format or the new ProfileStore format
	var check map[string]interface{}
	if err := json.Unmarshal(data, &check); err != nil {
		return nil, fmt.Errorf("parsing credentials file: %w", err)
	}

	if _, ok := check["profiles"]; !ok {
		// Old format migration
		var oldCreds Credentials
		if err := json.Unmarshal(data, &oldCreds); err != nil {
			return nil, fmt.Errorf("parsing legacy credentials: %w", err)
		}
		oldCreds.ProfileName = "default"

		store := &ProfileStore{
			ActiveProfile: "default",
			Profiles: map[string]*Credentials{
				"default": &oldCreds,
			},
		}

		// Save the migrated format silently back to disk
		_ = SaveProfileStore(store)
		return store, nil
	}

	// New format
	var store ProfileStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parsing profile store: %w", err)
	}

	if store.Profiles == nil {
		store.Profiles = make(map[string]*Credentials)
	}

	for name, creds := range store.Profiles {
		creds.ProfileName = name
	}

	return &store, nil
}

// LoadCredentials gets the active credential profile based on context.
// Priority:
// 1. BBKT_PROFILE environment variable (or --profile CLI flag equivalent)
// 2. Exact match of local `git config user.email` to a profile's email
// 3. The configured 'ActiveProfile' in credentials.json
func LoadCredentials() (*Credentials, error) {
	store, err := LoadProfileStore()
	if err != nil {
		return nil, err
	}

	// 1. Explicit Override
	if override := os.Getenv("BBKT_PROFILE"); override != "" {
		if creds, ok := store.Profiles[override]; ok {
			return creds, nil
		}
		return nil, fmt.Errorf("override profile '%s' not found in store", override)
	}

	// 2. Magic Context Inference
	if ws, _, err := GetLocalRepoInfo(); err == nil && ws != "" {
		for _, creds := range store.Profiles {
			for _, accessible := range creds.AccessibleWorkspaces {
				if strings.EqualFold(accessible, ws) {
					return creds, nil
				}
			}
		}
	}

	// 3. Fallback to Active Profile
	creds, ok := store.Profiles[store.ActiveProfile]
	if !ok {
		if len(store.Profiles) > 0 {
			// Panic recovery: just use whatever we have
			for _, first := range store.Profiles {
				return first, nil
			}
		}
		return nil, fmt.Errorf("active profile '%s' not found in store, and no other profiles found", store.ActiveProfile)
	}

	return creds, nil
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

// APITokenLogin prompts the user for email + API Token and stores them.
func APITokenLogin(profileName string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Atlassian API Token Authentication (Basic Auth)")
	fmt.Println("=============================================")
	fmt.Println()
	fmt.Println("Create an API Token at:")
	fmt.Println("  https://id.atlassian.com/manage-profile/security/api-tokens")
	fmt.Println()
	fmt.Println("Note: This replaces the deprecated Bitbucket App Passwords system.")
	fmt.Println()
	fmt.Println("Recommended Scopes:")
	fmt.Println("  read:workspace, read:account, read:user, read:repository:bitbucket,")
	fmt.Println("  write:repository:bitbucket, read:pullrequest:bitbucket,")
	fmt.Println("  write:pullrequest:bitbucket, read:pipeline:bitbucket,")
	fmt.Println("  write:pipeline:bitbucket")
	fmt.Println()
	fmt.Println("Tools are dynamically disabled if your token omits specific scopes.")
	fmt.Println("To explicitly deny the MCP server access to tools despite having full scopes, use:")
	fmt.Println("  export BITBUCKET_DISABLED_TOOLS=\"delete_repository,delete_branch...\"")
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

	fmt.Print("API Token: ")
	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading API Token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("API Token is required")
	}

	// Verify credentials by hitting the user API
	fmt.Println("\nVerifying credentials...")
	client := NewClient(email, token, "")
	userData, scopesStr, err := client.GetWithScopes("/user")
	if err != nil {
		if strings.Contains(err.Error(), "403 Forbidden") {
			fmt.Println("\nToken verified successfully (403 Forbidden on /user means token is valid but lacks 'account' scopes).")
		} else {
			return fmt.Errorf("credential verification failed: %w\n\nCheck that your email and API Token are correct", err)
		}
	}

	if userData != nil {
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
	}

	creds := &Credentials{
		ProfileName:          profileName,
		AuthType:             AuthTypeAPIToken,
		CreatedAt:            time.Now(),
		Email:                email,
		APIToken:             token,
		Scopes:               scopesStr,
		AccessibleWorkspaces: FetchAccessibleWorkspaces(client),
	}

	if err := SaveProfile(creds); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	path, _ := CredentialsPath()
	fmt.Printf("\nCredentials saved to: %s\n", path)
	fmt.Println("You can now use the Bitbucket MCP server.")
	return nil
}

// FetchAccessibleWorkspaces retrieves all workspace slugs the client can access.
func FetchAccessibleWorkspaces(client *Client) []string {
	var slugs []string
	res, err := client.ListWorkspaces(ListWorkspacesArgs{Pagelen: 100})
	if err == nil && res != nil {
		for _, w := range res.Values {
			slugs = append(slugs, w.Slug)
		}
	}
	return slugs
}
