package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// captureCreateCommentBody wires up a client whose CreatePRComment POST
// is intercepted; returns a pointer that holds the raw JSON body after the call.
func captureCreateCommentBody(t *testing.T) (client *Client, body *string) {
	t.Helper()
	var captured string
	client = newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		captured = string(b)
		_, _ = w.Write([]byte(`{"id":1,"content":{"raw":"stub"}}`))
	})
	return client, &captured
}

// Regression for PR #1: Inline.From/To lacked omitempty, so a nil pointer
// marshaled as "from":null and Bitbucket anchored the comment to both diff
// sides (rendered twice).
func TestCreatePRComment_AddedLineOmitsFromField(t *testing.T) {
	c, body := captureCreateCommentBody(t)
	if _, err := c.CreatePRComment(CreatePRCommentArgs{
		Workspace: "w", RepoSlug: "r", PRID: 1,
		Content: "hi", FilePath: "a.go", LineTo: 42,
	}); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(*body, `"from"`) {
		t.Errorf("added-line comment JSON should omit 'from', got: %s", *body)
	}
	if !strings.Contains(*body, `"to":42`) {
		t.Errorf("JSON should include 'to':42, got: %s", *body)
	}
	if !strings.Contains(*body, `"path":"a.go"`) {
		t.Errorf("JSON should include 'path':\"a.go\", got: %s", *body)
	}
}

func TestCreatePRComment_RemovedLineOmitsToField(t *testing.T) {
	c, body := captureCreateCommentBody(t)
	if _, err := c.CreatePRComment(CreatePRCommentArgs{
		Workspace: "w", RepoSlug: "r", PRID: 1,
		Content: "hi", FilePath: "a.go", LineFrom: 7,
	}); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(*body, `"to"`) {
		t.Errorf("removed-line comment JSON should omit 'to', got: %s", *body)
	}
	if !strings.Contains(*body, `"from":7`) {
		t.Errorf("JSON should include 'from':7, got: %s", *body)
	}
}

func TestCreatePRComment_BothSidesIncludesBothFields(t *testing.T) {
	c, body := captureCreateCommentBody(t)
	if _, err := c.CreatePRComment(CreatePRCommentArgs{
		Workspace: "w", RepoSlug: "r", PRID: 1,
		Content: "hi", FilePath: "a.go", LineFrom: 5, LineTo: 10,
	}); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(*body, `"from":5`) || !strings.Contains(*body, `"to":10`) {
		t.Errorf("context-line comment should include both from and to, got: %s", *body)
	}
}

func TestCreatePRComment_NonInlineOmitsInlineObject(t *testing.T) {
	c, body := captureCreateCommentBody(t)
	if _, err := c.CreatePRComment(CreatePRCommentArgs{
		Workspace: "w", RepoSlug: "r", PRID: 1,
		Content: "general comment, no file",
	}); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(*body, `"inline"`) {
		t.Errorf("non-inline comment should omit 'inline' key entirely, got: %s", *body)
	}
}

func TestCreatePRComment_ReplySetsParent(t *testing.T) {
	c, body := captureCreateCommentBody(t)
	if _, err := c.CreatePRComment(CreatePRCommentArgs{
		Workspace: "w", RepoSlug: "r", PRID: 1,
		Content: "reply", ParentID: 99,
	}); err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(*body), &parsed); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, *body)
	}
	parent, ok := parsed["parent"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'parent' object, got: %s", *body)
	}
	id, _ := parent["id"].(float64)
	if int(id) != 99 {
		t.Errorf("parent.id = %v, want 99", parent["id"])
	}
}

func TestCreatePRComment_ValidatesRequiredFields(t *testing.T) {
	c := NewClient("", "", "x") // no server — validation must short-circuit
	cases := []struct {
		name string
		args CreatePRCommentArgs
	}{
		{"missing workspace", CreatePRCommentArgs{RepoSlug: "r", PRID: 1, Content: "c"}},
		{"missing repo", CreatePRCommentArgs{Workspace: "w", PRID: 1, Content: "c"}},
		{"missing pr id", CreatePRCommentArgs{Workspace: "w", RepoSlug: "r", Content: "c"}},
		{"missing content", CreatePRCommentArgs{Workspace: "w", RepoSlug: "r", PRID: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.CreatePRComment(tc.args)
			if err == nil {
				t.Fatalf("%s: expected validation error, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), "required") {
				t.Errorf("%s: expected 'required' in error, got: %v", tc.name, err)
			}
		})
	}
}

// Regression: ResolvePRComment posted a nil body with a JSON Content-Type,
// which Bitbucket rejects with 400. It must send an empty JSON object {}
// (matching the approve/decline body-less action convention).
func TestResolvePRComment_SendsEmptyJSONBody(t *testing.T) {
	var method, path, body, ctype string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		ctype = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{}`))
	})

	if err := c.ResolvePRComment(CommentActionArgs{
		Workspace: "w", RepoSlug: "r", PRID: 1, CommentID: 99,
	}); err != nil {
		t.Fatal(err)
	}

	if method != http.MethodPost {
		t.Errorf("method = %q, want POST", method)
	}
	if !strings.HasSuffix(path, "/pullrequests/1/comments/99/resolve") {
		t.Errorf("path = %q, want .../comments/99/resolve", path)
	}
	if strings.TrimSpace(body) != "{}" {
		t.Errorf("resolve body = %q, want {} (empty body 400s on Bitbucket)", body)
	}
	if ctype != "application/json" {
		t.Errorf("content-type = %q, want application/json", ctype)
	}
}
