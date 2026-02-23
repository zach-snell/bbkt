package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var issuesCmd = &cobra.Command{
	Use:     "issues",
	Aliases: []string{"issue"},
	Short:   "Manage Bitbucket issues",
}

var issuesListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug]",
	Short: "List issues in a repository",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		state, _ := cmd.Flags().GetString("state")
		kind, _ := cmd.Flags().GetString("kind")
		priority, _ := cmd.Flags().GetString("priority")
		search, _ := cmd.Flags().GetString("search")
		sort, _ := cmd.Flags().GetString("sort")

		client := getClient()
		result, err := client.ListIssues(bitbucket.ListIssuesArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			State:     state,
			Kind:      kind,
			Priority:  priority,
			Search:    search,
			Sort:      sort,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No issues found.")
				return
			}
			t := NewTable()
			t.Header("ID", "Title", "State", "Kind", "Priority", "Assignee")
			for _, issue := range result.Values {
				assignee := "-"
				if issue.Assignee != nil {
					assignee = issue.Assignee.DisplayName
				}
				t.Row(
					fmt.Sprintf("#%d", issue.ID),
					Truncate(issue.Title, 45),
					issue.State,
					issue.Kind,
					issue.Priority,
					assignee,
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
	},
}

var issuesGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug] [issue-id]",
	Short: "Get details for a specific issue",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		issueID, err := strconv.Atoi(trailing[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid Issue ID: %s\n", trailing[0])
			os.Exit(1)
		}

		client := getClient()
		result, err := client.GetIssue(bitbucket.GetIssueArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			IssueID:   issueID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Issue #%d: %s\n", result.ID, result.Title)
			KV("State", result.State)
			KV("Kind", result.Kind)
			KV("Priority", result.Priority)
			if result.Assignee != nil {
				KV("Assignee", result.Assignee.DisplayName)
			} else {
				KV("Assignee", "unassigned")
			}
			if result.Reporter != nil {
				KV("Reporter", result.Reporter.DisplayName)
			}
			if result.Content.Raw != "" {
				KV("Description", Truncate(result.Content.Raw, 80))
			}
			KVf("Votes", "%d", result.Votes)
			KVf("Watches", "%d", result.Watches)
			KV("Created", FormatTime(result.CreatedOn))
			KV("Updated", FormatTime(result.UpdatedOn))
		})
	},
}

var issuesCreateCmd = &cobra.Command{
	Use:   "create [workspace] [repo-slug]",
	Short: "Create a new issue",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		title, _ := cmd.Flags().GetString("title")
		content, _ := cmd.Flags().GetString("content")
		kind, _ := cmd.Flags().GetString("kind")
		priority, _ := cmd.Flags().GetString("priority")
		assignee, _ := cmd.Flags().GetString("assignee")

		if title == "" {
			fmt.Fprintln(os.Stderr, "Error: title is required")
			os.Exit(1)
		}

		client := getClient()
		result, err := client.CreateIssue(bitbucket.CreateIssueArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Title:     title,
			Content:   content,
			Kind:      kind,
			Priority:  priority,
			Assignee:  assignee,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Created Issue #%d: %s\n", result.ID, result.Title)
			KV("Kind", result.Kind)
			KV("Priority", result.Priority)
			KV("State", result.State)
			KV("Created", FormatTime(result.CreatedOn))
		})
	},
}

var issuesUpdateCmd = &cobra.Command{
	Use:   "update [workspace] [repo-slug] [issue-id]",
	Short: "Update an existing issue",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		issueID, err := strconv.Atoi(trailing[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid Issue ID: %s\n", trailing[0])
			os.Exit(1)
		}

		updateArgs := bitbucket.UpdateIssueArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			IssueID:   issueID,
		}

		// Only pass flags that were explicitly set
		if cmd.Flags().Changed("title") {
			v, _ := cmd.Flags().GetString("title")
			updateArgs.Title = &v
		}
		if cmd.Flags().Changed("content") {
			v, _ := cmd.Flags().GetString("content")
			updateArgs.Content = &v
		}
		if cmd.Flags().Changed("state") {
			v, _ := cmd.Flags().GetString("state")
			updateArgs.State = &v
		}
		if cmd.Flags().Changed("kind") {
			v, _ := cmd.Flags().GetString("kind")
			updateArgs.Kind = &v
		}
		if cmd.Flags().Changed("priority") {
			v, _ := cmd.Flags().GetString("priority")
			updateArgs.Priority = &v
		}
		if cmd.Flags().Changed("assignee") {
			v, _ := cmd.Flags().GetString("assignee")
			updateArgs.Assignee = &v
		}

		client := getClient()
		result, err := client.UpdateIssue(updateArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Updated Issue #%d: %s\n", result.ID, result.Title)
			KV("State", result.State)
			KV("Kind", result.Kind)
			KV("Priority", result.Priority)
			if result.Assignee != nil {
				KV("Assignee", result.Assignee.DisplayName)
			}
			KV("Updated", FormatTime(result.UpdatedOn))
		})
	},
}

func init() {
	RootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesGetCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesUpdateCmd)

	issuesListCmd.Flags().String("state", "", "Filter by state (new, open, resolved, on hold, invalid, duplicate, wontfix, closed)")
	issuesListCmd.Flags().String("kind", "", "Filter by kind (bug, enhancement, proposal, task)")
	issuesListCmd.Flags().String("priority", "", "Filter by priority (trivial, minor, major, critical, blocker)")
	issuesListCmd.Flags().StringP("search", "q", "", "Search query string")
	issuesListCmd.Flags().String("sort", "", "Sort field (e.g. -updated_on)")

	issuesCreateCmd.Flags().StringP("title", "t", "", "Title of the issue (required)")
	issuesCreateCmd.Flags().StringP("content", "m", "", "Description of the issue (markdown)")
	issuesCreateCmd.Flags().String("kind", "bug", "Kind: bug, enhancement, proposal, task")
	issuesCreateCmd.Flags().String("priority", "major", "Priority: trivial, minor, major, critical, blocker")
	issuesCreateCmd.Flags().String("assignee", "", "Assignee account ID")

	issuesUpdateCmd.Flags().StringP("title", "t", "", "New title for the issue")
	issuesUpdateCmd.Flags().StringP("content", "m", "", "New description of the issue")
	issuesUpdateCmd.Flags().String("state", "", "New state (resolved, closed, etc)")
	issuesUpdateCmd.Flags().String("kind", "", "New kind")
	issuesUpdateCmd.Flags().String("priority", "", "New priority")
	issuesUpdateCmd.Flags().String("assignee", "", "New assignee account ID (or 'unassign')")
}
