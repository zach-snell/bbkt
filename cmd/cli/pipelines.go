package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var pipelinesCmd = &cobra.Command{
	Use:     "pipelines",
	Aliases: []string{"pipe"},
	Short:   "Manage and trigger Bitbucket Pipelines",
}

var pipelinesListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug]",
	Short: "List pipeline runs for a repository",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		status, _ := cmd.Flags().GetString("status")
		sort, _ := cmd.Flags().GetString("sort")

		client := getClient()
		result, err := client.ListPipelines(bitbucket.ListPipelinesArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Status:    status,
			Sort:      sort,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No pipelines found.")
				return
			}
			t := NewTable()
			t.Header("#", "State", "Branch", "Duration", "Created")
			for _, p := range result.Values {
				state := "-"
				if p.State != nil {
					state = p.State.Name
					if p.State.Result != nil {
						state = p.State.Result.Name
					}
				}
				branch := "-"
				if p.Target != nil && p.Target.RefName != "" {
					branch = p.Target.RefName
				}
				t.Row(
					fmt.Sprintf("%d", p.BuildNumber),
					state,
					branch,
					FormatDuration(p.DurationSecs),
					FormatTime(p.CreatedOn),
				)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
	},
}

var pipelinesGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug] [pipeline-uuid]",
	Short: "Get details for a single pipeline run",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		result, err := client.GetPipeline(bitbucket.GetPipelineArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Pipeline #%d\n", result.BuildNumber)
			KV("UUID", result.UUID)
			if result.State != nil {
				state := result.State.Name
				if result.State.Result != nil {
					state = result.State.Result.Name
				}
				KV("State", state)
			}
			if result.Target != nil {
				KV("Branch", result.Target.RefName)
				KV("Ref Type", result.Target.RefType)
			}
			KV("Trigger", result.TriggerName)
			if result.Creator != nil {
				KV("Creator", result.Creator.DisplayName)
			}
			KV("Duration", FormatDuration(result.DurationSecs))
			KV("Created", FormatTime(result.CreatedOn))
			KV("Completed", FormatTimePtr(result.CompletedOn))
		})
	},
}

var pipelinesTriggerCmd = &cobra.Command{
	Use:   "trigger [workspace] [repo-slug]",
	Short: "Trigger a new pipeline run",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, _, err := ParseArgs(args, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		refName, _ := cmd.Flags().GetString("ref-name")
		refType, _ := cmd.Flags().GetString("ref-type")
		pattern, _ := cmd.Flags().GetString("pattern")

		interactive := false
		if refName == "" {
			interactive = true
			fmt.Println("Missing required arguments. Entering interactive mode...")

			if refType == "" {
				refType = "branch" // default
			}

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Reference Type").
						Options(
							huh.NewOption("Branch", "branch"),
							huh.NewOption("Tag", "tag"),
							huh.NewOption("Bookmark", "bookmark"),
						).
						Value(&refType),
					huh.NewInput().
						Title("Reference Name").
						Value(&refName).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("reference name is required")
							}
							return nil
						}),
					huh.NewInput().
						Title("Pipeline Pattern (optional)").
						Description("Leave empty for default pipeline").
						Value(&pattern),
				),
			)
			err := form.Run()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Pipeline trigger cancelled.")
				os.Exit(1)
			}
		}

		if refName == "" {
			fmt.Fprintln(os.Stderr, "Error: ref-name is required")
			os.Exit(1)
		}

		if interactive {
			fmt.Printf("Triggering pipeline on %s '%s'...\n", refType, refName)
		}

		client := getClient()
		result, err := client.TriggerPipeline(bitbucket.TriggerPipelineArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			RefName:   refName,
			RefType:   refType,
			Pattern:   pattern,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			fmt.Printf("Triggered Pipeline #%d\n", result.BuildNumber)
			KV("UUID", result.UUID)
			if result.State != nil {
				KV("State", result.State.Name)
			}
			if result.Target != nil {
				KV("Branch", result.Target.RefName)
			}
			KV("Created", FormatTime(result.CreatedOn))
		})
	},
}

var pipelinesStopCmd = &cobra.Command{
	Use:   "stop [workspace] [repo-slug] [pipeline-uuid]",
	Short: "Stop a running pipeline",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		err = client.StopPipeline(bitbucket.StopPipelineArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Pipeline '%s' stopped successfully.\n", trailing[0])
	},
}

var pipelinesStepsCmd = &cobra.Command{
	Use:   "steps [workspace] [repo-slug] [pipeline-uuid]",
	Short: "List steps in a pipeline run",
	Args:  cobra.RangeArgs(1, 3),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		result, err := client.ListPipelineSteps(bitbucket.ListPipelineStepsArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No steps found.")
				return
			}
			t := NewTable()
			t.Header("Name", "State", "Duration", "Started")
			for _, s := range result.Values {
				state := "-"
				if s.State != nil {
					state = s.State.Name
					if s.State.Result != nil {
						state = s.State.Result.Name
					}
				}
				t.Row(
					s.Name,
					state,
					FormatDuration(s.DurationSecs),
					FormatTimePtr(s.StartedOn),
				)
			}
			t.Flush()
		})
	},
}

var pipelinesLogsCmd = &cobra.Command{
	Use:   "log [workspace] [repo-slug] [pipeline-uuid] [step-uuid]",
	Short: "Get the log output for a specific pipeline step",
	Args:  cobra.RangeArgs(2, 4),
	Run: func(cmd *cobra.Command, args []string) {
		workspace, repoSlug, trailing, err := ParseArgs(args, 2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := getClient()
		result, err := client.GetPipelineStepLog(bitbucket.GetPipelineStepLogArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
			StepUUID:     trailing[1],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(result))
	},
}

func init() {
	RootCmd.AddCommand(pipelinesCmd)
	pipelinesCmd.AddCommand(pipelinesListCmd)
	pipelinesCmd.AddCommand(pipelinesGetCmd)
	pipelinesCmd.AddCommand(pipelinesTriggerCmd)
	pipelinesCmd.AddCommand(pipelinesStopCmd)
	pipelinesCmd.AddCommand(pipelinesStepsCmd)
	pipelinesCmd.AddCommand(pipelinesLogsCmd)

	pipelinesListCmd.Flags().String("status", "", "Filter by status (e.g. SUCCESSFUL, FAILED, INPROGRESS)")
	pipelinesListCmd.Flags().String("sort", "-created_on", "Sort field")

	pipelinesTriggerCmd.Flags().StringP("ref-name", "r", "", "Branch or tag name to run pipeline on (required)")
	pipelinesTriggerCmd.Flags().StringP("ref-type", "t", "branch", "Reference type: branch or tag")
	pipelinesTriggerCmd.Flags().StringP("pattern", "p", "", "Custom pipeline pattern name to trigger (optional)")
}
