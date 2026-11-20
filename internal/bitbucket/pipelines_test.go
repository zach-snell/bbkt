package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func captureTriggerBody(t *testing.T) (client *Client, body *TriggerPipelineRequest, raw *string) {
	t.Helper()
	var req TriggerPipelineRequest
	var captured string
	client = newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		captured = string(b)
		if err := json.Unmarshal(b, &req); err != nil {
			t.Errorf("trigger body is not JSON: %v", err)
		}
		_, _ = w.Write([]byte(`{"uuid":"{p}","build_number":7}`))
	})
	return client, &req, &captured
}

func TestTriggerPipeline_SendsVariables(t *testing.T) {
	c, body, _ := captureTriggerBody(t)
	_, err := c.TriggerPipeline(TriggerPipelineArgs{
		Workspace: "w", RepoSlug: "r", RefName: "main", Pattern: "build",
		Variables: []PipelineVariable{
			{Key: "FLAVOUR", Value: "release"},
			{Key: "TOKEN", Value: "s3cr=t", Secured: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []PipelineVariable{
		{Key: "FLAVOUR", Value: "release"},
		{Key: "TOKEN", Value: "s3cr=t", Secured: true},
	}
	if len(body.Variables) != len(want) {
		t.Fatalf("variables = %+v, want %+v", body.Variables, want)
	}
	for i := range want {
		if body.Variables[i] != want[i] {
			t.Errorf("variables[%d] = %+v, want %+v", i, body.Variables[i], want[i])
		}
	}
	if body.Target.Selector == nil || body.Target.Selector.Pattern != "build" {
		t.Errorf("selector = %+v, want custom pattern build", body.Target.Selector)
	}
}

func TestTriggerPipeline_OmitsVariablesWhenNone(t *testing.T) {
	c, _, raw := captureTriggerBody(t)
	if _, err := c.TriggerPipeline(TriggerPipelineArgs{Workspace: "w", RepoSlug: "r", RefName: "main"}); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(*raw), &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["variables"]; ok {
		t.Errorf("body should omit 'variables' when none given, got: %s", *raw)
	}
}
