package knowledge

import (
	"context"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
	"testing"
)

func TestSearchUpdateDelete(t *testing.T) {
	ctx := context.Background()
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	k, err := New(s.DB)
	if err != nil {
		t.Fatal(err)
	}
	d := Document{ID: "a", Source: "synthetic", URL: "fixture:a", Title: "أَمَان Linux", Text: "إدارة صلاحيات Linux وإصلاح CVE-2025-1234"}
	if err = k.Put(ctx, d); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"امان", "ادارة", "CVE-2025-1234"} {
		h, err := k.Search(ctx, q, 5)
		if err != nil || len(h) != 1 || h[0].URL != "fixture:a" || h[0].SHA256 == "" {
			t.Fatal(q, h, err)
		}
	}
	h, err := k.Search(ctx, `" OR *`, 5)
	if err != nil || len(h) != 0 {
		t.Fatal(h, err)
	}
	d.Text = "patched version"
	if err = k.Put(ctx, d); err != nil {
		t.Fatal(err)
	}
	h, err = k.Search(ctx, "1234", 5)
	if err != nil || len(h) != 0 {
		t.Fatal("stale index", h, err)
	}
	var n int
	if err = s.DB.QueryRow(`SELECT count(*) FROM document_versions`).Scan(&n); err != nil || n != 2 {
		t.Fatal(n, err)
	}
	if err = k.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	h, err = k.Search(ctx, "patched", 5)
	if err != nil || len(h) != 0 {
		t.Fatal(h, err)
	}
}
func FuzzNormalize(f *testing.F) {
	f.Add("أَمَان AI")
	f.Fuzz(func(t *testing.T, s string) { _ = Normalize(s) })
}

func TestNaturalQuestionRetrieval(t *testing.T) {
	ctx := context.Background()
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	k, err := New(s.DB)
	if err != nil {
		t.Fatal(err)
	}
	if err = k.Put(ctx, Document{ID: "port", Source: "fixture", URL: "fixture:port", Title: "KEYSTONE", Text: "The KEYSTONE service listens on port 43871."}); err != nil {
		t.Fatal(err)
	}
	hits, err := k.Search(ctx, "What port does KEYSTONE use?", 5)
	if err != nil || len(hits) == 0 {
		t.Fatalf("natural question lost matching reference: %v %v", hits, err)
	}
}
