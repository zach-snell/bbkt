package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
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
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pagelen, page, QueryValueEscape(sort))

	if args.Status != "" {
		path += "&status=" + QueryValueEscape(args.Status)
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

	pipe, err := pathUUID("pipeline_uuid", args.PipelineUUID)
	if err != nil {
		return nil, err
	}

	return GetJSON[Pipeline](c, fmt.Sprintf("/repositories/%s/%s/pipelines/%s",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pipe))
}

// ErrWaitTimeout is returned by WaitPipeline when the run is still going at the deadline.
var ErrWaitTimeout = errors.New("timed out waiting for pipeline")

type WaitPipelineArgs struct {
	GetPipelineArgs
	Interval time.Duration // poll interval; default 15s
	Timeout  time.Duration // 0 waits until ctx is done
	// OnPoll is called after every successful poll; returning an error stops the wait.
	OnPoll func(*Pipeline) error
}

// PipelineSettled reports whether a run will not change without outside action:
// it has completed, or it is paused on a manual step.
func PipelineSettled(p *Pipeline) bool {
	if p == nil || p.State == nil {
		return false
	}
	return p.State.Name == "COMPLETED" || (p.State.Stage != nil && p.State.Stage.Name == "PAUSED")
}

// WaitPipeline polls a pipeline run until it settles (see PipelineSettled).
// On timeout it returns the last seen run together with ErrWaitTimeout.
func (c *Client) WaitPipeline(ctx context.Context, args WaitPipelineArgs) (*Pipeline, error) {
	interval := args.Interval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	if args.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, args.Timeout)
		defer cancel()
	}

	waitErr := func() error {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ErrWaitTimeout
		}
		return ctx.Err()
	}
	type poll struct {
		p   *Pipeline
		err error
	}

	var last *Pipeline
	failures := 0
	for {
		// The HTTP layer has no context, so poll in a goroutine and stop
		// waiting on cancel; an abandoned request ends at the client timeout.
		ch := make(chan poll, 1)
		go func() {
			p, err := c.GetPipeline(args.GetPipelineArgs)
			ch <- poll{p, err}
		}()
		var r poll
		select {
		case <-ctx.Done():
			return last, waitErr()
		case r = <-ch:
		}
		if ctx.Err() != nil {
			return last, waitErr()
		}

		p, err := r.p, r.err
		switch {
		case err != nil:
			// Long waits outlive the odd network blip; give up after 3 in a row.
			failures++
			if failures >= 3 {
				return last, err
			}
		default:
			failures = 0
			last = p
			if args.OnPoll != nil {
				if err := args.OnPoll(p); err != nil {
					return p, err
				}
			}
			if PipelineSettled(p) {
				return p, nil
			}
		}

		select {
		case <-ctx.Done():
			return last, waitErr()
		case <-time.After(interval):
		}
	}
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

	// Empty body + JSON Content-Type 400s on Bitbucket; send {} like the
	// other body-less action endpoints (resolve/approve/decline).
	pipe, err := pathUUID("pipeline_uuid", args.PipelineUUID)
	if err != nil {
		return err
	}
	_, err = c.Post(fmt.Sprintf("/repositories/%s/%s/pipelines/%s/stopPipeline",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pipe), map[string]interface{}{})
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

	pipe, err := pathUUID("pipeline_uuid", args.PipelineUUID)
	if err != nil {
		return nil, err
	}

	return GetPaginated[PipelineStep](c, fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pipe))
}

const maxStepPages = 20

// ListAllPipelineSteps follows `next` and returns every step of a run, like
// ListAllPRComments. Next stays set if the walk was cut short.
func (c *Client) ListAllPipelineSteps(args ListPipelineStepsArgs) (*Paginated[PipelineStep], error) {
	first, err := c.ListPipelineSteps(args)
	if err != nil {
		return nil, err
	}
	all := first.Values
	next := first.Next
	seen := map[string]bool{}
	for pages := 1; next != "" && pages < maxStepPages && !seen[next]; pages++ {
		seen[next] = true
		path, err := NormalizeAPIPath(next)
		if err != nil {
			return nil, fmt.Errorf("following steps pagination: %w", err)
		}
		res, err := GetPaginated[PipelineStep](c, path)
		if err != nil {
			return nil, err
		}
		all = append(all, res.Values...)
		next = res.Next
	}
	first.Values = all
	first.Size = len(all)
	first.Next = next
	first.Previous = ""
	return first, nil
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

	pipe, err := pathUUID("pipeline_uuid", args.PipelineUUID)
	if err != nil {
		return nil, err
	}
	step, err := pathUUID("step_uuid", args.StepUUID)
	if err != nil {
		return nil, err
	}

	raw, _, err := c.GetRaw(fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/%s/log",
		QueryEscape(args.Workspace), QueryEscape(args.RepoSlug), pipe, step))
	return raw, err
}
