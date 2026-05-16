package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
	mcpserver "github.com/zach-snell/bbkt/internal/mcp"
)

var port int
var noAuth bool

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Bitbucket MCP Server (for Claude Desktop, Cursor, etc.)",
	Long: `Starts the Model Context Protocol (MCP) server for Bitbucket.

By default the server speaks stdio — the right mode for local MCP
clients (Claude Desktop, Cursor). Use --port to switch to the HTTP
Streamable transport for remote/network clients.

Credentials are resolved in this order: BITBUCKET_ACCESS_TOKEN env,
then BITBUCKET_USERNAME + BITBUCKET_API_TOKEN env, then the stored
profile at ~/.config/bbkt/credentials.json (selected by --profile,
BBKT_PROFILE, or active_profile).

At startup the server introspects the token's granted scopes and
silently drops tools the token can't use, so the AI agent never
sees write tools it would fail to call. Use BITBUCKET_DISABLED_TOOLS
to explicitly disable additional tools.`,
	Example: `  bbkt mcp                              # stdio (default)
  bbkt mcp --port 8080                  # HTTP Streamable on :8080
  bbkt --profile work mcp               # use the "work" profile
  bbkt mcp --no-auth                    # start without creds (tools return auth-required)`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on for HTTP Streamable transport")
	mcpCmd.Flags().BoolVar(&noAuth, "no-auth", false, "Start server without authentication (tools will return auth-required errors when called)")
}

func runServer() {
	var s *mcp.Server

	if noAuth {
		s = mcpserver.NewUnauthenticated()
	} else {
		// Priority: env vars > stored credentials
		username := os.Getenv("BITBUCKET_USERNAME")
		password := os.Getenv("BITBUCKET_API_TOKEN")
		token := os.Getenv("BITBUCKET_ACCESS_TOKEN")

		if token != "" || (username != "" && password != "") {
			s = mcpserver.New(username, password, token)
		} else {
			creds, err := bitbucket.LoadCredentials()
			if err != nil {
				fmt.Fprintf(os.Stderr, "No credentials found. Either:\n")
				fmt.Fprintf(os.Stderr, "  1. Run: bbkt auth          (API token — recommended)\n")
				fmt.Fprintf(os.Stderr, "  2. Run: bbkt auth --oauth   (OAuth via browser)\n")
				fmt.Fprintf(os.Stderr, "  3. Set BITBUCKET_ACCESS_TOKEN env var\n")
				fmt.Fprintf(os.Stderr, "  4. Set BITBUCKET_USERNAME + BITBUCKET_API_TOKEN env vars\n")
				os.Exit(1)
			}

			switch {
			case creds.IsAPIToken() || creds.IsOAuth():
				s = mcpserver.NewFromCredentials(creds)
			default:
				fmt.Fprintf(os.Stderr, "Unknown auth type in stored credentials: %s\n", creds.AuthType)
				os.Exit(1)
			}
		}
	}

	if port != 0 {
		fmt.Printf("Starting Bitbucket MCP Server on :%d (HTTP Streamable)\n", port)
		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return s
		}, &mcp.StreamableHTTPOptions{JSONResponse: false})

		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 3 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}
}
