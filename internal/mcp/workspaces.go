package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManageWorkspacesArgs struct {
	Action    string `json:"action" jsonschema:"Action to perform: 'list', 'get'" jsonschema_enum:"list,get"`
	Workspace string `json:"workspace,omitempty" jsonschema:"Workspace slug or UUID (required for 'get')"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Number of results per page (default 25)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
}

// ManageWorkspacesHandler handles list and get operations for workspaces.
func ManageWorkspacesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManageWorkspacesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageWorkspacesArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListWorkspaces(bitbucket.ListWorkspacesArgs{
				Pagelen: args.Pagelen,
				Page:    args.Page,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list workspaces: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get":
			if args.Workspace == "" {
				return ToolResultError("workspace is required for 'get' action"), nil, nil
			}
			ws, err := c.GetWorkspace(bitbucket.GetWorkspaceArgs{
				Workspace: args.Workspace,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get workspace: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(ws, "", "  ")
			return ToolResultText(string(data)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
