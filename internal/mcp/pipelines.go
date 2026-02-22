package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

type ManagePipelinesArgs struct {
	Action       string `json:"action" jsonschema:"Action to perform: 'list', 'get', 'trigger', 'stop', 'list-steps', 'get-step-log'" jsonschema_enum:"list,get,trigger,stop,list-steps,get-step-log"`
	Workspace    string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug     string `json:"repo_slug" jsonschema:"Repository slug"`
	PipelineUUID string `json:"pipeline_uuid,omitempty" jsonschema:"Pipeline UUID"`
	StepUUID     string `json:"step_uuid,omitempty" jsonschema:"Step UUID (for 'get-step-log')"`
	RefType      string `json:"ref_type,omitempty" jsonschema:"Reference type: branch or tag (default branch) (for 'trigger')"`
	RefName      string `json:"ref_name,omitempty" jsonschema:"Branch or tag name to run pipeline on (for 'trigger')"`
	Pattern      string `json:"pattern,omitempty" jsonschema:"Custom pipeline pattern name to trigger (for 'trigger')"`
	Page         int    `json:"page,omitempty" jsonschema:"Page number"`
	Pagelen      int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
	Sort         string `json:"sort,omitempty" jsonschema:"Sort field"`
	Status       string `json:"status,omitempty" jsonschema:"Filter by status"`
}

// ManagePipelinesHandler handles the consolidated pipeline operations.
func ManagePipelinesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, ManagePipelinesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ManagePipelinesArgs) (*mcp.CallToolResult, any, error) {
		switch args.Action {
		case "list":
			result, err := c.ListPipelines(bitbucket.ListPipelinesArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				Page:      args.Page,
				Pagelen:   args.Pagelen,
				Sort:      args.Sort,
				Status:    args.Status,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list pipelines: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get":
			if args.PipelineUUID == "" {
				return ToolResultError("pipeline_uuid is required for 'get' action"), nil, nil
			}
			pipe, err := c.GetPipeline(bitbucket.GetPipelineArgs{
				Workspace:    args.Workspace,
				RepoSlug:     args.RepoSlug,
				PipelineUUID: args.PipelineUUID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get pipeline: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(pipe, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "trigger":
			if args.RefName == "" {
				return ToolResultError("ref_name is required for 'trigger' action"), nil, nil
			}
			pipe, err := c.TriggerPipeline(bitbucket.TriggerPipelineArgs{
				Workspace: args.Workspace,
				RepoSlug:  args.RepoSlug,
				RefType:   args.RefType,
				RefName:   args.RefName,
				Pattern:   args.Pattern,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to trigger pipeline: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(pipe, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "stop":
			if args.PipelineUUID == "" {
				return ToolResultError("pipeline_uuid is required for 'stop' action"), nil, nil
			}
			if err := c.StopPipeline(bitbucket.StopPipelineArgs{
				Workspace:    args.Workspace,
				RepoSlug:     args.RepoSlug,
				PipelineUUID: args.PipelineUUID,
			}); err != nil {
				return ToolResultError(fmt.Sprintf("failed to stop pipeline: %v", err)), nil, nil
			}
			return ToolResultText("Pipeline stopped successfully"), nil, nil

		case "list-steps":
			if args.PipelineUUID == "" {
				return ToolResultError("pipeline_uuid is required for 'list-steps' action"), nil, nil
			}
			result, err := c.ListPipelineSteps(bitbucket.ListPipelineStepsArgs{
				Workspace:    args.Workspace,
				RepoSlug:     args.RepoSlug,
				PipelineUUID: args.PipelineUUID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to list pipeline steps: %v", err)), nil, nil
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			return ToolResultText(string(data)), nil, nil

		case "get-step-log":
			if args.PipelineUUID == "" || args.StepUUID == "" {
				return ToolResultError("pipeline_uuid and step_uuid are required for 'get-step-log' action"), nil, nil
			}
			raw, err := c.GetPipelineStepLog(bitbucket.GetPipelineStepLogArgs{
				Workspace:    args.Workspace,
				RepoSlug:     args.RepoSlug,
				PipelineUUID: args.PipelineUUID,
				StepUUID:     args.StepUUID,
			})
			if err != nil {
				return ToolResultError(fmt.Sprintf("failed to get step log: %v", err)), nil, nil
			}
			return ToolResultText(string(raw)), nil, nil

		default:
			return ToolResultError(fmt.Sprintf("unknown action: %s", args.Action)), nil, nil
		}
	}
}
