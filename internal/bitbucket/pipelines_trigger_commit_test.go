package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func triggerTarget(t *testing.T, args TriggerPipelineArgs) map[string]any {
	t.Helper()
	var body map[string]any
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"build_number":1}`))
	})
	args.Workspace, args.RepoSlug = "ws", "repo"
	if _, err := c.TriggerPipeline(args); err != nil {
		t.Fatal(err)
	}
	return body["target"].(map[string]any)
}

func TestTriggerPipeline_RefAtCommit(t *testing.T) {
	tgt := triggerTarget(t, TriggerPipelineArgs{RefName: "main", Commit: "abc1234", Pattern: "deploy"})
	if tgt["type"] != "pipeline_ref_target" || tgt["ref_name"] != "main" || tgt["ref_type"] != "branch" {
		t.Fatalf("target = %v", tgt)
	}
	if c := tgt["commit"].(map[string]any); c["type"] != "commit" || c["hash"] != "abc1234" {
		t.Fatalf("commit = %v", c)
	}
}

func TestTriggerPipeline_BareCommit(t *testing.T) {
	tgt := triggerTarget(t, TriggerPipelineArgs{Commit: "abc1234", Pattern: "deploy"})
	if tgt["type"] != "pipeline_commit_target" {
		t.Fatalf("type = %v", tgt["type"])
	}
	if _, ok := tgt["ref_name"]; ok {
		t.Fatalf("bare commit target must not carry ref fields: %v", tgt)
	}
	if tgt["selector"].(map[string]any)["pattern"] != "deploy" {
		t.Fatalf("selector = %v", tgt["selector"])
	}
}

func TestTriggerPipeline_RefOnlyHasNoCommit(t *testing.T) {
	tgt := triggerTarget(t, TriggerPipelineArgs{RefName: "main"})
	if _, ok := tgt["commit"]; ok {
		t.Fatalf("unexpected commit: %v", tgt)
	}
}

func TestTriggerPipeline_RejectsBadCommit(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("request should not be sent")
	})
	for _, bad := range []string{"main", "abc", "abc1234/../x", "g123456"} {
		if _, err := c.TriggerPipeline(TriggerPipelineArgs{Workspace: "ws", RepoSlug: "repo", Commit: bad}); err == nil {
			t.Errorf("commit %q accepted", bad)
		}
	}
}
