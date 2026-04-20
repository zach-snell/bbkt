package bitbucket

import (
	"net/http"
	"testing"
)

// Regression for CHANGE-2770: Bitbucket deprecated /workspaces on 2026-02-25.
// ListWorkspaces must call /user/workspaces (CHANGE-3022 replacement, which
// returns the same paginated envelope but scoped to the authenticated user).
// If a future refactor reverts this, the CLI's `bbkt workspaces list` would
// silently 410 against prod — pin the path.
func TestListWorkspaces_UsesUserWorkspacesPath(t *testing.T) {
	var gotPath string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"values":[],"pagelen":25,"size":0,"page":1}`))
	})
	if _, err := c.ListWorkspaces(ListWorkspacesArgs{}); err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if gotPath != "/user/workspaces" {
		t.Errorf("ListWorkspaces hit %q, want /user/workspaces (the /workspaces path was deprecated by CHANGE-2770)", gotPath)
	}
}

// Client.Scopes() previously probed /workspace (singular) first, which has
// always returned 404 — no such endpoint exists. Pin the call to /user so
// the extra 404 probe doesn't get reintroduced.
func TestScopes_HitsUserEndpointDirectly(t *testing.T) {
	var paths []string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("X-Oauth-Scopes", "account, repository")
		_, _ = w.Write([]byte(`{}`))
	})
	scopes, err := c.Scopes()
	if err != nil {
		t.Fatalf("Scopes: %v", err)
	}
	if len(paths) != 1 || paths[0] != "/user" {
		t.Errorf("Scopes() called %v, want exactly [/user]", paths)
	}
	if len(scopes) != 2 || scopes[0] != "account" || scopes[1] != "repository" {
		t.Errorf("parsed scopes = %v, want [account repository]", scopes)
	}
}
