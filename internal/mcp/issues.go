package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListIssuesHandler lists issues for a repository.
func ListIssuesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListIssuesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListIssuesArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListIssues(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list issues: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetIssueHandler gets details for a single issue.
func GetIssueHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetIssueArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetIssueArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.GetIssue(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get issue: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// CreateIssueHandler creates a new issue.
func CreateIssueHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CreateIssueArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CreateIssueArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.CreateIssue(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to create issue: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// UpdateIssueHandler updates an existing issue.
func UpdateIssueHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.UpdateIssueArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.UpdateIssueArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.UpdateIssue(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to update issue: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}
