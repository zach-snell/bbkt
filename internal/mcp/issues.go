package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManageIssuesArgs struct {
	Action    string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'create', 'update'" jsonschema_enum:"list,get,create,update"`
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	IssueID   int    `json:"issue_id,omitempty" jsonschema:"Issue ID (for 'get', 'update')"`
	Title     string `json:"title,omitempty" jsonschema:"Issue title (for 'create', 'update')"`
	Content   string `json:"content,omitempty" jsonschema:"Issue description content (for 'create', 'update')"`
	State     string `json:"state,omitempty" jsonschema:"Issue state: new, open, resolved, on hold, invalid, duplicate, wontfix, closed (for 'list', 'create', 'update')"`
	Kind      string `json:"kind,omitempty" jsonschema:"Issue kind: bug, enhancement, proposal, task (for 'create', 'update')"`
	Priority  string `json:"priority,omitempty" jsonschema:"Issue priority: trivial, minor, major, critical, blocker (for 'create', 'update')"`
	Assignee  string `json:"assignee,omitempty" jsonschema:"Account ID of the user assigned to the issue (for 'create', 'update')"`
	Query     string `json:"query,omitempty" jsonschema:"Filter query (for 'list')"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
}

// ManageIssuesHandler handles the consolidated issue operations.
func ManageIssuesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManageIssuesArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListIssues(bitbucket.ListIssuesArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				State:     args.State,
				Page:      args.Page,
				Pagelen:   args.Pagelen,
				Search:    args.Query, // map query to search
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list issues: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get":
			if args.IssueID == 0 {
				return ToolResultError("issue_id is required for 'get' action"), nil, nil
			}
			result, err := c.GetIssue(bitbucket.GetIssueArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				IssueID:   args.IssueID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get issue: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "create":
			if args.Title == "" {
				return ToolResultError("title is required for 'create' action"), nil, nil
			}
			result, err := c.CreateIssue(bitbucket.CreateIssueArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Title:     args.Title,
				Content:   args.Content,
				Kind:      args.Kind,
				Priority:  args.Priority,
				Assignee:  args.Assignee,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create issue: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "update":
			if args.IssueID == 0 {
				return ToolResultError("issue_id is required for 'update' action"), nil, nil
			}

			var title, content, state, kind, priority, assignee *string
			if args.Title != "" {
				title = &args.Title
			}
			if args.Content != "" {
				content = &args.Content
			}
			if args.State != "" {
				state = &args.State
			}
			if args.Kind != "" {
				kind = &args.Kind
			}
			if args.Priority != "" {
				priority = &args.Priority
			}
			if args.Assignee != "" {
				if args.Assignee == "unassigned" {
					empty := ""
					assignee = &empty
				} else {
					assignee = &args.Assignee
				}
			}

			result, err := c.UpdateIssue(bitbucket.UpdateIssueArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				IssueID:   args.IssueID,
				Title:     title,
				Content:   content,
				State:     state,
				Kind:      kind,
				Priority:  priority,
				Assignee:  assignee,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to update issue: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
