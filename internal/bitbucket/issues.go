package bitbucket

import (
	"encoding/json"
	"fmt"
	"time"
)

// Issue represents a Bitbucket issue.
type Issue struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   Content   `json:"content"`
	State     string    `json:"state"`
	Priority  string    `json:"priority"`
	Kind      string    `json:"kind"`
	Assignee  *User     `json:"assignee,omitempty"`
	Reporter  *User     `json:"reporter,omitempty"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Votes     int       `json:"votes"`
	Watches   int       `json:"watches"`
	Links     Links     `json:"links"`
}

type ListIssuesArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	State     string `json:"state,omitempty" jsonschema:"Filter by state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed)"`
	Kind      string `json:"kind,omitempty" jsonschema:"Filter by kind (bug, enhancement, proposal, task)"`
	Priority  string `json:"priority,omitempty" jsonschema:"Filter by priority (trivial, minor, major, critical, blocker)"`
	Search    string `json:"search,omitempty" jsonschema:"Search query"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
}

// ListIssues lists issues for a repository.
func (c *Client) ListIssues(args ListIssuesArgs) (*Paginated[Issue], error) {
	if args.Workspace == "" || args.RepoSlug == "" {
		return nil, fmt.Errorf("workspace and repo_slug are required")
	}

	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}
	page := args.Page
	if page == 0 {
		page = 1
	}

	path := fmt.Sprintf("/repositories/%s/%s/issues?pagelen=%d&page=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pagelen, page)

	// Build query string for BB QL
	var queries []string
	if args.State != "" {
		queries = append(queries, fmt.Sprintf("state=%q", args.State))
	}
	if args.Kind != "" {
		queries = append(queries, fmt.Sprintf("kind=%q", args.Kind))
	}
	if args.Priority != "" {
		queries = append(queries, fmt.Sprintf("priority=%q", args.Priority))
	}

	if len(queries) > 0 {
		path += "&q=" + QueryEscape(joinQueries(queries, " AND "))
	}

	if args.Search != "" {
		// q search string is separate from general query
		path += "&search=" + QueryEscape(args.Search)
	}

	if args.Sort != "" {
		path += "&sort=" + QueryEscape(args.Sort)
	}

	return GetPaginated[Issue](c, path)
}

// Helper to join queries
func joinQueries(queries []string, sep string) string {
	if len(queries) == 0 {
		return ""
	}
	result := queries[0]
	for _, q := range queries[1:] {
		result += sep + q
	}
	return result
}

type GetIssueArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	IssueID   int    `json:"issue_id" jsonschema:"Issue ID"`
}

// GetIssue gets details for a single issue.
func (c *Client) GetIssue(args GetIssueArgs) (*Issue, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.IssueID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and issue_id are required")
	}

	return GetJSON[Issue](c, fmt.Sprintf("/repositories/%s/%s/issues/%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.IssueID))
}

type CreateIssueArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Title     string `json:"title" jsonschema:"Title of the issue"`
	Content   string `json:"content,omitempty" jsonschema:"Description of the issue (markdown)"`
	Kind      string `json:"kind,omitempty" jsonschema:"Kind: bug, enhancement, proposal, task (default: bug)"`
	Priority  string `json:"priority,omitempty" jsonschema:"Priority: trivial, minor, major, critical, blocker (default: major)"`
	Assignee  string `json:"assignee,omitempty" jsonschema:"Assignee account ID"`
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(args CreateIssueArgs) (*Issue, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Title == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and title are required")
	}

	kind := args.Kind
	if kind == "" {
		kind = "bug"
	}
	priority := args.Priority
	if priority == "" {
		priority = "major"
	}

	body := map[string]interface{}{
		"title":    args.Title,
		"kind":     kind,
		"priority": priority,
	}

	if args.Content != "" {
		body["content"] = map[string]string{"raw": args.Content}
	}

	if args.Assignee != "" {
		body["assignee"] = map[string]string{"account_id": args.Assignee}
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s/issues",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %v", err)
	}

	var issue Issue
	if err := json.Unmarshal(respData, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &issue, nil
}

type UpdateIssueArgs struct {
	Workspace string  `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string  `json:"repo_slug" jsonschema:"Repository slug"`
	IssueID   int     `json:"issue_id" jsonschema:"Issue ID"`
	Title     *string `json:"title,omitempty" jsonschema:"New title"`
	Content   *string `json:"content,omitempty" jsonschema:"New description"`
	State     *string `json:"state,omitempty" jsonschema:"New state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed)"`
	Kind      *string `json:"kind,omitempty" jsonschema:"New kind"`
	Priority  *string `json:"priority,omitempty" jsonschema:"New priority"`
	Assignee  *string `json:"assignee,omitempty" jsonschema:"New assignee account ID (or empty string to unassign)"`
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(args UpdateIssueArgs) (*Issue, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.IssueID == 0 {
		return nil, fmt.Errorf("workspace, repo_slug, and issue_id are required")
	}

	body := map[string]interface{}{}
	if args.Title != nil {
		body["title"] = *args.Title
	}
	if args.Content != nil {
		body["content"] = map[string]string{"raw": *args.Content}
	}
	if args.State != nil {
		body["state"] = *args.State
	}
	if args.Kind != nil {
		body["kind"] = *args.Kind
	}
	if args.Priority != nil {
		body["priority"] = *args.Priority
	}
	if args.Assignee != nil {
		if *args.Assignee == "" {
			body["assignee"] = nil // Unassign
		} else {
			body["assignee"] = map[string]string{"account_id": *args.Assignee}
		}
	}

	respData, err := c.Put(fmt.Sprintf("/repositories/%s/%s/issues/%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.IssueID), body)
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %v", err)
	}

	var issue Issue
	if err := json.Unmarshal(respData, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &issue, nil
}
