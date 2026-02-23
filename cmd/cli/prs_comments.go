package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var prCommentsCmd = &cobra.Command{
	Use:     "comments",
	Aliases: []string{"comment"},
	Short:   "Manage pull request comments",
}

var prCommentsListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug] [pr-id]",
	Short: "List comments on a pull request",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		prID, err := strconv.Atoi(trailing[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid PR ID: %s\n", trailing[0])
			os.Exit(1)
		}

		client := getClient()
		result, err := client.ListPRComments(bitbucket.ListPRCommentsArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No comments found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Author", "Content", "Created")
			for _, c := range result.Values {
				author := "-"
				if c.User != nil {
					author = c.User.DisplayName
				}
				content := Truncate(c.Content.Raw, 50)
				if c.Deleted {
					content = "(deleted)"
				}
				t.Row(
					fmt.Sprintf("%d", c.ID),
					author,
					content,
					FormatTime(c.CreatedOn),
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
	},
}

var prCommentsAddCmd = &cobra.Command{
	Use:   "add [workspace] [repo-slug] [pr-id]",
	Short: "Add a comment to a pull request",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		prID, err := strconv.Atoi(trailing[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid PR ID: %s\n", trailing[0])
			os.Exit(1)
		}

		content, _ := cmd.Flags().GetString("content")
		if content == "" {
			fmt.Fprintln(os.Stderr, "Error: content is required")
			os.Exit(1)
		}

		parentID, _ := cmd.Flags().GetInt("parent")
		file, _ := cmd.Flags().GetString("file")
		toCode, _ := cmd.Flags().GetInt("to")
		fromCode, _ := cmd.Flags().GetInt("from")

		client := getClient()
		result, err := client.CreatePRComment(bitbucket.CreatePRCommentArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
			Content:   content,
			ParentID:  parentID,
			FilePath:  file,
			LineTo:    toCode,
			LineFrom:  fromCode,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Added comment #%d\n", result.ID)
			if result.User != nil {
				KV("Author", result.User.DisplayName)
			}
			KV("Content", Truncate(result.Content.Raw, 80))
			if result.Inline != nil {
				KV("File", result.Inline.Path)
			}
			KV("Created", FormatTime(result.CreatedOn))
		})
	},
}

var prCommentsResolveCmd = &cobra.Command{
	Use:   "resolve [workspace] [repo-slug] [pr-id] [comment-id]",
	Short: "Resolve a comment thread",
	Args:  cobra.RangeArgs(2, 4),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		prID, err := strconv.Atoi(trailing[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid PR ID: %s\n", trailing[0])
			os.Exit(1)
		}

		commentID, err := strconv.Atoi(trailing[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid Comment ID: %s\n", trailing[1])
			os.Exit(1)
		}

		client := getClient()
		err = client.ResolvePRComment(bitbucket.CommentActionArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
			CommentID: commentID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Comment thread %d resolved successfully.\n", commentID)
	},
}

func init() {
	prsCmd.AddCommand(prCommentsCmd)
	prCommentsCmd.AddCommand(prCommentsListCmd)
	prCommentsCmd.AddCommand(prCommentsAddCmd)
	prCommentsCmd.AddCommand(prCommentsResolveCmd)

	prCommentsAddCmd.Flags().StringP("content", "m", "", "Comment content (markdown supported)")
	prCommentsAddCmd.Flags().Int("parent", 0, "Parent comment ID to reply to")
	prCommentsAddCmd.Flags().String("file", "", "File path for inline comments")
	prCommentsAddCmd.Flags().Int("to", 0, "Line number the comment applies to (for new or modified lines)")
	prCommentsAddCmd.Flags().Int("from", 0, "Line number the comment applies to (for deleted lines)")
}
