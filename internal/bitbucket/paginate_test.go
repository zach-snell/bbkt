package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a client whose baseURL points at a test server.
func newTestClient(baseURL string) *Client {
	c := NewClient("user", "pass", "")
	c.baseURL = baseURL
	return c
}

func TestRequestPaginated_RejectsNonGET(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	defer srv.Close()
	c := newTestClient(srv.URL)

	for _, m := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		if _, _, err := c.RequestPaginated(m, "/x", []byte(`{}`), 0); err == nil {
			t.Errorf("RequestPaginated(%s) = nil err, want rejection", m)
		}
	}
	if hits != 0 {
		t.Fatalf("expected zero requests for rejected write methods, got %d", hits)
	}
}

func TestRequestPaginated_MergesAndCaps(t *testing.T) {
	const total = 5
	writeHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeHits++
		}
		page := 1
		fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
		if page < 1 {
			page = 1
		}
		resp := map[string]any{"values": []int{page}}
		if page < total {
			// Relative next cursor (host-pinning rejects a foreign host, which is
			// the desired security behavior — see the off-host rejection above).
			resp["next"] = fmt.Sprintf("/2.0/items?page=%d", page+1)
		}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
	defer srv.Close()
	c := newTestClient(srv.URL)

	var out struct {
		Count     int  `json:"count"`
		Pages     int  `json:"pages_fetched"`
		Truncated bool `json:"truncated"`
	}

	// Cap at 3 of 5 pages -> truncated.
	_, body, err := c.RequestPaginated("GET", "/items", nil, 3)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("bad merged json: %v", err)
	}
	if out.Pages != 3 || out.Count != 3 || !out.Truncated {
		t.Fatalf("cap=3: got pages=%d count=%d truncated=%v, want 3/3/true", out.Pages, out.Count, out.Truncated)
	}

	// maxPages<=0 must be bounded but fetch all 5 real pages here (not truncated).
	out = struct {
		Count     int  `json:"count"`
		Pages     int  `json:"pages_fetched"`
		Truncated bool `json:"truncated"`
	}{}
	_, body2, err := c.RequestPaginated("GET", "/items", nil, -1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if err := json.Unmarshal(body2, &out); err != nil {
		t.Fatalf("bad merged json: %v", err)
	}
	if out.Pages != total || out.Truncated {
		t.Fatalf("maxPages=-1: got pages=%d truncated=%v, want %d/false", out.Pages, out.Truncated, total)
	}
	if writeHits != 0 {
		t.Fatalf("pagination issued %d non-GET requests, want 0", writeHits)
	}
}

func TestRequestPaginated_LoopIsBounded(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		// Always point next at the same self-referential (relative) cursor.
		fmt.Fprint(w, `{"values":[1],"next":"/2.0/loop?c=1"}`)
	}))
	defer srv.Close()
	c := newTestClient(srv.URL)

	_, body, err := c.RequestPaginated("GET", "/loop?c=1", nil, 0)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(string(body), `"truncated": true`) {
		t.Errorf("expected truncated=true on a self-referential cursor")
	}
	if requests > 3 {
		t.Fatalf("self-referential cursor made %d requests, expected it bounded", requests)
	}
}
