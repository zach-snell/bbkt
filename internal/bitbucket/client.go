package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
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

	// Cached scopes
	apiTokenScopes []string
	scopesFetched  bool

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

// ensureValidToken checks if the OAuth token is expired and refreshes if needed.
func (c *Client) ensureValidToken() error {
	if c.oauthCreds == nil {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.oauthCreds.IsExpired() {
		return nil
	}

	if err := RefreshOAuth(c.oauthCreds); err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}

	c.token = c.oauthCreds.AccessToken
	return nil
}

// do executes an HTTP request with auth headers.
func (c *Client) do(method, path string, bodyData []byte, contentType string) (*http.Response, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, err
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

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	// Auto-retry once on 401 if we have OAuth with a refresh token
	if resp.StatusCode == http.StatusUnauthorized && c.oauthCreds != nil && c.oauthCreds.RefreshToken != "" {
		resp.Body.Close()
		c.mu.Lock()
		c.oauthCreds.CreatedAt = time.Time{} // force expiry
		c.mu.Unlock()
		if err := c.ensureValidToken(); err != nil {
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
		req2.Header.Set("Authorization", "Bearer "+c.token)
		if contentType != "" {
			req2.Header.Set("Content-Type", contentType)
		}
		req2.Header.Set("Accept", "application/json")
		return c.http.Do(req2)
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
		return nil, parseAPIError(resp.StatusCode, data)
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
		return nil, resp.Header.Get("X-Oauth-Scopes"), parseAPIError(resp.StatusCode, data)
	}

	return data, resp.Header.Get("X-Oauth-Scopes"), nil
}

// Scopes dynamically fetches and returns the token scopes by calling the API if not already cached.
func (c *Client) Scopes() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.scopesFetched {
		return c.apiTokenScopes, nil
	}

	if c.oauthCreds != nil && c.oauthCreds.Scopes != "" {
		c.apiTokenScopes = parseScopesString(c.oauthCreds.Scopes)
		c.scopesFetched = true
		return c.apiTokenScopes, nil
	}

	_, scopesStr, _ := c.GetWithScopes("/workspace")
	if scopesStr == "" {
		// Try /user as fallback if workspace fails
		_, scopesStr, _ = c.GetWithScopes("/user")
	}

	if scopesStr == "" {
		return nil, fmt.Errorf("failed to reliably fetch token scopes: API did not return X-OAuth-Scopes header")
	}

	c.apiTokenScopes = parseScopesString(scopesStr)
	c.scopesFetched = true
	return c.apiTokenScopes, nil
}

func parseScopesString(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var scopes []string
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	return scopes
}

// GetRaw performs a GET and returns raw bytes (for file content).
func (c *Client) GetRaw(path string) (data []byte, contentType string, err error) {
	resp, doErr := c.do(http.MethodGet, path, nil, "")
	if doErr != nil {
		return nil, "", doErr
	}
	defer resp.Body.Close()

	d, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, "", fmt.Errorf("reading response: %w", readErr)
	}

	if resp.StatusCode >= 400 {
		return nil, "", parseAPIError(resp.StatusCode, d)
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
		return parseAPIError(resp.StatusCode, data)
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

// QueryEscape URL-encodes a string for use in paths.
func QueryEscape(s string) string {
	return url.PathEscape(s)
}

func parseAPIError(statusCode int, body []byte) error {
	if statusCode == http.StatusForbidden {
		return fmt.Errorf("403 Forbidden: Permission denied. Ensure your Bitbucket App Password has the required scopes for this operation. Additional details: %s", string(body))
	}
	return fmt.Errorf("API error %d: %s", statusCode, string(body))
}
