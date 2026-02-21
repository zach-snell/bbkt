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

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Bitbucket MCP Server",
	Long: `Starts the Model Context Protocol (MCP) server for Bitbucket.
By default, this runs on stdio. You can provide a --port flag to
run it using the HTTP Streamable transport.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on for HTTP Streamable transport")
}

func runServer() {
	// Priority: env vars > stored credentials
	username := os.Getenv("BITBUCKET_USERNAME")
	password := os.Getenv("BITBUCKET_API_TOKEN")
	token := os.Getenv("BITBUCKET_ACCESS_TOKEN")

	var s *mcp.Server

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
