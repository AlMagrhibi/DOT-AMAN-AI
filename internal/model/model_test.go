package model

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPArabicAndStream(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"ok"}`)
			return
		}
		if r.URL.Path == "/v1/models" {
			fmt.Fprint(w, `{"data":[{"id":"local"}]}`)
			return
		}
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), `"stream":true`) {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"أمان\"}}]}\n\ndata: [DONE]\n\n")
			return
		}
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"أمان AI"}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`)
	}))
	defer s.Close()
	h, err := New(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = h.Health(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = h.ModelInfo(ctx); err != nil {
		t.Fatal(err)
	}
	r, err := h.Complete(ctx, "مرحبا Linux", 10)
	if err != nil || r.Text != "أمان AI" || r.Usage.Output != 2 {
		t.Fatal(r, err)
	}
	r, err = h.StreamCompletion(ctx, nil, 10, func(s string) error { return nil })
	if err != nil || r.Text != "أمان" || r.TTFT <= 0 {
		t.Fatal(r, err)
	}
}
func TestRedirectAndCancellation(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "http://example.com", 302) }))
	defer s.Close()
	h, _ := New(s.URL)
	if err := h.Health(context.Background()); err == nil {
		t.Fatal("redirect accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.Health(ctx); err == nil {
		t.Fatal("cancel ignored")
	}
}
func TestParseSuggestions(t *testing.T) {
	for _, s := range []string{`{"calls":[{"name":"shell","arguments":{}}]}`, `{"calls":[]} {}`, `{"calls":null}`, `{"calls":[],"extra":1}`, `{"calls":[{"name":"read_file","arguments":{"path":"../x"}}]}`} {
		if _, err := ParseSuggestions(s); err == nil {
			t.Fatal(s)
		}
	}
	v, err := ParseSuggestions(`{"calls":[{"name":"system_info","arguments":{}}]}`)
	if err != nil || len(v) != 1 {
		t.Fatal(v, err)
	}
}
func TestTimeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	h, _ := New(s.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := h.Health(ctx); err == nil {
		t.Fatal("timeout ignored")
	}
}
func FuzzSuggestions(f *testing.F) {
	f.Add(`{"calls":[]}`)
	f.Fuzz(func(t *testing.T, s string) { _, _ = ParseSuggestions(s) })
}

func TestInferenceIgnoresExternalProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://192.0.2.1:9")
	t.Setenv("HTTPS_PROXY", "http://192.0.2.1:9")
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"status":"ok"}`) }))
	defer s.Close()
	h, err := New(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err = h.Health(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"http://example.com", "http://169.254.169.254", "http://10.0.0.1", "https://127.0.0.1"} {
		if _, err = New(u); err == nil {
			t.Fatal("external endpoint accepted", u)
		}
	}
}
