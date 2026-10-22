// dotaman-semantic-eval runs an independent local embedding/retrieval check.
// It refuses fixture endpoints by requiring a real loopback embeddings API.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/knowledge"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
)

type query struct {
	Query    string `json:"query"`
	Expected string `json:"expected"`
	Language string `json:"language"`
}

type result struct {
	Scope             string  `json:"scope"`
	Endpoint          string  `json:"endpoint"`
	Documents         int     `json:"documents"`
	Queries           int     `json:"queries"`
	RecallAt5         float64 `json:"recall_at_5"`
	ArabicQueries     int     `json:"arabic_queries"`
	ArabicHits        int     `json:"arabic_hits"`
	EmbeddingsIndexed int     `json:"embeddings_indexed"`
	MeanLatencyMS     float64 `json:"mean_latency_ms"`
	Cases             []any   `json:"cases"`
	Passed            bool    `json:"passed"`
}

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:8082", "numeric loopback embedding server")
	corpusPath := flag.String("corpus", "testdata/semantic/corpus.json", "corpus JSON")
	queriesPath := flag.String("queries", "testdata/semantic/queries.json", "query JSON")
	output := flag.String("output", "", "optional JSON output path")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	e, err := knowledge.NewHTTPEmbedder(*endpoint)
	if err != nil {
		fatal(err)
	}
	var docs []knowledge.Document
	if err = readJSON(*corpusPath, &docs); err != nil {
		fatal(err)
	}
	var queries []query
	if err = readJSON(*queriesPath, &queries); err != nil {
		fatal(err)
	}
	dir, err := os.MkdirTemp("", "dotaman-semantic-")
	if err != nil {
		fatal(err)
	}
	defer os.RemoveAll(dir)
	s, err := storage.Open(dir)
	if err != nil {
		fatal(err)
	}
	defer s.Close()
	i, err := knowledge.New(s.DB)
	if err != nil {
		fatal(err)
	}
	i.SetEmbedder(e)
	for _, doc := range docs {
		if err = i.Put(ctx, doc); err != nil {
			fatal(err)
		}
	}
	indexed, err := i.EnsureEmbeddings(ctx, 10000)
	if err != nil {
		fatal(err)
	}
	r := result{Scope: "real local nomic-embed-text-v1.5 GGUF semantic retrieval; independent synthetic corpus", Endpoint: *endpoint, Documents: len(docs), Queries: len(queries), EmbeddingsIndexed: indexed, Cases: []any{}}
	totalLatency := time.Duration(0)
	hits := 0
	for _, q := range queries {
		start := time.Now()
		found, searchErr := i.Search(ctx, q.Query, 5)
		latency := time.Since(start)
		totalLatency += latency
		caseResult := map[string]any{"query": q.Query, "expected": q.Expected, "language": q.Language, "latency_ms": float64(latency.Microseconds()) / 1000, "error": ""}
		ids := []string{}
		for _, h := range found {
			ids = append(ids, h.ID)
			if h.ID == q.Expected {
				hits++
				if q.Language == "ar" {
					r.ArabicHits++
				}
				break
			}
		}
		if q.Language == "ar" {
			r.ArabicQueries++
		}
		caseResult["retrieved"] = ids
		if searchErr != nil {
			caseResult["error"] = searchErr.Error()
		}
		r.Cases = append(r.Cases, caseResult)
	}
	if len(queries) > 0 {
		r.RecallAt5 = float64(hits) / float64(len(queries))
		r.MeanLatencyMS = float64(totalLatency.Microseconds()) / 1000 / float64(len(queries))
	}
	r.Passed = r.RecallAt5 >= 0.75 && (r.ArabicQueries == 0 || float64(r.ArabicHits)/float64(r.ArabicQueries) >= 0.5)
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fatal(err)
	}
	if *output != "" {
		if err = os.WriteFile(*output, append(b, '\n'), 0600); err != nil {
			fatal(err)
		}
	}
	fmt.Println(string(b))
	if !r.Passed {
		os.Exit(1)
	}
}

func readJSON(path string, out any) error {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
