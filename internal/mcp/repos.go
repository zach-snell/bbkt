package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListRepositoriesHandler lists repositories in a workspace.
func ListRepositoriesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListRepositoriesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListRepositoriesArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListRepositories(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list repositories: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetRepositoryHandler gets details for a single repository.
func GetRepositoryHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetRepositoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetRepositoryArgs) (*mcp.CallToolResult, any, error) {
		repo, err := c.GetRepository(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get repository: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(repo, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// CreateRepositoryHandler creates a new repository in a workspace.
func CreateRepositoryHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CreateRepositoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CreateRepositoryArgs) (*mcp.CallToolResult, any, error) {
		repo, err := c.CreateRepository(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to create repository: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(repo, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// DeleteRepositoryHandler deletes a repository.
func DeleteRepositoryHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.DeleteRepositoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.DeleteRepositoryArgs) (*mcp.CallToolResult, any, error) {
		if err := c.DeleteRepository(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to delete repository: %v", err)), nil, nil
		}

		return ToolResultText("Repository deleted successfully"), nil, nil
	}
}
