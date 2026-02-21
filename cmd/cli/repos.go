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

		PrintJSON(result)
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

		PrintJSON(result)
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

		PrintJSON(result)
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
