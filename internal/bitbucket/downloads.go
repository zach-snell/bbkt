package bitbucket

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Download is a file in a repository's Downloads section.
type Download struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Downloads int       `json:"downloads"`
	CreatedOn time.Time `json:"created_on"`
	User      *User     `json:"user"`
	Links     Links     `json:"links"`
}

type ListDownloadsArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default 25)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
}

// ListDownloads lists a repository's Downloads, newest first.
func (c *Client) ListDownloads(args ListDownloadsArgs) (*Paginated[Download], error) {
	if args.Workspace == "" || args.RepoSlug == "" {
		return nil, fmt.Errorf("workspace and repo_slug are required")
	}
	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}
	page := args.Page
	if page == 0 {
		page = 1
	}
	return GetPaginated[Download](c, fmt.Sprintf("/repositories/%s/%s/downloads?pagelen=%d&page=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pagelen, page))
}

type GetDownloadArgs struct {
	Workspace string
	RepoSlug  string
	Name      string
}

// GetDownload streams one Downloads file into w and returns the bytes written.
// Bitbucket answers with a redirect to signed storage; net/http follows it and
// drops the Authorization header on the cross-host hop.
func (c *Client) GetDownload(args GetDownloadArgs, w io.Writer) (int64, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Name == "" {
		return 0, fmt.Errorf("workspace, repo_slug, and file name are required")
	}
	// Same client but no whole-request timeout: bundles can be hundreds of MB.
	hc := &http.Client{Transport: c.http.Transport, CheckRedirect: c.http.CheckRedirect}
	resp, err := c.doWith(hc, http.MethodGet, fmt.Sprintf("/repositories/%s/%s/downloads/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.Name)), nil, "", "*/*")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return 0, parseAuthError(resp, data)
	}
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return n, fmt.Errorf("reading download: %w", err)
	}
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		return n, fmt.Errorf("download truncated: got %d of %d bytes", n, resp.ContentLength)
	}
	return n, nil
}
