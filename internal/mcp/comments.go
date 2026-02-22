package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManagePRCommentsArgs struct {
	Action    string `json:"action" jsonschema:"Action to perform: 'list', 'create', 'update', 'delete', 'resolve', 'unresolve'" jsonschema_enum:"list,create,update,delete,resolve,unresolve"`
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID      int    `json:"pr_id" jsonschema:"Pull request ID"`
	CommentID int    `json:"comment_id,omitempty" jsonschema:"Comment ID (for 'update', 'delete', 'resolve', 'unresolve')"`
	Content   string `json:"content,omitempty" jsonschema:"Markdown content (for 'create', 'update')"`
	ParentID  int    `json:"parent_id,omitempty" jsonschema:"Parent comment ID to reply to (for 'create')"`
	FilePath  string `json:"file_path,omitempty" jsonschema:"File path for inline comments (for 'create')"`
	LineFrom  int    `json:"line_from,omitempty" jsonschema:"Line number the comment applies to for deleted lines (for 'create')"`
	LineTo    int    `json:"line_to,omitempty" jsonschema:"Line number the comment applies to for new/modified lines (for 'create')"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default 50)"`
}

// ManagePRCommentsHandler handles the consolidated PR comments operations.
func ManagePRCommentsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManagePRCommentsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManagePRCommentsArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListPRComments(bitbucket.ListPRCommentsArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
				Page:      args.Page,
				Pagelen:   args.Pagelen,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list PR comments: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "create":
			if args.Content == "" {
				return ToolResultError("content is required for 'create' action"), nil, nil
			}
			comment, err := c.CreatePRComment(bitbucket.CreatePRCommentArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
				Content:   args.Content,
				ParentID:  args.ParentID,
				FilePath:  args.FilePath,
				LineFrom:  args.LineFrom,
				LineTo:    args.LineTo,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to create comment: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(comment, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "update":
			if args.CommentID == 0 || args.Content == "" {
				return ToolResultError("comment_id and content are required for 'update' action"), nil, nil
			}
			comment, err := c.UpdatePRComment(bitbucket.UpdatePRCommentArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
				CommentID: args.CommentID,
				Content:   args.Content,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to update comment: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(comment, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "delete":
			if args.CommentID == 0 {
				return ToolResultError("comment_id is required for 'delete' action"), nil, nil
			}
			if err := c.DeletePRComment(bitbucket.CommentActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
				CommentID: args.CommentID,
			}); err != nil {
				return ToolResultError(fmt.Sprintf("failed to delete comment: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Comment #%d deleted successfully", args.CommentID)), nil, nil

		case "resolve":
			if args.CommentID == 0 {
				return ToolResultError("comment_id is required for 'resolve' action"), nil, nil
			}
			if err := c.ResolvePRComment(bitbucket.CommentActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
				CommentID: args.CommentID,
			}); err != nil {
				return ToolResultError(fmt.Sprintf("failed to resolve comment: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Comment #%d resolved", args.CommentID)), nil, nil

		case "unresolve":
			if args.CommentID == 0 {
				return ToolResultError("comment_id is required for 'unresolve' action"), nil, nil
			}
			if err := c.UnresolvePRComment(bitbucket.CommentActionArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				PRID:      args.PRID,
				CommentID: args.CommentID,
			}); err != nil {
				return ToolResultError(fmt.Sprintf("failed to unresolve comment: %v", err)), nil, nil
			}
			return ToolResultText(fmt.Sprintf("Comment #%d reopened", args.CommentID)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
