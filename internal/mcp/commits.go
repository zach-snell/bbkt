package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListCommitsHandler lists commits for a repository or branch.
func ListCommitsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListCommitsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListCommitsArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListCommits(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list commits: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetCommitHandler gets a single commit by hash.
func GetCommitHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetCommitArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetCommitArgs) (*mcp.CallToolResult, any, error) {
		commit, err := c.GetCommit(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get commit: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(commit, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetDiffHandler gets the diff between two revisions or for a single commit.
func GetDiffHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetDiffArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetDiffArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetDiff(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get diff: %v", err)), nil, nil
		}

		return ToolResultText(string(raw)), nil, nil
	}
}

// GetDiffStatHandler gets the diff stat for a revision spec.
func GetDiffStatHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetDiffStatArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetDiffStatArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.GetDiffStat(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get diffstat: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}
