package bitbucket

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// withTokenEndpoint points RefreshOAuth (and OAuthLogin's token exchange) at a
// fake server for the duration of a test.
func withTokenEndpoint(t *testing.T, srv *httptest.Server) {
	t.Helper()
	orig := tokenEndpoint
	tokenEndpoint = srv.URL
	t.Cleanup(func() { tokenEndpoint = orig })
}

// sandboxHome redirects $HOME to a tempdir so credential persistence stays
// isolated between tests.
func sandboxHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestRefreshOAuth_Success(t *testing.T) {
	sandboxHome(t)

	var gotGrant, gotRefresh, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotGrant = r.Form.Get("grant_type")
		gotRefresh = r.Form.Get("refresh_token")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "new-access",
			"refresh_token": "rotated-refresh",
			"token_type": "bearer",
			"expires_in": 7200,
			"scopes": "account repository"
		}`))
	}))
	t.Cleanup(srv.Close)
	withTokenEndpoint(t, srv)

	creds := &Credentials{
		ProfileName:  "default",
		AuthType:     AuthTypeOAuth,
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ClientID:     "cid",
		ClientSecret: "csec",
		ExpiresIn:    3600,
		CreatedAt:    time.Now().Add(-2 * time.Hour),
	}

	if err := RefreshOAuth(creds); err != nil {
		t.Fatalf("RefreshOAuth: %v", err)
	}

	if gotGrant != "refresh_token" {
		t.Errorf("grant_type sent = %q, want refresh_token", gotGrant)
	}
	if gotRefresh != "old-refresh" {
		t.Errorf("refresh_token sent = %q, want old-refresh", gotRefresh)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("cid:csec"))
	if gotAuth != wantAuth {
		t.Errorf("Authorization = %q, want %q", gotAuth, wantAuth)
	}
	if creds.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", creds.AccessToken)
	}
	if creds.RefreshToken != "rotated-refresh" {
		t.Errorf("RefreshToken not rotated, got %q", creds.RefreshToken)
	}
	if creds.ExpiresIn != 7200 {
		t.Errorf("ExpiresIn = %d, want 7200", creds.ExpiresIn)
	}

	store, err := LoadProfileStore()
	if err != nil {
		t.Fatalf("LoadProfileStore: %v", err)
	}
	saved := store.Profiles["default"]
	if saved == nil || saved.AccessToken != "new-access" {
		t.Errorf("refreshed creds not persisted: %+v", saved)
	}
}

// Some OAuth providers (including Bitbucket historically) omit refresh_token
// from refresh responses; callers must keep reusing the old one rather than
// clobbering it with empty.
func TestRefreshOAuth_PreservesRefreshTokenWhenServerOmitsIt(t *testing.T) {
	sandboxHome(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"access_token": "new-access",
			"token_type": "bearer",
			"expires_in": 3600
		}`))
	}))
	t.Cleanup(srv.Close)
	withTokenEndpoint(t, srv)

	creds := &Credentials{
		ProfileName: "default", AuthType: AuthTypeOAuth,
		AccessToken: "old", RefreshToken: "original-refresh",
		ClientID: "cid", ClientSecret: "csec",
	}
	if err := RefreshOAuth(creds); err != nil {
		t.Fatal(err)
	}
	if creds.RefreshToken != "original-refresh" {
		t.Errorf("RefreshToken should be preserved when server omits it, got %q", creds.RefreshToken)
	}
	if creds.AccessToken != "new-access" {
		t.Errorf("AccessToken should still update, got %q", creds.AccessToken)
	}
}

func TestRefreshOAuth_ServerErrorKeepsOldCreds(t *testing.T) {
	sandboxHome(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh token expired"}`))
	}))
	t.Cleanup(srv.Close)
	withTokenEndpoint(t, srv)

	creds := &Credentials{
		ProfileName: "default", AuthType: AuthTypeOAuth,
		AccessToken: "still-valid", RefreshToken: "still-valid-refresh",
		ClientID: "cid", ClientSecret: "csec",
	}
	err := RefreshOAuth(creds)
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("error should include server response body: %v", err)
	}
	if creds.AccessToken != "still-valid" {
		t.Errorf("failed refresh should not clobber access_token, got %q", creds.AccessToken)
	}
	if creds.RefreshToken != "still-valid-refresh" {
		t.Errorf("failed refresh should not clobber refresh_token, got %q", creds.RefreshToken)
	}
}

// Defensive: a 200 response with empty access_token must not overwrite a valid
// in-memory token. This can't currently happen with Bitbucket but it's a
// one-line guard against a server-side regression wrecking every client.
func TestRefreshOAuth_RejectsEmptyAccessToken(t *testing.T) {
	sandboxHome(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"","refresh_token":"r","expires_in":3600}`))
	}))
	t.Cleanup(srv.Close)
	withTokenEndpoint(t, srv)

	creds := &Credentials{
		ProfileName: "default", AuthType: AuthTypeOAuth,
		AccessToken: "still-valid", RefreshToken: "still-valid-refresh",
		ClientID: "cid", ClientSecret: "csec",
	}
	err := RefreshOAuth(creds)
	if err == nil {
		t.Fatal("expected error for empty access_token")
	}
	if creds.AccessToken != "still-valid" {
		t.Errorf("empty-token response should not clobber existing token, got %q", creds.AccessToken)
	}
}

// TestDo_401AutoRetry exercises Client.do()'s refresh-and-retry path: the API
// 401s (stale token), the client transparently refreshes, then retries the
// original request with the new bearer.
func TestDo_401AutoRetry(t *testing.T) {
	sandboxHome(t)

	refreshSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"new-bearer","token_type":"bearer","expires_in":3600}`))
	}))
	t.Cleanup(refreshSrv.Close)
	withTokenEndpoint(t, refreshSrv)

	var calls int
	var tokensSeen []string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		tokensSeen = append(tokensSeen, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if calls == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(apiSrv.Close)

	creds := &Credentials{
		ProfileName:  "default",
		AuthType:     AuthTypeOAuth,
		AccessToken:  "stale-bearer",
		RefreshToken: "refresh-tok",
		ClientID:     "cid",
		ClientSecret: "csec",
		ExpiresIn:    3600,
		CreatedAt:    time.Now(),
	}
	// RefreshOAuth persists via SaveProfile, which needs a pre-existing store.
	if err := SaveProfile(creds); err != nil {
		t.Fatal(err)
	}

	c := NewClientFromCredentials(creds)
	c.baseURL = apiSrv.URL

	data, err := c.Get("/anything")
	if err != nil {
		t.Fatalf("Get after 401 retry: %v", err)
	}
	if !strings.Contains(string(data), `"ok":true`) {
		t.Errorf("retry did not get success body: %s", data)
	}
	if calls != 2 {
		t.Fatalf("expected 2 API calls (401 + retry), got %d", calls)
	}
	if len(tokensSeen) != 2 {
		t.Fatalf("expected 2 tokens recorded, got %d", len(tokensSeen))
	}
	if tokensSeen[0] != "stale-bearer" {
		t.Errorf("first call should send stale token, got %q", tokensSeen[0])
	}
	if tokensSeen[1] != "new-bearer" {
		t.Errorf("retry should send refreshed token, got %q", tokensSeen[1])
	}
}

// TestDo_401WithoutOAuth_DoesNotRetry confirms that API token (Basic Auth)
// 401s pass through — retry is OAuth-only and silently retrying Basic Auth
// would mask a permission problem.
func TestDo_401WithoutOAuth_DoesNotRetry(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient("user@example.com", "bad-token", "")
	c.baseURL = srv.URL

	if _, err := c.Get("/foo"); err == nil {
		t.Fatal("expected 401 to surface as error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for Basic Auth), got %d", calls)
	}
}
