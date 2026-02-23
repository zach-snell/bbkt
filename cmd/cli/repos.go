package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Manage Bitbucket repositories",
}

var reposListCmd = &cobra.Command{
	Use:   "list [workspace]",
	Short: "List repositories in a workspace",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var workspace string
		if len(args) == 1 {
			workspace = args[0]
		} else {
			ws, _, _, err := ParseArgs(args, -1) // -1 means we just want workspace
			if err != nil {
				// Fallback to just GetLocalRepoInfo
				w, _, err := bitbucket.GetLocalRepoInfo()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: must specify a workspace or run inside a git repo\n")
					os.Exit(1)
				}
				workspace = w
			} else {
				workspace = ws
			}
		}

		query, _ := cmd.Flags().GetString("query")
		role, _ := cmd.Flags().GetString("role")
		sort, _ := cmd.Flags().GetString("sort")

		client := getClient()
		result, err := client.ListRepositories(bitbucket.ListRepositoriesArgs{
			Workspace: workspace,
			Query:     query,
			Role:      role,
			Sort:      sort,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
	},
}

var reposGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug]",
	Short: "Get details for a specific repository",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		result, err := client.GetRepository(bitbucket.GetRepositoryArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
	},
}

var reposCreateCmd = &cobra.Command{
	Use:   "create [workspace] [repo-slug]",
	Short: "Create a new repository",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
	},
}

var reposDeleteCmd = &cobra.Command{
	Use:   "delete [workspace] [repo-slug]",
	Short: "Delete a repository",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		err = client.DeleteRepository(bitbucket.DeleteRepositoryArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Repository '%s/%s' deleted successfully.\n", workspace, repoSlug)
	},
}

func init() {
	RootCmd.AddCommand(reposCmd)
	reposCmd.AddCommand(reposListCmd)
	reposCmd.AddCommand(reposGetCmd)
	reposCmd.AddCommand(reposCreateCmd)
	reposCmd.AddCommand(reposDeleteCmd)

	reposListCmd.Flags().StringP("query", "q", "", "Filter repositories using Bitbucket query syntax")
	reposListCmd.Flags().String("role", "", "Filter by role (owner, admin, contributor, member)")
	reposListCmd.Flags().String("sort", "", "Sort field (e.g. -updated_on)")

	reposCreateCmd.Flags().String("description", "", "Repository description")
	reposCreateCmd.Flags().String("language", "", "Primary programming language")
	reposCreateCmd.Flags().String("project", "", "Project key to assign the repo to")
	reposCreateCmd.Flags().Bool("private", true, "Is this repository private?")
}
