// Package evaluation computes retrieval metrics on a separately stored labeled corpus.
package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/knowledge"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
	"os"
	"path/filepath"
	"time"
)

type Query struct {
	Query    string `json:"query"`
	Expected string `json:"expected"`
	Language string `json:"language"`
}
type Case struct {
	Query   Query         `json:"case"`
	Found   bool          `json:"found"`
	IDs     []string      `json:"retrieved"`
	Latency time.Duration `json:"latency_ns"`
}
type Results struct {
	Corpus           int     `json:"documents"`
	Queries          int     `json:"queries"`
	Recall           float64 `json:"recall_at_5"`
	Precision        float64 `json:"precision_at_5"`
	CitationAccuracy float64 `json:"citation_provenance_accuracy"`
	Cases            []Case  `json:"cases"`
	Scope            string  `json:"scope"`
}

func Run(ctx context.Context, fixtures string) (Results, error) {
	r := Results{Scope: "Small synthetic bilingual lexical benchmark; no claim of broad RAG quality or answer faithfulness"}
	var docs []knowledge.Document
	var queries []Query
	for name, dst := range map[string]any{"corpus.json": &docs, "queries.json": &queries} {
		b, err := os.ReadFile(filepath.Join(fixtures, name))
		if err != nil {
			return r, err
		}
		if err = json.Unmarshal(b, dst); err != nil {
			return r, err
		}
	}
	if len(docs) == 0 || len(queries) == 0 {
		return r, fmt.Errorf("empty evaluation")
	}
	dir, err := os.MkdirTemp("", "dotaman-eval-")
	if err != nil {
		return r, err
	}
	defer os.RemoveAll(dir)
	s, err := storage.Open(dir)
	if err != nil {
		return r, err
	}
	defer s.Close()
	k, err := knowledge.New(s.DB)
	if err != nil {
		return r, err
	}
	for _, d := range docs {
		if err = k.Put(ctx, d); err != nil {
			return r, err
		}
	}
	r.Corpus = len(docs)
	r.Queries = len(queries)
	hits := 0
	validCitations, totalCitations := 0, 0
	for _, q := range queries {
		start := time.Now()
		found, err := k.Search(ctx, q.Query, 5)
		if err != nil {
			return r, err
		}
		c := Case{Query: q, Latency: time.Since(start)}
		seen := map[string]bool{}
		for _, h := range found {
			totalCitations++
			if h.URL == "fixture:"+h.ID && h.SHA256 != "" {
				validCitations++
			}
			if !seen[h.ID] {
				c.IDs = append(c.IDs, h.ID)
				seen[h.ID] = true
			}
			if h.ID == q.Expected {
				c.Found = true
			}
		}
		if c.Found {
			hits++
		}
		r.Cases = append(r.Cases, c)
	}
	r.Recall = float64(hits) / float64(len(queries))
	r.Precision = float64(hits) / float64(len(queries)*5)
	if totalCitations > 0 {
		r.CitationAccuracy = float64(validCitations) / float64(totalCitations)
	}
	return r, nil
}
