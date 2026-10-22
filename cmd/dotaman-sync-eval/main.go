// Command dotaman-sync-eval exercises the real built-in CISA connector in
// Cloud. It writes only metadata and a few search IDs; feed contents stay in
// the temporary SQLite database and are never committed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/knowledge"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/storage"
	"os"
	"time"
)

type report struct {
	SourceBefore []knowledge.Source    `json:"source_before"`
	FirstSync    string                `json:"first_sync"`
	SecondSync   string                `json:"second_sync"`
	SourceAfter  []knowledge.Source    `json:"source_after"`
	History      []knowledge.SyncEvent `json:"history"`
	SearchIDs    []string              `json:"search_ids"`
	DurationMS   int64                 `json:"duration_ms"`
	Passed       bool                  `json:"passed"`
}

func main() {
	outPath := flag.String("output", "docs/validation/cisa-real.json", "metadata report path")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp("", "dotaman-cisa-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	s, err := storage.Open(dir)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	k, err := knowledge.New(s.DB)
	if err != nil {
		panic(err)
	}
	r := report{}
	start := time.Now()
	r.SourceBefore, err = k.Sources(ctx)
	if err != nil {
		panic(err)
	}
	if err = k.Sync(ctx, "cisa-kev"); err != nil {
		r.FirstSync = err.Error()
	} else {
		r.FirstSync = "ok"
	}
	if err = k.Sync(ctx, "cisa-kev"); err != nil {
		r.SecondSync = err.Error()
	} else {
		r.SecondSync = "ok"
	}
	r.SourceAfter, err = k.Sources(ctx)
	if err != nil {
		panic(err)
	}
	r.History, err = k.SyncHistory(ctx, "cisa-kev", 4)
	if err != nil {
		panic(err)
	}
	hits, searchErr := k.Search(ctx, "CVE-2024", 5)
	if searchErr == nil {
		for _, h := range hits {
			r.SearchIDs = append(r.SearchIDs, h.ID)
		}
	}
	r.DurationMS = time.Since(start).Milliseconds()
	for _, s := range r.SourceAfter {
		if s.ID == "cisa-kev" && s.Status == "synced" && s.DocumentCount > 0 && r.FirstSync == "ok" && r.SecondSync == "ok" && len(r.History) >= 2 && len(r.SearchIDs) > 0 {
			r.Passed = true
		}
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(*outPath, append(b, '\n'), 0600); err != nil {
		panic(err)
	}
	fmt.Printf("CISA sync evaluation: passed=%v docs=%d history=%d duration=%dms\n", r.Passed, len(r.SearchIDs), len(r.History), r.DurationMS)
	if !r.Passed {
		os.Exit(1)
	}
}
