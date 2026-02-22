package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManagePullRequestsArgs struct {
	Action            string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'create', 'update', 'merge', 'approve', 'unapprove', 'decline', 'get-diff', 'get-diffstat', 'get-commits'" jsonschema_enum:"list,get,create,update,merge,approve,unapprove,decline,get-diff,get-diffstat,get-commits"`
	Workspace         string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug          string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID              int    `json:"pr_id,omitempty" jsonschema:"Pull request ID"`
	Title             string `json:"title,omitempty" jsonschema:"Title of the pull request (for 'create', 'update')"`
	Description       string `json:"description,omitempty" jsonschema:"Description of the pull request (for 'create', 'update')"`
	SourceBranch      string `json:"source_branch,omitempty" jsonschema:"Source branch name (for 'create')"`
	DestinationBranch string `json:"destination_branch,omitempty" jsonschema:"Destination branch name (for 'create')"`
	CloseSourceBranch bool   `json:"close_source_branch,omitempty" jsonschema:"Close source branch (for 'create', 'merge')"`
	Draft             bool   `json:"draft,omitempty" jsonschema:"Create as a draft PR (for 'create')"`
	Message           string `json:"message,omitempty" jsonschema:"Commit message (for 'merge')"`
	MergeStrategy     string `json:"merge_strategy,omitempty" jsonschema:"Merge strategy (e.g. merge_commit, squash, fast_forward) (for 'merge')"`
	State             string `json:"state,omitempty" jsonschema:"Filter by state (MERGED, SUPERSEDED, OPEN, DECLINED) (for 'list')"`
	Query             string `json:"query,omitempty" jsonschema:"Filter query (for 'list')"`
	Page              int    `json:"page,omitempty" jsonschema:"Page number"`
	Pagelen           int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
}

// ManagePullRequestsHandler handles the consolidated pull request operations.
func ManagePullRequestsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManagePullRequestsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManagePullRequestsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListPullRequests(bitbucket.ListPullRequestsArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				State:     args.State,
				Pagelen:   args.Pagelen,
				Page:      args.Page,
				Query:     args.Query,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list pull requests: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'get' action"), nil, nil
			}
			pr, err := c.GetPullRequest(bitbucket.GetPullRequestArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get pull request: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(pr, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "create":
			if args.Title == "" || args.SourceBranch == "" {
				return ToolResultError("title and source_branch are required for 'create' action"), nil, nil
			}
			pr, err := c.CreatePullRequest(bitbucket.CreatePullRequestArgs{
				Workspace:         args.Workspace,
				RepoSlug:          args.RepoSlug,
				Title:             args.Title,
				Description:       args.Description,
				SourceBranch:      args.SourceBranch,
				DestinationBranch: args.DestinationBranch,
				CloseSourceBranch: args.CloseSourceBranch,
				Draft:             args.Draft,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create pull request: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(pr, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "update":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'update' action"), nil, nil
			}

			var title, description *string
			if args.Title != "" {
				title = &args.Title
			}
			if args.Description != "" {
				description = &args.Description
			}

			pr, err := c.UpdatePullRequest(bitbucket.UpdatePullRequestArgs{
				Workspace:   args.Workspace,
				RepoSlug:    args.RepoSlug,
				PRID:        args.PRID,
				Title:       title,
				Description: description,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to update pull request: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(pr, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "merge":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'merge' action"), nil, nil
			}
			pr, err := c.MergePullRequest(bitbucket.MergePullRequestArgs{
				Workspace:         args.Workspace,
				RepoSlug:          args.RepoSlug,
				PRID:              args.PRID,
				Message:           args.Message,
				CloseSourceBranch: args.CloseSourceBranch,
				MergeStrategy:     args.MergeStrategy,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to merge pull request: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(pr, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "approve":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'approve' action"), nil, nil
			}
			if err := c.ApprovePullRequest(bitbucket.PullRequestActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
			}); err != nil {
				return ToolResultError(fmt.Sprintf("failed to approve pull request: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Pull request #%d approved", args.PRID)), nil, nil

		case "unapprove":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'unapprove' action"), nil, nil
			}
			if err := c.UnapprovePullRequest(bitbucket.PullRequestActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
			}); err != nil {
				return ToolResultError(fmt.Sprintf("failed to unapprove pull request: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Pull request #%d unapproved", args.PRID)), nil, nil

		case "decline":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'decline' action"), nil, nil
			}
			if err := c.DeclinePullRequest(bitbucket.PullRequestActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
			}); err != nil {
				return ToolResultError(fmt.Sprintf("failed to decline pull request: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Pull request #%d declined", args.PRID)), nil, nil

		case "get-diff":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'get-diff' action"), nil, nil
			}
			raw, err := c.GetPRDiff(bitbucket.PullRequestActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get PR diff: %v", err)), nil, nil
			}
			return ToolResultText(string(raw)), nil, nil

		case "get-diffstat":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'get-diffstat' action"), nil, nil
			}
			result, err := c.GetPRDiffStat(bitbucket.PullRequestActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get PR diffstat: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get-commits":
			if args.PRID == 0 {
				return ToolResultError("pr_id is required for 'get-commits' action"), nil, nil
			}
			result, err := c.ListPRCommits(bitbucket.PullRequestActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list PR commits: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
