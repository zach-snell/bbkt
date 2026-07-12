package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// isWriteMethod reports whether a method mutates state. GET/HEAD are safe reads;
// everything else can write and is gated behind BBKT_API_ALLOW_WRITE.
func isWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, "":
		return false
	}
	return true
}

// redactQuery drops the query string from a path for logging, since callers may
// put secrets (tokens, access_token, etc.) in query parameters.
func redactQuery(path string) string {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		return path[:i] + "?[redacted]"
	}
	return path
}

// writePassthroughEnabled reports whether raw write methods are allowed through
// the MCP passthrough. Off by default: the typed manage_* tools cover common
// writes with their own guardrails, so unrestricted write passthrough would add
// an unaudited foot-gun (e.g. DELETE /repositories/x/y) that bypasses them.
func writePassthroughEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BBKT_API_ALLOW_WRITE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// APIRequestArgs is the input for the bitbucket_api passthrough tool.
type APIRequestArgs struct {
	Path     string `json:"path" jsonschema:"Bitbucket API v2 path, e.g. '/repositories/{workspace}/{repo}/pipelines'. A leading /2.0 and the api.bitbucket.org host are optional."`
	Method   string `json:"method,omitempty" jsonschema:"HTTP method (default GET)" jsonschema_enum:"GET,POST,PUT,PATCH,DELETE"`
	Body     string `json:"body,omitempty" jsonschema:"Raw JSON request body for POST/PUT/PATCH"`
	Paginate bool   `json:"paginate,omitempty" jsonschema:"Follow pagination and merge all 'values' (GET collections)"`
	MaxPages int    `json:"max_pages,omitempty" jsonschema:"Max pages when paginate is true (default 10; a cap is reported as truncated)"`
}

// APIRequestHandler backs bitbucket_api: an authenticated passthrough to any
// Bitbucket v2 endpoint. It exists so the model uses bbkt's authenticated client
// for endpoints without a typed tool, instead of reading credentials and calling
// the API itself. The real HTTP status is prepended so the model sees non-2xx
// responses (with the API's own error body) rather than a swallowed failure.
func APIRequestHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, APIRequestArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args APIRequestArgs) (*mcp.CallToolResult, any, error) {
		method := args.Method
		if method == "" {
			method = http.MethodGet
		}

		if isWriteMethod(method) && !writePassthroughEnabled() {
			fmt.Fprintf(os.Stderr, "[bitbucket_api] BLOCKED write %s %s (BBKT_API_ALLOW_WRITE not set)\n", strings.ToUpper(method), redactQuery(args.Path))
			return ToolResultError(fmt.Sprintf(
				"write method %s is disabled for bitbucket_api. Use a typed manage_* tool for writes when one fits; to allow raw write passthrough, set BBKT_API_ALLOW_WRITE=1 on the MCP server.",
				strings.ToUpper(method))), nil, nil
		}

		var body []byte
		if args.Body != "" {
			body = []byte(args.Body)
		}

		var status int
		var data []byte
		var err error
		if args.Paginate {
			maxPages := args.MaxPages
			if maxPages <= 0 {
				maxPages = 10 // cap MCP pagination so results don't flood context (also guards negative values)
			}
			status, data, err = c.RequestPaginated(method, args.Path, body, maxPages)
		} else {
			status, data, err = c.Request(method, args.Path, body)
		}

		// Log every passthrough call as telemetry — the set of paths hit here is
		// the backlog of endpoints worth wrapping with a typed tool. Redact the
		// query string, which may carry caller-supplied secrets.
		fmt.Fprintf(os.Stderr, "[bitbucket_api] %s %s -> %d\n", method, redactQuery(args.Path), status)

		if err != nil {
			return ToolResultError(fmt.Sprintf("request failed: %v", err)), nil, nil
		}
		payload := fmt.Sprintf("HTTP %d\n%s", status, string(data))
		if status >= 400 {
			return ToolResultError(payload), nil, nil
		}
		return ToolResultText(payload), nil, nil
	}
}
