package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var sourceCmd = &cobra.Command{
	Use:     "source",
	Aliases: []string{"src"},
	Short:   "Read and search repository source code",
}

var sourceReadCmd = &cobra.Command{
	Use:   "read [workspace] [repo-slug] [path]",
	Short: "Get the raw content of a file",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ref, _ := cmd.Flags().GetString("ref")

		client := getClient()
		content, _, err := client.GetFileContent(bitbucket.GetFileContentArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Path:      trailing[0],
			Ref:       ref,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Print raw content directly
		fmt.Print(string(content))
	},
}

var sourceTreeCmd = &cobra.Command{
	Use:   "tree [workspace] [repo-slug] [path]",
	Short: "List files and directories",
	Args:  cobra.RangeArgs(0, 3), // Path is optional, workspace/repo are optional
	Run: func(cmd *cobra.Command, args []string) {
		// Because path is optional, if len(args) == 0: git, path=""
		// If len(args) == 1: git, path=args[0]
		// If len(args) == 2: args[0]=ws, args[1]=rs, path=""
		// If len(args) == 3: args[0]=ws, args[1]=rs, path=args[2]

		var workspace, repoSlug, path string

		if len(args) == 0 || len(args) == 1 {
			ws, rs, err := bitbucket.GetLocalRepoInfo()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			workspace = ws
			repoSlug = rs
			if len(args) == 1 {
				path = args[0]
			}
		} else if len(args) == 2 || len(args) == 3 {
			workspace = args[0]
			repoSlug = args[1]
			if len(args) == 3 {
				path = args[2]
			}
		}

		ref, _ := cmd.Flags().GetString("ref")
		maxDepth, _ := cmd.Flags().GetInt("max-depth")

		client := getClient()
		result, err := client.ListDirectory(bitbucket.ListDirectoryArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Path:      path,
			Ref:       ref,
			MaxDepth:  maxDepth,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No entries found.")
				return
			}
			t := NewTable()
			t.Header("Type", "Path", "Size")
			for _, entry := range result.Values {
				entryType := "file"
				if entry.Type == "commit_directory" {
					entryType = "dir"
				}
				size := "-"
				if entry.Size > 0 {
					size = formatFileSize(entry.Size)
				}
				t.Row(entryType, entry.Path, size)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
	},
}

var sourceHistoryCmd = &cobra.Command{
	Use:   "history [workspace] [repo-slug] [path]",
	Short: "Get the commit history for a specific file",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ref, _ := cmd.Flags().GetString("ref")

		client := getClient()
		result, err := client.GetFileHistory(bitbucket.GetFileHistoryArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Path:      trailing[0],
			Ref:       ref,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No history found.")
				return
			}
			t := NewTable()
			t.Header("Hash", "Author", "Date", "Message")
			for _, raw := range result.Values {
				// Parse each raw commit JSON
				var commit struct {
					Hash    string `json:"hash"`
					Message string `json:"message"`
					Date    string `json:"date"`
					Author  struct {
						Raw  string `json:"raw"`
						User *struct {
							DisplayName string `json:"display_name"`
						} `json:"user"`
					} `json:"author"`
				}
				if err := json.Unmarshal(raw, &commit); err != nil {
					continue
				}
				hash := commit.Hash
				if len(hash) > 10 {
					hash = hash[:10]
				}
				author := commit.Author.Raw
				if commit.Author.User != nil {
					author = commit.Author.User.DisplayName
				}
				msg := Truncate(commit.Message, 50)
				t.Row(hash, author, commit.Date[:10], msg)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
	},
}

var sourceSearchCmd = &cobra.Command{
	Use:   "search [workspace] [repo-slug] [query]",
	Short: "Search for code in a repository",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		result, err := client.SearchCode(bitbucket.SearchCodeArgs{
			Workspace:   workspace,
			RepoSlug:    repoSlug,
			SearchQuery: trailing[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Bitbucket returns custom JSON structure for code search
		fmt.Println(string(result))
	},
}

var sourceWriteCmd = &cobra.Command{
	Use:   "write [workspace] [repo-slug] [path]",
	Short: "Write or update a file in the repository",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		content, _ := cmd.Flags().GetString("content")
		if content == "" {
			fmt.Fprintln(os.Stderr, "Error: content is required")
			os.Exit(1)
		}

		message, _ := cmd.Flags().GetString("message")
		branch, _ := cmd.Flags().GetString("branch")
		author, _ := cmd.Flags().GetString("author")

		client := getClient()
		err = client.WriteFile(bitbucket.WriteFileArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Path:      trailing[0],
			Content:   content,
			Message:   message,
			Branch:    branch,
			Author:    author,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully wrote file '%s'\n", trailing[0])
	},
}

var sourceDeleteCmd = &cobra.Command{
	Use:   "delete [workspace] [repo-slug] [path]",
	Short: "Delete a file from the repository",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		message, _ := cmd.Flags().GetString("message")
		branch, _ := cmd.Flags().GetString("branch")
		author, _ := cmd.Flags().GetString("author")

		client := getClient()
		err = client.DeleteFile(bitbucket.DeleteFileArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Path:      trailing[0],
			Message:   message,
			Branch:    branch,
			Author:    author,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully deleted file '%s'\n", trailing[0])
	},
}

func init() {
	RootCmd.AddCommand(sourceCmd)
	sourceCmd.AddCommand(sourceReadCmd)
	sourceCmd.AddCommand(sourceTreeCmd)
	sourceCmd.AddCommand(sourceHistoryCmd)
	sourceCmd.AddCommand(sourceSearchCmd)
	sourceCmd.AddCommand(sourceWriteCmd)
	sourceCmd.AddCommand(sourceDeleteCmd)

	sourceReadCmd.Flags().String("ref", "", "Commit hash, branch, or tag (default: HEAD)")

	sourceTreeCmd.Flags().String("ref", "", "Commit hash, branch, or tag (default: HEAD)")
	sourceTreeCmd.Flags().Int("max-depth", 1, "Maximum depth of recursion")

	sourceHistoryCmd.Flags().String("ref", "", "Commit hash, branch, or tag (default: HEAD)")

	sourceWriteCmd.Flags().StringP("content", "c", "", "Content to write to the file")
	sourceWriteCmd.Flags().StringP("message", "m", "", "Commit message")
	sourceWriteCmd.Flags().StringP("branch", "b", "", "Branch to commit to")
	sourceWriteCmd.Flags().StringP("author", "a", "", "Commit author in 'Name <email>' format")

	sourceDeleteCmd.Flags().StringP("message", "m", "", "Commit message")
	sourceDeleteCmd.Flags().StringP("branch", "b", "", "Branch to commit to")
	sourceDeleteCmd.Flags().StringP("author", "a", "", "Commit author in 'Name <email>' format")
}

// formatFileSize formats bytes into human-readable size.
func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fK", kb)
	}
	mb := kb / 1024
	return fmt.Sprintf("%.1fM", mb)
}
