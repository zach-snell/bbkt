package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListPullRequestsHandler lists pull requests for a repository.
func ListPullRequestsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListPullRequestsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListPullRequestsArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListPullRequests(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list pull requests: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetPullRequestHandler gets details for a single pull request.
func GetPullRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetPullRequestArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetPullRequestArgs) (*mcp.CallToolResult, any, error) {
		pr, err := c.GetPullRequest(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get pull request: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(pr, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// CreatePullRequestHandler creates a new pull request.
func CreatePullRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CreatePullRequestArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CreatePullRequestArgs) (*mcp.CallToolResult, any, error) {
		pr, err := c.CreatePullRequest(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to create pull request: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(pr, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// UpdatePullRequestHandler updates an existing pull request.
func UpdatePullRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.UpdatePullRequestArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.UpdatePullRequestArgs) (*mcp.CallToolResult, any, error) {
		pr, err := c.UpdatePullRequest(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to update pull request: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(pr, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// MergePullRequestHandler merges a pull request.
func MergePullRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.MergePullRequestArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.MergePullRequestArgs) (*mcp.CallToolResult, any, error) {
		pr, err := c.MergePullRequest(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to merge pull request: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(pr, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// ApprovePullRequestHandler approves a pull request.
func ApprovePullRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
		if err := c.ApprovePullRequest(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to approve pull request: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Pull request #%d approved", args.PRID)), nil, nil
	}
}

// UnapprovePullRequestHandler removes approval from a pull request.
func UnapprovePullRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
		if err := c.UnapprovePullRequest(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to unapprove pull request: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Pull request #%d unapproved", args.PRID)), nil, nil
	}
}

// DeclinePullRequestHandler declines a pull request.
func DeclinePullRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
		if err := c.DeclinePullRequest(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to decline pull request: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Pull request #%d declined", args.PRID)), nil, nil
	}
}

// GetPRDiffHandler gets the diff for a pull request.
func GetPRDiffHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetPRDiff(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get PR diff: %v", err)), nil, nil
		}

		return ToolResultText(string(raw)), nil, nil
	}
}

// GetPRDiffStatHandler gets the diffstat for a pull request.
func GetPRDiffStatHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.GetPRDiffStat(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get PR diffstat: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// ListPRCommitsHandler lists commits in a pull request.
func ListPRCommitsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.PullRequestActionArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListPRCommits(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list PR commits: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}
