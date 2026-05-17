package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var reposCmd = &cobra.Command{
	Use:     "repos",
	GroupID: groupData,
	Short:   "List, get, create, and delete Bitbucket repositories",
	Long: `Manage repositories in a Bitbucket workspace. Omit positional
workspace/repo args inside a Bitbucket git clone and they will be
inferred from .git/config.`,
	Example: `  bbkt repos list                       # repos in current workspace
  bbkt repos list my-team               # repos in a specific workspace
  bbkt repos get my-team my-repo
  bbkt repos create my-team new-repo --description "..."
  bbkt repos delete my-team old-repo`,
}

var reposListCmd = &cobra.Command{
	Use:   "list [workspace]",
	Short: "List repositories in a workspace (omit to infer from current git clone)",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, _, _, err := ParseArgs(cmd, args, -1)
		if err != nil {
			return err
		}

		query, _ := cmd.Flags().GetString("query")
		role, _ := cmd.Flags().GetString("role")
		sort, _ := cmd.Flags().GetString("sort")
		page, pagelen := paginationArgs(cmd)

		client := getClient()
		result, err := client.ListRepositories(bitbucket.ListRepositoriesArgs{
			Workspace: workspace,
			Query:     query,
			Role:      role,
			Sort:      sort,
			Page:      page,
			Pagelen:   pagelen,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No repositories found.")
				return
			}
			t := NewTable()
			t.Header("Full Name", "Language", "Visibility", "Updated")
			for _, r := range result.Values {
				lang := r.Language
				if lang == "" {
					lang = "-"
				}
				t.Row(r.FullName, lang, FormatPrivate(r.IsPrivate), FormatTime(r.UpdatedOn))
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
		return nil
	},
}

var reposGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug]",
	Short: "Get details for a specific repository (omit args to infer from git)",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}

		client := getClient()
		result, err := client.GetRepository(bitbucket.GetRepositoryArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Repository: %s\n", result.FullName)
			KV("Slug", result.Slug)
			KV("UUID", result.UUID)
			if result.Description != "" {
				KV("Description", Truncate(result.Description, 80))
			}
			lang := result.Language
			if lang == "" {
				lang = "-"
			}
			KV("Language", lang)
			KV("SCM", result.SCM)
			KV("Visibility", FormatPrivate(result.IsPrivate))
			if result.MainBranch != nil {
				KV("Main Branch", result.MainBranch.Name)
			}
			if result.Owner != nil {
				KV("Owner", result.Owner.DisplayName)
			}
			if result.Project != nil {
				KV("Project", result.Project.Key)
			}
			KV("Created", FormatTime(result.CreatedOn))
			KV("Updated", FormatTime(result.UpdatedOn))
		})
		return nil
	},
}

var reposCreateCmd = &cobra.Command{
	Use:   "create [workspace] [repo-slug]",
	Short: "Create a new repository",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}

		desc, _ := cmd.Flags().GetString("description")
		lang, _ := cmd.Flags().GetString("language")
		project, _ := cmd.Flags().GetString("project")

		isPrivatePtr := new(bool)
		*isPrivatePtr, _ = cmd.Flags().GetBool("private")

		client := getClient()
		result, err := client.CreateRepository(bitbucket.CreateRepositoryArgs{
			Workspace:   workspace,
			RepoSlug:    repoSlug,
			Description: desc,
			Language:    lang,
			ProjectKey:  project,
			IsPrivate:   isPrivatePtr,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Created repository: %s\n", result.FullName)
			KV("Slug", result.Slug)
			KV("Visibility", FormatPrivate(result.IsPrivate))
			if result.Language != "" {
				KV("Language", result.Language)
			}
			KV("Created", FormatTime(result.CreatedOn))
		})
		return nil
	},
}

var reposDeleteCmd = &cobra.Command{
	Use:   "delete [workspace] [repo-slug]",
	Short: "Delete a repository (destructive — no confirmation)",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}

		client := getClient()
		if err := client.DeleteRepository(bitbucket.DeleteRepositoryArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
		}); err != nil {
			return err
		}

		fmt.Printf("Repository '%s/%s' deleted successfully.\n", workspace, repoSlug)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(reposCmd)
	reposCmd.AddCommand(reposListCmd)
	reposCmd.AddCommand(reposGetCmd)
	reposCmd.AddCommand(reposCreateCmd)
	reposCmd.AddCommand(reposDeleteCmd)

	reposListCmd.Flags().StringP("query", "q", "", "Filter repositories using Bitbucket query syntax")
	reposListCmd.Flags().String("role", "", "Filter by role: owner | admin | contributor | member")
	reposListCmd.Flags().String("sort", "", "Sort field (e.g. -updated_on for newest first)")
	addPaginationFlags(reposListCmd)

	reposCreateCmd.Flags().String("description", "", "Repository description")
	reposCreateCmd.Flags().String("language", "", "Primary programming language")
	reposCreateCmd.Flags().String("project", "", "Project key to assign the repo to")
	reposCreateCmd.Flags().Bool("private", true, "Create as a private repository")
}
