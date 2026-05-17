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
//
// The /user/workspaces endpoint returns workspace_access envelopes rather
// than bare workspaces — each row is {administrator, workspace:{uuid,slug,
// links}}, where workspace_base omits Name and IsPrivate. We flatten the
// envelope here so callers see a clean []Workspace; Name/IsPrivate stay
// zero on listing rows (use GetWorkspace for those).
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
	// /user/workspaces (added in CHANGE-3022) returns a paginated envelope
	// of workspace_access objects (not bare workspaces) scoped to the
	// authenticated user.
	path := fmt.Sprintf("/user/workspaces?pagelen=%d&page=%d", pagelen, page)
	raw, err := GetPaginated[workspaceAccess](c, path)
	if err != nil {
		return nil, err
	}

	out := &Paginated[Workspace]{
		Size:     raw.Size,
		Page:     raw.Page,
		PageLen:  raw.PageLen,
		Next:     raw.Next,
		Previous: raw.Previous,
		Values:   make([]Workspace, 0, len(raw.Values)),
	}
	for _, a := range raw.Values {
		w := a.Workspace
		w.IsAdmin = a.Administrator
		out.Values = append(out.Values, w)
	}
	return out, nil
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
