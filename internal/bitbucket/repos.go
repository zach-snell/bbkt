package bitbucket

import (
	"encoding/json"
	"fmt"
)

type ListRepositoriesArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default 25)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Query     string `json:"query,omitempty" jsonschema:"Bitbucket query filter (e.g. name~'myrepo')"`
	Role      string `json:"role,omitempty" jsonschema:"Filter by role: owner, admin, contributor, member"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field (e.g. -updated_on)"`
}

// ListRepositories lists repositories in a workspace.
func (c *Client) ListRepositories(args ListRepositoriesArgs) (*Paginated[Repository], error) {
	if args.Workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}
	page := args.Page
	if page == 0 {
		page = 1
	}

	path := fmt.Sprintf("/repositories/%s?pagelen=%d&page=%d", QueryEscape(args.Workspace), pagelen, page)
	if args.Query != "" {
		path += "&q=" + QueryEscape(args.Query)
	}
	if args.Role != "" {
		path += "&role=" + QueryEscape(args.Role)
	}
	if args.Sort != "" {
		path += "&sort=" + QueryEscape(args.Sort)
	}

	return GetPaginated[Repository](c, path)
}

type GetRepositoryArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
}

// GetRepository gets details for a single repository.
func (c *Client) GetRepository(args GetRepositoryArgs) (*Repository, error) {
	if args.Workspace == "" || args.RepoSlug == "" {
		return nil, fmt.Errorf("workspace and repo_slug are required")
	}

	return GetJSON[Repository](c, fmt.Sprintf("/repositories/%s/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)))
}

type CreateRepositoryArgs struct {
	Workspace   string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug    string `json:"repo_slug" jsonschema:"Repository slug (URL-friendly name)"`
	Description string `json:"description,omitempty" jsonschema:"Repository description"`
	Language    string `json:"language,omitempty" jsonschema:"Primary programming language"`
	IsPrivate   *bool  `json:"is_private,omitempty" jsonschema:"Whether the repo is private (default true)"`
	ProjectKey  string `json:"project_key,omitempty" jsonschema:"Project key to assign the repo to"`
}

// CreateRepository creates a new repository in a workspace.
func (c *Client) CreateRepository(args CreateRepositoryArgs) (*Repository, error) {
	if args.Workspace == "" || args.RepoSlug == "" {
		return nil, fmt.Errorf("workspace and repo_slug are required")
	}

	body := map[string]interface{}{
		"scm": "git",
	}

	if args.Description != "" {
		body["description"] = args.Description
	}
	if args.Language != "" {
		body["language"] = args.Language
	}

	if args.IsPrivate != nil {
		body["is_private"] = *args.IsPrivate
	} else {
		body["is_private"] = true
	}

	if args.ProjectKey != "" {
		body["project"] = map[string]string{"key": args.ProjectKey}
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %v", err)
	}

	var repo Repository
	if err := json.Unmarshal(respData, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &repo, nil
}

type DeleteRepositoryArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
}

// DeleteRepository deletes a repository.
func (c *Client) DeleteRepository(args DeleteRepositoryArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" {
		return fmt.Errorf("workspace and repo_slug are required")
	}

	return c.Delete(fmt.Sprintf("/repositories/%s/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)))
}
