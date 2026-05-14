package bitbucket

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newBearerClient starts an httptest.Server running handler and returns a
// Client configured with bearer auth pointing at it.
func newBearerClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("", "", "test-token")
	c.baseURL = srv.URL
	return c
}

func TestAcceptHeader_GetIsJSON(t *testing.T) {
	var got string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Accept")
		_, _ = w.Write([]byte("{}"))
	})
	if _, err := c.Get("/foo"); err != nil {
		t.Fatal(err)
	}
	if got != "application/json" {
		t.Errorf("Get Accept = %q, want application/json", got)
	}
}

// Regression for PR #2: step log endpoint returns application/octet-stream and
// other raw endpoints (diff, source) return text/plain. Accept must not pin a
// specific type or Bitbucket will 406.
func TestAcceptHeader_GetRawIsWildcard(t *testing.T) {
	var got string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("raw"))
	})
	if _, _, err := c.GetRaw("/log"); err != nil {
		t.Fatal(err)
	}
	if got != "*/*" {
		t.Errorf("GetRaw Accept = %q, want */*", got)
	}
}

func TestAcceptHeader_PostSendsJSONAndContentType(t *testing.T) {
	var gotAccept, gotCT string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		gotCT = r.Header.Get("Content-Type")
		_, _ = w.Write([]byte("{}"))
	})
	if _, err := c.Post("/foo", map[string]string{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	if gotAccept != "application/json" {
		t.Errorf("Post Accept = %q, want application/json", gotAccept)
	}
	if gotCT != "application/json" {
		t.Errorf("Post Content-Type = %q, want application/json", gotCT)
	}
}

func TestAuth_BearerToken(t *testing.T) {
	var got string
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("{}"))
	})
	if _, err := c.Get("/foo"); err != nil {
		t.Fatal(err)
	}
	if got != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", got)
	}
}

func TestAuth_BasicAuth(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(srv.Close)
	c := NewClient("user@example.com", "api-token-xyz", "")
	c.baseURL = srv.URL

	if _, err := c.Get("/foo"); err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:api-token-xyz"))
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

// parseAPIError specializes 403 with a scope hint; users rely on the hint to
// know what to fix when their token lacks permissions.
func TestGet_403ErrorMentionsScopes(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
	})
	_, err := c.Get("/foo")
	if err == nil {
		t.Fatal("expected 403 to return an error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "scope") {
		t.Errorf("403 error should mention scopes, got: %v", err)
	}
}

// 401 with WWW-Authenticate: BitbucketCustom and no X-Accepted-OAuth-Scopes
// is the fingerprint of a classic (unscoped) Atlassian API token being
// rejected by Bitbucket's BitbucketCustom auth layer. Scopes() should detect
// this and produce a message that points users at the "Create API token
// with scopes" button — not the misleading "X-OAuth-Scopes header missing"
// error we used to emit.
func TestScopes_ClassicTokenFingerprintProducesActionableError(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", "BitbucketCustom realm=Bitbucket.org HTTP")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type":"error","error":{"message":"Token is invalid, expired, or not supported for this endpoint."}}`))
	})

	_, err := c.Scopes()
	if err == nil {
		t.Fatal("expected error for classic-token fingerprint")
	}
	msg := err.Error()
	for _, want := range []string{"Create API token with scopes", "id.atlassian.com", "scoped"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Scopes() error should mention %q, got: %s", want, msg)
		}
	}
}

// A scoped token that simply lacks read:account on /user (403 with
// X-OAuth-Scopes populated) is the existing happy-degraded path: Scopes()
// returns the parsed scope list, no error.
func TestScopes_403WithScopesHeaderIsSuccess(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Oauth-Scopes", "read:repository:bitbucket, read:workspace:bitbucket")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"missing read:account"}}`))
	})

	scopes, err := c.Scopes()
	if err != nil {
		t.Fatalf("Scopes() should succeed when X-Oauth-Scopes is populated, got: %v", err)
	}
	if len(scopes) != 2 {
		t.Errorf("expected 2 scopes, got %v", scopes)
	}
}

// AuthError must be reachable via errors.As so callers can fingerprint
// failures programmatically (e.g. the login flow's switch on
// IsClassicTokenRejection).
func TestAuthError_ReachableViaErrorsAs(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", "BitbucketCustom realm=Bitbucket.org HTTP")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
	})

	_, err := c.Get("/foo")
	if err == nil {
		t.Fatal("expected 401 to return an error")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("error should unwrap to *AuthError, got %T: %v", err, err)
	}
	if !authErr.IsClassicTokenRejection() {
		t.Errorf("IsClassicTokenRejection should be true for BitbucketCustom 401 with no X-Accepted-OAuth-Scopes")
	}
}

func TestGet_NonForbiddenErrorIsPassThrough(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	})
	_, err := c.Get("/foo")
	if err == nil {
		t.Fatal("expected 500 to return an error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should include status code, got: %v", err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "scope") {
		t.Errorf("500 error should not mention scopes, got: %v", err)
	}
}

func TestGetRaw_ReturnsBodyAndContentType(t *testing.T) {
	c := newBearerClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("step log bytes"))
	})
	data, ct, err := c.GetRaw("/pipelines/x/steps/y/log")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "step log bytes" {
		t.Errorf("body = %q, want 'step log bytes'", data)
	}
	if !strings.HasPrefix(ct, "application/octet-stream") {
		t.Errorf("content-type = %q, want application/octet-stream", ct)
	}
}
