package bitbucket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

const baseURL = "https://api.bitbucket.org/2.0"

// Client is the Bitbucket API v2.0 HTTP client.
type Client struct {
	http     *http.Client
	baseURL  string
	username string
	password string // API token for Basic Auth
	token    string // bearer access token

	// OAuth credentials for auto-refresh
	oauthCreds *Credentials

	// Cached scopes, guarded by scopesMu (separate from mu because Scopes
	// performs requests, which take mu to read the token).
	scopesMu       sync.Mutex
	apiTokenScopes []string
	scopesFetched  bool

	// mu guards token and oauthCreds.
	mu sync.Mutex
}

// NewClient creates a Bitbucket API client.
// Provide either (username + password) for Basic Auth or token for Bearer Auth.
func NewClient(username, password, token string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:  baseURL,
		username: username,
		password: password,
		token:    token,
	}
}

// NewClientFromCredentials creates a client from stored credentials, preserving cached scopes.
func NewClientFromCredentials(creds *Credentials) *Client {
	c := &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		oauthCreds: creds,
	}
	if creds.IsOAuth() {
		c.token = creds.AccessToken
	} else if creds.IsAPIToken() {
		c.username = creds.Email
		c.password = creds.APIToken
	}
	return c
}

// SetBaseURLForTesting points the client at u (e.g. an httptest.Server URL)
// instead of the Bitbucket Cloud API. It exists so other packages' tests can
// exercise code that uses a Client without network access; production code
// must not call it.
func (c *Client) SetBaseURLForTesting(u string) {
	c.baseURL = u
}

// ensureValidToken checks if the OAuth token is expired and refreshes if
// needed. It returns the bearer token to use, captured under the lock so
// concurrent refreshes cannot race with readers.
func (c *Client) ensureValidToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.oauthCreds == nil || !c.oauthCreds.IsExpired() {
		return c.token, nil
	}

	if err := RefreshOAuth(c.oauthCreds); err != nil {
		if !errors.Is(err, ErrTokenNotPersisted) {
			return "", fmt.Errorf("refreshing token: %w", err)
		}
		// The refresh itself worked; keep using the new token in this
		// process even though it could not be written back to disk.
		_, _ = fmt.Fprintf(stderrOut, "warning: %v\n", err)
	}

	c.token = c.oauthCreds.AccessToken
	return c.token, nil
}

// canRefresh reports whether OAuth credentials with a refresh token exist.
func (c *Client) canRefresh() bool {
	if c.oauthCreds == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.oauthCreds.RefreshToken != ""
}

// do executes an HTTP request with auth headers.
// An optional accept parameter overrides the default "application/json" Accept header.
func (c *Client) do(method, path string, bodyData []byte, contentType string, accept ...string) (*http.Response, error) {
	return c.doWith(c.http, method, path, bodyData, contentType, accept...)
}

// doWith is do on a caller-chosen http.Client (e.g. one without the 30s
// whole-request timeout, for large downloads).
func (c *Client) doWith(hc *http.Client, method, path string, bodyData []byte, contentType string, accept ...string) (*http.Response, error) {
	if err := rejectDotSegments(path); err != nil {
		return nil, err
	}

	token, err := c.ensureValidToken()
	if err != nil {
		return nil, err
	}

	acceptHeader := "application/json"
	if len(accept) > 0 && accept[0] != "" {
		acceptHeader = accept[0]
	}

	u := c.baseURL + path

	var bodyReader io.Reader
	if bodyData != nil {
		bodyReader = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", acceptHeader)

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	// Auto-retry once on 401 if we have OAuth with a refresh token
	if resp.StatusCode == http.StatusUnauthorized && c.canRefresh() {
		resp.Body.Close()
		c.mu.Lock()
		// Force expiry only if no concurrent request has already refreshed
		// past the token that just got the 401.
		if c.token == token {
			c.oauthCreds.CreatedAt = time.Time{}
		}
		c.mu.Unlock()
		newToken, err := c.ensureValidToken()
		if err != nil {
			return nil, fmt.Errorf("refreshing after 401: %w", err)
		}

		var retryBodyReader io.Reader
		if bodyData != nil {
			retryBodyReader = bytes.NewReader(bodyData)
		}

		req2, err := http.NewRequest(method, u, retryBodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating retry request: %w", err)
		}
		req2.Header.Set("Authorization", "Bearer "+newToken)
		if contentType != "" {
			req2.Header.Set("Content-Type", contentType)
		}
		req2.Header.Set("Accept", acceptHeader)
		return hc.Do(req2)
	}

	return resp, nil
}

// Get performs a GET request and returns the response body.
func (c *Client) Get(path string) ([]byte, error) {
	resp, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAuthError(resp, data)
	}

	return data, nil
}

// GetWithScopes performs a GET request and returns the response body and the x-oauth-scopes header.
func (c *Client) GetWithScopes(path string) (body []byte, scopes string, err error) {
	resp, err := c.do(http.MethodGet, path, nil, "")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, resp.Header.Get("X-Oauth-Scopes"), parseAuthError(resp, data)
	}

	return data, resp.Header.Get("X-Oauth-Scopes"), nil
}

// Scopes dynamically fetches and returns the token scopes by calling the API if not already cached.
func (c *Client) Scopes() ([]string, error) {
	c.scopesMu.Lock()
	defer c.scopesMu.Unlock()

	if c.scopesFetched {
		return c.apiTokenScopes, nil
	}

	c.mu.Lock()
	oauthScopes := ""
	if c.oauthCreds != nil {
		oauthScopes = c.oauthCreds.Scopes
	}
	c.mu.Unlock()
	if oauthScopes != "" {
		c.apiTokenScopes = parseScopesString(oauthScopes)
		c.scopesFetched = true
		return c.apiTokenScopes, nil
	}

	// /user returns the X-OAuth-Scopes header and works for both OAuth and API
	// token auth. A 403 here is fine — the header still populates, so a token
	// with Bitbucket scopes but no read:account succeeds.
	_, scopesStr, err := c.GetWithScopes("/user")

	// Header populated → success regardless of any non-2xx status the call returned.
	if scopesStr != "" {
		c.apiTokenScopes = parseScopesString(scopesStr)
		c.scopesFetched = true
		return c.apiTokenScopes, nil
	}

	if err != nil {
		return nil, fmt.Errorf("%s\n\n(underlying error: %w)", describeAuthFailure(err), err)
	}
	return nil, fmt.Errorf("failed to reliably fetch token scopes: API did not return X-OAuth-Scopes header")
}

func parseScopesString(s string) []string {
	if s == "" {
		return nil
	}
	// API tokens report scopes comma-separated (X-OAuth-Scopes); the OAuth
	// token endpoint returns them space-separated. Accept both.
	scopes := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	if len(scopes) == 0 {
		return nil
	}
	return scopes
}

// GetRaw performs a GET and returns raw bytes (for file content, logs, etc.).
func (c *Client) GetRaw(path string) (data []byte, contentType string, err error) {
	resp, doErr := c.do(http.MethodGet, path, nil, "", "*/*")
	if doErr != nil {
		return nil, "", doErr
	}
	defer resp.Body.Close()

	d, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, "", fmt.Errorf("reading response: %w", readErr)
	}

	if resp.StatusCode >= 400 {
		return nil, "", parseAuthError(resp, d)
	}

	return d, resp.Header.Get("Content-Type"), nil
}

// Post performs a POST request with a JSON body.
//
//nolint:dupl // post and put are structurally identical
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyData = b
	}

	resp, err := c.do(http.MethodPost, path, bodyData, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// PostMultipart performs a POST request using multipart/form-data.
// It takes a map of form fields and a map of file fields (where key is the field name and value is the file content).
func (c *Client) PostMultipart(path string, fields map[string]string, files map[string][]byte) ([]byte, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for key, val := range fields {
		if err := w.WriteField(key, val); err != nil {
			return nil, fmt.Errorf("writing field %s: %w", key, err)
		}
	}

	for key, fileBytes := range files {
		// Bitbucket API expects the form field name to be the file path.
		// We use CreateFormFile with the key as both fieldname and filename.
		fw, err := w.CreateFormFile(key, key)
		if err != nil {
			return nil, fmt.Errorf("creating form file %s: %w", key, err)
		}
		if _, err := fw.Write(fileBytes); err != nil {
			return nil, fmt.Errorf("writing file %s: %w", key, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	resp, err := c.do(http.MethodPost, path, b.Bytes(), w.FormDataContentType())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// Put performs a PUT request with a JSON body.
//
//nolint:dupl // post and put are structurally identical
func (c *Client) Put(path string, body interface{}) ([]byte, error) {
	var bodyData []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		bodyData = b
	}

	resp, err := c.do(http.MethodPut, path, bodyData, "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) error {
	resp, err := c.do(http.MethodDelete, path, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return parseAuthError(resp, data)
	}

	return nil
}

// Paginated is the standard Bitbucket pagination envelope.
type Paginated[T any] struct {
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	PageLen  int    `json:"pagelen"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Values   []T    `json:"values"`
}

// GetPaginated performs a GET and unmarshals the paginated response.
func GetPaginated[T any](c *Client, path string) (*Paginated[T], error) {
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result Paginated[T]
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling paginated response: %w", err)
	}

	return &result, nil
}

// GetJSON performs a GET and unmarshals the JSON response.
func GetJSON[T any](c *Client, path string) (*T, error) {
	data, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &result, nil
}

// QueryEscape URL-encodes a string for use as a path segment.
func QueryEscape(s string) string {
	return url.PathEscape(s)
}

// QueryValueEscape URL-encodes a string for use as a query-string value, so
// reserved characters such as & = + are percent-encoded. A space stays %20
// rather than becoming +, matching what these values have always sent.
func QueryValueEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func parseAPIError(statusCode int, body []byte) error {
	if statusCode == http.StatusForbidden {
		return fmt.Errorf("403 Forbidden: Permission denied. Ensure your API token has the required scopes for this operation. Additional details: %s", string(body))
	}
	return fmt.Errorf("API error %d: %s", statusCode, string(body))
}

// AuthError captures the diagnostic signature of a 401/403 from Bitbucket so
// callers can distinguish "classic unscoped Atlassian token rejected by
// policy" from "scoped token missing required scope" — both surface as 401
// at the network layer but differ in headers.
//
// Fingerprint guide:
//   - 401, WWW-Authenticate: BitbucketCustom, no X-Accepted-OAuth-Scopes
//     → IsClassicTokenRejection. Most common cause: classic Atlassian API
//     token (the unscoped "Create API token" button at id.atlassian.com).
//     Bitbucket REST requires scoped tokens since Sep 2025 phase-2 of
//     app-password deprecation. Same shape also fires for revoked tokens
//     and email/owner mismatch — so phrase as "most likely cause".
//   - 401/403, X-Accepted-OAuth-Scopes populated → scoped token reached the
//     OAuth2/scope-evaluation layer but is missing the listed scope.
type AuthError struct {
	StatusCode          int
	WWWAuthenticate     string
	AcceptedOAuthScopes string
	OAuthScopes         string
	Body                string
}

func (e *AuthError) Error() string {
	if e.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("403 Forbidden: Permission denied. Ensure your API token has the required scopes for this operation. Additional details: %s", e.Body)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Body)
}

// IsClassicTokenRejection reports whether the response shape matches a
// classic (unscoped) Atlassian API token being rejected by Bitbucket's
// BitbucketCustom auth layer. The signature also covers revoked tokens and
// wrong-email pairings, so present this to users as the leading hypothesis,
// not a certainty.
func (e *AuthError) IsClassicTokenRejection() bool {
	return e.StatusCode == http.StatusUnauthorized &&
		strings.Contains(e.WWWAuthenticate, "BitbucketCustom") &&
		e.AcceptedOAuthScopes == ""
}

// parseAuthError returns an *AuthError for 401/403 (preserving response
// headers needed for fingerprinting) and falls back to parseAPIError for
// other status codes.
func parseAuthError(resp *http.Response, body []byte) error {
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		return parseAPIError(resp.StatusCode, body)
	}
	return &AuthError{
		StatusCode:          resp.StatusCode,
		WWWAuthenticate:     resp.Header.Get("WWW-Authenticate"),
		AcceptedOAuthScopes: resp.Header.Get("X-Accepted-Oauth-Scopes"),
		OAuthScopes:         resp.Header.Get("X-Oauth-Scopes"),
		Body:                string(body),
	}
}

// classicTokenHelp is the canonical user-facing explanation for the
// classic-token-rejection fingerprint. Shared between login flow and
// runtime scope introspection so we phrase it the same way everywhere.
// Bitbucket answers a mistyped token and an unscoped (classic) token the
// same way, so both causes are named rather than guessing one.
var classicTokenHelp = `Bitbucket rejected this email + token.

Common causes:
  - The token was pasted incompletely, or the email is not the Atlassian
    account that owns the token.
  - It is a classic (unscoped) token, made with the plain "Create API
    token" button. Bitbucket only accepts tokens made with "Create API
    token with scopes" (Bitbucket scopes).
  - The token was revoked or has expired.

Create a new one at https://id.atlassian.com/manage-profile/security/api-tokens
with "Create API token with scopes" -> Bitbucket. Read-only scopes:
` + scopeLines(ReadOnlyScopes) + `To allow writes as well, add:
` + strings.TrimSuffix(scopeLines(WriteScopes), "\n")

// describeAuthFailure returns a user-friendly explanation for an auth
// failure, branching on AuthError fingerprint. Returns the original error
// string for non-AuthError inputs.
func describeAuthFailure(err error) string {
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		return err.Error()
	}
	if authErr.IsClassicTokenRejection() {
		return classicTokenHelp
	}
	if authErr.AcceptedOAuthScopes != "" {
		return fmt.Sprintf("Token is missing required scope(s): %s. Re-mint the token at id.atlassian.com with the listed scope(s) added.", authErr.AcceptedOAuthScopes)
	}
	return authErr.Error()
}
