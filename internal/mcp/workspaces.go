package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListWorkspacesHandler returns workspaces for the authenticated user.
func ListWorkspacesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListWorkspacesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListWorkspacesArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListWorkspaces(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list workspaces: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetWorkspaceHandler returns details for a single workspace.
func GetWorkspaceHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetWorkspaceArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetWorkspaceArgs) (*mcp.CallToolResult, any, error) {
		ws, err := c.GetWorkspace(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get workspace: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(ws, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}
