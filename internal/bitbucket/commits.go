package bitbucket

import (
	"fmt"
)

type ListCommitsArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Revision  string `json:"revision,omitempty" jsonschema:"Branch name or commit hash to list commits for"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Include   string `json:"include,omitempty" jsonschema:"Include commits reachable from this ref"`
	Exclude   string `json:"exclude,omitempty" jsonschema:"Exclude commits reachable from this ref"`
	Path      string `json:"path,omitempty" jsonschema:"Filter commits that touch this file path"`
}

// ListCommits lists commits for a repository or branch.
func (c *Client) ListCommits(args ListCommitsArgs) (*Paginated[Commit], error) {
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

	var endpoint string
	if args.Revision != "" {
		endpoint = fmt.Sprintf("/repositories/%s/%s/commits/%s?pagelen=%d&page=%d",
			QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.Revision), pagelen, page)
	} else {
		endpoint = fmt.Sprintf("/repositories/%s/%s/commits?pagelen=%d&page=%d",
			QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pagelen, page)
	}

	if args.Include != "" {
		endpoint += "&include=" + QueryEscape(args.Include)
	}
	if args.Exclude != "" {
		endpoint += "&exclude=" + QueryEscape(args.Exclude)
	}
	if args.Path != "" {
		endpoint += "&path=" + QueryEscape(args.Path)
	}

	return GetPaginated[Commit](c, endpoint)
}

type GetCommitArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Commit    string `json:"commit" jsonschema:"Commit hash"`
}

// GetCommit gets a single commit by hash.
func (c *Client) GetCommit(args GetCommitArgs) (*Commit, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Commit == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and commit are required")
	}

	return GetJSON[Commit](c, fmt.Sprintf("/repositories/%s/%s/commit/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.Commit)))
}

type GetDiffArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Spec      string `json:"spec" jsonschema:"Diff spec: single commit hash or 'hash1..hash2'"`
	Path      string `json:"path,omitempty" jsonschema:"Filter diff to this file path"`
}

// GetDiff gets the diff between two revisions or for a single commit.
func (c *Client) GetDiff(args GetDiffArgs) ([]byte, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Spec == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and spec are required")
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s/diff/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.Spec)
	if args.Path != "" {
		endpoint += "?path=" + args.Path
	}

	raw, _, err := c.GetRaw(endpoint)
	return raw, err
}

type GetDiffStatArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Spec      string `json:"spec" jsonschema:"Diff spec: single commit hash or 'hash1..hash2'"`
}

// GetDiffStat gets the diff stat for a revision spec.
func (c *Client) GetDiffStat(args GetDiffStatArgs) (*Paginated[DiffStat], error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Spec == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and spec are required")
	}

	return GetPaginated[DiffStat](c, fmt.Sprintf("/repositories/%s/%s/diffstat/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.Spec))
}
