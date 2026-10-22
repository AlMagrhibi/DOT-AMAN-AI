package knowledge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
)

func TestFetchSourceFallbackAndConditional(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/primary" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		_, _ = w.Write([]byte(`{"vulnerabilities":[{"cveID":"CVE-2026-0001","vulnerabilityName":"Example","shortDescription":"desc","requiredAction":"patch","dateAdded":"2026-01-01"}]}`))
	}))
	defer server.Close()
	store := mustStore(t)
	defer store.Close()
	i, err := New(store.DB)
	if err != nil {
		t.Fatal(err)
	}
	i.sourceClient = server.Client()
	first, etag, modified, notModified, err := i.fetchSource(context.Background(), []string{server.URL + "/primary", server.URL + "/fallback"}, "", "")
	if err != nil || notModified || etag != `"v1"` || modified == "" || !strings.Contains(string(first.body), "CVE-2026-0001") {
		t.Fatalf("fallback result=%+v etag=%q modified=%q notModified=%v err=%v", first, etag, modified, notModified, err)
	}
	_, etag, _, notModified, err = i.fetchSource(context.Background(), []string{server.URL + "/fallback"}, etag, modified)
	if err != nil || !notModified || etag != `"v1"` {
		t.Fatalf("conditional result etag=%q notModified=%v err=%v", etag, notModified, err)
	}
}

func TestSyncErrorRetainsLastGoodDocument(t *testing.T) {
	s := mustStore(t)
	defer s.Close()
	i, err := New(s.DB)
	if err != nil {
		t.Fatal(err)
	}
	if err = i.Put(context.Background(), Document{ID: "good", Source: "cisa-kev", URL: "fixture:good", Title: "Good", Text: "last valid"}); err != nil {
		t.Fatal(err)
	}
	if err = i.syncError(context.Background(), "cisa-kev", fmt.Errorf("network unavailable")); err == nil {
		t.Fatal("expected sync error")
	}
	hits, err := i.Search(context.Background(), "last valid", 5)
	if err != nil || len(hits) != 1 || hits[0].ID != "good" {
		t.Fatalf("last good document lost: hits=%v err=%v", hits, err)
	}
	history, err := i.SyncHistory(context.Background(), "cisa-kev", 1)
	if err != nil || len(history) != 1 || history[0].Status != "error" {
		t.Fatalf("history=%v err=%v", history, err)
	}
}

func mustStore(t *testing.T) *storage.Store {
	t.Helper()
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
