package bitbucket

import (
	"fmt"
	"net/url"
)

type ListWorkspacesArgs struct {
	Pagelen int `json:"pagelen,omitempty" jsonschema:"Number of results per page (default 25, max 100)"`
	Page    int `json:"page,omitempty" jsonschema:"Page number (1-based)"`
}

// ListWorkspaces returns workspaces for the authenticated user.
func (c *Client) ListWorkspaces(args ListWorkspacesArgs) (*Paginated[Workspace], error) {
	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}
	page := args.Page
	if page == 0 {
		page = 1
	}

	path := fmt.Sprintf("/workspaces?pagelen=%d&page=%d", pagelen, page)
	return GetPaginated[Workspace](c, path)
}

type GetWorkspaceArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug or UUID"`
}

// GetWorkspace returns details for a single workspace.
func (c *Client) GetWorkspace(args GetWorkspaceArgs) (*Workspace, error) {
	if args.Workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	return GetJSON[Workspace](c, fmt.Sprintf("/workspaces/%s", url.QueryEscape(args.Workspace)))
}
