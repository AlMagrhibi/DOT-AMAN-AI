// dotaman-eval exercises a real local llama-server. No fixture fallback is allowed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/app"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/config"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/knowledge"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/memory"
	"github.com/AlMagrhibi/DOT-AMAN-AI/internal/model"
)

type sample struct {
	Name    string  `json:"name"`
	Prompt  string  `json:"prompt"`
	Text    string  `json:"text"`
	Output  int     `json:"output_tokens"`
	Input   int     `json:"input_tokens"`
	Seconds float64 `json:"seconds"`
	TTFT    float64 `json:"ttft_seconds"`
	Rate    float64 `json:"end_to_end_tokens_per_second"`
	Passed  bool    `json:"passed"`
	Error   string  `json:"error,omitempty"`
}
type check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}
type report struct {
	Scope    string         `json:"scope"`
	Model    string         `json:"model"`
	Date     string         `json:"date_utc"`
	Settings map[string]any `json:"settings"`
	Samples  []sample       `json:"samples"`
	Checks   []check        `json:"checks"`
	P50      float64        `json:"latency_p50_seconds"`
	P95      float64        `json:"latency_p95_seconds"`
	MeanRate float64        `json:"aggregate_end_to_end_tokens_per_second"`
	Passed   bool           `json:"all_checks_passed"`
}

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:8081", "numeric loopback server")
	runs := flag.Int("runs", 3, "repetitions per prompt, 1..20")
	flag.Parse()
	if *runs < 1 || *runs > 20 {
		fmt.Fprintln(os.Stderr, "runs must be 1..20")
		os.Exit(2)
	}
	if err := run(*endpoint, *runs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(endpoint string, runs int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	p, err := model.New(endpoint)
	if err != nil {
		return err
	}
	if err = p.Health(ctx); err != nil {
		return fmt.Errorf("real model health prerequisite: %w", err)
	}
	info, err := p.ModelInfo(ctx)
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(info), "fixture") || strings.Contains(strings.ToLower(info), "mock") {
		return fmt.Errorf("fixture model refused")
	}
	r := report{Scope: "Codex Cloud real local GGUF smoke/benchmark; small sample, not target-device performance or comprehensive model quality", Model: info, Date: time.Now().UTC().Format(time.RFC3339), Settings: map[string]any{"repetitions": runs, "max_output": 96, "temperature": 0.2, "thinking": false, "stream": true, "latency_percentiles": "nearest rank", "warmup": "one excluded English completion; server prefix cache may remain enabled", "quality_rubric": "language presence plus lexical least-privilege access/minimum evidence; manual semantic review still required"}}
	add := func(name string, ok bool, detail string) { r.Checks = append(r.Checks, check{name, ok, detail}) }
	_, err = p.Complete(ctx, "Reply with READY.", 16)
	add("warmup", err == nil, fmt.Sprint(err))
	for n := 0; n < runs; n++ {
		for _, q := range []struct {
			name, prompt string
			arabic       bool
		}{{"english", "In English, explain least privilege in two short sentences.", false}, {"arabic", "اشرح بالعربية مبدأ أقل امتياز في جملتين قصيرتين.", true}} {
			text := ""
			res, e := p.StreamCompletion(ctx, []model.Message{{Role: "system", Content: app.System + " Answer briefly. Follow the explicitly requested language."}, {Role: "user", Content: fmt.Sprintf("%s [case %d]", q.prompt, n)}}, 96, func(s string) error { text += s; return nil })
			s := sample{Name: q.name, Prompt: q.prompt, Text: text, Input: res.Usage.Input, Output: res.Usage.Output, Seconds: res.Duration.Seconds(), TTFT: res.TTFT.Seconds()}
			if s.Seconds > 0 {
				s.Rate = float64(s.Output) / s.Seconds
			}
			language := false
			for _, c := range text {
				if q.arabic && unicode.Is(unicode.Arabic, c) || !q.arabic && c >= 'a' && c <= 'z' {
					language = true
				}
			}
			lower := strings.ToLower(text)
			concept := false
			if q.arabic {
				for _, w := range []string{"صلاح", "وصول", "أذون", "اذون"} {
					if strings.Contains(lower, w) {
						concept = true
					}
				}
				minimal := false
				for _, w := range []string{"أقل", "اقل", "الحد الأدنى", "الحد الادني", "ضرور", "مطلوب"} {
					if strings.Contains(lower, w) {
						minimal = true
					}
				}
				concept = concept && minimal
			} else {
				concept = strings.Contains(lower, "access") && (strings.Contains(lower, "minimum") || strings.Contains(lower, "necessary") || strings.Contains(lower, "required"))
			}
			s.Passed = e == nil && language && concept && text == res.Text && s.Output > 0 && s.TTFT > 0
			if !concept {
				s.Error = "least-privilege concept rubric missing; requires manual semantic review"
			}
			if e != nil {
				s.Error = e.Error()
			}
			r.Samples = append(r.Samples, s)
		}
	}
	valid, correct := 0, 0
	for n := 0; n < runs*2; n++ {
		calls, e := p.Suggest(ctx, []model.Message{{Role: "system", Content: "Return JSON following the supplied schema. Choose exactly one system_info tool with empty arguments. Never execute anything."}, {Role: "user", Content: "Inspect local kernel and architecture using system_info only."}})
		if e == nil {
			valid++
			if len(calls) == 1 && calls[0].Name == "system_info" {
				correct++
			}
		}
		add(fmt.Sprintf("schema_%d", n), e == nil, fmt.Sprintf("calls=%v error=%v", calls, e))
	}
	add("schema_selection", correct == runs*2, fmt.Sprintf("valid=%d correct=%d total=%d; small smoke set, not a global 95%% claim", valid, correct, runs*2))
	cancelCtx, stop := context.WithCancel(ctx)
	chunks := 0
	_, e := p.StreamCompletion(cancelCtx, []model.Message{{Role: "user", Content: "List Linux security practices in detail."}}, 96, func(s string) error {
		if s != "" {
			chunks++
			stop()
			return cancelCtx.Err()
		}
		return nil
	})
	stop()
	add("stream_cancellation", e != nil && chunks > 0, fmt.Sprintf("chunks=%d error=%v", chunks, e))
	dir, e := os.MkdirTemp("", "dotaman-real-eval-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(dir)
	c, e := config.Default()
	if e != nil {
		return e
	}
	c.Endpoint = endpoint
	c.Root = dir
	c.DataDir = filepath.Join(dir, "data")
	c.MaxOutput = 96
	c.Mode = "analysis"
	a, e := app.Open(c, p)
	if e != nil {
		return e
	}
	d := knowledge.Document{ID: "keystone", Source: "synthetic-real-eval", URL: "fixture:keystone", Title: "KEYSTONE", Text: "The synthetic KEYSTONE service listens on TCP port 43871. This is evaluation data, not a real vulnerability."}
	if e = a.Knowledge.Put(ctx, d); e != nil {
		a.Close()
		return e
	}
	o, e := a.Run(ctx, "", "What port does KEYSTONE use?", nil, nil)
	add("rag_grounded_answer", e == nil && len(o.Citations) > 0 && strings.Contains(o.Text, "43871"), fmt.Sprintf("answer=%s error=%v", o.Text, e))
	id := o.Session
	_, e = a.Memory.Put(ctx, memory.Item{Session: id, Kind: "project", Text: "The synthetic project codename is Cedar-Falcon-728.", Source: "explicit synthetic evaluator", Reviewed: true})
	if e != nil {
		a.Close()
		return e
	}
	if e = a.Close(); e != nil {
		return e
	}
	a, e = app.Open(c, p)
	if e != nil {
		return e
	}
	defer a.Close()
	messages, e := a.Store.Messages(ctx, id)
	add("session_restart", e == nil && len(messages) == 2, fmt.Sprintf("messages=%d error=%v", len(messages), e))
	o, e = a.Run(ctx, id, "What is the synthetic project codename?", nil, nil)
	add("reviewed_memory_recall", e == nil && strings.Contains(o.Text, "Cedar-Falcon-728"), fmt.Sprintf("answer=%s error=%v", o.Text, e))
	a.Config.Mode = "auto"
	o, e = a.Run(ctx, "", "Use system_info once to report the real local kernel and architecture. Do not read files or use other tools.", nil, nil)
	executed := false
	for _, x := range o.Results {
		if x.Name == "system_info" && x.Error == "" && strings.Contains(x.Output, "Linux") {
			executed = true
		}
	}
	add("agent_real_system_info", e == nil && executed, fmt.Sprintf("tools=%v error=%v", o.Results, e))
	times := []float64{}
	tokens := 0
	seconds := 0.0
	r.Passed = true
	for _, s := range r.Samples {
		r.Passed = r.Passed && s.Passed
		if s.Seconds > 0 {
			times = append(times, s.Seconds)
			seconds += s.Seconds
			tokens += s.Output
		}
	}
	for _, x := range r.Checks {
		r.Passed = r.Passed && x.Passed
	}
	sort.Float64s(times)
	if len(times) > 0 {
		r.P50 = times[int(math.Ceil(float64(len(times))*.5))-1]
		r.P95 = times[int(math.Ceil(float64(len(times))*.95))-1]
	}
	if seconds > 0 {
		r.MeanRate = float64(tokens) / seconds
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(r); err != nil {
		return err
	}
	if !r.Passed {
		return fmt.Errorf("real-model evaluation has failed checks; inspect JSON")
	}
	return nil
}
