package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

// ListPipelinesHandler lists pipeline runs for a repository.
func ListPipelinesHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListPipelinesArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListPipelinesArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListPipelines(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list pipelines: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetPipelineHandler gets details for a single pipeline run.
func GetPipelineHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetPipelineArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetPipelineArgs) (*mcp.CallToolResult, any, error) {
		pipe, err := c.GetPipeline(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get pipeline: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(pipe, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// TriggerPipelineHandler triggers a new pipeline run.
func TriggerPipelineHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.TriggerPipelineArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.TriggerPipelineArgs) (*mcp.CallToolResult, any, error) {
		pipe, err := c.TriggerPipeline(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to trigger pipeline: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(pipe, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// StopPipelineHandler stops a running pipeline.
func StopPipelineHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.StopPipelineArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.StopPipelineArgs) (*mcp.CallToolResult, any, error) {
		if err := c.StopPipeline(args); err != nil {
			return ToolResultError(fmt.Sprintf("failed to stop pipeline: %v", err)), nil, nil
		}

		return ToolResultText("Pipeline stopped successfully"), nil, nil
	}
}

// ListPipelineStepsHandler lists steps in a pipeline.
func ListPipelineStepsHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.ListPipelineStepsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.ListPipelineStepsArgs) (*mcp.CallToolResult, any, error) {
		result, err := c.ListPipelineSteps(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to list pipeline steps: %v", err)), nil, nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return ToolResultText(string(data)), nil, nil
	}
}

// GetPipelineStepLogHandler gets the log output for a pipeline step.
func GetPipelineStepLogHandler(c *bitbucket.Client) func(context.Context, *mcp.CallToolRequest, bitbucket.GetPipelineStepLogArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args bitbucket.GetPipelineStepLogArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetPipelineStepLog(args)
		if err != nil {
			return ToolResultError(fmt.Sprintf("failed to get step log: %v", err)), nil, nil
		}

		return ToolResultText(string(raw)), nil, nil
	}
}
