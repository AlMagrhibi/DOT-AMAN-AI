package memory

import (
	"context"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/model"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
	"testing"
)

func TestReviewIsolationCorrectionDeletion(t *testing.T) {
	ctx := context.Background()
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, err := s.NewSession(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.NewSession(ctx, "b")
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(s.DB)
	if err != nil {
		t.Fatal(err)
	}
	id, err := m.Put(ctx, Item{Session: a, Kind: "project", Text: "old fact", Source: "user", Reviewed: false})
	if err != nil {
		t.Fatal(err)
	}
	x, err := m.Recall(ctx, a, "")
	if err != nil || len(x) != 0 {
		t.Fatal(x, err)
	}
	if err = m.Correct(ctx, b, id, "wrong", "user"); err == nil {
		t.Fatal("cross session")
	}
	if err = m.Correct(ctx, a, id, "new fact", "user correction"); err != nil {
		t.Fatal(err)
	}
	x, err = m.Recall(ctx, a, "old")
	if err != nil || len(x) != 0 {
		t.Fatal(x, err)
	}
	x, err = m.Recall(ctx, a, "new")
	if err != nil || len(x) != 1 {
		t.Fatal(x, err)
	}
	if err = m.Delete(ctx, a, id); err != nil {
		t.Fatal(err)
	}
	x, err = m.Recall(ctx, a, "")
	if err != nil || len(x) != 0 {
		t.Fatal(x, err)
	}
	if err = s.Add(ctx, a, "user", "original"); err != nil {
		t.Fatal(err)
	}
	msgs, err := s.Messages(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Summary(ctx, a, 1, msgs[0].ID, msgs[0].ID, "unverified summary"); err != nil {
		t.Fatal(err)
	}
}

func TestHierarchicalSummariesKeepOriginals(t *testing.T) {
	ctx := context.Background()
	s, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.NewSession(ctx, "long")
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 48; n++ {
		if err = s.Add(ctx, id, "user", "قرار معلوم مصدره المستخدم"); err != nil {
			t.Fatal(err)
		}
	}
	m, err := New(s.DB)
	if err != nil {
		t.Fatal(err)
	}
	messages, err := s.Messages(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Compress(ctx, id, messages, model.Mock{Text: "unverified decisions"}, 4096); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = s.DB.QueryRow(`SELECT count(*) FROM summaries WHERE level=2`).Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	before, err := m.SummaryContext(ctx, id)
	if err != nil || before == "" {
		t.Fatal(before, err)
	}
	if err = m.Compress(ctx, id, messages, model.Mock{Text: "unchanged"}, 4096); err != nil {
		t.Fatal(err)
	}
	after, err := m.SummaryContext(ctx, id)
	if err != nil || before != after {
		t.Fatal("not idempotent", err)
	}
	original, err := s.Messages(ctx, id)
	if err != nil || len(original) != 48 {
		t.Fatal(len(original), err)
	}
}
