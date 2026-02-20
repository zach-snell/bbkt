package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
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
	password string // app password or API token
	token    string // bearer access token

	// OAuth credentials for auto-refresh
	oauthCreds *Credentials
	mu         sync.Mutex
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

// NewClientFromOAuth creates a client from stored OAuth credentials with auto-refresh.
func NewClientFromOAuth(creds *Credentials) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		token:      creds.AccessToken,
		oauthCreds: creds,
	}
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
func (c *Client) do(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, err
	}

	u := c.baseURL + path

	req, err := http.NewRequest(method, u, body)
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

		req2, err := http.NewRequest(method, u, body)
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
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// GetRaw performs a GET and returns raw bytes (for file content).
func (c *Client) GetRaw(path string) ([]byte, string, error) {
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
		return nil, "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, resp.Header.Get("Content-Type"), nil
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		reader = strings.NewReader(string(data))
	}

	resp, err := c.do(http.MethodPost, path, reader, "application/json")
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
func (c *Client) Put(path string, body interface{}) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		reader = strings.NewReader(string(data))
	}

	resp, err := c.do(http.MethodPut, path, reader, "application/json")
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
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
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
