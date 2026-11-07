package bitbucket

import (
	"net/http"
	"strings"
	"testing"
)

func TestQueryEscape_Values(t *testing.T) {
	if got := QueryValueEscape("a&b+c"); got != "a%26b%2Bc" {
		t.Errorf("QueryValueEscape = %q, want a%%26b%%2Bc", got)
	}
	if got := QueryValueEscape(`state = "OPEN"`); got != "state%20%3D%20%22OPEN%22" {
		t.Errorf("QueryValueEscape space = %q, want state%%20%%3D%%20%%22OPEN%%22", got)
	}
	if got := QueryEscape("my repo"); got != "my%20repo" {
		t.Errorf("QueryEscape path segment = %q, want my%%20repo", got)
	}
}

func TestQueryEscape_Requests(t *testing.T) {
	var gotQuery string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"values":[],"pagelen":25,"size":0,"page":1}`))
	})

	t.Run("code search", func(t *testing.T) {
		if _, err := c.SearchCode(SearchCodeArgs{Workspace: "ws", RepoSlug: "r", SearchQuery: "a&b+c"}); err != nil {
			t.Fatal(err)
		}
		if want := "search_query=a%26b%2Bc"; !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	})
	t.Run("pull requests", func(t *testing.T) {
		if _, err := c.ListPullRequests(ListPullRequestsArgs{Workspace: "ws", RepoSlug: "r", State: "A&B", Query: "x&y+z"}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"state=A%26B", "q=x%26y%2Bz"} {
			if !strings.Contains(gotQuery, want) {
				t.Errorf("query %q missing %q", gotQuery, want)
			}
		}
	})
	t.Run("repos", func(t *testing.T) {
		if _, err := c.ListRepositories(ListRepositoriesArgs{Workspace: "ws", Query: "x&y+z"}); err != nil {
			t.Fatal(err)
		}
		if want := "q=x%26y%2Bz"; !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	})
}
