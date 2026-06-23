package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var prCommentsCmd = &cobra.Command{
	Use:     "comments",
	Aliases: []string{"comment"},
	Short:   "List, add, and resolve pull request comments",
	Long: `Manage comments on a pull request. Pass --file with --to (new
file line) or --from (old file line) for inline diff comments;
without those, the comment attaches to the PR overview. Use
--parent to reply to an existing comment ID.

Alias: comment`,
	Example: `  bbkt prs comments list 42
  bbkt prs comments add 42 -m "Looks good"
  bbkt prs comments add 42 -m "Nit" --file src/main.go --to 17
  bbkt prs comments add 42 -m "Reply" --parent 9876
  bbkt prs comments resolve 42 9876`,
}

var prCommentsListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug] <pr-id>",
	Short: "List comments on a pull request",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 1)
		if err != nil {
			return err
		}

		prID, err := strconv.Atoi(trailing[0])
		if err != nil {
			return fmt.Errorf("invalid PR ID %q (must be a number)", trailing[0])
		}

		page, pagelen := paginationArgs(cmd)
		client := getClient()
		result, err := client.ListPRComments(bitbucket.ListPRCommentsArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
			Page:      page,
			Pagelen:   pagelen,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No comments found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Author", "Resolved", "Content", "Created")
			for _, c := range result.Values {
				author := "-"
				if c.User != nil {
					author = c.User.DisplayName
				}
				content := Truncate(c.Content.Raw, 50)
				if c.Deleted {
					content = "(deleted)"
				}
				resolved := "-"
				if c.Resolution != nil {
					resolved = "✓"
					if c.Resolution.User != nil && c.Resolution.User.DisplayName != "" {
						resolved = "✓ " + c.Resolution.User.DisplayName
					}
				}
				t.Row(
					fmt.Sprintf("%d", c.ID),
					author,
					resolved,
					content,
					FormatTime(c.CreatedOn),
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
		return nil
	},
}

var prCommentsAddCmd = &cobra.Command{
	Use:   "add [workspace] [repo-slug] <pr-id>",
	Short: "Add a comment to a pull request",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 1)
		if err != nil {
			return err
		}

		prID, err := strconv.Atoi(trailing[0])
		if err != nil {
			return fmt.Errorf("invalid PR ID %q (must be a number)", trailing[0])
		}

		content, _ := cmd.Flags().GetString("content")
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
			return err
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
		return nil
	},
}

var prCommentsResolveCmd = &cobra.Command{
	Use:   "resolve [workspace] [repo-slug] <pr-id> <comment-id>",
	Short: "Resolve a comment thread",
	Args:  cobra.RangeArgs(2, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 2)
		if err != nil {
			return err
		}

		prID, err := strconv.Atoi(trailing[0])
		if err != nil {
			return fmt.Errorf("invalid PR ID %q (must be a number)", trailing[0])
		}

		commentID, err := strconv.Atoi(trailing[1])
		if err != nil {
			return fmt.Errorf("invalid comment ID %q (must be a number)", trailing[1])
		}

		client := getClient()
		if err := client.ResolvePRComment(bitbucket.CommentActionArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
			CommentID: commentID,
		}); err != nil {
			return err
		}

		fmt.Printf("Comment thread %d resolved successfully.\n", commentID)
		return nil
	},
}

func init() {
	prsCmd.AddCommand(prCommentsCmd)
	prCommentsCmd.AddCommand(prCommentsListCmd)
	prCommentsCmd.AddCommand(prCommentsAddCmd)
	prCommentsCmd.AddCommand(prCommentsResolveCmd)

	addPaginationFlags(prCommentsListCmd)

	prCommentsAddCmd.Flags().StringP("content", "m", "", "Comment body (markdown supported)")
	prCommentsAddCmd.Flags().Int("parent", 0, "Reply to this comment ID (creates a threaded reply)")
	prCommentsAddCmd.Flags().String("file", "", "File path for an inline diff comment (pair with --to or --from)")
	prCommentsAddCmd.Flags().Int("to", 0, "Line in the new file for additions/context (requires --file)")
	prCommentsAddCmd.Flags().Int("from", 0, "Line in the old file for deletions (requires --file)")
	_ = prCommentsAddCmd.MarkFlagRequired("content")
}
