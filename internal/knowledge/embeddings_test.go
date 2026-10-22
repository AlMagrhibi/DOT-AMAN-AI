package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
)

type testEmbedder struct{}

func (testEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for n, text := range texts {
		text = strings.ToLower(text)
		if strings.Contains(text, "firewall") || strings.Contains(text, "network traffic") {
			out[n] = []float32{1, 0}
		} else {
			out[n] = []float32{0, 1}
		}
	}
	return out, nil
}

func TestHybridSemanticSearch(t *testing.T) {
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	i, err := New(s.DB)
	if err != nil {
		t.Fatal(err)
	}
	i.SetEmbedder(testEmbedder{})
	if err = i.Put(context.Background(), Document{ID: "firewall", Source: "fixture", URL: "fixture:firewall", Title: "Firewall", Text: "A firewall filters network traffic between trust zones."}); err != nil {
		t.Fatal(err)
	}
	if err = i.Put(context.Background(), Document{ID: "memory", Source: "fixture", URL: "fixture:memory", Title: "Memory safety", Text: "Memory safety prevents invalid pointer access."}); err != nil {
		t.Fatal(err)
	}
	hits, err := i.Search(context.Background(), "What protects network traffic?", 1)
	if err != nil || len(hits) != 1 || hits[0].ID != "firewall" {
		t.Fatalf("semantic search failed: hits=%+v err=%v", hits, err)
	}
	var count int
	if err = s.DB.QueryRow(`SELECT count(*) FROM chunk_embeddings`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("embedding index count=%d err=%v", count, err)
	}
}

func TestHTTPEmbedderLoopbackAndResponseValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var input struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		data := make([]map[string]any, len(input.Input))
		for n := range input.Input {
			data[n] = map[string]any{"index": n, "embedding": []float32{1, 2, 3}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()
	e, err := NewHTTPEmbedder(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	v, err := e.Embed(context.Background(), []string{"local"})
	if err != nil || len(v) != 1 || len(v[0]) != 3 {
		t.Fatalf("vectors=%v err=%v", v, err)
	}
	if _, err = NewHTTPEmbedder("http://example.com"); err == nil {
		t.Fatal("external endpoint accepted")
	}
	if _, err = NewHTTPEmbedder("http://192.168.1.1:8082"); err == nil {
		t.Fatal("numeric non-loopback endpoint accepted")
	}
	if _, err = NewHTTPEmbedder(fmt.Sprintf("https://127.0.0.1:%s", strings.TrimPrefix(server.URL, "http://127.0.0.1:"))); err == nil {
		t.Fatal("HTTPS endpoint accepted")
	}
}
