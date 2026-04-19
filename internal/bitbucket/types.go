package bitbucket

import "time"

// Workspace represents a Bitbucket workspace.
type Workspace struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IsPrivate bool   `json:"is_private"`
	Type      string `json:"type"`
	Links     Links  `json:"links"`
}

// Repository represents a Bitbucket repository.
type Repository struct {
	UUID        string    `json:"uuid"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	IsPrivate   bool      `json:"is_private"`
	Language    string    `json:"language"`
	SCM         string    `json:"scm"`
	Size        int64     `json:"size"`
	MainBranch  *Branch   `json:"mainbranch"`
	Owner       *User     `json:"owner"`
	Project     *Project  `json:"project"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	Links       Links     `json:"links"`
}

// Branch represents a branch ref.
type Branch struct {
	Name   string  `json:"name"`
	Target *Commit `json:"target"`
	Type   string  `json:"type"`
	Links  Links   `json:"links"`
}

// Tag represents a tag ref.
type Tag struct {
	Name   string  `json:"name"`
	Target *Commit `json:"target"`
	Type   string  `json:"type"`
	Links  Links   `json:"links"`
}

// Commit represents a single commit.
type Commit struct {
	Hash       string    `json:"hash"`
	Message    string    `json:"message"`
	Date       time.Time `json:"date"`
	Author     *Author   `json:"author"`
	Parents    []Commit  `json:"parents"`
	Repository *MinRepo  `json:"repository"`
	Type       string    `json:"type"`
	Links      Links     `json:"links"`
}

// Author represents a commit author.
type Author struct {
	Raw  string `json:"raw"`
	User *User  `json:"user"`
}

// User represents a Bitbucket user.
type User struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	AccountID   string `json:"account_id"`
	Type        string `json:"type"`
	Links       Links  `json:"links"`
}

// MinRepo is a minimal repository reference used in nested objects.
type MinRepo struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Type     string `json:"type"`
}

// Project represents a Bitbucket project.
type Project struct {
	UUID string `json:"uuid"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// PullRequest represents a pull request.
type PullRequest struct {
	ID                int           `json:"id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	State             string        `json:"state"`
	Source            PREndpoint    `json:"source"`
	Destination       PREndpoint    `json:"destination"`
	Author            *User         `json:"author"`
	Reviewers         []User        `json:"reviewers"`
	Participants      []Participant `json:"participants"`
	MergeCommit       *Commit       `json:"merge_commit"`
	CloseSourceBranch bool          `json:"close_source_branch"`
	CommentCount      int           `json:"comment_count"`
	TaskCount         int           `json:"task_count"`
	Draft             bool          `json:"draft"`
	CreatedOn         time.Time     `json:"created_on"`
	UpdatedOn         time.Time     `json:"updated_on"`
	Links             Links         `json:"links"`
}

// PREndpoint represents a PR source or destination.
type PREndpoint struct {
	Branch     *Branch  `json:"branch,omitempty"`
	Commit     *Commit  `json:"commit,omitempty"`
	Repository *MinRepo `json:"repository,omitempty"`
}

// Participant represents a PR participant.
type Participant struct {
	User     *User  `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
	State    string `json:"state"`
}

// PRComment represents a comment on a PR.
type PRComment struct {
	ID        int        `json:"id"`
	Content   Content    `json:"content"`
	User      *User      `json:"user"`
	CreatedOn time.Time  `json:"created_on"`
	UpdatedOn time.Time  `json:"updated_on"`
	Inline    *Inline    `json:"inline"`
	Parent    *ParentRef `json:"parent"`
	Deleted   bool       `json:"deleted"`
	Pending   bool       `json:"pending"`
	Type      string     `json:"type"`
	Links     Links      `json:"links"`
}

// Content represents rich content with raw/markup/html.
type Content struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup,omitempty"`
	HTML   string `json:"html,omitempty"`
}

// Inline represents inline comment location.
type Inline struct {
	From *int   `json:"from,omitempty"`
	To   *int   `json:"to,omitempty"`
	Path string `json:"path"`
}

// ParentRef points to a parent comment.
type ParentRef struct {
	ID int `json:"id"`
}

// Pipeline represents a pipeline run.
type Pipeline struct {
	UUID         string      `json:"uuid"`
	BuildNumber  int         `json:"build_number"`
	State        *PipeState  `json:"state"`
	Target       *PipeTarget `json:"target"`
	Creator      *User       `json:"creator"`
	CreatedOn    time.Time   `json:"created_on"`
	CompletedOn  *time.Time  `json:"completed_on"`
	DurationSecs int         `json:"duration_in_seconds"`
	TriggerName  string      `json:"trigger_name"`
	Links        Links       `json:"links"`
}

// PipeState is the pipeline state.
type PipeState struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Result *PipeResult `json:"result"`
	Stage  *PipeStage  `json:"stage"`
}

// PipeResult is the pipeline result.
type PipeResult struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// PipeStage is the pipeline stage.
type PipeStage struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// PipeTarget is the pipeline target.
type PipeTarget struct {
	Type    string `json:"type"`
	RefType string `json:"ref_type"`
	RefName string `json:"ref_name"`
}

// PipelineStep represents a single step in a pipeline.
type PipelineStep struct {
	UUID         string     `json:"uuid"`
	State        *PipeState `json:"state"`
	Name         string     `json:"name"`
	StartedOn    *time.Time `json:"started_on"`
	CompletedOn  *time.Time `json:"completed_on"`
	DurationSecs int        `json:"duration_in_seconds"`
	Links        Links      `json:"links"`
}

// DiffStat represents a single file diff stat.
type DiffStat struct {
	Status       string       `json:"status"`
	LinesAdded   int          `json:"lines_added"`
	LinesRemoved int          `json:"lines_removed"`
	Old          *DiffStatRef `json:"old"`
	New          *DiffStatRef `json:"new"`
	Type         string       `json:"type"`
}

// DiffStatRef is a path reference in a diffstat.
type DiffStatRef struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// TreeEntry is a file or directory in the source tree.
type TreeEntry struct {
	Path       string   `json:"path"`
	Type       string   `json:"type"` // commit_file or commit_directory
	Size       int64    `json:"size"`
	Commit     *Commit  `json:"commit"`
	Attributes []string `json:"attributes"`
	Links      Links    `json:"links"`
}

// Links is a map of link objects.
type Links map[string]interface{}

// APIError is the standard Bitbucket error response.
type APIError struct {
	Type  string `json:"type"`
	Error struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
}

// CreatePRRequest is the body for creating a pull request.
type CreatePRRequest struct {
	Title             string     `json:"title"`
	Description       string     `json:"description,omitempty"`
	Source            PREndpoint `json:"source"`
	Destination       PREndpoint `json:"destination,omitempty"`
	CloseSourceBranch bool       `json:"close_source_branch,omitempty"`
	Reviewers         []User     `json:"reviewers,omitempty"`
	Draft             bool       `json:"draft,omitempty"`
}

// CreateBranchRequest is the body for creating a branch.
type CreateBranchRequest struct {
	Name   string            `json:"name"`
	Target map[string]string `json:"target"`
}

// TriggerPipelineRequest is the body for triggering a pipeline.
type TriggerPipelineRequest struct {
	Target    PipeTriggerTarget  `json:"target"`
	Variables []PipelineVariable `json:"variables,omitempty"`
}

// PipeTriggerTarget specifies the pipeline trigger target.
type PipeTriggerTarget struct {
	Type     string            `json:"type"`
	RefType  string            `json:"ref_type"`
	RefName  string            `json:"ref_name"`
	Selector *PipelineSelector `json:"selector,omitempty"`
}

// PipelineSelector for custom pipelines.
type PipelineSelector struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

// PipelineVariable represents a pipeline variable.
type PipelineVariable struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured"`
}

// MergePRRequest is the body for merging a pull request.
type MergePRRequest struct {
	Type              string `json:"type,omitempty"`
	Message           string `json:"message,omitempty"`
	CloseSourceBranch bool   `json:"close_source_branch,omitempty"`
	MergeStrategy     string `json:"merge_strategy,omitempty"`
}

// CreateCommentRequest is the body for creating a PR comment.
type CreateCommentRequest struct {
	Content Content    `json:"content"`
	Inline  *Inline    `json:"inline,omitempty"`
	Parent  *ParentRef `json:"parent,omitempty"`
}
