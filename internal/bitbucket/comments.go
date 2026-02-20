package bitbucket

import (
	"encoding/json"
	"fmt"
)

type ListPRCommentsArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID      int    `json:"pr_id" jsonschema:"Pull request ID"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default 50)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
}

// ListPRComments lists comments on a pull request.
func (c *Client) ListPRComments(args ListPRCommentsArgs) (*Paginated[PRComment], error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and pr_id are required")
	}

	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 50
	}
	page := args.Page
	if page == 0 {
		page = 1
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments?pagelen=%d&page=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID, pagelen, page)

	return GetPaginated[PRComment](c, path)
}

type CreatePRCommentArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID      int    `json:"pr_id" jsonschema:"Pull request ID"`
	Content   string `json:"content" jsonschema:"Markdown content of the comment"`
	FilePath  string `json:"file_path,omitempty" jsonschema:"File path for inline comments"`
	LineTo    int    `json:"line_to,omitempty" jsonschema:"Line number the comment applies to (for new/modified lines)"`
	LineFrom  int    `json:"line_from,omitempty" jsonschema:"Line number the comment applies to (for deleted lines)"`
	ParentID  int    `json:"parent_id,omitempty" jsonschema:"Parent comment ID to reply to"`
}

// CreatePRComment creates a comment on a pull request.
func (c *Client) CreatePRComment(args CreatePRCommentArgs) (*PRComment, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 || args.Content == "" {
		return nil, fmt.Errorf("workspace, repo_slug, pr_id, and content are required")
	}

	body := CreateCommentRequest{
		Content: Content{Raw: args.Content},
	}

	// Inline comment support
	if args.FilePath != "" {
		body.Inline = &Inline{
			Path: args.FilePath,
		}
		if args.LineTo > 0 {
			lineTo := args.LineTo
			body.Inline.To = &lineTo
		}
		if args.LineFrom > 0 {
			lineFrom := args.LineFrom
			body.Inline.From = &lineFrom
		}
	}

	// Reply to parent comment
	if args.ParentID > 0 {
		body.Parent = &ParentRef{ID: args.ParentID}
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %v", err)
	}

	var comment PRComment
	if err := json.Unmarshal(respData, &comment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &comment, nil
}

type UpdatePRCommentArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID      int    `json:"pr_id" jsonschema:"Pull request ID"`
	CommentID int    `json:"comment_id" jsonschema:"Comment ID to update"`
	Content   string `json:"content" jsonschema:"New markdown content"`
}

// UpdatePRComment updates an existing comment.
func (c *Client) UpdatePRComment(args UpdatePRCommentArgs) (*PRComment, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 || args.CommentID == 0 || args.Content == "" {
		return nil, fmt.Errorf("workspace, repo_slug, pr_id, comment_id, and content are required")
	}

	body := map[string]interface{}{
		"content": map[string]string{"raw": args.Content},
	}

	respData, err := c.Put(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID, args.CommentID), body)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %v", err)
	}

	var comment PRComment
	if err := json.Unmarshal(respData, &comment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &comment, nil
}

type CommentActionArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	PRID      int    `json:"pr_id" jsonschema:"Pull request ID"`
	CommentID int    `json:"comment_id" jsonschema:"Comment ID"`
}

// DeletePRComment deletes a comment on a pull request.
func (c *Client) DeletePRComment(args CommentActionArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 || args.CommentID == 0 {
		return fmt.Errorf("workspace, repo_slug, pr_id, and comment_id are required")
	}

	return c.Delete(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID, args.CommentID))
}

// ResolvePRComment resolves a comment thread.
func (c *Client) ResolvePRComment(args CommentActionArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 || args.CommentID == 0 {
		return fmt.Errorf("workspace, repo_slug, pr_id, and comment_id are required")
	}

	_, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d/resolve",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID, args.CommentID), nil)
	return err
}

// UnresolvePRComment reopens a resolved comment thread.
func (c *Client) UnresolvePRComment(args CommentActionArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.PRID == 0 || args.CommentID == 0 {
		return fmt.Errorf("workspace, repo_slug, pr_id, and comment_id are required")
	}

	return c.Delete(fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments/%d/resolve",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PRID, args.CommentID))
}
