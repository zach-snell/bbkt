package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// GetFileContentHandler reads a file's content from the repository.
func GetFileContentHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetFileContentArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetFileContentArgs) (*mcp.CallToolResult, any, error) {
		raw, contentType, err := c.GetFileContent(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get file content: %v", err)), nil, nil
		}

		// If it looks like JSON (directory listing), format it nicely
		if strings.Contains(contentType, "application/json") {
			var prettyJSON interface{}
			if err := json.Unmarshal(raw, &prettyJSON); err == nil {
				data, _ := json.MarshalIndent(prettyJSON, "", "  ")
				return ToolResultText(string(data)), nil, nil
			}
		}

		return ToolResultText(string(raw)), nil, nil
	}
}

// ListDirectoryHandler lists files and directories at a given path.
func ListDirectoryHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListDirectoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListDirectoryArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListDirectory(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list directory: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetFileHistoryHandler gets the commit history for a specific file.
func GetFileHistoryHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetFileHistoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetFileHistoryArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.GetFileHistory(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get file history: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// SearchCodeHandler searches for code in a repository using Bitbucket's code search.
func SearchCodeHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.SearchCodeArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.SearchCodeArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.SearchCode(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to search code: %v", err)), nil, nil
		}

		var prettyJSON interface{}
		if err := json.Unmarshal(raw, &prettyJSON); err == nil {
			data, _ := json.MarshalIndent(prettyJSON, "", "  ")
			return ToolResultText(string(data)), nil, nil
		}

		return ToolResultText(string(raw)), nil, nil
	}
}

// WriteFileHandler writes or updates a file in the repository.
func WriteFileHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.WriteFileArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.WriteFileArgs) (*mcp.CallToolResult, any, error) {
		err := c.WriteFile(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to write file: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Successfully wrote %s", args.Path)), nil, nil
	}
}

// DeleteFileHandler deletes a file from the repository.
func DeleteFileHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.DeleteFileArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.DeleteFileArgs) (*mcp.CallToolResult, any, error) {
		err := c.DeleteFile(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to delete file: %v", err)), nil, nil
		}

		return ToolResultText(fmt.Sprintf("Successfully deleted %s", args.Path)), nil, nil
	}
}
