package bitbucket

import (
	"encoding/json"
	"fmt"
)

type ListPullRequestsArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	State     string `json:"state,omitempty" jsonschema:"Filter by state (MERGED, SUPERSEDED, OPEN, DECLINED, default OPEN)"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Query     string `json:"query,omitempty" jsonschema:"Filter query"`
}

// ListPullRequests lists pull requests for a repository.
func (c *Client) ListPullRequests(args ListPullRequestsArgs) (*Paginated[PullRequest], error) {
	if args.Workspace == "" || args.RepoSlug == "" {
		return nil, fmt.Errorf("workspace and repo_slug are required")
	}

	state := args.State
	if state == "" {
		state = "OPEN"
	}
	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}
	page := args.Page
	if page == 0 {
		page = 1
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?state=%s&pagelen=%d&page=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), state, pagelen, page)
	if args.Query != "" {
		path += "&q=" + QueryEscape(args.Query)
	}

	return GetPaginated[PullRequest](c, path)
}

type GetPullRequestArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID      int    `json:"pr_id" jsonschema:"Pull request ID"`
}

// GetPullRequest gets details for a single pull request.
func (c *Client) GetPullRequest(args GetPullRequestArgs) (*PullRequest, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	return GetJSON[PullRequest](c, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID))
}

type CreatePullRequestArgs struct {
	Workspace         string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug          string `json:"repo_slug" jsonschema:"Repository slug"`
	Title             string `json:"title" jsonschema:"Title of the pull request"`
	SourceBranch      string `json:"source_branch" jsonschema:"Source branch name"`
	DestinationBranch string `json:"destination_branch,omitempty" jsonschema:"Destination branch name (optional, defaults to repo default)"`
	Description       string `json:"description,omitempty" jsonschema:"Description of the pull request"`
	CloseSourceBranch bool   `json:"close_source_branch,omitempty" jsonschema:"Close source branch on merge"`
	Draft             bool   `json:"draft,omitempty" jsonschema:"Create as a draft PR"`
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(args CreatePullRequestArgs) (*PullRequest, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Title == "" || args.SourceBranch == "" {
		return nil, fmt.Errorf("workspace, repo_slug, title, and source_branch are required")
	}

	body := CreatePRRequest{
		Title: args.Title,
		Source: PREndpoint{
			Branch: &Branch{Name: args.SourceBranch},
			Repository: &MinRepo{
				FullName: fmt.Sprintf("%s/%s", args.Workspace, args.RepoSlug),
			},
		},
		Description:       args.Description,
		CloseSourceBranch: args.CloseSourceBranch,
		Draft:             args.Draft,
	}

	if args.DestinationBranch != "" {
		body.Destination = PREndpoint{
			Branch: &Branch{Name: args.DestinationBranch},
			Repository: &MinRepo{
				FullName: fmt.Sprintf("%s/%s", args.Workspace, args.RepoSlug),
			},
		}
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %v", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(respData, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &pr, nil
}

type UpdatePullRequestArgs struct {
	Workspace   string  `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug    string  `json:"repo_slug" jsonschema:"Repository slug"`
	PRID        int     `json:"pr_id" jsonschema:"Pull request ID"`
	Title       *string `json:"title,omitempty" jsonschema:"New title for the pull request"`
	Description *string `json:"description,omitempty" jsonschema:"New description for the pull request"`
}

// UpdatePullRequest updates an existing pull request.
func (c *Client) UpdatePullRequest(args UpdatePullRequestArgs) (*PullRequest, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	body := map[string]interface{}{}
	if args.Title != nil {
		body["title"] = *args.Title
	}
	if args.Description != nil {
		body["description"] = *args.Description
	}

	respData, err := c.Put(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID), body)
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %v", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(respData, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &pr, nil
}

type MergePullRequestArgs struct {
	Workspace         string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug          string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID              int    `json:"pr_id" jsonschema:"Pull request ID"`
	CloseSourceBranch bool   `json:"close_source_branch,omitempty" jsonschema:"Close source branch"`
	MergeStrategy     string `json:"merge_strategy,omitempty" jsonschema:"Merge strategy (e.g. merge_commit, squash, fast_forward, default: merge_commit)"`
	Message           string `json:"message,omitempty" jsonschema:"Commit message"`
}

// MergePullRequest merges a pull request.
func (c *Client) MergePullRequest(args MergePullRequestArgs) (*PullRequest, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	strategy := args.MergeStrategy
	if strategy == "" {
		strategy = "merge_commit"
	}

	body := MergePRRequest{
		Type:              "pullrequest",
		CloseSourceBranch: args.CloseSourceBranch,
		MergeStrategy:     strategy,
		Message:           args.Message,
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID), body)
	if err != nil {
		return nil, fmt.Errorf("failed to merge pull request: %v", err)
	}

	var pr PullRequest
	if err := json.Unmarshal(respData, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &pr, nil
}

type PullRequestActionArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID      int    `json:"pr_id" jsonschema:"Pull request ID"`
}

// ApprovePullRequest approves a pull request.
func (c *Client) ApprovePullRequest(args PullRequestActionArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	_, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID), map[string]interface{}{})
	return err
}

// UnapprovePullRequest removes approval from a pull request.
func (c *Client) UnapprovePullRequest(args PullRequestActionArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	return c.Delete(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/approve",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID))
}

// DeclinePullRequest declines a pull request.
func (c *Client) DeclinePullRequest(args PullRequestActionArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	_, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/decline",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID), map[string]interface{}{})
	return err
}

// GetPRDiff gets the diff for a pull request.
func (c *Client) GetPRDiff(args PullRequestActionArgs) ([]byte, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	raw, _, err := c.GetRaw(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diff",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID))
	return raw, err
}

// GetPRDiffStat gets the diffstat for a pull request.
func (c *Client) GetPRDiffStat(args PullRequestActionArgs) (*Paginated[DiffStat], error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	return GetPaginated[DiffStat](c, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/diffstat",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID))
}

// ListPRCommits lists commits in a pull request.
func (c *Client) ListPRCommits(args PullRequestActionArgs) (*Paginated[Commit], error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	return GetPaginated[Commit](c, fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/commits",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID))
}
