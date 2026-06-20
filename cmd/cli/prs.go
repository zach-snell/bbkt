package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var prsCmd = &cobra.Command{
	Use:     "prs",
	Aliases: []string{"pr"},
	GroupID: groupData,
	Short:   "Create, list, review, approve, merge, and decline pull requests",
	Long: `Manage pull requests. Workspace and repo are inferred from your
current git clone when omitted, so most commands can be run as
'bbkt prs <verb>' inside a Bitbucket repo.

Alias: pr`,
	Example: `  bbkt prs list                           # open PRs on current repo
  bbkt prs list --state MERGED            # filter by state
  bbkt prs get 42                         # details on PR #42
  bbkt prs create -t "Fix bug" -s feat/x
  bbkt prs merge 42 --strategy squash
  bbkt prs approve 42
  bbkt prs comments add 42 -m "LGTM"`,
}

var prsListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug]",
	Short: "List pull requests (omit workspace/repo to infer from git)",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}

		query, _ := cmd.Flags().GetString("query")
		state, _ := cmd.Flags().GetString("state")
		page, pagelen := paginationArgs(cmd)

		client := getClient()
		result, err := client.ListPullRequests(bitbucket.ListPullRequestsArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Query:     query,
			State:     state,
			Page:      page,
			Pagelen:   pagelen,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No pull requests found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Title", "State", "Author", "Source", "Updated")
			for _, pr := range result.Values {
				author := "-"
				if pr.Author != nil {
					author = pr.Author.DisplayName
				}
				branch := "-"
				if pr.Source.Branch != nil {
					branch = pr.Source.Branch.Name
				}
				title := Truncate(pr.Title, 50)
				t.Row(
					fmt.Sprintf("#%d", pr.ID),
					title,
					pr.State,
					author,
					branch,
					FormatTime(pr.UpdatedOn),
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
		return nil
	},
}

var prsGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug] <pr-id>",
	Short: "Get details for a specific pull request",
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

		client := getClient()
		result, err := client.GetPullRequest(bitbucket.GetPullRequestArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Pull Request #%d: %s\n", result.ID, result.Title)
			KV("State", result.State)
			if result.Draft {
				KV("Draft", "yes")
			}
			if result.Author != nil {
				KV("Author", result.Author.DisplayName)
			}
			if result.Source.Branch != nil {
				KV("Source", result.Source.Branch.Name)
			}
			if result.Destination.Branch != nil {
				KV("Destination", result.Destination.Branch.Name)
			}
			if result.Description != "" {
				KV("Description", Truncate(result.Description, 80))
			}
			KVf("Comments", "%d", result.CommentCount)
			KVf("Tasks", "%d", result.TaskCount)
			if len(result.Reviewers) > 0 {
				names := make([]string, len(result.Reviewers))
				for i, r := range result.Reviewers {
					names[i] = r.DisplayName
				}
				KV("Reviewers", strings.Join(names, ", "))
			}
			KV("Close Branch", FormatBool(result.CloseSourceBranch))
			KV("Created", FormatTime(result.CreatedOn))
			KV("Updated", FormatTime(result.UpdatedOn))
		})
		return nil
	},
}

var prsCreateCmd = &cobra.Command{
	Use:   "create [workspace] [repo-slug]",
	Short: "Create a new pull request (prompts interactively if --title/--source missing)",
	Args:  cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}

		title, _ := cmd.Flags().GetString("title")
		source, _ := cmd.Flags().GetString("source")
		dest, _ := cmd.Flags().GetString("destination")
		desc, _ := cmd.Flags().GetString("description")
		closeSource, _ := cmd.Flags().GetBool("close-source-branch")
		draft, _ := cmd.Flags().GetBool("draft")

		interactive := false
		if title == "" || source == "" {
			interactive = true
			fmt.Println("Missing required arguments. Entering interactive mode...")

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Title").
						Value(&title).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("title is required")
							}
							return nil
						}),
					huh.NewInput().
						Title("Source Branch").
						Value(&source).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("source branch is required")
							}
							return nil
						}),
					huh.NewInput().
						Title("Destination Branch (optional)").
						Value(&dest),
					huh.NewText().
						Title("Description (optional)").
						Value(&desc),
					huh.NewConfirm().
						Title("Close source branch after merge?").
						Value(&closeSource),
					huh.NewConfirm().
						Title("Create as draft?").
						Value(&draft),
				),
			)
			if err := form.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Draft PR creation cancelled.")
				return err
			}
		}

		if title == "" || source == "" {
			return fmt.Errorf("title and source branch are required")
		}

		if interactive {
			fmt.Println("Creating pull request...")
		}

		client := getClient()
		result, err := client.CreatePullRequest(bitbucket.CreatePullRequestArgs{
			Workspace:         workspace,
			RepoSlug:          repoSlug,
			Title:             title,
			SourceBranch:      source,
			DestinationBranch: dest,
			Description:       desc,
			CloseSourceBranch: closeSource,
			Draft:             draft,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Created Pull Request #%d: %s\n", result.ID, result.Title)
			KV("State", result.State)
			if result.Source.Branch != nil {
				KV("Source", result.Source.Branch.Name)
			}
			if result.Destination.Branch != nil {
				KV("Destination", result.Destination.Branch.Name)
			}
			if result.Draft {
				KV("Draft", "yes")
			}
			KV("Created", FormatTime(result.CreatedOn))
		})
		return nil
	},
}

var prsMergeCmd = &cobra.Command{
	Use:   "merge [workspace] [repo-slug] <pr-id>",
	Short: "Merge a pull request",
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

		strategy, _ := cmd.Flags().GetString("strategy")
		msg, _ := cmd.Flags().GetString("message")
		closeSource, _ := cmd.Flags().GetBool("close-source-branch")

		client := getClient()
		result, err := client.MergePullRequest(bitbucket.MergePullRequestArgs{
			Workspace:         workspace,
			RepoSlug:          repoSlug,
			PRID:              prID,
			MergeStrategy:     strategy,
			Message:           msg,
			CloseSourceBranch: closeSource,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Merged Pull Request #%d: %s\n", result.ID, result.Title)
			KV("State", result.State)
			if result.MergeCommit != nil {
				KV("Merge Commit", result.MergeCommit.Hash[:12])
			}
			KV("Updated", FormatTime(result.UpdatedOn))
		})
		return nil
	},
}

var prsApproveCmd = &cobra.Command{
	Use:   "approve [workspace] [repo-slug] <pr-id>",
	Short: "Approve a pull request",
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

		client := getClient()
		if err := client.ApprovePullRequest(bitbucket.PullRequestActionArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
		}); err != nil {
			return err
		}

		PrintOrJSON(cmd, map[string]any{"id": prID, "approved": true}, func() {
			fmt.Printf("Pull request #%d approved successfully.\n", prID)
		})
		return nil
	},
}

var prsDeclineCmd = &cobra.Command{
	Use:   "decline [workspace] [repo-slug] <pr-id>",
	Short: "Decline a pull request",
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

		client := getClient()
		if err := client.DeclinePullRequest(bitbucket.PullRequestActionArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
		}); err != nil {
			return err
		}

		PrintOrJSON(cmd, map[string]any{"id": prID, "declined": true}, func() {
			fmt.Printf("Pull request #%d declined successfully.\n", prID)
		})
		return nil
	},
}

func init() {
	RootCmd.AddCommand(prsCmd)
	prsCmd.AddCommand(prsListCmd)
	prsCmd.AddCommand(prsGetCmd)
	prsCmd.AddCommand(prsCreateCmd)
	prsCmd.AddCommand(prsMergeCmd)
	prsCmd.AddCommand(prsApproveCmd)
	prsCmd.AddCommand(prsDeclineCmd)

	prsListCmd.Flags().StringP("query", "q", "", "Filter pull requests using Bitbucket query syntax")
	prsListCmd.Flags().String("state", "OPEN", "Filter by state: OPEN | MERGED | SUPERSEDED | DECLINED")
	addPaginationFlags(prsListCmd)

	prsCreateCmd.Flags().StringP("title", "t", "", "Title of the pull request")
	prsCreateCmd.Flags().StringP("source", "s", "", "Source branch name")
	prsCreateCmd.Flags().StringP("destination", "d", "", "Destination branch name (defaults to repo default branch)")
	prsCreateCmd.Flags().String("description", "", "Description of the pull request (markdown supported)")
	prsCreateCmd.Flags().Bool("close-source-branch", true, "Close source branch on merge")
	prsCreateCmd.Flags().Bool("draft", false, "Create as a draft PR")

	prsMergeCmd.Flags().String("strategy", "merge_commit", "Merge strategy: merge_commit | squash | fast_forward")
	prsMergeCmd.Flags().StringP("message", "m", "", "Commit message for the merge commit")
	prsMergeCmd.Flags().Bool("close-source-branch", true, "Close source branch after merge")
}
