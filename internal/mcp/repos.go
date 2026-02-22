package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManageRepositoriesArgs struct {
	Action      string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'create', 'delete'" jsonschema_enum:"list,get,create,delete"`
	Workspace   string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug    string `json:"repo_slug,omitempty" jsonschema:"Repository slug (required for 'get', 'create', 'delete')"`
	Description string `json:"description,omitempty" jsonschema:"Repository description (for 'create')"`
	Language    string `json:"language,omitempty" jsonschema:"Primary programming language (for 'create')"`
	IsPrivate   *bool  `json:"is_private,omitempty" jsonschema:"Whether the repo is private (default true, for 'create')"`
	ProjectKey  string `json:"project_key,omitempty" jsonschema:"Project key to assign the repo to (for 'create')"`
	Pagelen     int    `json:"pagelen,omitempty" jsonschema:"Results per page (default 25)"`
	Page        int    `json:"page,omitempty" jsonschema:"Page number"`
	Query       string `json:"query,omitempty" jsonschema:"Bitbucket query filter (e.g. name~'myrepo')"`
	Role        string `json:"role,omitempty" jsonschema:"Filter by role: owner, admin, contributor, member"`
	Sort        string `json:"sort,omitempty" jsonschema:"Sort field (e.g. -updated_on)"`
}

// ManageRepositoriesHandler handles the consolidated repository operations.
func ManageRepositoriesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManageRepositoriesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageRepositoriesArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListRepositories(bitbucket.ListRepositoriesArgs{
				Workspace: args.Workspace,
				Pagelen:   args.Pagelen,
				Page:      args.Page,
				Query:     args.Query,
				Role:      args.Role,
				Sort:      args.Sort,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list repositories: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get":
			if args.Workspace == "" || args.RepoSlug == "" {
				return ToolResultError("workspace and repo_slug are required for 'get' action"), nil, nil
			}
			repo, err := c.GetRepository(bitbucket.GetRepositoryArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get repository: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(repo, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "create":
			if args.Workspace == "" || args.RepoSlug == "" {
				return ToolResultError("workspace and repo_slug are required for 'create' action"), nil, nil
			}
			repo, err := c.CreateRepository(bitbucket.CreateRepositoryArgs{
				Workspace:   args.Workspace,
				RepoSlug:    args.RepoSlug,
				Description: args.Description,
				Language:    args.Language,
				IsPrivate:   args.IsPrivate,
				ProjectKey:  args.ProjectKey,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create repository: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(repo, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "delete":
			if args.Workspace == "" || args.RepoSlug == "" {
				return ToolResultError("workspace and repo_slug are required for 'delete' action"), nil, nil
			}
			err := c.DeleteRepository(bitbucket.DeleteRepositoryArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete repository: %v", err)), nil, nil
			}
			return ToolResultText("Repository deleted successfully"), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
