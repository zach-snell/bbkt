package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManageSourceArgs struct {
	Action    string `json:"action" jsonschema:"Action to perform: 'read_file', 'list_directory', 'get_history', 'search', 'write_file', 'delete_file'" jsonschema_enum:"read_file,list_directory,get_history,search,write_file,delete_file"`
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Path      string `json:"path,omitempty" jsonschema:"Path to the file or directory"`
	Ref       string `json:"ref,omitempty" jsonschema:"Commit hash, branch, or tag (default: HEAD)"`
	Query     string `json:"query,omitempty" jsonschema:"Search query"`
	Content   string `json:"content,omitempty" jsonschema:"Content to write to the file"`
	Message   string `json:"message,omitempty" jsonschema:"Commit message"`
	Branch    string `json:"branch,omitempty" jsonschema:"Branch to commit to"`
	Author    string `json:"author,omitempty" jsonschema:"Commit author in 'Name <email>' format"`
	MaxDepth  int    `json:"max_depth,omitempty" jsonschema:"Maximum depth of recursion (for list_directory)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
}

// ManageSourceHandler handles the consolidated source file and directory operations.
func ManageSourceHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManageSourceArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageSourceArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "read_file":
			if args.Path == "" {
				return ToolResultError("path is required for 'read_file' action"), nil, nil
			}
			raw, contentType, err := c.GetFileContent(bitbucket.GetFileContentArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Path:      args.Path,
				Ref:       args.Ref,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get file content: %v", err)), nil, nil
			}

			if strings.Contains(contentType, "application/json") {
				var prettyJSON interface{}
				if err := json.Unmarshal(raw, &prettyJSON); err == nil {
					data, _ := json.MarshalIndent(prettyJSON, "", "  ")
					return ToolResultText(string(data)), nil, nil
				}
			}
			return ToolResultText(string(raw)), nil, nil

		case "list_directory":
			result, err := c.ListDirectory(bitbucket.ListDirectoryArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Path:      args.Path,
				Ref:       args.Ref,
				MaxDepth:  args.MaxDepth,
				Pagelen:   args.Pagelen,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list directory: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get_history":
			if args.Path == "" {
				return ToolResultError("path is required for 'get_history' action"), nil, nil
			}
			result, err := c.GetFileHistory(bitbucket.GetFileHistoryArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Path:      args.Path,
				Ref:       args.Ref,
				Pagelen:   args.Pagelen,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get file history: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "search":
			if args.Query == "" {
				return ToolResultError("query is required for 'search' action"), nil, nil
			}
			raw, err := c.SearchCode(bitbucket.SearchCodeArgs{
				Workspace:   args.Workspace,
				RepoSlug:    args.RepoSlug,
				SearchQuery: args.Query,
				Page:        args.Page,
				Pagelen:     args.Pagelen,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to search code: %v", err)), nil, nil
			}
			var prettyJSON interface{}
			if err := json.Unmarshal(raw, &prettyJSON); err == nil {
				data, _ := json.MarshalIndent(prettyJSON, "", "  ")
				return ToolResultText(string(data)), nil, nil
			}
			return ToolResultText(string(raw)), nil, nil

		case "write_file":
			if args.Path == "" || args.Content == "" || args.Message == "" {
				return ToolResultError("path, content, and message are required for 'write_file' action"), nil, nil
			}
			err := c.WriteFile(bitbucket.WriteFileArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Path:      args.Path,
				Content:   args.Content,
				Message:   args.Message,
				Branch:    args.Branch,
				Author:    args.Author,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to write file: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Successfully wrote %s", args.Path)), nil, nil

		case "delete_file":
			if args.Path == "" || args.Message == "" {
				return ToolResultError("path and message are required for 'delete_file' action"), nil, nil
			}
			err := c.DeleteFile(bitbucket.DeleteFileArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Path:      args.Path,
				Message:   args.Message,
				Branch:    args.Branch,
				Author:    args.Author,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete file: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Successfully deleted %s", args.Path)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
