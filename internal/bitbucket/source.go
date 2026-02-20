package bitbucket

import (
	"encoding/json"
	"fmt"
)

type GetFileContentArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Path      string `json:"path" jsonschema:"Path to the file"`
	Ref       string `json:"ref,omitempty" jsonschema:"Commit hash, branch, or tag (default: HEAD)"`
}

// GetFileContent reads a file's content from the repository.
func (c *Client) GetFileContent(args GetFileContentArgs) (content []byte, contentType string, err error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Path == "" {
		return nil, "", fmt.Errorf("workspace, repo_slug, and path are required")
	}

	var endpoint string
	if args.Ref != "" {
		endpoint = fmt.Sprintf("/repositories/%s/%s/src/%s/%s",
			QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.Ref), args.Path)
	} else {
		endpoint = fmt.Sprintf("/repositories/%s/%s/src/HEAD/%s",
			QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.Path)
	}

	return c.GetRaw(endpoint)
}

type ListDirectoryArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Path      string `json:"path,omitempty" jsonschema:"Path to the directory"`
	Ref       string `json:"ref,omitempty" jsonschema:"Commit hash, branch, or tag (default: HEAD)"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default: 100)"`
	MaxDepth  int    `json:"max_depth,omitempty" jsonschema:"Maximum depth of recursion (default: 1)"`
}

// ListDirectory lists files and directories at a given path.
func (c *Client) ListDirectory(args ListDirectoryArgs) (*Paginated[TreeEntry], error) {
	if args.Workspace == "" || args.RepoSlug == "" {
		return nil, fmt.Errorf("workspace and repo_slug are required")
	}

	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 100
	}
	maxDepth := args.MaxDepth
	if maxDepth == 0 {
		maxDepth = 1
	}

	var endpoint string
	if args.Ref != "" {
		if args.Path != "" {
			endpoint = fmt.Sprintf("/repositories/%s/%s/src/%s/%s",
				QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.Ref), args.Path)
		} else {
			endpoint = fmt.Sprintf("/repositories/%s/%s/src/%s/",
				QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.Ref))
		}
	} else {
		if args.Path != "" {
			endpoint = fmt.Sprintf("/repositories/%s/%s/src/HEAD/%s",
				QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.Path)
		} else {
			endpoint = fmt.Sprintf("/repositories/%s/%s/src/HEAD/",
				QueryEscape(args.Workspace), QueryEscape(args.RepoSlug))
		}
	}

	endpoint += fmt.Sprintf("?pagelen=%d&max_depth=%d", pagelen, maxDepth)

	return GetPaginated[TreeEntry](c, endpoint)
}

type GetFileHistoryArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Path      string `json:"path" jsonschema:"Path to the file"`
	Ref       string `json:"ref,omitempty" jsonschema:"Commit hash, branch, or tag (default: HEAD)"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default: 25)"`
}

// GetFileHistory gets the commit history for a specific file.
func (c *Client) GetFileHistory(args GetFileHistoryArgs) (*Paginated[json.RawMessage], error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.Path == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and path are required")
	}

	ref := args.Ref
	if ref == "" {
		ref = "HEAD"
	}
	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s/filehistory/%s/%s?pagelen=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(ref), args.Path, pagelen)

	// Filehistory returns commit objects with file metadata
	return GetPaginated[json.RawMessage](c, endpoint)
}

type SearchCodeArgs struct {
	Workspace   string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug    string `json:"repo_slug" jsonschema:"Repository slug"`
	SearchQuery string `json:"query" jsonschema:"Search query"`
	Pagelen     int    `json:"pagelen,omitempty" jsonschema:"Results per page (default: 25)"`
	Page        int    `json:"page,omitempty" jsonschema:"Page number"`
}

// SearchCode searches for code in a repository using Bitbucket's code search.
func (c *Client) SearchCode(args SearchCodeArgs) ([]byte, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.SearchQuery == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and query are required")
	}

	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}
	page := args.Page
	if page == 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s/search/code?search_query=%s&pagelen=%d&page=%d",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), QueryEscape(args.SearchQuery), pagelen, page)

	return c.Get(endpoint)
}

type WriteFileArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Path      string `json:"path" jsonschema:"Path to the file"`
	Content   string `json:"content" jsonschema:"Content to write to the file"`
	Message   string `json:"message" jsonschema:"Commit message"`
	Branch    string `json:"branch,omitempty" jsonschema:"Branch to commit to"`
	Author    string `json:"author,omitempty" jsonschema:"Commit author in 'Name <email>' format"`
}

// WriteFile writes or updates a file in the repository.
func (c *Client) WriteFile(args WriteFileArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.Path == "" {
		return fmt.Errorf("workspace, repo_slug, and path are required")
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s/src",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug))

	fields := make(map[string]string)
	if args.Message != "" {
		fields["message"] = args.Message
	}
	if args.Branch != "" {
		fields["branch"] = args.Branch
	}
	if args.Author != "" {
		fields["author"] = args.Author
	}

	files := map[string][]byte{
		args.Path: []byte(args.Content),
	}

	_, err := c.PostMultipart(endpoint, fields, files)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

type DeleteFileArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Path      string `json:"path" jsonschema:"Path to the file to delete"`
	Message   string `json:"message" jsonschema:"Commit message"`
	Branch    string `json:"branch,omitempty" jsonschema:"Branch to commit to"`
	Author    string `json:"author,omitempty" jsonschema:"Commit author in 'Name <email>' format"`
}

// DeleteFile deletes a file from the repository.
func (c *Client) DeleteFile(args DeleteFileArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.Path == "" {
		return fmt.Errorf("workspace, repo_slug, and path are required")
	}

	endpoint := fmt.Sprintf("/repositories/%s/%s/src",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug))

	fields := make(map[string]string)
	if args.Message != "" {
		fields["message"] = args.Message
	}
	if args.Branch != "" {
		fields["branch"] = args.Branch
	}
	if args.Author != "" {
		fields["author"] = args.Author
	}

	// Bitbucket API expects "files" as the key and the path as the value
	// However, we just send it as a regular text field
	fields["files"] = args.Path

	_, err := c.PostMultipart(endpoint, fields, nil)
	if err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}

	return nil
}
