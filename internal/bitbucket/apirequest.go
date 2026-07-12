package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// apiHost is the only host the passthrough will talk to. baseURL already pins
// requests here, but NormalizeAPIPath rejects full URLs pointing elsewhere so a
// caller can't be tricked into aiming the passthrough at another host.
const apiHost = "api.bitbucket.org"

// NormalizeAPIPath turns a user- or model-supplied API reference into a path
// relative to the v2 base ("/2.0"). It is forgiving on purpose — the whole
// point of the passthrough is that the model should not have to remember an
// exact form. All of these normalize to "/repositories/ws/repo":
//
//	repositories/ws/repo
//	/repositories/ws/repo
//	/2.0/repositories/ws/repo
//	https://api.bitbucket.org/2.0/repositories/ws/repo
//
// A full URL pointing at any host other than api.bitbucket.org is rejected.
func NormalizeAPIPath(raw string) (string, error) {
	p := strings.TrimSpace(raw)
	if p == "" {
		return "", fmt.Errorf("empty API path")
	}

	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		u, err := url.Parse(p)
		if err != nil {
			return "", fmt.Errorf("invalid URL %q: %w", raw, err)
		}
		if !strings.EqualFold(u.Host, apiHost) {
			return "", fmt.Errorf("refusing host %q: the passthrough only talks to %s", u.Host, apiHost)
		}
		p = u.EscapedPath()
		if u.RawQuery != "" {
			p += "?" + u.RawQuery
		}
	}

	p = strings.TrimPrefix(p, "/")
	p = strings.TrimPrefix(p, "2.0/")
	if p == "2.0" {
		p = ""
	}
	return "/" + p, nil
}

// Request performs an arbitrary authenticated call against the Bitbucket v2 API
// and returns the raw HTTP status and body. Unlike Get/Post, it does NOT turn a
// non-2xx into an error — the caller sees the true status and body, so the model
// gets the real API error (e.g. a 404 body explaining the bad path) instead of a
// swallowed failure. This backs `bbkt api` / the bitbucket_api MCP tool.
func (c *Client) Request(method, rawPath string, body []byte) (status int, respBody []byte, err error) {
	path, err := NormalizeAPIPath(rawPath)
	if err != nil {
		return 0, nil, err
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}

	contentType := ""
	if len(body) > 0 {
		contentType = "application/json"
	}

	resp, err := c.do(method, path, body, contentType)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("reading response: %w", err)
	}
	return resp.StatusCode, data, nil
}

// RequestPaginated follows Bitbucket's `next` cursor and merges every page's
// `values` into a single JSON envelope. It is meant for GETs against collection
// endpoints; a non-paginated response is returned unchanged. maxPages caps the
// walk (<=0 means no cap); when the cap stops the walk before the last page, the
// result is flagged truncated with the next cursor, so a cap is never silent.
func (c *Client) RequestPaginated(method, rawPath string, body []byte, maxPages int) (status int, respBody []byte, err error) {
	// Pagination only makes sense for reads. Reject writes up front so a
	// paginated POST/PUT/PATCH/DELETE can never re-issue the mutation against
	// each `next` page.
	m := strings.ToUpper(strings.TrimSpace(method))
	if m == "" {
		m = http.MethodGet
	}
	if m != http.MethodGet && m != http.MethodHead {
		return 0, nil, fmt.Errorf("pagination is only supported for GET/HEAD requests, not %s", m)
	}

	// Bound the walk so a self-referential or very large `next` chain can't run
	// unbounded. maxPages <= 0 means "as many as the hard cap allows".
	const hardMaxPages = 100
	if maxPages <= 0 || maxPages > hardMaxPages {
		maxPages = hardMaxPages
	}

	var all []json.RawMessage
	path := rawPath
	pages := 0
	lastStatus := 0
	nextCursor := ""
	truncated := false
	seen := map[string]bool{}

	for {
		// Always a bodyless GET/HEAD; `next` cursors are read URLs.
		st, data, reqErr := c.Request(m, path, nil)
		if reqErr != nil {
			return st, nil, reqErr
		}
		lastStatus = st
		if st >= 400 {
			// Surface the error body as-is; don't pretend it paginated.
			return st, data, nil
		}

		var env struct {
			Values []json.RawMessage `json:"values"`
			Next   string            `json:"next"`
		}
		if jerr := json.Unmarshal(data, &env); jerr != nil || env.Values == nil {
			// Not a paginated envelope. If it's the very first response, hand it
			// back untouched; otherwise stop and return what we've merged.
			if pages == 0 {
				return st, data, nil
			}
			break
		}

		all = append(all, env.Values...)
		pages++
		nextCursor = env.Next

		if nextCursor == "" {
			break
		}
		if pages >= maxPages {
			truncated = true
			break
		}
		if seen[nextCursor] {
			truncated = true // self-referential cursor loop
			break
		}
		seen[nextCursor] = true
		path = nextCursor
	}

	out := map[string]any{
		"values":        all,
		"count":         len(all),
		"pages_fetched": pages,
	}
	if truncated {
		out["truncated"] = true
		out["next"] = nextCursor
	}
	merged, mErr := json.MarshalIndent(out, "", "  ")
	if mErr != nil {
		return lastStatus, nil, fmt.Errorf("merging pages: %w", mErr)
	}
	return lastStatus, merged, nil
}
