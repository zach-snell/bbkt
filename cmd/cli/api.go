package cli

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:     "api <path> [METHOD path]",
	GroupID: groupData,
	Short:   "Make an authenticated Bitbucket API request (escape hatch for endpoints without a typed command)",
	Long: `Send an authenticated request to the Bitbucket Cloud REST API v2 using your
active bbkt profile, and print the raw JSON response.

Use this for endpoints bbkt doesn't wrap with a typed command — instead of
digging up your token and calling the API by hand. Auth, token refresh, and the
host stay inside bbkt; your credentials are never exposed to the caller.

The path is forgiving: a leading "/2.0", a bare path, or a full
https://api.bitbucket.org/... URL all work. The default method is GET; set
another with -X or by passing "METHOD path" positionally.`,
	Example: `  bbkt api /user
  bbkt api /repositories/myws/myrepo/pipelines --paginate
  bbkt api -X POST /repositories/myws/myrepo/pullrequests -d @pr.json
  bbkt api DELETE /repositories/myws/myrepo/pullrequests/42/approve`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		method, _ := cmd.Flags().GetString("method")
		path := args[0]

		// Forgiving positional form: `bbkt api POST /path`. If the first arg is a
		// bare HTTP method and a second arg is present, treat it as the method.
		if len(args) == 2 {
			if isHTTPMethod(args[0]) {
				method = strings.ToUpper(args[0])
				path = args[1]
			} else {
				return fmt.Errorf("with two positional args the first must be an HTTP method (got %q)", args[0])
			}
		}
		if method == "" {
			method = http.MethodGet
		}

		body, err := readAPIBody(cmd)
		if err != nil {
			return err
		}

		client := getClient()
		paginate, _ := cmd.Flags().GetBool("paginate")
		maxPages, _ := cmd.Flags().GetInt("max-pages")

		var status int
		var data []byte
		if paginate {
			status, data, err = client.RequestPaginated(method, path, body, maxPages)
		} else {
			status, data, err = client.Request(method, path, body)
		}
		if err != nil {
			return err
		}

		os.Stdout.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			fmt.Println()
		}
		if status >= 400 {
			// Body (the real API error) is already on stdout; signal failure too.
			return fmt.Errorf("bitbucket API returned HTTP %d", status)
		}
		return nil
	},
}

// readAPIBody resolves the --data flag: a literal string, or @file to read the
// body from a file ("@-" reads stdin).
func readAPIBody(cmd *cobra.Command) ([]byte, error) {
	d, _ := cmd.Flags().GetString("data")
	if d == "" {
		return nil, nil
	}
	if strings.HasPrefix(d, "@") {
		name := d[1:]
		if name == "-" {
			return os.ReadFile("/dev/stdin")
		}
		b, err := os.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("reading --data file: %w", err)
		}
		return b, nil
	}
	return []byte(d), nil
}

func isHTTPMethod(s string) bool {
	switch strings.ToUpper(s) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	}
	return false
}

func init() {
	apiCmd.Flags().StringP("method", "X", "", "HTTP method (default GET)")
	apiCmd.Flags().StringP("data", "d", "", "Request body as a string, or @file (@- for stdin)")
	apiCmd.Flags().Bool("paginate", false, "Follow pagination and merge all 'values' (GET collections)")
	apiCmd.Flags().Int("max-pages", 0, "Max pages when --paginate (0 = all, hard-capped at 100; a cap is reported as truncated)")
	RootCmd.AddCommand(apiCmd)
}
