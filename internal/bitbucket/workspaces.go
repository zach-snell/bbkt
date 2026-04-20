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

	// /workspaces was deprecated by Bitbucket on 2026-02-25 (CHANGE-2770).
	// /user/workspaces (added in CHANGE-3022) returns the same paginated
	// envelope but scoped to the authenticated user's workspace memberships.
	path := fmt.Sprintf("/user/workspaces?pagelen=%d&page=%d", pagelen, page)
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
