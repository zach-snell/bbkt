package bitbucket

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

// newVCRClient returns a Client wired to a go-vcr cassette at
// testdata/fixtures/<cassette>.yaml. Replays by default; set BBKT_VCR_RECORD=1
// to hit the real API and overwrite the cassette (requires valid credentials
// in the env and a pre-warmed sandbox — see docs/TESTING.md).
func newVCRClient(t *testing.T, cassette string) *Client {
	t.Helper()

	mode := recorder.ModeReplayOnly
	if os.Getenv("BBKT_VCR_RECORD") == "1" {
		mode = recorder.ModeRecordOnce
	}

	r, err := recorder.New(
		filepath.Join("testdata", "fixtures", cassette),
		recorder.WithMode(mode),
		recorder.WithMatcher(matchMethodAndURL),
		recorder.WithHook(scrubSecrets, recorder.BeforeSaveHook),
	)
	if err != nil {
		t.Fatalf("vcr recorder: %v", err)
	}
	t.Cleanup(func() { _ = r.Stop() })

	c := NewClient("", "", "recorded-token")
	if mode == recorder.ModeRecordOnce {
		// During recording, honor real creds from the standard profile lookup.
		if creds, err := LoadCredentials(); err == nil {
			c = NewClientFromCredentials(creds)
		}
	}
	c.http = r.GetDefaultClient()
	return c
}

// matchMethodAndURL is a minimal matcher that pairs a request to an
// interaction by method and fully-qualified URL only. We don't care about
// matching HTTP plumbing (Host, Proto, User-Agent, Authorization) when
// replaying fixtures.
func matchMethodAndURL(r *http.Request, i cassette.Request) bool {
	return r.Method == i.Method && r.URL.String() == i.URL
}

// scrubSecrets redacts anything that could identify a real account before the
// cassette is written to disk.
func scrubSecrets(i *cassette.Interaction) error {
	for _, h := range []http.Header{i.Request.Headers, i.Response.Headers} {
		for _, k := range []string{"Authorization", "Set-Cookie", "Cookie", "X-Request-Id", "X-Served-By"} {
			h.Del(k)
		}
	}
	return nil
}

// TestFixture_ListPRs_NestedStructure verifies that a full PullRequest
// response — with embedded source/destination branches, reviewers, and
// timestamps — unmarshals cleanly. These are exactly the fields most likely
// to break silently when Bitbucket reshapes its response.
func TestFixture_ListPRs_NestedStructure(t *testing.T) {
	c := newVCRClient(t, "list_prs")

	result, err := c.ListPullRequests(ListPullRequestsArgs{
		Workspace: "demo-ws",
		RepoSlug:  "demo-repo",
	})
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if len(result.Values) < 1 {
		t.Fatalf("expected at least 1 PR, got %d", len(result.Values))
	}

	pr := result.Values[0]
	if pr.ID == 0 {
		t.Errorf("pr.ID = 0, want non-zero")
	}
	if pr.Title == "" {
		t.Errorf("pr.Title is empty")
	}
	if pr.State == "" {
		t.Errorf("pr.State is empty")
	}
	if pr.Source.Branch == nil || pr.Source.Branch.Name == "" {
		t.Errorf("pr.Source.Branch not parsed: %+v", pr.Source)
	}
	if pr.Destination.Branch == nil || pr.Destination.Branch.Name == "" {
		t.Errorf("pr.Destination.Branch not parsed: %+v", pr.Destination)
	}
	if pr.Author == nil || pr.Author.DisplayName == "" {
		t.Errorf("pr.Author not parsed: %+v", pr.Author)
	}
	if pr.CreatedOn.IsZero() {
		t.Errorf("pr.CreatedOn is zero — time.Time RFC3339 parsing broken?")
	}
}

// TestFixture_InlineComments_NullFromParsesAsNil verifies the inverse of
// PR #1: Bitbucket sends "from": null on added-line comments, and our *int
// must decode that as a nil pointer (not an error, not 0).
func TestFixture_InlineComments_NullFromParsesAsNil(t *testing.T) {
	c := newVCRClient(t, "list_comments")

	result, err := c.ListPRComments(ListPRCommentsArgs{
		Workspace: "demo-ws",
		RepoSlug:  "demo-repo",
		PRID:      1,
	})
	if err != nil {
		t.Fatalf("ListPRComments: %v", err)
	}

	var sawInlineAddedLine, sawInlineRemovedLine bool
	for _, cm := range result.Values {
		if cm.Inline == nil {
			continue
		}
		if cm.Inline.From == nil && cm.Inline.To != nil {
			sawInlineAddedLine = true
		}
		if cm.Inline.To == nil && cm.Inline.From != nil {
			sawInlineRemovedLine = true
		}
	}
	if !sawInlineAddedLine {
		t.Errorf("fixture should include an added-line inline comment (From=nil, To=*int)")
	}
	if !sawInlineRemovedLine {
		t.Errorf("fixture should include a removed-line inline comment (From=*int, To=nil)")
	}
}

// TestFixture_ListWorkspaces_FlattensWorkspaceAccess verifies that the
// workspace_access envelope returned by /user/workspaces is flattened into
// usable Workspace rows. Regression: before the fix, the listing decoded
// into Workspace directly and every row came back with empty Slug/UUID
// because the real fields live one level deep under `.workspace`.
func TestFixture_ListWorkspaces_FlattensWorkspaceAccess(t *testing.T) {
	c := newVCRClient(t, "list_workspaces")

	result, err := c.ListWorkspaces(ListWorkspacesArgs{})
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(result.Values) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(result.Values))
	}

	for i, w := range result.Values {
		if w.Slug == "" {
			t.Errorf("row %d: Slug empty — workspace_access envelope not flattened", i)
		}
		if w.UUID == "" {
			t.Errorf("row %d: UUID empty — workspace_access envelope not flattened", i)
		}
	}

	// IsAdmin must come from the envelope's `administrator` field, not the
	// inner workspace_base. Pin both values so a refactor that drops the
	// flag (or copies it from the wrong place) fails loudly.
	if result.Values[0].IsAdmin {
		t.Errorf("row 0 should be non-admin (administrator=false in fixture)")
	}
	if !result.Values[1].IsAdmin {
		t.Errorf("row 1 should be admin (administrator=true in fixture)")
	}
}

// TestFixture_Pipeline_NestedState verifies that a Pipeline's deeply-nested
// State/Result/Stage/Target structure unmarshals cleanly. Pipelines return
// pointer-typed nested objects that are easy to drop during refactors.
func TestFixture_Pipeline_NestedState(t *testing.T) {
	c := newVCRClient(t, "get_pipeline")

	pipeline, err := c.GetPipeline(GetPipelineArgs{
		Workspace:    "demo-ws",
		RepoSlug:     "demo-repo",
		PipelineUUID: "{11111111-1111-1111-1111-111111111111}",
	})
	if err != nil {
		t.Fatalf("GetPipeline: %v", err)
	}
	if pipeline.UUID == "" {
		t.Errorf("pipeline.UUID empty")
	}
	if pipeline.BuildNumber == 0 {
		t.Errorf("pipeline.BuildNumber = 0")
	}
	if pipeline.State == nil {
		t.Fatalf("pipeline.State not parsed")
	}
	if pipeline.State.Name == "" {
		t.Errorf("pipeline.State.Name empty")
	}
	if pipeline.State.Result == nil || pipeline.State.Result.Name == "" {
		t.Errorf("pipeline.State.Result not parsed: %+v", pipeline.State.Result)
	}
	if pipeline.Target == nil || pipeline.Target.RefName == "" {
		t.Errorf("pipeline.Target not parsed: %+v", pipeline.Target)
	}
	if pipeline.CompletedOn == nil {
		t.Errorf("pipeline.CompletedOn should be a non-nil *time.Time for a finished pipeline")
	} else if !strings.Contains(pipeline.CompletedOn.Format("2006-01-02"), "20") {
		t.Errorf("pipeline.CompletedOn looks wrong: %v", pipeline.CompletedOn)
	}
}
