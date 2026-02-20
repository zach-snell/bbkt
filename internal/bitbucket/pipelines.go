package bitbucket

import (
	"encoding/json"
	"fmt"
)

type ListPipelinesArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Pagelen   int    `json:"pagelen,omitempty" jsonschema:"Results per page"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field (default -created_on)"`
	Status    string `json:"status,omitempty" jsonschema:"Filter by status"`
}

// ListPipelines lists pipeline runs for a repository.
func (c *Client) ListPipelines(args ListPipelinesArgs) (*Paginated[Pipeline], error) {
	if args.Workspace == "" || args.RepoSlug == "" {
		return nil, fmt.Errorf("workspace and repo_slug are required")
	}

	pagelen := args.Pagelen
	if pagelen == 0 {
		pagelen = 25
	}
	page := args.Page
	if page == 0 {
		page = 1
	}
	sort := args.Sort
	if sort == "" {
		sort = "-created_on"
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines?pagelen=%d&page=%d&sort=%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pagelen, page, QueryEscape(sort))

	if args.Status != "" {
		path += "&status=" + QueryEscape(args.Status)
	}

	return GetPaginated[Pipeline](c, path)
}

type GetPipelineArgs struct {
	Workspace    string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug     string `json:"repo_slug" jsonschema:"Repository slug"`
	PipelineUUID string `json:"pipeline_uuid" jsonschema:"Pipeline UUID"`
}

// GetPipeline gets details for a single pipeline run.
func (c *Client) GetPipeline(args GetPipelineArgs) (*Pipeline, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PipelineUUID == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and pipeline_uuid are required")
	}

	return GetJSON[Pipeline](c, fmt.Sprintf("/repositories/%s/%s/pipelines/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PipelineUUID))
}

type TriggerPipelineArgs struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	RefName   string `json:"ref_name" jsonschema:"Branch or tag name to run pipeline on"`
	RefType   string `json:"ref_type,omitempty" jsonschema:"Reference type: branch or tag (default branch)"`
	Pattern   string `json:"pattern,omitempty" jsonschema:"Custom pipeline pattern name to trigger"`
}

// TriggerPipeline triggers a new pipeline run.
func (c *Client) TriggerPipeline(args TriggerPipelineArgs) (*Pipeline, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.RefName == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and ref_name are required")
	}

	refType := args.RefType
	if refType == "" {
		refType = "branch"
	}

	body := TriggerPipelineRequest{
		Target: PipeTriggerTarget{
			Type:    "pipeline_ref_target",
			RefType: refType,
			RefName: args.RefName,
		},
	}

	if args.Pattern != "" {
		body.Target.Selector = &PipelineSelector{
			Type:    "custom",
			Pattern: args.Pattern,
		}
	}

	respData, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pipelines",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug)), body)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger pipeline: %v", err)
	}

	var pipe Pipeline
	if err := json.Unmarshal(respData, &pipe); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &pipe, nil
}

type StopPipelineArgs struct {
	Workspace    string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug     string `json:"repo_slug" jsonschema:"Repository slug"`
	PipelineUUID string `json:"pipeline_uuid" jsonschema:"Pipeline UUID to stop"`
}

// StopPipeline stops a running pipeline.
func (c *Client) StopPipeline(args StopPipelineArgs) error {
	if args.Workspace == "" || args.RepoSlug == "" || args.PipelineUUID == "" {
		return fmt.Errorf("workspace, repo_slug, and pipeline_uuid are required")
	}

	_, err := c.Post(fmt.Sprintf("/repositories/%s/%s/pipelines/%s/stopPipeline",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PipelineUUID), nil)
	return err
}

type ListPipelineStepsArgs struct {
	Workspace    string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug     string `json:"repo_slug" jsonschema:"Repository slug"`
	PipelineUUID string `json:"pipeline_uuid" jsonschema:"Pipeline UUID"`
}

// ListPipelineSteps lists steps in a pipeline.
func (c *Client) ListPipelineSteps(args ListPipelineStepsArgs) (*Paginated[PipelineStep], error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PipelineUUID == "" {
		return nil, fmt.Errorf("workspace, repo_slug, and pipeline_uuid are required")
	}

	return GetPaginated[PipelineStep](c, fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PipelineUUID))
}

type GetPipelineStepLogArgs struct {
	Workspace    string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug     string `json:"repo_slug" jsonschema:"Repository slug"`
	PipelineUUID string `json:"pipeline_uuid" jsonschema:"Pipeline UUID"`
	StepUUID     string `json:"step_uuid" jsonschema:"Step UUID"`
}

// GetPipelineStepLog gets the log output for a pipeline step.
func (c *Client) GetPipelineStepLog(args GetPipelineStepLogArgs) ([]byte, error) {
	if args.Workspace == "" || args.RepoSlug == "" || args.PipelineUUID == "" || args.StepUUID == "" {
		return nil, fmt.Errorf("workspace, repo_slug, pipeline_uuid, and step_uuid are required")
	}

	raw, _, err := c.GetRaw(fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/%s/log",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), args.PipelineUUID, args.StepUUID))
	return raw, err
}
