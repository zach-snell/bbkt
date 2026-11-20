package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var pipelinesCmd = &cobra.Command{
	Use:     "pipelines",
	Aliases: []string{"pipe"},
	GroupID: groupData,
	Short:   "List, trigger, stop, and inspect Bitbucket Pipelines runs",
	Long: `Manage Bitbucket Pipelines runs and inspect their steps and logs.
Workspace/repo are inferred from your git clone when omitted.

Alias: pipe`,
	Example: `  bbkt pipelines list                         # recent runs on current repo
  bbkt pipelines list --status FAILED         # only failed
  bbkt pipelines trigger -r main              # run default pipeline on main
  bbkt pipelines trigger -r feature/x --pattern deploy   # run a custom: pipeline
  bbkt pipelines watch {pipeline-uuid}        # follow until done; non-zero exit unless SUCCESSFUL
  bbkt pipelines steps {pipeline-uuid}
  bbkt pipelines log {pipeline-uuid} {step-uuid}
  bbkt pipelines stop {pipeline-uuid}`,
}

var pipelinesListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug]",
	Short: "List pipeline runs for a repository",
	Args:  cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}

		status, _ := cmd.Flags().GetString("status")
		sort, _ := cmd.Flags().GetString("sort")
		page, pagelen := paginationArgs(cmd)

		client := getClient()
		result, err := client.ListPipelines(bitbucket.ListPipelinesArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Status:    status,
			Sort:      sort,
			Page:      page,
			Pagelen:   pagelen,
		})
		if err != nil {
			return err
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
		return nil
	},
}

var pipelinesGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug] <pipeline-uuid>",
	Short: "Get details for a single pipeline run",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 1)
		if err != nil {
			return err
		}

		client := getClient()
		result, err := client.GetPipeline(bitbucket.GetPipelineArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
		})
		if err != nil {
			return err
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
		return nil
	},
}

var pipelinesTriggerCmd = &cobra.Command{
	Use:   "trigger [workspace] [repo-slug]",
	Short: "Trigger a new pipeline run (prompts interactively if --ref-name missing)",
	Args:  cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}

		refName, _ := cmd.Flags().GetString("ref-name")
		refType, _ := cmd.Flags().GetString("ref-type")
		pattern, _ := cmd.Flags().GetString("pattern")

		interactive := false
		if refName == "" {
			if outputJSON(cmd) {
				return fmt.Errorf("missing required flag --ref-name under --json (interactive prompts are disabled in --json mode)")
			}
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
			if err := form.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Pipeline trigger cancelled.")
				return err
			}
		}

		if refName == "" {
			return fmt.Errorf("ref-name is required")
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
			return err
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
		return nil
	},
}

var pipelinesStopCmd = &cobra.Command{
	Use:   "stop [workspace] [repo-slug] <pipeline-uuid>",
	Short: "Stop a running pipeline",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 1)
		if err != nil {
			return err
		}

		client := getClient()
		if err := client.StopPipeline(bitbucket.StopPipelineArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
		}); err != nil {
			return err
		}

		PrintOrJSON(cmd, map[string]any{"uuid": trailing[0], "stopped": true}, func() {
			fmt.Printf("Pipeline '%s' stopped successfully.\n", trailing[0])
		})
		return nil
	},
}

var pipelinesStepsCmd = &cobra.Command{
	Use:   "steps [workspace] [repo-slug] <pipeline-uuid>",
	Short: "List steps in a pipeline run",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 1)
		if err != nil {
			return err
		}

		client := getClient()
		result, err := client.ListPipelineSteps(bitbucket.ListPipelineStepsArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
		})
		if err != nil {
			return err
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
		return nil
	},
}

var pipelinesLogsCmd = &cobra.Command{
	Use:   "log [workspace] [repo-slug] <pipeline-uuid> <step-uuid>",
	Short: "Get the log output for a specific pipeline step",
	Args:  cobra.RangeArgs(2, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 2)
		if err != nil {
			return err
		}

		client := getClient()
		result, err := client.GetPipelineStepLog(bitbucket.GetPipelineStepLogArgs{
			Workspace:    workspace,
			RepoSlug:     repoSlug,
			PipelineUUID: trailing[0],
			StepUUID:     trailing[1],
		})
		if err != nil {
			return err
		}

		fmt.Println(string(result))
		return nil
	},
}

func pipeStateLabel(s *bitbucket.PipeState) string {
	switch {
	case s == nil:
		return "-"
	case s.Result != nil:
		return s.Result.Name
	case s.Stage != nil && s.Stage.Name != "" && s.Stage.Name != s.Name:
		return s.Name + "/" + s.Stage.Name
	default:
		return s.Name
	}
}

// lastLines returns at most n trailing lines of s.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

var pipelinesWatchCmd = &cobra.Command{
	Use:   "watch [workspace] [repo-slug] <pipeline-uuid>",
	Short: "Follow a pipeline run until it finishes; exits non-zero unless it succeeds",
	Long: `Poll a pipeline run and print each step as its state changes. Returns when the
run completes or pauses on a manual step. On failure the tail of each failed
step's log is printed. Exit status is 0 only for a SUCCESSFUL run.`,
	Args: cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 1)
		if err != nil {
			return err
		}
		interval, _ := cmd.Flags().GetDuration("interval")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		logLines, _ := cmd.Flags().GetInt("log-lines")
		jsonOut := outputJSON(cmd)

		client := getClient()
		stepsArgs := bitbucket.ListPipelineStepsArgs{Workspace: workspace, RepoSlug: repoSlug, PipelineUUID: trailing[0]}
		var steps []bitbucket.PipelineStep
		seen := map[string]string{}

		p, err := client.WaitPipeline(cmd.Context(), bitbucket.WaitPipelineArgs{
			GetPipelineArgs: bitbucket.GetPipelineArgs{Workspace: workspace, RepoSlug: repoSlug, PipelineUUID: trailing[0]},
			Interval:        interval,
			Timeout:         timeout,
			OnPoll: func(p *bitbucket.Pipeline) error {
				res, err := client.ListAllPipelineSteps(stepsArgs)
				if err != nil {
					return nil // step progress is best-effort; the run state drives the wait
				}
				steps = res.Values
				for _, s := range steps {
					label := pipeStateLabel(s.State)
					if seen[s.UUID] != label && !jsonOut {
						fmt.Printf("%s  #%d  %-45s %s\n", time.Now().Format("15:04:05"), p.BuildNumber, s.Name, label)
					}
					seen[s.UUID] = label
				}
				return nil
			},
		})
		if p == nil {
			return err
		}

		if jsonOut {
			PrintJSON(p)
		} else {
			fmt.Printf("Pipeline #%d: %s (%s)\n", p.BuildNumber, pipeStateLabel(p.State), FormatDuration(p.DurationSecs))
			for _, s := range steps {
				if s.State == nil || s.State.Result == nil || s.State.Result.Name == "SUCCESSFUL" ||
					s.State.Result.Name == "NOT_RUN" || s.State.Result.Name == "STOPPED" || logLines <= 0 {
					continue
				}
				log, lerr := client.GetPipelineStepLog(bitbucket.GetPipelineStepLogArgs{
					Workspace: workspace, RepoSlug: repoSlug, PipelineUUID: trailing[0], StepUUID: s.UUID,
				})
				if lerr != nil {
					continue
				}
				fmt.Printf("\n--- %s (%s), last %d lines ---\n%s\n", s.Name, s.State.Result.Name, logLines, lastLines(string(log), logLines))
			}
		}

		switch {
		case errors.Is(err, bitbucket.ErrWaitTimeout):
			return fmt.Errorf("pipeline #%d still %s after %s", p.BuildNumber, pipeStateLabel(p.State), timeout)
		case err != nil:
			return err
		case p.State.Result == nil:
			return fmt.Errorf("pipeline #%d is %s", p.BuildNumber, pipeStateLabel(p.State))
		case p.State.Result.Name != "SUCCESSFUL":
			return fmt.Errorf("pipeline #%d finished %s", p.BuildNumber, p.State.Result.Name)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(pipelinesCmd)
	pipelinesCmd.AddCommand(pipelinesWatchCmd)
	pipelinesWatchCmd.Flags().Duration("interval", 15*time.Second, "Poll interval")
	pipelinesWatchCmd.Flags().Duration("timeout", 0, "Give up after this long (0 = no limit)")
	pipelinesWatchCmd.Flags().Int("log-lines", 30, "Log lines to print from each failed step (0 = none)")
	pipelinesCmd.AddCommand(pipelinesListCmd)
	pipelinesCmd.AddCommand(pipelinesGetCmd)
	pipelinesCmd.AddCommand(pipelinesTriggerCmd)
	pipelinesCmd.AddCommand(pipelinesStopCmd)
	pipelinesCmd.AddCommand(pipelinesStepsCmd)
	pipelinesCmd.AddCommand(pipelinesLogsCmd)

	pipelinesListCmd.Flags().String("status", "", "Filter by status: SUCCESSFUL | FAILED | INPROGRESS | STOPPED")
	pipelinesListCmd.Flags().String("sort", "-created_on", "Sort field (prefix with - for desc)")
	addPaginationFlags(pipelinesListCmd)

	pipelinesTriggerCmd.Flags().StringP("ref-name", "r", "", "Branch or tag name to run pipeline on")
	pipelinesTriggerCmd.Flags().StringP("ref-type", "t", "branch", "Reference type: branch | tag | bookmark")
	// No -p shorthand: it collides with the global persistent --profile (-p),
	// which panics cobra when the flags merge. Long --pattern only.
	pipelinesTriggerCmd.Flags().String("pattern", "", "Name of a 'custom:' pipeline from bitbucket-pipelines.yml (omit to run the branch's default pipeline)")
}
