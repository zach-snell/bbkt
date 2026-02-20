package bitbucket

import (
	"encoding/json"
	"fmt"
)

type ListBranchesArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Query     string `json:"query,omitempty" jsonschema:"Filter query"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field"`
}

// ListBranches lists branches in a repository.
func (c *Client) ListBranches(args ListBranchesArgs) (*Paginated[Branch], error) {
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

	path := fmt.Sprintf("/repositories/%s/%s/refs/branches?pagelen=%d&page=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pagelen, page)
	if args.Query != "" {
		path += "&q=" + QueryEscape(args.Query)
	}
	if args.Sort != "" {
		path += "&sort=" + QueryEscape(args.Sort)
	}

	return GetPaginated[Branch](c, path)
}

type CreateBranchArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Name      string `json:"name" jsonschema:"Branch name"`
	Target    string `json:"target" jsonschema:"Target commit hash to branch from"`
}

// CreateBranch creates a new branch from a commit hash.
func (c *Client) CreateBranch(args CreateBranchArgs) (*Branch, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Name == "" || args.Target == "" {
		return nil, fmt.Errorf("workspace, repo_slug, name, and target are required")
	}

	body := CreateBranchRequest{
		Name:   args.Name,
		Target: map[string]string{"hash": args.Target},
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s/refs/branches",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %v", err)
	}

	var branch Branch
	if err := json.Unmarshal(respData, &branch); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &branch, nil
}

type DeleteBranchArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Name      string `json:"name" jsonschema:"Branch name to delete"`
}

// DeleteBranch deletes a branch.
func (c *Client) DeleteBranch(args DeleteBranchArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.Name == "" {
		return fmt.Errorf("workspace, repo_slug, and name are required")
	}

	return c.Delete(fmt.Sprintf("/repositories/%s/%s/refs/branches/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.Name)))
}

type ListTagsArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
}

// ListTags lists tags in a repository.
func (c *Client) ListTags(args ListTagsArgs) (*Paginated[Tag], error) {
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

	path := fmt.Sprintf("/repositories/%s/%s/refs/tags?pagelen=%d&page=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pagelen, page)

	return GetPaginated[Tag](c, path)
}

type CreateTagArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Name      string `json:"name" jsonschema:"Tag name"`
	Target    string `json:"target" jsonschema:"Target commit hash"`
}

// CreateTag creates a new tag.
func (c *Client) CreateTag(args CreateTagArgs) (*Tag, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Name == "" || args.Target == "" {
		return nil, fmt.Errorf("workspace, repo_slug, name, and target are required")
	}

	body := map[string]interface{}{
		"name":   args.Name,
		"target": map[string]string{"hash": args.Target},
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s/refs/tags",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %v", err)
	}

	var tag Tag
	if err := json.Unmarshal(respData, &tag); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &tag, nil
}
