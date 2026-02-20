package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListBranchesHandler lists branches in a repository.
func ListBranchesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListBranchesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListBranchesArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListBranches(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list branches: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// CreateBranchHandler creates a new branch from a commit hash.
func CreateBranchHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CreateBranchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CreateBranchArgs) (*mcp.CallToolResult, any, error) {
		branch, err := c.CreateBranch(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to create branch: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(branch, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// DeleteBranchHandler deletes a branch.
func DeleteBranchHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.DeleteBranchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.DeleteBranchArgs) (*mcp.CallToolResult, any, error) {
		if err := c.DeleteBranch(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to delete branch: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Branch '%s' deleted successfully", args.Name)), nil, nil
	}
}

// ListTagsHandler lists tags in a repository.
func ListTagsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListTagsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListTagsArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListTags(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list tags: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// CreateTagHandler creates a new tag.
func CreateTagHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CreateTagArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CreateTagArgs) (*mcp.CallToolResult, any, error) {
		tag, err := c.CreateTag(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to create tag: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(tag, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}
