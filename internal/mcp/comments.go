package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListPRCommentsHandler lists comments on a pull request.
func ListPRCommentsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListPRCommentsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListPRCommentsArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListPRComments(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list PR comments: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// CreatePRCommentHandler creates a comment on a pull request.
func CreatePRCommentHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CreatePRCommentArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CreatePRCommentArgs) (*mcp.CallToolResult, any, error) {
		comment, err := c.CreatePRComment(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to create comment: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(comment, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// UpdatePRCommentHandler updates an existing comment.
func UpdatePRCommentHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.UpdatePRCommentArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.UpdatePRCommentArgs) (*mcp.CallToolResult, any, error) {
		comment, err := c.UpdatePRComment(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to update comment: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(comment, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// DeletePRCommentHandler deletes a comment on a pull request.
func DeletePRCommentHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CommentActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CommentActionArgs) (*mcp.CallToolResult, any, error) {
		if err := c.DeletePRComment(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to delete comment: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Comment #%d deleted successfully", args.CommentID)), nil, nil
	}
}

// ResolvePRCommentHandler resolves a comment thread.
func ResolvePRCommentHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CommentActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CommentActionArgs) (*mcp.CallToolResult, any, error) {
		if err := c.ResolvePRComment(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to resolve comment: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Comment #%d resolved", args.CommentID)), nil, nil
	}
}

// UnresolvePRCommentHandler reopens a resolved comment thread.
func UnresolvePRCommentHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.CommentActionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.CommentActionArgs) (*mcp.CallToolResult, any, error) {
		if err := c.UnresolvePRComment(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to unresolve comment: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Comment #%d reopened", args.CommentID)), nil, nil
	}
}
