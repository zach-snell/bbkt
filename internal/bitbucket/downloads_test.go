package bitbucket

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListDownloads_Path(t *testing.T) {
	var got string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RequestURI()
		_, _ = w.Write([]byte(`{"values":[{"name":"a.zip","size":3}],"page":1}`))
	})
	res, err := c.ListDownloads(ListDownloadsArgs{Workspace: "ws", RepoSlug: "repo"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/repositories/ws/repo/downloads?pagelen=25&page=1" || res.Values[0].Name != "a.zip" {
		t.Fatalf("uri=%s values=%+v", got, res.Values)
	}
}

func TestGetDownload_FollowsRedirectWithoutAuth(t *testing.T) {
	var storageAuth string
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storageAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("bundle-bytes"))
	}))
	t.Cleanup(storage.Close)

	var apiPath string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		apiPath = r.URL.EscapedPath()
		// net/http compares hostnames without ports, so use a different name.
		http.Redirect(w, r, strings.Replace(storage.URL, "127.0.0.1", "localhost", 1)+"/signed", http.StatusFound)
	})

	var buf bytes.Buffer
	n, err := c.GetDownload(GetDownloadArgs{Workspace: "ws", RepoSlug: "repo", Name: "my file.zip"}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "bundle-bytes" || n != int64(buf.Len()) {
		t.Fatalf("got %q (%d bytes)", buf.String(), n)
	}
	if apiPath != "/repositories/ws/repo/downloads/my%20file.zip" {
		t.Fatalf("api path = %s", apiPath)
	}
	if storageAuth != "" {
		t.Fatalf("Authorization leaked to storage host: %q", storageAuth)
	}
}

func TestGetDownload_NotFound(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
	})
	_, err := c.GetDownload(GetDownloadArgs{Workspace: "ws", RepoSlug: "repo", Name: "x"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "404") && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetDownload_RejectsDotSegments(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("request should not be sent")
	})
	if _, err := c.GetDownload(GetDownloadArgs{Workspace: "ws", RepoSlug: "repo", Name: ".."}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected '..' to be rejected")
	}
}
