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
	Short:   "Manage Bitbucket pull requests",
}

var prsListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug]",
	Short: "List pull requests in a repository (omit workspace/repo to infer from git)",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		query, _ := cmd.Flags().GetString("query")
		state, _ := cmd.Flags().GetString("state")

		client := getClient()
		result, err := client.ListPullRequests(bitbucket.ListPullRequestsArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Query:     query,
			State:     state,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintJSON(result)
	},
}

var prsGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug] [pr-id]",
	Short: "Get details for a specific pull request",
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
		result, err := client.GetPullRequest(bitbucket.GetPullRequestArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintJSON(result)
	},
}

var prsCreateCmd = &cobra.Command{
	Use:   "create [workspace] [repo-slug]",
	Short: "Create a new pull request",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
			err := form.Run()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Draft PR creation cancelled.")
				os.Exit(1)
			}
		}

		if title == "" || source == "" {
			fmt.Fprintln(os.Stderr, "Error: title and source branch are required")
			os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintJSON(result)
	},
}

var prsMergeCmd = &cobra.Command{
	Use:   "merge [workspace] [repo-slug] [pr-id]",
	Short: "Merge a pull request",
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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintJSON(result)
	},
}

var prsApproveCmd = &cobra.Command{
	Use:   "approve [workspace] [repo-slug] [pr-id]",
	Short: "Approve a pull request",
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
		err = client.ApprovePullRequest(bitbucket.PullRequestActionArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Pull request #%d approved successfully.\n", prID)
	},
}

var prsDeclineCmd = &cobra.Command{
	Use:   "decline [workspace] [repo-slug] [pr-id]",
	Short: "Decline a pull request",
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
		err = client.DeclinePullRequest(bitbucket.PullRequestActionArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			PRID:      prID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Pull request #%d declined successfully.\n", prID)
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
	prsListCmd.Flags().String("state", "OPEN", "Filter by state (MERGED, SUPERSEDED, OPEN, DECLINED)")

	prsCreateCmd.Flags().StringP("title", "t", "", "Title of the pull request (required)")
	prsCreateCmd.Flags().StringP("source", "s", "", "Source branch name (required)")
	prsCreateCmd.Flags().StringP("destination", "d", "", "Destination branch name (optional, defaults to repo default)")
	prsCreateCmd.Flags().String("description", "", "Description of the pull request")
	prsCreateCmd.Flags().Bool("close-source-branch", true, "Close source branch on merge")
	prsCreateCmd.Flags().Bool("draft", false, "Create as a draft PR")

	prsMergeCmd.Flags().String("strategy", "merge_commit", "Merge strategy (merge_commit, squash, fast_forward)")
	prsMergeCmd.Flags().StringP("message", "m", "", "Commit message")
	prsMergeCmd.Flags().Bool("close-source-branch", true, "Close source branch")
}
