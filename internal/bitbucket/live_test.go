//go:build live

// Live smoke tests. Hit the real Bitbucket API to catch drift that unit
// tests and recorded fixtures cannot: undocumented response fields, surprise
// 4xx/5xx behavior, auth flow regressions.
//
// Run with:
//
//	go test -tags=live ./internal/bitbucket/...
//
// Requires:
//   - A bbkt profile saved via `bbkt auth ...` (or BBKT_PROFILE env var)
//   - BBKT_LIVE_WORKSPACE set to a workspace slug the profile can access
//   - BBKT_LIVE_REPO (optional) set to a repo slug for repo-level tests
//
// All tests in this file are read-only. Do not add destructive operations
// (create/update/delete) without an explicit opt-in flag and a dedicated
// throwaway sandbox.

package bitbucket

import (
	"os"
	"strings"
	"testing"
)

func liveClient(t *testing.T) *Client {
	t.Helper()
	if os.Getenv("BBKT_LIVE_WORKSPACE") == "" {
		t.Skip("skipping live test: BBKT_LIVE_WORKSPACE not set")
	}
	// Prefer BITBUCKET_ACCESS_TOKEN (how CI threads in a consumer token)
	// and fall back to the stored profile for local invocation.
	if tok := os.Getenv("BITBUCKET_ACCESS_TOKEN"); tok != "" {
		return NewClient("", "", tok)
	}
	if u, p := os.Getenv("BITBUCKET_USERNAME"), os.Getenv("BITBUCKET_API_TOKEN"); u != "" && p != "" {
		return NewClient(u, p, "")
	}
	creds, err := LoadCredentials()
	if err != nil {
		t.Skipf("skipping live test: no credentials available (%v)", err)
	}
	return NewClientFromCredentials(creds)
}

// isMachineToken is true for client_credentials (consumer) tokens, which are
// workspace-scoped rather than user-scoped. Several endpoints return empty
// for these tokens where a user token would return data.
func isMachineToken() bool {
	return os.Getenv("BITBUCKET_ACCESS_TOKEN") != ""
}

func liveWorkspace() string { return os.Getenv("BBKT_LIVE_WORKSPACE") }
func liveRepo() string      { return os.Getenv("BBKT_LIVE_REPO") }

func TestLive_ListWorkspaces(t *testing.T) {
	c := liveClient(t)

	result, err := c.ListWorkspaces(ListWorkspacesArgs{Pagelen: 10})
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if isMachineToken() {
		// client_credentials tokens are consumer-scoped, not user-scoped, so
		// /user/workspaces returns empty. Just assert the envelope parsed.
		if result.PageLen == 0 {
			t.Errorf("paginated envelope looks unparsed: %+v", result)
		}
		return
	}
	if len(result.Values) == 0 {
		t.Fatal("authenticated profile has no workspaces — unexpected")
	}

	ws := liveWorkspace()
	var found bool
	for _, w := range result.Values {
		if strings.EqualFold(w.Slug, ws) {
			found = true
			if w.UUID == "" {
				t.Errorf("workspace %q has no UUID in response", w.Slug)
			}
			break
		}
	}
	if !found {
		t.Errorf("configured BBKT_LIVE_WORKSPACE=%q not in the first page of workspaces", ws)
	}
}

func TestLive_GetWorkspace(t *testing.T) {
	c := liveClient(t)
	ws, err := c.GetWorkspace(GetWorkspaceArgs{Workspace: liveWorkspace()})
	if err != nil {
		t.Fatalf("GetWorkspace(%q): %v", liveWorkspace(), err)
	}
	if ws.Slug == "" || ws.UUID == "" {
		t.Errorf("workspace fields empty: %+v", ws)
	}
}

func TestLive_ListRepositories(t *testing.T) {
	c := liveClient(t)
	result, err := c.ListRepositories(ListRepositoriesArgs{
		Workspace: liveWorkspace(),
		Pagelen:   5,
	})
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(result.Values) == 0 {
		t.Skipf("workspace %q has no repositories — skipping shape checks", liveWorkspace())
	}
	repo := result.Values[0]
	if repo.Slug == "" || repo.FullName == "" {
		t.Errorf("repo fields empty: %+v", repo)
	}
}

func TestLive_ListPullRequests(t *testing.T) {
	c := liveClient(t)
	if liveRepo() == "" {
		t.Skip("skipping live PR test: BBKT_LIVE_REPO not set")
	}
	result, err := c.ListPullRequests(ListPullRequestsArgs{
		Workspace: liveWorkspace(),
		RepoSlug:  liveRepo(),
		State:     "OPEN",
		Pagelen:   5,
	})
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	// Empty open-PR list is fine; just assert the envelope parsed.
	if result.PageLen == 0 && result.Page == 0 && result.Size == 0 && len(result.Values) == 0 {
		t.Errorf("paginated envelope looks unparsed (all zero): %+v", result)
	}
}

// TestLive_Scopes exercises the X-OAuth-Scopes header path — the only place
// we parse a response header rather than the body.
func TestLive_Scopes(t *testing.T) {
	c := liveClient(t)
	scopes, err := c.Scopes()
	if err != nil {
		t.Fatalf("Scopes: %v", err)
	}
	if len(scopes) == 0 {
		t.Errorf("expected at least one scope, got none")
	}
}
